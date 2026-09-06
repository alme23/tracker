// tracker/internal/binproto/decoder_test.go
package binproto

import (
	"net"
	"testing"
	"time"

	"github.com/alme23/tracker/internal/models"
	"github.com/google/uuid"
)

// ============ Тесты для NewDecoder() ============

func TestNewDecoder(t *testing.T) {
	snapshot := createTestSnapshot()
	encoder := NewEncoder()
	data, _ := encoder.Encode(snapshot)

	decoder := NewDecoder(data)

	if decoder == nil {
		t.Fatal("NewDecoder returned nil")
	}

	if decoder.buf == nil {
		t.Error("buf is nil")
	}
}

// ============ Тесты для Decode() ============

func TestDecode(t *testing.T) {
	original := createTestSnapshot()
	encoder := NewEncoder()
	data, err := encoder.Encode(original)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoder := NewDecoder(data)
	decoded, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Timestamp (int64)
	if decoded.Timestamp != original.Timestamp {
		t.Errorf("Timestamp: %d != %d", decoded.Timestamp, original.Timestamp)
	}

	if decoded.User.Username != original.User.Username {
		t.Errorf("User.Username: %s != %s", decoded.User.Username, original.User.Username)
	}

	if decoded.OS.Name != original.OS.Name {
		t.Errorf("OS.Name: %s != %s", decoded.OS.Name, original.OS.Name)
	}

	if decoded.Processor.Model != original.Processor.Model {
		t.Errorf("Processor.Model: %s != %s", decoded.Processor.Model, original.Processor.Model)
	}

	if decoded.RAM.TotalBytes != original.RAM.TotalBytes {
		t.Errorf("RAM.TotalBytes: %d != %d", decoded.RAM.TotalBytes, original.RAM.TotalBytes)
	}

	if decoded.Host.Hostname != original.Host.Hostname {
		t.Errorf("Host.Hostname: %s != %s", decoded.Host.Hostname, original.Host.Hostname)
	}
}

// ============ Тесты на ошибки ============

func TestDecodeInvalidMagic(t *testing.T) {
	data := []byte("BAD00rest of data")

	decoder := NewDecoder(data)
	_, err := decoder.Decode()

	if err == nil {
		t.Error("Expected error for invalid magic header")
	}

	t.Logf("Error: %v", err)
}

func TestDecodeEmptyData(t *testing.T) {
	decoder := NewDecoder([]byte{})
	_, err := decoder.Decode()

	if err == nil {
		t.Error("Expected error for empty data")
	}
}

func TestDecodeTruncatedData(t *testing.T) {
	snapshot := createTestSnapshot()
	encoder := NewEncoder()
	fullData, _ := encoder.Encode(snapshot)

	truncatedData := fullData[:len(fullData)/2]

	decoder := NewDecoder(truncatedData)
	_, err := decoder.Decode()

	if err == nil {
		t.Error("Expected error for truncated data")
	}
}

// ============ Тесты на Round-trip ============

func TestRoundTrip(t *testing.T) {
	original := createTestSnapshot()

	encoder := NewEncoder()
	data, err := encoder.Encode(original)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoder := NewDecoder(data)
	decoded, err := decoder.Decode()
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	assertSnapshotsEqual(t, original, decoded)
}

// ============ Тесты для readString() ============

func TestReadString(t *testing.T) {
	encoder := NewEncoder()
	_ = encoder.writeString("hello")
	data := encoder.buf.Bytes()

	decoder := NewDecoder(data)
	result, err := decoder.readString()

	if err != nil {
		t.Fatalf("readString failed: %v", err)
	}

	if result != "hello" {
		t.Errorf("readString() = %q, want %q", result, "hello")
	}
}

func TestReadStringEmpty(t *testing.T) {
	encoder := NewEncoder()
	_ = encoder.writeString("")
	data := encoder.buf.Bytes()

	decoder := NewDecoder(data)
	result, err := decoder.readString()

	if err != nil {
		t.Fatalf("readString failed: %v", err)
	}

	if result != "" {
		t.Errorf("readString() = %q, want empty", result)
	}
}

