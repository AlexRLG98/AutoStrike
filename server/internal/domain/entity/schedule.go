package entity

import (
	"time"

	"github.com/robfig/cron/v3"
)

// ScheduleStatus represents the status of a schedule
type ScheduleStatus string

const (
	ScheduleStatusActive   ScheduleStatus = "active"
	ScheduleStatusPaused   ScheduleStatus = "paused"
	ScheduleStatusDisabled ScheduleStatus = "disabled"
)

// ScheduleFrequency represents how often a schedule runs
type ScheduleFrequency string

const (
	FrequencyOnce    ScheduleFrequency = "once"
	FrequencyHourly  ScheduleFrequency = "hourly"
	FrequencyDaily   ScheduleFrequency = "daily"
	FrequencyWeekly  ScheduleFrequency = "weekly"
	FrequencyMonthly ScheduleFrequency = "monthly"
	FrequencyCron    ScheduleFrequency = "cron"
)

// Schedule represents a scheduled execution
type Schedule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	ScenarioID  string            `json:"scenario_id"`
	AgentPaw    string            `json:"agent_paw,omitempty"` // Empty = any available agent
	Frequency   ScheduleFrequency `json:"frequency"`
	CronExpr    string            `json:"cron_expr,omitempty"` // Only for cron frequency
	SafeMode    bool              `json:"safe_mode"`
	Status      ScheduleStatus    `json:"status"`
	NextRunAt   *time.Time        `json:"next_run_at,omitempty"`
	LastRunAt   *time.Time        `json:"last_run_at,omitempty"`
	LastRunID   string            `json:"last_run_id,omitempty"` // Last execution ID
	CreatedBy   string            `json:"created_by"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// ScheduleRun represents a single run of a schedule
type ScheduleRun struct {
	ID          string    `json:"id"`
	ScheduleID  string    `json:"schedule_id"`
	ExecutionID string    `json:"execution_id"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Status      string    `json:"status"` // pending, running, completed, failed
	Error       string    `json:"error,omitempty"`
}

// CalculateNextRun calculates the next run time based on frequency
func (s *Schedule) CalculateNextRun(from time.Time) *time.Time {
	if s.Status != ScheduleStatusActive {
		return nil
	}

	var next time.Time
	switch s.Frequency {
	case FrequencyOnce:
		// One-time schedules don't have a next run after execution
		if s.LastRunAt != nil {
			return nil
		}
		if s.NextRunAt != nil {
			return s.NextRunAt
		}
		// If no start_at was provided, run immediately
		return &from
	case FrequencyHourly:
		next = from.Add(time.Hour)
	case FrequencyDaily:
		next = from.AddDate(0, 0, 1)
	case FrequencyWeekly:
		next = from.AddDate(0, 0, 7)
	case FrequencyMonthly:
		next = from.AddDate(0, 1, 0)
	case FrequencyCron:
		// Parse cron expression and calculate next run
		if s.CronExpr == "" {
			return nil
		}
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		schedule, err := parser.Parse(s.CronExpr)
		if err != nil {
			return nil
		}
		next = schedule.Next(from)
		return &next
	default:
		return nil
	}
	return &next
}

// IsReadyToRun checks if the schedule should run now
func (s *Schedule) IsReadyToRun(now time.Time) bool {
	if s.Status != ScheduleStatusActive {
		return false
	}
	if s.NextRunAt == nil {
		return false
	}
	return !now.Before(*s.NextRunAt)
}

// ValidateCronExpr validates a cron expression and returns an error if invalid
func ValidateCronExpr(cronExpr string) error {
	if cronExpr == "" {
		return nil
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(cronExpr)
	return err
}
