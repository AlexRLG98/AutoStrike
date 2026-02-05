package application

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"text/template"
	"time"

	"autostrike/internal/domain/entity"
	"autostrike/internal/domain/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// NotificationService handles notification-related business logic
type NotificationService struct {
	notificationRepo repository.NotificationRepository
	userRepo         repository.UserRepository
	smtpConfig       *entity.SMTPConfig
	dashboardURL     string
	templates        map[entity.NotificationType]entity.EmailTemplate
	logger           *zap.Logger
	emailSemaphore   chan struct{} // Bounds concurrent email goroutines
}

// NewNotificationService creates a new notification service
func NewNotificationService(
	notificationRepo repository.NotificationRepository,
	userRepo repository.UserRepository,
	smtpConfig *entity.SMTPConfig,
	dashboardURL string,
	logger *zap.Logger,
) *NotificationService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &NotificationService{
		notificationRepo: notificationRepo,
		userRepo:         userRepo,
		smtpConfig:       smtpConfig,
		dashboardURL:     dashboardURL,
		templates:        entity.DefaultEmailTemplates(),
		logger:           logger,
		emailSemaphore:   make(chan struct{}, 10), // Max 10 concurrent email sends
	}
}

// SetSMTPConfig updates the SMTP configuration
func (s *NotificationService) SetSMTPConfig(config *entity.SMTPConfig) {
	s.smtpConfig = config
}

// GetSMTPConfig returns the current SMTP configuration (without password)
func (s *NotificationService) GetSMTPConfig() *entity.SMTPConfig {
	if s.smtpConfig == nil {
		return nil
	}
	// Return a copy without the password
	return &entity.SMTPConfig{
		Host:     s.smtpConfig.Host,
		Port:     s.smtpConfig.Port,
		Username: s.smtpConfig.Username,
		From:     s.smtpConfig.From,
		UseTLS:   s.smtpConfig.UseTLS,
	}
}

// CreateSettings creates notification settings for a user
func (s *NotificationService) CreateSettings(ctx context.Context, settings *entity.NotificationSettings) error {
	settings.ID = uuid.New().String()
	settings.CreatedAt = time.Now()
	settings.UpdatedAt = time.Now()
	return s.notificationRepo.CreateSettings(ctx, settings)
}

// UpdateSettings updates notification settings
func (s *NotificationService) UpdateSettings(ctx context.Context, settings *entity.NotificationSettings) error {
	settings.UpdatedAt = time.Now()
	return s.notificationRepo.UpdateSettings(ctx, settings)
}

// GetSettingsByUserID gets notification settings for a user
func (s *NotificationService) GetSettingsByUserID(ctx context.Context, userID string) (*entity.NotificationSettings, error) {
	return s.notificationRepo.FindSettingsByUserID(ctx, userID)
}

// DeleteSettings deletes notification settings
func (s *NotificationService) DeleteSettings(ctx context.Context, id string) error {
	return s.notificationRepo.DeleteSettings(ctx, id)
}

// GetNotificationsByUserID gets notifications for a user
func (s *NotificationService) GetNotificationsByUserID(ctx context.Context, userID string, limit int) ([]*entity.Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.notificationRepo.FindNotificationsByUserID(ctx, userID, limit)
}

// GetUnreadCount gets the count of unread notifications for a user
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID string) (int, error) {
	notifications, err := s.notificationRepo.FindUnreadByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	return len(notifications), nil
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, id string) error {
	return s.notificationRepo.MarkAsRead(ctx, id)
}

// MarkAsReadForUser marks a notification as read only if owned by the user
func (s *NotificationService) MarkAsReadForUser(ctx context.Context, id string, userID string) error {
	notification, err := s.notificationRepo.FindNotificationByID(ctx, id)
	if err != nil || notification == nil {
		return fmt.Errorf("notification not found or not owned by user")
	}

	if notification.UserID != userID {
		return fmt.Errorf("notification not found or not owned by user")
	}

	return s.notificationRepo.MarkAsRead(ctx, id)
}

// MarkAllAsRead marks all notifications as read for a user
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID string) error {
	return s.notificationRepo.MarkAllAsRead(ctx, userID)
}

// sendEmailAsync sends email asynchronously with semaphore control
func (s *NotificationService) sendEmailAsync(to string, notificationType entity.NotificationType, data map[string]any) {
	go func() {
		s.emailSemaphore <- struct{}{}
		defer func() { <-s.emailSemaphore }()
		if err := s.sendEmail(to, notificationType, data); err != nil {
			s.logger.Error("Failed to send email",
				zap.String("to", to),
				zap.String("type", string(notificationType)),
				zap.Error(err),
			)
		}
	}()
}

// shouldSendEmail checks if email should be sent for a setting
func shouldSendEmail(setting *entity.NotificationSettings) bool {
	return setting.Channel == entity.ChannelEmail && setting.EmailAddress != ""
}

