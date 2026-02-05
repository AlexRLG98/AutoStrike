package application

import (
	"context"
	"testing"
	"time"

	"autostrike/internal/domain/entity"
)

// mockNotificationRepo implements repository.NotificationRepository for tests
type mockNotificationRepo struct {
	settings      map[string]*entity.NotificationSettings
	notifications map[string]*entity.Notification
}

func newMockNotificationRepo() *mockNotificationRepo {
	return &mockNotificationRepo{
		settings:      make(map[string]*entity.NotificationSettings),
		notifications: make(map[string]*entity.Notification),
	}
}

func (m *mockNotificationRepo) CreateSettings(ctx context.Context, settings *entity.NotificationSettings) error {
	m.settings[settings.ID] = settings
	return nil
}

func (m *mockNotificationRepo) UpdateSettings(ctx context.Context, settings *entity.NotificationSettings) error {
	m.settings[settings.ID] = settings
	return nil
}

func (m *mockNotificationRepo) DeleteSettings(ctx context.Context, id string) error {
	delete(m.settings, id)
	return nil
}

func (m *mockNotificationRepo) FindSettingsByID(ctx context.Context, id string) (*entity.NotificationSettings, error) {
	if s, ok := m.settings[id]; ok {
		return s, nil
	}
	return nil, nil
}

