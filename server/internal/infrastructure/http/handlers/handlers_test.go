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
	"autostrike/internal/infrastructure/websocket"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// Mock repositories for creating test services
type mockAgentRepo struct {
	agents    map[string]*entity.Agent
	findErr   error
	createErr error
	deleteErr error
}

func newMockAgentRepo() *mockAgentRepo {
	return &mockAgentRepo{agents: make(map[string]*entity.Agent)}
}

func (m *mockAgentRepo) Create(ctx context.Context, agent *entity.Agent) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.agents[agent.Paw] = agent
	return nil
}
func (m *mockAgentRepo) Update(ctx context.Context, agent *entity.Agent) error {
	m.agents[agent.Paw] = agent
	return nil
}
func (m *mockAgentRepo) Delete(ctx context.Context, paw string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.agents, paw)
	return nil
}
func (m *mockAgentRepo) FindByPaw(ctx context.Context, paw string) (*entity.Agent, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	agent, ok := m.agents[paw]
	if !ok {
		return nil, errors.New("not found")
	}
	return agent, nil
}
func (m *mockAgentRepo) FindByPaws(ctx context.Context, paws []string) ([]*entity.Agent, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	var result []*entity.Agent
	for _, paw := range paws {
		if agent, ok := m.agents[paw]; ok {
			result = append(result, agent)
		}
	}
	return result, nil
}
func (m *mockAgentRepo) FindAll(ctx context.Context) ([]*entity.Agent, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	result := make([]*entity.Agent, 0, len(m.agents))
	for _, a := range m.agents {
		result = append(result, a)
	}
	return result, nil
}
func (m *mockAgentRepo) FindByStatus(ctx context.Context, status entity.AgentStatus) ([]*entity.Agent, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	result := make([]*entity.Agent, 0)
	for _, a := range m.agents {
		if a.Status == status {
			result = append(result, a)
		}
	}
	return result, nil
}
func (m *mockAgentRepo) FindByPlatform(ctx context.Context, platform string) ([]*entity.Agent, error) {
	return nil, nil
}
func (m *mockAgentRepo) UpdateLastSeen(ctx context.Context, paw string) error {
	if m.findErr != nil {
		return m.findErr
	}
	return nil
}

// Agent Handler Tests
func TestNewAgentHandler(t *testing.T) {
	repo := newMockAgentRepo()
	svc := application.NewAgentService(repo)
	handler := NewAgentHandler(svc)
	if handler == nil {
		t.Error("Expected non-nil handler")
	}
}

func TestAgentHandler_RegisterRoutes(t *testing.T) {
	repo := newMockAgentRepo()
	svc := application.NewAgentService(repo)
	handler := NewAgentHandler(svc)

	router := gin.New()
	api := router.Group("/api")
	handler.RegisterRoutes(api)

	// Verify routes are registered by making requests
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/agents", nil)
	router.ServeHTTP(w, req)
	// Should not be 404 (route registered)
	if w.Code == http.StatusNotFound {
		t.Error("Route /api/agents not registered")
	}
}

