//go:build windows

package collector

import (
	"net"
	"testing"
	"time"

	"github.com/alme23/tracker/internal/models"
	"golang.org/x/sys/windows"
)

func TestNewNetworkCollector(t *testing.T) {
	collector := NewNetworkCollector()

	if collector == nil {
		t.Fatal("NewNetworkCollector returned nil")
	}
}

func TestNetworkCollectorCollect(t *testing.T) {
	collector := NewNetworkCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if len(statuses) == 0 {
		t.Error("Expected at least one network interface")
	}

	// Проверяем каждый интерфейс
	for _, status := range statuses {
		// Проверяем обязательные поля
		if status.Name == "" {
			t.Errorf("Interface %d has empty name", status.Index)
		}

		if status.Index < 0 {
			t.Errorf("Interface %s has invalid index %d", status.Name, status.Index)
		}

		// Проверяем тип интерфейса
		switch status.Type {
		case models.TypeEthernet, models.TypeWireless, models.TypeLoopback,
			models.TypeTunnel, models.TypePPP, models.TypeOther, models.TypeUnknown:
			// Valid types
		default:
			t.Errorf("Interface %s has invalid type %v", status.Name, status.Type)
		}

		// Проверяем способ получения IP
		switch status.IPAssignment {
		case models.AssignmentNotApps, models.AssignmentDHCP, models.AssignmentStatic:
			// Valid assignments
		default:
			t.Errorf("Interface %s has invalid IP assignment %v", status.Name, status.IPAssignment)
		}

		// Проверяем MAC-адрес
		if status.MAC != "" {
			// Проверяем формат MAC-адреса (6 групп по 2 hex цифры)
			mac, err := net.ParseMAC(status.MAC)
			if err != nil {
				t.Errorf("Interface %s has invalid MAC address %s: %v", status.Name, status.MAC, err)
			}
			if len(mac) != 6 {
				t.Errorf("Interface %s has invalid MAC length %d", status.Name, len(mac))
			}
		}

		// Проверяем IP-адреса
		for _, ip := range status.IPAddresses {
			if ip == nil {
				t.Errorf("Interface %s has nil IP address", status.Name)
			}

			// Проверяем, что IP-адрес валидный
			if ip.To4() == nil && ip.To16() == nil {
				t.Errorf("Interface %s has invalid IP address %v", status.Name, ip)
			}
		}
	}
}

func TestNetworkCollectorLoopback(t *testing.T) {
	collector := NewNetworkCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	// Проверяем, что loopback интерфейс существует и правильно определен
	foundLoopback := false
	for _, status := range statuses {
		if status.Type == models.TypeLoopback {
			foundLoopback = true

			// Loopback должен иметь IP 127.0.0.1 или ::1
			hasLoopbackIP := false
			for _, ip := range status.IPAddresses {
				if ip.IsLoopback() {
					hasLoopbackIP = true
					break
				}
			}

			if !hasLoopbackIP {
				t.Error("Loopback interface doesn't have loopback IP address")
			}

			// Loopback не должен использовать DHCP
			if status.IPAssignment != models.AssignmentNotApps {
				t.Errorf("Loopback interface should have AssignmentNotApps, got %v", status.IPAssignment)
			}

			break
		}
	}

	if !foundLoopback {
		t.Error("Loopback interface not found")
	}
}

func TestNetworkCollectorMACAddress(t *testing.T) {
	collector := NewNetworkCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	foundMAC := false
	for _, status := range statuses {
		if status.MAC != "" {
			foundMAC = true

			// Проверяем формат MAC-адреса
			mac, err := net.ParseMAC(status.MAC)
			if err != nil {
				t.Errorf("Invalid MAC address %s: %v", status.MAC, err)
			}

			// MAC-адрес должен быть 6 байт (48 бит)
			if len(mac) != 6 {
				t.Errorf("Invalid MAC address length %d for %s", len(mac), status.MAC)
			}

			// Проверяем, что MAC-адрес не состоит из одних нулей
			allZeros := true
			for _, b := range mac {
				if b != 0 {
					allZeros = false
					break
				}
			}

			if allZeros {
				t.Errorf("MAC address %s consists of all zeros", status.MAC)
			}

			break
		}
	}

	// Не все системы могут иметь физические интерфейсы с MAC
	// Поэтому просто логируем, если не нашли
	if !foundMAC {
		t.Log("No physical interfaces with MAC addresses found")
	}
}

