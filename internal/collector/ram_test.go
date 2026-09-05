//go:build windows

package collector

import (
	"testing"
	"unsafe"
)

func TestNewRAMCollector(t *testing.T) {
	collector := NewRAMCollector()

	if collector == nil {
		t.Fatal("NewRAMCollector returned nil")
	}
}

func TestRAMCollectorCollect(t *testing.T) {
	collector := NewRAMCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Проверяем основные поля
	if info.TotalBytes == 0 {
		t.Error("TotalBytes is 0")
	}

	if info.AvailableBytes > info.TotalBytes {
		t.Error("AvailableBytes > TotalBytes")
	}

	if info.TotalPageFile == 0 {
		t.Error("TotalPageFile is 0")
	}

	if info.AvailablePageFile > info.TotalPageFile {
		t.Error("AvailablePageFile > TotalPageFile")
	}

	// Логируем информацию
	t.Logf("Total Physical: %d bytes (%.2f GB)",
		info.TotalBytes, float64(info.TotalBytes)/1024/1024/1024)
	t.Logf("Available Physical: %d bytes (%.2f GB)",
		info.AvailableBytes, float64(info.AvailableBytes)/1024/1024/1024)
	t.Logf("Total Page File: %d bytes (%.2f GB)",
		info.TotalPageFile, float64(info.TotalPageFile)/1024/1024/1024)
	t.Logf("Available Page File: %d bytes (%.2f GB)",
		info.AvailablePageFile, float64(info.AvailablePageFile)/1024/1024/1024)
}

func TestMemoryStatusExStructure(t *testing.T) {
	// Проверяем размер структуры
	size := unsafe.Sizeof(memoryStatusEx{})

	// MEMORYSTATUSEX должен быть 64 байта
	if size != 64 {
		t.Errorf("memoryStatusEx size = %d, want 64", size)
	}

	t.Logf("memoryStatusEx size: %d bytes", size)
}

func TestRAMInfoConsistency(t *testing.T) {
	collector := NewRAMCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Использованная память
	usedBytes := info.TotalBytes - info.AvailableBytes
	if usedBytes > info.TotalBytes {
		t.Error("Used bytes > total bytes")
	}

	// Процент использования
	if info.TotalBytes > 0 {
		usagePercent := float64(usedBytes) / float64(info.TotalBytes) * 100

		if usagePercent < 0 || usagePercent > 100 {
			t.Errorf("Usage percent out of range: %.2f%%", usagePercent)
		}

		t.Logf("Memory usage: %.2f%%", usagePercent)
	}

	// Page file usage
	if info.TotalPageFile > 0 {
		pageFileUsed := info.TotalPageFile - info.AvailablePageFile
		pageFilePercent := float64(pageFileUsed) / float64(info.TotalPageFile) * 100

		t.Logf("Page file usage: %.2f%%", pageFilePercent)
	}
}

func TestRAMCollectorRepeatedCalls(t *testing.T) {
	collector := NewRAMCollector()

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

	// Общая память не должна меняться
	if first.TotalBytes != second.TotalBytes {
		t.Errorf("TotalBytes changed: %d vs %d", first.TotalBytes, second.TotalBytes)
	}

	// Доступная память может немного отличаться
	difference := int64(first.AvailableBytes) - int64(second.AvailableBytes)
	if difference < 0 {
		difference = -difference
	}

	// Разница должна быть меньше 1 GB (память могла измениться)
	if difference > 1024*1024*1024 {
		t.Errorf("AvailableBytes changed too much: %d bytes", difference)
	}
}

func TestRAMInfoMinimumRequirements(t *testing.T) {
	collector := NewRAMCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Минимальные требования для Windows 10/11
	minRAM := uint64(1 * 1024 * 1024 * 1024) // 1 GB

	if info.TotalBytes < minRAM {
		t.Errorf("Total RAM (%d bytes) is less than minimum (%d bytes)",
			info.TotalBytes, minRAM)
	}
}

func TestRAMInfoFormattedOutput(t *testing.T) {
	collector := NewRAMCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Форматируем в GB
	totalGB := float64(info.TotalBytes) / 1024 / 1024 / 1024
	availableGB := float64(info.AvailableBytes) / 1024 / 1024 / 1024
	usedGB := totalGB - availableGB

	t.Logf("RAM: %.2f GB total, %.2f GB used, %.2f GB available",
		totalGB, usedGB, availableGB)

	// Проверяем, что значения разумные
	if totalGB < 0.5 || totalGB > 1024 {
		t.Errorf("Total RAM is unreasonable: %.2f GB", totalGB)
	}
}

