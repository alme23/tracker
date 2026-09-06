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

// Constants for GetComputerNameEx
const (
	computerNameNetBIOS                   = 0
	computerNameDNSHostname               = 1
	computerNameDNSDomain                 = 2
	computerNameDNSFullyQualified         = 3
	computerNamePhysicalNetBIOS           = 4
	computerNamePhysicalDNSHostname       = 5
	computerNamePhysicalDNSDomain         = 6
	computerNamePhysicalDNSFullyQualified = 7
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

// HostCollector collects information about the host computer
type HostCollector struct{}

// NewHostCollector creates a new HostCollector
func NewHostCollector() *HostCollector {
	return &HostCollector{}
}

// Collect gathers information about the host
func (c *HostCollector) Collect() (models.HostInfo, error) {
	info := models.HostInfo{}

	// 1. DNS hostname
	info.Hostname = c.getComputerName(computerNameDNSHostname)
	if info.Hostname == "" {
		info.Hostname, _ = os.Hostname()
	}

	// 2. Fully qualified domain name (FQDN)
	info.FQDN = c.getComputerName(computerNameDNSFullyQualified)

	// 3. Physical hostname
	info.PhysicalHostname = c.getComputerName(computerNamePhysicalDNSHostname)

	// 4. Physical FQDN
	info.PhysicalFQDN = c.getComputerName(computerNamePhysicalDNSFullyQualified)

	// 5. Domain
	info.Domain = c.getComputerName(computerNameDNSDomain)

	// 6. Workgroup (if not in a domain)
	if info.Domain == "" {
		info.Workgroup = c.getWorkgroup()
	}

	// 7. Uptime and boot time
	info.UpTimeSeconds = c.getUpTimeSeconds()

	now := time.Now().Unix()
	if now < 0 {
		// System clock is before 1970 (unlikely), set boot time to 0
		info.BootTime = 0
	} else {
		currentTime := uint64(now)
		if info.UpTimeSeconds < currentTime {
			info.BootTime = currentTime - info.UpTimeSeconds
		} else {
			// If uptime exceeds current time (unlikely), set to 0
			info.BootTime = 0
		}
	}

	// 8. Time zone
	info.TimeZone, info.TimeZoneOffset = c.getTimeZone()

	// 9. Hardware info (including BIOS and motherboard)
	c.collectHardwareInfo(&info)

	return info, nil
}

// getComputerName returns the computer name of the specified type
func (c *HostCollector) getComputerName(nameType uint32) string {
	// Size variable must be declared inside the method because WinAPI overwrites it
	var size uint32 = 256
	buffer := make([]uint16, size)

	// #nosec G103 -- safe use of unsafe.SliceData with local buffer
	ret, _, _ := procGetComputerNameEx.Call(
		uintptr(nameType),
		uintptr(unsafe.Pointer(unsafe.SliceData(buffer))),
		uintptr(unsafe.Pointer(&size)),
	)

	if ret == 0 || size == 0 {
		return ""
	}

	// Read only up to the size returned by Windows
	return syscall.UTF16ToString(buffer[:size])
}

// getWorkgroup returns the workgroup name
func (c *HostCollector) getWorkgroup() string {
	// 1. Try to read from LanmanWorkstation parameters
	netKey, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Services\LanmanWorkstation\Parameters`,
		registry.QUERY_VALUE,
	)
	if err == nil {
		defer func() {
			_ = netKey.Close()
		}()
		if wg, _, err := netKey.GetStringValue("Domain"); err == nil && wg != "" {
			return wg
		}
	}

	// 2. Fallback: try Tcpip parameters
	tcpKey, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Services\Tcpip\Parameters`,
		registry.QUERY_VALUE,
	)
	if err == nil {
		defer func() {
			_ = tcpKey.Close()
		}()
		if wg, _, err := tcpKey.GetStringValue("Workgroup"); err == nil && wg != "" {
			return wg
		}
	}

	// 3. Default fallback for Windows local machines
	return "WORKGROUP"
}

// getUpTimeSeconds returns the system uptime in seconds
func (c *HostCollector) getUpTimeSeconds() uint64 {
	ret, _, _ := procGetTickCount64.Call()
	if ret == 0 {
		return 0
	}

	// GetTickCount64 returns milliseconds
	return uint64(ret) / 1000
}

// getTimeZone returns the time zone name and offset in minutes
func (c *HostCollector) getTimeZone() (name string, offsetMinutes int16) {
	var tzInfo timeZoneInformation

	// #nosec G103 -- safe use of unsafe.Pointer with local struct
	ret, _, _ := procGetTimeZoneInformation.Call(
		uintptr(unsafe.Pointer(&tzInfo)),
	)

	// TIME_ZONE_ID_INVALID
	const timeZoneIDInvalid = 0xFFFFFFFF

	// #nosec G115 -- WinAPI returns DWORD (uint32), safe to convert
	if uint32(ret) == timeZoneIDInvalid {
		return "UTC", 0
	}

	currentBias := tzInfo.Bias
	var zoneName string

	switch ret {
	case 2: // TIME_ZONE_ID_DAYLIGHT
		currentBias += tzInfo.DaylightBias
		zoneName = syscall.UTF16ToString(tzInfo.DaylightName[:])

	case 1: // TIME_ZONE_ID_STANDARD
		currentBias += tzInfo.StandardBias
		zoneName = syscall.UTF16ToString(tzInfo.StandardName[:])

	default: // TIME_ZONE_ID_UNKNOWN (0) or other states
		zoneName = syscall.UTF16ToString(tzInfo.StandardName[:])
	}

	// #nosec G115 -- Time zone offset is always within int16 range (UTC-12 to UTC+14 = -720 to 840 minutes)
	offsetMinutes = -int16(currentBias)
	return zoneName, offsetMinutes
}

// collectHardwareInfo collects hardware information from BIOS
func (c *HostCollector) collectHardwareInfo(info *models.HostInfo) {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\BIOS`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return
	}
	defer func() {
		_ = k.Close()
	}()

	// System information
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

	// BIOS information
	if biosVendor, _, err := k.GetStringValue("BIOSVendor"); err == nil {
		info.BIOSVendor = biosVendor
	}

	if biosVersion, _, err := k.GetStringValue("BIOSVersion"); err == nil {
		info.BIOSVersion = biosVersion
	}

	if biosDate, _, err := k.GetStringValue("BIOSReleaseDate"); err == nil {
		info.BIOSDate = biosDate
	}

	// #nosec G115 -- BIOS major release is a small number (typically 0-255)
	if majorRelease, _, err := k.GetIntegerValue("BiosMajorRelease"); err == nil {
		info.BIOSMajorRelease = uint32(majorRelease)
	}

	// #nosec G115 -- BIOS minor release is a small number (typically 0-255)
	if minorRelease, _, err := k.GetIntegerValue("BiosMinorRelease"); err == nil {
		info.BIOSMinorRelease = uint32(minorRelease)
	}

	// Motherboard information
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
