# Makefile for Windows POS Service

# Variables
APP_NAME := pos-service
VERSION := 1.0.0
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO := go
GOFLAGS := -v
LDFLAGS := -s -w \
	-X main.Version=$(VERSION) \
	-X main.BuildTime=$(BUILD_TIME) \
	-X main.GitCommit=$(GIT_COMMIT)

# Directories
BUILD_DIR := build
DIST_DIR := dist
LOGS_DIR := logs
INTERNAL_DIR := internal
CMD_DIR := cmd

# Binary names
SERVICE_BIN := $(BUILD_DIR)/$(APP_NAME).exe
GUI_BIN := $(BUILD_DIR)/$(APP_NAME)-gui.exe
INSTALLER_MSI := $(DIST_DIR)/$(APP_NAME)-setup-$(VERSION).msi

# Go build tags
TAGS := timetzdata

.PHONY: all clean build test lint install-tools

# Default target
all: clean build

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod verify
	$(GO) mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	gofmt -s -w .

# Lint code
lint:
	@echo "Running linters..."
	golangci-lint run --timeout 5m ./...

# Security scan
security:
	@echo "Running security scan..."
	gosec -fmt json -out security-report.json ./...
	govulncheck ./...

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	$(GO) test -v -race ./...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

# Generate code (if needed)
generate:
	@echo "Generating code..."
	$(GO) generate ./...

# Build service binary (Windows)
build-service:
	@echo "Building service binary..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GO) build \
		$(GOFLAGS) \
		-tags $(TAGS) \
		-ldflags="$(LDFLAGS) -H windowsgui" \
		-o $(SERVICE_BIN) \
		$(CMD_DIR)/service/main.go

# Build GUI binary (Windows, requires CGO for systray)
build-gui:
	@echo "Building GUI binary..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 $(GO) build \
		$(GOFLAGS) \
		-tags desktop,$(TAGS) \
		-ldflags="$(LDFLAGS) -H windowsgui" \
		-o $(GUI_BIN) \
		$(CMD_DIR)/gui/main.go

# Build all binaries
build: build-service build-gui

# Compress binaries with UPX
compress:
	@echo "Compressing binaries..."
	@which upx > /dev/null 2>&1 || (echo "UPX not found. Install it first." && exit 1)
	upx --best --lzma $(SERVICE_BIN)
	upx --best --lzma $(GUI_BIN)

# Build installer (requires WiX Toolset)
build-installer:
	@echo "Building MSI installer..."
	@mkdir -p $(DIST_DIR)
	@which candle > /dev/null 2>&1 || (echo "WiX Toolset not found. Install it first." && exit 1)
	candle.exe -dVersion=$(VERSION) \
		-dProductCode="{GENERATE-NEW-GUID}" \
		-dUpgradeCode="{YOUR-FIXED-UPGRADE-GUID}" \
		-arch x64 \
		-out $(BUILD_DIR)/installer.wixobj \
		build/windows/installer.wxs
	light.exe -ext WixUIExtension \
		-cultures:en-US \
		-out $(INSTALLER_MSI) \
		$(BUILD_DIR)/installer.wixobj

# Sign binaries and installer (requires code signing certificate)
sign:
	@echo "Signing binaries..."
	@which signtool > /dev/null 2>&1 || (echo "SignTool not found. Install Windows SDK." && exit 1)
	signtool sign /f cert.pfx /p $(CERT_PASSWORD) \
		/t http://timestamp.digicert.com \
		/d "POS Service" \
		$(SERVICE_BIN) $(GUI_BIN)
	signtool sign /f cert.pfx /p $(CERT_PASSWORD) \
		/t http://timestamp.digicert.com \
		/d "POS Service Installer" \
		$(INSTALLER_MSI)

# Complete build pipeline
release: clean deps fmt lint test build compress build-installer sign
	@echo "Release build complete!"
	@echo "Installer: $(INSTALLER_MSI)"

# Development build (no compression, no signing)
dev: clean build
	@echo "Development build complete!"

# Run service in debug mode
run-debug:
	@echo "Running service in debug mode..."
	$(GO) run $(CMD_DIR)/service/main.go -debug

# Install service locally (requires admin)
install-local:
	@echo "Installing service locally..."
	$(SERVICE_BIN) -install

