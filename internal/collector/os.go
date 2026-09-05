// tracker/internal/collector/os.go

//go:build windows

package collector

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/alme23/tracker/internal/models"
	"github.com/google/uuid"
	"golang.org/x/sys/windows/registry"
)

// Константы продуктов Windows
const (
	PRODUCT_BUSINESS                            = 0x00000006
	PRODUCT_BUSINESS_N                          = 0x00000010
	PRODUCT_CLUSTER_SERVER                      = 0x00000012
	PRODUCT_CLUSTER_SERVER_V                    = 0x00000040
	PRODUCT_CORE                                = 0x00000065
	PRODUCT_CORE_COUNTRYSPECIFIC                = 0x00000063
	PRODUCT_CORE_N                              = 0x00000062
	PRODUCT_CORE_SINGLELANGUAGE                 = 0x00000064
	PRODUCT_DATACENTER_EVALUATION_SERVER        = 0x00000050
	PRODUCT_DATACENTER_A_SERVER                 = 0x00000059
	PRODUCT_DATACENTER_SERVER                   = 0x00000008
	PRODUCT_DATACENTER_SERVER_CORE              = 0x0000000C
	PRODUCT_DATACENTER_SERVER_CORE_V            = 0x00000027
	PRODUCT_DATACENTER_SERVER_V                 = 0x00000025
	PRODUCT_EDUCATION                           = 0x00000079
	PRODUCT_EDUCATION_N                         = 0x0000007A
	PRODUCT_ENTERPRISE                          = 0x00000004
	PRODUCT_ENTERPRISE_E                        = 0x00000046
	PRODUCT_ENTERPRISE_N                        = 0x0000001B
	PRODUCT_ENTERPRISE_N_EVALUATION             = 0x00000054
	PRODUCT_ENTERPRISE_S                        = 0x0000007D
	PRODUCT_ENTERPRISE_S_EVALUATION             = 0x00000081
	PRODUCT_ENTERPRISE_S_N                      = 0x0000007E
	PRODUCT_ENTERPRISE_S_N_EVALUATION           = 0x00000082
	PRODUCT_ENTERPRISE_EVALUATION               = 0x00000048
	PRODUCT_ENTERPRISE_SERVER                   = 0x0000000A
	PRODUCT_ENTERPRISE_SERVER_CORE              = 0x0000000E
	PRODUCT_ENTERPRISE_SERVER_CORE_V            = 0x00000029
	PRODUCT_ENTERPRISE_SERVER_IA64              = 0x0000000F
	PRODUCT_ENTERPRISE_SERVER_V                 = 0x00000026
	PRODUCT_ESSENTIALBUSINESS_SERVER_ADDL       = 0x0000003C
	PRODUCT_ESSENTIALBUSINESS_SERVER_ADDLSVC    = 0x0000003E
	PRODUCT_ESSENTIALBUSINESS_SERVER_MGMT       = 0x0000003B
	PRODUCT_ESSENTIALBUSINESS_SERVER_MGMTSVC    = 0x0000003D
	PRODUCT_HOME_BASIC                          = 0x00000002
	PRODUCT_HOME_BASIC_E                        = 0x00000043
	PRODUCT_HOME_BASIC_N                        = 0x00000005
	PRODUCT_HOME_PREMIUM                        = 0x00000003
	PRODUCT_HOME_PREMIUM_E                      = 0x00000044
	PRODUCT_HOME_PREMIUM_N                      = 0x0000001A
	PRODUCT_HOME_PREMIUM_SERVER                 = 0x00000022
	PRODUCT_HOME_SERVER                         = 0x00000013
	PRODUCT_MEDIUMBUSINESS_SERVER_MANAGEMENT    = 0x0000001E
	PRODUCT_MEDIUMBUSINESS_SERVER_MESSAGING     = 0x00000020
	PRODUCT_MEDIUMBUSINESS_SERVER_SECURITY      = 0x0000001F
	PRODUCT_MOBILE_CORE                         = 0x00000068
	PRODUCT_MOBILE_ENTERPRISE                   = 0x00000085
	PRODUCT_MULTIPOINT_PREMIUM_SERVER           = 0x0000004D
	PRODUCT_MULTIPOINT_STANDARD_SERVER          = 0x0000004C
	PRODUCT_PROFESSIONAL                        = 0x00000030
	PRODUCT_PROFESSIONAL_E                      = 0x00000045
	PRODUCT_PROFESSIONAL_N                      = 0x00000031
	PRODUCT_PROFESSIONAL_WMC                    = 0x00000067
	PRODUCT_SB_SOLUTION_SERVER                  = 0x00000032
	PRODUCT_SB_SOLUTION_SERVER_EM               = 0x00000036
	PRODUCT_SERVER_FOR_SB_SOLUTIONS             = 0x00000033
	PRODUCT_SERVER_FOR_SB_SOLUTIONS_EM          = 0x00000037
	PRODUCT_SERVER_FOUNDATION                   = 0x00000021
	PRODUCT_SMALLBUSINESS_SERVER                = 0x00000009
	PRODUCT_SMALLBUSINESS_SERVER_PREMIUM        = 0x00000019
	PRODUCT_SMALLBUSINESS_SERVER_PREMIUM_CORE   = 0x0000003F
	PRODUCT_SOLUTION_EMBEDDEDSERVER             = 0x00000038
	PRODUCT_STANDARD_EVALUATION_SERVER          = 0x0000004F
	PRODUCT_STANDARD_A_SERVER                   = 0x00000058
	PRODUCT_STANDARD_SERVER                     = 0x00000007
	PRODUCT_STANDARD_SERVER_CORE                = 0x0000000D
	PRODUCT_STANDARD_SERVER_CORE_V              = 0x00000028
	PRODUCT_STANDARD_SERVER_V                   = 0x00000024
	PRODUCT_STARTER                             = 0x0000000B
	PRODUCT_STARTER_E                           = 0x00000042
	PRODUCT_STARTER_N                           = 0x0000002F
	PRODUCT_STORAGE_ENTERPRISE_SERVER           = 0x00000017
	PRODUCT_STORAGE_ENTERPRISE_SERVER_CORE      = 0x0000002E
	PRODUCT_STORAGE_EXPRESS_SERVER              = 0x00000014
	PRODUCT_STORAGE_EXPRESS_SERVER_CORE         = 0x0000002B
	PRODUCT_STORAGE_STANDARD_EVALUATION_SERVER  = 0x00000057
	PRODUCT_STORAGE_STANDARD_SERVER             = 0x00000015
	PRODUCT_STORAGE_STANDARD_SERVER_CORE        = 0x0000002A
	PRODUCT_STORAGE_WORKGROUP_EVALUATION_SERVER = 0x00000056
	PRODUCT_STORAGE_WORKGROUP_SERVER            = 0x00000016
	PRODUCT_STORAGE_WORKGROUP_SERVER_CORE       = 0x0000002C
	PRODUCT_ULTIMATE                            = 0x00000001
	PRODUCT_ULTIMATE_E                          = 0x00000047
	PRODUCT_ULTIMATE_N                          = 0x0000001C
	PRODUCT_WEB_SERVER                          = 0x00000011
	PRODUCT_WEB_SERVER_CORE                     = 0x0000001D
	PRODUCT_UNLICENSED                          = 0xABCDABCD
	PRODUCT_PROFESSIONAL_STUDENT                = 0x00000078
	PRODUCT_PROFESSIONAL_STUDENT_N              = 0x00000077
	PRODUCT_PRO_CHINA                           = 0x00000071
	PRODUCT_PRO_SINGLE_LANGUAGE                 = 0x00000074
	PRODUCT_PRO_WORKSTATION                     = 0x000000A1
	PRODUCT_PRO_WORKSTATION_N                   = 0x000000A2
	PRODUCT_PRO_FOR_EDUCATION                   = 0x000000A3
	PRODUCT_PRO_FOR_EDUCATION_N                 = 0x000000A4
	PRODUCT_AZURE_SERVER_CORE                   = 0x000000A8
	PRODUCT_AZURE_NANO_SERVER                   = 0x000000A9
	PRODUCT_ENTERPRISE_FOR_VIRTUAL_DESKTOPS     = 0x000000B1
	PRODUCT_IOBALANCE_SERVER                    = 0x000000B2
	PRODUCT_MULTI_SESSION                       = 0x000000AF
	PRODUCT_CLOUD_HOST_INFRASTRUCTURE_SERVER    = 0x000000B4
	PRODUCT_CLOUD_STORAGE_SERVER                = 0x000000B6
	PRODUCT_SERVER_V                            = 0x000000B7
	PRODUCT_SERVER                              = 0x000000B8
	PRODUCT_WINDOWS_10_ENTERPRISE               = 0x000000B9
	PRODUCT_WINDOWS_10_EDUCATION                = 0x000000BA
	PRODUCT_WINDOWS_10_PRO                      = 0x000000BB
	PRODUCT_WINDOWS_10_HOME                     = 0x000000BC
	PRODUCT_WINDOWS_10_TEAM                     = 0x000000BD
	PRODUCT_WINDOWS_10_PRO_WORKSTATION          = 0x000000BE
	PRODUCT_WINDOWS_10_PRO_EDUCATION            = 0x000000BF
	PRODUCT_WINDOWS_10_S                        = 0x000000C0
	PRODUCT_WINDOWS_10_ENTERPRISE_N             = 0x000000C1
	PRODUCT_WINDOWS_10_EDUCATION_N              = 0x000000C2
	PRODUCT_WINDOWS_10_PRO_N                    = 0x000000C3
	PRODUCT_WINDOWS_10_HOME_N                   = 0x000000C4
	PRODUCT_WINDOWS_10_TEAM_N                   = 0x000000C5
	PRODUCT_WINDOWS_10_PRO_WORKSTATION_N        = 0x000000C6
	PRODUCT_WINDOWS_10_PRO_EDUCATION_N          = 0x000000C7
	PRODUCT_WINDOWS_10_S_N                      = 0x000000C8
	PRODUCT_WINDOWS_10_ENTERPRISE_S             = 0x000000C9
	PRODUCT_WINDOWS_10_EDUCATION_S              = 0x000000CA
	PRODUCT_WINDOWS_10_PRO_S                    = 0x000000CB
	PRODUCT_WINDOWS_10_HOME_S                   = 0x000000CC
	PRODUCT_WINDOWS_10_ENTERPRISE_S_N           = 0x000000CD
	PRODUCT_WINDOWS_10_EDUCATION_S_N            = 0x000000CE
	PRODUCT_WINDOWS_10_PRO_S_N                  = 0x000000CF
	PRODUCT_WINDOWS_10_HOME_S_N                 = 0x000000D0
	PRODUCT_WINDOWS_10_ENTERPRISE_EVAL          = 0x000000D1
	PRODUCT_WINDOWS_10_EDUCATION_EVAL           = 0x000000D2
	PRODUCT_WINDOWS_10_PRO_EVAL                 = 0x000000D3
	PRODUCT_WINDOWS_10_HOME_EVAL                = 0x000000D4
	PRODUCT_WINDOWS_10_ENTERPRISE_N_EVAL        = 0x000000D5
	PRODUCT_WINDOWS_10_EDUCATION_N_EVAL         = 0x000000D6
	PRODUCT_WINDOWS_10_PRO_N_EVAL               = 0x000000D7
	PRODUCT_WINDOWS_10_HOME_N_EVAL              = 0x000000D8
)

