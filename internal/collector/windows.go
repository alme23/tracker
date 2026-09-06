// tracker/internal/collector/windows.go

//go:build windows

package collector

import "syscall"

var (
	modKernel32                          = syscall.NewLazyDLL("kernel32.dll")
	procGetLogicalProcessorInformationEx = modKernel32.NewProc("GetLogicalProcessorInformationEx")
	procGlobalMemoryStatusEx             = modKernel32.NewProc("GlobalMemoryStatusEx")
	procGetProductInfo                   = modKernel32.NewProc("GetProductInfo")
	procGetTickCount64                   = modKernel32.NewProc("GetTickCount64")
	procGetComputerNameEx                = modKernel32.NewProc("GetComputerNameExW")
	procGetTimeZoneInformation           = modKernel32.NewProc("GetTimeZoneInformation")
	procGetFirmwareEnvironmentVariable   = modKernel32.NewProc("GetFirmwareEnvironmentVariableW")
	procGetSystemFirmwareTable           = modKernel32.NewProc("GetSystemFirmwareTable")
	procGetSystemDefaultLocaleName       = modKernel32.NewProc("GetSystemDefaultLocaleName")

	modNtdll          = syscall.NewLazyDLL("ntdll.dll")
	procRtlGetVersion = modNtdll.NewProc("RtlGetVersion")

	modAdvapi32              = syscall.NewLazyDLL("advapi32.dll")
	modSecur32               = syscall.NewLazyDLL("secur32.dll")
	modNetapi32              = syscall.NewLazyDLL("netapi32.dll")
	procGetUserNameEx        = modSecur32.NewProc("GetUserNameExW")
	procCheckTokenMembership = modAdvapi32.NewProc("CheckTokenMembership")
	procNetUserGetInfo       = modNetapi32.NewProc("NetUserGetInfo")
	procNetApiBufferFree     = modNetapi32.NewProc("NetApiBufferFree")
)
