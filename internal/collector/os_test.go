//go:build windows

package collector

import (
	"runtime"
	"testing"
	"time"

	"github.com/alme23/tracker/internal/models"
	"github.com/google/uuid"
)

// ============ Тесты для Collect() ============

func TestOSCollectorCollect(t *testing.T) {
	collector := NewOSCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if info.Name == "" {
		t.Error("Name is empty")
	}
	if info.BuildNumber == "" {
		t.Error("BuildNumber is empty")
	}
	if info.Architecture == models.UnknownArch {
		t.Error("Architecture is unknown")
	}
}

func TestOSCollectorCaching(t *testing.T) {
	collector := NewOSCollector()

	first, _ := collector.Collect()
	start := time.Now()
	second, _ := collector.Collect()
	duration := time.Since(start)

	if first.Name != second.Name {
		t.Error("Cached result differs")
	}
	t.Logf("Cached call: %v", duration)
}

// ============ Тесты для getArchitecture() ============

func TestGetArchitecture(t *testing.T) {
	collector := NewOSCollector()
	arch := collector.getArchitecture()

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

// ============ Тесты для getOSDetails() ============

func TestGetOSDetails(t *testing.T) {
	collector := NewOSCollector()
	details := collector.getOSDetails()

	if details.name == "" {
		t.Error("name is empty")
	}
	if details.buildNumber == "" {
		t.Error("buildNumber is empty")
	}
	if details.kernelVersion == "" {
		t.Error("kernelVersion is empty")
	}

	t.Logf("Name: %s", details.name)
	t.Logf("Edition: %s", details.edition)
	t.Logf("Build: %s", details.buildNumber)
	t.Logf("Kernel: %s", details.kernelVersion)
	t.Logf("ProductID: %s", details.productID)
	t.Logf("Owner: %s", details.registeredOwner)
	t.Logf("InstallType: %s", details.installationType)
}

// ============ Тесты для getWindowsVersionNumbers() ============

func TestGetWindowsVersionNumbers(t *testing.T) {
	collector := NewOSCollector()

	major, minor, build, isServer := collector.getWindowsVersionNumbers()

	if major == 0 {
		t.Error("Major is 0")
	}
	if build == 0 {
		t.Error("Build is 0")
	}

	t.Logf("Version: %d.%d.%d (Server: %v)", major, minor, build, isServer)
}

// ============ Тесты для getWindowsVersionName() ============

func TestGetWindowsVersionName(t *testing.T) {
	collector := NewOSCollector()

	tests := []struct {
		name     string
		major    uint32
		minor    uint32
		build    uint32
		expected string
	}{
		{"Win11", 10, 0, 22000, "Windows 11"},
		{"Win10", 10, 0, 19045, "Windows 10"},
		{"Win8.1", 6, 3, 9600, "Windows 8.1"},
		{"Win8", 6, 2, 9200, "Windows 8"},
		{"Win7", 6, 1, 7601, "Windows 7"},
		{"Vista", 6, 0, 6002, "Windows Vista"},
		{"XP", 5, 1, 2600, "Windows XP"},
		{"2000", 5, 0, 2195, "Windows 2000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.getWindowsVersionName(tt.major, tt.minor, tt.build)
			if result != tt.expected {
				t.Errorf("getWindowsVersionName(%d,%d,%d) = %s, want %s",
					tt.major, tt.minor, tt.build, result, tt.expected)
			}
		})
	}
}

// ============ Тесты для getEditionFromAPI() ============

func TestGetEditionFromAPI(t *testing.T) {
	collector := NewOSCollector()
	major, minor, _, _ := collector.getWindowsVersionNumbers()

	edition := collector.getEditionFromAPI(major, minor)

	if edition == "" {
		t.Error("Edition is empty")
	}

	t.Logf("Edition: %s", edition)
}

// ============ Тесты для getEditionName() ============

func TestGetEditionName(t *testing.T) {
	collector := NewOSCollector()

	tests := []struct {
		productType uint32
		expected    string
	}{
		{PRODUCT_PROFESSIONAL, "Professional"},
		{PRODUCT_ENTERPRISE, "Enterprise"},
		{PRODUCT_CORE, "Home"},
		{PRODUCT_EDUCATION, "Education"},
		{0xFFFFFFFF, "Edition 0xFFFFFFFF"},
	}

	for _, tt := range tests {
		result := collector.getEditionName(tt.productType)
		if result != tt.expected {
			t.Errorf("getEditionName(0x%X) = %s, want %s",
				tt.productType, result, tt.expected)
		}
	}
}

// ============ Тесты для getSystemLocale() ============

func TestGetSystemLocale(t *testing.T) {
	collector := NewOSCollector()
	locale := collector.getSystemLocale()

	if locale == "" || locale == "UNKNOWN" {
		t.Error("Locale is empty or unknown")
	}

	t.Logf("Locale: %s", locale)
}

// ============ Тесты для getInstallDate() ============

func TestGetInstallDate(t *testing.T) {
	collector := NewOSCollector()
	installDate := collector.getInstallDate()

	if installDate == 0 {
		t.Error("InstallDate is 0")
	}

	t.Logf("InstallDate: %d (%s)", installDate, time.Unix(installDate, 0).Format("2006-01-02"))
}

// ============ Тесты для readInstallTimestamp() ============

