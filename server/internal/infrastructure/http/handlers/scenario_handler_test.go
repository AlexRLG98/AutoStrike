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
	"autostrike/internal/domain/service"

	"github.com/gin-gonic/gin"
)

// testScenarioRepo is a mock repository for scenario handler tests
type testScenarioRepo struct {
	scenarios map[string]*entity.Scenario
	err       error
}

func newTestScenarioRepo() *testScenarioRepo {
	return &testScenarioRepo{scenarios: make(map[string]*entity.Scenario)}
}

func (m *testScenarioRepo) Create(ctx context.Context, s *entity.Scenario) error {
	if m.err != nil {
		return m.err
	}
	m.scenarios[s.ID] = s
	return nil
}

func (m *testScenarioRepo) Update(ctx context.Context, s *entity.Scenario) error {
	if m.err != nil {
		return m.err
	}
	m.scenarios[s.ID] = s
	return nil
}

func (m *testScenarioRepo) Delete(ctx context.Context, id string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.scenarios, id)
	return nil
}

func (m *testScenarioRepo) FindByID(ctx context.Context, id string) (*entity.Scenario, error) {
	if m.err != nil {
		return nil, m.err
	}
	s, ok := m.scenarios[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return s, nil
}

func (m *testScenarioRepo) FindAll(ctx context.Context) ([]*entity.Scenario, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make([]*entity.Scenario, 0, len(m.scenarios))
	for _, s := range m.scenarios {
		result = append(result, s)
	}
	return result, nil
}

func (m *testScenarioRepo) FindByTag(ctx context.Context, tag string) ([]*entity.Scenario, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*entity.Scenario
	for _, s := range m.scenarios {
		for _, t := range s.Tags {
			if t == tag {
				result = append(result, s)
				break
			}
		}
	}
	return result, nil
}

func (m *testScenarioRepo) ImportFromYAML(ctx context.Context, path string) error {
	return m.err
}

// testTechniqueRepo is a mock technique repository for validation
type testTechniqueRepo struct {
	techniques map[string]*entity.Technique
	err        error
}

func newTestTechniqueRepo() *testTechniqueRepo {
	return &testTechniqueRepo{techniques: make(map[string]*entity.Technique)}
}

func (m *testTechniqueRepo) Create(ctx context.Context, t *entity.Technique) error  { return m.err }
func (m *testTechniqueRepo) Update(ctx context.Context, t *entity.Technique) error  { return m.err }
func (m *testTechniqueRepo) Delete(ctx context.Context, id string) error            { return m.err }
func (m *testTechniqueRepo) FindByID(ctx context.Context, id string) (*entity.Technique, error) {
	if m.err != nil {
		return nil, m.err
	}
	t, ok := m.techniques[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return t, nil
}
func (m *testTechniqueRepo) FindAll(ctx context.Context) ([]*entity.Technique, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make([]*entity.Technique, 0, len(m.techniques))
	for _, t := range m.techniques {
		result = append(result, t)
	}
	return result, nil
}
func (m *testTechniqueRepo) FindByTactic(ctx context.Context, tactic entity.TacticType) ([]*entity.Technique, error) {
	return nil, nil
}
func (m *testTechniqueRepo) FindByPlatform(ctx context.Context, platform string) ([]*entity.Technique, error) {
	return nil, nil
}
func (m *testTechniqueRepo) ImportFromYAML(ctx context.Context, path string) error { return nil }

// createTestScenarioService creates a test scenario service
func createTestScenarioService(scenarioRepo *testScenarioRepo, techRepo *testTechniqueRepo) *application.ScenarioService {
	validator := service.NewTechniqueValidator()
	return application.NewScenarioService(scenarioRepo, techRepo, validator)
}

// withAuth wraps a handler with authentication context for testing
func withAuth(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", "test-user-id")
		handler(c)
	}
}

func TestNewScenarioHandler(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)
	if handler == nil {
		t.Error("Expected non-nil handler")
	}
}

