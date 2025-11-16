# CLAUDE.md - Windows POS Service Project Instructions

## Project Overview
Build a lightweight, secure Windows service application in Go that runs as a background service with offline-first architecture, encrypted data storage, and server synchronization capabilities.

## Core Requirements

### Architecture Specifications
- **Single Binary Application**: Compile to a single executable with all dependencies embedded
- **Windows Service**: Auto-install as Windows service on first run
- **HTTP Server**: Use Fiber v3.0.0+ (lightweight, fast) or Chi v5.0.12+ as fallback
- **Database**: SQLite with dual-key encryption (machine-unique key + server key)
- **Config Sync**: Every 59 seconds from server start time
- **Offline Mode**: Continue operation for 24 hours without server connection
- **GUI**: Login screen for initial setup with PIN authentication
- **System Tray**: Application runs in tray, doesn't exit on window close
- **Installer**: MSI installer with install/uninstall options

## Technology Stack (Use Latest Versions)

### Core Dependencies
```go
// go.mod dependencies
module pos-service

go 1.22

require (
    github.com/gofiber/fiber/v3 v3.0.0-beta.3      // HTTP server (lightweight)
    github.com/kardianos/service v1.2.2             // Windows service management
    modernc.org/sqlite v1.33.1                      // Pure Go SQLite (no CGO)
    github.com/knadh/koanf/v2 v2.1.2               // Configuration management
    go.uber.org/zap v1.27.0                        // Structured logging
    fyne.io/systray v1.11.0                        // System tray (requires CGO)
    github.com/wailsapp/wails/v2 v2.9.2            // GUI framework
    github.com/google/uuid v1.6.0                  // UUID generation
    golang.org/x/crypto v0.28.0                    // Encryption
    github.com/robfig/cron/v3 v3.0.1               // Cron scheduler
    github.com/spf13/cobra v1.8.1                  // CLI framework
)
```

## Project Structure

```
pos-service/
├── cmd/
│   ├── service/          # Main service entry point
│   │   └── main.go
│   ├── installer/        # MSI installer builder
│   │   └── main.go
│   └── gui/             # GUI application
│       └── main.go
├── internal/
│   ├── config/          # Configuration management
│   │   ├── config.go
│   │   ├── encryption.go
│   │   └── server.go
│   ├── database/        # Database layer
│   │   ├── sqlite.go
│   │   ├── encryption.go
│   │   ├── migrations/
│   │   └── models.go
│   ├── service/         # Windows service logic
│   │   ├── service.go
│   │   ├── installer.go
│   │   └── manager.go
│   ├── server/          # HTTP server
│   │   ├── server.go
│   │   ├── routes.go
│   │   └── middleware.go
│   ├── sync/            # Data synchronization
│   │   ├── sync.go
│   │   ├── queue.go
│   │   └── scheduler.go
│   ├── security/        # Security utilities
│   │   ├── machine_id.go
│   │   ├── encryption.go
│   │   └── token.go
│   ├── gui/            # GUI components
│   │   ├── login.go
│   │   ├── setup.go
│   │   └── tray.go
│   └── api/            # API client for server
│       ├── client.go
│       └── models.go
├── pkg/
│   ├── utils/          # Utility functions
│   └── constants/      # Application constants
├── build/
│   ├── windows/        # Windows-specific build files
│   │   ├── installer.wxs  # WiX installer configuration
│   │   ├── icon.ico
│   │   └── manifest.xml
│   └── scripts/        # Build scripts
├── configs/            # Default configurations
│   └── default.json
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Implementation Steps

### Phase 1: Core Service Foundation (Priority 1)

#### 1.1 Machine ID Generation
```go
// internal/security/machine_id.go
// Generate unique machine ID using:
// - Windows Product ID from registry
// - MAC address of primary network interface
// - CPU serial number
// - Combine using SHA256 hash
// Store in registry: HKLM\SOFTWARE\POSService\MachineID
```

#### 1.2 Database Setup with Encryption
```go
// internal/database/sqlite.go
// Features:
// - SQLite with WAL mode enabled
// - Dual-key encryption:
//   1. Machine key (derived from machine ID)
//   2. Server key (fetched from API)
// - Use SQLCipher or custom encryption layer
// - Database location: %PROGRAMDATA%\POSService\data.db
// 
// PRAGMA settings:
// PRAGMA journal_mode=WAL;
// PRAGMA busy_timeout=5000;
// PRAGMA cache_size=-64000;
// PRAGMA temp_store=MEMORY;
// PRAGMA synchronous=NORMAL;
```

#### 1.3 Configuration Management
```go
// internal/config/config.go
// Configuration structure:
type Config struct {
    ServerURL      string `json:"server_url"`
    StoreID        string `json:"store_id"`
    Port           int    `json:"port"`
    SyncInterval   int    `json:"sync_interval"` // seconds, default 59
    MaxOfflineHours int   `json:"max_offline_hours"` // default 24
    LogLevel       string `json:"log_level"`
    Encrypted      bool   `json:"encrypted"`
}

