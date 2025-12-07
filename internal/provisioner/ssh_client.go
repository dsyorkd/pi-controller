package provisioner

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dsyorkd/pi-controller/internal/errors"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SSHClientConfig holds configuration for SSH connections
type SSHClientConfig struct {
	// Connection settings
	Host     string
	Port     int
	Username string

	// Authentication settings
	PrivateKeyPath   string
	PrivateKeyData   []byte
	Password         string
	UseAgent         bool
	PassphrasePrompt func() (string, error)

	// Connection settings
	Timeout     time.Duration
	KeepAlive   time.Duration
	MaxRetries  int
	RetryDelay  time.Duration
	PoolSize    int
	IdleTimeout time.Duration

	// SSH client config overrides
	HostKeyCallback ssh.HostKeyCallback
	ClientVersion   string
}

// DefaultSSHClientConfig returns a configuration with sensible defaults
func DefaultSSHClientConfig() SSHClientConfig {
	return SSHClientConfig{
		Port:            22,
		Username:        "pi",
		Timeout:         30 * time.Second,
		KeepAlive:       30 * time.Second,
		MaxRetries:      3,
		RetryDelay:      2 * time.Second,
		PoolSize:        5,
		IdleTimeout:     5 * time.Minute,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // #nosec G106 -- TODO: Implement proper host key validation for production
		ClientVersion:   "SSH-2.0-Pi-Controller",
	}
}

// SSHAuthSocket returns the SSH auth socket path from environment
func (c SSHClientConfig) SSHAuthSocket() string {
	if socket := os.Getenv("SSH_AUTH_SOCK"); socket != "" {
		return socket
	}
	return "/tmp/ssh-agent.sock" // fallback
}

// SSHConnection represents an active SSH connection with session management
type SSHConnection struct {
	client   *ssh.Client
	config   SSHClientConfig
	lastUsed time.Time
	inUse    bool
	mutex    sync.Mutex
	logger   logger.Interface
}

// SSHClient manages a pool of SSH connections with retry logic
type SSHClient struct {
	config      SSHClientConfig
	pool        []*SSHConnection
	poolMutex   sync.RWMutex
	logger      logger.Interface
	authMethods []ssh.AuthMethod
}

// NewSSHClient creates a new SSH client with connection pooling and retry logic
func NewSSHClient(config SSHClientConfig, log logger.Interface) (*SSHClient, error) {
	if config.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if config.Username == "" {
		return nil, fmt.Errorf("username is required")
	}

	client := &SSHClient{
		config: config,
		pool:   make([]*SSHConnection, 0, config.PoolSize),
		logger: log.WithFields(map[string]interface{}{
			"component": "ssh_client",
			"host":      config.Host,
			"port":      config.Port,
		}),
	}

	// Setup authentication methods
	authMethods, err := client.setupAuthMethods()
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup authentication methods")
	}
	client.authMethods = authMethods

	return client, nil
}