func TestAgentHandler_ListAgents(t *testing.T) {
	repo := newMockAgentRepo()
	repo.agents["paw1"] = &entity.Agent{Paw: "paw1", Hostname: "host1"}
	repo.agents["paw2"] = &entity.Agent{Paw: "paw2", Hostname: "host2"}
	svc := application.NewAgentService(repo)
	handler := NewAgentHandler(svc)

	router := gin.New()
	router.GET("/agents", handler.ListAgents)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/agents", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAgentHandler_ListAgents_Error(t *testing.T) {
	repo := newMockAgentRepo()
	repo.findErr = errors.New("db error")
	svc := application.NewAgentService(repo)
	handler := NewAgentHandler(svc)

	router := gin.New()
	router.GET("/agents", handler.ListAgents)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/agents", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestAgentHandler_GetAgent(t *testing.T) {
	repo := newMockAgentRepo()
	repo.agents["paw1"] = &entity.Agent{Paw: "paw1", Hostname: "host1"}
	svc := application.NewAgentService(repo)
	handler := NewAgentHandler(svc)

	router := gin.New()
	router.GET("/agents/:paw", handler.GetAgent)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/agents/paw1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAgentHandler_GetAgent_NotFound(t *testing.T) {
	repo := newMockAgentRepo()
	svc := application.NewAgentService(repo)
	handler := NewAgentHandler(svc)

	router := gin.New()
	router.GET("/agents/:paw", handler.GetAgent)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/agents/missing", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestAgentHandler_RegisterAgent(t *testing.T) {
	repo := newMockAgentRepo()
	svc := application.NewAgentService(repo)
	handler := NewAgentHandler(svc)

	router := gin.New()
	router.POST("/agents", handler.RegisterAgent)

	body := RegisterAgentRequest{
		Paw:       "new-paw",
		Hostname:  "new-host",
		Username:  "user",
		Platform:  "linux",
		Executors: []string{"sh"},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/agents", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}

func TestAgentHandler_RegisterAgent_BadRequest(t *testing.T) {
	repo := newMockAgentRepo()
	svc := application.NewAgentService(repo)
	handler := NewAgentHandler(svc)

	router := gin.New()
	router.POST("/agents", handler.RegisterAgent)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/agents", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAgentHandler_RegisterAgent_ServiceError(t *testing.T) {
	repo := newMockAgentRepo()
	repo.createErr = errors.New("create error")
	svc := application.NewAgentService(repo)
	handler := NewAgentHandler(svc)

	router := gin.New()
	router.POST("/agents", handler.RegisterAgent)

	body := RegisterAgentRequest{
		Paw:       "new-paw",
		Hostname:  "new-host",
		Username:  "user",
		Platform:  "linux",
		Executors: []string{"sh"},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/agents", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestAgentHandler_DeleteAgent(t *testing.T) {
	repo := newMockAgentRepo()
	repo.agents["paw1"] = &entity.Agent{Paw: "paw1"}
	svc := application.NewAgentService(repo)
	handler := NewAgentHandler(svc)

	router := gin.New()
	router.DELETE("/agents/:paw", handler.DeleteAgent)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/agents/paw1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}
}

func TestAgentHandler_DeleteAgent_Error(t *testing.T) {
	repo := newMockAgentRepo()
	repo.deleteErr = errors.New("delete error")
	svc := application.NewAgentService(repo)
	handler := NewAgentHandler(svc)

	router := gin.New()
	router.DELETE("/agents/:paw", handler.DeleteAgent)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/agents/paw1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestAgentHandler_Heartbeat(t *testing.T) {
	repo := newMockAgentRepo()
	repo.agents["paw1"] = &entity.Agent{Paw: "paw1"}
	svc := application.NewAgentService(repo)
	handler := NewAgentHandler(svc)

	router := gin.New()
	router.POST("/agents/:paw/heartbeat", handler.Heartbeat)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/agents/paw1/heartbeat", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAgentHandler_Heartbeat_Error(t *testing.T) {
	repo := newMockAgentRepo()
	repo.findErr = errors.New("heartbeat error")
	svc := application.NewAgentService(repo)
	handler := NewAgentHandler(svc)

	router := gin.New()
	router.POST("/agents/:paw/heartbeat", handler.Heartbeat)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/agents/paw1/heartbeat", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestAgentHandler_ListAgents_AllTrue(t *testing.T) {
	repo := newMockAgentRepo()
	repo.agents["paw1"] = &entity.Agent{Paw: "paw1", Hostname: "host1", Status: entity.AgentOnline}
	repo.agents["paw2"] = &entity.Agent{Paw: "paw2", Hostname: "host2", Status: entity.AgentOffline}
	svc := application.NewAgentService(repo)
	handler := NewAgentHandler(svc)

	router := gin.New()
	router.GET("/agents", handler.ListAgents)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/agents?all=true", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var agents []*entity.Agent
	if err := json.Unmarshal(w.Body.Bytes(), &agents); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(agents))
	}
}

func TestAgentHandler_ListAgents_AllTrue_Error(t *testing.T) {
	repo := newMockAgentRepo()
	repo.findErr = errors.New("db error")
	svc := application.NewAgentService(repo)
	handler := NewAgentHandler(svc)

	router := gin.New()
	router.GET("/agents", handler.ListAgents)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/agents?all=true", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestAgentHandler_ListAgents_NilAgents(t *testing.T) {
	// Test that nil agents are converted to empty array
	repo := newMockAgentRepo()
	// Empty repo returns nil slice
	svc := application.NewAgentService(repo)
	handler := NewAgentHandler(svc)

	router := gin.New()
	router.GET("/agents", handler.ListAgents)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/agents", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Should return [] not null
	if w.Body.String() != "[]" {
		t.Errorf("Expected empty array '[]', got '%s'", w.Body.String())
	}
}

// Technique mock repo
type mockTechniqueRepo struct {
	techniques map[string]*entity.Technique
	findErr    error
	importErr  error
	createErr  error
	updateErr  error
}

func newMockTechniqueRepo() *mockTechniqueRepo {
	return &mockTechniqueRepo{techniques: make(map[string]*entity.Technique)}
}

func (m *mockTechniqueRepo) Create(ctx context.Context, t *entity.Technique) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.techniques[t.ID] = t
	return nil
}
func (m *mockTechniqueRepo) Update(ctx context.Context, t *entity.Technique) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.techniques[t.ID] = t
	return nil
}
func (m *mockTechniqueRepo) Delete(ctx context.Context, id string) error            { return nil }
func (m *mockTechniqueRepo) FindByID(ctx context.Context, id string) (*entity.Technique, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	t, ok := m.techniques[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return t, nil
}
func (m *mockTechniqueRepo) FindAll(ctx context.Context) ([]*entity.Technique, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	result := make([]*entity.Technique, 0, len(m.techniques))
	for _, t := range m.techniques {
		result = append(result, t)
	}
	return result, nil
}
func (m *mockTechniqueRepo) FindByTactic(ctx context.Context, tactic entity.TacticType) ([]*entity.Technique, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return []*entity.Technique{}, nil
}
func (m *mockTechniqueRepo) FindByPlatform(ctx context.Context, platform string) ([]*entity.Technique, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return []*entity.Technique{}, nil
}
func (m *mockTechniqueRepo) ImportFromYAML(ctx context.Context, path string) error {
	return m.importErr
}

// Technique Handler Tests
func TestNewTechniqueHandler(t *testing.T) {
	repo := newMockTechniqueRepo()
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)
	if handler == nil {
		t.Error("Expected non-nil handler")
	}
}

