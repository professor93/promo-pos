package server

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/professor93/promo-pos/internal/api"
	"github.com/professor93/promo-pos/pkg/constants"
)

// Server represents the HTTP server
type Server struct {
	app    *fiber.App
	port   int
	config *Config
}

// Config holds server configuration
type Config struct {
	Port                int
	MaxConcurrentConns  int
	RateLimitPerMinute  int
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	IdleTimeout         time.Duration
	DisableStartupMessage bool
}

// DefaultConfig returns the default server configuration
func DefaultConfig() *Config {
	return &Config{
		Port:               constants.DefaultPort,
		MaxConcurrentConns: constants.DefaultMaxConcurrentConnections,
		RateLimitPerMinute: constants.DefaultRateLimitPerMinute,
		ReadTimeout:        30 * time.Second,
		WriteTimeout:       30 * time.Second,
		IdleTimeout:        120 * time.Second,
	}
}

// New creates a new HTTP server
func New(cfg *Config) *Server {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Create Fiber app with custom config
	app := fiber.New(fiber.Config{
		AppName:      constants.AppName,
		ServerHeader: constants.AppDisplayName,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
		Concurrency:  cfg.MaxConcurrentConns,
		DisableStartupMessage: cfg.DisableStartupMessage,
		// Custom error handler
		ErrorHandler: customErrorHandler,
	})

	// Add middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(cors.New())

	server := &Server{
		app:    app,
		port:   cfg.Port,
		config: cfg,
	}

	// Setup routes
	server.setupRoutes()

	return server
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.app.Get("/health", s.handleHealth)

	// Status endpoint
	s.app.Get("/status", s.handleStatus)

	// Config endpoint
	s.app.Get("/config", s.handleGetConfig)

	// Data endpoint
	s.app.Post("/data", s.handleData)

	// Sync endpoint
	s.app.Post("/sync", s.handleSync)

	// Service control endpoints
	s.app.Post("/service/start", s.handleServiceStart)
	s.app.Post("/service/stop", s.handleServiceStop)
	s.app.Post("/service/restart", s.handleServiceRestart)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	return s.app.Listen(addr)
}

// StartWithContext starts the server with graceful shutdown support
func (s *Server) StartWithContext(ctx context.Context) error {
	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- s.Start()
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		// Graceful shutdown
		return s.Shutdown()
	case err := <-errChan:
		return err
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	return s.app.Shutdown()
}

// ShutdownWithTimeout shuts down the server with timeout
func (s *Server) ShutdownWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return s.app.ShutdownWithContext(ctx)
}

// GetApp returns the underlying Fiber app
func (s *Server) GetApp() *fiber.App {
	return s.app
}

// customErrorHandler handles errors and returns standardized API responses
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := err.Error()

	// Check if it's a Fiber error
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// Map HTTP status to application code
	var appCode int
	switch code {
	case fiber.StatusBadRequest:
		appCode = api.CodeErrorBadRequest
	case fiber.StatusUnauthorized:
		appCode = api.CodeErrorUnauthorized
	case fiber.StatusForbidden:
		appCode = api.CodeErrorForbidden
	case fiber.StatusNotFound:
		appCode = api.CodeErrorNotFound
	default:
		appCode = api.CodeErrorInternal
	}

	// Return standardized error response
	return c.Status(code).JSON(api.NewErrorResponse(appCode, message))
}

// handleHealth handles health check requests
func (s *Server) handleHealth(c *fiber.Ctx) error {
	health := api.HealthCheck{
		Healthy:    true,
		Version:    "1.0.0", // TODO: Get from build info
		Timestamp:  time.Now().Format(time.RFC3339),
		DatabaseOK: true, // TODO: Check actual database
		ConfigOK:   true, // TODO: Check actual config
	}

	response := api.NewSuccessResponse(
		api.CodeSuccess,
		"Service is healthy",
		health,
	)

	return c.JSON(response)
}

// handleStatus handles status requests
func (s *Server) handleStatus(c *fiber.Ctx) error {
	status := api.ServiceStatus{
		Status:         "running",
		LastSyncTime:   time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
		OfflineHours:   0,
		IsHealthy:      true,
		WindowsService: "running",
	}

	response := api.NewSuccessResponse(
		api.CodeSuccess,
		"Status retrieved successfully",
		status,
	)

	return c.JSON(response)
}

// handleGetConfig handles config retrieval requests
func (s *Server) handleGetConfig(c *fiber.Ctx) error {
	config := map[string]interface{}{
		"port":         s.port,
		"max_conns":    s.config.MaxConcurrentConns,
		"read_timeout": s.config.ReadTimeout.String(),
	}

	response := api.NewSuccessResponse(
		api.CodeDataRetrieved,
		"Configuration retrieved successfully",
		config,
	)

	return c.JSON(response)
}

// handleData handles data endpoint requests
func (s *Server) handleData(c *fiber.Ctx) error {
	// TODO: Implement actual data handling

	response := api.NewSuccessResponse(
		api.CodeDataCreated,
		"Data processed successfully",
		map[string]string{
			"status": "processed",
		},
	)

	return c.JSON(response)
}

// handleSync handles sync requests
func (s *Server) handleSync(c *fiber.Ctx) error {
	// TODO: Implement actual sync logic

	response := api.NewSuccessResponse(
		api.CodeSyncSuccess,
		"Sync completed successfully",
		map[string]interface{}{
			"synced_at":    time.Now().Format(time.RFC3339),
			"records_synced": 0,
		},
	)

	return c.JSON(response)
}

// handleServiceStart handles service start requests
func (s *Server) handleServiceStart(c *fiber.Ctx) error {
	// TODO: Implement actual service start logic

	response := api.NewSuccessResponse(
		api.CodeServiceStarted,
		"Service started successfully",
		map[string]string{
			"status": "started",
		},
	)

	return c.JSON(response)
}

// handleServiceStop handles service stop requests
func (s *Server) handleServiceStop(c *fiber.Ctx) error {
	// TODO: Implement actual service stop logic

	response := api.NewSuccessResponse(
		api.CodeServiceStopped,
		"Service stopped successfully",
		map[string]string{
			"status": "stopped",
		},
	)

	return c.JSON(response)
}

// handleServiceRestart handles service restart requests
func (s *Server) handleServiceRestart(c *fiber.Ctx) error {
	// TODO: Implement actual service restart logic

	response := api.NewSuccessResponse(
		api.CodeServiceRestarted,
		"Service restarted successfully",
		map[string]string{
			"status": "restarted",
		},
	)

	return c.JSON(response)
}
