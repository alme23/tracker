//go:build windows


package collector

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func GetVideoInfo() ([]VideoInfo, error) {
	// Инициализируем пустой массив, чтобы в JSON никогда не было null
	gpus := make([]VideoInfo, 0)

	// Открываем системный класс реестра Windows, отвечающий за видеоадаптеры (Display Adapters)
	// Этот GUID жестко закреплен Microsoft за всеми видеокартами во всех версиях Windows
	classKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}`, registry.READ)
	if err != nil {
		return gpus, fmt.Errorf("failed to open video class registry key: %w", err)
	}
	defer classKey.Close()

	// Получаем список подпапок (обычно это "0000", "0001", "Properties" и т.д.)
	subKeys, err := classKey.ReadSubKeyNames(-1)
	if err != nil {
		return gpus, err
	}

	for _, subKey := range subKeys {
		// Интересуют только числовые папки устройств ("0000", "0001")
		if len(subKey) != 4 {
			continue
		}

		gpuKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}\`+subKey, registry.QUERY_VALUE)
		if err != nil {
			continue
		}

		// 1. Получаем коммерческое название видеокарты
		gpuName, _, err := gpuKey.GetStringValue("DriverDesc")
		if err != nil || gpuName == "" {
			gpuKey.Close()
			continue
		}

		var vendorID uint32
		var deviceID uint32
		var hardwareInformationMemorySize uint64

		// 2. Парсим Vendor ID и Device ID из системной строки PnP (MatchingDeviceId)
		// Строка выглядит так: "pci\ven_8086&dev_5926..."
		pnpID, _, err := gpuKey.GetStringValue("MatchingDeviceId")
		if err == nil && pnpID != "" {
			pnpID = strings.ToLower(pnpID)

			// Ищем VEN_ (Vendor)
			if venIdx := strings.Index(pnpID, "ven_"); venIdx != -1 {
				fmt.Sscanf(pnpID[venIdx+4:], "%x", &vendorID)
			}
			// Ищем DEV_ (Device)
			if devIdx := strings.Index(pnpID, "dev_"); devIdx != -1 {
				fmt.Sscanf(pnpID[devIdx+4:], "%x", &deviceID)
			}
		}

		// 3. Получаем объем видеопамяти
		// Windows хранит размер памяти в байтах как HardwareInformation.MemorySize
		memSize, _, err := gpuKey.GetIntegerValue("HardwareInformation.MemorySize")
		if err == nil {
			hardwareInformationMemorySize = memSize
		}

		// Распределяем память согласно типу графики:
		// У встроенной графики Intel (0x8086) выделенной памяти (Dedicated) физически нет,
		// драйвер рапортует общую память. Для дискретных карт пишем в Dedicated.
		var dedicated uint64
		var shared uint64

		if vendorID == 0x8086 {
			shared = hardwareInformationMemorySize
			// Если Windows вернул 0 (такое бывает у Intel Iris в реестре),
			// по умолчанию под нужды графики Intel резервирует до половины системной RAM, но оставим фактический 0
		} else {
			dedicated = hardwareInformationMemorySize
		}

		gpus = append(gpus, VideoInfo{
			Name:            gpuName,
			VendorID:        vendorID,
			DeviceID:        deviceID,
			DedicatedMemory: dedicated,
			SharedMemory:    shared,
		})

		gpuKey.Close()
	}

	return gpus, nil
}