func TestTechniqueHandler_RegisterRoutes(t *testing.T) {
	repo := newMockTechniqueRepo()
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	api := router.Group("/api")
	handler.RegisterRoutes(api)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/techniques", nil)
	router.ServeHTTP(w, req)

	if w.Code == http.StatusNotFound {
		t.Error("Route /api/techniques not registered")
	}
}

func TestTechniqueHandler_ListTechniques(t *testing.T) {
	repo := newMockTechniqueRepo()
	repo.techniques["T1059"] = &entity.Technique{ID: "T1059", Name: "Command Execution"}
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.GET("/techniques", handler.ListTechniques)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/techniques", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTechniqueHandler_ListTechniques_Error(t *testing.T) {
	repo := newMockTechniqueRepo()
	repo.findErr = errors.New("db error")
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.GET("/techniques", handler.ListTechniques)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/techniques", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestTechniqueHandler_GetTechnique(t *testing.T) {
	repo := newMockTechniqueRepo()
	repo.techniques["T1059"] = &entity.Technique{ID: "T1059", Name: "Command Execution"}
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.GET("/techniques/:id", handler.GetTechnique)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/techniques/T1059", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTechniqueHandler_GetTechnique_NotFound(t *testing.T) {
	repo := newMockTechniqueRepo()
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.GET("/techniques/:id", handler.GetTechnique)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/techniques/missing", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestTechniqueHandler_GetByTactic(t *testing.T) {
	repo := newMockTechniqueRepo()
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.GET("/techniques/tactic/:tactic", handler.GetByTactic)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/techniques/tactic/execution", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTechniqueHandler_GetByTactic_Error(t *testing.T) {
	repo := newMockTechniqueRepo()
	repo.findErr = errors.New("db error")
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.GET("/techniques/tactic/:tactic", handler.GetByTactic)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/techniques/tactic/execution", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestTechniqueHandler_GetByPlatform(t *testing.T) {
	repo := newMockTechniqueRepo()
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.GET("/techniques/platform/:platform", handler.GetByPlatform)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/techniques/platform/windows", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTechniqueHandler_GetByPlatform_Error(t *testing.T) {
	repo := newMockTechniqueRepo()
	repo.findErr = errors.New("db error")
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.GET("/techniques/platform/:platform", handler.GetByPlatform)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/techniques/platform/windows", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestTechniqueHandler_GetCoverage(t *testing.T) {
	repo := newMockTechniqueRepo()
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.GET("/techniques/coverage", handler.GetCoverage)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/techniques/coverage", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTechniqueHandler_GetCoverage_Error(t *testing.T) {
	repo := newMockTechniqueRepo()
	repo.findErr = errors.New("db error")
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.GET("/techniques/coverage", handler.GetCoverage)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/techniques/coverage", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestTechniqueHandler_ImportTechniques(t *testing.T) {
	repo := newMockTechniqueRepo()
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.POST("/techniques/import", handler.ImportTechniques)

	body := ImportRequest{Path: "/path/to/techniques.yaml"}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/techniques/import", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTechniqueHandler_ImportTechniques_BadRequest(t *testing.T) {
	repo := newMockTechniqueRepo()
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.POST("/techniques/import", handler.ImportTechniques)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/techniques/import", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestTechniqueHandler_ImportTechniques_Error(t *testing.T) {
	repo := newMockTechniqueRepo()
	repo.importErr = errors.New("import error")
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.POST("/techniques/import", handler.ImportTechniques)

	body := ImportRequest{Path: "/path/to/techniques.yaml"}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/techniques/import", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestTechniqueHandler_ListTechniques_NilTechniques(t *testing.T) {
	repo := newMockTechniqueRepo()
	// Empty repo returns nil slice
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.GET("/techniques", handler.ListTechniques)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/techniques", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Should return [] not null
	if w.Body.String() != "[]" {
		t.Errorf("Expected empty array '[]', got '%s'", w.Body.String())
	}
}

