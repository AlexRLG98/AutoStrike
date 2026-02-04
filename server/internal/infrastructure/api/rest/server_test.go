package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"autostrike/internal/application"
	"autostrike/internal/domain/entity"
	"autostrike/internal/domain/service"
	"autostrike/internal/infrastructure/websocket"

	"go.uber.org/zap"
)

// Mock repository implementations for testing
type mockAgentRepo struct{}

func (m *mockAgentRepo) Create(ctx context.Context, agent *entity.Agent) error       { return nil }
func (m *mockAgentRepo) Update(ctx context.Context, agent *entity.Agent) error       { return nil }
func (m *mockAgentRepo) Delete(ctx context.Context, paw string) error                { return nil }
func (m *mockAgentRepo) FindByPaw(ctx context.Context, paw string) (*entity.Agent, error) {
	return &entity.Agent{Paw: paw}, nil
}
func (m *mockAgentRepo) FindByPaws(ctx context.Context, paws []string) ([]*entity.Agent, error) {
	return []*entity.Agent{}, nil
}
func (m *mockAgentRepo) FindAll(ctx context.Context) ([]*entity.Agent, error) {
	return []*entity.Agent{}, nil
}
func (m *mockAgentRepo) FindByStatus(ctx context.Context, status entity.AgentStatus) ([]*entity.Agent, error) {
	return []*entity.Agent{}, nil
}
func (m *mockAgentRepo) FindByPlatform(ctx context.Context, platform string) ([]*entity.Agent, error) {
	return []*entity.Agent{}, nil
}
func (m *mockAgentRepo) UpdateLastSeen(ctx context.Context, paw string) error { return nil }

type mockScenarioRepo struct{}

func (m *mockScenarioRepo) Create(ctx context.Context, scenario *entity.Scenario) error { return nil }
func (m *mockScenarioRepo) Update(ctx context.Context, scenario *entity.Scenario) error { return nil }
func (m *mockScenarioRepo) Delete(ctx context.Context, id string) error                 { return nil }
func (m *mockScenarioRepo) FindByID(ctx context.Context, id string) (*entity.Scenario, error) {
	return &entity.Scenario{ID: id}, nil
}
func (m *mockScenarioRepo) FindAll(ctx context.Context) ([]*entity.Scenario, error) {
	return []*entity.Scenario{}, nil
}
func (m *mockScenarioRepo) FindByTag(ctx context.Context, tag string) ([]*entity.Scenario, error) {
	return []*entity.Scenario{}, nil
}
func (m *mockScenarioRepo) ImportFromYAML(ctx context.Context, path string) error { return nil }

type mockTechniqueRepo struct{}

func (m *mockTechniqueRepo) Create(ctx context.Context, technique *entity.Technique) error {
	return nil
}
func (m *mockTechniqueRepo) Update(ctx context.Context, technique *entity.Technique) error {
	return nil
}
func (m *mockTechniqueRepo) Delete(ctx context.Context, id string) error { return nil }
func (m *mockTechniqueRepo) FindByID(ctx context.Context, id string) (*entity.Technique, error) {
	return &entity.Technique{ID: id}, nil
}
func (m *mockTechniqueRepo) FindAll(ctx context.Context) ([]*entity.Technique, error) {
	return []*entity.Technique{}, nil
}
func (m *mockTechniqueRepo) FindByTactic(ctx context.Context, tactic entity.TacticType) ([]*entity.Technique, error) {
	return []*entity.Technique{}, nil
}
func (m *mockTechniqueRepo) FindByPlatform(ctx context.Context, platform string) ([]*entity.Technique, error) {
	return []*entity.Technique{}, nil
}
func (m *mockTechniqueRepo) ImportFromYAML(ctx context.Context, path string) error { return nil }

type mockResultRepo struct{}

