//go:build windows

package collector

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/alme23/tracker/internal/models"
	"golang.org/x/sys/windows"
)

// DiskCollector collects information about disk volumes
type DiskCollector struct{}

// NewDiskCollector creates a new DiskCollector
func NewDiskCollector() *DiskCollector {
	return &DiskCollector{}
}

// Collect gathers information about all local volumes
func (c *DiskCollector) Collect() (models.DiskStatuses, error) {
	var result models.DiskStatuses

	var volBuf [50]uint16

	// #nosec G103 -- safe use of unsafe.SliceData with local array
	handle, err := windows.FindFirstVolume(unsafe.SliceData(volBuf[:]), uint32(len(volBuf)))
	if err != nil {
		return nil, fmt.Errorf("WinAPI FindFirstVolume error: %w", err)
	}
	defer func() {
		_ = windows.FindVolumeClose(handle)
	}()

	for {
		volumeGUIDPath := windows.UTF16ToString(volBuf[:])

		if info, ok := c.processVolume(volumeGUIDPath); ok {
			result = append(result, info)
		}

		// #nosec G103 -- safe use of unsafe.SliceData with local array
		if !c.nextVolume(handle, unsafe.SliceData(volBuf[:]), uint32(len(volBuf))) {
			break
		}
	}

	return result, nil
}

// processVolume processes a single volume
func (c *DiskCollector) processVolume(volumeGUIDPath string) (models.DriveInfo, bool) {
	volumePtr, err := windows.UTF16PtrFromString(volumeGUIDPath)
	if err != nil {
		return models.DriveInfo{}, false
	}

	rawDriveType := windows.GetDriveType(volumePtr)

	// Skip network drives and unknown types
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

// mapDriveType converts a Windows drive type to the internal model
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

// getMountPath returns the first mount path of a volume
func (c *DiskCollector) getMountPath(volumePtr *uint16) string {
	var pathNamesBuf [1024]uint16
	var returnLen uint32

	// #nosec G103 -- safe use of unsafe.SliceData with local array
	err := windows.GetVolumePathNamesForVolumeName(
		volumePtr,
		unsafe.SliceData(pathNamesBuf[:]),
		uint32(len(pathNamesBuf)),
		&returnLen,
	)

	if err != nil || returnLen == 0 || pathNamesBuf[0] == 0 {
		return ""
	}

	return windows.UTF16ToString(pathNamesBuf[:returnLen])
}

// formatMountPath formats a mount path (e.g., "C:\" -> "C:")
func (c *DiskCollector) formatMountPath(mountPath string) string {
	if len(mountPath) == 3 && mountPath[1] == ':' && mountPath[2] == '\\' {
		return mountPath[:2]
	}
	return strings.TrimSuffix(mountPath, "\\")
}

// collectVolumeInfoAndSpace collects volume information and free space
func (c *DiskCollector) collectVolumeInfoAndSpace(info *models.DriveInfo, mountPathPtr *uint16) {
	var volumeNameBuf [256]uint16
	var volumeSerial uint32
	var maxComponentLength uint32
	var fileSystemFlags uint32
	var fsNameBuf [256]uint16
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64

	// #nosec G103 -- safe use of unsafe.SliceData with local array
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
		info.SerialNumber = volumeSerial
		info.FSType = windows.UTF16ToString(fsNameBuf[:])
		info.IsReady = true
	} else {
		info.FSType = UNKNOWN
		info.IsReady = false
	}

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

// nextVolume moves to the next volume
func (c *DiskCollector) nextVolume(handle windows.Handle, bufPtr *uint16, size uint32) bool {
	err := windows.FindNextVolume(handle, bufPtr, size)
	return err == nil
}