func (m *mockNotificationRepo) FindSettingsByUserID(ctx context.Context, userID string) (*entity.NotificationSettings, error) {
	for _, s := range m.settings {
		if s.UserID == userID {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockNotificationRepo) FindAllEnabledSettings(ctx context.Context) ([]*entity.NotificationSettings, error) {
	var result []*entity.NotificationSettings
	for _, s := range m.settings {
		if s.Enabled {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockNotificationRepo) CreateNotification(ctx context.Context, notification *entity.Notification) error {
	m.notifications[notification.ID] = notification
	return nil
}

func (m *mockNotificationRepo) UpdateNotification(ctx context.Context, notification *entity.Notification) error {
	m.notifications[notification.ID] = notification
	return nil
}

func (m *mockNotificationRepo) FindNotificationByID(ctx context.Context, id string) (*entity.Notification, error) {
	if n, ok := m.notifications[id]; ok {
		return n, nil
	}
	return nil, nil
}

func (m *mockNotificationRepo) FindNotificationsByUserID(ctx context.Context, userID string, limit int) ([]*entity.Notification, error) {
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

func (m *mockNotificationRepo) FindUnreadByUserID(ctx context.Context, userID string) ([]*entity.Notification, error) {
	var result []*entity.Notification
	for _, n := range m.notifications {
		if n.UserID == userID && !n.Read {
			result = append(result, n)
		}
	}
	return result, nil
}

func (m *mockNotificationRepo) MarkAsRead(ctx context.Context, id string) error {
	if n, ok := m.notifications[id]; ok {
		n.Read = true
	}
	return nil
}

func (m *mockNotificationRepo) MarkAllAsRead(ctx context.Context, userID string) error {
	for _, n := range m.notifications {
		if n.UserID == userID {
			n.Read = true
		}
	}
	return nil
}

// mockUserRepoForNotification implements repository.UserRepository for notification tests
type mockUserRepoForNotification struct{}

func (m *mockUserRepoForNotification) Create(ctx context.Context, user *entity.User) error {
	return nil
}

func (m *mockUserRepoForNotification) Update(ctx context.Context, user *entity.User) error {
	return nil
}

func (m *mockUserRepoForNotification) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockUserRepoForNotification) FindByID(ctx context.Context, id string) (*entity.User, error) {
	return nil, nil
}

func (m *mockUserRepoForNotification) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	return nil, nil
}

func (m *mockUserRepoForNotification) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	return nil, nil
}

func (m *mockUserRepoForNotification) FindAll(ctx context.Context) ([]*entity.User, error) {
	return nil, nil
}

func (m *mockUserRepoForNotification) FindActive(ctx context.Context) ([]*entity.User, error) {
	return nil, nil
}

func (m *mockUserRepoForNotification) UpdateLastLogin(ctx context.Context, id string) error {
	return nil
}

func (m *mockUserRepoForNotification) Deactivate(ctx context.Context, id string) error {
	return nil
}

func (m *mockUserRepoForNotification) Reactivate(ctx context.Context, id string) error {
	return nil
}

func (m *mockUserRepoForNotification) CountByRole(ctx context.Context, role entity.UserRole) (int, error) {
	return 0, nil
}

func (m *mockUserRepoForNotification) DeactivateAdminIfNotLast(ctx context.Context, id string) error {
	return nil
}

func TestNewNotificationService(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	if service == nil {
		t.Fatal("NewNotificationService returned nil")
	}
}

func TestNotificationService_CreateSettings(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	settings := &entity.NotificationSettings{
		UserID:           "user-1",
		Channel:          entity.ChannelEmail,
		Enabled:          true,
		EmailAddress:     "test@example.com",
		NotifyOnStart:    true,
		NotifyOnComplete: true,
		NotifyOnFailure:  true,
	}

	err := service.CreateSettings(context.Background(), settings)
	if err != nil {
		t.Fatalf("CreateSettings failed: %v", err)
	}

	if settings.ID == "" {
		t.Error("Settings ID should be set")
	}

	if settings.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestNotificationService_UpdateSettings(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	settings := &entity.NotificationSettings{
		ID:           "settings-1",
		UserID:       "user-1",
		Channel:      entity.ChannelEmail,
		Enabled:      true,
		EmailAddress: "test@example.com",
	}
	repo.settings[settings.ID] = settings

	settings.EmailAddress = "new@example.com"
	err := service.UpdateSettings(context.Background(), settings)
	if err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	if settings.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestNotificationService_GetSettingsByUserID(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	settings := &entity.NotificationSettings{
		ID:           "settings-1",
		UserID:       "user-1",
		Channel:      entity.ChannelEmail,
		Enabled:      true,
		EmailAddress: "test@example.com",
	}
	repo.settings[settings.ID] = settings

	result, err := service.GetSettingsByUserID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetSettingsByUserID failed: %v", err)
	}

	if result == nil {
		t.Fatal("GetSettingsByUserID returned nil")
	}

	if result.EmailAddress != "test@example.com" {
		t.Errorf("EmailAddress = %q, want %q", result.EmailAddress, "test@example.com")
	}
}

func TestNotificationService_DeleteSettings(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	settings := &entity.NotificationSettings{
		ID:     "settings-1",
		UserID: "user-1",
	}
	repo.settings[settings.ID] = settings

	err := service.DeleteSettings(context.Background(), settings.ID)
	if err != nil {
		t.Fatalf("DeleteSettings failed: %v", err)
	}

	if _, ok := repo.settings[settings.ID]; ok {
		t.Error("Settings should be deleted")
	}
}

func TestNotificationService_GetNotificationsByUserID(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	notification := &entity.Notification{
		ID:        "notif-1",
		UserID:    "user-1",
		Type:      entity.NotificationExecutionStarted,
		Title:     "Test",
		Message:   "Test message",
		CreatedAt: time.Now(),
	}
	repo.notifications[notification.ID] = notification

	result, err := service.GetNotificationsByUserID(context.Background(), "user-1", 50)
	if err != nil {
		t.Fatalf("GetNotificationsByUserID failed: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("len(result) = %d, want 1", len(result))
	}
}

func TestNotificationService_GetUnreadCount(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	// Add 3 notifications, 2 unread
	repo.notifications["notif-1"] = &entity.Notification{
		ID: "notif-1", UserID: "user-1", Read: false,
	}
	repo.notifications["notif-2"] = &entity.Notification{
		ID: "notif-2", UserID: "user-1", Read: false,
	}
	repo.notifications["notif-3"] = &entity.Notification{
		ID: "notif-3", UserID: "user-1", Read: true,
	}

	count, err := service.GetUnreadCount(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetUnreadCount failed: %v", err)
	}

	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestNotificationService_MarkAsRead(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	notification := &entity.Notification{
		ID:     "notif-1",
		UserID: "user-1",
		Read:   false,
	}
	repo.notifications[notification.ID] = notification

	err := service.MarkAsRead(context.Background(), notification.ID)
	if err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}

	if !repo.notifications[notification.ID].Read {
		t.Error("Notification should be marked as read")
	}
}

func TestNotificationService_MarkAllAsRead(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	repo.notifications["notif-1"] = &entity.Notification{
		ID: "notif-1", UserID: "user-1", Read: false,
	}
	repo.notifications["notif-2"] = &entity.Notification{
		ID: "notif-2", UserID: "user-1", Read: false,
	}

	err := service.MarkAllAsRead(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("MarkAllAsRead failed: %v", err)
	}

	for _, n := range repo.notifications {
		if !n.Read {
			t.Error("All notifications should be marked as read")
		}
	}
}

func TestNotificationService_GetSMTPConfig(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}

	// Test with nil SMTP config
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)
	if service.GetSMTPConfig() != nil {
		t.Error("GetSMTPConfig should return nil when not configured")
	}

	// Test with SMTP config
	smtpConfig := &entity.SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "secret",
		From:     "noreply@example.com",
		UseTLS:   true,
	}
	service = NewNotificationService(repo, userRepo, smtpConfig, "https://localhost:8443", nil)

	config := service.GetSMTPConfig()
	if config == nil {
		t.Fatal("GetSMTPConfig should not return nil")
	}

	if config.Password != "" {
		t.Error("Password should not be exposed")
	}

	if config.Host != "smtp.example.com" {
		t.Errorf("Host = %q, want %q", config.Host, "smtp.example.com")
	}
}

func TestNotificationService_SetSMTPConfig(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	smtpConfig := &entity.SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "noreply@example.com",
	}

	service.SetSMTPConfig(smtpConfig)

	config := service.GetSMTPConfig()
	if config == nil {
		t.Fatal("GetSMTPConfig should not return nil after SetSMTPConfig")
	}
}

