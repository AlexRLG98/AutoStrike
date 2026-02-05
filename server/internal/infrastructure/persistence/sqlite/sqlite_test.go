package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"autostrike/internal/domain/entity"

	_ "github.com/mattn/go-sqlite3"
)

// Test constants for foreign key dependencies
const (
	testScenarioID = "scenario-1"
	testUserID     = "user-1"
	testAgentPaw   = "agent-1"
	testExecID     = "exec-1"
	testTechID     = "T1059"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	if err := InitSchema(db); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}
	return db
}

// createTestScenario creates a scenario for foreign key references in tests
func createTestScenario(t *testing.T, db *sql.DB, id string) {
	_, err := db.Exec(`INSERT INTO scenarios (id, name, description, phases, tags, created_at, updated_at)
		VALUES (?, 'Test Scenario', 'Test', '[]', '', datetime('now'), datetime('now'))`, id)
	if err != nil {
		t.Fatalf("Failed to create test scenario: %v", err)
	}
}

// createTestUser creates a user for foreign key references in tests
func createTestUser(t *testing.T, db *sql.DB, id string) {
	_, err := db.Exec(`INSERT INTO users (id, username, email, password_hash, role, is_active, created_at, updated_at)
		VALUES (?, ?, ?, 'hash', 'admin', 1, datetime('now'), datetime('now'))`, id, "user_"+id, id+"@test.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
}

// createTestTechnique creates a technique for foreign key references in tests
func createTestTechnique(t *testing.T, db *sql.DB, id string) {
	_, err := db.Exec(`INSERT INTO techniques (id, name, description, tactic, platforms, executors, detection, is_safe, created_at)
		VALUES (?, 'Test Technique', 'Test', 'discovery', '["linux"]', '["sh"]', '', 1, datetime('now'))`, id)
	if err != nil {
		t.Fatalf("Failed to create test technique: %v", err)
	}
}

// createTestAgent creates an agent for foreign key references in tests
func createTestAgent(t *testing.T, db *sql.DB, paw string) {
	_, err := db.Exec(`INSERT INTO agents (paw, hostname, username, platform, executors, status, last_seen, created_at)
		VALUES (?, 'testhost', 'testuser', 'linux', '["sh"]', 'online', datetime('now'), datetime('now'))`, paw)
	if err != nil {
		t.Fatalf("Failed to create test agent: %v", err)
	}
}

// createTestExecution creates an execution for foreign key references in tests
func createTestExecution(t *testing.T, db *sql.DB, id, scenarioID string) {
	_, err := db.Exec(`INSERT INTO executions (id, scenario_id, status, started_at, safe_mode)
		VALUES (?, ?, 'running', datetime('now'), 1)`, id, scenarioID)
	if err != nil {
		t.Fatalf("Failed to create test execution: %v", err)
	}
}

// setupTestDBWithFKData sets up test DB with common foreign key dependencies
func setupTestDBWithFKData(t *testing.T) *sql.DB {
	db := setupTestDB(t)
	createTestScenario(t, db, testScenarioID)
	createTestUser(t, db, testUserID)
	createTestTechnique(t, db, testTechID)
	createTestAgent(t, db, testAgentPaw)
	return db
}

// Schema tests
func TestInitSchema(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = InitSchema(db)
	if err != nil {
		t.Fatalf("InitSchema failed: %v", err)
	}

	// Verify tables exist
	tables := []string{"agents", "techniques", "scenarios", "executions", "execution_results"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("Table %s not created: %v", table, err)
		}
	}
}

// Agent Repository tests
func TestNewAgentRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAgentRepository(db)
	if repo == nil {
		t.Error("Expected non-nil repository")
	}
}

func TestAgentRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAgentRepository(db)
	ctx := context.Background()

	agent := &entity.Agent{
		Paw:       "test-paw",
		Hostname:  "test-host",
		Username:  "test-user",
		Platform:  "linux",
		Executors: []string{"sh", "bash"},
		Status:    entity.AgentOnline,
		LastSeen:  time.Now(),
		}

	err := repo.Create(ctx, agent)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
}

func TestAgentRepository_FindByPaw(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAgentRepository(db)
	ctx := context.Background()

	agent := &entity.Agent{
		Paw:       "test-paw",
		Hostname:  "test-host",
		Username:  "test-user",
		Platform:  "linux",
		Executors: []string{"sh"},
		Status:    entity.AgentOnline,
		LastSeen:  time.Now(),
		}
	_ = repo.Create(ctx, agent)

	found, err := repo.FindByPaw(ctx, "test-paw")
	if err != nil {
		t.Fatalf("FindByPaw failed: %v", err)
	}
	if found.Hostname != "test-host" {
		t.Errorf("Expected hostname test-host, got %s", found.Hostname)
	}
}

func TestAgentRepository_FindByPaw_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAgentRepository(db)
	ctx := context.Background()

	_, err := repo.FindByPaw(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent agent")
	}
}

func TestAgentRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAgentRepository(db)
	ctx := context.Background()

	agent := &entity.Agent{
		Paw:       "test-paw",
		Hostname:  "old-host",
		Username:  "test-user",
		Platform:  "linux",
		Executors: []string{"sh"},
		Status:    entity.AgentOnline,
		LastSeen:  time.Now(),
		}
	_ = repo.Create(ctx, agent)

	agent.Hostname = "new-host"
	err := repo.Update(ctx, agent)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	found, _ := repo.FindByPaw(ctx, "test-paw")
	if found.Hostname != "new-host" {
		t.Errorf("Expected hostname new-host, got %s", found.Hostname)
	}
}

func TestAgentRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAgentRepository(db)
	ctx := context.Background()

	agent := &entity.Agent{
		Paw:       "test-paw",
		Hostname:  "test-host",
		Username:  "test-user",
		Platform:  "linux",
		Executors: []string{"sh"},
		Status:    entity.AgentOnline,
		LastSeen:  time.Now(),
		}
	_ = repo.Create(ctx, agent)

	err := repo.Delete(ctx, "test-paw")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = repo.FindByPaw(ctx, "test-paw")
	if err == nil {
		t.Error("Expected error after delete")
	}
}

func TestAgentRepository_FindAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAgentRepository(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		agent := &entity.Agent{
			Paw:       "paw-" + string(rune('a'+i)),
			Hostname:  "host",
			Username:  "user",
			Platform:  "linux",
			Executors: []string{"sh"},
			Status:    entity.AgentOnline,
			LastSeen:  time.Now(),
				}
	_ = repo.Create(ctx, agent)
	}

	agents, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(agents) != 3 {
		t.Errorf("Expected 3 agents, got %d", len(agents))
	}
}

func TestAgentRepository_FindByStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAgentRepository(db)
	ctx := context.Background()

	online := &entity.Agent{
		Paw: "online", Hostname: "h", Username: "u", Platform: "linux",
		Executors: []string{"sh"}, Status: entity.AgentOnline,
		LastSeen: time.Now(), CreatedAt: time.Now(),
	}
	offline := &entity.Agent{
		Paw: "offline", Hostname: "h", Username: "u", Platform: "linux",
		Executors: []string{"sh"}, Status: entity.AgentOffline,
		LastSeen: time.Now(), CreatedAt: time.Now(),
	}
	_ = repo.Create(ctx, online)
	_ = repo.Create(ctx, offline)

	agents, err := repo.FindByStatus(ctx, entity.AgentOnline)
	if err != nil {
		t.Fatalf("FindByStatus failed: %v", err)
	}
	if len(agents) != 1 {
		t.Errorf("Expected 1 online agent, got %d", len(agents))
	}
}

func TestAgentRepository_FindByPlatform(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAgentRepository(db)
	ctx := context.Background()

	linux := &entity.Agent{
		Paw: "linux", Hostname: "h", Username: "u", Platform: "linux",
		Executors: []string{"sh"}, Status: entity.AgentOnline,
		LastSeen: time.Now(), CreatedAt: time.Now(),
	}
	windows := &entity.Agent{
		Paw: "windows", Hostname: "h", Username: "u", Platform: "windows",
		Executors: []string{"cmd"}, Status: entity.AgentOnline,
		LastSeen: time.Now(), CreatedAt: time.Now(),
	}
	_ = repo.Create(ctx, linux)
	_ = repo.Create(ctx, windows)

	agents, err := repo.FindByPlatform(ctx, "linux")
	if err != nil {
		t.Fatalf("FindByPlatform failed: %v", err)
	}
	if len(agents) != 1 {
		t.Errorf("Expected 1 linux agent, got %d", len(agents))
	}
}

func TestAgentRepository_UpdateLastSeen(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAgentRepository(db)
	ctx := context.Background()

	agent := &entity.Agent{
		Paw: "test", Hostname: "h", Username: "u", Platform: "linux",
		Executors: []string{"sh"}, Status: entity.AgentOffline,
		LastSeen: time.Now().Add(-time.Hour), CreatedAt: time.Now(),
	}
	_ = repo.Create(ctx, agent)

	err := repo.UpdateLastSeen(ctx, "test")
	if err != nil {
		t.Fatalf("UpdateLastSeen failed: %v", err)
	}

	found, _ := repo.FindByPaw(ctx, "test")
	if found.Status != entity.AgentOnline {
		t.Error("Expected agent to be online after UpdateLastSeen")
	}
}

func TestAgentRepository_FindByPaws(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAgentRepository(db)
	ctx := context.Background()

	// Create multiple agents
	agents := []*entity.Agent{
		{Paw: "paw-1", Hostname: "host1", Username: "user1", Platform: "linux", Executors: []string{"sh"}, Status: entity.AgentOnline, LastSeen: time.Now(), CreatedAt: time.Now()},
		{Paw: "paw-2", Hostname: "host2", Username: "user2", Platform: "windows", Executors: []string{"cmd"}, Status: entity.AgentOnline, LastSeen: time.Now(), CreatedAt: time.Now()},
		{Paw: "paw-3", Hostname: "host3", Username: "user3", Platform: "linux", Executors: []string{"bash"}, Status: entity.AgentOffline, LastSeen: time.Now(), CreatedAt: time.Now()},
	}
	for _, agent := range agents {
		_ = repo.Create(ctx, agent)
	}

	// Test: Find multiple agents by paws
	found, err := repo.FindByPaws(ctx, []string{"paw-1", "paw-3"})
	if err != nil {
		t.Fatalf("FindByPaws failed: %v", err)
	}
	if len(found) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(found))
	}

	// Verify correct agents were found
	pawSet := make(map[string]bool)
	for _, a := range found {
		pawSet[a.Paw] = true
	}
	if !pawSet["paw-1"] || !pawSet["paw-3"] {
		t.Error("Expected to find paw-1 and paw-3")
	}
}

func TestAgentRepository_FindByPaws_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAgentRepository(db)
	ctx := context.Background()

	// Test: Empty paws slice returns empty result
	found, err := repo.FindByPaws(ctx, []string{})
	if err != nil {
		t.Fatalf("FindByPaws with empty slice failed: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("Expected 0 agents for empty paws, got %d", len(found))
	}
}