// Encryption: AES-256-GCM using machine ID as key
// Storage: %PROGRAMDATA%\POSService\config.enc
// Fallback: Keep last valid config for 24 hours
```

### Phase 2: Windows Service Implementation

#### 2.1 Service Installation
```go
// internal/service/installer.go
// Auto-install on first run:
// 1. Check if service exists
// 2. If not, install with:
//    - Name: "POSService"
//    - Display: "POS Background Service"
//    - Start Type: Automatic (Delayed Start)
//    - Recovery: Restart on failure
// 3. Set service account (LocalSystem or custom)
// 4. Start service after installation
```

#### 2.2 HTTP Server Setup
```go
// internal/server/server.go
// Use Fiber v3 with:
// - Dynamic port from config
// - Middleware: Logger, Recovery, CORS
// - Rate limiting: 100 req/min per IP
// - Graceful shutdown support
// - Memory optimization for 2-4GB systems

// Routes:
// GET  /health          - Health check
// GET  /status          - Service status
// POST /data           - Data endpoint
// GET  /config         - Get current config
// POST /sync           - Force sync
```

### Phase 3: Synchronization System

#### 3.1 Server Communication
```go
// internal/api/client.go
// API Client features:
// - Token-based authentication
// - Exponential backoff retry (1s, 2s, 4s...max 5min)
// - Circuit breaker pattern
// - Request timeout: 30 seconds
// - Connection pooling
```

#### 3.2 Sync Scheduler
```go
// internal/sync/scheduler.go
// Sync implementation:
// - Timer starts from application launch
// - Sync every 59 seconds exactly
// - Priority queue for failed syncs
// - Batch processing for efficiency
// - Conflict resolution: Server wins
// - Track last successful sync time
// - After 24 hours without sync: HTTP returns 503
```

### Phase 4: GUI Implementation

#### 4.1 Login Flow
```go
// internal/gui/login.go
// Login screens using Wails:
// Screen 1: Show Machine ID, Enter Store ID
// Screen 2: Confirm Machine ID + Store ID, Enter PIN
// Screen 3: Success/Error message
// 
// After successful login:
// - Store auth token securely
// - Minimize to system tray
// - Start background service
```

#### 4.2 System Tray
```go
// internal/gui/tray.go
// System tray features:
// - Icon with tooltip showing status
// - Right-click menu:
//   - Status (connected/offline)
//   - Last sync time
//   - Force sync now
//   - Open logs
//   - Settings
//   - About
//   - Exit (stops GUI only, not service)
```

### Phase 5: Installer Development

#### 5.1 MSI Installer
```xml
<!-- build/windows/installer.wxs -->
<!-- WiX configuration for:
  - Install/Uninstall options dialog
  - Service registration
  - Firewall rules
  - Registry entries
  - Start menu shortcuts
  - Desktop shortcut (optional)
  - Silent install support: msiexec /i pos-service.msi /quiet
-->
```

#### 5.2 Build Script
```makefile
# Makefile targets:
# make build-service    - Build service binary
# make build-gui       - Build GUI with embedded resources
# make build-installer - Create MSI installer
# make build-all      - Complete build pipeline
# make clean          - Clean build artifacts

# Build flags for small binary:
# -ldflags="-s -w"    - Strip debug info
# -tags timetzdata    - Embed timezone data
# upx --best         - Compress binary (optional)
```

### Phase 6: Security Implementation

#### 6.1 Encryption Keys
```go
// internal/security/encryption.go
// Key management:
// 1. Machine Key: PBKDF2(machineID, salt, 10000 iterations)
// 2. Server Key: Fetched via HTTPS, rotated monthly
// 3. Config Encryption: AES-256-GCM
// 4. Database Encryption: ChaCha20-Poly1305
// 
// Key storage:
// - Machine key: Derived on demand
// - Server key: Encrypted with machine key, stored locally
```

#### 6.2 Token Management
```go
// internal/security/token.go
// JWT token handling:
// - Store in Windows Credential Manager
// - Refresh before expiry
// - Fallback to re-authentication
```

## Critical Implementation Details

### 1. Binary Size Optimization
```go
// Compile flags for minimal size:
go build -ldflags="-s -w -X main.version=1.0.0" -tags timetzdata
// Use UPX compression (reduces 50-70%):
upx --best --lzma pos-service.exe
// Target: < 15MB compressed
```

### 2. Memory Management
```go
// For 2-4GB RAM systems:
// - Set GOGC=50 (more aggressive GC)
// - Limit SQLite cache: 64MB max
// - HTTP server: Max 100 concurrent connections
// - Use sync.Pool for frequently allocated objects
// - Profile with pprof regularly
```

### 3. Error Handling
```go
// Offline grace period logic:
type SyncStatus struct {
    LastSuccess    time.Time
    FailureCount   int
    IsOffline      bool
}