func TestNotificationService_NotifyExecutionStarted(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	// Add enabled settings with NotifyOnStart
	settings := &entity.NotificationSettings{
		ID:            "settings-1",
		UserID:        "user-1",
		Channel:       entity.ChannelEmail,
		Enabled:       true,
		NotifyOnStart: true,
	}
	repo.settings[settings.ID] = settings

	execution := &entity.Execution{
		ID:        "exec-1",
		StartedAt: time.Now(),
		SafeMode:  true,
	}

	err := service.NotifyExecutionStarted(context.Background(), execution, "Test Scenario")
	if err != nil {
		t.Fatalf("NotifyExecutionStarted failed: %v", err)
	}

	// Should have created a notification
	if len(repo.notifications) != 1 {
		t.Errorf("len(notifications) = %d, want 1", len(repo.notifications))
	}
}

func TestNotificationService_NotifyExecutionCompleted(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	// Add enabled settings with NotifyOnComplete
	settings := &entity.NotificationSettings{
		ID:               "settings-1",
		UserID:           "user-1",
		Channel:          entity.ChannelEmail,
		Enabled:          true,
		NotifyOnComplete: true,
	}
	repo.settings[settings.ID] = settings

	execution := &entity.Execution{
		ID:        "exec-1",
		StartedAt: time.Now(),
		Score: &entity.SecurityScore{
			Overall:    85.0,
			Blocked:    4,
			Detected:   1,
			Successful: 1,
			Total:      6,
		},
	}

	err := service.NotifyExecutionCompleted(context.Background(), execution, "Test Scenario")
	if err != nil {
		t.Fatalf("NotifyExecutionCompleted failed: %v", err)
	}

	if len(repo.notifications) != 1 {
		t.Errorf("len(notifications) = %d, want 1", len(repo.notifications))
	}
}

func TestNotificationService_NotifyExecutionCompleted_WithScoreAlert(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	// Add enabled settings with NotifyOnComplete and score alert
	settings := &entity.NotificationSettings{
		ID:                  "settings-1",
		UserID:              "user-1",
		Channel:             entity.ChannelEmail,
		Enabled:             true,
		NotifyOnComplete:    true,
		NotifyOnScoreAlert:  true,
		ScoreAlertThreshold: 70.0,
	}
	repo.settings[settings.ID] = settings

	execution := &entity.Execution{
		ID:        "exec-1",
		StartedAt: time.Now(),
		Score: &entity.SecurityScore{
			Overall:    50.0, // Below threshold
			Blocked:    2,
			Detected:   1,
			Successful: 3,
			Total:      6,
		},
	}

	err := service.NotifyExecutionCompleted(context.Background(), execution, "Test Scenario")
	if err != nil {
		t.Fatalf("NotifyExecutionCompleted failed: %v", err)
	}

	// Should have created 2 notifications: completion + score alert
	if len(repo.notifications) != 2 {
		t.Errorf("len(notifications) = %d, want 2", len(repo.notifications))
	}
}

