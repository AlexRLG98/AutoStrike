package handlers

import (
	"encoding/json"
	"net/http"

	"autostrike/internal/application"
	"autostrike/internal/infrastructure/websocket"

	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = gorillaws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub          *websocket.Hub
	agentService *application.AgentService
	logger       *zap.Logger
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *websocket.Hub, agentService *application.AgentService, logger *zap.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		hub:          hub,
		agentService: agentService,
		logger:       logger,
	}
}

// HandleAgentConnection handles WebSocket connections from agents
func (h *WebSocketHandler) HandleAgentConnection(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade WebSocket", zap.Error(err))
		return
	}

	// Create client with empty paw (will be set on registration)
	client := websocket.NewClient(h.hub, conn, "", h.logger)

	// Register client
	h.hub.Register(client)

	// Start read/write pumps
	go client.WritePump()
	go client.ReadPump(h.handleMessage)
}

// handleMessage processes incoming WebSocket messages
func (h *WebSocketHandler) handleMessage(client *websocket.Client, msg *websocket.Message) {
	switch msg.Type {
	case "register":
		h.handleRegister(client, msg.Payload)
	case "heartbeat":
		h.handleHeartbeat(client, msg.Payload)
	case "task_result":
		h.handleTaskResult(client, msg.Payload)
	default:
		h.logger.Warn("Unknown message type", zap.String("type", msg.Type))
	}
}

// RegisterPayload represents agent registration data
type RegisterPayload struct {
	Paw       string   `json:"paw"`
	Hostname  string   `json:"hostname"`
	Username  string   `json:"username"`
	Platform  string   `json:"platform"`
	Executors []string `json:"executors"`
}

func (h *WebSocketHandler) handleRegister(client *websocket.Client, payload json.RawMessage) {
	var reg RegisterPayload
	if err := json.Unmarshal(payload, &reg); err != nil {
		h.logger.Error("Failed to parse register payload", zap.Error(err))
		return
	}

	h.logger.Info("Agent registering",
		zap.String("paw", reg.Paw),
		zap.String("hostname", reg.Hostname),
		zap.String("platform", reg.Platform),
	)

	// Update client paw
	client.SetAgentPaw(reg.Paw)

	// Register/update agent in database
	ctx := client.Context()
	err := h.agentService.RegisterOrUpdate(ctx, reg.Paw, reg.Hostname, reg.Username, reg.Platform, reg.Executors)
	if err != nil {
		h.logger.Error("Failed to register agent", zap.Error(err), zap.String("paw", reg.Paw))
		return
	}

	// Send acknowledgment
	_ = client.Send("registered", map[string]string{"status": "ok", "paw": reg.Paw})
}

func (h *WebSocketHandler) handleHeartbeat(client *websocket.Client, payload json.RawMessage) {
	paw := client.GetAgentPaw()
	if paw == "" {
		return
	}

	ctx := client.Context()
	if err := h.agentService.UpdateHeartbeat(ctx, paw); err != nil {
		h.logger.Error("Failed to update heartbeat", zap.Error(err), zap.String("paw", paw))
	}
}

// TaskResultPayload represents task execution result from agent
type TaskResultPayload struct {
	TaskID   string `json:"task_id"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
}

func (h *WebSocketHandler) handleTaskResult(client *websocket.Client, payload json.RawMessage) {
	var result TaskResultPayload
	if err := json.Unmarshal(payload, &result); err != nil {
		h.logger.Warn("Failed to parse task result payload", zap.Error(err))
		return
	}

	h.logger.Info("Received task result",
		zap.String("paw", client.GetAgentPaw()),
		zap.String("task_id", result.TaskID),
		zap.Int("exit_code", result.ExitCode),
	)

	// Send acknowledgment back to agent
	_ = client.Send("task_ack", map[string]string{"task_id": result.TaskID, "status": "received"})
}

// RegisterRoutes registers WebSocket routes
func (h *WebSocketHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/ws/agent", h.HandleAgentConnection)
}
