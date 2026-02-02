package services

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/dsyorkd/pi-controller/internal/config"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SSHExecutorImpl implements the SSHExecutor interface
type SSHExecutorImpl struct {
	config *config.SSHConfig
	logger logger.Interface
	signer ssh.Signer
}

// NewSSHExecutor creates a new SSH executor instance
func NewSSHExecutor(config *config.SSHConfig, logger logger.Interface) *SSHExecutorImpl {
	executor := &SSHExecutorImpl{
		config: config,
		logger: logger.WithField("component", "ssh-executor"),
	}

	// Load private key during initialization
	if err := executor.loadPrivateKey(); err != nil {
		logger.WithError(err).Warn("Failed to load SSH private key - SSH functionality may be limited or unavailable")
	}

	return executor
}

// loadPrivateKey loads the SSH private key from file or environment variable
func (s *SSHExecutorImpl) loadPrivateKey() error {
	var keyBytes []byte
	var err error

	// Try from environment variable first
	if s.config.PrivateKeyEnv != "" {
		if envKey := os.Getenv(s.config.PrivateKeyEnv); envKey != "" {
			keyBytes = []byte(envKey)
			s.logger.Debug("SSH private key loaded from environment variable")
		}
	}

	// Fallback to file if not in environment or if file is specified
	if len(keyBytes) == 0 && s.config.PrivateKeyPath != "" {
		keyBytes, err = os.ReadFile(s.config.PrivateKeyPath)
		if err != nil {
			return fmt.Errorf("failed to read private key file %s: %w", s.config.PrivateKeyPath, err)
		}
		s.logger.Debug("SSH private key loaded from file")
	}

	if len(keyBytes) == 0 {
		return fmt.Errorf("no SSH private key configured")
	}

	// Parse the private key
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	s.signer = signer
	return nil
}

// Execute executes a command on a remote node via SSH
func (s *SSHExecutorImpl) Execute(ctx context.Context, nodeIP string, command string) (string, error) {
	s.logger.WithFields(map[string]interface{}{
		"node_ip": nodeIP,
		"command": command,
	}).Debug("Executing SSH command")

	if s.signer == nil {
		return "", fmt.Errorf("SSH private key not loaded")
	}

	authMethods := []ssh.AuthMethod{
		ssh.PublicKeys(s.signer),
	}

	timeout, _ := time.ParseDuration(s.config.Timeout)
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Create SSH client config
	sshConfig := &ssh.ClientConfig{
		User: s.config.User,
		Auth: authMethods,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			if s.config.StrictHostKeyChecking {
				// Implement strict host key checking using known_hosts file
				kh, err := knownhosts.New(s.config.KnownHostsFile)
				if err != nil {
					s.logger.Warnf("Failed to load known_hosts file: %v. Continuing without strict checking.", err)
					return nil // Don't fail if known_hosts file itself fails to load
				}
				return kh(hostname, remote, key)
			}
			return nil // Insecure, but allows connections without host key setup
		},
		Timeout: timeout,
	}

	// Connect to SSH server
	conn, err := s.dialWithContext(ctx, "tcp", fmt.Sprintf("%s:%d", nodeIP, s.config.Port), sshConfig)
	if err != nil {
		return "", fmt.Errorf("failed to connect to SSH server at %s:%d: %w", nodeIP, s.config.Port, err)
	}
	defer conn.Close()

	// Create a session
	session, err := conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Run the command
	output, err := session.CombinedOutput(command)
	if err != nil {
		return "", fmt.Errorf("failed to execute command '%s': %s: %w", command, string(output), err)
	}

	s.logger.WithFields(map[string]interface{}{
		"node_ip": nodeIP,
		"command": command,
	}).Debug("SSH command executed successfully")

	return string(output), nil
}

// dialWithContext dials an address with context timeout
func (s *SSHExecutorImpl) dialWithContext(ctx context.Context, network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	conn, err := (&net.Dialer{Timeout: config.Timeout}).DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}

