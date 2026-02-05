package entity

import (
	"testing"
)

func TestHasPermission(t *testing.T) {
	tests := []struct {
		name       string
		role       UserRole
		permission Permission
		expected   bool
	}{
		// Admin has all permissions
		{"admin has users:view", RoleAdmin, PermissionUsersView, true},
		{"admin has users:create", RoleAdmin, PermissionUsersCreate, true},
		{"admin has executions:start", RoleAdmin, PermissionExecutionsStart, true},
		{"admin has settings:edit", RoleAdmin, PermissionSettingsEdit, true},

		// RSSI has view and analytics but not edit
		{"rssi has users:view", RoleRSSI, PermissionUsersView, true},
		{"rssi no users:create", RoleRSSI, PermissionUsersCreate, false},
		{"rssi has analytics:compare", RoleRSSI, PermissionAnalyticsCompare, true},
		{"rssi no executions:start", RoleRSSI, PermissionExecutionsStart, false},

		// Operator can execute but not manage users
		{"operator no users:view", RoleOperator, PermissionUsersView, false},
		{"operator has executions:start", RoleOperator, PermissionExecutionsStart, true},
		{"operator has scenarios:create", RoleOperator, PermissionScenariosCreate, true},

		// Analyst has view and analytics
		{"analyst has agents:view", RoleAnalyst, PermissionAgentsView, true},
		{"analyst has analytics:compare", RoleAnalyst, PermissionAnalyticsCompare, true},
		{"analyst no executions:start", RoleAnalyst, PermissionExecutionsStart, false},

		// Viewer has basic view only
		{"viewer has agents:view", RoleViewer, PermissionAgentsView, true},
		{"viewer has techniques:view", RoleViewer, PermissionTechniquesView, true},
		{"viewer no analytics:compare", RoleViewer, PermissionAnalyticsCompare, false},
		{"viewer no executions:start", RoleViewer, PermissionExecutionsStart, false},

		// Invalid role has no permissions
		{"invalid role", UserRole("invalid"), PermissionUsersView, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasPermission(tt.role, tt.permission)
			if result != tt.expected {
				t.Errorf("HasPermission(%s, %s) = %v, want %v", tt.role, tt.permission, result, tt.expected)
			}
		})
	}
}

func TestGetRolePermissions(t *testing.T) {
	tests := []struct {
		name        string
		role        UserRole
		minExpected int
	}{
		{"admin has most permissions", RoleAdmin, 25},
		{"rssi has moderate permissions", RoleRSSI, 10},
		{"operator has moderate permissions", RoleOperator, 15},
		{"analyst has some permissions", RoleAnalyst, 8},
		{"viewer has few permissions", RoleViewer, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perms := GetRolePermissions(tt.role)
			if len(perms) < tt.minExpected {
				t.Errorf("GetRolePermissions(%s) returned %d permissions, expected at least %d", tt.role, len(perms), tt.minExpected)
			}
		})
	}
}

func TestGetPermissionCategories(t *testing.T) {
	categories := GetPermissionCategories()

	if len(categories) < 5 {
		t.Errorf("Expected at least 5 permission categories, got %d", len(categories))
	}

	// Check that required categories exist
	expectedCategories := []string{"Users", "Agents", "Techniques", "Scenarios", "Executions", "Analytics", "Settings", "Scheduler"}
	categoryNames := make(map[string]bool)
	for _, cat := range categories {
		categoryNames[cat.Name] = true
	}

	for _, expected := range expectedCategories {
		if !categoryNames[expected] {
			t.Errorf("Missing expected category: %s", expected)
		}
	}

	// Check that each category has permissions
	for _, cat := range categories {
		if len(cat.Permissions) == 0 {
			t.Errorf("Category %s has no permissions", cat.Name)
		}
		if cat.Description == "" {
			t.Errorf("Category %s has no description", cat.Name)
		}
	}
}

func TestGetPermissionInfo(t *testing.T) {
	info := GetPermissionInfo()

	if len(info) < 20 {
		t.Errorf("Expected at least 20 permission info entries, got %d", len(info))
	}

	// Check that each permission has required fields
	for _, p := range info {
		if p.Permission == "" {
			t.Error("Permission info has empty permission")
		}
		if p.Name == "" {
			t.Errorf("Permission %s has empty name", p.Permission)
		}
		if p.Description == "" {
			t.Errorf("Permission %s has empty description", p.Permission)
		}
		if p.Category == "" {
			t.Errorf("Permission %s has empty category", p.Permission)
		}
	}
}

func TestGetPermissionMatrix(t *testing.T) {
	matrix := GetPermissionMatrix()

	if matrix == nil {
		t.Fatal("GetPermissionMatrix returned nil")
	}

	// Check roles
	if len(matrix.Roles) != 5 {
		t.Errorf("Expected 5 roles in matrix, got %d", len(matrix.Roles))
	}

	// Check categories
	if len(matrix.Categories) < 5 {
		t.Errorf("Expected at least 5 categories in matrix, got %d", len(matrix.Categories))
	}

	// Check permissions
	if len(matrix.Permissions) < 20 {
		t.Errorf("Expected at least 20 permissions in matrix, got %d", len(matrix.Permissions))
	}

	// Check matrix mapping
	if len(matrix.Matrix) != 5 {
		t.Errorf("Expected 5 roles in matrix mapping, got %d", len(matrix.Matrix))
	}

	// Verify admin has all permissions in the matrix
	adminPerms := matrix.Matrix[RoleAdmin]
	if len(adminPerms) < 25 {
		t.Errorf("Admin should have at least 25 permissions, got %d", len(adminPerms))
	}
}

func TestRolePermissionConsistency(t *testing.T) {
	// Verify that admin has all permissions that other roles have
	allPerms := make(map[Permission]bool)
	for _, role := range ValidRoles() {
		for _, perm := range GetRolePermissions(role) {
			allPerms[perm] = true
		}
	}

	adminPerms := GetRolePermissions(RoleAdmin)
	adminPermSet := make(map[Permission]bool)
	for _, p := range adminPerms {
		adminPermSet[p] = true
	}

	for perm := range allPerms {
		if !adminPermSet[perm] {
			t.Errorf("Admin is missing permission that exists in another role: %s", perm)
		}
	}
}

func TestPermissionConstants(t *testing.T) {
	// Verify permission constant naming convention
	allPermissions := []Permission{
		PermissionUsersView, PermissionUsersCreate, PermissionUsersEdit, PermissionUsersDelete,
		PermissionAgentsView, PermissionAgentsCreate, PermissionAgentsDelete,
		PermissionTechniquesView, PermissionTechniquesImport,
		PermissionScenariosView, PermissionScenariosCreate, PermissionScenariosEdit, PermissionScenariosDelete, PermissionScenariosImport, PermissionScenariosExport,
		PermissionExecutionsView, PermissionExecutionsStart, PermissionExecutionsStop,
		PermissionAnalyticsView, PermissionAnalyticsCompare, PermissionAnalyticsExport,
		PermissionSettingsView, PermissionSettingsEdit,
		PermissionSchedulerView, PermissionSchedulerCreate, PermissionSchedulerEdit, PermissionSchedulerDelete,
	}

	for _, perm := range allPermissions {
		if perm == "" {
			t.Error("Found empty permission constant")
		}
		// Check format: category:action
		if len(string(perm)) < 5 {
			t.Errorf("Permission %s appears to be malformed", perm)
		}
	}
}
