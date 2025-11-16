package security

import (
	"bytes"
	"testing"
)

func TestConfigEncryption_EncryptDecrypt(t *testing.T) {
	machineID := "test-machine-id-12345"

	// Create config encryption
	ce, err := NewConfigEncryption(machineID)
	if err != nil {
		t.Fatalf("Failed to create config encryption: %v", err)
	}

	testCases := []struct {
		name      string
		plaintext string
	}{
		{"Empty string", ""},
		{"Short string", "Hello, World!"},
		{"Long string", "This is a much longer string that contains multiple sentences. It should test the encryption with a larger payload to ensure it handles varying sizes correctly."},
		{"Special characters", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"Unicode", "Hello 世界 🌍 Привет مرحبا"},
		{"JSON", `{"server_url":"https://example.com","store_id":"12345","port":8080}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := ce.Encrypt([]byte(tc.plaintext))
			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}

			if ciphertext == "" {
				t.Fatal("Ciphertext is empty")
			}

			// Decrypt
			decrypted, err := ce.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}

			// Verify
			if string(decrypted) != tc.plaintext {
				t.Errorf("Decrypted text doesn't match original.\nExpected: %q\nGot: %q", tc.plaintext, string(decrypted))
			}
		})
	}
}

func TestConfigEncryption_DifferentMachineIDs(t *testing.T) {
	plaintext := []byte("sensitive config data")

	// Encrypt with first machine ID
	ce1, err := NewConfigEncryption("machine-1")
	if err != nil {
		t.Fatalf("Failed to create config encryption 1: %v", err)
	}

	ciphertext1, err := ce1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encryption 1 failed: %v", err)
	}

	// Try to decrypt with different machine ID
	ce2, err := NewConfigEncryption("machine-2")
	if err != nil {
		t.Fatalf("Failed to create config encryption 2: %v", err)
	}

	_, err = ce2.Decrypt(ciphertext1)
	if err == nil {
		t.Error("Expected decryption to fail with different machine ID, but it succeeded")
	}
}

func TestConfigEncryption_InvalidInput(t *testing.T) {
	ce, err := NewConfigEncryption("test-machine")
	if err != nil {
		t.Fatalf("Failed to create config encryption: %v", err)
	}

	testCases := []struct {
		name       string
		ciphertext string
	}{
		{"Empty string", ""},
		{"Invalid base64", "not-valid-base64!!!"},
		{"Too short", "YWJj"},
		{"Random data", "SGVsbG8gV29ybGQ="},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ce.Decrypt(tc.ciphertext)
			if err == nil {
				t.Error("Expected decryption to fail, but it succeeded")
			}
		})
	}
}

func TestDatabaseEncryption_EncryptDecrypt(t *testing.T) {
	// Generate a server key
	serverKey, err := GenerateServerKey()
	if err != nil {
		t.Fatalf("Failed to generate server key: %v", err)
	}

	// Create database encryption
	de, err := NewDatabaseEncryption(serverKey)
	if err != nil {
		t.Fatalf("Failed to create database encryption: %v", err)
	}

	testCases := []struct {
		name      string
		plaintext string
	}{
		{"Empty string", ""},
		{"Short string", "user data"},
		{"Long string", "This is very important database data that needs to be encrypted with the server key only. It contains sensitive information."},
		{"Binary-like data", "\x00\x01\x02\x03\x04\x05"},
		{"JSON", `{"id":123,"name":"John Doe","balance":1000.50}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := de.Encrypt([]byte(tc.plaintext))
			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}

			// Decrypt
			decrypted, err := de.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}

			// Verify
			if string(decrypted) != tc.plaintext {
				t.Errorf("Decrypted text doesn't match.\nExpected: %q\nGot: %q", tc.plaintext, string(decrypted))
			}
		})
	}
}

func TestDatabaseEncryption_DifferentKeys(t *testing.T) {
	plaintext := []byte("database record")

	// Create first encryption with key 1
	key1, _ := GenerateServerKey()
	de1, err := NewDatabaseEncryption(key1)
	if err != nil {
		t.Fatalf("Failed to create database encryption 1: %v", err)
	}

	ciphertext, err := de1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Try to decrypt with different key
	key2, _ := GenerateServerKey()
	de2, err := NewDatabaseEncryption(key2)
	if err != nil {
		t.Fatalf("Failed to create database encryption 2: %v", err)
	}

	_, err = de2.Decrypt(ciphertext)
	if err == nil {
		t.Error("Expected decryption to fail with different key, but it succeeded")
	}
}

