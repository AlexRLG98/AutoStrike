package rest

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"autostrike/internal/application"
	"autostrike/internal/domain/entity"
	"autostrike/internal/infrastructure/http/handlers"
	"autostrike/internal/infrastructure/http/middleware"
	"autostrike/internal/infrastructure/websocket"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Route path constants to avoid duplication
const (
	routeUserByID = "/users/:id"
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
	Agent        *application.AgentService
	Scenario     *application.ScenarioService
	Execution    *application.ExecutionService
	Technique    *application.TechniqueService
	Auth         *application.AuthService
	Analytics    *application.AnalyticsService
	Notification *application.NotificationService
	Schedule     *application.ScheduleService
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
		// Use NoAuth middleware to set default user context for handlers that check user_id
		api.Use(middleware.NoAuthMiddleware())
		logger.Warn("Authentication middleware DISABLED - set ENABLE_AUTH=true and JWT_SECRET in production")
	}

	// Register routes with permission middleware
	registerRoutesWithPermissions(api, services, hub, logger)

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

// registerRoutesWithPermissions registers all API routes with appropriate permission middleware
func registerRoutesWithPermissions(api *gin.RouterGroup, services *Services, hub *websocket.Hub, logger *zap.Logger) {
	// Helper to create permission middleware
	perm := middleware.PermissionMiddleware
	adminOnly := middleware.RoleMiddleware("admin")

	// Auth protected routes (GET /auth/me)
	if services.Auth != nil {
		authHandler := handlers.NewAuthHandler(services.Auth)
		authHandler.RegisterProtectedRoutes(api)

		// Admin routes (requires admin role)
		adminHandler := handlers.NewAdminHandler(services.Auth)
		admin := api.Group("/admin")
		admin.Use(adminOnly)
		{
			admin.GET("/users", adminHandler.ListUsers)
			admin.GET(routeUserByID, adminHandler.GetUser)
			admin.POST("/users", adminHandler.CreateUser)
			admin.PUT(routeUserByID, adminHandler.UpdateUser)
			admin.PUT(routeUserByID+"/role", adminHandler.UpdateUserRole)
			admin.DELETE(routeUserByID, adminHandler.DeactivateUser)
			admin.POST(routeUserByID+"/reactivate", adminHandler.ReactivateUser)
			admin.POST(routeUserByID+"/reset-password", adminHandler.ResetPassword)
		}
	}

	// Permission routes (all authenticated users can view)
	permissionHandler := handlers.NewPermissionHandler()
	permissions := api.Group("/permissions")
	{
		permissions.GET("/matrix", permissionHandler.GetPermissionMatrix)
		permissions.GET("/me", permissionHandler.GetMyPermissions)
	}

	// Agents - view for all, create/delete requires permission
	agentHandler := handlers.NewAgentHandler(services.Agent)
	agents := api.Group("/agents")
	{
		agents.GET("", perm(entity.PermissionAgentsView), agentHandler.ListAgents)
		agents.GET("/:paw", perm(entity.PermissionAgentsView), agentHandler.GetAgent)
		agents.POST("", perm(entity.PermissionAgentsCreate), agentHandler.RegisterAgent)
		agents.DELETE("/:paw", perm(entity.PermissionAgentsDelete), agentHandler.DeleteAgent)
		agents.POST("/:paw/heartbeat", perm(entity.PermissionAgentsView), agentHandler.Heartbeat)
	}

	// Techniques - view for all, import requires permission
	techniqueHandler := handlers.NewTechniqueHandler(services.Technique)
	techniques := api.Group("/techniques")
	{
		techniques.GET("", perm(entity.PermissionTechniquesView), techniqueHandler.ListTechniques)
		techniques.GET("/coverage", perm(entity.PermissionTechniquesView), techniqueHandler.GetCoverage)
		techniques.GET("/tactic/:tactic", perm(entity.PermissionTechniquesView), techniqueHandler.GetByTactic)
		techniques.GET("/platform/:platform", perm(entity.PermissionTechniquesView), techniqueHandler.GetByPlatform)
		techniques.GET("/:id", perm(entity.PermissionTechniquesView), techniqueHandler.GetTechnique)
		techniques.POST("/import", perm(entity.PermissionTechniquesImport), techniqueHandler.ImportTechniques)
	}

	// Executions - view for all, start/stop requires permission
	var executionHandler *handlers.ExecutionHandler
	if hub != nil {
		executionHandler = handlers.NewExecutionHandlerWithHub(services.Execution, hub)
	} else {
		executionHandler = handlers.NewExecutionHandler(services.Execution)
	}
	executions := api.Group("/executions")
	{
		executions.GET("", perm(entity.PermissionExecutionsView), executionHandler.ListExecutions)
		executions.GET("/:id", perm(entity.PermissionExecutionsView), executionHandler.GetExecution)
		executions.GET("/:id/results", perm(entity.PermissionExecutionsView), executionHandler.GetResults)
		executions.POST("", perm(entity.PermissionExecutionsStart), executionHandler.StartExecution)
		executions.POST("/:id/stop", perm(entity.PermissionExecutionsStop), executionHandler.StopExecution)
		executions.POST("/:id/complete", perm(entity.PermissionExecutionsView), executionHandler.CompleteExecution)
	}

	// Scenarios - view for all, create/edit/delete/import/export requires permission
	scenarioHandler := handlers.NewScenarioHandler(services.Scenario)
	scenarios := api.Group("/scenarios")
	{
		scenarios.GET("", perm(entity.PermissionScenariosView), scenarioHandler.ListScenarios)
		scenarios.GET("/tag/:tag", perm(entity.PermissionScenariosView), scenarioHandler.GetScenariosByTag)
		scenarios.GET("/export", perm(entity.PermissionScenariosExport), scenarioHandler.ExportScenarios)
		scenarios.GET("/:id", perm(entity.PermissionScenariosView), scenarioHandler.GetScenario)
		scenarios.GET("/:id/export", perm(entity.PermissionScenariosExport), scenarioHandler.ExportScenario)
		scenarios.POST("", perm(entity.PermissionScenariosCreate), scenarioHandler.CreateScenario)
		scenarios.POST("/import", perm(entity.PermissionScenariosImport), scenarioHandler.ImportScenarios)
		scenarios.PUT("/:id", perm(entity.PermissionScenariosEdit), scenarioHandler.UpdateScenario)
		scenarios.DELETE("/:id", perm(entity.PermissionScenariosDelete), scenarioHandler.DeleteScenario)
	}

	// Analytics - view/compare/export requires respective permissions
	if services.Analytics != nil {
		analyticsHandler := handlers.NewAnalyticsHandler(services.Analytics)
		analytics := api.Group("/analytics")
		{
			analytics.GET("/period", perm(entity.PermissionAnalyticsView), analyticsHandler.GetPeriodStats)
			analytics.GET("/comparison", perm(entity.PermissionAnalyticsCompare), analyticsHandler.CompareScores)
			analytics.GET("/trend", perm(entity.PermissionAnalyticsView), analyticsHandler.GetScoreTrend)
			analytics.GET("/summary", perm(entity.PermissionAnalyticsView), analyticsHandler.GetExecutionSummary)
		}
	}

	// Notifications - requires various permissions
	if services.Notification != nil {
		notificationHandler := handlers.NewNotificationHandler(services.Notification)
		notifications := api.Group("/notifications")
		{
			// User notifications - any authenticated user
			notifications.GET("", notificationHandler.GetNotifications)
			notifications.GET("/unread/count", notificationHandler.GetUnreadCount)
			notifications.POST("/:id/read", notificationHandler.MarkAsRead)
			notifications.POST("/read-all", notificationHandler.MarkAllAsRead)
			// Settings - user can manage their own
			notifications.GET("/settings", notificationHandler.GetSettings)
			notifications.POST("/settings", notificationHandler.CreateSettings)
			notifications.PUT("/settings/:id", notificationHandler.UpdateSettings)
			notifications.DELETE("/settings/:id", notificationHandler.DeleteSettings)
			// SMTP config - admin only
			notifications.GET("/smtp", adminOnly, notificationHandler.GetSMTPConfig)
			notifications.POST("/smtp/test", adminOnly, notificationHandler.TestSMTP)
		}
	}

	// Schedules - view for all, create/edit/delete requires permission
	if services.Schedule != nil {
		scheduleHandler := handlers.NewScheduleHandler(services.Schedule)
		schedules := api.Group("/schedules")
		{
			schedules.GET("", perm(entity.PermissionSchedulerView), scheduleHandler.GetAll)
			schedules.GET("/:id", perm(entity.PermissionSchedulerView), scheduleHandler.GetByID)
			schedules.GET("/:id/runs", perm(entity.PermissionSchedulerView), scheduleHandler.GetRuns)
			schedules.POST("", perm(entity.PermissionSchedulerCreate), scheduleHandler.Create)
			schedules.PUT("/:id", perm(entity.PermissionSchedulerEdit), scheduleHandler.Update)
			schedules.DELETE("/:id", perm(entity.PermissionSchedulerDelete), scheduleHandler.Delete)
			schedules.POST("/:id/pause", perm(entity.PermissionSchedulerEdit), scheduleHandler.Pause)
			schedules.POST("/:id/resume", perm(entity.PermissionSchedulerEdit), scheduleHandler.Resume)
			schedules.POST("/:id/run", perm(entity.PermissionExecutionsStart), scheduleHandler.RunNow)
		}
	}

	logger.Info("Routes registered with permission middleware")
}

// Run starts the HTTP server
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

// Router returns the underlying gin router for testing
func (s *Server) Router() *gin.Engine {
	return s.router
}
