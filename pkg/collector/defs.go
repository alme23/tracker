//go:build windows

package collector

import "syscall"

const (
	ComputerNamePhysicalDnsHostname = 5
	RelationProcessorCore           = 0
	RelationCache                   = 2

	IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS = 0x00560000
	IOCTL_STORAGE_QUERY_PROPERTY         = 0x002D1400

	PropertyStandardQuery            = 0
	StorageDeviceProperty            = 0
	StorageDeviceSeekPenaltyProperty = 7

	GENERIC_READ     = 0x80000000
	GENERIC_WRITE    = 0x40000000
	FILE_SHARE_READ  = 0x00000001
	FILE_SHARE_WRITE = 0x00000002
	OPEN_EXISTING    = 3
)

// Константы типов подключения из Win32 API (NETSETUP_JOIN_STATUS)
const (
	NetSetupUnknownStatus = 0
	NetSetupUnjoined      = 1
	NetSetupWorkgroupName = 2
	NetSetupDomainName    = 3

	// Константы для GetComputerNameExW
	ComputerNameDnsDomain = 2
	ComputerNameNetBIOS   = 4
)

const (
	// NameDisplay возвращает имя в формате отображения (например, "Иван Иванов")
	NameDisplay = 3
)

// Константы форматов имен для GetComputerNameExW
const (
	ComputerNamePhysicalNetBIOS      = 4 // Короткое NetBIOS имя
	ComputerNamePhysicalDnsFullyQual = 7 // Полный FQDN (имя_пк.домен.local)
	ComputerNamePhysicalDnsDomain    = 6 // Только имя DNS домена (домен.local)
)

type SYSTEM_INFO struct {
	ProcessorArchitecture     uint16
	Reserved                  uint16
	PageSize                  uint32
	MinimumApplicationAddress uintptr
	MaximumApplicationAddress uintptr
	ActiveProcessorMask       uintptr
	NumberOfProcessors        uint32
	ProcessorType             uint32
	AllocationGranularity     uint16
	ProcessorLevel            uint16
	ProcessorRevision         uint16
}

type MEMORYSTATUSEX struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

type DISK_EXTENT struct {
	DiskNumber     uint32
	StartingOffset int64
	ExtentLength   int64
}

type VOLUME_DISK_EXTENTS struct {
	NumberOfDiskExtents uint32
	Extents             [1]DISK_EXTENT
}

type STORAGE_PROPERTY_QUERY struct {
	PropertyId           uint32
	QueryType            uint32
	AdditionalParameters [1]byte
}

type STORAGE_DEVICE_DESCRIPTOR struct {
	Version               uint32
	Size                  uint32
	DeviceType            byte
	DeviceTypeModifier    byte
	RemovableMedia        bool
	CommandQueueing       bool
	VendorIdOffset        uint32
	ProductIdOffset       uint32
	ProductRevisionOffset uint32
	SerialNumberOffset    uint32
	BusType               uint32
	RawPropertiesLength   uint32
	RawDeviceProperties   [1]byte
}

type DEVICE_SEEK_PENALTY_DESCRIPTOR struct {
	Version           uint32
	Size              uint32
	IncursSeekPenalty bool
}

type CACHE_DESCRIPTOR struct {
	Level         byte
	Associativity byte
	LineSize      uint16
	Size          uint32
	Type          uint32
}

type SYSTEM_LOGICAL_PROCESSOR_INFORMATION_EX struct {
	Relationship uint32
	Size         uint32
}

var (
	modadvapi32      = syscall.NewLazyDLL("advapi32.dll")
	procGetUserNameW = modadvapi32.NewProc("GetUserNameW")

	modkernel32                          = syscall.NewLazyDLL("kernel32.dll")
	procGetSystemInfo                    = modkernel32.NewProc("GetSystemInfo")
	procGlobalMemoryStatusEx             = modkernel32.NewProc("GlobalMemoryStatusEx")
	procGetComputerNameExW               = modkernel32.NewProc("GetComputerNameExW")
	procGetLogicalDriveStringsW          = modkernel32.NewProc("GetLogicalDriveStringsW")
	procGetDiskFreeSpaceExW              = modkernel32.NewProc("GetDiskFreeSpaceExW")
	procGetLogicalProcessorInformationEx = modkernel32.NewProc("GetLogicalProcessorInformationEx")
	procGetDriveTypeW                    = modkernel32.NewProc("GetDriveTypeW")

	// moduser32               = syscall.NewLazyDLL("user32.dll")
	// procEnumDisplayDevicesW = moduser32.NewProc("EnumDisplayDevicesW")

	modnetapi32               = syscall.NewLazyDLL("netapi32.dll")
	procNetGetJoinInformation = modnetapi32.NewProc("NetGetJoinInformation")
	procNetApiBufferFree      = modnetapi32.NewProc("NetApiBufferFree")

	modsecur32         = syscall.NewLazyDLL("secur32.dll")
	procGetUserNameExW = modsecur32.NewProc("GetUserNameExW")
)
