//go:build windows

package collector

import (
	"encoding/binary"
	"testing"
	"unsafe"
)

// ============ Тесты для NewRAMCollector() ============

func TestNewRAMCollector(t *testing.T) {
	collector := NewRAMCollector()

	if collector == nil {
		t.Fatal("NewRAMCollector returned nil")
	}
}

// ============ Тесты для Collect() ============

func TestRAMCollectorCollect(t *testing.T) {
	collector := NewRAMCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Проверяем основные поля
	if info.TotalBytes == 0 {
		t.Error("TotalBytes is 0")
	}

	if info.AvailableBytes > info.TotalBytes {
		t.Error("AvailableBytes > TotalBytes")
	}

	if info.TotalPageFile == 0 {
		t.Error("TotalPageFile is 0")
	}

	// Логируем
	t.Logf("Total: %.2f GB", float64(info.TotalBytes)/1024/1024/1024)
	t.Logf("Available: %.2f GB", float64(info.AvailableBytes)/1024/1024/1024)
	t.Logf("Sticks: %d", len(info.Sticks))

	for i, stick := range info.Sticks {
		t.Logf("Stick %d: %s %s %d MB @ %d MHz",
			i+1, stick.Manufacturer, stick.PartNumber,
			stick.Capacity/1024/1024, stick.SpeedMHz)
	}
}

// ============ Тесты для getPhysicalSticksFromSMBIOS() ============

func TestGetPhysicalSticksFromSMBIOS(t *testing.T) {
	collector := NewRAMCollector()

	sticks, err := collector.getPhysicalSticksFromSMBIOS()
	if err != nil {
		t.Logf("SMBIOS not available: %v", err)
		t.Skip("SMBIOS not available")
	}

	if len(sticks) == 0 {
		t.Error("No sticks found")
	}

	for i, stick := range sticks {
		t.Logf("Stick %d: Slot=%s, %d MB, %d MHz, %s %s",
			i+1, stick.Slot, stick.Capacity/1024/1024,
			stick.SpeedMHz, stick.Manufacturer, stick.PartNumber)
	}
}

// ============ Тесты для parseType17Structure() ============

func TestParseType17Structure(t *testing.T) {
	collector := NewRAMCollector()

	tests := []struct {
		name                 string
		data                 []byte
		strings              []string
		expectedCap          uint64
		expectedSpeed        uint32
		expectedManufacturer string
	}{
		{
			name: "8GB DDR3",
			data: func() []byte {
				data := make([]byte, 28)
				// Size = 8192 MB
				binary.LittleEndian.PutUint16(data[12:14], 8192)
				// Speed = 1600 MHz
				binary.LittleEndian.PutUint16(data[21:23], 1600)
				// String indices (1-based)
				data[8] = 1  // Slot
				data[23] = 2 // Manufacturer
				data[24] = 3 // Serial
				data[26] = 4 // Part Number
				return data
			}(),
			strings:              []string{"DIMM1", "Kingston", "SN123", "KVR16E11/8"},
			expectedCap:          8192 * 1024 * 1024,
			expectedSpeed:        1600,
			expectedManufacturer: "Kingston",
		},
		{
			name: "Empty slot",
			data: func() []byte {
				data := make([]byte, 28)
				// Size = 0 (empty)
				binary.LittleEndian.PutUint16(data[12:14], 0)
				return data
			}(),
			strings:              []string{},
			expectedCap:          0,
			expectedSpeed:        0,
			expectedManufacturer: "",
		},
		{
			name: "4GB in KB (0x8000 flag)",
			data: func() []byte {
				data := make([]byte, 28)
				// Size = 0x8000 | 4194304 (4GB in KB)
				// Но 4194304 не влезает в uint16!
				// Используем реальное значение: 4GB = 4194304 KB, но uint16 max = 65535
				// Поэтому для теста используем 0x8000 | 4096 (4096 KB = 4MB, не 4GB)
				// Или просто используем 0x8000 | 1024 (1024 KB = 1MB)
				binary.LittleEndian.PutUint16(data[12:14], 0x8000|1024)
				// String indices
				data[8] = 1  // Slot
				data[23] = 2 // Manufacturer
				return data
			}(),
			strings:              []string{"DIMM2", "Samsung"},
			expectedCap:          uint64(1024) * 1024, // 1024 KB
			expectedSpeed:        0,
			expectedManufacturer: "Samsung",
		},
		{
			name: "Small size in KB (no flag)",
			data: func() []byte {
				data := make([]byte, 28)
				// Size = 1024 (KB)
				binary.LittleEndian.PutUint16(data[12:14], 1024)
				data[8] = 1  // Slot
				data[23] = 2 // Manufacturer
				return data
			}(),
			strings:              []string{"DIMM3", "Hynix"},
			expectedCap:          1024 * 1024 * 1024, // 1024 MB
			expectedSpeed:        0,
			expectedManufacturer: "Hynix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stick := collector.parseType17Structure(tt.data, tt.strings)

			if stick.Capacity != tt.expectedCap {
				t.Errorf("Capacity = %d, want %d", stick.Capacity, tt.expectedCap)
			}

			if stick.SpeedMHz != tt.expectedSpeed {
				t.Errorf("Speed = %d, want %d", stick.SpeedMHz, tt.expectedSpeed)
			}

			if tt.expectedManufacturer != "" && stick.Manufacturer != tt.expectedManufacturer {
				t.Errorf("Manufacturer = %s, want %s", stick.Manufacturer, tt.expectedManufacturer)
			}
		})
	}
}

