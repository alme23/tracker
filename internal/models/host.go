// tracker/internal/models/host.go
package models

import (
	"time"
)

// HostInfo содержит информацию о хосте (компьютере)
type HostInfo struct {
	Hostname         string    `json:"hostname"`                    // DNS имя компьютера
	FQDN             string    `json:"fqdn"`                        // Полное DNS имя
	PhysicalHostname string    `json:"physical_hostname,omitempty"` // Физическое имя (для VM)
	PhysicalFQDN     string    `json:"physical_fqdn,omitempty"`     // Полное физическое имя (для VM)
	Domain           string    `json:"domain"`                      // Домен
	Workgroup        string    `json:"workgroup,omitempty"`         // Рабочая группа (если не в домене)
	UpTimeSeconds    uint64    `json:"uptime_seconds"`              // Время работы в секундах
	BootTime         time.Time `json:"boot_time"`                   // Время последней загрузки
	TimeZone         string    `json:"timezone"`                    // Часовой пояс
	TimeZoneOffset   int16     `json:"timezone_offset"`             // Смещение в минутах от UTC

	// Информация о производителе и модели
	Manufacturer string `json:"manufacturer"`      // Производитель системы
	Model        string `json:"model"`             // Модель системы
	SKU          string `json:"sku,omitempty"`     // SKU (Stock Keeping Unit)
	Family       string `json:"family,omitempty"`  // Семейство системы
	Version      string `json:"version,omitempty"` // Версия системы
	SerialNumber string `json:"serial_number"`     // Серийный номер

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