func TestTechniqueHandler_GetByTactic_NilTechniques(t *testing.T) {
	repo := newMockTechniqueRepo()
	// Empty repo returns nil slice for tactic
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.GET("/techniques/tactic/:tactic", handler.GetByTactic)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/techniques/tactic/nonexistent", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Should return [] not null
	if w.Body.String() != "[]" {
		t.Errorf("Expected empty array '[]', got '%s'", w.Body.String())
	}
}

func TestTechniqueHandler_GetByPlatform_NilTechniques(t *testing.T) {
	repo := newMockTechniqueRepo()
	// Empty repo returns nil slice for platform
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.GET("/techniques/platform/:platform", handler.GetByPlatform)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/techniques/platform/nonexistent", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Should return [] not null
	if w.Body.String() != "[]" {
		t.Errorf("Expected empty array '[]', got '%s'", w.Body.String())
	}
}

// Execution mock repos
type mockResultRepo struct {
	executions     map[string]*entity.Execution
	results        map[string][]*entity.ExecutionResult
	err            error
	findResultsErr error
}

func newMockResultRepo() *mockResultRepo {
	return &mockResultRepo{
		executions: make(map[string]*entity.Execution),
		results:    make(map[string][]*entity.ExecutionResult),
	}
}

func (m *mockResultRepo) CreateExecution(ctx context.Context, e *entity.Execution) error {
	if m.err != nil {
		return m.err
	}
	m.executions[e.ID] = e
	return nil
}
func (m *mockResultRepo) UpdateExecution(ctx context.Context, e *entity.Execution) error {
	if m.err != nil {
		return m.err
	}
	m.executions[e.ID] = e
	return nil
}
func (m *mockResultRepo) FindExecutionByID(ctx context.Context, id string) (*entity.Execution, error) {
	if m.err != nil {
		return nil, m.err
	}
	e, ok := m.executions[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return e, nil
}
func (m *mockResultRepo) FindExecutionsByScenario(ctx context.Context, scenarioID string) ([]*entity.Execution, error) {
	return nil, nil
}
func (m *mockResultRepo) FindRecentExecutions(ctx context.Context, limit int) ([]*entity.Execution, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make([]*entity.Execution, 0, len(m.executions))
	for _, e := range m.executions {
		result = append(result, e)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}
func (m *mockResultRepo) CreateResult(ctx context.Context, r *entity.ExecutionResult) error {
	if m.err != nil {
		return m.err
	}
	m.results[r.ExecutionID] = append(m.results[r.ExecutionID], r)
	return nil
}
func (m *mockResultRepo) UpdateResult(ctx context.Context, r *entity.ExecutionResult) error {
	return nil
}
func (m *mockResultRepo) FindResultByID(ctx context.Context, id string) (*entity.ExecutionResult, error) {
	for _, results := range m.results {
		for _, r := range results {
			if r.ID == id {
				return r, nil
			}
		}
	}
	return &entity.ExecutionResult{ID: id}, nil
}
func (m *mockResultRepo) FindResultsByExecution(ctx context.Context, executionID string) ([]*entity.ExecutionResult, error) {
	if m.findResultsErr != nil {
		return nil, m.findResultsErr
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.results[executionID], nil
}
func (m *mockResultRepo) FindResultsByTechnique(ctx context.Context, techniqueID string) ([]*entity.ExecutionResult, error) {
	return nil, nil
}
func (m *mockResultRepo) FindExecutionsByDateRange(ctx context.Context, start, end time.Time) ([]*entity.Execution, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*entity.Execution
	for _, e := range m.executions {
		if !e.StartedAt.Before(start) && !e.StartedAt.After(end) {
			result = append(result, e)
		}
	}
	return result, nil
}
func (m *mockResultRepo) FindCompletedExecutionsByDateRange(ctx context.Context, start, end time.Time) ([]*entity.Execution, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*entity.Execution
	for _, e := range m.executions {
		if e.Status == entity.ExecutionCompleted && !e.StartedAt.Before(start) && !e.StartedAt.After(end) {
			result = append(result, e)
		}
	}
	return result, nil
}

type mockScenarioRepo struct {
	scenarios map[string]*entity.Scenario
	err       error
}

func newMockScenarioRepo() *mockScenarioRepo {
	return &mockScenarioRepo{scenarios: make(map[string]*entity.Scenario)}
}

func (m *mockScenarioRepo) Create(ctx context.Context, s *entity.Scenario) error  { return m.err }
func (m *mockScenarioRepo) Update(ctx context.Context, s *entity.Scenario) error  { return m.err }
func (m *mockScenarioRepo) Delete(ctx context.Context, id string) error           { return m.err }
func (m *mockScenarioRepo) FindByID(ctx context.Context, id string) (*entity.Scenario, error) {
	if m.err != nil {
		return nil, m.err
	}
	s, ok := m.scenarios[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return s, nil
}
func (m *mockScenarioRepo) FindAll(ctx context.Context) ([]*entity.Scenario, error) { return nil, nil }
func (m *mockScenarioRepo) FindByTag(ctx context.Context, tag string) ([]*entity.Scenario, error) {
	return nil, nil
}
func (m *mockScenarioRepo) ImportFromYAML(ctx context.Context, path string) error { return nil }

// Execution Handler Tests
func TestNewExecutionHandler(t *testing.T) {
	resultRepo := newMockResultRepo()
	scenarioRepo := newMockScenarioRepo()
	techRepo := newMockTechniqueRepo()
	agentRepo := newMockAgentRepo()

	svc := application.NewExecutionService(resultRepo, scenarioRepo, techRepo, agentRepo, nil, nil)
	handler := NewExecutionHandler(svc)
	if handler == nil {
		t.Error("Expected non-nil handler")
	}
}

func TestExecutionHandler_RegisterRoutes(t *testing.T) {
	resultRepo := newMockResultRepo()
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)
	handler := NewExecutionHandler(svc)

	router := gin.New()
	api := router.Group("/api")
	handler.RegisterRoutes(api)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/executions", nil)
	router.ServeHTTP(w, req)

	if w.Code == http.StatusNotFound {
		t.Error("Route /api/executions not registered")
	}
}

func TestExecutionHandler_ListExecutions(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{ID: "e1", Status: entity.ExecutionCompleted}
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)
	handler := NewExecutionHandler(svc)

	router := gin.New()
	router.GET("/executions", handler.ListExecutions)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/executions", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestExecutionHandler_ListExecutions_Error(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.err = errors.New("db error")
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)
	handler := NewExecutionHandler(svc)

	router := gin.New()
	router.GET("/executions", handler.ListExecutions)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/executions", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestExecutionHandler_GetExecution(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{ID: "e1", Status: entity.ExecutionCompleted}
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)
	handler := NewExecutionHandler(svc)

	router := gin.New()
	router.GET("/executions/:id", handler.GetExecution)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/executions/e1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestExecutionHandler_GetExecution_NotFound(t *testing.T) {
	resultRepo := newMockResultRepo()
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)
	handler := NewExecutionHandler(svc)

	router := gin.New()
	router.GET("/executions/:id", handler.GetExecution)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/executions/missing", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestExecutionHandler_GetResults(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.results["e1"] = []*entity.ExecutionResult{
		{ID: "r1", ExecutionID: "e1"},
	}
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)
	handler := NewExecutionHandler(svc)

	router := gin.New()
	router.GET("/executions/:id/results", handler.GetResults)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/executions/e1/results", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestExecutionHandler_GetResults_Error(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.err = errors.New("db error")
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)
	handler := NewExecutionHandler(svc)

	router := gin.New()
	router.GET("/executions/:id/results", handler.GetResults)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/executions/e1/results", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestExecutionHandler_StartExecution_BadRequest(t *testing.T) {
	resultRepo := newMockResultRepo()
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)
	handler := NewExecutionHandler(svc)

	router := gin.New()
	router.POST("/executions", handler.StartExecution)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/executions", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestExecutionHandler_StartExecution_ServiceError(t *testing.T) {
	resultRepo := newMockResultRepo()
	scenarioRepo := newMockScenarioRepo()
	scenarioRepo.err = errors.New("scenario not found")
	svc := application.NewExecutionService(resultRepo, scenarioRepo, nil, nil, nil, nil)
	handler := NewExecutionHandler(svc)

	router := gin.New()
	router.POST("/executions", handler.StartExecution)

	body := StartExecutionRequest{
		ScenarioID: "s1",
		AgentPaws:  []string{"paw1"},
		SafeMode:   true,
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/executions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestExecutionHandler_StartExecution_Success(t *testing.T) {
	resultRepo := newMockResultRepo()
	scenarioRepo := newMockScenarioRepo()
	scenarioRepo.scenarios["s1"] = &entity.Scenario{
		ID:   "s1",
		Name: "Test Scenario",
		Phases: []entity.Phase{
			{Name: "Phase1", Techniques: []string{"T1059"}},
		},
	}
	techRepo := newMockTechniqueRepo()
	techRepo.techniques["T1059"] = &entity.Technique{
		ID:        "T1059",
		Platforms: []string{"linux"},
		Executors: []entity.Executor{{Type: "sh", Command: "echo test"}},
	}
	agentRepo := newMockAgentRepo()
	agentRepo.agents["paw1"] = &entity.Agent{
		Paw:       "paw1",
		Status:    entity.AgentOnline,
		Platform:  "linux",
		Executors: []string{"sh"},
		LastSeen:  time.Now(),
	}

	validator := service.NewTechniqueValidator()
	orchestrator := service.NewAttackOrchestrator(agentRepo, techRepo, validator, nil)
	calculator := service.NewScoreCalculator()

	svc := application.NewExecutionService(resultRepo, scenarioRepo, techRepo, agentRepo, orchestrator, calculator)

	// Create handler with hub to test broadcast
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	go hub.Run()

	handler := NewExecutionHandlerWithHub(svc, hub)

	router := gin.New()
	router.POST("/executions", handler.StartExecution)

	body := StartExecutionRequest{
		ScenarioID: "s1",
		AgentPaws:  []string{"paw1"},
		SafeMode:   false,
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/executions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	// Allow time for broadcast to process
	time.Sleep(10 * time.Millisecond)
}

func TestExecutionHandler_CompleteExecution(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{
		ID:        "e1",
		Status:    entity.ExecutionRunning,
		StartedAt: time.Now(),
	}
	resultRepo.results["e1"] = []*entity.ExecutionResult{}

	calculator := service.NewScoreCalculator()
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, calculator)
	handler := NewExecutionHandler(svc)

	router := gin.New()
	router.POST("/executions/:id/complete", handler.CompleteExecution)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/executions/e1/complete", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestExecutionHandler_CompleteExecution_Error(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.err = errors.New("execution not found")
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)
	handler := NewExecutionHandler(svc)

	router := gin.New()
	router.POST("/executions/:id/complete", handler.CompleteExecution)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/executions/e1/complete", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestRegisterAgentRequest_Struct(t *testing.T) {
	req := RegisterAgentRequest{
		Paw:       "test-paw",
		Hostname:  "test-host",
		Username:  "test-user",
		Platform:  "linux",
		Executors: []string{"sh", "bash"},
	}

	if req.Paw != "test-paw" {
		t.Errorf("Paw = %s, want test-paw", req.Paw)
	}
}