// RTL_OSVERSIONINFOEXW структура для RtlGetVersion
type rtlOSVersionInfoEx struct {
	OSVersionInfoSize uint32
	MajorVersion      uint32
	MinorVersion      uint32
	BuildNumber       uint32
	PlatformId        uint32
	CSDVersion        [128]uint16
	ServicePackMajor  uint16
	ServicePackMinor  uint16
	SuiteMask         uint16
	ProductType       byte
	Reserved          byte
}

// osDetails содержит всю информацию об ОС
type osDetails struct {
	name             string
	edition          string
	buildNumber      string
	kernelVersion    string
	productID        string
	registeredOwner  string
	registeredOrg    string
	installationType string
	displayVersion   string
}

type OSCollector struct {
	cache     *models.OSInfo
	cacheTime time.Time
	mu        sync.RWMutex
}

func NewOSCollector() *OSCollector {
	return &OSCollector{}
}

// Collect собирает расширенную информацию об ОС Windows
func (c *OSCollector) Collect() (models.OSInfo, error) {
	// Проверяем кэш (действителен 5 минут)
	c.mu.RLock()
	if c.cache != nil && time.Since(c.cacheTime) < 5*time.Minute {
		cached := *c.cache
		c.mu.RUnlock()
		return cached, nil
	}
	c.mu.RUnlock()

	var info models.OSInfo

	// Архитектура
	info.Architecture = c.getArchitecture()

	// Получаем все данные из одного источника
	details := c.getOSDetails()

	// Заполняем основные поля
	info.Name = details.name
	info.Edition = details.edition
	info.BuildNumber = details.buildNumber
	info.KernelVersion = details.kernelVersion
	info.ProductID = details.productID
	info.RegisteredOwner = details.registeredOwner
	info.RegisteredOrg = details.registeredOrg
	info.InstallationType = details.installationType

	// Дополнительная информация
	info.Locale = c.getSystemLocale()
	info.InstallDate = c.getInstallDate()
	info.SecureBootLines = c.isSecureBootEnabled()
	info.PowerShellVer = c.getPowerShellVersion()
	info.MachineGUID = c.getMachineGUID()

	// Виртуализация
	info.IsVirtual = c.checkVirtualization()
	info.IsHypervisor = c.checkHypervisorHost()

	// Сохраняем в кэш
	c.mu.Lock()
	infoCopy := info
	c.cache = &infoCopy
	c.cacheTime = time.Now()
	c.mu.Unlock()

	return info, nil
}

