//go:build windows

package collector

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alme23/tracker/internal/models"
)

// Определяем ошибку локально
var ErrNilSnapshot = errors.New("snapshot is nil")

func TestNewSystemCollector(t *testing.T) {
	collector := NewSystemCollector(2 * time.Second)

	if collector == nil {
		t.Fatal("NewSystemCollector returned nil")
	}

	// Проверяем, что все коллекторы инициализированы
	if collector.serviceColl == nil {
		t.Error("serviceColl is nil")
	}
	if collector.networkColl == nil {
		t.Error("networkColl is nil")
	}
	if collector.osColl == nil {
		t.Error("osColl is nil")
	}
	if collector.processorColl == nil {
		t.Error("processorColl is nil")
	}
	if collector.ramColl == nil {
		t.Error("ramColl is nil")
	}
	if collector.diskColl == nil {
		t.Error("diskColl is nil")
	}
	if collector.hostColl == nil {
		t.Error("hostColl is nil")
	}
	if collector.userColl == nil {
		t.Error("userColl is nil")
	}
}

func TestCollectAll(t *testing.T) {
	collector := NewSystemCollector(2 * time.Second)

	start := time.Now()
	snapshot, err := collector.CollectAll()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("CollectAll failed: %v", err)
	}

	if snapshot == nil {
		t.Fatal("Snapshot is nil")
	}

	// Проверяем Timestamp
	if snapshot.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}

	// Проверяем, что Timestamp не в будущем
	if snapshot.Timestamp.After(time.Now().Add(time.Minute)) {
		t.Error("Timestamp is in the future")
	}

	// Проверяем, что Timestamp не слишком старый
	if snapshot.Timestamp.Before(time.Now().Add(-5 * time.Minute)) {
		t.Error("Timestamp is too old")
	}

	t.Logf("Collection completed in %v", duration)
}

func TestCollectAllDataCompleteness(t *testing.T) {
	collector := NewSystemCollector(2 * time.Second)

	snapshot, err := collector.CollectAll()
	if err != nil {
		t.Fatalf("CollectAll failed: %v", err)
	}

	// Проверяем User
	if snapshot.User.Username == "" {
		t.Error("User.Username is empty")
	}
	if snapshot.User.FullName == "" {
		t.Error("User.FullName is empty")
	}
	if snapshot.User.ProfilePath == "" {
		t.Error("User.ProfilePath is empty")
	}

	// Проверяем OS
	if snapshot.OS.Name == "" {
		t.Error("OS.Name is empty")
	}
	if snapshot.OS.BuildNumber == "" {
		t.Error("OS.BuildNumber is empty")
	}
	if snapshot.OS.Architecture == models.UnknownArch {
		t.Error("OS.Architecture is unknown")
	}

	// Проверяем Processor
	if snapshot.Processor.Model == "" {
		t.Error("Processor.Model is empty")
	}
	if snapshot.Processor.VendorID == "" {
		t.Error("Processor.VendorID is empty")
	}
	if snapshot.Processor.PhysicalCores == 0 {
		t.Error("Processor.PhysicalCores is 0")
	}
	if snapshot.Processor.LogicalProcessors == 0 {
		t.Error("Processor.LogicalProcessors is 0")
	}

	// Проверяем RAM
	if snapshot.RAM.TotalBytes == 0 {
		t.Error("RAM.TotalBytes is 0")
	}
	if snapshot.RAM.AvailableBytes > snapshot.RAM.TotalBytes {
		t.Error("RAM.AvailableBytes > RAM.TotalBytes")
	}

	// Проверяем Drives
	if len(snapshot.Drives) == 0 {
		t.Error("No drives found")
	}

	// Проверяем Services
	if len(snapshot.Services) == 0 {
		t.Error("No services found")
	}

	// Проверяем Network
	if len(snapshot.Network) == 0 {
		t.Error("No network interfaces found")
	}

	// Проверяем Host
	if snapshot.Host.Hostname == "" {
		t.Error("Host.Hostname is empty")
	}
}

func TestCollectAllConcurrent(t *testing.T) {
	collector := NewSystemCollector(2 * time.Second)

	// Запускаем несколько сборов параллельно
	const numGoroutines = 5
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			snapshot, err := collector.CollectAll()
			if err != nil {
				errChan <- err
				return
			}
			if snapshot == nil {
				errChan <- ErrNilSnapshot
				return
			}
			errChan <- nil
		}()
	}

	// Проверяем ошибки
	for i := 0; i < numGoroutines; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("Concurrent CollectAll failed: %v", err)
		}
	}
}

func TestCollectAllPerformance(t *testing.T) {
	collector := NewSystemCollector(1 * time.Second)

	// Первый запуск может быть медленнее из-за инициализации
	_, err := collector.CollectAll()
	if err != nil {
		t.Fatalf("First CollectAll failed: %v", err)
	}

	// Замеряем последующие запуски
	start := time.Now()
	_, err = collector.CollectAll()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Second CollectAll failed: %v", err)
	}

	// Сбор не должен занимать более 30 секунд
	if duration > 30*time.Second {
		t.Errorf("CollectAll took too long: %v", duration)
	}

	t.Logf("CollectAll duration: %v", duration)
}

