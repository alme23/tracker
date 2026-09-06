//go:build windows

package collector

import (
	"os"
	"os/user"
	"strings"
	"syscall"
	"unsafe"

	"github.com/alme23/tracker/internal/models"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Constants for GetUserNameEx
const (
	nameUnknown       = 0
	nameSamCompatible = 2  // DOMAIN\Username
	nameDisplay       = 3  // Full display name
	nameUserPrincipal = 8  // user@domain.com
	nameDNSDomain     = 12 // Full DNS domain name
)

// USER_INFO_3 structure for NetUserGetInfo
type userInfo3 struct {
	Name            *uint16
	Password        *uint16
	PasswordAge     uint32
	Priv            uint32
	HomeDir         *uint16
	Comment         *uint16
	Flags           uint32
	ScriptPath      *uint16
	AuthFlags       uint32
	FullName        *uint16
	UsrComment      *uint16
	Parms           *uint16
	Workstations    *uint16
	LastLogon       uint32
	LastLogoff      uint32
	AcctExpires     uint32
	MaxStorage      uint32
	UnitsPerWeek    uint32
	LogonHours      uintptr
	BadPwCount      uint32
	NumLogons       uint32
	LogonServer     *uint16
	CountryCode     uint32
	CodePage        uint32
	UserID          uint32
	PrimaryGroupID  uint32
	Profile         *uint16
	HomeDirDrive    *uint16
	PasswordExpired uint32
}

// UserCollector collects information about the current user
type UserCollector struct{}

// NewUserCollector creates a new UserCollector
func NewUserCollector() *UserCollector {
	return &UserCollector{}
}

// Collect gathers information about the current user
func (c *UserCollector) Collect() (models.UserInfo, error) {
	info := models.UserInfo{}

	// 1. Get username in DOMAIN\Username format (SamCompatible)
	info.Username = c.getUserName(nameSamCompatible)
	if info.Username == "" {
		if u, err := user.Current(); err == nil {
			info.Username = u.Username
		}
	}

	// Extract short domain name from SamCompatible
	if strings.Contains(info.Username, "\\") {
		parts := strings.SplitN(info.Username, "\\", 2)
		info.Domain = parts[0]
	}

	// 2. Full display name
	info.FullName = c.getUserName(nameDisplay)
	if info.FullName == "" || info.FullName == info.Username {
		// Try to get from NetUserGetInfo
		if fullName := c.getFullNameFromNetAPI(info.Username); fullName != "" {
			info.FullName = fullName
		} else {
			info.FullName = info.Username
		}
	}

	// 3. Full DNS domain name (if available)
	info.DomainFull = c.getFullDomainName()

	// 4. Determine user type
	if info.DomainFull != "" {
		info.IsDomainUser = true
		info.IsLocalUser = false
		info.Workgroup = ""
	} else {
		info.IsDomainUser = false
		info.IsLocalUser = true
		info.Workgroup = c.getWorkgroup()
		if !strings.Contains(info.Username, "\\") {
			info.Domain = ""
		}
	}

	// 5. Check administrator rights
	info.IsAdmin = c.isAdmin()

	// 6. Profile path
	info.ProfilePath = c.getProfilePath()

	return info, nil
}

// getFullDomainName returns the full DNS domain name
func (c *UserCollector) getFullDomainName() string {
	// Method 1: From UPN (user@domain.com)
	upn := c.getUserName(nameUserPrincipal)
	if strings.Contains(upn, "@") {
		parts := strings.SplitN(upn, "@", 2)
		return parts[1]
	}

	// Method 2: From DNS Domain
	dnsDomain := c.getUserName(nameDNSDomain)
	if dnsDomain != "" && !strings.Contains(dnsDomain, "\\") {
		return dnsDomain
	}

	// Method 3: From registry
	if domain := c.getDomainFromRegistry(); domain != "" {
		return domain
	}

	return ""
}

// getDomainFromRegistry returns the full domain name from the registry
func (c *UserCollector) getDomainFromRegistry() string {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Services\Tcpip\Parameters`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return ""
	}
	defer func() {
		_ = k.Close()
	}()

	if domain, _, err := k.GetStringValue("Domain"); err == nil && domain != "" {
		return domain
	}

	if domain, _, err := k.GetStringValue("NV Domain"); err == nil && domain != "" {
		return domain
	}

	return ""
}

// getWorkgroup returns the workgroup name
func (c *UserCollector) getWorkgroup() string {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Services\LanmanWorkstation\Parameters`,
		registry.QUERY_VALUE,
	)
	if err == nil {
		defer func() {
			_ = k.Close()
		}()
		if wg, _, err := k.GetStringValue("Domain"); err == nil && wg != "" {
			return wg
		}
	}

	return "WORKGROUP"
}

// getUserName returns the user name in the specified format
func (c *UserCollector) getUserName(nameFormat uint32) string {
	var size uint32 = 256
	buffer := make([]uint16, size)

	ret, _, _ := procGetUserNameEx.Call(
		uintptr(nameFormat),
		uintptr(unsafe.Pointer(unsafe.SliceData(buffer))),
		uintptr(unsafe.Pointer(&size)),
	)

	if ret == 0 || size == 0 {
		return ""
	}

	return windows.UTF16ToString(buffer[:size])
}

// getFullNameFromNetAPI returns the full name via NetUserGetInfo
func (c *UserCollector) getFullNameFromNetAPI(username string) string {
	// Extract only the username (without domain)
	userName := username
	for i := len(username) - 1; i >= 0; i-- {
		if username[i] == '\\' {
			userName = username[i+1:]
			break
		}
	}

	userNamePtr, err := syscall.UTF16PtrFromString(userName)
	if err != nil {
		return ""
	}

	var userInfoPtr *userInfo3
	ret, _, _ := procNetUserGetInfo.Call(
		0,
		uintptr(unsafe.Pointer(userNamePtr)),
		3,
		uintptr(unsafe.Pointer(&userInfoPtr)),
	)

	if ret != 0 || userInfoPtr == nil {
		return ""
	}

	// Free memory
	defer func() {
		ret, _, _ := procNetAPIBufferFree.Call(uintptr(unsafe.Pointer(userInfoPtr)))
		if ret != 0 {
			// Free error — ignore
			_ = ret
		}
	}()

	if userInfoPtr.FullName != nil {
		return syscall.UTF16ToString((*[256]uint16)(unsafe.Pointer(userInfoPtr.FullName))[:])
	}

	return ""
}

// isAdmin checks if the user is an administrator
func (c *UserCollector) isAdmin() bool {
	sid, err := windows.StringToSid("S-1-5-32-544")
	if err != nil {
		return false
	}

	var isMember bool
	ret, _, _ := procCheckTokenMembership.Call(
		0,
		uintptr(unsafe.Pointer(sid)),
		uintptr(unsafe.Pointer(&isMember)),
	)

	return ret != 0 && isMember
}

// getProfilePath returns the user profile path
func (c *UserCollector) getProfilePath() string {
	if profile := os.Getenv("USERPROFILE"); profile != "" {
		return profile
	}
	return ""
}
