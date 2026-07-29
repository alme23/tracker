//go:build windows

package collector

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

// GetDiskStats обходит логические диски Windows и игнорирует сетевые накопители
func GetDiskStats() ([]DiskInfo, error) {
	disks := make([]DiskInfo, 0)
	var bufferSize uint32 = 256
	var buf [256]uint16

	ret, _, _ := procGetLogicalDriveStringsW.Call(uintptr(bufferSize), uintptr(unsafe.Pointer(&buf[0])))
	if ret == 0 {
		return nil, fmt.Errorf("GetLogicalDriveStringsW failed")
	}

	var currentDrive []uint16

	for i := 0; i < len(buf); i++ {
		char := buf[i]
		if char == 0 {
			if len(currentDrive) == 0 {
				break
			}
			driveStr := syscall.UTF16ToString(currentDrive)
			currentDrive = currentDrive[:0]

			drivePtr, _ := syscall.UTF16PtrFromString(driveStr)
			dtRet, _, _ := procGetDriveTypeW.Call(uintptr(unsafe.Pointer(drivePtr)))

			// ИСПРАВЛЕНО: Полностью игнорируем сетевые диски (DRIVE_REMOTE = 4)
			// Также можно игнорировать CD-ROM (DRIVE_CDROM = 5), если они не нужны
			if dtRet == 4 {
				continue
			}

			diskType := "HDD"
			switch dtRet {
			case 2:
				diskType = "USB/Removable"
			case 5:
				diskType = "CDROM"
			}

			var freeBytes, totalBytes, totalFreeBytes uint64
			diskRet, _, _ := procGetDiskFreeSpaceExW.Call(
				uintptr(unsafe.Pointer(drivePtr)),
				uintptr(unsafe.Pointer(&freeBytes)),
				uintptr(unsafe.Pointer(&totalBytes)),
				uintptr(unsafe.Pointer(&totalFreeBytes)),
			)

			if diskRet != 0 {
				vendor, sn, hwType, err := getPhysicalDiskInfo(driveStr)
				if err != nil {
					vendor = "Unknown Vendor (Requires Admin)"
					sn = "Unknown S/N (Requires Admin)"
				} else if dtRet == 3 && hwType != "" { // 3 = DRIVE_FIXED
					diskType = hwType
				}

				disks = append(disks, DiskInfo{
					Drive:        driveStr,
					Type:         diskType,
					Vendor:       vendor,
					SerialNumber: sn,
					TotalBytes:   totalBytes,
					FreeBytes:    totalFreeBytes,
				})
			}
		} else {
			currentDrive = append(currentDrive, char)
		}
	}
	return disks, nil
}

func getPhysicalDiskInfo(driveStr string) (vendor string, sn string, hwType string, err error) {
	vendor, sn, hwType = "Generic", "Unknown", "HDD"
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("fault: %v", r)
		}
	}()

	volumeName := "\\\\.\\" + strings.TrimSuffix(driveStr, "\\")
	volPtr, err := syscall.UTF16PtrFromString(volumeName)
	if err != nil {
		return
	}

	hVolume, err := syscall.CreateFile(volPtr, GENERIC_READ|GENERIC_WRITE, FILE_SHARE_READ|FILE_SHARE_WRITE, nil, OPEN_EXISTING, 0, 0)
	if err != nil {
		return "", "", "", err
	}
	defer func() { _ = syscall.CloseHandle(hVolume) }()

	var extents VOLUME_DISK_EXTENTS
	var bytesReturned uint32
	err = syscall.DeviceIoControl(hVolume, IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS, nil, 0, (*byte)(unsafe.Pointer(&extents)), uint32(unsafe.Sizeof(extents)), &bytesReturned, nil)
	if err != nil {
		return "", "", "", err
	}

	physName := fmt.Sprintf("\\\\.\\PhysicalDrive%d", extents.Extents[0].DiskNumber)
	physPtr, _ := syscall.UTF16PtrFromString(physName)
	hDevice, err := syscall.CreateFile(physPtr, GENERIC_READ|GENERIC_WRITE, FILE_SHARE_READ|FILE_SHARE_WRITE, nil, OPEN_EXISTING, 0, 0)
	if err != nil {
		return "", "", "", err
	}
	defer func() { _ = syscall.CloseHandle(hDevice) }()

	var query STORAGE_PROPERTY_QUERY
	query.PropertyId, query.QueryType = StorageDeviceProperty, PropertyStandardQuery
	var ioBuf [1024]byte

	err = syscall.DeviceIoControl(hDevice, IOCTL_STORAGE_QUERY_PROPERTY, (*byte)(unsafe.Pointer(&query)), uint32(unsafe.Sizeof(query)), &ioBuf[0], uint32(len(ioBuf)), &bytesReturned, nil)
	if err == nil && bytesReturned >= uint32(unsafe.Sizeof(STORAGE_DEVICE_DESCRIPTOR{})) {
		desc := (*STORAGE_DEVICE_DESCRIPTOR)(unsafe.Pointer(&ioBuf[0]))
		if desc.VendorIdOffset > 0 && desc.VendorIdOffset < bytesReturned && desc.VendorIdOffset != 0xFFFFFFFF {
			vendor = getStringFromArray(ioBuf[:], desc.VendorIdOffset, bytesReturned)
		}
		if desc.ProductIdOffset > 0 && desc.ProductIdOffset < bytesReturned && desc.ProductIdOffset != 0xFFFFFFFF {
			prod := getStringFromArray(ioBuf[:], desc.ProductIdOffset, bytesReturned)
			if vendor == "Generic" || vendor == "" {
				vendor = prod
			} else {
				vendor += " " + prod
			}
		}
		if desc.SerialNumberOffset > 0 && desc.SerialNumberOffset < bytesReturned && desc.SerialNumberOffset != 0xFFFFFFFF {
			sn = getStringFromArray(ioBuf[:], desc.SerialNumberOffset, bytesReturned)
		}
		if desc.BusType == 17 {
			hwType = "NVMe"
		}
	}

	if hwType != "NVMe" {
		var penaltyQuery STORAGE_PROPERTY_QUERY
		penaltyQuery.PropertyId, penaltyQuery.QueryType = StorageDeviceSeekPenaltyProperty, PropertyStandardQuery
		var penaltyDesc DEVICE_SEEK_PENALTY_DESCRIPTOR
		err = syscall.DeviceIoControl(hDevice, IOCTL_STORAGE_QUERY_PROPERTY, (*byte)(unsafe.Pointer(&penaltyQuery)), uint32(unsafe.Sizeof(penaltyQuery)), (*byte)(unsafe.Pointer(&penaltyDesc)), uint32(unsafe.Sizeof(penaltyDesc)), &bytesReturned, nil)
		if err == nil && !penaltyDesc.IncursSeekPenalty {
			hwType = "SSD"
		}
	}
	return strings.TrimSpace(vendor), strings.TrimSpace(sn), hwType, nil
}

func getStringFromArray(buf []byte, offset uint32, maxLen uint32) string {
	if offset >= uint32(len(buf)) || offset >= maxLen {
		return ""
	}
	var sb strings.Builder
	for i := offset; i < maxLen && i < uint32(len(buf)); i++ {
		if buf[i] == 0 {
			break
		}
		sb.WriteByte(buf[i])
	}
	return sb.String()
}
