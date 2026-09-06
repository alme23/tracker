package models

// SystemSnapshot представляет собой полный слепок состояния системы на текущий момент
type SystemSnapshot struct {
	Timestamp int64            `json:"timestamp"`
	User      UserInfo         `json:"user"`
	OS        OSInfo           `json:"os"`
	Processor ProcessorInfo    `json:"processor"`
	RAM       RAMInfo          `json:"ram"`
	Drives    DiskStatuses     `json:"drives"`
	Services  ServicesStatuses `json:"services"`
	Network   NetworkStatuses  `json:"network"`
	Host      HostInfo         `json:"host"`
}