// getArchitecture определяет архитектуру процессора
func (c *OSCollector) getArchitecture() models.ArchFamilyType {
	switch runtime.GOARCH {
	case "amd64":
		return models.AMD64
	case "386":
		return models.I386
	case "arm":
		return models.ARM
	case "arm64":
		return models.ARM64
	case "loong64":
		return models.LOONG64
	case "mips":
		return models.MIPS
	case "mips64":
		return models.MIPS64
	case "ppc64":
		return models.PPC64
	case "riscv64":
		return models.RISCV64
	case "s390x":
		return models.S390X
	case "wasm":
		return models.WASM
	default:
		return models.UnknownArch
	}
}

// getOSDetails собирает всю информацию об ОС за один проход
func (c *OSCollector) getOSDetails() osDetails {
	d := osDetails{
		name:             "Windows",
		edition:          "UNKNOWN",
		buildNumber:      "0",
		kernelVersion:    "0.0.0",
		productID:        "UNKNOWN",
		registeredOwner:  "UNKNOWN",
		registeredOrg:    "UNKNOWN",
		installationType: "UNKNOWN",
	}

	// Получаем версию через RtlGetVersion (теперь 4 значения)
	major, minor, build, _ := c.getWindowsVersionNumbers()
	d.buildNumber = fmt.Sprintf("%d", build)
	d.kernelVersion = fmt.Sprintf("%d.%d.%d", major, minor, build)

	// Получаем точное название Windows на основе мажорных версий и билдов
	d.name = c.getWindowsVersionName(major, minor, build)

	// Открываем реестр один раз
	regKey, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return d
	}
	defer func() {
		_ = regKey.Close()
	}()

	// Читаем данные из реестра.
	// ProductName больше не перезаписывает d.name безусловно, так как в Win11/Server22+ реестр врет!
	if productName, _, err := regKey.GetStringValue("ProductName"); err == nil {
		// Используем как запасной вариант, только если базовый маппинг не справился
		if strings.HasPrefix(d.name, "Windows ") && strings.Contains(d.name, fmt.Sprintf("%d.%d", major, minor)) {
			d.name = productName
		}
	}

	// Добавляем версию релиза (например, 23H2, 24H2) к названию
	if displayVer, _, err := regKey.GetStringValue("DisplayVersion"); err == nil && displayVer != "" {
		d.displayVersion = displayVer
		if !strings.Contains(strings.ToLower(d.name), strings.ToLower(displayVer)) {
			d.name = fmt.Sprintf("%s %s", d.name, displayVer)
		}
	}

	if productID, _, err := regKey.GetStringValue("ProductID"); err == nil {
		d.productID = productID
	}

	if owner, _, err := regKey.GetStringValue("RegisteredOwner"); err == nil {
		d.registeredOwner = owner
	}

	if org, _, err := regKey.GetStringValue("RegisteredOrganization"); err == nil {
		d.registeredOrg = org
	}

	if installType, _, err := regKey.GetStringValue("InstallationType"); err == nil {
		d.installationType = installType
	}

	// Получаем редакцию через GetProductInfo (с нашей исправленной валидацией ret == 0)
	d.edition = c.getEditionFromAPI(major, minor)

	// Если не удалось через API, пробуем из реестра
	if d.edition == "UNKNOWN" || d.edition == "PRODUCT_UNDEFINED" {
		if edition, _, err := regKey.GetStringValue("EditionID"); err == nil {
			d.edition = edition
		}
	}

	return d
}

