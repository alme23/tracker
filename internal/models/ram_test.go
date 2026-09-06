// tracker/internal/models/ram_test.go
package models

import (
	"encoding/json"
	"testing"
)

// ============ Тесты для RAMInfo полей ============

func TestRAMInfoFields(t *testing.T) {
	ram := RAMInfo{
		TotalBytes:        42869710848,
		AvailableBytes:    23083425792,
		TotalPageFile:     68526669824,
		AvailablePageFile: 37414522880,
	}

	if ram.TotalBytes == 0 {
		t.Error("TotalBytes is 0")
	}

	if ram.AvailableBytes > ram.TotalBytes {
		t.Error("AvailableBytes > TotalBytes")
	}

	if ram.TotalPageFile == 0 {
		t.Error("TotalPageFile is 0")
	}
}

// ============ Тесты для RAMStick полей ============

func TestRAMStickFields(t *testing.T) {
	stick := RAMStick{
		Slot:         "DIMM1",
		Capacity:     8589934592,
		SpeedMHz:     1600,
		Manufacturer: "Kingston",
		SerialNumber: "SN123456",
		PartNumber:   "KVR16E11/8",
	}

	if stick.Slot == "" {
		t.Error("Slot is empty")
	}

	if stick.Capacity == 0 {
		t.Error("Capacity is 0")
	}

	if stick.Manufacturer == "" {
		t.Error("Manufacturer is empty")
	}
}

// ============ Тесты для RAMInfo JSON ============

func TestRAMInfoJSON(t *testing.T) {
	ram := RAMInfo{
		TotalBytes:        42869710848,
		AvailableBytes:    23083425792,
		TotalPageFile:     68526669824,
		AvailablePageFile: 37414522880,
		Sticks: []RAMStick{
			{
				Slot:         "DIMM1",
				Capacity:     8589934592,
				SpeedMHz:     1600,
				Manufacturer: "Kingston",
				PartNumber:   "KVR16E11/8",
			},
		},
	}

	data, err := json.Marshal(ram)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	t.Logf("JSON: %s", data)

	var restored RAMInfo
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if restored.TotalBytes != ram.TotalBytes {
		t.Errorf("TotalBytes: %d != %d", restored.TotalBytes, ram.TotalBytes)
	}

	if len(restored.Sticks) != len(ram.Sticks) {
		t.Errorf("Sticks count: %d != %d", len(restored.Sticks), len(ram.Sticks))
	}

	if restored.Sticks[0].Slot != ram.Sticks[0].Slot {
		t.Errorf("Stick Slot: %s != %s", restored.Sticks[0].Slot, ram.Sticks[0].Slot)
	}
}

// ============ Тесты для omitempty ============

func TestRAMInfoJSONOmitEmpty(t *testing.T) {
	ram := RAMInfo{
		TotalBytes: 42869710848,
	}

	data, err := json.Marshal(ram)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	jsonStr := string(data)

	// Sticks не должны быть в JSON, если пустые
	if containsSubstring(jsonStr, "sticks") {
		t.Error("sticks should be omitted when empty")
	}
}

// ============ Тесты для RAMStick JSON ============

func TestRAMStickJSON(t *testing.T) {
	stick := RAMStick{
		Slot:         "DIMM1",
		Capacity:     8589934592,
		SpeedMHz:     1600,
		Manufacturer: "Kingston",
		SerialNumber: "SN123456",
		PartNumber:   "KVR16E11/8",
	}

	data, err := json.Marshal(stick)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	t.Logf("JSON: %s", data)

	var restored RAMStick
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if restored.Slot != stick.Slot {
		t.Errorf("Slot: %s != %s", restored.Slot, stick.Slot)
	}

	if restored.Capacity != stick.Capacity {
		t.Errorf("Capacity: %d != %d", restored.Capacity, stick.Capacity)
	}

	if restored.SpeedMHz != stick.SpeedMHz {
		t.Errorf("SpeedMHz: %d != %d", restored.SpeedMHz, stick.SpeedMHz)
	}

	if restored.Manufacturer != stick.Manufacturer {
		t.Errorf("Manufacturer: %s != %s", restored.Manufacturer, stick.Manufacturer)
	}
}

// ============ Тесты на консистентность ============

func TestRAMInfoConsistency(t *testing.T) {
	ram := RAMInfo{
		TotalBytes:     42869710848,
		AvailableBytes: 23083425792,
	}

	// Использованная память
	usedBytes := ram.TotalBytes - ram.AvailableBytes
	if usedBytes > ram.TotalBytes {
		t.Error("Used bytes > total bytes")
	}

	// Процент использования
	usagePercent := float64(usedBytes) / float64(ram.TotalBytes) * 100
	if usagePercent < 0 || usagePercent > 100 {
		t.Errorf("Usage percent out of range: %.2f%%", usagePercent)
	}
}

// ============ Тесты на суммарный объем планок ============

func TestRAMSticksTotalCapacity(t *testing.T) {
	ram := RAMInfo{
		TotalBytes: 34359738368, // 32 GB
		Sticks: []RAMStick{
			{Slot: "DIMM1", Capacity: 8589934592}, // 8 GB
			{Slot: "DIMM2", Capacity: 8589934592}, // 8 GB
			{Slot: "DIMM3", Capacity: 8589934592}, // 8 GB
			{Slot: "DIMM4", Capacity: 8589934592}, // 8 GB
		},
	}

	var totalSticksCapacity uint64
	for _, stick := range ram.Sticks {
		totalSticksCapacity += stick.Capacity
	}

	if totalSticksCapacity != ram.TotalBytes {
		t.Errorf("Sticks capacity (%d) != TotalBytes (%d)",
			totalSticksCapacity, ram.TotalBytes)
	}
}

// ============ Бенчмарки ============

func BenchmarkRAMInfoJSON(b *testing.B) {
	ram := RAMInfo{
		TotalBytes:        42869710848,
		AvailableBytes:    23083425792,
		TotalPageFile:     68526669824,
		AvailablePageFile: 37414522880,
		Sticks: []RAMStick{
			{Slot: "DIMM1", Capacity: 8589934592, SpeedMHz: 1600},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = json.Marshal(ram)
	}
}

func BenchmarkRAMStickJSON(b *testing.B) {
	stick := RAMStick{
		Slot:         "DIMM1",
		Capacity:     8589934592,
		SpeedMHz:     1600,
		Manufacturer: "Kingston",
		PartNumber:   "KVR16E11/8",
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = json.Marshal(stick)
	}
}
