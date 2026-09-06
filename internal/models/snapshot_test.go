// tracker/internal/models/snapshot_test.go
package models

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ============ Тесты для SystemSnapshot полей ============

func TestSystemSnapshotFields(t *testing.T) {
	snapshot := SystemSnapshot{
		Timestamp: time.Now().Unix(),
		User: UserInfo{
			Username: `ACME\john.doe`,
		},
		OS: OSInfo{
			Name: "Windows 10",
		},
		Processor: ProcessorInfo{
			Model: "Intel Xeon",
		},
		RAM: RAMInfo{
			TotalBytes: 42869710848,
		},
		Drives: DiskStatuses{
			{Letter: "C:", Type: DriveFixed},
		},
		Services: ServicesStatuses{
			{Name: "RDP"},
		},
		Network: NetworkStatuses{
			{Index: 1, Name: "Ethernet"},
		},
		Host: HostInfo{
			Hostname: "DESKTOP-ABC123",
		},
	}

	if snapshot.Timestamp == 0 {
		t.Error("Timestamp is 0")
	}

	if snapshot.User.Username == "" {
		t.Error("User.Username is empty")
	}

	if snapshot.OS.Name == "" {
		t.Error("OS.Name is empty")
	}

	if snapshot.Processor.Model == "" {
		t.Error("Processor.Model is empty")
	}

	if snapshot.RAM.TotalBytes == 0 {
		t.Error("RAM.TotalBytes is 0")
	}

	if len(snapshot.Drives) == 0 {
		t.Error("Drives is empty")
	}

	if len(snapshot.Services) == 0 {
		t.Error("Services is empty")
	}

	if len(snapshot.Network) == 0 {
		t.Error("Network is empty")
	}

	if snapshot.Host.Hostname == "" {
		t.Error("Host.Hostname is empty")
	}
}

// ============ Тесты для GetTimestamp() ============

func TestSystemSnapshotGetTimestamp(t *testing.T) {
	unixTime := int64(1747109492)
	snapshot := SystemSnapshot{Timestamp: unixTime}

	// Проверяем конвертацию в time.Time
	timeValue := time.Unix(snapshot.Timestamp, 0)

	if timeValue.Year() != 2025 {
		t.Errorf("Year = %d, want 2025", timeValue.Year())
	}

	if timeValue.Month() != 5 {
		t.Errorf("Month = %d, want 5", timeValue.Month())
	}

	if timeValue.Day() != 13 {
		t.Errorf("Day = %d, want 13", timeValue.Day())
	}
}

// ============ Тесты для SystemSnapshot JSON ============

