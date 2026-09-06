package models

import (
	"encoding/json"
	"fmt"
)

// DriveType is a lightweight type (1 byte) for storing the physical drive type
type DriveType uint8

// Physical drive types
const (
	// DriveUnknown means the drive type is unknown
	DriveUnknown DriveType = iota
	// DriveNoRootDir means the drive has no root directory
	DriveNoRootDir
	// DriveRemovable is a removable drive (USB flash, external HDD)
	DriveRemovable
	// DriveFixed is a fixed drive (HDD/SSD/NVMe)
	DriveFixed
	// DriveRemote is a network drive (SMB/NFS)
	DriveRemote
	// DriveCDROM is an optical drive (CD/DVD/Blu-ray)
	DriveCDROM
	// DriveRAM is a virtual RAM disk
	DriveRAM
)

// String returns the string representation of DriveType
func (t DriveType) String() string {
	switch t {
	case DriveUnknown:
		return UNKNOWN
	case DriveNoRootDir:
		return "NO_ROOT_DIR"
	case DriveRemovable:
		return "REMOVABLE"
	case DriveFixed:
		return "FIXED"
	case DriveRemote:
		return "REMOTE"
	case DriveCDROM:
		return "CD_ROM"
	case DriveRAM:
		return "RAM_DISK"
	default:
		return UNKNOWN
	}
}

// MarshalJSON serializes DriveType to JSON
func (t DriveType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// UnmarshalJSON deserializes DriveType from JSON
func (t *DriveType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch s {
	case "NO_ROOT_DIR":
		*t = DriveNoRootDir
	case "REMOVABLE":
		*t = DriveRemovable
	case "FIXED":
		*t = DriveFixed
	case "REMOTE":
		*t = DriveRemote
	case "CD_ROM":
		*t = DriveCDROM
	case "RAM_DISK":
		*t = DriveRAM
	default:
		*t = DriveUnknown
	}
	return nil
}

// DriveInfo contains detailed information about a logical drive
type DriveInfo struct {
	Letter       string    `json:"letter"`        // Letter is the drive letter (e.g., "C:") or mount point path
	Type         DriveType `json:"type"`          // Type is the physical drive type
	FSType       string    `json:"fs_type"`       // FSType is the file system type (e.g., "NTFS", "exFAT")
	TotalBytes   uint64    `json:"total_bytes"`   // TotalBytes is the total drive size in bytes
	FreeBytes    uint64    `json:"free_bytes"`    // FreeBytes is the available free space in bytes
	UsedBytes    uint64    `json:"used_bytes"`    // UsedBytes is the used space in bytes
	VolumeName   string    `json:"volume_name"`   // VolumeName is the volume label
	SerialNumber uint32    `json:"serial_number"` // SerialNumber is the raw volume serial number
	IsReady      bool      `json:"is_ready"`      // IsReady indicates whether the drive is ready for use
}

// GetSerialNumberString returns the serial number in "XXXX-XXXX" format
func (d *DriveInfo) GetSerialNumberString() string {
	if d.SerialNumber == 0 {
		return ""
	}
	return fmt.Sprintf("%04X-%04X", (d.SerialNumber>>16)&0xFFFF, d.SerialNumber&0xFFFF)
}

// GetSerialNumberHex returns the serial number in hexadecimal format
func (d *DriveInfo) GetSerialNumberHex() string {
	if d.SerialNumber == 0 {
		return ""
	}
	return fmt.Sprintf("%08X", d.SerialNumber)
}

// DiskStatuses is a slice of DriveInfo
type DiskStatuses []DriveInfo