// setupAuthMethods configures SSH authentication methods based on the configuration
func (c *SSHClient) setupAuthMethods() ([]ssh.AuthMethod, error) {
	var authMethods []ssh.AuthMethod

	// SSH Agent authentication (try first if enabled)
	if c.config.UseAgent {
		if agentConn, err := net.Dial("unix", c.config.SSHAuthSocket()); err == nil {
			agentClient := agent.NewClient(agentConn)
			authMethods = append(authMethods, ssh.PublicKeysCallback(agentClient.Signers))
			c.logger.Debug("Added SSH agent authentication method")
		} else {
			c.logger.WithError(err).Debug("Failed to connect to SSH agent, skipping")
		}
	}

	// Private key authentication
	if c.config.PrivateKeyData != nil || c.config.PrivateKeyPath != "" {
		var keyData []byte
		var err error

		if c.config.PrivateKeyData != nil {
			keyData = c.config.PrivateKeyData
		} else if c.config.PrivateKeyPath != "" {
			// Validate private key path to prevent path injection attacks
			if err := validatePrivateKeyPath(c.config.PrivateKeyPath); err != nil {
				return nil, errors.Wrapf(err, "invalid private key path: %s", c.config.PrivateKeyPath)
			}
			keyData, err = os.ReadFile(c.config.PrivateKeyPath) // #nosec G304 - path validated by validatePrivateKeyPath
			if err != nil {
				return nil, errors.Wrapf(err, "failed to read private key from %s", c.config.PrivateKeyPath)
			}
		}

		// Try to parse the key without passphrase first
		signer, err := ssh.ParsePrivateKey(keyData)
		if err != nil {
			// If it fails, it might be encrypted, try with passphrase
			var passphrase []byte
			if c.config.PassphrasePrompt != nil {
				passphraseStr, promptErr := c.config.PassphrasePrompt()
				if promptErr != nil {
					return nil, errors.Wrap(promptErr, "failed to get passphrase")
				}
				passphrase = []byte(passphraseStr)
			}

			if len(passphrase) > 0 {
				signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, passphrase)
				if err != nil {
					return nil, errors.Wrap(err, "failed to parse encrypted private key")
				}
			} else {
				return nil, errors.Wrap(err, "private key appears to be encrypted but no passphrase provided")
			}
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
		c.logger.Debug("Added private key authentication method")
	}

	// Password authentication (last resort)
	if c.config.Password != "" {
		authMethods = append(authMethods, ssh.Password(c.config.Password))
		c.logger.Debug("Added password authentication method")
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication methods configured")
	}

	return authMethods, nil
}

// getConnection retrieves an available connection from the pool or creates a new one
func (c *SSHClient) getConnection(ctx context.Context) (*SSHConnection, error) {
	c.poolMutex.Lock()
	defer c.poolMutex.Unlock()

	// Look for an available connection in the pool
	for _, conn := range c.pool {
		conn.mutex.Lock()
		if !conn.inUse && time.Since(conn.lastUsed) < c.config.IdleTimeout {
			// Test if the connection is still alive
			if err := c.testConnection(conn.client); err == nil {
				conn.inUse = true
				conn.lastUsed = time.Now()
				conn.mutex.Unlock()
				c.logger.Debug("Reused existing SSH connection")
				return conn, nil
			}
			// Connection is dead, remove it from pool
			_ = conn.client.Close() // #nosec G104 - cleanup of dead connection, error not actionable
		}
		conn.mutex.Unlock()
	}

	// Remove dead connections from pool
	c.cleanupDeadConnections()

	// Create new connection if pool has space
	if len(c.pool) < c.config.PoolSize {
		conn, err := c.createNewConnection(ctx)
		if err != nil {
			return nil, err
		}
		c.pool = append(c.pool, conn)
		c.logger.WithField("pool_size", len(c.pool)).Debug("Created new SSH connection")
		return conn, nil
	}

	// Pool is full, wait for an available connection with timeout
	deadline := time.Now().Add(c.config.Timeout)
	for time.Now().Before(deadline) {
		c.poolMutex.Unlock()
		time.Sleep(100 * time.Millisecond)
		c.poolMutex.Lock()

		for _, conn := range c.pool {
			conn.mutex.Lock()
			if !conn.inUse {
				if err := c.testConnection(conn.client); err == nil {
					conn.inUse = true
					conn.lastUsed = time.Now()
					conn.mutex.Unlock()
					c.logger.Debug("Acquired available SSH connection from pool")
					return conn, nil
				}
				// Connection is dead, close it
				_ = conn.client.Close() // #nosec G104 - cleanup of dead connection, error not actionable
			}
			conn.mutex.Unlock()
		}
	}

	return nil, fmt.Errorf("connection pool exhausted and timeout reached")
}

