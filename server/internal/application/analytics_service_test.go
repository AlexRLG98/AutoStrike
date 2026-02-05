package application

import (
	"context"
	"testing"
	"time"

	"autostrike/internal/domain/entity"
)

// mockResultRepoForAnalytics implements repository.ResultRepository for analytics tests
type mockResultRepoForAnalytics struct {
	executions []*entity.Execution
}

func (m *mockResultRepoForAnalytics) CreateExecution(ctx context.Context, execution *entity.Execution) error {
	m.executions = append(m.executions, execution)
	return nil
}

func (m *mockResultRepoForAnalytics) UpdateExecution(ctx context.Context, execution *entity.Execution) error {
	for i, e := range m.executions {
		if e.ID == execution.ID {
			m.executions[i] = execution
			return nil
		}
	}
	return nil
}

func (m *mockResultRepoForAnalytics) FindExecutionByID(ctx context.Context, id string) (*entity.Execution, error) {
	for _, e := range m.executions {
		if e.ID == id {
			return e, nil
		}
	}
	return nil, nil
}

func (m *mockResultRepoForAnalytics) FindExecutionsByScenario(ctx context.Context, scenarioID string) ([]*entity.Execution, error) {
	var results []*entity.Execution
	for _, e := range m.executions {
		if e.ScenarioID == scenarioID {
			results = append(results, e)
		}
	}
	return results, nil
}

func (m *mockResultRepoForAnalytics) FindRecentExecutions(ctx context.Context, limit int) ([]*entity.Execution, error) {
	if len(m.executions) <= limit {
		return m.executions, nil
	}
	return m.executions[:limit], nil
}

func (m *mockResultRepoForAnalytics) FindExecutionsByDateRange(ctx context.Context, start, end time.Time) ([]*entity.Execution, error) {
	var results []*entity.Execution
	for _, e := range m.executions {
		if !e.StartedAt.Before(start) && !e.StartedAt.After(end) {
			results = append(results, e)
		}
	}
	return results, nil
}

func (m *mockResultRepoForAnalytics) FindCompletedExecutionsByDateRange(ctx context.Context, start, end time.Time) ([]*entity.Execution, error) {
	var results []*entity.Execution
	for _, e := range m.executions {
		if e.Status == entity.ExecutionCompleted && !e.StartedAt.Before(start) && !e.StartedAt.After(end) {
			results = append(results, e)
		}
	}
	return results, nil
}

func (m *mockResultRepoForAnalytics) CreateResult(ctx context.Context, result *entity.ExecutionResult) error {
	return nil
}

func (m *mockResultRepoForAnalytics) UpdateResult(ctx context.Context, result *entity.ExecutionResult) error {
	return nil
}

func (m *mockResultRepoForAnalytics) FindResultByID(ctx context.Context, id string) (*entity.ExecutionResult, error) {
	return nil, nil
}

func (m *mockResultRepoForAnalytics) FindResultsByExecution(ctx context.Context, executionID string) ([]*entity.ExecutionResult, error) {
	return nil, nil
}

func (m *mockResultRepoForAnalytics) FindResultsByTechnique(ctx context.Context, techniqueID string) ([]*entity.ExecutionResult, error) {
	return nil, nil
}

func createTestExecution(id, scenarioID string, score float64, blocked, detected, successful, total int, startedAt time.Time, status entity.ExecutionStatus) *entity.Execution {
	completedAt := startedAt.Add(10 * time.Minute)
	return &entity.Execution{
		ID:          id,
		ScenarioID:  scenarioID,
		Status:      status,
		StartedAt:   startedAt,
		CompletedAt: &completedAt,
		Score: &entity.SecurityScore{
			Overall:    score,
			Blocked:    blocked,
			Detected:   detected,
			Successful: successful,
			Total:      total,
		},
	}
}

func TestNewAnalyticsService(t *testing.T) {
	repo := &mockResultRepoForAnalytics{}
	service := NewAnalyticsService(repo)

	if service == nil {
		t.Fatal("NewAnalyticsService returned nil")
	}
}

func TestAnalyticsService_GetPeriodStats_NoExecutions(t *testing.T) {
	repo := &mockResultRepoForAnalytics{}
	service := NewAnalyticsService(repo)

	ctx := context.Background()
	now := time.Now()
	start := now.AddDate(0, 0, -7)

	stats, err := service.GetPeriodStats(ctx, start, now, "test")
	if err != nil {
		t.Fatalf("GetPeriodStats failed: %v", err)
	}

	if stats.Period != "test" {
		t.Errorf("Period = %q, want 'test'", stats.Period)
	}
	if stats.ExecutionCount != 0 {
		t.Errorf("ExecutionCount = %d, want 0", stats.ExecutionCount)
	}
	if stats.AverageScore != 0 {
		t.Errorf("AverageScore = %f, want 0", stats.AverageScore)
	}
}

