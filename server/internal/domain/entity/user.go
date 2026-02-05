package entity

import (
	"time"
)

// UserRole represents the authorization level of a user
type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleRSSI     UserRole = "rssi"     // Security Officer - view reports, analytics
	RoleOperator UserRole = "operator" // Can execute scenarios
	RoleAnalyst  UserRole = "analyst"  // Read-only with analytics
	RoleViewer   UserRole = "viewer"   // Read-only basic
)

// User represents an authenticated user in the system
type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"` // Never exposed in JSON
	Role         UserRole   `json:"role"`
	IsActive     bool       `json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// IsAdmin returns true if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsRSSI returns true if the user has RSSI role or higher
func (u *User) IsRSSI() bool {
	return u.Role == RoleAdmin || u.Role == RoleRSSI
}

// IsOperator returns true if the user has operator or admin role
func (u *User) IsOperator() bool {
	return u.Role == RoleAdmin || u.Role == RoleOperator
}

// IsAnalyst returns true if the user can analyze data
func (u *User) IsAnalyst() bool {
	return u.Role == RoleAdmin || u.Role == RoleRSSI || u.Role == RoleOperator || u.Role == RoleAnalyst
}

// CanExecute returns true if the user can start executions
func (u *User) CanExecute() bool {
	return u.Role == RoleAdmin || u.Role == RoleOperator
}

// CanView returns true if the user can view data (all active users can)
func (u *User) CanView() bool {
	return u.IsActive
}

// CanManageUsers returns true if the user can manage other users
func (u *User) CanManageUsers() bool {
	return u.Role == RoleAdmin
}

// ValidRoles returns all valid user roles
func ValidRoles() []UserRole {
	return []UserRole{RoleAdmin, RoleRSSI, RoleOperator, RoleAnalyst, RoleViewer}
}

// IsValidRole checks if a role string is valid
func IsValidRole(role string) bool {
	for _, r := range ValidRoles() {
		if string(r) == role {
			return true
		}
	}
	return false
}

// RoleDisplayName returns a human-readable name for a role
func (r UserRole) DisplayName() string {
	switch r {
	case RoleAdmin:
		return "Administrator"
	case RoleRSSI:
		return "Security Officer (RSSI)"
	case RoleOperator:
		return "Operator"
	case RoleAnalyst:
		return "Analyst"
	case RoleViewer:
		return "Viewer"
	default:
		return string(r)
	}
}
