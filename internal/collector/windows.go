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

	modNtdll          = syscall.NewLazyDLL("ntdll.dll")
	procRtlGetVersion = modNtdll.NewProc("RtlGetVersion")

	modAdvapi32                    = syscall.NewLazyDLL("advapi32.dll")
	modSecur32                     = syscall.NewLazyDLL("secur32.dll")
	modNetapi32                    = syscall.NewLazyDLL("netapi32.dll")
	modWtsapi32                    = syscall.NewLazyDLL("wtsapi32.dll")
	procGetUserNameEx              = modSecur32.NewProc("GetUserNameExW")
	procOpenProcessToken           = modAdvapi32.NewProc("OpenProcessToken")
	procGetTokenInformation        = modAdvapi32.NewProc("GetTokenInformation")
	procCheckTokenMembership       = modAdvapi32.NewProc("CheckTokenMembership")
	procGetCurrentProcess          = modKernel32.NewProc("GetCurrentProcess")
	procProcessIdToSessionId       = modKernel32.NewProc("ProcessIdToSessionId")
	procNetUserGetInfo             = modNetapi32.NewProc("NetUserGetInfo")
	procNetApiBufferFree           = modNetapi32.NewProc("NetApiBufferFree")
	procWTSQuerySessionInformation = modWtsapi32.NewProc("WTSQuerySessionInformationW")
)