func TestAnalyticsService_GetPeriodStats_WithExecutions(t *testing.T) {
	now := time.Now()
	repo := &mockResultRepoForAnalytics{
		executions: []*entity.Execution{
			createTestExecution("exec-1", "scenario-1", 80.0, 4, 2, 4, 10, now.AddDate(0, 0, -2), entity.ExecutionCompleted),
			createTestExecution("exec-2", "scenario-1", 60.0, 3, 3, 4, 10, now.AddDate(0, 0, -1), entity.ExecutionCompleted),
		},
	}
	service := NewAnalyticsService(repo)

	ctx := context.Background()
	start := now.AddDate(0, 0, -7)

	stats, err := service.GetPeriodStats(ctx, start, now, "weekly")
	if err != nil {
		t.Fatalf("GetPeriodStats failed: %v", err)
	}

	if stats.ExecutionCount != 2 {
		t.Errorf("ExecutionCount = %d, want 2", stats.ExecutionCount)
	}
	if stats.AverageScore != 70.0 {
		t.Errorf("AverageScore = %f, want 70.0", stats.AverageScore)
	}
	if stats.TotalBlocked != 7 {
		t.Errorf("TotalBlocked = %d, want 7", stats.TotalBlocked)
	}
	if stats.TotalDetected != 5 {
		t.Errorf("TotalDetected = %d, want 5", stats.TotalDetected)
	}
}

func TestAnalyticsService_CompareScores(t *testing.T) {
	now := time.Now()
	repo := &mockResultRepoForAnalytics{
		executions: []*entity.Execution{
			// Current period (last 7 days)
			createTestExecution("exec-1", "scenario-1", 80.0, 4, 2, 4, 10, now.AddDate(0, 0, -2), entity.ExecutionCompleted),
			// Previous period (8-14 days ago)
			createTestExecution("exec-2", "scenario-1", 60.0, 3, 3, 4, 10, now.AddDate(0, 0, -10), entity.ExecutionCompleted),
		},
	}
	service := NewAnalyticsService(repo)

	ctx := context.Background()
	comparison, err := service.CompareScores(ctx, 7)
	if err != nil {
		t.Fatalf("CompareScores failed: %v", err)
	}

	if comparison.Current.ExecutionCount != 1 {
		t.Errorf("Current.ExecutionCount = %d, want 1", comparison.Current.ExecutionCount)
	}
	if comparison.Previous.ExecutionCount != 1 {
		t.Errorf("Previous.ExecutionCount = %d, want 1", comparison.Previous.ExecutionCount)
	}
	if comparison.ScoreChange != 20.0 {
		t.Errorf("ScoreChange = %f, want 20.0", comparison.ScoreChange)
	}
	if comparison.ScoreTrend != "improving" {
		t.Errorf("ScoreTrend = %q, want 'improving'", comparison.ScoreTrend)
	}
}

func TestAnalyticsService_CompareScores_Declining(t *testing.T) {
	now := time.Now()
	repo := &mockResultRepoForAnalytics{
		executions: []*entity.Execution{
			// Current period - lower score
			createTestExecution("exec-1", "scenario-1", 40.0, 2, 2, 6, 10, now.AddDate(0, 0, -2), entity.ExecutionCompleted),
			// Previous period - higher score
			createTestExecution("exec-2", "scenario-1", 80.0, 4, 4, 2, 10, now.AddDate(0, 0, -10), entity.ExecutionCompleted),
		},
	}
	service := NewAnalyticsService(repo)

	ctx := context.Background()
	comparison, err := service.CompareScores(ctx, 7)
	if err != nil {
		t.Fatalf("CompareScores failed: %v", err)
	}

	if comparison.ScoreTrend != "declining" {
		t.Errorf("ScoreTrend = %q, want 'declining'", comparison.ScoreTrend)
	}
}

func TestAnalyticsService_CompareScores_Stable(t *testing.T) {
	now := time.Now()
	repo := &mockResultRepoForAnalytics{
		executions: []*entity.Execution{
			// Current period
			createTestExecution("exec-1", "scenario-1", 72.0, 4, 2, 4, 10, now.AddDate(0, 0, -2), entity.ExecutionCompleted),
			// Previous period - similar score
			createTestExecution("exec-2", "scenario-1", 70.0, 4, 2, 4, 10, now.AddDate(0, 0, -10), entity.ExecutionCompleted),
		},
	}
	service := NewAnalyticsService(repo)

	ctx := context.Background()
	comparison, err := service.CompareScores(ctx, 7)
	if err != nil {
		t.Fatalf("CompareScores failed: %v", err)
	}

	if comparison.ScoreTrend != "stable" {
		t.Errorf("ScoreTrend = %q, want 'stable'", comparison.ScoreTrend)
	}
}

