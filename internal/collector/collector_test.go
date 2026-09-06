//go:build windows

package collector

import (
	"sync"
	"testing"
	"time"
)

// ============ Тесты для NewSystemCollector() ============

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

// ============ Тесты для CollectAll() ============

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

	// Проверяем Timestamp (int64)
	if snapshot.Timestamp == 0 {
		t.Error("Timestamp is 0")
	}

	// Проверяем, что Timestamp близок к текущему времени
	now := time.Now().Unix()
	if snapshot.Timestamp < now-60 || snapshot.Timestamp > now+60 {
		t.Errorf("Timestamp %d is not close to now %d", snapshot.Timestamp, now)
	}

	t.Logf("Collection completed in %v", duration)
	t.Logf("Timestamp: %d", snapshot.Timestamp)
}

// ============ Проверка полноты данных ============

func TestCollectAllDataCompleteness(t *testing.T) {
	collector := NewSystemCollector(2 * time.Second)

	snapshot, err := collector.CollectAll()
	if err != nil {
		t.Fatalf("CollectAll failed: %v", err)
	}

	// User
	if snapshot.User.Username == "" {
		t.Error("User.Username is empty")
	}

	// OS
	if snapshot.OS.Name == "" {
		t.Error("OS.Name is empty")
	}

	// Processor
	if snapshot.Processor.Model == "" {
		t.Error("Processor.Model is empty")
	}

	// RAM
	if snapshot.RAM.TotalBytes == 0 {
		t.Error("RAM.TotalBytes is 0")
	}

	// Drives
	if len(snapshot.Drives) == 0 {
		t.Error("No drives found")
	}

	// Services
	if len(snapshot.Services) == 0 {
		t.Error("No services found")
	}

	// Network
	if len(snapshot.Network) == 0 {
		t.Error("No network interfaces found")
	}

	// Host
	if snapshot.Host.Hostname == "" {
		t.Error("Host.Hostname is empty")
	}
}

// ============ Конкурентный доступ ============

func TestCollectAllConcurrent(t *testing.T) {
	collector := NewSystemCollector(2 * time.Second)

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

	for i := 0; i < numGoroutines; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("Concurrent CollectAll failed: %v", err)
		}
	}
}

// ============ Производительность ============

func TestCollectAllPerformance(t *testing.T) {
	collector := NewSystemCollector(1 * time.Second)

	// Первый запуск
	_, err := collector.CollectAll()
	if err != nil {
		t.Fatalf("First CollectAll failed: %v", err)
	}

	// Второй запуск
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

// ============ Обработка ошибок ============

func TestCollectAllErrorHandling(t *testing.T) {
	collector := NewSystemCollector(1 * time.Nanosecond)

	snapshot, err := collector.CollectAll()

	// Функция должна завершиться без паники
	_ = snapshot
	_ = err
}

// ============ Проверка гонок данных ============

func TestCollectAllNoDataRace(t *testing.T) {
	collector := NewSystemCollector(2 * time.Second)

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

// ============ Индивидуальные коллекторы ============

func TestCollectAllIndividualCollectors(t *testing.T) {
	collector := NewSystemCollector(2 * time.Second)

	t.Run("OSCollector", func(t *testing.T) {
		data, err := collector.osColl.Collect()
		if err != nil {
			t.Errorf("OS collector failed: %v", err)
		}
		if data.Name == "" {
			t.Error("OS name is empty")
		}
	})

	t.Run("NetworkCollector", func(t *testing.T) {
		data, err := collector.networkColl.Collect()
		if err != nil {
			t.Errorf("Network collector failed: %v", err)
		}
		if len(data) == 0 {
			t.Error("No network interfaces")
		}
	})

	t.Run("ServiceCollector", func(t *testing.T) {
		data, err := collector.serviceColl.Collect()
		if err != nil {
			t.Errorf("Service collector failed: %v", err)
		}
		if len(data) == 0 {
			t.Error("No services")
		}
	})

	t.Run("ProcessorCollector", func(t *testing.T) {
		data, err := collector.processorColl.Collect()
		if err != nil {
			t.Errorf("Processor collector failed: %v", err)
		}
		if data.Model == "" {
			t.Error("Processor model is empty")
		}
	})

	t.Run("RAMCollector", func(t *testing.T) {
		data, err := collector.ramColl.Collect()
		if err != nil {
			t.Errorf("RAM collector failed: %v", err)
		}
		if data.TotalBytes == 0 {
			t.Error("RAM total is 0")
		}
	})

	t.Run("DiskCollector", func(t *testing.T) {
		data, err := collector.diskColl.Collect()
		if err != nil {
			t.Errorf("Disk collector failed: %v", err)
		}
		if len(data) == 0 {
			t.Error("No disks")
		}
	})

	t.Run("HostCollector", func(t *testing.T) {
		data, err := collector.hostColl.Collect()
		if err != nil {
			t.Errorf("Host collector failed: %v", err)
		}
		if data.Hostname == "" {
			t.Error("Hostname is empty")
		}
	})

	t.Run("UserCollector", func(t *testing.T) {
		data, err := collector.userColl.Collect()
		if err != nil {
			t.Errorf("User collector failed: %v", err)
		}
		if data.Username == "" {
			t.Error("Username is empty")
		}
	})
}

// ============ Бенчмарки ============

func BenchmarkCollectAll(b *testing.B) {
	collector := NewSystemCollector(2 * time.Second)

	b.ResetTimer()
	for b.Loop() {
		_, _ = collector.CollectAll()
	}
}

func BenchmarkCollectAllParallel(b *testing.B) {
	collector := NewSystemCollector(2 * time.Second)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = collector.CollectAll()
		}
	})
}

// ============ Вспомогательные переменные ============

var ErrNilSnapshot = &snapshotError{"snapshot is nil"}

type snapshotError struct {
	msg string
}

func (e *snapshotError) Error() string {
	return e.msg
}
