# PROJECT_FEATURES.md - Windows POS Service Feature Specifications

## Executive Summary

A lightweight, secure Windows background service written in Go that provides offline-first POS functionality with encrypted data storage and automatic server synchronization.

## Core Features

### 1. Service Management

#### 1.1 Auto-Installation
- **Trigger**: First run detection (service not found in Windows Services)
- **Process**:
    - Check Windows service registry
    - If not installed, register as Windows service
    - Configure auto-start with delayed start
    - Set recovery options (restart on failure)
- **Service Properties**:
    - Name: `POSService`
    - Display Name: `POS Background Service`
    - Description: `Secure offline-first POS synchronization service`
    - Startup Type: Automatic (Delayed Start)
    - Account: LocalSystem or custom service account

#### 1.2 Installer Package
- **Type**: MSI installer (Windows Installer)
- **Options**:
    - Install: Fresh installation with service registration
    - Uninstall: Complete removal including:
        - Service deregistration
        - Configuration cleanup
        - Database removal (with option to preserve)
        - Registry cleanup
- **Features**:
    - Silent installation support: `/quiet` flag
    - Custom installation path
    - Preserve data on upgrade
    - Rollback capability

### 2. Security Architecture

#### 2.1 Machine Identification
- **Unique ID Generation**:
  ```
  MachineID = SHA256(
    Windows Product ID +
    Primary MAC Address +
    CPU Serial Number +
    Motherboard UUID
  )
  ```
- **Storage**: Windows Registry `HKLM\SOFTWARE\POSService\MachineID`
- **Immutability**: Generated once, never changes
- **Usage**: Base for all encryption keys

#### 2.2 Dual-Key Encryption System

##### Database Encryption
- **Key 1 - Machine Key**:
    - Derived from Machine ID
    - Algorithm: PBKDF2 with 10,000 iterations
    - Never leaves the machine
    - Regenerated on each use (not stored)

- **Key 2 - Server Key**:
    - Fetched from server API
    - Rotated monthly by server
    - Stored locally (encrypted with Machine Key)
    - Required for database access

- **Encryption Method**:
    - Database: SQLCipher or custom AES-256-GCM layer
    - Each table encrypted separately
    - Key combination: `FinalKey = HMAC(MachineKey, ServerKey)`

##### Configuration Encryption
- **Algorithm**: AES-256-GCM
- **Key**: Derived from Machine ID
- **Storage**: `%PROGRAMDATA%\POSService\config.enc`
- **Contents**: Server URL, port, credentials, sync settings

### 3. Data Synchronization

#### 3.1 Sync Schedule
- **Interval**: Every 59 seconds from application start
- **Timer Type**: Fixed interval (not sliding)
- **Example Timeline**:
  ```
  Start: 10:00:00
  Sync 1: 10:00:59
  Sync 2: 10:01:58
  Sync 3: 10:02:57
  ...continues every 59 seconds
  ```

#### 3.2 Sync Process
- **Direction**: Bidirectional (upload local changes, download server updates)
- **Conflict Resolution**: Server wins (last-write-wins with server priority)
- **Batch Size**: Configurable (default 100 records)
- **Retry Logic**:
    - Exponential backoff: 1s, 2s, 4s, 8s, 16s, 32s, max 5 minutes
    - Max retries: 10 before marking as failed
    - Failed items queued for next sync

#### 3.3 Offline Mode
- **Grace Period**: 24 hours from last successful sync
- **Behavior During Grace Period**:
    - Continue normal operations
    - Use cached configuration
    - Queue all changes for sync
    - Show warning in system tray

- **After Grace Period Expires**:
    - HTTP server returns `503 Service Unavailable`
    - Local operations continue
    - Critical alert in logs
    - System tray shows error icon
    - Requires manual intervention

### 4. Configuration Management

#### 4.1 Server-Driven Configuration
- **Initial Fetch**: On first successful authentication
- **Updates**: Retrieved during each sync
- **Contents**:
  ```json
  {
    "port": 8080,
    "sync_interval": 59,
    "max_offline_hours": 24,
    "api_endpoints": {...},
    "feature_flags": {...},
    "business_rules": {...}
  }
  ```

#### 4.2 Configuration Persistence
- **Primary**: Server provides latest config
- **Fallback**: Last known good config (encrypted)
- **Validity**: Cached config valid for 24 hours
- **Override**: Local config file for development/testing

### 5. Authentication System