func TestReadInstallTimestamp(t *testing.T) {
	collector := NewOSCollector()

	paths := []string{
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion\Source OS (Updated on)`,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion\Source OS`,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
	}

	for _, path := range paths {
		val := collector.readInstallTimestamp(path)
		t.Logf("%s: %d", path, val)
	}
}

// ============ Тесты для getPowerShellVersion() ============

func TestGetPowerShellVersion(t *testing.T) {
	collector := NewOSCollector()
	version := collector.getPowerShellVersion()

	if version == "" {
		t.Error("PowerShell version is empty")
	}

	t.Logf("PowerShell: %s", version)
}

// ============ Тесты для readPSVersion() ============

func TestReadPSVersion(t *testing.T) {
	collector := NewOSCollector()

	// Core
	coreVer := collector.readPSVersion(`SOFTWARE\Microsoft\PowerShellCore\InstalledVersions`, "SemanticVersion")
	t.Logf("Core: %s", coreVer)

	// Windows PowerShell 3+
	ps3Ver := collector.readPSVersion(`SOFTWARE\Microsoft\PowerShell\3\PowerShellEngine`, "PowerShellVersion")
	t.Logf("PS 3+: %s", ps3Ver)

	// Windows PowerShell 1-2
	ps1Ver := collector.readPSVersion(`SOFTWARE\Microsoft\PowerShell\1\PowerShellEngine`, "PowerShellVersion")
	t.Logf("PS 1-2: %s", ps1Ver)
}

// ============ Тесты для isSecureBootEnabled() ============

func TestIsSecureBootEnabled(t *testing.T) {
	collector := NewOSCollector()
	secureBoot := collector.isSecureBootEnabled()

	t.Logf("Secure Boot: %v", secureBoot)
}

// ============ Тесты для isSecureBootEnabledFromRegistry() ============

func TestIsSecureBootEnabledFromRegistry(t *testing.T) {
	collector := NewOSCollector()
	secureBoot := collector.isSecureBootEnabledFromRegistry()

	t.Logf("Secure Boot (registry): %v", secureBoot)
}

// ============ Тесты для getMachineGUID() ============

func TestGetMachineGUID(t *testing.T) {
	collector := NewOSCollector()
	guid := collector.getMachineGUID()

	if guid.String() == "00000000-0000-0000-0000-000000000000" {
		t.Error("GUID is zero")
	}

	if _, err := uuid.Parse(guid.String()); err != nil {
		t.Errorf("Invalid GUID: %s", guid.String())
	}

	t.Logf("GUID: %s", guid.String())
}

// ============ Тесты для checkVirtualization() ============

func TestCheckVirtualization(t *testing.T) {
	collector := NewOSCollector()
	isVirtual := collector.checkVirtualization()

	t.Logf("Is Virtual: %v", isVirtual)
}

// ============ Тесты для checkHypervisorHost() ============

func TestCheckHypervisorHost(t *testing.T) {
	collector := NewOSCollector()
	isHypervisor := collector.checkHypervisorHost()

	t.Logf("Is Hypervisor: %v", isHypervisor)
}

// ============ Тесты для checkHyperVRegistry() ============

func TestCheckHyperVRegistry(t *testing.T) {
	collector := NewOSCollector()
	isHyperV := collector.checkHyperVRegistry()

	t.Logf("Hyper-V Registry: %v", isHyperV)
}

// ============ Тесты для checkKeyExists() ============

func TestCheckKeyExists(t *testing.T) {
	collector := NewOSCollector()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Existing key", `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, true},
		{"Non-existent key", `SOFTWARE\NonExistent\Key`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.checkKeyExists(tt.path)
			if result != tt.expected {
				t.Errorf("checkKeyExists(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// ============ Бенчмарки ============

func BenchmarkOSCollector(b *testing.B) {
	collector := NewOSCollector()

	b.ResetTimer()
	for b.Loop() {
		_, _ = collector.Collect()
	}
}

func BenchmarkOSCollectorCached(b *testing.B) {
	collector := NewOSCollector()
	_, _ = collector.Collect()

	b.ResetTimer()
	for b.Loop() {
		_, _ = collector.Collect()
	}
}

func BenchmarkGetWindowsVersionName(b *testing.B) {
	collector := NewOSCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.getWindowsVersionName(10, 0, 22621)
	}
}

func BenchmarkGetEditionName(b *testing.B) {
	collector := NewOSCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.getEditionName(PRODUCT_PROFESSIONAL)
	}
}

func BenchmarkGetSystemLocale(b *testing.B) {
	collector := NewOSCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.getSystemLocale()
	}
}

func BenchmarkGetInstallDate(b *testing.B) {
	collector := NewOSCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.getInstallDate()
	}
}

func BenchmarkGetPowerShellVersion(b *testing.B) {
	collector := NewOSCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.getPowerShellVersion()
	}
}

func BenchmarkIsSecureBootEnabled(b *testing.B) {
	collector := NewOSCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.isSecureBootEnabled()
	}
}

func BenchmarkGetMachineGUID(b *testing.B) {
	collector := NewOSCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.getMachineGUID()
	}
}

func BenchmarkCheckVirtualization(b *testing.B) {
	collector := NewOSCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.checkVirtualization()
	}
}

func BenchmarkCheckHypervisorHost(b *testing.B) {
	collector := NewOSCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.checkHypervisorHost()
	}
}
