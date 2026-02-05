package handlers

import (
	"net/http"

	"autostrike/internal/domain/entity"

	"github.com/gin-gonic/gin"
)

const errNotAuthenticated = "not authenticated"

// PermissionHandler handles permission-related HTTP requests
type PermissionHandler struct{}

// NewPermissionHandler creates a new permission handler
func NewPermissionHandler() *PermissionHandler {
	return &PermissionHandler{}
}

// RegisterRoutes registers permission routes
func (h *PermissionHandler) RegisterRoutes(r *gin.RouterGroup) {
	perms := r.Group("/permissions")
	{
		perms.GET("", h.GetPermissionMatrix)
		perms.GET("/me", h.GetMyPermissions)
		perms.GET("/roles", h.GetRoles)
	}
}

// GetPermissionMatrix returns the complete permission matrix
func (h *PermissionHandler) GetPermissionMatrix(c *gin.Context) {
	// Require authentication
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotAuthenticated})
		return
	}

	matrix := entity.GetPermissionMatrix()
	c.JSON(http.StatusOK, matrix)
}

// GetMyPermissions returns the current user's permissions
func (h *PermissionHandler) GetMyPermissions(c *gin.Context) {
	// Require authentication - check both user_id and role
	_, userExists := c.Get("user_id")
	if !userExists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotAuthenticated})
		return
	}

	role, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotAuthenticated})
		return
	}

	roleStr, ok := role.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid role format"})
		return
	}

	userRole := entity.UserRole(roleStr)
	permissions := entity.GetRolePermissions(userRole)

	// Convert to string slice for JSON
	permStrings := make([]string, len(permissions))
	for i, p := range permissions {
		permStrings[i] = string(p)
	}

	c.JSON(http.StatusOK, gin.H{
		"role":        roleStr,
		"permissions": permStrings,
	})
}

// RoleInfo represents information about a role
type RoleInfo struct {
	Role        string   `json:"role"`
	DisplayName string   `json:"display_name"`
	Permissions []string `json:"permissions"`
}

// GetRoles returns all available roles with their permissions
func (h *PermissionHandler) GetRoles(c *gin.Context) {
	// Require authentication
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotAuthenticated})
		return
	}

	roles := entity.ValidRoles()
	result := make([]RoleInfo, len(roles))

	for i, role := range roles {
		perms := entity.GetRolePermissions(role)
		permStrings := make([]string, len(perms))
		for j, p := range perms {
			permStrings[j] = string(p)
		}

		result[i] = RoleInfo{
			Role:        string(role),
			DisplayName: role.DisplayName(),
			Permissions: permStrings,
		}
	}

	c.JSON(http.StatusOK, gin.H{"roles": result})
}
