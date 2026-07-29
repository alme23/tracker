//go:build windows

package collector

import (
	"strings"

	"golang.org/x/sys/windows/registry"
)

func GetBaseBoardInfo() (BaseBoardInfo, error) {
	var info BaseBoardInfo
	biosKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DESCRIPTION\System\BIOS`, registry.QUERY_VALUE)
	if err == nil {
		defer func() { _ = biosKey.Close() }()

		vendor, _, _ := biosKey.GetStringValue("BaseBoardVendor")
		info.Vendor = strings.TrimSpace(vendor)

		product, _, _ := biosKey.GetStringValue("BaseBoardProduct")
		info.Product = strings.TrimSpace(product)

		version, _, _ := biosKey.GetStringValue("BaseBoardVersion")
		info.Version = strings.TrimSpace(version)

		sn, _, _ := biosKey.GetStringValue("BaseBoardSerialNumber")
		if sn == "" {
			sn, _, _ = biosKey.GetStringValue("SystemSerialNumber")
		}
		info.SerialNumber = strings.TrimSpace(sn)
	}

	if info.Vendor == "" {
		info.Vendor = "Unknown Vendor"
	}
	if info.Product == "" {
		info.Product = "Unknown Product"
	}
	if info.Version == "" {
		info.Version = "Unknown Version"
	}
	if info.SerialNumber == "" {
		info.SerialNumber = "Unknown/To be filled by O.E.M."
	}
	return info, nil
}
