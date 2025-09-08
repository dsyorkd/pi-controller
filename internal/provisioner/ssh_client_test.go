package provisioner

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dsyorkd/pi-controller/internal/logger"
)

// MockSSHServer implements a minimal SSH server for testing
type MockSSHServer struct {
	listener   net.Listener
	config     *ssh.ServerConfig
	commands   map[string]CommandResponse
	files      map[string][]byte
	testLogger logger.Interface
}

// CommandResponse represents a mock command response
type CommandResponse struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// NewMockSSHServer creates a new mock SSH server for testing
func NewMockSSHServer(t *testing.T) *MockSSHServer {
	// Generate a test host key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	hostSigner, err := ssh.NewSignerFromKey(privateKey)
	require.NoError(t, err)

	config := &ssh.ServerConfig{
		NoClientAuth: true, // Allow connections without authentication for testing
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if string(pass) == "testpass" {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %s", c.User())
		},
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			return nil, nil // Accept any public key for testing
		},
	}
	config.AddHostKey(hostSigner)

	server := &MockSSHServer{
		config:     config,
		commands:   make(map[string]CommandResponse),
		files:      make(map[string][]byte),
		testLogger: logger.Default(),
	}

	return server
}

// SetCommandResponse sets the response for a specific command
func (s *MockSSHServer) SetCommandResponse(command string, response CommandResponse) {
	s.commands[command] = response
}

// SetFile sets the content for a file in the mock SFTP server
func (s *MockSSHServer) SetFile(path string, content []byte) {
	s.files[path] = content
}

// Start starts the mock SSH server
func (s *MockSSHServer) Start() error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	s.listener = listener

	go s.acceptConnections()
	return nil
}

// GetAddress returns the server address
func (s *MockSSHServer) GetAddress() string {
	return s.listener.Addr().String()
}

// Stop stops the mock SSH server
func (s *MockSSHServer) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// acceptConnections handles incoming SSH connections
func (s *MockSSHServer) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return // Server stopped
		}
		go s.handleConnection(conn)
	}
}

// handleConnection handles a single SSH connection
func (s *MockSSHServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		return
	}
	defer sshConn.Close()

	// Handle global requests
	go ssh.DiscardRequests(reqs)

	// Handle channels
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}

		go s.handleSession(channel, requests)
	}
}

// handleSession handles a single SSH session
func (s *MockSSHServer) handleSession(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()

	for req := range requests {
		switch req.Type {
		case "exec":
			command := string(req.Payload[4:]) // Skip the length prefix
			response, exists := s.commands[command]
			if !exists {
				response = CommandResponse{
					Stderr:   fmt.Sprintf("command not found: %s", command),
					ExitCode: 127,
				}
			}

			// Send stdout
			if response.Stdout != "" {
				channel.Write([]byte(response.Stdout))
			}

			// Send stderr
			if response.Stderr != "" {
				// For simplicity in testing, we'll just write stderr to the same channel
				// In a real implementation, you'd need to handle extended data properly
				channel.Write([]byte(response.Stderr))
			}

			// Send exit status
			channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{uint32(response.ExitCode)}))
			req.Reply(true, nil)
			return

		case "subsystem":
			if string(req.Payload[4:]) == "sftp" {
				req.Reply(true, nil)
				s.handleSFTP(channel)
				return
			}
			req.Reply(false, nil)

		default:
			req.Reply(false, nil)
		}
	}
}

// handleSFTP provides a minimal SFTP implementation for testing
func (s *MockSSHServer) handleSFTP(channel ssh.Channel) {
	// This is a very basic SFTP mock - in real tests you'd want to use
	// a proper SFTP server implementation or library
	defer channel.Close()

	// For now, just close the channel to simulate SFTP subsystem
	// In production tests, you'd implement the SFTP protocol
}

