// tracker/internal/collector/host.go

//go:build windows

package collector

import (
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/alme23/tracker/internal/models"
	"golang.org/x/sys/windows/registry"
)

// Константы для GetComputerNameEx
const (
	computerNameNetBIOS                   = 0
	computerNameDnsHostname               = 1
	computerNameDnsDomain                 = 2
	computerNameDnsFullyQualified         = 3
	computerNamePhysicalNetBIOS           = 4
	computerNamePhysicalDnsHostname       = 5
	computerNamePhysicalDnsDomain         = 6
	computerNamePhysicalDnsFullyQualified = 7
)

type timeZoneInformation struct {
	Bias         int32
	StandardName [32]uint16
	StandardDate syscall.Systemtime
	StandardBias int32
	DaylightName [32]uint16
	DaylightDate syscall.Systemtime
	DaylightBias int32
}

type HostCollector struct{}

func NewHostCollector() *HostCollector {
	return &HostCollector{}
}

// Collect собирает информацию о хосте
func (c *HostCollector) Collect() (models.HostInfo, error) {
	info := models.HostInfo{}

	// 1. DNS имя компьютера
	info.Hostname = c.getComputerName(computerNameDnsHostname)
	if info.Hostname == "" {
		info.Hostname, _ = os.Hostname()
	}

	// 2. Полное DNS имя (FQDN)
	info.FQDN = c.getComputerName(computerNameDnsFullyQualified)

	// 3. Физическое имя
	info.PhysicalHostname = c.getComputerName(computerNamePhysicalDnsHostname)

	// 4. Полное физическое имя
	info.PhysicalFQDN = c.getComputerName(computerNamePhysicalDnsFullyQualified)

	// 5. Домен
	info.Domain = c.getComputerName(computerNameDnsDomain)

	// 6. Рабочая группа (если не в домене)
	if info.Domain == "" {
		info.Workgroup = c.getWorkgroup()
	}

	// 7. Аптайм
	info.UpTimeSeconds = c.getUpTimeSeconds()
	info.BootTime = time.Now().Add(-time.Duration(info.UpTimeSeconds) * time.Second)

	// 8. Часовой пояс
	info.TimeZone, info.TimeZoneOffset = c.getTimeZone()

	// 9. Информация о железе (включая BIOS и материнскую плату)
	c.collectHardwareInfo(&info)

	return info, nil
}

// getComputerName получает имя компьютера
func (c *HostCollector) getComputerName(nameType uint32) string {
	var size uint32 = 256
	buffer := make([]uint16, size)

	ret, _, _ := procGetComputerNameEx.Call(
		uintptr(nameType),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&size)),
	)

	if ret == 0 {
		return ""
	}

	return syscall.UTF16ToString(buffer[:size])
}

// getWorkgroup получает рабочую группу
func (c *HostCollector) getWorkgroup() string {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Services\Tcpip\Parameters`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return ""
	}
	defer k.Close()

	if workgroup, _, err := k.GetStringValue("Domain"); err == nil {
		return workgroup
	}

	return ""
}

// getUpTimeSeconds получает время работы в секундах
func (c *HostCollector) getUpTimeSeconds() uint64 {
	ret, _, _ := procGetTickCount64.Call()
	if ret == 0 {
		return 0
	}

	// GetTickCount64 возвращает миллисекунды
	return uint64(ret) / 1000
}

// getTimeZone получает информацию о часовом поясе
func (c *HostCollector) getTimeZone() (string, int16) {
	var tzInfo timeZoneInformation

	ret, _, _ := procGetTimeZoneInformation.Call(
		uintptr(unsafe.Pointer(&tzInfo)),
	)

	if ret == 0xFFFFFFFF { // TIME_ZONE_ID_INVALID
		return "UTC", 0
	}

	// Bias - это смещение в минутах (UTC = local + bias)
	offsetMinutes := -int16(tzInfo.Bias)

	zoneName := ""
	if ret == 2 { // TIME_ZONE_ID_DAYLIGHT
		zoneName = syscall.UTF16ToString(tzInfo.DaylightName[:])
	} else {
		zoneName = syscall.UTF16ToString(tzInfo.StandardName[:])
	}

	return zoneName, offsetMinutes
}

// collectHardwareInfo собирает информацию о железе из BIOS
func (c *HostCollector) collectHardwareInfo(info *models.HostInfo) {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\BIOS`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return
	}
	defer k.Close()

	// Системная информация
	if manufacturer, _, err := k.GetStringValue("SystemManufacturer"); err == nil {
		info.Manufacturer = manufacturer
	}

	if model, _, err := k.GetStringValue("SystemProductName"); err == nil {
		info.Model = model
	}

	if sku, _, err := k.GetStringValue("SystemSKU"); err == nil {
		info.SKU = sku
	}

	if family, _, err := k.GetStringValue("SystemFamily"); err == nil {
		info.Family = family
	}

	if version, _, err := k.GetStringValue("SystemVersion"); err == nil {
		info.Version = version
	}

	if serial, _, err := k.GetStringValue("SystemSerialNumber"); err == nil {
		info.SerialNumber = serial
	}

	// BIOS информация
	if biosVendor, _, err := k.GetStringValue("BIOSVendor"); err == nil {
		info.BIOSVendor = biosVendor
	}

	if biosVersion, _, err := k.GetStringValue("BIOSVersion"); err == nil {
		info.BIOSVersion = biosVersion
	}

	if biosDate, _, err := k.GetStringValue("BIOSReleaseDate"); err == nil {
		info.BIOSDate = biosDate
	}

	if majorRelease, _, err := k.GetIntegerValue("BiosMajorRelease"); err == nil {
		info.BIOSMajorRelease = uint32(majorRelease)
	}

	if minorRelease, _, err := k.GetIntegerValue("BiosMinorRelease"); err == nil {
		info.BIOSMinorRelease = uint32(minorRelease)
	}

	// Информация о материнской плате
	if boardManufacturer, _, err := k.GetStringValue("BaseBoardManufacturer"); err == nil {
		info.BaseBoardManufacturer = boardManufacturer
	}

	if boardProduct, _, err := k.GetStringValue("BaseBoardProduct"); err == nil {
		info.BaseBoardProduct = boardProduct
	}

	if boardVersion, _, err := k.GetStringValue("BaseBoardVersion"); err == nil {
		info.BaseBoardVersion = boardVersion
	}
}
