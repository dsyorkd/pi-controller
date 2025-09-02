package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"go.etcd.io/bbolt"
	"golang.org/x/crypto/pbkdf2"
)

// EncryptionConfig holds encryption configuration
type EncryptionConfig struct {
	Enabled           bool   `yaml:"enabled"`
	KeyFile           string `yaml:"key_file"`
	KeyFromEnv        string `yaml:"key_from_env"`
	EncryptedDBPath   string `yaml:"encrypted_db_path"`
	PragmaKey         string `yaml:"pragma_key"`
	PBKDF2Iterations  int    `yaml:"pbkdf2_iterations"`
	GenerateKeyIfMissing bool `yaml:"generate_key_if_missing"`
}

// DefaultEncryptionConfig returns secure default encryption configuration
func DefaultEncryptionConfig() *EncryptionConfig {
	return &EncryptionConfig{
		Enabled:              false, // Disabled by default for compatibility
		KeyFile:              "data/db.key",
		KeyFromEnv:           "PI_CONTROLLER_DB_KEY",
		EncryptedDBPath:      "data/encrypted.db",
		PBKDF2Iterations:     100000,
		GenerateKeyIfMissing: true,
	}
}

// EncryptedStorage provides encrypted storage for sensitive data
type EncryptedStorage struct {
	config *EncryptionConfig
	logger *logrus.Entry
	db     *bbolt.DB
	gcm    cipher.AEAD
}

// NewEncryptedStorage creates a new encrypted storage instance
func NewEncryptedStorage(config *EncryptionConfig, logger *logrus.Logger) (*EncryptedStorage, error) {
	if config == nil {
		config = DefaultEncryptionConfig()
	}

	if !config.Enabled {
		return nil, nil // Return nil if encryption is disabled
	}

	es := &EncryptedStorage{
		config: config,
		logger: logger.WithField("component", "encrypted-storage"),
	}

	// Initialize encryption key
	key, err := es.loadOrGenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to load encryption key: %w", err)
	}

	// Create AES-GCM cipher
	if err := es.initializeCipher(key); err != nil {
		return nil, fmt.Errorf("failed to initialize cipher: %w", err)
	}

	// Open encrypted database
	if err := es.openDatabase(); err != nil {
		return nil, fmt.Errorf("failed to open encrypted database: %w", err)
	}

	es.logger.Info("Encrypted storage initialized successfully")
	return es, nil
}

// loadOrGenerateKey loads encryption key from file or environment, generates if missing
func (es *EncryptedStorage) loadOrGenerateKey() ([]byte, error) {
	// Try to load from environment variable first
	if es.config.KeyFromEnv != "" {
		if envKey := os.Getenv(es.config.KeyFromEnv); envKey != "" {
			key, err := base64.StdEncoding.DecodeString(envKey)
			if err != nil {
				return nil, fmt.Errorf("failed to decode key from environment: %w", err)
			}
			if len(key) != 32 {
				return nil, fmt.Errorf("encryption key from environment must be 32 bytes (256 bits)")
			}
			es.logger.Info("Encryption key loaded from environment variable")
			return key, nil
		}
	}

	// Try to load from file
	if es.config.KeyFile != "" {
		if _, err := os.Stat(es.config.KeyFile); err == nil {
			keyData, err := os.ReadFile(es.config.KeyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read key file: %w", err)
			}

			key, err := base64.StdEncoding.DecodeString(string(keyData))
			if err != nil {
				return nil, fmt.Errorf("failed to decode key from file: %w", err)
			}

			if len(key) != 32 {
				return nil, fmt.Errorf("encryption key from file must be 32 bytes (256 bits)")
			}

			es.logger.WithField("key_file", es.config.KeyFile).Info("Encryption key loaded from file")
			return key, nil
		}
	}

	// Generate new key if allowed
	if es.config.GenerateKeyIfMissing {
		return es.generateAndSaveKey()
	}

	return nil, fmt.Errorf("no encryption key found and key generation is disabled")
}

// generateAndSaveKey generates a new encryption key and saves it to file
func (es *EncryptedStorage) generateAndSaveKey() ([]byte, error) {
	// Generate 256-bit key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Save to file if path is configured
	if es.config.KeyFile != "" {
		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(es.config.KeyFile), 0700); err != nil {
			return nil, fmt.Errorf("failed to create key directory: %w", err)
		}

		// Encode key as base64
		keyB64 := base64.StdEncoding.EncodeToString(key)

		// Write to file with restrictive permissions
		if err := os.WriteFile(es.config.KeyFile, []byte(keyB64), 0600); err != nil {
			return nil, fmt.Errorf("failed to save key file: %w", err)
		}

		es.logger.WithField("key_file", es.config.KeyFile).Warn("Generated new encryption key and saved to file")
		es.logger.Warn("Please backup this key file and consider using environment variables for production")
	}

	return key, nil
}

