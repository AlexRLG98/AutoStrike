package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"autostrike/internal/application"
	"autostrike/internal/domain/entity"
	"autostrike/internal/domain/repository"
	"autostrike/internal/domain/service"
	"autostrike/internal/infrastructure/api/rest"
	"autostrike/internal/infrastructure/persistence/sqlite"
	"autostrike/internal/infrastructure/websocket"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	// Register SQLite3 driver for database/sql
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Load .env file (optional - won't fail if not found)
	_ = godotenv.Load()

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
	userRepo := sqlite.NewUserRepository(db)
	notificationRepo := sqlite.NewNotificationRepository(db)
	scheduleRepo := sqlite.NewScheduleRepository(db)

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
	analyticsService := application.NewAnalyticsService(resultRepo)

	// Initialize notification service with SMTP config from environment
	notificationService := initNotificationService(notificationRepo, userRepo, logger)

	// Initialize schedule service
	scheduleService := application.NewScheduleService(scheduleRepo, executionService, logger)

	// Initialize auth service (JWT secret from environment)
	jwtSecret := os.Getenv("JWT_SECRET")
	var authService *application.AuthService
	if jwtSecret != "" {
		authService = application.NewAuthService(userRepo, jwtSecret)
		// Ensure default admin user exists
		result, err := authService.EnsureDefaultAdmin(context.Background())
		if err != nil {
			logger.Warn("Failed to create default admin user", zap.Error(err))
		} else if result.Created {
			if result.GeneratedPassword != "" {
				// Log the generated password so the operator can use it
				logger.Info("=======================================================")
				logger.Info("Default admin user created with auto-generated password")
				logger.Info("Username: admin")
				logger.Info("Password: " + result.GeneratedPassword)
				logger.Info("IMPORTANT: Change this password immediately after first login!")
				logger.Info("Or set DEFAULT_ADMIN_PASSWORD env var before first startup.")
				logger.Info("=======================================================")
			} else {
				logger.Info("Default admin user created with password from DEFAULT_ADMIN_PASSWORD env var")
			}
		} else {
			logger.Debug("Default admin user already exists, skipping creation")
		}
	}

	// Auto-import techniques from configs directory at startup
	autoImportTechniques(techniqueService, logger)

	// Auto-import scenarios from configs directory at startup
	autoImportScenarios(scenarioService, logger)

	// Initialize WebSocket hub
	hub := websocket.NewHub(logger)

	// Set callback to mark agents offline when they disconnect
	hub.SetOnAgentDisconnect(func(paw string) {
		ctx := context.Background()
		if err := agentService.MarkAgentOffline(ctx, paw); err != nil {
			logger.Warn("Failed to mark agent offline", zap.String("paw", paw), zap.Error(err))
		} else {
			logger.Info("Agent marked offline", zap.String("paw", paw))
		}
	})

	go hub.Run()

	// Start background job to clean up stale agents (every 60 seconds)
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			ctx := context.Background()
			if err := agentService.CheckStaleAgents(ctx, 2*time.Minute); err != nil {
				logger.Warn("Failed to check stale agents", zap.Error(err))
			}
		}
	}()

	// Initialize HTTP server
	services := &rest.Services{
		Agent:        agentService,
		Scenario:     scenarioService,
		Execution:    executionService,
		Technique:    techniqueService,
		Auth:         authService,
		Analytics:    analyticsService,
		Notification: notificationService,
		Schedule:     scheduleService,
	}
	server := rest.NewServer(services, hub, logger)

	// Start the scheduler
	scheduleService.Start()

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

	// Stop the scheduler
	scheduleService.Stop()
}

func autoImportTechniques(service *application.TechniqueService, logger *zap.Logger) {
	// Import techniques from all YAML files in configs/techniques
	paths := []string{
		"./configs/techniques/discovery.yaml",
		"./configs/techniques/execution.yaml",
		"./configs/techniques/persistence.yaml",
		"./configs/techniques/defense-evasion.yaml",
	}

	imported := 0
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			if err := service.ImportTechniques(context.Background(), path); err != nil {
				logger.Warn("Failed to import techniques", zap.String("path", path), zap.Error(err))
			} else {
				imported++
				logger.Info("Imported techniques", zap.String("path", path))
			}
		}
	}

	if imported > 0 {
		logger.Info("Auto-imported technique files", zap.Int("count", imported))
	}
}

func autoImportScenarios(service *application.ScenarioService, logger *zap.Logger) {
	// Import scenarios from all YAML files in configs/scenarios
	paths := []string{
		"./configs/scenarios/default.yaml",
	}

	imported := 0
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			if err := service.ImportScenarios(context.Background(), path); err != nil {
				logger.Warn("Failed to import scenarios", zap.String("path", path), zap.Error(err))
			} else {
				imported++
				logger.Info("Imported scenarios", zap.String("path", path))
			}
		}
	}

	if imported > 0 {
		logger.Info("Auto-imported scenario files", zap.Int("count", imported))
	}
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

// initNotificationService initializes the notification service with SMTP config from environment
func initNotificationService(
	notificationRepo repository.NotificationRepository,
	userRepo repository.UserRepository,
	logger *zap.Logger,
) *application.NotificationService {
	// Get SMTP config from environment variables
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPortStr := os.Getenv("SMTP_PORT")
	smtpUsername := os.Getenv("SMTP_USERNAME")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	smtpFrom := os.Getenv("SMTP_FROM")
	smtpUseTLS := os.Getenv("SMTP_USE_TLS") == "true"
	dashboardURL := os.Getenv("DASHBOARD_URL")

	if dashboardURL == "" {
		dashboardURL = "https://localhost:8443"
	}

	var smtpConfig *entity.SMTPConfig
	if smtpHost != "" {
		smtpPort := 587 // Default SMTP port
		if smtpPortStr != "" {
			if p, err := strconv.Atoi(smtpPortStr); err == nil {
				smtpPort = p
			}
		}

		smtpConfig = &entity.SMTPConfig{
			Host:     smtpHost,
			Port:     smtpPort,
			Username: smtpUsername,
			Password: smtpPassword,
			From:     smtpFrom,
			UseTLS:   smtpUseTLS,
		}

		if smtpConfig.IsValid() {
			logger.Info("SMTP configuration loaded",
				zap.String("host", smtpHost),
				zap.Int("port", smtpPort),
				zap.String("from", smtpFrom),
				zap.Bool("use_tls", smtpUseTLS),
			)
		} else {
			logger.Warn("SMTP configuration incomplete - email notifications disabled")
			smtpConfig = nil
		}
	} else {
		logger.Info("SMTP not configured - email notifications disabled")
	}

	return application.NewNotificationService(notificationRepo, userRepo, smtpConfig, dashboardURL, logger)
}
