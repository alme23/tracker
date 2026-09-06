package models

import (
	"encoding/json"
	"net"
)

// IPAssignment is a lightweight type (1 byte) for storing how IP was assigned
type IPAssignment uint8

// IP assignment methods
const (
	// AssignmentUnknown means the assignment method is unknown
	AssignmentUnknown IPAssignment = iota
	// AssignmentDHCP means IP was assigned via DHCP
	AssignmentDHCP
	// AssignmentStatic means IP was statically assigned
	AssignmentStatic
	// AssignmentNotApps means IP assignment is not applicable (loopback)
	AssignmentNotApps
)

// String returns the string representation of IPAssignment
func (a IPAssignment) String() string {
	switch a {
	case AssignmentUnknown:
		return UNKNOWN
	case AssignmentDHCP:
		return "DHCP"
	case AssignmentStatic:
		return "STATIC"
	case AssignmentNotApps:
		return "NOT_APPLICABLE"
	default:
		return UNKNOWN
	}
}

// MarshalJSON serializes IPAssignment to JSON
func (a IPAssignment) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

// UnmarshalJSON deserializes IPAssignment from JSON
func (a *IPAssignment) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	switch s {
	case "DHCP":
		*a = AssignmentDHCP
	case "STATIC":
		*a = AssignmentStatic
	case "NOT_APPLICABLE":
		*a = AssignmentNotApps
	default:
		*a = AssignmentUnknown
	}
	return nil
}

// InterfaceType is a lightweight type for storing network interface type
type InterfaceType uint8

// Network interface types
const (
	// TypeUnknown means the interface type is unknown
	TypeUnknown InterfaceType = iota
	// TypeEthernet is an Ethernet interface
	TypeEthernet
	// TypeWireless is a wireless (Wi-Fi) interface
	TypeWireless
	// TypeLoopback is a loopback interface
	TypeLoopback
	// TypeTunnel is a tunnel interface
	TypeTunnel
	// TypePPP is a Point-to-Point Protocol interface
	TypePPP
	// TypeOther is another interface type
	TypeOther
)

// String returns the string representation of InterfaceType
func (t InterfaceType) String() string {
	switch t {
	case TypeUnknown:
		return UNKNOWN
	case TypeEthernet:
		return "ETHERNET"
	case TypeWireless:
		return "WIRELESS"
	case TypeLoopback:
		return "LOOPBACK"
	case TypeTunnel:
		return "TUNNEL"
	case TypePPP:
		return "POINT_TO_POINT"
	case TypeOther:
		return "OTHER"
	default:
		return UNKNOWN
	}
}

// MarshalJSON serializes InterfaceType to JSON
func (t InterfaceType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// UnmarshalJSON deserializes InterfaceType from JSON
func (t *InterfaceType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	switch s {
	case "ETHERNET":
		*t = TypeEthernet
	case "WIRELESS":
		*t = TypeWireless
	case "LOOPBACK":
		*t = TypeLoopback
	case "TUNNEL":
		*t = TypeTunnel
	case "POINT_TO_POINT":
		*t = TypePPP
	case "OTHER":
		*t = TypeOther
	default:
		*t = TypeUnknown
	}
	return nil
}

// InterfaceInfo contains detailed information about a network adapter
type InterfaceInfo struct {
	Index        int           `json:"index"`         // Interface index
	Name         string        `json:"name"`          // Interface name (GUID)
	Description  string        `json:"description"`   // Interface description
	MAC          string        `json:"mac"`           // MAC address
	Type         InterfaceType `json:"type"`          // Interface type
	Operational  bool          `json:"operational"`   // Whether interface is up
	IPAssignment IPAssignment  `json:"ip_assignment"` // How IP was assigned
	IPAddresses  []net.IP      `json:"ip_addresses"`  // List of IP addresses
}

// NetworkStatuses is a slice of InterfaceInfo
type NetworkStatuses []InterfaceInfo