func TestScenarioHandler_RegisterRoutes(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	api := router.Group("/api")
	handler.RegisterRoutes(api)

	// Verify routes are registered
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/scenarios", nil)
	router.ServeHTTP(w, req)
	if w.Code == http.StatusNotFound {
		t.Error("Route /api/scenarios not registered")
	}
}

func TestScenarioHandler_ListScenarios(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	scenarioRepo.scenarios["s1"] = &entity.Scenario{
		ID:   "s1",
		Name: "Test Scenario",
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T1082"}, Order: 1},
		},
	}
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.GET("/scenarios", withAuth(handler.ListScenarios))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/scenarios", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestScenarioHandler_ListScenarios_Error(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	scenarioRepo.err = errors.New("database error")
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.GET("/scenarios", withAuth(handler.ListScenarios))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/scenarios", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestScenarioHandler_GetScenario(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	scenarioRepo.scenarios["s1"] = &entity.Scenario{
		ID:   "s1",
		Name: "Test Scenario",
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T1082"}, Order: 1},
		},
	}
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.GET("/scenarios/:id", withAuth(handler.GetScenario))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/scenarios/s1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var scenario entity.Scenario
	if err := json.NewDecoder(w.Body).Decode(&scenario); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}
	if scenario.ID != "s1" {
		t.Errorf("Expected scenario ID s1, got %s", scenario.ID)
	}
}

func TestScenarioHandler_GetScenario_NotFound(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.GET("/scenarios/:id", withAuth(handler.GetScenario))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/scenarios/nonexistent", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestScenarioHandler_GetScenariosByTag(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	scenarioRepo.scenarios["s1"] = &entity.Scenario{
		ID:   "s1",
		Name: "Test Scenario",
		Tags: []string{"apt29", "discovery"},
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T1082"}, Order: 1},
		},
	}
	scenarioRepo.scenarios["s2"] = &entity.Scenario{
		ID:   "s2",
		Name: "Another Scenario",
		Tags: []string{"execution"},
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T1059"}, Order: 1},
		},
	}
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.GET("/scenarios/tag/:tag", withAuth(handler.GetScenariosByTag))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/scenarios/tag/apt29", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var scenarios []*entity.Scenario
	if err := json.NewDecoder(w.Body).Decode(&scenarios); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}
	if len(scenarios) != 1 {
		t.Errorf("Expected 1 scenario, got %d", len(scenarios))
	}
}

