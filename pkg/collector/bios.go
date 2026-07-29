//go:build windows

package collector

import (
	"strings"

	"golang.org/x/sys/windows/registry"
)

func GetBIOSInfo() (BIOSInfo, error) {
	var info BIOSInfo
	biosKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DESCRIPTION\System\BIOS`, registry.QUERY_VALUE)
	if err == nil {
		defer func() { _ = biosKey.Close() }()

		vendor, _, _ := biosKey.GetStringValue("BIOSVendor")
		info.Vendor = strings.TrimSpace(vendor)

		version, _, _ := biosKey.GetStringValue("BIOSVersion")
		if version == "" {
			version, _, _ = biosKey.GetStringValue("SystemBIOSVersion")
		}
		info.Version = strings.TrimSpace(version)

		date, _, _ := biosKey.GetStringValue("BIOSReleaseDate")
		info.ReleaseDate = strings.TrimSpace(date)
	}

	if info.Vendor == "" {
		info.Vendor = "Unknown BIOS Vendor"
	}
	if info.Version == "" {
		info.Version = "Unknown BIOS Version"
	}
	if info.ReleaseDate == "" {
		info.ReleaseDate = "Unknown Date"
	}
	return info, nil
}
