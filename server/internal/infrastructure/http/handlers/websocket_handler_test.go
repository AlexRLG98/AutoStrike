package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"autostrike/internal/application"
	"autostrike/internal/domain/entity"
	"autostrike/internal/infrastructure/websocket"

	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Mock AgentRepository for WebSocket handler testing
type wsTestAgentRepo struct {
	mu          sync.RWMutex
	agents      map[string]*entity.Agent
	findErr     error
	createErr   error
	updateErr   error
	lastSeenErr error
}

func newWSTestAgentRepo() *wsTestAgentRepo {
	return &wsTestAgentRepo{
		agents: make(map[string]*entity.Agent),
	}
}

func (m *wsTestAgentRepo) Create(ctx context.Context, agent *entity.Agent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return m.createErr
	}
	m.agents[agent.Paw] = agent
	return nil
}

func (m *wsTestAgentRepo) Update(ctx context.Context, agent *entity.Agent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateErr != nil {
		return m.updateErr
	}
	m.agents[agent.Paw] = agent
	return nil
}

func (m *wsTestAgentRepo) Delete(ctx context.Context, paw string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.agents, paw)
	return nil
}

func (m *wsTestAgentRepo) FindByPaw(ctx context.Context, paw string) (*entity.Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.findErr != nil {
		return nil, m.findErr
	}
	agent, ok := m.agents[paw]
	if !ok {
		return nil, errors.New("agent not found")
	}
	return agent, nil
}

func (m *wsTestAgentRepo) FindAll(ctx context.Context) ([]*entity.Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.findErr != nil {
		return nil, m.findErr
	}
	result := make([]*entity.Agent, 0, len(m.agents))
	for _, agent := range m.agents {
		result = append(result, agent)
	}
	return result, nil
}

func (m *wsTestAgentRepo) FindByStatus(ctx context.Context, status entity.AgentStatus) ([]*entity.Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.findErr != nil {
		return nil, m.findErr
	}
	var result []*entity.Agent
	for _, agent := range m.agents {
		if agent.Status == status {
			result = append(result, agent)
		}
	}
	return result, nil
}

func (m *wsTestAgentRepo) FindByPlatform(ctx context.Context, platform string) ([]*entity.Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.findErr != nil {
		return nil, m.findErr
	}
	var result []*entity.Agent
	for _, agent := range m.agents {
		if agent.Platform == platform {
			result = append(result, agent)
		}
	}
	return result, nil
}

func (m *wsTestAgentRepo) UpdateLastSeen(ctx context.Context, paw string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lastSeenErr != nil {
		return m.lastSeenErr
	}
	agent, ok := m.agents[paw]
	if ok {
		agent.LastSeen = time.Now()
	}
	return nil
}

func TestNewWebSocketHandler(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)

	handler := NewWebSocketHandler(hub, agentService, logger)

	if handler == nil {
		t.Fatal("NewWebSocketHandler returned nil")
	}

	if handler.hub != hub {
		t.Error("Handler hub not set correctly")
	}

	if handler.agentService != agentService {
		t.Error("Handler agentService not set correctly")
	}

	if handler.logger != logger {
		t.Error("Handler logger not set correctly")
	}
}

func TestWebSocketHandler_RegisterRoutes(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)

	handler := NewWebSocketHandler(hub, agentService, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler.RegisterRoutes(router)

	// Check that the route was registered
	routes := router.Routes()
	found := false
	for _, route := range routes {
		if route.Path == "/ws/agent" && route.Method == "GET" {
			found = true
			break
		}
	}

	if !found {
		t.Error("WebSocket route /ws/agent not registered")
	}
}

func TestWebSocketHandler_HandleAgentConnection(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	go hub.Run()

	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)

	handler := NewWebSocketHandler(hub, agentService, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws/agent", handler.HandleAgentConnection)

	server := httptest.NewServer(router)
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/agent"

	// Connect WebSocket client
	conn, resp, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect WebSocket: %v", err)
	}
	defer conn.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("Expected status 101, got %d", resp.StatusCode)
	}
}