func TestAgentRepository_FindByPaws_NonExistent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAgentRepository(db)
	ctx := context.Background()

	// Test: Non-existent paws return empty result (not an error)
	found, err := repo.FindByPaws(ctx, []string{"nonexistent-1", "nonexistent-2"})
	if err != nil {
		t.Fatalf("FindByPaws with non-existent paws failed: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("Expected 0 agents for non-existent paws, got %d", len(found))
	}
}

func TestAgentRepository_FindByPaws_Partial(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAgentRepository(db)
	ctx := context.Background()

	// Create one agent
	agent := &entity.Agent{
		Paw: "existing", Hostname: "host", Username: "user", Platform: "linux",
		Executors: []string{"sh"}, Status: entity.AgentOnline,
		LastSeen: time.Now(), CreatedAt: time.Now(),
	}
	_ = repo.Create(ctx, agent)

	// Test: Mixed existing and non-existing paws
	found, err := repo.FindByPaws(ctx, []string{"existing", "nonexistent"})
	if err != nil {
		t.Fatalf("FindByPaws with partial match failed: %v", err)
	}
	if len(found) != 1 {
		t.Errorf("Expected 1 agent for partial match, got %d", len(found))
	}
	if found[0].Paw != "existing" {
		t.Errorf("Expected paw 'existing', got '%s'", found[0].Paw)
	}
}

func TestAgentRepository_FindByPaws_Single(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAgentRepository(db)
	ctx := context.Background()

	agent := &entity.Agent{
		Paw: "single-paw", Hostname: "host", Username: "user", Platform: "darwin",
		Executors: []string{"zsh"}, Status: entity.AgentOnline,
		LastSeen: time.Now(), CreatedAt: time.Now(),
	}
	_ = repo.Create(ctx, agent)

	// Test: Single paw lookup
	found, err := repo.FindByPaws(ctx, []string{"single-paw"})
	if err != nil {
		t.Fatalf("FindByPaws with single paw failed: %v", err)
	}
	if len(found) != 1 {
		t.Errorf("Expected 1 agent, got %d", len(found))
	}
	if found[0].Hostname != "host" {
		t.Errorf("Expected hostname 'host', got '%s'", found[0].Hostname)
	}
}

// Technique Repository tests
func TestNewTechniqueRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTechniqueRepository(db)
	if repo == nil {
		t.Error("Expected non-nil repository")
	}
}

func TestTechniqueRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTechniqueRepository(db)
	ctx := context.Background()

	tech := &entity.Technique{
		ID:          "T1059",
		Name:        "Command Execution",
		Description: "Execute commands",
		Tactic:      entity.TacticExecution,
		Platforms:   []string{"windows", "linux"},
		Executors: []entity.Executor{
			{Type: "sh", Command: "whoami"},
		},
		IsSafe:    true,
		}

	err := repo.Create(ctx, tech)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
}

func TestTechniqueRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTechniqueRepository(db)
	ctx := context.Background()

	tech := &entity.Technique{
		ID:        "T1059",
		Name:      "Command Execution",
		Tactic:    entity.TacticExecution,
		Platforms: []string{"linux"},
		Executors: []entity.Executor{{Type: "sh", Command: "whoami"}},
		IsSafe:    true,
		}
	_ = repo.Create(ctx, tech)

	found, err := repo.FindByID(ctx, "T1059")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.Name != "Command Execution" {
		t.Errorf("Expected name Command Execution, got %s", found.Name)
	}
}

func TestTechniqueRepository_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTechniqueRepository(db)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent technique")
	}
}

func TestTechniqueRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTechniqueRepository(db)
	ctx := context.Background()

	tech := &entity.Technique{
		ID:        "T1059",
		Name:      "Old Name",
		Tactic:    entity.TacticExecution,
		Platforms: []string{"linux"},
		Executors: []entity.Executor{{Type: "sh", Command: "whoami"}},
		IsSafe:    true,
		}
	_ = repo.Create(ctx, tech)

	tech.Name = "New Name"
	err := repo.Update(ctx, tech)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	found, _ := repo.FindByID(ctx, "T1059")
	if found.Name != "New Name" {
		t.Errorf("Expected name New Name, got %s", found.Name)
	}
}

func TestTechniqueRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTechniqueRepository(db)
	ctx := context.Background()

	tech := &entity.Technique{
		ID:        "T1059",
		Name:      "Test",
		Tactic:    entity.TacticExecution,
		Platforms: []string{"linux"},
		Executors: []entity.Executor{{Type: "sh", Command: "whoami"}},
		}
	_ = repo.Create(ctx, tech)

	err := repo.Delete(ctx, "T1059")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = repo.FindByID(ctx, "T1059")
	if err == nil {
		t.Error("Expected error after delete")
	}
}

func TestTechniqueRepository_FindAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTechniqueRepository(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		tech := &entity.Technique{
			ID:        "T" + string(rune('0'+i)),
			Name:      "Tech",
			Tactic:    entity.TacticExecution,
			Platforms: []string{"linux"},
			Executors: []entity.Executor{{Type: "sh", Command: "cmd"}},
				}
	_ = repo.Create(ctx, tech)
	}

	techniques, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(techniques) != 3 {
		t.Errorf("Expected 3 techniques, got %d", len(techniques))
	}
}

func TestTechniqueRepository_FindByTactic(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTechniqueRepository(db)
	ctx := context.Background()

	exec := &entity.Technique{
		ID: "T1", Name: "Exec", Tactic: entity.TacticExecution,
		Platforms: []string{"linux"}, Executors: []entity.Executor{{Type: "sh", Command: "c"}},
		}
	persist := &entity.Technique{
		ID: "T2", Name: "Persist", Tactic: entity.TacticPersistence,
		Platforms: []string{"linux"}, Executors: []entity.Executor{{Type: "sh", Command: "c"}},
		}
	_ = repo.Create(ctx, exec)
	_ = repo.Create(ctx, persist)

	techniques, err := repo.FindByTactic(ctx, entity.TacticExecution)
	if err != nil {
		t.Fatalf("FindByTactic failed: %v", err)
	}
	if len(techniques) != 1 {
		t.Errorf("Expected 1 execution technique, got %d", len(techniques))
	}
}

func TestTechniqueRepository_FindByPlatform(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTechniqueRepository(db)
	ctx := context.Background()

	linux := &entity.Technique{
		ID: "T1", Name: "Linux", Tactic: entity.TacticExecution,
		Platforms: []string{"linux"}, Executors: []entity.Executor{{Type: "sh", Command: "c"}},
		}
	windows := &entity.Technique{
		ID: "T2", Name: "Windows", Tactic: entity.TacticExecution,
		Platforms: []string{"windows"}, Executors: []entity.Executor{{Type: "cmd", Command: "c"}},
		}
	_ = repo.Create(ctx, linux)
	_ = repo.Create(ctx, windows)

	techniques, err := repo.FindByPlatform(ctx, "linux")
	if err != nil {
		t.Fatalf("FindByPlatform failed: %v", err)
	}
	if len(techniques) != 1 {
		t.Errorf("Expected 1 linux technique, got %d", len(techniques))
	}
}

// Scenario Repository tests
func TestNewScenarioRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewScenarioRepository(db)
	if repo == nil {
		t.Error("Expected non-nil repository")
	}
}

func TestScenarioRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScenarioRepository(db)
	ctx := context.Background()

	scenario := &entity.Scenario{
		ID:          "s1",
		Name:        "Test Scenario",
		Description: "A test",
		Phases: []entity.Phase{
			{Name: "Phase1", Techniques: []string{"T1059"}},
		},
		Tags:      []string{"test"},
			UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, scenario)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
}

func TestScenarioRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScenarioRepository(db)
	ctx := context.Background()

	scenario := &entity.Scenario{
		ID:   "s1",
		Name: "Test",
		Phases: []entity.Phase{
			{Name: "Phase1", Techniques: []string{"T1059"}},
		},
		Tags:      []string{"test"},
			UpdatedAt: time.Now(),
	}
	_ = repo.Create(ctx, scenario)

	found, err := repo.FindByID(ctx, "s1")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.Name != "Test" {
		t.Errorf("Expected name Test, got %s", found.Name)
	}
}

func TestScenarioRepository_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScenarioRepository(db)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent scenario")
	}
}

func TestScenarioRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScenarioRepository(db)
	ctx := context.Background()

	scenario := &entity.Scenario{
		ID:        "s1",
		Name:      "Old Name",
		Phases:    []entity.Phase{{Name: "P1", Techniques: []string{"T1"}}},
		Tags:      []string{},
			UpdatedAt: time.Now(),
	}
	_ = repo.Create(ctx, scenario)

	scenario.Name = "New Name"
	err := repo.Update(ctx, scenario)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	found, _ := repo.FindByID(ctx, "s1")
	if found.Name != "New Name" {
		t.Errorf("Expected name New Name, got %s", found.Name)
	}
}

func TestScenarioRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScenarioRepository(db)
	ctx := context.Background()

	scenario := &entity.Scenario{
		ID:        "s1",
		Name:      "Test",
		Phases:    []entity.Phase{{Name: "P1", Techniques: []string{"T1"}}},
		Tags:      []string{},
			UpdatedAt: time.Now(),
	}
	_ = repo.Create(ctx, scenario)

	err := repo.Delete(ctx, "s1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = repo.FindByID(ctx, "s1")
	if err == nil {
		t.Error("Expected error after delete")
	}
}

func TestScenarioRepository_FindAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScenarioRepository(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		scenario := &entity.Scenario{
			ID:        "s" + string(rune('0'+i)),
			Name:      "Scenario",
			Phases:    []entity.Phase{{Name: "P", Techniques: []string{"T1"}}},
			Tags:      []string{},
					UpdatedAt: time.Now(),
		}
	_ = repo.Create(ctx, scenario)
	}

	scenarios, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(scenarios) != 3 {
		t.Errorf("Expected 3 scenarios, got %d", len(scenarios))
	}
}

func TestScenarioRepository_FindByTag(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScenarioRepository(db)
	ctx := context.Background()

	tagged := &entity.Scenario{
		ID:        "s1",
		Name:      "Tagged",
		Phases:    []entity.Phase{{Name: "P", Techniques: []string{"T1"}}},
		Tags:      []string{"important"},
			UpdatedAt: time.Now(),
	}
	untagged := &entity.Scenario{
		ID:        "s2",
		Name:      "Untagged",
		Phases:    []entity.Phase{{Name: "P", Techniques: []string{"T1"}}},
		Tags:      []string{"other"},
			UpdatedAt: time.Now(),
	}
	_ = repo.Create(ctx, tagged)
	_ = repo.Create(ctx, untagged)

	scenarios, err := repo.FindByTag(ctx, "important")
	if err != nil {
		t.Fatalf("FindByTag failed: %v", err)
	}
	if len(scenarios) != 1 {
		t.Errorf("Expected 1 tagged scenario, got %d", len(scenarios))
	}
}