func TestImportRequest_Struct(t *testing.T) {
	req := ImportRequest{Path: "/path/to/file.yaml"}
	if req.Path != "/path/to/file.yaml" {
		t.Errorf("Path = %s, want /path/to/file.yaml", req.Path)
	}
}

func TestStartExecutionRequest_Struct(t *testing.T) {
	req := StartExecutionRequest{
		ScenarioID: "s1",
		AgentPaws:  []string{"paw1", "paw2"},
		SafeMode:   true,
	}

	if req.ScenarioID != "s1" {
		t.Errorf("ScenarioID = %s, want s1", req.ScenarioID)
	}
	if len(req.AgentPaws) != 2 {
		t.Errorf("AgentPaws length = %d, want 2", len(req.AgentPaws))
	}
	if !req.SafeMode {
		t.Error("SafeMode should be true")
	}
}

func TestExecutionHandler_StopExecution(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{
		ID:        "e1",
		Status:    entity.ExecutionRunning,
		StartedAt: time.Now(),
	}
	resultRepo.results["e1"] = []*entity.ExecutionResult{}

	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)
	handler := NewExecutionHandler(svc)

	router := gin.New()
	router.POST("/executions/:id/stop", handler.StopExecution)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/executions/e1/stop", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestExecutionHandler_StopExecution_NotFound(t *testing.T) {
	resultRepo := newMockResultRepo()
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)
	handler := NewExecutionHandler(svc)

	router := gin.New()
	router.POST("/executions/:id/stop", handler.StopExecution)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/executions/missing/stop", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestExecutionHandler_StopExecution_AlreadyCompleted(t *testing.T) {
	resultRepo := newMockResultRepo()
	now := time.Now()
	resultRepo.executions["e1"] = &entity.Execution{
		ID:          "e1",
		Status:      entity.ExecutionCompleted,
		StartedAt:   now,
		CompletedAt: &now,
	}

	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)
	handler := NewExecutionHandler(svc)

	router := gin.New()
	router.POST("/executions/:id/stop", handler.StopExecution)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/executions/e1/stop", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}
}

