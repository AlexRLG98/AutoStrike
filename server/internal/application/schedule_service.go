package application

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"autostrike/internal/domain/entity"
	"autostrike/internal/domain/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ErrScheduleNotFound is returned when a schedule is not found
var ErrScheduleNotFound = errors.New("schedule not found")

// ScheduleService handles schedule-related business logic
type ScheduleService struct {
	scheduleRepo     repository.ScheduleRepository
	executionService *ExecutionService
	logger           *zap.Logger
	stopChan         chan struct{}
	wg               sync.WaitGroup
	running          bool
	mu               sync.Mutex
}

// NewScheduleService creates a new schedule service
func NewScheduleService(
	scheduleRepo repository.ScheduleRepository,
	executionService *ExecutionService,
	logger *zap.Logger,
) *ScheduleService {
	return &ScheduleService{
		scheduleRepo:     scheduleRepo,
		executionService: executionService,
		logger:           logger,
		stopChan:         make(chan struct{}),
	}
}

// CreateScheduleRequest represents the request to create a schedule
type CreateScheduleRequest struct {
	Name        string                   `json:"name" binding:"required"`
	Description string                   `json:"description"`
	ScenarioID  string                   `json:"scenario_id" binding:"required"`
	AgentPaw    string                   `json:"agent_paw"`
	Frequency   entity.ScheduleFrequency `json:"frequency" binding:"required"`
	CronExpr    string                   `json:"cron_expr"`
	SafeMode    bool                     `json:"safe_mode"`
	StartAt     *time.Time               `json:"start_at"`
}

// ErrInvalidCronExpr is returned when a cron expression is invalid
var ErrInvalidCronExpr = errors.New("invalid cron expression")

// Create creates a new schedule
func (s *ScheduleService) Create(ctx context.Context, req *CreateScheduleRequest, userID string) (*entity.Schedule, error) {
	// Validate cron expression if frequency is cron
	if req.Frequency == entity.FrequencyCron {
		if req.CronExpr == "" {
			return nil, fmt.Errorf("cron expression required for cron frequency: %w", ErrInvalidCronExpr)
		}
		if entity.ValidateCronExpr(req.CronExpr) != nil {
			return nil, fmt.Errorf("invalid cron expression '%s': %w", req.CronExpr, ErrInvalidCronExpr)
		}
	}

	now := time.Now()

	schedule := &entity.Schedule{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		ScenarioID:  req.ScenarioID,
		AgentPaw:    req.AgentPaw,
		Frequency:   req.Frequency,
		CronExpr:    req.CronExpr,
		SafeMode:    req.SafeMode,
		Status:      entity.ScheduleStatusActive,
		CreatedBy:   userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Calculate next run time
	if req.StartAt != nil && req.StartAt.After(now) {
		schedule.NextRunAt = req.StartAt
	} else {
		nextRun := schedule.CalculateNextRun(now)
		schedule.NextRunAt = nextRun
	}

	if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	s.logger.Info("Schedule created",
		zap.String("id", schedule.ID),
		zap.String("name", schedule.Name),
		zap.String("frequency", string(schedule.Frequency)),
	)

	return schedule, nil
}

// Update updates an existing schedule
func (s *ScheduleService) Update(ctx context.Context, id string, req *CreateScheduleRequest) (*entity.Schedule, error) {
	// Validate cron expression if frequency is cron
	if req.Frequency == entity.FrequencyCron {
		if req.CronExpr == "" {
			return nil, fmt.Errorf("cron expression required for cron frequency: %w", ErrInvalidCronExpr)
		}
		if entity.ValidateCronExpr(req.CronExpr) != nil {
			return nil, fmt.Errorf("invalid cron expression '%s': %w", req.CronExpr, ErrInvalidCronExpr)
		}
	}

	schedule, err := s.scheduleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if schedule == nil {
		return nil, ErrScheduleNotFound
	}

	schedule.Name = req.Name
	schedule.Description = req.Description
	schedule.ScenarioID = req.ScenarioID
	schedule.AgentPaw = req.AgentPaw
	schedule.Frequency = req.Frequency
	schedule.CronExpr = req.CronExpr
	schedule.SafeMode = req.SafeMode
	schedule.UpdatedAt = time.Now()

	// Recalculate next run time if frequency changed
	if req.StartAt != nil && req.StartAt.After(time.Now()) {
		schedule.NextRunAt = req.StartAt
	} else {
		nextRun := schedule.CalculateNextRun(time.Now())
		schedule.NextRunAt = nextRun
	}

	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		return nil, fmt.Errorf("failed to update schedule: %w", err)
	}

	return schedule, nil
}

// Delete deletes a schedule
func (s *ScheduleService) Delete(ctx context.Context, id string) error {
	return s.scheduleRepo.Delete(ctx, id)
}

// GetByID retrieves a schedule by ID
func (s *ScheduleService) GetByID(ctx context.Context, id string) (*entity.Schedule, error) {
	return s.scheduleRepo.FindByID(ctx, id)
}

// GetAll retrieves all schedules
func (s *ScheduleService) GetAll(ctx context.Context) ([]*entity.Schedule, error) {
	return s.scheduleRepo.FindAll(ctx)
}

// GetByStatus retrieves schedules by status
func (s *ScheduleService) GetByStatus(ctx context.Context, status entity.ScheduleStatus) ([]*entity.Schedule, error) {
	return s.scheduleRepo.FindByStatus(ctx, status)
}

