//go:build windows

package collector

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"unsafe"

	"github.com/alme23/tracker/internal/models"
)

// System constants for firmware table access
const (
	providerRSMB = 0x52534D42 // "RSMB" in BigEndian/DWORD format for SMBIOS
)

// Static errors
var (
	ErrSMBIOSNotAvailable  = errors.New("SMBIOS tables unavailable")
	ErrInvalidSMBIOSBuffer = errors.New("invalid SMBIOS buffer format")
)

// memoryStatusEx is an exact copy of the MEMORYSTATUSEX structure from Win32 API
type memoryStatusEx struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

// smbiosTableStructure is the SMBIOS table header
type smbiosTableStructure struct {
	Type   uint8
	Length uint8
	Handle uint16
}

// RAMCollector collects information about memory
type RAMCollector struct{}

// NewRAMCollector creates a new RAMCollector
func NewRAMCollector() *RAMCollector {
	return &RAMCollector{}
}

// Collect gathers memory and stick information natively without WMI
func (c *RAMCollector) Collect() (models.RAMInfo, error) {
	var info models.RAMInfo

	// 1. Memory usage statistics (WinAPI)
	var memStatus memoryStatusEx
	memStatus.dwLength = uint32(unsafe.Sizeof(memStatus))

	ret, _, err := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memStatus)))
	if ret == 0 {
		return info, fmt.Errorf("WinAPI GlobalMemoryStatusEx error: %w", err)
	}

	info.TotalBytes = memStatus.ullTotalPhys
	info.AvailableBytes = memStatus.ullAvailPhys
	info.TotalPageFile = memStatus.ullTotalPageFile
	info.AvailablePageFile = memStatus.ullAvailPageFile

	// 2. Collect physical sticks directly from SMBIOS Type 17
	sticks, err := c.getPhysicalSticksFromSMBIOS()
	if err == nil {
		info.Sticks = sticks
	}

	return info, nil
}

// getPhysicalSticksFromSMBIOS reads firmware and extracts memory slot information
func (c *RAMCollector) getPhysicalSticksFromSMBIOS() ([]models.RAMStick, error) {
	// First call to get the exact SMBIOS table size in bytes
	ret, _, _ := procGetSystemFirmwareTable.Call(
		uintptr(providerRSMB),
		0,
		0,
		0,
	)
	if ret == 0 {
		return nil, ErrSMBIOSNotAvailable
	}

	buffer := make([]byte, ret)
	ret, _, err := procGetSystemFirmwareTable.Call(
		uintptr(providerRSMB),
		0,
		uintptr(unsafe.Pointer(unsafe.SliceData(buffer))),
		uintptr(ret),
	)
	if ret == 0 {
		return nil, fmt.Errorf("GetSystemFirmwareTable read error: %w", err)
	}

	// Skip 8 bytes of RSMB Windows output header
	if len(buffer) < 8 {
		return nil, ErrInvalidSMBIOSBuffer
	}
	smbiosData := buffer[8:]

	var sticks []models.RAMStick
	offset := 0

	for offset+4 <= len(smbiosData) {
		// Read current SMBIOS structure header
		header := smbiosTableStructure{
			Type:   smbiosData[offset],
			Length: smbiosData[offset+1],
			Handle: binary.LittleEndian.Uint16(smbiosData[offset+2 : offset+4]),
		}

		if int(header.Length) < 4 || offset+int(header.Length) > len(smbiosData) {
			break
		}

		// Extract structure data and text string block
		structBytes := smbiosData[offset : offset+int(header.Length)]

		// Each structure ends with double null (0x00 0x00), find the end of string block
		stringOffset := offset + int(header.Length)
		endStrings := stringOffset
		for endStrings+1 < len(smbiosData) {
			if smbiosData[endStrings] == 0 && smbiosData[endStrings+1] == 0 {
				endStrings += 2
				break
			}
			endStrings++
		}

		stringBytes := smbiosData[stringOffset:endStrings]
		stringsList := c.parseSMBIOSStrings(stringBytes)

		// Type 17 is Memory Device (memory stick structure)
		if header.Type == 17 && len(structBytes) >= 28 {
			stick := c.parseType17Structure(structBytes, stringsList)
			// Add only actually installed sticks (size > 0)
			if stick.Capacity > 0 {
				sticks = append(sticks, stick)
			}
		}

		// Move to the next SMBIOS structure
		offset = endStrings
	}

	return sticks, nil
}

// parseType17Structure parses raw bytes of Memory Device structure
func (c *RAMCollector) parseType17Structure(data []byte, textStrings []string) models.RAMStick {
	var stick models.RAMStick

	// Offset 0x0C: Stick size (2 bytes)
	rawSize := binary.LittleEndian.Uint16(data[12:14])
	if rawSize == 0 || rawSize == 0xFFFF {
		return stick // Empty slot
	}

	// If high bit is 1, size is in megabytes, otherwise in kilobytes
	if (rawSize & 0x8000) == 0 {
		stick.Capacity = uint64(rawSize) * 1024 * 1024
	} else {
		stick.Capacity = uint64(rawSize&0x7FFF) * 1024
	}

	// Offset 0x15: Memory speed in MHz (2 bytes)
	if len(data) >= 23 {
		stick.SpeedMHz = uint32(binary.LittleEndian.Uint16(data[21:23]))
	}

	// Extract text string indices (SMBIOS indexing starts at 1)
	getString := func(indexByte byte) string {
		idx := int(indexByte)
		if idx > 0 && idx <= len(textStrings) {
			return textStrings[idx-1]
		}
		return "Unknown"
	}

	// String indices inside Type 17 structure
	if len(data) >= 8 {
		stick.Slot = getString(data[8])
	}
	if len(data) >= 24 {
		stick.Manufacturer = getString(data[23])
	}
	if len(data) >= 25 {
		stick.SerialNumber = getString(data[24])
	}
	if len(data) >= 27 {
		stick.PartNumber = getString(data[26])
	}

	return stick
}

// parseSMBIOSStrings splits a block of null-separated strings into a Go string slice
func (c *RAMCollector) parseSMBIOSStrings(data []byte) []string {
	var res []string

	// Use SplitSeq for more efficient iteration
	for part := range bytes.SplitSeq(data, []byte{0}) {
		s := string(bytes.TrimSpace(part))
		if s != "" {
			res = append(res, s)
		}
	}

	return res
}