// CopyFile copies a file from local machine to remote node
func (s *SSHExecutorImpl) CopyFile(ctx context.Context, nodeIP string, localPath string, remotePath string) error {
	s.logger.WithFields(map[string]interface{}{
		"node_ip":     nodeIP,
		"local_path":  localPath,
		"remote_path": remotePath,
	}).Debug("Copying file via SSH/SFTP")

	if s.signer == nil {
		return fmt.Errorf("SSH private key not loaded")
	}

	authMethods := []ssh.AuthMethod{
		ssh.PublicKeys(s.signer),
	}

	timeout, _ := time.ParseDuration(s.config.Timeout)
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	sshConfig := &ssh.ClientConfig{
		User: s.config.User,
		Auth: authMethods,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			if s.config.StrictHostKeyChecking {
				kh, err := knownhosts.New(s.config.KnownHostsFile)
				if err != nil {
					s.logger.Warnf("Failed to load known_hosts file: %v. Continuing without strict checking.", err)
					return nil
				}
				return kh(hostname, remote, key)
			}
			return nil
		},
		Timeout: timeout,
	}

	conn, err := s.dialWithContext(ctx, "tcp", fmt.Sprintf("%s:%d", nodeIP, s.config.Port), sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server at %s:%d: %w", nodeIP, s.config.Port, err)
	}
	defer conn.Close()

	// Create new SFTP client
	client, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer client.Close()

	// Open local file
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file %s: %w", localPath, err)
	}
	defer localFile.Close()

	// Open remote file for writing
	remoteFile, err := client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file %s: %w", remotePath, err)
	}
	defer remoteFile.Close()

	// Copy data
	if _, err := io.Copy(remoteFile, localFile); err != nil {
		return fmt.Errorf("failed to copy file %s to %s: %w", localPath, remotePath, err)
	}

	s.logger.WithFields(map[string]interface{}{
		"node_ip":     nodeIP,
		"local_path":  localPath,
		"remote_path": remotePath,
	}).Debug("File copied successfully via SSH/SFTP")

	return nil
}

// CopyContent copies content to a file on remote node via SFTP
func (s *SSHExecutorImpl) CopyContent(ctx context.Context, nodeIP string, content string, remotePath string) error {
	s.logger.WithFields(map[string]interface{}{
		"node_ip":     nodeIP,
		"content_len": len(content),
		"remote_path": remotePath,
	}).Debug("Copying content via SSH/SFTP")

	if s.signer == nil {
		return fmt.Errorf("SSH private key not loaded")
	}

	authMethods := []ssh.AuthMethod{
		ssh.PublicKeys(s.signer),
	}

	timeout, _ := time.ParseDuration(s.config.Timeout)
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	sshConfig := &ssh.ClientConfig{
		User: s.config.User,
		Auth: authMethods,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			if s.config.StrictHostKeyChecking {
				kh, err := knownhosts.New(s.config.KnownHostsFile)
				if err != nil {
					s.logger.Warnf("Failed to load known_hosts file: %v. Continuing without strict checking.", err)
					return nil
				}
				return kh(hostname, remote, key)
			}
			return nil
		},
		Timeout: timeout,
	}

	conn, err := s.dialWithContext(ctx, "tcp", fmt.Sprintf("%s:%d", nodeIP, s.config.Port), sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server at %s:%d: %w", nodeIP, s.config.Port, err)
	}
	defer conn.Close()

	// Create new SFTP client
	client, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer client.Close()

	// Open the remote file for writing
	remoteFile, err := client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file %s: %w", remotePath, err)
	}
	defer remoteFile.Close()

	// Write content to the remote file
	if _, err := remoteFile.Write([]byte(content)); err != nil {
		return fmt.Errorf("failed to write content to remote file %s: %w", remotePath, err)
	}

	s.logger.WithFields(map[string]interface{}{
		"node_ip":     nodeIP,
		"remote_path": remotePath,
	}).Debug("Content copied successfully via SSH/SFTP")

	return nil
}
