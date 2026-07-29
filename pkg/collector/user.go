//go:build windows

package collector

import (
	"os"
	"syscall"
	"unsafe"
)

// GetCurrentUserInfo собирает логин, полное имя (DisplayName), домен или рабочую группу
func GetCurrentUserInfo() (UserInfo, error) {
	var info UserInfo

	// 1. Сбор технического логина (GetUserNameW)
	var userSize uint32 = 0
	_, _, _ = procGetUserNameW.Call(uintptr(0), uintptr(unsafe.Pointer(&userSize)))
	if userSize == 0 {
		userSize = 256
	}

	bufUser := make([]uint16, userSize)
	ret, _, _ := procGetUserNameW.Call(
		uintptr(unsafe.Pointer(&bufUser[0])), // ИСПРАВЛЕНО: Явный указатель на первый элемент
		uintptr(unsafe.Pointer(&userSize)),
	)

	if ret == 0 {
		info.UserName = "Unknown (API Error)"
	} else {
		info.UserName = syscall.UTF16ToString(bufUser)
	}

	// 2. Сбор Полного Имени (DisplayName) через GetUserNameExW
	var displaySize uint32 = 0
	// Первый вызов замеряет необходимый размер буфера под DisplayName
	_, _, _ = procGetUserNameExW.Call(
		uintptr(NameDisplay),
		uintptr(0),
		uintptr(unsafe.Pointer(&displaySize)),
	)

	if displaySize > 0 {
		bufDisplay := make([]uint16, displaySize)
		// ИСПРАВЛЕНО: Передаем указатель на буфер данных &bufDisplay[0] вместо &displaySize!
		r1, _, _ := procGetUserNameExW.Call(
			uintptr(NameDisplay),
			uintptr(unsafe.Pointer(&bufDisplay[0])), // Передаем адрес первого элемента слайса
			uintptr(unsafe.Pointer(&displaySize)),
		)
		if r1 != 0 {
			info.DisplayName = syscall.UTF16ToString(bufDisplay[:displaySize])
		}
	}

	if info.DisplayName == "" {
		info.DisplayName = info.UserName
	}

	// 3. Сбор статуса подключения через NetGetJoinInformation
	var nameBuffer *uint16
	var joinStatus uint32

	r1, _, _ := procNetGetJoinInformation.Call(
		0,
		uintptr(unsafe.Pointer(&nameBuffer)),
		uintptr(unsafe.Pointer(&joinStatus)),
	)

	// Если вызов успешный, освобождаем буфер Windows ОДОБРЕННЫМ методом,
	// если он не равен nil и не равен -1, но саму память руками НЕ трогаем.
	if r1 == 0 && nameBuffer != nil {
		addr := uintptr(unsafe.Pointer(nameBuffer))
		if addr != 0 && addr != 0xffffffffffffffff && addr != 0xffffffff && addr != ^uintptr(0) {
			_, _, _ = procNetApiBufferFree.Call(uintptr(unsafe.Pointer(nameBuffer)))
		}
	}

	// Безопасный разбор статуса без ручного чтения nameBuffer
	if r1 == 0 {
		switch joinStatus {
		case NetSetupDomainName:
			info.JoinType = "Domain"

			// Запрашиваем полное DNS-имя домена (FQDN) через стабильный GetComputerNameExW
			var dnsDomainSize uint32 = 256
			dnsDomainBuf := make([]uint16, dnsDomainSize)

			r2, _, _ := procGetComputerNameExW.Call(
				uintptr(ComputerNameDnsDomain),
				uintptr(unsafe.Pointer(&dnsDomainBuf[0])), // ИСПРАВЛЕНО: Явный указатель на буфер
				uintptr(unsafe.Pointer(&dnsDomainSize)),
			)

			if r2 != 0 && dnsDomainSize > 0 {
				info.Domain = syscall.UTF16ToString(dnsDomainBuf[:dnsDomainSize])
			} else {
				// Фоллбэк на домен из переменных окружения Windows
				info.Domain = os.Getenv("USERDOMAIN")
			}

		case NetSetupWorkgroupName:
			info.JoinType = "Workgroup"

			// Для рабочей группы получаем имя через USERDOMAIN (быстро и на 100% безопасно)
			wg := os.Getenv("USERDOMAIN")
			if wg != "" {
				info.Domain = wg
			} else {
				info.Domain = "WORKGROUP"
			}

		case NetSetupUnjoined:
			info.JoinType = "Unjoined"
			info.Domain = "N/A"

		default:
			info.JoinType = "Unknown"
			info.Domain = "N/A"
		}
	} else {
		// Если NetGetJoinInformation вообще завершился с ошибкой,
		// используем чистый фоллбэк на переменные окружения Windows
		if os.Getenv("USERDNSDOMAIN") != "" {
			info.JoinType = "Domain"
			info.Domain = os.Getenv("USERDNSDOMAIN")
		} else if os.Getenv("USERDOMAIN") != "" {
			info.JoinType = "Workgroup"
			info.Domain = os.Getenv("USERDOMAIN")
		} else {
			info.JoinType = "Unknown"
			info.Domain = "WORKGROUP"
		}
	}

	return info, nil
}