// getWindowsVersionNumbers получает версию Windows через RtlGetVersion
func (c *OSCollector) getWindowsVersionNumbers() (major, minor, build uint32, isServer bool) {
	var versionInfo rtlOSVersionInfoEx
	versionInfo.OSVersionInfoSize = uint32(unsafe.Sizeof(versionInfo))

	ret, _, _ := procRtlGetVersion.Call(
		uintptr(unsafe.Pointer(&versionInfo)),
	)

	if ret == 0 {
		// VER_NT_WORKSTATION (1) — это обычная десктопная Windows
		// VER_NT_DOMAIN_CONTROLLER (2) и VER_NT_SERVER (3) — сервера
		isServer = versionInfo.ProductType != 1
		return versionInfo.MajorVersion, versionInfo.MinorVersion, versionInfo.BuildNumber, isServer
	}

	return 10, 0, 0, false
}

// getWindowsVersionName возвращает название Windows
func (c *OSCollector) getWindowsVersionName(major, minor, build uint32) string {
	// Получаем информацию о сервере
	_, _, _, isServer := c.getWindowsVersionNumbers()

	// Сначала обрабатываем главную мажорную ветку
	switch major {
	case 10:
		if minor == 0 {
			if build >= 22000 {
				// Windows 11 или Windows Server 2022+
				if isServer {
					if build >= 26100 {
						return "Windows Server 2025"
					}
					return "Windows Server 2022"
				}
				return "Windows 11"
			}
			// Windows 10 или Windows Server 2016/2019
			if isServer {
				if build >= 17763 {
					return "Windows Server 2019"
				}
				return "Windows Server 2016"
			}
			return "Windows 10"
		}
	case 6:
		switch minor {
		case 3:
			if isServer {
				return "Windows Server 2012 R2"
			}
			return "Windows 8.1"
		case 2:
			if isServer {
				return "Windows Server 2012"
			}
			return "Windows 8"
		case 1:
			if isServer {
				return "Windows Server 2008 R2"
			}
			return "Windows 7"
		case 0:
			if isServer {
				return "Windows Server 2008"
			}
			return "Windows Vista"
		}
	case 5:
		switch minor {
		case 2:
			if isServer {
				return "Windows Server 2003"
			}
			return "Windows XP x64"
		case 1:
			return "Windows XP"
		case 0:
			if isServer {
				return "Windows 2000 Server"
			}
			return "Windows 2000"
		}
	}

	// Дефолтный вариант для серверов или новых версий
	if isServer {
		return fmt.Sprintf("Windows Server %d.%d", major, minor)
	}
	return fmt.Sprintf("Windows %d.%d", major, minor)
}

// getEditionFromAPI получает редакцию через GetProductInfo
func (c *OSCollector) getEditionFromAPI(major, minor uint32) string {
	var returnedType uint32

	// Обратите внимание: GetProductInfo принимает DWORD (uint32).
	// Мы передаем 0, 0 в качестве Service Pack major/minor, как требует API для актуальных ОС.
	ret, _, _ := procGetProductInfo.Call(
		uintptr(major),
		uintptr(minor),
		0,
		0,
		uintptr(unsafe.Pointer(&returnedType)),
	)

	// Если возвращен 0 (FALSE), значит вызов функции WinAPI завершился ошибкой
	if ret == 0 {
		return "UNKNOWN"
	}

	// Дополнительная защита: если функция вернула TRUE, но тип продукта PRODUCT_UNDEFINED (0x00000000)
	if returnedType == 0 {
		return "PRODUCT_UNDEFINED"
	}

	return c.getEditionName(returnedType)
}