// NotifyExecutionStarted sends notifications for execution start
func (s *NotificationService) NotifyExecutionStarted(ctx context.Context, execution *entity.Execution, scenarioName string) error {
	settings, err := s.notificationRepo.FindAllEnabledSettings(ctx)
	if err != nil {
		return err
	}

	data := map[string]any{
		"ScenarioName": scenarioName,
		"ExecutionID":  execution.ID,
		"StartedAt":    execution.StartedAt.Format(time.RFC1123),
		"SafeMode":     execution.SafeMode,
		"DashboardURL": s.dashboardURL,
	}

	for _, setting := range settings {
		if !setting.NotifyOnStart {
			continue
		}

		notification := &entity.Notification{
			ID:        uuid.New().String(),
			UserID:    setting.UserID,
			Type:      entity.NotificationExecutionStarted,
			Title:     fmt.Sprintf("Execution Started: %s", scenarioName),
			Message:   fmt.Sprintf("Attack simulation started for scenario '%s'", scenarioName),
			Data:      data,
			CreatedAt: time.Now(),
		}

		if s.notificationRepo.CreateNotification(ctx, notification) != nil {
			continue // Don't fail on individual notification errors
		}

		if shouldSendEmail(setting) {
			s.sendEmailAsync(setting.EmailAddress, entity.NotificationExecutionStarted, data)
		}
	}

	return nil
}

// buildExecutionCompletedData builds notification data from execution
func buildExecutionCompletedData(execution *entity.Execution, scenarioName, dashboardURL string) (map[string]any, float64) {
	score := 0.0
	blocked, detected, successful, total := 0, 0, 0, 0
	if execution.Score != nil {
		score = execution.Score.Overall
		blocked = execution.Score.Blocked
		detected = execution.Score.Detected
		successful = execution.Score.Successful
		total = execution.Score.Total
	}

	data := map[string]any{
		"ScenarioName": scenarioName,
		"ExecutionID":  execution.ID,
		"Score":        fmt.Sprintf("%.1f", score),
		"Blocked":      blocked,
		"Detected":     detected,
		"Successful":   successful,
		"Total":        total,
		"DashboardURL": dashboardURL,
	}
	return data, score
}

// processScoreAlert handles score alert notification if threshold exceeded
func (s *NotificationService) processScoreAlert(ctx context.Context, setting *entity.NotificationSettings, data map[string]any, score float64) {
	if !setting.NotifyOnScoreAlert || score >= setting.ScoreAlertThreshold {
		return
	}

	alertData := make(map[string]any)
	for k, v := range data {
		alertData[k] = v
	}
	alertData["Threshold"] = fmt.Sprintf("%.1f", setting.ScoreAlertThreshold)

	alertNotification := &entity.Notification{
		ID:        uuid.New().String(),
		UserID:    setting.UserID,
		Type:      entity.NotificationScoreAlert,
		Title:     fmt.Sprintf("Low Score Alert: %.1f%%", score),
		Message:   fmt.Sprintf("Security score %.1f%% is below threshold %.1f%%", score, setting.ScoreAlertThreshold),
		Data:      alertData,
		CreatedAt: time.Now(),
	}

	if s.notificationRepo.CreateNotification(ctx, alertNotification) != nil {
		return
	}

	if shouldSendEmail(setting) {
		s.sendEmailAsync(setting.EmailAddress, entity.NotificationScoreAlert, alertData)
	}
}

// NotifyExecutionCompleted sends notifications for execution completion
func (s *NotificationService) NotifyExecutionCompleted(ctx context.Context, execution *entity.Execution, scenarioName string) error {
	settings, err := s.notificationRepo.FindAllEnabledSettings(ctx)
	if err != nil {
		return err
	}

	data, score := buildExecutionCompletedData(execution, scenarioName, s.dashboardURL)

	for _, setting := range settings {
		if !setting.NotifyOnComplete {
			continue
		}

		notification := &entity.Notification{
			ID:        uuid.New().String(),
			UserID:    setting.UserID,
			Type:      entity.NotificationExecutionCompleted,
			Title:     fmt.Sprintf("Execution Completed: %.1f%%", score),
			Message:   fmt.Sprintf("Attack simulation completed for '%s' with score %.1f%%", scenarioName, score),
			Data:      data,
			CreatedAt: time.Now(),
		}

		if s.notificationRepo.CreateNotification(ctx, notification) != nil {
			continue
		}

		if shouldSendEmail(setting) {
			s.sendEmailAsync(setting.EmailAddress, entity.NotificationExecutionCompleted, data)
		}

		s.processScoreAlert(ctx, setting, data, score)
	}

	return nil
}