func TestDatabaseEncryption_InvalidKeySize(t *testing.T) {
	testCases := []struct {
		name    string
		keySize int
	}{
		{"Too short", 16},
		{"Too long", 64},
		{"Empty", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := make([]byte, tc.keySize)
			_, err := NewDatabaseEncryption(key)
			if err == nil {
				t.Error("Expected error for invalid key size, but got none")
			}
		})
	}
}

func TestGenerateServerKey(t *testing.T) {
	// Generate multiple keys
	key1, err := GenerateServerKey()
	if err != nil {
		t.Fatalf("Failed to generate key 1: %v", err)
	}

	key2, err := GenerateServerKey()
	if err != nil {
		t.Fatalf("Failed to generate key 2: %v", err)
	}

	// Verify key length
	if len(key1) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key1))
	}

	// Verify keys are different
	if bytes.Equal(key1, key2) {
		t.Error("Generated keys are identical, expected them to be random")
	}
}

func TestServerKeyBase64Conversion(t *testing.T) {
	// Generate a key
	key, err := GenerateServerKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Convert to base64
	keyB64 := ServerKeyToBase64(key)

	// Convert back
	recoveredKey, err := ServerKeyFromBase64(keyB64)
	if err != nil {
		t.Fatalf("Failed to convert from base64: %v", err)
	}

	// Verify
	if !bytes.Equal(key, recoveredKey) {
		t.Error("Key recovered from base64 doesn't match original")
	}
}

func TestServerKeyFromBase64_Invalid(t *testing.T) {
	testCases := []struct {
		name   string
		base64 string
	}{
		{"Invalid base64", "not-valid-base64!!!"},
		{"Wrong length", "YWJjZGVm"}, // "abcdef" in base64
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ServerKeyFromBase64(tc.base64)
			if err == nil {
				t.Error("Expected error for invalid base64 key, but got none")
			}
		})
	}
}

func TestEncryptionSeparation(t *testing.T) {
	// This test verifies that Type 1 and Type 2 encryption are completely separate
	plaintext := []byte("test data")

	// Type 1: Config encryption
	ce, err := NewConfigEncryption("machine-123")
	if err != nil {
		t.Fatalf("Failed to create config encryption: %v", err)
	}

	configCiphertext, err := ce.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Config encryption failed: %v", err)
	}

	// Type 2: Database encryption
	serverKey, _ := GenerateServerKey()
	de, err := NewDatabaseEncryption(serverKey)
	if err != nil {
		t.Fatalf("Failed to create database encryption: %v", err)
	}

	dbCiphertext, err := de.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Database encryption failed: %v", err)
	}

	// Verify ciphertexts are different
	if configCiphertext == dbCiphertext {
		t.Error("Config and database ciphertexts are identical, expected them to be different")
	}

	// Verify Type 1 key cannot decrypt Type 2 data
	_, err = ce.Decrypt(dbCiphertext)
	if err == nil {
		t.Error("Config encryption should not be able to decrypt database data")
	}

	// Verify Type 2 key cannot decrypt Type 1 data
	_, err = de.Decrypt(configCiphertext)
	if err == nil {
		t.Error("Database encryption should not be able to decrypt config data")
	}
}

func BenchmarkConfigEncryption(b *testing.B) {
	ce, _ := NewConfigEncryption("benchmark-machine")
	plaintext := []byte("benchmark data for config encryption testing")

	b.Run("Encrypt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ce.Encrypt(plaintext)
		}
	})

	ciphertext, _ := ce.Encrypt(plaintext)
	b.Run("Decrypt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ce.Decrypt(ciphertext)
		}
	})
}

func BenchmarkDatabaseEncryption(b *testing.B) {
	key, _ := GenerateServerKey()
	de, _ := NewDatabaseEncryption(key)
	plaintext := []byte("benchmark data for database encryption testing")

	b.Run("Encrypt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			de.Encrypt(plaintext)
		}
	})

	ciphertext, _ := de.Encrypt(plaintext)
	b.Run("Decrypt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			de.Decrypt(ciphertext)
		}
	})
}
