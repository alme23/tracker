//go:build windows

package collector

import (
	"testing"

	"github.com/alme23/tracker/internal/models"
	"golang.org/x/sys/windows"
)

// ============ Тесты для NewDiskCollector() ============

func TestNewDiskCollector(t *testing.T) {
	collector := NewDiskCollector()

	if collector == nil {
		t.Fatal("NewDiskCollector returned nil")
	}
}

// ============ Тесты для Collect() ============

func TestDiskCollectorCollect(t *testing.T) {
	collector := NewDiskCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if len(statuses) == 0 {
		t.Error("No drives found")
	}

	t.Logf("Found %d drives", len(statuses))

	for _, drive := range statuses {
		t.Logf("Drive %s: Type=%s, FS=%s", drive.Letter, drive.Type.String(), drive.FSType)
	}
}

// ============ Тесты для processVolume() ============

func TestProcessVolume(t *testing.T) {
	collector := NewDiskCollector()

	// Тест с пустой строкой
	_, ok := collector.processVolume("")
	if ok {
		t.Error("processVolume should return false for empty string")
	}

	// Тест с невалидной строкой
	_, ok = collector.processVolume("invalid_volume_path")
	if ok {
		t.Error("processVolume should return false for invalid path")
	}
}

// ============ Тесты для mapDriveType() ============

func TestMapDriveType(t *testing.T) {
	collector := NewDiskCollector()

	tests := []struct {
		name     string
		rawType  uint32
		expected models.DriveType
	}{
		{"Fixed", windows.DRIVE_FIXED, models.DriveFixed},
		{"Removable", windows.DRIVE_REMOVABLE, models.DriveRemovable},
		{"RAM", windows.DRIVE_RAMDISK, models.DriveRAM},
		{"CDROM", windows.DRIVE_CDROM, models.DriveCDROM},
		{"NoRootDir", windows.DRIVE_NO_ROOT_DIR, models.DriveNoRootDir},
		{"Remote", windows.DRIVE_REMOTE, models.DriveRemote},
		{"Unknown", windows.DRIVE_UNKNOWN, models.DriveUnknown},
		{"Invalid 999", 999, models.DriveUnknown},
		{"Invalid 100", 100, models.DriveUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.mapDriveType(tt.rawType)
			if result != tt.expected {
				t.Errorf("mapDriveType(%d) = %v, want %v", tt.rawType, result, tt.expected)
			}
		})
	}
}

// ============ Тесты для formatMountPath() ============

func TestFormatMountPath(t *testing.T) {
	collector := NewDiskCollector()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Drive C", `C:\`, "C:"},
		{"Drive D", `D:\`, "D:"},
		{"Mount point", `C:\Mount\`, `C:\Mount`},
		{"No trailing slash", `C:\Mount`, `C:\Mount`},
		{"Empty", "", ""},
		{"Long path", `C:\Very\Long\Path\`, `C:\Very\Long\Path`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.formatMountPath(tt.input)
			if result != tt.expected {
				t.Errorf("formatMountPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// ============ Интеграционные тесты (требуют реальные тома) ============

func TestGetMountPathIntegration(t *testing.T) {
	collector := NewDiskCollector()

	// Получаем реальный том
	var volBuf [50]uint16
	handle, err := windows.FindFirstVolume(&volBuf[0], uint32(len(volBuf)))
	if err != nil {
		t.Skip("Cannot find volumes")
	}
	defer windows.FindVolumeClose(handle)

	volumeGUIDPath := windows.UTF16ToString(volBuf[:])
	volumePtr, err := windows.UTF16PtrFromString(volumeGUIDPath)
	if err != nil {
		t.Skip("Cannot convert volume path")
	}

	mountPath := collector.getMountPath(volumePtr)

	if mountPath == "" {
		t.Skip("No mount path for first volume (may be system reserved)")
	}

	t.Logf("Mount path: %s", mountPath)
}

func TestCollectVolumeInfoAndSpaceIntegration(t *testing.T) {
	collector := NewDiskCollector()

	// Получаем реальный том с буквой диска
	var volBuf [50]uint16
	handle, err := windows.FindFirstVolume(&volBuf[0], uint32(len(volBuf)))
	if err != nil {
		t.Skip("Cannot find volumes")
	}
	defer windows.FindVolumeClose(handle)

	// Перебираем тома, пока не найдем с буквой
	for {
		volumeGUIDPath := windows.UTF16ToString(volBuf[:])
		volumePtr, err := windows.UTF16PtrFromString(volumeGUIDPath)
		if err == nil {
			mountPath := collector.getMountPath(volumePtr)
			if mountPath != "" {
				mountPathPtr, err := windows.UTF16PtrFromString(mountPath)
				if err == nil {
					info := &models.DriveInfo{}
					collector.collectVolumeInfoAndSpace(info, mountPathPtr)

					t.Logf("Volume: %s", info.VolumeName)
					t.Logf("Serial: %d", info.SerialNumber)
					t.Logf("FS: %s", info.FSType)
					t.Logf("Ready: %v", info.IsReady)
					t.Logf("Total: %d", info.TotalBytes)
					t.Logf("Free: %d", info.FreeBytes)

					return
				}
			}
		}

		if !collector.nextVolume(handle, &volBuf[0], uint32(len(volBuf))) {
			break
		}
	}

	t.Skip("No volume with mount path found")
}

func TestNextVolumeIntegration(t *testing.T) {
	collector := NewDiskCollector()

	var volBuf [50]uint16
	handle, err := windows.FindFirstVolume(&volBuf[0], uint32(len(volBuf)))
	if err != nil {
		t.Skip("Cannot find volumes")
	}
	defer windows.FindVolumeClose(handle)

	count := 0
	for {
		count++
		if !collector.nextVolume(handle, &volBuf[0], uint32(len(volBuf))) {
			break
		}
	}

	t.Logf("Found %d volumes", count)

	if count == 0 {
		t.Error("No volumes found")
	}
}

// ============ Бенчмарки ============

func BenchmarkDiskCollectorCollect(b *testing.B) {
	collector := NewDiskCollector()

	b.ResetTimer()
	for b.Loop() {
		_, _ = collector.Collect()
	}
}

func BenchmarkMapDriveType(b *testing.B) {
	collector := NewDiskCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.mapDriveType(windows.DRIVE_FIXED)
	}
}

func BenchmarkFormatMountPath(b *testing.B) {
	collector := NewDiskCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.formatMountPath(`C:\`)
	}
}