func TestMemoryStatusExFields(t *testing.T) {
	var memStatus memoryStatusEx
	memStatus.dwLength = uint32(unsafe.Sizeof(memStatus))

	ret, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memStatus)))
	if ret == 0 {
		t.Fatal("GlobalMemoryStatusEx failed")
	}

	// Проверяем, что все поля заполнены
	if memStatus.ullTotalPhys == 0 {
		t.Error("ullTotalPhys is 0")
	}

	if memStatus.ullTotalPageFile == 0 {
		t.Error("ullTotalPageFile is 0")
	}

	// Memory Load должен быть от 0 до 100
	if memStatus.dwMemoryLoad > 100 {
		t.Errorf("dwMemoryLoad = %d, want <= 100", memStatus.dwMemoryLoad)
	}

	t.Logf("Memory Load: %d%%", memStatus.dwMemoryLoad)
	t.Logf("Total Physical: %d", memStatus.ullTotalPhys)
	t.Logf("Available Physical: %d", memStatus.ullAvailPhys)
	t.Logf("Total Page File: %d", memStatus.ullTotalPageFile)
	t.Logf("Available Page File: %d", memStatus.ullAvailPageFile)
}

func TestMemoryStatusExSize(t *testing.T) {
	size := unsafe.Sizeof(memoryStatusEx{})

	// MEMORYSTATUSEX должен быть 64 байта на x64
	if size != 64 {
		t.Errorf("memoryStatusEx size = %d, want 64", size)
	}

	// Проверяем смещения полей
	ms := &memoryStatusEx{}

	dwLengthOffset := unsafe.Offsetof(ms.dwLength)
	dwMemoryLoadOffset := unsafe.Offsetof(ms.dwMemoryLoad)
	ullTotalPhysOffset := unsafe.Offsetof(ms.ullTotalPhys)
	ullAvailPhysOffset := unsafe.Offsetof(ms.ullAvailPhys)
	ullTotalPageFileOffset := unsafe.Offsetof(ms.ullTotalPageFile)
	ullAvailPageFileOffset := unsafe.Offsetof(ms.ullAvailPageFile)

	t.Logf("dwLength offset: %d", dwLengthOffset)
	t.Logf("dwMemoryLoad offset: %d", dwMemoryLoadOffset)
	t.Logf("ullTotalPhys offset: %d", ullTotalPhysOffset)
	t.Logf("ullAvailPhys offset: %d", ullAvailPhysOffset)
	t.Logf("ullTotalPageFile offset: %d", ullTotalPageFileOffset)
	t.Logf("ullAvailPageFile offset: %d", ullAvailPageFileOffset)

	// Проверяем правильность смещений
	if dwLengthOffset != 0 {
		t.Error("dwLength should be at offset 0")
	}
	if dwMemoryLoadOffset != 4 {
		t.Error("dwMemoryLoad should be at offset 4")
	}
	if ullTotalPhysOffset != 8 {
		t.Error("ullTotalPhys should be at offset 8")
	}
	if ullAvailPhysOffset != 16 {
		t.Error("ullAvailPhys should be at offset 16")
	}
	if ullTotalPageFileOffset != 24 {
		t.Error("ullTotalPageFile should be at offset 24")
	}
	if ullAvailPageFileOffset != 32 {
		t.Error("ullAvailPageFile should be at offset 32")
	}
}

func TestRAMInfoDataIntegrity(t *testing.T) {
	collector := NewRAMCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Проверяем, что все значения байтовые (кратные 4096 для страниц)
	pageSize := uint64(4096)

	if info.TotalBytes%pageSize != 0 {
		t.Logf("TotalBytes (%d) is not multiple of page size (%d)",
			info.TotalBytes, pageSize)
	}

	if info.AvailableBytes%pageSize != 0 {
		t.Logf("AvailableBytes (%d) is not multiple of page size (%d)",
			info.AvailableBytes, pageSize)
	}

	// Доступная память должна быть меньше или равна общей
	if info.AvailableBytes > info.TotalBytes {
		t.Error("AvailableBytes > TotalBytes")
	}

	// Страничный файл должен быть больше или равен физической памяти
	if info.TotalPageFile < info.TotalBytes {
		t.Logf("Page file (%d) < physical RAM (%d)",
			info.TotalPageFile, info.TotalBytes)
	}
}

func BenchmarkRAMCollector(b *testing.B) {
	collector := NewRAMCollector()

	b.ResetTimer()
	for b.Loop() {
		_, err := collector.Collect()
		if err != nil {
			b.Fatalf("Collect failed: %v", err)
		}
	}
}

func BenchmarkRAMCollectorParallel(b *testing.B) {
	collector := NewRAMCollector()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := collector.Collect()
			if err != nil {
				b.Fatalf("Collect failed: %v", err)
			}
		}
	})
}