// getEditionName преобразует ID в читаемое название (полная версия)
func (c *OSCollector) getEditionName(productType uint32) string {
	editions := map[uint32]string{
		PRODUCT_BUSINESS:                            "Business",
		PRODUCT_BUSINESS_N:                          "Business N",
		PRODUCT_CLUSTER_SERVER:                      "Cluster Server",
		PRODUCT_CLUSTER_SERVER_V:                    "Cluster Server V",
		PRODUCT_CORE:                                "Home",
		PRODUCT_CORE_COUNTRYSPECIFIC:                "Home China",
		PRODUCT_CORE_N:                              "Home N",
		PRODUCT_CORE_SINGLELANGUAGE:                 "Home Single Language",
		PRODUCT_DATACENTER_EVALUATION_SERVER:        "Server Datacenter Evaluation",
		PRODUCT_DATACENTER_A_SERVER:                 "Server Datacenter A",
		PRODUCT_DATACENTER_SERVER:                   "Server Datacenter",
		PRODUCT_DATACENTER_SERVER_CORE:              "Server Datacenter Core",
		PRODUCT_DATACENTER_SERVER_CORE_V:            "Server Datacenter Core V",
		PRODUCT_DATACENTER_SERVER_V:                 "Server Datacenter V",
		PRODUCT_EDUCATION:                           "Education",
		PRODUCT_EDUCATION_N:                         "Education N",
		PRODUCT_ENTERPRISE:                          "Enterprise",
		PRODUCT_ENTERPRISE_E:                        "Enterprise E",
		PRODUCT_ENTERPRISE_N:                        "Enterprise N",
		PRODUCT_ENTERPRISE_N_EVALUATION:             "Enterprise N Evaluation",
		PRODUCT_ENTERPRISE_S:                        "Enterprise S",
		PRODUCT_ENTERPRISE_S_EVALUATION:             "Enterprise S Evaluation",
		PRODUCT_ENTERPRISE_S_N:                      "Enterprise S N",
		PRODUCT_ENTERPRISE_S_N_EVALUATION:           "Enterprise S N Evaluation",
		PRODUCT_ENTERPRISE_EVALUATION:               "Enterprise Evaluation",
		PRODUCT_ENTERPRISE_SERVER:                   "Server Enterprise",
		PRODUCT_ENTERPRISE_SERVER_CORE:              "Server Enterprise Core",
		PRODUCT_ENTERPRISE_SERVER_CORE_V:            "Server Enterprise Core V",
		PRODUCT_ENTERPRISE_SERVER_IA64:              "Server Enterprise IA64",
		PRODUCT_ENTERPRISE_SERVER_V:                 "Server Enterprise V",
		PRODUCT_ESSENTIALBUSINESS_SERVER_ADDL:       "Essential Business Server ADDL",
		PRODUCT_ESSENTIALBUSINESS_SERVER_ADDLSVC:    "Essential Business Server ADDLSVC",
		PRODUCT_ESSENTIALBUSINESS_SERVER_MGMT:       "Essential Business Server MGMT",
		PRODUCT_ESSENTIALBUSINESS_SERVER_MGMTSVC:    "Essential Business Server MGMTSVC",
		PRODUCT_HOME_BASIC:                          "Home Basic",
		PRODUCT_HOME_BASIC_E:                        "Home Basic E",
		PRODUCT_HOME_BASIC_N:                        "Home Basic N",
		PRODUCT_HOME_PREMIUM:                        "Home Premium",
		PRODUCT_HOME_PREMIUM_E:                      "Home Premium E",
		PRODUCT_HOME_PREMIUM_N:                      "Home Premium N",
		PRODUCT_HOME_PREMIUM_SERVER:                 "Home Premium Server",
		PRODUCT_HOME_SERVER:                         "Home Server",
		PRODUCT_MEDIUMBUSINESS_SERVER_MANAGEMENT:    "Medium Business Server Management",
		PRODUCT_MEDIUMBUSINESS_SERVER_MESSAGING:     "Medium Business Server Messaging",
		PRODUCT_MEDIUMBUSINESS_SERVER_SECURITY:      "Medium Business Server Security",
		PRODUCT_MOBILE_CORE:                         "Mobile",
		PRODUCT_MOBILE_ENTERPRISE:                   "Mobile Enterprise",
		PRODUCT_MULTIPOINT_PREMIUM_SERVER:           "MultiPoint Premium Server",
		PRODUCT_MULTIPOINT_STANDARD_SERVER:          "MultiPoint Standard Server",
		PRODUCT_PROFESSIONAL:                        "Professional",
		PRODUCT_PROFESSIONAL_E:                      "Professional E",
		PRODUCT_PROFESSIONAL_N:                      "Professional N",
		PRODUCT_PROFESSIONAL_WMC:                    "Professional with Media Center",
		PRODUCT_PROFESSIONAL_STUDENT:                "Professional Student",
		PRODUCT_PROFESSIONAL_STUDENT_N:              "Professional Student N",
		PRODUCT_PRO_CHINA:                           "Professional China",
		PRODUCT_PRO_SINGLE_LANGUAGE:                 "Professional Single Language",
		PRODUCT_PRO_WORKSTATION:                     "Professional Workstation",
		PRODUCT_PRO_WORKSTATION_N:                   "Professional Workstation N",
		PRODUCT_PRO_FOR_EDUCATION:                   "Professional for Education",
		PRODUCT_PRO_FOR_EDUCATION_N:                 "Professional for Education N",
		PRODUCT_SB_SOLUTION_SERVER:                  "SB Solution Server",
		PRODUCT_SB_SOLUTION_SERVER_EM:               "SB Solution Server EM",
		PRODUCT_SERVER_FOR_SB_SOLUTIONS:             "Server for SB Solutions",
		PRODUCT_SERVER_FOR_SB_SOLUTIONS_EM:          "Server for SB Solutions EM",
		PRODUCT_SERVER_FOUNDATION:                   "Server Foundation",
		PRODUCT_SMALLBUSINESS_SERVER:                "Small Business Server",
		PRODUCT_SMALLBUSINESS_SERVER_PREMIUM:        "Small Business Server Premium",
		PRODUCT_SMALLBUSINESS_SERVER_PREMIUM_CORE:   "Small Business Server Premium Core",
		PRODUCT_SOLUTION_EMBEDDEDSERVER:             "Solution Embedded Server",
		PRODUCT_STANDARD_EVALUATION_SERVER:          "Server Standard Evaluation",
		PRODUCT_STANDARD_A_SERVER:                   "Server Standard A",
		PRODUCT_STANDARD_SERVER:                     "Server Standard",
		PRODUCT_STANDARD_SERVER_CORE:                "Server Standard Core",
		PRODUCT_STANDARD_SERVER_CORE_V:              "Server Standard Core V",
		PRODUCT_STANDARD_SERVER_V:                   "Server Standard V",
		PRODUCT_STARTER:                             "Starter",
		PRODUCT_STARTER_E:                           "Starter E",
		PRODUCT_STARTER_N:                           "Starter N",
		PRODUCT_STORAGE_ENTERPRISE_SERVER:           "Storage Server Enterprise",
		PRODUCT_STORAGE_ENTERPRISE_SERVER_CORE:      "Storage Server Enterprise Core",
		PRODUCT_STORAGE_EXPRESS_SERVER:              "Storage Server Express",
		PRODUCT_STORAGE_EXPRESS_SERVER_CORE:         "Storage Server Express Core",
		PRODUCT_STORAGE_STANDARD_EVALUATION_SERVER:  "Storage Server Standard Evaluation",
		PRODUCT_STORAGE_STANDARD_SERVER:             "Storage Server Standard",
		PRODUCT_STORAGE_STANDARD_SERVER_CORE:        "Storage Server Standard Core",
		PRODUCT_STORAGE_WORKGROUP_EVALUATION_SERVER: "Storage Server Workgroup Evaluation",
		PRODUCT_STORAGE_WORKGROUP_SERVER:            "Storage Server Workgroup",
		PRODUCT_STORAGE_WORKGROUP_SERVER_CORE:       "Storage Server Workgroup Core",
		PRODUCT_ULTIMATE:                            "Ultimate",
		PRODUCT_ULTIMATE_E:                          "Ultimate E",
		PRODUCT_ULTIMATE_N:                          "Ultimate N",
		PRODUCT_WEB_SERVER:                          "Web Server",
		PRODUCT_WEB_SERVER_CORE:                     "Web Server Core",
		PRODUCT_UNLICENSED:                          "Unlicensed",
		PRODUCT_AZURE_SERVER_CORE:                   "Azure Server Core",
		PRODUCT_AZURE_NANO_SERVER:                   "Azure Nano Server",
		PRODUCT_ENTERPRISE_FOR_VIRTUAL_DESKTOPS:     "Enterprise for Virtual Desktops",
		PRODUCT_IOBALANCE_SERVER:                    "IO Balance Server",
		PRODUCT_MULTI_SESSION:                       "Multi Session",
		PRODUCT_CLOUD_HOST_INFRASTRUCTURE_SERVER:    "Cloud Host Infrastructure Server",
		PRODUCT_CLOUD_STORAGE_SERVER:                "Cloud Storage Server",
		PRODUCT_SERVER_V:                            "Server V",
		PRODUCT_SERVER:                              "Server",
		PRODUCT_WINDOWS_10_ENTERPRISE:               "Windows 10 Enterprise",
		PRODUCT_WINDOWS_10_EDUCATION:                "Windows 10 Education",
		PRODUCT_WINDOWS_10_PRO:                      "Windows 10 Professional",
		PRODUCT_WINDOWS_10_HOME:                     "Windows 10 Home",
		PRODUCT_WINDOWS_10_TEAM:                     "Windows 10 Team",
		PRODUCT_WINDOWS_10_PRO_WORKSTATION:          "Windows 10 Pro Workstation",
		PRODUCT_WINDOWS_10_PRO_EDUCATION:            "Windows 10 Pro Education",
		PRODUCT_WINDOWS_10_S:                        "Windows 10 S",
		PRODUCT_WINDOWS_10_ENTERPRISE_N:             "Windows 10 Enterprise N",
		PRODUCT_WINDOWS_10_EDUCATION_N:              "Windows 10 Education N",
		PRODUCT_WINDOWS_10_PRO_N:                    "Windows 10 Professional N",
		PRODUCT_WINDOWS_10_HOME_N:                   "Windows 10 Home N",
		PRODUCT_WINDOWS_10_TEAM_N:                   "Windows 10 Team N",
		PRODUCT_WINDOWS_10_PRO_WORKSTATION_N:        "Windows 10 Pro Workstation N",
		PRODUCT_WINDOWS_10_PRO_EDUCATION_N:          "Windows 10 Pro Education N",
		PRODUCT_WINDOWS_10_S_N:                      "Windows 10 S N",
		PRODUCT_WINDOWS_10_ENTERPRISE_S:             "Windows 10 Enterprise S",
		PRODUCT_WINDOWS_10_EDUCATION_S:              "Windows 10 Education S",
		PRODUCT_WINDOWS_10_PRO_S:                    "Windows 10 Professional S",
		PRODUCT_WINDOWS_10_HOME_S:                   "Windows 10 Home S",
		PRODUCT_WINDOWS_10_ENTERPRISE_S_N:           "Windows 10 Enterprise S N",
		PRODUCT_WINDOWS_10_EDUCATION_S_N:            "Windows 10 Education S N",
		PRODUCT_WINDOWS_10_PRO_S_N:                  "Windows 10 Professional S N",
		PRODUCT_WINDOWS_10_HOME_S_N:                 "Windows 10 Home S N",
		PRODUCT_WINDOWS_10_ENTERPRISE_EVAL:          "Windows 10 Enterprise Evaluation",
		PRODUCT_WINDOWS_10_EDUCATION_EVAL:           "Windows 10 Education Evaluation",
		PRODUCT_WINDOWS_10_PRO_EVAL:                 "Windows 10 Professional Evaluation",
		PRODUCT_WINDOWS_10_HOME_EVAL:                "Windows 10 Home Evaluation",
		PRODUCT_WINDOWS_10_ENTERPRISE_N_EVAL:        "Windows 10 Enterprise N Evaluation",
		PRODUCT_WINDOWS_10_EDUCATION_N_EVAL:         "Windows 10 Education N Evaluation",
		PRODUCT_WINDOWS_10_PRO_N_EVAL:               "Windows 10 Professional N Evaluation",
		PRODUCT_WINDOWS_10_HOME_N_EVAL:              "Windows 10 Home N Evaluation",
	}

	if name, exists := editions[productType]; exists {
		return name
	}

	return fmt.Sprintf("Edition 0x%X", productType)
}