func TestExecutionHandler_StopExecution_PendingExecution(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{
		ID:        "e1",
		Status:    entity.ExecutionPending,
		StartedAt: time.Now(),
	}
	resultRepo.results["e1"] = []*entity.ExecutionResult{}

	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)
	handler := NewExecutionHandler(svc)

	router := gin.New()
	router.POST("/executions/:id/stop", handler.StopExecution)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/executions/e1/stop", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestExecutionHandler_StopExecution_InternalError(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{
		ID:        "e1",
		Status:    entity.ExecutionRunning,
		StartedAt: time.Now(),
	}
	resultRepo.findResultsErr = errors.New("database connection failed")

	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)
	handler := NewExecutionHandler(svc)

	router := gin.New()
	router.POST("/executions/:id/stop", handler.StopExecution)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/executions/e1/stop", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestNewExecutionHandlerWithHub(t *testing.T) {
	resultRepo := newMockResultRepo()
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)

	// Create handler with hub
	handler := NewExecutionHandlerWithHub(svc, nil)
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
	if handler.service != svc {
		t.Error("Service not set correctly")
	}
}

func TestExecutionHandler_BroadcastExecutionEvent_NilHub(t *testing.T) {
	resultRepo := newMockResultRepo()
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)

	// Handler with nil hub should not panic
	handler := NewExecutionHandler(svc)
	handler.broadcastExecutionEvent("test_event", "exec-123", map[string]string{"status": "test"})
	// No panic means success
}