// Pause pauses a schedule
func (s *ScheduleService) Pause(ctx context.Context, id string) (*entity.Schedule, error) {
	schedule, err := s.scheduleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if schedule == nil {
		return nil, ErrScheduleNotFound
	}

	schedule.Status = entity.ScheduleStatusPaused
	schedule.UpdatedAt = time.Now()

	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}

// Resume resumes a paused schedule
func (s *ScheduleService) Resume(ctx context.Context, id string) (*entity.Schedule, error) {
	schedule, err := s.scheduleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if schedule == nil {
		return nil, ErrScheduleNotFound
	}

	schedule.Status = entity.ScheduleStatusActive
	schedule.UpdatedAt = time.Now()

	// Recalculate next run time
	nextRun := schedule.CalculateNextRun(time.Now())
	schedule.NextRunAt = nextRun

	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}

// GetRuns retrieves runs for a schedule
func (s *ScheduleService) GetRuns(ctx context.Context, scheduleID string, limit int) ([]*entity.ScheduleRun, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.scheduleRepo.FindRunsByScheduleID(ctx, scheduleID, limit)
}

// Start starts the background scheduler
func (s *ScheduleService) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	s.wg.Add(1)
	go s.runScheduler()

	s.logger.Info("Scheduler started")
}

// Stop stops the background scheduler
func (s *ScheduleService) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopChan)
	s.mu.Unlock()

	s.wg.Wait()
	s.logger.Info("Scheduler stopped")
}

// runScheduler runs the background scheduler loop
func (s *ScheduleService) runScheduler() {
	defer s.wg.Done()

	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds for better precision
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkAndRunDueSchedules()
		}
	}
}

// checkAndRunDueSchedules checks for due schedules and runs them
func (s *ScheduleService) checkAndRunDueSchedules() {
	ctx := context.Background()
	now := time.Now()

	schedules, err := s.scheduleRepo.FindActiveSchedulesDue(ctx, now)
	if err != nil {
		s.logger.Error("Failed to find due schedules", zap.Error(err))
		return
	}

	for _, schedule := range schedules {
		s.runSchedule(ctx, schedule)
	}
}

// runSchedule executes a single schedule
func (s *ScheduleService) runSchedule(ctx context.Context, schedule *entity.Schedule) {
	s.logger.Info("Running scheduled execution",
		zap.String("schedule_id", schedule.ID),
		zap.String("name", schedule.Name),
		zap.String("scenario_id", schedule.ScenarioID),
	)

	// Create schedule run record
	run := &entity.ScheduleRun{
		ID:         uuid.New().String(),
		ScheduleID: schedule.ID,
		StartedAt:  time.Now(),
		Status:     "running",
	}

	// Build agent paws list
	var agentPaws []string
	if schedule.AgentPaw != "" {
		agentPaws = []string{schedule.AgentPaw}
	}

	// Start the execution
	result, err := s.executionService.StartExecution(ctx, schedule.ScenarioID, agentPaws, schedule.SafeMode)
	if err != nil {
		s.logger.Error("Failed to start scheduled execution",
			zap.String("schedule_id", schedule.ID),
			zap.Error(err),
		)
		run.Status = "failed"
		run.Error = err.Error()
		completedAt := time.Now()
		run.CompletedAt = &completedAt
	} else {
		run.ExecutionID = result.Execution.ID
		run.Status = "started"
	}

	// Save the run record
	if err := s.scheduleRepo.CreateRun(ctx, run); err != nil {
		s.logger.Error("Failed to save schedule run", zap.Error(err))
	}

	// Update the schedule with last run info and calculate next run
	now := time.Now()
	schedule.LastRunAt = &now
	if result != nil && result.Execution != nil {
		schedule.LastRunID = result.Execution.ID
	}
	schedule.NextRunAt = schedule.CalculateNextRun(now)
	schedule.UpdatedAt = now

	// If one-time schedule and it ran, disable it
	if schedule.Frequency == entity.FrequencyOnce {
		schedule.Status = entity.ScheduleStatusDisabled
	}

	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		s.logger.Error("Failed to update schedule after run", zap.Error(err))
	}
}

// RunNow manually triggers a schedule to run immediately
func (s *ScheduleService) RunNow(ctx context.Context, id string) (*entity.ScheduleRun, error) {
	schedule, err := s.scheduleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if schedule == nil {
		return nil, ErrScheduleNotFound
	}

	// Create schedule run record
	run := &entity.ScheduleRun{
		ID:         uuid.New().String(),
		ScheduleID: schedule.ID,
		StartedAt:  time.Now(),
		Status:     "running",
	}

	// Build agent paws list
	var agentPaws []string
	if schedule.AgentPaw != "" {
		agentPaws = []string{schedule.AgentPaw}
	}

	// Start the execution
	result, err := s.executionService.StartExecution(ctx, schedule.ScenarioID, agentPaws, schedule.SafeMode)
	if err != nil {
		run.Status = "failed"
		run.Error = err.Error()
		completedAt := time.Now()
		run.CompletedAt = &completedAt
	} else {
		run.ExecutionID = result.Execution.ID
		run.Status = "started"
	}

	// Save the run record
	if err := s.scheduleRepo.CreateRun(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to save schedule run: %w", err)
	}

	// Update schedule with last run info
	now := time.Now()
	schedule.LastRunAt = &now
	if result != nil && result.Execution != nil {
		schedule.LastRunID = result.Execution.ID
	}
	schedule.UpdatedAt = now

	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		s.logger.Warn("Failed to update schedule after manual run", zap.Error(err))
	}

	return run, nil
}