// generateTestKeyPair generates a test SSH key pair
func generateTestKeyPair(t *testing.T) ([]byte, ssh.Signer) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create private key in PEM format
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)

	// Create SSH signer
	signer, err := ssh.ParsePrivateKey(privateKeyBytes)
	require.NoError(t, err)

	return privateKeyBytes, signer
}

func TestSSHClientConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        SSHClientConfig
		expectedError string
		setupCallback func(*testing.T) SSHClientConfig
	}{
		{
			name: "valid configuration",
			setupCallback: func(t *testing.T) SSHClientConfig {
				config := DefaultSSHClientConfig()
				config.Host = "127.0.0.1"
				config.Username = "testuser"
				config.Password = "testpass"
				return config
			},
		},
		{
			name: "missing host",
			setupCallback: func(t *testing.T) SSHClientConfig {
				config := DefaultSSHClientConfig()
				config.Host = ""
				config.Username = "testuser"
				config.Password = "testpass"
				return config
			},
			expectedError: "host is required",
		},
		{
			name: "missing username",
			setupCallback: func(t *testing.T) SSHClientConfig {
				config := DefaultSSHClientConfig()
				config.Host = "127.0.0.1"
				config.Username = ""
				config.Password = "testpass"
				return config
			},
			expectedError: "username is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.setupCallback(t)
			client, err := NewSSHClient(config, logger.Default())

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				if client != nil {
					client.Close()
				}
			}
		})
	}
}

func TestSSHClientAuthMethods(t *testing.T) {
	tests := []struct {
		name            string
		setupConfig     func(*testing.T) SSHClientConfig
		expectedMethods int
		expectError     bool
	}{
		{
			name: "password authentication",
			setupConfig: func(t *testing.T) SSHClientConfig {
				config := DefaultSSHClientConfig()
				config.Host = "127.0.0.1"
				config.Username = "testuser"
				config.Password = "testpass"
				return config
			},
			expectedMethods: 1,
		},
		{
			name: "private key authentication",
			setupConfig: func(t *testing.T) SSHClientConfig {
				config := DefaultSSHClientConfig()
				config.Host = "127.0.0.1"
				config.Username = "testuser"
				privateKeyData, _ := generateTestKeyPair(t)
				config.PrivateKeyData = privateKeyData
				return config
			},
			expectedMethods: 1,
		},
		{
			name: "both password and key authentication",
			setupConfig: func(t *testing.T) SSHClientConfig {
				config := DefaultSSHClientConfig()
				config.Host = "127.0.0.1"
				config.Username = "testuser"
				config.Password = "testpass"
				privateKeyData, _ := generateTestKeyPair(t)
				config.PrivateKeyData = privateKeyData
				return config
			},
			expectedMethods: 2,
		},
		{
			name: "no authentication methods",
			setupConfig: func(t *testing.T) SSHClientConfig {
				config := DefaultSSHClientConfig()
				config.Host = "127.0.0.1"
				config.Username = "testuser"
				// No password or key
				return config
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.setupConfig(t)
			client, err := NewSSHClient(config, logger.Default())

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
				assert.Len(t, client.authMethods, tt.expectedMethods)
				client.Close()
			}
		})
	}
}

