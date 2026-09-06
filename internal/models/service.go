package models

// ServiceStatus contains collected information about a service
type ServiceStatus struct {
	Name        string `json:"name"`         // Display name (e.g., "RDP", "VNC")
	ServiceName string `json:"service_name"` // Windows service name (e.g., "TermService")
	Installed   bool   `json:"installed"`    // Whether the service is installed
	Running     bool   `json:"running"`      // Whether the service is currently running
	Port        uint16 `json:"port"`         // Active port number (e.g., 3389, 5900)
	PortOpen    bool   `json:"port_open"`    // Whether the port is accessible from outside
}

// ServicesStatuses is a slice of ServiceStatus
type ServicesStatuses []ServiceStatus