#### 5.1 Initial Setup Flow

##### Screen 1: Store Registration
- **Display**: Machine ID (read-only)
- **Input**: Store ID
- **Validation**: Format check (alphanumeric, length)
- **Action**: Next → Screen 2

##### Screen 2: PIN Authentication
- **Display**:
    - Machine ID (read-only)
    - Store ID (read-only)
- **Input**: PIN Code (6-8 digits)
- **Validation**: Server verification
- **Action**:
    - Success → Minimize to tray
    - Failure → Show error, retry

#### 5.2 Token Management
- **Type**: JWT (JSON Web Token)
- **Storage**: Windows Credential Manager
- **Refresh**: Automatic before expiry
- **Fallback**: Re-authenticate with stored credentials
- **Security**: Never logged or transmitted in plain text

### 6. User Interface

#### 6.1 GUI Application
- **Framework**: Wails v2 (Go + Web Technologies)
- **Purpose**: Initial setup and configuration
- **Features**:
    - Modern, responsive design
    - Windows native look and feel
    - Minimal resource usage
    - No external dependencies

#### 6.2 System Tray
- **Behavior**:
    - Always running when service active
    - Close button minimizes to tray (doesn't exit)
    - Right-click for menu

- **Menu Options**:
  ```
  ┌─────────────────────────┐
  │ ● POS Service - Online  │
  ├─────────────────────────┤
  │ Status: Connected       │
  │ Last Sync: 2 min ago    │
  ├─────────────────────────┤
  │ Force Sync Now          │
  │ View Logs               │
  │ Settings                │
  ├─────────────────────────┤
  │ About                   │
  │ Exit GUI                │
  └─────────────────────────┘
  ```

### 7. HTTP Server

#### 7.1 Server Configuration
- **Port**: Dynamically assigned by server config
- **Framework**: Fiber v3 (or Chi as fallback)
- **Binding**: `localhost:{port}` (local only by default)

#### 7.2 Endpoints
```
GET  /health         → {"status":"ok","version":"1.0.0"}
GET  /status         → {"online":true,"last_sync":"2024-01-01T10:00:00Z"}
POST /transactions   → Process POS transaction
GET  /products       → Retrieve product catalog
POST /sync/force     → Trigger immediate sync
GET  /config         → Get current configuration
```

#### 7.3 Performance Requirements
- **Response Time**: p99 < 50ms
- **Concurrent Connections**: Max 100
- **Request Rate**: Handle 5-10 req/sec burst
- **Memory Usage**: < 50MB for HTTP server

### 8. Database Design

#### 8.1 SQLite Configuration
```sql
PRAGMA journal_mode = WAL;        -- Write-Ahead Logging
PRAGMA busy_timeout = 5000;       -- 5 second timeout
PRAGMA cache_size = -64000;       -- 64MB cache
PRAGMA temp_store = MEMORY;       -- Temp tables in RAM
PRAGMA synchronous = NORMAL;      -- Balance safety/speed
PRAGMA foreign_keys = ON;         -- Enforce FK constraints
```

#### 8.2 Core Tables
```sql
-- Transactions table
CREATE TABLE transactions (
    id TEXT PRIMARY KEY,
    store_id TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    data BLOB NOT NULL,  -- Encrypted JSON
    sync_status INTEGER DEFAULT 0,
    sync_attempts INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Configuration cache
CREATE TABLE config (
    key TEXT PRIMARY KEY,
    value BLOB NOT NULL,  -- Encrypted
    expires_at INTEGER,
    created_at INTEGER NOT NULL
);

-- Sync queue
CREATE TABLE sync_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    operation TEXT NOT NULL,
    table_name TEXT NOT NULL,
    record_id TEXT NOT NULL,
    data BLOB NOT NULL,
    priority INTEGER DEFAULT 0,
    attempts INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL
);
```

### 9. Logging and Monitoring

#### 9.1 Logging Configuration
- **Framework**: Uber's Zap (structured logging)
- **Location**: `%PROGRAMDATA%\POSService\logs\`
- **Rotation**: 10MB files, keep 7 days
- **Levels**:
    - Production: INFO and above
    - Development: DEBUG and above
- **Format**: JSON for parsing, text for development

#### 9.2 Log Categories
```
SERVICE   - Service lifecycle events
SYNC      - Synchronization operations
API       - External API calls
DATABASE  - Database operations
SECURITY  - Authentication, encryption
HTTP      - HTTP server requests
CONFIG    - Configuration changes
ERROR     - All errors with stack traces
```

### 10. Resource Constraints

#### 10.1 Memory Management
- **Target**: < 150MB idle, < 300MB active
- **Techniques**:
    - Aggressive GC: `GOGC=50`
    - Object pooling for frequent allocations
    - Streaming for large data sets
    - Bounded queues and buffers

#### 10.2 Binary Size
- **Target**: < 15MB compressed
- **Optimization**:
    - Strip debug symbols: `-ldflags="-s -w"`
    - UPX compression: `upx --best --lzma`
    - Minimal dependencies
    - No embedded assets except icons

### 11. Error Handling

#### 11.1 Failure Scenarios

##### Network Failures
- Automatic retry with backoff
- Queue changes locally
- Continue offline operation
- Clear status indication

##### Database Corruption
- Automatic backup before migrations
- Integrity check on startup
- Recovery from backup
- Alert administrators

##### Service Crashes
- Windows service recovery (auto-restart)
- Preserve sync queue
- Resume from last checkpoint
- Crash report generation

### 12. Deployment Features

#### 12.1 Update Mechanism
- **Check**: Every sync checks for updates
- **Download**: Background download of new version
- **Install**: Schedule during maintenance window
- **Rollback**: Keep previous version for rollback

#### 12.2 Multi-Store Support
- **Isolation**: Each store has unique:
    - Store ID
    - Database file
    - Configuration
    - Sync queue
- **Management**: Central management console (future)

### 13. Performance Metrics

| Metric | Target | Critical Threshold |
|--------|--------|-------------------|
| Service Startup | < 3 seconds | > 10 seconds |
| HTTP Response (p99) | < 50ms | > 200ms |
| Sync Duration | < 5 seconds | > 30 seconds |
| Memory Usage (Idle) | < 150MB | > 300MB |
| Memory Usage (Active) | < 300MB | > 500MB |
| CPU Usage (Idle) | < 2% | > 5% |
| CPU Usage (Sync) | < 10% | > 25% |
| Database Size | < 500MB | > 1GB |
| Log Size (Daily) | < 50MB | > 200MB |

### 14. Compliance and Standards

#### 14.1 Security Standards
- **Encryption**: AES-256-GCM minimum
- **Key Length**: 256-bit minimum
- **Password/PIN**: Hashed with bcrypt
- **TLS**: Version 1.2 minimum
- **Certificates**: Validate server certificates

#### 14.2 Data Handling
- **PCI DSS**: No credit card data in logs
- **GDPR**: Data encryption at rest
- **Audit Trail**: All access logged
- **Data Retention**: Configurable per regulations

### 15. Development Guidelines

#### 15.1 Code Quality
- **Testing**: Minimum 80% coverage for critical paths
- **Linting**: golangci-lint with strict settings
- **Documentation**: All exported functions documented
- **Examples**: Usage examples for main features

#### 15.2 Build Pipeline
```yaml
stages:
  1. lint:     golangci-lint run
  2. test:     go test -race -cover ./...
  3. build:    go build -ldflags="-s -w"
  4. compress: upx --best --lzma
  5. package:  wix candle && light
  6. sign:     signtool sign
  7. verify:   integration tests
```

---

## Implementation Priority

### Phase 1 - Core (Week 1-2)
- [ ] Machine ID generation
- [ ] Basic service skeleton
- [ ] SQLite setup with encryption
- [ ] Configuration management

### Phase 2 - Service (Week 3-4)
- [ ] Windows service installation
- [ ] HTTP server setup
- [ ] Basic API endpoints
- [ ] Logging framework

### Phase 3 - Sync (Week 5-6)
- [ ] Server communication
- [ ] Sync scheduler
- [ ] Offline queue
- [ ] Conflict resolution

### Phase 4 - GUI (Week 7-8)
- [ ] Login screens
- [ ] System tray
- [ ] Status monitoring
- [ ] Settings interface

### Phase 5 - Installer (Week 9-10)
- [ ] MSI creation
- [ ] Install/uninstall logic
- [ ] Service registration
- [ ] Testing on clean systems

### Phase 6 - Hardening (Week 11-12)
- [ ] Security audit
- [ ] Performance optimization
- [ ] Error handling improvements
- [ ] Documentation completion

---

**Document Version**: 1.0.0  
**Last Updated**: 2024  
**Status**: Ready for Implementation