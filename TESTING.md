# Testing Guide

## Prerequisites

- Go 1.24+ installed
- Network access for downloading dependencies (first time)

## Running Tests

### All Tests

```bash
# Download dependencies (first time only)
go mod download
go mod verify

# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with race detection
go test -race ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Specific Package Tests

```bash
# Security package (encryption, machine ID)
go test -v ./internal/security

# Database package
go test -v ./internal/database

# Config package
go test -v ./internal/config
```

### Benchmarks

```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific package benchmarks
go test -bench=. ./internal/security

# With memory allocation stats
go test -bench=. -benchmem ./...
```

## Test Coverage

### Phase 1 (Completed)

✅ **Security Package** (`internal/security`)
- Machine ID generation and caching
- Type 1 encryption (Config - AES-256-GCM with hard-coded key + machine ID salt)
- Type 2 encryption (Database - ChaCha20-Poly1305 with server key)
- Encryption separation verification
- Invalid input handling
- Base64 key conversion

✅ **Database Package** (`internal/database`)
- SQLite database creation and initialization
- Settings table CRUD operations
- Automatic encryption/decryption
- Data-at-rest encryption verification
- Transaction handling
- Invalid server key handling

### Phase 2 (Pending)

⏳ **Service Package** (`internal/service`)
- Windows/Linux service installation
- Service start/stop/restart
- Service status checking

⏳ **Server Package** (`internal/server`)
- HTTP server initialization
- Health check endpoint
- Status endpoint
- Service control endpoints
- API response format validation

## Test Structure

Each test file follows this pattern:

1. **Setup**: Helper functions to create test fixtures
2. **Positive Tests**: Normal operation scenarios
3. **Negative Tests**: Error handling and edge cases
4. **Benchmarks**: Performance measurements

## Running Tests in CI/CD

```bash
# Full test suite with coverage
make test

# With race detection
go test -race -coverprofile=coverage.out -covermode=atomic ./...

# Generate coverage report
go tool cover -html=coverage.out -o coverage.html
```

## Troubleshooting

### Network Issues

If you encounter network errors when downloading dependencies:

```bash
# Use Go proxy (default)
go env -w GOPROXY=https://proxy.golang.org,direct

# Or use direct mode (no proxy)
go env -w GOPROXY=direct

# Download all dependencies
go mod download
```

### Platform-Specific Tests

Some tests are platform-specific:

- **Windows**: Machine ID uses Windows registry
- **Linux**: Machine ID uses `/etc/machine-id` and `/proc/cpuinfo`

Tests will automatically adapt to the platform or skip if not applicable.

## Test Data

Test files use temporary directories that are automatically cleaned up:

- Database tests: `/tmp/posservice-test-*`
- Config tests: OS temp directory
- All test data is ephemeral and deleted after tests complete

## Expected Results

All tests should pass on both Windows and Linux platforms:

```
PASS: internal/security
PASS: internal/database
PASS: internal/config

Coverage: ~80%+ on critical paths
```

## Manual Testing

For integration testing with actual Windows service:

```bash
# Build the service
make build-windows   # or build-linux

# Install service (requires admin/sudo)
./build/pos-service.exe -install

# Start service
./build/pos-service.exe -start

# Check status
./build/pos-service.exe -status

# Stop service
./build/pos-service.exe -stop

# Uninstall service
./build/pos-service.exe -uninstall
```

## Performance Expectations

Benchmark results should show:

- **Config Encryption**: < 50 μs per operation
- **Database Encryption**: < 30 μs per operation
- **Machine ID Generation**: < 1ms (first call), < 1μs (cached)
- **Database Operations**: < 100 μs per setting read/write

Run benchmarks to verify:

```bash
go test -bench=. -benchmem ./internal/security
go test -bench=. -benchmem ./internal/database
```
