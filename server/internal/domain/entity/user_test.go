package entity

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUser_Roles(t *testing.T) {
	tests := []struct {
		name           string
		role           UserRole
		isAdmin        bool
		isRSSI         bool
		isOperator     bool
		isAnalyst      bool
		canExecute     bool
		canView        bool
		canManageUsers bool
	}{
		{"admin", RoleAdmin, true, true, true, true, true, true, true},
		{"rssi", RoleRSSI, false, true, false, true, false, true, false},
		{"operator", RoleOperator, false, false, true, true, true, true, false},
		{"analyst", RoleAnalyst, false, false, false, true, false, true, false},
		{"viewer", RoleViewer, false, false, false, false, false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Role: tt.role, IsActive: true}

			if user.IsAdmin() != tt.isAdmin {
				t.Errorf("IsAdmin() = %v, want %v", user.IsAdmin(), tt.isAdmin)
			}
			if user.IsRSSI() != tt.isRSSI {
				t.Errorf("IsRSSI() = %v, want %v", user.IsRSSI(), tt.isRSSI)
			}
			if user.IsOperator() != tt.isOperator {
				t.Errorf("IsOperator() = %v, want %v", user.IsOperator(), tt.isOperator)
			}
			if user.IsAnalyst() != tt.isAnalyst {
				t.Errorf("IsAnalyst() = %v, want %v", user.IsAnalyst(), tt.isAnalyst)
			}
			if user.CanExecute() != tt.canExecute {
				t.Errorf("CanExecute() = %v, want %v", user.CanExecute(), tt.canExecute)
			}
			if user.CanView() != tt.canView {
				t.Errorf("CanView() = %v, want %v", user.CanView(), tt.canView)
			}
			if user.CanManageUsers() != tt.canManageUsers {
				t.Errorf("CanManageUsers() = %v, want %v", user.CanManageUsers(), tt.canManageUsers)
			}
		})
	}
}

func TestUser_CanView_InactiveUser(t *testing.T) {
	user := &User{Role: RoleAdmin, IsActive: false}
	if user.CanView() {
		t.Error("Inactive user should not be able to view")
	}
}

func TestValidRoles(t *testing.T) {
	roles := ValidRoles()

	if len(roles) != 5 {
		t.Errorf("Expected 5 roles, got %d", len(roles))
	}

	expected := []UserRole{RoleAdmin, RoleRSSI, RoleOperator, RoleAnalyst, RoleViewer}
	for i, role := range roles {
		if role != expected[i] {
			t.Errorf("ValidRoles()[%d] = %v, want %v", i, role, expected[i])
		}
	}
}

func TestIsValidRole(t *testing.T) {
	tests := []struct {
		role  string
		valid bool
	}{
		{"admin", true},
		{"rssi", true},
		{"operator", true},
		{"analyst", true},
		{"viewer", true},
		{"invalid", false},
		{"", false},
		{"Admin", false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			if IsValidRole(tt.role) != tt.valid {
				t.Errorf("IsValidRole(%q) = %v, want %v", tt.role, IsValidRole(tt.role), tt.valid)
			}
		})
	}
}

func TestUser_JSONSerialization(t *testing.T) {
	now := time.Now()
	user := &User{
		ID:           "user-123",
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "secret-hash-should-not-appear",
		Role:         RoleAdmin,
		IsActive:     true,
		LastLoginAt:  &now,
		CreatedAt:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal user: %v", err)
	}

	jsonStr := string(data)

	// PasswordHash should NOT appear in JSON (json:"-" tag)
	if contains(jsonStr, "secret-hash") {
		t.Error("PasswordHash should not be included in JSON output")
	}
	if contains(jsonStr, "password_hash") {
		t.Error("password_hash field should not be included in JSON output")
	}

	// Other fields should appear
	if !contains(jsonStr, "user-123") {
		t.Error("ID should be included in JSON output")
	}
	if !contains(jsonStr, "testuser") {
		t.Error("Username should be included in JSON output")
	}
	if !contains(jsonStr, "test@example.com") {
		t.Error("Email should be included in JSON output")
	}
	if !contains(jsonStr, "admin") {
		t.Error("Role should be included in JSON output")
	}
	if !contains(jsonStr, "is_active") {
		t.Error("IsActive should be included in JSON output")
	}
	if !contains(jsonStr, "last_login_at") {
		t.Error("LastLoginAt should be included in JSON output")
	}
}

func TestUser_JSONSerialization_NilLastLogin(t *testing.T) {
	user := &User{
		ID:           "user-123",
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "secret",
		Role:         RoleAdmin,
		IsActive:     true,
		LastLoginAt:  nil, // Not set
		CreatedAt:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal user: %v", err)
	}

	jsonStr := string(data)

	// LastLoginAt should be omitted when nil (omitempty)
	if contains(jsonStr, "last_login_at") {
		t.Error("LastLoginAt should be omitted when nil")
	}
}

func TestUserRole_Constants(t *testing.T) {
	if RoleAdmin != "admin" {
		t.Errorf("RoleAdmin = %q, want 'admin'", RoleAdmin)
	}
	if RoleRSSI != "rssi" {
		t.Errorf("RoleRSSI = %q, want 'rssi'", RoleRSSI)
	}
	if RoleOperator != "operator" {
		t.Errorf("RoleOperator = %q, want 'operator'", RoleOperator)
	}
	if RoleAnalyst != "analyst" {
		t.Errorf("RoleAnalyst = %q, want 'analyst'", RoleAnalyst)
	}
	if RoleViewer != "viewer" {
		t.Errorf("RoleViewer = %q, want 'viewer'", RoleViewer)
	}
}

func TestUserRole_DisplayName(t *testing.T) {
	tests := []struct {
		role    UserRole
		display string
	}{
		{RoleAdmin, "Administrator"},
		{RoleRSSI, "Security Officer (RSSI)"},
		{RoleOperator, "Operator"},
		{RoleAnalyst, "Analyst"},
		{RoleViewer, "Viewer"},
		{UserRole("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if tt.role.DisplayName() != tt.display {
				t.Errorf("DisplayName() = %q, want %q", tt.role.DisplayName(), tt.display)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
