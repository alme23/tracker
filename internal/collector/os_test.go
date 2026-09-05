//go:build windows

package collector

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/alme23/tracker/internal/models"
	"github.com/google/uuid"
)

func TestNewOSCollector(t *testing.T) {
	collector := NewOSCollector()

	if collector == nil {
		t.Fatal("NewOSCollector returned nil")
	}
}

func TestOSCollectorCollect(t *testing.T) {
	collector := NewOSCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Проверяем обязательные поля
	if info.Name == "" {
		t.Error("Name is empty")
	}

	if info.BuildNumber == "" {
		t.Error("BuildNumber is empty")
	}

	if info.KernelVersion == "" {
		t.Error("KernelVersion is empty")
	}

	if info.Architecture == models.UnknownArch {
		t.Error("Architecture is unknown")
	}

	// Логируем информацию
	t.Logf("Name: %s", info.Name)
	t.Logf("Edition: %s", info.Edition)
	t.Logf("Build: %s", info.BuildNumber)
	t.Logf("Kernel: %s", info.KernelVersion)
	t.Logf("Architecture: %s", info.Architecture.String())
	t.Logf("Locale: %s", info.Locale)
	t.Logf("Install Date: %s", info.InstallDate)
	t.Logf("Installation Type: %s", info.InstallationType)
	t.Logf("PowerShell: %s", info.PowerShellVer)
	t.Logf("Secure Boot: %v", info.SecureBootLines)
	t.Logf("Machine GUID: %s", info.MachineGUID.String())
	t.Logf("Product ID: %s", info.ProductID)
	t.Logf("Registered Owner: %s", info.RegisteredOwner)
	t.Logf("Registered Org: %s", info.RegisteredOrg)
	t.Logf("Is Virtual: %v", info.IsVirtual)
	t.Logf("Is Hypervisor: %v", info.IsHypervisor)
}

func TestGetArchitecture(t *testing.T) {
	collector := NewOSCollector()

	arch := collector.getArchitecture()

	t.Logf("Architecture: %s", arch.String())

	// Проверяем соответствие с runtime.GOARCH
	switch runtime.GOARCH {
	case "amd64":
		if arch != models.AMD64 {
			t.Errorf("Expected AMD64, got %v", arch)
		}
	case "386":
		if arch != models.I386 {
			t.Errorf("Expected I386, got %v", arch)
		}
	case "arm64":
		if arch != models.ARM64 {
			t.Errorf("Expected ARM64, got %v", arch)
		}
	}
}

func TestGetOSDetails(t *testing.T) {
	collector := NewOSCollector()

	details := collector.getOSDetails()

	t.Logf("Name: %s", details.name)
	t.Logf("Edition: %s", details.edition)
	t.Logf("Build: %s", details.buildNumber)
	t.Logf("Kernel: %s", details.kernelVersion)
	t.Logf("Product ID: %s", details.productID)
	t.Logf("Owner: %s", details.registeredOwner)
	t.Logf("Org: %s", details.registeredOrg)
	t.Logf("Installation Type: %s", details.installationType)

	if details.name == "" {
		t.Error("Name is empty")
	}
}

func TestGetWindowsVersionNumbers(t *testing.T) {
	collector := NewOSCollector()

	major, minor, build := collector.getWindowsVersionNumbers()

	t.Logf("Windows Version: %d.%d.%d", major, minor, build)

	if major == 0 {
		t.Error("Major version is 0")
	}

	if build == 0 {
		t.Error("Build number is 0")
	}
}

func TestGetWindowsVersionName(t *testing.T) {
	collector := NewOSCollector()

	tests := []struct {
		name     string
		major    uint32
		minor    uint32
		build    uint32
		expected string
	}{
		{"Windows 11", 10, 0, 22000, "Windows 11"},
		{"Windows 10", 10, 0, 19045, "Windows 10"},
		{"Windows 8.1", 6, 3, 9600, "Windows 8.1"},
		{"Windows 8", 6, 2, 9200, "Windows 8"},
		{"Windows 7", 6, 1, 7601, "Windows 7"},
		{"Windows Vista", 6, 0, 6002, "Windows Vista"},
		{"Windows XP", 5, 1, 2600, "Windows XP"},
		{"Windows 2000", 5, 0, 2195, "Windows 2000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.getWindowsVersionName(tt.major, tt.minor, tt.build)
			if result != tt.expected {
				t.Errorf("getWindowsVersionName(%d, %d, %d) = %s, want %s",
					tt.major, tt.minor, tt.build, result, tt.expected)
			}
		})
	}
}

func TestGetEditionName(t *testing.T) {
	collector := NewOSCollector()

	tests := []struct {
		name        string
		productType uint32
		expected    string
	}{
		{"Professional", PRODUCT_PROFESSIONAL, "Professional"},
		{"Enterprise", PRODUCT_ENTERPRISE, "Enterprise"},
		{"Home", PRODUCT_CORE, "Home"},
		{"Education", PRODUCT_EDUCATION, "Education"},
		{"Server Standard", PRODUCT_STANDARD_SERVER, "Server Standard"},
		{"Server Datacenter", PRODUCT_DATACENTER_SERVER, "Server Datacenter"},
		{"Unknown", 0xFFFFFFFF, "Edition 0xFFFFFFFF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.getEditionName(tt.productType)
			if result != tt.expected {
				t.Errorf("getEditionName(0x%X) = %s, want %s",
					tt.productType, result, tt.expected)
			}
		})
	}
}