# Uninstall service locally (requires admin)
uninstall-local:
	@echo "Uninstalling service locally..."
	$(SERVICE_BIN) -uninstall

# Start service (requires admin)
start-service:
	@echo "Starting service..."
	$(SERVICE_BIN) -start

# Stop service (requires admin)
stop-service:
	@echo "Stopping service..."
	$(SERVICE_BIN) -stop

# View service logs
logs:
	@echo "Opening service logs..."
	@type %ProgramData%\POSService\logs\*.log 2>nul || echo "No logs found"

# Database migrations
migrate-up:
	@echo "Running database migrations..."
	goose -dir $(INTERNAL_DIR)/database/migrations sqlite3 ./data.db up

migrate-down:
	@echo "Rolling back database migration..."
	goose -dir $(INTERNAL_DIR)/database/migrations sqlite3 ./data.db down

migrate-status:
	@echo "Database migration status..."
	goose -dir $(INTERNAL_DIR)/database/migrations sqlite3 ./data.db status

# Generate API documentation
docs:
	@echo "Generating API documentation..."
	swag init -g $(CMD_DIR)/service/main.go -o ./docs

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR) coverage.* security-report.json
	@echo "Clean complete!"

# Docker build (for CI/CD)
docker-build:
	@echo "Building Docker image for CI/CD..."
	docker build -t $(APP_NAME):$(VERSION) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-f build/Dockerfile .

# Build for Linux (multiplatform support)
build-linux:
	@echo "Building Linux binary..."
	@mkdir -p $(BUILD_DIR)/linux
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build \
		$(GOFLAGS) \
		-tags $(TAGS) \
		-ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/linux/$(APP_NAME) \
		$(CMD_DIR)/service/main.go
	@echo "Linux binary: $(BUILD_DIR)/linux/$(APP_NAME)"

# Build for Windows (multiplatform support)
build-windows: build-service build-gui
	@echo "Windows binaries built!"

# Build all platforms
build-all: build-linux build-windows
	@echo "All platform binaries built!"
	@echo "  Linux:   $(BUILD_DIR)/linux/$(APP_NAME)"
	@echo "  Windows: $(BUILD_DIR)/$(APP_NAME).exe"
	@echo "  Windows GUI: $(BUILD_DIR)/$(APP_NAME)-gui.exe"

# Help target
help:
	@echo "Windows POS Service - Makefile Help"
	@echo "===================================="
	@echo ""
	@echo "Available targets:"
	@echo "  make all            - Clean and build everything"
	@echo "  make deps           - Download and verify dependencies"
	@echo "  make build          - Build service and GUI binaries"
	@echo "  make build-service  - Build service binary only"
	@echo "  make build-gui      - Build GUI binary only"
	@echo "  make compress       - Compress binaries with UPX"
	@echo "  make build-installer- Build MSI installer"
	@echo "  make sign           - Sign binaries and installer"
	@echo "  make release        - Complete release build"
	@echo "  make dev            - Development build (no compression)"
	@echo "  make test           - Run tests with coverage"
	@echo "  make test-verbose   - Run tests with verbose output"
	@echo "  make bench          - Run benchmarks"
	@echo "  make lint           - Run linters"
	@echo "  make security       - Run security scan"
	@echo "  make fmt            - Format code"
	@echo "  make run-debug      - Run service in debug mode"
	@echo "  make install-local  - Install service locally"
	@echo "  make uninstall-local- Uninstall service locally"
	@echo "  make start-service  - Start installed service"
	@echo "  make stop-service   - Stop running service"
	@echo "  make logs           - View service logs"
	@echo "  make migrate-up     - Run database migrations"
	@echo "  make migrate-down   - Rollback database migration"
	@echo "  make docs           - Generate API documentation"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make help           - Show this help message"
	@echo ""
	@echo "Environment variables:"
	@echo "  CERT_PASSWORD - Password for code signing certificate"
	@echo ""
	@echo "Requirements:"
	@echo "  - Go 1.22+"
	@echo "  - UPX (for compression)"
	@echo "  - WiX Toolset (for MSI)"
	@echo "  - SignTool (for signing)"
	@echo "  - MinGW-w64 (for CGO/systray)"

# Set default target
.DEFAULT_GOAL := help