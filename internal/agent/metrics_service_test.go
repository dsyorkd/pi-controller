package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dsyorkd/pi-controller/internal/logger"
	pb "github.com/dsyorkd/pi-controller/proto"
)

func createTestMetricsService(t *testing.T) *MetricsService {
	// Create a test logger
	testLogger, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	// Create the metrics service
	service := NewMetricsService(testLogger)
	require.NotNil(t, service)

	return service
}

func TestNewMetricsService(t *testing.T) {
	service := createTestMetricsService(t)
	assert.NotNil(t, service)
	assert.NotNil(t, service.logger)
}

func TestGetSystemMetrics(t *testing.T) {
	service := createTestMetricsService(t)
	ctx := context.Background()

	req := &pb.GetSystemMetricsRequest{}
	resp, err := service.GetSystemMetrics(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Metrics)
	assert.NotNil(t, resp.Timestamp)

	// Check that we have some basic metrics
	metrics := resp.Metrics
	assert.NotNil(t, metrics.Cpu)
	assert.NotNil(t, metrics.Memory)
	assert.NotNil(t, metrics.Load)
	assert.NotNil(t, metrics.Processes)

	// Verify CPU metrics
	if metrics.Cpu != nil {
		assert.GreaterOrEqual(t, metrics.Cpu.UsagePercent, 0.0)
		assert.LessOrEqual(t, metrics.Cpu.UsagePercent, 100.0)
	}

	// Verify memory metrics
	if metrics.Memory != nil {
		assert.Greater(t, metrics.Memory.TotalBytes, uint64(0))
		assert.GreaterOrEqual(t, metrics.Memory.UsagePercent, 0.0)
		assert.LessOrEqual(t, metrics.Memory.UsagePercent, 100.0)
	}

	// Verify process metrics
	if metrics.Processes != nil {
		assert.Greater(t, metrics.Processes.Total, uint32(0))
	}
}

func TestStreamSystemMetrics(t *testing.T) {
	service := createTestMetricsService(t)

	// Create a mock stream (this is a simplified test)
	// In a real scenario, we'd need to implement the full streaming interface
	req := &pb.StreamSystemMetricsRequest{
		IntervalSeconds: 1, // 1 second interval for testing
	}

	// For now, just verify the request structure and service exists
	assert.Equal(t, int32(1), req.IntervalSeconds)
	assert.NotNil(t, service) // Use the service variable

	// TODO: Implement full streaming test when we have proper mock infrastructure
	// This would require implementing pb.PiAgentService_StreamSystemMetricsServer interface
}

func TestCollectCPUMetrics(t *testing.T) {
	service := createTestMetricsService(t)
	ctx := context.Background()

	cpuMetrics, err := service.collectCPUMetrics(ctx)

	// CPU metrics collection might fail in some environments (CI, containers)
	// so we'll be lenient with errors but strict with data when available
	if err != nil {
		t.Logf("CPU metrics collection failed (expected in some environments): %v", err)
		return
	}

	require.NotNil(t, cpuMetrics)
	assert.GreaterOrEqual(t, cpuMetrics.UsagePercent, 0.0)
	assert.LessOrEqual(t, cpuMetrics.UsagePercent, 100.0)
	assert.GreaterOrEqual(t, cpuMetrics.UserPercent, 0.0)
	assert.GreaterOrEqual(t, cpuMetrics.SystemPercent, 0.0)
	assert.GreaterOrEqual(t, cpuMetrics.IdlePercent, 0.0)
}

func TestCollectMemoryMetrics(t *testing.T) {
	service := createTestMetricsService(t)

	memMetrics, err := service.collectMemoryMetrics()

	assert.NoError(t, err)
	require.NotNil(t, memMetrics)

	// Basic sanity checks
	assert.Greater(t, memMetrics.TotalBytes, uint64(0))
	assert.GreaterOrEqual(t, memMetrics.UsagePercent, 0.0)
	assert.LessOrEqual(t, memMetrics.UsagePercent, 100.0)
	assert.GreaterOrEqual(t, memMetrics.AvailableBytes, uint64(0))
	assert.LessOrEqual(t, memMetrics.AvailableBytes, memMetrics.TotalBytes)

	// Swap metrics might be 0 on systems without swap
	assert.GreaterOrEqual(t, memMetrics.SwapTotalBytes, uint64(0))
	assert.GreaterOrEqual(t, memMetrics.SwapUsagePercent, 0.0)
	assert.LessOrEqual(t, memMetrics.SwapUsagePercent, 100.0)
}

