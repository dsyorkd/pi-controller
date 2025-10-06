package storage

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultEncryptionConfig(t *testing.T) {
	config := DefaultEncryptionConfig()

	assert.False(t, config.Enabled)
	assert.Equal(t, "data/db.key", config.KeyFile)
	assert.Equal(t, "PI_CONTROLLER_DB_KEY", config.KeyFromEnv)
	assert.Equal(t, "data/encrypted.db", config.EncryptedDBPath)
	assert.Equal(t, 100000, config.PBKDF2Iterations)
	assert.True(t, config.GenerateKeyIfMissing)
}

func TestNewEncryptedStorage_Disabled(t *testing.T) {
	config := &EncryptionConfig{
		Enabled: false,
	}
	logger := logrus.New()

	storage, err := NewEncryptedStorage(config, logger)

	assert.NoError(t, err)
	assert.Nil(t, storage)
}

func TestNewEncryptedStorage_WithGeneratedKey(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "encryption-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := &EncryptionConfig{
		Enabled:              true,
		KeyFile:              filepath.Join(tmpDir, "test.key"),
		EncryptedDBPath:      filepath.Join(tmpDir, "test.db"),
		GenerateKeyIfMissing: true,
	}
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	storage, err := NewEncryptedStorage(config, logger)
	require.NoError(t, err)
	require.NotNil(t, storage)
	defer storage.Close()

	// Verify key file was created
	_, err = os.Stat(config.KeyFile)
	assert.NoError(t, err)

	// Verify database was created
	_, err = os.Stat(config.EncryptedDBPath)
	assert.NoError(t, err)
}

func TestNewEncryptedStorage_FromEnv(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "encryption-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Generate a test key
	testKey := make([]byte, 32)
	for i := range testKey {
		testKey[i] = byte(i)
	}
	encodedKey := base64.StdEncoding.EncodeToString(testKey)

	// Set environment variable
	envVar := "TEST_PI_CONTROLLER_KEY"
	os.Setenv(envVar, encodedKey)
	defer os.Unsetenv(envVar)

	config := &EncryptionConfig{
		Enabled:              true,
		KeyFromEnv:           envVar,
		EncryptedDBPath:      filepath.Join(tmpDir, "test.db"),
		GenerateKeyIfMissing: false,
	}
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	storage, err := NewEncryptedStorage(config, logger)
	require.NoError(t, err)
	require.NotNil(t, storage)
	defer storage.Close()
}

func TestNewEncryptedStorage_FromFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "encryption-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Generate and save a test key
	testKey := make([]byte, 32)
	for i := range testKey {
		testKey[i] = byte(i)
	}
	encodedKey := base64.StdEncoding.EncodeToString(testKey)

	keyFile := filepath.Join(tmpDir, "test.key")
	err = os.WriteFile(keyFile, []byte(encodedKey), 0600)
	require.NoError(t, err)

	config := &EncryptionConfig{
		Enabled:              true,
		KeyFile:              keyFile,
		EncryptedDBPath:      filepath.Join(tmpDir, "test.db"),
		GenerateKeyIfMissing: false,
	}
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	storage, err := NewEncryptedStorage(config, logger)
	require.NoError(t, err)
	require.NotNil(t, storage)
	defer storage.Close()
}

func TestEncryptedStorage_StoreAndRetrieve(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "encryption-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := &EncryptionConfig{
		Enabled:              true,
		KeyFile:              filepath.Join(tmpDir, "test.key"),
		EncryptedDBPath:      filepath.Join(tmpDir, "test.db"),
		GenerateKeyIfMissing: true,
	}
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	storage, err := NewEncryptedStorage(config, logger)
	require.NoError(t, err)
	require.NotNil(t, storage)
	defer storage.Close()

	// Store data
	testData := []byte("sensitive information")
	err = storage.Store("test-bucket", "test-key", testData)
	assert.NoError(t, err)

	// Retrieve data
	retrieved, err := storage.Retrieve("test-bucket", "test-key")
	assert.NoError(t, err)
	assert.Equal(t, testData, retrieved)
}

func TestEncryptedStorage_Delete(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "encryption-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := &EncryptionConfig{
		Enabled:              true,
		KeyFile:              filepath.Join(tmpDir, "test.key"),
		EncryptedDBPath:      filepath.Join(tmpDir, "test.db"),
		GenerateKeyIfMissing: true,
	}
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	storage, err := NewEncryptedStorage(config, logger)
	require.NoError(t, err)
	require.NotNil(t, storage)
	defer storage.Close()

	// Store data
	testData := []byte("data to delete")
	err = storage.Store("test-bucket", "test-key", testData)
	require.NoError(t, err)

	// Delete data
	err = storage.Delete("test-bucket", "test-key")
	assert.NoError(t, err)

	// Try to retrieve deleted data
	_, err = storage.Retrieve("test-bucket", "test-key")
	assert.Error(t, err)
}

