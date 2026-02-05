package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"autostrike/internal/domain/entity"
	"autostrike/internal/domain/service"
)

type mockResultRepo struct {
	executions       map[string]*entity.Execution
	results          map[string][]*entity.ExecutionResult
	err              error
	createResultErr  error
	findResultsErr   error
	updateResultErr  error
}

func newMockResultRepo() *mockResultRepo {
	return &mockResultRepo{
		executions: make(map[string]*entity.Execution),
		results:    make(map[string][]*entity.ExecutionResult),
	}
}

func (m *mockResultRepo) CreateExecution(ctx context.Context, e *entity.Execution) error {
	if m.err != nil {
		return m.err
	}
	m.executions[e.ID] = e
	return nil
}

func (m *mockResultRepo) UpdateExecution(ctx context.Context, e *entity.Execution) error {
	if m.err != nil {
		return m.err
	}
	m.executions[e.ID] = e
	return nil
}

func (m *mockResultRepo) FindExecutionByID(ctx context.Context, id string) (*entity.Execution, error) {
	if m.err != nil {
		return nil, m.err
	}
	e, ok := m.executions[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return e, nil
}

func (m *mockResultRepo) FindExecutionsByScenario(ctx context.Context, scenarioID string) ([]*entity.Execution, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*entity.Execution
	for _, e := range m.executions {
		if e.ScenarioID == scenarioID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockResultRepo) FindRecentExecutions(ctx context.Context, limit int) ([]*entity.Execution, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make([]*entity.Execution, 0, len(m.executions))
	for _, e := range m.executions {
		result = append(result, e)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (m *mockResultRepo) CreateResult(ctx context.Context, r *entity.ExecutionResult) error {
	if m.createResultErr != nil {
		return m.createResultErr
	}
	if m.err != nil {
		return m.err
	}
	m.results[r.ExecutionID] = append(m.results[r.ExecutionID], r)
	return nil
}

func (m *mockResultRepo) UpdateResult(ctx context.Context, r *entity.ExecutionResult) error {
	if m.updateResultErr != nil {
		return m.updateResultErr
	}
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *mockResultRepo) FindResultByID(ctx context.Context, id string) (*entity.ExecutionResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, results := range m.results {
		for _, r := range results {
			if r.ID == id {
				return r, nil
			}
		}
	}
	return nil, errors.New("result not found")
}

func (m *mockResultRepo) FindResultsByExecution(ctx context.Context, executionID string) ([]*entity.ExecutionResult, error) {
	if m.findResultsErr != nil {
		return nil, m.findResultsErr
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.results[executionID], nil
}

func (m *mockResultRepo) FindResultsByTechnique(ctx context.Context, techniqueID string) ([]*entity.ExecutionResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*entity.ExecutionResult
	for _, results := range m.results {
		for _, r := range results {
			if r.TechniqueID == techniqueID {
				result = append(result, r)
			}
		}
	}
	return result, nil
}

func (m *mockResultRepo) FindExecutionsByDateRange(ctx context.Context, start, end time.Time) ([]*entity.Execution, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*entity.Execution
	for _, e := range m.executions {
		if !e.StartedAt.Before(start) && !e.StartedAt.After(end) {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockResultRepo) FindCompletedExecutionsByDateRange(ctx context.Context, start, end time.Time) ([]*entity.Execution, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*entity.Execution
	for _, e := range m.executions {
		if e.Status == entity.ExecutionCompleted && !e.StartedAt.Before(start) && !e.StartedAt.After(end) {
			result = append(result, e)
		}
	}
	return result, nil
}

func TestNewExecutionService(t *testing.T) {
	resultRepo := newMockResultRepo()
	scenarioRepo := newMockScenarioRepo()
	techRepo := newMockTechniqueRepo()
	agentRepo := newMockAgentRepo()
	validator := service.NewTechniqueValidator()
	orchestrator := service.NewAttackOrchestrator(agentRepo, techRepo, validator, nil)
	calculator := service.NewScoreCalculator()

	svc := NewExecutionService(resultRepo, scenarioRepo, techRepo, agentRepo, orchestrator, calculator)
	if svc == nil {
		t.Fatal("Expected non-nil service")
	}
}

func TestGetExecution(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{ID: "e1", ScenarioID: "s1"}

	svc := &ExecutionService{resultRepo: resultRepo}
	exec, err := svc.GetExecution(context.Background(), "e1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if exec.ScenarioID != "s1" {
		t.Errorf("Expected scenario s1, got %s", exec.ScenarioID)
	}
}

func TestGetExecutionNotFound(t *testing.T) {
	resultRepo := newMockResultRepo()
	svc := &ExecutionService{resultRepo: resultRepo}

	_, err := svc.GetExecution(context.Background(), "invalid")
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestGetExecutionResults(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.results["e1"] = []*entity.ExecutionResult{
		{ID: "r1", ExecutionID: "e1"},
		{ID: "r2", ExecutionID: "e1"},
	}

	svc := &ExecutionService{resultRepo: resultRepo}
	results, err := svc.GetExecutionResults(context.Background(), "e1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestGetRecentExecutions(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{ID: "e1"}
	resultRepo.executions["e2"] = &entity.Execution{ID: "e2"}
	resultRepo.executions["e3"] = &entity.Execution{ID: "e3"}

	svc := &ExecutionService{resultRepo: resultRepo}
	execs, err := svc.GetRecentExecutions(context.Background(), 2)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(execs) > 2 {
		t.Errorf("Expected at most 2 executions, got %d", len(execs))
	}
}

func TestStartExecutionScenarioNotFound(t *testing.T) {
	resultRepo := newMockResultRepo()
	scenarioRepo := newMockScenarioRepo()
	agentRepo := newMockAgentRepo()

	svc := &ExecutionService{
		resultRepo:   resultRepo,
		scenarioRepo: scenarioRepo,
		agentRepo:    agentRepo,
	}

	_, err := svc.StartExecution(context.Background(), "invalid", []string{"paw1"}, false)
	if err == nil {
		t.Fatal("Expected error for missing scenario")
	}
}

func TestStartExecutionAgentNotFound(t *testing.T) {
	resultRepo := newMockResultRepo()
	scenarioRepo := newMockScenarioRepo()
	scenarioRepo.scenarios["s1"] = &entity.Scenario{ID: "s1"}
	agentRepo := newMockAgentRepo()

	svc := &ExecutionService{
		resultRepo:   resultRepo,
		scenarioRepo: scenarioRepo,
		agentRepo:    agentRepo,
	}

	_, err := svc.StartExecution(context.Background(), "s1", []string{"invalid"}, false)
	if err == nil {
		t.Fatal("Expected error for missing agent")
	}
}

func TestStartExecutionAgentOffline(t *testing.T) {
	resultRepo := newMockResultRepo()
	scenarioRepo := newMockScenarioRepo()
	scenarioRepo.scenarios["s1"] = &entity.Scenario{ID: "s1"}
	agentRepo := newMockAgentRepo()
	agentRepo.agents["paw1"] = &entity.Agent{Paw: "paw1", Status: entity.AgentOffline}

	svc := &ExecutionService{
		resultRepo:   resultRepo,
		scenarioRepo: scenarioRepo,
		agentRepo:    agentRepo,
	}

	_, err := svc.StartExecution(context.Background(), "s1", []string{"paw1"}, false)
	if err == nil {
		t.Fatal("Expected error for offline agent")
	}
}

func TestCompleteExecution(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{
		ID:     "e1",
		Status: entity.ExecutionRunning,
	}
	resultRepo.results["e1"] = []*entity.ExecutionResult{
		{ID: "r1", Status: entity.StatusSuccess},
	}
	calculator := service.NewScoreCalculator()

	svc := &ExecutionService{
		resultRepo: resultRepo,
		calculator: calculator,
	}

	err := svc.CompleteExecution(context.Background(), "e1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	exec := resultRepo.executions["e1"]
	if exec.Status != entity.ExecutionCompleted {
		t.Errorf("Expected status Completed, got %v", exec.Status)
	}
}

func TestCompleteExecutionNotFound(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.err = errors.New("not found")

	svc := &ExecutionService{resultRepo: resultRepo}
	err := svc.CompleteExecution(context.Background(), "invalid")
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestUpdateResult(t *testing.T) {
	resultRepo := newMockResultRepo()
	// The implementation uses resultID as executionID when looking up results
	// So we need to use the resultID "r1" as the key
	resultRepo.results["r1"] = []*entity.ExecutionResult{
		{ID: "r1", ExecutionID: "e1", Status: entity.StatusPending},
	}

	svc := &ExecutionService{resultRepo: resultRepo}
	err := svc.UpdateResult(context.Background(), "r1", entity.StatusSuccess, "output", false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestUpdateResultNotFound(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.results["e1"] = []*entity.ExecutionResult{}

	svc := &ExecutionService{resultRepo: resultRepo}
	err := svc.UpdateResult(context.Background(), "invalid", entity.StatusSuccess, "", false)
	if err == nil {
		t.Fatal("Expected error for not found result")
	}
}

func TestStartExecutionSuccess(t *testing.T) {
	resultRepo := newMockResultRepo()
	scenarioRepo := newMockScenarioRepo()
	scenarioRepo.scenarios["s1"] = &entity.Scenario{
		ID:   "s1",
		Name: "Test",
		Phases: []entity.Phase{
			{Name: "Phase1", Techniques: []string{"T1059"}},
		},
	}
	techRepo := newMockTechniqueRepo()
	techRepo.techniques["T1059"] = &entity.Technique{
		ID:        "T1059",
		Platforms: []string{"linux"},
		Executors: []entity.Executor{{Type: "sh", Command: "echo test"}},
	}
	agentRepo := newMockAgentRepo()
	agentRepo.agents["paw1"] = &entity.Agent{
		Paw:       "paw1",
		Status:    entity.AgentOnline,
		Platform:  "linux",
		Executors: []string{"sh"},
		LastSeen:  time.Now(),
	}
	validator := service.NewTechniqueValidator()
	orchestrator := service.NewAttackOrchestrator(agentRepo, techRepo, validator, nil)
	calculator := service.NewScoreCalculator()

	svc := NewExecutionService(resultRepo, scenarioRepo, techRepo, agentRepo, orchestrator, calculator)
	result, err := svc.StartExecution(context.Background(), "s1", []string{"paw1"}, false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result == nil || result.Execution == nil {
		t.Fatal("Expected execution to be created")
	}
	if result.Execution.Status != entity.ExecutionRunning {
		t.Errorf("Expected status Running, got %v", result.Execution.Status)
	}
	if len(result.Tasks) == 0 {
		t.Error("Expected tasks to be returned")
	}
}

func TestUpdateResultRepoError(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.err = errors.New("db error")

	svc := &ExecutionService{resultRepo: resultRepo}
	err := svc.UpdateResult(context.Background(), "r1", entity.StatusSuccess, "", false)
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestCompleteExecutionResultsError(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{
		ID:     "e1",
		Status: entity.ExecutionRunning,
	}
	// Use specific error for FindResultsByExecution
	resultRepo.findResultsErr = errors.New("results error")

	svc := &ExecutionService{resultRepo: resultRepo}
	err := svc.CompleteExecution(context.Background(), "e1")
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestStartExecutionPlanError(t *testing.T) {
	resultRepo := newMockResultRepo()
	scenarioRepo := newMockScenarioRepo()
	scenarioRepo.scenarios["s1"] = &entity.Scenario{
		ID:   "s1",
		Name: "Test",
		Phases: []entity.Phase{
			{Name: "Phase1", Techniques: []string{"T1059"}},
		},
	}
	techRepo := newMockTechniqueRepo()
	// Missing technique causes plan error
	agentRepo := newMockAgentRepo()
	agentRepo.agents["paw1"] = &entity.Agent{
		Paw:       "paw1",
		Status:    entity.AgentOnline,
		Platform:  "linux",
		Executors: []string{"sh"},
		LastSeen:  time.Now(),
	}
	validator := service.NewTechniqueValidator()
	orchestrator := service.NewAttackOrchestrator(agentRepo, techRepo, validator, nil)
	calculator := service.NewScoreCalculator()

	svc := NewExecutionService(resultRepo, scenarioRepo, techRepo, agentRepo, orchestrator, calculator)
	_, err := svc.StartExecution(context.Background(), "s1", []string{"paw1"}, false)
	if err == nil {
		t.Fatal("Expected error for plan failure")
	}
}

func TestStartExecutionCreateError(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.err = errors.New("create error")
	scenarioRepo := newMockScenarioRepo()
	scenarioRepo.scenarios["s1"] = &entity.Scenario{
		ID:   "s1",
		Name: "Test",
		Phases: []entity.Phase{
			{Name: "Phase1", Techniques: []string{"T1059"}},
		},
	}
	techRepo := newMockTechniqueRepo()
	techRepo.techniques["T1059"] = &entity.Technique{
		ID:        "T1059",
		Platforms: []string{"linux"},
		Executors: []entity.Executor{{Type: "sh", Command: "echo test"}},
	}
	agentRepo := newMockAgentRepo()
	agentRepo.agents["paw1"] = &entity.Agent{
		Paw:       "paw1",
		Status:    entity.AgentOnline,
		Platform:  "linux",
		Executors: []string{"sh"},
		LastSeen:  time.Now(),
	}
	validator := service.NewTechniqueValidator()
	orchestrator := service.NewAttackOrchestrator(agentRepo, techRepo, validator, nil)
	calculator := service.NewScoreCalculator()

	svc := NewExecutionService(resultRepo, scenarioRepo, techRepo, agentRepo, orchestrator, calculator)
	_, err := svc.StartExecution(context.Background(), "s1", []string{"paw1"}, false)
	if err == nil {
		t.Fatal("Expected error for create failure")
	}
}

func TestStartExecutionCreateResultError(t *testing.T) {
	resultRepo := newMockResultRepo()
	// Only CreateResult will fail
	resultRepo.createResultErr = errors.New("create result error")
	scenarioRepo := newMockScenarioRepo()
	scenarioRepo.scenarios["s1"] = &entity.Scenario{
		ID:   "s1",
		Name: "Test",
		Phases: []entity.Phase{
			{Name: "Phase1", Techniques: []string{"T1059"}},
		},
	}
	techRepo := newMockTechniqueRepo()
	techRepo.techniques["T1059"] = &entity.Technique{
		ID:        "T1059",
		Platforms: []string{"linux"},
		Executors: []entity.Executor{{Type: "sh", Command: "echo test"}},
	}
	agentRepo := newMockAgentRepo()
	agentRepo.agents["paw1"] = &entity.Agent{
		Paw:       "paw1",
		Status:    entity.AgentOnline,
		Platform:  "linux",
		Executors: []string{"sh"},
		LastSeen:  time.Now(),
	}
	validator := service.NewTechniqueValidator()
	orchestrator := service.NewAttackOrchestrator(agentRepo, techRepo, validator, nil)
	calculator := service.NewScoreCalculator()

	svc := NewExecutionService(resultRepo, scenarioRepo, techRepo, agentRepo, orchestrator, calculator)
	_, err := svc.StartExecution(context.Background(), "s1", []string{"paw1"}, false)
	if err == nil {
		t.Fatal("Expected error for create result failure")
	}
}

func TestCancelExecutionSuccess(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{
		ID:     "e1",
		Status: entity.ExecutionRunning,
	}
	resultRepo.results["e1"] = []*entity.ExecutionResult{
		{ID: "r1", ExecutionID: "e1", Status: entity.StatusPending},
		{ID: "r2", ExecutionID: "e1", Status: entity.StatusRunning},
		{ID: "r3", ExecutionID: "e1", Status: entity.StatusSuccess},
	}

	svc := &ExecutionService{resultRepo: resultRepo}
	err := svc.CancelExecution(context.Background(), "e1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	exec := resultRepo.executions["e1"]
	if exec.Status != entity.ExecutionCancelled {
		t.Errorf("Expected status Cancelled, got %v", exec.Status)
	}
	if exec.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestCancelExecutionPending(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{
		ID:     "e1",
		Status: entity.ExecutionPending,
	}
	resultRepo.results["e1"] = []*entity.ExecutionResult{}

	svc := &ExecutionService{resultRepo: resultRepo}
	err := svc.CancelExecution(context.Background(), "e1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	exec := resultRepo.executions["e1"]
	if exec.Status != entity.ExecutionCancelled {
		t.Errorf("Expected status Cancelled, got %v", exec.Status)
	}
}

func TestCancelExecutionNotFound(t *testing.T) {
	resultRepo := newMockResultRepo()

	svc := &ExecutionService{resultRepo: resultRepo}
	err := svc.CancelExecution(context.Background(), "invalid")
	if err == nil {
		t.Fatal("Expected error for not found execution")
	}
}

func TestCancelExecutionAlreadyCompleted(t *testing.T) {
	resultRepo := newMockResultRepo()
	now := time.Now()
	resultRepo.executions["e1"] = &entity.Execution{
		ID:          "e1",
		Status:      entity.ExecutionCompleted,
		CompletedAt: &now,
	}

	svc := &ExecutionService{resultRepo: resultRepo}
	err := svc.CancelExecution(context.Background(), "e1")
	if err == nil {
		t.Fatal("Expected error for already completed execution")
	}
}

func TestCancelExecutionAlreadyCancelled(t *testing.T) {
	resultRepo := newMockResultRepo()
	now := time.Now()
	resultRepo.executions["e1"] = &entity.Execution{
		ID:          "e1",
		Status:      entity.ExecutionCancelled,
		CompletedAt: &now,
	}

	svc := &ExecutionService{resultRepo: resultRepo}
	err := svc.CancelExecution(context.Background(), "e1")
	if err == nil {
		t.Fatal("Expected error for already cancelled execution")
	}
}

func TestCancelExecutionResultsError(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{
		ID:     "e1",
		Status: entity.ExecutionRunning,
	}
	resultRepo.findResultsErr = errors.New("results error")

	svc := &ExecutionService{resultRepo: resultRepo}
	err := svc.CancelExecution(context.Background(), "e1")
	if err == nil {
		t.Fatal("Expected error for results fetch failure")
	}
}

func TestCancelExecutionUpdateResultError(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{
		ID:     "e1",
		Status: entity.ExecutionRunning,
	}
	resultRepo.results["e1"] = []*entity.ExecutionResult{
		{ID: "r1", ExecutionID: "e1", Status: entity.StatusPending},
	}
	resultRepo.updateResultErr = errors.New("update error")

	svc := &ExecutionService{resultRepo: resultRepo}
	err := svc.CancelExecution(context.Background(), "e1")
	if err == nil {
		t.Fatal("Expected error for update result failure")
	}
}

func TestUpdateResultByID_Success(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{
		ID:     "e1",
		Status: entity.ExecutionRunning,
	}
	resultRepo.results["e1"] = []*entity.ExecutionResult{
		{ID: "r1", ExecutionID: "e1", Status: entity.StatusPending},
	}
	calculator := service.NewScoreCalculator()

	svc := &ExecutionService{resultRepo: resultRepo, calculator: calculator}
	err := svc.UpdateResultByID(context.Background(), "r1", entity.StatusSuccess, "test output", 0, "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestUpdateResultByID_NotFound(t *testing.T) {
	resultRepo := newMockResultRepo()

	svc := &ExecutionService{resultRepo: resultRepo}
	err := svc.UpdateResultByID(context.Background(), "invalid", entity.StatusSuccess, "", 0, "")
	if err == nil {
		t.Fatal("Expected error for not found result")
	}
}

func TestUpdateResultByID_UpdateError(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.results["e1"] = []*entity.ExecutionResult{
		{ID: "r1", ExecutionID: "e1", Status: entity.StatusPending},
	}
	resultRepo.updateResultErr = errors.New("update failed")

	svc := &ExecutionService{resultRepo: resultRepo}
	err := svc.UpdateResultByID(context.Background(), "r1", entity.StatusSuccess, "", 0, "")
	if err == nil {
		t.Fatal("Expected error for update failure")
	}
}

func TestUpdateResultByID_AutoCompletes(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{
		ID:     "e1",
		Status: entity.ExecutionRunning,
	}
	// Only one result that we're about to complete
	resultRepo.results["e1"] = []*entity.ExecutionResult{
		{ID: "r1", ExecutionID: "e1", Status: entity.StatusSuccess}, // Already marked success in mock after update
	}
	calculator := service.NewScoreCalculator()

	svc := &ExecutionService{resultRepo: resultRepo, calculator: calculator}
	err := svc.UpdateResultByID(context.Background(), "r1", entity.StatusSuccess, "done", 0, "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// After update, execution should be completed
	exec := resultRepo.executions["e1"]
	if exec.Status != entity.ExecutionCompleted {
		t.Errorf("Expected execution to be completed, got %v", exec.Status)
	}
}

func TestUpdateResultByID_DoesNotAutoCompleteWithPending(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{
		ID:     "e1",
		Status: entity.ExecutionRunning,
	}
	// Two results, one still pending
	resultRepo.results["e1"] = []*entity.ExecutionResult{
		{ID: "r1", ExecutionID: "e1", Status: entity.StatusSuccess},
		{ID: "r2", ExecutionID: "e1", Status: entity.StatusPending},
	}
	calculator := service.NewScoreCalculator()

	svc := &ExecutionService{resultRepo: resultRepo, calculator: calculator}
	err := svc.UpdateResultByID(context.Background(), "r1", entity.StatusSuccess, "done", 0, "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Execution should still be running
	exec := resultRepo.executions["e1"]
	if exec.Status != entity.ExecutionRunning {
		t.Errorf("Expected execution to still be running, got %v", exec.Status)
	}
}

func TestUpdateResultByID_AgentPawValidation(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.executions["e1"] = &entity.Execution{
		ID:     "e1",
		Status: entity.ExecutionRunning,
	}
	resultRepo.results["e1"] = []*entity.ExecutionResult{
		{ID: "r1", ExecutionID: "e1", AgentPaw: "agent-1", Status: entity.StatusPending},
	}
	calculator := service.NewScoreCalculator()

	svc := &ExecutionService{resultRepo: resultRepo, calculator: calculator}

	// Should fail with wrong agent paw
	err := svc.UpdateResultByID(context.Background(), "r1", entity.StatusSuccess, "output", 0, "wrong-agent")
	if err == nil {
		t.Fatal("Expected error for wrong agent paw")
	}
	if !strings.Contains(err.Error(), "not authorized") {
		t.Errorf("Expected authorization error, got: %v", err)
	}

	// Should succeed with correct agent paw
	err = svc.UpdateResultByID(context.Background(), "r1", entity.StatusSuccess, "output", 0, "agent-1")
	if err != nil {
		t.Fatalf("Expected no error with correct agent paw, got %v", err)
	}
}

func TestCheckAndCompleteExecution_FindResultsError(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.findResultsErr = errors.New("db error")

	svc := &ExecutionService{resultRepo: resultRepo}
	// Should not return error even if find fails
	err := svc.checkAndCompleteExecution(context.Background(), "e1")
	if err != nil {
		t.Fatalf("Expected no error (graceful handling), got %v", err)
	}
}

func TestCheckAndCompleteExecution_EmptyResults(t *testing.T) {
	resultRepo := newMockResultRepo()
	resultRepo.results["e1"] = []*entity.ExecutionResult{}

	svc := &ExecutionService{resultRepo: resultRepo}
	err := svc.checkAndCompleteExecution(context.Background(), "e1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestStartExecutionFindByPawsError(t *testing.T) {
	resultRepo := newMockResultRepo()
	scenarioRepo := newMockScenarioRepo()
	scenarioRepo.scenarios["s1"] = &entity.Scenario{ID: "s1"}
	agentRepo := newMockAgentRepo()
	agentRepo.findErr = errors.New("db connection failed")

	svc := &ExecutionService{
		resultRepo:   resultRepo,
		scenarioRepo: scenarioRepo,
		agentRepo:    agentRepo,
	}

	_, err := svc.StartExecution(context.Background(), "s1", []string{"paw1"}, false)
	if err == nil {
		t.Fatal("Expected error for FindByPaws failure")
	}
	if !errors.Is(err, agentRepo.findErr) && err.Error() != "failed to load agents: db connection failed" {
		t.Errorf("Expected FindByPaws error, got %v", err)
	}
}

func TestDefaultExecutorForPlatform(t *testing.T) {
	tests := []struct {
		platform string
		expected string
	}{
		{"windows", "cmd"},
		{"darwin", "bash"},
		{"linux", "sh"},
		{"freebsd", "sh"},
		{"", "sh"},
	}

	for _, tt := range tests {
		result := defaultExecutorForPlatform(tt.platform)
		if result != tt.expected {
			t.Errorf("defaultExecutorForPlatform(%s) = %s, want %s", tt.platform, result, tt.expected)
		}
	}
}

func TestDetermineExecutor_TechniqueNotFound(t *testing.T) {
	techRepo := newMockTechniqueRepo()
	techRepo.err = errors.New("technique not found")

	svc := &ExecutionService{techniqueRepo: techRepo}

	// Windows agent should get "cmd" when technique lookup fails
	windowsAgent := &entity.Agent{Platform: "windows", Executors: []string{"cmd", "powershell"}}
	result := svc.determineExecutor(context.Background(), "T9999", windowsAgent)
	if result != "cmd" {
		t.Errorf("Expected 'cmd' for Windows agent, got '%s'", result)
	}

	// Linux agent should get "sh" when technique lookup fails
	linuxAgent := &entity.Agent{Platform: "linux", Executors: []string{"sh", "bash"}}
	result = svc.determineExecutor(context.Background(), "T9999", linuxAgent)
	if result != "sh" {
		t.Errorf("Expected 'sh' for Linux agent, got '%s'", result)
	}

	// macOS agent should get "bash" when technique lookup fails
	darwinAgent := &entity.Agent{Platform: "darwin", Executors: []string{"bash", "zsh"}}
	result = svc.determineExecutor(context.Background(), "T9999", darwinAgent)
	if result != "bash" {
		t.Errorf("Expected 'bash' for macOS agent, got '%s'", result)
	}
}

func TestDetermineExecutor_NilAgent(t *testing.T) {
	svc := &ExecutionService{}
	result := svc.determineExecutor(context.Background(), "T1059", nil)
	if result != "sh" {
		t.Errorf("Expected 'sh' for nil agent, got '%s'", result)
	}
}
