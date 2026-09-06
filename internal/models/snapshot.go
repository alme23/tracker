package models

// SystemSnapshot represents a complete snapshot of the system state at a given moment
type SystemSnapshot struct {
	Timestamp int64            `json:"timestamp"` // Unix timestamp when the snapshot was taken
	User      UserInfo         `json:"user"`      // Information about the current user
	OS        OSInfo           `json:"os"`        // Operating system information
	Processor ProcessorInfo    `json:"processor"` // Processor characteristics
	RAM       RAMInfo          `json:"ram"`       // Memory information
	Drives    DiskStatuses     `json:"drives"`    // List of drives
	Services  ServicesStatuses `json:"services"`  // Service status (RDP, VNC)
	Network   NetworkStatuses  `json:"network"`   // Network interfaces
	Host      HostInfo         `json:"host"`      // Host (computer) information
}
