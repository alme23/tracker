//go:build !windows

package collector

import "fmt"

// Заглушки для компиляции пакета collector в контексте Linux-сервера
func GetHostInfo() (HostInfo, error) {
	return HostInfo{ComputerName: "Linux-Stub", FQDN: "N/A", DomainName: "N/A"}, nil
}
func GetCurrentUserInfo() (UserInfo, error)    { return UserInfo{}, fmt.Errorf("Windows only") }
func GetBaseBoardInfo() (BaseBoardInfo, error) { return BaseBoardInfo{}, fmt.Errorf("Windows only") }
func GetBIOSInfo() (BIOSInfo, error)           { return BIOSInfo{}, fmt.Errorf("Windows only") }
func GetCPUInfo() (CPUInfo, error)             { return CPUInfo{}, fmt.Errorf("Windows only") }
func GetVideoInfo() ([]VideoInfo, error)       { return []VideoInfo{}, fmt.Errorf("Windows only") }
func GetMemoryStats() (HardwareInfo, error)    { return HardwareInfo{}, fmt.Errorf("Windows only") }
func GetDiskStats() ([]DiskInfo, error)        { return []DiskInfo{}, fmt.Errorf("Windows only") }
