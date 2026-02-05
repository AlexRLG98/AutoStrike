package sqlite

import (
	"database/sql"
	"fmt"
	"strings"
)

// InitSchema initializes the database schema
func InitSchema(db *sql.DB) error {
	// Enable foreign key enforcement (disabled by default in SQLite)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	schema := `
	-- Agents table
	CREATE TABLE IF NOT EXISTS agents (
		paw TEXT PRIMARY KEY,
		hostname TEXT NOT NULL,
		username TEXT NOT NULL,
		platform TEXT NOT NULL,
		executors TEXT NOT NULL,
		status TEXT NOT NULL,
		last_seen DATETIME NOT NULL,
		created_at DATETIME NOT NULL
	);

	-- Techniques table
	CREATE TABLE IF NOT EXISTS techniques (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		tactic TEXT NOT NULL,
		platforms TEXT NOT NULL,
		executors TEXT NOT NULL,
		detection TEXT,
		is_safe BOOLEAN DEFAULT 1,
		created_at DATETIME NOT NULL
	);

	-- Scenarios table
	CREATE TABLE IF NOT EXISTS scenarios (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		phases TEXT NOT NULL,
		tags TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	-- Executions table
	CREATE TABLE IF NOT EXISTS executions (
		id TEXT PRIMARY KEY,
		scenario_id TEXT NOT NULL,
		status TEXT NOT NULL,
		started_at DATETIME NOT NULL,
		completed_at DATETIME,
		safe_mode BOOLEAN DEFAULT 1,
		score_overall REAL DEFAULT 0,
		score_blocked INTEGER DEFAULT 0,
		score_detected INTEGER DEFAULT 0,
		score_successful INTEGER DEFAULT 0,
		score_total INTEGER DEFAULT 0,
		FOREIGN KEY (scenario_id) REFERENCES scenarios(id)
	);

	-- Execution results table
	CREATE TABLE IF NOT EXISTS execution_results (
		id TEXT PRIMARY KEY,
		execution_id TEXT NOT NULL,
		technique_id TEXT NOT NULL,
		agent_paw TEXT NOT NULL,
		status TEXT NOT NULL,
		output TEXT,
		exit_code INTEGER DEFAULT 0,
		detected BOOLEAN DEFAULT 0,
		started_at DATETIME NOT NULL,
		completed_at DATETIME,
		FOREIGN KEY (execution_id) REFERENCES executions(id),
		FOREIGN KEY (technique_id) REFERENCES techniques(id),
		FOREIGN KEY (agent_paw) REFERENCES agents(paw)
	);

	-- Adversary profiles table
	CREATE TABLE IF NOT EXISTS adversary_profiles (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		techniques TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);

	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'viewer',
		is_active BOOLEAN NOT NULL DEFAULT 1,
		last_login_at DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	-- Notification settings table
	CREATE TABLE IF NOT EXISTS notification_settings (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL UNIQUE,
		channel TEXT NOT NULL DEFAULT 'email',
		enabled BOOLEAN NOT NULL DEFAULT 0,
		email_address TEXT,
		webhook_url TEXT,
		notify_on_start BOOLEAN DEFAULT 0,
		notify_on_complete BOOLEAN DEFAULT 1,
		notify_on_failure BOOLEAN DEFAULT 1,
		notify_on_score_alert BOOLEAN DEFAULT 1,
		score_alert_threshold REAL DEFAULT 50.0,
		notify_on_agent_offline BOOLEAN DEFAULT 0,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	-- Notifications table
	CREATE TABLE IF NOT EXISTS notifications (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		type TEXT NOT NULL,
		title TEXT NOT NULL,
		message TEXT,
		data TEXT,
		read BOOLEAN DEFAULT 0,
		sent_at DATETIME,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	-- Schedules table
	CREATE TABLE IF NOT EXISTS schedules (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		scenario_id TEXT NOT NULL,
		agent_paw TEXT,
		frequency TEXT NOT NULL,
		cron_expr TEXT,
		safe_mode BOOLEAN DEFAULT 1,
		status TEXT NOT NULL DEFAULT 'active',
		next_run_at DATETIME,
		last_run_at DATETIME,
		last_run_id TEXT,
		created_by TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		FOREIGN KEY (scenario_id) REFERENCES scenarios(id),
		FOREIGN KEY (created_by) REFERENCES users(id)
	);

	-- Schedule runs table
	CREATE TABLE IF NOT EXISTS schedule_runs (
		id TEXT PRIMARY KEY,
		schedule_id TEXT NOT NULL,
		execution_id TEXT,
		started_at DATETIME NOT NULL,
		completed_at DATETIME,
		status TEXT NOT NULL DEFAULT 'pending',
		error TEXT,
		FOREIGN KEY (schedule_id) REFERENCES schedules(id),
		FOREIGN KEY (execution_id) REFERENCES executions(id)
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
	CREATE INDEX IF NOT EXISTS idx_agents_platform ON agents(platform);
	CREATE INDEX IF NOT EXISTS idx_techniques_tactic ON techniques(tactic);
	CREATE INDEX IF NOT EXISTS idx_executions_scenario ON executions(scenario_id);
	CREATE INDEX IF NOT EXISTS idx_execution_results_execution ON execution_results(execution_id);
	CREATE INDEX IF NOT EXISTS idx_execution_results_technique ON execution_results(technique_id);
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_notification_settings_user ON notification_settings(user_id);
	CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id);
	CREATE INDEX IF NOT EXISTS idx_notifications_read ON notifications(read);
	CREATE INDEX IF NOT EXISTS idx_schedules_status ON schedules(status);
	CREATE INDEX IF NOT EXISTS idx_schedules_scenario ON schedules(scenario_id);
	CREATE INDEX IF NOT EXISTS idx_schedules_next_run ON schedules(next_run_at);
	CREATE INDEX IF NOT EXISTS idx_schedule_runs_schedule ON schedule_runs(schedule_id);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return err
	}

	// Run migrations
	return Migrate(db)
}

// Migrate runs database migrations for existing databases
func Migrate(db *sql.DB) error {
	// Migration: Add is_active and last_login_at columns to users table
	if err := addColumnIfNotExists(db, "users", "is_active", "BOOLEAN NOT NULL DEFAULT 1"); err != nil {
		return fmt.Errorf("failed to add is_active column: %w", err)
	}
	if err := addColumnIfNotExists(db, "users", "last_login_at", "DATETIME"); err != nil {
		return fmt.Errorf("failed to add last_login_at column: %w", err)
	}

	return nil
}

// addColumnIfNotExists adds a column to a table if it doesn't already exist
func addColumnIfNotExists(db *sql.DB, table, column, definition string) error {
	// Check if column exists
	query := fmt.Sprintf("PRAGMA table_info(%s)", table)
	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	var exists bool
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return err
		}
		if strings.EqualFold(name, column) {
			exists = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if !exists {
		alterQuery := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)
		_, err := db.Exec(alterQuery)
		return err
	}

	return nil
}