func TestScenarioHandler_CreateScenario(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	techRepo := newTestTechniqueRepo()
	techRepo.techniques["T1082"] = &entity.Technique{
		ID:        "T1082",
		Name:      "System Information Discovery",
		Tactic:    entity.TacticDiscovery,
		Platforms: []string{"windows", "linux"},
	}
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.POST("/scenarios", withAuth(handler.CreateScenario))

	body := CreateScenarioRequest{
		Name:        "New Scenario",
		Description: "A test scenario",
		Phases: []entity.Phase{
			{Name: "Discovery", Techniques: []string{"T1082"}, Order: 1},
		},
		Tags: []string{"test"},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/scenarios", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestScenarioHandler_CreateScenario_BadRequest(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.POST("/scenarios", withAuth(handler.CreateScenario))

	// Missing required fields
	body := `{"description": "missing name and phases"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/scenarios", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestScenarioHandler_UpdateScenario(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	scenarioRepo.scenarios["s1"] = &entity.Scenario{
		ID:        "s1",
		Name:      "Original Name",
		CreatedAt: time.Now(),
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T1082"}, Order: 1},
		},
	}
	techRepo := newTestTechniqueRepo()
	techRepo.techniques["T1082"] = &entity.Technique{
		ID:        "T1082",
		Name:      "System Information Discovery",
		Tactic:    entity.TacticDiscovery,
		Platforms: []string{"windows", "linux"},
	}
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.PUT("/scenarios/:id", withAuth(handler.UpdateScenario))

	body := UpdateScenarioRequest{
		Name:        "Updated Name",
		Description: "Updated description",
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T1082"}, Order: 1},
		},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/scenarios/s1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestScenarioHandler_UpdateScenario_PreservesAuthor(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	scenarioRepo.scenarios["s1"] = &entity.Scenario{
		ID:        "s1",
		Name:      "Original Name",
		Author:    "original-author",
		CreatedAt: time.Now(),
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T1082"}, Order: 1},
		},
	}
	techRepo := newTestTechniqueRepo()
	techRepo.techniques["T1082"] = &entity.Technique{
		ID:        "T1082",
		Name:      "System Information Discovery",
		Tactic:    entity.TacticDiscovery,
		Platforms: []string{"windows", "linux"},
	}
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.PUT("/scenarios/:id", withAuth(handler.UpdateScenario))

	body := UpdateScenarioRequest{
		Name:        "Updated Name",
		Description: "Updated description",
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T1082"}, Order: 1},
		},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/scenarios/s1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify Author was preserved
	updated := scenarioRepo.scenarios["s1"]
	if updated.Author != "original-author" {
		t.Errorf("Expected Author to be preserved as 'original-author', got '%s'", updated.Author)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("Expected Name to be updated to 'Updated Name', got '%s'", updated.Name)
	}
}

func TestScenarioHandler_UpdateScenario_NotFound(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.PUT("/scenarios/:id", withAuth(handler.UpdateScenario))

	body := UpdateScenarioRequest{
		Name: "Updated Name",
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T1082"}, Order: 1},
		},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/scenarios/nonexistent", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestScenarioHandler_DeleteScenario(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	scenarioRepo.scenarios["s1"] = &entity.Scenario{
		ID:   "s1",
		Name: "To Delete",
	}
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.DELETE("/scenarios/:id", withAuth(handler.DeleteScenario))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/scenarios/s1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}
}

func TestScenarioHandler_DeleteScenario_Error(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	scenarioRepo.err = errors.New("database error")
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.DELETE("/scenarios/:id", withAuth(handler.DeleteScenario))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/scenarios/s1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestScenarioHandler_GetScenariosByTag_Error(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	scenarioRepo.err = errors.New("database error")
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.GET("/scenarios/tag/:tag", withAuth(handler.GetScenariosByTag))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/scenarios/tag/apt29", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestScenarioHandler_CreateScenario_ValidationError(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	techRepo := newTestTechniqueRepo()
	// Don't add the technique - this will cause validation to fail
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.POST("/scenarios", withAuth(handler.CreateScenario))

	body := CreateScenarioRequest{
		Name:        "Test Scenario",
		Description: "A test scenario",
		Phases: []entity.Phase{
			{Name: "Discovery", Techniques: []string{"T9999"}, Order: 1}, // Invalid technique
		},
		Tags: []string{"test"},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/scenarios", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestScenarioHandler_CreateScenario_ServiceError(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	scenarioRepo.err = errors.New("database error")
	techRepo := newTestTechniqueRepo()
	techRepo.techniques["T1082"] = &entity.Technique{
		ID:        "T1082",
		Name:      "System Information Discovery",
		Tactic:    entity.TacticDiscovery,
		Platforms: []string{"windows", "linux"},
	}
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.POST("/scenarios", withAuth(handler.CreateScenario))

	body := CreateScenarioRequest{
		Name:        "Test Scenario",
		Description: "A test scenario",
		Phases: []entity.Phase{
			{Name: "Discovery", Techniques: []string{"T1082"}, Order: 1},
		},
		Tags: []string{"test"},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/scenarios", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestScenarioHandler_UpdateScenario_BadRequest(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	scenarioRepo.scenarios["s1"] = &entity.Scenario{
		ID:        "s1",
		Name:      "Original Name",
		CreatedAt: time.Now(),
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T1082"}, Order: 1},
		},
	}
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.PUT("/scenarios/:id", withAuth(handler.UpdateScenario))

	// Missing required fields
	body := `{"description": "missing name and phases"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/scenarios/s1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestScenarioHandler_UpdateScenario_ValidationError(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	scenarioRepo.scenarios["s1"] = &entity.Scenario{
		ID:        "s1",
		Name:      "Original Name",
		CreatedAt: time.Now(),
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T1082"}, Order: 1},
		},
	}
	techRepo := newTestTechniqueRepo()
	// Don't add the technique - this will cause validation to fail
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.PUT("/scenarios/:id", withAuth(handler.UpdateScenario))

	body := UpdateScenarioRequest{
		Name: "Updated Name",
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T9999"}, Order: 1}, // Invalid technique
		},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/scenarios/s1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

// testScenarioRepoWithUpdateError is a mock that succeeds on find but fails on update
type testScenarioRepoWithUpdateError struct {
	scenarios map[string]*entity.Scenario
}

func (m *testScenarioRepoWithUpdateError) Create(ctx context.Context, s *entity.Scenario) error {
	return nil
}
func (m *testScenarioRepoWithUpdateError) Update(ctx context.Context, s *entity.Scenario) error {
	return errors.New("database error on update")
}
func (m *testScenarioRepoWithUpdateError) Delete(ctx context.Context, id string) error {
	return nil
}
func (m *testScenarioRepoWithUpdateError) FindByID(ctx context.Context, id string) (*entity.Scenario, error) {
	s, ok := m.scenarios[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return s, nil
}
func (m *testScenarioRepoWithUpdateError) FindAll(ctx context.Context) ([]*entity.Scenario, error) {
	return nil, nil
}
func (m *testScenarioRepoWithUpdateError) FindByTag(ctx context.Context, tag string) ([]*entity.Scenario, error) {
	return nil, nil
}
func (m *testScenarioRepoWithUpdateError) ImportFromYAML(ctx context.Context, path string) error {
	return nil
}

func TestScenarioHandler_UpdateScenario_ServiceError(t *testing.T) {
	scenarioRepo := &testScenarioRepoWithUpdateError{
		scenarios: map[string]*entity.Scenario{
			"s1": {
				ID:        "s1",
				Name:      "Original Name",
				CreatedAt: time.Now(),
				Phases: []entity.Phase{
					{Name: "Phase 1", Techniques: []string{"T1082"}, Order: 1},
				},
			},
		},
	}
	techRepo := newTestTechniqueRepo()
	techRepo.techniques["T1082"] = &entity.Technique{
		ID:        "T1082",
		Name:      "System Information Discovery",
		Tactic:    entity.TacticDiscovery,
		Platforms: []string{"windows", "linux"},
	}
	validator := service.NewTechniqueValidator()
	svc := application.NewScenarioService(scenarioRepo, techRepo, validator)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.PUT("/scenarios/:id", withAuth(handler.UpdateScenario))

	body := UpdateScenarioRequest{
		Name:        "Updated Name",
		Description: "Updated description",
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T1082"}, Order: 1},
		},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/scenarios/s1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestScenarioHandler_ListScenarios_NilScenarios(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	// Empty repo - returns nil slice
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.GET("/scenarios", withAuth(handler.ListScenarios))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/scenarios", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Should return [] not null
	if w.Body.String() != "[]" {
		t.Errorf("Expected empty array '[]', got '%s'", w.Body.String())
	}
}

func TestScenarioHandler_GetScenariosByTag_NilScenarios(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	// Empty repo - returns nil slice
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.GET("/scenarios/tag/:tag", withAuth(handler.GetScenariosByTag))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/scenarios/tag/nonexistent", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Should return [] not null
	if w.Body.String() != "[]" {
		t.Errorf("Expected empty array '[]', got '%s'", w.Body.String())
	}
}

func TestCreateScenarioRequest_Struct(t *testing.T) {
	req := CreateScenarioRequest{
		Name:        "Test",
		Description: "Test description",
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T1082"}, Order: 1},
		},
		Tags: []string{"test"},
	}

	if req.Name != "Test" {
		t.Errorf("Name = %s, want Test", req.Name)
	}
}

func TestUpdateScenarioRequest_Struct(t *testing.T) {
	req := UpdateScenarioRequest{
		Name:        "Updated",
		Description: "Updated description",
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T1082"}, Order: 1},
		},
		Tags: []string{"updated"},
	}

	if req.Name != "Updated" {
		t.Errorf("Name = %s, want Updated", req.Name)
	}
}

func TestScenarioHandler_ExportScenarios(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	scenarioRepo.scenarios["s1"] = &entity.Scenario{
		ID:   "s1",
		Name: "Test Scenario 1",
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T1082"}, Order: 1},
		},
	}
	scenarioRepo.scenarios["s2"] = &entity.Scenario{
		ID:   "s2",
		Name: "Test Scenario 2",
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T1059"}, Order: 1},
		},
	}
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.GET("/scenarios/export", withAuth(handler.ExportScenarios))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/scenarios/export", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check Content-Disposition header
	contentDisp := w.Header().Get("Content-Disposition")
	if contentDisp == "" {
		t.Error("Expected Content-Disposition header")
	}

	var export ScenarioExport
	if err := json.Unmarshal(w.Body.Bytes(), &export); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if export.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", export.Version)
	}

	if len(export.Scenarios) != 2 {
		t.Errorf("Expected 2 scenarios, got %d", len(export.Scenarios))
	}
}

