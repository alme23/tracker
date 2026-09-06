//go:build windows

package collector

import (
	"testing"
	"time"

	"github.com/alme23/tracker/internal/models"
)

// ============ Тесты для NewHostCollector() ============

func TestNewHostCollector(t *testing.T) {
	collector := NewHostCollector()

	if collector == nil {
		t.Fatal("NewHostCollector returned nil")
	}
}

// ============ Тесты для Collect() ============

func TestHostCollectorCollect(t *testing.T) {
	collector := NewHostCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Проверяем обязательные поля
	if info.Hostname == "" {
		t.Error("Hostname is empty")
	}

	if info.UpTimeSeconds == 0 {
		t.Error("UpTimeSeconds is 0")
	}

	if info.BootTime == 0 {
		t.Error("BootTime is 0")
	}

	if info.TimeZone == "" {
		t.Error("TimeZone is empty")
	}

	// Логируем
	t.Logf("Hostname: %s", info.Hostname)
	t.Logf("FQDN: %s", info.FQDN)
	t.Logf("PhysicalHostname: %s", info.PhysicalHostname)
	t.Logf("PhysicalFQDN: %s", info.PhysicalFQDN)
	t.Logf("Domain: %s", info.Domain)
	t.Logf("Workgroup: %s", info.Workgroup)
	t.Logf("UpTime: %d seconds", info.UpTimeSeconds)
	t.Logf("BootTime: %d (%s)", info.BootTime, time.Unix(int64(info.BootTime), 0).Format("2006-01-02 15:04:05"))
	t.Logf("TimeZone: %s (offset %d)", info.TimeZone, info.TimeZoneOffset)
	t.Logf("Manufacturer: %s", info.Manufacturer)
	t.Logf("Model: %s", info.Model)
}

// ============ Тесты для getComputerName() ============

func TestHostGetComputerName(t *testing.T) {
	collector := NewHostCollector()

	tests := []struct {
		name     string
		nameType uint32
	}{
		{"NetBIOS", computerNameNetBIOS},
		{"DNS Hostname", computerNameDNSHostname},
		{"DNS Domain", computerNameDNSDomain},
		{"DNS Fully Qualified", computerNameDNSFullyQualified},
		{"Physical NetBIOS", computerNamePhysicalNetBIOS},
		{"Physical DNS Hostname", computerNamePhysicalDNSHostname},
		{"Physical DNS Domain", computerNamePhysicalDNSDomain},
		{"Physical DNS Fully Qualified", computerNamePhysicalDNSFullyQualified},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.getComputerName(tt.nameType)
			t.Logf("%s: %s", tt.name, result)

			if result == "" {
				t.Logf("%s returned empty (may be normal)", tt.name)
			}
		})
	}
}

func TestHostGetComputerNameEmpty(t *testing.T) {
	collector := NewHostCollector()

	// Передаем невалидный тип
	result := collector.getComputerName(999)

	if result != "" {
		t.Errorf("Invalid name type should return empty, got: %s", result)
	}
}

// ============ Тесты для getWorkgroup() ============

func TestHostGetWorkgroup(t *testing.T) {
	collector := NewHostCollector()

	workgroup := collector.getWorkgroup()

	t.Logf("Workgroup: %s", workgroup)

	// Рабочая группа может быть пустой (если в домене)
	if workgroup == "" {
		t.Log("Workgroup is empty (may be in domain)")
	}
}

// ============ Тесты для getUpTimeSeconds() ============

func TestHostGetUpTimeSeconds(t *testing.T) {
	collector := NewHostCollector()

	uptime := collector.getUpTimeSeconds()

	t.Logf("Uptime: %d seconds", uptime)

	if uptime == 0 {
		t.Error("Uptime is 0")
	}

	// Проверяем, что аптайм не слишком большой (больше 100 лет)
	if uptime > 100*365*24*3600 {
		t.Errorf("Uptime is too large: %d", uptime)
	}
}

// ============ Тесты для getTimeZone() ============

