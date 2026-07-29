//go:build windows

package collector

import (
	"strings"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

func GetCPUInfo() (CPUInfo, error) {
	var info CPUInfo

	defer func() {
		if r := recover(); r != nil {
			if info.PhysicalCores == 0 {
				info.PhysicalCores = info.LogicalProcessors
			}
		}
	}()

	var sysInfo SYSTEM_INFO
	_, _, _ = procGetSystemInfo.Call(uintptr(unsafe.Pointer(&sysInfo)))
	info.LogicalProcessors = sysInfo.NumberOfProcessors

	cpuKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DESCRIPTION\System\CentralProcessor\0`, registry.QUERY_VALUE)
	if err == nil {
		name, _, err := cpuKey.GetStringValue("ProcessorNameString")
		if err == nil {
			info.Name = strings.TrimSpace(name)
		}
		speed, _, err := cpuKey.GetIntegerValue("~MHz")
		if err == nil {
			info.MaxSpeedMHz = uint32(speed)
		}
		_ = cpuKey.Close()
	}
	if info.Name == "" {
		info.Name = "Unknown CPU"
	}

	var returnedLength uint32 = 0
	_, _, _ = procGetLogicalProcessorInformationEx.Call(uintptr(RelationProcessorCore|RelationCache), uintptr(0), uintptr(unsafe.Pointer(&returnedLength)))

	if returnedLength > 0 {
		buffer := make([]byte, returnedLength)
		ret, _, _ := procGetLogicalProcessorInformationEx.Call(uintptr(RelationProcessorCore|RelationCache), uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&returnedLength)))

		if ret != 0 {
			offset := uint32(0)
			for offset < returnedLength {
				if offset+8 > returnedLength {
					break
				}

				infoEx := (*SYSTEM_LOGICAL_PROCESSOR_INFORMATION_EX)(unsafe.Pointer(&buffer[offset]))
				if infoEx.Size == 0 || offset+infoEx.Size > returnedLength {
					break
				}

				switch infoEx.Relationship {
				case RelationProcessorCore:
					info.PhysicalCores++
				case RelationCache:
					// Проверяем, что Payload физически существует в рамках этой подструктуры (размер больше заголовка)
					if infoEx.Size > 8 {
						// ИСПРАВЛЕНО: Смещение +8 и приведение типов объединены в одно выражение.
						// Линтер больше не будет ругаться, а сборщик мусора Go отработает корректно.
						cache := (*CACHE_DESCRIPTOR)(unsafe.Pointer(uintptr(unsafe.Pointer(infoEx)) + 8))

						switch cache.Level {
						case 1:
							info.CacheL1Bytes += uint64(cache.Size)
						case 2:
							info.CacheL2Bytes += uint64(cache.Size)
						case 3:
							info.CacheL3Bytes += uint64(cache.Size)
						}
					}
				}
				offset += infoEx.Size
			}
		}
	}

	if info.PhysicalCores == 0 {
		info.PhysicalCores = info.LogicalProcessors
	}
	return info, nil
}
