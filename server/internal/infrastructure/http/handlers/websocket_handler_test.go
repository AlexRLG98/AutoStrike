package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
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

func (m *wsTestAgentRepo) FindByPaws(ctx context.Context, paws []string) ([]*entity.Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
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
	// Test that CheckOrigin allows localhost origins (default allowed origins)
	tests := []struct {
		name     string
		origin   string
		expected bool
	}{
		{"localhost http 3000", "http://localhost:3000", true},
		{"localhost https 3000", "https://localhost:3000", true},
		{"localhost http 8443", "http://localhost:8443", true},
		{"localhost https 8443", "https://localhost:8443", true},
		{"no origin header", "", true}, // Same-origin or non-browser requests
		{"unknown origin", "http://different-origin.com", false},
		{"malicious origin", "http://evil.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://localhost:8443", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			result := upgrader.CheckOrigin(req)
			if result != tt.expected {
				t.Errorf("CheckOrigin(%q) = %v, expected %v", tt.origin, result, tt.expected)
			}
		})
	}
}

func TestGetAllowedOrigins_Default(t *testing.T) {
	// Clear ALLOWED_ORIGINS to test default behavior
	originalValue := os.Getenv("ALLOWED_ORIGINS")
	os.Unsetenv("ALLOWED_ORIGINS")
	defer func() {
		if originalValue != "" {
			os.Setenv("ALLOWED_ORIGINS", originalValue)
		}
	}()

	origins := getAllowedOrigins()

	expected := []string{
		"http://localhost:3000",
		"https://localhost:3000",
		"http://localhost:8443",
		"https://localhost:8443",
	}

	if len(origins) != len(expected) {
		t.Errorf("Expected %d default origins, got %d", len(expected), len(origins))
	}

	for i, exp := range expected {
		if origins[i] != exp {
			t.Errorf("Expected origin[%d] = '%s', got '%s'", i, exp, origins[i])
		}
	}
}

func TestGetAllowedOrigins_CustomSingle(t *testing.T) {
	originalValue := os.Getenv("ALLOWED_ORIGINS")
	os.Setenv("ALLOWED_ORIGINS", "https://myapp.com")
	defer func() {
		if originalValue != "" {
			os.Setenv("ALLOWED_ORIGINS", originalValue)
		} else {
			os.Unsetenv("ALLOWED_ORIGINS")
		}
	}()

	origins := getAllowedOrigins()

	if len(origins) != 1 {
		t.Errorf("Expected 1 origin, got %d", len(origins))
	}

	if origins[0] != "https://myapp.com" {
		t.Errorf("Expected 'https://myapp.com', got '%s'", origins[0])
	}
}

func TestGetAllowedOrigins_CustomMultiple(t *testing.T) {
	originalValue := os.Getenv("ALLOWED_ORIGINS")
	os.Setenv("ALLOWED_ORIGINS", "https://app1.com,https://app2.com,http://localhost:5000")
	defer func() {
		if originalValue != "" {
			os.Setenv("ALLOWED_ORIGINS", originalValue)
		} else {
			os.Unsetenv("ALLOWED_ORIGINS")
		}
	}()

	origins := getAllowedOrigins()

	if len(origins) != 3 {
		t.Errorf("Expected 3 origins, got %d", len(origins))
	}

	expected := []string{"https://app1.com", "https://app2.com", "http://localhost:5000"}
	for i, exp := range expected {
		if origins[i] != exp {
			t.Errorf("Expected origin[%d] = '%s', got '%s'", i, exp, origins[i])
		}
	}
}

