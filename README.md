# POS Service - Lightweight Windows/Linux Background Service

A secure, offline-first POS (Point of Sale) background service written in Go with encrypted data storage and server synchronization.

## Features

### 🔐 Security
- **Dual Encryption Strategy**:
  - **Type 1 (Config)**: AES-256-GCM with hard-coded key + machine ID salt
  - **Type 2 (Database)**: ChaCha20-Poly1305 with server key only
- Machine-unique identification
- Data encryption at rest
- Secure key derivation (PBKDF2, 10000 iterations)

### 📡 Offline-First Architecture
- Continues operation for 24 hours without server connection
- Automatic sync every 59 seconds
- Graceful degradation when offline
- Local SQLite database with encrypted storage

### 🖥️ Cross-Platform Support
- **Windows**: Windows Service with registry storage
- **Linux**: systemd/init.d service with file-based storage
- Single binary deployment
- Platform-specific machine ID generation

### 🌐 HTTP API
- RESTful endpoints with standardized responses
- Health checks and status monitoring
- Service control (start/stop/restart)
- Configuration management
- Data synchronization

### 📊 Response Format
All API endpoints return standardized responses:
```json
{
  "ok": true,
  "code": 1,
  "message": "Success",
  "result": {...},
  "meta": {...}
}
```
- **Positive codes** (1-999): Success operations
- **Negative codes** (-1 to -999): Error conditions

## Quick Start

### Prerequisites
- Go 1.24+
- For Windows builds: MinGW-w64 (for CGO/systray support)
- For Linux builds: Standard Linux development tools

### Installation

```bash
# Clone the repository
git clone https://github.com/professor93/promo-pos.git
cd promo-pos

# Download dependencies
go mod download

# Build for current platform
make dev

# Or build for specific platforms
make build-linux    # Linux binary
make build-windows  # Windows binary
make build-all      # All platforms
```

### Running the Service

#### Development Mode (Foreground)
```bash
# Run in debug mode
./build/dev/pos-service -debug

# Or with go run
go run cmd/service/main.go -debug
```

#### Production Mode (Background Service)

**On Windows:**
```powershell
# Install and start service (requires Admin)
pos-service.exe -install

# Manage service
pos-service.exe -start
pos-service.exe -stop
pos-service.exe -restart
pos-service.exe -status

# Uninstall service
pos-service.exe -uninstall
```

**On Linux:**
```bash
# Install and start service (requires sudo)
sudo ./pos-service -install

# Manage service
sudo ./pos-service -start
sudo ./pos-service -stop
sudo ./pos-service -restart
./pos-service -status

# Uninstall service
sudo ./pos-service -uninstall
```

## API Endpoints

### Health & Status

#### GET /health
Health check endpoint
```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "ok": true,
  "code": 1,
  "message": "Service is healthy",
  "result": {
    "healthy": true,
    "version": "1.0.0",
    "timestamp": "2025-11-16T10:00:00Z",
    "database_ok": true,
    "config_ok": true
  }
}
```

#### GET /status
Service status
```bash
curl http://localhost:8080/status
```

Response:
```json
{
  "ok": true,
  "code": 1,
  "message": "Status retrieved successfully",
  "result": {
    "status": "running",
    "last_sync_time": "2025-11-16T09:55:00Z",
    "offline_hours": 0,
    "is_healthy": true,
    "windows_service": "running"
  }
}
```

### Configuration

#### GET /config
Get current configuration
```bash
curl http://localhost:8080/config
```

### Data Operations

#### POST /data
Process data
```bash
curl -X POST http://localhost:8080/data \
  -H "Content-Type: application/json" \
  -d '{"key":"value"}'
```

#### POST /sync
Force synchronization
```bash
curl -X POST http://localhost:8080/sync
```

### Service Control

#### POST /service/start
Start the service
```bash
curl -X POST http://localhost:8080/service/start
```

#### POST /service/stop
Stop the service
```bash
curl -X POST http://localhost:8080/service/stop
```

#### POST /service/restart
Restart the service
```bash
curl -X POST http://localhost:8080/service/restart
```

## Configuration

Configuration is stored in encrypted format at:
- **Windows**: `%PROGRAMDATA%\POSService\config.enc`
- **Linux**: `/var/lib/posservice/config.enc`

Default configuration:
```json
{
  "server_url": "",
  "store_id": "",
  "port": 8080,
  "sync_interval": 59,
  "max_offline_hours": 24,
  "log_level": "info"
}
```

## Database

SQLite database with encrypted settings table at:
- **Windows**: `%PROGRAMDATA%\POSService\data.db`
- **Linux**: `/var/lib/posservice/data.db`

### Settings Table
```sql
CREATE TABLE settings (
    key   VARCHAR(255) PRIMARY KEY,
    value TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

All data is encrypted with the server key using ChaCha20-Poly1305.

## Development

### Running Tests

```bash
# All tests
make test