// Result Repository tests
func TestNewResultRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewResultRepository(db)
	if repo == nil {
		t.Error("Expected non-nil repository")
	}
}

func TestResultRepository_CreateExecution(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewResultRepository(db)
	ctx := context.Background()

	// Create scenario first for foreign key constraint
	createTestScenario(t, db, "s1")

	exec := &entity.Execution{
		ID:         "e1",
		ScenarioID: "s1",
		Status:     entity.ExecutionRunning,
		StartedAt:  time.Now(),
		SafeMode:   true,
	}

	err := repo.CreateExecution(ctx, exec)
	if err != nil {
		t.Fatalf("CreateExecution failed: %v", err)
	}
}

func TestResultRepository_FindExecutionByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewResultRepository(db)
	ctx := context.Background()

	// Create scenario first for foreign key constraint
	createTestScenario(t, db, "s1")

	exec := &entity.Execution{
		ID:         "e1",
		ScenarioID: "s1",
		Status:     entity.ExecutionRunning,
		StartedAt:  time.Now(),
	}
	_ = repo.CreateExecution(ctx, exec)

	found, err := repo.FindExecutionByID(ctx, "e1")
	if err != nil {
		t.Fatalf("FindExecutionByID failed: %v", err)
	}
	if found.ScenarioID != "s1" {
		t.Errorf("Expected scenario s1, got %s", found.ScenarioID)
	}
}

func TestResultRepository_FindExecutionByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewResultRepository(db)
	ctx := context.Background()

	_, err := repo.FindExecutionByID(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent execution")
	}
}

func TestResultRepository_UpdateExecution(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewResultRepository(db)
	ctx := context.Background()

	// Create scenario first for foreign key constraint
	createTestScenario(t, db, "s1")

	exec := &entity.Execution{
		ID:         "e1",
		ScenarioID: "s1",
		Status:     entity.ExecutionRunning,
		StartedAt:  time.Now(),
	}
	_ = repo.CreateExecution(ctx, exec)

	now := time.Now()
	exec.Status = entity.ExecutionCompleted
	exec.CompletedAt = &now
	exec.Score = &entity.SecurityScore{Overall: 0.8, Blocked: 1, Detected: 1, Successful: 8, Total: 10}

	err := repo.UpdateExecution(ctx, exec)
	if err != nil {
		t.Fatalf("UpdateExecution failed: %v", err)
	}

	found, _ := repo.FindExecutionByID(ctx, "e1")
	if found.Status != entity.ExecutionCompleted {
		t.Error("Expected status Completed")
	}
}

func TestResultRepository_FindExecutionsByScenario(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewResultRepository(db)
	ctx := context.Background()

	// Create scenarios first for foreign key constraint
	createTestScenario(t, db, "s1")
	createTestScenario(t, db, "s2")

	for i := 0; i < 2; i++ {
		exec := &entity.Execution{
			ID:         "e" + string(rune('0'+i)),
			ScenarioID: "s1",
			Status:     entity.ExecutionCompleted,
			StartedAt:  time.Now(),
		}
		_ = repo.CreateExecution(ctx, exec)
	}
	exec3 := &entity.Execution{
		ID:         "e3",
		ScenarioID: "s2",
		Status:     entity.ExecutionCompleted,
		StartedAt:  time.Now(),
	}
	_ = repo.CreateExecution(ctx, exec3)

	executions, err := repo.FindExecutionsByScenario(ctx, "s1")
	if err != nil {
		t.Fatalf("FindExecutionsByScenario failed: %v", err)
	}
	if len(executions) != 2 {
		t.Errorf("Expected 2 executions for s1, got %d", len(executions))
	}
}

func TestResultRepository_FindRecentExecutions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewResultRepository(db)
	ctx := context.Background()

	// Create scenario first for foreign key constraint
	createTestScenario(t, db, "s1")

	for i := 0; i < 5; i++ {
		exec := &entity.Execution{
			ID:         "e" + string(rune('0'+i)),
			ScenarioID: "s1",
			Status:     entity.ExecutionCompleted,
			StartedAt:  time.Now(),
		}
		_ = repo.CreateExecution(ctx, exec)
	}

	executions, err := repo.FindRecentExecutions(ctx, 3)
	if err != nil {
		t.Fatalf("FindRecentExecutions failed: %v", err)
	}
	if len(executions) != 3 {
		t.Errorf("Expected 3 recent executions, got %d", len(executions))
	}
}

func TestResultRepository_CreateResult(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewResultRepository(db)
	ctx := context.Background()

	// Create foreign key dependencies
	createTestScenario(t, db, "s1")
	createTestExecution(t, db, "e1", "s1")
	createTestTechnique(t, db, "T1059")
	createTestAgent(t, db, "paw1")

	result := &entity.ExecutionResult{
		ID:          "r1",
		ExecutionID: "e1",
		TechniqueID: "T1059",
		AgentPaw:    "paw1",
		Status:      entity.StatusPending,
		StartedAt:   time.Now(),
	}

	err := repo.CreateResult(ctx, result)
	if err != nil {
		t.Fatalf("CreateResult failed: %v", err)
	}
}

func TestResultRepository_UpdateResult(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewResultRepository(db)
	ctx := context.Background()

	// Create foreign key dependencies
	createTestScenario(t, db, "s1")
	createTestExecution(t, db, "e1", "s1")
	createTestTechnique(t, db, "T1059")
	createTestAgent(t, db, "paw1")

	result := &entity.ExecutionResult{
		ID:          "r1",
		ExecutionID: "e1",
		TechniqueID: "T1059",
		AgentPaw:    "paw1",
		Status:      entity.StatusPending,
		StartedAt:   time.Now(),
	}
	_ = repo.CreateResult(ctx, result)

	now := time.Now()
	result.Status = entity.StatusSuccess
	result.Output = "test output"
	result.CompletedAt = &now

	err := repo.UpdateResult(ctx, result)
	if err != nil {
		t.Fatalf("UpdateResult failed: %v", err)
	}
}