func TestUpgrader_CheckOrigin_WithCustomOrigins(t *testing.T) {
	originalValue := os.Getenv("ALLOWED_ORIGINS")
	os.Setenv("ALLOWED_ORIGINS", "https://trusted.com,https://allowed.org")
	defer func() {
		if originalValue != "" {
			os.Setenv("ALLOWED_ORIGINS", originalValue)
		} else {
			os.Unsetenv("ALLOWED_ORIGINS")
		}
	}()

	tests := []struct {
		name     string
		origin   string
		expected bool
	}{
		{"trusted origin", "https://trusted.com", true},
		{"allowed origin", "https://allowed.org", true},
		{"no origin", "", true},
		{"untrusted origin", "https://evil.com", false},
		{"localhost not allowed", "http://localhost:3000", false}, // Not in custom list
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://localhost:8443", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			result := upgrader.CheckOrigin(req)
			if result != tt.expected {
				t.Errorf("CheckOrigin(%q) = %v, expected %v", tt.origin, result, tt.expected)
			}
		})
	}
}

func TestUpgrader_CheckOrigin_WithWhitespace(t *testing.T) {
	originalValue := os.Getenv("ALLOWED_ORIGINS")
	os.Setenv("ALLOWED_ORIGINS", "https://site1.com, https://site2.com , https://site3.com")
	defer func() {
		if originalValue != "" {
			os.Setenv("ALLOWED_ORIGINS", originalValue)
		} else {
			os.Unsetenv("ALLOWED_ORIGINS")
		}
	}()

	// CheckOrigin uses TrimSpace, so origins with whitespace should still match
	tests := []struct {
		origin   string
		expected bool
	}{
		{"https://site1.com", true},
		{"https://site2.com", true},
		{"https://site3.com", true},
	}

	for _, tt := range tests {
		req, _ := http.NewRequest("GET", "http://localhost:8443", nil)
		req.Header.Set("Origin", tt.origin)

		result := upgrader.CheckOrigin(req)
		if result != tt.expected {
			t.Errorf("CheckOrigin(%q) = %v, expected %v", tt.origin, result, tt.expected)
		}
	}
}

func TestWebSocketHandler_RegisterRoutes_Dashboard(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)

	handler := NewWebSocketHandler(hub, agentService, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler.RegisterRoutes(router)

	// Check that both routes were registered
	routes := router.Routes()
	agentFound := false
	dashboardFound := false
	for _, route := range routes {
		if route.Path == "/ws/agent" && route.Method == "GET" {
			agentFound = true
		}
		if route.Path == "/ws/dashboard" && route.Method == "GET" {
			dashboardFound = true
		}
	}

	if !agentFound {
		t.Error("WebSocket route /ws/agent not registered")
	}
	if !dashboardFound {
		t.Error("WebSocket route /ws/dashboard not registered")
	}
}

func TestWebSocketHandler_HandleDashboardConnection(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	go hub.Run()

	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)

	handler := NewWebSocketHandler(hub, agentService, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws/dashboard", handler.HandleDashboardConnection)

	server := httptest.NewServer(router)
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/dashboard"

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

func TestWebSocketHandler_HandleDashboardMessage_Ping(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	go hub.Run()

	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)

	handler := NewWebSocketHandler(hub, agentService, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws/dashboard", handler.HandleDashboardConnection)

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/dashboard"

	conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send ping message
	pingMsg := map[string]interface{}{
		"type":    "ping",
		"payload": map[string]interface{}{},
	}

	if err := conn.WriteJSON(pingMsg); err != nil {
		t.Fatalf("Failed to send ping: %v", err)
	}

	// Read pong response
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}
	var response map[string]interface{}
	if err := conn.ReadJSON(&response); err != nil {
		t.Fatalf("Failed to read pong response: %v", err)
	}

	if response["type"] != "pong" {
		t.Errorf("Expected type 'pong', got '%v'", response["type"])
	}
}