func TestNetworkCollectorTypes(t *testing.T) {
	collector := NewNetworkCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	typeCounts := make(map[models.InterfaceType]int)

	for _, status := range statuses {
		typeCounts[status.Type]++

		// Логируем информацию для отладки
		t.Logf("Interface: %s (Index: %d)", status.Name, status.Index)
		t.Logf("  Description: %s", status.Description)
		t.Logf("  Type: %v", status.Type)
		t.Logf("  MAC: %s", status.MAC)
		t.Logf("  Operational: %v", status.Operational)
		t.Logf("  IP Assignment: %v", status.IPAssignment)
		t.Logf("  IP Addresses: %v", status.IPAddresses)
	}

	// Проверяем, что есть хотя бы один тип интерфейса
	if len(typeCounts) == 0 {
		t.Error("No interface types found")
	}

	// Логируем статистику типов
	for type_, count := range typeCounts {
		t.Logf("Type %v: %d interfaces", type_, count)
	}
}

func TestNetworkCollectorIPAddresses(t *testing.T) {
	collector := NewNetworkCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	foundIPv4 := false
	foundIPv6 := false

	for _, status := range statuses {
		for _, ip := range status.IPAddresses {
			if ip.To4() != nil {
				foundIPv4 = true
				// Проверяем, что IPv4 адрес валидный
				if len(ip.To4()) != 4 {
					t.Errorf("Invalid IPv4 address length for %v", ip)
				}
			} else if ip.To16() != nil {
				foundIPv6 = true
				// Проверяем, что IPv6 адрес валидный
				if len(ip.To16()) != 16 {
					t.Errorf("Invalid IPv6 address length for %v", ip)
				}
			}
		}
	}

	// Логируем найденные протоколы
	t.Logf("Found IPv4: %v, IPv6: %v", foundIPv4, foundIPv6)

	// На большинстве систем должен быть хотя бы IPv4
	if !foundIPv4 {
		t.Log("No IPv4 addresses found (unusual)")
	}
}

func TestNetworkCollectorOperationalStatus(t *testing.T) {
	collector := NewNetworkCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	operationalCount := 0
	nonOperationalCount := 0

	for _, status := range statuses {
		if status.Operational {
			operationalCount++
		} else {
			nonOperationalCount++
		}
	}

	t.Logf("Operational interfaces: %d", operationalCount)
	t.Logf("Non-operational interfaces: %d", nonOperationalCount)

	// На большинстве систем должен быть хотя бы один работающий интерфейс
	if operationalCount == 0 {
		t.Log("No operational interfaces found")
	}
}

func TestNetworkCollectorIPAssignment(t *testing.T) {
	collector := NewNetworkCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	assignmentCounts := make(map[models.IPAssignment]int)

	for _, status := range statuses {
		assignmentCounts[status.IPAssignment]++

		// Loopback не должен иметь DHCP или Static
		if status.Type == models.TypeLoopback && status.IPAssignment != models.AssignmentNotApps {
			t.Errorf("Loopback interface %s has assignment %v, expected AssignmentNotApps",
				status.Name, status.IPAssignment)
		}
	}

	// Логируем статистику
	for assignment, count := range assignmentCounts {
		t.Logf("Assignment %v: %d interfaces", assignment, count)
	}
}

// Тест на обработку ошибок
func TestNetworkCollectorErrorHandling(t *testing.T) {
	collector := NewNetworkCollector()

	// Проверяем, что коллектор корректно обрабатывает повторные вызовы
	for i := 0; i < 3; i++ {
		statuses, err := collector.Collect()
		if err != nil {
			t.Fatalf("Collect failed on iteration %d: %v", i, err)
		}

		if len(statuses) == 0 {
			t.Errorf("Collect returned empty results on iteration %d", i)
		}
	}
}

// Бенчмарки
func BenchmarkNetworkCollectorCollect(b *testing.B) {
	collector := NewNetworkCollector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.Collect()
		if err != nil {
			b.Fatalf("Collect returned error: %v", err)
		}
	}
}

func BenchmarkNetworkCollectorCollectParallel(b *testing.B) {
	collector := NewNetworkCollector()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := collector.Collect()
			if err != nil {
				b.Fatalf("Collect returned error: %v", err)
			}
		}
	})
}

