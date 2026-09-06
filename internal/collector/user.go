// internal/collector/user.go
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

// Константы для GetUserNameEx
const (
	nameUnknown       = 0
	nameSamCompatible = 2  // DOMAIN\Username
	nameDisplay       = 3  // Полное имя пользователя
	nameUserPrincipal = 8  // user@domain.com
	nameDnsDomain     = 12 // Полное DNS имя домена
)

// Структура USER_INFO_3 для NetUserGetInfo
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
	UserId          uint32
	PrimaryGroupId  uint32
	Profile         *uint16
	HomeDirDrive    *uint16
	PasswordExpired uint32
}

type UserCollector struct{}

func NewUserCollector() *UserCollector {
	return &UserCollector{}
}

// Collect собирает информацию о текущем пользователе
func (c *UserCollector) Collect() (models.UserInfo, error) {
	info := models.UserInfo{}

	// 1. Получаем имя пользователя в формате DOMAIN\Username (SamCompatible)
	info.Username = c.getUserName(nameSamCompatible)
	if info.Username == "" {
		if u, err := user.Current(); err == nil {
			info.Username = u.Username
		}
	}

	// Извлекаем короткое имя домена из SamCompatible
	if strings.Contains(info.Username, "\\") {
		parts := strings.SplitN(info.Username, "\\", 2)
		info.Domain = parts[0]
	}

	// 2. Полное имя пользователя
	info.FullName = c.getUserName(nameDisplay)
	if info.FullName == "" || info.FullName == info.Username {
		// Пробуем получить из NetUserGetInfo
		if fullName := c.getFullNameFromNetAPI(info.Username); fullName != "" {
			info.FullName = fullName
		} else {
			info.FullName = info.Username
		}
	}

	// 3. Полное DNS имя домена (если есть)
	info.DomainFull = c.getFullDomainName()

	// 4. Определяем тип пользователя
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

	// 5. Проверяем права администратора
	info.IsAdmin = c.isAdmin()

	// 6. Путь к профилю
	info.ProfilePath = c.getProfilePath()

	return info, nil
}

// getFullDomainName получает полное DNS имя домена
func (c *UserCollector) getFullDomainName() string {
	// Способ 1: Из UPN (user@domain.com)
	upn := c.getUserName(nameUserPrincipal)
	if strings.Contains(upn, "@") {
		parts := strings.SplitN(upn, "@", 2)
		return parts[1]
	}

	// Способ 2: Из DNS Domain
	dnsDomain := c.getUserName(nameDnsDomain)
	if dnsDomain != "" && !strings.Contains(dnsDomain, "\\") {
		return dnsDomain
	}

	// Способ 3: Из реестра
	if domain := c.getDomainFromRegistry(); domain != "" {
		return domain
	}

	return ""
}

// getDomainFromRegistry получает полное имя домена из реестра
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

// getWorkgroup получает имя рабочей группы
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

// getUserName получает имя пользователя
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

// getFullNameFromNetAPI получает полное имя через NetUserGetInfo
func (c *UserCollector) getFullNameFromNetAPI(username string) string {
	// Извлекаем только имя пользователя (без домена)
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
		0, // NULL - локальный компьютер
		uintptr(unsafe.Pointer(userNamePtr)),
		3, // Уровень информации
		uintptr(unsafe.Pointer(&userInfoPtr)),
	)

	if ret != 0 {
		return ""
	}
	defer procNetApiBufferFree.Call(uintptr(unsafe.Pointer(userInfoPtr)))

	if userInfoPtr.FullName != nil {
		return syscall.UTF16ToString((*[256]uint16)(unsafe.Pointer(userInfoPtr.FullName))[:])
	}

	return ""
}

// isAdmin проверяет, является ли пользователь администратором
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

// getProfilePath получает путь к профилю
func (c *UserCollector) getProfilePath() string {
	if profile := os.Getenv("USERPROFILE"); profile != "" {
		return profile
	}
	return ""
}