func TestReadStringTruncated(t *testing.T) {
	data := []byte{0x05}

	decoder := NewDecoder(data)
	_, err := decoder.readString()

	if err == nil {
		t.Error("Expected error for truncated string")
	}
}

// ============ Тесты для decodeUser() ============

func TestDecodeUser(t *testing.T) {
	original := &models.UserInfo{
		Username:     `ACME\john.doe`,
		FullName:     "John Doe",
		Domain:       "ACME",
		DomainFull:   "acme.corp.local",
		Workgroup:    "",
		ProfilePath:  `C:\Users\john.doe`,
		IsAdmin:      true,
		IsDomainUser: true,
		IsLocalUser:  false,
	}

	encoder := NewEncoder()
	_ = encoder.encodeUser(original)
	data := encoder.buf.Bytes()

	decoder := NewDecoder(data)
	decoded := &models.UserInfo{}
	err := decoder.decodeUser(decoded)

	if err != nil {
		t.Fatalf("decodeUser failed: %v", err)
	}

	if decoded.Username != original.Username {
		t.Errorf("Username: %s != %s", decoded.Username, original.Username)
	}

	if decoded.IsAdmin != original.IsAdmin {
		t.Errorf("IsAdmin: %v != %v", decoded.IsAdmin, original.IsAdmin)
	}
}

// ============ Тесты для decodeOS() ============

func TestDecodeOS(t *testing.T) {
	original := &models.OSInfo{
		Name:         "Windows 10",
		Edition:      "Professional",
		BuildNumber:  "19045",
		Architecture: models.AMD64,
		InstallDate:  1730764800,
		MachineGUID:  models.BinaryUUID(uuid.New()),
		IsHypervisor: true,
	}

	encoder := NewEncoder()
	_ = encoder.encodeOS(original)
	data := encoder.buf.Bytes()

	decoder := NewDecoder(data)
	decoded := &models.OSInfo{}
	err := decoder.decodeOS(decoded)

	if err != nil {
		t.Fatalf("decodeOS failed: %v", err)
	}

	if decoded.Name != original.Name {
		t.Errorf("Name: %s != %s", decoded.Name, original.Name)
	}

	if decoded.Architecture != original.Architecture {
		t.Errorf("Architecture: %v != %v", decoded.Architecture, original.Architecture)
	}

	if decoded.InstallDate != original.InstallDate {
		t.Errorf("InstallDate: %d != %d", decoded.InstallDate, original.InstallDate)
	}

	if decoded.MachineGUID != original.MachineGUID {
		t.Errorf("MachineGUID: %s != %s", decoded.MachineGUID, original.MachineGUID)
	}
}

// ============ Тесты для decodeProcessor() ============

func TestDecodeProcessor(t *testing.T) {
	original := &models.ProcessorInfo{
		Model:             "Intel Xeon",
		VendorID:          "GenuineIntel",
		PhysicalCores:     4,
		LogicalProcessors: 8,
		BaseSpeedMHz:      3591,
		SMTEnabled:        true,
		L1CacheBytes:      262144,
		L2CacheBytes:      1048576,
		L3CacheBytes:      10485760,
	}

	encoder := NewEncoder()
	_ = encoder.encodeProcessor(original)
	data := encoder.buf.Bytes()

	decoder := NewDecoder(data)
	decoded := &models.ProcessorInfo{}
	err := decoder.decodeProcessor(decoded)

	if err != nil {
		t.Fatalf("decodeProcessor failed: %v", err)
	}

	if decoded.Model != original.Model {
		t.Errorf("Model: %s != %s", decoded.Model, original.Model)
	}

	if decoded.PhysicalCores != original.PhysicalCores {
		t.Errorf("PhysicalCores: %d != %d", decoded.PhysicalCores, original.PhysicalCores)
	}

	if decoded.L3CacheBytes != original.L3CacheBytes {
		t.Errorf("L3CacheBytes: %d != %d", decoded.L3CacheBytes, original.L3CacheBytes)
	}
}

// ============ Тесты для decodeRAM() ============

