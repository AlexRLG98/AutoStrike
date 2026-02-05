package handlers

import (
	"net/http"
	"strconv"
	"time"

	"autostrike/internal/application"

	"github.com/gin-gonic/gin"
)

const errAnalyticsNotAuthenticated = "not authenticated"

// AnalyticsHandler handles analytics-related HTTP requests
type AnalyticsHandler struct {
	analyticsService *application.AnalyticsService
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(analyticsService *application.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
	}
}

// RegisterRoutes registers the analytics routes
func (h *AnalyticsHandler) RegisterRoutes(router *gin.RouterGroup) {
	analytics := router.Group("/analytics")
	{
		analytics.GET("/compare", h.CompareScores)
		analytics.GET("/trend", h.GetScoreTrend)
		analytics.GET("/summary", h.GetExecutionSummary)
		analytics.GET("/period", h.GetPeriodStats)
	}
}

// CompareScores godoc
// @Summary Compare scores between periods
// @Description Compare security scores between current and previous period
// @Tags analytics
// @Accept json
// @Produce json
// @Param days query int false "Period in days (default: 7)"
// @Success 200 {object} application.ScoreComparison
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/analytics/compare [get]
func (h *AnalyticsHandler) CompareScores(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errAnalyticsNotAuthenticated})
		return
	}

	days := 7 // Default to weekly comparison
	if daysParam := c.Query("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	comparison, err := h.analyticsService.CompareScores(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compare scores"})
		return
	}

	c.JSON(http.StatusOK, comparison)
}

// GetScoreTrend godoc
// @Summary Get score trend over time
// @Description Get score trend data points over a specified period
// @Tags analytics
// @Accept json
// @Produce json
// @Param days query int false "Number of days to analyze (default: 30)"
// @Success 200 {object} application.ScoreTrend
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/analytics/trend [get]
func (h *AnalyticsHandler) GetScoreTrend(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errAnalyticsNotAuthenticated})
		return
	}

	days := 30 // Default to 30 days
	if daysParam := c.Query("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	trend, err := h.analyticsService.GetScoreTrend(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get score trend"})
		return
	}

	c.JSON(http.StatusOK, trend)
}

// GetExecutionSummary godoc
// @Summary Get execution summary
// @Description Get overall execution analytics summary
// @Tags analytics
// @Accept json
// @Produce json
// @Param days query int false "Number of days to analyze (default: 30)"
// @Success 200 {object} application.ExecutionSummary
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/analytics/summary [get]
func (h *AnalyticsHandler) GetExecutionSummary(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errAnalyticsNotAuthenticated})
		return
	}

	days := 30 // Default to 30 days
	if daysParam := c.Query("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	summary, err := h.analyticsService.GetExecutionSummary(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get execution summary"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetPeriodStats godoc
// @Summary Get period statistics
// @Description Get statistics for a specific time period
// @Tags analytics
// @Accept json
// @Produce json
// @Param start query string true "Start date (RFC3339 format)"
// @Param end query string true "End date (RFC3339 format)"
// @Success 200 {object} application.PeriodStats
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/analytics/period [get]
func (h *AnalyticsHandler) GetPeriodStats(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errAnalyticsNotAuthenticated})
		return
	}

	startStr := c.Query("start")
	endStr := c.Query("end")

	if startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start and end dates are required"})
		return
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start date format"})
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end date format"})
		return
	}

	if end.Before(start) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end date must be after start date"})
		return
	}

	stats, err := h.analyticsService.GetPeriodStats(c.Request.Context(), start, end, "custom")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get period stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}
