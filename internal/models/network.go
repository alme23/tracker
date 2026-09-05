package models

import (
	"encoding/json"
	"net"
)

// IPAssignment — низкоуровневый тип (1 байт) для хранения способа получения IP
type IPAssignment uint8

// Объявляем энум через iota. Числа присвоятся автоматически (0, 1, 2, 3)
const (
	AssignmentUnknown IPAssignment = iota
	AssignmentDHCP
	AssignmentStatic
	AssignmentNotApps
)

// String возвращает текстовое представление (полезно для логов или printf)
func (a IPAssignment) String() string {
	switch a {
	case AssignmentUnknown:
		return "UNKNOWN"
	case AssignmentDHCP:
		return "DHCP"
	case AssignmentStatic:
		return "STATIC"
	case AssignmentNotApps:
		return "NOT_APPLICABLE"
	default:
		return "UNKNOWN"
	}
}

// MarshalJSON перехватывает стандартный маршалинг Go и превращает число в понятную строку для JSON
func (a IPAssignment) MarshalJSON() ([]byte, error) {
	// Сериализуем строковое представление нашей константы
	return json.Marshal(a.String())
}

// UnmarshalJSON (опционально) позволит серверу правильно распарсить строку обратно в наш uint8
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

type InterfaceType uint8

const (
	TypeUnknown InterfaceType = iota
	TypeEthernet
	TypeWireless
	TypeLoopback
	TypeTunnel
	TypePPP
	TypeOther
)

// String возвращает тип интерфейса в UPPERCASE-стиле
func (t InterfaceType) String() string {
	switch t {
	case TypeUnknown:
		return "UNKNOWN"
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
		return "UNKNOWN"
	}
}

// UnmarshalJSON (опционально) позволит серверу правильно распарсить строку обратно в наш uint8
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

func (t InterfaceType) MarshalJSON() ([]byte, error) { return json.Marshal(t.String()) }

// InterfaceInfo содержит детальную информацию о сетевом адаптере
type InterfaceInfo struct {
	Index        int           `json:"index"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	MAC          string        `json:"mac"`
	Type         InterfaceType `json:"type"`
	Operational  bool          `json:"operational"`
	IPAssignment IPAssignment  `json:"ip_assignment"` // Наш супер-легкий тип
	IPAddresses  []net.IP      `json:"ip_addresses"`
}

type NetworkStatuses []InterfaceInfo