func TestDecodeRAM(t *testing.T) {
	original := &models.RAMInfo{
		TotalBytes:     42869710848,
		AvailableBytes: 23083425792,
		Sticks: []models.RAMStick{
			{Slot: "DIMM1", Capacity: 8589934592, SpeedMHz: 1600},
			{Slot: "DIMM2", Capacity: 8589934592, SpeedMHz: 1600},
		},
	}

	encoder := NewEncoder()
	_ = encoder.encodeRAM(original)
	data := encoder.buf.Bytes()

	decoder := NewDecoder(data)
	decoded := &models.RAMInfo{}
	err := decoder.decodeRAM(decoded)

	if err != nil {
		t.Fatalf("decodeRAM failed: %v", err)
	}

	if decoded.TotalBytes != original.TotalBytes {
		t.Errorf("TotalBytes: %d != %d", decoded.TotalBytes, original.TotalBytes)
	}

	if len(decoded.Sticks) != len(original.Sticks) {
		t.Errorf("Sticks count: %d != %d", len(decoded.Sticks), len(original.Sticks))
	}

	if decoded.Sticks[0].Slot != original.Sticks[0].Slot {
		t.Errorf("Stick[0].Slot: %s != %s", decoded.Sticks[0].Slot, original.Sticks[0].Slot)
	}
}

// ============ Тесты для decodeDrives() ============

func TestDecodeDrives(t *testing.T) {
	original := models.DiskStatuses{
		{
			Letter:       "C:",
			Type:         models.DriveFixed,
			FSType:       "NTFS",
			TotalBytes:   498951073792,
			SerialNumber: 0xDEA5D590,
			IsReady:      true,
		},
	}

	encoder := NewEncoder()
	_ = encoder.encodeDrives(original)
	data := encoder.buf.Bytes()

	decoder := NewDecoder(data)
	decoded := &models.DiskStatuses{}
	err := decoder.decodeDrives(decoded)

	if err != nil {
		t.Fatalf("decodeDrives failed: %v", err)
	}

	if len(*decoded) != len(original) {
		t.Errorf("Drives count: %d != %d", len(*decoded), len(original))
	}

	if (*decoded)[0].Letter != original[0].Letter {
		t.Errorf("Drive[0].Letter: %s != %s", (*decoded)[0].Letter, original[0].Letter)
	}

	if (*decoded)[0].SerialNumber != original[0].SerialNumber {
		t.Errorf("Drive[0].SerialNumber: %d != %d", (*decoded)[0].SerialNumber, original[0].SerialNumber)
	}
}

// ============ Тесты для decodeServices() ============

func TestDecodeServices(t *testing.T) {
	original := models.ServicesStatuses{
		{
			Name:        "RDP",
			ServiceName: "TermService",
			Installed:   true,
			Running:     true,
			Port:        3389,
			PortOpen:    true,
		},
	}

	encoder := NewEncoder()
	_ = encoder.encodeServices(original)
	data := encoder.buf.Bytes()

	decoder := NewDecoder(data)
	decoded := &models.ServicesStatuses{}
	err := decoder.decodeServices(decoded)

	if err != nil {
		t.Fatalf("decodeServices failed: %v", err)
	}

	if len(*decoded) != len(original) {
		t.Errorf("Services count: %d != %d", len(*decoded), len(original))
	}

	if (*decoded)[0].Port != original[0].Port {
		t.Errorf("Service[0].Port: %d != %d", (*decoded)[0].Port, original[0].Port)
	}
}

// ============ Тесты для decodeNetwork() ============

func TestDecodeNetwork(t *testing.T) {
	original := models.NetworkStatuses{
		{
			Index:        1,
			Name:         "Ethernet",
			Description:  "Intel Gigabit",
			MAC:          "a0:48:1c:a9:f6:c0",
			Type:         models.TypeEthernet,
			Operational:  true,
			IPAssignment: models.AssignmentStatic,
			IPAddresses:  []net.IP{net.ParseIP("172.17.113.11")},
		},
	}

	encoder := NewEncoder()
	_ = encoder.encodeNetwork(original)
	data := encoder.buf.Bytes()

	decoder := NewDecoder(data)
	decoded := &models.NetworkStatuses{}
	err := decoder.decodeNetwork(decoded)

	if err != nil {
		t.Fatalf("decodeNetwork failed: %v", err)
	}

	if len(*decoded) != len(original) {
		t.Errorf("Network count: %d != %d", len(*decoded), len(original))
	}

	if (*decoded)[0].MAC != original[0].MAC {
		t.Errorf("Network[0].MAC: %s != %s", (*decoded)[0].MAC, original[0].MAC)
	}

	if len((*decoded)[0].IPAddresses) != len(original[0].IPAddresses) {
		t.Errorf("IP count: %d != %d", len((*decoded)[0].IPAddresses), len(original[0].IPAddresses))
	}
}

