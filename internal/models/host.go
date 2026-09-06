// tracker/internal/models/host.go
package models

import (
	"fmt"
	"time"
)

// HostInfo содержит информацию о хосте (компьютере)
type HostInfo struct {
	Hostname         string `json:"hostname"`          // DNS имя компьютера
	FQDN             string `json:"fqdn"`              // Полное DNS имя
	PhysicalHostname string `json:"physical_hostname"` // Физическое имя (для VM)
	PhysicalFQDN     string `json:"physical_fqdn"`     // Полное физическое имя (для VM)
	Domain           string `json:"domain"`            // Домен
	Workgroup        string `json:"workgroup"`         // Рабочая группа (если не в домене)
	UpTimeSeconds    uint64 `json:"uptime_seconds"`    // Время работы в секундах
	BootTime         uint64 `json:"boot_time"`         // Время последней загрузки
	TimeZone         string `json:"timezone"`          // Часовой пояс
	TimeZoneOffset   int16  `json:"timezone_offset"`   // Смещение в минутах от UTC

	// Информация о производителе и модели
	Manufacturer string `json:"manufacturer"`  // Производитель системы
	Model        string `json:"model"`         // Модель системы
	SKU          string `json:"sku"`           // SKU (Stock Keeping Unit)
	Family       string `json:"family"`        // Семейство системы
	Version      string `json:"version"`       // Версия системы
	SerialNumber string `json:"serial_number"` // Серийный номер

	// Информация о BIOS
	BIOSVendor       string `json:"bios_vendor"`        // Производитель BIOS
	BIOSVersion      string `json:"bios_version"`       // Версия BIOS
	BIOSDate         string `json:"bios_date"`          // Дата BIOS
	BIOSMajorRelease uint32 `json:"bios_major_release"` // Мажорная версия BIOS
	BIOSMinorRelease uint32 `json:"bios_minor_release"` // Минорная версия BIOS

	// Информация о материнской плате
	BaseBoardManufacturer string `json:"baseboard_manufacturer"` // Производитель платы
	BaseBoardProduct      string `json:"baseboard_product"`      // Модель платы
	BaseBoardVersion      string `json:"baseboard_version"`      // Версия платы
}

// GetBootTime возвращает время загрузки как time.Time
func (h *HostInfo) GetBootTime() time.Time {
	if h.BootTime == 0 {
		return time.Time{}
	}
	return time.Unix(int64(h.BootTime), 0)
}

// GetBootTimeString возвращает время загрузки в формате "YYYY-MM-DD HH:MM:SS"
func (h *HostInfo) GetBootTimeString() string {
	if h.BootTime == 0 {
		return "UNKNOWN"
	}
	return time.Unix(int64(h.BootTime), 0).Format("2006-01-02 15:04:05")
}

// GetUpTime возвращает время работы как time.Duration
func (h *HostInfo) GetUpTime() time.Duration {
	return time.Duration(h.UpTimeSeconds) * time.Second
}

// GetUpTimeString возвращает время работы в читаемом виде
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
