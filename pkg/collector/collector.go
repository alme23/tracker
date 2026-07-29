package collector

import (
	"runtime"
	"time"
)

// CollectMetrics координирует и агрегирует вызовы всех модулей сбора данных.
func CollectMetrics() (Metrics, error) {
	var metrics Metrics

	// 1. Сбор информации об ОС и Хосте
	if host, err := GetHostInfo(); err == nil {
		metrics.Host = host
	}
	metrics.Timestamp = time.Now().Unix()
	metrics.Host.Architecture = runtime.GOARCH

	// 2. Сбор данных об активном пользователе Windows
	if usr, err := GetCurrentUserInfo(); err == nil {
		metrics.User = usr
	}

	// 3. Сбор данных о материнской плате из реестра
	if bb, err := GetBaseBoardInfo(); err == nil {
		metrics.BaseBoard = bb
	}

	// 4. Сбор данных о BIOS из реестра
	if bios, err := GetBIOSInfo(); err == nil {
		metrics.BIOS = bios
	}

	// 5. Сбор данных о процессоре и кэшах через Win32 API
	if cpu, err := GetCPUInfo(); err == nil {
		metrics.CPU = cpu
	}

	// 6. Сбор данных о видеокартах из системного класса устройств
	if video, err := GetVideoInfo(); err == nil {
		metrics.Video = video
	}

	// 7. Сбор данных об оперативной памяти через GlobalMemoryStatusEx
	if mem, err := GetMemoryStats(); err == nil {
		metrics.Memory = mem
	}

	// 8. Сбор данных о логических и физических накопителях (NVMe/SSD/HDD)
	if disks, err := GetDiskStats(); err == nil {
		metrics.Disks = disks
	}

	// 9. Сбор данных о сетевых интерфейсах и сопоставление DHCP из реестра
	if netIfs, err := GetNetworkInterfaces(); err == nil {
		metrics.Network = netIfs
	}

	return metrics, nil
}