func TestNotificationService_NotifyExecutionFailed(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	// Add enabled settings with NotifyOnFailure
	settings := &entity.NotificationSettings{
		ID:              "settings-1",
		UserID:          "user-1",
		Channel:         entity.ChannelEmail,
		Enabled:         true,
		NotifyOnFailure: true,
	}
	repo.settings[settings.ID] = settings

	execution := &entity.Execution{
		ID:        "exec-1",
		StartedAt: time.Now(),
	}

	err := service.NotifyExecutionFailed(context.Background(), execution, "Test Scenario", "Connection timeout")
	if err != nil {
		t.Fatalf("NotifyExecutionFailed failed: %v", err)
	}

	if len(repo.notifications) != 1 {
		t.Errorf("len(notifications) = %d, want 1", len(repo.notifications))
	}
}

func TestNotificationService_NotifyAgentOffline(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	// Add enabled settings with NotifyOnAgentOffline
	settings := &entity.NotificationSettings{
		ID:                   "settings-1",
		UserID:               "user-1",
		Channel:              entity.ChannelEmail,
		Enabled:              true,
		NotifyOnAgentOffline: true,
	}
	repo.settings[settings.ID] = settings

	agent := &entity.Agent{
		Paw:      "test-paw",
		Hostname: "test-host",
		Platform: "linux",
		LastSeen: time.Now(),
	}

	err := service.NotifyAgentOffline(context.Background(), agent)
	if err != nil {
		t.Fatalf("NotifyAgentOffline failed: %v", err)
	}

	if len(repo.notifications) != 1 {
		t.Errorf("len(notifications) = %d, want 1", len(repo.notifications))
	}
}

func TestNotificationService_TestSMTPConnection_NotConfigured(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	err := service.TestSMTPConnection(context.Background(), "test@example.com")
	if err == nil {
		t.Error("TestSMTPConnection should fail when SMTP not configured")
	}
}

func TestNotificationService_GetNotificationsByUserID_DefaultLimit(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	// Test with limit <= 0 (should default to 50)
	_, err := service.GetNotificationsByUserID(context.Background(), "user-1", 0)
	if err != nil {
		t.Fatalf("GetNotificationsByUserID failed: %v", err)
	}

	_, err = service.GetNotificationsByUserID(context.Background(), "user-1", -1)
	if err != nil {
		t.Fatalf("GetNotificationsByUserID failed: %v", err)
	}
}

func TestNotificationService_MarkAsReadForUser_Success(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	notification := &entity.Notification{
		ID:     "notif-1",
		UserID: "user-1",
		Read:   false,
	}
	repo.notifications[notification.ID] = notification

	err := service.MarkAsReadForUser(context.Background(), "notif-1", "user-1")
	if err != nil {
		t.Fatalf("MarkAsReadForUser failed: %v", err)
	}

	if !repo.notifications["notif-1"].Read {
		t.Error("Notification should be marked as read")
	}
}

func TestNotificationService_MarkAsReadForUser_NotOwned(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	notification := &entity.Notification{
		ID:     "notif-1",
		UserID: "user-1",
		Read:   false,
	}
	repo.notifications[notification.ID] = notification

	// Try to mark as read with different user
	err := service.MarkAsReadForUser(context.Background(), "notif-1", "user-2")
	if err == nil {
		t.Error("MarkAsReadForUser should fail when user doesn't own notification")
	}
}

func TestNotificationService_MarkAsReadForUser_NotFound(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	err := service.MarkAsReadForUser(context.Background(), "nonexistent", "user-1")
	if err == nil {
		t.Error("MarkAsReadForUser should fail when notification not found")
	}
}

func TestNotificationService_NotifyExecutionCompleted_NilScore(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	settings := &entity.NotificationSettings{
		ID:               "settings-1",
		UserID:           "user-1",
		Channel:          entity.ChannelEmail,
		Enabled:          true,
		NotifyOnComplete: true,
	}
	repo.settings[settings.ID] = settings

	execution := &entity.Execution{
		ID:        "exec-1",
		StartedAt: time.Now(),
		Score:     nil, // nil score
	}

	err := service.NotifyExecutionCompleted(context.Background(), execution, "Test Scenario")
	if err != nil {
		t.Fatalf("NotifyExecutionCompleted failed: %v", err)
	}

	if len(repo.notifications) != 1 {
		t.Errorf("len(notifications) = %d, want 1", len(repo.notifications))
	}
}

