package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"

	"autostrike/internal/application"
	"autostrike/internal/domain/entity"

	"github.com/gin-gonic/gin"
)

// Notification handler error messages and routes
const (
	errNotifNotAuthenticated = "not authenticated"
	errSettingsNotFound      = "settings not found"
	errFailedToGetSettings   = "failed to get settings"
	routeSettings            = "/settings"
)

// NotificationHandler handles notification-related HTTP requests
type NotificationHandler struct {
	notificationService *application.NotificationService
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(notificationService *application.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

// RegisterRoutes registers the notification routes
func (h *NotificationHandler) RegisterRoutes(router *gin.RouterGroup) {
	notifications := router.Group("/notifications")
	{
		notifications.GET("", h.GetNotifications)
		notifications.GET("/unread/count", h.GetUnreadCount)
		notifications.POST("/:id/read", h.MarkAsRead)
		notifications.POST("/read-all", h.MarkAllAsRead)

		// Settings
		notifications.GET(routeSettings, h.GetSettings)
		notifications.POST(routeSettings, h.CreateSettings)
		notifications.PUT(routeSettings, h.UpdateSettings)
		notifications.DELETE(routeSettings, h.DeleteSettings)

		// SMTP config (admin only)
		notifications.GET("/smtp", h.GetSMTPConfig)
		notifications.POST("/smtp/test", h.TestSMTP)
	}
}

// NotificationSettingsRequest represents the request to create/update notification settings
type NotificationSettingsRequest struct {
	Channel              string  `json:"channel" binding:"required,oneof=email webhook"`
	Enabled              bool    `json:"enabled"`
	EmailAddress         string  `json:"email_address"`
	WebhookURL           string  `json:"webhook_url"`
	NotifyOnStart        bool    `json:"notify_on_start"`
	NotifyOnComplete     bool    `json:"notify_on_complete"`
	NotifyOnFailure      bool    `json:"notify_on_failure"`
	NotifyOnScoreAlert   bool    `json:"notify_on_score_alert"`
	ScoreAlertThreshold  float64 `json:"score_alert_threshold"`
	NotifyOnAgentOffline bool    `json:"notify_on_agent_offline"`
}

// Validate validates the notification settings request
func (r *NotificationSettingsRequest) Validate() error {
	if r.Channel == "email" && r.Enabled {
		if r.EmailAddress == "" {
			return fmt.Errorf("email address is required when channel is email")
		}
		if _, err := mail.ParseAddress(r.EmailAddress); err != nil {
			return fmt.Errorf("invalid email address format")
		}
	}
	if r.Channel == "webhook" && r.Enabled {
		if r.WebhookURL == "" {
			return fmt.Errorf("webhook URL is required when channel is webhook")
		}
		if _, err := url.ParseRequestURI(r.WebhookURL); err != nil {
			return fmt.Errorf("invalid webhook URL format")
		}
	}
	return nil
}

// GetNotifications godoc
// @Summary Get notifications
// @Description Get notifications for the current user
// @Tags notifications
// @Accept json
// @Produce json
// @Param limit query int false "Limit (default: 50)"
// @Success 200 {array} entity.Notification
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/notifications [get]
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotifNotAuthenticated})
		return
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	notifications, err := h.notificationService.GetNotificationsByUserID(c.Request.Context(), userID.(string), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get notifications"})
		return
	}

	if notifications == nil {
		notifications = []*entity.Notification{}
	}

	c.JSON(http.StatusOK, notifications)
}

// GetUnreadCount godoc
// @Summary Get unread count
// @Description Get count of unread notifications
// @Tags notifications
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/notifications/unread/count [get]
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotifNotAuthenticated})
		return
	}

	count, err := h.notificationService.GetUnreadCount(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get unread count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// MarkAsRead godoc
// @Summary Mark notification as read
// @Description Mark a notification as read
// @Tags notifications
// @Accept json
// @Produce json
// @Param id path string true "Notification ID"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/notifications/{id}/read [post]
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotifNotAuthenticated})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "notification ID required"})
		return
	}

	// Verify ownership by marking as read with user ID verification
	if err := h.notificationService.MarkAsReadForUser(c.Request.Context(), id, userID.(string)); err != nil {
		if err.Error() == "notification not found or not owned by user" {
			c.JSON(http.StatusForbidden, gin.H{"error": "notification not found or access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// MarkAllAsRead godoc
// @Summary Mark all notifications as read
// @Description Mark all notifications as read for the current user
// @Tags notifications
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/notifications/read-all [post]
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotifNotAuthenticated})
		return
	}

	if h.notificationService.MarkAllAsRead(c.Request.Context(), userID.(string)) != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark all as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetSettings godoc
// @Summary Get notification settings
// @Description Get notification settings for the current user
// @Tags notifications
// @Accept json
// @Produce json
// @Success 200 {object} entity.NotificationSettings
// @Failure 401 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/notifications/settings [get]
func (h *NotificationHandler) GetSettings(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotifNotAuthenticated})
		return
	}

	settings, err := h.notificationService.GetSettingsByUserID(c.Request.Context(), userID.(string))
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": errSettingsNotFound})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": errFailedToGetSettings})
		return
	}
	if settings == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errSettingsNotFound})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// CreateSettings godoc