// getSystemLocale вычитывает код языка установки ОС напрямую через WinAPI
func (c *OSCollector) getSystemLocale() string {
	// Максимальная длина имени локали в Windows (LOCALE_NAME_MAX_LENGTH = 85)
	const localeNameMaxLength = 85
	buf := make([]uint16, localeNameMaxLength)

	ret, _, _ := procGetSystemDefaultLocaleName.Call(
		uintptr(unsafe.Pointer(unsafe.SliceData(buf))),
		uintptr(localeNameMaxLength),
	)

	// Если функция вернула длину строки (> 0), преобразуем UTF-16 в Go-строку
	if ret > 0 {
		return syscall.UTF16ToString(buf)
	}

	// Резервный вариант, если WinAPI сбойнул
	return "UNKNOWN"
}

// getInstallDate возвращает дату установки Windows как Unix timestamp
func (c *OSCollector) getInstallDate() int64 {
	// Список ключей от самого надежного до текущего
	paths := []string{
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion\Source OS (Updated on)`,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion\Source OS`,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
	}

	for _, path := range paths {
		if val := c.readInstallTimestamp(path); val > 0 {
			return int64(val)
		}
	}

	return 0
}

// readInstallTimestamp читает timestamp из реестра
func (c *OSCollector) readInstallTimestamp(path string) uint64 {
	regKey, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err != nil {
		return 0
	}
	defer func() {
		_ = regKey.Close()
	}()

	val, _, err := regKey.GetIntegerValue("InstallDate")
	if err != nil {
		return 0
	}
	return val
}

