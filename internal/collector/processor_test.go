//go:build windows

package collector

import (
	"encoding/binary"
	"testing"

	"github.com/alme23/tracker/internal/models"
)

// ============ Тесты для NewProcessorCollector() ============

func TestNewProcessorCollector(t *testing.T) {
	collector := NewProcessorCollector()

	if collector == nil {
		t.Fatal("NewProcessorCollector returned nil")
	}
}

// ============ Тесты для Collect() ============

func TestProcessorCollectorCollect(t *testing.T) {
	collector := NewProcessorCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Проверяем обязательные поля
	if info.Model == "" {
		t.Error("Model is empty")
	}

	if info.VendorID == "" {
		t.Error("VendorID is empty")
	}

	if info.PhysicalCores == 0 {
		t.Error("PhysicalCores is 0")
	}

	if info.LogicalProcessors == 0 {
		t.Error("LogicalProcessors is 0")
	}

	if info.PhysicalCores > info.LogicalProcessors {
		t.Errorf("PhysicalCores (%d) > LogicalProcessors (%d)",
			info.PhysicalCores, info.LogicalProcessors)
	}

	t.Logf("Model: %s", info.Model)
	t.Logf("Vendor: %s", info.VendorID)
	t.Logf("Cores: %d/%d", info.PhysicalCores, info.LogicalProcessors)
	t.Logf("SMT: %v", info.SMTEnabled)
	t.Logf("L1: %d KB", info.L1CacheBytes/1024)
	t.Logf("L2: %d KB", info.L2CacheBytes/1024)
	t.Logf("L3: %d MB", info.L3CacheBytes/1024/1024)
}

// ============ Тесты для collectFromRegistry() ============

func TestCollectFromRegistry(t *testing.T) {
	collector := NewProcessorCollector()
	info := &models.ProcessorInfo{}

	err := collector.collectFromRegistry(info)
	if err != nil {
		t.Fatalf("collectFromRegistry failed: %v", err)
	}

	if info.Model == "" {
		t.Error("Model is empty")
	}

	if info.VendorID == "" {
		t.Error("VendorID is empty")
	}

	t.Logf("Model: %s", info.Model)
	t.Logf("Vendor: %s", info.VendorID)
	t.Logf("ProcessorID: %s", info.ProcessorID)
	t.Logf("BaseSpeed: %d MHz", info.BaseSpeedMHz)
	t.Logf("VT-x: %v", info.HardwareVirtAvail)
	t.Logf("NX: %v", info.NXBitSupported)
}

// ============ Тесты для enrichProcessorTopology() ============

func TestEnrichProcessorTopology(t *testing.T) {
	collector := NewProcessorCollector()
	info := &models.ProcessorInfo{}

	collector.enrichProcessorTopology(info)

	if info.LogicalProcessors == 0 {
		t.Error("LogicalProcessors is 0")
	}

	t.Logf("LogicalProcessors: %d", info.LogicalProcessors)
	t.Logf("PhysicalCores: %d", info.PhysicalCores)
	t.Logf("SMT: %v", info.SMTEnabled)
	t.Logf("L1: %d", info.L1CacheBytes)
	t.Logf("L2: %d", info.L2CacheBytes)
	t.Logf("L3: %d", info.L3CacheBytes)
}

// ============ Тесты для setFallbackInfo() ============

func TestSetFallbackInfo(t *testing.T) {
	collector := NewProcessorCollector()

	tests := []struct {
		name              string
		logicalProcessors uint32
		expectedCores     uint32
		expectedSMT       bool
	}{
		{"8 threads", 8, 4, true},
		{"4 threads", 4, 2, true},
		{"2 threads", 2, 1, true},
		{"1 thread", 1, 1, false},
		{"0 threads", 0, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &models.ProcessorInfo{}
			collector.setFallbackInfo(info, tt.logicalProcessors)

			if info.LogicalProcessors != tt.logicalProcessors {
				t.Errorf("LogicalProcessors = %d, want %d",
					info.LogicalProcessors, tt.logicalProcessors)
			}

			if info.PhysicalCores != tt.expectedCores {
				t.Errorf("PhysicalCores = %d, want %d",
					info.PhysicalCores, tt.expectedCores)
			}

			if info.SMTEnabled != tt.expectedSMT {
				t.Errorf("SMTEnabled = %v, want %v",
					info.SMTEnabled, tt.expectedSMT)
			}
		})
	}
}

// ============ Тесты для parseProcessorData() ============

func TestParseProcessorData(t *testing.T) {
	collector := NewProcessorCollector()

	// Создаем тестовый буфер с кэшем
	buffer := make([]byte, 64)

	// Структура relationCache
	binary.LittleEndian.PutUint32(buffer[0:4], relationCache)
	binary.LittleEndian.PutUint32(buffer[4:8], 24) // Размер структуры

	// Cache Level 1
	buffer[8] = 1
	// Cache Size = 64KB
	binary.LittleEndian.PutUint32(buffer[12:16], 64*1024)

	info := &models.ProcessorInfo{}
	collector.parseProcessorData(buffer, info, 4)

	if info.L1CacheBytes != 64*1024 {
		t.Errorf("L1CacheBytes = %d, want %d", info.L1CacheBytes, 64*1024)
	}
}

