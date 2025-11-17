package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/professor93/promo-pos/internal/security"
	"github.com/professor93/promo-pos/pkg/constants"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// DB represents the database connection with encryption
type DB struct {
	conn       *sql.DB
	encryption *security.DatabaseEncryption
	dbPath     string
	mu         sync.RWMutex
}

// Config holds database configuration
type Config struct {
	ServerKey []byte // 32-byte server key for encryption
	DataDir   string // Directory for database file
}

// New creates a new database instance with server-key encryption
func New(cfg *Config) (*DB, error) {
	if len(cfg.ServerKey) != 32 {
		return nil, fmt.Errorf("server key must be exactly 32 bytes")
	}

	// Create encryption handler
	encryption, err := security.NewDatabaseEncryption(cfg.ServerKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create database encryption: %w", err)
	}

	// Determine database path
	dataDir := cfg.DataDir
	if dataDir == "" {
		dataDir = os.Getenv("PROGRAMDATA")
		if dataDir == "" {
			dataDir = "." // Fallback for development
		}
		dataDir = filepath.Join(dataDir, "POSService")
	}

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, constants.DatabaseFileName)

	// Open SQLite database
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	// Set PRAGMA options for performance and reliability
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA cache_size=-64000", // 64MB cache
		"PRAGMA temp_store=MEMORY",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
	}

	for _, pragma := range pragmas {
		if _, err := conn.Exec(pragma); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to set pragma %s: %w", pragma, err)
		}
	}

	db := &DB{
		conn:       conn,
		encryption: encryption,
		dbPath:     dbPath,
	}

	// Initialize schema
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// initSchema creates the required database tables
func (db *DB) initSchema() error {
	// Create settings table
	settingsTableSQL := `
	CREATE TABLE IF NOT EXISTS settings (
		key   VARCHAR(255) PRIMARY KEY,
		value TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TRIGGER IF NOT EXISTS settings_updated_at
	AFTER UPDATE ON settings
	FOR EACH ROW
	BEGIN
		UPDATE settings SET updated_at = CURRENT_TIMESTAMP WHERE key = NEW.key;
	END;
	`

	if _, err := db.conn.Exec(settingsTableSQL); err != nil {
		return fmt.Errorf("failed to create settings table: %w", err)
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// Ping checks if the database connection is alive
func (db *DB) Ping() error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.conn == nil {
		return fmt.Errorf("database connection is nil")
	}

	return db.conn.Ping()
}

// GetConnection returns the underlying SQL connection (use with caution)
func (db *DB) GetConnection() *sql.DB {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.conn
}

// --- Settings Table Methods ---

// GetSetting retrieves a setting value by key (decrypts automatically)
func (db *DB) GetSetting(key string) (string, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var encryptedValue string
	query := "SELECT value FROM settings WHERE key = ?"

	err := db.conn.QueryRow(query, key).Scan(&encryptedValue)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("setting not found: %s", key)
	}
	if err != nil {
		return "", fmt.Errorf("failed to query setting: %w", err)
	}

	// Decrypt value
	decryptedValue, err := db.encryption.Decrypt(encryptedValue)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt setting value: %w", err)
	}

	return string(decryptedValue), nil
}

// SetSetting stores a setting value by key (encrypts automatically)
func (db *DB) SetSetting(key, value string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Encrypt value
	encryptedValue, err := db.encryption.Encrypt([]byte(value))
	if err != nil {
		return fmt.Errorf("failed to encrypt setting value: %w", err)
	}

	// Upsert (INSERT OR REPLACE)
	query := `
		INSERT INTO settings (key, value, created_at, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at = CURRENT_TIMESTAMP
	`

	if _, err := db.conn.Exec(query, key, encryptedValue); err != nil {
		return fmt.Errorf("failed to set setting: %w", err)
	}

	return nil
}

// DeleteSetting deletes a setting by key
func (db *DB) DeleteSetting(key string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	query := "DELETE FROM settings WHERE key = ?"

	result, err := db.conn.Exec(query, key)
	if err != nil {
		return fmt.Errorf("failed to delete setting: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("setting not found: %s", key)
	}

	return nil
}

// GetAllSettings retrieves all settings (decrypts automatically)
func (db *DB) GetAllSettings() (map[string]string, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	query := "SELECT key, value FROM settings"

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query settings: %w", err)
	}
	defer rows.Close()

	settings := make(map[string]string)

	for rows.Next() {
		var key, encryptedValue string
		if err := rows.Scan(&key, &encryptedValue); err != nil {
			return nil, fmt.Errorf("failed to scan setting row: %w", err)
		}

		// Decrypt value
		decryptedValue, err := db.encryption.Decrypt(encryptedValue)
		if err != nil {
			// Log error but continue with other settings
			fmt.Printf("Warning: failed to decrypt setting %s: %v\n", key, err)
			continue
		}

		settings[key] = string(decryptedValue)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating settings: %w", err)
	}

	return settings, nil
}

// SettingExists checks if a setting key exists
func (db *DB) SettingExists(key string) (bool, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM settings WHERE key = ?)"

	err := db.conn.QueryRow(query, key).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check setting existence: %w", err)
	}

	return exists, nil
}

// Transaction executes a function within a database transaction
func (db *DB) Transaction(fn func(*sql.Tx) error) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("failed to rollback transaction: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