func TestWebSocketHandler_HandleMessage_Register(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	go hub.Run()

	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)

	handler := NewWebSocketHandler(hub, agentService, logger)

	// Create a mock client
	client := websocket.NewClient(hub, nil, "", logger)

	// Create register message
	payload := RegisterPayload{
		Paw:       "test-agent-123",
		Hostname:  "test-host",
		Username:  "testuser",
		Platform:  "linux",
		Executors: []string{"sh", "bash"},
	}
	payloadBytes, _ := json.Marshal(payload)

	msg := &websocket.Message{
		Type:    "register",
		Payload: payloadBytes,
	}

	// Handle the message
	handler.handleMessage(client, msg)

	// Verify agent was registered
	if client.GetAgentPaw() != "test-agent-123" {
		t.Errorf("Expected client paw 'test-agent-123', got '%s'", client.GetAgentPaw())
	}

	// Verify agent exists in repo
	agent, err := repo.FindByPaw(context.Background(), "test-agent-123")
	if err != nil {
		t.Fatalf("Agent not found in repo: %v", err)
	}

	if agent.Hostname != "test-host" {
		t.Errorf("Expected hostname 'test-host', got '%s'", agent.Hostname)
	}

	if agent.Platform != "linux" {
		t.Errorf("Expected platform 'linux', got '%s'", agent.Platform)
	}
}

func TestWebSocketHandler_HandleMessage_Heartbeat(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	go hub.Run()

	repo := newWSTestAgentRepo()
	// Pre-create agent
	repo.agents["test-agent"] = &entity.Agent{
		Paw:      "test-agent",
		Hostname: "test-host",
		LastSeen: time.Now().Add(-1 * time.Hour),
	}

	agentService := application.NewAgentService(repo)
	handler := NewWebSocketHandler(hub, agentService, logger)

	// Create a mock client with paw set
	client := websocket.NewClient(hub, nil, "test-agent", logger)

	msg := &websocket.Message{
		Type:    "heartbeat",
		Payload: json.RawMessage(`{}`),
	}

	oldLastSeen := repo.agents["test-agent"].LastSeen

	// Handle heartbeat
	handler.handleMessage(client, msg)

	// Verify last seen was updated
	if !repo.agents["test-agent"].LastSeen.After(oldLastSeen) {
		t.Error("LastSeen was not updated")
	}
}

func TestWebSocketHandler_HandleMessage_HeartbeatWithoutPaw(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)

	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)
	handler := NewWebSocketHandler(hub, agentService, logger)

	// Create client without paw
	client := websocket.NewClient(hub, nil, "", logger)

	msg := &websocket.Message{
		Type:    "heartbeat",
		Payload: json.RawMessage(`{}`),
	}

	// Should not panic or error
	handler.handleMessage(client, msg)
}

func TestWebSocketHandler_HandleMessage_TaskResult(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)

	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)
	handler := NewWebSocketHandler(hub, agentService, logger)

	client := websocket.NewClient(hub, nil, "test-agent", logger)

	payload := TaskResultPayload{
		TaskID:   "task-123",
		ExitCode: 0,
		Output:   "command executed successfully",
	}
	payloadBytes, _ := json.Marshal(payload)

	msg := &websocket.Message{
		Type:    "task_result",
		Payload: payloadBytes,
	}

	// Should not panic
	handler.handleMessage(client, msg)
}

func TestWebSocketHandler_HandleTaskResult_ValidPayload(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)

	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)
	handler := NewWebSocketHandler(hub, agentService, logger)

	client := websocket.NewClient(hub, nil, "test-agent", logger)

	payload := TaskResultPayload{
		TaskID:   "task-456",
		ExitCode: 1,
		Output:   "error output",
		Error:    "command failed",
	}
	payloadBytes, _ := json.Marshal(payload)

	// Should not panic
	handler.handleTaskResult(client, payloadBytes)
}

func TestWebSocketHandler_HandleTaskResult_InvalidPayload(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)

	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)
	handler := NewWebSocketHandler(hub, agentService, logger)

	client := websocket.NewClient(hub, nil, "test-agent", logger)

	// Invalid JSON
	handler.handleTaskResult(client, json.RawMessage(`invalid json`))
	// Should not panic, just log warning
}

func TestTaskResultPayload_JSONMarshal(t *testing.T) {
	payload := TaskResultPayload{
		TaskID:   "task-789",
		ExitCode: 0,
		Output:   "success output",
		Error:    "",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded TaskResultPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.TaskID != payload.TaskID {
		t.Errorf("Expected TaskID '%s', got '%s'", payload.TaskID, decoded.TaskID)
	}

	if decoded.ExitCode != payload.ExitCode {
		t.Errorf("Expected ExitCode %d, got %d", payload.ExitCode, decoded.ExitCode)
	}

	if decoded.Output != payload.Output {
		t.Errorf("Expected Output '%s', got '%s'", payload.Output, decoded.Output)
	}
}

func TestTaskResultPayload_WithError(t *testing.T) {
	payload := TaskResultPayload{
		TaskID:   "task-error",
		ExitCode: 127,
		Output:   "",
		Error:    "command not found",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded TaskResultPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Error != "command not found" {
		t.Errorf("Expected Error 'command not found', got '%s'", decoded.Error)
	}
}

func TestWebSocketHandler_HandleMessage_UnknownType(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)

	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)
	handler := NewWebSocketHandler(hub, agentService, logger)

	client := websocket.NewClient(hub, nil, "test-agent", logger)

	msg := &websocket.Message{
		Type:    "unknown_type",
		Payload: json.RawMessage(`{}`),
	}

	// Should not panic, just log warning
	handler.handleMessage(client, msg)
}

