// tracker/internal/models/service_test.go
package models

import (
	"encoding/json"
	"testing"
)

// ============ Тесты для ServiceStatus полей ============

func TestServiceStatusFields(t *testing.T) {
	service := ServiceStatus{
		Name:        "RDP",
		ServiceName: "TermService",
		Installed:   true,
		Running:     true,
		Port:        3389,
		PortOpen:    true,
	}

	if service.Name == "" {
		t.Error("Name is empty")
	}

	if service.ServiceName == "" {
		t.Error("ServiceName is empty")
	}

	if service.Port == 0 {
		t.Error("Port is 0")
	}
}

// ============ Тесты для ServiceStatus JSON ============

func TestServiceStatusJSON(t *testing.T) {
	service := ServiceStatus{
		Name:        "RDP",
		ServiceName: "TermService",
		Installed:   true,
		Running:     true,
		Port:        3389,
		PortOpen:    true,
	}

	data, err := json.Marshal(service)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	t.Logf("JSON: %s", data)

	var restored ServiceStatus
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if restored.Name != service.Name {
		t.Errorf("Name: %s != %s", restored.Name, service.Name)
	}

	if restored.ServiceName != service.ServiceName {
		t.Errorf("ServiceName: %s != %s", restored.ServiceName, service.ServiceName)
	}

	if restored.Installed != service.Installed {
		t.Errorf("Installed: %v != %v", restored.Installed, service.Installed)
	}

	if restored.Running != service.Running {
		t.Errorf("Running: %v != %v", restored.Running, service.Running)
	}

	if restored.Port != service.Port {
		t.Errorf("Port: %d != %d", restored.Port, service.Port)
	}

	if restored.PortOpen != service.PortOpen {
		t.Errorf("PortOpen: %v != %v", restored.PortOpen, service.PortOpen)
	}
}

// ============ Тесты для ServicesStatuses ============

func TestServicesStatusesType(t *testing.T) {
	var statuses ServicesStatuses

	if statuses != nil {
		t.Error("Empty ServicesStatuses should be nil")
	}

	statuses = append(statuses, ServiceStatus{Name: "RDP"})

	if len(statuses) != 1 {
		t.Errorf("Len = %d, want 1", len(statuses))
	}

	if statuses[0].Name != "RDP" {
		t.Errorf("Name = %s, want RDP", statuses[0].Name)
	}
}

// ============ Тесты для ServicesStatuses JSON ============

func TestServicesStatusesJSON(t *testing.T) {
	statuses := ServicesStatuses{
		{
			Name:        "RDP",
			ServiceName: "TermService",
			Installed:   true,
			Running:     true,
			Port:        3389,
			PortOpen:    true,
		},
		{
			Name:        "VNC",
			ServiceName: "unknown",
			Installed:   false,
			Running:     false,
			Port:        5900,
			PortOpen:    false,
		},
	}

	data, err := json.Marshal(statuses)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	t.Logf("JSON: %s", data)

	var restored ServicesStatuses
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if len(restored) != len(statuses) {
		t.Errorf("Len: %d != %d", len(restored), len(statuses))
	}

	for i := range restored {
		if restored[i].Name != statuses[i].Name {
			t.Errorf("Name[%d]: %s != %s", i, restored[i].Name, statuses[i].Name)
		}
	}
}

// ============ Тесты на валидацию портов ============

func TestServiceStatusPortValidation(t *testing.T) {
	tests := []struct {
		name     string
		port     uint16
		expected bool
	}{
		{"Valid port 1", 1, true},
		{"Valid port 80", 80, true},
		{"Valid port 3389", 3389, true},
		{"Valid port 5900", 5900, true},
		{"Valid port 65535", 65535, true},
		{"Invalid port 0", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ServiceStatus{Port: tt.port}

			isValid := service.Port > 0

			if isValid != tt.expected {
				t.Errorf("Port %d: valid = %v, want %v", tt.port, isValid, tt.expected)
			}
		})
	}
}

// ============ Тесты на логику состояния ============

func TestServiceStatusLogic(t *testing.T) {
	tests := []struct {
		name     string
		service  ServiceStatus
		portOpen bool
	}{
		{
			name: "Running with open port",
			service: ServiceStatus{
				Running:  true,
				PortOpen: true,
			},
			portOpen: true,
		},
		{
			name: "Running with closed port",
			service: ServiceStatus{
				Running:  true,
				PortOpen: false,
			},
			portOpen: false,
		},
		{
			name: "Not running",
			service: ServiceStatus{
				Running:  false,
				PortOpen: false,
			},
			portOpen: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Порт может быть открыт только если служба запущена
			if tt.service.PortOpen && !tt.service.Running {
				t.Error("Port cannot be open if service is not running")
			}

			if tt.service.PortOpen != tt.portOpen {
				t.Errorf("PortOpen = %v, want %v", tt.service.PortOpen, tt.portOpen)
			}
		})
	}
}

// ============ Бенчмарки ============

func BenchmarkServiceStatusJSON(b *testing.B) {
	service := ServiceStatus{
		Name:        "RDP",
		ServiceName: "TermService",
		Installed:   true,
		Running:     true,
		Port:        3389,
		PortOpen:    true,
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = json.Marshal(service)
	}
}

func BenchmarkServicesStatusesJSON(b *testing.B) {
	statuses := ServicesStatuses{
		{Name: "RDP", ServiceName: "TermService", Port: 3389},
		{Name: "VNC", ServiceName: "unknown", Port: 5900},
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = json.Marshal(statuses)
	}
}
