package repository

import (
	"context"

	"autostrike/internal/domain/entity"
)

// AgentRepository defines the interface for agent persistence
type AgentRepository interface {
	Create(ctx context.Context, agent *entity.Agent) error
	Update(ctx context.Context, agent *entity.Agent) error
	Delete(ctx context.Context, paw string) error
	FindByPaw(ctx context.Context, paw string) (*entity.Agent, error)
	FindByPaws(ctx context.Context, paws []string) ([]*entity.Agent, error) // Batch query to avoid N+1
	FindAll(ctx context.Context) ([]*entity.Agent, error)
	FindByStatus(ctx context.Context, status entity.AgentStatus) ([]*entity.Agent, error)
	FindByPlatform(ctx context.Context, platform string) ([]*entity.Agent, error)
	UpdateLastSeen(ctx context.Context, paw string) error
}

// TechniqueRepository defines the interface for technique persistence
type TechniqueRepository interface {
	Create(ctx context.Context, technique *entity.Technique) error
	Update(ctx context.Context, technique *entity.Technique) error
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*entity.Technique, error)
	FindAll(ctx context.Context) ([]*entity.Technique, error)
	FindByTactic(ctx context.Context, tactic entity.TacticType) ([]*entity.Technique, error)
	FindByPlatform(ctx context.Context, platform string) ([]*entity.Technique, error)
	ImportFromYAML(ctx context.Context, path string) error
}

// ScenarioRepository defines the interface for scenario persistence
type ScenarioRepository interface {
	Create(ctx context.Context, scenario *entity.Scenario) error
	Update(ctx context.Context, scenario *entity.Scenario) error
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*entity.Scenario, error)
	FindAll(ctx context.Context) ([]*entity.Scenario, error)
	FindByTag(ctx context.Context, tag string) ([]*entity.Scenario, error)
	ImportFromYAML(ctx context.Context, path string) error
}

// ResultRepository defines the interface for execution result persistence
type ResultRepository interface {
	CreateExecution(ctx context.Context, execution *entity.Execution) error
	UpdateExecution(ctx context.Context, execution *entity.Execution) error
	FindExecutionByID(ctx context.Context, id string) (*entity.Execution, error)
	FindExecutionsByScenario(ctx context.Context, scenarioID string) ([]*entity.Execution, error)
	FindRecentExecutions(ctx context.Context, limit int) ([]*entity.Execution, error)

	CreateResult(ctx context.Context, result *entity.ExecutionResult) error
	UpdateResult(ctx context.Context, result *entity.ExecutionResult) error
	FindResultByID(ctx context.Context, id string) (*entity.ExecutionResult, error)
	FindResultsByExecution(ctx context.Context, executionID string) ([]*entity.ExecutionResult, error)
	FindResultsByTechnique(ctx context.Context, techniqueID string) ([]*entity.ExecutionResult, error)
}

// UserRepository defines the interface for user persistence
type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*entity.User, error)
	FindByUsername(ctx context.Context, username string) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindAll(ctx context.Context) ([]*entity.User, error)
}
