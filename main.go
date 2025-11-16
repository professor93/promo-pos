// cmd/service/main.go
// Windows POS Service - Main Entry Point
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kardianos/service"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	serviceName        = "POSService"
	serviceDisplayName = "POS Background Service"
	serviceDescription = "Secure offline-first POS synchronization service"
)

// Version information (set during build)
var (
	Version   = "1.0.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// Program implements the service.Interface
type Program struct {
	logger     *zap.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	httpServer *HTTPServer    // Your HTTP server implementation
	syncMgr    *SyncManager   // Your sync manager implementation
	database   *Database      // Your database implementation
	config     *Configuration // Your config implementation
}

func main() {
	// Parse command-line flags
	var (
		flagInstall   = flag.Bool("install", false, "Install service")
		flagUninstall = flag.Bool("uninstall", false, "Uninstall service")
		flagStart     = flag.Bool("start", false, "Start service")
		flagStop      = flag.Bool("stop", false, "Stop service")
		flagDebug     = flag.Bool("debug", false, "Run in debug mode (console)")
		flagVersion   = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	// Show version information
	if *flagVersion {
		fmt.Printf("POS Service v%s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		os.Exit(0)
	}

	// Initialize logger
	logger := initLogger(*flagDebug)
	defer logger.Sync()

	// Create program instance
	prg := &Program{
		logger: logger,
	}

	// Create service configuration
	svcConfig := &service.Config{
		Name:        serviceName,
		DisplayName: serviceDisplayName,
		Description: serviceDescription,
		Arguments:   []string{},
	}

	// Create service
	s, err := service.New(prg, svcConfig)
	if err != nil {
		logger.Fatal("Failed to create service", zap.Error(err))
	}

	// Handle service control commands
	if *flagInstall {
		err = s.Install()
		if err != nil {
			logger.Fatal("Failed to install service", zap.Error(err))
		}
		logger.Info("Service installed successfully")

		// Auto-start after installation
		err = s.Start()
		if err != nil {
			logger.Warn("Failed to start service after installation", zap.Error(err))
		} else {
			logger.Info("Service started successfully")
		}
		return
	}

	if *flagUninstall {
		err = s.Stop()
		if err != nil {
			logger.Warn("Failed to stop service before uninstall", zap.Error(err))
		}

		err = s.Uninstall()
		if err != nil {
			logger.Fatal("Failed to uninstall service", zap.Error(err))
		}
		logger.Info("Service uninstalled successfully")
		return
	}

	if *flagStart {
		err = s.Start()
		if err != nil {
			logger.Fatal("Failed to start service", zap.Error(err))
		}
		logger.Info("Service start command sent")
		return
	}

	if *flagStop {
		err = s.Stop()
		if err != nil {
			logger.Fatal("Failed to stop service", zap.Error(err))
		}
		logger.Info("Service stop command sent")
		return
	}

	// Check if service is already installed
	status, err := s.Status()
	if err == nil {
		// Service is installed, run as service
		if *flagDebug {
			// Run in console mode for debugging
			logger.Info("Running in debug mode")
			prg.runDebugMode()
		} else {
			// Run as Windows service
			err = s.Run()
			if err != nil {
				logger.Error("Service run failed", zap.Error(err))
			}
		}
	} else {
		// Service not installed, auto-install and start
		logger.Info("Service not found, installing...")

		err = s.Install()
		if err != nil {
			logger.Fatal("Failed to auto-install service", zap.Error(err))
		}

		logger.Info("Service installed, starting...")
		err = s.Start()
		if err != nil {
			logger.Fatal("Failed to auto-start service", zap.Error(err))
		}

		logger.Info("Service installed and started successfully")
		fmt.Println("POS Service has been installed and started as a Windows service.")
		fmt.Println("You can now close this window.")
	}
}

// Start implements service.Interface
func (p *Program) Start(s service.Service) error {
	p.logger.Info("Starting POS Service", zap.String("version", Version))

	// Create context for graceful shutdown
	p.ctx, p.cancel = context.WithCancel(context.Background())

	// Start service in goroutine
	go p.run()

	return nil
}

// Stop implements service.Interface
func (p *Program) Stop(s service.Service) error {
	p.logger.Info("Stopping POS Service")

	// Signal shutdown
	if p.cancel != nil {
		p.cancel()
	}

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop HTTP server
	if p.httpServer != nil {
		if err := p.httpServer.Shutdown(shutdownCtx); err != nil {
			p.logger.Error("HTTP server shutdown error", zap.Error(err))
		}
	}

	// Stop sync manager
	if p.syncMgr != nil {
		if err := p.syncMgr.Stop(shutdownCtx); err != nil {
			p.logger.Error("Sync manager shutdown error", zap.Error(err))
		}
	}

	// Close database
	if p.database != nil {
		if err := p.database.Close(); err != nil {
			p.logger.Error("Database close error", zap.Error(err))
		}
	}

	p.logger.Info("POS Service stopped")
	return nil
}

// run is the main service loop
func (p *Program) run() {
	p.logger.Info("POS Service running")

	// Initialize configuration
	config, err := LoadConfiguration(p.logger)
	if err != nil {
		p.logger.Fatal("Failed to load configuration", zap.Error(err))
		return
	}
	p.config = config

	// Initialize machine ID
	machineID, err := GetMachineID()
	if err != nil {
		p.logger.Fatal("Failed to get machine ID", zap.Error(err))
		return
	}
	p.logger.Info("Machine ID initialized", zap.String("machine_id", machineID))

	// Initialize database with encryption
	db, err := NewDatabase(p.logger, p.config, machineID)
	if err != nil {
		p.logger.Fatal("Failed to initialize database", zap.Error(err))
		return
	}
	p.database = db

	// Initialize sync manager
	syncMgr, err := NewSyncManager(p.logger, p.config, p.database)
	if err != nil {
		p.logger.Fatal("Failed to initialize sync manager", zap.Error(err))
		return
	}
	p.syncMgr = syncMgr

	// Start sync scheduler (every 59 seconds)
	go p.startSyncScheduler()

	// Initialize and start HTTP server
	httpServer, err := NewHTTPServer(p.logger, p.config, p.database)
	if err != nil {
		p.logger.Fatal("Failed to initialize HTTP server", zap.Error(err))
		return
	}
	p.httpServer = httpServer

	// Start HTTP server
	go func() {
		port := p.config.Port
		if port == 0 {
			port = 8080 // Default port
		}

		p.logger.Info("Starting HTTP server", zap.Int("port", port))
		if err := p.httpServer.Listen(fmt.Sprintf(":%d", port)); err != nil {
			p.logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	<-p.ctx.Done()
	p.logger.Info("Shutdown signal received")
}

// runDebugMode runs the service in console mode for debugging
func (p *Program) runDebugMode() {
	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start service
	p.Start(nil)

	// Wait for interrupt
	<-sigChan

	// Stop service
	p.Stop(nil)
}

// startSyncScheduler starts the 59-second sync timer
func (p *Program) startSyncScheduler() {
	p.logger.Info("Starting sync scheduler (59 second interval)")

	// Create ticker for 59 seconds
	ticker := time.NewTicker(59 * time.Second)
	defer ticker.Stop()

	// Perform initial sync
	p.performSync()

	for {
		select {
		case <-ticker.C:
			p.performSync()
		case <-p.ctx.Done():
			p.logger.Info("Sync scheduler stopped")
			return
		}
	}
}

// performSync executes a sync operation
func (p *Program) performSync() {
	p.logger.Debug("Starting sync operation")

	ctx, cancel := context.WithTimeout(p.ctx, 30*time.Second)
	defer cancel()

	if err := p.syncMgr.Sync(ctx); err != nil {
		p.logger.Error("Sync failed", zap.Error(err))

		// Check if offline for more than 24 hours
		if p.syncMgr.IsOfflineExpired() {
			p.logger.Error("Service offline for more than 24 hours")
			p.httpServer.SetOfflineMode(true)
		}
	} else {
		p.logger.Debug("Sync completed successfully")
		p.httpServer.SetOfflineMode(false)
	}
}

// initLogger initializes the zap logger
func initLogger(debug bool) *zap.Logger {
	// Create log directory
	logDir := filepath.Join(os.Getenv("ProgramData"), "POSService", "logs")
	os.MkdirAll(logDir, 0755)

	// Configure logger
	config := zap.NewProductionConfig()
	if debug {
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		config.Development = true
	} else {
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// Set output paths
	logFile := filepath.Join(logDir, fmt.Sprintf("pos-service-%s.log",
		time.Now().Format("2006-01-02")))
	config.OutputPaths = []string{logFile}
	if debug {
		config.OutputPaths = append(config.OutputPaths, "stdout")
	}

	// Configure encoding
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	return logger
}

// Placeholder structs - implement these in separate files
type HTTPServer struct {
	// Implementation in internal/server/server.go
}

func NewHTTPServer(logger *zap.Logger, config *Configuration, db *Database) (*HTTPServer, error) {
	// TODO: Implement
	return &HTTPServer{}, nil
}

func (h *HTTPServer) Listen(addr string) error {
	// TODO: Implement
	return nil
}

func (h *HTTPServer) Shutdown(ctx context.Context) error {
	// TODO: Implement
	return nil
}

func (h *HTTPServer) SetOfflineMode(offline bool) {
	// TODO: Implement
}

type SyncManager struct {
	// Implementation in internal/sync/sync.go
}

func NewSyncManager(logger *zap.Logger, config *Configuration, db *Database) (*SyncManager, error) {
	// TODO: Implement
	return &SyncManager{}, nil
}

func (s *SyncManager) Sync(ctx context.Context) error {
	// TODO: Implement
	return nil
}

func (s *SyncManager) Stop(ctx context.Context) error {
	// TODO: Implement
	return nil
}

func (s *SyncManager) IsOfflineExpired() bool {
	// TODO: Implement - check if last sync > 24 hours ago
	return false
}

type Database struct {
	// Implementation in internal/database/sqlite.go
}

func NewDatabase(logger *zap.Logger, config *Configuration, machineID string) (*Database, error) {
	// TODO: Implement
	return &Database{}, nil
}

func (d *Database) Close() error {
	// TODO: Implement
	return nil
}

type Configuration struct {
	Port            int
	SyncInterval    int
	MaxOfflineHours int
	ServerURL       string
	// Add more fields as needed
}

func LoadConfiguration(logger *zap.Logger) (*Configuration, error) {
	// TODO: Implement - load from encrypted config file
	return &Configuration{
		Port:            8080,
		SyncInterval:    59,
		MaxOfflineHours: 24,
	}, nil
}

func GetMachineID() (string, error) {
	// TODO: Implement - generate unique machine ID
	// Use Windows Product ID + MAC address + CPU serial
	return "PLACEHOLDER-MACHINE-ID", nil
}
