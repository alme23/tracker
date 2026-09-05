//go:build windows

package collector

import (
	"os"
	"os/user"
	"strings"
	"testing"
)

func TestNewUserCollector(t *testing.T) {
	collector := NewUserCollector()

	if collector == nil {
		t.Fatal("NewUserCollector returned nil")
	}
}

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

	// Логируем информацию
	t.Logf("Username: %s", info.Username)
	t.Logf("Full Name: %s", info.FullName)
	t.Logf("Domain: %s", info.Domain)
	t.Logf("Domain Full: %s", info.DomainFull)
	t.Logf("Workgroup: %s", info.Workgroup)
	t.Logf("Profile: %s", info.ProfilePath)
	t.Logf("Is Admin: %v", info.IsAdmin)
	t.Logf("Is Domain User: %v", info.IsDomainUser)
	t.Logf("Is Local User: %v", info.IsLocalUser)
}

func TestGetUserName(t *testing.T) {
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

			// Не все форматы могут быть доступны
			if result == "" {
				t.Logf("%s returned empty (may be normal)", tt.name)
			}
		})
	}
}

func TestGetFullDomainName(t *testing.T) {
	collector := NewUserCollector()

	domainFull := collector.getFullDomainName()

	t.Logf("Full Domain: %s", domainFull)

	// Домен может быть пустым для локальных пользователей
	if domainFull == "" {
		t.Log("Full domain is empty (local user)")
	}
}

func TestGetDomainFromRegistry(t *testing.T) {
	collector := NewUserCollector()

	domain := collector.getDomainFromRegistry()

	t.Logf("Domain from registry: %s", domain)

	// Домен может быть пустым
	if domain == "" {
		t.Log("Domain is empty (workgroup)")
	}
}

func TestIsAdmin(t *testing.T) {
	collector := NewUserCollector()

	isAdmin := collector.isAdmin()

	t.Logf("Is Admin: %v", isAdmin)

	// Проверяем через user.Current() для сравнения
	if u, err := user.Current(); err == nil {
		t.Logf("Current user: %s", u.Username)
		t.Logf("UID: %s", u.Uid)
	}
}

func TestGetProfilePath(t *testing.T) {
	collector := NewUserCollector()

	profilePath := collector.getProfilePath()

	t.Logf("Profile path: %s", profilePath)

	if profilePath == "" {
		t.Error("Profile path is empty")
	}

	// Проверяем через переменные окружения
	userProfile := os.Getenv("USERPROFILE")
	if userProfile != "" && profilePath != userProfile {
		t.Errorf("Profile path (%s) != USERPROFILE (%s)", profilePath, userProfile)
	}
}

func TestDomainUserDetection(t *testing.T) {
	collector := NewUserCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Проверяем логику определения типа пользователя
	if info.IsDomainUser && info.IsLocalUser {
		t.Error("User cannot be both domain and local")
	}

	if !info.IsDomainUser && !info.IsLocalUser {
		t.Error("User must be either domain or local")
	}

	// Для доменного пользователя
	if info.IsDomainUser {
		if info.DomainFull == "" {
			t.Error("Domain user should have DomainFull")
		}
		if info.Domain == "" {
			t.Error("Domain user should have Domain")
		}
	}

	// Для локального пользователя
	if info.IsLocalUser {
		if info.Workgroup == "" {
			t.Log("Local user without workgroup (may be normal)")
		}
	}
}

func TestUsernameParsing(t *testing.T) {
	collector := NewUserCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Проверяем формат Username
	if strings.Contains(info.Username, "\\") {
		parts := strings.SplitN(info.Username, "\\", 2)
		if len(parts) != 2 {
			t.Errorf("Invalid username format: %s", info.Username)
		}

		// Domain должен совпадать с первой частью
		if info.Domain != parts[0] {
			t.Errorf("Domain (%s) != username prefix (%s)", info.Domain, parts[0])
		}

		t.Logf("Domain: %s, Username: %s", parts[0], parts[1])
	} else {
		// Локальный пользователь без домена
		t.Logf("Username without domain: %s", info.Username)
	}
}

func TestUserInfoConsistency(t *testing.T) {
	collector := NewUserCollector()

	info, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Проверяем консистентность
	if info.IsDomainUser && info.Workgroup != "" {
		t.Logf("Domain user with workgroup: %s (unusual)", info.Workgroup)
	}

	if info.IsLocalUser && info.DomainFull != "" {
		t.Errorf("Local user with DomainFull: %s", info.DomainFull)
	}

	// FullName не должен быть пустым
	if info.FullName == "" {
		t.Error("FullName is empty")
	}
}

func TestUserCollectorRepeatedCalls(t *testing.T) {
	collector := NewUserCollector()

	// Первый вызов
	first, err := collector.Collect()
	if err != nil {
		t.Fatalf("First Collect failed: %v", err)
	}

	// Второй вызов
	second, err := collector.Collect()
	if err != nil {
		t.Fatalf("Second Collect failed: %v", err)
	}

	// Данные не должны меняться
	if first.Username != second.Username {
		t.Errorf("Username changed: %s vs %s", first.Username, second.Username)
	}

	if first.Domain != second.Domain {
		t.Errorf("Domain changed: %s vs %s", first.Domain, second.Domain)
	}

	if first.IsAdmin != second.IsAdmin {
		t.Errorf("IsAdmin changed: %v vs %v", first.IsAdmin, second.IsAdmin)
	}
}

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

func BenchmarkUserCollector(b *testing.B) {
	collector := NewUserCollector()

	b.ResetTimer()
	for b.Loop() {
		_, err := collector.Collect()
		if err != nil {
			b.Fatalf("Collect failed: %v", err)
		}
	}
}

func BenchmarkUserCollectorParallel(b *testing.B) {
	collector := NewUserCollector()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := collector.Collect()
			if err != nil {
				b.Fatalf("Collect failed: %v", err)
			}
		}
	})
}
