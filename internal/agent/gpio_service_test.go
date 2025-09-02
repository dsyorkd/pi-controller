package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dsyorkd/pi-controller/internal/logger"
	pb "github.com/dsyorkd/pi-controller/proto"
)

func createTestGPIOService(t *testing.T) *GPIOService {
	// Create a test logger
	testLogger, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	// Create the GPIO service
	service, err := NewGPIOService(testLogger)
	require.NoError(t, err)
	require.NotNil(t, service)

	return service
}

func TestNewGPIOService(t *testing.T) {
	service := createTestGPIOService(t)
	assert.NotNil(t, service)
	assert.NotNil(t, service.controller)
	assert.NotNil(t, service.logger)
}

func TestInitializeAndClose(t *testing.T) {
	service := createTestGPIOService(t)
	ctx := context.Background()

	// Test initialization
	err := service.Initialize(ctx)
	assert.NoError(t, err)

	// Test close
	err = service.Close()
	assert.NoError(t, err)
}

func TestAgentHealth(t *testing.T) {
	service := createTestGPIOService(t)
	ctx := context.Background()

	// Initialize service
	err := service.Initialize(ctx)
	require.NoError(t, err)
	defer service.Close()

	// Test health check
	req := &pb.AgentHealthRequest{}
	resp, err := service.AgentHealth(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Status)
	assert.NotEmpty(t, resp.Version)
	assert.NotNil(t, resp.Timestamp)
	assert.True(t, resp.GpioAvailable) // Should be true in mock mode
}

func TestGetSystemInfo(t *testing.T) {
	service := createTestGPIOService(t)
	ctx := context.Background()

	req := &pb.GetSystemInfoRequest{}
	resp, err := service.GetSystemInfo(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "pi-agent", resp.Hostname)
	assert.Equal(t, "linux", resp.Platform)
	assert.Equal(t, "arm64", resp.Architecture)
	assert.Equal(t, int32(4), resp.CpuCores)
	assert.Greater(t, resp.MemoryTotal, int64(0))
	assert.Greater(t, resp.MemoryAvailable, int64(0))
	assert.NotEmpty(t, resp.KernelVersion)
	assert.NotNil(t, resp.Timestamp)
}

func TestConfigureGPIOPin(t *testing.T) {
	service := createTestGPIOService(t)
	ctx := context.Background()

	// Initialize service
	err := service.Initialize(ctx)
	require.NoError(t, err)
	defer service.Close()

	// Test configuring a valid pin
	req := &pb.ConfigureGPIOPinRequest{
		Pin:       18,
		Direction: pb.AgentGPIODirection_AGENT_GPIO_DIRECTION_OUTPUT,
		PullMode:  pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_NONE,
	}

	resp, err := service.ConfigureGPIOPin(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.Message)
	assert.NotNil(t, resp.ConfiguredAt)
}

func TestWriteAndReadGPIOPin(t *testing.T) {
	service := createTestGPIOService(t)
	ctx := context.Background()

	// Initialize service
	err := service.Initialize(ctx)
	require.NoError(t, err)
	defer service.Close()

	// First configure the pin as output
	configReq := &pb.ConfigureGPIOPinRequest{
		Pin:       18,
		Direction: pb.AgentGPIODirection_AGENT_GPIO_DIRECTION_OUTPUT,
		PullMode:  pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_NONE,
	}

	configResp, err := service.ConfigureGPIOPin(ctx, configReq)
	require.NoError(t, err)
	require.True(t, configResp.Success)

	// Test writing HIGH
	writeReq := &pb.WriteGPIOPinRequest{
		Pin:   18,
		Value: 1,
	}

	writeResp, err := service.WriteGPIOPin(ctx, writeReq)
	assert.NoError(t, err)
	assert.NotNil(t, writeResp)
	assert.Equal(t, int32(18), writeResp.Pin)
	assert.Equal(t, int32(1), writeResp.Value)
	assert.NotNil(t, writeResp.Timestamp)

	// Test reading the pin
	readReq := &pb.ReadGPIOPinRequest{
		Pin: 18,
	}

	readResp, err := service.ReadGPIOPin(ctx, readReq)
	assert.NoError(t, err)
	assert.NotNil(t, readResp)
	assert.Equal(t, int32(18), readResp.Pin)
	assert.Equal(t, int32(1), readResp.Value) // Should read back HIGH
	assert.NotNil(t, readResp.Timestamp)
}