// getPowerShellVersion возвращает версию PowerShell (сначала ищет Core 7+, затем 3+, затем 1-2)
func (c *OSCollector) getPowerShellVersion() string {
	// 1. Проверяем современный PowerShell 6/7+ (Core)
	if ver := c.readPSVersion(`SOFTWARE\Microsoft\PowerShellCore\InstalledVersions`, "SemanticVersion"); ver != "" {
		return "Core " + ver
	}

	// 2. Проверяем встроенный Windows PowerShell 3.0 - 5.1
	if ver := c.readPSVersion(`SOFTWARE\Microsoft\PowerShell\3\PowerShellEngine`, "PowerShellVersion"); ver != "" {
		return ver
	}

	// 3. Проверяем старый встроенный Windows PowerShell 1.0 - 2.0
	if ver := c.readPSVersion(`SOFTWARE\Microsoft\PowerShell\1\PowerShellEngine`, "PowerShellVersion"); ver != "" {
		return ver
	}

	return "UNKNOWN"
}

// Вспомогательный неэкспортируемый метод для чистоты кода и защиты от дублирования defer
func (c *OSCollector) readPSVersion(path, valueName string) string {
	regKey, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer func() {
		_ = regKey.Close()
	}()

	ver, _, err := regKey.GetStringValue(valueName)
	if err != nil {
		return ""
	}
	return ver
}

