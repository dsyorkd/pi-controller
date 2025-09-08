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

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/dsyorkd/pi-controller/internal/errors"
	"github.com/dsyorkd/pi-controller/internal/logger"
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
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Implement proper host key validation
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
			keyData, err = os.ReadFile(c.config.PrivateKeyPath)
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
			conn.client.Close()
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
				conn.client.Close()
			}
			conn.mutex.Unlock()
		}
	}

	return nil, fmt.Errorf("connection pool exhausted and timeout reached")
}

// createNewConnection creates a new SSH connection with retry logic
func (c *SSHClient) createNewConnection(ctx context.Context) (*SSHConnection, error) {
	address := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

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
			for {
				select {
				case <-t.C:
					_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
					if err != nil {
						c.logger.WithError(err).Debug("Keep-alive failed, connection may be dead")
						return
					}
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
			conn.client.Close()
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

// ExecuteCommand executes a command on the remote host
func (c *SSHClient) ExecuteCommand(ctx context.Context, command string) (*CommandResult, error) {
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
	localFile, err := os.Open(localPath)
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
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return errors.Wrapf(err, "failed to create local directory: %s", localDir)
	}

	// Create local file
	localFile, err := os.Create(localPath)
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
			localFile, err := os.Open(localPath)
			if err != nil {
				return errors.Wrapf(err, "failed to open local file: %s", localPath)
			}

			remoteFile, err := sftpClient.Create(remotePath)
			if err != nil {
				localFile.Close()
				return errors.Wrapf(err, "failed to create remote file: %s", remotePath)
			}

			_, err = io.Copy(remoteFile, localFile)
			localFile.Close()
			remoteFile.Close()

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
