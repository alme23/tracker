// tracker/internal/collector/processor.go

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
		return fmt.Errorf("ошибка открытия ветки реестра CPU: %w", err)
	}
	defer func() {
		_ = k.Close()
	}()

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
	// Первый вызов для определения размера буфера
	_, _, _ = procGetLogicalProcessorInformationEx.Call(
		uintptr(relationAll),
		0,
		uintptr(unsafe.Pointer(&returnedLength)),
	)

	if returnedLength == 0 {
		c.setFallbackInfo(info, fallbackThreads)
		return
	}

	buffer := make([]byte, returnedLength)

	ret, _, _ := procGetLogicalProcessorInformationEx.Call(
		uintptr(relationAll),
		uintptr(unsafe.Pointer(unsafe.SliceData(buffer))), // Безопасный SliceData вместо &buffer[0]
		uintptr(unsafe.Pointer(&returnedLength)),
	)

	if ret == 0 {
		c.setFallbackInfo(info, fallbackThreads)
		return
	}

	// Сбрасываем счетчики перед парсингом, чтобы избежать дублирования при повторных вызовах
	info.PhysicalCores = 0
	info.LogicalProcessors = 0

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
	var offset uint32
	var numaNodesCount uint32

	for offset+8 <= uint32(len(buffer)) {
		relationship := binary.LittleEndian.Uint32(buffer[offset : offset+4])
		structSize := binary.LittleEndian.Uint32(buffer[offset+4 : offset+8])

		if structSize == 0 || structSize < 8 || offset+structSize > uint32(len(buffer)) {
			break
		}

		structBytes := buffer[offset : offset+structSize]

		switch relationship {
		case relationProcessorCore:
			c.parseProcessorCore(structBytes, info)

		case relationNumaNode:
			numaNodesCount++

		case relationCache:
			c.parseCache(structBytes, info)
		}

		offset += structSize
	}

	info.NUMAEnabled = numaNodesCount > 1

	// Если WinAPI не вернул данные, используем фолбэки
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

// parseProcessorCore парсит ядро на 64-битной Windows
func (c *ProcessorCollector) parseProcessorCore(structBytes []byte, info *models.ProcessorInfo) {
	// Для relationProcessorCore размер структуры на x64 составляет минимум 32 байта
	if len(structBytes) < 32 {
		return
	}

	info.PhysicalCores++

	// Флаг LTP_EMPTY (индекс 8)
	coreFlags := structBytes[8]
	if coreFlags == 1 {
		info.SMTEnabled = true
	}

	// На x64 Windows маска процессора (GroupMask.Mask) находится строго на смещении 24
	// и занимает 8 байт (uint64)
	mask := binary.LittleEndian.Uint64(structBytes[24:32])

	var threadCount uint32
	tempMask := mask
	for tempMask > 0 {
		if tempMask&1 == 1 {
			threadCount++
		}
		tempMask >>= 1
	}

	// Если маска по какой-то причине пустая (баг виртуализации/эмуляции),
	// берем базовый fallback в 1 поток
	if threadCount == 0 {
		threadCount = 1
	}

	// Дополнительная проверка на SMT:
	// Если маска одного физического ядра содержит больше 1 бита (потока), значит SMT гарантированно активен
	if threadCount > 1 {
		info.SMTEnabled = true
	}

	info.LogicalProcessors += threadCount
}

// parseCache парсит кэш
func (c *ProcessorCollector) parseCache(structBytes []byte, info *models.ProcessorInfo) {
	if len(structBytes) < 16 {
		return
	}

	cacheLevel := structBytes[8]
	cacheSize := uint64(binary.LittleEndian.Uint32(structBytes[12:16]))

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

	// Если кэш не определен, пробуем из реестра
	if info.L1CacheBytes == 0 || info.L2CacheBytes == 0 || info.L3CacheBytes == 0 {
		c.getCacheFromRegistry(info)
	}

	// Последний fallback для популярных CPU вроде Xeon E5-1620
	if info.L1CacheBytes == 0 {
		info.L1CacheBytes = uint64(info.PhysicalCores) * 64 * 1024
	}
	if info.L2CacheBytes == 0 {
		info.L2CacheBytes = uint64(info.PhysicalCores) * 256 * 1024
	}
	if info.L3CacheBytes == 0 {
		info.L3CacheBytes = 10 * 1024 * 1024
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
	defer func() {
		_ = k.Close()
	}()

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
