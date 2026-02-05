package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"autostrike/internal/application"
	"autostrike/internal/domain/entity"

	"github.com/gin-gonic/gin"
)

// mockNotificationRepoForHandler implements repository.NotificationRepository for handler tests
type mockNotificationRepoForHandler struct {
	settings      map[string]*entity.NotificationSettings
	notifications map[string]*entity.Notification
}

func newMockNotificationRepoForHandler() *mockNotificationRepoForHandler {
	return &mockNotificationRepoForHandler{
		settings:      make(map[string]*entity.NotificationSettings),
		notifications: make(map[string]*entity.Notification),
	}
}

func (m *mockNotificationRepoForHandler) CreateSettings(ctx context.Context, settings *entity.NotificationSettings) error {
	m.settings[settings.ID] = settings
	return nil
}

func (m *mockNotificationRepoForHandler) UpdateSettings(ctx context.Context, settings *entity.NotificationSettings) error {
	m.settings[settings.ID] = settings
	return nil
}

func (m *mockNotificationRepoForHandler) DeleteSettings(ctx context.Context, id string) error {
	delete(m.settings, id)
	return nil
}

func (m *mockNotificationRepoForHandler) FindSettingsByID(ctx context.Context, id string) (*entity.NotificationSettings, error) {
	if s, ok := m.settings[id]; ok {
		return s, nil
	}
	return nil, nil
}

func (m *mockNotificationRepoForHandler) FindSettingsByUserID(ctx context.Context, userID string) (*entity.NotificationSettings, error) {
	for _, s := range m.settings {
		if s.UserID == userID {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockNotificationRepoForHandler) FindAllEnabledSettings(ctx context.Context) ([]*entity.NotificationSettings, error) {
	var result []*entity.NotificationSettings
	for _, s := range m.settings {
		if s.Enabled {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockNotificationRepoForHandler) CreateNotification(ctx context.Context, notification *entity.Notification) error {
	m.notifications[notification.ID] = notification
	return nil
}

func (m *mockNotificationRepoForHandler) UpdateNotification(ctx context.Context, notification *entity.Notification) error {
	m.notifications[notification.ID] = notification
	return nil
}

func (m *mockNotificationRepoForHandler) FindNotificationByID(ctx context.Context, id string) (*entity.Notification, error) {
	if n, ok := m.notifications[id]; ok {
		return n, nil
	}
	return nil, nil
}

func (m *mockNotificationRepoForHandler) FindNotificationsByUserID(ctx context.Context, userID string, limit int) ([]*entity.Notification, error) {
	var result []*entity.Notification
	for _, n := range m.notifications {
		if n.UserID == userID {
			result = append(result, n)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockNotificationRepoForHandler) FindUnreadByUserID(ctx context.Context, userID string) ([]*entity.Notification, error) {
	var result []*entity.Notification
	for _, n := range m.notifications {
		if n.UserID == userID && !n.Read {
			result = append(result, n)
		}
	}
	return result, nil
}

func (m *mockNotificationRepoForHandler) MarkAsRead(ctx context.Context, id string) error {
	if n, ok := m.notifications[id]; ok {
		n.Read = true
	}
	return nil
}

func (m *mockNotificationRepoForHandler) MarkAllAsRead(ctx context.Context, userID string) error {
	for _, n := range m.notifications {
		if n.UserID == userID {
			n.Read = true
		}
	}
	return nil
}

// mockUserRepoForNotificationHandler implements repository.UserRepository for notification handler tests
type mockUserRepoForNotificationHandler struct{}

func (m *mockUserRepoForNotificationHandler) Create(ctx context.Context, user *entity.User) error {
	return nil
}

func (m *mockUserRepoForNotificationHandler) Update(ctx context.Context, user *entity.User) error {
	return nil
}

func (m *mockUserRepoForNotificationHandler) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockUserRepoForNotificationHandler) FindByID(ctx context.Context, id string) (*entity.User, error) {
	return nil, nil
}

func (m *mockUserRepoForNotificationHandler) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	return nil, nil
}

func (m *mockUserRepoForNotificationHandler) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	return nil, nil
}

func (m *mockUserRepoForNotificationHandler) FindAll(ctx context.Context) ([]*entity.User, error) {
	return nil, nil
}

func (m *mockUserRepoForNotificationHandler) FindActive(ctx context.Context) ([]*entity.User, error) {
	return nil, nil
}

func (m *mockUserRepoForNotificationHandler) UpdateLastLogin(ctx context.Context, id string) error {
	return nil
}

func (m *mockUserRepoForNotificationHandler) Deactivate(ctx context.Context, id string) error {
	return nil
}

func (m *mockUserRepoForNotificationHandler) Reactivate(ctx context.Context, id string) error {
	return nil
}

func (m *mockUserRepoForNotificationHandler) CountByRole(ctx context.Context, role entity.UserRole) (int, error) {
	return 0, nil
}

func (m *mockUserRepoForNotificationHandler) DeactivateAdminIfNotLast(ctx context.Context, id string) error {
	return nil
}

func setupNotificationHandler() (*NotificationHandler, *mockNotificationRepoForHandler) {
	repo := newMockNotificationRepoForHandler()
	userRepo := &mockUserRepoForNotificationHandler{}
	service := application.NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)
	handler := NewNotificationHandler(service)
	return handler, repo
}

func setupNotificationRouter(handler *NotificationHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Middleware to set user_id for authenticated routes
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user-id")
		c.Next()
	})

	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)
	return router
}

