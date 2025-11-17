package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/professor93/promo-pos/internal/security"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "posservice-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Generate server key
	serverKey, err := security.GenerateServerKey()
	if err != nil {
		t.Fatalf("Failed to generate server key: %v", err)
	}

	// Create database
	db, err := New(&Config{
		ServerKey: serverKey,
		DataDir:   tmpDir,
	})
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create database: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestNew(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Verify database is created
	if db == nil {
		t.Fatal("Database is nil")
	}

	// Verify connection works
	err := db.Ping()
	if err != nil {
		t.Errorf("Database ping failed: %v", err)
	}
}

func TestNew_InvalidServerKey(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "posservice-test-*")
	defer os.RemoveAll(tmpDir)

	testCases := []struct {
		name      string
		keyLength int
	}{
		{"Empty key", 0},
		{"Too short", 16},
		{"Too long", 64},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			invalidKey := make([]byte, tc.keyLength)
			_, err := New(&Config{
				ServerKey: invalidKey,
				DataDir:   tmpDir,
			})
			if err == nil {
				t.Error("Expected error for invalid server key, but got none")
			}
		})
	}
}

func TestSetSetting(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testCases := []struct {
		key   string
		value string
	}{
		{"app_name", "POS Service"},
		{"version", "1.0.0"},
		{"feature_flag_1", "true"},
		{"config_json", `{"enabled":true,"timeout":30}`},
		{"unicode", "Hello 世界 🌍"},
	}

	for _, tc := range testCases {
		t.Run(tc.key, func(t *testing.T) {
			err := db.SetSetting(tc.key, tc.value)
			if err != nil {
				t.Errorf("SetSetting failed: %v", err)
			}
		})
	}
}

func TestGetSetting(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Set a value
	key := "test_key"
	expectedValue := "test_value_12345"

	err := db.SetSetting(key, expectedValue)
	if err != nil {
		t.Fatalf("SetSetting failed: %v", err)
	}

	// Get the value
	value, err := db.GetSetting(key)
	if err != nil {
		t.Fatalf("GetSetting failed: %v", err)
	}

	if value != expectedValue {
		t.Errorf("Retrieved value doesn't match.\nExpected: %q\nGot: %q", expectedValue, value)
	}
}

func TestGetSetting_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := db.GetSetting("non_existent_key")
	if err == nil {
		t.Error("Expected error for non-existent key, but got none")
	}
}

func TestSetSetting_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	key := "update_test"
	value1 := "original_value"
	value2 := "updated_value"

	// Set initial value
	err := db.SetSetting(key, value1)
	if err != nil {
		t.Fatalf("Initial SetSetting failed: %v", err)
	}

	// Update value
	err = db.SetSetting(key, value2)
	if err != nil {
		t.Fatalf("Update SetSetting failed: %v", err)
	}

	// Verify updated value
	value, err := db.GetSetting(key)
	if err != nil {
		t.Fatalf("GetSetting failed: %v", err)
	}

	if value != value2 {
		t.Errorf("Expected updated value %q, got %q", value2, value)
	}
}

func TestDeleteSetting(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	key := "delete_test"
	value := "to_be_deleted"

	// Set value
	err := db.SetSetting(key, value)
	if err != nil {
		t.Fatalf("SetSetting failed: %v", err)
	}

	// Delete
	err = db.DeleteSetting(key)
	if err != nil {
		t.Errorf("DeleteSetting failed: %v", err)
	}

	// Verify deleted
	_, err = db.GetSetting(key)
	if err == nil {
		t.Error("Expected error after deletion, but got none")
	}
}

func TestDeleteSetting_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := db.DeleteSetting("non_existent_key")
	if err == nil {
		t.Error("Expected error when deleting non-existent key, but got none")
	}
}

