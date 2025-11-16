package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kardianos/service"
	"github.com/professor93/promo-pos/pkg/constants"
)

// Program implements the service.Interface from kardianos/service
type Program struct {
	ctx    context.Context
	cancel context.CancelFunc

	// Callbacks for lifecycle events
	onStart func(ctx context.Context) error
	onStop  func() error

	// Service instance
	svc service.Service

	// Logger
	logger service.Logger
}

// Config holds service configuration
type Config struct {
	Name        string
	DisplayName string
	Description string

	// Lifecycle callbacks
	OnStart func(ctx context.Context) error
	OnStop  func() error
}

// New creates a new service program
func New(cfg *Config) (*Program, error) {
	if cfg.Name == "" {
		cfg.Name = constants.WindowsServiceName
	}
	if cfg.DisplayName == "" {
		cfg.DisplayName = constants.WindowsServiceDisplayName
	}
	if cfg.Description == "" {
		cfg.Description = constants.WindowsServiceDescription
	}

	ctx, cancel := context.WithCancel(context.Background())

	p := &Program{
		ctx:     ctx,
		cancel:  cancel,
		onStart: cfg.OnStart,
		onStop:  cfg.OnStop,
	}

	// Create service configuration
	svcConfig := &service.Config{
		Name:        cfg.Name,
		DisplayName: cfg.DisplayName,
		Description: cfg.Description,
	}

	// Create service
	svc, err := service.New(p, svcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	p.svc = svc

	// Get logger
	logger, err := svc.Logger(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get logger: %w", err)
	}
	p.logger = logger

	return p, nil
}

// Start implements service.Interface
func (p *Program) Start(s service.Service) error {
	p.logger.Info("Service starting...")

	if p.onStart != nil {
		go func() {
			if err := p.onStart(p.ctx); err != nil {
				p.logger.Errorf("Service start callback failed: %v", err)
			}
		}()
	}

	p.logger.Info("Service started successfully")
	return nil
}

// Stop implements service.Interface
func (p *Program) Stop(s service.Service) error {
	p.logger.Info("Service stopping...")

	// Cancel context to signal shutdown
	p.cancel()

	if p.onStop != nil {
		if err := p.onStop(); err != nil {
			p.logger.Errorf("Service stop callback failed: %v", err)
		}
	}

	// Give time for graceful shutdown
	time.Sleep(1 * time.Second)

	p.logger.Info("Service stopped successfully")
	return nil
}

// Run starts the service and blocks until it's stopped
func (p *Program) Run() error {
	return p.svc.Run()
}

// Install installs the service
func (p *Program) Install() error {
	return p.svc.Install()
}

// Uninstall removes the service
func (p *Program) Uninstall() error {
	return p.svc.Uninstall()
}

// Start starts the service
func (p *Program) StartService() error {
	return p.svc.Start()
}

// Stop stops the service
func (p *Program) StopService() error {
	return p.svc.Stop()
}

// Restart restarts the service
func (p *Program) Restart() error {
	return p.svc.Restart()
}

// Status returns the current service status
func (p *Program) Status() (service.Status, error) {
	return p.svc.Status()
}

// StatusString returns a human-readable status string
func (p *Program) StatusString() (string, error) {
	status, err := p.Status()
	if err != nil {
		return "", err
	}

	switch status {
	case service.StatusUnknown:
		return "unknown", nil
	case service.StatusRunning:
		return "running", nil
	case service.StatusStopped:
		return "stopped", nil
	default:
		return fmt.Sprintf("status-%d", status), nil
	}
}

// IsInstalled checks if the service is installed
func (p *Program) IsInstalled() bool {
	status, err := p.Status()
	if err != nil {
		return false
	}
	return status != service.StatusUnknown
}

// IsRunning checks if the service is currently running
func (p *Program) IsRunning() bool {
	status, err := p.Status()
	if err != nil {
		return false
	}
	return status == service.StatusRunning
}

// Logger returns the service logger
func (p *Program) Logger() service.Logger {
	return p.logger
}

// Context returns the service context
func (p *Program) Context() context.Context {
	return p.ctx
}

// Manager provides high-level service management operations
type Manager struct {
	program *Program
}

// NewManager creates a new service manager
func NewManager(cfg *Config) (*Manager, error) {
	program, err := New(cfg)
	if err != nil {
		return nil, err
	}

	return &Manager{
		program: program,
	}, nil
}

// InstallAndStart installs and starts the service
func (m *Manager) InstallAndStart() error {
	// Check if already installed
	if m.program.IsInstalled() {
		log.Println("Service already installed")
	} else {
		// Install
		if err := m.program.Install(); err != nil {
			return fmt.Errorf("failed to install service: %w", err)
		}
		log.Println("Service installed successfully")
	}

	// Start if not running
	if !m.program.IsRunning() {
		if err := m.program.StartService(); err != nil {
			return fmt.Errorf("failed to start service: %w", err)
		}
		log.Println("Service started successfully")
	} else {
		log.Println("Service already running")
	}

	return nil
}

// StopAndUninstall stops and uninstalls the service
func (m *Manager) StopAndUninstall() error {
	// Stop if running
	if m.program.IsRunning() {
		if err := m.program.StopService(); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}
		log.Println("Service stopped successfully")

		// Wait for service to stop
		time.Sleep(2 * time.Second)
	}

	// Uninstall
	if m.program.IsInstalled() {
		if err := m.program.Uninstall(); err != nil {
			return fmt.Errorf("failed to uninstall service: %w", err)
		}
		log.Println("Service uninstalled successfully")
	} else {
		log.Println("Service not installed")
	}

	return nil
}

// GetStatus returns detailed service status
func (m *Manager) GetStatus() (string, bool, error) {
	statusStr, err := m.program.StatusString()
	if err != nil {
		return "", false, err
	}

	isRunning := m.program.IsRunning()
	return statusStr, isRunning, nil
}

// GetProgram returns the underlying program
func (m *Manager) GetProgram() *Program {
	return m.program
}