func (m *mockResultRepo) CreateExecution(ctx context.Context, execution *entity.Execution) error {
	return nil
}
func (m *mockResultRepo) UpdateExecution(ctx context.Context, execution *entity.Execution) error {
	return nil
}
func (m *mockResultRepo) FindExecutionByID(ctx context.Context, id string) (*entity.Execution, error) {
	return &entity.Execution{ID: id}, nil
}
func (m *mockResultRepo) FindExecutionsByScenario(ctx context.Context, scenarioID string) ([]*entity.Execution, error) {
	return []*entity.Execution{}, nil
}
func (m *mockResultRepo) FindRecentExecutions(ctx context.Context, limit int) ([]*entity.Execution, error) {
	return []*entity.Execution{}, nil
}
func (m *mockResultRepo) CreateResult(ctx context.Context, result *entity.ExecutionResult) error {
	return nil
}
func (m *mockResultRepo) UpdateResult(ctx context.Context, result *entity.ExecutionResult) error {
	return nil
}
func (m *mockResultRepo) FindResultByID(ctx context.Context, id string) (*entity.ExecutionResult, error) {
	return &entity.ExecutionResult{ID: id}, nil
}
func (m *mockResultRepo) FindResultsByExecution(ctx context.Context, executionID string) ([]*entity.ExecutionResult, error) {
	return []*entity.ExecutionResult{}, nil
}
func (m *mockResultRepo) FindResultsByTechnique(ctx context.Context, techniqueID string) ([]*entity.ExecutionResult, error) {
	return []*entity.ExecutionResult{}, nil
}

func TestNewServerConfig_Default(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("AGENT_SECRET")
	os.Unsetenv("ENABLE_AUTH")

	config := NewServerConfig()

	if config.JWTSecret != "" {
		t.Errorf("Expected empty JWT secret, got '%s'", config.JWTSecret)
	}
	if config.AgentSecret != "" {
		t.Errorf("Expected empty agent secret, got '%s'", config.AgentSecret)
	}
	// Auth is disabled by default when JWT_SECRET is not set (dev mode)
	if config.EnableAuth {
		t.Error("Expected EnableAuth to be false when JWT_SECRET is not set")
	}
}

func TestNewServerConfig_EnabledWithSecret(t *testing.T) {
	os.Setenv("JWT_SECRET", "my-secret-key")
	os.Unsetenv("ENABLE_AUTH")
	defer os.Unsetenv("JWT_SECRET")

	config := NewServerConfig()

	// Auth is automatically enabled when JWT_SECRET is set
	if !config.EnableAuth {
		t.Error("Expected EnableAuth to be true when JWT_SECRET is set")
	}
}

func TestNewServerConfig_WithEnv(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-jwt-secret")
	os.Setenv("AGENT_SECRET", "test-agent-secret")
	os.Setenv("ENABLE_AUTH", "true")
	defer func() {
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("AGENT_SECRET")
		os.Unsetenv("ENABLE_AUTH")
	}()

	config := NewServerConfig()

	if config.JWTSecret != "test-jwt-secret" {
		t.Errorf("Expected JWT secret 'test-jwt-secret', got '%s'", config.JWTSecret)
	}
	if config.AgentSecret != "test-agent-secret" {
		t.Errorf("Expected agent secret 'test-agent-secret', got '%s'", config.AgentSecret)
	}
	if !config.EnableAuth {
		t.Error("Expected EnableAuth to be true")
	}
}

func TestNewServerConfig_AuthDisabled(t *testing.T) {
	os.Setenv("ENABLE_AUTH", "false")
	defer os.Unsetenv("ENABLE_AUTH")

	config := NewServerConfig()

	if config.EnableAuth {
		t.Error("Expected EnableAuth to be false when ENABLE_AUTH=false")
	}
}

func TestNewServerWithConfig_AuthDisabled(t *testing.T) {
	logger := zap.NewNop()
	services := &Services{
		Agent:     application.NewAgentService(&mockAgentRepo{}),
		Scenario:  application.NewScenarioService(&mockScenarioRepo{}, &mockTechniqueRepo{}, service.NewTechniqueValidator()),
		Technique: application.NewTechniqueService(&mockTechniqueRepo{}),
		Execution: application.NewExecutionService(&mockResultRepo{}, &mockScenarioRepo{}, &mockTechniqueRepo{}, &mockAgentRepo{}, nil, nil),
		Auth:      nil,
	}
	hub := websocket.NewHub(logger)

	config := &ServerConfig{
		JWTSecret:   "",
		AgentSecret: "",
		EnableAuth:  false,
	}

	server := NewServerWithConfig(services, hub, logger, config)

	if server == nil {
		t.Fatal("NewServerWithConfig returned nil")
	}

	if server.router == nil {
		t.Error("Server router is nil")
	}
}