func TestEncryptedStorage_List(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "encryption-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := &EncryptionConfig{
		Enabled:              true,
		KeyFile:              filepath.Join(tmpDir, "test.key"),
		EncryptedDBPath:      filepath.Join(tmpDir, "test.db"),
		GenerateKeyIfMissing: true,
	}
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	storage, err := NewEncryptedStorage(config, logger)
	require.NoError(t, err)
	require.NotNil(t, storage)
	defer storage.Close()

	// Store multiple items
	storage.Store("test-bucket", "key1", []byte("data1"))
	storage.Store("test-bucket", "key2", []byte("data2"))
	storage.Store("test-bucket", "key3", []byte("data3"))

	// List keys
	keys, err := storage.List("test-bucket")
	assert.NoError(t, err)
	assert.Len(t, keys, 3)
	assert.Contains(t, keys, "key1")
	assert.Contains(t, keys, "key2")
	assert.Contains(t, keys, "key3")
}

func TestEncryptedStorage_ListEmptyBucket(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "encryption-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := &EncryptionConfig{
		Enabled:              true,
		KeyFile:              filepath.Join(tmpDir, "test.key"),
		EncryptedDBPath:      filepath.Join(tmpDir, "test.db"),
		GenerateKeyIfMissing: true,
	}
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	storage, err := NewEncryptedStorage(config, logger)
	require.NoError(t, err)
	require.NotNil(t, storage)
	defer storage.Close()

	// List non-existent bucket
	keys, err := storage.List("non-existent-bucket")
	assert.NoError(t, err)
	assert.Empty(t, keys)
}

func TestEncryptedStorage_DeriveKeyFromPassword(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "encryption-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := &EncryptionConfig{
		Enabled:              true,
		KeyFile:              filepath.Join(tmpDir, "test.key"),
		EncryptedDBPath:      filepath.Join(tmpDir, "test.db"),
		GenerateKeyIfMissing: true,
		PBKDF2Iterations:     10000,
	}
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	storage, err := NewEncryptedStorage(config, logger)
	require.NoError(t, err)
	require.NotNil(t, storage)
	defer storage.Close()

	password := "test-password"
	salt := []byte("test-salt-123456")

	// Derive key
	key1 := storage.DeriveKeyFromPassword(password, salt)
	assert.Len(t, key1, 32)

	// Derive again with same password and salt - should be identical
	key2 := storage.DeriveKeyFromPassword(password, salt)
	assert.Equal(t, key1, key2)

	// Different password should give different key
	key3 := storage.DeriveKeyFromPassword("different-password", salt)
	assert.NotEqual(t, key1, key3)
}

func TestEncryptedStorage_GetStats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "encryption-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := &EncryptionConfig{
		Enabled:              true,
		KeyFile:              filepath.Join(tmpDir, "test.key"),
		EncryptedDBPath:      filepath.Join(tmpDir, "test.db"),
		GenerateKeyIfMissing: true,
	}
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	storage, err := NewEncryptedStorage(config, logger)
	require.NoError(t, err)
	require.NotNil(t, storage)
	defer storage.Close()

	stats := storage.GetStats()
	assert.NotNil(t, stats)
	assert.True(t, stats["enabled"].(bool))
	assert.NotEmpty(t, stats["database_path"])
}

func TestEncryptedStorage_EncryptDecrypt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "encryption-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := &EncryptionConfig{
		Enabled:              true,
		KeyFile:              filepath.Join(tmpDir, "test.key"),
		EncryptedDBPath:      filepath.Join(tmpDir, "test.db"),
		GenerateKeyIfMissing: true,
	}
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	storage, err := NewEncryptedStorage(config, logger)
	require.NoError(t, err)
	require.NotNil(t, storage)
	defer storage.Close()

	// Test data
	plaintext := []byte("This is sensitive data that needs encryption")

	// Encrypt
	ciphertext, err := storage.encrypt(plaintext)
	assert.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	// Decrypt
	decrypted, err := storage.decrypt(ciphertext)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptedStorage_RetrieveNonExistent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "encryption-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := &EncryptionConfig{
		Enabled:              true,
		KeyFile:              filepath.Join(tmpDir, "test.key"),
		EncryptedDBPath:      filepath.Join(tmpDir, "test.db"),
		GenerateKeyIfMissing: true,
	}
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	storage, err := NewEncryptedStorage(config, logger)
	require.NoError(t, err)
	require.NotNil(t, storage)
	defer storage.Close()

	// Try to retrieve non-existent key
	_, err = storage.Retrieve("test-bucket", "non-existent-key")
	assert.Error(t, err)
}

func TestEncryptedStorage_InvalidKeySize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "encryption-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create invalid key file (wrong size)
	invalidKey := make([]byte, 16) // Should be 32
	encodedKey := base64.StdEncoding.EncodeToString(invalidKey)

	keyFile := filepath.Join(tmpDir, "invalid.key")
	err = os.WriteFile(keyFile, []byte(encodedKey), 0600)
	require.NoError(t, err)

	config := &EncryptionConfig{
		Enabled:              true,
		KeyFile:              keyFile,
		EncryptedDBPath:      filepath.Join(tmpDir, "test.db"),
		GenerateKeyIfMissing: false,
	}
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	storage, err := NewEncryptedStorage(config, logger)
	assert.Error(t, err)
	assert.Nil(t, storage)
	assert.Contains(t, err.Error(), "must be 32 bytes")
}

func TestBucketConstants(t *testing.T) {
	assert.Equal(t, "jwt_tokens", JWTTokensBucket)
	assert.Equal(t, "api_keys", APIKeysBucket)
	assert.Equal(t, "user_credentials", UserCredsBucket)
	assert.Equal(t, "config_secrets", ConfigSecretsBucket)
	assert.Equal(t, "audit_logs", AuditLogsBucket)
}
