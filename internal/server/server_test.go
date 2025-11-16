package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/professor93/promo-pos/internal/api"
)

func TestNew(t *testing.T) {
	server := New(nil)
	if server == nil {
		t.Fatal("Server is nil")
	}

	if server.app == nil {
		t.Error("Fiber app is nil")
	}

	if server.port != 8080 {
		t.Errorf("Expected default port 8080, got %d", server.port)
	}
}

func TestNew_CustomConfig(t *testing.T) {
	cfg := &Config{
		Port:               9090,
		MaxConcurrentConns: 200,
		ReadTimeout:        60 * time.Second,
	}

	server := New(cfg)
	if server.port != 9090 {
		t.Errorf("Expected port 9090, got %d", server.port)
	}

	if server.config.MaxConcurrentConns != 200 {
		t.Errorf("Expected max conns 200, got %d", server.config.MaxConcurrentConns)
	}
}

func TestHealthEndpoint(t *testing.T) {
	server := New(nil)
	app := server.GetApp()

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Parse response
	var apiResp api.APIResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &apiResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify APIResponse structure
	if !apiResp.OK {
		t.Error("Expected OK to be true")
	}

	if apiResp.Code != api.CodeSuccess {
		t.Errorf("Expected code %d, got %d", api.CodeSuccess, apiResp.Code)
	}

	if apiResp.Message == "" {
		t.Error("Message is empty")
	}

	// Verify health check data
	if apiResp.Result == nil {
		t.Error("Result is nil")
	}
}

func TestStatusEndpoint(t *testing.T) {
	server := New(nil)
	app := server.GetApp()

	req := httptest.NewRequest("GET", "/status", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Parse response
	var apiResp api.APIResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &apiResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify APIResponse structure
	if !apiResp.OK {
		t.Error("Expected OK to be true")
	}

	if apiResp.Code <= 0 {
		t.Errorf("Expected positive code, got %d", apiResp.Code)
	}
}

func TestConfigEndpoint(t *testing.T) {
	server := New(nil)
	app := server.GetApp()

	req := httptest.NewRequest("GET", "/config", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	var apiResp api.APIResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &apiResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify response
	if !apiResp.OK {
		t.Error("Expected OK to be true")
	}

	if apiResp.Code != api.CodeDataRetrieved {
		t.Errorf("Expected code %d, got %d", api.CodeDataRetrieved, apiResp.Code)
	}
}

func TestDataEndpoint(t *testing.T) {
	server := New(nil)
	app := server.GetApp()

	req := httptest.NewRequest("POST", "/data", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	var apiResp api.APIResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &apiResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify response
	if !apiResp.OK {
		t.Error("Expected OK to be true")
	}

	if apiResp.Code != api.CodeDataCreated {
		t.Errorf("Expected code %d, got %d", api.CodeDataCreated, apiResp.Code)
	}
}

func TestSyncEndpoint(t *testing.T) {
	server := New(nil)
	app := server.GetApp()

	req := httptest.NewRequest("POST", "/sync", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	var apiResp api.APIResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &apiResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify response
	if !apiResp.OK {
		t.Error("Expected OK to be true")
	}

	if apiResp.Code != api.CodeSyncSuccess {
		t.Errorf("Expected code %d, got %d", api.CodeSyncSuccess, apiResp.Code)
	}
}

func TestServiceStartEndpoint(t *testing.T) {
	server := New(nil)
	app := server.GetApp()

	req := httptest.NewRequest("POST", "/service/start", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	var apiResp api.APIResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &apiResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify response
	if !apiResp.OK {
		t.Error("Expected OK to be true")
	}

	if apiResp.Code != api.CodeServiceStarted {
		t.Errorf("Expected code %d, got %d", api.CodeServiceStarted, apiResp.Code)
	}
}

func TestServiceStopEndpoint(t *testing.T) {
	server := New(nil)
	app := server.GetApp()

	req := httptest.NewRequest("POST", "/service/stop", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	var apiResp api.APIResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &apiResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify response
	if !apiResp.OK {
		t.Error("Expected OK to be true")
	}

	if apiResp.Code != api.CodeServiceStopped {
		t.Errorf("Expected code %d, got %d", api.CodeServiceStopped, apiResp.Code)
	}
}

func TestServiceRestartEndpoint(t *testing.T) {
	server := New(nil)
	app := server.GetApp()

	req := httptest.NewRequest("POST", "/service/restart", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	var apiResp api.APIResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &apiResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify response
	if !apiResp.OK {
		t.Error("Expected OK to be true")
	}

	if apiResp.Code != api.CodeServiceRestarted {
		t.Errorf("Expected code %d, got %d", api.CodeServiceRestarted, apiResp.Code)
	}
}

func TestNotFoundEndpoint(t *testing.T) {
	server := New(nil)
	app := server.GetApp()

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should return 404
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	// Parse response
	var apiResp api.APIResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &apiResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify error response
	if apiResp.OK {
		t.Error("Expected OK to be false for 404")
	}

	if apiResp.Code >= 0 {
		t.Errorf("Expected negative error code, got %d", apiResp.Code)
	}
}

func TestAPIResponseFormat(t *testing.T) {
	server := New(nil)
	app := server.GetApp()

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/health"},
		{"GET", "/status"},
		{"GET", "/config"},
		{"POST", "/data"},
		{"POST", "/sync"},
		{"POST", "/service/start"},
		{"POST", "/service/stop"},
		{"POST", "/service/restart"},
	}

	for _, ep := range endpoints {
		t.Run(ep.path, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			// Parse response
			var apiResp api.APIResponse
			body, _ := io.ReadAll(resp.Body)
			if err := json.Unmarshal(body, &apiResp); err != nil {
				t.Fatalf("Failed to parse APIResponse: %v", err)
			}

			// Verify all required fields exist
			if apiResp.Message == "" {
				t.Error("Message field is empty")
			}

			// Success responses should have positive code
			if apiResp.OK && apiResp.Code <= 0 {
				t.Errorf("Success response has non-positive code: %d", apiResp.Code)
			}

			// Error responses should have negative code
			if !apiResp.OK && apiResp.Code >= 0 {
				t.Errorf("Error response has non-negative code: %d", apiResp.Code)
			}
		})
	}
}

func TestGracefulShutdown(t *testing.T) {
	cfg := &Config{
		Port:                  8888,
		DisableStartupMessage: true,
	}
	server := New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start server in background
	go func() {
		server.StartWithContext(ctx)
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context (triggers shutdown)
	cancel()

	// Wait for shutdown
	time.Sleep(200 * time.Millisecond)

	// Server should be shut down (this test verifies no panic occurs)
}

func BenchmarkHealthEndpoint(b *testing.B) {
	server := New(nil)
	app := server.GetApp()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		app.Test(req)
	}
}

func BenchmarkStatusEndpoint(b *testing.B) {
	server := New(nil)
	app := server.GetApp()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/status", nil)
		app.Test(req)
	}
}
