package websocket

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewHub(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	if hub == nil {
		t.Fatal("NewHub returned nil")
	}

	if hub.clients == nil {
		t.Error("clients map is nil")
	}

	if hub.agents == nil {
		t.Error("agents map is nil")
	}

	if hub.broadcast == nil {
		t.Error("broadcast channel is nil")
	}

	if hub.register == nil {
		t.Error("register channel is nil")
	}

	if hub.unregister == nil {
		t.Error("unregister channel is nil")
	}
}

func TestHub_GetConnectedAgents_Empty(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	agents := hub.GetConnectedAgents()

	if len(agents) != 0 {
		t.Errorf("Expected empty agents list, got %d", len(agents))
	}
}

func TestHub_IsAgentConnected_NotConnected(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	if hub.IsAgentConnected("non-existent-paw") {
		t.Error("Expected IsAgentConnected to return false for non-existent agent")
	}
}

func TestHub_SendToAgent_NotConnected(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	result := hub.SendToAgent("non-existent-paw", []byte("test message"))

	if result {
		t.Error("Expected SendToAgent to return false for non-existent agent")
	}
}

func TestHub_handleRegister(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	client := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		agentPaw: "test-agent",
	}

	hub.handleRegister(client)

	if _, ok := hub.clients[client]; !ok {
		t.Error("Client was not registered")
	}

	if hub.agents["test-agent"] != client {
		t.Error("Agent was not registered with paw")
	}
}

func TestHub_handleRegister_NoAgentPaw(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	client := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		agentPaw: "", // No paw
	}

	hub.handleRegister(client)

	if _, ok := hub.clients[client]; !ok {
		t.Error("Client was not registered")
	}

	if len(hub.agents) != 0 {
		t.Error("Client without paw should not be in agents map")
	}
}

func TestHub_handleUnregister(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	client := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		agentPaw: "test-agent",
	}

	// First register
	hub.handleRegister(client)

	// Then unregister
	hub.handleUnregister(client)

	if _, ok := hub.clients[client]; ok {
		t.Error("Client was not unregistered")
	}

	if _, ok := hub.agents["test-agent"]; ok {
		t.Error("Agent was not unregistered")
	}
}

func TestHub_handleUnregister_NotRegistered(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	client := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		agentPaw: "test-agent",
	}

	// Unregister without registering first (should not panic)
	hub.handleUnregister(client)

	if len(hub.clients) != 0 {
		t.Error("Clients map should be empty")
	}
}

func TestHub_GetConnectedAgents(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	client1 := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		agentPaw: "agent-1",
	}

	client2 := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		agentPaw: "agent-2",
	}

	hub.handleRegister(client1)
	hub.handleRegister(client2)

	agents := hub.GetConnectedAgents()

	if len(agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(agents))
	}

	// Check both agents are in the list
	foundAgent1, foundAgent2 := false, false
	for _, paw := range agents {
		if paw == "agent-1" {
			foundAgent1 = true
		}
		if paw == "agent-2" {
			foundAgent2 = true
		}
	}

	if !foundAgent1 {
		t.Error("agent-1 not found in connected agents")
	}
	if !foundAgent2 {
		t.Error("agent-2 not found in connected agents")
	}
}

func TestHub_IsAgentConnected(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	client := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		agentPaw: "test-agent",
	}

	hub.handleRegister(client)

	if !hub.IsAgentConnected("test-agent") {
		t.Error("Expected IsAgentConnected to return true for registered agent")
	}
}

func TestHub_SendToAgent(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	client := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		agentPaw: "test-agent",
	}

	hub.handleRegister(client)

	message := []byte("test message")
	result := hub.SendToAgent("test-agent", message)

	if !result {
		t.Error("Expected SendToAgent to return true")
	}

	// Check if message was sent to client's send channel
	select {
	case received := <-client.send:
		if string(received) != string(message) {
			t.Errorf("Expected message %s, got %s", message, received)
		}
	default:
		t.Error("No message received on client's send channel")
	}
}