func TestSystemSnapshotJSON(t *testing.T) {
	snapshot := SystemSnapshot{
		Timestamp: 1747109492,
		User: UserInfo{
			Username: `ACME\john.doe`,
			FullName: "John Doe",
		},
		OS: OSInfo{
			Name:         "Windows 10 22H2",
			Architecture: AMD64,
			MachineGUID:  BinaryUUID(uuid.New()),
		},
		Processor: ProcessorInfo{
			Model:         "Intel Xeon E5-1620",
			PhysicalCores: 4,
		},
		RAM: RAMInfo{
			TotalBytes: 42869710848,
		},
		Host: HostInfo{
			Hostname: "DESKTOP-ABC123",
			BootTime: 1747109492,
		},
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	t.Logf("JSON length: %d bytes", len(data))

	var restored SystemSnapshot
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if restored.Timestamp != snapshot.Timestamp {
		t.Errorf("Timestamp: %d != %d", restored.Timestamp, snapshot.Timestamp)
	}

	if restored.User.Username != snapshot.User.Username {
		t.Errorf("User.Username: %s != %s", restored.User.Username, snapshot.User.Username)
	}

	if restored.OS.Name != snapshot.OS.Name {
		t.Errorf("OS.Name: %s != %s", restored.OS.Name, snapshot.OS.Name)
	}

	if restored.Processor.Model != snapshot.Processor.Model {
		t.Errorf("Processor.Model: %s != %s", restored.Processor.Model, snapshot.Processor.Model)
	}

	if restored.RAM.TotalBytes != snapshot.RAM.TotalBytes {
		t.Errorf("RAM.TotalBytes: %d != %d", restored.RAM.TotalBytes, snapshot.RAM.TotalBytes)
	}

	if restored.Host.Hostname != snapshot.Host.Hostname {
		t.Errorf("Host.Hostname: %s != %s", restored.Host.Hostname, snapshot.Host.Hostname)
	}
}

// ============ Тесты на JSON поля ============

func TestSystemSnapshotJSONFields(t *testing.T) {
	snapshot := SystemSnapshot{}

	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	jsonStr := string(data)

	requiredFields := []string{
		"timestamp",
		"user",
		"os",
		"processor",
		"ram",
		"drives",
		"services",
		"network",
		"host",
	}

	for _, field := range requiredFields {
		if !containsSubstring(jsonStr, `"`+field+`"`) {
			t.Errorf("JSON missing field: %s", field)
		}
	}
}

// ============ Тесты для пустого snapshot ============

func TestSystemSnapshotEmpty(t *testing.T) {
	snapshot := SystemSnapshot{}

	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	t.Logf("Empty JSON: %s", data)

	var restored SystemSnapshot
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
}

// ============ Тесты для полного snapshot ============

func TestSystemSnapshotFull(t *testing.T) {
	snapshot := createFullSnapshot()

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	t.Logf("Full JSON:\n%s", data)

	var restored SystemSnapshot
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// Проверяем ключевые поля
	if restored.Timestamp != snapshot.Timestamp {
		t.Errorf("Timestamp: %d != %d", restored.Timestamp, snapshot.Timestamp)
	}

	if restored.OS.Architecture != snapshot.OS.Architecture {
		t.Errorf("Architecture: %v != %v", restored.OS.Architecture, snapshot.OS.Architecture)
	}

	if restored.Processor.PhysicalCores != snapshot.Processor.PhysicalCores {
		t.Errorf("PhysicalCores: %d != %d", restored.Processor.PhysicalCores, snapshot.Processor.PhysicalCores)
	}

	if len(restored.Drives) != len(snapshot.Drives) {
		t.Errorf("Drives: %d != %d", len(restored.Drives), len(snapshot.Drives))
	}

	if len(restored.Services) != len(snapshot.Services) {
		t.Errorf("Services: %d != %d", len(restored.Services), len(snapshot.Services))
	}

	if len(restored.Network) != len(snapshot.Network) {
		t.Errorf("Network: %d != %d", len(restored.Network), len(snapshot.Network))
	}
}

// ============ Вспомогательная функция ============

func createFullSnapshot() SystemSnapshot {
	return SystemSnapshot{
		Timestamp: time.Now().Unix(),
		User: UserInfo{
			Username:     `ACME\john.doe`,
			FullName:     "John Doe",
			Domain:       "ACME",
			DomainFull:   "acme.corp.local",
			ProfilePath:  `C:\Users\john.doe`,
			IsAdmin:      true,
			IsDomainUser: true,
			IsLocalUser:  false,
		},
		OS: OSInfo{
			Name:          "Windows 10 22H2",
			Edition:       "Professional",
			BuildNumber:   "19045",
			KernelVersion: "10.0.19045",
			Architecture:  AMD64,
			Locale:        "en-US",
			InstallDate:   1730764800,
			MachineGUID:   BinaryUUID(uuid.New()),
			IsVirtual:     false,
			IsHypervisor:  true,
		},
		Processor: ProcessorInfo{
			Model:             "Intel(R) Xeon(R) CPU E5-1620 0 @ 3.60GHz",
			VendorID:          "GenuineIntel",
			PhysicalCores:     4,
			LogicalProcessors: 8,
			BaseSpeedMHz:      3591,
			SMTEnabled:        true,
			L1CacheBytes:      262144,
			L2CacheBytes:      1048576,
			L3CacheBytes:      10485760,
		},
		RAM: RAMInfo{
			TotalBytes:        42869710848,
			AvailableBytes:    23083425792,
			TotalPageFile:     68526669824,
			AvailablePageFile: 37414522880,
		},
		Drives: DiskStatuses{
			{
				Letter:       "C:",
				Type:         DriveFixed,
				FSType:       "NTFS",
				TotalBytes:   498951073792,
				FreeBytes:    254503497728,
				SerialNumber: 0xDEA5D590,
				IsReady:      true,
			},
		},
		Services: ServicesStatuses{
			{
				Name:        "RDP",
				ServiceName: "TermService",
				Installed:   true,
				Running:     true,
				Port:        3389,
				PortOpen:    true,
			},
		},
		Network: NetworkStatuses{
			{
				Index:        1,
				Name:         "Ethernet",
				Description:  "Intel Gigabit",
				MAC:          "a0:48:1c:a9:f6:c0",
				Type:         TypeEthernet,
				Operational:  true,
				IPAssignment: AssignmentStatic,
				IPAddresses:  []net.IP{net.ParseIP("172.17.113.11")},
			},
		},
		Host: HostInfo{
			Hostname:      "DESKTOP-ABC123",
			FQDN:          "DESKTOP-ABC123.acme.corp.local",
			Domain:        "acme.corp.local",
			UpTimeSeconds: 9798016,
			BootTime:      1747109492,
			Manufacturer:  "Dell Inc.",
			Model:         "Precision T3610",
			BIOSVendor:    "Dell Inc.",
			BIOSVersion:   "A18",
		},
	}
}

// ============ Бенчмарки ============

func BenchmarkSystemSnapshotJSON(b *testing.B) {
	snapshot := createFullSnapshot()

	b.ResetTimer()
	for b.Loop() {
		_, _ = json.Marshal(snapshot)
	}
}

func BenchmarkSystemSnapshotJSONIndent(b *testing.B) {
	snapshot := createFullSnapshot()

	b.ResetTimer()
	for b.Loop() {
		_, _ = json.MarshalIndent(snapshot, "", "  ")
	}
}

// Вспомогательная функция
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