func TestScenarioHandler_ExportScenarios_WithIDs(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	scenarioRepo.scenarios["s1"] = &entity.Scenario{
		ID:   "s1",
		Name: "Test Scenario 1",
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T1082"}, Order: 1},
		},
	}
	scenarioRepo.scenarios["s2"] = &entity.Scenario{
		ID:   "s2",
		Name: "Test Scenario 2",
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T1059"}, Order: 1},
		},
	}
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.GET("/scenarios/export", withAuth(handler.ExportScenarios))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/scenarios/export?ids=s1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var export ScenarioExport
	_ = json.Unmarshal(w.Body.Bytes(), &export)

	if len(export.Scenarios) != 1 {
		t.Errorf("Expected 1 scenario, got %d", len(export.Scenarios))
	}
}

func TestScenarioHandler_ExportScenarios_NotFound(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.GET("/scenarios/export", withAuth(handler.ExportScenarios))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/scenarios/export?ids=nonexistent", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestScenarioHandler_ExportScenario(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	scenarioRepo.scenarios["s1"] = &entity.Scenario{
		ID:   "s1",
		Name: "Test Scenario",
		Phases: []entity.Phase{
			{Name: "Phase 1", Techniques: []string{"T1082"}, Order: 1},
		},
	}
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.GET("/scenarios/:id/export", withAuth(handler.ExportScenario))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/scenarios/s1/export", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var export ScenarioExport
	_ = json.Unmarshal(w.Body.Bytes(), &export)

	if len(export.Scenarios) != 1 {
		t.Errorf("Expected 1 scenario, got %d", len(export.Scenarios))
	}

	if export.Scenarios[0].ID != "s1" {
		t.Errorf("Expected scenario ID s1, got %s", export.Scenarios[0].ID)
	}
}