func TestHub_SendToAgent_FullChannel(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	// Create client with small buffer
	client := &Client{
		hub:      hub,
		send:     make(chan []byte, 1), // Only 1 capacity
		agentPaw: "test-agent",
	}

	hub.handleRegister(client)

	// Fill the channel
	client.send <- []byte("first")

	// Try to send another message (should return false due to full channel)
	result := hub.SendToAgent("test-agent", []byte("second"))

	if result {
		t.Error("Expected SendToAgent to return false when channel is full")
	}
}

func TestHub_Broadcast(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	// This test just verifies Broadcast doesn't panic
	// and sends to the broadcast channel
	go func() {
		<-hub.broadcast // Consume the message
	}()

	hub.Broadcast([]byte("test broadcast"))
}

func TestHub_handleBroadcast(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	client1 := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		agentPaw: "agent-1",
	}

	client2 := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		agentPaw: "agent-2",
	}

	hub.handleRegister(client1)
	hub.handleRegister(client2)

	message := []byte("broadcast message")
	hub.handleBroadcast(message)

	// Check both clients received the message
	select {
	case received := <-client1.send:
		if string(received) != string(message) {
			t.Errorf("client1: Expected message %s, got %s", message, received)
		}
	default:
		t.Error("client1: No message received")
	}

	select {
	case received := <-client2.send:
		if string(received) != string(message) {
			t.Errorf("client2: Expected message %s, got %s", message, received)
		}
	default:
		t.Error("client2: No message received")
	}
}

func TestHub_Register_Channel(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	// Start hub in background
	go hub.Run()

	client := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		agentPaw: "channel-test-agent",
	}

	// Use the public Register method
	hub.Register(client)

	// Wait for registration to be processed
	time.Sleep(50 * time.Millisecond)

	// Verify agent is connected
	if !hub.IsAgentConnected("channel-test-agent") {
		t.Error("Agent should be connected after Register")
	}
}

func TestHub_Unregister_Channel(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	// Start hub in background
	go hub.Run()

	client := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		agentPaw: "unregister-test-agent",
	}

	// Register first
	hub.Register(client)
	time.Sleep(50 * time.Millisecond)

	if !hub.IsAgentConnected("unregister-test-agent") {
		t.Fatal("Agent should be connected")
	}

	// Then unregister
	hub.Unregister(client)
	time.Sleep(50 * time.Millisecond)

	if hub.IsAgentConnected("unregister-test-agent") {
		t.Error("Agent should not be connected after Unregister")
	}
}

func TestHub_Register_MultipleAgents(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	go hub.Run()

	clients := make([]*Client, 5)
	for i := 0; i < 5; i++ {
		clients[i] = &Client{
			hub:      hub,
			send:     make(chan []byte, 256),
			agentPaw: "multi-agent-" + string(rune('A'+i)),
		}
		hub.Register(clients[i])
	}

	time.Sleep(100 * time.Millisecond)

	agents := hub.GetConnectedAgents()
	if len(agents) != 5 {
		t.Errorf("Expected 5 agents, got %d", len(agents))
	}
}

func TestHub_handleBroadcast_FullChannel(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	// Create client with full send channel
	client := &Client{
		hub:      hub,
		send:     make(chan []byte, 1),
		agentPaw: "full-channel-agent",
	}

	hub.handleRegister(client)

	// Fill the channel
	client.send <- []byte("blocking message")

	// Broadcast should remove client with full channel
	hub.handleBroadcast([]byte("broadcast"))

	// Client should be removed
	if _, ok := hub.clients[client]; ok {
		t.Error("Client with full channel should be removed")
	}
}

func TestHub_ConcurrentAccess(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	go hub.Run()

	// Concurrent registration
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			client := &Client{
				hub:      hub,
				send:     make(chan []byte, 256),
				agentPaw: "concurrent-agent-" + string(rune('0'+id)),
			}
			hub.Register(client)
			time.Sleep(10 * time.Millisecond)
			hub.Unregister(client)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Hub should still work
	agents := hub.GetConnectedAgents()
	if agents == nil {
		t.Error("GetConnectedAgents should not return nil")
	}
}
