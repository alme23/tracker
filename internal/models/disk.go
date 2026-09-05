// tracker/internal/models/disk.go
package models

import (
	"encoding/json"
	"fmt"
)

// DriveType — легкий тип (1 байт) для хранения физического типа диска
type DriveType uint8

const (
	DriveUnknown   DriveType = iota
	DriveNoRootDir           // Диск без корневой директории
	DriveRemovable           // Съемный диск (флешка, внешний USB-накопитель)
	DriveFixed               // Встроенный HDD/SSD/NVMe
	DriveRemote              // Сетевой диск (SMB/NFS)
	DriveCDROM               // Оптический привод CD/DVD/Blu-ray
	DriveRAM                 // Виртуальный диск в оперативной памяти
)

func (t DriveType) String() string {
	switch t {
	case DriveUnknown:
		return "UNKNOWN"
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
		return "UNKNOWN"
	}
}

func (t DriveType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

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

// DriveInfo содержит детальную информацию о логическом диске
type DriveInfo struct {
	Letter       string    `json:"letter"`        // Буква диска ("C:") или путь монтирования
	Type         DriveType `json:"type"`          // Тип диска
	FSType       string    `json:"fs_type"`       // Тип файловой системы ("NTFS", "exFAT")
	TotalBytes   uint64    `json:"total_bytes"`   // Общий объем в байтах
	FreeBytes    uint64    `json:"free_bytes"`    // Свободный объем в байтах
	UsedBytes    uint64    `json:"used_bytes"`    // Использованный объем
	VolumeName   string    `json:"volume_name"`   // Метка тома
	SerialNumber uint32    `json:"serial_number"` // Серийный номер
	IsReady      bool      `json:"is_ready"`      // Готов ли диск
}

// GetSerialNumberString возвращает серийный номер в формате "XXXX-XXXX"
func (d *DriveInfo) GetSerialNumberString() string {
	if d.SerialNumber == 0 {
		return ""
	}

	// Формат: XXXX-XXXX
	return fmt.Sprintf("%04X-%04X",
		(d.SerialNumber>>16)&0xFFFF,
		d.SerialNumber&0xFFFF)
}

// GetSerialNumberHex возвращает серийный номер в шестнадцатеричном формате
func (d *DriveInfo) GetSerialNumberHex() string {
	if d.SerialNumber == 0 {
		return ""
	}

	return fmt.Sprintf("%08X", d.SerialNumber)
}

type DiskStatuses []DriveInfo
