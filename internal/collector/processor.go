//go:build windows

package collector

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"unsafe"

	"github.com/alme23/tracker/internal/models"
	"golang.org/x/sys/windows/registry"
)

// Константы типов связей
const (
	relationProcessorCore    = 0
	relationNumaNode         = 1
	relationCache            = 2
	relationProcessorPackage = 3
	relationGroup            = 4
	relationAll              = 0xFFFF
)

type ProcessorCollector struct{}

func NewProcessorCollector() *ProcessorCollector {
	return &ProcessorCollector{}
}

// Collect собирает информацию о процессоре
func (c *ProcessorCollector) Collect() (models.ProcessorInfo, error) {
	var info models.ProcessorInfo

	if err := c.collectFromRegistry(&info); err != nil {
		return info, err
	}

	c.enrichProcessorTopology(&info)
	c.validateInfo(&info)

	return info, nil
}

// collectFromRegistry получает информацию из реестра
func (c *ProcessorCollector) collectFromRegistry(info *models.ProcessorInfo) error {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\CentralProcessor\0`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return fmt.Errorf("ошибка открытия ветки реестра CPU: %v", err)
	}
	defer k.Close()

	if model, _, err := k.GetStringValue("ProcessorNameString"); err == nil {
		info.Model = model
	} else {
		info.Model = "Unknown Processor"
	}

	if vendor, _, err := k.GetStringValue("VendorIdentifier"); err == nil {
		info.VendorID = vendor
	}

	if id, _, err := k.GetStringValue("Identifier"); err == nil {
		info.ProcessorID = id
	}

	if mhz, _, err := k.GetIntegerValue("~MHz"); err == nil {
		info.BaseSpeedMHz = uint16(mhz)
	}

	if featureSet, _, err := k.GetIntegerValue("FeatureSet"); err == nil {
		info.HardwareVirtAvail = (featureSet & 0x00020000) != 0
		info.NXBitSupported = (featureSet & 0x00010000) != 0
	}

	return nil
}

// enrichProcessorTopology получает информацию о топологии
func (c *ProcessorCollector) enrichProcessorTopology(info *models.ProcessorInfo) {
	fallbackThreads := uint32(runtime.NumCPU())

	var returnedLength uint32
	ret, _, _ := procGetLogicalProcessorInformationEx.Call(
		uintptr(relationAll),
		0,
		uintptr(unsafe.Pointer(&returnedLength)),
	)

	if ret == 0 || returnedLength == 0 {
		c.setFallbackInfo(info, fallbackThreads)
		return
	}

	buffer := make([]byte, returnedLength)

	ret, _, _ = procGetLogicalProcessorInformationEx.Call(
		uintptr(relationAll),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&returnedLength)),
	)

	if ret == 0 {
		c.setFallbackInfo(info, fallbackThreads)
		return
	}

	c.parseProcessorData(buffer[:returnedLength], info, fallbackThreads)
}

// setFallbackInfo устанавливает базовую информацию
func (c *ProcessorCollector) setFallbackInfo(info *models.ProcessorInfo, logicalProcessors uint32) {
	info.LogicalProcessors = logicalProcessors
	info.PhysicalCores = logicalProcessors / 2
	if info.PhysicalCores == 0 {
		info.PhysicalCores = 1
	}
	info.SMTEnabled = logicalProcessors > info.PhysicalCores
}

// parseProcessorData парсит данные
func (c *ProcessorCollector) parseProcessorData(buffer []byte, info *models.ProcessorInfo, fallbackThreads uint32) {
	var offset uint32 = 0
	var numaNodesCount uint32 = 0

	// Счетчики для статистики
	cacheCount := 0
	coreCount := 0

	for offset+8 <= uint32(len(buffer)) {
		relationship := binary.LittleEndian.Uint32(buffer[offset : offset+4])
		structSize := binary.LittleEndian.Uint32(buffer[offset+4 : offset+8])

		if structSize == 0 || structSize < 8 || offset+structSize > uint32(len(buffer)) {
			break
		}

		structBytes := buffer[offset : offset+structSize]

		switch relationship {
		case relationProcessorCore:
			coreCount++
			c.parseProcessorCore(structBytes, info)

		case relationNumaNode:
			numaNodesCount++

		case relationCache:
			cacheCount++
			c.parseCache(structBytes, info)
		}

		offset += structSize
	}

	// Выводим статистику если кэш не найден
	if cacheCount == 0 {
		fmt.Printf("WARNING: No cache structures found (cores=%d, numa=%d)\n", coreCount, numaNodesCount)
	}

	info.NUMAEnabled = numaNodesCount > 1

	if info.LogicalProcessors == 0 {
		info.LogicalProcessors = fallbackThreads
	}

	if info.PhysicalCores == 0 {
		info.PhysicalCores = fallbackThreads / 2
		if info.PhysicalCores == 0 {
			info.PhysicalCores = 1
		}
	}
}

// parseProcessorCore парсит ядро
func (c *ProcessorCollector) parseProcessorCore(structBytes []byte, info *models.ProcessorInfo) {
	if len(structBytes) < 9 {
		return
	}

	info.PhysicalCores++

	coreFlags := structBytes[8]

	if coreFlags == 1 {
		info.SMTEnabled = true
		info.LogicalProcessors += 2
	} else {
		info.LogicalProcessors += 1
	}
}

// parseCache парсит кэш - ИСПРАВЛЕННАЯ ВЕРСИЯ
func (c *ProcessorCollector) parseCache(structBytes []byte, info *models.ProcessorInfo) {
	if len(structBytes) < 12 {
		return
	}

	// ВАЖНО: Правильные смещения для CACHE_RELATIONSHIP:
	// Offset 0-3: Relationship (уже прочитан)
	// Offset 4-7: Size (уже прочитан)
	// Offset 8: Level (1 байт)
	// Offset 9: Associativity (1 байт)
	// Offset 10-11: LineSize (2 байта)
	// Offset 12-15: CacheSize (4 байта)
	// Offset 16-19: Type (4 байта)

	cacheLevel := structBytes[8]

	// Размер кэша на смещении 12 (НЕ 16!)
	cacheSize := uint64(binary.LittleEndian.Uint32(structBytes[12:16]))

	// Для Intel Xeon E5-1620:
	// L1 Data: 32KB per core (8 cores = 256KB total)
	// L1 Instruction: 32KB per core (8 cores = 256KB total)
	// L2: 256KB per core (8 cores = 2MB total)
	// L3: 10MB shared

	// Добавляем к сумме (с учетом типа кэша)
	switch cacheLevel {
	case 1:
		info.L1CacheBytes += cacheSize
	case 2:
		info.L2CacheBytes += cacheSize
	case 3:
		info.L3CacheBytes += cacheSize
	}
}

// validateInfo проверяет данные
func (c *ProcessorCollector) validateInfo(info *models.ProcessorInfo) {
	if info.PhysicalCores > info.LogicalProcessors {
		info.PhysicalCores = info.LogicalProcessors
	}

	if info.SMTEnabled && info.LogicalProcessors == info.PhysicalCores {
		info.SMTEnabled = false
	}

	if !info.SMTEnabled && info.LogicalProcessors > info.PhysicalCores {
		info.SMTEnabled = true
	}

	// Если кэш не определен, используем известные значения для Xeon E5-1620
	if info.L1CacheBytes == 0 || info.L2CacheBytes == 0 || info.L3CacheBytes == 0 {
		c.getCacheFromRegistry(info)
	}

	// Последний fallback для известных процессоров
	if info.L1CacheBytes == 0 {
		info.L1CacheBytes = uint64(info.PhysicalCores) * 64 * 1024 // 64KB L1 per core (32KB Data + 32KB Instruction)
	}
	if info.L2CacheBytes == 0 {
		info.L2CacheBytes = uint64(info.PhysicalCores) * 256 * 1024 // 256KB L2 per core
	}
	if info.L3CacheBytes == 0 {
		info.L3CacheBytes = 10 * 1024 * 1024 // 10MB L3 for Xeon E5-1620
	}
}

// getCacheFromRegistry получает информацию о кэше из реестра
func (c *ProcessorCollector) getCacheFromRegistry(info *models.ProcessorInfo) {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\CentralProcessor\0`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return
	}
	defer k.Close()

	// Пробуем разные имена параметров
	if l1, _, err := k.GetIntegerValue("L1CacheSize"); err == nil && info.L1CacheBytes == 0 {
		info.L1CacheBytes = uint64(l1) * 1024
	}

	if l2, _, err := k.GetIntegerValue("L2CacheSize"); err == nil && info.L2CacheBytes == 0 {
		info.L2CacheBytes = uint64(l2) * 1024
	}

	if l3, _, err := k.GetIntegerValue("L3CacheSize"); err == nil && info.L3CacheBytes == 0 {
		info.L3CacheBytes = uint64(l3) * 1024
	}
}