func TestHostGetTimeZone(t *testing.T) {
	collector := NewHostCollector()

	zoneName, offset := collector.getTimeZone()

	t.Logf("TimeZone: %s", zoneName)
	t.Logf("Offset: %d minutes", offset)

	if zoneName == "" {
		t.Error("TimeZone name is empty")
	}

	// Проверяем диапазон смещения (UTC-12 до UTC+14)
	if offset < -12*60 || offset > 14*60 {
		t.Errorf("TimeZone offset out of range: %d", offset)
	}
}

// ============ Тесты для collectHardwareInfo() ============

func TestHostCollectHardwareInfo(t *testing.T) {
	collector := NewHostCollector()
	info := &models.HostInfo{}

	collector.collectHardwareInfo(info)

	t.Logf("Manufacturer: %s", info.Manufacturer)
	t.Logf("Model: %s", info.Model)
	t.Logf("SKU: %s", info.SKU)
	t.Logf("Family: %s", info.Family)
	t.Logf("Version: %s", info.Version)
	t.Logf("Serial: %s", info.SerialNumber)
	t.Logf("BIOS Vendor: %s", info.BIOSVendor)
	t.Logf("BIOS Version: %s", info.BIOSVersion)
	t.Logf("BIOS Date: %s", info.BIOSDate)
	t.Logf("BIOS Major: %d", info.BIOSMajorRelease)
	t.Logf("BIOS Minor: %d", info.BIOSMinorRelease)
	t.Logf("BaseBoard Manufacturer: %s", info.BaseBoardManufacturer)
	t.Logf("BaseBoard Product: %s", info.BaseBoardProduct)
	t.Logf("BaseBoard Version: %s", info.BaseBoardVersion)

	// Проверяем, что хотя бы что-то заполнено
	if info.Manufacturer == "" && info.Model == "" && info.BIOSVendor == "" {
		t.Error("All hardware info is empty")
	}
}

// ============ Тесты на консистентность ============

func TestHostCollectorConsistency(t *testing.T) {
	collector := NewHostCollector()

	first, err := collector.Collect()
	if err != nil {
		t.Fatalf("First Collect failed: %v", err)
	}

	second, err := collector.Collect()
	if err != nil {
		t.Fatalf("Second Collect failed: %v", err)
	}

	if first.Hostname != second.Hostname {
		t.Errorf("Hostname changed: %s vs %s", first.Hostname, second.Hostname)
	}

	if first.BootTime != second.BootTime {
		t.Errorf("BootTime changed: %d vs %d", first.BootTime, second.BootTime)
	}

	if first.TimeZoneOffset != second.TimeZoneOffset {
		t.Errorf("TimeZoneOffset changed: %d vs %d", first.TimeZoneOffset, second.TimeZoneOffset)
	}
}

// ============ Тесты на конкурентность ============

func TestHostCollectorConcurrent(t *testing.T) {
	collector := NewHostCollector()

	const numGoroutines = 10
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := collector.Collect()
			errChan <- err
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("Concurrent Collect failed: %v", err)
		}
	}
}

// ============ Бенчмарки ============

func BenchmarkHostCollectorCollect(b *testing.B) {
	collector := NewHostCollector()

	b.ResetTimer()
	for b.Loop() {
		_, _ = collector.Collect()
	}
}

func BenchmarkHostGetComputerName(b *testing.B) {
	collector := NewHostCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.getComputerName(computerNameDNSHostname)
	}
}

func BenchmarkHostGetUpTimeSeconds(b *testing.B) {
	collector := NewHostCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.getUpTimeSeconds()
	}
}

func BenchmarkHostGetTimeZone(b *testing.B) {
	collector := NewHostCollector()

	b.ResetTimer()
	for b.Loop() {
		_, _ = collector.getTimeZone()
	}
}

func BenchmarkHostCollectHardwareInfo(b *testing.B) {
	collector := NewHostCollector()
	info := &models.HostInfo{}

	b.ResetTimer()
	for b.Loop() {
		collector.collectHardwareInfo(info)
	}
}
