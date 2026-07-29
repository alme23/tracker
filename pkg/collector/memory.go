//go:build windows

package collector

import (
	"fmt"
	"unsafe"
)

func GetMemoryStats() (HardwareInfo, error) {
	var info HardwareInfo
	var memStatus MEMORYSTATUSEX
	memStatus.Length = uint32(unsafe.Sizeof(memStatus))

	ret, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memStatus)))
	if ret == 0 {
		return info, fmt.Errorf("GlobalMemoryStatusEx failed")
	}

	info.TotalRAMBytes = memStatus.TotalPhys
	info.FreeRAMBytes = memStatus.AvailPhys
	return info, nil
}
