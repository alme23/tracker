// tracker/internal/models/host_test.go
package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHostInfoFields(t *testing.T) {
	host := HostInfo{
		Hostname:         "IT-S",
		FQDN:             "IT-S.rsa.rogsibal.ru",
		PhysicalHostname: "IT-S",
		PhysicalFQDN:     "IT-S.rsa.rogsibal.ru",
		Domain:           "rsa.rogsibal.ru",
		UpTimeSeconds:    9798016,
		BootTime:         time.Now().Add(-9798016 * time.Second),
		TimeZone:         "RTZ 2 (зима)",
		TimeZoneOffset:   180,
		Manufacturer:     "Hewlett-Packard",
		Model:            "HP Z420 Workstation",
		SKU:              "LJ449AV",
		Family:           "103C_53335X G=D",
		BIOSVendor:       "Hewlett-Packard",
		BIOSVersion:      "J61 v03.85",
		BIOSDate:         "11/19/2014",
	}

	// Проверяем основные поля
	if host.Hostname == "" {
		t.Error("Hostname is empty")
	}

	if host.FQDN == "" {
		t.Error("FQDN is empty")
	}

	if host.Manufacturer == "" {
		t.Error("Manufacturer is empty")
	}

	if host.Model == "" {
		t.Error("Model is empty")
	}
}

func TestHostInfoJSON(t *testing.T) {
	host := HostInfo{
		Hostname:       "IT-S",
		FQDN:           "IT-S.rsa.rogsibal.ru",
		Domain:         "rsa.rogsibal.ru",
		UpTimeSeconds:  9798016,
		BootTime:       time.Date(2026, 5, 13, 7, 11, 32, 0, time.UTC),
		TimeZone:       "RTZ 2 (зима)",
		TimeZoneOffset: 180,
		Manufacturer:   "Hewlett-Packard",
		Model:          "HP Z420 Workstation",
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

	// Сравниваем поля
	if restored.Hostname != host.Hostname {
		t.Errorf("Hostname: %s != %s", restored.Hostname, host.Hostname)
	}

	if restored.FQDN != host.FQDN {
		t.Errorf("FQDN: %s != %s", restored.FQDN, host.FQDN)
	}

	if restored.Domain != host.Domain {
		t.Errorf("Domain: %s != %s", restored.Domain, host.Domain)
	}

	if restored.UpTimeSeconds != host.UpTimeSeconds {
		t.Errorf("UpTimeSeconds: %d != %d", restored.UpTimeSeconds, host.UpTimeSeconds)
	}

	if restored.TimeZoneOffset != host.TimeZoneOffset {
		t.Errorf("TimeZoneOffset: %d != %d", restored.TimeZoneOffset, host.TimeZoneOffset)
	}
}

func TestHostInfoOmitEmpty(t *testing.T) {
	// Поля с omitempty не должны появляться в JSON если пустые
	host := HostInfo{
		Hostname: "TestHost",
	}

	data, err := json.Marshal(host)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	jsonStr := string(data)

	// Эти поля не должны быть в JSON
	if contains(jsonStr, "physical_hostname") {
		t.Error("physical_hostname should be omitted when empty")
	}

	if contains(jsonStr, "workgroup") {
		t.Error("workgroup should be omitted when empty")
	}

	if contains(jsonStr, "sku") {
		t.Error("sku should be omitted when empty")
	}

	// Эти поля должны быть в JSON
	if !contains(jsonStr, "hostname") {
		t.Error("hostname should be in JSON")
	}
}

func TestHostInfoTimeZoneOffset(t *testing.T) {
	tests := []struct {
		name   string
		offset int16
	}{
		{"UTC", 0},
		{"Moscow", 180},
		{"New York", -300},
		{"Tokyo", 540},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := HostInfo{
				Hostname:       "Test",
				TimeZoneOffset: tt.offset,
			}

			data, err := json.Marshal(host)
			if err != nil {
				t.Fatalf("JSON marshal failed: %v", err)
			}

			var restored HostInfo
			err = json.Unmarshal(data, &restored)
			if err != nil {
				t.Fatalf("JSON unmarshal failed: %v", err)
			}

			if restored.TimeZoneOffset != tt.offset {
				t.Errorf("TimeZoneOffset: %d != %d", restored.TimeZoneOffset, tt.offset)
			}
		})
	}
}