// NotifyExecutionFailed sends notifications for execution failure
func (s *NotificationService) NotifyExecutionFailed(ctx context.Context, execution *entity.Execution, scenarioName string, errMsg string) error {
	settings, err := s.notificationRepo.FindAllEnabledSettings(ctx)
	if err != nil {
		return err
	}

	data := map[string]any{
		"ScenarioName": scenarioName,
		"ExecutionID":  execution.ID,
		"Error":        errMsg,
		"DashboardURL": s.dashboardURL,
	}

	for _, setting := range settings {
		if !setting.NotifyOnFailure {
			continue
		}

		notification := &entity.Notification{
			ID:        uuid.New().String(),
			UserID:    setting.UserID,
			Type:      entity.NotificationExecutionFailed,
			Title:     fmt.Sprintf("Execution Failed: %s", scenarioName),
			Message:   fmt.Sprintf("Attack simulation failed for '%s': %s", scenarioName, errMsg),
			Data:      data,
			CreatedAt: time.Now(),
		}

		if s.notificationRepo.CreateNotification(ctx, notification) != nil {
			continue
		}

		if shouldSendEmail(setting) {
			s.sendEmailAsync(setting.EmailAddress, entity.NotificationExecutionFailed, data)
		}
	}

	return nil
}

// NotifyAgentOffline sends notifications when an agent goes offline
func (s *NotificationService) NotifyAgentOffline(ctx context.Context, agent *entity.Agent) error {
	settings, err := s.notificationRepo.FindAllEnabledSettings(ctx)
	if err != nil {
		return err
	}

	data := map[string]any{
		"Hostname":     agent.Hostname,
		"Paw":          agent.Paw,
		"Platform":     agent.Platform,
		"LastSeen":     agent.LastSeen.Format(time.RFC1123),
		"DashboardURL": s.dashboardURL,
	}

	for _, setting := range settings {
		if !setting.NotifyOnAgentOffline {
			continue
		}

		notification := &entity.Notification{
			ID:        uuid.New().String(),
			UserID:    setting.UserID,
			Type:      entity.NotificationAgentOffline,
			Title:     fmt.Sprintf("Agent Offline: %s", agent.Hostname),
			Message:   fmt.Sprintf("Agent '%s' (%s) has gone offline", agent.Hostname, agent.Paw),
			Data:      data,
			CreatedAt: time.Now(),
		}

		if s.notificationRepo.CreateNotification(ctx, notification) != nil {
			continue
		}

		if shouldSendEmail(setting) {
			s.sendEmailAsync(setting.EmailAddress, entity.NotificationAgentOffline, data)
		}
	}

	return nil
}

// renderEmailTemplate renders a template with data
func renderEmailTemplate(tmplStr string, data map[string]any) (string, error) {
	tmpl, err := template.New("email").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// buildEmailMessage constructs the email message
func buildEmailMessage(from, to, subject, body string) string {
	msg := strings.Builder{}
	msg.WriteString(fmt.Sprintf("From: %s\r\n", from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)
	return msg.String()
}

// sendEmailTLS sends email over TLS connection
func (s *NotificationService) sendEmailTLS(addr, to string, auth smtp.Auth, msg string) error {
	tlsConfig := &tls.Config{
		ServerName: s.smtpConfig.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.smtpConfig.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	if err := client.Mail(s.smtpConfig.From); err != nil {
		return fmt.Errorf("SMTP MAIL command failed: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT command failed: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA command failed: %w", err)
	}

	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close email writer: %w", err)
	}

	return client.Quit()
}

// sendEmail sends an email using the configured SMTP server
func (s *NotificationService) sendEmail(to string, notificationType entity.NotificationType, data map[string]any) error {
	if s.smtpConfig == nil || !s.smtpConfig.IsValid() {
		return fmt.Errorf("SMTP not configured")
	}

	tmpl, ok := s.templates[notificationType]
	if !ok {
		return fmt.Errorf("template not found for notification type: %s", notificationType)
	}

	subject, err := renderEmailTemplate(tmpl.Subject, data)
	if err != nil {
		return fmt.Errorf("failed to render subject: %w", err)
	}

	body, err := renderEmailTemplate(tmpl.Body, data)
	if err != nil {
		return fmt.Errorf("failed to render body: %w", err)
	}

	msg := buildEmailMessage(s.smtpConfig.From, to, subject, body)
	addr := fmt.Sprintf("%s:%d", s.smtpConfig.Host, s.smtpConfig.Port)

	var auth smtp.Auth
	if s.smtpConfig.Username != "" && s.smtpConfig.Password != "" {
		auth = smtp.PlainAuth("", s.smtpConfig.Username, s.smtpConfig.Password, s.smtpConfig.Host)
	}

	if s.smtpConfig.UseTLS {
		return s.sendEmailTLS(addr, to, auth, msg)
	}

	return smtp.SendMail(addr, auth, s.smtpConfig.From, []string{to}, []byte(msg))
}

// TestSMTPConnection tests the SMTP connection
func (s *NotificationService) TestSMTPConnection(ctx context.Context, to string) error {
	if s.smtpConfig == nil || !s.smtpConfig.IsValid() {
		return fmt.Errorf("SMTP not configured")
	}

	data := map[string]any{
		"ScenarioName": "Test Scenario",
		"ExecutionID":  "test-123",
		"StartedAt":    time.Now().Format(time.RFC1123),
		"SafeMode":     true,
		"DashboardURL": s.dashboardURL,
	}

	return s.sendEmail(to, entity.NotificationExecutionStarted, data)
}
