//go:build windows

package collector

import (
	"strings"
	"testing"
	"time"

	"github.com/alme23/tracker/internal/models"
)

func TestNewHostCollector(t *testing.T) {
	collector := NewHostCollector()

	if collector == nil {
		t.Fatal("NewHostCollector returned nil")
	}
}

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
		t.Error("Uptime is 0")
	}

	if info.BootTime.IsZero() {
		t.Error("BootTime is zero")
	}

	if info.TimeZone == "" {
		t.Error("TimeZone is empty")
	}

	// Логируем информацию
	t.Logf("Hostname: %s", info.Hostname)
	t.Logf("FQDN: %s", info.FQDN)
	t.Logf("Physical Hostname: %s", info.PhysicalHostname)
	t.Logf("Physical FQDN: %s", info.PhysicalFQDN)
	t.Logf("Domain: %s", info.Domain)
	t.Logf("Workgroup: %s", info.Workgroup)
	t.Logf("Uptime: %d seconds", info.UpTimeSeconds)
	t.Logf("BootTime: %s", info.BootTime.Format("2006-01-02 15:04:05"))
	t.Logf("TimeZone: %s (offset %d)", info.TimeZone, info.TimeZoneOffset)
	t.Logf("Manufacturer: %s", info.Manufacturer)
	t.Logf("Model: %s", info.Model)
	t.Logf("Serial: %s", info.SerialNumber)
	t.Logf("BIOS: %s %s (%s)", info.BIOSVendor, info.BIOSVersion, info.BIOSDate)
}

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

			// Не все типы могут быть доступны
			if result == "" {
				t.Logf("%s returned empty (may be normal)", tt.name)
			}
		})
	}
}

func TestGetWorkgroup(t *testing.T) {
	collector := NewHostCollector()

	workgroup := collector.getWorkgroup()

	t.Logf("Workgroup: %s", workgroup)

	// Рабочая группа может быть пустой, если компьютер в домене
	if workgroup == "" {
		t.Log("Workgroup is empty (may be in domain)")
	}
}

func TestGetUpTimeSeconds(t *testing.T) {
	collector := NewHostCollector()

	uptime := collector.getUpTimeSeconds()

	t.Logf("Uptime: %d seconds", uptime)

	if uptime == 0 {
		t.Error("Uptime is 0")
	}

	// Проверяем, что uptime не слишком большой (больше 100 лет)
	if uptime > 100*365*24*3600 {
		t.Errorf("Uptime is too large: %d", uptime)
	}
}

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

func TestCollectHardwareInfo(t *testing.T) {
	collector := NewHostCollector()
	info := &models.HostInfo{}

	collector.collectHardwareInfo(info)

	// Логируем информацию о железе
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
	if info.Manufacturer == "" && info.Model == "" {
		t.Error("Both manufacturer and model are empty")
	}
}

func TestHostInfoVirtualMachineDetection(t *testing.T) {
	collector := NewHostCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Проверяем, различаются ли физическое и виртуальное имена
	if info.PhysicalHostname != "" && info.PhysicalHostname != info.Hostname {
		t.Logf("Virtual machine detected!")
		t.Logf("  Virtual: %s", info.Hostname)
		t.Logf("  Physical: %s", info.PhysicalHostname)
	} else {
		t.Log("Physical machine (or VM with same names)")
	}
}

func TestHostInfoDomainDetection(t *testing.T) {
	collector := NewHostCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if info.Domain != "" {
		t.Logf("Computer is in domain: %s", info.Domain)
		if info.Workgroup != "" {
			t.Logf("Also has workgroup: %s", info.Workgroup)
		}
	} else {
		t.Logf("Computer is in workgroup: %s", info.Workgroup)
	}
}

func TestBootTimeCalculation(t *testing.T) {
	collector := NewHostCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Проверяем, что BootTime + Uptime ≈ Now
	calculatedNow := info.BootTime.Add(time.Duration(info.UpTimeSeconds) * time.Second)
	difference := time.Since(calculatedNow)

	// Разница должна быть меньше 5 секунд
	if difference > 5*time.Second || difference < -5*time.Second {
		t.Errorf("BootTime calculation is off by %v", difference)
	}

	t.Logf("BootTime: %s", info.BootTime.Format("2006-01-02 15:04:05"))
	t.Logf("Uptime: %d seconds", info.UpTimeSeconds)
	t.Logf("Calculated Now: %s", calculatedNow.Format("2006-01-02 15:04:05"))
	t.Logf("Actual Now: %s", time.Now().Format("2006-01-02 15:04:05"))
}

func TestTimeZoneNormalization(t *testing.T) {
	collector := NewHostCollector()

	zoneName, offset := collector.getTimeZone()

	// Нормализуем название зоны
	normalizedZone := normalizeTimeZoneName(zoneName)

	t.Logf("Original: %s", zoneName)
	t.Logf("Normalized: %s", normalizedZone)
	t.Logf("Offset: %d minutes", offset)

	if normalizedZone == "" {
		t.Error("Normalized zone is empty")
	}
}

// normalizeTimeZoneName нормализует название часового пояса
func normalizeTimeZoneName(name string) string {
	replacements := map[string]string{
		"RTZ 2 (зима)":          "Moscow Standard Time",
		"RTZ 2 (лето)":          "Moscow Daylight Time",
		"Russian Standard Time": "Moscow Standard Time",
		"Russian Daylight Time": "Moscow Daylight Time",
	}

	if normalized, ok := replacements[name]; ok {
		return normalized
	}
	return name
}

func TestHostInfoFieldsConsistency(t *testing.T) {
	collector := NewHostCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// FQDN должен содержать Hostname
	if info.FQDN != "" && !strings.Contains(info.FQDN, info.Hostname) {
		t.Errorf("FQDN (%s) doesn't contain hostname (%s)", info.FQDN, info.Hostname)
	}

	// PhysicalFQDN должен содержать PhysicalHostname
	if info.PhysicalFQDN != "" && info.PhysicalHostname != "" &&
		!strings.Contains(info.PhysicalFQDN, info.PhysicalHostname) {
		t.Errorf("PhysicalFQDN (%s) doesn't contain PhysicalHostname (%s)",
			info.PhysicalFQDN, info.PhysicalHostname)
	}

	// Если есть домен, FQDN должен содержать его
	if info.Domain != "" && info.FQDN != "" &&
		!strings.Contains(info.FQDN, info.Domain) {
		t.Errorf("FQDN (%s) doesn't contain domain (%s)", info.FQDN, info.Domain)
	}
}

func BenchmarkHostCollector(b *testing.B) {
	collector := NewHostCollector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.Collect()
		if err != nil {
			b.Fatalf("Collect failed: %v", err)
		}
	}
}

func BenchmarkHostCollectorParallel(b *testing.B) {
	collector := NewHostCollector()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := collector.Collect()
			if err != nil {
				b.Fatalf("Collect failed: %v", err)
			}
		}
	})
}
