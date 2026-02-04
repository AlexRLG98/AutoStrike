package rest

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"autostrike/internal/application"
	"autostrike/internal/infrastructure/http/handlers"
	"autostrike/internal/infrastructure/http/middleware"
	"autostrike/internal/infrastructure/websocket"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server represents the HTTP REST server
type Server struct {
	router *gin.Engine
	logger *zap.Logger
}

// ServerConfig contains server configuration options
type ServerConfig struct {
	JWTSecret     string
	AgentSecret   string
	EnableAuth    bool
	DashboardPath string // Path to dashboard dist folder (empty = disabled)
}

// Services groups all application services for dependency injection
type Services struct {
	Agent     *application.AgentService
	Scenario  *application.ScenarioService
	Execution *application.ExecutionService
	Technique *application.TechniqueService
	Auth      *application.AuthService
}

// NewServerConfig creates a server config from environment variables
// Auth is automatically enabled when JWT_SECRET is set, disabled otherwise.
// This allows easy development (no secret = no auth) while being secure in production.
// Can be explicitly controlled with ENABLE_AUTH=true/false.
func NewServerConfig() *ServerConfig {
	jwtSecret := os.Getenv("JWT_SECRET")
	enableAuthEnv := os.Getenv("ENABLE_AUTH")

	// Default: auth enabled only if JWT_SECRET is provided
	enableAuth := jwtSecret != ""

	// Allow explicit override via ENABLE_AUTH
	if enableAuthEnv == "true" {
		enableAuth = true
	} else if enableAuthEnv == "false" {
		enableAuth = false
	}

	// Default dashboard path to ../dashboard/dist relative to working directory
	dashboardPath := os.Getenv("DASHBOARD_PATH")
	if dashboardPath == "" {
		dashboardPath = "../dashboard/dist"
	}

	return &ServerConfig{
		JWTSecret:     jwtSecret,
		AgentSecret:   os.Getenv("AGENT_SECRET"),
		EnableAuth:    enableAuth,
		DashboardPath: dashboardPath,
	}
}

// NewServer creates a new REST server with all routes configured
func NewServer(
	services *Services,
	hub *websocket.Hub,
	logger *zap.Logger,
) *Server {
	return NewServerWithConfig(services, hub, logger, NewServerConfig())
}

// NewServerWithConfig creates a new REST server with explicit configuration
func NewServerWithConfig(
	services *Services,
	hub *websocket.Hub,
	logger *zap.Logger,
	config *ServerConfig,
) *Server {
	// Fail fast if auth is explicitly enabled but JWT_SECRET is missing
	if config.EnableAuth && config.JWTSecret == "" {
		logger.Fatal("Configuration error: ENABLE_AUTH=true but JWT_SECRET is not set. " +
			"Either set JWT_SECRET or set ENABLE_AUTH=false for development mode.")
	}

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Global middleware
	router.Use(middleware.LoggingMiddleware(logger))
	router.Use(middleware.RecoveryMiddleware(logger))

	// Health check (always public) - includes auth status for frontend
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":       "ok",
			"auth_enabled": config.EnableAuth,
		})
	})

	// WebSocket routes (uses agent auth)
	if hub != nil {
		wsHandler := handlers.NewWebSocketHandler(hub, services.Agent, logger)
		wsHandler.SetExecutionService(services.Execution)
		wsHandler.RegisterRoutes(router)
	}

	// Auth routes (public - no auth middleware required)
	if services.Auth != nil {
		authHandler := handlers.NewAuthHandler(services.Auth)
		authHandler.RegisterRoutes(router)
	}

	// API v1 routes
	api := router.Group("/api/v1")

	// Apply authentication middleware if enabled
	if config.EnableAuth && config.JWTSecret != "" {
		authConfig := &middleware.AuthConfig{
			JWTSecret:   config.JWTSecret,
			AgentSecret: config.AgentSecret,
		}
		api.Use(middleware.AuthMiddleware(authConfig))
		logger.Info("Authentication middleware enabled for API routes")
	} else {
		logger.Warn("Authentication middleware DISABLED - set ENABLE_AUTH=true and JWT_SECRET in production")
	}

	{
		// Auth protected routes (GET /auth/me)
		if services.Auth != nil {
			authHandler := handlers.NewAuthHandler(services.Auth)
			authHandler.RegisterProtectedRoutes(api)
		}

		// Agents
		agentHandler := handlers.NewAgentHandler(services.Agent)
		agentHandler.RegisterRoutes(api)

		// Techniques
		techniqueHandler := handlers.NewTechniqueHandler(services.Technique)
		techniqueHandler.RegisterRoutes(api)

		// Executions (with WebSocket support for real-time notifications)
		var executionHandler *handlers.ExecutionHandler
		if hub != nil {
			executionHandler = handlers.NewExecutionHandlerWithHub(services.Execution, hub)
		} else {
			executionHandler = handlers.NewExecutionHandler(services.Execution)
		}
		executionHandler.RegisterRoutes(api)

		// Scenarios
		scenarioHandler := handlers.NewScenarioHandler(services.Scenario)
		scenarioHandler.RegisterRoutes(api)
	}

	// Serve dashboard static files if path is configured
	if config.DashboardPath != "" {
		setupDashboardRoutes(router, config.DashboardPath, logger)
	}

	return &Server{
		router: router,
		logger: logger,
	}
}

// setupDashboardRoutes configures static file serving for the dashboard SPA
func setupDashboardRoutes(router *gin.Engine, dashboardPath string, logger *zap.Logger) {
	absPath, err := filepath.Abs(dashboardPath)
	if err != nil {
		logger.Warn("Invalid dashboard path", zap.String("path", dashboardPath), zap.Error(err))
		return
	}

	indexFile := filepath.Join(absPath, "index.html")
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		logger.Warn("Dashboard index.html not found", zap.String("path", indexFile))
		return
	}

	// Serve static assets
	router.Static("/assets", filepath.Join(absPath, "assets"))

	// Serve other static files at root
	router.StaticFile("/vite.svg", filepath.Join(absPath, "vite.svg"))
	router.StaticFile("/favicon.ico", filepath.Join(absPath, "favicon.ico"))

	// SPA fallback: serve index.html for all other routes
	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		// Don't serve index.html for API or WebSocket routes
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ws/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})
			return
		}
		c.File(indexFile)
	})

	logger.Info("Dashboard serving enabled", zap.String("path", absPath))
}

// Run starts the HTTP server
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

// Router returns the underlying gin router for testing
func (s *Server) Router() *gin.Engine {
	return s.router
}
