package application

import (
	"context"
	"time"

	"autostrike/internal/domain/entity"
	"autostrike/internal/domain/repository"
)

// AnalyticsService provides analytics and reporting functionality
type AnalyticsService struct {
	resultRepo repository.ResultRepository
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(resultRepo repository.ResultRepository) *AnalyticsService {
	return &AnalyticsService{
		resultRepo: resultRepo,
	}
}

// PeriodStats represents statistics for a time period
type PeriodStats struct {
	Period          string          `json:"period"`
	StartDate       time.Time       `json:"start_date"`
	EndDate         time.Time       `json:"end_date"`
	ExecutionCount  int             `json:"execution_count"`
	AverageScore    float64         `json:"average_score"`
	TotalBlocked    int             `json:"total_blocked"`
	TotalDetected   int             `json:"total_detected"`
	TotalSuccessful int             `json:"total_successful"`
	TotalTechniques int             `json:"total_techniques"`
	ScoreByTactic   map[string]float64 `json:"score_by_tactic,omitempty"`
}

// ScoreComparison represents comparison between two periods
type ScoreComparison struct {
	Current       PeriodStats `json:"current"`
	Previous      PeriodStats `json:"previous"`
	ScoreChange   float64     `json:"score_change"`
	ScoreTrend    string      `json:"score_trend"` // "improving", "declining", "stable"
	BlockedChange int         `json:"blocked_change"`
	DetectedChange int        `json:"detected_change"`
}

// TrendDataPoint represents a single data point in a trend
type TrendDataPoint struct {
	Date           string  `json:"date"`
	AverageScore   float64 `json:"average_score"`
	ExecutionCount int     `json:"execution_count"`
	Blocked        int     `json:"blocked"`
	Detected       int     `json:"detected"`
	Successful     int     `json:"successful"`
}

// ScoreTrend represents score trend over time
type ScoreTrend struct {
	Period     string           `json:"period"` // "7d", "30d", "90d"
	DataPoints []TrendDataPoint `json:"data_points"`
	Summary    TrendSummary     `json:"summary"`
}

// TrendSummary provides summary statistics for a trend
type TrendSummary struct {
	StartScore       float64 `json:"start_score"`
	EndScore         float64 `json:"end_score"`
	AverageScore     float64 `json:"average_score"`
	MaxScore         float64 `json:"max_score"`
	MinScore         float64 `json:"min_score"`
	TotalExecutions  int     `json:"total_executions"`
	OverallTrend     string  `json:"overall_trend"` // "improving", "declining", "stable"
	PercentageChange float64 `json:"percentage_change"`
}

// ExecutionSummary provides overall execution analytics
type ExecutionSummary struct {
	TotalExecutions     int                `json:"total_executions"`
	CompletedExecutions int                `json:"completed_executions"`
	AverageScore        float64            `json:"average_score"`
	BestScore           float64            `json:"best_score"`
	WorstScore          float64            `json:"worst_score"`
	ScoresByScenario    map[string]float64 `json:"scores_by_scenario"`
	ExecutionsByStatus  map[string]int     `json:"executions_by_status"`
}

// GetPeriodStats calculates statistics for a given time period
func (s *AnalyticsService) GetPeriodStats(ctx context.Context, start, end time.Time, periodLabel string) (*PeriodStats, error) {
	executions, err := s.resultRepo.FindCompletedExecutionsByDateRange(ctx, start, end)
	if err != nil {
		return nil, err
	}

	stats := &PeriodStats{
		Period:        periodLabel,
		StartDate:     start,
		EndDate:       end,
		ScoreByTactic: make(map[string]float64),
	}

	if len(executions) == 0 {
		return stats, nil
	}

	stats.ExecutionCount = len(executions)

	var totalScore float64
	var scoredCount int
	tacticScores := make(map[string][]float64)

	for _, exec := range executions {
		if exec.Score != nil {
			totalScore += exec.Score.Overall
			scoredCount++
			stats.TotalBlocked += exec.Score.Blocked
			stats.TotalDetected += exec.Score.Detected
			stats.TotalSuccessful += exec.Score.Successful
			stats.TotalTechniques += exec.Score.Total

			// Aggregate tactic scores
			for tactic, score := range exec.Score.ByTactic {
				tacticScores[tactic] = append(tacticScores[tactic], score)
			}
		}
	}

	if scoredCount > 0 {
		stats.AverageScore = totalScore / float64(scoredCount)
	}

	// Calculate average per tactic
	for tactic, scores := range tacticScores {
		var sum float64
		for _, score := range scores {
			sum += score
		}
		stats.ScoreByTactic[tactic] = sum / float64(len(scores))
	}

	return stats, nil
}

// CompareScores compares scores between two consecutive periods
func (s *AnalyticsService) CompareScores(ctx context.Context, periodDays int) (*ScoreComparison, error) {
	now := time.Now()

	// Current period
	currentEnd := now
	currentStart := now.AddDate(0, 0, -periodDays)

	// Previous period
	previousEnd := currentStart
	previousStart := previousEnd.AddDate(0, 0, -periodDays)

	currentStats, err := s.GetPeriodStats(ctx, currentStart, currentEnd, "current")
	if err != nil {
		return nil, err
	}

	previousStats, err := s.GetPeriodStats(ctx, previousStart, previousEnd, "previous")
	if err != nil {
		return nil, err
	}

	comparison := &ScoreComparison{
		Current:        *currentStats,
		Previous:       *previousStats,
		ScoreChange:   currentStats.AverageScore - previousStats.AverageScore,
		BlockedChange:  currentStats.TotalBlocked - previousStats.TotalBlocked,
		DetectedChange: currentStats.TotalDetected - previousStats.TotalDetected,
	}

	// Determine trend
	if comparison.ScoreChange > 5 {
		comparison.ScoreTrend = "improving"
	} else if comparison.ScoreChange < -5 {
		comparison.ScoreTrend = "declining"
	} else {
		comparison.ScoreTrend = "stable"
	}

	return comparison, nil
}

// scoreTracker tracks min/max scores during iteration
type scoreTracker struct {
	maxScore float64
	minScore float64
	firstSet bool
}

// updateMinMax updates the tracker with a new score
func (st *scoreTracker) updateMinMax(score float64) {
	if st.firstSet {
		st.maxScore = score
		st.minScore = score
		st.firstSet = false
		return
	}
	if score > st.maxScore {
		st.maxScore = score
	}
	if score < st.minScore {
		st.minScore = score
	}
}

// processDayExecutions processes executions for a single day and returns data point
func processDayExecutions(dayExecs []*entity.Execution, dateKey string, tracker *scoreTracker) (TrendDataPoint, float64) {
	point := TrendDataPoint{
		Date:           dateKey,
		ExecutionCount: len(dayExecs),
	}

	if len(dayExecs) == 0 {
		return point, 0
	}

	var dayTotal float64
	var scoredCount int
	for _, exec := range dayExecs {
		if exec.Score == nil {
			continue
		}
		scoredCount++
		dayTotal += exec.Score.Overall
		point.Blocked += exec.Score.Blocked
		point.Detected += exec.Score.Detected
		point.Successful += exec.Score.Successful
		tracker.updateMinMax(exec.Score.Overall)
	}
	// Calculate average only from executions that have scores
	if scoredCount > 0 {
		point.AverageScore = dayTotal / float64(scoredCount)
	}
	return point, point.AverageScore
}

// calculateTrendSummary computes the summary from collected scores
func calculateTrendSummary(allScores []float64, totalExecutions int, tracker *scoreTracker) TrendSummary {
	summary := TrendSummary{
		TotalExecutions: totalExecutions,
		MaxScore:        tracker.maxScore,
		MinScore:        tracker.minScore,
	}

	if len(allScores) == 0 {
		return summary
	}

	var totalScore float64
	for _, score := range allScores {
		totalScore += score
	}

	summary.AverageScore = totalScore / float64(len(allScores))
	summary.StartScore = allScores[0]
	summary.EndScore = allScores[len(allScores)-1]

	// Calculate percentage change, handling zero division edge cases
	if summary.StartScore > 0 {
		summary.PercentageChange = ((summary.EndScore - summary.StartScore) / summary.StartScore) * 100
	} else if summary.EndScore > 0 {
		// StartScore is 0 but EndScore is positive: represents full improvement from baseline
		summary.PercentageChange = 100
	}
	// If both are 0, PercentageChange remains 0 (no change)

	summary.OverallTrend = determineTrend(summary.PercentageChange)
	return summary
}

// determineTrend returns trend label based on percentage change
func determineTrend(percentageChange float64) string {
	if percentageChange > 5 {
		return "improving"
	}
	if percentageChange < -5 {
		return "declining"
	}
	return "stable"
}

// periodLabel returns the label for a given number of days
func periodLabel(days int) string {
	switch days {
	case 30:
		return "30d"
	case 90:
		return "90d"
	default:
		return "7d"
	}
}

// GetScoreTrend gets score trend over time
func (s *AnalyticsService) GetScoreTrend(ctx context.Context, days int) (*ScoreTrend, error) {
	now := time.Now()
	start := now.AddDate(0, 0, -days)

	executions, err := s.resultRepo.FindCompletedExecutionsByDateRange(ctx, start, now)
	if err != nil {
		return nil, err
	}

	// Group executions by date
	dateGroups := make(map[string][]*entity.Execution)
	for _, exec := range executions {
		dateKey := exec.StartedAt.Format("2006-01-02")
		dateGroups[dateKey] = append(dateGroups[dateKey], exec)
	}

	// Generate data points for each day
	dataPoints := make([]TrendDataPoint, 0)
	var allScores []float64
	tracker := &scoreTracker{firstSet: true}

	for d := start; !d.After(now); d = d.AddDate(0, 0, 1) {
		dateKey := d.Format("2006-01-02")
		point, avgScore := processDayExecutions(dateGroups[dateKey], dateKey, tracker)
		dataPoints = append(dataPoints, point)
		if point.ExecutionCount > 0 {
			allScores = append(allScores, avgScore)
		}
	}

	return &ScoreTrend{
		Period:     periodLabel(days),
		DataPoints: dataPoints,
		Summary:    calculateTrendSummary(allScores, len(executions), tracker),
	}, nil
}

// executionSummaryBuilder builds execution summary incrementally
type executionSummaryBuilder struct {
	summary        *ExecutionSummary
	scenarioScores map[string][]float64
	completedScores []float64
	scoreTracker   *scoreTracker
}

// newExecutionSummaryBuilder creates a new builder
func newExecutionSummaryBuilder(totalExecutions int) *executionSummaryBuilder {
	return &executionSummaryBuilder{
		summary: &ExecutionSummary{
			TotalExecutions:    totalExecutions,
			ScoresByScenario:   make(map[string]float64),
			ExecutionsByStatus: make(map[string]int),
		},
		scenarioScores:  make(map[string][]float64),
		completedScores: []float64{},
		scoreTracker:    &scoreTracker{firstSet: true},
	}
}

// processExecution processes a single execution
func (b *executionSummaryBuilder) processExecution(exec *entity.Execution) {
	b.summary.ExecutionsByStatus[string(exec.Status)]++

	if exec.Status != entity.ExecutionCompleted || exec.Score == nil {
		return
	}

	b.summary.CompletedExecutions++
	score := exec.Score.Overall
	b.completedScores = append(b.completedScores, score)
	b.scenarioScores[exec.ScenarioID] = append(b.scenarioScores[exec.ScenarioID], score)
	b.scoreTracker.updateMinMax(score)
}

// finalize calculates final averages and returns the summary
func (b *executionSummaryBuilder) finalize() *ExecutionSummary {
	b.summary.BestScore = b.scoreTracker.maxScore
	b.summary.WorstScore = b.scoreTracker.minScore

	if len(b.completedScores) > 0 {
		b.summary.AverageScore = averageScore(b.completedScores)
	}

	for scenarioID, scores := range b.scenarioScores {
		b.summary.ScoresByScenario[scenarioID] = averageScore(scores)
	}

	return b.summary
}

// averageScore calculates the average of a slice of scores
func averageScore(scores []float64) float64 {
	var total float64
	for _, score := range scores {
		total += score
	}
	return total / float64(len(scores))
}

// GetExecutionSummary provides overall execution analytics
func (s *AnalyticsService) GetExecutionSummary(ctx context.Context, days int) (*ExecutionSummary, error) {
	now := time.Now()
	start := now.AddDate(0, 0, -days)

	executions, err := s.resultRepo.FindExecutionsByDateRange(ctx, start, now)
	if err != nil {
		return nil, err
	}

	builder := newExecutionSummaryBuilder(len(executions))
	for _, exec := range executions {
		builder.processExecution(exec)
	}

	return builder.finalize(), nil
}
