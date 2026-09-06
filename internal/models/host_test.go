// tracker/internal/models/host_test.go
package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// ============ Тесты для HostInfo полей ============

func TestHostInfoFields(t *testing.T) {
	host := HostInfo{
		Hostname:      "IT-S",
		FQDN:          "IT-S.rsa.rogsibal.ru",
		Domain:        "rsa.rogsibal.ru",
		UpTimeSeconds: 9798016,
		BootTime:      1747109492,
		Manufacturer:  "Hewlett-Packard",
		Model:         "HP Z420 Workstation",
		BIOSVendor:    "Hewlett-Packard",
		BIOSVersion:   "J61 v03.85",
	}

	if host.Hostname == "" {
		t.Error("Hostname is empty")
	}

	if host.BootTime == 0 {
		t.Error("BootTime is 0")
	}

	if host.Manufacturer == "" {
		t.Error("Manufacturer is empty")
	}
}

// ============ Тесты для GetBootTime() ============

func TestGetBootTime(t *testing.T) {
	tests := []struct {
		name     string
		bootTime uint64
		expected time.Time
	}{
		{
			name:     "Zero",
			bootTime: 0,
			expected: time.Time{},
		},
		{
			name:     "Valid timestamp",
			bootTime: 1747109492,
			expected: time.Unix(1747109492, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := HostInfo{BootTime: tt.bootTime}
			result := host.GetBootTime()

			if !result.Equal(tt.expected) {
				t.Errorf("GetBootTime() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// ============ Тесты для GetBootTimeString() ============

func TestGetBootTimeString(t *testing.T) {
	tests := []struct {
		name     string
		bootTime uint64
		expected string
	}{
		{
			name:     "Zero",
			bootTime: 0,
			expected: "UNKNOWN",
		},
		{
			name:     "Valid timestamp",
			bootTime: 1747109492,
			expected: "2025-05-13 07:11:32",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := HostInfo{BootTime: tt.bootTime}
			result := host.GetBootTimeString()

			if result != tt.expected {
				t.Errorf("GetBootTimeString() = %s, want %s", result, tt.expected)
			}
		})
	}
}

// ============ Тесты для GetUpTime() ============

func TestGetUpTime(t *testing.T) {
	host := HostInfo{UpTimeSeconds: 3600}

	duration := host.GetUpTime()

	if duration != time.Hour {
		t.Errorf("GetUpTime() = %v, want %v", duration, time.Hour)
	}
}

// ============ Тесты для GetUpTimeString() ============

func TestGetUpTimeString(t *testing.T) {
	tests := []struct {
		name     string
		uptime   uint64
		expected string
	}{
		{
			name:     "Minutes only",
			uptime:   300,
			expected: "5 minutes",
		},
		{
			name:     "Hours and minutes",
			uptime:   9000,
			expected: "2 hours, 30 minutes",
		},
		{
			name:     "Days, hours, minutes",
			uptime:   9798016,
			expected: "113 days, 9 hours, 40 minutes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := HostInfo{UpTimeSeconds: tt.uptime}
			result := host.GetUpTimeString()

			if result != tt.expected {
				t.Errorf("GetUpTimeString() = %s, want %s", result, tt.expected)
			}
		})
	}
}

// ============ Тесты для HostInfo JSON ============

func TestHostInfoJSON(t *testing.T) {
	host := HostInfo{
		Hostname:       "IT-S",
		FQDN:           "IT-S.rsa.rogsibal.ru",
		Domain:         "rsa.rogsibal.ru",
		UpTimeSeconds:  9798016,
		BootTime:       1747109492,
		Manufacturer:   "Hewlett-Packard",
		Model:          "HP Z420 Workstation",
		TimeZoneOffset: 180,
	}

	data, err := json.Marshal(host)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	t.Logf("JSON: %s", data)

	var restored HostInfo
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if restored.Hostname != host.Hostname {
		t.Errorf("Hostname: %s != %s", restored.Hostname, host.Hostname)
	}

	if restored.BootTime != host.BootTime {
		t.Errorf("BootTime: %d != %d", restored.BootTime, host.BootTime)
	}

	if restored.UpTimeSeconds != host.UpTimeSeconds {
		t.Errorf("UpTimeSeconds: %d != %d", restored.UpTimeSeconds, host.UpTimeSeconds)
	}

	if restored.TimeZoneOffset != host.TimeZoneOffset {
		t.Errorf("TimeZoneOffset: %d != %d", restored.TimeZoneOffset, host.TimeZoneOffset)
	}
}

// ============ Тесты на проверку JSON полей ============

func TestHostInfoJSONFields(t *testing.T) {
	host := HostInfo{
		Hostname: "TestHost",
	}

	data, err := json.Marshal(host)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	jsonStr := string(data)

	// Проверяем наличие обязательных полей
	requiredFields := []string{
		"hostname",
		"fqdn",
		"physical_hostname",
		"physical_fqdn",
		"domain",
		"workgroup",
		"uptime_seconds",
		"boot_time",
		"timezone",
		"timezone_offset",
		"manufacturer",
		"model",
		"sku",
		"family",
		"version",
		"serial_number",
		"bios_vendor",
		"bios_version",
		"bios_date",
		"bios_major_release",
		"bios_minor_release",
		"baseboard_manufacturer",
		"baseboard_product",
		"baseboard_version",
	}

	for _, field := range requiredFields {
		if !strings.Contains(jsonStr, `"`+field+`"`) {
			t.Errorf("JSON missing field: %s", field)
		}
	}
}

// ============ Бенчмарки ============

func BenchmarkGetBootTime(b *testing.B) {
	host := HostInfo{BootTime: 1747109492}

	b.ResetTimer()
	for b.Loop() {
		_ = host.GetBootTime()
	}
}

func BenchmarkGetBootTimeString(b *testing.B) {
	host := HostInfo{BootTime: 1747109492}

	b.ResetTimer()
	for b.Loop() {
		_ = host.GetBootTimeString()
	}
}

func BenchmarkGetUpTimeString(b *testing.B) {
	host := HostInfo{UpTimeSeconds: 9798016}

	b.ResetTimer()
	for b.Loop() {
		_ = host.GetUpTimeString()
	}
}

func BenchmarkHostInfoJSON(b *testing.B) {
	host := HostInfo{
		Hostname:     "IT-S",
		FQDN:         "IT-S.rsa.rogsibal.ru",
		Manufacturer: "Hewlett-Packard",
		Model:        "HP Z420 Workstation",
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = json.Marshal(host)
	}
}
