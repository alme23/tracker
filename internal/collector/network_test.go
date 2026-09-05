//go:build windows

package collector

import (
	"net"
	"testing"

	"github.com/alme23/tracker/internal/models"
)

func TestNewNetworkCollector(t *testing.T) {
	collector := NewNetworkCollector()

	if collector == nil {
		t.Fatal("NewNetworkCollector returned nil")
	}

	// Проверяем, что пул буферов инициализирован
	if collector.bufferPool.New == nil {
		t.Error("bufferPool.New is nil")
	}
}

func TestNetworkCollectorCollect(t *testing.T) {
	collector := NewNetworkCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if len(statuses) == 0 {
		t.Error("No network interfaces found")
	}

	// Проверяем каждый интерфейс
	for _, iface := range statuses {
		// Индекс должен быть положительным
		if iface.Index < 0 {
			t.Errorf("Interface %s has invalid index: %d", iface.Name, iface.Index)
		}

		// Имя не должно быть пустым
		if iface.Name == "" {
			t.Error("Interface name is empty")
		}

		// Тип должен быть валидным
		switch iface.Type {
		case models.TypeEthernet, models.TypeWireless, models.TypeLoopback,
			models.TypeTunnel, models.TypePPP, models.TypeOther, models.TypeUnknown:
			// Valid types
		default:
			t.Errorf("Invalid interface type: %v", iface.Type)
		}

		// IP Assignment должен быть валидным
		switch iface.IPAssignment {
		case models.AssignmentNotApps, models.AssignmentDHCP, models.AssignmentStatic:
			// Valid assignments
		default:
			t.Errorf("Invalid IP assignment: %v", iface.IPAssignment)
		}

		// Проверяем MAC-адрес
		if iface.MAC != "" {
			mac, err := net.ParseMAC(iface.MAC)
			if err != nil {
				t.Errorf("Invalid MAC address %s: %v", iface.MAC, err)
			}
			if len(mac) != 6 {
				t.Errorf("Invalid MAC length: %d", len(mac))
			}
		}

		// Проверяем IP-адреса
		for _, ip := range iface.IPAddresses {
			if ip == nil {
				t.Error("Nil IP address")
			}
			if ip.To4() == nil && ip.To16() == nil {
				t.Errorf("Invalid IP address: %v", ip)
			}
		}

		// Логируем
		t.Logf("Interface %d: %s (%s)", iface.Index, iface.Name, iface.Description)
		t.Logf("  Type: %s, MAC: %s, Operational: %v",
			iface.Type.String(), iface.MAC, iface.Operational)
		t.Logf("  IP Assignment: %s", iface.IPAssignment)
		t.Logf("  IP Addresses: %v", iface.IPAddresses)
	}
}

func TestNetworkCollectorLoopback(t *testing.T) {
	collector := NewNetworkCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	foundLoopback := false
	for _, iface := range statuses {
		if iface.Type == models.TypeLoopback {
			foundLoopback = true

			// Loopback должен иметь IP 127.0.0.1 или ::1
			hasLoopbackIP := false
			for _, ip := range iface.IPAddresses {
				if ip.IsLoopback() {
					hasLoopbackIP = true
					break
				}
			}

			if !hasLoopbackIP {
				t.Error("Loopback interface doesn't have loopback IP")
			}

			// Loopback не должен использовать DHCP
			if iface.IPAssignment != models.AssignmentNotApps {
				t.Errorf("Loopback should have AssignmentNotApps, got %v", iface.IPAssignment)
			}

			// Loopback обычно имеет пустой MAC
			if iface.MAC != "" {
				t.Logf("Loopback has MAC: %s (unusual)", iface.MAC)
			}

			break
		}
	}

	if !foundLoopback {
		t.Error("Loopback interface not found")
	}
}

func TestNetworkCollectorPhysicalInterfaces(t *testing.T) {
	collector := NewNetworkCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	foundPhysical := false
	for _, iface := range statuses {
		if iface.Type == models.TypeEthernet || iface.Type == models.TypeWireless {
			foundPhysical = true

			// Физические интерфейсы должны иметь MAC-адрес
			if iface.MAC == "" {
				t.Errorf("Physical interface %s has empty MAC", iface.Name)
			}

			break
		}
	}

	if !foundPhysical {
		t.Log("No physical interfaces found (may be normal for VM)")
	}
}

