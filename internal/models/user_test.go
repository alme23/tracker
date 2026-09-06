// tracker/internal/models/user_test.go
package models

import (
	"encoding/json"
	"strings"
	"testing"
)

// ============ Тесты для UserInfo полей ============

func TestUserInfoFields(t *testing.T) {
	user := UserInfo{
		Username:     `DOMAIN\User`,
		FullName:     "User User",
		Domain:       "DOMAIN",
		DomainFull:   "domain.tld",
		ProfilePath:  `C:\Users\User`,
		IsAdmin:      true,
		IsDomainUser: true,
		IsLocalUser:  false,
	}

	if user.Username == "" {
		t.Error("Username is empty")
	}

	if user.FullName == "" {
		t.Error("FullName is empty")
	}

	if user.ProfilePath == "" {
		t.Error("ProfilePath is empty")
	}
}

// ============ Тесты для UserInfo JSON ============

func TestUserInfoJSON(t *testing.T) {
	user := UserInfo{
		Username:     `DOMAIN\User`,
		FullName:     "User User",
		Domain:       "DOMAIN",
		DomainFull:   "domain.tld",
		ProfilePath:  `C:\Users\User`,
		IsAdmin:      true,
		IsDomainUser: true,
		IsLocalUser:  false,
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	t.Logf("JSON: %s", data)

	var restored UserInfo
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if restored.Username != user.Username {
		t.Errorf("Username: %s != %s", restored.Username, user.Username)
	}

	if restored.FullName != user.FullName {
		t.Errorf("FullName: %s != %s", restored.FullName, user.FullName)
	}

	if restored.Domain != user.Domain {
		t.Errorf("Domain: %s != %s", restored.Domain, user.Domain)
	}

	if restored.DomainFull != user.DomainFull {
		t.Errorf("DomainFull: %s != %s", restored.DomainFull, user.DomainFull)
	}

	if restored.IsAdmin != user.IsAdmin {
		t.Errorf("IsAdmin: %v != %v", restored.IsAdmin, user.IsAdmin)
	}

	if restored.IsDomainUser != user.IsDomainUser {
		t.Errorf("IsDomainUser: %v != %v", restored.IsDomainUser, user.IsDomainUser)
	}
}

// ============ Тесты для доменного пользователя ============

func TestUserInfoDomainUser(t *testing.T) {
	user := UserInfo{
		Username:     `DOMAIN\User`,
		Domain:       "DOMAIN",
		DomainFull:   "domain.tld",
		IsDomainUser: true,
		IsLocalUser:  false,
	}

	if user.Domain == "" {
		t.Error("Domain is empty for domain user")
	}

	if user.DomainFull == "" {
		t.Error("DomainFull is empty for domain user")
	}

	if user.Workgroup != "" {
		t.Error("Workgroup should be empty for domain user")
	}

	if !user.IsDomainUser {
		t.Error("IsDomainUser should be true")
	}

	if user.IsLocalUser {
		t.Error("IsLocalUser should be false")
	}
}

// ============ Тесты для локального пользователя ============

func TestUserInfoLocalUser(t *testing.T) {
	user := UserInfo{
		Username:     `COMPUTER\Admin`,
		FullName:     "Администратор",
		Domain:       "",
		DomainFull:   "",
		Workgroup:    "WORKGROUP",
		IsDomainUser: false,
		IsLocalUser:  true,
	}

	if user.Domain != "" {
		t.Error("Domain should be empty for local user")
	}

	if user.DomainFull != "" {
		t.Error("DomainFull should be empty for local user")
	}

	if user.Workgroup == "" {
		t.Error("Workgroup should not be empty for local user")
	}

	if user.IsDomainUser {
		t.Error("IsDomainUser should be false")
	}

	if !user.IsLocalUser {
		t.Error("IsLocalUser should be true")
	}
}

// ============ Тесты на логику ============

func TestUserInfoLogic(t *testing.T) {
	tests := []struct {
		name         string
		user         UserInfo
		expectedType string
	}{
		{
			name: "Domain user",
			user: UserInfo{
				IsDomainUser: true,
				IsLocalUser:  false,
			},
			expectedType: "domain",
		},
		{
			name: "Local user",
			user: UserInfo{
				IsDomainUser: false,
				IsLocalUser:  true,
			},
			expectedType: "local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Проверяем, что пользователь не может быть одновременно доменным и локальным
			if tt.user.IsDomainUser && tt.user.IsLocalUser {
				t.Error("User cannot be both domain and local")
			}

			// Проверяем, что пользователь должен быть либо доменным, либо локальным
			if !tt.user.IsDomainUser && !tt.user.IsLocalUser {
				t.Error("User must be either domain or local")
			}
		})
	}
}

// ============ Тесты на JSON поля ============

func TestUserInfoJSONFields(t *testing.T) {
	user := UserInfo{}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	jsonStr := string(data)

	requiredFields := []string{
		"username",
		"full_name",
		"domain",
		"domain_full",
		"workgroup",
		"profile_path",
		"is_admin",
		"is_domain_user",
		"is_local_user",
	}

	for _, field := range requiredFields {
		if !strings.Contains(jsonStr, `"`+field+`"`) {
			t.Errorf("JSON missing field: %s", field)
		}
	}
}

// ============ Тесты на формат Username ============

func TestUserInfoUsernameFormat(t *testing.T) {
	tests := []struct {
		name       string
		username   string
		isDomain   bool
		domainPart string
	}{
		{
			name:       "Domain user",
			username:   `ACME\john.doe`,
			isDomain:   true,
			domainPart: "ACME",
		},
		{
			name:       "Local user with computer name",
			username:   `DESKTOP-ABC123\Admin`,
			isDomain:   false,
			domainPart: "DESKTOP-ABC123",
		},
		{
			name:       "UPN format",
			username:   "john.doe@acme.corp.local",
			isDomain:   true,
			domainPart: "acme.corp.local",
		},
		{
			name:       "Simple username",
			username:   "Admin",
			isDomain:   false,
			domainPart: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hasDomainPart bool
			var domainPart string

			if strings.Contains(tt.username, "\\") {
				parts := strings.SplitN(tt.username, "\\", 2)
				domainPart = parts[0]
				hasDomainPart = true
			} else if strings.Contains(tt.username, "@") {
				parts := strings.SplitN(tt.username, "@", 2)
				domainPart = parts[1]
				hasDomainPart = true
			}

			if hasDomainPart != (domainPart != "") {
				t.Errorf("Has domain part = %v, domain part = %q", hasDomainPart, domainPart)
			}

			// Проверяем, что извлекли правильную часть
			if tt.domainPart != "" && domainPart != tt.domainPart {
				t.Errorf("Domain part = %s, want %s", domainPart, tt.domainPart)
			}
		})
	}
}

// ============ Бенчмарки ============

func BenchmarkUserInfoJSON(b *testing.B) {
	user := UserInfo{
		Username:     `DOMAIN\User`,
		FullName:     "User User",
		Domain:       "DOMAIN",
		DomainFull:   "domain.tld",
		ProfilePath:  `C:\Users\User`,
		IsAdmin:      true,
		IsDomainUser: true,
		IsLocalUser:  false,
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = json.Marshal(user)
	}
}
