// tracker/internal/models/network_test.go
package models

import (
	"encoding/json"
	"net"
	"testing"
)

// ============ Тесты для IPAssignment.String() ============

func TestIPAssignmentString(t *testing.T) {
	tests := []struct {
		name       string
		assignment IPAssignment
		expected   string
	}{
		{"Unknown", AssignmentUnknown, "UNKNOWN"},
		{"DHCP", AssignmentDHCP, "DHCP"},
		{"Static", AssignmentStatic, "STATIC"},
		{"NotApplicable", AssignmentNotApps, "NOT_APPLICABLE"},
		{"Invalid", IPAssignment(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.assignment.String()
			if result != tt.expected {
				t.Errorf("IPAssignment(%d).String() = %s, want %s",
					tt.assignment, result, tt.expected)
			}
		})
	}
}

// ============ Тесты для IPAssignment.MarshalJSON() ============

func TestIPAssignmentMarshalJSON(t *testing.T) {
	tests := []struct {
		name       string
		assignment IPAssignment
		expected   string
	}{
		{"Unknown", AssignmentUnknown, `"UNKNOWN"`},
		{"DHCP", AssignmentDHCP, `"DHCP"`},
		{"Static", AssignmentStatic, `"STATIC"`},
		{"NotApplicable", AssignmentNotApps, `"NOT_APPLICABLE"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.assignment.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON failed: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("MarshalJSON() = %s, want %s", data, tt.expected)
			}
		})
	}
}

// ============ Тесты для IPAssignment.UnmarshalJSON() ============

func TestIPAssignmentUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected IPAssignment
	}{
		{"Unknown", `"UNKNOWN"`, AssignmentUnknown},
		{"DHCP", `"DHCP"`, AssignmentDHCP},
		{"Static", `"STATIC"`, AssignmentStatic},
		{"NotApplicable", `"NOT_APPLICABLE"`, AssignmentNotApps},
		{"Invalid", `"INVALID"`, AssignmentUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var assignment IPAssignment
			err := assignment.UnmarshalJSON([]byte(tt.json))
			if err != nil {
				t.Fatalf("UnmarshalJSON failed: %v", err)
			}

			if assignment != tt.expected {
				t.Errorf("UnmarshalJSON(%s) = %v, want %v",
					tt.json, assignment, tt.expected)
			}
		})
	}
}

// ============ Тесты для InterfaceType.String() ============

func TestInterfaceTypeString(t *testing.T) {
	tests := []struct {
		name          string
		interfaceType InterfaceType
		expected      string
	}{
		{"Unknown", TypeUnknown, "UNKNOWN"},
		{"Ethernet", TypeEthernet, "ETHERNET"},
		{"Wireless", TypeWireless, "WIRELESS"},
		{"Loopback", TypeLoopback, "LOOPBACK"},
		{"Tunnel", TypeTunnel, "TUNNEL"},
		{"PPP", TypePPP, "POINT_TO_POINT"},
		{"Other", TypeOther, "OTHER"},
		{"Invalid", InterfaceType(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.interfaceType.String()
			if result != tt.expected {
				t.Errorf("InterfaceType(%d).String() = %s, want %s",
					tt.interfaceType, result, tt.expected)
			}
		})
	}
}

// ============ Тесты для InterfaceType.MarshalJSON() ============

func TestInterfaceTypeMarshalJSON(t *testing.T) {
	tests := []struct {
		name          string
		interfaceType InterfaceType
		expected      string
	}{
		{"Unknown", TypeUnknown, `"UNKNOWN"`},
		{"Ethernet", TypeEthernet, `"ETHERNET"`},
		{"Wireless", TypeWireless, `"WIRELESS"`},
		{"Loopback", TypeLoopback, `"LOOPBACK"`},
		{"Tunnel", TypeTunnel, `"TUNNEL"`},
		{"PPP", TypePPP, `"POINT_TO_POINT"`},
		{"Other", TypeOther, `"OTHER"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.interfaceType.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON failed: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("MarshalJSON() = %s, want %s", data, tt.expected)
			}
		})
	}
}

// ============ Тесты для InterfaceType.UnmarshalJSON() ============

func TestInterfaceTypeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected InterfaceType
	}{
		{"Unknown", `"UNKNOWN"`, TypeUnknown},
		{"Ethernet", `"ETHERNET"`, TypeEthernet},
		{"Wireless", `"WIRELESS"`, TypeWireless},
		{"Loopback", `"LOOPBACK"`, TypeLoopback},
		{"Tunnel", `"TUNNEL"`, TypeTunnel},
		{"PPP", `"POINT_TO_POINT"`, TypePPP},
		{"Other", `"OTHER"`, TypeOther},
		{"Invalid", `"INVALID"`, TypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var interfaceType InterfaceType
			err := interfaceType.UnmarshalJSON([]byte(tt.json))
			if err != nil {
				t.Fatalf("UnmarshalJSON failed: %v", err)
			}

			if interfaceType != tt.expected {
				t.Errorf("UnmarshalJSON(%s) = %v, want %v",
					tt.json, interfaceType, tt.expected)
			}
		})
	}
}