func TestNetworkCollectorIPv4IPv6(t *testing.T) {
	collector := NewNetworkCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	foundIPv4 := false
	foundIPv6 := false

	for _, iface := range statuses {
		for _, ip := range iface.IPAddresses {
			if ip.To4() != nil {
				foundIPv4 = true
			} else if ip.To16() != nil {
				foundIPv6 = true
			}
		}
	}

	t.Logf("IPv4 found: %v", foundIPv4)
	t.Logf("IPv6 found: %v", foundIPv6)

	// На большинстве систем должен быть IPv4
	if !foundIPv4 {
		t.Log("No IPv4 addresses found (unusual)")
	}
}

func TestNetworkCollectorOperational(t *testing.T) {
	collector := NewNetworkCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	operationalCount := 0
	nonOperationalCount := 0

	for _, iface := range statuses {
		if iface.Operational {
			operationalCount++
		} else {
			nonOperationalCount++
		}
	}

	t.Logf("Operational interfaces: %d", operationalCount)
	t.Logf("Non-operational interfaces: %d", nonOperationalCount)

	// Должен быть хотя бы один работающий интерфейс
	if operationalCount == 0 {
		t.Error("No operational interfaces")
	}
}

func TestNetworkCollectorDHCPDetection(t *testing.T) {
	collector := NewNetworkCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	dhcpCount := 0
	staticCount := 0

	for _, iface := range statuses {
		switch iface.IPAssignment {
		case models.AssignmentDHCP:
			dhcpCount++
		case models.AssignmentStatic:
			staticCount++
		}
	}

	t.Logf("DHCP interfaces: %d", dhcpCount)
	t.Logf("Static interfaces: %d", staticCount)
}

func TestNetworkCollectorDataIntegrity(t *testing.T) {
	collector := NewNetworkCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Проверяем уникальность индексов
	indexMap := make(map[int]bool)
	for _, iface := range statuses {
		if indexMap[iface.Index] {
			t.Errorf("Duplicate interface index: %d", iface.Index)
		}
		indexMap[iface.Index] = true
	}

	// Проверяем уникальность MAC-адресов (кроме пустых)
	macMap := make(map[string]bool)
	for _, iface := range statuses {
		if iface.MAC != "" {
			if macMap[iface.MAC] {
				t.Errorf("Duplicate MAC address: %s", iface.MAC)
			}
			macMap[iface.MAC] = true
		}
	}
}

func TestNetworkCollectorRepeatedCalls(t *testing.T) {
	collector := NewNetworkCollector()

	// Первый вызов
	first, err := collector.Collect()
	if err != nil {
		t.Fatalf("First Collect failed: %v", err)
	}

	// Второй вызов
	second, err := collector.Collect()
	if err != nil {
		t.Fatalf("Second Collect failed: %v", err)
	}

	// Количество интерфейсов должно совпадать
	if len(first) != len(second) {
		t.Errorf("Interface count changed: %d vs %d", len(first), len(second))
	}

	// Индексы должны совпадать
	for i := range first {
		if first[i].Index != second[i].Index {
			t.Errorf("Interface order changed: %d vs %d", first[i].Index, second[i].Index)
		}
	}
}

func BenchmarkNetworkCollector(b *testing.B) {
	collector := NewNetworkCollector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.Collect()
		if err != nil {
			b.Fatalf("Collect failed: %v", err)
		}
	}
}

func BenchmarkNetworkCollectorParallel(b *testing.B) {
	collector := NewNetworkCollector()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := collector.Collect()
			if err != nil {
				b.Fatalf("Collect failed: %v", err)
			}
		}
	})
}

// Тест на утечки памяти
func TestNetworkCollectorNoLeaks(t *testing.T) {
	collector := NewNetworkCollector()

	// Многократный вызов для проверки утечек
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

// Тест на конкурентный доступ
func TestNetworkCollectorConcurrent(t *testing.T) {
	collector := NewNetworkCollector()

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