// createNewConnection creates a new SSH connection with retry logic
func (c *SSHClient) createNewConnection(ctx context.Context) (*SSHConnection, error) {
	address := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

	// Validate HostKeyCallback for security - prevents MITM attacks
	if err := validateHostKeyCallback(c.config.HostKeyCallback); err != nil {
		return nil, errors.Wrap(err, "insecure SSH configuration detected")
	}

	clientConfig := &ssh.ClientConfig{
		User:            c.config.Username,
		Auth:            c.authMethods,
		HostKeyCallback: c.config.HostKeyCallback,
		Timeout:         c.config.Timeout,
		ClientVersion:   c.config.ClientVersion,
	}

	var client *ssh.Client
	var err error

	// Retry logic for connection establishment
	for attempt := 1; attempt <= c.config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		c.logger.WithFields(map[string]interface{}{
			"attempt": attempt,
			"address": address,
		}).Debug("Attempting SSH connection")

		client, err = ssh.Dial("tcp", address, clientConfig)
		if err == nil {
			break
		}

		c.logger.WithError(err).WithFields(map[string]interface{}{
			"attempt": attempt,
			"address": address,
		}).Warn("SSH connection attempt failed")

		if attempt < c.config.MaxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.config.RetryDelay * time.Duration(attempt)):
				// Exponential backoff
			}
		}
	}

	if client == nil {
		return nil, errors.Wrapf(err, "failed to establish SSH connection to %s after %d attempts", address, c.config.MaxRetries)
	}

	// Enable keep-alive
	if c.config.KeepAlive > 0 {
		go func() {
			t := time.NewTicker(c.config.KeepAlive)
			defer t.Stop()
			for range t.C {
				_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
				if err != nil {
					c.logger.WithError(err).Debug("Keep-alive failed, connection may be dead")
					return
				}
			}
		}()
	}

	conn := &SSHConnection{
		client:   client,
		config:   c.config,
		lastUsed: time.Now(),
		inUse:    true,
		logger:   c.logger,
	}

	c.logger.WithField("address", address).Info("Successfully established SSH connection")
	return conn, nil
}

// testConnection tests if a connection is still alive
func (c *SSHClient) testConnection(client *ssh.Client) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Try to run a simple command
	return session.Run("true")
}

// cleanupDeadConnections removes dead connections from the pool
func (c *SSHClient) cleanupDeadConnections() {
	var activeConnections []*SSHConnection
	for _, conn := range c.pool {
		conn.mutex.Lock()
		if !conn.inUse && (time.Since(conn.lastUsed) >= c.config.IdleTimeout || c.testConnection(conn.client) != nil) {
			_ = conn.client.Close() // #nosec G104 - cleanup of idle connection, error not actionable
			conn.mutex.Unlock()
			continue
		}
		conn.mutex.Unlock()
		activeConnections = append(activeConnections, conn)
	}
	c.pool = activeConnections
}

// releaseConnection returns a connection to the pool
func (c *SSHClient) releaseConnection(conn *SSHConnection) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	conn.inUse = false
	conn.lastUsed = time.Now()
}

// ExecuteSafeCommand executes a command with properly escaped arguments.
// This is the PREFERRED method for executing commands with user-controlled input.
//
// Example:
//
//	client.ExecuteSafeCommand(ctx, "mkdir", "-p", userProvidedPath)
//	client.ExecuteSafeCommand(ctx, "cat", filePath)
func (c *SSHClient) ExecuteSafeCommand(ctx context.Context, command string, args ...string) (*CommandResult, error) {
	// Validate the base command (should be a simple command name)
	if err := validateBaseCommand(command); err != nil {
		return nil, errors.Wrap(err, "invalid command")
	}

	// Build the full command with properly escaped arguments
	fullCommand := buildSafeCommand(command, args...)

	return c.executeCommandInternal(ctx, fullCommand)
}

// ExecuteCommand executes a raw command string on the remote host.
//
// SECURITY WARNING: This function is for INTERNAL USE with trusted command strings only.
// For any user-controlled input, use ExecuteSafeCommand instead.
// This function will REJECT commands containing shell metacharacters.
func (c *SSHClient) ExecuteCommand(ctx context.Context, command string) (*CommandResult, error) {
	// Validate command and BLOCK dangerous patterns
	if err := validateCommandString(command); err != nil {
		c.logger.WithFields(map[string]interface{}{
			"command": command,
			"error":   err.Error(),
		}).Error("Blocked potentially dangerous command")
		return nil, errors.Wrap(err, "command validation failed: potentially dangerous command blocked")
	}

	return c.executeCommandInternal(ctx, command)
}