func TestExecutionHandler_BroadcastExecutionEvent_WithHub(t *testing.T) {
	resultRepo := newMockResultRepo()
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)

	// Create a real hub
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)

	// Start hub in goroutine
	go hub.Run()

	// Create handler with hub
	handler := NewExecutionHandlerWithHub(svc, hub)

	// This should not panic and should broadcast
	handler.broadcastExecutionEvent("test_event", "exec-123", map[string]string{"status": "test"})

	// Small delay to let broadcast process
	time.Sleep(10 * time.Millisecond)
}

func TestExecutionHandler_ListExecutions_NilResults(t *testing.T) {
	resultRepo := newMockResultRepo()
	// Empty repo returns nil slice
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)
	handler := NewExecutionHandler(svc)

	router := gin.New()
	router.GET("/executions", handler.ListExecutions)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/executions", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Should return [] not null
	if w.Body.String() != "[]" {
		t.Errorf("Expected empty array '[]', got '%s'", w.Body.String())
	}
}

func TestExecutionHandler_GetResults_NilResults(t *testing.T) {
	resultRepo := newMockResultRepo()
	// No results for this execution ID
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)
	handler := NewExecutionHandler(svc)

	router := gin.New()
	router.GET("/executions/:id/results", handler.GetResults)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/executions/e1/results", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Should return [] not null
	if w.Body.String() != "[]" {
		t.Errorf("Expected empty array '[]', got '%s'", w.Body.String())
	}
}

func TestExecutionHandler_StartExecution_EmptyAgentPaws(t *testing.T) {
	resultRepo := newMockResultRepo()
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)
	handler := NewExecutionHandler(svc)

	router := gin.New()
	router.POST("/executions", handler.StartExecution)

	body := StartExecutionRequest{
		ScenarioID: "s1",
		AgentPaws:  []string{}, // Empty
		SafeMode:   true,
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/executions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestExecutionHandler_DispatchTasksToAgents_NilHub(t *testing.T) {
	resultRepo := newMockResultRepo()
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)
	handler := NewExecutionHandler(svc) // No hub

	// This should not panic
	tasks := []application.TaskDispatchInfo{
		{ResultID: "r1", AgentPaw: "paw1", TechniqueID: "T1082", Command: "echo test"},
	}
	handler.dispatchTasksToAgents(tasks)
}

func TestExecutionHandler_DispatchTasksToAgents_WithHub(t *testing.T) {
	resultRepo := newMockResultRepo()
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)

	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	go hub.Run()

	handler := NewExecutionHandlerWithHub(svc, hub)

	// Dispatch to non-existent agent - should mark as failed
	tasks := []application.TaskDispatchInfo{
		{ResultID: "r1", AgentPaw: "nonexistent", TechniqueID: "T1082", Command: "echo test"},
	}
	handler.dispatchTasksToAgents(tasks)

	// Give time for processing
	time.Sleep(10 * time.Millisecond)
}

func TestExecutionHandler_MarkResultAsFailed_NilService(t *testing.T) {
	// Handler with nil service should not panic
	handler := &ExecutionHandler{service: nil, hub: nil}
	handler.markResultAsFailed("result-123", "test reason")
}

func TestExecutionHandler_CompleteExecution_WithHub(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{
		ID:        "e1",
		Status:    entity.ExecutionRunning,
		StartedAt: time.Now(),
	}
	resultRepo.results["e1"] = []*entity.ExecutionResult{}

	calculator := service.NewScoreCalculator()
	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, calculator)

	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	go hub.Run()

	handler := NewExecutionHandlerWithHub(svc, hub)

	router := gin.New()
	router.POST("/executions/:id/complete", handler.CompleteExecution)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/executions/e1/complete", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	time.Sleep(10 * time.Millisecond)
}

func TestExecutionHandler_StopExecution_WithHub(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{
		ID:        "e1",
		Status:    entity.ExecutionRunning,
		StartedAt: time.Now(),
	}
	resultRepo.results["e1"] = []*entity.ExecutionResult{}

	svc := application.NewExecutionService(resultRepo, nil, nil, nil, nil, nil)

	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	go hub.Run()

	handler := NewExecutionHandlerWithHub(svc, hub)

	router := gin.New()
	router.POST("/executions/:id/stop", handler.StopExecution)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/executions/e1/stop", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	time.Sleep(10 * time.Millisecond)
}