func TestNewServerWithConfig_AuthEnabled(t *testing.T) {
	logger := zap.NewNop()
	services := &Services{
		Agent:     application.NewAgentService(&mockAgentRepo{}),
		Scenario:  application.NewScenarioService(&mockScenarioRepo{}, &mockTechniqueRepo{}, service.NewTechniqueValidator()),
		Technique: application.NewTechniqueService(&mockTechniqueRepo{}),
		Execution: application.NewExecutionService(&mockResultRepo{}, &mockScenarioRepo{}, &mockTechniqueRepo{}, &mockAgentRepo{}, nil, nil),
		Auth:      nil,
	}
	hub := websocket.NewHub(logger)

	config := &ServerConfig{
		JWTSecret:   "test-secret",
		AgentSecret: "agent-secret",
		EnableAuth:  true,
	}

	server := NewServerWithConfig(services, hub, logger, config)

	if server == nil {
		t.Fatal("NewServerWithConfig returned nil")
	}
}

func TestNewServerWithConfig_NoHub(t *testing.T) {
	logger := zap.NewNop()
	services := &Services{
		Agent:     application.NewAgentService(&mockAgentRepo{}),
		Scenario:  application.NewScenarioService(&mockScenarioRepo{}, &mockTechniqueRepo{}, service.NewTechniqueValidator()),
		Technique: application.NewTechniqueService(&mockTechniqueRepo{}),
		Execution: application.NewExecutionService(&mockResultRepo{}, &mockScenarioRepo{}, &mockTechniqueRepo{}, &mockAgentRepo{}, nil, nil),
		Auth:      nil,
	}

	config := &ServerConfig{EnableAuth: false}

	// Should not panic with nil hub
	server := NewServerWithConfig(services, nil, logger, config)

	if server == nil {
		t.Fatal("NewServerWithConfig returned nil")
	}
}

func TestNewServer(t *testing.T) {
	os.Setenv("ENABLE_AUTH", "false")
	defer os.Unsetenv("ENABLE_AUTH")

	logger := zap.NewNop()
	services := &Services{
		Agent:     application.NewAgentService(&mockAgentRepo{}),
		Scenario:  application.NewScenarioService(&mockScenarioRepo{}, &mockTechniqueRepo{}, service.NewTechniqueValidator()),
		Technique: application.NewTechniqueService(&mockTechniqueRepo{}),
		Execution: application.NewExecutionService(&mockResultRepo{}, &mockScenarioRepo{}, &mockTechniqueRepo{}, &mockAgentRepo{}, nil, nil),
		Auth:      nil,
	}
	hub := websocket.NewHub(logger)

	server := NewServer(services, hub, logger)

	if server == nil {
		t.Fatal("NewServer returned nil")
	}
}

func TestServer_Router(t *testing.T) {
	logger := zap.NewNop()
	services := &Services{
		Agent:     application.NewAgentService(&mockAgentRepo{}),
		Scenario:  application.NewScenarioService(&mockScenarioRepo{}, &mockTechniqueRepo{}, service.NewTechniqueValidator()),
		Technique: application.NewTechniqueService(&mockTechniqueRepo{}),
		Execution: application.NewExecutionService(&mockResultRepo{}, &mockScenarioRepo{}, &mockTechniqueRepo{}, &mockAgentRepo{}, nil, nil),
		Auth:      nil,
	}

	config := &ServerConfig{EnableAuth: false}
	server := NewServerWithConfig(services, nil, logger, config)

	router := server.Router()
	if router == nil {
		t.Error("Router() returned nil")
	}
}

