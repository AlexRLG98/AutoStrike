package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"autostrike/internal/application"
	"autostrike/internal/domain/entity"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// mockUserRepo implements repository.UserRepository for testing
type mockUserRepo struct {
	users     map[string]*entity.User
	findErr   error
	createErr error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*entity.User)}
}

func (m *mockUserRepo) Create(ctx context.Context, user *entity.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) Update(ctx context.Context, user *entity.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) Delete(ctx context.Context, id string) error {
	delete(m.users, id)
	return nil
}

func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*entity.User, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	user, ok := m.users[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return user, nil
}

func (m *mockUserRepo) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *mockUserRepo) FindAll(ctx context.Context) ([]*entity.User, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	result := make([]*entity.User, 0, len(m.users))
	for _, user := range m.users {
		result = append(result, user)
	}
	return result, nil
}

func TestNewAuthHandler(t *testing.T) {
	repo := newMockUserRepo()
	service := application.NewAuthService(repo, "test-secret")
	handler := NewAuthHandler(service)

	if handler == nil {
		t.Fatal("NewAuthHandler returned nil")
	}
}

func TestAuthHandler_RegisterRoutes(t *testing.T) {
	repo := newMockUserRepo()
	service := application.NewAuthService(repo, "test-secret")
	handler := NewAuthHandler(service)

	router := gin.New()
	handler.RegisterRoutes(router)

	// Check routes are registered
	routes := router.Routes()
	expectedPaths := map[string]string{
		"/api/v1/auth/login":   "POST",
		"/api/v1/auth/refresh": "POST",
		"/api/v1/auth/logout":  "POST",
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

func TestAuthHandler_Login_Success(t *testing.T) {
	repo := newMockUserRepo()
	service := application.NewAuthService(repo, "test-secret")
	handler := NewAuthHandler(service)

	// Create user with hashed password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), 10)
	repo.users["user-1"] = &entity.User{
		ID:           "user-1",
		Username:     "testuser",
		PasswordHash: string(hashedPassword),
		Role:         entity.RoleAdmin,
	}

	router := gin.New()
	router.POST("/login", handler.Login)

	body := LoginRequest{
		Username: "testuser",
		Password: "password123",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response application.TokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if response.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	repo := newMockUserRepo()
	service := application.NewAuthService(repo, "test-secret")
	handler := NewAuthHandler(service)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), 10)
	repo.users["user-1"] = &entity.User{
		ID:           "user-1",
		Username:     "testuser",
		PasswordHash: string(hashedPassword),
	}

	router := gin.New()
	router.POST("/login", handler.Login)

	body := LoginRequest{
		Username: "testuser",
		Password: "wrongpassword",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuthHandler_Login_UserNotFound(t *testing.T) {
	repo := newMockUserRepo()
	service := application.NewAuthService(repo, "test-secret")
	handler := NewAuthHandler(service)

	router := gin.New()
	router.POST("/login", handler.Login)

	body := LoginRequest{
		Username: "nonexistent",
		Password: "password123",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuthHandler_Login_BadRequest(t *testing.T) {
	repo := newMockUserRepo()
	service := application.NewAuthService(repo, "test-secret")
	handler := NewAuthHandler(service)

	router := gin.New()
	router.POST("/login", handler.Login)

	// Missing password
	body := `{"username": "testuser"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAuthHandler_Login_InvalidJSON(t *testing.T) {
	repo := newMockUserRepo()
	service := application.NewAuthService(repo, "test-secret")
	handler := NewAuthHandler(service)

	router := gin.New()
	router.POST("/login", handler.Login)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/login", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAuthHandler_Refresh_Success(t *testing.T) {
	repo := newMockUserRepo()
	service := application.NewAuthService(repo, "test-secret")
	handler := NewAuthHandler(service)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), 10)
	repo.users["user-1"] = &entity.User{
		ID:           "user-1",
		Username:     "testuser",
		PasswordHash: string(hashedPassword),
		Role:         entity.RoleAdmin,
	}

	// Login first to get tokens
	tokens, _ := service.Login(context.Background(), "testuser", "password123")

	router := gin.New()
	router.POST("/refresh", handler.Refresh)

	body := RefreshRequest{
		RefreshToken: tokens.RefreshToken,
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/refresh", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_Refresh_InvalidToken(t *testing.T) {
	repo := newMockUserRepo()
	service := application.NewAuthService(repo, "test-secret")
	handler := NewAuthHandler(service)

	router := gin.New()
	router.POST("/refresh", handler.Refresh)

	body := RefreshRequest{
		RefreshToken: "invalid-token",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/refresh", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuthHandler_Refresh_BadRequest(t *testing.T) {
	repo := newMockUserRepo()
	service := application.NewAuthService(repo, "test-secret")
	handler := NewAuthHandler(service)

	router := gin.New()
	router.POST("/refresh", handler.Refresh)

	// Missing refresh_token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/refresh", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	repo := newMockUserRepo()
	service := application.NewAuthService(repo, "test-secret")
	handler := NewAuthHandler(service)

	router := gin.New()
	router.POST("/logout", handler.Logout)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/logout", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAuthHandler_Me_Success(t *testing.T) {
	repo := newMockUserRepo()
	service := application.NewAuthService(repo, "test-secret")
	handler := NewAuthHandler(service)

	repo.users["user-1"] = &entity.User{
		ID:       "user-1",
		Username: "testuser",
		Email:    "test@example.com",
		Role:     entity.RoleAdmin,
	}

	router := gin.New()
	router.GET("/me", func(c *gin.Context) {
		// Simulate auth middleware setting user_id
		c.Set("user_id", "user-1")
		handler.Me(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/me", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var user entity.User
	if err := json.Unmarshal(w.Body.Bytes(), &user); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Username = %q, want 'testuser'", user.Username)
	}
}

func TestAuthHandler_Me_NotAuthenticated(t *testing.T) {
	repo := newMockUserRepo()
	service := application.NewAuthService(repo, "test-secret")
	handler := NewAuthHandler(service)

	router := gin.New()
	router.GET("/me", handler.Me)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/me", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuthHandler_Me_UserNotFound(t *testing.T) {
	repo := newMockUserRepo()
	service := application.NewAuthService(repo, "test-secret")
	handler := NewAuthHandler(service)

	router := gin.New()
	router.GET("/me", func(c *gin.Context) {
		c.Set("user_id", "nonexistent")
		handler.Me(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/me", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestAuthHandler_Me_InvalidUserID(t *testing.T) {
	repo := newMockUserRepo()
	service := application.NewAuthService(repo, "test-secret")
	handler := NewAuthHandler(service)

	router := gin.New()
	router.GET("/me", func(c *gin.Context) {
		// Set invalid type for user_id
		c.Set("user_id", 123)
		handler.Me(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/me", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestAuthHandler_RegisterProtectedRoutes(t *testing.T) {
	repo := newMockUserRepo()
	service := application.NewAuthService(repo, "test-secret")
	handler := NewAuthHandler(service)

	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterProtectedRoutes(api)

	routes := router.Routes()
	found := false
	for _, route := range routes {
		if route.Path == "/api/v1/auth/me" && route.Method == "GET" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Route GET /api/v1/auth/me not found")
	}
}

func TestLoginRequest_Struct(t *testing.T) {
	req := LoginRequest{
		Username: "testuser",
		Password: "password123",
	}

	if req.Username != "testuser" {
		t.Errorf("Username = %q, want 'testuser'", req.Username)
	}
	if req.Password != "password123" {
		t.Errorf("Password = %q, want 'password123'", req.Password)
	}
}

func TestRefreshRequest_Struct(t *testing.T) {
	req := RefreshRequest{
		RefreshToken: "some-token",
	}

	if req.RefreshToken != "some-token" {
		t.Errorf("RefreshToken = %q, want 'some-token'", req.RefreshToken)
	}
}
