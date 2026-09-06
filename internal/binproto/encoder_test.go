// tracker/internal/binproto/encoder_test.go
package binproto

import (
	"bytes"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/alme23/tracker/internal/models"
	"github.com/google/uuid"
)

// ============ Тесты для NewEncoder() ============

func TestNewEncoder(t *testing.T) {
	encoder := NewEncoder()

	if encoder == nil {
		t.Fatal("NewEncoder returned nil")
	}

	if encoder.buf == nil {
		t.Error("buf is nil")
	}
}

// ============ Тесты для Encode() ============

func TestEncode(t *testing.T) {
	snapshot := createTestSnapshot()

	encoder := NewEncoder()
	data, err := encoder.Encode(snapshot)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Encoded data is empty")
	}

	// Проверяем magic header
	if len(data) < len(MagicHeader) {
		t.Fatal("Data too short")
	}

	if string(data[:len(MagicHeader)]) != MagicHeader {
		t.Errorf("Magic header = %q, want %q", data[:len(MagicHeader)], MagicHeader)
	}
}

// ============ Тесты на размер данных ============

func TestEncodeSize(t *testing.T) {
	snapshot := createTestSnapshot()

	encoder := NewEncoder()
	binaryData, err := encoder.Encode(snapshot)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	jsonData, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	t.Logf("Binary size: %d bytes", len(binaryData))
	t.Logf("JSON size: %d bytes", len(jsonData))

	compressionRatio := float64(len(binaryData)) / float64(len(jsonData)) * 100
	t.Logf("Compression ratio: %.1f%%", compressionRatio)

	// Бинарный формат должен быть меньше JSON
	if len(binaryData) >= len(jsonData) {
		t.Errorf("Binary (%d) should be smaller than JSON (%d)", len(binaryData), len(jsonData))
	}
}

// ============ Тесты на повторное использование ============

func TestEncodeReuse(t *testing.T) {
	encoder := NewEncoder()

	// Первый вызов
	snapshot1 := createTestSnapshot()
	data1, err := encoder.Encode(snapshot1)
	if err != nil {
		t.Fatalf("First Encode failed: %v", err)
	}

	// Второй вызов
	snapshot2 := createTestSnapshot()
	data2, err := encoder.Encode(snapshot2)
	if err != nil {
		t.Fatalf("Second Encode failed: %v", err)
	}

	// Результаты должны быть одинаковыми
	if !bytes.Equal(data1, data2) {
		t.Error("Encoder should produce identical results for identical input")
	}
}

// ============ Тесты на пустой snapshot ============

func TestEncodeEmptySnapshot(t *testing.T) {
	snapshot := &models.SystemSnapshot{
		Timestamp: time.Now().Unix(),
	}

	encoder := NewEncoder()
	data, err := encoder.Encode(snapshot)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Encoded data is empty")
	}

	t.Logf("Empty snapshot size: %d bytes", len(data))
}

// ============ Тесты writeString ============