func TestWebSocketHandler_DashboardReceivesBroadcast(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	go hub.Run()

	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)

	handler := NewWebSocketHandler(hub, agentService, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws/dashboard", handler.HandleDashboardConnection)

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/dashboard"

	conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Wait for client to be registered
	time.Sleep(100 * time.Millisecond)

	// Broadcast a message
	broadcastMsg := map[string]interface{}{
		"type": "execution_cancelled",
		"payload": map[string]interface{}{
			"execution_id": "test-exec-123",
		},
	}
	msgBytes, _ := json.Marshal(broadcastMsg)
	hub.Broadcast(msgBytes)

	// Read broadcast message
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}
	var response map[string]interface{}
	if err := conn.ReadJSON(&response); err != nil {
		t.Fatalf("Failed to read broadcast: %v", err)
	}

	if response["type"] != "execution_cancelled" {
		t.Errorf("Expected type 'execution_cancelled', got '%v'", response["type"])
	}
}

func TestWebSocketHandler_HandleDashboardMessage_OtherTypes(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	go hub.Run()

	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)

	handler := NewWebSocketHandler(hub, agentService, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws/dashboard", handler.HandleDashboardConnection)

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/dashboard"

	conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send a non-ping message (should be ignored, not cause error)
	otherMsg := map[string]interface{}{
		"type":    "some_other_type",
		"payload": map[string]interface{}{"data": "test"},
	}

	if err := conn.WriteJSON(otherMsg); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Give time for message to be processed
	time.Sleep(100 * time.Millisecond)

	// Connection should still be alive - send a ping to verify
	pingMsg := map[string]interface{}{
		"type":    "ping",
		"payload": map[string]interface{}{},
	}
	if err := conn.WriteJSON(pingMsg); err != nil {
		t.Fatalf("Connection died after other message: %v", err)
	}

	// Read pong response to confirm connection is healthy
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}
	var response map[string]interface{}
	if err := conn.ReadJSON(&response); err != nil {
		t.Fatalf("Failed to read pong response: %v", err)
	}

	if response["type"] != "pong" {
		t.Errorf("Expected type 'pong', got '%v'", response["type"])
	}
}

func TestWebSocketHandler_HandleAgentConnection_UpgradeFail(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)

	handler := NewWebSocketHandler(hub, agentService, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws/agent", handler.HandleAgentConnection)

	// Make a regular HTTP request (not a WebSocket upgrade request)
	// This will cause the upgrade to fail
	req := httptest.NewRequest("GET", "/ws/agent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should not crash and should return error (bad request or similar)
	// The upgrade fails silently (logs error and returns)
	// Status won't be set by gin if upgrade fails early
}

func TestWebSocketHandler_HandleDashboardConnection_UpgradeFail(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)

	handler := NewWebSocketHandler(hub, agentService, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws/dashboard", handler.HandleDashboardConnection)

	// Make a regular HTTP request (not a WebSocket upgrade request)
	req := httptest.NewRequest("GET", "/ws/dashboard", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should not crash
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

func TestWebSocketHandler_SetExecutionService(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)

	handler := NewWebSocketHandler(hub, agentService, logger)

	// Initially nil
	if handler.executionService != nil {
		t.Error("executionService should be nil initially")
	}

	// Create a mock execution service (we just need to test that it gets set)
	// We can pass nil since we're just testing the setter
	handler.SetExecutionService(nil)

	// Now it should be set (to nil in this case, but the method was called)
	// The real test is that the method exists and can be called
}

func TestWebSocketHandler_HandleTaskResult_WithExecutionService(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)

	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)
	handler := NewWebSocketHandler(hub, agentService, logger)

	// Set execution service to nil to test the nil check path
	handler.SetExecutionService(nil)

	client := websocket.NewClient(hub, nil, "test-agent", logger)

	payload := TaskResultPayload{
		TaskID:      "task-with-service",
		TechniqueID: "T1059",
		Success:     true,
		ExitCode:    0,
		Output:      "command executed successfully",
	}
	payloadBytes, _ := json.Marshal(payload)

	// Should not panic even without execution service
	handler.handleTaskResult(client, payloadBytes)
}