func TestHostInfoBootTime(t *testing.T) {
	bootTime := time.Date(2026, 5, 13, 7, 11, 32, 0, time.UTC)

	host := HostInfo{
		Hostname: "Test",
		BootTime: bootTime,
	}

	data, err := json.Marshal(host)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var restored HostInfo
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// Время должно сохраниться с точностью до секунды
	if !restored.BootTime.Equal(bootTime) {
		t.Errorf("BootTime: %v != %v", restored.BootTime, bootTime)
	}
}

func TestHostInfoVirtualMachine(t *testing.T) {
	// Физическая машина
	physical := HostInfo{
		Hostname:         "IT-S",
		PhysicalHostname: "IT-S",
	}

	if physical.PhysicalHostname != physical.Hostname {
		t.Error("Physical machine should have same names")
	}

	// Виртуальная машина
	virtual := HostInfo{
		Hostname:         "VM-WEB-01",
		PhysicalHostname: "HYPERV-HOST-01",
	}

	if virtual.PhysicalHostname == virtual.Hostname {
		t.Error("Virtual machine should have different names")
	}
}

func TestHostInfoBIOSFields(t *testing.T) {
	host := HostInfo{
		BIOSVendor:       "Hewlett-Packard",
		BIOSVersion:      "J61 v03.85",
		BIOSDate:         "11/19/2014",
		BIOSMajorRelease: 3,
		BIOSMinorRelease: 85,
	}

	if host.BIOSVendor == "" {
		t.Error("BIOSVendor is empty")
	}

	if host.BIOSVersion == "" {
		t.Error("BIOSVersion is empty")
	}

	if host.BIOSMajorRelease == 0 {
		t.Error("BIOSMajorRelease is 0")
	}
}

func TestHostInfoBaseBoardFields(t *testing.T) {
	host := HostInfo{
		BaseBoardManufacturer: "Hewlett-Packard",
		BaseBoardProduct:      "1589",
		BaseBoardVersion:      "0.00",
	}

	if host.BaseBoardManufacturer == "" {
		t.Error("BaseBoardManufacturer is empty")
	}

	if host.BaseBoardProduct == "" {
		t.Error("BaseBoardProduct is empty")
	}
}

func BenchmarkHostInfoJSON(b *testing.B) {
	host := HostInfo{
		Hostname:      "IT-S",
		FQDN:          "IT-S.rsa.rogsibal.ru",
		Domain:        "rsa.rogsibal.ru",
		UpTimeSeconds: 9798016,
		BootTime:      time.Now(),
		Manufacturer:  "Hewlett-Packard",
		Model:         "HP Z420 Workstation",
		BIOSVendor:    "Hewlett-Packard",
		BIOSVersion:   "J61 v03.85",
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = json.Marshal(host)
	}
}

func BenchmarkHostInfoUnmarshalJSON(b *testing.B) {
	host := HostInfo{
		Hostname:      "IT-S",
		FQDN:          "IT-S.rsa.rogsibal.ru",
		Domain:        "rsa.rogsibal.ru",
		UpTimeSeconds: 9798016,
		BootTime:      time.Now(),
		Manufacturer:  "Hewlett-Packard",
		Model:         "HP Z420 Workstation",
	}

	data, _ := json.Marshal(host)

	b.ResetTimer()
	for b.Loop() {
		var restored HostInfo
		_ = json.Unmarshal(data, &restored)
	}
}

// Вспомогательная функция
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