func TestWriteString(t *testing.T) {
	encoder := NewEncoder()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Empty", "", false},
		{"Short", "hello", false},
		{"Long", string(make([]byte, 1000)), false},
		{"Too long", string(make([]byte, 70000)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder.buf.Reset()

			err := encoder.writeString(tt.input)

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// ============ Тесты writeFlags ============

func TestWriteFlags(t *testing.T) {
	encoder := NewEncoder()

	tests := []struct {
		name     string
		flags    []bool
		expected byte
	}{
		{"All false", []bool{false, false, false}, 0x00},
		{"First true", []bool{true, false, false}, 0x01},
		{"Second true", []bool{false, true, false}, 0x02},
		{"Third true", []bool{false, false, true}, 0x04},
		{"All true", []bool{true, true, true}, 0x07},
		{"Four flags", []bool{true, false, true, false}, 0x05},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder.buf.Reset()

			err := encoder.writeFlags(tt.flags...)
			if err != nil {
				t.Fatalf("writeFlags failed: %v", err)
			}

			result := encoder.buf.Bytes()
			if len(result) != 1 {
				t.Fatalf("Expected 1 byte, got %d", len(result))
			}

			if result[0] != tt.expected {
				t.Errorf("Flags = 0x%02X, want 0x%02X", result[0], tt.expected)
			}
		})
	}
}

// ============ Тесты для каждой под-структуры ============

func TestEncodeUser(t *testing.T) {
	encoder := NewEncoder()
	user := &models.UserInfo{
		Username:    `ACME\john.doe`,
		FullName:    "John Doe",
		Domain:      "ACME",
		DomainFull:  "acme.corp.local",
		ProfilePath: `C:\Users\john.doe`,
		IsAdmin:     true,
	}

	encoder.buf.Reset()
	err := encoder.encodeUser(user)
	if err != nil {
		t.Fatalf("encodeUser failed: %v", err)
	}

	if encoder.buf.Len() == 0 {
		t.Error("Encoded user is empty")
	}
}

func TestEncodeOS(t *testing.T) {
	encoder := NewEncoder()
	osInfo := &models.OSInfo{
		Name:         "Windows 10",
		Edition:      "Professional",
		BuildNumber:  "19045",
		Architecture: models.AMD64,
		MachineGUID:  models.BinaryUUID(uuid.New()),
	}

	encoder.buf.Reset()
	err := encoder.encodeOS(osInfo)
	if err != nil {
		t.Fatalf("encodeOS failed: %v", err)
	}

	if encoder.buf.Len() == 0 {
		t.Error("Encoded OS is empty")
	}
}

func TestEncodeProcessor(t *testing.T) {
	encoder := NewEncoder()
	processor := &models.ProcessorInfo{
		Model:         "Intel Xeon",
		VendorID:      "GenuineIntel",
		PhysicalCores: 4,
		L1CacheBytes:  262144,
	}

	encoder.buf.Reset()
	err := encoder.encodeProcessor(processor)
	if err != nil {
		t.Fatalf("encodeProcessor failed: %v", err)
	}

	if encoder.buf.Len() == 0 {
		t.Error("Encoded processor is empty")
	}
}

func TestEncodeRAM(t *testing.T) {
	encoder := NewEncoder()
	ram := &models.RAMInfo{
		TotalBytes: 42869710848,
		Sticks: []models.RAMStick{
			{Slot: "DIMM1", Capacity: 8589934592},
		},
	}

	encoder.buf.Reset()
	err := encoder.encodeRAM(ram)
	if err != nil {
		t.Fatalf("encodeRAM failed: %v", err)
	}

	if encoder.buf.Len() == 0 {
		t.Error("Encoded RAM is empty")
	}
}

func TestEncodeDrives(t *testing.T) {
	encoder := NewEncoder()
	drives := models.DiskStatuses{
		{Letter: "C:", Type: models.DriveFixed, FSType: "NTFS"},
		{Letter: "D:", Type: models.DriveCDROM},
	}

	encoder.buf.Reset()
	err := encoder.encodeDrives(drives)
	if err != nil {
		t.Fatalf("encodeDrives failed: %v", err)
	}

	if encoder.buf.Len() == 0 {
		t.Error("Encoded drives is empty")
	}
}

func TestEncodeServices(t *testing.T) {
	encoder := NewEncoder()
	services := models.ServicesStatuses{
		{Name: "RDP", Port: 3389, Installed: true, Running: true},
		{Name: "VNC", Port: 5900},
	}

	encoder.buf.Reset()
	err := encoder.encodeServices(services)
	if err != nil {
		t.Fatalf("encodeServices failed: %v", err)
	}

	if encoder.buf.Len() == 0 {
		t.Error("Encoded services is empty")
	}
}

func TestEncodeNetwork(t *testing.T) {
	encoder := NewEncoder()
	network := models.NetworkStatuses{
		{
			Index:        1,
			Name:         "Ethernet",
			Type:         models.TypeEthernet,
			IPAssignment: models.AssignmentStatic,
			IPAddresses:  []net.IP{net.ParseIP("192.168.1.1")},
		},
	}

	encoder.buf.Reset()
	err := encoder.encodeNetwork(network)
	if err != nil {
		t.Fatalf("encodeNetwork failed: %v", err)
	}

	if encoder.buf.Len() == 0 {
		t.Error("Encoded network is empty")
	}
}

func TestEncodeHost(t *testing.T) {
	encoder := NewEncoder()
	host := &models.HostInfo{
		Hostname: "DESKTOP-ABC123",
		FQDN:     "DESKTOP-ABC123.acme.corp.local",
		Domain:   "acme.corp.local",
		BootTime: 1747109492,
	}

	encoder.buf.Reset()
	err := encoder.encodeHost(host)
	if err != nil {
		t.Fatalf("encodeHost failed: %v", err)
	}

	if encoder.buf.Len() == 0 {
		t.Error("Encoded host is empty")
	}
}

// ============ Бенчмарки ============

func BenchmarkEncode(b *testing.B) {
	snapshot := createTestSnapshot()
	encoder := NewEncoder()

	b.ResetTimer()
	for b.Loop() {
		_, _ = encoder.Encode(snapshot)
	}
}

func BenchmarkEncodeParallel(b *testing.B) {
	snapshot := createTestSnapshot()

	b.RunParallel(func(pb *testing.PB) {
		encoder := NewEncoder()
		for pb.Next() {
			_, _ = encoder.Encode(snapshot)
		}
	})
}

func BenchmarkWriteString(b *testing.B) {
	encoder := NewEncoder()

	b.ResetTimer()
	for b.Loop() {
		encoder.buf.Reset()
		_ = encoder.writeString("test string")
	}
}
