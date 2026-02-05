package handlers

import (
	"errors"
	"net/http"

	"autostrike/internal/application"
	"autostrike/internal/domain/entity"

	"github.com/gin-gonic/gin"
)

// Admin handler error messages
const (
	errAdminAccessRequired = "admin access required"
	errUserIDRequired      = "user id is required"
	errUserNotFound        = "user not found"
	errInvalidRole         = "invalid role"
	timeFormatISO8601      = "2006-01-02T15:04:05Z"
)

// AdminHandler handles admin-related HTTP requests
type AdminHandler struct {
	authService *application.AuthService
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(authService *application.AuthService) *AdminHandler {
	return &AdminHandler{authService: authService}
}

// RegisterRoutes registers admin routes (requires authentication and admin role)
func (h *AdminHandler) RegisterRoutes(r *gin.RouterGroup) {
	admin := r.Group("/admin")
	{
		users := admin.Group("/users")
		{
			users.GET("", h.ListUsers)
			users.GET("/:id", h.GetUser)
			users.POST("", h.CreateUser)
			users.PUT("/:id", h.UpdateUser)
			users.PUT("/:id/role", h.UpdateUserRole)
			users.DELETE("/:id", h.DeactivateUser)
			users.POST("/:id/reactivate", h.ReactivateUser)
			users.POST("/:id/reset-password", h.ResetPassword)
		}
	}
}

// ListUsersResponse represents the list users response
type ListUsersResponse struct {
	Users []*UserResponse `json:"users"`
	Total int             `json:"total"`
}

// UserResponse represents a user in API responses
type UserResponse struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	Email       string  `json:"email"`
	Role        string  `json:"role"`
	RoleDisplay string  `json:"role_display"`
	IsActive    bool    `json:"is_active"`
	LastLoginAt *string `json:"last_login_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// toUserResponse converts an entity.User to a UserResponse
func toUserResponse(user *entity.User) *UserResponse {
	resp := &UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		Role:        string(user.Role),
		RoleDisplay: user.Role.DisplayName(),
		IsActive:    user.IsActive,
		CreatedAt:   user.CreatedAt.Format(timeFormatISO8601),
		UpdatedAt:   user.UpdatedAt.Format(timeFormatISO8601),
	}
	if user.LastLoginAt != nil {
		formatted := user.LastLoginAt.Format(timeFormatISO8601)
		resp.LastLoginAt = &formatted
	}
	return resp
}

// ListUsers returns all users
func (h *AdminHandler) ListUsers(c *gin.Context) {
	// Check admin permission
	if !h.isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": errAdminAccessRequired})
		return
	}

	users, err := h.authService.GetAllUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get users"})
		return
	}

	response := make([]*UserResponse, len(users))
	for i, user := range users {
		response[i] = toUserResponse(user)
	}

	c.JSON(http.StatusOK, ListUsersResponse{
		Users: response,
		Total: len(response),
	})
}

// GetUser returns a specific user
func (h *AdminHandler) GetUser(c *gin.Context) {
	if !h.isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": errAdminAccessRequired})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errUserIDRequired})
		return
	}

	user, err := h.authService.GetUser(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, application.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": errUserNotFound})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// CreateUserRequest represents the create user request body
type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role" binding:"required"`
}

// CreateUser creates a new user
func (h *AdminHandler) CreateUser(c *gin.Context) {
	if !h.isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": errAdminAccessRequired})
		return
	}

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate role
	if !entity.IsValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       errInvalidRole,
			"valid_roles": entity.ValidRoles(),
		})
		return
	}

	user, err := h.authService.CreateUser(
		c.Request.Context(),
		req.Username,
		req.Email,
		req.Password,
		entity.UserRole(req.Role),
	)
	if err != nil {
		if errors.Is(err, application.ErrUserAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "username or email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, toUserResponse(user))
}

// UpdateUserRequest represents the update user request body
type UpdateUserRequest struct {
	Username string `json:"username" binding:"omitempty,min=3,max=50"`
	Email    string `json:"email" binding:"omitempty,email"`
	Role     string `json:"role" binding:"omitempty"`
}

// handleUpdateError handles common update errors and returns true if error was handled
func handleUpdateError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, application.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": errUserNotFound})
	case errors.Is(err, application.ErrUserAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": "username or email already exists"})
	case errors.Is(err, application.ErrInvalidRole):
		c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidRole})
	default:
		return false
	}
	return true
}

