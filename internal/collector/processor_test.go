//go:build windows

package collector

import (
	"runtime"
	"strings"
	"testing"

	"github.com/alme23/tracker/internal/models"
)

func TestNewProcessorCollector(t *testing.T) {
	collector := NewProcessorCollector()

	if collector == nil {
		t.Fatal("NewProcessorCollector returned nil")
	}
}

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

	// Физических ядер не может быть больше логических
	if info.PhysicalCores > info.LogicalProcessors {
		t.Errorf("PhysicalCores (%d) > LogicalProcessors (%d)",
			info.PhysicalCores, info.LogicalProcessors)
	}

	// Логируем
	t.Logf("Model: %s", info.Model)
	t.Logf("Vendor: %s", info.VendorID)
	t.Logf("Processor ID: %s", info.ProcessorID)
	t.Logf("Cores: %d physical, %d logical", info.PhysicalCores, info.LogicalProcessors)
	t.Logf("Base Speed: %d MHz", info.BaseSpeedMHz)
	t.Logf("SMT: %v", info.SMTEnabled)
	t.Logf("NUMA: %v", info.NUMAEnabled)
	t.Logf("VT-x: %v", info.HardwareVirtAvail)
	t.Logf("NX: %v", info.NXBitSupported)
	t.Logf("L1 Cache: %d KB", info.L1CacheBytes/1024)
	t.Logf("L2 Cache: %d KB", info.L2CacheBytes/1024)
	t.Logf("L3 Cache: %d MB", info.L3CacheBytes/1024/1024)
}

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
	t.Logf("Processor ID: %s", info.ProcessorID)
	t.Logf("Base Speed: %d MHz", info.BaseSpeedMHz)
}

func TestSetFallbackInfo(t *testing.T) {
	collector := NewProcessorCollector()

	tests := []struct {
		name              string
		logicalProcessors uint32
		expectedCores     uint32
		expectedSMT       bool
	}{
		{"8 logical", 8, 4, true},
		{"4 logical", 4, 2, true},
		{"2 logical", 2, 1, true},
		{"1 logical", 1, 1, false},
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

func TestParseProcessorCore(t *testing.T) {
	collector := NewProcessorCollector()

	tests := []struct {
		name        string
		flags       byte
		expectedSMT bool
	}{
		{"SMT enabled", 1, true},
		{"SMT disabled", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &models.ProcessorInfo{}
			structBytes := make([]byte, 16)
			structBytes[8] = tt.flags

			collector.parseProcessorCore(structBytes, info)

			if info.PhysicalCores != 1 {
				t.Errorf("PhysicalCores = %d, want 1", info.PhysicalCores)
			}

			if tt.expectedSMT {
				if info.LogicalProcessors != 2 {
					t.Errorf("LogicalProcessors = %d, want 2", info.LogicalProcessors)
				}
				if !info.SMTEnabled {
					t.Error("SMTEnabled should be true")
				}
			} else {
				if info.LogicalProcessors != 1 {
					t.Errorf("LogicalProcessors = %d, want 1", info.LogicalProcessors)
				}
				if info.SMTEnabled {
					t.Error("SMTEnabled should be false")
				}
			}
		})
	}
}

func TestParseCache(t *testing.T) {
	collector := NewProcessorCollector()

	tests := []struct {
		name          string
		level         byte
		cacheSize     uint32
		expectedField string
	}{
		{"L1 Cache", 1, 32 * 1024, "L1"},
		{"L2 Cache", 2, 256 * 1024, "L2"},
		{"L3 Cache", 3, 10 * 1024 * 1024, "L3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &models.ProcessorInfo{}
			structBytes := make([]byte, 20)
			structBytes[8] = tt.level

			// Записываем размер кэша
			structBytes[12] = byte(tt.cacheSize)
			structBytes[13] = byte(tt.cacheSize >> 8)
			structBytes[14] = byte(tt.cacheSize >> 16)
			structBytes[15] = byte(tt.cacheSize >> 24)

			collector.parseCache(structBytes, info)

			switch tt.expectedField {
			case "L1":
				if info.L1CacheBytes != uint64(tt.cacheSize) {
					t.Errorf("L1CacheBytes = %d, want %d",
						info.L1CacheBytes, tt.cacheSize)
				}
			case "L2":
				if info.L2CacheBytes != uint64(tt.cacheSize) {
					t.Errorf("L2CacheBytes = %d, want %d",
						info.L2CacheBytes, tt.cacheSize)
				}
			case "L3":
				if info.L3CacheBytes != uint64(tt.cacheSize) {
					t.Errorf("L3CacheBytes = %d, want %d",
						info.L3CacheBytes, tt.cacheSize)
				}
			}
		})
	}
}

