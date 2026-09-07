//go:build windows

package collector

import (
	"encoding/binary"
	"fmt"
	"math"
	"runtime"
	"unsafe"

	"github.com/alme23/tracker/internal/models"
	"golang.org/x/sys/windows/registry"
)

// Relationship type constants
const (
	relationProcessorCore    = 0
	relationNumaNode         = 1
	relationCache            = 2
	relationProcessorPackage = 3
	relationGroup            = 4
	relationAll              = 0xFFFF
)

// ProcessorCollector collects information about the processor
type ProcessorCollector struct{}

// NewProcessorCollector creates a new ProcessorCollector
func NewProcessorCollector() *ProcessorCollector {
	return &ProcessorCollector{}
}

// Collect gathers information about the processor
func (c *ProcessorCollector) Collect() (models.ProcessorInfo, error) {
	var info models.ProcessorInfo

	if err := c.collectFromRegistry(&info); err != nil {
		return info, err
	}

	c.enrichProcessorTopology(&info)
	c.validateInfo(&info)

	return info, nil
}

// collectFromRegistry gets information from the registry
func (c *ProcessorCollector) collectFromRegistry(info *models.ProcessorInfo) error {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\CentralProcessor\0`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return fmt.Errorf("CPU registry key open error: %w", err)
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
		if mhz > math.MaxUint16 {
			info.BaseSpeedMHz = math.MaxUint16
		} else {
			info.BaseSpeedMHz = uint16(mhz)
		}
	}

	if featureSet, _, err := k.GetIntegerValue("FeatureSet"); err == nil {
		info.HardwareVirtAvail = (featureSet & 0x00020000) != 0
		info.NXBitSupported = (featureSet & 0x00010000) != 0
	}

	return nil
}

// enrichProcessorTopology gets topology information
func (c *ProcessorCollector) enrichProcessorTopology(info *models.ProcessorInfo) {
	var fallbackThreads uint32
	numCPU := runtime.NumCPU()
	if numCPU <= 0 {
		numCPU = 1
	}
	if numCPU > math.MaxUint32 {
		numCPU = math.MaxUint32
	}
	fallbackThreads = uint32(numCPU)

	var returnedLength uint32
	// #nosec G103 -- passing pointer to Windows API
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

	// #nosec G103 -- passing pointer to Windows API
	ret, _, _ = procGetLogicalProcessorInformationEx.Call(
		uintptr(relationAll),
		uintptr(unsafe.Pointer(unsafe.SliceData(buffer))),
		uintptr(unsafe.Pointer(&returnedLength)),
	)

	if ret == 0 {
		c.setFallbackInfo(info, fallbackThreads)
		return
	}

	c.parseProcessorData(buffer[:returnedLength], info, fallbackThreads)
}

// setFallbackInfo sets basic information on failure
func (c *ProcessorCollector) setFallbackInfo(info *models.ProcessorInfo, logicalProcessors uint32) {
	info.LogicalProcessors = logicalProcessors
	info.PhysicalCores = logicalProcessors / 2
	if info.PhysicalCores == 0 {
		info.PhysicalCores = 1
	}
	info.SMTEnabled = logicalProcessors > info.PhysicalCores
}

// parseProcessorData parses processor topology data
func (c *ProcessorCollector) parseProcessorData(buffer []byte, info *models.ProcessorInfo, fallbackThreads uint32) {
	var offset, numaNodesCount uint32

	// Statistics counters
	cacheCount := 0
	coreCount := 0

	// Safe conversion of buffer length to uint32
	// #nosec G115
	bufferLen := uint32(len(buffer))
	if uint64(len(buffer)) > math.MaxUint32 {
		bufferLen = math.MaxUint32
	}

	for offset+8 <= bufferLen {
		relationship := binary.LittleEndian.Uint32(buffer[offset : offset+4])
		structSize := binary.LittleEndian.Uint32(buffer[offset+4 : offset+8])

		if structSize == 0 || structSize < 8 || offset+structSize > bufferLen {
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

	// Log warning if no cache structures found
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

// parseProcessorCore parses a processor core structure
func (c *ProcessorCollector) parseProcessorCore(structBytes []byte, info *models.ProcessorInfo) {
	if len(structBytes) < 32 {
		return
	}

	info.PhysicalCores++

	coreFlags := structBytes[8]
	if coreFlags == 1 {
		info.SMTEnabled = true
	}

	mask := binary.LittleEndian.Uint64(structBytes[24:32])

	var threadCount uint32
	tempMask := mask
	for tempMask > 0 {
		if tempMask&1 == 1 {
			threadCount++
		}
		tempMask >>= 1
	}

	if threadCount == 0 {
		threadCount = 1
	}

	if threadCount > 1 {
		info.SMTEnabled = true
	}

	info.LogicalProcessors += threadCount
}

// parseCache parses cache information
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

// validateInfo validates and normalizes the collected data
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

	// Use registry as fallback for cache info
	if info.L1CacheBytes == 0 || info.L2CacheBytes == 0 || info.L3CacheBytes == 0 {
		c.getCacheFromRegistry(info)
	}

	// Last resort fallback
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

// getCacheFromRegistry gets cache information from the registry
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
