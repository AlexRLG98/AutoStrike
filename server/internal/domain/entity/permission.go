package entity

// Permission represents a specific action that can be performed in the system
type Permission string

const (
	// User management permissions
	PermissionUsersView   Permission = "users:view"
	PermissionUsersCreate Permission = "users:create"
	PermissionUsersEdit   Permission = "users:edit"
	PermissionUsersDelete Permission = "users:delete"

	// Agent permissions
	PermissionAgentsView   Permission = "agents:view"
	PermissionAgentsCreate Permission = "agents:create"
	PermissionAgentsDelete Permission = "agents:delete"

	// Technique permissions
	PermissionTechniquesView   Permission = "techniques:view"
	PermissionTechniquesImport Permission = "techniques:import"

	// Scenario permissions
	PermissionScenariosView   Permission = "scenarios:view"
	PermissionScenariosCreate Permission = "scenarios:create"
	PermissionScenariosEdit   Permission = "scenarios:edit"
	PermissionScenariosDelete Permission = "scenarios:delete"
	PermissionScenariosImport Permission = "scenarios:import"
	PermissionScenariosExport Permission = "scenarios:export"

	// Execution permissions
	PermissionExecutionsView  Permission = "executions:view"
	PermissionExecutionsStart Permission = "executions:start"
	PermissionExecutionsStop  Permission = "executions:stop"

	// Analytics permissions
	PermissionAnalyticsView    Permission = "analytics:view"
	PermissionAnalyticsCompare Permission = "analytics:compare"
	PermissionAnalyticsExport  Permission = "analytics:export"

	// Settings permissions
	PermissionSettingsView Permission = "settings:view"
	PermissionSettingsEdit Permission = "settings:edit"

	// Scheduler permissions
	PermissionSchedulerView   Permission = "scheduler:view"
	PermissionSchedulerCreate Permission = "scheduler:create"
	PermissionSchedulerEdit   Permission = "scheduler:edit"
	PermissionSchedulerDelete Permission = "scheduler:delete"
)

// PermissionCategory groups related permissions
type PermissionCategory struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions"`
}