func TestGetSystemLocale(t *testing.T) {
	collector := NewOSCollector()

	locale := collector.getSystemLocale()

	t.Logf("Locale: %s", locale)

	if locale == "" || locale == "UNKNOWN" {
		t.Error("Locale is empty or unknown")
	}
}

func TestGetInstallDate(t *testing.T) {
	collector := NewOSCollector()

	installDate := collector.getInstallDate()

	t.Logf("Install Date: %s", installDate)

	if installDate == "" || installDate == "UNKNOWN" {
		t.Error("Install date is empty or unknown")
	}

	// Проверяем формат даты
	if _, err := time.Parse("2006-01-02", installDate); err != nil {
		t.Errorf("Invalid date format: %s", installDate)
	}
}

func TestGetPowerShellVersion(t *testing.T) {
	collector := NewOSCollector()

	version := collector.getPowerShellVersion()

	t.Logf("PowerShell Version: %s", version)

	// PowerShell может быть не установлен
	if version == "" {
		t.Log("PowerShell version is empty (may not be installed)")
	}
}

func TestIsSecureBootEnabled(t *testing.T) {
	collector := NewOSCollector()

	secureBoot := collector.isSecureBootEnabled()

	t.Logf("Secure Boot: %v", secureBoot)

	// Secure Boot может быть недоступен на старых системах
}

func TestGetMachineGUID(t *testing.T) {
	collector := NewOSCollector()

	guid := collector.getMachineGUID()

	t.Logf("Machine GUID: %s", guid.String())

	// Проверяем, что GUID не нулевой
	if guid.String() == "00000000-0000-0000-0000-000000000000" {
		t.Error("Machine GUID is zero")
	}

	// Проверяем формат UUID
	if _, err := uuid.Parse(guid.String()); err != nil {
		t.Errorf("Invalid GUID format: %s", guid.String())
	}
}

func TestCheckVirtualization(t *testing.T) {
	collector := NewOSCollector()

	isVirtual := collector.checkVirtualization()

	t.Logf("Is Virtual: %v", isVirtual)
}

func TestCheckHypervisorHost(t *testing.T) {
	collector := NewOSCollector()

	isHypervisor := collector.checkHypervisorHost()

	t.Logf("Is Hypervisor Host: %v", isHypervisor)
}

func TestCheckHyperVRegistry(t *testing.T) {
	collector := NewOSCollector()

	isHyperV := collector.checkHyperVRegistry()

	t.Logf("Hyper-V Registry: %v", isHyperV)
}

func TestOSCollectorCaching(t *testing.T) {
	collector := NewOSCollector()

	// Первый вызов
	first, err := collector.Collect()
	if err != nil {
		t.Fatalf("First Collect failed: %v", err)
	}

	// Второй вызов (должен взять из кэша)
	start := time.Now()
	second, err := collector.Collect()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Second Collect failed: %v", err)
	}

	// Проверяем, что результаты совпадают
	if first.Name != second.Name {
		t.Error("Cached result differs")
	}

	if first.BuildNumber != second.BuildNumber {
		t.Error("Cached build number differs")
	}

	t.Logf("Second call took %v (cached)", duration)
}

func TestOSInfoConsistency(t *testing.T) {
	collector := NewOSCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// KernelVersion должен содержать BuildNumber
	if !strings.Contains(info.KernelVersion, info.BuildNumber) {
		t.Errorf("KernelVersion (%s) doesn't contain BuildNumber (%s)",
			info.KernelVersion, info.BuildNumber)
	}

	// Name должен содержать "Windows"
	if !strings.Contains(info.Name, "Windows") {
		t.Errorf("Name (%s) doesn't contain 'Windows'", info.Name)
	}
}

func TestEditionNameCoverage(t *testing.T) {
	collector := NewOSCollector()

	// Проверяем, что все основные редакции определены
	editions := []uint32{
		PRODUCT_PROFESSIONAL,
		PRODUCT_ENTERPRISE,
		PRODUCT_CORE,
		PRODUCT_EDUCATION,
		PRODUCT_STANDARD_SERVER,
		PRODUCT_DATACENTER_SERVER,
	}

	for _, edition := range editions {
		name := collector.getEditionName(edition)
		if name == "" || strings.HasPrefix(name, "Edition 0x") {
			t.Errorf("Edition 0x%X not properly defined", edition)
		}
	}
}

func BenchmarkOSCollector(b *testing.B) {
	collector := NewOSCollector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.Collect()
		if err != nil {
			b.Fatalf("Collect failed: %v", err)
		}
	}
}

func BenchmarkOSCollectorCached(b *testing.B) {
	collector := NewOSCollector()

	// Предварительно заполняем кэш
	_, _ = collector.Collect()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.Collect()
		if err != nil {
			b.Fatalf("Collect failed: %v", err)
		}
	}
}
