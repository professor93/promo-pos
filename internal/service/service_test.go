package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	cfg := &Config{
		Name:        "TestService",
		DisplayName: "Test Service",
		Description: "Test service description",
	}

	program, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if program == nil {
		t.Fatal("Program is nil")
	}

	if program.ctx == nil {
		t.Error("Context is nil")
	}

	if program.svc == nil {
		t.Error("Service is nil")
	}

	if program.logger == nil {
		t.Error("Logger is nil")
	}
}

func TestNew_DefaultConfig(t *testing.T) {
	cfg := &Config{}

	program, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create service with default config: %v", err)
	}

	if program == nil {
		t.Fatal("Program is nil")
	}

	// Verify defaults were applied
	if program.svc == nil {
		t.Error("Service not created with default config")
	}
}

func TestProgram_Lifecycle(t *testing.T) {
	startCalled := false
	stopCalled := false

	cfg := &Config{
		Name:        "TestLifecycleService",
		DisplayName: "Test Lifecycle Service",
		OnStart: func(ctx context.Context) error {
			startCalled = true
			return nil
		},
		OnStop: func() error {
			stopCalled = true
			return nil
		},
	}

	program, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Test Start
	err = program.Start(program.svc)
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	// Give time for start callback to execute
	time.Sleep(100 * time.Millisecond)

	if !startCalled {
		t.Error("OnStart callback was not called")
	}

	// Test Stop
	err = program.Stop(program.svc)
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	if !stopCalled {
		t.Error("OnStop callback was not called")
	}
}

func TestProgram_StartError(t *testing.T) {
	expectedErr := errors.New("start error")

	cfg := &Config{
		Name:        "TestErrorService",
		DisplayName: "Test Error Service",
		OnStart: func(ctx context.Context) error {
			return expectedErr
		},
	}

	program, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Start should not return error directly (runs in goroutine)
	// But the error should be logged
	err = program.Start(program.svc)
	if err != nil {
		t.Errorf("Unexpected error from Start: %v", err)
	}

	// Give time for error to be logged
	time.Sleep(100 * time.Millisecond)
}

func TestProgram_Context(t *testing.T) {
	cfg := &Config{
		Name: "TestContextService",
	}

	program, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := program.Context()
	if ctx == nil {
		t.Fatal("Context is nil")
	}

	// Verify context is not cancelled initially
	select {
	case <-ctx.Done():
		t.Error("Context is already cancelled")
	default:
		// Good
	}

	// Stop service (should cancel context)
	program.Stop(program.svc)

	// Verify context is cancelled after stop
	select {
	case <-ctx.Done():
		// Good
	case <-time.After(2 * time.Second):
		t.Error("Context was not cancelled after stop")
	}
}

func TestProgram_StatusString(t *testing.T) {
	cfg := &Config{
		Name: "TestStatusService",
	}

	program, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Get status string
	status, err := program.StatusString()
	if err != nil {
		t.Errorf("StatusString failed: %v", err)
	}

	// Status should be valid (unknown, running, or stopped)
	validStatuses := map[string]bool{
		"unknown": true,
		"running": true,
		"stopped": true,
	}

	if !validStatuses[status] {
		t.Logf("Note: Status is '%s' (may be platform-specific)", status)
	}
}

func TestProgram_IsInstalled(t *testing.T) {
	cfg := &Config{
		Name: "TestIsInstalledService",
	}

	program, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Check if installed (should work even if service management is not available)
	_ = program.IsInstalled()

	// IsInstalled should not panic
}

func TestProgram_IsRunning(t *testing.T) {
	cfg := &Config{
		Name: "TestIsRunningService",
	}

	program, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Check if running (should work even if service management is not available)
	_ = program.IsRunning()

	// IsRunning should not panic
}

func TestNewManager(t *testing.T) {
	cfg := &Config{
		Name:        "TestManagerService",
		DisplayName: "Test Manager Service",
	}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	if manager == nil {
		t.Fatal("Manager is nil")
	}

	if manager.program == nil {
		t.Error("Manager program is nil")
	}
}

func TestManager_GetProgram(t *testing.T) {
	cfg := &Config{
		Name: "TestGetProgramService",
	}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	program := manager.GetProgram()
	if program == nil {
		t.Error("GetProgram returned nil")
	}
}

func TestManager_GetStatus(t *testing.T) {
	cfg := &Config{
		Name: "TestGetStatusService",
	}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	status, isRunning, err := manager.GetStatus()
	if err != nil {
		t.Errorf("GetStatus failed: %v", err)
	}

	t.Logf("Service status: %s, running: %v", status, isRunning)

	// Status should be a valid string
	if status == "" {
		t.Error("Status string is empty")
	}
}

func TestProgram_Logger(t *testing.T) {
	cfg := &Config{
		Name: "TestLoggerService",
	}

	program, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	logger := program.Logger()
	if logger == nil {
		t.Error("Logger is nil")
	}

	// Test logging (should not panic)
	logger.Info("Test info message")
	logger.Warning("Test warning message")
	logger.Error("Test error message")
}

func TestProgram_StartStop_Concurrent(t *testing.T) {
	cfg := &Config{
		Name: "TestConcurrentService",
		OnStart: func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		},
	}

	program, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Start
	err = program.Start(program.svc)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Stop concurrently
	done := make(chan bool)
	go func() {
		program.Stop(program.svc)
		done <- true
	}()

	// Wait for stop to complete
	select {
	case <-done:
		// Good
	case <-time.After(5 * time.Second):
		t.Error("Stop did not complete within timeout")
	}
}

func BenchmarkNew(b *testing.B) {
	cfg := &Config{
		Name: "BenchService",
	}

	for i := 0; i < b.N; i++ {
		_, _ = New(cfg)
	}
}

func BenchmarkStart(b *testing.B) {
	cfg := &Config{
		Name: "BenchStartService",
		OnStart: func(ctx context.Context) error {
			return nil
		},
	}

	program, _ := New(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Start(program.svc)
	}
}

func BenchmarkStop(b *testing.B) {
	cfg := &Config{
		Name: "BenchStopService",
		OnStop: func() error {
			return nil
		},
	}

	program, _ := New(cfg)
	program.Start(program.svc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Stop(program.svc)
	}
}