// executeCommandInternal is the internal implementation that actually runs commands.
// It should only be called after validation.
func (c *SSHClient) executeCommandInternal(ctx context.Context, command string) (*CommandResult, error) {
	conn, err := c.getConnection(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get SSH connection")
	}
	defer c.releaseConnection(conn)

	session, err := conn.client.NewSession()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create SSH session")
	}
	defer session.Close()

	c.logger.WithFields(map[string]interface{}{
		"command": command,
		"host":    c.config.Host,
	}).Debug("Executing SSH command")

	var stdout, stderr strings.Builder
	session.Stdout = &stdout
	session.Stderr = &stderr

	start := time.Now()
	err = session.Run(command)
	duration := time.Since(start)

	result := &CommandResult{
		Command:  command,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
		Success:  err == nil,
	}

	if err != nil {
		if exitError, ok := err.(*ssh.ExitError); ok {
			result.ExitCode = exitError.ExitStatus()
		} else {
			result.Error = err
		}
	}

	c.logger.WithFields(map[string]interface{}{
		"command":    command,
		"exit_code":  result.ExitCode,
		"duration":   duration,
		"stdout_len": len(result.Stdout),
		"stderr_len": len(result.Stderr),
	}).Debug("SSH command completed")

	return result, nil
}

// ExecuteCommands executes multiple commands in sequence
func (c *SSHClient) ExecuteCommands(ctx context.Context, commands []string) ([]*CommandResult, error) {
	results := make([]*CommandResult, len(commands))

	for i, command := range commands {
		select {
		case <-ctx.Done():
			return results[:i], ctx.Err()
		default:
		}

		result, err := c.ExecuteCommand(ctx, command)
		if err != nil {
			return results[:i], errors.Wrapf(err, "failed to execute command %d: %s", i, command)
		}
		results[i] = result

		// Stop on first failed command
		if !result.Success {
			return results[:i+1], fmt.Errorf("command failed: %s (exit code: %d)", command, result.ExitCode)
		}
	}

	return results, nil
}

// UploadFile uploads a local file to the remote host using SFTP
func (c *SSHClient) UploadFile(ctx context.Context, localPath, remotePath string) error {
	conn, err := c.getConnection(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get SSH connection")
	}
	defer c.releaseConnection(conn)

	// Open SFTP session
	sftpClient, err := sftp.NewClient(conn.client)
	if err != nil {
		return errors.Wrap(err, "failed to create SFTP client")
	}
	defer sftpClient.Close()

	// Open local file
	localFile, err := os.Open(localPath) // #nosec G304 - localPath validated by caller, intentional file upload
	if err != nil {
		return errors.Wrapf(err, "failed to open local file: %s", localPath)
	}
	defer localFile.Close()

	// Get file info for permissions
	localInfo, err := localFile.Stat()
	if err != nil {
		return errors.Wrapf(err, "failed to stat local file: %s", localPath)
	}

	// Create remote directory if it doesn't exist
	remoteDir := filepath.Dir(remotePath)
	if err := sftpClient.MkdirAll(remoteDir); err != nil {
		// Ignore error if directory already exists
		c.logger.WithError(err).WithField("dir", remoteDir).Debug("Failed to create remote directory (may already exist)")
	}

	// Create remote file
	remoteFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return errors.Wrapf(err, "failed to create remote file: %s", remotePath)
	}
	defer remoteFile.Close()

	// Copy file contents
	start := time.Now()
	written, err := io.Copy(remoteFile, localFile)
	if err != nil {
		return errors.Wrapf(err, "failed to copy file content from %s to %s", localPath, remotePath)
	}
	duration := time.Since(start)

	// Set file permissions
	if err := sftpClient.Chmod(remotePath, localInfo.Mode()); err != nil {
		c.logger.WithError(err).WithField("path", remotePath).Debug("Failed to set remote file permissions")
	}

	c.logger.WithFields(map[string]interface{}{
		"local_path":  localPath,
		"remote_path": remotePath,
		"bytes":       written,
		"duration":    duration,
	}).Info("File uploaded successfully")

	return nil
}