func TestCollectAllErrorHandling(t *testing.T) {
	// Создаем коллектор с очень маленьким таймаутом
	collector := NewSystemCollector(1 * time.Nanosecond)

	snapshot, err := collector.CollectAll()

	// Проверяем, что функция завершается (не зависает)
	_ = snapshot
	_ = err

	// Даже с ошибкой, функция должна вернуть управление
	// Ошибки от отдельных коллекторов не должны приводить к panic
}

func TestSystemSnapshotJSON(t *testing.T) {
	collector := NewSystemCollector(2 * time.Second)

	snapshot, err := collector.CollectAll()
	if err != nil {
		t.Fatalf("CollectAll failed: %v", err)
	}

	// Проверяем, что snapshot можно сериализовать в JSON
	jsonData, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("JSON data is empty")
	}

	// Проверяем, что JSON можно десериализовать обратно
	var restored models.SystemSnapshot
	if err := json.Unmarshal(jsonData, &restored); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// Сравниваем ключевые поля
	if restored.Timestamp.IsZero() {
		t.Error("Restored timestamp is zero")
	}

	if restored.User.Username != snapshot.User.Username {
		t.Error("Restored user mismatch")
	}

	if restored.OS.Name != snapshot.OS.Name {
		t.Error("Restored OS mismatch")
	}

	if restored.Processor.Model != snapshot.Processor.Model {
		t.Error("Restored processor mismatch")
	}

	if restored.RAM.TotalBytes != snapshot.RAM.TotalBytes {
		t.Error("Restored RAM mismatch")
	}

	if len(restored.Drives) != len(snapshot.Drives) {
		t.Error("Restored drives count mismatch")
	}

	if len(restored.Services) != len(snapshot.Services) {
		t.Error("Restored services count mismatch")
	}

	if len(restored.Network) != len(snapshot.Network) {
		t.Error("Restored network count mismatch")
	}

	if restored.Host.Hostname != snapshot.Host.Hostname {
		t.Error("Restored host mismatch")
	}
}

func TestCollectAllIndividualCollectors(t *testing.T) {
	collector := NewSystemCollector(2 * time.Second)

	t.Run("OSCollector", func(t *testing.T) {
		osData, err := collector.osColl.Collect()
		if err != nil {
			t.Errorf("OS collector failed: %v", err)
		}
		if osData.Name == "" {
			t.Error("OS name is empty")
		}
	})

	t.Run("NetworkCollector", func(t *testing.T) {
		networkData, err := collector.networkColl.Collect()
		if err != nil {
			t.Errorf("Network collector failed: %v", err)
		}
		if len(networkData) == 0 {
			t.Error("No network interfaces")
		}
	})

	t.Run("ServiceCollector", func(t *testing.T) {
		servicesData, err := collector.serviceColl.Collect()
		if err != nil {
			t.Errorf("Service collector failed: %v", err)
		}
		if len(servicesData) == 0 {
			t.Error("No services")
		}
	})

	t.Run("ProcessorCollector", func(t *testing.T) {
		procData, err := collector.processorColl.Collect()
		if err != nil {
			t.Errorf("Processor collector failed: %v", err)
		}
		if procData.Model == "" {
			t.Error("Processor model is empty")
		}
	})

	t.Run("RAMCollector", func(t *testing.T) {
		ramData, err := collector.ramColl.Collect()
		if err != nil {
			t.Errorf("RAM collector failed: %v", err)
		}
		if ramData.TotalBytes == 0 {
			t.Error("RAM total is 0")
		}
	})

	t.Run("DiskCollector", func(t *testing.T) {
		diskData, err := collector.diskColl.Collect()
		if err != nil {
			t.Errorf("Disk collector failed: %v", err)
		}
		if len(diskData) == 0 {
			t.Error("No disks")
		}
	})

	t.Run("HostCollector", func(t *testing.T) {
		hostData, err := collector.hostColl.Collect()
		if err != nil {
			t.Errorf("Host collector failed: %v", err)
		}
		if hostData.Hostname == "" {
			t.Error("Hostname is empty")
		}
	})

	t.Run("UserCollector", func(t *testing.T) {
		userData, err := collector.userColl.Collect()
		if err != nil {
			t.Errorf("User collector failed: %v", err)
		}
		if userData.Username == "" {
			t.Error("Username is empty")
		}
	})
}

func TestCollectAllNoDataRace(t *testing.T) {
	collector := NewSystemCollector(2 * time.Second)

	// Запускаем с флагом -race для проверки гонок данных
	const numGoroutines = 10
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := collector.CollectAll()
			if err != nil {
				t.Errorf("CollectAll failed: %v", err)
			}
		}()
	}

	wg.Wait()
}

func BenchmarkCollectAll(b *testing.B) {
	collector := NewSystemCollector(2 * time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.CollectAll()
		if err != nil {
			b.Fatalf("CollectAll failed: %v", err)
		}
	}
}

func BenchmarkCollectAllParallel(b *testing.B) {
	collector := NewSystemCollector(2 * time.Second)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := collector.CollectAll()
			if err != nil {
				b.Fatalf("CollectAll failed: %v", err)
			}
		}
	})
}