// PermissionInfo provides detailed information about a permission
type PermissionInfo struct {
	Permission  Permission `json:"permission"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Category    string     `json:"category"`
}

// RolePermissions maps roles to their permissions
var RolePermissions = map[UserRole][]Permission{
	RoleAdmin: {
		// Full access to everything
		PermissionUsersView, PermissionUsersCreate, PermissionUsersEdit, PermissionUsersDelete,
		PermissionAgentsView, PermissionAgentsCreate, PermissionAgentsDelete,
		PermissionTechniquesView, PermissionTechniquesImport,
		PermissionScenariosView, PermissionScenariosCreate, PermissionScenariosEdit, PermissionScenariosDelete, PermissionScenariosImport, PermissionScenariosExport,
		PermissionExecutionsView, PermissionExecutionsStart, PermissionExecutionsStop,
		PermissionAnalyticsView, PermissionAnalyticsCompare, PermissionAnalyticsExport,
		PermissionSettingsView, PermissionSettingsEdit,
		PermissionSchedulerView, PermissionSchedulerCreate, PermissionSchedulerEdit, PermissionSchedulerDelete,
	},
	RoleRSSI: {
		// Security officer - full view access, analytics, reports
		PermissionUsersView,
		PermissionAgentsView,
		PermissionTechniquesView,
		PermissionScenariosView, PermissionScenariosExport,
		PermissionExecutionsView,
		PermissionAnalyticsView, PermissionAnalyticsCompare, PermissionAnalyticsExport,
		PermissionSettingsView,
		PermissionSchedulerView,
	},
	RoleOperator: {
		// Operator - can execute and manage scenarios
		PermissionAgentsView, PermissionAgentsCreate, PermissionAgentsDelete,
		PermissionTechniquesView, PermissionTechniquesImport,
		PermissionScenariosView, PermissionScenariosCreate, PermissionScenariosEdit, PermissionScenariosDelete, PermissionScenariosImport, PermissionScenariosExport,
		PermissionExecutionsView, PermissionExecutionsStart, PermissionExecutionsStop,
		PermissionAnalyticsView,
		PermissionSettingsView,
		PermissionSchedulerView, PermissionSchedulerCreate, PermissionSchedulerEdit, PermissionSchedulerDelete,
	},
	RoleAnalyst: {
		// Analyst - read-only with analytics capabilities
		PermissionAgentsView,
		PermissionTechniquesView,
		PermissionScenariosView, PermissionScenariosExport,
		PermissionExecutionsView,
		PermissionAnalyticsView, PermissionAnalyticsCompare, PermissionAnalyticsExport,
		PermissionSettingsView,
		PermissionSchedulerView,
	},
	RoleViewer: {
		// Viewer - basic read-only access
		PermissionAgentsView,
		PermissionTechniquesView,
		PermissionScenariosView,
		PermissionExecutionsView,
		PermissionSettingsView,
	},
}

// GetPermissionCategories returns all permission categories with their permissions
func GetPermissionCategories() []PermissionCategory {
	return []PermissionCategory{
		{
			Name:        "Users",
			Description: "User management permissions",
			Permissions: []Permission{PermissionUsersView, PermissionUsersCreate, PermissionUsersEdit, PermissionUsersDelete},
		},
		{
			Name:        "Agents",
			Description: "Agent management permissions",
			Permissions: []Permission{PermissionAgentsView, PermissionAgentsCreate, PermissionAgentsDelete},
		},
		{
			Name:        "Techniques",
			Description: "Technique management permissions",
			Permissions: []Permission{PermissionTechniquesView, PermissionTechniquesImport},
		},
		{
			Name:        "Scenarios",
			Description: "Scenario management permissions",
			Permissions: []Permission{PermissionScenariosView, PermissionScenariosCreate, PermissionScenariosEdit, PermissionScenariosDelete, PermissionScenariosImport, PermissionScenariosExport},
		},
		{
			Name:        "Executions",
			Description: "Execution management permissions",
			Permissions: []Permission{PermissionExecutionsView, PermissionExecutionsStart, PermissionExecutionsStop},
		},
		{
			Name:        "Analytics",
			Description: "Analytics and reporting permissions",
			Permissions: []Permission{PermissionAnalyticsView, PermissionAnalyticsCompare, PermissionAnalyticsExport},
		},
		{
			Name:        "Settings",
			Description: "System settings permissions",
			Permissions: []Permission{PermissionSettingsView, PermissionSettingsEdit},
		},
		{
			Name:        "Scheduler",
			Description: "Scheduled execution permissions",
			Permissions: []Permission{PermissionSchedulerView, PermissionSchedulerCreate, PermissionSchedulerEdit, PermissionSchedulerDelete},
		},
	}
}

// GetPermissionInfo returns detailed information about all permissions
func GetPermissionInfo() []PermissionInfo {
	return []PermissionInfo{
		// Users
		{PermissionUsersView, "View Users", "View list of users", "Users"},
		{PermissionUsersCreate, "Create Users", "Create new users", "Users"},
		{PermissionUsersEdit, "Edit Users", "Edit existing users", "Users"},
		{PermissionUsersDelete, "Delete Users", "Deactivate or delete users", "Users"},
		// Agents
		{PermissionAgentsView, "View Agents", "View connected agents", "Agents"},
		{PermissionAgentsCreate, "Create Agents", "Register new agents", "Agents"},
		{PermissionAgentsDelete, "Delete Agents", "Remove agents", "Agents"},
		// Techniques
		{PermissionTechniquesView, "View Techniques", "View MITRE techniques", "Techniques"},
		{PermissionTechniquesImport, "Import Techniques", "Import techniques from YAML", "Techniques"},
		// Scenarios
		{PermissionScenariosView, "View Scenarios", "View attack scenarios", "Scenarios"},
		{PermissionScenariosCreate, "Create Scenarios", "Create new scenarios", "Scenarios"},
		{PermissionScenariosEdit, "Edit Scenarios", "Edit existing scenarios", "Scenarios"},
		{PermissionScenariosDelete, "Delete Scenarios", "Delete scenarios", "Scenarios"},
		{PermissionScenariosImport, "Import Scenarios", "Import scenarios from JSON", "Scenarios"},
		{PermissionScenariosExport, "Export Scenarios", "Export scenarios to JSON", "Scenarios"},
		// Executions
		{PermissionExecutionsView, "View Executions", "View execution history and results", "Executions"},
		{PermissionExecutionsStart, "Start Executions", "Start new attack simulations", "Executions"},
		{PermissionExecutionsStop, "Stop Executions", "Stop running executions", "Executions"},
		// Analytics
		{PermissionAnalyticsView, "View Analytics", "View security analytics", "Analytics"},
		{PermissionAnalyticsCompare, "Compare Analytics", "Compare scores across periods", "Analytics"},
		{PermissionAnalyticsExport, "Export Analytics", "Export reports and analytics", "Analytics"},
		// Settings
		{PermissionSettingsView, "View Settings", "View system settings", "Settings"},
		{PermissionSettingsEdit, "Edit Settings", "Modify system settings", "Settings"},
		// Scheduler
		{PermissionSchedulerView, "View Schedules", "View scheduled executions", "Scheduler"},
		{PermissionSchedulerCreate, "Create Schedules", "Create scheduled executions", "Scheduler"},
		{PermissionSchedulerEdit, "Edit Schedules", "Edit scheduled executions", "Scheduler"},
		{PermissionSchedulerDelete, "Delete Schedules", "Delete scheduled executions", "Scheduler"},
	}
}

// HasPermission checks if a role has a specific permission
func HasPermission(role UserRole, permission Permission) bool {
	permissions, exists := RolePermissions[role]
	if !exists {
		return false
	}
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// GetRolePermissions returns all permissions for a specific role
func GetRolePermissions(role UserRole) []Permission {
	return RolePermissions[role]
}

// PermissionMatrix represents the full permission matrix for all roles
type PermissionMatrix struct {
	Roles       []UserRole                  `json:"roles"`
	Categories  []PermissionCategory        `json:"categories"`
	Permissions []PermissionInfo            `json:"permissions"`
	Matrix      map[UserRole][]Permission   `json:"matrix"`
}

// GetPermissionMatrix returns the complete permission matrix
func GetPermissionMatrix() *PermissionMatrix {
	return &PermissionMatrix{
		Roles:       ValidRoles(),
		Categories:  GetPermissionCategories(),
		Permissions: GetPermissionInfo(),
		Matrix:      RolePermissions,
	}
}
