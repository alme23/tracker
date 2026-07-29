//go:build !windows

package collector

import "fmt"

func GetHostInfo() (HostInfo, error)           { return HostInfo{}, fmt.Errorf("Windows only") }
func GetCurrentUserInfo() (UserInfo, error)    { return UserInfo{}, fmt.Errorf("Windows only") }
func GetBaseBoardInfo() (BaseBoardInfo, error) { return BaseBoardInfo{}, fmt.Errorf("Windows only") }
func GetBIOSInfo() (BIOSInfo, error)           { return BIOSInfo{}, fmt.Errorf("Windows only") }
func GetCPUInfo() (CPUInfo, error)             { return CPUInfo{}, fmt.Errorf("Windows only") }
func GetVideoInfo() ([]VideoInfo, error)       { return []VideoInfo{}, fmt.Errorf("Windows only") }
func GetMemoryStats() (HardwareInfo, error)    { return HardwareInfo{}, fmt.Errorf("Windows only") }
func GetDiskStats() ([]DiskInfo, error)        { return []DiskInfo{}, fmt.Errorf("Windows only") }

// ИСПРАВЛЕНО: Добавлена недостающая заглушка для сетевых интерфейсов, чтобы macOS (darwin) не ругался
func GetNetworkInterfaces() ([]NetworkInterface, error) { return []NetworkInterface{}, nil }

// Заглушка для служб RDP/VNC (остается на месте)
func GetRemoteAccessServices() (RemoteAccessInfo, error) {
	return RemoteAccessInfo{RDPStatus: "N/A", VNCStatus: "N/A"}, nil
}
