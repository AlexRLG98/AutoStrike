package application

import (
	"context"
	"fmt"
	"time"

	"autostrike/internal/domain/entity"
	"autostrike/internal/domain/repository"
)

// AgentService handles agent-related business logic
type AgentService struct {
	repo repository.AgentRepository
}

// NewAgentService creates a new agent service
func NewAgentService(repo repository.AgentRepository) *AgentService {
	return &AgentService{repo: repo}
}

// RegisterAgent registers a new agent or updates existing one
func (s *AgentService) RegisterAgent(ctx context.Context, agent *entity.Agent) error {
	existing, err := s.repo.FindByPaw(ctx, agent.Paw)
	if err == nil && existing != nil {
		// Update existing agent
		existing.Status = entity.AgentOnline
		existing.LastSeen = time.Now()
		existing.Platform = agent.Platform
		existing.Executors = agent.Executors
		existing.Hostname = agent.Hostname
		existing.Username = agent.Username
		return s.repo.Update(ctx, existing)
	}

	// Create new agent
	agent.Status = entity.AgentOnline
	agent.LastSeen = time.Now()
	agent.CreatedAt = time.Now()
	return s.repo.Create(ctx, agent)
}

// Heartbeat updates agent's last seen timestamp
func (s *AgentService) Heartbeat(ctx context.Context, paw string) error {
	return s.repo.UpdateLastSeen(ctx, paw)
}

// GetAgent retrieves an agent by paw
func (s *AgentService) GetAgent(ctx context.Context, paw string) (*entity.Agent, error) {
	return s.repo.FindByPaw(ctx, paw)
}

// GetAllAgents retrieves all agents
func (s *AgentService) GetAllAgents(ctx context.Context) ([]*entity.Agent, error) {
	return s.repo.FindAll(ctx)
}

// GetOnlineAgents retrieves only online agents
func (s *AgentService) GetOnlineAgents(ctx context.Context) ([]*entity.Agent, error) {
	return s.repo.FindByStatus(ctx, entity.AgentOnline)
}

// MarkAgentOffline marks an agent as offline
func (s *AgentService) MarkAgentOffline(ctx context.Context, paw string) error {
	agent, err := s.repo.FindByPaw(ctx, paw)
	if err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	agent.Status = entity.AgentOffline
	return s.repo.Update(ctx, agent)
}

// DeleteAgent removes an agent
func (s *AgentService) DeleteAgent(ctx context.Context, paw string) error {
	return s.repo.Delete(ctx, paw)
}

// CheckStaleAgents marks agents as offline if they haven't been seen recently
func (s *AgentService) CheckStaleAgents(ctx context.Context, timeout time.Duration) error {
	agents, err := s.repo.FindByStatus(ctx, entity.AgentOnline)
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-timeout)
	for _, agent := range agents {
		if agent.LastSeen.Before(cutoff) {
			agent.Status = entity.AgentOffline
			if err := s.repo.Update(ctx, agent); err != nil {
				return err
			}
		}
	}

	return nil
}

// RegisterOrUpdate registers a new agent or updates an existing one (WebSocket handler)
func (s *AgentService) RegisterOrUpdate(ctx context.Context, paw, hostname, username, platform string, executors []string) error {
	agent := &entity.Agent{
		Paw:       paw,
		Hostname:  hostname,
		Username:  username,
		Platform:  platform,
		Executors: executors,
	}
	return s.RegisterAgent(ctx, agent)
}

// UpdateHeartbeat updates the last seen timestamp for an agent (WebSocket handler)
func (s *AgentService) UpdateHeartbeat(ctx context.Context, paw string) error {
	return s.Heartbeat(ctx, paw)
}
