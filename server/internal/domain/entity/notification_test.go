package entity

import (
	"testing"
)

func TestNotificationType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		got      NotificationType
		expected string
	}{
		{"ExecutionStarted", NotificationExecutionStarted, "execution_started"},
		{"ExecutionCompleted", NotificationExecutionCompleted, "execution_completed"},
		{"ExecutionFailed", NotificationExecutionFailed, "execution_failed"},
		{"ScoreAlert", NotificationScoreAlert, "score_alert"},
		{"AgentOffline", NotificationAgentOffline, "agent_offline"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.got) != tt.expected {
				t.Errorf("NotificationType %s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestNotificationChannel_Constants(t *testing.T) {
	tests := []struct {
		name     string
		got      NotificationChannel
		expected string
	}{
		{"Email", ChannelEmail, "email"},
		{"Webhook", ChannelWebhook, "webhook"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.got) != tt.expected {
				t.Errorf("NotificationChannel %s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestSMTPConfig_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		config *SMTPConfig
		want   bool
	}{
		{
			name: "Valid config",
			config: &SMTPConfig{
				Host:     "smtp.example.com",
				Port:     587,
				Username: "user",
				Password: "pass",
				From:     "noreply@example.com",
				UseTLS:   true,
			},
			want: true,
		},
		{
			name: "Valid config without auth",
			config: &SMTPConfig{
				Host: "smtp.example.com",
				Port: 25,
				From: "noreply@example.com",
			},
			want: true,
		},
		{
			name: "Missing host",
			config: &SMTPConfig{
				Port: 587,
				From: "noreply@example.com",
			},
			want: false,
		},
		{
			name: "Port is zero",
			config: &SMTPConfig{
				Host: "smtp.example.com",
				Port: 0,
				From: "noreply@example.com",
			},
			want: false,
		},
		{
			name: "Negative port",
			config: &SMTPConfig{
				Host: "smtp.example.com",
				Port: -1,
				From: "noreply@example.com",
			},
			want: false,
		},
		{
			name: "Missing from",
			config: &SMTPConfig{
				Host: "smtp.example.com",
				Port: 587,
			},
			want: false,
		},
		{
			name:   "Empty config",
			config: &SMTPConfig{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsValid(); got != tt.want {
				t.Errorf("SMTPConfig.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultEmailTemplates(t *testing.T) {
	templates := DefaultEmailTemplates()

	// Check that all notification types have templates
	expectedTypes := []NotificationType{
		NotificationExecutionStarted,
		NotificationExecutionCompleted,
		NotificationExecutionFailed,
		NotificationScoreAlert,
		NotificationAgentOffline,
	}

	if len(templates) != len(expectedTypes) {
		t.Errorf("DefaultEmailTemplates() returned %d templates, want %d", len(templates), len(expectedTypes))
	}

	for _, nt := range expectedTypes {
		t.Run(string(nt), func(t *testing.T) {
			tmpl, ok := templates[nt]
			if !ok {
				t.Errorf("Missing template for notification type %q", nt)
				return
			}

			if tmpl.Subject == "" {
				t.Errorf("Template for %q has empty subject", nt)
			}

			if tmpl.Body == "" {
				t.Errorf("Template for %q has empty body", nt)
			}
		})
	}
}

func TestDefaultEmailTemplates_ExecutionStarted(t *testing.T) {
	templates := DefaultEmailTemplates()
	tmpl := templates[NotificationExecutionStarted]

	// Verify template contains expected placeholders
	if !containsString(tmpl.Subject, "{{.ScenarioName}}") {
		t.Error("ExecutionStarted subject should contain ScenarioName placeholder")
	}

	expectedPlaceholders := []string{
		"{{.ScenarioName}}",
		"{{.ExecutionID}}",
		"{{.StartedAt}}",
		"{{.SafeMode}}",
		"{{.DashboardURL}}",
	}

	for _, ph := range expectedPlaceholders {
		if !containsString(tmpl.Body, ph) {
			t.Errorf("ExecutionStarted body should contain %s placeholder", ph)
		}
	}
}

func TestDefaultEmailTemplates_ExecutionCompleted(t *testing.T) {
	templates := DefaultEmailTemplates()
	tmpl := templates[NotificationExecutionCompleted]

	// Verify template contains expected placeholders
	if !containsString(tmpl.Subject, "{{.Score}}") {
		t.Error("ExecutionCompleted subject should contain Score placeholder")
	}

	expectedPlaceholders := []string{
		"{{.ScenarioName}}",
		"{{.ExecutionID}}",
		"{{.Score}}",
		"{{.Blocked}}",
		"{{.Detected}}",
		"{{.Successful}}",
		"{{.Total}}",
		"{{.DashboardURL}}",
	}

	for _, ph := range expectedPlaceholders {
		if !containsString(tmpl.Body, ph) {
			t.Errorf("ExecutionCompleted body should contain %s placeholder", ph)
		}
	}
}

func TestDefaultEmailTemplates_ExecutionFailed(t *testing.T) {
	templates := DefaultEmailTemplates()
	tmpl := templates[NotificationExecutionFailed]

	expectedPlaceholders := []string{
		"{{.ScenarioName}}",
		"{{.ExecutionID}}",
		"{{.Error}}",
		"{{.DashboardURL}}",
	}

	for _, ph := range expectedPlaceholders {
		if !containsString(tmpl.Body, ph) {
			t.Errorf("ExecutionFailed body should contain %s placeholder", ph)
		}
	}
}

func TestDefaultEmailTemplates_ScoreAlert(t *testing.T) {
	templates := DefaultEmailTemplates()
	tmpl := templates[NotificationScoreAlert]

	if !containsString(tmpl.Subject, "{{.Score}}") {
		t.Error("ScoreAlert subject should contain Score placeholder")
	}

	expectedPlaceholders := []string{
		"{{.ScenarioName}}",
		"{{.ExecutionID}}",
		"{{.Score}}",
		"{{.Threshold}}",
		"{{.DashboardURL}}",
	}

	for _, ph := range expectedPlaceholders {
		if !containsString(tmpl.Body, ph) {
			t.Errorf("ScoreAlert body should contain %s placeholder", ph)
		}
	}
}

func TestDefaultEmailTemplates_AgentOffline(t *testing.T) {
	templates := DefaultEmailTemplates()
	tmpl := templates[NotificationAgentOffline]

	if !containsString(tmpl.Subject, "{{.Hostname}}") {
		t.Error("AgentOffline subject should contain Hostname placeholder")
	}

	expectedPlaceholders := []string{
		"{{.Hostname}}",
		"{{.Paw}}",
		"{{.Platform}}",
		"{{.LastSeen}}",
		"{{.DashboardURL}}",
	}

	for _, ph := range expectedPlaceholders {
		if !containsString(tmpl.Body, ph) {
			t.Errorf("AgentOffline body should contain %s placeholder", ph)
		}
	}
}

func TestNotificationSettings_Struct(t *testing.T) {
	settings := &NotificationSettings{
		ID:                   "settings-1",
		UserID:               "user-1",
		Channel:              ChannelEmail,
		Enabled:              true,
		EmailAddress:         "test@example.com",
		WebhookURL:           "",
		NotifyOnStart:        true,
		NotifyOnComplete:     true,
		NotifyOnFailure:      true,
		NotifyOnScoreAlert:   true,
		ScoreAlertThreshold:  70.0,
		NotifyOnAgentOffline: true,
	}

	if settings.ID != "settings-1" {
		t.Errorf("ID = %q, want %q", settings.ID, "settings-1")
	}
	if settings.Channel != ChannelEmail {
		t.Errorf("Channel = %q, want %q", settings.Channel, ChannelEmail)
	}
	if !settings.Enabled {
		t.Error("Enabled should be true")
	}
	if settings.ScoreAlertThreshold != 70.0 {
		t.Errorf("ScoreAlertThreshold = %v, want 70.0", settings.ScoreAlertThreshold)
	}
}

func TestNotification_Struct(t *testing.T) {
	notification := &Notification{
		ID:      "notif-1",
		UserID:  "user-1",
		Type:    NotificationExecutionCompleted,
		Title:   "Test Title",
		Message: "Test Message",
		Data:    map[string]any{"key": "value"},
		Read:    false,
	}

	if notification.ID != "notif-1" {
		t.Errorf("ID = %q, want %q", notification.ID, "notif-1")
	}
	if notification.Type != NotificationExecutionCompleted {
		t.Errorf("Type = %q, want %q", notification.Type, NotificationExecutionCompleted)
	}
	if notification.Read {
		t.Error("Read should be false")
	}
	if notification.Data["key"] != "value" {
		t.Error("Data should contain key=value")
	}
}

func TestEmailTemplate_Struct(t *testing.T) {
	tmpl := EmailTemplate{
		Subject: "Test Subject",
		Body:    "Test Body",
	}

	if tmpl.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want %q", tmpl.Subject, "Test Subject")
	}
	if tmpl.Body != "Test Body" {
		t.Errorf("Body = %q, want %q", tmpl.Body, "Test Body")
	}
}

// containsString checks if s contains substr
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
