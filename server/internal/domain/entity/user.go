package entity

import (
	"time"
)

// UserRole represents the authorization level of a user
type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleOperator UserRole = "operator"
	RoleViewer   UserRole = "viewer"
)

// User represents an authenticated user in the system
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never exposed in JSON
	Role         UserRole  `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// IsAdmin returns true if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsOperator returns true if the user has operator or admin role
func (u *User) IsOperator() bool {
	return u.Role == RoleAdmin || u.Role == RoleOperator
}

// CanExecute returns true if the user can start executions
func (u *User) CanExecute() bool {
	return u.Role == RoleAdmin || u.Role == RoleOperator
}

// CanView returns true if the user can view data (all roles can)
func (u *User) CanView() bool {
	return true
}

// ValidRoles returns all valid user roles
func ValidRoles() []UserRole {
	return []UserRole{RoleAdmin, RoleOperator, RoleViewer}
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
