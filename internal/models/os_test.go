// tracker/internal/models/os_test.go
package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ============ Тесты для OSInfo.GetInstallDateString() ============

func TestGetInstallDateString(t *testing.T) {
	tests := []struct {
		name        string
		installDate int64
		expected    string
	}{
		{"Zero", 0, "UNKNOWN"},
		{"Valid date", 1730764800, "2024-11-05"},
		{"Old date", 0, "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			osInfo := OSInfo{InstallDate: tt.installDate}
			result := osInfo.GetInstallDateString()

			if result != tt.expected {
				t.Errorf("GetInstallDateString() = %s, want %s", result, tt.expected)
			}
		})
	}
}

// ============ Тесты для OSInfo.GetInstallDateTime() ============

func TestGetInstallDateTime(t *testing.T) {
	tests := []struct {
		name        string
		installDate int64
		expected    time.Time
	}{
		{"Zero", 0, time.Time{}},
		{"Valid", 1730764800, time.Unix(1730764800, 0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			osInfo := OSInfo{InstallDate: tt.installDate}
			result := osInfo.GetInstallDateTime()

			if !result.Equal(tt.expected) {
				t.Errorf("GetInstallDateTime() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// ============ Тесты для BinaryUUID.String() ============

func TestBinaryUUIDString(t *testing.T) {
	// Создаем UUID
	id := uuid.New()
	binaryUUID := BinaryUUID(id)

	result := binaryUUID.String()

	if result != id.String() {
		t.Errorf("String() = %s, want %s", result, id.String())
	}
}

// ============ Тесты для BinaryUUID.MarshalJSON() ============

func TestBinaryUUIDMarshalJSON(t *testing.T) {
	id := uuid.New()
	binaryUUID := BinaryUUID(id)

	data, err := binaryUUID.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	expected := `"` + id.String() + `"`
	if string(data) != expected {
		t.Errorf("MarshalJSON() = %s, want %s", data, expected)
	}
}

// ============ Тесты для BinaryUUID.UnmarshalJSON() ============

func TestBinaryUUIDUnmarshalJSON(t *testing.T) {
	id := uuid.New()
	jsonData := `"` + id.String() + `"`

	var binaryUUID BinaryUUID
	err := binaryUUID.UnmarshalJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if binaryUUID.String() != id.String() {
		t.Errorf("UnmarshalJSON() = %s, want %s", binaryUUID.String(), id.String())
	}
}

func TestBinaryUUIDUnmarshalJSONInvalid(t *testing.T) {
	var binaryUUID BinaryUUID

	err := binaryUUID.UnmarshalJSON([]byte(`"invalid-uuid"`))
	if err == nil {
		t.Error("UnmarshalJSON should fail for invalid UUID")
	}
}

// ============ Тесты для BinaryUUID JSON round-trip ============

func TestBinaryUUIDJSONRoundTrip(t *testing.T) {
	id := uuid.New()
	original := BinaryUUID(id)

	data, err := original.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	var restored BinaryUUID
	err = restored.UnmarshalJSON(data)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if restored != original {
		t.Errorf("Round trip failed: %v != %v", restored, original)
	}
}

// ============ Тесты для ArchFamilyType.String() ============

func TestArchFamilyTypeString(t *testing.T) {
	tests := []struct {
		name     string
		arch     ArchFamilyType
		expected string
	}{
		{"AMD64", AMD64, "AMD64"},
		{"ARM", ARM, "ARM"},
		{"ARM64", ARM64, "ARM64"},
		{"I386", I386, "I386"},
		{"LOONG64", LOONG64, "LOONG64"},
		{"MIPS", MIPS, "MIPS"},
		{"MIPS64", MIPS64, "MIPS64"},
		{"PPC64", PPC64, "PPC64"},
		{"RISCV64", RISCV64, "RISCV64"},
		{"S390X", S390X, "S390X"},
		{"WASM", WASM, "WASM"},
		{"Unknown", UnknownArch, "UNKNOWN"},
		{"Invalid", ArchFamilyType(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.arch.String()
			if result != tt.expected {
				t.Errorf("ArchFamilyType(%d).String() = %s, want %s",
					tt.arch, result, tt.expected)
			}
		})
	}
}

// ============ Тесты для ArchFamilyType.MarshalJSON() ============

func TestArchFamilyTypeMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		arch     ArchFamilyType
		expected string
	}{
		{"AMD64", AMD64, `"AMD64"`},
		{"ARM64", ARM64, `"ARM64"`},
		{"Unknown", UnknownArch, `"UNKNOWN"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.arch.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON failed: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("MarshalJSON() = %s, want %s", data, tt.expected)
			}
		})
	}
}

// ============ Тесты для ArchFamilyType.UnmarshalJSON() ============

func TestArchFamilyTypeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected ArchFamilyType
	}{
		{"AMD64", `"AMD64"`, AMD64},
		{"ARM", `"ARM"`, ARM},
		{"ARM64", `"ARM64"`, ARM64},
		{"I386", `"I386"`, I386},
		{"LOONG64", `"LOONG64"`, LOONG64},
		{"MIPS", `"MIPS"`, MIPS},
		{"MIPS64", `"MIPS64"`, MIPS64},
		{"PPC64", `"PPC64"`, PPC64},
		{"RISCV64", `"RISCV64"`, RISCV64},
		{"S390X", `"S390X"`, S390X},
		{"WASM", `"WASM"`, WASM},
		{"Invalid", `"INVALID"`, UnknownArch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var arch ArchFamilyType
			err := arch.UnmarshalJSON([]byte(tt.json))
			if err != nil {
				t.Fatalf("UnmarshalJSON failed: %v", err)
			}

			if arch != tt.expected {
				t.Errorf("UnmarshalJSON(%s) = %v, want %v",
					tt.json, arch, tt.expected)
			}
		})
	}
}

// ============ Тесты для ArchFamilyType JSON round-trip ============

func TestArchFamilyTypeJSONRoundTrip(t *testing.T) {
	archs := []ArchFamilyType{
		AMD64, ARM, ARM64, I386,
		LOONG64, MIPS, MIPS64, PPC64,
		RISCV64, S390X, WASM, UnknownArch,
	}

	for _, original := range archs {
		data, err := original.MarshalJSON()
		if err != nil {
			t.Fatalf("MarshalJSON failed: %v", err)
		}

		var restored ArchFamilyType
		err = restored.UnmarshalJSON(data)
		if err != nil {
			t.Fatalf("UnmarshalJSON failed: %v", err)
		}

		if restored != original {
			t.Errorf("Round trip failed: %v -> %v", original, restored)
		}
	}
}

// ============ Тесты для OSInfo JSON ============

func TestOSInfoJSON(t *testing.T) {
	id := uuid.New()

	osInfo := OSInfo{
		Name:          "Windows 10 22H2",
		Edition:       "Professional",
		BuildNumber:   "19045",
		KernelVersion: "10.0.19045",
		Architecture:  AMD64,
		Locale:        "ru-RU",
		InstallDate:   1730764800,
		MachineGUID:   BinaryUUID(id),
		IsVirtual:     false,
		IsHypervisor:  true,
	}

	data, err := json.Marshal(osInfo)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	t.Logf("JSON: %s", data)

	var restored OSInfo
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if restored.Name != osInfo.Name {
		t.Errorf("Name: %s != %s", restored.Name, osInfo.Name)
	}

	if restored.Architecture != osInfo.Architecture {
		t.Errorf("Architecture: %v != %v", restored.Architecture, osInfo.Architecture)
	}

	if restored.InstallDate != osInfo.InstallDate {
		t.Errorf("InstallDate: %d != %d", restored.InstallDate, osInfo.InstallDate)
	}

	if restored.MachineGUID != osInfo.MachineGUID {
		t.Errorf("MachineGUID: %s != %s", restored.MachineGUID, osInfo.MachineGUID)
	}
}

// ============ Бенчмарки ============

func BenchmarkGetInstallDateString(b *testing.B) {
	osInfo := OSInfo{InstallDate: 1730764800}

	b.ResetTimer()
	for b.Loop() {
		_ = osInfo.GetInstallDateString()
	}
}

func BenchmarkGetInstallDateTime(b *testing.B) {
	osInfo := OSInfo{InstallDate: 1730764800}

	b.ResetTimer()
	for b.Loop() {
		_ = osInfo.GetInstallDateTime()
	}
}

func BenchmarkBinaryUUIDString(b *testing.B) {
	id := BinaryUUID(uuid.New())

	b.ResetTimer()
	for b.Loop() {
		_ = id.String()
	}
}

func BenchmarkBinaryUUIDMarshalJSON(b *testing.B) {
	id := BinaryUUID(uuid.New())

	b.ResetTimer()
	for b.Loop() {
		_, _ = id.MarshalJSON()
	}
}

func BenchmarkArchFamilyTypeString(b *testing.B) {
	b.ResetTimer()
	for b.Loop() {
		_ = AMD64.String()
	}
}

func BenchmarkArchFamilyTypeMarshalJSON(b *testing.B) {
	b.ResetTimer()
	for b.Loop() {
		_, _ = AMD64.MarshalJSON()
	}
}

func BenchmarkOSInfoJSON(b *testing.B) {
	osInfo := OSInfo{
		Name:         "Windows 10",
		Architecture: AMD64,
		InstallDate:  1730764800,
		MachineGUID:  BinaryUUID(uuid.New()),
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = json.Marshal(osInfo)
	}
}