// If offline > 24 hours:
// - HTTP returns 503 Service Unavailable
// - Log critical error
// - Show system tray warning
// - Continue local operations
```

### 4. Logging Strategy
```go
// Use zap with:
// - Rotation: 10MB files, keep 7 days
// - Levels: Debug (dev), Info (production)
// - Location: %PROGRAMDATA%\POSService\logs\
// - Include: timestamp, level, caller, message, context
```

## Testing Requirements

### Unit Tests
- Machine ID generation consistency
- Encryption/decryption roundtrip
- Config parsing and validation
- Sync queue operations
- Database operations with encryption

### Integration Tests
- Service installation/uninstallation
- HTTP server endpoints
- Server communication with retries
- Offline mode transition
- GUI login flow

### System Tests
- Memory leak detection (run 24 hours)
- Sync reliability under network issues
- Service recovery after crash
- Installer on fresh Windows
- Upgrade scenarios

## Deployment Checklist

- [ ] Code signing certificate for exe/msi
- [ ] Version numbering (semantic versioning)
- [ ] Update mechanism design
- [ ] Rollback strategy
- [ ] Performance benchmarks documented
- [ ] Security audit completed
- [ ] Documentation for IT administrators
- [ ] Support troubleshooting guide

## Build Commands Sequence

```bash
# 1. Install dependencies
go mod download
go mod verify

# 2. Generate machine-specific files
go generate ./...

# 3. Run tests
go test -v -race ./...

# 4. Build service binary
GOOS=windows GOARCH=amd64 go build \
  -ldflags="-s -w -H windowsgui" \
  -o build/pos-service.exe \
  cmd/service/main.go

# 5. Build GUI binary
go build -tags desktop \
  -ldflags="-s -w -H windowsgui" \
  -o build/pos-gui.exe \
  cmd/gui/main.go

# 6. Compress binaries
upx --best --lzma build/*.exe

# 7. Build MSI installer
candle.exe build/windows/installer.wxs
light.exe -out pos-service-setup.msi installer.wixobj

# 8. Sign binaries and MSI
signtool sign /f cert.pfx /p password /t http://timestamp.digicert.com *.exe *.msi
```

## Performance Targets

- Service startup: < 3 seconds
- HTTP response time: p99 < 50ms
- Sync operation: < 5 seconds for 1000 records
- Memory usage: < 150MB idle, < 300MB active
- CPU usage: < 2% idle, < 10% during sync
- Binary size: < 15MB compressed
- Installer size: < 20MB

## Security Considerations

1. **Never log sensitive data** (tokens, keys, PINs)
2. **Use secure random** for all cryptographic operations
3. **Validate all inputs** from server and HTTP requests
4. **Implement rate limiting** on all endpoints
5. **Use prepared statements** for all SQL queries
6. **Rotate logs** to prevent disk fill
7. **Fail secure**: On any security error, deny access
8. **Audit trail**: Log all authentication attempts

## Notes for Claude Code

1. Start with Phase 1 (Core Foundation) before moving to other phases
2. Use the latest stable versions of all libraries (check pkg.go.dev)
3. Implement comprehensive error handling with proper logging
4. Add context.Context to all long-running operations
5. Use interfaces for testability (especially for external dependencies)
6. Document all exported functions and types
7. Include examples in documentation comments
8. Follow Go best practices and idioms
9. Keep functions small and focused (< 50 lines ideally)
10. Write tests alongside implementation

## Success Criteria

- [ ] Service runs continuously for 7+ days without memory leak
- [ ] Handles 1000+ sync operations without data loss
- [ ] Recovers automatically from all transient failures
- [ ] Installs successfully on Windows 10/11 (x64)
- [ ] Binary size under 15MB compressed
- [ ] Memory usage stable under 300MB
- [ ] All critical paths have 80%+ test coverage
- [ ] Documentation complete for operations team

---

**Important**: This is a production-critical system. Prioritize reliability and data integrity over performance optimization. Test thoroughly in offline scenarios. Implement graceful degradation for all external dependencies.