func TestSSHClientCommandExecution(t *testing.T) {
	// Start mock SSH server
	server := NewMockSSHServer(t)
	require.NoError(t, server.Start())
	defer server.Stop()

	// Set up expected commands
	server.SetCommandResponse("echo test", CommandResponse{
		Stdout:   "test\n",
		ExitCode: 0,
	})
	server.SetCommandResponse("false", CommandResponse{
		ExitCode: 1,
	})
	server.SetCommandResponse("echo error >&2", CommandResponse{
		Stdout:   "error\n", // Our mock writes stderr to stdout for simplicity
		ExitCode: 0,
	})

	// Parse server address
	host, portStr, err := net.SplitHostPort(server.GetAddress())
	require.NoError(t, err)

	// Create SSH client config
	config := DefaultSSHClientConfig()
	config.Host = host
	config.Port = parseInt(portStr)
	config.Username = "testuser"
	config.Password = "testpass"
	config.MaxRetries = 1
	config.RetryDelay = 100 * time.Millisecond

	client, err := NewSSHClient(config, logger.Default())
	require.NoError(t, err)
	defer client.Close()

	tests := []struct {
		name             string
		command          string
		expectedSuccess  bool
		expectedExitCode int
		expectStdout     string
		expectStderr     string
	}{
		{
			name:             "successful command",
			command:          "echo test",
			expectedSuccess:  true,
			expectedExitCode: 0,
			expectStdout:     "test\n",
		},
		{
			name:             "failed command",
			command:          "false",
			expectedSuccess:  false,
			expectedExitCode: 1,
		},
		{
			name:             "command with stderr",
			command:          "echo error >&2",
			expectedSuccess:  true,
			expectedExitCode: 0,
			// Note: Our mock server writes stderr to stdout for simplicity
			expectStdout: "error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := client.ExecuteCommand(ctx, tt.command)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.expectedSuccess, result.Success)
			assert.Equal(t, tt.expectedExitCode, result.ExitCode)
			assert.Equal(t, tt.command, result.Command)

			if tt.expectStdout != "" {
				assert.Equal(t, tt.expectStdout, result.Stdout)
			}
			if tt.expectStderr != "" {
				assert.Equal(t, tt.expectStderr, result.Stderr)
			}
		})
	}
}

func TestSSHClientMultipleCommands(t *testing.T) {
	// Start mock SSH server
	server := NewMockSSHServer(t)
	require.NoError(t, server.Start())
	defer server.Stop()

	// Set up expected commands
	commands := []string{"cmd1", "cmd2", "cmd3"}
	for i, cmd := range commands {
		server.SetCommandResponse(cmd, CommandResponse{
			Stdout:   fmt.Sprintf("output%d\n", i+1),
			ExitCode: 0,
		})
	}
	server.SetCommandResponse("failing_cmd", CommandResponse{
		ExitCode: 1,
	})

	// Parse server address
	host, portStr, err := net.SplitHostPort(server.GetAddress())
	require.NoError(t, err)

	// Create SSH client
	config := DefaultSSHClientConfig()
	config.Host = host
	config.Port = parseInt(portStr)
	config.Username = "testuser"
	config.Password = "testpass"

	client, err := NewSSHClient(config, logger.Default())
	require.NoError(t, err)
	defer client.Close()

	t.Run("successful commands", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		results, err := client.ExecuteCommands(ctx, commands)
		require.NoError(t, err)
		require.Len(t, results, 3)

		for i, result := range results {
			assert.True(t, result.Success)
			assert.Equal(t, 0, result.ExitCode)
			assert.Equal(t, commands[i], result.Command)
			assert.Equal(t, fmt.Sprintf("output%d\n", i+1), result.Stdout)
		}
	})

	t.Run("command sequence with failure", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		commandsWithFailure := []string{"cmd1", "failing_cmd", "cmd3"}
		results, err := client.ExecuteCommands(ctx, commandsWithFailure)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "command failed")
		assert.Len(t, results, 2) // Should stop after failure

		// First command should succeed
		assert.True(t, results[0].Success)
		assert.Equal(t, "cmd1", results[0].Command)

		// Second command should fail
		assert.False(t, results[1].Success)
		assert.Equal(t, "failing_cmd", results[1].Command)
		assert.Equal(t, 1, results[1].ExitCode)
	})
}