func TestAnalyticsService_GetScoreTrend(t *testing.T) {
	now := time.Now()
	repo := &mockResultRepoForAnalytics{
		executions: []*entity.Execution{
			createTestExecution("exec-1", "scenario-1", 80.0, 4, 2, 4, 10, now.AddDate(0, 0, -5), entity.ExecutionCompleted),
			createTestExecution("exec-2", "scenario-1", 85.0, 5, 2, 3, 10, now.AddDate(0, 0, -3), entity.ExecutionCompleted),
			createTestExecution("exec-3", "scenario-1", 90.0, 6, 2, 2, 10, now.AddDate(0, 0, -1), entity.ExecutionCompleted),
		},
	}
	service := NewAnalyticsService(repo)

	ctx := context.Background()
	trend, err := service.GetScoreTrend(ctx, 7)
	if err != nil {
		t.Fatalf("GetScoreTrend failed: %v", err)
	}

	if trend.Period != "7d" {
		t.Errorf("Period = %q, want '7d'", trend.Period)
	}
	if len(trend.DataPoints) != 8 { // 7 days + today
		t.Errorf("DataPoints count = %d, want 8", len(trend.DataPoints))
	}
	if trend.Summary.TotalExecutions != 3 {
		t.Errorf("Summary.TotalExecutions = %d, want 3", trend.Summary.TotalExecutions)
	}
	if trend.Summary.MaxScore != 90.0 {
		t.Errorf("Summary.MaxScore = %f, want 90.0", trend.Summary.MaxScore)
	}
	if trend.Summary.MinScore != 80.0 {
		t.Errorf("Summary.MinScore = %f, want 80.0", trend.Summary.MinScore)
	}
}

func TestAnalyticsService_GetScoreTrend_30Days(t *testing.T) {
	repo := &mockResultRepoForAnalytics{}
	service := NewAnalyticsService(repo)

	ctx := context.Background()
	trend, err := service.GetScoreTrend(ctx, 30)
	if err != nil {
		t.Fatalf("GetScoreTrend failed: %v", err)
	}

	if trend.Period != "30d" {
		t.Errorf("Period = %q, want '30d'", trend.Period)
	}
}

func TestAnalyticsService_GetScoreTrend_90Days(t *testing.T) {
	repo := &mockResultRepoForAnalytics{}
	service := NewAnalyticsService(repo)

	ctx := context.Background()
	trend, err := service.GetScoreTrend(ctx, 90)
	if err != nil {
		t.Fatalf("GetScoreTrend failed: %v", err)
	}

	if trend.Period != "90d" {
		t.Errorf("Period = %q, want '90d'", trend.Period)
	}
}

func TestAnalyticsService_GetExecutionSummary(t *testing.T) {
	now := time.Now()
	repo := &mockResultRepoForAnalytics{
		executions: []*entity.Execution{
			createTestExecution("exec-1", "scenario-1", 80.0, 4, 2, 4, 10, now.AddDate(0, 0, -5), entity.ExecutionCompleted),
			createTestExecution("exec-2", "scenario-1", 60.0, 3, 3, 4, 10, now.AddDate(0, 0, -3), entity.ExecutionCompleted),
			createTestExecution("exec-3", "scenario-2", 90.0, 6, 2, 2, 10, now.AddDate(0, 0, -1), entity.ExecutionCompleted),
			{
				ID:         "exec-4",
				ScenarioID: "scenario-1",
				Status:     entity.ExecutionFailed,
				StartedAt:  now.AddDate(0, 0, -2),
			},
		},
	}
	service := NewAnalyticsService(repo)

	ctx := context.Background()
	summary, err := service.GetExecutionSummary(ctx, 30)
	if err != nil {
		t.Fatalf("GetExecutionSummary failed: %v", err)
	}

	if summary.TotalExecutions != 4 {
		t.Errorf("TotalExecutions = %d, want 4", summary.TotalExecutions)
	}
	if summary.CompletedExecutions != 3 {
		t.Errorf("CompletedExecutions = %d, want 3", summary.CompletedExecutions)
	}
	if summary.BestScore != 90.0 {
		t.Errorf("BestScore = %f, want 90.0", summary.BestScore)
	}
	if summary.WorstScore != 60.0 {
		t.Errorf("WorstScore = %f, want 60.0", summary.WorstScore)
	}
	// (80 + 60 + 90) / 3 = 76.67
	expectedAvg := float64(80+60+90) / 3
	if summary.AverageScore != expectedAvg {
		t.Errorf("AverageScore = %f, want %f", summary.AverageScore, expectedAvg)
	}
	if summary.ExecutionsByStatus["completed"] != 3 {
		t.Errorf("ExecutionsByStatus[completed] = %d, want 3", summary.ExecutionsByStatus["completed"])
	}
	if summary.ExecutionsByStatus["failed"] != 1 {
		t.Errorf("ExecutionsByStatus[failed] = %d, want 1", summary.ExecutionsByStatus["failed"])
	}
}