func TestWebSocketHandler_HandleTaskResult_SuccessStatus(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)

	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)
	handler := NewWebSocketHandler(hub, agentService, logger)

	client := websocket.NewClient(hub, nil, "test-agent", logger)

	payload := TaskResultPayload{
		TaskID:      "task-success",
		TechniqueID: "T1082",
		Success:     true,
		ExitCode:    0,
		Output:      "System info collected",
	}
	payloadBytes, _ := json.Marshal(payload)

	// Should handle successfully
	handler.handleTaskResult(client, payloadBytes)
}

func TestWebSocketHandler_HandleTaskResult_FailedStatus(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)

	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)
	handler := NewWebSocketHandler(hub, agentService, logger)

	client := websocket.NewClient(hub, nil, "test-agent", logger)

	payload := TaskResultPayload{
		TaskID:      "task-failed",
		TechniqueID: "T1059",
		Success:     false,
		ExitCode:    1,
		Output:      "",
		Error:       "Command not found",
	}
	payloadBytes, _ := json.Marshal(payload)

	// Should handle failure case
	handler.handleTaskResult(client, payloadBytes)
}

func TestWebSocketHandler_HandleTaskResult_WithErrorOutput(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)

	repo := newWSTestAgentRepo()
	agentService := application.NewAgentService(repo)
	handler := NewWebSocketHandler(hub, agentService, logger)

	client := websocket.NewClient(hub, nil, "test-agent", logger)

	payload := TaskResultPayload{
		TaskID:      "task-with-error",
		TechniqueID: "T1016",
		Success:     false,
		ExitCode:    127,
		Output:      "partial output",
		Error:       "command failed with error",
	}
	payloadBytes, _ := json.Marshal(payload)

	// Error and output should be combined
	handler.handleTaskResult(client, payloadBytes)
}

// Mock result repository for testing handleTaskResult with real execution service
type wsTestResultRepo struct {
	results    map[string]*entity.ExecutionResult
	executions map[string]*entity.Execution
	updateErr  error
}

func newWSTestResultRepo() *wsTestResultRepo {
	return &wsTestResultRepo{
		results:    make(map[string]*entity.ExecutionResult),
		executions: make(map[string]*entity.Execution),
	}
}

func (m *wsTestResultRepo) CreateExecution(ctx context.Context, exec *entity.Execution) error {
	m.executions[exec.ID] = exec
	return nil
}

func (m *wsTestResultRepo) UpdateExecution(ctx context.Context, exec *entity.Execution) error {
	m.executions[exec.ID] = exec
	return nil
}

func (m *wsTestResultRepo) FindExecutionByID(ctx context.Context, id string) (*entity.Execution, error) {
	if exec, ok := m.executions[id]; ok {
		return exec, nil
	}
	return nil, errors.New("execution not found")
}

func (m *wsTestResultRepo) FindExecutionsByScenario(ctx context.Context, scenarioID string) ([]*entity.Execution, error) {
	return []*entity.Execution{}, nil
}

func (m *wsTestResultRepo) FindRecentExecutions(ctx context.Context, limit int) ([]*entity.Execution, error) {
	return []*entity.Execution{}, nil
}

func (m *wsTestResultRepo) CreateResult(ctx context.Context, result *entity.ExecutionResult) error {
	m.results[result.ID] = result
	return nil
}

func (m *wsTestResultRepo) UpdateResult(ctx context.Context, result *entity.ExecutionResult) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.results[result.ID] = result
	return nil
}

func (m *wsTestResultRepo) FindResultByID(ctx context.Context, id string) (*entity.ExecutionResult, error) {
	if r, ok := m.results[id]; ok {
		return r, nil
	}
	return nil, errors.New("result not found")
}

func (m *wsTestResultRepo) FindResultsByExecution(ctx context.Context, executionID string) ([]*entity.ExecutionResult, error) {
	var results []*entity.ExecutionResult
	for _, r := range m.results {
		if r.ExecutionID == executionID {
			results = append(results, r)
		}
	}
	return results, nil
}