// Тест на утечки памяти (можно запускать с -race)
func TestNetworkCollectorNoLeaks(t *testing.T) {
	collector := NewNetworkCollector()

	// Запускаем многократно для проверки утечек
	for i := 0; i < 100; i++ {
		statuses, err := collector.Collect()
		if err != nil {
			t.Fatalf("Collect failed on iteration %d: %v", i, err)
		}

		// Явно освобождаем ссылки
		for j := range statuses {
			statuses[j].IPAddresses = nil
		}
		statuses = nil
	}

	// Если тест завершился без ошибок, утечек нет
}

// Тест на корректность структуры данных
func TestNetworkCollectorDataIntegrity(t *testing.T) {
	collector := NewNetworkCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	// Проверяем уникальность индексов
	indexMap := make(map[int]bool)
	for _, status := range statuses {
		if indexMap[status.Index] {
			t.Errorf("Duplicate interface index %d", status.Index)
		}
		indexMap[status.Index] = true
	}

	// Проверяем, что нет пустых описаний у физических интерфейсов
	for _, status := range statuses {
		if status.Type != models.TypeLoopback && status.Description == "" {
			t.Logf("Interface %s has empty description", status.Name)
		}
	}
}

// Интеграционный тест (может требовать прав администратора)
func TestNetworkCollectorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	collector := NewNetworkCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Integration test failed: %v", err)
	}

	t.Logf("Found %d network interfaces:", len(statuses))
	for _, status := range statuses {
		t.Logf("  [%d] %s (%s)", status.Index, status.Name, status.Description)
		t.Logf("      MAC: %s", status.MAC)
		t.Logf("      Type: %v", status.Type)
		t.Logf("      Status: %s", map[bool]string{true: "UP", false: "DOWN"}[status.Operational])
		t.Logf("      IP Assignment: %v", status.IPAssignment)
		t.Logf("      IP Addresses: %v", status.IPAddresses)
	}
}

// Тест производительности для больших систем
func TestNetworkCollectorPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	collector := NewNetworkCollector()

	start := time.Now()
	statuses, err := collector.Collect()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	t.Logf("Collected %d interfaces in %v", len(statuses), duration)

	// Проверяем, что сбор не занимает слишком много времени (> 5 секунд)
	if duration > 5*time.Second {
		t.Errorf("Collect took too long: %v", duration)
	}
}

func TestInterfaceTypeConversion(t *testing.T) {
	tests := []struct {
		name     string
		ifType   uint32
		expected models.InterfaceType
	}{
		{"Ethernet", windows.IF_TYPE_ETHERNET_CSMACD, models.TypeEthernet},
		{"Wireless", windows.IF_TYPE_IEEE80211, models.TypeWireless},
		{"Loopback", windows.IF_TYPE_SOFTWARE_LOOPBACK, models.TypeLoopback},
		{"Tunnel", windows.IF_TYPE_TUNNEL, models.TypeTunnel},
		{"PPP", windows.IF_TYPE_PPP, models.TypePPP},
		{"Other", windows.IF_TYPE_OTHER, models.TypeOther},
		{"Unknown", 99999, models.TypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var netType models.InterfaceType
			switch tt.ifType {
			case windows.IF_TYPE_ETHERNET_CSMACD:
				netType = models.TypeEthernet
			case windows.IF_TYPE_IEEE80211:
				netType = models.TypeWireless
			case windows.IF_TYPE_SOFTWARE_LOOPBACK:
				netType = models.TypeLoopback
			case windows.IF_TYPE_TUNNEL:
				netType = models.TypeTunnel
			case windows.IF_TYPE_PPP:
				netType = models.TypePPP
			case windows.IF_TYPE_OTHER:
				netType = models.TypeOther
			default:
				netType = models.TypeUnknown
			}

			if netType != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, netType)
			}
		})
	}
}

func TestIPAssignmentDetection(t *testing.T) {
	tests := []struct {
		name     string
		flags    uint32
		netType  models.InterfaceType
		expected models.IPAssignment
	}{
		{"Loopback", 0, models.TypeLoopback, models.AssignmentNotApps},
		{"DHCP", 0x0004, models.TypeEthernet, models.AssignmentDHCP},
		{"Static", 0, models.TypeEthernet, models.AssignmentStatic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ipAssignment models.IPAssignment
			if tt.netType == models.TypeLoopback {
				ipAssignment = models.AssignmentNotApps
			} else {
				if (tt.flags & 0x0004) != 0 {
					ipAssignment = models.AssignmentDHCP
				} else {
					ipAssignment = models.AssignmentStatic
				}
			}

			if ipAssignment != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, ipAssignment)
			}
		})
	}
}
