//go:build windows

package collector

import (
	"fmt"
	"os"
	"testing"
	"time"
	"unsafe"

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

	if info.BootTime.IsZero() {
		t.Error("BootTime is zero")
	}

	if info.TimeZone == "" {
		t.Error("TimeZone is empty")
	}

	// Логируем
	t.Logf("Hostname: %s", info.Hostname)
	t.Logf("FQDN: %s", info.FQDN)
	t.Logf("Domain: %s", info.Domain)
	t.Logf("Uptime: %d seconds", info.UpTimeSeconds)
	t.Logf("TimeZone: %s (offset %d)", info.TimeZone, info.TimeZoneOffset)
}

// ============ Тесты для getComputerName() ============

func TestGetComputerName(t *testing.T) {
	collector := NewHostCollector()

	tests := []struct {
		name     string
		nameType uint32
	}{
		{"NetBIOS", computerNameNetBIOS},
		{"DNS Hostname", computerNameDnsHostname},
		{"DNS Domain", computerNameDnsDomain},
		{"DNS Fully Qualified", computerNameDnsFullyQualified},
		{"Physical NetBIOS", computerNamePhysicalNetBIOS},
		{"Physical DNS Hostname", computerNamePhysicalDnsHostname},
		{"Physical DNS Domain", computerNamePhysicalDnsDomain},
		{"Physical DNS Fully Qualified", computerNamePhysicalDnsFullyQualified},
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

func TestGetComputerNameDNS(t *testing.T) {
	collector := NewHostCollector()

	hostname := collector.getComputerName(computerNameDnsHostname)

	if hostname == "" {
		t.Error("DNS hostname is empty")
	}

	// Сравниваем с os.Hostname()
	osHostname, _ := os.Hostname()
	t.Logf("DNS: %s, os.Hostname: %s", hostname, osHostname)
}

// ============ Тесты для getWorkgroup() ============

func TestGetWorkgroup(t *testing.T) {
	collector := NewHostCollector()

	workgroup := collector.getWorkgroup()

	t.Logf("Workgroup: %s", workgroup)

	// Рабочая группа может быть пустой (если в домене)
	if workgroup == "" {
		t.Log("Workgroup is empty (may be in domain)")
	}
}

// ============ Тесты для getUpTimeSeconds() ============

func TestGetUpTimeSeconds(t *testing.T) {
	collector := NewHostCollector()

	uptime := collector.getUpTimeSeconds()

	t.Logf("Uptime: %d seconds (%s)", uptime, formatDuration(uptime))

	if uptime == 0 {
		t.Error("Uptime is 0")
	}

	// Проверяем, что uptime не слишком большой (больше 100 лет)
	if uptime > 100*365*24*3600 {
		t.Errorf("Uptime is too large: %d", uptime)
	}
}

// ============ Тесты для getTimeZone() ============

func TestGetTimeZone(t *testing.T) {
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

func TestGetTimeZoneDirectCall(t *testing.T) {
	var tzInfo timeZoneInformation

	ret, _, _ := procGetTimeZoneInformation.Call(
		uintptr(unsafe.Pointer(&tzInfo)),
	)

	t.Logf("GetTimeZoneInformation returned: %d", ret)
	t.Logf("Bias: %d", tzInfo.Bias)
	t.Logf("StandardBias: %d", tzInfo.StandardBias)
	t.Logf("DaylightBias: %d", tzInfo.DaylightBias)

	if ret == 0xFFFFFFFF {
		t.Error("GetTimeZoneInformation failed")
	}
}

// ============ Тесты для collectHardwareInfo() ============

func TestCollectHardwareInfo(t *testing.T) {
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

// ============ Вспомогательные функции ============

func formatDuration(seconds uint64) string {
	duration := time.Duration(seconds) * time.Second

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%d days, %d hours, %d minutes", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%d hours, %d minutes", hours, minutes)
	}
	return fmt.Sprintf("%d minutes", minutes)
}

// ============ Бенчмарки ============

func BenchmarkHostCollectorCollect(b *testing.B) {
	collector := NewHostCollector()

	b.ResetTimer()
	for b.Loop() {
		_, _ = collector.Collect()
	}
}

func BenchmarkGetComputerName(b *testing.B) {
	collector := NewHostCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.getComputerName(computerNameDnsHostname)
	}
}

func BenchmarkGetWorkgroup(b *testing.B) {
	collector := NewHostCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.getWorkgroup()
	}
}

func BenchmarkGetUpTimeSeconds(b *testing.B) {
	collector := NewHostCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.getUpTimeSeconds()
	}
}

func BenchmarkGetTimeZone(b *testing.B) {
	collector := NewHostCollector()

	b.ResetTimer()
	for b.Loop() {
		_, _ = collector.getTimeZone()
	}
}

func BenchmarkCollectHardwareInfo(b *testing.B) {
	collector := NewHostCollector()
	info := &models.HostInfo{}

	b.ResetTimer()
	for b.Loop() {
		collector.collectHardwareInfo(info)
	}
}
