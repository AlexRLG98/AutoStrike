package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"autostrike/internal/application"
	"autostrike/internal/domain/entity"

	"github.com/gin-gonic/gin"
)

const (
	errScenarioNotFound        = "scenario not found"
	errScenarioNotAuthenticated = "not authenticated"
)

// ScenarioHandler handles scenario-related HTTP requests
type ScenarioHandler struct {
	service *application.ScenarioService
}

// NewScenarioHandler creates a new scenario handler
func NewScenarioHandler(service *application.ScenarioService) *ScenarioHandler {
	return &ScenarioHandler{service: service}
}

// RegisterRoutes registers scenario routes
func (h *ScenarioHandler) RegisterRoutes(r *gin.RouterGroup) {
	scenarios := r.Group("/scenarios")
	{
		scenarios.GET("", h.ListScenarios)
		scenarios.GET("/export", h.ExportScenarios)
		scenarios.POST("/import", h.ImportScenarios)
		scenarios.GET("/tag/:tag", h.GetScenariosByTag) // Must be before /:id
		scenarios.GET("/:id", h.GetScenario)
		scenarios.GET("/:id/export", h.ExportScenario)
		scenarios.POST("", h.CreateScenario)
		scenarios.PUT("/:id", h.UpdateScenario)
		scenarios.DELETE("/:id", h.DeleteScenario)
	}
}

// ListScenarios returns all scenarios
func (h *ScenarioHandler) ListScenarios(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errScenarioNotAuthenticated})
		return
	}

	scenarios, err := h.service.GetAllScenarios(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return empty array instead of null
	if scenarios == nil {
		scenarios = []*entity.Scenario{}
	}
	c.JSON(http.StatusOK, scenarios)
}

// GetScenario returns a specific scenario
func (h *ScenarioHandler) GetScenario(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errScenarioNotAuthenticated})
		return
	}

	id := c.Param("id")

	scenario, err := h.service.GetScenario(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errScenarioNotFound})
		return
	}

	c.JSON(http.StatusOK, scenario)
}

// GetScenariosByTag returns scenarios filtered by tag
func (h *ScenarioHandler) GetScenariosByTag(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errScenarioNotAuthenticated})
		return
	}

	tag := c.Param("tag")

	scenarios, err := h.service.GetScenariosByTag(c.Request.Context(), tag)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return empty array instead of null
	if scenarios == nil {
		scenarios = []*entity.Scenario{}
	}
	c.JSON(http.StatusOK, scenarios)
}

// CreateScenarioRequest represents the request body for scenario creation
type CreateScenarioRequest struct {
	Name        string         `json:"name" binding:"required"`
	Description string         `json:"description"`
	Phases      []entity.Phase `json:"phases" binding:"required"`
	Tags        []string       `json:"tags,omitempty"`
}

// CreateScenario creates a new scenario
func (h *ScenarioHandler) CreateScenario(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errScenarioNotAuthenticated})
		return
	}

	var req CreateScenarioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	scenario := &entity.Scenario{
		Name:        req.Name,
		Description: req.Description,
		Phases:      req.Phases,
		Tags:        req.Tags,
	}

	if err := h.service.CreateScenario(c.Request.Context(), scenario); err != nil {
		// Check if it's a validation error
		if _, ok := err.(*application.ValidationError); ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, scenario)
}

// UpdateScenarioRequest represents the request body for scenario update
type UpdateScenarioRequest struct {
	Name        string         `json:"name" binding:"required"`
	Description string         `json:"description"`
	Phases      []entity.Phase `json:"phases" binding:"required"`
	Tags        []string       `json:"tags,omitempty"`
}

// UpdateScenario updates an existing scenario
func (h *ScenarioHandler) UpdateScenario(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errScenarioNotAuthenticated})
		return
	}

	id := c.Param("id")

	// Check if scenario exists
	existing, err := h.service.GetScenario(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errScenarioNotFound})
		return
	}

	var req UpdateScenarioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	scenario := &entity.Scenario{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Phases:      req.Phases,
		Tags:        req.Tags,
		Author:      existing.Author, // Preserve non-updatable fields
		CreatedAt:   existing.CreatedAt,
	}

	if err := h.service.UpdateScenario(c.Request.Context(), scenario); err != nil {
		// Check if it's a validation error
		if _, ok := err.(*application.ValidationError); ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, scenario)
}

