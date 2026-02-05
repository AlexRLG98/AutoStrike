package entity

import "time"

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationExecutionStarted   NotificationType = "execution_started"
	NotificationExecutionCompleted NotificationType = "execution_completed"
	NotificationExecutionFailed    NotificationType = "execution_failed"
	NotificationScoreAlert         NotificationType = "score_alert"
	NotificationAgentOffline       NotificationType = "agent_offline"
)

// NotificationChannel represents the delivery channel
type NotificationChannel string

const (
	ChannelEmail   NotificationChannel = "email"
	ChannelWebhook NotificationChannel = "webhook"
)

// NotificationSettings represents user notification preferences
type NotificationSettings struct {
	ID                   string              `json:"id"`
	UserID               string              `json:"user_id"`
	Channel              NotificationChannel `json:"channel"`
	Enabled              bool                `json:"enabled"`
	EmailAddress         string              `json:"email_address,omitempty"`
	WebhookURL           string              `json:"webhook_url,omitempty"`
	NotifyOnStart        bool                `json:"notify_on_start"`
	NotifyOnComplete     bool                `json:"notify_on_complete"`
	NotifyOnFailure      bool                `json:"notify_on_failure"`
	NotifyOnScoreAlert   bool                `json:"notify_on_score_alert"`
	ScoreAlertThreshold  float64             `json:"score_alert_threshold"` // Alert if score below this
	NotifyOnAgentOffline bool                `json:"notify_on_agent_offline"`
	CreatedAt            time.Time           `json:"created_at"`
	UpdatedAt            time.Time           `json:"updated_at"`
}

// Notification represents a notification record
type Notification struct {
	ID        string           `json:"id"`
	UserID    string           `json:"user_id"`
	Type      NotificationType `json:"type"`
	Title     string           `json:"title"`
	Message   string           `json:"message"`
	Data      map[string]any   `json:"data,omitempty"`
	Read      bool             `json:"read"`
	SentAt    *time.Time       `json:"sent_at,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
}

// SMTPConfig represents SMTP server configuration
type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"-"` // Never expose password in JSON
	From     string `json:"from"`
	UseTLS   bool   `json:"use_tls"`
}

// IsValid checks if the SMTP configuration is valid
func (c *SMTPConfig) IsValid() bool {
	return c.Host != "" && c.Port > 0 && c.From != ""
}

// EmailTemplate represents an email template
type EmailTemplate struct {
	Subject string
	Body    string
}

// DefaultEmailTemplates returns the default email templates
func DefaultEmailTemplates() map[NotificationType]EmailTemplate {
	return map[NotificationType]EmailTemplate{
		NotificationExecutionStarted: {
			Subject: "AutoStrike: Execution Started - {{.ScenarioName}}",
			Body: `Hello,

An attack simulation has started on AutoStrike.

Scenario: {{.ScenarioName}}
Execution ID: {{.ExecutionID}}
Started At: {{.StartedAt}}
Safe Mode: {{.SafeMode}}

You can monitor the progress at: {{.DashboardURL}}/executions/{{.ExecutionID}}

Best regards,
AutoStrike Platform`,
		},
		NotificationExecutionCompleted: {
			Subject: "AutoStrike: Execution Completed - Score: {{.Score}}%",
			Body: `Hello,

An attack simulation has completed on AutoStrike.

Scenario: {{.ScenarioName}}
Execution ID: {{.ExecutionID}}
Status: Completed
Security Score: {{.Score}}%

Results:
- Blocked: {{.Blocked}}
- Detected: {{.Detected}}
- Successful: {{.Successful}}
- Total: {{.Total}}

View full results at: {{.DashboardURL}}/executions/{{.ExecutionID}}

Best regards,
AutoStrike Platform`,
		},
		NotificationExecutionFailed: {
			Subject: "AutoStrike: Execution Failed - {{.ScenarioName}}",
			Body: `Hello,

An attack simulation has failed on AutoStrike.

Scenario: {{.ScenarioName}}
Execution ID: {{.ExecutionID}}
Status: Failed
Error: {{.Error}}

Please check the dashboard for more details: {{.DashboardURL}}/executions/{{.ExecutionID}}

Best regards,
AutoStrike Platform`,
		},
		NotificationScoreAlert: {
			Subject: "AutoStrike: Low Security Score Alert - {{.Score}}%",
			Body: `Hello,

A security score alert has been triggered on AutoStrike.

Scenario: {{.ScenarioName}}
Execution ID: {{.ExecutionID}}
Security Score: {{.Score}}%
Alert Threshold: {{.Threshold}}%

This score is below your configured alert threshold. Please review your security controls.

View details at: {{.DashboardURL}}/executions/{{.ExecutionID}}

Best regards,
AutoStrike Platform`,
		},
		NotificationAgentOffline: {
			Subject: "AutoStrike: Agent Offline - {{.Hostname}}",
			Body: `Hello,

An agent has gone offline on AutoStrike.

Agent: {{.Hostname}}
Paw: {{.Paw}}
Platform: {{.Platform}}
Last Seen: {{.LastSeen}}

Please check the agent status at: {{.DashboardURL}}/agents

Best regards,
AutoStrike Platform`,
		},
	}
}
