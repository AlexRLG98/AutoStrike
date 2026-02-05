package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"autostrike/internal/domain/entity"

	"github.com/gin-gonic/gin"
)

const testUserID = "test-user"

func TestNewPermissionHandler(t *testing.T) {
	handler := NewPermissionHandler()
	if handler == nil {
		t.Fatal("NewPermissionHandler returned nil")
	}
}

func TestPermissionHandler_RegisterRoutes(t *testing.T) {
	handler := NewPermissionHandler()

	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	routes := router.Routes()
	expectedPaths := map[string]string{
		"/api/v1/permissions":       "GET",
		"/api/v1/permissions/me":    "GET",
		"/api/v1/permissions/roles": "GET",
	}

	for path, method := range expectedPaths {
		found := false
		for _, route := range routes {
			if route.Path == path && route.Method == method {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Route %s %s not found", method, path)
		}
	}
}

func TestPermissionHandler_GetPermissionMatrix(t *testing.T) {
	handler := NewPermissionHandler()

	router := gin.New()
	router.GET("/permissions", func(c *gin.Context) {
		c.Set("user_id", "test-user-id")
		handler.GetPermissionMatrix(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/permissions", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result entity.PermissionMatrix
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(result.Roles) != 5 {
		t.Errorf("Expected 5 roles, got %d", len(result.Roles))
	}

	if len(result.Categories) < 5 {
		t.Errorf("Expected at least 5 categories, got %d", len(result.Categories))
	}

	if len(result.Permissions) < 20 {
		t.Errorf("Expected at least 20 permissions, got %d", len(result.Permissions))
	}

	if len(result.Matrix) != 5 {
		t.Errorf("Expected 5 roles in matrix, got %d", len(result.Matrix))
	}
}

func TestPermissionHandler_GetMyPermissions_Success(t *testing.T) {
	handler := NewPermissionHandler()

	router := gin.New()
	router.GET("/permissions/me", func(c *gin.Context) {
		c.Set("user_id", testUserID)
		c.Set("role", "admin")
		handler.GetMyPermissions(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/permissions/me", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result struct {
		Role        string   `json:"role"`
		Permissions []string `json:"permissions"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result.Role != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", result.Role)
	}

	if len(result.Permissions) < 25 {
		t.Errorf("Expected admin to have at least 25 permissions, got %d", len(result.Permissions))
	}
}

func TestPermissionHandler_GetMyPermissions_Viewer(t *testing.T) {
	handler := NewPermissionHandler()

	router := gin.New()
	router.GET("/permissions/me", func(c *gin.Context) {
		c.Set("user_id", testUserID)
		c.Set("role", "viewer")
		handler.GetMyPermissions(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/permissions/me", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result struct {
		Role        string   `json:"role"`
		Permissions []string `json:"permissions"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result.Role != "viewer" {
		t.Errorf("Expected role 'viewer', got '%s'", result.Role)
	}

	// Viewer should have fewer permissions than admin
	if len(result.Permissions) > 10 {
		t.Errorf("Expected viewer to have at most 10 permissions, got %d", len(result.Permissions))
	}
}

func TestPermissionHandler_GetMyPermissions_NotAuthenticated(t *testing.T) {
	handler := NewPermissionHandler()

	router := gin.New()
	router.GET("/permissions/me", handler.GetMyPermissions)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/permissions/me", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestPermissionHandler_GetMyPermissions_InvalidRole(t *testing.T) {
	handler := NewPermissionHandler()

	router := gin.New()
	router.GET("/permissions/me", func(c *gin.Context) {
		c.Set("user_id", testUserID)
		c.Set("role", 123) // Invalid type
		handler.GetMyPermissions(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/permissions/me", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestPermissionHandler_GetPermissionMatrix_NotAuthenticated(t *testing.T) {
	handler := NewPermissionHandler()

	router := gin.New()
	router.GET("/permissions", handler.GetPermissionMatrix)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/permissions", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestPermissionHandler_GetRoles(t *testing.T) {
	handler := NewPermissionHandler()

	router := gin.New()
	router.GET("/permissions/roles", func(c *gin.Context) {
		c.Set("user_id", "test-user-id")
		handler.GetRoles(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/permissions/roles", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result struct {
		Roles []RoleInfo `json:"roles"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(result.Roles) != 5 {
		t.Errorf("Expected 5 roles, got %d", len(result.Roles))
	}

	// Check each role has required fields
	for _, role := range result.Roles {
		if role.Role == "" {
			t.Error("Role has empty role field")
		}
		if role.DisplayName == "" {
			t.Errorf("Role %s has empty display name", role.Role)
		}
		if len(role.Permissions) == 0 {
			t.Errorf("Role %s has no permissions", role.Role)
		}
	}

	// Verify role order (should match ValidRoles())
	expectedRoles := []string{"admin", "rssi", "operator", "analyst", "viewer"}
	for i, expected := range expectedRoles {
		if result.Roles[i].Role != expected {
			t.Errorf("Expected role %s at index %d, got %s", expected, i, result.Roles[i].Role)
		}
	}
}

func TestPermissionHandler_GetRoles_NotAuthenticated(t *testing.T) {
	handler := NewPermissionHandler()

	router := gin.New()
	router.GET("/permissions/roles", handler.GetRoles)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/permissions/roles", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestPermissionHandler_GetRoles_PermissionCounts(t *testing.T) {
	handler := NewPermissionHandler()

	router := gin.New()
	router.GET("/permissions/roles", func(c *gin.Context) {
		c.Set("user_id", "test-user-id")
		handler.GetRoles(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/permissions/roles", nil)
	router.ServeHTTP(w, req)

	var result struct {
		Roles []RoleInfo `json:"roles"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &result)

	// Admin should have most permissions
	adminPerms := 0
	viewerPerms := 0
	for _, role := range result.Roles {
		if role.Role == "admin" {
			adminPerms = len(role.Permissions)
		}
		if role.Role == "viewer" {
			viewerPerms = len(role.Permissions)
		}
	}

	if adminPerms <= viewerPerms {
		t.Errorf("Admin should have more permissions than viewer (admin: %d, viewer: %d)", adminPerms, viewerPerms)
	}
}