// ============ Тесты для decodeHost() ============

func TestDecodeHost(t *testing.T) {
	original := &models.HostInfo{
		Hostname:       "DESKTOP-ABC123",
		FQDN:           "DESKTOP-ABC123.acme.corp.local",
		Domain:         "acme.corp.local",
		UpTimeSeconds:  9798016,
		BootTime:       1747109492,
		TimeZoneOffset: 180,
		Manufacturer:   "Dell Inc.",
		Model:          "Precision T3610",
	}

	encoder := NewEncoder()
	_ = encoder.encodeHost(original)
	data := encoder.buf.Bytes()

	decoder := NewDecoder(data)
	decoded := &models.HostInfo{}
	err := decoder.decodeHost(decoded)

	if err != nil {
		t.Fatalf("decodeHost failed: %v", err)
	}

	if decoded.Hostname != original.Hostname {
		t.Errorf("Hostname: %s != %s", decoded.Hostname, original.Hostname)
	}

	if decoded.BootTime != original.BootTime {
		t.Errorf("BootTime: %d != %d", decoded.BootTime, original.BootTime)
	}

	if decoded.TimeZoneOffset != original.TimeZoneOffset {
		t.Errorf("TimeZoneOffset: %d != %d", decoded.TimeZoneOffset, original.TimeZoneOffset)
	}
}

// ============ Вспомогательные функции ============

func assertSnapshotsEqual(t *testing.T, original, decoded *models.SystemSnapshot) {
	t.Helper()

	// Timestamp
	if original.Timestamp != decoded.Timestamp {
		t.Errorf("Timestamp: %d != %d", original.Timestamp, decoded.Timestamp)
	}

	// User
	if original.User.Username != decoded.User.Username {
		t.Errorf("User.Username: %s != %s", original.User.Username, decoded.User.Username)
	}
	if original.User.IsAdmin != decoded.User.IsAdmin {
		t.Errorf("User.IsAdmin: %v != %v", original.User.IsAdmin, decoded.User.IsAdmin)
	}

	// OS
	if original.OS.Name != decoded.OS.Name {
		t.Errorf("OS.Name: %s != %s", original.OS.Name, decoded.OS.Name)
	}
	if original.OS.Architecture != decoded.OS.Architecture {
		t.Errorf("OS.Architecture: %v != %v", original.OS.Architecture, decoded.OS.Architecture)
	}
	if original.OS.MachineGUID != decoded.OS.MachineGUID {
		t.Errorf("OS.MachineGUID: %s != %s", original.OS.MachineGUID, decoded.OS.MachineGUID)
	}

	// Processor
	if original.Processor.Model != decoded.Processor.Model {
		t.Errorf("Processor.Model: %s != %s", original.Processor.Model, decoded.Processor.Model)
	}
	if original.Processor.PhysicalCores != decoded.Processor.PhysicalCores {
		t.Errorf("Processor.PhysicalCores: %d != %d", original.Processor.PhysicalCores, decoded.Processor.PhysicalCores)
	}

	// RAM
	if original.RAM.TotalBytes != decoded.RAM.TotalBytes {
		t.Errorf("RAM.TotalBytes: %d != %d", original.RAM.TotalBytes, decoded.RAM.TotalBytes)
	}

	// Drives
	if len(original.Drives) != len(decoded.Drives) {
		t.Errorf("Drives count: %d != %d", len(original.Drives), len(decoded.Drives))
	}

	// Services
	if len(original.Services) != len(decoded.Services) {
		t.Errorf("Services count: %d != %d", len(original.Services), len(decoded.Services))
	}

	// Network
	if len(original.Network) != len(decoded.Network) {
		t.Errorf("Network count: %d != %d", len(original.Network), len(decoded.Network))
	}

	// Host
	if original.Host.Hostname != decoded.Host.Hostname {
		t.Errorf("Host.Hostname: %s != %s", original.Host.Hostname, decoded.Host.Hostname)
	}
}

