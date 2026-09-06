// tracker/internal/models/processor_test.go
package models

import (
	"encoding/json"
	"testing"
)

// ============ Тесты для ProcessorInfo полей ============

func TestProcessorInfoFields(t *testing.T) {
	processor := ProcessorInfo{
		Model:             "Intel(R) Xeon(R) CPU E5-1620 0 @ 3.60GHz",
		VendorID:          "GenuineIntel",
		ProcessorID:       "Intel64 Family 6 Model 45 Stepping 7",
		PhysicalCores:     4,
		LogicalProcessors: 8,
		BaseSpeedMHz:      3591,
		L1CacheBytes:      262144,
		L2CacheBytes:      1048576,
		L3CacheBytes:      10485760,
	}

	if processor.Model == "" {
		t.Error("Model is empty")
	}

	if processor.PhysicalCores == 0 {
		t.Error("PhysicalCores is 0")
	}

	if processor.LogicalProcessors == 0 {
		t.Error("LogicalProcessors is 0")
	}
}

// ============ Тесты на валидацию данных ============

func TestProcessorInfoValidation(t *testing.T) {
	processor := ProcessorInfo{
		PhysicalCores:     4,
		LogicalProcessors: 8,
	}

	// Физических ядер не может быть больше логических
	if processor.PhysicalCores > processor.LogicalProcessors {
		t.Error("PhysicalCores > LogicalProcessors")
	}

	// Логических не может быть меньше физических
	if processor.LogicalProcessors < processor.PhysicalCores {
		t.Error("LogicalProcessors < PhysicalCores")
	}
}

// ============ Тесты на SMT ============

func TestProcessorInfoSMT(t *testing.T) {
	tests := []struct {
		name              string
		physicalCores     uint32
		logicalProcessors uint32
		expectedSMT       bool
	}{
		{"With SMT", 4, 8, true},
		{"Without SMT", 4, 4, false},
		{"Single core", 1, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := ProcessorInfo{
				PhysicalCores:     tt.physicalCores,
				LogicalProcessors: tt.logicalProcessors,
			}

			smtEnabled := processor.LogicalProcessors > processor.PhysicalCores

			if smtEnabled != tt.expectedSMT {
				t.Errorf("SMT = %v, want %v", smtEnabled, tt.expectedSMT)
			}
		})
	}
}

// ============ Тесты на кэш ============

func TestProcessorInfoCache(t *testing.T) {
	processor := ProcessorInfo{
		PhysicalCores: 4,
		L1CacheBytes:  262144,   // 256 KB
		L2CacheBytes:  1048576,  // 1 MB
		L3CacheBytes:  10485760, // 10 MB
	}

	// L1 per core
	l1PerCore := processor.L1CacheBytes / uint64(processor.PhysicalCores)
	if l1PerCore != 65536 { // 64 KB
		t.Errorf("L1 per core = %d, want 65536", l1PerCore)
	}

	// L2 per core
	l2PerCore := processor.L2CacheBytes / uint64(processor.PhysicalCores)
	if l2PerCore != 262144 { // 256 KB
		t.Errorf("L2 per core = %d, want 262144", l2PerCore)
	}
}

// ============ Тесты для JSON ============

func TestProcessorInfoJSON(t *testing.T) {
	processor := ProcessorInfo{
		Model:             "Intel Xeon E5-1620",
		VendorID:          "GenuineIntel",
		PhysicalCores:     4,
		LogicalProcessors: 8,
		BaseSpeedMHz:      3591,
		HardwareVirtAvail: true,
		NXBitSupported:    true,
		SMTEnabled:        true,
		L1CacheBytes:      262144,
		L2CacheBytes:      1048576,
		L3CacheBytes:      10485760,
	}

	data, err := json.Marshal(processor)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	t.Logf("JSON: %s", data)

	var restored ProcessorInfo
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if restored.Model != processor.Model {
		t.Errorf("Model: %s != %s", restored.Model, processor.Model)
	}

	if restored.PhysicalCores != processor.PhysicalCores {
		t.Errorf("PhysicalCores: %d != %d", restored.PhysicalCores, processor.PhysicalCores)
	}

	if restored.LogicalProcessors != processor.LogicalProcessors {
		t.Errorf("LogicalProcessors: %d != %d", restored.LogicalProcessors, processor.LogicalProcessors)
	}

	if restored.L1CacheBytes != processor.L1CacheBytes {
		t.Errorf("L1CacheBytes: %d != %d", restored.L1CacheBytes, processor.L1CacheBytes)
	}

	if restored.SMTEnabled != processor.SMTEnabled {
		t.Errorf("SMTEnabled: %v != %v", restored.SMTEnabled, processor.SMTEnabled)
	}
}

// ============ Тесты на JSON поля ============

func TestProcessorInfoJSONFields(t *testing.T) {
	processor := ProcessorInfo{}

	data, err := json.Marshal(processor)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	jsonStr := string(data)

	requiredFields := []string{
		"model",
		"vendor_id",
		"processor_id",
		"physical_cores",
		"logical_processors",
		"base_speed_mhz",
		"hardware_virt_available",
		"nx_bit_supported",
		"smt_enabled",
		"numa_enabled",
		"l1_cache_bytes",
		"l2_cache_bytes",
		"l3_cache_bytes",
	}

	for _, field := range requiredFields {
		if !containsJSONField(jsonStr, field) {
			t.Errorf("JSON missing field: %s", field)
		}
	}
}

// ============ Бенчмарки ============

func BenchmarkProcessorInfoJSON(b *testing.B) {
	processor := ProcessorInfo{
		Model:             "Intel Xeon E5-1620",
		PhysicalCores:     4,
		LogicalProcessors: 8,
		L1CacheBytes:      262144,
		L2CacheBytes:      1048576,
		L3CacheBytes:      10485760,
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = json.Marshal(processor)
	}
}

// Вспомогательная функция
func containsJSONField(jsonStr, field string) bool {
	return len(jsonStr) >= len(field)+2 &&
		(jsonStr == field ||
			len(jsonStr) > 0 && containsSubstring(jsonStr, `"`+field+`"`))
}
