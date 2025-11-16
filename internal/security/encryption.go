package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/sha3"
)

const (
	// PBKDF2 iterations for key derivation
	pbkdf2Iterations = 10000

	// AES-256 requires 32 byte key
	aes256KeySize = 32

	// ChaCha20-Poly1305 requires 32 byte key
	chacha20KeySize = 32
)

// IMPORTANT: This hard-coded key should be replaced with an environment variable
// or secure key management system in production builds
const hardcodedEncryptionKey = "YourSuperSecretHardcodedKeyHere-ChangeInProduction!"

var (
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	ErrInvalidKey        = errors.New("invalid encryption key")
)

// ConfigEncryption handles TYPE 1 encryption (Medium Importance)
// Uses AES-256-GCM with hard-coded key + machine ID as salt
type ConfigEncryption struct {
	machineID string
	key       []byte
}

// NewConfigEncryption creates a new config encryption handler
// It derives the encryption key from the hard-coded key and machine ID salt
func NewConfigEncryption(machineID string) (*ConfigEncryption, error) {
	if machineID == "" {
		return nil, errors.New("machine ID cannot be empty")
	}

	// Derive key using PBKDF2 with hard-coded key and machine ID as salt
	key := pbkdf2.Key(
		[]byte(hardcodedEncryptionKey),
		[]byte(machineID),
		pbkdf2Iterations,
		aes256KeySize,
		sha3.New256,
	)

	return &ConfigEncryption{
		machineID: machineID,
		key:       key,
	}, nil
}

// Encrypt encrypts data using AES-256-GCM
// Returns base64-encoded ciphertext
func (ce *ConfigEncryption) Encrypt(plaintext []byte) (string, error) {
	// Create AES cipher
	block, err := aes.NewCipher(ce.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and seal
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// Return base64 encoded
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext using AES-256-GCM
func (ce *ConfigEncryption) Decrypt(ciphertextB64 string) ([]byte, error) {
	// Decode base64
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(ce.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Check minimum length
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidCiphertext
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt and open
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// DatabaseEncryption handles TYPE 2 encryption (Very Important)
// Uses ChaCha20-Poly1305 with server key ONLY
type DatabaseEncryption struct {
	serverKey []byte
}

// NewDatabaseEncryption creates a new database encryption handler
// serverKey must be exactly 32 bytes (256 bits) fetched from the server
func NewDatabaseEncryption(serverKey []byte) (*DatabaseEncryption, error) {
	if len(serverKey) != chacha20KeySize {
		return nil, fmt.Errorf("%w: server key must be %d bytes", ErrInvalidKey, chacha20KeySize)
	}

	return &DatabaseEncryption{
		serverKey: serverKey,
	}, nil
}

// Encrypt encrypts data using ChaCha20-Poly1305
// Returns base64-encoded ciphertext
func (de *DatabaseEncryption) Encrypt(plaintext []byte) (string, error) {
	// Create ChaCha20-Poly1305 AEAD
	aead, err := chacha20poly1305.New(de.serverKey)
	if err != nil {
		return "", fmt.Errorf("failed to create chacha20poly1305: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and seal
	ciphertext := aead.Seal(nonce, nonce, plaintext, nil)

	// Return base64 encoded
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext using ChaCha20-Poly1305
func (de *DatabaseEncryption) Decrypt(ciphertextB64 string) ([]byte, error) {
	// Decode base64
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	// Create ChaCha20-Poly1305 AEAD
	aead, err := chacha20poly1305.New(de.serverKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create chacha20poly1305: %w", err)
	}

	// Check minimum length
	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidCiphertext
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt and open
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// GenerateServerKey generates a new random 256-bit key for database encryption
// This should typically be called on the server side
func GenerateServerKey() ([]byte, error) {
	key := make([]byte, chacha20KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate server key: %w", err)
	}
	return key, nil
}

// ServerKeyToBase64 converts a server key to base64 string for transmission
func ServerKeyToBase64(key []byte) string {
	return base64.StdEncoding.EncodeToString(key)
}

// ServerKeyFromBase64 converts a base64 string back to server key bytes
func ServerKeyFromBase64(keyB64 string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode server key: %w", err)
	}

	if len(key) != chacha20KeySize {
		return nil, fmt.Errorf("invalid server key length: expected %d, got %d", chacha20KeySize, len(key))
	}

	return key, nil
}