func TestServer_HealthEndpoint(t *testing.T) {
	logger := zap.NewNop()
	services := &Services{
		Agent:     application.NewAgentService(&mockAgentRepo{}),
		Scenario:  application.NewScenarioService(&mockScenarioRepo{}, &mockTechniqueRepo{}, service.NewTechniqueValidator()),
		Technique: application.NewTechniqueService(&mockTechniqueRepo{}),
		Execution: application.NewExecutionService(&mockResultRepo{}, &mockScenarioRepo{}, &mockTechniqueRepo{}, &mockAgentRepo{}, nil, nil),
		Auth:      nil,
	}

	config := &ServerConfig{EnableAuth: false}
	server := NewServerWithConfig(services, nil, logger, config)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify response body contains auth_enabled
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}
	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", response["status"])
	}
	if response["auth_enabled"] != false {
		t.Errorf("Expected auth_enabled false, got %v", response["auth_enabled"])
	}
}

func TestServer_HealthEndpoint_AuthEnabled(t *testing.T) {
	logger := zap.NewNop()
	services := &Services{
		Agent:     application.NewAgentService(&mockAgentRepo{}),
		Scenario:  application.NewScenarioService(&mockScenarioRepo{}, &mockTechniqueRepo{}, service.NewTechniqueValidator()),
		Technique: application.NewTechniqueService(&mockTechniqueRepo{}),
		Execution: application.NewExecutionService(&mockResultRepo{}, &mockScenarioRepo{}, &mockTechniqueRepo{}, &mockAgentRepo{}, nil, nil),
		Auth:      nil,
	}

	config := &ServerConfig{
		EnableAuth: true,
		JWTSecret:  "test-secret",
	}
	server := NewServerWithConfig(services, nil, logger, config)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}
	if response["auth_enabled"] != true {
		t.Errorf("Expected auth_enabled true, got %v", response["auth_enabled"])
	}
}

func TestServer_APIRoutes(t *testing.T) {
	logger := zap.NewNop()
	services := &Services{
		Agent:     application.NewAgentService(&mockAgentRepo{}),
		Scenario:  application.NewScenarioService(&mockScenarioRepo{}, &mockTechniqueRepo{}, service.NewTechniqueValidator()),
		Technique: application.NewTechniqueService(&mockTechniqueRepo{}),
		Execution: application.NewExecutionService(&mockResultRepo{}, &mockScenarioRepo{}, &mockTechniqueRepo{}, &mockAgentRepo{}, nil, nil),
		Auth:      nil,
	}

	config := &ServerConfig{EnableAuth: false}
	server := NewServerWithConfig(services, nil, logger, config)

	routes := server.Router().Routes()
	expectedPaths := []string{"/health", "/api/v1/agents", "/api/v1/techniques"}

	for _, expected := range expectedPaths {
		found := false
		for _, route := range routes {
			if route.Path == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected route %s not found", expected)
		}
	}
}

func TestServer_Run_InvalidAddress(t *testing.T) {
	logger := zap.NewNop()
	services := &Services{
		Agent:     application.NewAgentService(&mockAgentRepo{}),
		Scenario:  application.NewScenarioService(&mockScenarioRepo{}, &mockTechniqueRepo{}, service.NewTechniqueValidator()),
		Technique: application.NewTechniqueService(&mockTechniqueRepo{}),
		Execution: application.NewExecutionService(&mockResultRepo{}, &mockScenarioRepo{}, &mockTechniqueRepo{}, &mockAgentRepo{}, nil, nil),
		Auth:      nil,
	}

	config := &ServerConfig{EnableAuth: false}
	server := NewServerWithConfig(services, nil, logger, config)

	// Run in goroutine and expect it to fail with invalid address
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run("invalid-address:99999")
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Error("Expected error with invalid address")
		}
	case <-time.After(500 * time.Millisecond):
		// Server might be trying to bind, which is also acceptable
	}
}

func TestNewServerConfig_EnableAuthTrueNoSecret(t *testing.T) {
	os.Unsetenv("JWT_SECRET")
	os.Setenv("ENABLE_AUTH", "true")
	defer os.Unsetenv("ENABLE_AUTH")

	config := NewServerConfig()

	// ENABLE_AUTH=true should enable auth even without secret
	if !config.EnableAuth {
		t.Error("Expected EnableAuth to be true when ENABLE_AUTH=true")
	}
}