func TestWebSocketHandler_HandleRegister_InvalidPayload(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)

	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)
	handler := NewWebSocketHandler(hub, agentService, logger)

	client := websocket.NewClient(hub, nil, "", logger)

	// Invalid JSON payload
	handler.handleRegister(client, json.RawMessage(`invalid json`))

	// Client paw should not be set
	if client.GetAgentPaw() != "" {
		t.Error("Client paw should not be set with invalid payload")
	}
}

func TestWebSocketHandler_HandleRegister_ServiceError(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)

	repo := newWSTestAgentRepo()
	repo.createErr = errors.New("database error")

	agentService := application.NewAgentService(repo)
	handler := NewWebSocketHandler(hub, agentService, logger)

	client := websocket.NewClient(hub, nil, "", logger)

	payload := RegisterPayload{
		Paw:       "test-agent",
		Hostname:  "test-host",
		Username:  "testuser",
		Platform:  "linux",
		Executors: []string{"sh"},
	}
	payloadBytes, _ := json.Marshal(payload)

	// Should not panic even with service error
	handler.handleRegister(client, payloadBytes)
}

func TestWebSocketHandler_HandleHeartbeat_ServiceError(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)

	repo := newWSTestAgentRepo()
	repo.lastSeenErr = errors.New("database error")

	agentService := application.NewAgentService(repo)
	handler := NewWebSocketHandler(hub, agentService, logger)

	client := websocket.NewClient(hub, nil, "test-agent", logger)

	// Should not panic even with service error
	handler.handleHeartbeat(client, json.RawMessage(`{}`))
}

func TestRegisterPayload_JSONMarshal(t *testing.T) {
	payload := RegisterPayload{
		Paw:       "agent-123",
		Hostname:  "my-host",
		Username:  "admin",
		Platform:  "windows",
		Executors: []string{"cmd", "powershell"},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	var decoded RegisterPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if decoded.Paw != payload.Paw {
		t.Errorf("Expected paw '%s', got '%s'", payload.Paw, decoded.Paw)
	}

	if decoded.Hostname != payload.Hostname {
		t.Errorf("Expected hostname '%s', got '%s'", payload.Hostname, decoded.Hostname)
	}

	if decoded.Username != payload.Username {
		t.Errorf("Expected username '%s', got '%s'", payload.Username, decoded.Username)
	}

	if decoded.Platform != payload.Platform {
		t.Errorf("Expected platform '%s', got '%s'", payload.Platform, decoded.Platform)
	}

	if len(decoded.Executors) != len(payload.Executors) {
		t.Errorf("Expected %d executors, got %d", len(payload.Executors), len(decoded.Executors))
	}
}

func TestUpgrader_CheckOrigin(t *testing.T) {
	// Test that CheckOrigin allows all origins (for development)
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("Origin", "http://different-origin.com")

	if !upgrader.CheckOrigin(req) {
		t.Error("Expected CheckOrigin to return true for all origins")
	}
}

func TestWebSocketHandler_FullIntegration(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	go hub.Run()

	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)
	handler := NewWebSocketHandler(hub, agentService, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler.RegisterRoutes(router)

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/agent"

	conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send register message
	regMsg := map[string]interface{}{
		"type": "register",
		"payload": map[string]interface{}{
			"paw":       "integration-test-agent",
			"hostname":  "test-host",
			"username":  "testuser",
			"platform":  "linux",
			"executors": []string{"sh", "bash"},
		},
	}

	if err := conn.WriteJSON(regMsg); err != nil {
		t.Fatalf("Failed to send register message: %v", err)
	}

	// Wait for message to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify agent was registered
	agent, err := repo.FindByPaw(context.Background(), "integration-test-agent")
	if err != nil {
		t.Fatalf("Agent not found: %v", err)
	}

	if agent.Hostname != "test-host" {
		t.Errorf("Expected hostname 'test-host', got '%s'", agent.Hostname)
	}

	// Send heartbeat
	heartbeatMsg := map[string]interface{}{
		"type":    "heartbeat",
		"payload": map[string]interface{}{},
	}

	if err := conn.WriteJSON(heartbeatMsg); err != nil {
		t.Fatalf("Failed to send heartbeat: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
}
