package main

import (
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

	"autostrike/internal/application"
	"autostrike/internal/domain/service"
	"autostrike/internal/infrastructure/api/rest"
	"autostrike/internal/infrastructure/persistence/sqlite"
	"autostrike/internal/infrastructure/websocket"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	// Register SQLite3 driver for database/sql
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	// Load configuration
	if err := loadConfig(); err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Initialize database
	db, err := sql.Open("sqlite3", viper.GetString("database.path"))
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize schema
	if err := sqlite.InitSchema(db); err != nil {
		logger.Fatal("Failed to initialize schema", zap.Error(err))
	}

	// Initialize repositories
	agentRepo := sqlite.NewAgentRepository(db)
	scenarioRepo := sqlite.NewScenarioRepository(db)
	techniqueRepo := sqlite.NewTechniqueRepository(db)
	resultRepo := sqlite.NewResultRepository(db)

	// Initialize domain services
	validator := service.NewTechniqueValidator()
	orchestrator := service.NewAttackOrchestrator(agentRepo, techniqueRepo, validator, logger)
	calculator := service.NewScoreCalculator()

	// Initialize application services
	agentService := application.NewAgentService(agentRepo)
	scenarioService := application.NewScenarioService(scenarioRepo, techniqueRepo, validator)
	executionService := application.NewExecutionService(
		resultRepo,
		scenarioRepo,
		techniqueRepo,
		agentRepo,
		orchestrator,
		calculator,
	)
	techniqueService := application.NewTechniqueService(techniqueRepo)

	// Initialize WebSocket hub
	hub := websocket.NewHub(logger)
	go hub.Run()

	// Initialize HTTP server
	server := rest.NewServer(
		agentService,
		scenarioService,
		executionService,
		techniqueService,
		hub,
		logger,
	)

	// Start server
	go func() {
		addr := viper.GetString("server.address")
		logger.Info("Starting AutoStrike server", zap.String("address", addr))
		if err := server.Run(addr); err != nil {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
}

func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Defaults
	viper.SetDefault("server.address", ":8443")
	viper.SetDefault("database.path", "./data/autostrike.db")
	viper.SetDefault("agent.beacon_interval", 30)

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}
	return nil
}