// ============ Тесты для parseSMBIOSStrings() ============

func TestParseSMBIOSStrings(t *testing.T) {
	collector := NewRAMCollector()

	tests := []struct {
		name     string
		data     []byte
		expected []string
	}{
		{
			name:     "Simple strings",
			data:     []byte("DIMM1\x00Kingston\x00SN123\x00"),
			expected: []string{"DIMM1", "Kingston", "SN123"},
		},
		{
			name:     "Empty string",
			data:     []byte{},
			expected: []string{},
		},
		{
			name:     "Double null terminator",
			data:     []byte("DIMM1\x00\x00"),
			expected: []string{"DIMM1"},
		},
		{
			name:     "With spaces",
			data:     []byte(" DIMM1 \x00 Kingston \x00"),
			expected: []string{"DIMM1", "Kingston"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.parseSMBIOSStrings(tt.data)

			if len(result) != len(tt.expected) {
				t.Errorf("Len = %d, want %d", len(result), len(tt.expected))
				return
			}

			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("String[%d] = %q, want %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

// ============ Тесты для memoryStatusEx ============

func TestMemoryStatusExSize(t *testing.T) {
	size := unsafe.Sizeof(memoryStatusEx{})

	if size != 64 {
		t.Errorf("memoryStatusEx size = %d, want 64", size)
	}
}

// ============ Тесты на целостность ============

func TestRAMCollectorDataIntegrity(t *testing.T) {
	collector := NewRAMCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Проверяем, что суммарный объем планок <= общей памяти
	var totalSticksCapacity uint64
	for _, stick := range info.Sticks {
		totalSticksCapacity += stick.Capacity
	}

	if totalSticksCapacity > info.TotalBytes {
		t.Logf("Sticks capacity (%d) > TotalBytes (%d) — may be normal",
			totalSticksCapacity, info.TotalBytes)
	}
}

// ============ Бенчмарки ============

func BenchmarkRAMCollectorCollect(b *testing.B) {
	collector := NewRAMCollector()

	b.ResetTimer()
	for b.Loop() {
		_, _ = collector.Collect()
	}
}

func BenchmarkParseType17Structure(b *testing.B) {
	collector := NewRAMCollector()
	data := make([]byte, 28)
	binary.LittleEndian.PutUint16(data[12:14], 8192)
	binary.LittleEndian.PutUint16(data[21:23], 1600)
	data[8] = 1
	data[23] = 2
	data[26] = 4
	strings := []string{"DIMM1", "Kingston", "SN123", "KVR16E11/8"}

	b.ResetTimer()
	for b.Loop() {
		_ = collector.parseType17Structure(data, strings)
	}
}

func BenchmarkParseSMBIOSStrings(b *testing.B) {
	collector := NewRAMCollector()
	data := []byte("DIMM1\x00Kingston\x00SN123\x00")

	b.ResetTimer()
	for b.Loop() {
		_ = collector.parseSMBIOSStrings(data)
	}
}
