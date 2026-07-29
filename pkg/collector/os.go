//go:build windows

package collector

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

// GetHostInfo собирает параметры ОС, Machine GUID, а также FQDN и имя домена ПК
func GetHostInfo() (HostInfo, error) {
	var info HostInfo

	defer func() {
		if r := recover(); r != nil {
			if info.ComputerName == "" {
				info.ComputerName = "Unknown-Host"
			}
			if info.OSName == "" {
				info.OSName = "Unknown Windows Version"
			}
		}
	}()

	// 1. Получаем короткое NetBIOS имя ПК
	var size uint32 = 256
	bufNetBIOS := make([]uint16, size)
	// ИСПРАВЛЕНО: Передаем указатель на нулевой элемент слайса &bufNetBIOS[0]
	ret, _, _ := procGetComputerNameExW.Call(
		uintptr(ComputerNamePhysicalNetBIOS),
		uintptr(unsafe.Pointer(&bufNetBIOS[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if ret != 0 && size > 0 {
		info.ComputerName = syscall.UTF16ToString(bufNetBIOS[:size])
	} else {
		info.ComputerName = os.Getenv("COMPUTERNAME")
	}

	// 2. Получаем Полное имя компьютера (FQDN)
	var fqdnSize uint32 = 512
	bufFQDN := make([]uint16, fqdnSize)
	// ИСПРАВЛЕНО: Передаем указатель на нулевой элемент слайса &bufFQDN[0]
	ret, _, _ = procGetComputerNameExW.Call(
		uintptr(ComputerNamePhysicalDnsFullyQual),
		uintptr(unsafe.Pointer(&bufFQDN[0])),
		uintptr(unsafe.Pointer(&fqdnSize)),
	)
	if ret != 0 && fqdnSize > 0 {
		info.FQDN = syscall.UTF16ToString(bufFQDN[:fqdnSize])
	} else {
		info.FQDN = info.ComputerName // Фоллбэк, если ПК вне домена
	}

	// 3. Получаем имя DNS-домена Active Directory
	var domainSize uint32 = 512
	bufDomain := make([]uint16, domainSize)
	// ИСПРАВЛЕНО: Передаем указатель на нулевой элемент слайса &bufDomain[0]
	ret, _, _ = procGetComputerNameExW.Call(
		uintptr(ComputerNamePhysicalDnsDomain),
		uintptr(unsafe.Pointer(&bufDomain[0])),
		uintptr(unsafe.Pointer(&domainSize)),
	)
	if ret != 0 && domainSize > 0 {
		info.DomainName = syscall.UTF16ToString(bufDomain[:domainSize])
	} else {
		info.DomainName = os.Getenv("USERDNSDOMAIN") // Фоллбэк на переменные окружения
	}

	// Очистка пустых значений для ПК в Workgroup
	if info.DomainName == "" {
		info.DomainName = "WORKGROUP"
	}

	// 4. Сбор архитектуры
	var sysInfo SYSTEM_INFO
	_, _, _ = procGetSystemInfo.Call(uintptr(unsafe.Pointer(&sysInfo)))
	switch sysInfo.ProcessorArchitecture {
	case 9:
		info.Architecture = "x64"
	case 5:
		info.Architecture = "ARM"
	case 12:
		info.Architecture = "ARM64"
	case 0:
		info.Architecture = "x86"
	default:
		info.Architecture = fmt.Sprintf("Unknown (%d)", sysInfo.ProcessorArchitecture)
	}

	// 5. Чтение детальных параметров Windows из реестра
	osKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err == nil {
		defer func() { _ = osKey.Close() }()

		productName, _, err := osKey.GetStringValue("ProductName")
		if err == nil {
			info.OSName = productName
		}

		edition, _, err := osKey.GetStringValue("EditionID")
		if err == nil {
			info.OSEdition = strings.TrimSpace(edition)
		}

		displayVer, _, err := osKey.GetStringValue("DisplayVersion")
		if err == nil && displayVer != "" {
			info.OSVersion = strings.TrimSpace(displayVer)
		} else {
			release, _, _ := osKey.GetStringValue("ReleaseId")
			info.OSVersion = strings.TrimSpace(release)
		}

		buildNum, _, err := osKey.GetStringValue("CurrentBuild")
		if err == nil {
			info.OSBuild = buildNum
			var bNum int
			_, _ = fmt.Sscanf(buildNum, "%d", &bNum)
			if bNum >= 22000 && strings.Contains(info.OSName, "Windows 10") {
				info.OSName = strings.Replace(info.OSName, "Windows 10", "Windows 11", 1)
			}
		}

		ubr, _, err := osKey.GetIntegerValue("UBR")
		if err == nil && info.OSBuild != "" {
			info.OSBuild = fmt.Sprintf("%s.%d", info.OSBuild, ubr)
		}

		prodID, _, err := osKey.GetStringValue("ProductId")
		if err == nil {
			info.ProductID = strings.TrimSpace(prodID)
		}

		instDate, _, err := osKey.GetIntegerValue("InstallDate")
		if err == nil && instDate > 0 {
			info.InstallDateUnix = int64(instDate)
			info.InstallDateStr = time.Unix(info.InstallDateUnix, 0).Format("2006-01-02 15:04:05")
		}
	}

	// Настройка дефолтных значений
	if info.OSName == "" {
		info.OSName = "Unknown Windows Version"
	}

	// 6. Чтение Machine GUID системы
	cryptoKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE)
	if err == nil {
		machGUID, _, err := cryptoKey.GetStringValue("MachineGuid")
		if err == nil {
			info.DeviceID = strings.ToUpper(strings.TrimSpace(machGUID))
		}
		_ = cryptoKey.Close()
	}
	if info.DeviceID == "" {
		info.DeviceID = "Unknown Device ID"
	}

	return info, nil
}
