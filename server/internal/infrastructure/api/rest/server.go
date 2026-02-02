package rest

import (
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

// NewServer creates a new REST server with all routes configured
func NewServer(
	agentService *application.AgentService,
	scenarioService *application.ScenarioService,
	executionService *application.ExecutionService,
	techniqueService *application.TechniqueService,
	hub *websocket.Hub,
	logger *zap.Logger,
) *Server {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Global middleware
	router.Use(middleware.LoggingMiddleware(logger))
	router.Use(middleware.RecoveryMiddleware(logger))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// WebSocket routes
	if hub != nil {
		wsHandler := handlers.NewWebSocketHandler(hub, agentService, logger)
		wsHandler.RegisterRoutes(router)
	}

	// API v1 routes
	api := router.Group("/api/v1")
	{
		// Agents
		agentHandler := handlers.NewAgentHandler(agentService)
		agentHandler.RegisterRoutes(api)

		// Techniques
		techniqueHandler := handlers.NewTechniqueHandler(techniqueService)
		techniqueHandler.RegisterRoutes(api)

		// Executions
		executionHandler := handlers.NewExecutionHandler(executionService)
		executionHandler.RegisterRoutes(api)
	}

	return &Server{
		router: router,
		logger: logger,
	}
}

// Run starts the HTTP server
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

// Router returns the underlying gin router for testing
func (s *Server) Router() *gin.Engine {
	return s.router
}