func TestSSHClientConnectionPooling(t *testing.T) {
	// Start mock SSH server
	server := NewMockSSHServer(t)
	require.NoError(t, server.Start())
	defer server.Stop()

	server.SetCommandResponse("test", CommandResponse{
		Stdout:   "test\n",
		ExitCode: 0,
	})

	// Parse server address
	host, portStr, err := net.SplitHostPort(server.GetAddress())
	require.NoError(t, err)

	// Create SSH client with small pool for testing
	config := DefaultSSHClientConfig()
	config.Host = host
	config.Port = parseInt(portStr)
	config.Username = "testuser"
	config.Password = "testpass"
	config.PoolSize = 2

	client, err := NewSSHClient(config, logger.Default())
	require.NoError(t, err)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute multiple commands concurrently
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()
			result, err := client.ExecuteCommand(ctx, "test")
			assert.NoError(t, err)
			assert.True(t, result.Success)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify pool size doesn't exceed limit
	client.poolMutex.RLock()
	assert.LessOrEqual(t, len(client.pool), config.PoolSize)
	client.poolMutex.RUnlock()
}

func TestCommandResult(t *testing.T) {
	tests := []struct {
		name           string
		result         *CommandResult
		expectedString string
	}{
		{
			name: "successful command",
			result: &CommandResult{
				Command:  "echo test",
				Success:  true,
				Duration: 100 * time.Millisecond,
			},
			expectedString: "Command: echo test | Status: SUCCESS | Duration: 100ms",
		},
		{
			name: "failed command",
			result: &CommandResult{
				Command:  "false",
				Success:  false,
				ExitCode: 1,
				Duration: 50 * time.Millisecond,
			},
			expectedString: "Command: false | Status: FAILED (exit code: 1) | Duration: 50ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedString, tt.result.String())
		})
	}
}

func TestSSHClientTimeout(t *testing.T) {
	// Create SSH client with very short timeout
	config := DefaultSSHClientConfig()
	config.Host = "192.0.2.1" // Non-routable address (RFC 5737)
	config.Port = 22
	config.Username = "testuser"
	config.Password = "testpass"
	config.Timeout = 100 * time.Millisecond
	config.MaxRetries = 1

	client, err := NewSSHClient(config, logger.Default())
	require.NoError(t, err)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = client.ExecuteCommand(ctx, "echo test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to establish SSH connection")
}

func TestSSHAuthSocket(t *testing.T) {
	config := DefaultSSHClientConfig()

	// Test environment variable first
	originalValue := os.Getenv("SSH_AUTH_SOCK")
	defer func() {
		if originalValue != "" {
			os.Setenv("SSH_AUTH_SOCK", originalValue)
		} else {
			os.Unsetenv("SSH_AUTH_SOCK")
		}
	}()

	// Test with no environment variable set
	os.Unsetenv("SSH_AUTH_SOCK")
	socket := config.SSHAuthSocket()
	assert.Equal(t, "/tmp/ssh-agent.sock", socket)

	// Test with environment variable
	testSocket := "/tmp/test-ssh-agent.sock"
	os.Setenv("SSH_AUTH_SOCK", testSocket)

	socket = config.SSHAuthSocket()
	assert.Equal(t, testSocket, socket)
}

// Helper function to parse port string to int
func parseInt(s string) int {
	var port int
	fmt.Sscanf(s, "%d", &port)
	return port
}

// Benchmark tests
func BenchmarkSSHClientCommandExecution(b *testing.B) {
	server := NewMockSSHServer(&testing.T{})
	server.Start()
	defer server.Stop()

	server.SetCommandResponse("true", CommandResponse{ExitCode: 0})

	host, portStr, _ := net.SplitHostPort(server.GetAddress())
	config := DefaultSSHClientConfig()
	config.Host = host
	config.Port = parseInt(portStr)
	config.Username = "testuser"
	config.Password = "testpass"

	client, _ := NewSSHClient(config, logger.Default())
	defer client.Close()

	b.ResetTimer()
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		_, err := client.ExecuteCommand(ctx, "true")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Integration test for real SSH functionality (skipped by default)
func TestSSHClientIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test would connect to a real SSH server for integration testing
	// You would need to set up proper test credentials and server
	t.Skip("Integration test requires real SSH server setup")
}
