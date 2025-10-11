package client

import (
	"testing"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPiAgentClientManager_GetClient(t *testing.T) {
	log := logger.Default()
	manager := NewPiAgentClientManager(log)

	node := &models.Node{
		ID:        1,
		Name:      "test-node",
		IPAddress: "192.168.1.100",
		Status:    models.NodeStatusReady,
	}

	// grpc.NewClient creates a lazy connection, so GetClient will succeed
	// even though no actual agent is running. The connection will fail
	// when the first RPC call is made.
	client, err := manager.GetClient(node)

	// Client creation should succeed (lazy connection)
	require.NoError(t, err)
	require.NotNil(t, client)

	// Clean up
	assert.NoError(t, manager.CloseAll())
}

func TestPiAgentClientManager_CloseAll(t *testing.T) {
	log := logger.Default()
	manager := NewPiAgentClientManager(log)

	// Should not error when closing with no connections
	err := manager.CloseAll()
	require.NoError(t, err)
}

func TestModelDirectionToProto(t *testing.T) {
	tests := []struct {
		input    models.GPIODirection
		expected string
	}{
		{models.GPIODirectionInput, "AGENT_GPIO_DIRECTION_INPUT"},
		{models.GPIODirectionOutput, "AGENT_GPIO_DIRECTION_OUTPUT"},
		{models.GPIODirection("invalid"), "AGENT_GPIO_DIRECTION_UNSPECIFIED"},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := modelDirectionToProto(tt.input)
			assert.Equal(t, tt.expected, result.String())
		})
	}
}

func TestModelPullModeToProto(t *testing.T) {
	tests := []struct {
		input    models.GPIOPullMode
		expected string
	}{
		{models.GPIOPullNone, "AGENT_GPIO_PULL_MODE_NONE"},
		{models.GPIOPullUp, "AGENT_GPIO_PULL_MODE_UP"},
		{models.GPIOPullDown, "AGENT_GPIO_PULL_MODE_DOWN"},
		{models.GPIOPullMode("invalid"), "AGENT_GPIO_PULL_MODE_UNSPECIFIED"},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := modelPullModeToProto(tt.input)
			assert.Equal(t, tt.expected, result.String())
		})
	}
}

func TestPiAgentClient_IsConnected(t *testing.T) {
	// Test with nil connection
	client := &PiAgentClient{
		conn: nil,
	}

	assert.False(t, client.IsConnected())
}

func TestPiAgentClientManager_CloseClient(t *testing.T) {
	log := logger.Default()
	manager := NewPiAgentClientManager(log)

	// Should not error when closing non-existent client
	err := manager.CloseClient(999)
	require.NoError(t, err)
}