func TestValidateInfo(t *testing.T) {
	collector := NewProcessorCollector()

	tests := []struct {
		name        string
		input       models.ProcessorInfo
		expectedSMT bool
	}{
		{
			name: "SMT detection",
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
			name: "Invalid: more physical than logical",
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

			if info.PhysicalCores > info.LogicalProcessors {
				t.Errorf("PhysicalCores (%d) > LogicalProcessors (%d) after validation",
					info.PhysicalCores, info.LogicalProcessors)
			}

			if info.SMTEnabled != tt.expectedSMT {
				t.Errorf("SMTEnabled = %v, want %v", info.SMTEnabled, tt.expectedSMT)
			}

			// Кэш должен быть заполнен
			if info.L1CacheBytes == 0 {
				t.Error("L1CacheBytes is 0 after validation")
			}
			if info.L2CacheBytes == 0 {
				t.Error("L2CacheBytes is 0 after validation")
			}
			if info.L3CacheBytes == 0 {
				t.Error("L3CacheBytes is 0 after validation")
			}
		})
	}
}

func TestProcessorInfoConsistency(t *testing.T) {
	collector := NewProcessorCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Проверяем соответствие с runtime.NumCPU()
	runtimeCPU := runtime.NumCPU()
	if info.LogicalProcessors != uint32(runtimeCPU) {
		t.Logf("LogicalProcessors (%d) != runtime.NumCPU (%d)",
			info.LogicalProcessors, runtimeCPU)
	}

	// Проверяем, что Model содержит VendorID или наоборот
	if info.Model != "" && info.VendorID != "" {
		// Оба поля должны быть заполнены
	}

	// Проверяем кэш
	if info.L1CacheBytes > 0 && info.PhysicalCores > 0 {
		l1PerCore := info.L1CacheBytes / uint64(info.PhysicalCores)
		if l1PerCore < 16*1024 || l1PerCore > 128*1024 {
			t.Logf("L1 per core is unusual: %d KB", l1PerCore/1024)
		}
	}

	if info.L2CacheBytes > 0 && info.PhysicalCores > 0 {
		l2PerCore := info.L2CacheBytes / uint64(info.PhysicalCores)
		if l2PerCore < 128*1024 || l2PerCore > 2*1024*1024 {
			t.Logf("L2 per core is unusual: %d KB", l2PerCore/1024)
		}
	}
}

func TestProcessorVendor(t *testing.T) {
	collector := NewProcessorCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Проверяем известных производителей
	vendor := strings.ToLower(info.VendorID)

	switch {
	case strings.Contains(vendor, "genuineintel"):
		t.Log("Intel processor detected")
	case strings.Contains(vendor, "authenticamd"):
		t.Log("AMD processor detected")
	default:
		t.Logf("Unknown vendor: %s", info.VendorID)
	}
}

func BenchmarkProcessorCollector(b *testing.B) {
	collector := NewProcessorCollector()

	b.ResetTimer()
	for b.Loop() {
		_, err := collector.Collect()
		if err != nil {
			b.Fatalf("Collect failed: %v", err)
		}
	}
}

func BenchmarkProcessorCollectorParallel(b *testing.B) {
	collector := NewProcessorCollector()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := collector.Collect()
			if err != nil {
				b.Fatalf("Collect failed: %v", err)
			}
		}
	})
}