func setupNotificationRouterWithAdmin(handler *NotificationHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Middleware to set user_id and admin role for admin routes
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user-id")
		c.Set("role", "admin")
		c.Next()
	})

	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)
	return router
}

func TestNewNotificationHandler(t *testing.T) {
	handler, _ := setupNotificationHandler()
	if handler == nil {
		t.Fatal("NewNotificationHandler returned nil")
	}
}

func TestNotificationHandler_RegisterRoutes(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	routes := router.Routes()
	expectedPaths := map[string]string{
		"/api/v1/notifications":              "GET",
		"/api/v1/notifications/unread/count": "GET",
		"/api/v1/notifications/:id/read":     "POST",
		"/api/v1/notifications/read-all":     "POST",
		"/api/v1/notifications/settings":     "GET",
		"/api/v1/notifications/smtp":         "GET",
		"/api/v1/notifications/smtp/test":    "POST",
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

func TestNotificationHandler_GetNotifications(t *testing.T) {
	handler, repo := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	// Add a notification
	repo.notifications["notif-1"] = &entity.Notification{
		ID:        "notif-1",
		UserID:    "test-user-id",
		Type:      entity.NotificationExecutionStarted,
		Title:     "Test",
		Message:   "Test message",
		CreatedAt: time.Now(),
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var notifications []*entity.Notification
	if err := json.Unmarshal(w.Body.Bytes(), &notifications); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(notifications) != 1 {
		t.Errorf("len(notifications) = %d, want 1", len(notifications))
	}
}

func TestNotificationHandler_GetNotifications_WithLimit(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications?limit=10", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestNotificationHandler_GetNotifications_Unauthenticated(t *testing.T) {
	handler, _ := setupNotificationHandler()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	// No middleware setting user_id
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestNotificationHandler_GetUnreadCount(t *testing.T) {
	handler, repo := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	// Add unread notifications
	repo.notifications["notif-1"] = &entity.Notification{
		ID: "notif-1", UserID: "test-user-id", Read: false,
	}
	repo.notifications["notif-2"] = &entity.Notification{
		ID: "notif-2", UserID: "test-user-id", Read: false,
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications/unread/count", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]int
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["count"] != 2 {
		t.Errorf("count = %d, want 2", response["count"])
	}
}

func TestNotificationHandler_MarkAsRead(t *testing.T) {
	handler, repo := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	repo.notifications["notif-1"] = &entity.Notification{
		ID: "notif-1", UserID: "test-user-id", Read: false,
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/notif-1/read", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	if !repo.notifications["notif-1"].Read {
		t.Error("Notification should be marked as read")
	}
}

func TestNotificationHandler_MarkAllAsRead(t *testing.T) {
	handler, repo := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	repo.notifications["notif-1"] = &entity.Notification{
		ID: "notif-1", UserID: "test-user-id", Read: false,
	}
	repo.notifications["notif-2"] = &entity.Notification{
		ID: "notif-2", UserID: "test-user-id", Read: false,
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/read-all", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	for _, n := range repo.notifications {
		if !n.Read {
			t.Error("All notifications should be marked as read")
		}
	}
}

func TestNotificationHandler_GetSettings(t *testing.T) {
	handler, repo := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	repo.settings["settings-1"] = &entity.NotificationSettings{
		ID:           "settings-1",
		UserID:       "test-user-id",
		Channel:      entity.ChannelEmail,
		Enabled:      true,
		EmailAddress: "test@example.com",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications/settings", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var settings entity.NotificationSettings
	if err := json.Unmarshal(w.Body.Bytes(), &settings); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if settings.EmailAddress != "test@example.com" {
		t.Errorf("EmailAddress = %q, want %q", settings.EmailAddress, "test@example.com")
	}
}

func TestNotificationHandler_GetSettings_NotFound(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications/settings", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestNotificationHandler_CreateSettings(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	body := NotificationSettingsRequest{
		Channel:          "email",
		Enabled:          true,
		EmailAddress:     "test@example.com",
		NotifyOnStart:    true,
		NotifyOnComplete: true,
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotificationHandler_CreateSettings_InvalidChannel(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	body := map[string]interface{}{
		"channel": "invalid",
		"enabled": true,
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestNotificationHandler_UpdateSettings(t *testing.T) {
	handler, repo := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	repo.settings["settings-1"] = &entity.NotificationSettings{
		ID:           "settings-1",
		UserID:       "test-user-id",
		Channel:      entity.ChannelEmail,
		Enabled:      true,
		EmailAddress: "old@example.com",
	}

	body := NotificationSettingsRequest{
		Channel:      "email",
		Enabled:      true,
		EmailAddress: "new@example.com",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/notifications/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotificationHandler_UpdateSettings_NotFound(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	body := NotificationSettingsRequest{
		Channel:      "email",
		Enabled:      true,
		EmailAddress: "test@example.com",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/notifications/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestNotificationHandler_DeleteSettings(t *testing.T) {
	handler, repo := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	repo.settings["settings-1"] = &entity.NotificationSettings{
		ID:     "settings-1",
		UserID: "test-user-id",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/notifications/settings", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotificationHandler_DeleteSettings_NotFound(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/notifications/settings", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestNotificationHandler_GetSMTPConfig_NotConfigured(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouterWithAdmin(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications/smtp", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestNotificationHandler_TestSMTP_InvalidEmail(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouterWithAdmin(handler)

	body := TestSMTPRequest{
		Email: "invalid-email",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/smtp/test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestNotificationHandler_TestSMTP_NotConfigured(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouterWithAdmin(handler)

	body := TestSMTPRequest{
		Email: "test@example.com",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/smtp/test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestNotificationHandler_GetUnreadCount_Unauthenticated(t *testing.T) {
	handler, _ := setupNotificationHandler()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications/unread/count", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestNotificationHandler_MarkAsRead_Unauthenticated(t *testing.T) {
	handler, _ := setupNotificationHandler()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/notif-1/read", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestNotificationHandler_MarkAsRead_NotOwned(t *testing.T) {
	handler, repo := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	// Add notification owned by different user
	repo.notifications["notif-1"] = &entity.Notification{
		ID: "notif-1", UserID: "other-user", Read: false,
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/notif-1/read", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotificationHandler_MarkAsRead_NotFound(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/nonexistent/read", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestNotificationHandler_MarkAllAsRead_Unauthenticated(t *testing.T) {
	handler, _ := setupNotificationHandler()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/read-all", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestNotificationHandler_GetSettings_Unauthenticated(t *testing.T) {
	handler, _ := setupNotificationHandler()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications/settings", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestNotificationHandler_CreateSettings_Unauthenticated(t *testing.T) {
	handler, _ := setupNotificationHandler()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	body := NotificationSettingsRequest{Channel: "email", Enabled: true, EmailAddress: "test@example.com"}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestNotificationHandler_CreateSettings_MissingEmail(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	body := NotificationSettingsRequest{
		Channel: "email",
		Enabled: true,
		// Missing EmailAddress
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestNotificationHandler_CreateSettings_InvalidEmail(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	body := NotificationSettingsRequest{
		Channel:      "email",
		Enabled:      true,
		EmailAddress: "not-an-email",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestNotificationHandler_CreateSettings_MissingWebhookURL(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	body := NotificationSettingsRequest{
		Channel: "webhook",
		Enabled: true,
		// Missing WebhookURL
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestNotificationHandler_CreateSettings_InvalidWebhookURL(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	body := NotificationSettingsRequest{
		Channel:    "webhook",
		Enabled:    true,
		WebhookURL: "not-a-url",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestNotificationHandler_UpdateSettings_Unauthenticated(t *testing.T) {
	handler, _ := setupNotificationHandler()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	body := NotificationSettingsRequest{Channel: "email", Enabled: true, EmailAddress: "test@example.com"}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/notifications/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestNotificationHandler_UpdateSettings_InvalidJSON(t *testing.T) {
	handler, repo := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	repo.settings["settings-1"] = &entity.NotificationSettings{
		ID:     "settings-1",
		UserID: "test-user-id",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/notifications/settings", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestNotificationHandler_UpdateSettings_ValidationError(t *testing.T) {
	handler, repo := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	repo.settings["settings-1"] = &entity.NotificationSettings{
		ID:     "settings-1",
		UserID: "test-user-id",
	}

	body := NotificationSettingsRequest{
		Channel:      "email",
		Enabled:      true,
		EmailAddress: "invalid-email",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/notifications/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestNotificationHandler_DeleteSettings_Unauthenticated(t *testing.T) {
	handler, _ := setupNotificationHandler()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/notifications/settings", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestNotificationHandler_GetSMTPConfig_Unauthenticated(t *testing.T) {
	handler, _ := setupNotificationHandler()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications/smtp", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestNotificationHandler_GetSMTPConfig_NotAdmin(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouter(handler) // No admin role

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications/smtp", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestNotificationHandler_TestSMTP_Unauthenticated(t *testing.T) {
	handler, _ := setupNotificationHandler()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	body := TestSMTPRequest{Email: "test@example.com"}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/smtp/test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestNotificationHandler_TestSMTP_NotAdmin(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouter(handler) // No admin role

	body := TestSMTPRequest{Email: "test@example.com"}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/smtp/test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestNotificationHandler_GetNotifications_InvalidLimit(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	// Invalid limit (too high)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications?limit=200", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestNotificationHandler_GetNotifications_NegativeLimit(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications?limit=-5", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestNotificationHandler_GetNotifications_NonNumericLimit(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications?limit=abc", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestNotificationHandler_CreateSettings_WebhookSuccess(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	body := NotificationSettingsRequest{
		Channel:      "webhook",
		Enabled:      true,
		WebhookURL:   "https://hooks.example.com/webhook",
		NotifyOnStart: true,
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/settings", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotificationHandler_CreateSettings_InvalidJSON(t *testing.T) {
	handler, _ := setupNotificationHandler()
	router := setupNotificationRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/settings", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestNotificationSettingsRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     NotificationSettingsRequest
		wantErr bool
	}{
		{
			name:    "email disabled no address",
			req:     NotificationSettingsRequest{Channel: "email", Enabled: false},
			wantErr: false,
		},
		{
			name:    "email enabled with valid address",
			req:     NotificationSettingsRequest{Channel: "email", Enabled: true, EmailAddress: "test@example.com"},
			wantErr: false,
		},
		{
			name:    "email enabled missing address",
			req:     NotificationSettingsRequest{Channel: "email", Enabled: true, EmailAddress: ""},
			wantErr: true,
		},
		{
			name:    "webhook disabled no url",
			req:     NotificationSettingsRequest{Channel: "webhook", Enabled: false},
			wantErr: false,
		},
		{
			name:    "webhook enabled with valid url",
			req:     NotificationSettingsRequest{Channel: "webhook", Enabled: true, WebhookURL: "https://example.com/hook"},
			wantErr: false,
		},
		{
			name:    "webhook enabled missing url",
			req:     NotificationSettingsRequest{Channel: "webhook", Enabled: true, WebhookURL: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"10", 10, false},
		{"0", 0, false},
		{"-5", -5, false},
		{"abc", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseInt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseInt(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseInt(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