func TestSetGPIOPWM(t *testing.T) {
	service := createTestGPIOService(t)
	ctx := context.Background()

	// Initialize service
	err := service.Initialize(ctx)
	require.NoError(t, err)
	defer service.Close()

	// First configure the pin for PWM
	configReq := &pb.ConfigureGPIOPinRequest{
		Pin:           18,
		Direction:     pb.AgentGPIODirection_AGENT_GPIO_DIRECTION_OUTPUT,
		PullMode:      pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_NONE,
		PwmFrequency:  1000,
		PwmDutyCycle:  50,
	}

	configResp, err := service.ConfigureGPIOPin(ctx, configReq)
	require.NoError(t, err)
	require.True(t, configResp.Success)

	// Test setting PWM
	pwmReq := &pb.SetGPIOPWMRequest{
		Pin:       18,
		Frequency: 2000,
		DutyCycle: 75,
	}

	pwmResp, err := service.SetGPIOPWM(ctx, pwmReq)
	assert.NoError(t, err)
	assert.NotNil(t, pwmResp)
	assert.True(t, pwmResp.Success)
	assert.NotEmpty(t, pwmResp.Message)
	assert.Equal(t, int32(18), pwmResp.Pin)
	assert.Equal(t, int32(2000), pwmResp.Frequency)
	assert.Equal(t, int32(75), pwmResp.DutyCycle)
	assert.NotNil(t, pwmResp.ConfiguredAt)
}

func TestListConfiguredPins(t *testing.T) {
	service := createTestGPIOService(t)
	ctx := context.Background()

	// Initialize service
	err := service.Initialize(ctx)
	require.NoError(t, err)
	defer service.Close()

	// Initially should have no configured pins
	listReq := &pb.ListConfiguredPinsRequest{}
	listResp, err := service.ListConfiguredPins(ctx, listReq)
	assert.NoError(t, err)
	assert.NotNil(t, listResp)
	assert.Empty(t, listResp.Pins)

	// Configure a pin
	configReq := &pb.ConfigureGPIOPinRequest{
		Pin:       18,
		Direction: pb.AgentGPIODirection_AGENT_GPIO_DIRECTION_OUTPUT,
		PullMode:  pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_NONE,
	}

	configResp, err := service.ConfigureGPIOPin(ctx, configReq)
	require.NoError(t, err)
	require.True(t, configResp.Success)

	// Now should have one configured pin
	listResp, err = service.ListConfiguredPins(ctx, listReq)
	assert.NoError(t, err)
	assert.NotNil(t, listResp)
	assert.Len(t, listResp.Pins, 1)

	pin := listResp.Pins[0]
	assert.Equal(t, int32(18), pin.Pin)
	assert.Equal(t, pb.AgentGPIODirection_AGENT_GPIO_DIRECTION_OUTPUT, pin.Direction)
	assert.Equal(t, pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_NONE, pin.PullMode)
	assert.NotNil(t, pin.LastUpdated)
}

func TestConversionFunctions(t *testing.T) {
	// Test direction conversions
	tests := []struct {
		name   string
		pbDir  pb.AgentGPIODirection
		expect string
	}{
		{"input", pb.AgentGPIODirection_AGENT_GPIO_DIRECTION_INPUT, "input"},
		{"output", pb.AgentGPIODirection_AGENT_GPIO_DIRECTION_OUTPUT, "output"},
		{"unspecified", pb.AgentGPIODirection_AGENT_GPIO_DIRECTION_UNSPECIFIED, "input"}, // defaults to input
	}

	for _, tt := range tests {
		t.Run("direction_"+tt.name, func(t *testing.T) {
			result := convertDirection(tt.pbDir)
			assert.Equal(t, tt.expect, string(result))
		})
	}

	// Test pull mode conversions
	pullTests := []struct {
		name   string
		pbPull pb.AgentGPIOPullMode
		expect string
	}{
		{"none", pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_NONE, "none"},
		{"up", pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_UP, "up"},
		{"down", pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_DOWN, "down"},
		{"unspecified", pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_UNSPECIFIED, "none"}, // defaults to none
	}

	for _, tt := range pullTests {
		t.Run("pull_mode_"+tt.name, func(t *testing.T) {
			result := convertPullMode(tt.pbPull)
			assert.Equal(t, tt.expect, string(result))
		})
	}
}

func TestInvalidOperations(t *testing.T) {
	service := createTestGPIOService(t)
	ctx := context.Background()

	// Initialize service
	err := service.Initialize(ctx)
	require.NoError(t, err)
	defer service.Close()

	// Test reading unconfigured pin
	readReq := &pb.ReadGPIOPinRequest{
		Pin: 99, // Invalid pin
	}

	_, err = service.ReadGPIOPin(ctx, readReq)
	assert.Error(t, err) // Should fail for unconfigured pin

	// Test writing to unconfigured pin
	writeReq := &pb.WriteGPIOPinRequest{
		Pin:   99, // Invalid pin
		Value: 1,
	}

	_, err = service.WriteGPIOPin(ctx, writeReq)
	assert.Error(t, err) // Should fail for unconfigured pin
}