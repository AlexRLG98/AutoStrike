package sqlite

import (
	"context"
	"database/sql"
	"time"

	"autostrike/internal/domain/entity"
)

// ResultRepository implements repository.ResultRepository using SQLite
type ResultRepository struct {
	db *sql.DB
}

// NewResultRepository creates a new SQLite result repository
func NewResultRepository(db *sql.DB) *ResultRepository {
	return &ResultRepository{db: db}
}

// CreateExecution creates a new execution
func (r *ResultRepository) CreateExecution(ctx context.Context, execution *entity.Execution) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO executions (id, scenario_id, status, started_at, safe_mode)
		VALUES (?, ?, ?, ?, ?)
	`, execution.ID, execution.ScenarioID, execution.Status, execution.StartedAt, execution.SafeMode)

	return err
}

// UpdateExecution updates an existing execution
func (r *ResultRepository) UpdateExecution(ctx context.Context, execution *entity.Execution) error {
	// Ensure Score is initialized to prevent nil pointer dereference
	score := execution.Score
	if score == nil {
		score = &entity.SecurityScore{}
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE executions SET status = ?, completed_at = ?,
		score_overall = ?, score_blocked = ?, score_detected = ?, score_successful = ?, score_total = ?
		WHERE id = ?
	`, execution.Status, execution.CompletedAt,
		score.Overall, score.Blocked, score.Detected, score.Successful, score.Total,
		execution.ID)

	return err
}

// FindExecutionByID finds an execution by ID
func (r *ResultRepository) FindExecutionByID(ctx context.Context, id string) (*entity.Execution, error) {
	execution := &entity.Execution{
		Score: &entity.SecurityScore{},
	}
	var completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, `
		SELECT id, scenario_id, status, started_at, completed_at, safe_mode,
		score_overall, score_blocked, score_detected, score_successful, score_total
		FROM executions WHERE id = ?
	`, id).Scan(&execution.ID, &execution.ScenarioID, &execution.Status, &execution.StartedAt, &completedAt,
		&execution.SafeMode, &execution.Score.Overall, &execution.Score.Blocked, &execution.Score.Detected,
		&execution.Score.Successful, &execution.Score.Total)

	if err != nil {
		return nil, err
	}

	if completedAt.Valid {
		execution.CompletedAt = &completedAt.Time
	}

	return execution, nil
}

// FindExecutionsByScenario finds executions by scenario ID
func (r *ResultRepository) FindExecutionsByScenario(ctx context.Context, scenarioID string) ([]*entity.Execution, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, scenario_id, status, started_at, completed_at, safe_mode,
		score_overall, score_blocked, score_detected, score_successful, score_total
		FROM executions WHERE scenario_id = ? ORDER BY started_at DESC
	`, scenarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanExecutions(rows)
}

// FindRecentExecutions finds recent executions
func (r *ResultRepository) FindRecentExecutions(ctx context.Context, limit int) ([]*entity.Execution, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, scenario_id, status, started_at, completed_at, safe_mode,
		score_overall, score_blocked, score_detected, score_successful, score_total
		FROM executions ORDER BY started_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanExecutions(rows)
}

// FindExecutionsByDateRange finds all executions within a date range
func (r *ResultRepository) FindExecutionsByDateRange(ctx context.Context, start, end time.Time) ([]*entity.Execution, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, scenario_id, status, started_at, completed_at, safe_mode,
		score_overall, score_blocked, score_detected, score_successful, score_total
		FROM executions
		WHERE started_at >= ? AND started_at <= ?
		ORDER BY started_at DESC
	`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanExecutions(rows)
}