func TestNotificationService_NotifyExecutionStarted_NotifyDisabled(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	// Settings with NotifyOnStart = false
	settings := &entity.NotificationSettings{
		ID:            "settings-1",
		UserID:        "user-1",
		Channel:       entity.ChannelEmail,
		Enabled:       true,
		NotifyOnStart: false,
	}
	repo.settings[settings.ID] = settings

	execution := &entity.Execution{
		ID:        "exec-1",
		StartedAt: time.Now(),
	}

	err := service.NotifyExecutionStarted(context.Background(), execution, "Test Scenario")
	if err != nil {
		t.Fatalf("NotifyExecutionStarted failed: %v", err)
	}

	// Should not create notification
	if len(repo.notifications) != 0 {
		t.Errorf("len(notifications) = %d, want 0", len(repo.notifications))
	}
}

func TestNotificationService_NotifyExecutionCompleted_NotifyDisabled(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	settings := &entity.NotificationSettings{
		ID:               "settings-1",
		UserID:           "user-1",
		Channel:          entity.ChannelEmail,
		Enabled:          true,
		NotifyOnComplete: false,
	}
	repo.settings[settings.ID] = settings

	execution := &entity.Execution{
		ID:        "exec-1",
		StartedAt: time.Now(),
	}

	err := service.NotifyExecutionCompleted(context.Background(), execution, "Test Scenario")
	if err != nil {
		t.Fatalf("NotifyExecutionCompleted failed: %v", err)
	}

	if len(repo.notifications) != 0 {
		t.Errorf("len(notifications) = %d, want 0", len(repo.notifications))
	}
}

func TestNotificationService_NotifyExecutionFailed_NotifyDisabled(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	settings := &entity.NotificationSettings{
		ID:              "settings-1",
		UserID:          "user-1",
		Channel:         entity.ChannelEmail,
		Enabled:         true,
		NotifyOnFailure: false,
	}
	repo.settings[settings.ID] = settings

	execution := &entity.Execution{
		ID:        "exec-1",
		StartedAt: time.Now(),
	}

	err := service.NotifyExecutionFailed(context.Background(), execution, "Test Scenario", "Error")
	if err != nil {
		t.Fatalf("NotifyExecutionFailed failed: %v", err)
	}

	if len(repo.notifications) != 0 {
		t.Errorf("len(notifications) = %d, want 0", len(repo.notifications))
	}
}

func TestNotificationService_NotifyAgentOffline_NotifyDisabled(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	settings := &entity.NotificationSettings{
		ID:                   "settings-1",
		UserID:               "user-1",
		Channel:              entity.ChannelEmail,
		Enabled:              true,
		NotifyOnAgentOffline: false,
	}
	repo.settings[settings.ID] = settings

	agent := &entity.Agent{
		Paw:      "test-paw",
		Hostname: "test-host",
		Platform: "linux",
		LastSeen: time.Now(),
	}

	err := service.NotifyAgentOffline(context.Background(), agent)
	if err != nil {
		t.Fatalf("NotifyAgentOffline failed: %v", err)
	}

	if len(repo.notifications) != 0 {
		t.Errorf("len(notifications) = %d, want 0", len(repo.notifications))
	}
}

func TestNotificationService_NotifyExecutionCompleted_ScoreAboveThreshold(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	settings := &entity.NotificationSettings{
		ID:                  "settings-1",
		UserID:              "user-1",
		Channel:             entity.ChannelEmail,
		Enabled:             true,
		NotifyOnComplete:    true,
		NotifyOnScoreAlert:  true,
		ScoreAlertThreshold: 70.0,
	}
	repo.settings[settings.ID] = settings

	execution := &entity.Execution{
		ID:        "exec-1",
		StartedAt: time.Now(),
		Score: &entity.SecurityScore{
			Overall: 85.0, // Above threshold
		},
	}

	err := service.NotifyExecutionCompleted(context.Background(), execution, "Test Scenario")
	if err != nil {
		t.Fatalf("NotifyExecutionCompleted failed: %v", err)
	}

	// Should have only 1 notification (no score alert)
	if len(repo.notifications) != 1 {
		t.Errorf("len(notifications) = %d, want 1", len(repo.notifications))
	}
}

