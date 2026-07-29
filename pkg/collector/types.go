package collector

import (
	"encoding/json"
	"net"
)

type IPAssignmentType string

const (
	IPAssignmentTypeStatic IPAssignmentType = "static"
	IPAssignmentTypeDHCP   IPAssignmentType = "dhcp"
)

func (t IPAssignmentType) String() string               { return string(t) }
func (t IPAssignmentType) MarshalJSON() ([]byte, error) { return json.Marshal(string(t)) }

type InterfaceStatus string

const (
	InterfaceStatusUp         InterfaceStatus = "up"
	InterfaceStatusDown       InterfaceStatus = "down"
	InterfaceStatusTesting    InterfaceStatus = "testing"
	InterfaceStatusDormant    InterfaceStatus = "dormant"
	InterfaceStatusNotPresent InterfaceStatus = "not_present"
	InterfaceStatusUnknown    InterfaceStatus = "unknown"
)

func (s InterfaceStatus) String() string               { return string(s) }
func (s InterfaceStatus) MarshalJSON() ([]byte, error) { return json.Marshal(string(s)) }

type MACAddress net.HardwareAddr

// Реализация интерфейса кодирования GOB для MACAddress
func (m MACAddress) GobEncode() ([]byte, error) {
	return []byte(m), nil
}

// Реализация интерфейса декодирования GOB для MACAddress
func (m *MACAddress) GobDecode(data []byte) error {
	*m = MACAddress(data)
	return nil
}

// ИСПРАВЛЕНО/ДОБАВЛЕНО: Явный метод String() для вывода MAC-адреса
func (m MACAddress) String() string {
	return net.HardwareAddr(m).String()
}

type IPAddress net.IP

// Реализация интерфейса кодирования GOB для IPAddress
func (ip IPAddress) GobEncode() ([]byte, error) {
	return []byte(ip), nil
}

// Реализация интерфейса декодирования GOB для IPAddress
func (ip *IPAddress) GobDecode(data []byte) error {
	*ip = IPAddress(data)
	return nil
}

// ИСПРАВЛЕНО/ДОБАВЛЕНО: Явный метод String() для вывода IP-адреса
func (ip IPAddress) String() string {
	return net.IP(ip).String()
}

type Metrics struct {
	Timestamp int64              `json:"timestamp"`
	Host      HostInfo           `json:"host"`
	User      UserInfo           `json:"user"`
	BaseBoard BaseBoardInfo      `json:"baseboard"`
	BIOS      BIOSInfo           `json:"bios"`
	CPU       CPUInfo            `json:"cpu"`
	Video     []VideoInfo        `json:"video"`
	Memory    HardwareInfo       `json:"memory"`
	Disks     []DiskInfo         `json:"disks"`
	Network   []NetworkInterface `json:"network"`
}

type HostInfo struct {
	ComputerName    string `gob:"computer_name"` // Короткое имя ПК (NetBIOS)
	FQDN            string `gob:"fqdn"`          // Полное имя компьютера (напр., pc01.corp.local)
	DomainName      string `gob:"domain_name"`   // Имя домена AD (напр., corp.local)
	OSName          string `gob:"os_name"`
	OSEdition       string `gob:"os_edition"`
	OSVersion       string `gob:"os_version"`
	OSBuild         string `gob:"os_build"`
	InstallDateUnix int64  `gob:"install_date_unix"`
	InstallDateStr  string `gob:"install_date_str"`
	Architecture    string `gob:"architecture"`
	ProductID       string `gob:"product_id"`
	DeviceID        string `gob:"device_id"`
}

type UserInfo struct {
	UserName    string `gob:"user_name"`
	DisplayName string `gob:"display_name"` // Новое поле для полного имени
	JoinType    string `gob:"join_type"`
	Domain      string `gob:"domain"`
}

type BaseBoardInfo struct {
	Vendor       string `json:"vendor"`
	Product      string `json:"product"`
	Version      string `json:"version"`
	SerialNumber string `json:"serial_number"`
}

type BIOSInfo struct {
	Vendor      string `json:"vendor"`
	Version     string `json:"version"`
	ReleaseDate string `json:"release_date"`
}

type CPUInfo struct {
	Name              string `json:"name"`
	PhysicalCores     uint32 `json:"physical_cores"`
	LogicalProcessors uint32 `json:"logical_processors"`
	MaxSpeedMHz       uint32 `json:"max_speed_mhz"`
	CacheL1Bytes      uint64 `json:"cache_l1_bytes"`
	CacheL2Bytes      uint64 `json:"cache_l2_bytes"`
	CacheL3Bytes      uint64 `json:"cache_l3_bytes"`
}

type VideoInfo struct {
	Name            string `json:"name"`             // Название (например, "NVIDIA GeForce RTX 4060")
	VendorID        uint32 `json:"vendor_id"`        // ID производителя (Vendor ID)
	DeviceID        uint32 `json:"device_id"`        // ID устройства (Device ID)
	DedicatedMemory uint64 `json:"dedicated_memory"` // Выделенная видеопамять в байтах (uint64)
	SharedMemory    uint64 `json:"shared_memory"`    // Разделяемая системная память в байтах (uint64)
}

type HardwareInfo struct {
	TotalRAMBytes uint64 `json:"total_ram_bytes"`
	FreeRAMBytes  uint64 `json:"free_ram_bytes"`
}

type DiskInfo struct {
	Drive        string `json:"drive"`
	Type         string `json:"type"`
	Vendor       string `json:"vendor"`
	SerialNumber string `json:"serial_number"`
	TotalBytes   uint64 `json:"total_bytes"`
	FreeBytes    uint64 `json:"free_bytes"`
}

type NetworkInterface struct {
	Name        string           `json:"name"`
	MAC         MACAddress       `json:"mac"`
	Status      InterfaceStatus  `json:"status"`
	IPType      IPAssignmentType `json:"ip_type"`
	IPAddresses []IPAddress      `json:"ip_addresses"`
}