func TestGetAllSettings(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Set multiple settings
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for k, v := range testData {
		err := db.SetSetting(k, v)
		if err != nil {
			t.Fatalf("SetSetting failed for %s: %v", k, err)
		}
	}

	// Get all settings
	settings, err := db.GetAllSettings()
	if err != nil {
		t.Fatalf("GetAllSettings failed: %v", err)
	}

	// Verify count
	if len(settings) != len(testData) {
		t.Errorf("Expected %d settings, got %d", len(testData), len(settings))
	}

	// Verify values
	for k, expectedV := range testData {
		actualV, exists := settings[k]
		if !exists {
			t.Errorf("Setting %s not found in results", k)
			continue
		}
		if actualV != expectedV {
			t.Errorf("Setting %s: expected %q, got %q", k, expectedV, actualV)
		}
	}
}

func TestSettingExists(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	key := "exists_test"
	value := "test_value"

	// Should not exist initially
	exists, err := db.SettingExists(key)
	if err != nil {
		t.Fatalf("SettingExists failed: %v", err)
	}
	if exists {
		t.Error("Setting should not exist yet")
	}

	// Set value
	err = db.SetSetting(key, value)
	if err != nil {
		t.Fatalf("SetSetting failed: %v", err)
	}

	// Should exist now
	exists, err = db.SettingExists(key)
	if err != nil {
		t.Fatalf("SettingExists failed: %v", err)
	}
	if !exists {
		t.Error("Setting should exist after being set")
	}
}

func TestTransaction(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Test successful transaction
	err := db.Transaction(func(tx *sql.Tx) error {
		// Get encrypted value for settings
		serverKey, _ := security.GenerateServerKey()
		de, _ := security.NewDatabaseEncryption(serverKey)
		encrypted, _ := de.Encrypt([]byte("tx_value"))

		_, err := tx.Exec("INSERT INTO settings (key, value) VALUES (?, ?)", "tx_key", encrypted)
		return err
	})

	if err != nil {
		t.Errorf("Transaction failed: %v", err)
	}

	// Verify data was committed
	exists, _ := db.SettingExists("tx_key")
	if !exists {
		t.Error("Transaction did not commit data")
	}
}

func TestEncryptionInDatabase(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	key := "encrypted_key"
	value := "sensitive_data_12345"

	// Set value (should be encrypted)
	err := db.SetSetting(key, value)
	if err != nil {
		t.Fatalf("SetSetting failed: %v", err)
	}

	// Read directly from database to verify encryption
	var storedValue string
	conn := db.GetConnection()
	err = conn.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&storedValue)
	if err != nil {
		t.Fatalf("Direct query failed: %v", err)
	}

	// Stored value should NOT match plaintext
	if storedValue == value {
		t.Error("Value in database is not encrypted")
	}

	// Verify we can decrypt it properly
	retrieved, err := db.GetSetting(key)
	if err != nil {
		t.Fatalf("GetSetting failed: %v", err)
	}

	if retrieved != value {
		t.Errorf("Decrypted value doesn't match.\nExpected: %q\nGot: %q", value, retrieved)
	}
}

func TestDatabasePath(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "posservice-test-*")
	defer os.RemoveAll(tmpDir)

	serverKey, _ := security.GenerateServerKey()
	db, err := New(&Config{
		ServerKey: serverKey,
		DataDir:   tmpDir,
	})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Verify database file exists
	dbPath := filepath.Join(tmpDir, "data.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("Database file not created at expected path: %s", dbPath)
	}
}

func BenchmarkSetSetting(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "posservice-bench-*")
	defer os.RemoveAll(tmpDir)

	serverKey, _ := security.GenerateServerKey()
	db, _ := New(&Config{
		ServerKey: serverKey,
		DataDir:   tmpDir,
	})
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.SetSetting("bench_key", "bench_value")
	}
}

func BenchmarkGetSetting(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "posservice-bench-*")
	defer os.RemoveAll(tmpDir)

	serverKey, _ := security.GenerateServerKey()
	db, _ := New(&Config{
		ServerKey: serverKey,
		DataDir:   tmpDir,
	})
	defer db.Close()

	db.SetSetting("bench_key", "bench_value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.GetSetting("bench_key")
	}
}
