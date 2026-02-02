package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512KB
)

// Client represents a WebSocket client connection
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	agentPaw string
	pawMu    sync.RWMutex
	logger   *zap.Logger
}

// Message represents a WebSocket message
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, agentPaw string, logger *zap.Logger) *Client {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		agentPaw: agentPaw,
		logger:   logger,
	}
}

// ReadPump reads messages from the WebSocket connection
func (c *Client) ReadPump(handler func(*Client, *Message)) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		c.logger.Error("Failed to set read deadline", zap.Error(err))
		return
	}
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("WebSocket error", zap.Error(err))
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			c.logger.Warn("Invalid message format", zap.Error(err))
			continue
		}

		handler(c, &msg)
	}
}

// WritePump writes messages to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !c.handleOutgoingMessage(message, ok) {
				return
			}
		case <-ticker.C:
			if !c.sendPing() {
				return
			}
		}
	}
}

// handleOutgoingMessage processes a message from the send channel
func (c *Client) handleOutgoingMessage(message []byte, ok bool) bool {
	if c.conn.SetWriteDeadline(time.Now().Add(writeWait)) != nil {
		return false
	}

	if !ok {
		_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
		return false
	}

	return c.writeMessageWithQueue(message)
}

// writeMessageWithQueue writes a message and any queued messages
func (c *Client) writeMessageWithQueue(message []byte) bool {
	w, err := c.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return false
	}

	if _, err := w.Write(message); err != nil {
		return false
	}

	if !c.writeQueuedMessages(w) {
		return false
	}

	return w.Close() == nil
}

// writeQueuedMessages writes any additional queued messages
func (c *Client) writeQueuedMessages(w interface{ Write([]byte) (int, error) }) bool {
	n := len(c.send)
	for i := 0; i < n; i++ {
		if _, err := w.Write([]byte{'\n'}); err != nil {
			return false
		}
		if _, err := w.Write(<-c.send); err != nil {
			return false
		}
	}
	return true
}

// sendPing sends a ping message to keep the connection alive
func (c *Client) sendPing() bool {
	if c.conn.SetWriteDeadline(time.Now().Add(writeWait)) != nil {
		return false
	}
	return c.conn.WriteMessage(websocket.PingMessage, nil) == nil
}

// Send sends a message to the client
func (c *Client) Send(msgType string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := Message{
		Type:    msgType,
		Payload: data,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	c.send <- msgBytes
	return nil
}

// GetAgentPaw returns the agent paw if this is an agent connection
func (c *Client) GetAgentPaw() string {
	c.pawMu.RLock()
	defer c.pawMu.RUnlock()
	return c.agentPaw
}

// SetAgentPaw sets the agent paw for this connection
func (c *Client) SetAgentPaw(paw string) {
	c.pawMu.Lock()
	defer c.pawMu.Unlock()
	c.agentPaw = paw
}

// Context returns a background context for database operations
func (c *Client) Context() context.Context {
	return context.Background()
}