// FindCompletedExecutionsByDateRange finds completed executions within a date range
func (r *ResultRepository) FindCompletedExecutionsByDateRange(ctx context.Context, start, end time.Time) ([]*entity.Execution, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, scenario_id, status, started_at, completed_at, safe_mode,
		score_overall, score_blocked, score_detected, score_successful, score_total
		FROM executions
		WHERE started_at >= ? AND started_at <= ? AND status = 'completed'
		ORDER BY started_at DESC
	`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanExecutions(rows)
}

// CreateResult creates a new execution result
func (r *ResultRepository) CreateResult(ctx context.Context, result *entity.ExecutionResult) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO execution_results (id, execution_id, technique_id, agent_paw, status, started_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, result.ID, result.ExecutionID, result.TechniqueID, result.AgentPaw, result.Status, result.StartedAt)

	return err
}

// UpdateResult updates an existing execution result
func (r *ResultRepository) UpdateResult(ctx context.Context, result *entity.ExecutionResult) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE execution_results SET status = ?, output = ?, exit_code = ?, detected = ?, completed_at = ?
		WHERE id = ?
	`, result.Status, result.Output, result.ExitCode, result.Detected, result.CompletedAt, result.ID)

	return err
}

// FindResultByID finds a result by its ID
func (r *ResultRepository) FindResultByID(ctx context.Context, id string) (*entity.ExecutionResult, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, execution_id, technique_id, agent_paw, status, output, exit_code, detected, started_at, completed_at
		FROM execution_results WHERE id = ?
	`, id)

	result := &entity.ExecutionResult{}
	var output sql.NullString
	var completedAt sql.NullTime

	err := row.Scan(
		&result.ID,
		&result.ExecutionID,
		&result.TechniqueID,
		&result.AgentPaw,
		&result.Status,
		&output,
		&result.ExitCode,
		&result.Detected,
		&result.StartedAt,
		&completedAt,
	)
	if err != nil {
		return nil, err
	}

	if output.Valid {
		result.Output = output.String
	}
	if completedAt.Valid {
		result.CompletedAt = &completedAt.Time
	}

	return result, nil
}

// FindResultsByExecution finds results by execution ID
func (r *ResultRepository) FindResultsByExecution(ctx context.Context, executionID string) ([]*entity.ExecutionResult, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, execution_id, technique_id, agent_paw, status, output, exit_code, detected, started_at, completed_at
		FROM execution_results WHERE execution_id = ? ORDER BY started_at
	`, executionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanResults(rows)
}

// FindResultsByTechnique finds results by technique ID
func (r *ResultRepository) FindResultsByTechnique(ctx context.Context, techniqueID string) ([]*entity.ExecutionResult, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, execution_id, technique_id, agent_paw, status, output, exit_code, detected, started_at, completed_at
		FROM execution_results WHERE technique_id = ? ORDER BY started_at DESC
	`, techniqueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanResults(rows)
}

func (r *ResultRepository) scanExecutions(rows *sql.Rows) ([]*entity.Execution, error) {
	var executions []*entity.Execution

	for rows.Next() {
		execution := &entity.Execution{
			Score: &entity.SecurityScore{},
		}
		var completedAt sql.NullTime

		err := rows.Scan(&execution.ID, &execution.ScenarioID, &execution.Status, &execution.StartedAt, &completedAt,
			&execution.SafeMode, &execution.Score.Overall, &execution.Score.Blocked, &execution.Score.Detected,
			&execution.Score.Successful, &execution.Score.Total)
		if err != nil {
			return nil, err
		}

		if completedAt.Valid {
			execution.CompletedAt = &completedAt.Time
		}

		executions = append(executions, execution)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return executions, nil
}

func (r *ResultRepository) scanResults(rows *sql.Rows) ([]*entity.ExecutionResult, error) {
	var results []*entity.ExecutionResult

	for rows.Next() {
		result := &entity.ExecutionResult{}
		var output sql.NullString
		var completedAt sql.NullTime

		err := rows.Scan(&result.ID, &result.ExecutionID, &result.TechniqueID, &result.AgentPaw,
			&result.Status, &output, &result.ExitCode, &result.Detected, &result.StartedAt, &completedAt)
		if err != nil {
			return nil, err
		}

		if output.Valid {
			result.Output = output.String
		}
		if completedAt.Valid {
			result.CompletedAt = &completedAt.Time
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
