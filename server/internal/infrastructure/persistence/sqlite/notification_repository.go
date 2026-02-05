package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"

	"autostrike/internal/domain/entity"
)

// NotificationRepository implements repository.NotificationRepository using SQLite
type NotificationRepository struct {
	db *sql.DB
}

// NewNotificationRepository creates a new SQLite notification repository
func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// CreateSettings creates new notification settings
func (r *NotificationRepository) CreateSettings(ctx context.Context, settings *entity.NotificationSettings) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO notification_settings (
			id, user_id, channel, enabled, email_address, webhook_url,
			notify_on_start, notify_on_complete, notify_on_failure,
			notify_on_score_alert, score_alert_threshold, notify_on_agent_offline,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, settings.ID, settings.UserID, settings.Channel, settings.Enabled,
		settings.EmailAddress, settings.WebhookURL,
		settings.NotifyOnStart, settings.NotifyOnComplete, settings.NotifyOnFailure,
		settings.NotifyOnScoreAlert, settings.ScoreAlertThreshold, settings.NotifyOnAgentOffline,
		settings.CreatedAt, settings.UpdatedAt)

	return err
}

// UpdateSettings updates notification settings
func (r *NotificationRepository) UpdateSettings(ctx context.Context, settings *entity.NotificationSettings) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE notification_settings SET
			channel = ?, enabled = ?, email_address = ?, webhook_url = ?,
			notify_on_start = ?, notify_on_complete = ?, notify_on_failure = ?,
			notify_on_score_alert = ?, score_alert_threshold = ?, notify_on_agent_offline = ?,
			updated_at = ?
		WHERE id = ?
	`, settings.Channel, settings.Enabled, settings.EmailAddress, settings.WebhookURL,
		settings.NotifyOnStart, settings.NotifyOnComplete, settings.NotifyOnFailure,
		settings.NotifyOnScoreAlert, settings.ScoreAlertThreshold, settings.NotifyOnAgentOffline,
		settings.UpdatedAt, settings.ID)

	return err
}

// FindSettingsByUserID finds notification settings by user ID
func (r *NotificationRepository) FindSettingsByUserID(ctx context.Context, userID string) (*entity.NotificationSettings, error) {
	settings := &entity.NotificationSettings{}
	var emailAddress, webhookURL sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, channel, enabled, email_address, webhook_url,
			notify_on_start, notify_on_complete, notify_on_failure,
			notify_on_score_alert, score_alert_threshold, notify_on_agent_offline,
			created_at, updated_at
		FROM notification_settings WHERE user_id = ?
	`, userID).Scan(
		&settings.ID, &settings.UserID, &settings.Channel, &settings.Enabled,
		&emailAddress, &webhookURL,
		&settings.NotifyOnStart, &settings.NotifyOnComplete, &settings.NotifyOnFailure,
		&settings.NotifyOnScoreAlert, &settings.ScoreAlertThreshold, &settings.NotifyOnAgentOffline,
		&settings.CreatedAt, &settings.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if emailAddress.Valid {
		settings.EmailAddress = emailAddress.String
	}
	if webhookURL.Valid {
		settings.WebhookURL = webhookURL.String
	}

	return settings, nil
}

// FindAllEnabledSettings finds all enabled notification settings
func (r *NotificationRepository) FindAllEnabledSettings(ctx context.Context) ([]*entity.NotificationSettings, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, channel, enabled, email_address, webhook_url,
			notify_on_start, notify_on_complete, notify_on_failure,
			notify_on_score_alert, score_alert_threshold, notify_on_agent_offline,
			created_at, updated_at
		FROM notification_settings WHERE enabled = 1
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settingsList []*entity.NotificationSettings
	for rows.Next() {
		settings := &entity.NotificationSettings{}
		var emailAddress, webhookURL sql.NullString

		err := rows.Scan(
			&settings.ID, &settings.UserID, &settings.Channel, &settings.Enabled,
			&emailAddress, &webhookURL,
			&settings.NotifyOnStart, &settings.NotifyOnComplete, &settings.NotifyOnFailure,
			&settings.NotifyOnScoreAlert, &settings.ScoreAlertThreshold, &settings.NotifyOnAgentOffline,
			&settings.CreatedAt, &settings.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if emailAddress.Valid {
			settings.EmailAddress = emailAddress.String
		}
		if webhookURL.Valid {
			settings.WebhookURL = webhookURL.String
		}

		settingsList = append(settingsList, settings)
	}

	return settingsList, rows.Err()
}

// DeleteSettings deletes notification settings
func (r *NotificationRepository) DeleteSettings(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM notification_settings WHERE id = ?`, id)
	return err
}

// CreateNotification creates a new notification
func (r *NotificationRepository) CreateNotification(ctx context.Context, notification *entity.Notification) error {
	dataJSON, err := json.Marshal(notification.Data)
	if err != nil {
		dataJSON = []byte("{}")
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO notifications (id, user_id, type, title, message, data, read, sent_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, notification.ID, notification.UserID, notification.Type, notification.Title,
		notification.Message, string(dataJSON), notification.Read, notification.SentAt, notification.CreatedAt)

	return err
}

// FindNotificationByID finds a notification by ID
func (r *NotificationRepository) FindNotificationByID(ctx context.Context, id string) (*entity.Notification, error) {
	notification := &entity.Notification{}
	var dataJSON string
	var sentAt sql.NullTime

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, type, title, message, data, read, sent_at, created_at
		FROM notifications WHERE id = ?
	`, id).Scan(
		&notification.ID, &notification.UserID, &notification.Type,
		&notification.Title, &notification.Message, &dataJSON,
		&notification.Read, &sentAt, &notification.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if dataJSON != "" {
		_ = json.Unmarshal([]byte(dataJSON), &notification.Data)
	}
	if sentAt.Valid {
		notification.SentAt = &sentAt.Time
	}

	return notification, nil
}

// FindNotificationsByUserID finds notifications by user ID
func (r *NotificationRepository) FindNotificationsByUserID(ctx context.Context, userID string, limit int) ([]*entity.Notification, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, type, title, message, data, read, sent_at, created_at
		FROM notifications WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanNotifications(rows)
}

// FindUnreadByUserID finds unread notifications by user ID
func (r *NotificationRepository) FindUnreadByUserID(ctx context.Context, userID string) ([]*entity.Notification, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, type, title, message, data, read, sent_at, created_at
		FROM notifications WHERE user_id = ? AND read = 0
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanNotifications(rows)
}

// MarkAsRead marks a notification as read
func (r *NotificationRepository) MarkAsRead(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE notifications SET read = 1 WHERE id = ?`, id)
	return err
}

// MarkAllAsRead marks all notifications as read for a user
func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE notifications SET read = 1 WHERE user_id = ?`, userID)
	return err
}

func (r *NotificationRepository) scanNotifications(rows *sql.Rows) ([]*entity.Notification, error) {
	var notifications []*entity.Notification

	for rows.Next() {
		notification := &entity.Notification{}
		var dataJSON string
		var sentAt sql.NullTime

		err := rows.Scan(
			&notification.ID, &notification.UserID, &notification.Type,
			&notification.Title, &notification.Message, &dataJSON,
			&notification.Read, &sentAt, &notification.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if dataJSON != "" {
			_ = json.Unmarshal([]byte(dataJSON), &notification.Data)
		}
		if sentAt.Valid {
			notification.SentAt = &sentAt.Time
		}

		notifications = append(notifications, notification)
	}

	return notifications, rows.Err()
}