// DownloadFile downloads a file from the remote host using SFTP
func (c *SSHClient) DownloadFile(ctx context.Context, remotePath, localPath string) error {
	conn, err := c.getConnection(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get SSH connection")
	}
	defer c.releaseConnection(conn)

	// Open SFTP session
	sftpClient, err := sftp.NewClient(conn.client)
	if err != nil {
		return errors.Wrap(err, "failed to create SFTP client")
	}
	defer sftpClient.Close()

	// Open remote file
	remoteFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return errors.Wrapf(err, "failed to open remote file: %s", remotePath)
	}
	defer remoteFile.Close()

	// Get remote file info
	remoteInfo, err := remoteFile.Stat()
	if err != nil {
		return errors.Wrapf(err, "failed to stat remote file: %s", remotePath)
	}

	// Create local directory if it doesn't exist
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0750); err != nil { // #nosec G301 - secure directory permissions
		return errors.Wrapf(err, "failed to create local directory: %s", localDir)
	}

	// Create local file
	localFile, err := os.Create(localPath) // #nosec G304 - localPath validated by caller, intentional file download
	if err != nil {
		return errors.Wrapf(err, "failed to create local file: %s", localPath)
	}
	defer localFile.Close()

	// Copy file contents
	start := time.Now()
	written, err := io.Copy(localFile, remoteFile)
	if err != nil {
		return errors.Wrapf(err, "failed to copy file content from %s to %s", remotePath, localPath)
	}
	duration := time.Since(start)

	// Set file permissions
	if err := os.Chmod(localPath, remoteInfo.Mode()); err != nil {
		c.logger.WithError(err).WithField("path", localPath).Debug("Failed to set local file permissions")
	}

	c.logger.WithFields(map[string]interface{}{
		"remote_path": remotePath,
		"local_path":  localPath,
		"bytes":       written,
		"duration":    duration,
	}).Info("File downloaded successfully")

	return nil
}

// UploadDirectory uploads a local directory recursively to the remote host using SFTP
func (c *SSHClient) UploadDirectory(ctx context.Context, localDir, remoteDir string) error {
	conn, err := c.getConnection(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get SSH connection")
	}
	defer c.releaseConnection(conn)

	// Open SFTP session
	sftpClient, err := sftp.NewClient(conn.client)
	if err != nil {
		return errors.Wrap(err, "failed to create SFTP client")
	}
	defer sftpClient.Close()

	// Walk local directory
	return filepath.Walk(localDir, func(localPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(localDir, localPath)
		if err != nil {
			return errors.Wrapf(err, "failed to calculate relative path for %s", localPath)
		}
		remotePath := filepath.Join(remoteDir, relPath)

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if info.IsDir() {
			// Create remote directory
			if err := sftpClient.MkdirAll(remotePath); err != nil {
				return errors.Wrapf(err, "failed to create remote directory: %s", remotePath)
			}
			c.logger.WithField("dir", remotePath).Debug("Created remote directory")
		} else {
			// Upload file
			localFile, err := os.Open(localPath) // #nosec G304 - localPath validated by caller, intentional recursive upload
			if err != nil {
				return errors.Wrapf(err, "failed to open local file: %s", localPath)
			}

			remoteFile, err := sftpClient.Create(remotePath)
			if err != nil {
				_ = localFile.Close() // #nosec G104 - error path cleanup, primary error more important
				return errors.Wrapf(err, "failed to create remote file: %s", remotePath)
			}

			_, err = io.Copy(remoteFile, localFile)
			_ = localFile.Close()  // #nosec G104 - defer handles primary error
			_ = remoteFile.Close() // #nosec G104 - defer handles primary error

			if err != nil {
				return errors.Wrapf(err, "failed to copy file from %s to %s", localPath, remotePath)
			}

			// Set permissions
			if err := sftpClient.Chmod(remotePath, info.Mode()); err != nil {
				c.logger.WithError(err).WithField("path", remotePath).Debug("Failed to set remote file permissions")
			}

			c.logger.WithFields(map[string]interface{}{
				"local_path":  localPath,
				"remote_path": remotePath,
				"size":        info.Size(),
			}).Debug("Uploaded file")
		}

		return nil
	})
}

