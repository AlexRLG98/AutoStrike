package sqlite

import (
	"context"
	"database/sql"
	"time"

	"autostrike/internal/domain/entity"
)

// ScheduleRepository implements repository.ScheduleRepository using SQLite
type ScheduleRepository struct {
	db *sql.DB
}

// NewScheduleRepository creates a new schedule repository
func NewScheduleRepository(db *sql.DB) *ScheduleRepository {
	return &ScheduleRepository{db: db}
}

// Create inserts a new schedule into the database
func (r *ScheduleRepository) Create(ctx context.Context, schedule *entity.Schedule) error {
	query := `
		INSERT INTO schedules (id, name, description, scenario_id, agent_paw, frequency, cron_expr, safe_mode, status, next_run_at, last_run_at, last_run_id, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		schedule.ID,
		schedule.Name,
		schedule.Description,
		schedule.ScenarioID,
		schedule.AgentPaw,
		schedule.Frequency,
		schedule.CronExpr,
		schedule.SafeMode,
		schedule.Status,
		schedule.NextRunAt,
		schedule.LastRunAt,
		schedule.LastRunID,
		schedule.CreatedBy,
		schedule.CreatedAt,
		schedule.UpdatedAt,
	)
	return err
}

// Update updates an existing schedule
func (r *ScheduleRepository) Update(ctx context.Context, schedule *entity.Schedule) error {
	query := `
		UPDATE schedules
		SET name = ?, description = ?, scenario_id = ?, agent_paw = ?, frequency = ?, cron_expr = ?, safe_mode = ?, status = ?, next_run_at = ?, last_run_at = ?, last_run_id = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		schedule.Name,
		schedule.Description,
		schedule.ScenarioID,
		schedule.AgentPaw,
		schedule.Frequency,
		schedule.CronExpr,
		schedule.SafeMode,
		schedule.Status,
		schedule.NextRunAt,
		schedule.LastRunAt,
		schedule.LastRunID,
		schedule.UpdatedAt,
		schedule.ID,
	)
	return err
}