# Specific package
go test -v ./internal/security
go test -v ./internal/database
go test -v ./internal/server

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Benchmarks
go test -bench=. -benchmem ./...
```

See [TESTING.md](TESTING.md) for detailed testing guide.

### Project Structure

```
promo-pos/
├── cmd/
│   └── service/          # Main service entry point
├── internal/
│   ├── api/              # API models and response structures
│   ├── config/           # Configuration management
│   ├── database/         # Database layer with encryption
│   ├── security/         # Encryption and machine ID
│   ├── server/           # HTTP server (Fiber v2)
│   └── service/          # Service wrapper
├── pkg/
│   ├── constants/        # Application constants
│   └── utils/            # Utility functions
├── build/                # Build artifacts
├── configs/              # Default configurations
├── CLAUDE.md            # Project specifications
├── TESTING.md           # Testing guide
├── Makefile             # Build automation
└── README.md            # This file
```

### Building

```bash
# Development build
make dev

# Production build (all platforms)
make build-all

# With compression (requires UPX)
make build-compressed

# Run linter
make lint

# Security scan
make security
```

## Architecture

### Encryption Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    ENCRYPTION SEPARATION                     │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  TYPE 1: Config File (Medium Importance)                    │
│  ├─ Encryption: AES-256-GCM                                 │
│  ├─ Key: HARD-CODED KEY (embedded in binary)               │
│  ├─ Salt: Machine ID (unique per machine)                  │
│  ├─ Derivation: PBKDF2(hardcodedKey, machineID, 10000)     │
│  ├─ Storage: %PROGRAMDATA%\POSService\config.enc           │
│  └─ Content: ServerURL, StoreID, Port, Intervals, etc.     │
│                                                              │
│  TYPE 2: Database (Very Important)                          │
│  ├─ Encryption: ChaCha20-Poly1305                          │
│  ├─ Key: SERVER KEY ONLY                                    │
│  ├─ Source: Fetched from API server via HTTPS              │
│  ├─ Storage: %PROGRAMDATA%\POSService\data.db              │
│  └─ Content: ALL database data including settings table     │
│                                                              │
│  SEPARATION RULES:                                          │
│  ✓ Hard-coded key + machine ID salt for config ONLY        │
│  ✓ Server key encrypts/decrypts database ONLY              │
│  ✗ NO cross-usage of keys                                  │
│  ✗ NO dual-key or hybrid encryption schemes                │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Service Lifecycle

1. **Startup**: Load config → Initialize database → Start HTTP server
2. **Running**: Handle requests → Sync every 59s → Monitor health
3. **Shutdown**: Stop HTTP server → Close database → Exit gracefully

### Offline Mode

- Continues operation for **24 hours** without server connection
- After 24 hours: Returns 503 Service Unavailable
- Automatic reconnection when server becomes available
- Pending syncs queued and processed on reconnection

## Performance Targets

- Service startup: < 3 seconds
- HTTP response time: p99 < 50ms
- Sync operation: < 5 seconds for 1000 records
- Memory usage: < 150MB idle, < 300MB active
- CPU usage: < 2% idle, < 10% during sync
- Binary size: < 15MB compressed

## Security Considerations

1. **Never log sensitive data** (tokens, keys, PINs)
2. **Encryption separation**: Config key and server key are NEVER mixed
3. **Machine-unique encryption**: Config is machine-specific
4. **Secure key storage**: Server key encrypted with config key in registry
5. **Prepared statements**: All SQL queries use parameterized statements
6. **Rate limiting**: 100 requests/minute per IP
7. **Graceful degradation**: Service continues with limited functionality when offline

## Troubleshooting

### Service won't start
```bash
# Check status
./pos-service -status

# View logs
# Windows: %PROGRAMDATA%\POSService\logs\
# Linux: /var/lib/posservice/logs/

# Run in debug mode
./pos-service -debug
```

### Database errors
- Verify server key is available
- Check database file permissions
- Ensure disk space is available

### Sync failures
- Check server URL configuration
- Verify network connectivity
- Review sync interval settings

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Commit your changes (`git commit -m 'Add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## License

[Add your license here]

## Support

For issues and questions:
- GitHub Issues: https://github.com/professor93/promo-pos/issues
- Documentation: See [CLAUDE.md](CLAUDE.md) for detailed specifications

## Version History

### v1.0.0 (Current)
- ✅ Phase 1: Core foundation (encryption, config, database)
- ✅ Phase 2: Service wrapper and HTTP server
- ✅ Comprehensive test coverage (2250+ lines of tests)
- ✅ Cross-platform support (Windows/Linux)
- ✅ Standardized API responses
- ⏳ Phase 3: GUI and system tray (planned)
- ⏳ Phase 4: Sync scheduler (planned)

---

**Built with ❤️ in Go**
