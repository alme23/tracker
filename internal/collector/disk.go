// tracker/internal/collector/disk.go

//go:build windows

package collector

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/alme23/tracker/internal/models"
	"golang.org/x/sys/windows"
)

type DiskCollector struct{}

func NewDiskCollector() *DiskCollector {
	return &DiskCollector{}
}

// Collect собирает информацию обо всех локальных томах
func (c *DiskCollector) Collect() (models.DiskStatuses, error) {
	var result models.DiskStatuses

	var volBuf [50]uint16

	// Использование unsafe.SliceData гарантирует безопасность границ массива для линтеров
	handle, err := windows.FindFirstVolume(unsafe.SliceData(volBuf[:]), uint32(len(volBuf)))
	if err != nil {
		return nil, fmt.Errorf("ошибка WinAPI FindFirstVolume: %w", err)
	}
	defer func() {
		_ = windows.FindVolumeClose(handle)
	}()

	for {
		volumeGUIDPath := windows.UTF16ToString(volBuf[:])

		if info, ok := c.processVolume(volumeGUIDPath); ok {
			result = append(result, info)
		}

		if !c.nextVolume(handle, unsafe.SliceData(volBuf[:]), uint32(len(volBuf))) {
			break
		}
	}

	return result, nil
}

// processVolume обрабатывает один том
func (c *DiskCollector) processVolume(volumeGUIDPath string) (models.DriveInfo, bool) {
	volumePtr, err := windows.UTF16PtrFromString(volumeGUIDPath)
	if err != nil {
		return models.DriveInfo{}, false
	}

	rawDriveType := windows.GetDriveType(volumePtr)

	// Отсекаем сетевые диски и неизвестные
	if rawDriveType == windows.DRIVE_REMOTE || rawDriveType == windows.DRIVE_UNKNOWN {
		return models.DriveInfo{}, false
	}

	diskType := c.mapDriveType(rawDriveType)

	mountPath := c.getMountPath(volumePtr)
	if mountPath == "" {
		return models.DriveInfo{}, false
	}

	info := models.DriveInfo{
		Type:   diskType,
		Letter: c.formatMountPath(mountPath),
	}

	mountPathPtr, err := windows.UTF16PtrFromString(mountPath)
	if err == nil {
		c.collectVolumeInfoAndSpace(&info, mountPathPtr)
	}

	return info, true
}

// mapDriveType преобразует системный тип в модель
func (c *DiskCollector) mapDriveType(rawType uint32) models.DriveType {
	switch rawType {
	case windows.DRIVE_FIXED:
		return models.DriveFixed
	case windows.DRIVE_REMOVABLE:
		return models.DriveRemovable
	case windows.DRIVE_RAMDISK:
		return models.DriveRAM
	case windows.DRIVE_CDROM:
		return models.DriveCDROM
	case windows.DRIVE_NO_ROOT_DIR:
		return models.DriveNoRootDir
	case windows.DRIVE_REMOTE:
		return models.DriveRemote
	default:
		return models.DriveUnknown
	}
}

// getMountPath получает первый путь монтирования
func (c *DiskCollector) getMountPath(volumePtr *uint16) string {
	var pathNamesBuf [1024]uint16
	var returnLen uint32

	err := windows.GetVolumePathNamesForVolumeName(
		volumePtr,
		unsafe.SliceData(pathNamesBuf[:]),
		uint32(len(pathNamesBuf)),
		&returnLen,
	)

	if err != nil || returnLen == 0 || pathNamesBuf[0] == 0 {
		return ""
	}

	// Читаем строго до returnLen, чтобы не сканировать лишний хвост буфера
	return windows.UTF16ToString(pathNamesBuf[:returnLen])
}

// formatMountPath форматирует путь монтирования
func (c *DiskCollector) formatMountPath(mountPath string) string {
	if len(mountPath) == 3 && mountPath[1] == ':' && mountPath[2] == '\\' {
		return mountPath[:2]
	}
	return strings.TrimSuffix(mountPath, "\\")
}

// collectVolumeInfoAndSpace собирает информацию о томе и свободном месте
func (c *DiskCollector) collectVolumeInfoAndSpace(info *models.DriveInfo, mountPathPtr *uint16) {
	var volumeNameBuf [256]uint16
	var volumeSerial uint32
	var maxComponentLength uint32
	var fileSystemFlags uint32
	var fsNameBuf [256]uint16
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64

	// Получаем информацию о томе
	err := windows.GetVolumeInformation(
		mountPathPtr,
		unsafe.SliceData(volumeNameBuf[:]),
		uint32(len(volumeNameBuf)),
		&volumeSerial,
		&maxComponentLength,
		&fileSystemFlags,
		unsafe.SliceData(fsNameBuf[:]),
		uint32(len(fsNameBuf)),
	)

	if err == nil {
		info.VolumeName = windows.UTF16ToString(volumeNameBuf[:])
		info.SerialNumber = volumeSerial // Родной uint32 серийник!
		info.FSType = windows.UTF16ToString(fsNameBuf[:])
		info.IsReady = true
	} else {
		info.FSType = "UNKNOWN"
		info.IsReady = false
	}

	// Получаем информацию о размере только для готовых дисков
	if info.IsReady {
		err = windows.GetDiskFreeSpaceEx(
			mountPathPtr,
			&freeBytesAvailable,
			&totalNumberOfBytes,
			&totalNumberOfFreeBytes,
		)

		if err == nil {
			info.TotalBytes = totalNumberOfBytes
			info.FreeBytes = freeBytesAvailable
			info.UsedBytes = totalNumberOfBytes - freeBytesAvailable
		}
	}
}

// nextVolume переходит к следующему тому
func (c *DiskCollector) nextVolume(handle windows.Handle, bufPtr *uint16, size uint32) bool {
	err := windows.FindNextVolume(handle, bufPtr, size)
	return err == nil
}