// ============ Тесты для InterfaceInfo JSON ============

func TestInterfaceInfoJSON(t *testing.T) {
	iface := InterfaceInfo{
		Index:        1,
		Name:         "Ethernet",
		Description:  "Intel(R) 82579LM Gigabit Network Connection",
		MAC:          "a0:48:1c:a9:f6:c0",
		Type:         TypeEthernet,
		Operational:  true,
		IPAssignment: AssignmentStatic,
		IPAddresses:  []net.IP{net.ParseIP("172.17.113.11")},
	}

	data, err := json.Marshal(iface)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	t.Logf("JSON: %s", data)

	var restored InterfaceInfo
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if restored.Index != iface.Index {
		t.Errorf("Index: %d != %d", restored.Index, iface.Index)
	}

	if restored.Name != iface.Name {
		t.Errorf("Name: %s != %s", restored.Name, iface.Name)
	}

	if restored.Type != iface.Type {
		t.Errorf("Type: %v != %v", restored.Type, iface.Type)
	}

	if restored.IPAssignment != iface.IPAssignment {
		t.Errorf("IPAssignment: %v != %v", restored.IPAssignment, iface.IPAssignment)
	}
}

// ============ Тесты для JSON round-trip ============

func TestIPAssignmentJSONRoundTrip(t *testing.T) {
	assignments := []IPAssignment{
		AssignmentUnknown,
		AssignmentDHCP,
		AssignmentStatic,
		AssignmentNotApps,
	}

	for _, original := range assignments {
		data, err := original.MarshalJSON()
		if err != nil {
			t.Fatalf("MarshalJSON failed: %v", err)
		}

		var restored IPAssignment
		err = restored.UnmarshalJSON(data)
		if err != nil {
			t.Fatalf("UnmarshalJSON failed: %v", err)
		}

		if restored != original {
			t.Errorf("Round trip failed: %v -> %v", original, restored)
		}
	}
}

func TestInterfaceTypeJSONRoundTrip(t *testing.T) {
	types := []InterfaceType{
		TypeUnknown,
		TypeEthernet,
		TypeWireless,
		TypeLoopback,
		TypeTunnel,
		TypePPP,
		TypeOther,
	}

	for _, original := range types {
		data, err := original.MarshalJSON()
		if err != nil {
			t.Fatalf("MarshalJSON failed: %v", err)
		}

		var restored InterfaceType
		err = restored.UnmarshalJSON(data)
		if err != nil {
			t.Fatalf("UnmarshalJSON failed: %v", err)
		}

		if restored != original {
			t.Errorf("Round trip failed: %v -> %v", original, restored)
		}
	}
}

// ============ Тесты для NetworkStatuses ============

func TestNetworkStatusesType(t *testing.T) {
	var statuses NetworkStatuses

	if statuses != nil {
		t.Error("Empty NetworkStatuses should be nil")
	}

	statuses = append(statuses, InterfaceInfo{Index: 1, Name: "Ethernet"})

	if len(statuses) != 1 {
		t.Errorf("Len = %d, want 1", len(statuses))
	}

	if statuses[0].Name != "Ethernet" {
		t.Errorf("Name = %s, want Ethernet", statuses[0].Name)
	}
}

// ============ Бенчмарки ============

func BenchmarkIPAssignmentString(b *testing.B) {
	for b.Loop() {
		_ = AssignmentDHCP.String()
	}
}

func BenchmarkIPAssignmentMarshalJSON(b *testing.B) {
	for b.Loop() {
		_, _ = AssignmentDHCP.MarshalJSON()
	}
}

func BenchmarkInterfaceTypeString(b *testing.B) {
	for b.Loop() {
		_ = TypeEthernet.String()
	}
}

func BenchmarkInterfaceTypeMarshalJSON(b *testing.B) {
	for b.Loop() {
		_, _ = TypeEthernet.MarshalJSON()
	}
}

func BenchmarkInterfaceInfoJSON(b *testing.B) {
	iface := InterfaceInfo{
		Index:        1,
		Name:         "Ethernet",
		Type:         TypeEthernet,
		IPAssignment: AssignmentStatic,
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = json.Marshal(iface)
	}
}
