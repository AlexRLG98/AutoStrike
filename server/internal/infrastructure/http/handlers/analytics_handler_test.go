package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"autostrike/internal/application"
	"autostrike/internal/domain/entity"

	"github.com/gin-gonic/gin"
)

// withAuthAnalytics wraps a handler with authentication context for testing
func withAuthAnalytics(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", "test-user-id")
		handler(c)
	}
}

func TestNewAnalyticsHandler(t *testing.T) {
	service := application.NewAnalyticsService(&mockResultRepoForHandler{})
	handler := NewAnalyticsHandler(service)

	if handler == nil {
		t.Fatal("NewAnalyticsHandler returned nil")
	}
}

func TestAnalyticsHandler_RegisterRoutes(t *testing.T) {
	service := application.NewAnalyticsService(&mockResultRepoForHandler{})
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	routes := router.Routes()
	expectedPaths := map[string]string{
		"/api/v1/analytics/compare": "GET",
		"/api/v1/analytics/trend":   "GET",
		"/api/v1/analytics/summary": "GET",
		"/api/v1/analytics/period":  "GET",
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

func TestAnalyticsHandler_CompareScores(t *testing.T) {
	now := time.Now()
	completedAt := now.Add(10 * time.Minute)
	repo := &mockResultRepoForHandler{
		executions: []*entity.Execution{
			{
				ID:          "exec-1",
				ScenarioID:  "scenario-1",
				Status:      entity.ExecutionCompleted,
				StartedAt:   now.AddDate(0, 0, -2),
				CompletedAt: &completedAt,
				Score: &entity.SecurityScore{
					Overall:    80.0,
					Blocked:    4,
					Detected:   2,
					Successful: 4,
					Total:      10,
				},
			},
		},
	}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/compare", withAuthAnalytics(handler.CompareScores))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/compare", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response application.ScoreComparison
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Current.ExecutionCount != 1 {
		t.Errorf("Current.ExecutionCount = %d, want 1", response.Current.ExecutionCount)
	}
}

func TestAnalyticsHandler_CompareScores_WithDays(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/compare", withAuthAnalytics(handler.CompareScores))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/compare?days=30", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAnalyticsHandler_GetScoreTrend(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/trend", withAuthAnalytics(handler.GetScoreTrend))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/trend", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response application.ScoreTrend
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Period != "30d" {
		t.Errorf("Period = %q, want '30d'", response.Period)
	}
}

func TestAnalyticsHandler_GetScoreTrend_WithDays(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/trend", withAuthAnalytics(handler.GetScoreTrend))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/trend?days=7", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response application.ScoreTrend
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Period != "7d" {
		t.Errorf("Period = %q, want '7d'", response.Period)
	}
}

func TestAnalyticsHandler_GetExecutionSummary(t *testing.T) {
	now := time.Now()
	completedAt := now.Add(10 * time.Minute)
	repo := &mockResultRepoForHandler{
		executions: []*entity.Execution{
			{
				ID:          "exec-1",
				ScenarioID:  "scenario-1",
				Status:      entity.ExecutionCompleted,
				StartedAt:   now.AddDate(0, 0, -2),
				CompletedAt: &completedAt,
				Score: &entity.SecurityScore{
					Overall:    80.0,
					Blocked:    4,
					Detected:   2,
					Successful: 4,
					Total:      10,
				},
			},
		},
	}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/summary", withAuthAnalytics(handler.GetExecutionSummary))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/summary", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response application.ExecutionSummary
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.TotalExecutions != 1 {
		t.Errorf("TotalExecutions = %d, want 1", response.TotalExecutions)
	}
}

func TestAnalyticsHandler_GetPeriodStats_Success(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/period", withAuthAnalytics(handler.GetPeriodStats))

	now := time.Now()
	start := url.QueryEscape(now.AddDate(0, 0, -7).Format(time.RFC3339))
	end := url.QueryEscape(now.Format(time.RFC3339))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/period?start="+start+"&end="+end, nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAnalyticsHandler_GetPeriodStats_MissingStart(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/period", withAuthAnalytics(handler.GetPeriodStats))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/period?end="+url.QueryEscape(time.Now().Format(time.RFC3339)), nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAnalyticsHandler_GetPeriodStats_MissingEnd(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/period", withAuthAnalytics(handler.GetPeriodStats))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/period?start="+url.QueryEscape(time.Now().Format(time.RFC3339)), nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAnalyticsHandler_GetPeriodStats_InvalidStartFormat(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/period", withAuthAnalytics(handler.GetPeriodStats))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/period?start=invalid&end="+url.QueryEscape(time.Now().Format(time.RFC3339)), nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAnalyticsHandler_GetPeriodStats_InvalidEndFormat(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/period", withAuthAnalytics(handler.GetPeriodStats))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/period?start="+url.QueryEscape(time.Now().Format(time.RFC3339))+"&end=invalid", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAnalyticsHandler_GetPeriodStats_EndBeforeStart(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/period", withAuthAnalytics(handler.GetPeriodStats))

	now := time.Now()
	end := url.QueryEscape(now.AddDate(0, 0, -7).Format(time.RFC3339))
	start := url.QueryEscape(now.Format(time.RFC3339))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/period?start="+start+"&end="+end, nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAnalyticsHandler_CompareScores_InvalidDays(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/compare", withAuthAnalytics(handler.CompareScores))

	// Invalid days should fall back to default
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/compare?days=invalid", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAnalyticsHandler_CompareScores_DaysOutOfRange(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/compare", withAuthAnalytics(handler.CompareScores))

	// Days > 365 should fall back to default
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/compare?days=1000", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// mockResultRepoForHandler implements repository.ResultRepository for handler tests
type mockResultRepoForHandler struct {
	executions []*entity.Execution
}