// mergeUserFields merges request fields with current user values
func mergeUserFields(req *UpdateUserRequest, currentUser *entity.User) (string, string, entity.UserRole, bool) {
	username := currentUser.Username
	if req.Username != "" {
		username = req.Username
	}

	email := currentUser.Email
	if req.Email != "" {
		email = req.Email
	}

	role := currentUser.Role
	if req.Role != "" {
		if !entity.IsValidRole(req.Role) {
			return "", "", "", false
		}
		role = entity.UserRole(req.Role)
	}

	return username, email, role, true
}

// UpdateUser updates a user's details
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	if !h.isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": errAdminAccessRequired})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errUserIDRequired})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	currentUser, err := h.authService.GetUser(c.Request.Context(), id)
	if err != nil {
		if !handleUpdateError(c, err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		}
		return
	}

	username, email, role, valid := mergeUserFields(&req, currentUser)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       errInvalidRole,
			"valid_roles": entity.ValidRoles(),
		})
		return
	}

	user, err := h.authService.UpdateUser(c.Request.Context(), id, username, email, role)
	if err != nil {
		if !handleUpdateError(c, err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		}
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// UpdateRoleRequest represents the update role request body
type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// UpdateUserRole updates only a user's role
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	if !h.isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": errAdminAccessRequired})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errUserIDRequired})
		return
	}

	var req UpdateRoleRequest
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role is required"})
		return
	}

	if !entity.IsValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       errInvalidRole,
			"valid_roles": entity.ValidRoles(),
		})
		return
	}

	user, err := h.authService.UpdateUserRole(c.Request.Context(), id, entity.UserRole(req.Role))
	if err != nil {
		if errors.Is(err, application.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": errUserNotFound})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user role"})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// DeactivateUser deactivates a user account
func (h *AdminHandler) DeactivateUser(c *gin.Context) {
	if !h.isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": errAdminAccessRequired})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errUserIDRequired})
		return
	}

	currentUserID, _ := c.Get("user_id")
	currentUserIDStr, _ := currentUserID.(string)

	if err := h.authService.DeactivateUser(c.Request.Context(), id, currentUserIDStr); err != nil {
		if errors.Is(err, application.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": errUserNotFound})
			return
		}
		if errors.Is(err, application.ErrCannotDeactivateSelf) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot deactivate your own account"})
			return
		}
		if errors.Is(err, application.ErrLastAdmin) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot deactivate the last admin user"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deactivate user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deactivated successfully"})
}

// ReactivateUser reactivates a deactivated user account
func (h *AdminHandler) ReactivateUser(c *gin.Context) {
	if !h.isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": errAdminAccessRequired})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errUserIDRequired})
		return
	}

	if err := h.authService.ReactivateUser(c.Request.Context(), id); err != nil {
		if errors.Is(err, application.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": errUserNotFound})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reactivate user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user reactivated successfully"})
}

// ResetPasswordRequest represents the reset password request body
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ResetPassword resets a user's password (admin action)
func (h *AdminHandler) ResetPassword(c *gin.Context) {
	if !h.isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": errAdminAccessRequired})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errUserIDRequired})
		return
	}

	var req ResetPasswordRequest
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new_password is required (min 8 characters)"})
		return
	}

	if err := h.authService.ResetPassword(c.Request.Context(), id, req.NewPassword); err != nil {
		if errors.Is(err, application.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": errUserNotFound})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password reset successfully"})
}

// isAdmin checks if the current user has admin role
func (h *AdminHandler) isAdmin(c *gin.Context) bool {
	role, exists := c.Get("role")
	if !exists {
		return false
	}
	roleStr, ok := role.(string)
	if !ok {
		return false
	}
	return roleStr == string(entity.RoleAdmin)
}

// GetRoles returns all valid roles (useful for UI dropdowns)
func (h *AdminHandler) GetRoles(c *gin.Context) {
	if !h.isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": errAdminAccessRequired})
		return
	}

	roles := entity.ValidRoles()
	response := make([]map[string]string, len(roles))
	for i, role := range roles {
		response[i] = map[string]string{
			"value":   string(role),
			"display": role.DisplayName(),
		}
	}

	c.JSON(http.StatusOK, gin.H{"roles": response})
}