func TestScenarioHandler_ExportScenario_NotFound(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.GET("/scenarios/:id/export", withAuth(handler.ExportScenario))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/scenarios/nonexistent/export", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestScenarioHandler_ImportScenarios(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	techRepo := newTestTechniqueRepo()
	techRepo.techniques["T1082"] = &entity.Technique{
		ID:        "T1082",
		Name:      "System Information Discovery",
		Tactic:    entity.TacticDiscovery,
		Platforms: []string{"windows", "linux"},
	}
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.POST("/scenarios/import", withAuth(handler.ImportScenarios))

	body := ImportScenariosRequest{
		Version: "1.0",
		Scenarios: []ImportScenarioRequest{
			{
				Name:        "Imported Scenario",
				Description: "A test scenario",
				Phases: []entity.Phase{
					{Name: "Discovery", Techniques: []string{"T1082"}, Order: 1},
				},
				Tags: []string{"imported"},
			},
		},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/scenarios/import", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var response ImportScenariosResponse
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	if response.Imported != 1 {
		t.Errorf("Expected 1 imported, got %d", response.Imported)
	}
	if response.Failed != 0 {
		t.Errorf("Expected 0 failed, got %d", response.Failed)
	}
}

func TestScenarioHandler_ImportScenarios_BadRequest(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.POST("/scenarios/import", withAuth(handler.ImportScenarios))

	// Invalid JSON
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/scenarios/import", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestScenarioHandler_ImportScenarios_Empty(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	techRepo := newTestTechniqueRepo()
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.POST("/scenarios/import", withAuth(handler.ImportScenarios))

	body := ImportScenariosRequest{
		Version:   "1.0",
		Scenarios: []ImportScenarioRequest{},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/scenarios/import", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestScenarioHandler_ImportScenarios_PartialFailure(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	techRepo := newTestTechniqueRepo()
	techRepo.techniques["T1082"] = &entity.Technique{
		ID:        "T1082",
		Name:      "System Information Discovery",
		Tactic:    entity.TacticDiscovery,
		Platforms: []string{"windows", "linux"},
	}
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.POST("/scenarios/import", withAuth(handler.ImportScenarios))

	body := ImportScenariosRequest{
		Version: "1.0",
		Scenarios: []ImportScenarioRequest{
			{
				Name: "Valid Scenario",
				Phases: []entity.Phase{
					{Name: "Discovery", Techniques: []string{"T1082"}, Order: 1},
				},
			},
			{
				Name: "Invalid Scenario",
				Phases: []entity.Phase{
					{Name: "Discovery", Techniques: []string{"T9999"}, Order: 1}, // Invalid technique
				},
			},
		},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/scenarios/import", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusMultiStatus {
		t.Errorf("Expected status 207, got %d: %s", w.Code, w.Body.String())
	}

	var response ImportScenariosResponse
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	if response.Imported != 1 {
		t.Errorf("Expected 1 imported, got %d", response.Imported)
	}
	if response.Failed != 1 {
		t.Errorf("Expected 1 failed, got %d", response.Failed)
	}
	if len(response.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(response.Errors))
	}
}

func TestScenarioHandler_ImportScenarios_AllFailed(t *testing.T) {
	scenarioRepo := newTestScenarioRepo()
	techRepo := newTestTechniqueRepo()
	// Don't add any techniques - all will fail validation
	svc := createTestScenarioService(scenarioRepo, techRepo)
	handler := NewScenarioHandler(svc)

	router := gin.New()
	router.POST("/scenarios/import", withAuth(handler.ImportScenarios))

	body := ImportScenariosRequest{
		Version: "1.0",
		Scenarios: []ImportScenarioRequest{
			{
				Name: "Invalid Scenario 1",
				Phases: []entity.Phase{
					{Name: "Discovery", Techniques: []string{"T9999"}, Order: 1},
				},
			},
			{
				Name: "Invalid Scenario 2",
				Phases: []entity.Phase{
					{Name: "Discovery", Techniques: []string{"T8888"}, Order: 1},
				},
			},
		},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/scenarios/import", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var response ImportScenariosResponse
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	if response.Imported != 0 {
		t.Errorf("Expected 0 imported, got %d", response.Imported)
	}
	if response.Failed != 2 {
		t.Errorf("Expected 2 failed, got %d", response.Failed)
	}
}

func TestScenarioExport_Struct(t *testing.T) {
	export := ScenarioExport{
		Version:    "1.0",
		ExportedAt: "2024-01-01T00:00:00Z",
		Scenarios:  []*entity.Scenario{},
	}

	if export.Version != "1.0" {
		t.Errorf("Version = %s, want 1.0", export.Version)
	}
}

func TestImportScenariosRequest_Struct(t *testing.T) {
	req := ImportScenariosRequest{
		Version: "1.0",
		Scenarios: []ImportScenarioRequest{
			{Name: "Test"},
		},
	}

	if len(req.Scenarios) != 1 {
		t.Errorf("Expected 1 scenario, got %d", len(req.Scenarios))
	}
}

func TestImportScenariosResponse_Struct(t *testing.T) {
	resp := ImportScenariosResponse{
		Imported:  1,
		Failed:    0,
		Errors:    []string{},
		Scenarios: []*entity.Scenario{},
	}

	if resp.Imported != 1 {
		t.Errorf("Imported = %d, want 1", resp.Imported)
	}
}