func (m *mockResultRepoForHandler) CreateExecution(ctx context.Context, execution *entity.Execution) error {
	return nil
}

func (m *mockResultRepoForHandler) UpdateExecution(ctx context.Context, execution *entity.Execution) error {
	return nil
}

func (m *mockResultRepoForHandler) FindExecutionByID(ctx context.Context, id string) (*entity.Execution, error) {
	return nil, nil
}

func (m *mockResultRepoForHandler) FindExecutionsByScenario(ctx context.Context, scenarioID string) ([]*entity.Execution, error) {
	return nil, nil
}

func (m *mockResultRepoForHandler) FindRecentExecutions(ctx context.Context, limit int) ([]*entity.Execution, error) {
	return m.executions, nil
}

func (m *mockResultRepoForHandler) FindExecutionsByDateRange(ctx context.Context, start, end time.Time) ([]*entity.Execution, error) {
	var results []*entity.Execution
	for _, e := range m.executions {
		if !e.StartedAt.Before(start) && !e.StartedAt.After(end) {
			results = append(results, e)
		}
	}
	return results, nil
}

func (m *mockResultRepoForHandler) FindCompletedExecutionsByDateRange(ctx context.Context, start, end time.Time) ([]*entity.Execution, error) {
	var results []*entity.Execution
	for _, e := range m.executions {
		if e.Status == entity.ExecutionCompleted && !e.StartedAt.Before(start) && !e.StartedAt.After(end) {
			results = append(results, e)
		}
	}
	return results, nil
}

func (m *mockResultRepoForHandler) CreateResult(ctx context.Context, result *entity.ExecutionResult) error {
	return nil
}

func (m *mockResultRepoForHandler) UpdateResult(ctx context.Context, result *entity.ExecutionResult) error {
	return nil
}

func (m *mockResultRepoForHandler) FindResultByID(ctx context.Context, id string) (*entity.ExecutionResult, error) {
	return nil, nil
}

func (m *mockResultRepoForHandler) FindResultsByExecution(ctx context.Context, executionID string) ([]*entity.ExecutionResult, error) {
	return nil, nil
}

func (m *mockResultRepoForHandler) FindResultsByTechnique(ctx context.Context, techniqueID string) ([]*entity.ExecutionResult, error) {
	return nil, nil
}

// Tests for unauthenticated access
func TestAnalyticsHandler_CompareScores_Unauthenticated(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/compare", handler.CompareScores)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/compare", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAnalyticsHandler_GetScoreTrend_Unauthenticated(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/trend", handler.GetScoreTrend)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/trend", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAnalyticsHandler_GetExecutionSummary_Unauthenticated(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/summary", handler.GetExecutionSummary)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/summary", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAnalyticsHandler_GetPeriodStats_Unauthenticated(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/period", handler.GetPeriodStats)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/period?start=2025-01-01T00:00:00Z&end=2025-01-02T00:00:00Z", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAnalyticsHandler_GetScoreTrend_InvalidDays(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/trend", withAuthAnalytics(handler.GetScoreTrend))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/trend?days=invalid", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAnalyticsHandler_GetScoreTrend_DaysOutOfRange(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/trend", withAuthAnalytics(handler.GetScoreTrend))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/trend?days=1000", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAnalyticsHandler_GetScoreTrend_NegativeDays(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/trend", withAuthAnalytics(handler.GetScoreTrend))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/trend?days=-5", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAnalyticsHandler_GetExecutionSummary_WithDays(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/summary", withAuthAnalytics(handler.GetExecutionSummary))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/summary?days=7", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAnalyticsHandler_GetExecutionSummary_InvalidDays(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/summary", withAuthAnalytics(handler.GetExecutionSummary))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/summary?days=invalid", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAnalyticsHandler_CompareScores_NegativeDays(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/compare", withAuthAnalytics(handler.CompareScores))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/compare?days=-10", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAnalyticsHandler_GetExecutionSummary_DaysOutOfRange(t *testing.T) {
	repo := &mockResultRepoForHandler{}
	service := application.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(service)

	router := gin.New()
	router.GET("/summary", withAuthAnalytics(handler.GetExecutionSummary))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/summary?days=500", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