func TestBuildExecutionCompletedData(t *testing.T) {
	execution := &entity.Execution{
		ID: "exec-1",
		Score: &entity.SecurityScore{
			Overall:    75.5,
			Blocked:    3,
			Detected:   2,
			Successful: 1,
			Total:      6,
		},
	}

	data, score := buildExecutionCompletedData(execution, "Test Scenario", "https://dashboard")

	if score != 75.5 {
		t.Errorf("score = %v, want 75.5", score)
	}

	if data["ScenarioName"] != "Test Scenario" {
		t.Errorf("ScenarioName = %v, want Test Scenario", data["ScenarioName"])
	}

	if data["Blocked"] != 3 {
		t.Errorf("Blocked = %v, want 3", data["Blocked"])
	}
}

func TestBuildExecutionCompletedData_NilScore(t *testing.T) {
	execution := &entity.Execution{
		ID:    "exec-1",
		Score: nil,
	}

	data, score := buildExecutionCompletedData(execution, "Test Scenario", "https://dashboard")

	if score != 0.0 {
		t.Errorf("score = %v, want 0.0", score)
	}

	if data["Blocked"] != 0 {
		t.Errorf("Blocked = %v, want 0", data["Blocked"])
	}
}

func TestShouldSendEmail(t *testing.T) {
	tests := []struct {
		name     string
		settings *entity.NotificationSettings
		want     bool
	}{
		{
			name: "email channel with address",
			settings: &entity.NotificationSettings{
				Channel:      entity.ChannelEmail,
				EmailAddress: "test@example.com",
			},
			want: true,
		},
		{
			name: "email channel without address",
			settings: &entity.NotificationSettings{
				Channel:      entity.ChannelEmail,
				EmailAddress: "",
			},
			want: false,
		},
		{
			name: "webhook channel",
			settings: &entity.NotificationSettings{
				Channel:      entity.ChannelWebhook,
				EmailAddress: "test@example.com",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSendEmail(tt.settings)
			if got != tt.want {
				t.Errorf("shouldSendEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildEmailMessage(t *testing.T) {
	msg := buildEmailMessage("from@example.com", "to@example.com", "Test Subject", "Test Body")

	if !contains(msg, "From: from@example.com") {
		t.Error("Message should contain From header")
	}
	if !contains(msg, "To: to@example.com") {
		t.Error("Message should contain To header")
	}
	if !contains(msg, "Subject: Test Subject") {
		t.Error("Message should contain Subject header")
	}
	if !contains(msg, "Test Body") {
		t.Error("Message should contain body")
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

func TestRenderEmailTemplate(t *testing.T) {
	tmpl := "Hello {{.Name}}, your score is {{.Score}}%"
	data := map[string]any{
		"Name":  "User",
		"Score": 85,
	}

	result, err := renderEmailTemplate(tmpl, data)
	if err != nil {
		t.Fatalf("renderEmailTemplate failed: %v", err)
	}

	expected := "Hello User, your score is 85%"
	if result != expected {
		t.Errorf("result = %q, want %q", result, expected)
	}
}

func TestRenderEmailTemplate_InvalidTemplate(t *testing.T) {
	tmpl := "Hello {{.Name"
	data := map[string]any{"Name": "User"}

	_, err := renderEmailTemplate(tmpl, data)
	if err == nil {
		t.Error("renderEmailTemplate should fail with invalid template")
	}
}

func TestRenderEmailTemplate_ExecutionError(t *testing.T) {
	// Template that references missing field in a way that causes execution error
	tmpl := "Hello {{.Name.Invalid}}"
	data := map[string]any{"Name": "User"}

	_, err := renderEmailTemplate(tmpl, data)
	if err == nil {
		t.Error("renderEmailTemplate should fail with execution error")
	}
}

func TestNotificationService_NotifyExecutionStarted_NoSettings(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	// No settings - should not create any notifications
	execution := &entity.Execution{
		ID:        "exec-1",
		StartedAt: time.Now(),
	}

	err := service.NotifyExecutionStarted(context.Background(), execution, "Test Scenario")
	if err != nil {
		t.Fatalf("NotifyExecutionStarted failed: %v", err)
	}

	if len(repo.notifications) != 0 {
		t.Errorf("len(notifications) = %d, want 0", len(repo.notifications))
	}
}

func TestNotificationService_NotifyExecutionCompleted_NoSettings(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	execution := &entity.Execution{
		ID:        "exec-1",
		StartedAt: time.Now(),
	}

	err := service.NotifyExecutionCompleted(context.Background(), execution, "Test Scenario")
	if err != nil {
		t.Fatalf("NotifyExecutionCompleted failed: %v", err)
	}

	if len(repo.notifications) != 0 {
		t.Errorf("len(notifications) = %d, want 0", len(repo.notifications))
	}
}

func TestNotificationService_NotifyExecutionFailed_NoSettings(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	execution := &entity.Execution{
		ID:        "exec-1",
		StartedAt: time.Now(),
	}

	err := service.NotifyExecutionFailed(context.Background(), execution, "Test Scenario", "Error")
	if err != nil {
		t.Fatalf("NotifyExecutionFailed failed: %v", err)
	}

	if len(repo.notifications) != 0 {
		t.Errorf("len(notifications) = %d, want 0", len(repo.notifications))
	}
}

func TestNotificationService_NotifyAgentOffline_NoSettings(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	agent := &entity.Agent{
		Paw:      "test-paw",
		Hostname: "test-host",
		Platform: "linux",
		LastSeen: time.Now(),
	}

	err := service.NotifyAgentOffline(context.Background(), agent)
	if err != nil {
		t.Fatalf("NotifyAgentOffline failed: %v", err)
	}

	if len(repo.notifications) != 0 {
		t.Errorf("len(notifications) = %d, want 0", len(repo.notifications))
	}
}

func TestNotificationService_NotifyExecutionStarted_DisabledSettings(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	// Add disabled settings
	settings := &entity.NotificationSettings{
		ID:            "settings-1",
		UserID:        "user-1",
		Channel:       entity.ChannelEmail,
		Enabled:       false, // Disabled
		NotifyOnStart: true,
	}
	repo.settings[settings.ID] = settings

	execution := &entity.Execution{
		ID:        "exec-1",
		StartedAt: time.Now(),
	}

	err := service.NotifyExecutionStarted(context.Background(), execution, "Test Scenario")
	if err != nil {
		t.Fatalf("NotifyExecutionStarted failed: %v", err)
	}

	// Disabled settings shouldn't create notifications
	if len(repo.notifications) != 0 {
		t.Errorf("len(notifications) = %d, want 0", len(repo.notifications))
	}
}

func TestNotificationService_NotifyExecutionCompleted_ScoreAlertDisabled(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	settings := &entity.NotificationSettings{
		ID:                  "settings-1",
		UserID:              "user-1",
		Channel:             entity.ChannelEmail,
		Enabled:             true,
		NotifyOnComplete:    true,
		NotifyOnScoreAlert:  false, // Disabled
		ScoreAlertThreshold: 70.0,
	}
	repo.settings[settings.ID] = settings

	execution := &entity.Execution{
		ID:        "exec-1",
		StartedAt: time.Now(),
		Score: &entity.SecurityScore{
			Overall: 50.0, // Below threshold but alert disabled
		},
	}

	err := service.NotifyExecutionCompleted(context.Background(), execution, "Test Scenario")
	if err != nil {
		t.Fatalf("NotifyExecutionCompleted failed: %v", err)
	}

	// Should only have completion notification, not score alert
	if len(repo.notifications) != 1 {
		t.Errorf("len(notifications) = %d, want 1", len(repo.notifications))
	}
}

func TestNotificationService_GetSettingsByUserID_NotFound(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	result, err := service.GetSettingsByUserID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetSettingsByUserID failed: %v", err)
	}

	if result != nil {
		t.Error("GetSettingsByUserID should return nil for nonexistent user")
	}
}

func TestNotificationService_SendEmail_TemplateNotFound(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	smtpConfig := &entity.SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "noreply@example.com",
		UseTLS:   false,
	}
	service := NewNotificationService(repo, userRepo, smtpConfig, "https://localhost:8443", nil)

	// Call sendEmail with an invalid notification type
	err := service.sendEmail("test@example.com", "invalid-type", map[string]any{})
	if err == nil {
		t.Error("sendEmail should fail with unknown notification type")
	}
}

func TestNotificationService_SendEmail_InvalidSMTPConfig(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	// Invalid SMTP config (missing required fields)
	smtpConfig := &entity.SMTPConfig{
		Host: "",
		Port: 0,
	}
	service := NewNotificationService(repo, userRepo, smtpConfig, "https://localhost:8443", nil)

	err := service.sendEmail("test@example.com", entity.NotificationExecutionStarted, map[string]any{
		"ScenarioName": "Test",
		"ExecutionID":  "exec-1",
		"StartedAt":    time.Now().Format(time.RFC1123),
		"SafeMode":     true,
		"DashboardURL": "https://localhost:8443",
	})
	if err == nil {
		t.Error("sendEmail should fail with invalid SMTP config")
	}
}

func TestBuildEmailMessage_AllFields(t *testing.T) {
	msg := buildEmailMessage(
		"sender@example.com",
		"recipient@example.com",
		"Test Subject with Special Chars: <>&",
		"This is the body\nwith multiple\nlines",
	)

	// Verify MIME headers
	if !contains(msg, "MIME-Version: 1.0") {
		t.Error("Message should contain MIME-Version header")
	}
	if !contains(msg, "Content-Type: text/plain") {
		t.Error("Message should contain Content-Type header")
	}
}

func TestNotificationService_NotifyExecutionCompleted_ScoreExactlyAtThreshold(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	settings := &entity.NotificationSettings{
		ID:                  "settings-1",
		UserID:              "user-1",
		Channel:             entity.ChannelEmail,
		Enabled:             true,
		NotifyOnComplete:    true,
		NotifyOnScoreAlert:  true,
		ScoreAlertThreshold: 70.0,
	}
	repo.settings[settings.ID] = settings

	execution := &entity.Execution{
		ID:        "exec-1",
		StartedAt: time.Now(),
		Score: &entity.SecurityScore{
			Overall: 70.0, // Exactly at threshold - should NOT trigger alert
		},
	}

	err := service.NotifyExecutionCompleted(context.Background(), execution, "Test Scenario")
	if err != nil {
		t.Fatalf("NotifyExecutionCompleted failed: %v", err)
	}

	// Score at threshold should NOT trigger alert (only below)
	if len(repo.notifications) != 1 {
		t.Errorf("len(notifications) = %d, want 1", len(repo.notifications))
	}
}

func TestNotificationService_NotifyExecutionCompleted_MultipleSettings(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	// Add multiple enabled settings
	repo.settings["settings-1"] = &entity.NotificationSettings{
		ID:               "settings-1",
		UserID:           "user-1",
		Channel:          entity.ChannelEmail,
		Enabled:          true,
		NotifyOnComplete: true,
	}
	repo.settings["settings-2"] = &entity.NotificationSettings{
		ID:               "settings-2",
		UserID:           "user-2",
		Channel:          entity.ChannelEmail,
		Enabled:          true,
		NotifyOnComplete: true,
	}

	execution := &entity.Execution{
		ID:        "exec-1",
		StartedAt: time.Now(),
	}

	err := service.NotifyExecutionCompleted(context.Background(), execution, "Test Scenario")
	if err != nil {
		t.Fatalf("NotifyExecutionCompleted failed: %v", err)
	}

	// Should create notifications for both users
	if len(repo.notifications) != 2 {
		t.Errorf("len(notifications) = %d, want 2", len(repo.notifications))
	}
}

func TestNotificationService_NotifyExecutionStarted_WithEmailChannel(t *testing.T) {
	repo := newMockNotificationRepo()
	userRepo := &mockUserRepoForNotification{}
	service := NewNotificationService(repo, userRepo, nil, "https://localhost:8443", nil)

	// Settings with email channel but no email address (shouldn't send email)
	settings := &entity.NotificationSettings{
		ID:            "settings-1",
		UserID:        "user-1",
		Channel:       entity.ChannelEmail,
		Enabled:       true,
		EmailAddress:  "", // Empty
		NotifyOnStart: true,
	}
	repo.settings[settings.ID] = settings

	execution := &entity.Execution{
		ID:        "exec-1",
		StartedAt: time.Now(),
		SafeMode:  true,
	}

	err := service.NotifyExecutionStarted(context.Background(), execution, "Test Scenario")
	if err != nil {
		t.Fatalf("NotifyExecutionStarted failed: %v", err)
	}

	// Notification should be created but email not sent (no address)
	if len(repo.notifications) != 1 {
		t.Errorf("len(notifications) = %d, want 1", len(repo.notifications))
	}
}
