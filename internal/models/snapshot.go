package models

import "time"

// SystemSnapshot представляет собой полный слепок состояния системы на текущий момент
type SystemSnapshot struct {
	Timestamp time.Time        `json:"timestamp"`
	User      UserInfo         `json:"user"`
	OS        OSInfo           `json:"os"`
	Processor ProcessorInfo    `json:"processor"`
	RAM       RAMInfo          `json:"ram"`
	Drives    DiskStatuses     `json:"drives"` // Добавляем новое поле
	Services  ServicesStatuses `json:"services"`
	Network   NetworkStatuses  `json:"network"`
	Host      HostInfo         `json:"host"`
}