func (m *wsTestResultRepo) FindResultsByTechnique(ctx context.Context, techniqueID string) ([]*entity.ExecutionResult, error) {
	return []*entity.ExecutionResult{}, nil
}

func (m *wsTestResultRepo) FindExecutionsByDateRange(ctx context.Context, start, end time.Time) ([]*entity.Execution, error) {
	return []*entity.Execution{}, nil
}

func (m *wsTestResultRepo) FindCompletedExecutionsByDateRange(ctx context.Context, start, end time.Time) ([]*entity.Execution, error) {
	return []*entity.Execution{}, nil
}

// Mock scenario repository
type wsTestScenarioRepo struct{}

func (m *wsTestScenarioRepo) Create(ctx context.Context, s *entity.Scenario) error  { return nil }
func (m *wsTestScenarioRepo) Update(ctx context.Context, s *entity.Scenario) error  { return nil }
func (m *wsTestScenarioRepo) Delete(ctx context.Context, id string) error           { return nil }
func (m *wsTestScenarioRepo) FindByID(ctx context.Context, id string) (*entity.Scenario, error) {
	return &entity.Scenario{ID: id}, nil
}
func (m *wsTestScenarioRepo) FindAll(ctx context.Context) ([]*entity.Scenario, error) {
	return []*entity.Scenario{}, nil
}
func (m *wsTestScenarioRepo) FindByTag(ctx context.Context, tag string) ([]*entity.Scenario, error) {
	return []*entity.Scenario{}, nil
}
func (m *wsTestScenarioRepo) ImportFromYAML(ctx context.Context, path string) error { return nil }

// Mock technique repository
type wsTestTechniqueRepo struct{}

func (m *wsTestTechniqueRepo) Create(ctx context.Context, t *entity.Technique) error  { return nil }
func (m *wsTestTechniqueRepo) Update(ctx context.Context, t *entity.Technique) error  { return nil }
func (m *wsTestTechniqueRepo) Delete(ctx context.Context, id string) error            { return nil }
func (m *wsTestTechniqueRepo) FindByID(ctx context.Context, id string) (*entity.Technique, error) {
	return &entity.Technique{ID: id}, nil
}
func (m *wsTestTechniqueRepo) FindAll(ctx context.Context) ([]*entity.Technique, error) {
	return []*entity.Technique{}, nil
}
func (m *wsTestTechniqueRepo) FindByTactic(ctx context.Context, tactic entity.TacticType) ([]*entity.Technique, error) {
	return []*entity.Technique{}, nil
}
func (m *wsTestTechniqueRepo) FindByPlatform(ctx context.Context, platform string) ([]*entity.Technique, error) {
	return []*entity.Technique{}, nil
}
func (m *wsTestTechniqueRepo) ImportFromYAML(ctx context.Context, path string) error { return nil }

func TestWebSocketHandler_HandleTaskResult_WithRealExecutionService(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	go hub.Run()

	agentRepo := newWSTestAgentRepo()
	agentService := application.NewAgentService(agentRepo)
	handler := NewWebSocketHandler(hub, agentService, logger)

	// Create mock repositories
	resultRepo := newWSTestResultRepo()
	scenarioRepo := &wsTestScenarioRepo{}
	techniqueRepo := &wsTestTechniqueRepo{}

	// Create real execution with a pending result
	resultRepo.executions["exec-1"] = &entity.Execution{
		ID:     "exec-1",
		Status: entity.ExecutionRunning,
	}
	resultRepo.results["task-result-test"] = &entity.ExecutionResult{
		ID:          "task-result-test",
		ExecutionID: "exec-1",
		AgentPaw:    "test-agent",
		Status:      entity.StatusPending,
	}

	// Create execution service
	execService := application.NewExecutionService(
		resultRepo,
		scenarioRepo,
		techniqueRepo,
		agentRepo,
		nil, // orchestrator not needed for UpdateResultByID
		nil, // calculator - need it for checkAndCompleteExecution
	)

	handler.SetExecutionService(execService)

	client := websocket.NewClient(hub, nil, "test-agent", logger)

	// Test successful result
	payload := TaskResultPayload{
		TaskID:      "task-result-test",
		TechniqueID: "T1082",
		Success:     true,
		ExitCode:    0,
		Output:      "System info collected successfully",
	}
	payloadBytes, _ := json.Marshal(payload)

	handler.handleTaskResult(client, payloadBytes)

	// Verify result was updated
	result, err := resultRepo.FindResultByID(context.Background(), "task-result-test")
	if err != nil {
		t.Fatalf("Result not found after update: %v", err)
	}
	if result.Status != entity.StatusSuccess {
		t.Errorf("Expected status 'success', got '%s'", result.Status)
	}
}

