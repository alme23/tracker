//go:build windows

package collector

import (
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/alme23/tracker/internal/models"
)

// ============ Тесты для NewServiceCollector() ============

func TestNewServiceCollector(t *testing.T) {
	timeout := 2 * time.Second
	collector := NewServiceCollector(timeout)

	if collector == nil {
		t.Fatal("NewServiceCollector returned nil")
	}

	if collector.dialTimeout != timeout {
		t.Errorf("dialTimeout = %v, want %v", collector.dialTimeout, timeout)
	}
}

// ============ Тесты для Collect() ============

func TestServiceCollectorCollect(t *testing.T) {
	collector := NewServiceCollector(2 * time.Second)

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if len(statuses) == 0 {
		t.Error("No services found")
	}

	// Проверяем, что RDP есть
	foundRDP := false
	for _, status := range statuses {
		if status.Name == "RDP" {
			foundRDP = true
			if status.ServiceName != "TermService" {
				t.Errorf("RDP service name = %s, want TermService", status.ServiceName)
			}
			break
		}
	}

	if !foundRDP {
		t.Error("RDP service not found")
	}

	// Логируем
	for _, status := range statuses {
		t.Logf("Service: %s", status.Name)
		t.Logf("  Windows Service: %s", status.ServiceName)
		t.Logf("  Installed: %v", status.Installed)
		t.Logf("  Running: %v", status.Running)
		t.Logf("  Port: %d", status.Port)
		t.Logf("  Port Open: %v", status.PortOpen)
	}
}

// ============ Тесты для processRdpService() ============

func TestProcessRdpService(t *testing.T) {
	collector := NewServiceCollector(1 * time.Second)
	status := &models.ServiceStatus{Name: "RDP", ServiceName: "TermService"}

	err := collector.processRdpService(status, "127.0.0.1")
	if err != nil {
		t.Fatalf("processRdpService failed: %v", err)
	}

	if !status.Installed {
		t.Error("RDP should be installed")
	}

	if status.Port == 0 {
		t.Error("RDP port is 0")
	}

	t.Logf("RDP: installed=%v, running=%v, port=%d",
		status.Installed, status.Running, status.Port)
}

// ============ Тесты для readRegistryPortGeneric() ============

func TestReadRegistryPortGeneric(t *testing.T) {
	collector := NewServiceCollector(1 * time.Second)

	tests := []struct {
		name        string
		keyPath     string
		valueName   string
		defaultPort string
	}{
		{
			name:        "RDP port",
			keyPath:     `SYSTEM\CurrentControlSet\Control\Terminal Server\WinStations\RDP-Tcp`,
			valueName:   "PortNumber",
			defaultPort: "3389",
		},
		{
			name:        "Non-existent key",
			keyPath:     `SOFTWARE\NonExistent\Key`,
			valueName:   "Port",
			defaultPort: "5900",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.readRegistryPortGeneric(tt.keyPath, tt.valueName, tt.defaultPort)

			if result == "" {
				t.Error("Result is empty")
			}

			port, err := strconv.ParseUint(result, 10, 16)
			if err != nil {
				t.Errorf("Invalid port: %s", result)
			}

			if port == 0 || port > 65535 {
				t.Errorf("Port out of range: %d", port)
			}

			t.Logf("%s: %s", tt.name, result)
		})
	}
}

// ============ Тесты для checkWindowsService() ============

func TestCheckWindowsService(t *testing.T) {
	collector := NewServiceCollector(1 * time.Second)

	tests := []struct {
		name        string
		serviceName string
		shouldExist bool
	}{
		{
			name:        "RDP exists",
			serviceName: "TermService",
			shouldExist: true,
		},
		{
			name:        "Non-existent",
			serviceName: "DefinitelyNotExisting12345",
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installed, running, err := collector.checkWindowsService(tt.serviceName)

			if err != nil {
				t.Fatalf("checkWindowsService failed: %v", err)
			}

			if installed != tt.shouldExist {
				t.Errorf("installed = %v, want %v", installed, tt.shouldExist)
			}

			t.Logf("%s: installed=%v, running=%v", tt.serviceName, installed, running)
		})
	}
}

// ============ Тесты для checkFirewallPort() ============

func TestCheckFirewallPort(t *testing.T) {
	collector := NewServiceCollector(1 * time.Second)

	tests := []struct {
		name     string
		host     string
		port     string
		expected bool
	}{
		{
			name:     "Closed port",
			host:     "127.0.0.1",
			port:     "1",
			expected: false,
		},
		{
			name:     "Invalid port",
			host:     "127.0.0.1",
			port:     "99999",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.checkFirewallPort(tt.host, tt.port)
			if result != tt.expected {
				t.Errorf("checkFirewallPort(%s, %s) = %v, want %v",
					tt.host, tt.port, result, tt.expected)
			}
		})
	}
}

// ============ Тесты для getLocalIP() ============

func TestGetLocalIP(t *testing.T) {
	collector := NewServiceCollector(1 * time.Second)

	ip := collector.getLocalIP()

	t.Logf("Local IP: %s", ip)

	if ip == "" {
		t.Error("Local IP is empty")
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		t.Errorf("Invalid IP: %s", ip)
	}
}

// ============ Тесты для getVncSignatures() ============

func TestGetVncSignatures(t *testing.T) {
	signatures := getVncSignatures()

	if len(signatures) == 0 {
		t.Fatal("No VNC signatures")
	}

	for _, sig := range signatures {
		if sig.BrandName == "" {
			t.Error("BrandName is empty")
		}
		if sig.ServiceName == "" {
			t.Error("ServiceName is empty")
		}
		if sig.RegistryKey == "" {
			t.Error("RegistryKey is empty")
		}
		if sig.ValueName == "" {
			t.Error("ValueName is empty")
		}
		if sig.DefaultPort == "" {
			t.Error("DefaultPort is empty")
		}

		t.Logf("VNC: %s (%s)", sig.BrandName, sig.ServiceName)
	}
}

// ============ Тесты на конкурентность ============

func TestServiceCollectorConcurrent(t *testing.T) {
	collector := NewServiceCollector(1 * time.Second)

	const numGoroutines = 5
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

func BenchmarkServiceCollectorCollect(b *testing.B) {
	collector := NewServiceCollector(2 * time.Second)

	b.ResetTimer()
	for b.Loop() {
		_, _ = collector.Collect()
	}
}

func BenchmarkProcessRdpService(b *testing.B) {
	collector := NewServiceCollector(1 * time.Second)
	status := &models.ServiceStatus{Name: "RDP", ServiceName: "TermService"}

	b.ResetTimer()
	for b.Loop() {
		_ = collector.processRdpService(status, "127.0.0.1")
	}
}

func BenchmarkCheckWindowsService(b *testing.B) {
	collector := NewServiceCollector(1 * time.Second)

	b.ResetTimer()
	for b.Loop() {
		_, _, _ = collector.checkWindowsService("TermService")
	}
}

func BenchmarkReadRegistryPortGeneric(b *testing.B) {
	collector := NewServiceCollector(1 * time.Second)

	b.ResetTimer()
	for b.Loop() {
		_ = collector.readRegistryPortGeneric(
			`SYSTEM\CurrentControlSet\Control\Terminal Server\WinStations\RDP-Tcp`,
			"PortNumber",
			"3389",
		)
	}
}

func BenchmarkGetLocalIP(b *testing.B) {
	collector := NewServiceCollector(1 * time.Second)

	b.ResetTimer()
	for b.Loop() {
		_ = collector.getLocalIP()
	}
}