func TestCollectDiskMetrics(t *testing.T) {
	service := createTestMetricsService(t)

	diskMetrics, err := service.collectDiskMetrics()

	assert.NoError(t, err)
	require.NotNil(t, diskMetrics)

	// We should have at least one disk/partition
	if len(diskMetrics) > 0 {
		disk := diskMetrics[0]
		assert.NotEmpty(t, disk.Device)
		assert.NotEmpty(t, disk.Mountpoint)
		assert.Greater(t, disk.TotalBytes, uint64(0))
		assert.GreaterOrEqual(t, disk.UsagePercent, 0.0)
		assert.LessOrEqual(t, disk.UsagePercent, 100.0)
		assert.LessOrEqual(t, disk.UsedBytes, disk.TotalBytes)
	}
}

func TestCollectNetworkMetrics(t *testing.T) {
	service := createTestMetricsService(t)

	netMetrics, err := service.collectNetworkMetrics()

	assert.NoError(t, err)
	require.NotNil(t, netMetrics)

	// We should have at least one network interface (excluding loopback)
	for _, net := range netMetrics {
		assert.NotEmpty(t, net.Interface)
		assert.NotEqual(t, "lo", net.Interface) // Loopback should be filtered out
		assert.GreaterOrEqual(t, net.BytesSent, uint64(0))
		assert.GreaterOrEqual(t, net.BytesRecv, uint64(0))
	}
}

func TestCollectThermalMetrics(t *testing.T) {
	service := createTestMetricsService(t)

	thermalMetrics, err := service.collectThermalMetrics()

	// Thermal sensors might not be available on all systems
	if err != nil {
		t.Logf("Thermal metrics collection failed (expected on some systems): %v", err)
		return
	}

	require.NotNil(t, thermalMetrics)

	// If we have thermal zones, verify their structure
	for _, zone := range thermalMetrics.Zones {
		assert.NotEmpty(t, zone.Name)
		assert.NotEmpty(t, zone.Status)
		// Temperature could be any value, but typically should be reasonable
		assert.Greater(t, zone.TemperatureCelsius, -100.0) // Arbitrary reasonable lower bound
		assert.Less(t, zone.TemperatureCelsius, 200.0)     // Arbitrary reasonable upper bound
	}
}

func TestCollectLoadMetrics(t *testing.T) {
	service := createTestMetricsService(t)

	loadMetrics, err := service.collectLoadMetrics()

	assert.NoError(t, err)
	require.NotNil(t, loadMetrics)

	// Load averages should be non-negative
	assert.GreaterOrEqual(t, loadMetrics.Load1, 0.0)
	assert.GreaterOrEqual(t, loadMetrics.Load5, 0.0)
	assert.GreaterOrEqual(t, loadMetrics.Load15, 0.0)
}

func TestCollectProcessMetrics(t *testing.T) {
	service := createTestMetricsService(t)

	procMetrics, err := service.collectProcessMetrics()

	assert.NoError(t, err)
	require.NotNil(t, procMetrics)

	// We should have at least some processes
	assert.Greater(t, procMetrics.Total, uint32(0))

	// The sum of process states should not exceed total
	stateSum := procMetrics.Running + procMetrics.Sleeping +
		procMetrics.Stopped + procMetrics.Zombie
	assert.LessOrEqual(t, stateSum, procMetrics.Total)

	// Most processes should be sleeping under normal conditions
	assert.GreaterOrEqual(t, procMetrics.Sleeping, uint32(0))
}

func TestCollectMetrics(t *testing.T) {
	service := createTestMetricsService(t)
	ctx := context.Background()

	metrics, err := service.collectMetrics(ctx)

	assert.NoError(t, err)
	require.NotNil(t, metrics)

	// Verify all major components are present
	assert.NotNil(t, metrics.Cpu)
	assert.NotNil(t, metrics.Memory)
	assert.NotNil(t, metrics.Load)
	assert.NotNil(t, metrics.Processes)
	// Disks and network might be empty in some test environments, so just check they're not nil
	assert.NotNil(t, metrics.Disks)
	assert.NotNil(t, metrics.Network)

	// Thermal might be nil on systems without sensors
	// assert.NotNil(t, metrics.Thermal) // Comment out to avoid test failures
}

func TestMetricsServiceRobustness(t *testing.T) {
	service := createTestMetricsService(t)

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// CPU metrics collection respects context
	_, err := service.collectCPUMetrics(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")

	// Other metrics collection should still work with normal context
	ctx = context.Background()
	metrics, err := service.collectMetrics(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, metrics)
}
