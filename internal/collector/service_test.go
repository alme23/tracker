//go:build windows

package collector

import (
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/alme23/tracker/internal/models"
)

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

func TestServiceCollectorCollect(t *testing.T) {
	collector := NewServiceCollector(2 * time.Second)

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if len(statuses) != 2 {
		t.Fatalf("Expected 2 services, got %d", len(statuses))
	}

	// Проверяем порядок
	if statuses[0].Name != "RDP" {
		t.Errorf("First service should be RDP, got %s", statuses[0].Name)
	}

	if statuses[1].Name != "VNC" && !contains(statuses[1].Name, "VNC (") {
		t.Errorf("Second service should be VNC, got %s", statuses[1].Name)
	}

	// Проверяем RDP
	if statuses[0].ServiceName != "TermService" {
		t.Errorf("RDP service name = %s, want TermService", statuses[0].ServiceName)
	}

	if statuses[0].Port == 0 {
		t.Error("RDP port is 0")
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
		{
			name:        "TightVNC port",
			keyPath:     `SOFTWARE\TightVNC\Server`,
			valueName:   "RfbPort",
			defaultPort: "5900",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.readRegistryPortGeneric(tt.keyPath, tt.valueName, tt.defaultPort)

			if result == "" {
				t.Error("Result is empty")
			}

			// Проверяем, что результат - валидный порт
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

func TestCheckWindowsService(t *testing.T) {
	collector := NewServiceCollector(1 * time.Second)

	tests := []struct {
		name        string
		serviceName string
		shouldExist bool
	}{
		{
			name:        "RDP service",
			serviceName: "TermService",
			shouldExist: true,
		},
		{
			name:        "Non-existent service",
			serviceName: "DefinitelyNotExistingService12345",
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
		{
			name:     "Invalid host",
			host:     "invalid.host.name",
			port:     "80",
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

func TestGetLocalIP(t *testing.T) {
	collector := NewServiceCollector(1 * time.Second)

	ip := collector.getLocalIP()

	t.Logf("Local IP: %s", ip)

	if ip == "" {
		t.Error("Local IP is empty")
	}

	// Проверяем, что IP валидный
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		t.Errorf("Invalid IP: %s", ip)
	}

	// Не должен быть 0.0.0.0 или ::
	if ip == "0.0.0.0" || ip == "::" {
		t.Errorf("Invalid IP: %s", ip)
	}
}

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

		// Проверяем порт
		if port, err := strconv.ParseUint(sig.DefaultPort, 10, 16); err != nil || port == 0 {
			t.Errorf("Invalid default port for %s: %s", sig.BrandName, sig.DefaultPort)
		}

		t.Logf("VNC: %s (%s)", sig.BrandName, sig.ServiceName)
	}
}

func TestVncDetection(t *testing.T) {
	collector := NewServiceCollector(1 * time.Second)

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Находим VNC статус
	var vncStatus *models.ServiceStatus
	for i := range statuses {
		if contains(statuses[i].Name, "VNC") {
			vncStatus = &statuses[i]
			break
		}
	}

	if vncStatus == nil {
		t.Fatal("VNC status not found")
	}

	// VNC может быть не установлен
	if !vncStatus.Installed {
		t.Log("VNC is not installed")
		if vncStatus.Running {
			t.Error("VNC is not installed but running")
		}
	} else {
		t.Logf("VNC installed: %s", vncStatus.Name)
		t.Logf("VNC service: %s", vncStatus.ServiceName)
		t.Logf("VNC port: %d", vncStatus.Port)
	}
}

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

func BenchmarkServiceCollector(b *testing.B) {
	collector := NewServiceCollector(2 * time.Second)

	b.ResetTimer()
	for b.Loop() {
		_, err := collector.Collect()
		if err != nil {
			b.Fatalf("Collect failed: %v", err)
		}
	}
}

func BenchmarkServiceCollectorParallel(b *testing.B) {
	collector := NewServiceCollector(2 * time.Second)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := collector.Collect()
			if err != nil {
				b.Fatalf("Collect failed: %v", err)
			}
		}
	})
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
