package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"autostrike/internal/application"
	"autostrike/internal/domain/entity"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// mockScheduleRepo implements repository.ScheduleRepository for testing
type mockScheduleRepo struct {
	schedules  map[string]*entity.Schedule
	runs       map[string][]*entity.ScheduleRun
	createErr  error
	updateErr  error
	deleteErr  error
	findErr    error
	createRunErr error
}

func newMockScheduleRepo() *mockScheduleRepo {
	return &mockScheduleRepo{
		schedules: make(map[string]*entity.Schedule),
		runs:      make(map[string][]*entity.ScheduleRun),
	}
}

func (m *mockScheduleRepo) Create(ctx context.Context, schedule *entity.Schedule) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.schedules[schedule.ID] = schedule
	return nil
}

func (m *mockScheduleRepo) Update(ctx context.Context, schedule *entity.Schedule) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.schedules[schedule.ID] = schedule
	return nil
}

func (m *mockScheduleRepo) Delete(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.schedules, id)
	return nil
}

func (m *mockScheduleRepo) FindByID(ctx context.Context, id string) (*entity.Schedule, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	if s, ok := m.schedules[id]; ok {
		return s, nil
	}
	return nil, nil
}

func (m *mockScheduleRepo) FindAll(ctx context.Context) ([]*entity.Schedule, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	var result []*entity.Schedule
	for _, s := range m.schedules {
		result = append(result, s)
	}
	return result, nil
}

