// tracker/internal/models/disk_test.go
package models

import (
	"encoding/json"
	"testing"
)

// ============ Тесты для DriveType.String() ============

func TestDriveTypeString(t *testing.T) {
	tests := []struct {
		name      string
		driveType DriveType
		expected  string
	}{
		{"Unknown", DriveUnknown, "UNKNOWN"},
		{"NoRootDir", DriveNoRootDir, "NO_ROOT_DIR"},
		{"Removable", DriveRemovable, "REMOVABLE"},
		{"Fixed", DriveFixed, "FIXED"},
		{"Remote", DriveRemote, "REMOTE"},
		{"CDROM", DriveCDROM, "CD_ROM"},
		{"RAM", DriveRAM, "RAM_DISK"},
		{"Invalid", DriveType(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.driveType.String()
			if result != tt.expected {
				t.Errorf("DriveType(%d).String() = %s, want %s",
					tt.driveType, result, tt.expected)
			}
		})
	}
}

// ============ Тесты для DriveType.MarshalJSON() ============

func TestDriveTypeMarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		driveType DriveType
		expected  string
	}{
		{"Unknown", DriveUnknown, `"UNKNOWN"`},
		{"NoRootDir", DriveNoRootDir, `"NO_ROOT_DIR"`},
		{"Removable", DriveRemovable, `"REMOVABLE"`},
		{"Fixed", DriveFixed, `"FIXED"`},
		{"Remote", DriveRemote, `"REMOTE"`},
		{"CDROM", DriveCDROM, `"CD_ROM"`},
		{"RAM", DriveRAM, `"RAM_DISK"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.driveType.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON failed: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("MarshalJSON() = %s, want %s", data, tt.expected)
			}
		})
	}
}

// ============ Тесты для DriveType.UnmarshalJSON() ============

func TestDriveTypeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected DriveType
	}{
		{"Unknown", `"UNKNOWN"`, DriveUnknown},
		{"NoRootDir", `"NO_ROOT_DIR"`, DriveNoRootDir},
		{"Removable", `"REMOVABLE"`, DriveRemovable},
		{"Fixed", `"FIXED"`, DriveFixed},
		{"Remote", `"REMOTE"`, DriveRemote},
		{"CDROM", `"CD_ROM"`, DriveCDROM},
		{"RAM", `"RAM_DISK"`, DriveRAM},
		{"Invalid", `"INVALID"`, DriveUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var driveType DriveType
			err := driveType.UnmarshalJSON([]byte(tt.json))
			if err != nil {
				t.Fatalf("UnmarshalJSON failed: %v", err)
			}

			if driveType != tt.expected {
				t.Errorf("UnmarshalJSON(%s) = %v, want %v",
					tt.json, driveType, tt.expected)
			}
		})
	}
}

// ============ Тесты для DriveType JSON round-trip ============

func TestDriveTypeJSONRoundTrip(t *testing.T) {
	types := []DriveType{
		DriveUnknown,
		DriveNoRootDir,
		DriveRemovable,
		DriveFixed,
		DriveRemote,
		DriveCDROM,
		DriveRAM,
	}

	for _, original := range types {
		data, err := original.MarshalJSON()
		if err != nil {
			t.Fatalf("MarshalJSON failed: %v", err)
		}

		var restored DriveType
		err = restored.UnmarshalJSON(data)
		if err != nil {
			t.Fatalf("UnmarshalJSON failed: %v", err)
		}

		if restored != original {
			t.Errorf("Round trip failed: %v -> %v", original, restored)
		}
	}
}

// ============ Тесты для GetSerialNumberString() ============

func TestGetSerialNumberString(t *testing.T) {
	tests := []struct {
		name         string
		serialNumber uint32
		expected     string
	}{
		{"Zero", 0, ""},
		{"Typical", 0x0C904DA6, "0C90-4DA6"},
		{"Max", 0xFFFFFFFF, "FFFF-FFFF"},
		{"Min", 0x00000001, "0000-0001"},
		{"All zeros", 0x00000000, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			drive := DriveInfo{SerialNumber: tt.serialNumber}
			result := drive.GetSerialNumberString()

			if result != tt.expected {
				t.Errorf("GetSerialNumberString() = %s, want %s", result, tt.expected)
			}
		})
	}
}

// ============ Тесты для GetSerialNumberHex() ============

func TestGetSerialNumberHex(t *testing.T) {
	tests := []struct {
		name         string
		serialNumber uint32
		expected     string
	}{
		{"Zero", 0, ""},
		{"Typical", 0x0C904DA6, "0C904DA6"},
		{"Max", 0xFFFFFFFF, "FFFFFFFF"},
		{"Min", 0x00000001, "00000001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			drive := DriveInfo{SerialNumber: tt.serialNumber}
			result := drive.GetSerialNumberHex()

			if result != tt.expected {
				t.Errorf("GetSerialNumberHex() = %s, want %s", result, tt.expected)
			}
		})
	}
}

// ============ Тесты для DriveInfo JSON ============

func TestDriveInfoJSON(t *testing.T) {
	drive := DriveInfo{
		Letter:       "C:",
		Type:         DriveFixed,
		FSType:       "NTFS",
		TotalBytes:   500000000000,
		FreeBytes:    250000000000,
		UsedBytes:    250000000000,
		VolumeName:   "System",
		SerialNumber: 0xDEA5D590,
		IsReady:      true,
	}

	data, err := json.Marshal(drive)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	t.Logf("JSON: %s", data)

	var restored DriveInfo
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if restored.Letter != drive.Letter {
		t.Errorf("Letter: %s != %s", restored.Letter, drive.Letter)
	}

	if restored.Type != drive.Type {
		t.Errorf("Type: %v != %v", restored.Type, drive.Type)
	}

	if restored.SerialNumber != drive.SerialNumber {
		t.Errorf("SerialNumber: %d != %d", restored.SerialNumber, drive.SerialNumber)
	}

	if restored.TotalBytes != drive.TotalBytes {
		t.Errorf("TotalBytes: %d != %d", restored.TotalBytes, drive.TotalBytes)
	}
}

// ============ Тесты для DiskStatuses ============

func TestDiskStatusesType(t *testing.T) {
	var statuses DiskStatuses

	if statuses != nil {
		t.Error("Empty DiskStatuses should be nil")
	}

	statuses = append(statuses, DriveInfo{Letter: "C:", Type: DriveFixed})

	if len(statuses) != 1 {
		t.Errorf("Len = %d, want 1", len(statuses))
	}

	if statuses[0].Letter != "C:" {
		t.Errorf("Letter = %s, want C:", statuses[0].Letter)
	}
}

// ============ Бенчмарки ============

func BenchmarkDriveTypeString(b *testing.B) {
	for b.Loop() {
		_ = DriveFixed.String()
	}
}

func BenchmarkDriveTypeMarshalJSON(b *testing.B) {
	for b.Loop() {
		_, _ = DriveFixed.MarshalJSON()
	}
}

func BenchmarkDriveTypeUnmarshalJSON(b *testing.B) {
	data := []byte(`"FIXED"`)

	b.ResetTimer()
	for b.Loop() {
		var dt DriveType
		_ = dt.UnmarshalJSON(data)
	}
}

func BenchmarkGetSerialNumberString(b *testing.B) {
	drive := DriveInfo{SerialNumber: 0x0C904DA6}

	b.ResetTimer()
	for b.Loop() {
		_ = drive.GetSerialNumberString()
	}
}

func BenchmarkGetSerialNumberHex(b *testing.B) {
	drive := DriveInfo{SerialNumber: 0x0C904DA6}

	b.ResetTimer()
	for b.Loop() {
		_ = drive.GetSerialNumberHex()
	}
}

func BenchmarkDriveInfoJSON(b *testing.B) {
	drive := DriveInfo{
		Letter:     "C:",
		Type:       DriveFixed,
		FSType:     "NTFS",
		TotalBytes: 500000000000,
		FreeBytes:  250000000000,
		IsReady:    true,
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = json.Marshal(drive)
	}
}
