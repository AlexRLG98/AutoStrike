package handlers

import (
	"net/http"
	"time"

	"autostrike/internal/application"
	"autostrike/internal/domain/entity"

	"github.com/gin-gonic/gin"
)

// Schedule handler error messages
const (
	errScheduleIDRequired = "schedule ID required"
	errScheduleNotFound   = "schedule not found"
)

// ScheduleHandler handles schedule-related HTTP requests
type ScheduleHandler struct {
	scheduleService *application.ScheduleService
}

// NewScheduleHandler creates a new schedule handler
func NewScheduleHandler(scheduleService *application.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{
		scheduleService: scheduleService,
	}
}

// RegisterRoutes registers the schedule routes
func (h *ScheduleHandler) RegisterRoutes(router *gin.RouterGroup) {
	schedules := router.Group("/schedules")
	{
		schedules.GET("", h.GetAll)
		schedules.GET("/:id", h.GetByID)
		schedules.POST("", h.Create)
		schedules.PUT("/:id", h.Update)
		schedules.DELETE("/:id", h.Delete)
		schedules.POST("/:id/pause", h.Pause)
		schedules.POST("/:id/resume", h.Resume)
		schedules.POST("/:id/run", h.RunNow)
		schedules.GET("/:id/runs", h.GetRuns)
	}
}

// CreateScheduleRequest represents the request to create a schedule
type CreateScheduleRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	ScenarioID  string `json:"scenario_id" binding:"required"`
	AgentPaw    string `json:"agent_paw"`
	Frequency   string `json:"frequency" binding:"required,oneof=once hourly daily weekly monthly cron"`
	CronExpr    string `json:"cron_expr"`
	SafeMode    bool   `json:"safe_mode"`
	StartAt     string `json:"start_at"`
}

// GetAll godoc
// @Summary Get all schedules
// @Description Get all schedules
// @Tags schedules
// @Accept json
// @Produce json
// @Success 200 {array} entity.Schedule
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/schedules [get]
func (h *ScheduleHandler) GetAll(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotAuthenticated})
		return
	}

	schedules, err := h.scheduleService.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get schedules"})
		return
	}

	if schedules == nil {
		schedules = []*entity.Schedule{}
	}

	c.JSON(http.StatusOK, schedules)
}

// GetByID godoc
// @Summary Get schedule by ID
// @Description Get a schedule by its ID
// @Tags schedules
// @Accept json
// @Produce json
// @Param id path string true "Schedule ID"
// @Success 200 {object} entity.Schedule
// @Failure 401 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/schedules/{id} [get]
func (h *ScheduleHandler) GetByID(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotAuthenticated})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errScheduleIDRequired})
		return
	}

	schedule, err := h.scheduleService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get schedule"})
		return
	}
	if schedule == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errScheduleNotFound})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// Create godoc
// @Summary Create a schedule
// @Description Create a new schedule
// @Tags schedules
// @Accept json
// @Produce json
// @Param body body CreateScheduleRequest true "Schedule data"
// @Success 201 {object} entity.Schedule
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/schedules [post]
func (h *ScheduleHandler) Create(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotAuthenticated})
		return
	}

	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var startAt *time.Time
	if req.StartAt != "" {
		t, err := time.Parse(time.RFC3339, req.StartAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_at format, use RFC3339"})
			return
		}
		startAt = &t
	}

	createReq := &application.CreateScheduleRequest{
		Name:        req.Name,
		Description: req.Description,
		ScenarioID:  req.ScenarioID,
		AgentPaw:    req.AgentPaw,
		Frequency:   entity.ScheduleFrequency(req.Frequency),
		CronExpr:    req.CronExpr,
		SafeMode:    req.SafeMode,
		StartAt:     startAt,
	}

	schedule, err := h.scheduleService.Create(c.Request.Context(), createReq, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, schedule)
}