func (m *mockScheduleRepo) FindByStatus(ctx context.Context, status entity.ScheduleStatus) ([]*entity.Schedule, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	var result []*entity.Schedule
	for _, s := range m.schedules {
		if s.Status == status {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockScheduleRepo) FindActiveSchedulesDue(ctx context.Context, now time.Time) ([]*entity.Schedule, error) {
	return nil, nil
}

func (m *mockScheduleRepo) FindByScenarioID(ctx context.Context, scenarioID string) ([]*entity.Schedule, error) {
	var result []*entity.Schedule
	for _, s := range m.schedules {
		if s.ScenarioID == scenarioID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockScheduleRepo) CreateRun(ctx context.Context, run *entity.ScheduleRun) error {
	if m.createRunErr != nil {
		return m.createRunErr
	}
	m.runs[run.ScheduleID] = append(m.runs[run.ScheduleID], run)
	return nil
}

func (m *mockScheduleRepo) UpdateRun(ctx context.Context, run *entity.ScheduleRun) error {
	return nil
}

func (m *mockScheduleRepo) FindRunsByScheduleID(ctx context.Context, scheduleID string, limit int) ([]*entity.ScheduleRun, error) {
	runs := m.runs[scheduleID]
	if len(runs) > limit {
		runs = runs[:limit]
	}
	return runs, nil
}

// setupRealScheduleHandler creates a handler with real service using mock repo
func setupRealScheduleHandler(repo *mockScheduleRepo) (*ScheduleHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	// Create real service with mock repo (nil execution service for most tests)
	service := application.NewScheduleService(repo, nil, logger)
	handler := NewScheduleHandler(service)

	router := gin.New()
	api := router.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Next()
	})
	handler.RegisterRoutes(api)

	return handler, router
}

// setupRealScheduleHandlerNoAuth creates a handler without auth middleware
func setupRealScheduleHandlerNoAuth(repo *mockScheduleRepo) (*ScheduleHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	service := application.NewScheduleService(repo, nil, logger)
	handler := NewScheduleHandler(service)

	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	return handler, router
}

func TestScheduleHandler_GetAll_Success(t *testing.T) {
	repo := newMockScheduleRepo()
	repo.schedules["sched-1"] = &entity.Schedule{
		ID:   "sched-1",
		Name: "Test Schedule",
	}
	_, router := setupRealScheduleHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var schedules []*entity.Schedule
	if err := json.Unmarshal(w.Body.Bytes(), &schedules); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(schedules) != 1 {
		t.Errorf("len(schedules) = %d, want 1", len(schedules))
	}
}

func TestScheduleHandler_GetAll_Empty(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var schedules []*entity.Schedule
	if err := json.Unmarshal(w.Body.Bytes(), &schedules); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(schedules) != 0 {
		t.Errorf("len(schedules) = %d, want 0", len(schedules))
	}
}

func TestScheduleHandler_GetAll_Unauthorized(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandlerNoAuth(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestScheduleHandler_GetAll_Error(t *testing.T) {
	repo := newMockScheduleRepo()
	repo.findErr = errors.New("database error")
	_, router := setupRealScheduleHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestScheduleHandler_GetByID_Success(t *testing.T) {
	repo := newMockScheduleRepo()
	repo.schedules["sched-1"] = &entity.Schedule{
		ID:   "sched-1",
		Name: "Test Schedule",
	}
	_, router := setupRealScheduleHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/sched-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var schedule entity.Schedule
	if err := json.Unmarshal(w.Body.Bytes(), &schedule); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if schedule.Name != "Test Schedule" {
		t.Errorf("Name = %q, want %q", schedule.Name, "Test Schedule")
	}
}

func TestScheduleHandler_GetByID_NotFound(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestScheduleHandler_GetByID_Unauthorized(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandlerNoAuth(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/sched-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestScheduleHandler_GetByID_Error(t *testing.T) {
	repo := newMockScheduleRepo()
	repo.findErr = errors.New("database error")
	_, router := setupRealScheduleHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/sched-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestScheduleHandler_Create_Success(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandler(repo)

	body := CreateScheduleRequest{
		Name:       "New Schedule",
		ScenarioID: "scenario-1",
		Frequency:  "daily",
		SafeMode:   true,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Status = %d, want %d. Body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var schedule entity.Schedule
	if err := json.Unmarshal(w.Body.Bytes(), &schedule); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if schedule.Name != "New Schedule" {
		t.Errorf("Name = %q, want %q", schedule.Name, "New Schedule")
	}
}

func TestScheduleHandler_Create_WithStartAt(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandler(repo)

	startAt := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	body := CreateScheduleRequest{
		Name:       "Scheduled Task",
		ScenarioID: "scenario-1",
		Frequency:  "once",
		StartAt:    startAt,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Status = %d, want %d. Body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestScheduleHandler_Create_InvalidStartAt(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandler(repo)

	body := CreateScheduleRequest{
		Name:       "Scheduled Task",
		ScenarioID: "scenario-1",
		Frequency:  "once",
		StartAt:    "invalid-date",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestScheduleHandler_Create_InvalidJSON(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestScheduleHandler_Create_MissingFields(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandler(repo)

	body := map[string]string{
		"name": "Test",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestScheduleHandler_Create_Unauthorized(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandlerNoAuth(repo)

	body := CreateScheduleRequest{
		Name:       "New Schedule",
		ScenarioID: "scenario-1",
		Frequency:  "daily",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestScheduleHandler_Create_Error(t *testing.T) {
	repo := newMockScheduleRepo()
	repo.createErr = errors.New("database error")
	_, router := setupRealScheduleHandler(repo)

	body := CreateScheduleRequest{
		Name:       "New Schedule",
		ScenarioID: "scenario-1",
		Frequency:  "daily",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestScheduleHandler_Update_Success(t *testing.T) {
	repo := newMockScheduleRepo()
	repo.schedules["sched-1"] = &entity.Schedule{
		ID:     "sched-1",
		Name:   "Old Name",
		Status: entity.ScheduleStatusActive,
	}
	_, router := setupRealScheduleHandler(repo)

	body := CreateScheduleRequest{
		Name:       "Updated Name",
		ScenarioID: "scenario-1",
		Frequency:  "hourly",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/schedules/sched-1", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var schedule entity.Schedule
	if err := json.Unmarshal(w.Body.Bytes(), &schedule); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if schedule.Name != "Updated Name" {
		t.Errorf("Name = %q, want %q", schedule.Name, "Updated Name")
	}
}

func TestScheduleHandler_Update_WithStartAt(t *testing.T) {
	repo := newMockScheduleRepo()
	repo.schedules["sched-1"] = &entity.Schedule{
		ID:     "sched-1",
		Name:   "Old Name",
		Status: entity.ScheduleStatusActive,
	}
	_, router := setupRealScheduleHandler(repo)

	startAt := time.Now().Add(48 * time.Hour).Format(time.RFC3339)
	body := CreateScheduleRequest{
		Name:       "Updated Name",
		ScenarioID: "scenario-1",
		Frequency:  "once",
		StartAt:    startAt,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/schedules/sched-1", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestScheduleHandler_Update_InvalidStartAt(t *testing.T) {
	repo := newMockScheduleRepo()
	repo.schedules["sched-1"] = &entity.Schedule{
		ID:     "sched-1",
		Name:   "Old Name",
		Status: entity.ScheduleStatusActive,
	}
	_, router := setupRealScheduleHandler(repo)

	body := CreateScheduleRequest{
		Name:       "Updated Name",
		ScenarioID: "scenario-1",
		Frequency:  "once",
		StartAt:    "bad-date-format",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/schedules/sched-1", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestScheduleHandler_Update_NotFound(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandler(repo)

	body := CreateScheduleRequest{
		Name:       "Test",
		ScenarioID: "scenario-1",
		Frequency:  "daily",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/schedules/nonexistent", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestScheduleHandler_Update_Unauthorized(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandlerNoAuth(repo)

	body := CreateScheduleRequest{
		Name:       "Test",
		ScenarioID: "scenario-1",
		Frequency:  "daily",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/schedules/sched-1", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestScheduleHandler_Update_InvalidJSON(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandler(repo)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/schedules/sched-1", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestScheduleHandler_Delete_Success(t *testing.T) {
	repo := newMockScheduleRepo()
	repo.schedules["sched-1"] = &entity.Schedule{ID: "sched-1"}
	_, router := setupRealScheduleHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/schedules/sched-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestScheduleHandler_Delete_Unauthorized(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandlerNoAuth(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/schedules/sched-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestScheduleHandler_Delete_Error(t *testing.T) {
	repo := newMockScheduleRepo()
	repo.deleteErr = errors.New("database error")
	_, router := setupRealScheduleHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/schedules/sched-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestScheduleHandler_Pause_Success(t *testing.T) {
	repo := newMockScheduleRepo()
	repo.schedules["sched-1"] = &entity.Schedule{
		ID:     "sched-1",
		Status: entity.ScheduleStatusActive,
	}
	_, router := setupRealScheduleHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/sched-1/pause", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var schedule entity.Schedule
	if err := json.Unmarshal(w.Body.Bytes(), &schedule); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if schedule.Status != entity.ScheduleStatusPaused {
		t.Errorf("Status = %q, want %q", schedule.Status, entity.ScheduleStatusPaused)
	}
}

func TestScheduleHandler_Pause_NotFound(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/nonexistent/pause", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestScheduleHandler_Pause_Unauthorized(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandlerNoAuth(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/sched-1/pause", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestScheduleHandler_Resume_Success(t *testing.T) {
	repo := newMockScheduleRepo()
	repo.schedules["sched-1"] = &entity.Schedule{
		ID:     "sched-1",
		Status: entity.ScheduleStatusPaused,
	}
	_, router := setupRealScheduleHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/sched-1/resume", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var schedule entity.Schedule
	if err := json.Unmarshal(w.Body.Bytes(), &schedule); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if schedule.Status != entity.ScheduleStatusActive {
		t.Errorf("Status = %q, want %q", schedule.Status, entity.ScheduleStatusActive)
	}
}

func TestScheduleHandler_Resume_NotFound(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/nonexistent/resume", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestScheduleHandler_Resume_Unauthorized(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandlerNoAuth(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/sched-1/resume", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestScheduleHandler_RunNow_NotFound(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/nonexistent/run", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestScheduleHandler_RunNow_Unauthorized(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandlerNoAuth(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/sched-1/run", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestScheduleHandler_GetRuns_Success(t *testing.T) {
	repo := newMockScheduleRepo()
	repo.runs["sched-1"] = []*entity.ScheduleRun{
		{ID: "run-1", ScheduleID: "sched-1", Status: "completed"},
		{ID: "run-2", ScheduleID: "sched-1", Status: "failed"},
	}
	_, router := setupRealScheduleHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/sched-1/runs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var runs []*entity.ScheduleRun
	if err := json.Unmarshal(w.Body.Bytes(), &runs); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(runs) != 2 {
		t.Errorf("len(runs) = %d, want 2", len(runs))
	}
}

func TestScheduleHandler_GetRuns_WithLimit(t *testing.T) {
	repo := newMockScheduleRepo()
	repo.runs["sched-1"] = []*entity.ScheduleRun{
		{ID: "run-1", ScheduleID: "sched-1"},
		{ID: "run-2", ScheduleID: "sched-1"},
		{ID: "run-3", ScheduleID: "sched-1"},
	}
	_, router := setupRealScheduleHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/sched-1/runs?limit=2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var runs []*entity.ScheduleRun
	if err := json.Unmarshal(w.Body.Bytes(), &runs); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(runs) != 2 {
		t.Errorf("len(runs) = %d, want 2", len(runs))
	}
}

func TestScheduleHandler_GetRuns_InvalidLimit(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandler(repo)

	// Invalid limit should use default (20)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/sched-1/runs?limit=invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestScheduleHandler_GetRuns_Empty(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/sched-1/runs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var runs []*entity.ScheduleRun
	if err := json.Unmarshal(w.Body.Bytes(), &runs); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(runs) != 0 {
		t.Errorf("len(runs) = %d, want 0", len(runs))
	}
}

func TestScheduleHandler_GetRuns_Unauthorized(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandlerNoAuth(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/sched-1/runs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestNewScheduleHandler(t *testing.T) {
	handler := NewScheduleHandler(nil)
	if handler == nil {
		t.Error("NewScheduleHandler returned nil")
	}
}

func TestScheduleHandler_RegisterRoutes(t *testing.T) {
	handler := NewScheduleHandler(nil)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")

	// Should not panic
	handler.RegisterRoutes(api)
}

func TestScheduleHandler_Create_CronFrequency(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandler(repo)

	body := CreateScheduleRequest{
		Name:       "Cron Schedule",
		ScenarioID: "scenario-1",
		Frequency:  "cron",
		CronExpr:   "0 * * * *",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Status = %d, want %d. Body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestScheduleHandler_Create_CronFrequency_MissingExpr(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandler(repo)

	body := CreateScheduleRequest{
		Name:       "Cron Schedule",
		ScenarioID: "scenario-1",
		Frequency:  "cron",
		// Missing CronExpr
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestScheduleHandler_Create_CronFrequency_InvalidExpr(t *testing.T) {
	repo := newMockScheduleRepo()
	_, router := setupRealScheduleHandler(repo)

	body := CreateScheduleRequest{
		Name:       "Cron Schedule",
		ScenarioID: "scenario-1",
		Frequency:  "cron",
		CronExpr:   "invalid cron",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