// @Summary Create notification settings
// @Description Create notification settings for the current user
// @Tags notifications
// @Accept json
// @Produce json
// @Param body body NotificationSettingsRequest true "Settings"
// @Success 201 {object} entity.NotificationSettings
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/notifications/settings [post]
func (h *NotificationHandler) CreateSettings(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotifNotAuthenticated})
		return
	}

	var req NotificationSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settings := &entity.NotificationSettings{
		UserID:               userID.(string),
		Channel:              entity.NotificationChannel(req.Channel),
		Enabled:              req.Enabled,
		EmailAddress:         req.EmailAddress,
		WebhookURL:           req.WebhookURL,
		NotifyOnStart:        req.NotifyOnStart,
		NotifyOnComplete:     req.NotifyOnComplete,
		NotifyOnFailure:      req.NotifyOnFailure,
		NotifyOnScoreAlert:   req.NotifyOnScoreAlert,
		ScoreAlertThreshold:  req.ScoreAlertThreshold,
		NotifyOnAgentOffline: req.NotifyOnAgentOffline,
	}

	if h.notificationService.CreateSettings(c.Request.Context(), settings) != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create settings"})
		return
	}

	c.JSON(http.StatusCreated, settings)
}

// UpdateSettings godoc
// @Summary Update notification settings
// @Description Update notification settings for the current user
// @Tags notifications
// @Accept json
// @Produce json
// @Param body body NotificationSettingsRequest true "Settings"
// @Success 200 {object} entity.NotificationSettings
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/notifications/settings [put]
func (h *NotificationHandler) UpdateSettings(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotifNotAuthenticated})
		return
	}

	var req NotificationSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing settings
	settings, err := h.notificationService.GetSettingsByUserID(c.Request.Context(), userID.(string))
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": errSettingsNotFound})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": errFailedToGetSettings})
		return
	}
	if settings == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errSettingsNotFound})
		return
	}

	// Update fields
	settings.Channel = entity.NotificationChannel(req.Channel)
	settings.Enabled = req.Enabled
	settings.EmailAddress = req.EmailAddress
	settings.WebhookURL = req.WebhookURL
	settings.NotifyOnStart = req.NotifyOnStart
	settings.NotifyOnComplete = req.NotifyOnComplete
	settings.NotifyOnFailure = req.NotifyOnFailure
	settings.NotifyOnScoreAlert = req.NotifyOnScoreAlert
	settings.ScoreAlertThreshold = req.ScoreAlertThreshold
	settings.NotifyOnAgentOffline = req.NotifyOnAgentOffline

	if h.notificationService.UpdateSettings(c.Request.Context(), settings) != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// DeleteSettings godoc
// @Summary Delete notification settings
// @Description Delete notification settings for the current user
// @Tags notifications
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/notifications/settings [delete]
func (h *NotificationHandler) DeleteSettings(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotifNotAuthenticated})
		return
	}

	settings, err := h.notificationService.GetSettingsByUserID(c.Request.Context(), userID.(string))
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": errSettingsNotFound})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": errFailedToGetSettings})
		return
	}
	if settings == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errSettingsNotFound})
		return
	}

	if h.notificationService.DeleteSettings(c.Request.Context(), settings.ID) != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// GetSMTPConfig godoc
// @Summary Get SMTP configuration
// @Description Get SMTP configuration (without password) - admin only
// @Tags notifications
// @Accept json
// @Produce json
// @Success 200 {object} entity.SMTPConfig
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Failure 404 {object} gin.H
// @Router /api/v1/notifications/smtp [get]
func (h *NotificationHandler) GetSMTPConfig(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotifNotAuthenticated})
		return
	}

	role, roleExists := c.Get("role")
	if !roleExists || role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin role required"})
		return
	}

	config := h.notificationService.GetSMTPConfig()
	if config == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SMTP not configured"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// TestSMTPRequest represents the request to test SMTP
type TestSMTPRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// TestSMTP godoc
// @Summary Test SMTP configuration
// @Description Send a test email to verify SMTP configuration - admin only
// @Tags notifications
// @Accept json
// @Produce json
// @Param body body TestSMTPRequest true "Test email address"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/notifications/smtp/test [post]
func (h *NotificationHandler) TestSMTP(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotifNotAuthenticated})
		return
	}

	role, roleExists := c.Get("role")
	if !roleExists || role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin role required"})
		return
	}

	var req TestSMTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.notificationService.TestSMTPConnection(c.Request.Context(), req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Test email sent"})
}

// Helper function to parse int
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
