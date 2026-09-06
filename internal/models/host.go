package models

import (
	"fmt"
	"math"
	"time"
)

// HostInfo contains information about the host (computer)
type HostInfo struct {
	Hostname         string `json:"hostname"`          // DNS hostname of the computer
	FQDN             string `json:"fqdn"`              // Fully qualified domain name
	PhysicalHostname string `json:"physical_hostname"` // Physical machine name (for VMs)
	PhysicalFQDN     string `json:"physical_fqdn"`     // Physical fully qualified domain name (for VMs)
	Domain           string `json:"domain"`            // Domain name
	Workgroup        string `json:"workgroup"`         // Workgroup name (if not in a domain)
	UpTimeSeconds    uint64 `json:"uptime_seconds"`    // System uptime in seconds
	BootTime         uint64 `json:"boot_time"`         // Last boot time as Unix timestamp
	TimeZone         string `json:"timezone"`          // Time zone name
	TimeZoneOffset   int16  `json:"timezone_offset"`   // Time zone offset in minutes from UTC

	Manufacturer string `json:"manufacturer"`  // System manufacturer (e.g., "Dell Inc.")
	Model        string `json:"model"`         // System model (e.g., "Precision T3610")
	SKU          string `json:"sku"`           // Stock keeping unit
	Family       string `json:"family"`        // System family
	Version      string `json:"version"`       // System version
	SerialNumber string `json:"serial_number"` // System serial number

	BIOSVendor       string `json:"bios_vendor"`        // BIOS vendor
	BIOSVersion      string `json:"bios_version"`       // BIOS version
	BIOSDate         string `json:"bios_date"`          // BIOS release date
	BIOSMajorRelease uint32 `json:"bios_major_release"` // BIOS major release number
	BIOSMinorRelease uint32 `json:"bios_minor_release"` // BIOS minor release number

	BaseBoardManufacturer string `json:"baseboard_manufacturer"` // Motherboard manufacturer
	BaseBoardProduct      string `json:"baseboard_product"`      // Motherboard model
	BaseBoardVersion      string `json:"baseboard_version"`      // Motherboard version
}

// GetBootTime returns the boot time as time.Time
func (h *HostInfo) GetBootTime() time.Time {
	if h.BootTime == 0 {
		return time.Time{}
	}

	// #nosec G115 -- BootTime is always a valid Unix timestamp from Windows
	return time.Unix(int64(h.BootTime), 0)
}

// GetBootTimeString returns the boot time in "YYYY-MM-DD HH:MM:SS" format
func (h *HostInfo) GetBootTimeString() string {
	if h.BootTime == 0 {
		return UNKNOWN
	}

	// #nosec G115 -- BootTime is always a valid Unix timestamp from Windows
	return time.Unix(int64(h.BootTime), 0).Format("2006-01-02 15:04:05")
}

// GetUpTime returns the uptime as time.Duration
func (h *HostInfo) GetUpTime() time.Duration {
	const maxSeconds = uint64(math.MaxInt64 / int64(time.Second))

	if h.UpTimeSeconds > maxSeconds {
		// Uptime exceeds maximum duration, cap it
		return time.Duration(math.MaxInt64)
	}

	return time.Duration(h.UpTimeSeconds) * time.Second
}

// GetUpTimeString returns the uptime in human-readable format
func (h *HostInfo) GetUpTimeString() string {
	duration := h.GetUpTime()

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%d days, %d hours, %d minutes", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%d hours, %d minutes", hours, minutes)
	}
	return fmt.Sprintf("%d minutes", minutes)
}