// ============ Тесты для parseProcessorCore() ============

func TestParseProcessorCore(t *testing.T) {
	collector := NewProcessorCollector()

	tests := []struct {
		name            string
		flags           byte
		mask            uint64
		expectedThreads uint32
		expectedSMT     bool
	}{
		{"Single thread", 0, 0x1, 1, false},
		{"Two threads (SMT)", 1, 0x3, 2, true},
		{"Four threads", 1, 0xF, 4, true},
		{"Empty mask", 0, 0x0, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &models.ProcessorInfo{}

			// Создаем структуру размером 32 байта
			structBytes := make([]byte, 32)
			structBytes[8] = tt.flags
			binary.LittleEndian.PutUint64(structBytes[24:32], tt.mask)

			collector.parseProcessorCore(structBytes, info)

			if info.LogicalProcessors != tt.expectedThreads {
				t.Errorf("LogicalProcessors = %d, want %d",
					info.LogicalProcessors, tt.expectedThreads)
			}

			if info.SMTEnabled != tt.expectedSMT {
				t.Errorf("SMTEnabled = %v, want %v",
					info.SMTEnabled, tt.expectedSMT)
			}
		})
	}
}

// ============ Тесты для parseCache() ============

func TestParseCache(t *testing.T) {
	collector := NewProcessorCollector()

	tests := []struct {
		name      string
		level     byte
		cacheSize uint32
	}{
		{"L1", 1, 32 * 1024},
		{"L2", 2, 256 * 1024},
		{"L3", 3, 8 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &models.ProcessorInfo{}
			structBytes := make([]byte, 16)
			structBytes[8] = tt.level
			binary.LittleEndian.PutUint32(structBytes[12:16], tt.cacheSize)

			collector.parseCache(structBytes, info)

			switch tt.level {
			case 1:
				if info.L1CacheBytes != uint64(tt.cacheSize) {
					t.Errorf("L1 = %d, want %d", info.L1CacheBytes, tt.cacheSize)
				}
			case 2:
				if info.L2CacheBytes != uint64(tt.cacheSize) {
					t.Errorf("L2 = %d, want %d", info.L2CacheBytes, tt.cacheSize)
				}
			case 3:
				if info.L3CacheBytes != uint64(tt.cacheSize) {
					t.Errorf("L3 = %d, want %d", info.L3CacheBytes, tt.cacheSize)
				}
			}
		})
	}
}

// ============ Тесты для validateInfo() ============

func TestValidateInfo(t *testing.T) {
	collector := NewProcessorCollector()

	tests := []struct {
		name        string
		input       models.ProcessorInfo
		expectedSMT bool
	}{
		{
			name: "SMT enabled",
			input: models.ProcessorInfo{
				PhysicalCores:     4,
				LogicalProcessors: 8,
			},
			expectedSMT: true,
		},
		{
			name: "No SMT",
			input: models.ProcessorInfo{
				PhysicalCores:     4,
				LogicalProcessors: 4,
			},
			expectedSMT: false,
		},
		{
			name: "Invalid",
			input: models.ProcessorInfo{
				PhysicalCores:     8,
				LogicalProcessors: 4,
			},
			expectedSMT: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := tt.input
			collector.validateInfo(&info)

			if info.SMTEnabled != tt.expectedSMT {
				t.Errorf("SMTEnabled = %v, want %v", info.SMTEnabled, tt.expectedSMT)
			}

			// Кэш должен быть заполнен
			if info.L1CacheBytes == 0 {
				t.Error("L1CacheBytes is 0")
			}
			if info.L2CacheBytes == 0 {
				t.Error("L2CacheBytes is 0")
			}
			if info.L3CacheBytes == 0 {
				t.Error("L3CacheBytes is 0")
			}
		})
	}
}

// ============ Тесты для getCacheFromRegistry() ============

func TestGetCacheFromRegistry(t *testing.T) {
	collector := NewProcessorCollector()
	info := &models.ProcessorInfo{}

	collector.getCacheFromRegistry(info)

	t.Logf("L1: %d", info.L1CacheBytes)
	t.Logf("L2: %d", info.L2CacheBytes)
	t.Logf("L3: %d", info.L3CacheBytes)
}

// ============ Бенчмарки ============

func BenchmarkProcessorCollectorCollect(b *testing.B) {
	collector := NewProcessorCollector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = collector.Collect()
	}
}

func BenchmarkCollectFromRegistry(b *testing.B) {
	collector := NewProcessorCollector()
	info := &models.ProcessorInfo{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = collector.collectFromRegistry(info)
	}
}

func BenchmarkEnrichProcessorTopology(b *testing.B) {
	collector := NewProcessorCollector()
	info := &models.ProcessorInfo{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.enrichProcessorTopology(info)
	}
}

func BenchmarkParseCache(b *testing.B) {
	collector := NewProcessorCollector()
	info := &models.ProcessorInfo{}
	structBytes := make([]byte, 16)
	structBytes[8] = 1
	binary.LittleEndian.PutUint32(structBytes[12:16], 64*1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.parseCache(structBytes, info)
	}
}
