//go:build windows

package collector

import (
	"testing"

	"github.com/alme23/tracker/internal/models"
	"golang.org/x/sys/windows"
)

func TestNewDiskCollector(t *testing.T) {
	collector := NewDiskCollector()

	if collector == nil {
		t.Fatal("NewDiskCollector returned nil")
	}
}

func TestDiskCollectorCollect(t *testing.T) {
	collector := NewDiskCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if len(statuses) == 0 {
		t.Error("No drives found")
	}

	// Проверяем каждый диск
	for _, drive := range statuses {
		// Буква диска не должна быть пустой
		if drive.Letter == "" {
			t.Error("Drive letter is empty")
		}

		// Тип диска должен быть валидным
		switch drive.Type {
		case models.DriveUnknown, models.DriveNoRootDir, models.DriveRemovable,
			models.DriveFixed, models.DriveRemote, models.DriveCDROM, models.DriveRAM:
			// Valid types
		default:
			t.Errorf("Invalid drive type: %v", drive.Type)
		}

		// Проверяем размеры
		if drive.TotalBytes > 0 && drive.FreeBytes > drive.TotalBytes {
			t.Errorf("Drive %s: free bytes > total bytes", drive.Letter)
		}

		if drive.UsedBytes > 0 && drive.UsedBytes > drive.TotalBytes {
			t.Errorf("Drive %s: used bytes > total bytes", drive.Letter)
		}

		// Логируем информацию
		t.Logf("Drive %s: Type=%s, FS=%s, Total=%d GB, Free=%d GB",
			drive.Letter,
			drive.Type.String(),
			drive.FSType,
			drive.TotalBytes/1024/1024/1024,
			drive.FreeBytes/1024/1024/1024,
		)
	}
}

func TestDiskCollectorSystemDrive(t *testing.T) {
	collector := NewDiskCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Проверяем наличие системного диска (обычно C:)
	foundSystemDrive := false
	for _, drive := range statuses {
		if drive.Letter == "C:" {
			foundSystemDrive = true

			// Системный диск должен быть фиксированным
			if drive.Type != models.DriveFixed {
				t.Errorf("System drive should be FIXED, got %s", drive.Type.String())
			}

			// Должен иметь файловую систему
			if drive.FSType == "" {
				t.Error("System drive has empty filesystem type")
			}

			// Должен быть готов
			if !drive.IsReady {
				t.Error("System drive is not ready")
			}

			// Должен иметь ненулевой размер
			if drive.TotalBytes == 0 {
				t.Error("System drive has 0 total bytes")
			}

			break
		}
	}

	if !foundSystemDrive {
		t.Error("System drive (C:) not found")
	}
}

func TestMapDriveType(t *testing.T) {
	collector := NewDiskCollector()

	tests := []struct {
		name     string
		rawType  uint32
		expected models.DriveType
	}{
		{"Fixed", windows.DRIVE_FIXED, models.DriveFixed},
		{"Removable", windows.DRIVE_REMOVABLE, models.DriveRemovable},
		{"RAM Disk", windows.DRIVE_RAMDISK, models.DriveRAM},
		{"CD-ROM", windows.DRIVE_CDROM, models.DriveCDROM},
		{"No Root Dir", windows.DRIVE_NO_ROOT_DIR, models.DriveNoRootDir},
		{"Remote", windows.DRIVE_REMOTE, models.DriveRemote},
		{"Unknown", windows.DRIVE_UNKNOWN, models.DriveUnknown},
		{"Invalid", 999, models.DriveUnknown},
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

func TestFormatMountPath(t *testing.T) {
	collector := NewDiskCollector()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Drive letter", `C:\`, "C:"},
		{"Drive letter 2", `D:\`, "D:"},
		{"Mount point", `C:\Mount\`, `C:\Mount`},
		{"Mount point no trailing", `C:\Mount`, `C:\Mount`},
		{"Root", `\`, ``},
		{"Empty", "", ""},
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

func TestDriveInfoCalculations(t *testing.T) {
	drive := models.DriveInfo{
		Letter:     "C:",
		TotalBytes: 1000,
		FreeBytes:  400,
	}

	// Проверяем UsedBytes
	if drive.UsedBytes != 0 {
		t.Error("UsedBytes should be 0 initially")
	}

	// Вычисляем UsedBytes
	drive.UsedBytes = drive.TotalBytes - drive.FreeBytes
	if drive.UsedBytes != 600 {
		t.Errorf("UsedBytes = %d, want 600", drive.UsedBytes)
	}

	// Проверяем, что FreeBytes не превышает TotalBytes
	if drive.FreeBytes > drive.TotalBytes {
		t.Error("FreeBytes > TotalBytes")
	}
}

func TestDriveTypeString(t *testing.T) {
	tests := []struct {
		driveType models.DriveType
		expected  string
	}{
		{models.DriveUnknown, "UNKNOWN"},
		{models.DriveNoRootDir, "NO_ROOT_DIR"},
		{models.DriveRemovable, "REMOVABLE"},
		{models.DriveFixed, "FIXED"},
		{models.DriveRemote, "REMOTE"},
		{models.DriveCDROM, "CD_ROM"},
		{models.DriveRAM, "RAM_DISK"},
	}

	for _, tt := range tests {
		result := tt.driveType.String()
		if result != tt.expected {
			t.Errorf("DriveType(%d).String() = %s, want %s", tt.driveType, result, tt.expected)
		}
	}
}

func TestDriveTypeJSON(t *testing.T) {
	tests := []struct {
		driveType models.DriveType
		expected  string
	}{
		{models.DriveFixed, `"FIXED"`},
		{models.DriveRemovable, `"REMOVABLE"`},
		{models.DriveCDROM, `"CD_ROM"`},
	}

	for _, tt := range tests {
		jsonData, err := tt.driveType.MarshalJSON()
		if err != nil {
			t.Errorf("MarshalJSON failed: %v", err)
		}

		if string(jsonData) != tt.expected {
			t.Errorf("MarshalJSON() = %s, want %s", jsonData, tt.expected)
		}
	}
}

func TestDiskCollectorCDROM(t *testing.T) {
	collector := NewDiskCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	foundCDROM := false
	for _, drive := range statuses {
		if drive.Type == models.DriveCDROM {
			foundCDROM = true
			t.Logf("CD-ROM drive found: %s", drive.Letter)

			// CD-ROM может быть не готов (пустой привод)
			if drive.IsReady {
				if drive.FSType == "" {
					t.Error("Ready CD-ROM has empty filesystem type")
				}
			}
			break
		}
	}

	if !foundCDROM {
		t.Log("No CD-ROM drives found (may be normal)")
	}
}

func TestDiskCollectorRemovable(t *testing.T) {
	collector := NewDiskCollector()

	statuses, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	foundRemovable := false
	for _, drive := range statuses {
		if drive.Type == models.DriveRemovable {
			foundRemovable = true
			t.Logf("Removable drive found: %s (FS: %s)", drive.Letter, drive.FSType)
			break
		}
	}

	if !foundRemovable {
		t.Log("No removable drives found (may be normal)")
	}
}

func BenchmarkDiskCollector(b *testing.B) {
	collector := NewDiskCollector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.Collect()
		if err != nil {
			b.Fatalf("Collect failed: %v", err)
		}
	}
}

func BenchmarkDiskCollectorParallel(b *testing.B) {
	collector := NewDiskCollector()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := collector.Collect()
			if err != nil {
				b.Fatalf("Collect failed: %v", err)
			}
		}
	})
}