func TestWebSocketHandler_HandleTaskResult_WithExecutionServiceError(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	go hub.Run()

	agentRepo := newWSTestAgentRepo()
	agentService := application.NewAgentService(agentRepo)
	handler := NewWebSocketHandler(hub, agentService, logger)

	// Create mock repositories
	resultRepo := newWSTestResultRepo()
	scenarioRepo := &wsTestScenarioRepo{}
	techniqueRepo := &wsTestTechniqueRepo{}

	// Create execution service (result doesn't exist so FindResultByID will fail)
	execService := application.NewExecutionService(
		resultRepo,
		scenarioRepo,
		techniqueRepo,
		agentRepo,
		nil,
		nil,
	)

	handler.SetExecutionService(execService)

	client := websocket.NewClient(hub, nil, "test-agent", logger)

	// Test with non-existent result (should trigger error path)
	payload := TaskResultPayload{
		TaskID:      "non-existent-task",
		TechniqueID: "T1082",
		Success:     true,
		ExitCode:    0,
		Output:      "Output",
	}
	payloadBytes, _ := json.Marshal(payload)

	// Should not panic, just log error
	handler.handleTaskResult(client, payloadBytes)
}

func TestWebSocketHandler_HandleTaskResult_FailedResultWithError(t *testing.T) {
	logger := zap.NewNop()
	hub := websocket.NewHub(logger)
	go hub.Run()

	agentRepo := newWSTestAgentRepo()
	agentService := application.NewAgentService(agentRepo)
	handler := NewWebSocketHandler(hub, agentService, logger)

	// Create mock repositories
	resultRepo := newWSTestResultRepo()
	scenarioRepo := &wsTestScenarioRepo{}
	techniqueRepo := &wsTestTechniqueRepo{}

	// Create real execution with a pending result
	resultRepo.executions["exec-2"] = &entity.Execution{
		ID:     "exec-2",
		Status: entity.ExecutionRunning,
	}
	resultRepo.results["task-failed-test"] = &entity.ExecutionResult{
		ID:          "task-failed-test",
		ExecutionID: "exec-2",
		AgentPaw:    "test-agent",
		Status:      entity.StatusPending,
	}

	execService := application.NewExecutionService(
		resultRepo,
		scenarioRepo,
		techniqueRepo,
		agentRepo,
		nil,
		nil,
	)

	handler.SetExecutionService(execService)

	client := websocket.NewClient(hub, nil, "test-agent", logger)

	// Test failed result with error message
	payload := TaskResultPayload{
		TaskID:      "task-failed-test",
		TechniqueID: "T1059",
		Success:     false,
		ExitCode:    127,
		Output:      "partial output",
		Error:       "command not found",
	}
	payloadBytes, _ := json.Marshal(payload)

	handler.handleTaskResult(client, payloadBytes)

	// Verify result was updated with failed status
	result, err := resultRepo.FindResultByID(context.Background(), "task-failed-test")
	if err != nil {
		t.Fatalf("Result not found after update: %v", err)
	}
	if result.Status != entity.StatusFailed {
		t.Errorf("Expected status 'failed', got '%s'", result.Status)
	}
}