// isSecureBootEnabled проверяет статус Secure Boot напрямую через UEFI
func (c *OSCollector) isSecureBootEnabled() bool {
	// Имя переменной UEFI для SecureBoot
	namePtr, err := syscall.UTF16PtrFromString("SecureBoot")
	if err != nil {
		return false
	}

	// GUID пространства имен UEFI для глобальных переменных
	guidPtr, err := syscall.UTF16PtrFromString("{8be4df61-93ca-11d2-aa0d-00e098032b8c}")
	if err != nil {
		return false
	}

	// Буфер для ответа (SecureBoot возвращает 1 байт: 1 - включен, 0 - выключен)
	var buffer byte

	ret, _, _ := procGetFirmwareEnvironmentVariable.Call(
		uintptr(unsafe.Pointer(namePtr)),
		uintptr(unsafe.Pointer(guidPtr)),
		uintptr(unsafe.Pointer(&buffer)),
		1, // Размер буфера в байтах
	)

	// Если функция вернула больше 0, значит переменная успешно прочитана
	if ret > 0 {
		return buffer == 1
	}

	// Резервный вариант: если к UEFI нет прямого доступа, откатываемся на ваш проверенный код с реестром
	return c.isSecureBootEnabledFromRegistry()
}

// Ваш текущий код как резервный метод
func (c *OSCollector) isSecureBootEnabledFromRegistry() bool {
	regKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\SecureBoot\State`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer func() {
		_ = regKey.Close()
	}()

	val, _, err := regKey.GetIntegerValue("UEFISecureBootEnabled")
	if err != nil {
		return false
	}
	return val == 1
}

// getMachineGUID возвращает уникальный UUID ОС
func (c *OSCollector) getMachineGUID() models.BinaryUUID {
	regKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE)
	if err != nil {
		return models.BinaryUUID{} // Идиоматичный возврат нулевого значения
	}
	defer func() {
		_ = regKey.Close()
	}()

	guidStr, _, err := regKey.GetStringValue("MachineGuid")
	if err != nil {
		return models.BinaryUUID{}
	}

	parsed, err := uuid.Parse(guidStr)
	if err != nil {
		return models.BinaryUUID{}
	}

	return models.BinaryUUID(parsed)
}

// checkVirtualization определяет, запущена ли ОС на виртуальной машине
func (c *OSCollector) checkVirtualization() bool {
	regKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DESCRIPTION\System`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer func() {
		_ = regKey.Close()
	}()

	// 1. Проверяем флаг гипервизора от процессора/ОС
	if val, _, err := regKey.GetIntegerValue("HypervisorPresent"); err == nil && val == 1 {
		return true
	}

	// 2. Исправлено: SystemBiosVersion — это REG_MULTI_SZ, читаем как срез строк
	biosVersions, _, err := regKey.GetStringsValue("SystemBiosVersion")
	if err != nil {
		return false
	}

	// Оптимизация: индикаторы сразу в нижнем регистре, чтобы не тратить ресурсы в цикле
	vmIndicators := []string{
		"vmware", "virtualbox", "qemu", "xen",
		"hyper-v", "kvm", "parallels", "virtual machine",
	}

	// Проверяем каждую строку из биоса на наличие совпадений
	for _, biosVendor := range biosVersions {
		lowerVendor := strings.ToLower(biosVendor)
		for _, indicator := range vmIndicators {
			if strings.Contains(lowerVendor, indicator) {
				return true
			}
		}
	}

	return false
}

// checkHypervisorHost проверяет, работает ли система как хост-гипервизор (Hyper-V)
func (c *OSCollector) checkHypervisorHost() bool {
	// Метод 1: Проверяем реестр (самый простой и надежный)
	if c.checkHyperVRegistry() {
		return true
	}

	return false
}

// checkHyperVRegistry проверяет ключи реестра Hyper-V
func (c *OSCollector) checkHyperVRegistry() bool {
	regKey, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion\Virtualization`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return false
	}
	defer func() {
		_ = regKey.Close()
	}()

	// Проверяем целочисленные значения
	valueNames := []string{
		"HyperVisorPresent",
		"HypervisorPresent",
		"Enabled",
		"IsHyperVPresent",
		"VirtualizationEnabled",
	}

	for _, name := range valueNames {
		if val, _, err := regKey.GetIntegerValue(name); err == nil && val == 1 {
			return true
		}
	}

	// Дополнительно проверяем строковые значения
	stringValueNames := []string{
		"State",
		"Status",
		"HyperVState",
	}

	for _, name := range stringValueNames {
		if val, _, err := regKey.GetStringValue(name); err == nil {
			// Линтер одобрит tagged switch
			switch val {
			case "Enabled", "Running", "Active", "1", "True":
				return true
			}
		}
	}

	// Проверяем наличие под-ключей
	subKeys := []string{
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion\Virtualization\Workspaces`,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion\Virtualization\GuestCommunicationServices`,
	}

	for _, subKey := range subKeys {
		if hasKey := c.checkKeyExists(subKey); hasKey {
			return true
		}
	}

	return false
}

// Вспомогательный метод для безопасной и изолированной проверки существования ключа
func (c *OSCollector) checkKeyExists(path string) bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer func() {
		_ = k.Close()
	}()
	return true
}
