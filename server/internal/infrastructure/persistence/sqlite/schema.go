package sqlite

import (
	"database/sql"
)

// InitSchema initializes the database schema
func InitSchema(db *sql.DB) error {
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
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
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
	`

	_, err := db.Exec(schema)
	return err
}