// Delete removes a schedule from the database using a transaction
func (r *ScheduleRepository) Delete(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Delete associated runs first
	_, err = tx.ExecContext(ctx, "DELETE FROM schedule_runs WHERE schedule_id = ?", id)
	if err != nil {
		return err
	}

	// Delete the schedule
	_, err = tx.ExecContext(ctx, "DELETE FROM schedules WHERE id = ?", id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// FindByID retrieves a schedule by ID
func (r *ScheduleRepository) FindByID(ctx context.Context, id string) (*entity.Schedule, error) {
	query := `
		SELECT id, name, description, scenario_id, agent_paw, frequency, cron_expr, safe_mode, status, next_run_at, last_run_at, last_run_id, created_by, created_at, updated_at
		FROM schedules WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanSchedule(row)
}

// FindAll retrieves all schedules
func (r *ScheduleRepository) FindAll(ctx context.Context) ([]*entity.Schedule, error) {
	query := `
		SELECT id, name, description, scenario_id, agent_paw, frequency, cron_expr, safe_mode, status, next_run_at, last_run_at, last_run_id, created_by, created_at, updated_at
		FROM schedules ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSchedules(rows)
}

// FindByStatus retrieves schedules by status
func (r *ScheduleRepository) FindByStatus(ctx context.Context, status entity.ScheduleStatus) ([]*entity.Schedule, error) {
	query := `
		SELECT id, name, description, scenario_id, agent_paw, frequency, cron_expr, safe_mode, status, next_run_at, last_run_at, last_run_id, created_by, created_at, updated_at
		FROM schedules WHERE status = ? ORDER BY next_run_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSchedules(rows)
}

// FindActiveSchedulesDue retrieves active schedules that are due to run
func (r *ScheduleRepository) FindActiveSchedulesDue(ctx context.Context, now time.Time) ([]*entity.Schedule, error) {
	query := `
		SELECT id, name, description, scenario_id, agent_paw, frequency, cron_expr, safe_mode, status, next_run_at, last_run_at, last_run_id, created_by, created_at, updated_at
		FROM schedules
		WHERE status = 'active' AND next_run_at IS NOT NULL AND next_run_at <= ?
		ORDER BY next_run_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSchedules(rows)
}

// FindByScenarioID retrieves schedules for a specific scenario
func (r *ScheduleRepository) FindByScenarioID(ctx context.Context, scenarioID string) ([]*entity.Schedule, error) {
	query := `
		SELECT id, name, description, scenario_id, agent_paw, frequency, cron_expr, safe_mode, status, next_run_at, last_run_at, last_run_id, created_by, created_at, updated_at
		FROM schedules WHERE scenario_id = ? ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, scenarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanSchedules(rows)
}

// CreateRun inserts a new schedule run
func (r *ScheduleRepository) CreateRun(ctx context.Context, run *entity.ScheduleRun) error {
	query := `
		INSERT INTO schedule_runs (id, schedule_id, execution_id, started_at, completed_at, status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	// Handle nullable execution_id
	var executionID sql.NullString
	if run.ExecutionID != "" {
		executionID = sql.NullString{String: run.ExecutionID, Valid: true}
	}
	_, err := r.db.ExecContext(ctx, query,
		run.ID,
		run.ScheduleID,
		executionID,
		run.StartedAt,
		run.CompletedAt,
		run.Status,
		run.Error,
	)
	return err
}

// UpdateRun updates a schedule run
func (r *ScheduleRepository) UpdateRun(ctx context.Context, run *entity.ScheduleRun) error {
	query := `
		UPDATE schedule_runs SET completed_at = ?, status = ?, error = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		run.CompletedAt,
		run.Status,
		run.Error,
		run.ID,
	)
	return err
}

// FindRunsByScheduleID retrieves runs for a schedule
func (r *ScheduleRepository) FindRunsByScheduleID(ctx context.Context, scheduleID string, limit int) ([]*entity.ScheduleRun, error) {
	query := `
		SELECT id, schedule_id, execution_id, started_at, completed_at, status, error
		FROM schedule_runs WHERE schedule_id = ?
		ORDER BY started_at DESC LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, scheduleID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*entity.ScheduleRun
	for rows.Next() {
		run := &entity.ScheduleRun{}
		var completedAt sql.NullTime
		var errStr sql.NullString
		var executionID sql.NullString
		err := rows.Scan(
			&run.ID,
			&run.ScheduleID,
			&executionID,
			&run.StartedAt,
			&completedAt,
			&run.Status,
			&errStr,
		)
		if err != nil {
			return nil, err
		}
		if executionID.Valid {
			run.ExecutionID = executionID.String
		}
		if completedAt.Valid {
			run.CompletedAt = &completedAt.Time
		}
		if errStr.Valid {
			run.Error = errStr.String
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return runs, nil
}

// scanSchedule scans a single schedule from a row
func (r *ScheduleRepository) scanSchedule(row *sql.Row) (*entity.Schedule, error) {
	schedule := &entity.Schedule{}
	var nextRunAt, lastRunAt sql.NullTime
	var agentPaw, description, cronExpr, lastRunID sql.NullString

	err := row.Scan(
		&schedule.ID,
		&schedule.Name,
		&description,
		&schedule.ScenarioID,
		&agentPaw,
		&schedule.Frequency,
		&cronExpr,
		&schedule.SafeMode,
		&schedule.Status,
		&nextRunAt,
		&lastRunAt,
		&lastRunID,
		&schedule.CreatedBy,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		schedule.Description = description.String
	}
	if agentPaw.Valid {
		schedule.AgentPaw = agentPaw.String
	}
	if cronExpr.Valid {
		schedule.CronExpr = cronExpr.String
	}
	if nextRunAt.Valid {
		schedule.NextRunAt = &nextRunAt.Time
	}
	if lastRunAt.Valid {
		schedule.LastRunAt = &lastRunAt.Time
	}
	if lastRunID.Valid {
		schedule.LastRunID = lastRunID.String
	}

	return schedule, nil
}

// applyNullableFields applies nullable field values to a schedule
func applyNullableFields(schedule *entity.Schedule, description, agentPaw, cronExpr, lastRunID sql.NullString, nextRunAt, lastRunAt sql.NullTime) {
	if description.Valid {
		schedule.Description = description.String
	}
	if agentPaw.Valid {
		schedule.AgentPaw = agentPaw.String
	}
	if cronExpr.Valid {
		schedule.CronExpr = cronExpr.String
	}
	if nextRunAt.Valid {
		schedule.NextRunAt = &nextRunAt.Time
	}
	if lastRunAt.Valid {
		schedule.LastRunAt = &lastRunAt.Time
	}
	if lastRunID.Valid {
		schedule.LastRunID = lastRunID.String
	}
}

// scanSchedules scans multiple schedules from rows
func (r *ScheduleRepository) scanSchedules(rows *sql.Rows) ([]*entity.Schedule, error) {
	var schedules []*entity.Schedule
	for rows.Next() {
		schedule := &entity.Schedule{}
		var nextRunAt, lastRunAt sql.NullTime
		var agentPaw, description, cronExpr, lastRunID sql.NullString

		err := rows.Scan(
			&schedule.ID,
			&schedule.Name,
			&description,
			&schedule.ScenarioID,
			&agentPaw,
			&schedule.Frequency,
			&cronExpr,
			&schedule.SafeMode,
			&schedule.Status,
			&nextRunAt,
			&lastRunAt,
			&lastRunID,
			&schedule.CreatedBy,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		applyNullableFields(schedule, description, agentPaw, cronExpr, lastRunID, nextRunAt, lastRunAt)
		schedules = append(schedules, schedule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return schedules, nil
}
