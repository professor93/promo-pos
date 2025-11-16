package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/professor93/promo-pos/internal/security"
	"github.com/professor93/promo-pos/pkg/constants"
)

// Config represents the application configuration
type Config struct {
	ServerURL       string `json:"server_url"`
	StoreID         string `json:"store_id"`
	Port            int    `json:"port"`
	SyncInterval    int    `json:"sync_interval"`     // seconds, default 59
	MaxOfflineHours int    `json:"max_offline_hours"` // default 24
	LogLevel        string `json:"log_level"`
	Encrypted       bool   `json:"encrypted"` // Whether this config is encrypted

	// Internal fields (not serialized)
	mu         sync.RWMutex      `json:"-"`
	encryption *security.ConfigEncryption `json:"-"`
	filePath   string            `json:"-"`
	lastSaved  time.Time         `json:"-"`
}

// Manager handles configuration loading, saving, and syncing
type Manager struct {
	config     *Config
	encryption *security.ConfigEncryption
	configPath string
	machineID  string
	mu         sync.RWMutex
}

// NewManager creates a new configuration manager
func NewManager(machineID string) (*Manager, error) {
	if machineID == "" {
		return nil, fmt.Errorf("machine ID cannot be empty")
	}

	// Create config encryption handler
	encryption, err := security.NewConfigEncryption(machineID)
	if err != nil {
		return nil, fmt.Errorf("failed to create config encryption: %w", err)
	}

	// Determine config path
	configDir := os.Getenv("PROGRAMDATA")
	if configDir == "" {
		configDir = "." // Fallback for development
	}
	configPath := filepath.Join(configDir, "POSService", constants.ConfigFileName)

	return &Manager{
		encryption: encryption,
		configPath: configPath,
		machineID:  machineID,
	}, nil
}

// Load loads the configuration from encrypted file
func (m *Manager) Load() (*Config, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if config file exists
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		// Config doesn't exist, return default config
		config := m.getDefaultConfig()
		m.config = config
		return config, nil
	}

	// Read encrypted config file
	encryptedData, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Decrypt config
	decryptedData, err := m.encryption.Decrypt(string(encryptedData))
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt config: %w", err)
	}

	// Parse JSON
	var config Config
	if err := json.Unmarshal(decryptedData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	config.encryption = m.encryption
	config.filePath = m.configPath
	config.Encrypted = true

	m.config = &config
	return &config, nil
}

// Save saves the current configuration to encrypted file
func (m *Manager) Save(config *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure directory exists
	configDir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Mark as encrypted
	config.Encrypted = true

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Encrypt config
	encryptedData, err := m.encryption.Encrypt(jsonData)
	if err != nil {
		return fmt.Errorf("failed to encrypt config: %w", err)
	}

	// Write to file with temp file + rename for atomicity
	tempPath := m.configPath + ".tmp"
	if err := os.WriteFile(tempPath, []byte(encryptedData), 0600); err != nil {
		return fmt.Errorf("failed to write temp config file: %w", err)
	}

	if err := os.Rename(tempPath, m.configPath); err != nil {
		os.Remove(tempPath) // Cleanup temp file
		return fmt.Errorf("failed to rename config file: %w", err)
	}

	config.lastSaved = time.Now()
	m.config = config

	return nil
}

// Get returns the current configuration (thread-safe)
func (m *Manager) Get() (*Config, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return nil, fmt.Errorf("configuration not loaded")
	}

	// Return a copy to prevent external modifications
	configCopy := *m.config
	return &configCopy, nil
}

// Update updates specific configuration fields and saves
func (m *Manager) Update(updateFunc func(*Config) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Apply update
	if err := updateFunc(m.config); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	// Save updated config
	return m.Save(m.config)
}

// getDefaultConfig returns the default configuration
func (m *Manager) getDefaultConfig() *Config {
	return &Config{
		ServerURL:       "",
		StoreID:         "",
		Port:            constants.DefaultPort,
		SyncInterval:    constants.DefaultSyncInterval,
		MaxOfflineHours: constants.DefaultMaxOfflineHours,
		LogLevel:        constants.DefaultLogLevel,
		Encrypted:       false,
		encryption:      m.encryption,
		filePath:        m.configPath,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.ServerURL == "" {
		return fmt.Errorf("server_url cannot be empty")
	}

	if c.StoreID == "" {
		return fmt.Errorf("store_id cannot be empty")
	}

	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	if c.SyncInterval < 1 {
		return fmt.Errorf("sync_interval must be at least 1 second")
	}

	if c.MaxOfflineHours < 1 {
		return fmt.Errorf("max_offline_hours must be at least 1 hour")
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log_level: must be debug, info, warn, or error")
	}

	return nil
}

// GetServerURL returns the server URL (thread-safe)
func (c *Config) GetServerURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ServerURL
}

// GetStoreID returns the store ID (thread-safe)
func (c *Config) GetStoreID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.StoreID
}

// GetPort returns the port (thread-safe)
func (c *Config) GetPort() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Port
}

// GetSyncInterval returns the sync interval in seconds (thread-safe)
func (c *Config) GetSyncInterval() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.SyncInterval
}

// GetMaxOfflineHours returns the max offline hours (thread-safe)
func (c *Config) GetMaxOfflineHours() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.MaxOfflineHours
}

// GetLogLevel returns the log level (thread-safe)
func (c *Config) GetLogLevel() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LogLevel
}
