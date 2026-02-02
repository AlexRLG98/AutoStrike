package websocket

import (
	"sync"

	"go.uber.org/zap"
)

// Hub manages WebSocket connections
type Hub struct {
	clients    map[*Client]bool
	agents     map[string]*Client
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	logger     *zap.Logger
}

// NewHub creates a new WebSocket hub
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		agents:     make(map[string]*Client),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.handleRegister(client)

		case client := <-h.unregister:
			h.handleUnregister(client)

		case message := <-h.broadcast:
			h.handleBroadcast(message)
		}
	}
}

// handleRegister handles client registration with proper mutex handling
func (h *Hub) handleRegister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = true
	paw := client.GetAgentPaw()
	if paw != "" {
		h.agents[paw] = client
		h.logger.Info("Agent connected", zap.String("paw", paw))
	}
}

// handleUnregister handles client unregistration with proper mutex handling
func (h *Hub) handleUnregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		paw := client.GetAgentPaw()
		if paw != "" {
			delete(h.agents, paw)
			h.logger.Info("Agent disconnected", zap.String("paw", paw))
		}
		close(client.send)
	}
}

// handleBroadcast handles message broadcasting with proper mutex handling
func (h *Hub) handleBroadcast(message []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(h.clients, client)
			paw := client.GetAgentPaw()
			if paw != "" {
				delete(h.agents, paw)
			}
		}
	}
}

// SendToAgent sends a message to a specific agent
func (h *Hub) SendToAgent(paw string, message []byte) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if client, ok := h.agents[paw]; ok {
		select {
		case client.send <- message:
			return true
		default:
			return false
		}
	}
	return false
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

// GetConnectedAgents returns list of connected agent paws
func (h *Hub) GetConnectedAgents() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	paws := make([]string, 0, len(h.agents))
	for paw := range h.agents {
		paws = append(paws, paw)
	}
	return paws
}

// IsAgentConnected checks if an agent is connected
func (h *Hub) IsAgentConnected(paw string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	_, ok := h.agents[paw]
	return ok
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}