func createTestSnapshot() *models.SystemSnapshot {
	return &models.SystemSnapshot{
		Timestamp: time.Now().Unix(),
		User: models.UserInfo{
			Username:     `ACME\john.doe`,
			FullName:     "John Doe",
			Domain:       "ACME",
			DomainFull:   "acme.corp.local",
			ProfilePath:  `C:\Users\john.doe`,
			IsAdmin:      true,
			IsDomainUser: true,
			IsLocalUser:  false,
		},
		OS: models.OSInfo{
			Name:          "Windows 10 22H2",
			Edition:       "Professional",
			BuildNumber:   "19045",
			KernelVersion: "10.0.19045",
			Architecture:  models.AMD64,
			Locale:        "en-US",
			InstallDate:   1730764800,
			MachineGUID:   models.BinaryUUID(uuid.New()),
			IsVirtual:     false,
			IsHypervisor:  true,
		},
		Processor: models.ProcessorInfo{
			Model:             "Intel Xeon E5-1620",
			VendorID:          "GenuineIntel",
			PhysicalCores:     4,
			LogicalProcessors: 8,
			BaseSpeedMHz:      3591,
			SMTEnabled:        true,
			L1CacheBytes:      262144,
			L2CacheBytes:      1048576,
			L3CacheBytes:      10485760,
		},
		RAM: models.RAMInfo{
			TotalBytes:     42869710848,
			AvailableBytes: 23083425792,
			Sticks: []models.RAMStick{
				{Slot: "DIMM1", Capacity: 8589934592, SpeedMHz: 1600},
			},
		},
		Drives: models.DiskStatuses{
			{
				Letter:       "C:",
				Type:         models.DriveFixed,
				FSType:       "NTFS",
				TotalBytes:   498951073792,
				FreeBytes:    254503497728,
				SerialNumber: 0xDEA5D590,
				IsReady:      true,
			},
		},
		Services: models.ServicesStatuses{
			{
				Name:        "RDP",
				ServiceName: "TermService",
				Installed:   true,
				Running:     true,
				Port:        3389,
				PortOpen:    true,
			},
		},
		Network: models.NetworkStatuses{
			{
				Index:        1,
				Name:         "Ethernet",
				Description:  "Intel Gigabit",
				MAC:          "a0:48:1c:a9:f6:c0",
				Type:         models.TypeEthernet,
				Operational:  true,
				IPAssignment: models.AssignmentStatic,
				IPAddresses:  []net.IP{net.ParseIP("172.17.113.11")},
			},
		},
		Host: models.HostInfo{
			Hostname:       "DESKTOP-ABC123",
			FQDN:           "DESKTOP-ABC123.acme.corp.local",
			Domain:         "acme.corp.local",
			UpTimeSeconds:  9798016,
			BootTime:       1747109492,
			TimeZoneOffset: 180,
			Manufacturer:   "Dell Inc.",
			Model:          "Precision T3610",
		},
	}
}

// ============ Бенчмарки ============

func BenchmarkDecode(b *testing.B) {
	snapshot := createTestSnapshot()
	encoder := NewEncoder()
	data, _ := encoder.Encode(snapshot)

	b.ResetTimer()
	for b.Loop() {
		decoder := NewDecoder(data)
		_, _ = decoder.Decode()
	}
}

func BenchmarkDecodeParallel(b *testing.B) {
	snapshot := createTestSnapshot()
	encoder := NewEncoder()
	data, _ := encoder.Encode(snapshot)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			decoder := NewDecoder(data)
			_, _ = decoder.Decode()
		}
	})
}

func BenchmarkRoundTrip(b *testing.B) {
	snapshot := createTestSnapshot()

	b.ResetTimer()
	for b.Loop() {
		encoder := NewEncoder()
		data, _ := encoder.Encode(snapshot)
		decoder := NewDecoder(data)
		_, _ = decoder.Decode()
	}
}