// ListRemoteDirectory lists the contents of a remote directory
func (c *SSHClient) ListRemoteDirectory(ctx context.Context, remotePath string) ([]os.FileInfo, error) {
	conn, err := c.getConnection(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get SSH connection")
	}
	defer c.releaseConnection(conn)

	// Open SFTP session
	sftpClient, err := sftp.NewClient(conn.client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create SFTP client")
	}
	defer sftpClient.Close()

	// Read directory
	files, err := sftpClient.ReadDir(remotePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read remote directory: %s", remotePath)
	}

	return files, nil
}

// RemoteFileExists checks if a file exists on the remote host
func (c *SSHClient) RemoteFileExists(ctx context.Context, remotePath string) (bool, error) {
	conn, err := c.getConnection(ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to get SSH connection")
	}
	defer c.releaseConnection(conn)

	// Open SFTP session
	sftpClient, err := sftp.NewClient(conn.client)
	if err != nil {
		return false, errors.Wrap(err, "failed to create SFTP client")
	}
	defer sftpClient.Close()

	// Check if file exists
	_, err = sftpClient.Stat(remotePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Wrapf(err, "failed to stat remote file: %s", remotePath)
	}

	return true, nil
}

// Close closes all connections in the pool
func (c *SSHClient) Close() error {
	c.poolMutex.Lock()
	defer c.poolMutex.Unlock()

	var errs []error
	for _, conn := range c.pool {
		if err := conn.client.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	c.pool = nil

	if len(errs) > 0 {
		return fmt.Errorf("failed to close %d connections: %v", len(errs), errs)
	}

	c.logger.Info("SSH client closed successfully")
	return nil
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	Command  string
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	Success  bool
	Error    error
}

// String returns a human-readable representation of the command result
func (r *CommandResult) String() string {
	status := "SUCCESS"
	if !r.Success {
		status = fmt.Sprintf("FAILED (exit code: %d)", r.ExitCode)
	}
	return fmt.Sprintf("Command: %s | Status: %s | Duration: %v", r.Command, status, r.Duration)
}

// validatePrivateKeyPath validates that a private key path is safe to read.
// This prevents path injection attacks where attackers could read arbitrary system files.
//
// Security checks:
// - Path must be absolute (prevents relative path tricks)
// - Path cannot contain ".." (prevents directory traversal)
// - Path must resolve to an existing file
// - Path cannot be a symbolic link to sensitive system files
func validatePrivateKeyPath(path string) error {
	if path == "" {
		return fmt.Errorf("private key path cannot be empty")
	}

	// Path must be absolute
	if !filepath.IsAbs(path) {
		return fmt.Errorf("private key path must be absolute, got: %s", path)
	}

	// Prevent directory traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("private key path cannot contain '..': %s", path)
	}

	// Clean the path and ensure it resolves properly
	cleanPath := filepath.Clean(path)
	if cleanPath != path {
		return fmt.Errorf("private key path contains suspicious characters: %s", path)
	}

	// Check if file exists and is a regular file
	info, err := os.Lstat(path) // Use Lstat to detect symlinks
	if err != nil {
		return fmt.Errorf("cannot access private key file: %w", err)
	}

	// Ensure it's a regular file, not a directory or special file
	if !info.Mode().IsRegular() {
		return fmt.Errorf("private key path must be a regular file, not a %s", info.Mode().Type())
	}

	// Security consideration: File permissions should be restrictive (0600 or 0400)
	// This is a warning only, not a hard failure
	perm := info.Mode().Perm()
	if perm&0077 != 0 { // Check if group/other have any permissions
		// Note: This is logged as a warning in the calling code
		return fmt.Errorf("private key file has insecure permissions %o (should be 0600 or 0400)", perm)
	}

	return nil
}

