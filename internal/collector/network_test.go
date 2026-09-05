//go:build windows

package collector

import (
	"net"
	"testing"

	"github.com/alme23/tracker/internal/models"
)

// ============ Тесты для NewNetworkCollector() ============

func TestNewNetworkCollector(t *testing.T) {
	collector := NewNetworkCollector()

	if collector == nil {
		t.Fatal("NewNetworkCollector returned nil")
	}
}

// ============ Тесты для Collect() ============

func TestNetworkCollectorCollect(t *testing.T) {
	collector := NewNetworkCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if len(statuses) == 0 {
		t.Error("No network interfaces found")
	}

	t.Logf("Found %d network interfaces", len(statuses))
}

// ============ Проверка данных ============

func TestNetworkCollectorDataIntegrity(t *testing.T) {
	collector := NewNetworkCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Уникальность индексов
	indexMap := make(map[int]bool)
	for _, iface := range statuses {
		if indexMap[iface.Index] {
			t.Errorf("Duplicate index: %d", iface.Index)
		}
		indexMap[iface.Index] = true
	}

	// Проверка MAC-адресов
	for _, iface := range statuses {
		if iface.MAC != "" {
			mac, err := net.ParseMAC(iface.MAC)
			if err != nil {
				t.Errorf("Invalid MAC %s: %v", iface.MAC, err)
			}
			if len(mac) != 6 {
				t.Errorf("Invalid MAC length: %d", len(mac))
			}
		}
	}

	// Проверка IP-адресов
	for _, iface := range statuses {
		for _, ip := range iface.IPAddresses {
			if ip == nil {
				t.Error("Nil IP")
			}
			if ip.To4() == nil && ip.To16() == nil {
				t.Errorf("Invalid IP: %v", ip)
			}
		}
	}

	// Проверка типов
	for _, iface := range statuses {
		switch iface.Type {
		case models.TypeEthernet, models.TypeWireless, models.TypeLoopback,
			models.TypeTunnel, models.TypePPP, models.TypeOther, models.TypeUnknown:
			// Valid
		default:
			t.Errorf("Invalid type: %v", iface.Type)
		}
	}

	// Проверка IP Assignment
	for _, iface := range statuses {
		switch iface.IPAssignment {
		case models.AssignmentNotApps, models.AssignmentDHCP, models.AssignmentStatic:
			// Valid
		default:
			t.Errorf("Invalid IP assignment: %v", iface.IPAssignment)
		}
	}
}

// ============ Проверка Loopback ============

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

			hasLoopbackIP := false
			for _, ip := range iface.IPAddresses {
				if ip.IsLoopback() {
					hasLoopbackIP = true
					break
				}
			}

			if !hasLoopbackIP {
				t.Error("Loopback doesn't have loopback IP")
			}

			if iface.IPAssignment != models.AssignmentNotApps {
				t.Errorf("Loopback should have AssignmentNotApps, got %v", iface.IPAssignment)
			}

			break
		}
	}

	if !foundLoopback {
		t.Error("Loopback not found")
	}
}

// ============ Проверка физических интерфейсов ============

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

// ============ Проверка IPv4/IPv6 ============

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

	t.Logf("IPv4: %v, IPv6: %v", foundIPv4, foundIPv6)
}

// ============ Проверка операционного статуса ============

func TestNetworkCollectorOperational(t *testing.T) {
	collector := NewNetworkCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	operationalCount := 0
	for _, iface := range statuses {
		if iface.Operational {
			operationalCount++
		}
	}

	t.Logf("Operational: %d", operationalCount)

	if operationalCount == 0 {
		t.Error("No operational interfaces")
	}
}

// ============ Повторные вызовы ============

func TestNetworkCollectorRepeatedCalls(t *testing.T) {
	collector := NewNetworkCollector()

	first, err := collector.Collect()
	if err != nil {
		t.Fatalf("First Collect failed: %v", err)
	}

	second, err := collector.Collect()
	if err != nil {
		t.Fatalf("Second Collect failed: %v", err)
	}

	if len(first) != len(second) {
		t.Errorf("Count changed: %d vs %d", len(first), len(second))
	}
}

// ============ Конкурентный доступ ============

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

// ============ Обработка переполнения буфера ============

func TestNetworkCollectorBufferOverflow(t *testing.T) {
	collector := NewNetworkCollector()

	for i := 0; i < 10; i++ {
		statuses, err := collector.Collect()
		if err != nil {
			t.Fatalf("Collect failed on iteration %d: %v", i, err)
		}
		if len(statuses) == 0 {
			t.Errorf("No interfaces on iteration %d", i)
		}
	}
}

// ============ Бенчмарки ============

func BenchmarkNetworkCollectorCollect(b *testing.B) {
	collector := NewNetworkCollector()

	b.ResetTimer()
	for b.Loop() {
		_, _ = collector.Collect()
	}
}

func BenchmarkNetworkCollectorParallel(b *testing.B) {
	collector := NewNetworkCollector()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = collector.Collect()
		}
	})
}
