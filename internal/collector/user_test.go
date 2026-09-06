//go:build windows

package collector

import (
	"os"
	"os/user"
	"strings"
	"testing"
)

// ============ Тесты для NewUserCollector() ============

func TestNewUserCollector(t *testing.T) {
	collector := NewUserCollector()

	if collector == nil {
		t.Fatal("NewUserCollector returned nil")
	}
}

// ============ Тесты для Collect() ============

func TestUserCollectorCollect(t *testing.T) {
	collector := NewUserCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Проверяем обязательные поля
	if info.Username == "" {
		t.Error("Username is empty")
	}

	if info.FullName == "" {
		t.Error("FullName is empty")
	}

	if info.ProfilePath == "" {
		t.Error("ProfilePath is empty")
	}

	// Логируем
	t.Logf("Username: %s", info.Username)
	t.Logf("FullName: %s", info.FullName)
	t.Logf("Domain: %s", info.Domain)
	t.Logf("DomainFull: %s", info.DomainFull)
	t.Logf("Workgroup: %s", info.Workgroup)
	t.Logf("ProfilePath: %s", info.ProfilePath)
	t.Logf("IsAdmin: %v", info.IsAdmin)
	t.Logf("IsDomainUser: %v", info.IsDomainUser)
	t.Logf("IsLocalUser: %v", info.IsLocalUser)
}

// ============ Тесты для getFullDomainName() ============

func TestUserGetFullDomainName(t *testing.T) {
	collector := NewUserCollector()

	domainFull := collector.getFullDomainName()

	t.Logf("Full Domain: %s", domainFull)

	// Домен может быть пустым для локальных пользователей
	if domainFull == "" {
		t.Log("Full domain is empty (local user)")
	}
}

// ============ Тесты для getDomainFromRegistry() ============

func TestUserGetDomainFromRegistry(t *testing.T) {
	collector := NewUserCollector()

	domain := collector.getDomainFromRegistry()

	t.Logf("Domain from registry: %s", domain)

	if domain == "" {
		t.Log("Domain is empty (workgroup)")
	}
}

// ============ Тесты для getWorkgroup() ============

func TestUserGetWorkgroup(t *testing.T) {
	collector := NewUserCollector()

	workgroup := collector.getWorkgroup()

	t.Logf("Workgroup: %s", workgroup)

	// Рабочая группа не должна быть пустой (есть fallback "WORKGROUP")
	if workgroup == "" {
		t.Error("Workgroup is empty (should have fallback)")
	}
}

// ============ Тесты для getUserName() ============

func TestUserGetUserName(t *testing.T) {
	collector := NewUserCollector()

	tests := []struct {
		name       string
		nameFormat uint32
	}{
		{"SamCompatible", nameSamCompatible},
		{"Display", nameDisplay},
		{"UserPrincipal", nameUserPrincipal},
		{"DnsDomain", nameDnsDomain},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.getUserName(tt.nameFormat)
			t.Logf("%s: %s", tt.name, result)

			if result == "" {
				t.Logf("%s returned empty (may be normal)", tt.name)
			}
		})
	}
}

func TestUserGetUserNameSamCompatible(t *testing.T) {
	collector := NewUserCollector()

	username := collector.getUserName(nameSamCompatible)

	if username == "" {
		t.Error("SamCompatible is empty")
	}

	t.Logf("SamCompatible: %s", username)

	// Проверяем формат DOMAIN\Username
	if strings.Contains(username, "\\") {
		parts := strings.SplitN(username, "\\", 2)
		if len(parts) != 2 {
			t.Errorf("Invalid SamCompatible format: %s", username)
		}
		t.Logf("Domain: %s, User: %s", parts[0], parts[1])
	}
}

// ============ Тесты для isAdmin() ============

func TestUserIsAdmin(t *testing.T) {
	collector := NewUserCollector()

	isAdmin := collector.isAdmin()

	t.Logf("Is Admin: %v", isAdmin)

	// Сравниваем с user.Current()
	if u, err := user.Current(); err == nil {
		t.Logf("Current user: %s", u.Username)
	}
}

// ============ Тесты для getProfilePath() ============

func TestUserGetProfilePath(t *testing.T) {
	collector := NewUserCollector()

	profilePath := collector.getProfilePath()

	t.Logf("Profile path: %s", profilePath)

	if profilePath == "" {
		t.Error("Profile path is empty")
	}

	// Сравниваем с переменной окружения
	userProfile := os.Getenv("USERPROFILE")
	if userProfile != "" && profilePath != userProfile {
		t.Errorf("Profile path (%s) != USERPROFILE (%s)", profilePath, userProfile)
	}
}

// ============ Тесты на консистентность ============

func TestUserCollectorConsistency(t *testing.T) {
	collector := NewUserCollector()

	first, err := collector.Collect()
	if err != nil {
		t.Fatalf("First Collect failed: %v", err)
	}

	second, err := collector.Collect()
	if err != nil {
		t.Fatalf("Second Collect failed: %v", err)
	}

	if first.Username != second.Username {
		t.Errorf("Username changed: %s vs %s", first.Username, second.Username)
	}

	if first.IsAdmin != second.IsAdmin {
		t.Errorf("IsAdmin changed: %v vs %v", first.IsAdmin, second.IsAdmin)
	}

	if first.IsDomainUser != second.IsDomainUser {
		t.Errorf("IsDomainUser changed: %v vs %v", first.IsDomainUser, second.IsDomainUser)
	}
}

// ============ Тесты на доменного/локального пользователя ============

func TestUserTypeDetection(t *testing.T) {
	collector := NewUserCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Пользователь не может быть одновременно доменным и локальным
	if info.IsDomainUser && info.IsLocalUser {
		t.Error("User cannot be both domain and local")
	}

	// Пользователь должен быть либо доменным, либо локальным
	if !info.IsDomainUser && !info.IsLocalUser {
		t.Error("User must be either domain or local")
	}

	if info.IsDomainUser {
		if info.DomainFull == "" {
			t.Error("Domain user should have DomainFull")
		}
		if info.Workgroup != "" {
			t.Error("Domain user should not have Workgroup")
		}
	}

	if info.IsLocalUser {
		if info.DomainFull != "" {
			t.Error("Local user should not have DomainFull")
		}
	}
}

// ============ Тесты на конкурентность ============

func TestUserCollectorConcurrent(t *testing.T) {
	collector := NewUserCollector()

	const numGoroutines = 10
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := collector.Collect()
			errChan <- err
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("Concurrent Collect failed: %v", err)
		}
	}
}

// ============ Бенчмарки ============

func BenchmarkUserCollectorCollect(b *testing.B) {
	collector := NewUserCollector()

	b.ResetTimer()
	for b.Loop() {
		_, _ = collector.Collect()
	}
}

func BenchmarkUserGetUserName(b *testing.B) {
	collector := NewUserCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.getUserName(nameSamCompatible)
	}
}

func BenchmarkUserGetFullDomainName(b *testing.B) {
	collector := NewUserCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.getFullDomainName()
	}
}

func BenchmarkUserIsAdmin(b *testing.B) {
	collector := NewUserCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.isAdmin()
	}
}

func BenchmarkUserGetProfilePath(b *testing.B) {
	collector := NewUserCollector()

	b.ResetTimer()
	for b.Loop() {
		_ = collector.getProfilePath()
	}
}