// ImportTechniquesJSON Tests
func TestTechniqueHandler_ImportTechniquesJSON_Success(t *testing.T) {
	repo := newMockTechniqueRepo()
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.POST("/techniques/import/json", handler.ImportTechniquesJSON)

	body := ImportJSONRequest{
		Techniques: []*entity.Technique{
			{ID: "T1082", Name: "System Info Discovery", Tactic: entity.TacticDiscovery, Platforms: []string{"windows", "linux"}},
			{ID: "T1083", Name: "File Discovery", Tactic: entity.TacticDiscovery, Platforms: []string{"windows"}},
		},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/techniques/import/json", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response ImportJSONResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Imported != 2 {
		t.Errorf("Expected 2 imported, got %d", response.Imported)
	}
	if response.Failed != 0 {
		t.Errorf("Expected 0 failed, got %d", response.Failed)
	}
}

func TestTechniqueHandler_ImportTechniquesJSON_BadRequest(t *testing.T) {
	repo := newMockTechniqueRepo()
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.POST("/techniques/import/json", handler.ImportTechniquesJSON)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/techniques/import/json", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestTechniqueHandler_ImportTechniquesJSON_CreateFailsUpdateSucceeds(t *testing.T) {
	repo := newMockTechniqueRepo()
	// Pre-populate with existing technique
	repo.techniques["T1082"] = &entity.Technique{ID: "T1082", Name: "Old Name"}
	repo.createErr = errors.New("technique already exists")
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.POST("/techniques/import/json", handler.ImportTechniquesJSON)

	body := ImportJSONRequest{
		Techniques: []*entity.Technique{
			{ID: "T1082", Name: "Updated Name", Tactic: entity.TacticDiscovery, Platforms: []string{"windows"}},
		},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/techniques/import/json", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response ImportJSONResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Imported != 1 {
		t.Errorf("Expected 1 imported (via update), got %d", response.Imported)
	}
	if response.Failed != 0 {
		t.Errorf("Expected 0 failed, got %d", response.Failed)
	}
}

func TestTechniqueHandler_ImportTechniquesJSON_BothFail(t *testing.T) {
	repo := newMockTechniqueRepo()
	repo.createErr = errors.New("create failed")
	repo.updateErr = errors.New("update failed")
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.POST("/techniques/import/json", handler.ImportTechniquesJSON)

	body := ImportJSONRequest{
		Techniques: []*entity.Technique{
			{ID: "T1082", Name: "Test", Tactic: entity.TacticDiscovery, Platforms: []string{"windows"}},
		},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/techniques/import/json", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response ImportJSONResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Imported != 0 {
		t.Errorf("Expected 0 imported, got %d", response.Imported)
	}
	if response.Failed != 1 {
		t.Errorf("Expected 1 failed, got %d", response.Failed)
	}
	if len(response.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(response.Errors))
	}
}

func TestTechniqueHandler_ImportTechniquesJSON_PartialSuccess(t *testing.T) {
	repo := newMockTechniqueRepo()
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.POST("/techniques/import/json", handler.ImportTechniquesJSON)

	body := ImportJSONRequest{
		Techniques: []*entity.Technique{
			{ID: "T1082", Name: "System Info", Tactic: entity.TacticDiscovery, Platforms: []string{"windows"}},
			{ID: "T1083", Name: "File Discovery", Tactic: entity.TacticDiscovery, Platforms: []string{"linux"}},
			{ID: "T1057", Name: "Process Discovery", Tactic: entity.TacticDiscovery, Platforms: []string{"windows", "linux"}},
		},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/techniques/import/json", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response ImportJSONResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Imported != 3 {
		t.Errorf("Expected 3 imported, got %d", response.Imported)
	}
	if response.Failed != 0 {
		t.Errorf("Expected 0 failed, got %d", response.Failed)
	}
}

func TestTechniqueHandler_ImportTechniquesJSON_EmptyTechniques(t *testing.T) {
	repo := newMockTechniqueRepo()
	svc := application.NewTechniqueService(repo)
	handler := NewTechniqueHandler(svc)

	router := gin.New()
	router.POST("/techniques/import/json", handler.ImportTechniquesJSON)

	body := ImportJSONRequest{
		Techniques: []*entity.Technique{},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/techniques/import/json", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Empty array should still return 200 with 0 imported
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response ImportJSONResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Imported != 0 {
		t.Errorf("Expected 0 imported, got %d", response.Imported)
	}
}

func TestImportJSONRequest_Struct(t *testing.T) {
	req := ImportJSONRequest{
		Techniques: []*entity.Technique{
			{ID: "T1082", Name: "Test"},
		},
	}
	if len(req.Techniques) != 1 {
		t.Errorf("Expected 1 technique, got %d", len(req.Techniques))
	}
}

func TestImportJSONResponse_Struct(t *testing.T) {
	resp := ImportJSONResponse{
		Imported: 5,
		Failed:   2,
		Errors:   []string{"error1", "error2"},
	}
	if resp.Imported != 5 {
		t.Errorf("Expected Imported=5, got %d", resp.Imported)
	}
	if resp.Failed != 2 {
		t.Errorf("Expected Failed=2, got %d", resp.Failed)
	}
	if len(resp.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(resp.Errors))
	}
}