// DeleteScenario deletes a scenario
func (h *ScenarioHandler) DeleteScenario(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errScenarioNotAuthenticated})
		return
	}

	id := c.Param("id")

	if err := h.service.DeleteScenario(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ScenarioExport represents the export format for scenarios
type ScenarioExport struct {
	Version    string             `json:"version"`
	ExportedAt string             `json:"exported_at"`
	Scenarios  []*entity.Scenario `json:"scenarios"`
}

// ExportScenarios exports all scenarios (or selected ones) as JSON
func (h *ScenarioHandler) ExportScenarios(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errScenarioNotAuthenticated})
		return
	}

	// Get optional IDs filter from query string
	idsParam := c.Query("ids")

	var scenarios []*entity.Scenario
	var err error

	if idsParam != "" {
		// Export specific scenarios
		ids := strings.Split(idsParam, ",")
		scenarios = make([]*entity.Scenario, 0, len(ids))
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			scenario, err := h.service.GetScenario(c.Request.Context(), id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("scenario %s not found", id)})
				return
			}
			scenarios = append(scenarios, scenario)
		}
	} else {
		// Export all scenarios
		scenarios, err = h.service.GetAllScenarios(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if scenarios == nil {
		scenarios = []*entity.Scenario{}
	}

	export := ScenarioExport{
		Version:    "1.0",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Scenarios:  scenarios,
	}

	// Set headers for file download
	filename := fmt.Sprintf("autostrike-scenarios-%s.json", time.Now().Format("2006-01-02"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/json")

	c.JSON(http.StatusOK, export)
}

// ExportScenario exports a single scenario as JSON
func (h *ScenarioHandler) ExportScenario(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errScenarioNotAuthenticated})
		return
	}

	id := c.Param("id")

	scenario, err := h.service.GetScenario(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errScenarioNotFound})
		return
	}

	export := ScenarioExport{
		Version:    "1.0",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Scenarios:  []*entity.Scenario{scenario},
	}

	// Set headers for file download
	filename := fmt.Sprintf("autostrike-scenario-%s.json", scenario.ID)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/json")

	c.JSON(http.StatusOK, export)
}

// ImportScenariosRequest represents the request body for importing scenarios
type ImportScenariosRequest struct {
	Version   string                   `json:"version"`
	Scenarios []ImportScenarioRequest  `json:"scenarios" binding:"required"`
}

// ImportScenarioRequest represents a single scenario in the import request
type ImportScenarioRequest struct {
	Name        string         `json:"name" binding:"required"`
	Description string         `json:"description"`
	Phases      []entity.Phase `json:"phases" binding:"required"`
	Tags        []string       `json:"tags,omitempty"`
}

// ImportScenariosResponse represents the response for importing scenarios
type ImportScenariosResponse struct {
	Imported int                `json:"imported"`
	Failed   int                `json:"failed"`
	Errors   []string           `json:"errors,omitempty"`
	Scenarios []*entity.Scenario `json:"scenarios"`
}

// ImportScenarios imports scenarios from JSON
func (h *ScenarioHandler) ImportScenarios(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errScenarioNotAuthenticated})
		return
	}

	var req ImportScenariosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Scenarios) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no scenarios provided"})
		return
	}

	response := ImportScenariosResponse{
		Scenarios: make([]*entity.Scenario, 0, len(req.Scenarios)),
		Errors:    make([]string, 0),
	}

	for i, scenarioReq := range req.Scenarios {
		scenario := &entity.Scenario{
			Name:        scenarioReq.Name,
			Description: scenarioReq.Description,
			Phases:      scenarioReq.Phases,
			Tags:        scenarioReq.Tags,
		}

		if err := h.service.CreateScenario(c.Request.Context(), scenario); err != nil {
			response.Failed++
			response.Errors = append(response.Errors, fmt.Sprintf("scenario %d (%s): %s", i+1, scenarioReq.Name, err.Error()))
			continue
		}

		response.Imported++
		response.Scenarios = append(response.Scenarios, scenario)
	}

	// Return appropriate status code
	if response.Failed > 0 && response.Imported == 0 {
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if response.Failed > 0 {
		// Partial success
		c.JSON(http.StatusMultiStatus, response)
		return
	}

	c.JSON(http.StatusCreated, response)
}
