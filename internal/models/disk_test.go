// tracker/internal/models/disk_test.go
package models

import (
	"encoding/json"
	"testing"
)

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
		// Marshal
		data, err := original.MarshalJSON()
		if err != nil {
			t.Fatalf("MarshalJSON failed for %v: %v", original, err)
		}

		// Unmarshal
		var restored DriveType
		err = restored.UnmarshalJSON(data)
		if err != nil {
			t.Fatalf("UnmarshalJSON failed for %s: %v", data, err)
		}

		// Сравниваем
		if restored != original {
			t.Errorf("Round trip failed: %v -> %v", original, restored)
		}
	}
}

func TestDriveInfoJSON(t *testing.T) {
	drive := DriveInfo{
		Letter:       "C:",
		Type:         DriveFixed,
		FSType:       "NTFS",
		TotalBytes:   500000000000,
		FreeBytes:    250000000000,
		UsedBytes:    250000000000,
		VolumeName:   "System",
		SerialNumber: 123456,
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

	if restored.TotalBytes != drive.TotalBytes {
		t.Errorf("TotalBytes: %d != %d", restored.TotalBytes, drive.TotalBytes)
	}
}

func TestDriveInfoSerialNumber(t *testing.T) {
	tests := []struct {
		name         string
		serialNumber uint32
		expectedHex  string
		expectedStr  string
	}{
		{
			name:         "Zero",
			serialNumber: 0,
			expectedHex:  "",
			expectedStr:  "",
		},
		{
			name:         "Typical",
			serialNumber: 0x0C904DA6,
			expectedHex:  "0C904DA6",
			expectedStr:  "0C90-4DA6",
		},
		{
			name:         "Max",
			serialNumber: 0xFFFFFFFF,
			expectedHex:  "FFFFFFFF",
			expectedStr:  "FFFF-FFFF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			drive := DriveInfo{
				SerialNumber: tt.serialNumber,
			}

			if got := drive.GetSerialNumberHex(); got != tt.expectedHex {
				t.Errorf("GetSerialNumberHex() = %s, want %s", got, tt.expectedHex)
			}

			if got := drive.GetSerialNumberString(); got != tt.expectedStr {
				t.Errorf("GetSerialNumberString() = %s, want %s", got, tt.expectedStr)
			}
		})
	}
}

func TestDriveInfoJSONWithSerialNumber(t *testing.T) {
	drive := DriveInfo{
		Letter:       "C:",
		Type:         DriveFixed,
		SerialNumber: 0x0C904DA6,
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

	if restored.SerialNumber != drive.SerialNumber {
		t.Errorf("SerialNumber: %d != %d", restored.SerialNumber, drive.SerialNumber)
	}

	// Проверяем форматирование
	if restored.GetSerialNumberString() != "0C90-4DA6" {
		t.Errorf("GetSerialNumberString() = %s, want 0C90-4DA6",
			restored.GetSerialNumberString())
	}
}

func TestDriveInfoFields(t *testing.T) {
	drive := DriveInfo{
		Letter:     "D:",
		Type:       DriveRemovable,
		FSType:     "FAT32",
		TotalBytes: 16000000000,
		FreeBytes:  8000000000,
		UsedBytes:  8000000000,
		IsReady:    true,
	}

	// Проверяем вычисляемые поля
	if drive.UsedBytes != drive.TotalBytes-drive.FreeBytes {
		t.Error("UsedBytes != TotalBytes - FreeBytes")
	}

	// Проверяем, что FreeBytes <= TotalBytes
	if drive.FreeBytes > drive.TotalBytes {
		t.Error("FreeBytes > TotalBytes")
	}

	// Проверяем, что UsedBytes <= TotalBytes
	if drive.UsedBytes > drive.TotalBytes {
		t.Error("UsedBytes > TotalBytes")
	}
}

func TestDiskStatusesType(t *testing.T) {
	var statuses DiskStatuses

	if len(statuses) == 0 {
		t.Log("Empty DiskStatuses is nil")
	}

	statuses = append(statuses, DriveInfo{
		Letter: "C:",
		Type:   DriveFixed,
	})

	if len(statuses) != 1 {
		t.Errorf("Len = %d, want 1", len(statuses))
	}

	if statuses[0].Letter != "C:" {
		t.Errorf("Letter = %s, want C:", statuses[0].Letter)
	}
}

func TestDriveTypeValues(t *testing.T) {
	// Проверяем, что значения констант уникальны
	values := map[DriveType]bool{
		DriveUnknown:   true,
		DriveNoRootDir: true,
		DriveRemovable: true,
		DriveFixed:     true,
		DriveRemote:    true,
		DriveCDROM:     true,
		DriveRAM:       true,
	}

	if len(values) != 7 {
		t.Errorf("Expected 7 unique drive types, got %d", len(values))
	}
}

func TestDriveTypeOrder(t *testing.T) {
	// Проверяем порядок констант
	if DriveUnknown != 0 {
		t.Error("DriveUnknown should be 0")
	}
	if DriveNoRootDir != 1 {
		t.Error("DriveNoRootDir should be 1")
	}
	if DriveRemovable != 2 {
		t.Error("DriveRemovable should be 2")
	}
	if DriveFixed != 3 {
		t.Error("DriveFixed should be 3")
	}
	if DriveRemote != 4 {
		t.Error("DriveRemote should be 4")
	}
	if DriveCDROM != 5 {
		t.Error("DriveCDROM should be 5")
	}
	if DriveRAM != 6 {
		t.Error("DriveRAM should be 6")
	}
}

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