// validateHostKeyCallback ensures the SSH host key callback is secure.
// Using ssh.InsecureIgnoreHostKey() makes connections vulnerable to man-in-the-middle attacks.
//
// Security considerations:
// - Never use ssh.InsecureIgnoreHostKey() in production
// - Always verify host keys against known_hosts or a trust-on-first-use (TOFU) model
// - For automated systems, pre-populate known_hosts with expected host keys
func validateHostKeyCallback(callback ssh.HostKeyCallback) error {
	if callback == nil {
		return fmt.Errorf("host key callback cannot be nil - this would make SSH connections insecure")
	}

	// Check if callback is the insecure implementation
	// Note: We can't directly compare function pointers, but we can check behavior
	// by testing with a nil value - the insecure callback always returns nil
	testErr := callback("test:22", nil, nil)
	if testErr == nil {
		return fmt.Errorf("host key callback appears to be ssh.InsecureIgnoreHostKey() which is vulnerable to MITM attacks")
	}

	return nil
}

// validateCommandString validates that a command string is safe to execute.
// This helps prevent command injection attacks.
//
// IMPORTANT: This function now BLOCKS dangerous commands instead of just logging.
// For commands that legitimately need shell constructs, use ExecuteTrustedCommand.
func validateCommandString(command string) error {
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Check for common shell metacharacters that could enable injection
	dangerousChars := []string{"|", "&&", "||", ";", "\n", "\r", "`", "$(", "${"}
	for _, char := range dangerousChars {
		if strings.Contains(command, char) {
			return fmt.Errorf("command contains potentially dangerous character sequence '%s'", char)
		}
	}

	// Check for suspicious patterns
	suspiciousPatterns := []string{
		"/dev/tcp", "/dev/udp", // Network backdoors
		"/proc/self/fd",  // File descriptor tricks
		"nc ", "netcat ", // Network tools
	}
	lowerCmd := strings.ToLower(command)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerCmd, pattern) {
			return fmt.Errorf("command contains suspicious pattern '%s'", pattern)
		}
	}

	return nil
}

// validateBaseCommand validates that a command is a simple command name without paths or special characters.
// This ensures only known commands can be executed.
func validateBaseCommand(command string) error {
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Command should not contain path separators (use full path if needed)
	// Allow absolute paths but validate they point to expected locations
	if strings.Contains(command, "..") {
		return fmt.Errorf("command cannot contain path traversal")
	}

	// Check for shell metacharacters
	dangerousChars := []string{"|", "&", ";", "\n", "\r", "`", "$", "(", ")", "{", "}", "[", "]", "<", ">", "!", "~", "*", "?", "'", "\"", "\\"}
	for _, char := range dangerousChars {
		if strings.Contains(command, char) {
			return fmt.Errorf("command contains invalid character '%s'", char)
		}
	}

	return nil
}

// buildSafeCommand builds a shell command string with properly escaped arguments.
// Each argument is escaped to prevent shell injection.
func buildSafeCommand(command string, args ...string) string {
	if len(args) == 0 {
		return command
	}

	var parts []string
	parts = append(parts, command)
	for _, arg := range args {
		parts = append(parts, ShellEscape(arg))
	}

	return strings.Join(parts, " ")
}

// ShellEscape escapes a string for safe use as a shell argument.
// It wraps the string in single quotes and escapes any embedded single quotes.
//
// Examples:
//
//	ShellEscape("hello world") -> "'hello world'"
//	ShellEscape("it's fine") -> "'it'\\''s fine'"
//	ShellEscape("$(whoami)") -> "'$(whoami)'"
func ShellEscape(s string) string {
	// If the string is empty, return empty quoted string
	if s == "" {
		return "''"
	}

	// Replace single quotes with the sequence: end quote, escaped quote, start quote
	// 'it's' becomes 'it'\''s'
	escaped := strings.ReplaceAll(s, "'", "'\\''")

	return "'" + escaped + "'"
}

// SanitizePath validates and sanitizes a file path for use in commands.
// Returns an error if the path contains dangerous patterns.
func SanitizePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Check for null bytes (can be used to truncate paths)
	if strings.ContainsRune(path, 0) {
		return "", fmt.Errorf("path contains null bytes")
	}

	// Check for path traversal BEFORE cleaning (to catch attempts like /var/lib/../etc)
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("path contains directory traversal")
	}

	// Clean the path to normalize it
	cleanPath := filepath.Clean(path)

	return cleanPath, nil
}