func TestResultRepository_FindResultsByExecution(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewResultRepository(db)
	ctx := context.Background()

	// Create foreign key dependencies
	createTestScenario(t, db, "s1")
	createTestExecution(t, db, "e1", "s1")
	createTestTechnique(t, db, "T1059")
	createTestAgent(t, db, "paw1")

	for i := 0; i < 3; i++ {
		result := &entity.ExecutionResult{
			ID:          "r" + string(rune('0'+i)),
			ExecutionID: "e1",
			TechniqueID: "T1059",
			AgentPaw:    "paw1",
			Status:      entity.StatusSuccess,
			StartedAt:   time.Now(),
		}
		_ = repo.CreateResult(ctx, result)
	}

	results, err := repo.FindResultsByExecution(ctx, "e1")
	if err != nil {
		t.Fatalf("FindResultsByExecution failed: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
}

func TestResultRepository_FindResultsByTechnique(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewResultRepository(db)
	ctx := context.Background()

	// Create foreign key dependencies
	createTestScenario(t, db, "s1")
	createTestExecution(t, db, "e1", "s1")
	createTestTechnique(t, db, "T1059")
	createTestTechnique(t, db, "T1055")
	createTestAgent(t, db, "paw1")

	r1 := &entity.ExecutionResult{
		ID: "r1", ExecutionID: "e1", TechniqueID: "T1059",
		AgentPaw: "paw1", Status: entity.StatusSuccess, StartedAt: time.Now(),
	}
	r2 := &entity.ExecutionResult{
		ID: "r2", ExecutionID: "e1", TechniqueID: "T1055",
		AgentPaw: "paw1", Status: entity.StatusSuccess, StartedAt: time.Now(),
	}
	_ = repo.CreateResult(ctx, r1)
	_ = repo.CreateResult(ctx, r2)

	results, err := repo.FindResultsByTechnique(ctx, "T1059")
	if err != nil {
		t.Fatalf("FindResultsByTechnique failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for T1059, got %d", len(results))
	}
}

// ImportFromYAML tests
func TestTechniqueRepository_ImportFromYAML(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTechniqueRepository(db)
	ctx := context.Background()

	// Create a temporary YAML file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "techniques.yaml")
	yamlContent := `
- id: "T1001"
  name: "Data Obfuscation"
  description: "Test technique"
  tactic: "command_and_control"
  platforms:
    - "windows"
    - "linux"
  executors:
    - type: "sh"
      command: "echo test"
  is_safe: true
- id: "T1002"
  name: "Data Compressed"
  description: "Another technique"
  tactic: "exfiltration"
  platforms:
    - "linux"
  executors:
    - type: "bash"
      command: "gzip test"
  is_safe: true
`
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write YAML file: %v", err)
	}

	err = repo.ImportFromYAML(ctx, yamlPath)
	if err != nil {
		t.Fatalf("ImportFromYAML failed: %v", err)
	}

	// Verify techniques were imported
	techniques, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(techniques) != 2 {
		t.Errorf("Expected 2 techniques, got %d", len(techniques))
	}
}

func TestTechniqueRepository_ImportFromYAML_FileNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTechniqueRepository(db)
	ctx := context.Background()

	err := repo.ImportFromYAML(ctx, "/nonexistent/path/techniques.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestTechniqueRepository_ImportFromYAML_InvalidYAML(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTechniqueRepository(db)
	ctx := context.Background()

	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "invalid.yaml")
	err := os.WriteFile(yamlPath, []byte("not: valid: yaml: ["), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	err = repo.ImportFromYAML(ctx, yamlPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

// Tests for JSON marshal errors in Create functions
func TestAgentRepository_Create_MarshalError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAgentRepository(db)
	ctx := context.Background()

	// Valid agent - should succeed
	agent := &entity.Agent{
		Paw:       "marshal-test",
		Hostname:  "host",
		Username:  "user",
		Platform:  "linux",
		Executors: []string{"sh"},
		Status:    entity.AgentOnline,
		LastSeen:  time.Now(),
		CreatedAt: time.Now(),
	}

	err := repo.Create(ctx, agent)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify it was created
	found, err := repo.FindByPaw(ctx, "marshal-test")
	if err != nil {
		t.Fatalf("FindByPaw failed: %v", err)
	}
	if found.Paw != "marshal-test" {
		t.Errorf("Expected paw 'marshal-test', got '%s'", found.Paw)
	}
}

// Test for rows.Err() path coverage
func TestAgentRepository_FindAll_EmptyDB(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAgentRepository(db)
	ctx := context.Background()

	// FindAll on empty database
	agents, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents in empty DB, got %d", len(agents))
	}
}

func TestScenarioRepository_Create_EmptyPhases(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScenarioRepository(db)
	ctx := context.Background()

	scenario := &entity.Scenario{
		ID:        "empty-phases",
		Name:      "Empty Phases Test",
		Phases:    []entity.Phase{},
		Tags:      []string{},
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, scenario)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := repo.FindByID(ctx, "empty-phases")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if len(found.Phases) != 0 {
		t.Errorf("Expected 0 phases, got %d", len(found.Phases))
	}
}

func TestResultRepository_FindResultsByExecution_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewResultRepository(db)
	ctx := context.Background()

	// Find results for non-existent execution
	results, err := repo.FindResultsByExecution(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("FindResultsByExecution failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestResultRepository_FindExecutionsByScenario_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewResultRepository(db)
	ctx := context.Background()

	executions, err := repo.FindExecutionsByScenario(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("FindExecutionsByScenario failed: %v", err)
	}
	if len(executions) != 0 {
		t.Errorf("Expected 0 executions, got %d", len(executions))
	}
}

func TestTechniqueRepository_FindByTactic_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTechniqueRepository(db)
	ctx := context.Background()

	techniques, err := repo.FindByTactic(ctx, entity.TacticDiscovery)
	if err != nil {
		t.Fatalf("FindByTactic failed: %v", err)
	}
	if len(techniques) != 0 {
		t.Errorf("Expected 0 techniques, got %d", len(techniques))
	}
}

// Test JSON unmarshal error paths by inserting invalid JSON
func TestAgentRepository_FindByPaw_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAgentRepository(db)
	ctx := context.Background()

	// Insert agent with invalid JSON executors directly
	_, err := db.Exec(`
		INSERT INTO agents (paw, hostname, username, platform, executors, status, last_seen, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "invalid-json-agent", "host", "user", "linux", "not valid json", "online", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Should still work, with empty executors
	agent, err := repo.FindByPaw(ctx, "invalid-json-agent")
	if err != nil {
		t.Fatalf("FindByPaw failed: %v", err)
	}
	if len(agent.Executors) != 0 {
		t.Errorf("Expected empty executors for invalid JSON, got %v", agent.Executors)
	}
}

func TestAgentRepository_FindAll_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewAgentRepository(db)
	ctx := context.Background()

	// Insert agent with invalid JSON
	_, _ = db.Exec(`
		INSERT INTO agents (paw, hostname, username, platform, executors, status, last_seen, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "agent-invalid", "host", "user", "linux", "{invalid json}", "online", time.Now(), time.Now())

	agents, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	// Should succeed with empty executors
	if len(agents) == 0 {
		t.Error("Expected at least 1 agent")
	}
}

func TestTechniqueRepository_FindAll_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTechniqueRepository(db)
	ctx := context.Background()

	// Insert technique with invalid JSON fields (include created_at)
	_, err := db.Exec(`
		INSERT INTO techniques (id, name, description, tactic, platforms, executors, detection, is_safe, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "T-invalid", "Test", "Desc", "execution", "not json", "not json", "not json", true, time.Now())
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	techniques, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(techniques) != 1 {
		t.Fatalf("Expected 1 technique, got %d", len(techniques))
	}
	// Should have empty arrays due to JSON parse error
	if len(techniques[0].Platforms) != 0 {
		t.Errorf("Expected empty platforms for invalid JSON")
	}
}

func TestScenarioRepository_FindAll_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScenarioRepository(db)
	ctx := context.Background()

	// Insert scenario with invalid JSON
	_, _ = db.Exec(`
		INSERT INTO scenarios (id, name, description, phases, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "s-invalid", "Test", "Desc", "not json", "not json", time.Now(), time.Now())

	scenarios, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(scenarios) != 1 {
		t.Errorf("Expected 1 scenario, got %d", len(scenarios))
	}
}

func TestResultRepository_FindRecentExecutions_WithScore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewResultRepository(db)
	ctx := context.Background()

	// Create scenario first for foreign key constraint
	createTestScenario(t, db, "s1")

	// Insert execution with score (using correct schema columns)
	_, err := db.Exec(`
		INSERT INTO executions (id, scenario_id, status, started_at, completed_at, safe_mode,
		score_overall, score_blocked, score_detected, score_successful, score_total)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "exec-score", "s1", "completed", time.Now(), time.Now(), true, 0.85, 2, 3, 5, 10)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	executions, err := repo.FindRecentExecutions(ctx, 10)
	if err != nil {
		t.Fatalf("FindRecentExecutions failed: %v", err)
	}
	if len(executions) != 1 {
		t.Errorf("Expected 1 execution, got %d", len(executions))
	}
	if executions[0].Score.Overall != 0.85 {
		t.Errorf("Expected score 0.85, got %f", executions[0].Score.Overall)
	}
}

func TestResultRepository_FindExecutionsByScenario_WithResults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewResultRepository(db)
	ctx := context.Background()

	// Create scenario first for foreign key constraint
	createTestScenario(t, db, "scenario-1")

	// Insert multiple executions for same scenario
	for i := 0; i < 3; i++ {
		exec := &entity.Execution{
			ID:         "exec-" + string(rune('a'+i)),
			ScenarioID: "scenario-1",
			Status:     entity.ExecutionCompleted,
			StartedAt:  time.Now(),
			SafeMode:   true,
		}
		_ = repo.CreateExecution(ctx, exec)
	}

	executions, err := repo.FindExecutionsByScenario(ctx, "scenario-1")
	if err != nil {
		t.Fatalf("FindExecutionsByScenario failed: %v", err)
	}
	if len(executions) != 3 {
		t.Errorf("Expected 3 executions, got %d", len(executions))
	}
}

// Scenario ImportFromYAML tests
func TestScenarioRepository_ImportFromYAML(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScenarioRepository(db)
	ctx := context.Background()

	// Create a temporary YAML file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "scenarios.yaml")
	yamlContent := `
- id: "scenario-1"
  name: "Test Scenario 1"
  description: "A test scenario"
  phases:
    - name: "Phase 1"
      techniques:
        - "T1059"
        - "T1082"
  tags:
    - "test"
    - "discovery"
- id: "scenario-2"
  name: "Test Scenario 2"
  description: "Another test scenario"
  phases:
    - name: "Initial"
      techniques:
        - "T1016"
  tags:
    - "quick"
`
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write YAML file: %v", err)
	}

	err = repo.ImportFromYAML(ctx, yamlPath)
	if err != nil {
		t.Fatalf("ImportFromYAML failed: %v", err)
	}

	// Verify scenarios were imported
	scenarios, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(scenarios) != 2 {
		t.Errorf("Expected 2 scenarios, got %d", len(scenarios))
	}
}

func TestScenarioRepository_ImportFromYAML_FileNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScenarioRepository(db)
	ctx := context.Background()

	err := repo.ImportFromYAML(ctx, "/nonexistent/path/scenarios.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestScenarioRepository_ImportFromYAML_InvalidYAML(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScenarioRepository(db)
	ctx := context.Background()

	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "invalid.yaml")
	err := os.WriteFile(yamlPath, []byte("not: valid: yaml: ["), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	err = repo.ImportFromYAML(ctx, yamlPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestScenarioRepository_ImportFromYAML_Upsert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScenarioRepository(db)
	ctx := context.Background()

	// First import
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "scenarios.yaml")
	yamlContent := `
- id: "upsert-test"
  name: "Original Name"
  description: "Original description"
  phases:
    - name: "Phase 1"
      techniques:
        - "T1059"
  tags:
    - "original"
`
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write YAML: %v", err)
	}

	err = repo.ImportFromYAML(ctx, yamlPath)
	if err != nil {
		t.Fatalf("First import failed: %v", err)
	}

	// Verify original
	scenario, err := repo.FindByID(ctx, "upsert-test")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if scenario.Name != "Original Name" {
		t.Errorf("Expected 'Original Name', got '%s'", scenario.Name)
	}

	// Second import with updated data
	yamlContent2 := `
- id: "upsert-test"
  name: "Updated Name"
  description: "Updated description"
  phases:
    - name: "Phase 1"
      techniques:
        - "T1059"
        - "T1082"
  tags:
    - "updated"
`
	yamlPath2 := filepath.Join(tmpDir, "scenarios2.yaml")
	err = os.WriteFile(yamlPath2, []byte(yamlContent2), 0644)
	if err != nil {
		t.Fatalf("Failed to write YAML: %v", err)
	}

	err = repo.ImportFromYAML(ctx, yamlPath2)
	if err != nil {
		t.Fatalf("Second import failed: %v", err)
	}

	// Verify update
	scenario, err = repo.FindByID(ctx, "upsert-test")
	if err != nil {
		t.Fatalf("FindByID after update failed: %v", err)
	}
	if scenario.Name != "Updated Name" {
		t.Errorf("Expected 'Updated Name', got '%s'", scenario.Name)
	}

	// Should still be only 1 scenario
	scenarios, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(scenarios) != 1 {
		t.Errorf("Expected 1 scenario after upsert, got %d", len(scenarios))
	}
}

func TestScenarioRepository_ImportFromYAML_EmptyFile(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScenarioRepository(db)
	ctx := context.Background()

	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "empty.yaml")
	err := os.WriteFile(yamlPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Empty file should not cause error (just imports nothing)
	err = repo.ImportFromYAML(ctx, yamlPath)
	if err != nil {
		t.Fatalf("ImportFromYAML with empty file failed: %v", err)
	}

	scenarios, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(scenarios) != 0 {
		t.Errorf("Expected 0 scenarios from empty file, got %d", len(scenarios))
	}
}

func TestScenarioRepository_ImportFromYAML_WithTimestamps(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScenarioRepository(db)
	ctx := context.Background()

	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "scenarios.yaml")
	// Scenario without timestamps (should be filled in)
	yamlContent := `
- id: "no-timestamps"
  name: "No Timestamps"
  description: "Test"
  phases:
    - name: "P1"
      techniques: ["T1"]
  tags: []
`
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write YAML: %v", err)
	}

	err = repo.ImportFromYAML(ctx, yamlPath)
	if err != nil {
		t.Fatalf("ImportFromYAML failed: %v", err)
	}

	scenario, err := repo.FindByID(ctx, "no-timestamps")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if scenario.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	if scenario.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

// Result repository FindResultByID tests
func TestResultRepository_FindResultByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewResultRepository(db)
	ctx := context.Background()

	// Create foreign key dependencies
	createTestScenario(t, db, "s1")
	createTestExecution(t, db, "exec-1", "s1")
	createTestTechnique(t, db, "T1059")
	createTestAgent(t, db, "agent-1")

	now := time.Now()
	result := &entity.ExecutionResult{
		ID:          "find-result-test",
		ExecutionID: "exec-1",
		TechniqueID: "T1059",
		AgentPaw:    "agent-1",
		Status:      entity.StatusPending,
		StartedAt:   now,
	}
	err := repo.CreateResult(ctx, result)
	if err != nil {
		t.Fatalf("CreateResult failed: %v", err)
	}

	// Update result with output (CreateResult doesn't insert output)
	result.Status = entity.StatusSuccess
	result.Output = "test output"
	result.CompletedAt = &now
	err = repo.UpdateResult(ctx, result)
	if err != nil {
		t.Fatalf("UpdateResult failed: %v", err)
	}

	found, err := repo.FindResultByID(ctx, "find-result-test")
	if err != nil {
		t.Fatalf("FindResultByID failed: %v", err)
	}
	if found.Output != "test output" {
		t.Errorf("Expected output 'test output', got '%s'", found.Output)
	}
	if found.Status != entity.StatusSuccess {
		t.Errorf("Expected status 'success', got '%s'", found.Status)
	}
}

func TestResultRepository_FindResultByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewResultRepository(db)
	ctx := context.Background()

	_, err := repo.FindResultByID(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent result")
	}
}

// Technique repository upsert tests
func TestTechniqueRepository_ImportFromYAML_Upsert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewTechniqueRepository(db)
	ctx := context.Background()

	// First import
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "techniques.yaml")
	yamlContent := `
- id: "T-UPSERT"
  name: "Original Technique"
  description: "Original description"
  tactic: "execution"
  platforms:
    - "linux"
  executors:
    - type: "sh"
      command: "echo original"
  is_safe: true
`
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write YAML: %v", err)
	}

	err = repo.ImportFromYAML(ctx, yamlPath)
	if err != nil {
		t.Fatalf("First import failed: %v", err)
	}

	// Verify original
	tech, err := repo.FindByID(ctx, "T-UPSERT")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if tech.Name != "Original Technique" {
		t.Errorf("Expected 'Original Technique', got '%s'", tech.Name)
	}

	// Second import with updated data
	yamlContent2 := `
- id: "T-UPSERT"
  name: "Updated Technique"
  description: "Updated description"
  tactic: "execution"
  platforms:
    - "linux"
    - "windows"
  executors:
    - type: "sh"
      command: "echo updated"
  is_safe: false
`
	yamlPath2 := filepath.Join(tmpDir, "techniques2.yaml")
	err = os.WriteFile(yamlPath2, []byte(yamlContent2), 0644)
	if err != nil {
		t.Fatalf("Failed to write YAML: %v", err)
	}

	err = repo.ImportFromYAML(ctx, yamlPath2)
	if err != nil {
		t.Fatalf("Second import failed: %v", err)
	}

	// Verify update
	tech, err = repo.FindByID(ctx, "T-UPSERT")
	if err != nil {
		t.Fatalf("FindByID after update failed: %v", err)
	}
	if tech.Name != "Updated Technique" {
		t.Errorf("Expected 'Updated Technique', got '%s'", tech.Name)
	}
	if tech.IsSafe {
		t.Error("Expected is_safe to be false after update")
	}
}

// User Repository tests
func TestNewUserRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	if repo == nil {
		t.Error("Expected non-nil repository")
	}
}

func TestUserRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &entity.User{
		ID:           "user-1",
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "$2a$10$hashedpassword",
		Role:         entity.RoleOperator,
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify timestamps are set
	if user.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	if user.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

func TestUserRepository_Create_WithExistingTimestamp(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	existingTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	user := &entity.User{
		ID:           "user-ts",
		Username:     "tsuser",
		Email:        "ts@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleViewer,
		CreatedAt:    existingTime,
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// CreatedAt should be preserved
	if !user.CreatedAt.Equal(existingTime) {
		t.Errorf("Expected CreatedAt to be preserved, got %v", user.CreatedAt)
	}
}

func TestUserRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &entity.User{
		ID:           "user-find-id",
		Username:     "finduser",
		Email:        "find@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleAdmin,
	}
	_ = repo.Create(ctx, user)

	found, err := repo.FindByID(ctx, "user-find-id")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.Username != "finduser" {
		t.Errorf("Expected username 'finduser', got '%s'", found.Username)
	}
	if found.Role != entity.RoleAdmin {
		t.Errorf("Expected role 'admin', got '%s'", found.Role)
	}
}

func TestUserRepository_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent user")
	}
}

func TestUserRepository_FindByUsername(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &entity.User{
		ID:           "user-find-username",
		Username:     "uniqueuser",
		Email:        "unique@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleOperator,
	}
	_ = repo.Create(ctx, user)

	found, err := repo.FindByUsername(ctx, "uniqueuser")
	if err != nil {
		t.Fatalf("FindByUsername failed: %v", err)
	}
	if found.ID != "user-find-username" {
		t.Errorf("Expected ID 'user-find-username', got '%s'", found.ID)
	}
	if found.Email != "unique@example.com" {
		t.Errorf("Expected email 'unique@example.com', got '%s'", found.Email)
	}
}

func TestUserRepository_FindByUsername_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	_, err := repo.FindByUsername(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent username")
	}
}

func TestUserRepository_FindByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &entity.User{
		ID:           "user-find-email",
		Username:     "emailuser",
		Email:        "specific@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleViewer,
	}
	_ = repo.Create(ctx, user)

	found, err := repo.FindByEmail(ctx, "specific@example.com")
	if err != nil {
		t.Fatalf("FindByEmail failed: %v", err)
	}
	if found.Username != "emailuser" {
		t.Errorf("Expected username 'emailuser', got '%s'", found.Username)
	}
}

func TestUserRepository_FindByEmail_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	_, err := repo.FindByEmail(ctx, "nonexistent@example.com")
	if err == nil {
		t.Error("Expected error for nonexistent email")
	}
}

func TestUserRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &entity.User{
		ID:           "user-update",
		Username:     "oldusername",
		Email:        "old@example.com",
		PasswordHash: "$2a$10$oldhash",
		Role:         entity.RoleViewer,
	}
	_ = repo.Create(ctx, user)

	// Update the user
	user.Username = "newusername"
	user.Email = "new@example.com"
	user.Role = entity.RoleOperator
	err := repo.Update(ctx, user)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	found, _ := repo.FindByID(ctx, "user-update")
	if found.Username != "newusername" {
		t.Errorf("Expected username 'newusername', got '%s'", found.Username)
	}
	if found.Email != "new@example.com" {
		t.Errorf("Expected email 'new@example.com', got '%s'", found.Email)
	}
	if found.Role != entity.RoleOperator {
		t.Errorf("Expected role 'operator', got '%s'", found.Role)
	}
}

func TestUserRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &entity.User{
		ID:           "user-delete",
		Username:     "deleteuser",
		Email:        "delete@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleViewer,
	}
	_ = repo.Create(ctx, user)

	err := repo.Delete(ctx, "user-delete")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = repo.FindByID(ctx, "user-delete")
	if err == nil {
		t.Error("Expected error after delete")
	}
}

func TestUserRepository_FindAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create multiple users
	for i := 0; i < 3; i++ {
		user := &entity.User{
			ID:           "user-all-" + string(rune('a'+i)),
			Username:     "user" + string(rune('a'+i)),
			Email:        "user" + string(rune('a'+i)) + "@example.com",
			PasswordHash: "$2a$10$hash",
			Role:         entity.RoleViewer,
		}
		_ = repo.Create(ctx, user)
	}

	users, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(users) != 3 {
		t.Errorf("Expected 3 users, got %d", len(users))
	}
}

func TestUserRepository_FindAll_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	users, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("Expected 0 users in empty DB, got %d", len(users))
	}
}

func TestUserRepository_Create_DuplicateUsername(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	user1 := &entity.User{
		ID:           "user-dup-1",
		Username:     "duplicate",
		Email:        "first@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleViewer,
	}
	_ = repo.Create(ctx, user1)

	user2 := &entity.User{
		ID:           "user-dup-2",
		Username:     "duplicate",
		Email:        "second@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleViewer,
	}

	err := repo.Create(ctx, user2)
	if err == nil {
		t.Error("Expected error for duplicate username")
	}
}

func TestUserRepository_Create_DuplicateEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	user1 := &entity.User{
		ID:           "user-dup-email-1",
		Username:     "first",
		Email:        "duplicate@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleViewer,
	}
	_ = repo.Create(ctx, user1)

	user2 := &entity.User{
		ID:           "user-dup-email-2",
		Username:     "second",
		Email:        "duplicate@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleViewer,
	}

	err := repo.Create(ctx, user2)
	if err == nil {
		t.Error("Expected error for duplicate email")
	}
}

func TestUserRepository_AllRoles(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	roles := []entity.UserRole{entity.RoleAdmin, entity.RoleOperator, entity.RoleViewer}

	for i, role := range roles {
		user := &entity.User{
			ID:           "user-role-" + string(rune('a'+i)),
			Username:     "roleuser" + string(rune('a'+i)),
			Email:        "role" + string(rune('a'+i)) + "@example.com",
			PasswordHash: "$2a$10$hash",
			Role:         role,
		}
		err := repo.Create(ctx, user)
		if err != nil {
			t.Fatalf("Create for role %s failed: %v", role, err)
		}

		found, err := repo.FindByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("FindByID for role %s failed: %v", role, err)
		}
		if found.Role != role {
			t.Errorf("Expected role %s, got %s", role, found.Role)
		}
	}
}

func TestUserRepository_FindActive(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create active user
	active := &entity.User{
		ID:           "user-active",
		Username:     "activeuser",
		Email:        "active@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleOperator,
		IsActive:     true,
	}
	_ = repo.Create(ctx, active)

	// Create inactive user using Deactivate after creation
	inactive := &entity.User{
		ID:           "user-inactive",
		Username:     "inactiveuser",
		Email:        "inactive@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleViewer,
		IsActive:     true, // Create as active first
	}
	_ = repo.Create(ctx, inactive)
	_ = repo.Deactivate(ctx, inactive.ID) // Then deactivate

	users, err := repo.FindActive(ctx)
	if err != nil {
		t.Fatalf("FindActive failed: %v", err)
	}

	// Should only have the active user
	found := false
	for _, u := range users {
		if u.ID == "user-active" {
			found = true
		}
		if u.ID == "user-inactive" {
			t.Error("Inactive user should not be in FindActive results")
		}
	}
	if !found {
		t.Error("Active user not found in FindActive results")
	}
}

func TestUserRepository_UpdateLastLogin(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &entity.User{
		ID:           "user-login",
		Username:     "loginuser",
		Email:        "login@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleOperator,
	}
	_ = repo.Create(ctx, user)

	err := repo.UpdateLastLogin(ctx, "user-login")
	if err != nil {
		t.Fatalf("UpdateLastLogin failed: %v", err)
	}

	found, _ := repo.FindByID(ctx, "user-login")
	if found.LastLoginAt == nil {
		t.Error("Expected LastLoginAt to be set")
	}
}

func TestUserRepository_Deactivate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &entity.User{
		ID:           "user-deactivate",
		Username:     "deactivateuser",
		Email:        "deactivate@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleOperator,
		IsActive:     true,
	}
	_ = repo.Create(ctx, user)

	err := repo.Deactivate(ctx, "user-deactivate")
	if err != nil {
		t.Fatalf("Deactivate failed: %v", err)
	}

	found, _ := repo.FindByID(ctx, "user-deactivate")
	if found.IsActive {
		t.Error("Expected user to be deactivated")
	}
}

func TestUserRepository_Reactivate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &entity.User{
		ID:           "user-reactivate",
		Username:     "reactivateuser",
		Email:        "reactivate@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleOperator,
		IsActive:     false,
	}
	_ = repo.Create(ctx, user)

	err := repo.Reactivate(ctx, "user-reactivate")
	if err != nil {
		t.Fatalf("Reactivate failed: %v", err)
	}

	found, _ := repo.FindByID(ctx, "user-reactivate")
	if !found.IsActive {
		t.Error("Expected user to be reactivated")
	}
}

func TestUserRepository_CountByRole(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create users with different roles
	for i := 0; i < 3; i++ {
		user := &entity.User{
			ID:           "admin-" + string(rune('a'+i)),
			Username:     "admin" + string(rune('a'+i)),
			Email:        "admin" + string(rune('a'+i)) + "@example.com",
			PasswordHash: "$2a$10$hash",
			Role:         entity.RoleAdmin,
			IsActive:     true,
		}
		_ = repo.Create(ctx, user)
	}

	operator := &entity.User{
		ID:           "operator-1",
		Username:     "operator1",
		Email:        "operator1@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleOperator,
		IsActive:     true,
	}
	_ = repo.Create(ctx, operator)

	count, err := repo.CountByRole(ctx, entity.RoleAdmin)
	if err != nil {
		t.Fatalf("CountByRole failed: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 admins, got %d", count)
	}

	count, err = repo.CountByRole(ctx, entity.RoleOperator)
	if err != nil {
		t.Fatalf("CountByRole failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 operator, got %d", count)
	}
}

func TestUserRepository_DeactivateAdminIfNotLast_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create two admin users
	admin1 := &entity.User{
		ID:           "admin-1",
		Username:     "admin1",
		Email:        "admin1@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleAdmin,
		IsActive:     true,
	}
	admin2 := &entity.User{
		ID:           "admin-2",
		Username:     "admin2",
		Email:        "admin2@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleAdmin,
		IsActive:     true,
	}
	_ = repo.Create(ctx, admin1)
	_ = repo.Create(ctx, admin2)

	// Should succeed because there's another admin
	err := repo.DeactivateAdminIfNotLast(ctx, "admin-1")
	if err != nil {
		t.Fatalf("DeactivateAdminIfNotLast failed: %v", err)
	}

	// Verify user is deactivated
	found, _ := repo.FindByID(ctx, "admin-1")
	if found.IsActive {
		t.Error("Expected user to be deactivated")
	}
}

func TestUserRepository_DeactivateAdminIfNotLast_LastAdmin(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create only one admin user
	admin := &entity.User{
		ID:           "admin-1",
		Username:     "admin1",
		Email:        "admin1@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleAdmin,
		IsActive:     true,
	}
	_ = repo.Create(ctx, admin)

	// Should fail because it's the last admin
	err := repo.DeactivateAdminIfNotLast(ctx, "admin-1")
	if err == nil {
		t.Error("Expected error when deactivating last admin")
	}
	if err != ErrLastAdmin {
		t.Errorf("Expected ErrLastAdmin, got %v", err)
	}

	// Verify user is still active
	found, _ := repo.FindByID(ctx, "admin-1")
	if !found.IsActive {
		t.Error("Expected user to still be active")
	}
}

func TestUserRepository_DeactivateAdminIfNotLast_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	err := repo.DeactivateAdminIfNotLast(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error when user not found")
	}
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestUserRepository_DeactivateAdminIfNotLast_NonAdmin(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create a non-admin user
	user := &entity.User{
		ID:           "user-1",
		Username:     "user1",
		Email:        "user1@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleOperator,
		IsActive:     true,
	}
	_ = repo.Create(ctx, user)

	// Should succeed for non-admin (no admin count check needed)
	err := repo.DeactivateAdminIfNotLast(ctx, "user-1")
	if err != nil {
		t.Fatalf("DeactivateAdminIfNotLast failed for non-admin: %v", err)
	}

	// Verify user is deactivated
	found, _ := repo.FindByID(ctx, "user-1")
	if found.IsActive {
		t.Error("Expected user to be deactivated")
	}
}

func TestUserRepository_DeactivateAdminIfNotLast_AlreadyInactive(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create two admin users, one inactive
	admin1 := &entity.User{
		ID:           "admin-1",
		Username:     "admin1",
		Email:        "admin1@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleAdmin,
		IsActive:     false, // Already inactive
	}
	admin2 := &entity.User{
		ID:           "admin-2",
		Username:     "admin2",
		Email:        "admin2@example.com",
		PasswordHash: "$2a$10$hash",
		Role:         entity.RoleAdmin,
		IsActive:     true,
	}
	_ = repo.Create(ctx, admin1)
	_ = repo.Create(ctx, admin2)

	// Deactivating already inactive user should succeed (idempotent)
	err := repo.DeactivateAdminIfNotLast(ctx, "admin-1")
	if err != nil {
		t.Fatalf("DeactivateAdminIfNotLast failed: %v", err)
	}
}

// Schedule Repository tests
func TestNewScheduleRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewScheduleRepository(db)
	if repo == nil {
		t.Error("Expected non-nil repository")
	}
}

func TestScheduleRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScheduleRepository(db)
	ctx := context.Background()

	// Create foreign key dependencies
	createTestScenario(t, db, "scenario-1")
	createTestUser(t, db, "user-1")

	now := time.Now()
	schedule := &entity.Schedule{
		ID:          "sched-1",
		Name:        "Test Schedule",
		Description: "A test schedule",
		ScenarioID:  "scenario-1",
		AgentPaw:    "agent-1",
		Frequency:   entity.FrequencyDaily,
		SafeMode:    true,
		Status:      entity.ScheduleStatusActive,
		CreatedBy:   "user-1",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := repo.Create(ctx, schedule)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
}

func TestScheduleRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScheduleRepository(db)
	ctx := context.Background()

	// Create foreign key dependencies
	createTestScenario(t, db, "scenario-1")
	createTestUser(t, db, "user-1")

	now := time.Now()
	schedule := &entity.Schedule{
		ID:          "sched-find",
		Name:        "Find Test",
		ScenarioID:  "scenario-1",
		Frequency:   entity.FrequencyHourly,
		Status:      entity.ScheduleStatusActive,
		CreatedBy:   "user-1",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = repo.Create(ctx, schedule)

	found, err := repo.FindByID(ctx, "sched-find")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found == nil {
		t.Fatal("FindByID returned nil")
	}
	if found.Name != "Find Test" {
		t.Errorf("Expected name 'Find Test', got '%s'", found.Name)
	}
}

func TestScheduleRepository_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScheduleRepository(db)
	ctx := context.Background()

	found, err := repo.FindByID(ctx, "nonexistent")
	if err == nil && found != nil {
		t.Error("Expected error or nil for nonexistent schedule")
	}
}

func TestScheduleRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScheduleRepository(db)
	ctx := context.Background()

	// Create foreign key dependencies
	createTestScenario(t, db, "scenario-1")
	createTestUser(t, db, "user-1")

	now := time.Now()
	schedule := &entity.Schedule{
		ID:          "sched-update",
		Name:        "Original Name",
		ScenarioID:  "scenario-1",
		Frequency:   entity.FrequencyDaily,
		Status:      entity.ScheduleStatusActive,
		CreatedBy:   "user-1",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = repo.Create(ctx, schedule)

	schedule.Name = "Updated Name"
	schedule.Frequency = entity.FrequencyHourly
	schedule.UpdatedAt = time.Now()

	err := repo.Update(ctx, schedule)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	found, _ := repo.FindByID(ctx, "sched-update")
	if found.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", found.Name)
	}
	if found.Frequency != entity.FrequencyHourly {
		t.Errorf("Expected frequency 'hourly', got '%s'", found.Frequency)
	}
}

func TestScheduleRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScheduleRepository(db)
	ctx := context.Background()

	now := time.Now()
	schedule := &entity.Schedule{
		ID:         "sched-delete",
		Name:       "Delete Test",
		ScenarioID: "scenario-1",
		Frequency:  entity.FrequencyDaily,
		Status:     entity.ScheduleStatusActive,
		CreatedBy:  "user-1",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_ = repo.Create(ctx, schedule)

	err := repo.Delete(ctx, "sched-delete")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	found, _ := repo.FindByID(ctx, "sched-delete")
	if found != nil {
		t.Error("Expected schedule to be deleted")
	}
}

func TestScheduleRepository_FindAll(t *testing.T) {
	db := setupTestDBWithFKData(t)
	defer db.Close()
	repo := NewScheduleRepository(db)
	ctx := context.Background()

	now := time.Now()
	for i := 0; i < 3; i++ {
		schedule := &entity.Schedule{
			ID:         "sched-all-" + string(rune('a'+i)),
			Name:       "Schedule " + string(rune('a'+i)),
			ScenarioID: "scenario-1",
			Frequency:  entity.FrequencyDaily,
			Status:     entity.ScheduleStatusActive,
			CreatedBy:  "user-1",
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		_ = repo.Create(ctx, schedule)
	}

	schedules, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(schedules) != 3 {
		t.Errorf("Expected 3 schedules, got %d", len(schedules))
	}
}

func TestScheduleRepository_FindByStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScheduleRepository(db)
	ctx := context.Background()

	// Create foreign key dependencies
	createTestScenario(t, db, "s1")
	createTestUser(t, db, testUserID)

	now := time.Now()
	active := &entity.Schedule{
		ID: "sched-active", Name: "Active", ScenarioID: "s1",
		Frequency: entity.FrequencyDaily, Status: entity.ScheduleStatusActive,
		CreatedBy: "user-1", CreatedAt: now, UpdatedAt: now,
	}
	paused := &entity.Schedule{
		ID: "sched-paused", Name: "Paused", ScenarioID: "s1",
		Frequency: entity.FrequencyDaily, Status: entity.ScheduleStatusPaused,
		CreatedBy: "user-1", CreatedAt: now, UpdatedAt: now,
	}
	_ = repo.Create(ctx, active)
	_ = repo.Create(ctx, paused)

	schedules, err := repo.FindByStatus(ctx, entity.ScheduleStatusActive)
	if err != nil {
		t.Fatalf("FindByStatus failed: %v", err)
	}
	if len(schedules) != 1 {
		t.Errorf("Expected 1 active schedule, got %d", len(schedules))
	}
}

func TestScheduleRepository_FindActiveSchedulesDue(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScheduleRepository(db)
	ctx := context.Background()

	// Create foreign key dependencies
	createTestScenario(t, db, "s1")
	createTestUser(t, db, testUserID)

	now := time.Now()
	pastTime := now.Add(-1 * time.Hour)
	futureTime := now.Add(1 * time.Hour)

	due := &entity.Schedule{
		ID: "sched-due", Name: "Due", ScenarioID: "s1",
		Frequency: entity.FrequencyDaily, Status: entity.ScheduleStatusActive,
		NextRunAt: &pastTime, CreatedBy: "user-1", CreatedAt: now, UpdatedAt: now,
	}
	notDue := &entity.Schedule{
		ID: "sched-not-due", Name: "Not Due", ScenarioID: "s1",
		Frequency: entity.FrequencyDaily, Status: entity.ScheduleStatusActive,
		NextRunAt: &futureTime, CreatedBy: "user-1", CreatedAt: now, UpdatedAt: now,
	}
	paused := &entity.Schedule{
		ID: "sched-paused-due", Name: "Paused Due", ScenarioID: "s1",
		Frequency: entity.FrequencyDaily, Status: entity.ScheduleStatusPaused,
		NextRunAt: &pastTime, CreatedBy: "user-1", CreatedAt: now, UpdatedAt: now,
	}
	_ = repo.Create(ctx, due)
	_ = repo.Create(ctx, notDue)
	_ = repo.Create(ctx, paused)

	schedules, err := repo.FindActiveSchedulesDue(ctx, now)
	if err != nil {
		t.Fatalf("FindActiveSchedulesDue failed: %v", err)
	}
	if len(schedules) != 1 {
		t.Errorf("Expected 1 due schedule, got %d", len(schedules))
	}
}

func TestScheduleRepository_FindByScenarioID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScheduleRepository(db)
	ctx := context.Background()

	// Create foreign key dependencies
	createTestScenario(t, db, "scenario-target")
	createTestScenario(t, db, "scenario-other")
	createTestUser(t, db, testUserID)

	now := time.Now()
	for i := 0; i < 2; i++ {
		schedule := &entity.Schedule{
			ID: "sched-scenario-" + string(rune('a'+i)), Name: "Schedule",
			ScenarioID: "scenario-target", Frequency: entity.FrequencyDaily,
			Status: entity.ScheduleStatusActive, CreatedBy: testUserID,
			CreatedAt: now, UpdatedAt: now,
		}
		_ = repo.Create(ctx, schedule)
	}
	other := &entity.Schedule{
		ID: "sched-other", Name: "Other", ScenarioID: "scenario-other",
		Frequency: entity.FrequencyDaily, Status: entity.ScheduleStatusActive,
		CreatedBy: testUserID, CreatedAt: now, UpdatedAt: now,
	}
	_ = repo.Create(ctx, other)

	schedules, err := repo.FindByScenarioID(ctx, "scenario-target")
	if err != nil {
		t.Fatalf("FindByScenarioID failed: %v", err)
	}
	if len(schedules) != 2 {
		t.Errorf("Expected 2 schedules, got %d", len(schedules))
	}
}

func TestScheduleRepository_CreateRun(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScheduleRepository(db)
	ctx := context.Background()

	// Create foreign key dependencies
	createTestScenario(t, db, "s1")
	createTestUser(t, db, testUserID)
	createTestExecution(t, db, testExecID, "s1")

	now := time.Now()
	schedule := &entity.Schedule{
		ID: "sched-run-test", Name: "Run Test", ScenarioID: "s1",
		Frequency: entity.FrequencyDaily, Status: entity.ScheduleStatusActive,
		CreatedBy: testUserID, CreatedAt: now, UpdatedAt: now,
	}
	_ = repo.Create(ctx, schedule)

	run := &entity.ScheduleRun{
		ID:          "run-1",
		ScheduleID:  "sched-run-test",
		ExecutionID: testExecID,
		StartedAt:   now,
		Status:      "running",
	}

	err := repo.CreateRun(ctx, run)
	if err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}
}

func TestScheduleRepository_CreateRun_NoExecutionID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScheduleRepository(db)
	ctx := context.Background()

	// Create foreign key dependencies
	createTestScenario(t, db, "s1")
	createTestUser(t, db, testUserID)

	now := time.Now()
	schedule := &entity.Schedule{
		ID: "sched-run-no-exec", Name: "Run No Exec", ScenarioID: "s1",
		Frequency: entity.FrequencyDaily, Status: entity.ScheduleStatusActive,
		CreatedBy: testUserID, CreatedAt: now, UpdatedAt: now,
	}
	_ = repo.Create(ctx, schedule)

	run := &entity.ScheduleRun{
		ID:          "run-no-exec",
		ScheduleID:  "sched-run-no-exec",
		ExecutionID: "", // No execution ID for failed runs
		StartedAt:   now,
		Status:      "failed",
		Error:       "connection error",
	}

	err := repo.CreateRun(ctx, run)
	if err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}
}

func TestScheduleRepository_UpdateRun(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScheduleRepository(db)
	ctx := context.Background()

	now := time.Now()
	schedule := &entity.Schedule{
		ID: "sched-update-run", Name: "Update Run", ScenarioID: "s1",
		Frequency: entity.FrequencyDaily, Status: entity.ScheduleStatusActive,
		CreatedBy: "user-1", CreatedAt: now, UpdatedAt: now,
	}
	_ = repo.Create(ctx, schedule)

	run := &entity.ScheduleRun{
		ID:          "run-update",
		ScheduleID:  "sched-update-run",
		ExecutionID: "exec-1",
		StartedAt:   now,
		Status:      "running",
	}
	_ = repo.CreateRun(ctx, run)

	completedAt := now.Add(5 * time.Minute)
	run.CompletedAt = &completedAt
	run.Status = "completed"

	err := repo.UpdateRun(ctx, run)
	if err != nil {
		t.Fatalf("UpdateRun failed: %v", err)
	}
}

func TestScheduleRepository_FindRunsByScheduleID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScheduleRepository(db)
	ctx := context.Background()

	// Create foreign key dependencies
	createTestScenario(t, db, "s1")
	createTestUser(t, db, testUserID)

	now := time.Now()
	schedule := &entity.Schedule{
		ID: "sched-find-runs", Name: "Find Runs", ScenarioID: "s1",
		Frequency: entity.FrequencyDaily, Status: entity.ScheduleStatusActive,
		CreatedBy: testUserID, CreatedAt: now, UpdatedAt: now,
	}
	_ = repo.Create(ctx, schedule)

	for i := 0; i < 3; i++ {
		run := &entity.ScheduleRun{
			ID:          "run-find-" + string(rune('a'+i)),
			ScheduleID:  "sched-find-runs",
			ExecutionID: "", // Empty execution ID for test
			StartedAt:   now.Add(time.Duration(i) * time.Hour),
			Status:      "completed",
		}
		_ = repo.CreateRun(ctx, run)
	}

	runs, err := repo.FindRunsByScheduleID(ctx, "sched-find-runs", 10)
	if err != nil {
		t.Fatalf("FindRunsByScheduleID failed: %v", err)
	}
	if len(runs) != 3 {
		t.Errorf("Expected 3 runs, got %d", len(runs))
	}
}

func TestScheduleRepository_FindRunsByScheduleID_WithLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScheduleRepository(db)
	ctx := context.Background()

	// Create foreign key dependencies
	createTestScenario(t, db, "s1")
	createTestUser(t, db, testUserID)

	now := time.Now()
	schedule := &entity.Schedule{
		ID: "sched-limit-runs", Name: "Limit Runs", ScenarioID: "s1",
		Frequency: entity.FrequencyDaily, Status: entity.ScheduleStatusActive,
		CreatedBy: testUserID, CreatedAt: now, UpdatedAt: now,
	}
	_ = repo.Create(ctx, schedule)

	for i := 0; i < 5; i++ {
		run := &entity.ScheduleRun{
			ID:          "run-limit-" + string(rune('a'+i)),
			ScheduleID:  "sched-limit-runs",
			ExecutionID: "", // Empty execution ID for test
			StartedAt:   now.Add(time.Duration(i) * time.Hour),
			Status:      "completed",
		}
		_ = repo.CreateRun(ctx, run)
	}

	runs, err := repo.FindRunsByScheduleID(ctx, "sched-limit-runs", 2)
	if err != nil {
		t.Fatalf("FindRunsByScheduleID failed: %v", err)
	}
	if len(runs) != 2 {
		t.Errorf("Expected 2 runs (limited), got %d", len(runs))
	}
}

func TestScheduleRepository_Delete_WithRuns(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewScheduleRepository(db)
	ctx := context.Background()

	now := time.Now()
	schedule := &entity.Schedule{
		ID: "sched-delete-runs", Name: "Delete Runs", ScenarioID: "s1",
		Frequency: entity.FrequencyDaily, Status: entity.ScheduleStatusActive,
		CreatedBy: "user-1", CreatedAt: now, UpdatedAt: now,
	}
	_ = repo.Create(ctx, schedule)

	// Add some runs
	for i := 0; i < 2; i++ {
		run := &entity.ScheduleRun{
			ID:          "run-delete-" + string(rune('a'+i)),
			ScheduleID:  "sched-delete-runs",
			ExecutionID: "exec-" + string(rune('a'+i)),
			StartedAt:   now,
			Status:      "completed",
		}
		_ = repo.CreateRun(ctx, run)
	}

	// Delete should cascade to runs
	err := repo.Delete(ctx, "sched-delete-runs")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	runs, _ := repo.FindRunsByScheduleID(ctx, "sched-delete-runs", 10)
	if len(runs) != 0 {
		t.Errorf("Expected 0 runs after delete, got %d", len(runs))
	}
}

// Notification Repository tests
func TestNewNotificationRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewNotificationRepository(db)
	if repo == nil {
		t.Error("Expected non-nil repository")
	}
}

func TestNotificationRepository_CreateSettings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "user-1")
	now := time.Now()
	settings := &entity.NotificationSettings{
		ID:                   "settings-1",
		UserID:               "user-1",
		Channel:              entity.ChannelEmail,
		Enabled:              true,
		EmailAddress:         "test@example.com",
		NotifyOnStart:        true,
		NotifyOnComplete:     true,
		NotifyOnFailure:      true,
		NotifyOnScoreAlert:   true,
		ScoreAlertThreshold:  70.0,
		NotifyOnAgentOffline: true,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	err := repo.CreateSettings(ctx, settings)
	if err != nil {
		t.Fatalf("CreateSettings failed: %v", err)
	}
}

func TestNotificationRepository_UpdateSettings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "user-1")
	now := time.Now()
	settings := &entity.NotificationSettings{
		ID:           "settings-update",
		UserID:       "user-1",
		Channel:      entity.ChannelEmail,
		Enabled:      true,
		EmailAddress: "old@example.com",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_ = repo.CreateSettings(ctx, settings)

	settings.EmailAddress = "new@example.com"
	settings.Enabled = false
	settings.UpdatedAt = time.Now()

	err := repo.UpdateSettings(ctx, settings)
	if err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}
}

func TestNotificationRepository_FindSettingsByUserID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "user-find")
	now := time.Now()
	settings := &entity.NotificationSettings{
		ID:           "settings-find",
		UserID:       "user-find",
		Channel:      entity.ChannelEmail,
		Enabled:      true,
		EmailAddress: "find@example.com",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_ = repo.CreateSettings(ctx, settings)

	found, err := repo.FindSettingsByUserID(ctx, "user-find")
	if err != nil {
		t.Fatalf("FindSettingsByUserID failed: %v", err)
	}
	if found.EmailAddress != "find@example.com" {
		t.Errorf("Expected email 'find@example.com', got '%s'", found.EmailAddress)
	}
}

func TestNotificationRepository_FindAllEnabledSettings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "user-1")
	createTestUser(t, db, "user-2")
	now := time.Now()
	enabled := &entity.NotificationSettings{
		ID: "settings-enabled", UserID: "user-1", Channel: entity.ChannelEmail,
		Enabled: true, EmailAddress: "enabled@example.com",
		CreatedAt: now, UpdatedAt: now,
	}
	disabled := &entity.NotificationSettings{
		ID: "settings-disabled", UserID: "user-2", Channel: entity.ChannelEmail,
		Enabled: false, EmailAddress: "disabled@example.com",
		CreatedAt: now, UpdatedAt: now,
	}
	_ = repo.CreateSettings(ctx, enabled)
	_ = repo.CreateSettings(ctx, disabled)

	settings, err := repo.FindAllEnabledSettings(ctx)
	if err != nil {
		t.Fatalf("FindAllEnabledSettings failed: %v", err)
	}
	if len(settings) != 1 {
		t.Errorf("Expected 1 enabled setting, got %d", len(settings))
	}
}

func TestNotificationRepository_DeleteSettings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "user-1")
	now := time.Now()
	settings := &entity.NotificationSettings{
		ID: "settings-delete", UserID: "user-1", Channel: entity.ChannelEmail,
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
	_ = repo.CreateSettings(ctx, settings)

	err := repo.DeleteSettings(ctx, "settings-delete")
	if err != nil {
		t.Fatalf("DeleteSettings failed: %v", err)
	}
}

func TestNotificationRepository_CreateNotification(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "user-1")
	notification := &entity.Notification{
		ID:        "notif-1",
		UserID:    "user-1",
		Type:      entity.NotificationExecutionStarted,
		Title:     "Test Notification",
		Message:   "This is a test notification",
		Data:      map[string]any{"key": "value"},
		Read:      false,
		CreatedAt: time.Now(),
	}

	err := repo.CreateNotification(ctx, notification)
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}
}

func TestNotificationRepository_FindNotificationByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "user-1")
	notification := &entity.Notification{
		ID:        "notif-find",
		UserID:    "user-1",
		Type:      entity.NotificationExecutionCompleted,
		Title:     "Find Test",
		Message:   "Test message",
		Data:      map[string]any{"score": 85.5},
		CreatedAt: time.Now(),
	}
	_ = repo.CreateNotification(ctx, notification)

	found, err := repo.FindNotificationByID(ctx, "notif-find")
	if err != nil {
		t.Fatalf("FindNotificationByID failed: %v", err)
	}
	if found.Title != "Find Test" {
		t.Errorf("Expected title 'Find Test', got '%s'", found.Title)
	}
	if found.Type != entity.NotificationExecutionCompleted {
		t.Errorf("Expected type 'execution_completed', got '%s'", found.Type)
	}
}

func TestNotificationRepository_FindNotificationsByUserID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "target-user")
	createTestUser(t, db, "other-user")
	for i := 0; i < 3; i++ {
		notification := &entity.Notification{
			ID:        "notif-user-" + string(rune('a'+i)),
			UserID:    "target-user",
			Type:      entity.NotificationExecutionStarted,
			Title:     "Notification " + string(rune('a'+i)),
			Message:   "Message",
			CreatedAt: time.Now().Add(time.Duration(i) * time.Hour),
		}
		_ = repo.CreateNotification(ctx, notification)
	}
	// Add notification for different user
	other := &entity.Notification{
		ID: "notif-other", UserID: "other-user",
		Type: entity.NotificationExecutionStarted, Title: "Other",
		Message: "Other", CreatedAt: time.Now(),
	}
	_ = repo.CreateNotification(ctx, other)

	notifications, err := repo.FindNotificationsByUserID(ctx, "target-user", 10)
	if err != nil {
		t.Fatalf("FindNotificationsByUserID failed: %v", err)
	}
	if len(notifications) != 3 {
		t.Errorf("Expected 3 notifications, got %d", len(notifications))
	}
}

func TestNotificationRepository_FindUnreadByUserID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "user-unread")
	unread := &entity.Notification{
		ID: "notif-unread", UserID: "user-unread",
		Type: entity.NotificationExecutionStarted, Title: "Unread",
		Message: "Unread", Read: false, CreatedAt: time.Now(),
	}
	read := &entity.Notification{
		ID: "notif-read", UserID: "user-unread",
		Type: entity.NotificationExecutionCompleted, Title: "Read",
		Message: "Read", Read: true, CreatedAt: time.Now(),
	}
	_ = repo.CreateNotification(ctx, unread)
	_ = repo.CreateNotification(ctx, read)

	notifications, err := repo.FindUnreadByUserID(ctx, "user-unread")
	if err != nil {
		t.Fatalf("FindUnreadByUserID failed: %v", err)
	}
	if len(notifications) != 1 {
		t.Errorf("Expected 1 unread notification, got %d", len(notifications))
	}
}

func TestNotificationRepository_MarkAsRead(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "user-1")
	notification := &entity.Notification{
		ID: "notif-mark-read", UserID: "user-1",
		Type: entity.NotificationExecutionStarted, Title: "Mark Read",
		Message: "Test", Read: false, CreatedAt: time.Now(),
	}
	_ = repo.CreateNotification(ctx, notification)

	err := repo.MarkAsRead(ctx, "notif-mark-read")
	if err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}

	found, _ := repo.FindNotificationByID(ctx, "notif-mark-read")
	if !found.Read {
		t.Error("Expected notification to be marked as read")
	}
}

func TestNotificationRepository_MarkAllAsRead(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "user-all-read")
	for i := 0; i < 3; i++ {
		notification := &entity.Notification{
			ID: "notif-all-read-" + string(rune('a'+i)), UserID: "user-all-read",
			Type: entity.NotificationExecutionStarted, Title: "Test",
			Message: "Test", Read: false, CreatedAt: time.Now(),
		}
		_ = repo.CreateNotification(ctx, notification)
	}

	err := repo.MarkAllAsRead(ctx, "user-all-read")
	if err != nil {
		t.Fatalf("MarkAllAsRead failed: %v", err)
	}

	unread, _ := repo.FindUnreadByUserID(ctx, "user-all-read")
	if len(unread) != 0 {
		t.Errorf("Expected 0 unread notifications, got %d", len(unread))
	}
}

func TestNotificationRepository_WithWebhookURL(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "user-webhook")
	now := time.Now()
	settings := &entity.NotificationSettings{
		ID:         "settings-webhook",
		UserID:     "user-webhook",
		Channel:    entity.ChannelWebhook,
		Enabled:    true,
		WebhookURL: "https://example.com/webhook",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_ = repo.CreateSettings(ctx, settings)

	found, err := repo.FindSettingsByUserID(ctx, "user-webhook")
	if err != nil {
		t.Fatalf("FindSettingsByUserID failed: %v", err)
	}
	if found.WebhookURL != "https://example.com/webhook" {
		t.Errorf("Expected webhook URL 'https://example.com/webhook', got '%s'", found.WebhookURL)
	}
}

func TestNotificationRepository_WithSentAt(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewNotificationRepository(db)
	ctx := context.Background()

	createTestUser(t, db, "user-1")
	sentAt := time.Now()
	notification := &entity.Notification{
		ID: "notif-sent", UserID: "user-1",
		Type: entity.NotificationExecutionCompleted, Title: "Sent",
		Message: "Test", SentAt: &sentAt, CreatedAt: time.Now(),
	}
	_ = repo.CreateNotification(ctx, notification)

	found, err := repo.FindNotificationByID(ctx, "notif-sent")
	if err != nil {
		t.Fatalf("FindNotificationByID failed: %v", err)
	}
	if found.SentAt == nil {
		t.Error("Expected SentAt to be set")
	}
}

// Result Repository additional tests for coverage
func TestResultRepository_FindExecutionsByDateRange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewResultRepository(db)
	ctx := context.Background()

	createTestScenario(t, db, "scenario-1")
	now := time.Now()
	start := now.Add(-24 * time.Hour)
	end := now.Add(24 * time.Hour)

	// Create executions within range
	for i := 0; i < 3; i++ {
		exec := &entity.Execution{
			ID:         "exec-range-" + string(rune('a'+i)),
			ScenarioID: "scenario-1",
			Status:     entity.ExecutionCompleted,
			StartedAt:  now,
			SafeMode:   true,
		}
		_ = repo.CreateExecution(ctx, exec)
	}

	// Create execution outside range
	oldExec := &entity.Execution{
		ID:         "exec-old",
		ScenarioID: "scenario-1",
		Status:     entity.ExecutionCompleted,
		StartedAt:  now.Add(-48 * time.Hour),
		SafeMode:   true,
	}
	_ = repo.CreateExecution(ctx, oldExec)

	executions, err := repo.FindExecutionsByDateRange(ctx, start, end)
	if err != nil {
		t.Fatalf("FindExecutionsByDateRange failed: %v", err)
	}
	if len(executions) != 3 {
		t.Errorf("Expected 3 executions in range, got %d", len(executions))
	}
}

func TestResultRepository_FindCompletedExecutionsByDateRange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewResultRepository(db)
	ctx := context.Background()

	createTestScenario(t, db, "scenario-1")
	now := time.Now()
	start := now.Add(-24 * time.Hour)
	end := now.Add(24 * time.Hour)

	// Create completed execution
	completed := &entity.Execution{
		ID:         "exec-completed",
		ScenarioID: "scenario-1",
		Status:     entity.ExecutionCompleted,
		StartedAt:  now,
	}
	_ = repo.CreateExecution(ctx, completed)

	// Create running execution
	running := &entity.Execution{
		ID:         "exec-running",
		ScenarioID: "scenario-1",
		Status:     entity.ExecutionRunning,
		StartedAt:  now,
	}
	_ = repo.CreateExecution(ctx, running)

	executions, err := repo.FindCompletedExecutionsByDateRange(ctx, start, end)
	if err != nil {
		t.Fatalf("FindCompletedExecutionsByDateRange failed: %v", err)
	}
	if len(executions) != 1 {
		t.Errorf("Expected 1 completed execution, got %d", len(executions))
	}
}

func TestResultRepository_UpdateExecution_NilScore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewResultRepository(db)
	ctx := context.Background()

	createTestScenario(t, db, "scenario-1")
	exec := &entity.Execution{
		ID:         "exec-nil-score",
		ScenarioID: "scenario-1",
		Status:     entity.ExecutionRunning,
		StartedAt:  time.Now(),
	}
	_ = repo.CreateExecution(ctx, exec)

	// Update without setting Score (nil)
	exec.Status = entity.ExecutionCompleted
	now := time.Now()
	exec.CompletedAt = &now

	err := repo.UpdateExecution(ctx, exec)
	if err != nil {
		t.Fatalf("UpdateExecution with nil score failed: %v", err)
	}
}