// Update godoc
// @Summary Update a schedule
// @Description Update an existing schedule
// @Tags schedules
// @Accept json
// @Produce json
// @Param id path string true "Schedule ID"
// @Param body body CreateScheduleRequest true "Schedule data"
// @Success 200 {object} entity.Schedule
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/schedules/{id} [put]
func (h *ScheduleHandler) Update(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotAuthenticated})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errScheduleIDRequired})
		return
	}

	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var startAt *time.Time
	if req.StartAt != "" {
		t, err := time.Parse(time.RFC3339, req.StartAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_at format, use RFC3339"})
			return
		}
		startAt = &t
	}

	updateReq := &application.CreateScheduleRequest{
		Name:        req.Name,
		Description: req.Description,
		ScenarioID:  req.ScenarioID,
		AgentPaw:    req.AgentPaw,
		Frequency:   entity.ScheduleFrequency(req.Frequency),
		CronExpr:    req.CronExpr,
		SafeMode:    req.SafeMode,
		StartAt:     startAt,
	}

	schedule, err := h.scheduleService.Update(c.Request.Context(), id, updateReq)
	if err != nil {
		if err.Error() == errScheduleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// Delete godoc
// @Summary Delete a schedule
// @Description Delete a schedule
// @Tags schedules
// @Accept json
// @Produce json
// @Param id path string true "Schedule ID"
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/schedules/{id} [delete]
func (h *ScheduleHandler) Delete(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotAuthenticated})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errScheduleIDRequired})
		return
	}

	if h.scheduleService.Delete(c.Request.Context(), id) != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete schedule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// Pause godoc
// @Summary Pause a schedule
// @Description Pause a schedule
// @Tags schedules
// @Accept json
// @Produce json
// @Param id path string true "Schedule ID"
// @Success 200 {object} entity.Schedule
// @Failure 401 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/schedules/{id}/pause [post]
func (h *ScheduleHandler) Pause(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotAuthenticated})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errScheduleIDRequired})
		return
	}

	schedule, err := h.scheduleService.Pause(c.Request.Context(), id)
	if err != nil {
		if err.Error() == errScheduleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// Resume godoc
// @Summary Resume a schedule
// @Description Resume a paused schedule
// @Tags schedules
// @Accept json
// @Produce json
// @Param id path string true "Schedule ID"
// @Success 200 {object} entity.Schedule
// @Failure 401 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/schedules/{id}/resume [post]
func (h *ScheduleHandler) Resume(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotAuthenticated})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errScheduleIDRequired})
		return
	}

	schedule, err := h.scheduleService.Resume(c.Request.Context(), id)
	if err != nil {
		if err.Error() == errScheduleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// RunNow godoc
// @Summary Run schedule immediately
// @Description Manually trigger a schedule to run now
// @Tags schedules
// @Accept json
// @Produce json
// @Param id path string true "Schedule ID"
// @Success 200 {object} entity.ScheduleRun
// @Failure 401 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/schedules/{id}/run [post]
func (h *ScheduleHandler) RunNow(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotAuthenticated})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errScheduleIDRequired})
		return
	}

	run, err := h.scheduleService.RunNow(c.Request.Context(), id)
	if err != nil {
		if err.Error() == errScheduleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, run)
}

// GetRuns godoc
// @Summary Get schedule runs
// @Description Get recent runs for a schedule
// @Tags schedules
// @Accept json
// @Produce json
// @Param id path string true "Schedule ID"
// @Param limit query int false "Limit (default: 20)"
// @Success 200 {array} entity.ScheduleRun
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/schedules/{id}/runs [get]
func (h *ScheduleHandler) GetRuns(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errNotAuthenticated})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errScheduleIDRequired})
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	runs, err := h.scheduleService.GetRuns(c.Request.Context(), id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get runs"})
		return
	}

	if runs == nil {
		runs = []*entity.ScheduleRun{}
	}

	c.JSON(http.StatusOK, runs)
}
