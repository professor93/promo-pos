package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/professor93/promo-pos/internal/config"
	"github.com/professor93/promo-pos/internal/database"
	"github.com/professor93/promo-pos/internal/security"
	"github.com/professor93/promo-pos/internal/server"
	"github.com/professor93/promo-pos/internal/service"
	"github.com/professor93/promo-pos/pkg/constants"
)

var (
	version   = "1.0.0"
	buildTime = "unknown"
	gitCommit = "unknown"
)

// Application holds the main application state
type Application struct {
	machineID     string
	config        *config.Manager
	db            *database.DB
	httpServer    *server.Server
	serviceManager *service.Manager
}

func main() {
	// Parse command-line flags
	var (
		installFlag   = flag.Bool("install", false, "Install the service")
		uninstallFlag = flag.Bool("uninstall", false, "Uninstall the service")
		startFlag     = flag.Bool("start", false, "Start the service")
		stopFlag      = flag.Bool("stop", false, "Stop the service")
		restartFlag   = flag.Bool("restart", false, "Restart the service")
		statusFlag    = flag.Bool("status", false, "Show service status")
		versionFlag   = flag.Bool("version", false, "Show version information")
		debugFlag     = flag.Bool("debug", false, "Run in debug mode (foreground)")
	)
	flag.Parse()

	// Show version
	if *versionFlag {
		fmt.Printf("%s v%s\n", constants.AppName, version)
		fmt.Printf("Build Time: %s\n", buildTime)
		fmt.Printf("Git Commit: %s\n", gitCommit)
		os.Exit(0)
	}

	// Initialize application
	app, err := NewApplication()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Handle service management commands
	if *installFlag {
		if err := app.serviceManager.InstallAndStart(); err != nil {
			log.Fatalf("Failed to install service: %v", err)
		}
		fmt.Println("Service installed and started successfully")
		os.Exit(0)
	}

	if *uninstallFlag {
		if err := app.serviceManager.StopAndUninstall(); err != nil {
			log.Fatalf("Failed to uninstall service: %v", err)
		}
		fmt.Println("Service uninstalled successfully")
		os.Exit(0)
	}

	if *startFlag {
		if err := app.serviceManager.GetProgram().StartService(); err != nil {
			log.Fatalf("Failed to start service: %v", err)
		}
		fmt.Println("Service started successfully")
		os.Exit(0)
	}

	if *stopFlag {
		if err := app.serviceManager.GetProgram().StopService(); err != nil {
			log.Fatalf("Failed to stop service: %v", err)
		}
		fmt.Println("Service stopped successfully")
		os.Exit(0)
	}

	if *restartFlag {
		if err := app.serviceManager.GetProgram().Restart(); err != nil {
			log.Fatalf("Failed to restart service: %v", err)
		}
		fmt.Println("Service restarted successfully")
		os.Exit(0)
	}

	if *statusFlag {
		status, isRunning, err := app.serviceManager.GetStatus()
		if err != nil {
			log.Fatalf("Failed to get status: %v", err)
		}
		fmt.Printf("Service Status: %s\n", status)
		fmt.Printf("Running: %v\n", isRunning)
		os.Exit(0)
	}

	// Run in debug mode (foreground)
	if *debugFlag {
		fmt.Println("Running in debug mode...")
		if err := app.RunDebug(); err != nil {
			log.Fatalf("Debug mode failed: %v", err)
		}
		return
	}

	// Run as service
	program := app.serviceManager.GetProgram()
	if err := program.Run(); err != nil {
		log.Fatalf("Service failed: %v", err)
	}
}

// NewApplication creates and initializes the application
func NewApplication() (*Application, error) {
	app := &Application{}

	// Get machine ID
	machineID, err := security.GetMachineID()
	if err != nil {
		return nil, fmt.Errorf("failed to get machine ID: %w", err)
	}
	app.machineID = machineID
	log.Printf("Machine ID: %s", machineID)

	// Initialize config manager
	configMgr, err := config.NewManager(machineID)
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}
	app.config = configMgr

	// Load configuration
	cfg, err := configMgr.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	log.Printf("Configuration loaded (Port: %d)", cfg.Port)

	// Initialize database (with a dummy server key for now)
	// TODO: Fetch server key from API
	serverKey, err := security.GenerateServerKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate server key: %w", err)
	}

	db, err := database.New(&database.Config{
		ServerKey: serverKey,
		DataDir:   "",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	app.db = db
	log.Println("Database initialized")

	// Initialize HTTP server
	serverCfg := &server.Config{
		Port: cfg.Port,
	}
	httpServer := server.New(serverCfg)
	app.httpServer = httpServer
	log.Printf("HTTP server configured on port %d", cfg.Port)

	// Initialize service manager
	serviceMgr, err := service.NewManager(&service.Config{
		Name:        constants.WindowsServiceName,
		DisplayName: constants.WindowsServiceDisplayName,
		Description: constants.WindowsServiceDescription,
		OnStart:     app.OnServiceStart,
		OnStop:      app.OnServiceStop,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create service manager: %w", err)
	}
	app.serviceManager = serviceMgr

	return app, nil
}

// OnServiceStart is called when the service starts
func (app *Application) OnServiceStart(ctx context.Context) error {
	log.Println("Service starting...")

	// Start HTTP server in background
	go func() {
		if err := app.httpServer.StartWithContext(ctx); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	log.Println("HTTP server started")

	// TODO: Start sync scheduler
	// TODO: Initialize other background tasks

	// Keep running until context is cancelled
	<-ctx.Done()
	log.Println("Service start context cancelled")

	return nil
}

// OnServiceStop is called when the service stops
func (app *Application) OnServiceStop() error {
	log.Println("Service stopping...")

	// Shutdown HTTP server
	if app.httpServer != nil {
		log.Println("Shutting down HTTP server...")
		if err := app.httpServer.ShutdownWithTimeout(5 * 1000000000); err != nil { // 5 seconds
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}

	// Close database
	if app.db != nil {
		log.Println("Closing database...")
		if err := app.db.Close(); err != nil {
			log.Printf("Database close error: %v", err)
		}
	}

	log.Println("Service stopped")
	return nil
}

// RunDebug runs the application in debug mode (foreground)
func (app *Application) RunDebug() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start service
	go func() {
		if err := app.OnServiceStart(ctx); err != nil {
			log.Printf("Service start error: %v", err)
		}
	}()

	fmt.Printf("\n%s v%s running in debug mode\n", constants.AppName, version)

	currentCfg, err := app.config.Get()
	if err != nil {
		log.Printf("Warning: Could not get config: %v", err)
	} else {
		fmt.Printf("HTTP server: http://localhost:%d\n", currentCfg.Port)
	}

	fmt.Println("Press Ctrl+C to stop...")
	fmt.Println()

	// Wait for interrupt signal
	<-sigChan
	fmt.Println("\nReceived interrupt signal, shutting down...")

	// Cancel context to stop service
	cancel()

	// Stop service
	if err := app.OnServiceStop(); err != nil {
		return fmt.Errorf("service stop error: %w", err)
	}

	fmt.Println("Shutdown complete")
	return nil
}