// initializeCipher initializes the AES-GCM cipher
func (es *EncryptedStorage) initializeCipher(key []byte) error {
	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	es.gcm = gcm
	return nil
}

// openDatabase opens the encrypted bolt database
func (es *EncryptedStorage) openDatabase() error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(es.config.EncryptedDBPath), 0700); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database with restrictive permissions
	db, err := bbolt.Open(es.config.EncryptedDBPath, 0600, &bbolt.Options{
		Timeout: 0,
	})
	if err != nil {
		return fmt.Errorf("failed to open encrypted database: %w", err)
	}

	es.db = db
	return nil
}

// Close closes the encrypted storage
func (es *EncryptedStorage) Close() error {
	if es.db != nil {
		return es.db.Close()
	}
	return nil
}

// Store encrypts and stores data in the encrypted storage
func (es *EncryptedStorage) Store(bucket, key string, data []byte) error {
	if es.db == nil {
		return fmt.Errorf("encrypted storage not initialized")
	}

	// Encrypt data
	encryptedData, err := es.encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}

	// Store in database
	return es.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		return b.Put([]byte(key), encryptedData)
	})
}

// Retrieve decrypts and retrieves data from encrypted storage
func (es *EncryptedStorage) Retrieve(bucket, key string) ([]byte, error) {
	if es.db == nil {
		return nil, fmt.Errorf("encrypted storage not initialized")
	}

	var encryptedData []byte
	err := es.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		
		data := b.Get([]byte(key))
		if data == nil {
			return fmt.Errorf("key %s not found in bucket %s", key, bucket)
		}
		
		encryptedData = make([]byte, len(data))
		copy(encryptedData, data)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Decrypt data
	return es.decrypt(encryptedData)
}

// Delete removes data from encrypted storage
func (es *EncryptedStorage) Delete(bucket, key string) error {
	if es.db == nil {
		return fmt.Errorf("encrypted storage not initialized")
	}

	return es.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		return b.Delete([]byte(key))
	})
}

// List returns all keys in a bucket
func (es *EncryptedStorage) List(bucket string) ([]string, error) {
	if es.db == nil {
		return nil, fmt.Errorf("encrypted storage not initialized")
	}

	var keys []string
	err := es.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil // Return empty list if bucket doesn't exist
		}

		return b.ForEach(func(k, v []byte) error {
			keys = append(keys, string(k))
			return nil
		})
	})

	return keys, err
}

// encrypt encrypts data using AES-GCM
func (es *EncryptedStorage) encrypt(data []byte) ([]byte, error) {
	// Create nonce
	nonce := make([]byte, es.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to create nonce: %w", err)
	}

	// Encrypt data
	ciphertext := es.gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// decrypt decrypts data using AES-GCM
func (es *EncryptedStorage) decrypt(data []byte) ([]byte, error) {
	nonceSize := es.gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	// Decrypt data
	plaintext, err := es.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return plaintext, nil
}

// DeriveKeyFromPassword derives an encryption key from a password using PBKDF2
func (es *EncryptedStorage) DeriveKeyFromPassword(password string, salt []byte) []byte {
	if len(salt) == 0 {
		// Create default salt (not recommended for production)
		salt = []byte("pi-controller-salt")
		es.logger.Warn("Using default salt for key derivation - not recommended for production")
	}

	return pbkdf2.Key([]byte(password), salt, es.config.PBKDF2Iterations, 32, sha256.New)
}

// GetStats returns encryption statistics
func (es *EncryptedStorage) GetStats() map[string]interface{} {
	if es.db == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	stats := es.db.Stats()
	return map[string]interface{}{
		"enabled":          true,
		"database_path":    es.config.EncryptedDBPath,
		"tx_stats":         stats.TxStats,
		"free_page_count":  stats.FreePageN,
		"pending_page_count": stats.PendingPageN,
		"free_alloc":       stats.FreeAlloc,
		"free_list_inuse":  stats.FreelistInuse,
	}
}

// Buckets for different types of sensitive data
const (
	JWTTokensBucket     = "jwt_tokens"
	APIKeysBucket       = "api_keys"
	UserCredsBucket     = "user_credentials"
	ConfigSecretsBucket = "config_secrets"
	AuditLogsBucket     = "audit_logs"
)
