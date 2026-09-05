// tracker/internal/collector/user.go

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
	if info.FullName == "" {
		info.FullName = info.Username
	}

	// 3. Полное DNS имя домена (если есть)
	info.DomainFull = c.getFullDomainName()

	// 4. Определяем тип пользователя
	if info.DomainFull != "" {
		// Доменный пользователь
		info.IsDomainUser = true
		info.IsLocalUser = false
	} else {
		// Локальный пользователь
		info.IsDomainUser = false
		info.IsLocalUser = true

		// Получаем рабочую группу
		info.Workgroup = c.getWorkgroup()

		// Для локального пользователя Domain = имя компьютера
		// Очищаем, чтобы не путать с доменом
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
	defer k.Close()

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
		`SYSTEM\CurrentControlSet\Services\Tcpip\Parameters`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return ""
	}
	defer k.Close()

	if workgroup, _, err := k.GetStringValue("Domain"); err == nil && workgroup != "" {
		return workgroup
	}

	return ""
}

// getUserName получает имя пользователя
func (c *UserCollector) getUserName(nameFormat uint32) string {
	var size uint32 = 256
	buffer := make([]uint16, size)

	ret, _, _ := procGetUserNameEx.Call(
		uintptr(nameFormat),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&size)),
	)

	if ret == 0 {
		return ""
	}

	return syscall.UTF16ToString(buffer[:size])
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