func TestAnalyticsService_GetExecutionSummary_NoExecutions(t *testing.T) {
	repo := &mockResultRepoForAnalytics{}
	service := NewAnalyticsService(repo)

	ctx := context.Background()
	summary, err := service.GetExecutionSummary(ctx, 30)
	if err != nil {
		t.Fatalf("GetExecutionSummary failed: %v", err)
	}

	if summary.TotalExecutions != 0 {
		t.Errorf("TotalExecutions = %d, want 0", summary.TotalExecutions)
	}
	if summary.AverageScore != 0 {
		t.Errorf("AverageScore = %f, want 0", summary.AverageScore)
	}
}

func TestAnalyticsService_GetPeriodStats_TacticScores(t *testing.T) {
	now := time.Now()
	completedAt := now.Add(10 * time.Minute)
	repo := &mockResultRepoForAnalytics{
		executions: []*entity.Execution{
			{
				ID:          "exec-1",
				ScenarioID:  "scenario-1",
				Status:      entity.ExecutionCompleted,
				StartedAt:   now.AddDate(0, 0, -2),
				CompletedAt: &completedAt,
				Score: &entity.SecurityScore{
					Overall:    80.0,
					Blocked:    4,
					Detected:   2,
					Successful: 4,
					Total:      10,
					ByTactic: map[string]float64{
						"discovery": 90.0,
						"execution": 70.0,
					},
				},
			},
			{
				ID:          "exec-2",
				ScenarioID:  "scenario-1",
				Status:      entity.ExecutionCompleted,
				StartedAt:   now.AddDate(0, 0, -1),
				CompletedAt: &completedAt,
				Score: &entity.SecurityScore{
					Overall:    60.0,
					Blocked:    3,
					Detected:   3,
					Successful: 4,
					Total:      10,
					ByTactic: map[string]float64{
						"discovery": 70.0,
						"execution": 50.0,
					},
				},
			},
		},
	}
	service := NewAnalyticsService(repo)

	ctx := context.Background()
	start := now.AddDate(0, 0, -7)

	stats, err := service.GetPeriodStats(ctx, start, now, "weekly")
	if err != nil {
		t.Fatalf("GetPeriodStats failed: %v", err)
	}

	if stats.ScoreByTactic["discovery"] != 80.0 {
		t.Errorf("ScoreByTactic[discovery] = %f, want 80.0", stats.ScoreByTactic["discovery"])
	}
	if stats.ScoreByTactic["execution"] != 60.0 {
		t.Errorf("ScoreByTactic[execution] = %f, want 60.0", stats.ScoreByTactic["execution"])
	}
}

func TestDetermineTrend(t *testing.T) {
	tests := []struct {
		name             string
		percentageChange float64
		expected         string
	}{
		{"improving high", 10.0, "improving"},
		{"improving edge", 5.1, "improving"},
		{"stable positive", 4.9, "stable"},
		{"stable zero", 0.0, "stable"},
		{"stable negative", -4.9, "stable"},
		{"declining edge", -5.1, "declining"},
		{"declining low", -10.0, "declining"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineTrend(tt.percentageChange)
			if result != tt.expected {
				t.Errorf("determineTrend(%f) = %q, want %q", tt.percentageChange, result, tt.expected)
			}
		})
	}
}

func TestPeriodLabel(t *testing.T) {
	tests := []struct {
		days     int
		expected string
	}{
		{7, "7d"},
		{30, "30d"},
		{90, "90d"},
		{14, "7d"}, // default
		{0, "7d"},  // default
	}

	for _, tt := range tests {
		result := periodLabel(tt.days)
		if result != tt.expected {
			t.Errorf("periodLabel(%d) = %q, want %q", tt.days, result, tt.expected)
		}
	}
}