func TestNewServerConfig_CustomDashboardPath(t *testing.T) {
	os.Setenv("DASHBOARD_PATH", "/custom/path")
	defer os.Unsetenv("DASHBOARD_PATH")

	config := NewServerConfig()

	if config.DashboardPath != "/custom/path" {
		t.Errorf("Expected dashboard path '/custom/path', got '%s'", config.DashboardPath)
	}
}

func TestServer_DashboardRoutes_InvalidPath(t *testing.T) {
	logger := zap.NewNop()
	services := &Services{
		Agent:     application.NewAgentService(&mockAgentRepo{}),
		Scenario:  application.NewScenarioService(&mockScenarioRepo{}, &mockTechniqueRepo{}, service.NewTechniqueValidator()),
		Technique: application.NewTechniqueService(&mockTechniqueRepo{}),
		Execution: application.NewExecutionService(&mockResultRepo{}, &mockScenarioRepo{}, &mockTechniqueRepo{}, &mockAgentRepo{}, nil, nil),
		Auth:      nil,
	}

	// Use a path that doesn't exist
	config := &ServerConfig{
		EnableAuth:    false,
		DashboardPath: "/nonexistent/path/that/does/not/exist",
	}

	// Should not panic, just log warning
	server := NewServerWithConfig(services, nil, logger, config)
	if server == nil {
		t.Fatal("Server should be created even with invalid dashboard path")
	}
}

func TestServer_NoRoute_APIPath(t *testing.T) {
	logger := zap.NewNop()
	services := &Services{
		Agent:     application.NewAgentService(&mockAgentRepo{}),
		Scenario:  application.NewScenarioService(&mockScenarioRepo{}, &mockTechniqueRepo{}, service.NewTechniqueValidator()),
		Technique: application.NewTechniqueService(&mockTechniqueRepo{}),
		Execution: application.NewExecutionService(&mockResultRepo{}, &mockScenarioRepo{}, &mockTechniqueRepo{}, &mockAgentRepo{}, nil, nil),
		Auth:      nil,
	}

	// Create temp directory with index.html for dashboard
	tmpDir := t.TempDir()
	indexFile := tmpDir + "/index.html"
	if err := os.WriteFile(indexFile, []byte("<html></html>"), 0644); err != nil {
		t.Fatalf("Failed to write index.html: %v", err)
	}
	if err := os.MkdirAll(tmpDir+"/assets", 0755); err != nil {
		t.Fatalf("Failed to create assets dir: %v", err)
	}

	config := &ServerConfig{
		EnableAuth:    false,
		DashboardPath: tmpDir,
	}

	server := NewServerWithConfig(services, nil, logger, config)

	// Request to non-existent API endpoint should return 404 JSON
	req, _ := http.NewRequest("GET", "/api/v1/nonexistent", nil)
	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestServer_NoRoute_NonAPIPath(t *testing.T) {
	logger := zap.NewNop()
	services := &Services{
		Agent:     application.NewAgentService(&mockAgentRepo{}),
		Scenario:  application.NewScenarioService(&mockScenarioRepo{}, &mockTechniqueRepo{}, service.NewTechniqueValidator()),
		Technique: application.NewTechniqueService(&mockTechniqueRepo{}),
		Execution: application.NewExecutionService(&mockResultRepo{}, &mockScenarioRepo{}, &mockTechniqueRepo{}, &mockAgentRepo{}, nil, nil),
		Auth:      nil,
	}

	// Create temp directory with index.html for dashboard
	tmpDir := t.TempDir()
	indexFile := tmpDir + "/index.html"
	if err := os.WriteFile(indexFile, []byte("<html>Dashboard</html>"), 0644); err != nil {
		t.Fatalf("Failed to write index.html: %v", err)
	}
	if err := os.MkdirAll(tmpDir+"/assets", 0755); err != nil {
		t.Fatalf("Failed to create assets dir: %v", err)
	}

	config := &ServerConfig{
		EnableAuth:    false,
		DashboardPath: tmpDir,
	}

	server := NewServerWithConfig(services, nil, logger, config)

	// Request to non-API path should serve index.html (SPA fallback)
	req, _ := http.NewRequest("GET", "/some/spa/route", nil)
	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "<html>Dashboard</html>" {
		t.Errorf("Expected index.html content, got %s", w.Body.String())
	}
}
