package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/dsyorkd/pi-controller/internal/logger"
	pb "github.com/dsyorkd/pi-controller/proto"
)

// MetricsService handles system metrics collection and reporting
type MetricsService struct {
	logger logger.Interface
	pb.UnimplementedPiAgentServiceServer
}

// NewMetricsService creates a new metrics service instance
func NewMetricsService(logger logger.Interface) *MetricsService {
	return &MetricsService{
		logger: logger.WithField("component", "metrics-service"),
	}
}

// GetSystemMetrics returns current system metrics
func (m *MetricsService) GetSystemMetrics(ctx context.Context, req *pb.GetSystemMetricsRequest) (*pb.GetSystemMetricsResponse, error) {
	m.logger.Debug("collecting system metrics")

	metrics, err := m.collectMetrics(ctx)
	if err != nil {
		m.logger.Error("failed to collect system metrics", "error", err)
		return nil, fmt.Errorf("failed to collect system metrics: %w", err)
	}

	return &pb.GetSystemMetricsResponse{
		Metrics:   metrics,
		Timestamp: timestamppb.Now(),
	}, nil
}

// StreamSystemMetrics streams system metrics at regular intervals
func (m *MetricsService) StreamSystemMetrics(req *pb.StreamSystemMetricsRequest, stream pb.PiAgentService_StreamSystemMetricsServer) error {
	interval := time.Duration(req.IntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second // Default to 5 seconds
	}

	m.logger.Info("starting system metrics stream", "interval", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			m.logger.Debug("metrics stream context cancelled")
			return stream.Context().Err()
		case <-ticker.C:
			metrics, err := m.collectMetrics(stream.Context())
			if err != nil {
				m.logger.Error("failed to collect metrics for stream", "error", err)
				continue
			}

			response := &pb.SystemMetricsResponse{
				Metrics:   metrics,
				Timestamp: timestamppb.Now(),
			}

			if err := stream.Send(response); err != nil {
				m.logger.Error("failed to send metrics to stream", "error", err)
				return fmt.Errorf("failed to send metrics: %w", err)
			}
		}
	}
}

// collectMetrics gathers all system metrics
func (m *MetricsService) collectMetrics(ctx context.Context) (*pb.SystemMetrics, error) {
	metrics := &pb.SystemMetrics{}

	// Collect CPU metrics
	cpuMetrics, err := m.collectCPUMetrics(ctx)
	if err != nil {
		m.logger.Warn("failed to collect CPU metrics", "error", err)
	} else {
		metrics.Cpu = cpuMetrics
	}

	// Collect memory metrics
	memMetrics, err := m.collectMemoryMetrics()
	if err != nil {
		m.logger.Warn("failed to collect memory metrics", "error", err)
	} else {
		metrics.Memory = memMetrics
	}

	// Collect disk metrics
	diskMetrics, err := m.collectDiskMetrics()
	if err != nil {
		m.logger.Warn("failed to collect disk metrics", "error", err)
	} else {
		metrics.Disks = diskMetrics
	}

	// Collect network metrics
	netMetrics, err := m.collectNetworkMetrics()
	if err != nil {
		m.logger.Warn("failed to collect network metrics", "error", err)
	} else {
		metrics.Network = netMetrics
	}

	// Collect thermal metrics
	thermalMetrics, err := m.collectThermalMetrics()
	if err != nil {
		m.logger.Warn("failed to collect thermal metrics", "error", err)
	} else {
		metrics.Thermal = thermalMetrics
	}

	// Collect load metrics
	loadMetrics, err := m.collectLoadMetrics()
	if err != nil {
		m.logger.Warn("failed to collect load metrics", "error", err)
	} else {
		metrics.Load = loadMetrics
	}

	// Collect process metrics
	procMetrics, err := m.collectProcessMetrics()
	if err != nil {
		m.logger.Warn("failed to collect process metrics", "error", err)
	} else {
		metrics.Processes = procMetrics
	}

	return metrics, nil
}

// collectCPUMetrics gathers CPU usage statistics
func (m *MetricsService) collectCPUMetrics(ctx context.Context) (*pb.CPUMetrics, error) {
	// Get overall CPU usage
	cpuPercent, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU percent: %w", err)
	}

	// Get per-core CPU usage
	perCorePercent, err := cpu.PercentWithContext(ctx, time.Second, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get per-core CPU percent: %w", err)
	}

	// Get CPU times for detailed breakdown
	cpuTimes, err := cpu.TimesWithContext(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU times: %w", err)
	}

	var usage float64
	if len(cpuPercent) > 0 {
		usage = cpuPercent[0]
	}

	cpuMetrics := &pb.CPUMetrics{
		UsagePercent:  usage,
		PerCoreUsage:  perCorePercent,
		UserPercent:   0,
		SystemPercent: 0,
		IdlePercent:   0,
		IowaitPercent: 0,
	}

	// Calculate detailed CPU stats if available
	if len(cpuTimes) > 0 {
		times := cpuTimes[0]
		total := times.User + times.System + times.Idle + times.Iowait + times.Nice + times.Irq + times.Softirq + times.Steal
		if total > 0 {
			cpuMetrics.UserPercent = (times.User / total) * 100
			cpuMetrics.SystemPercent = (times.System / total) * 100
			cpuMetrics.IdlePercent = (times.Idle / total) * 100
			cpuMetrics.IowaitPercent = (times.Iowait / total) * 100
		}
	}

	return cpuMetrics, nil
}

// collectMemoryMetrics gathers memory usage statistics
func (m *MetricsService) collectMemoryMetrics() (*pb.MemoryMetrics, error) {
	vmStats, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual memory stats: %w", err)
	}

	swapStats, err := mem.SwapMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get swap memory stats: %w", err)
	}

	return &pb.MemoryMetrics{
		TotalBytes:        vmStats.Total,
		AvailableBytes:    vmStats.Available,
		UsedBytes:         vmStats.Used,
		FreeBytes:         vmStats.Free,
		CachedBytes:       vmStats.Cached,
		BuffersBytes:      vmStats.Buffers,
		UsagePercent:      vmStats.UsedPercent,
		SwapTotalBytes:    swapStats.Total,
		SwapUsedBytes:     swapStats.Used,
		SwapUsagePercent:  swapStats.UsedPercent,
	}, nil
}

// collectDiskMetrics gathers disk usage statistics
func (m *MetricsService) collectDiskMetrics() ([]*pb.DiskMetrics, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk partitions: %w", err)
	}

	var diskMetrics []*pb.DiskMetrics

	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			m.logger.Warn("failed to get disk usage for partition", "partition", partition.Device, "error", err)
			continue
		}

		diskMetric := &pb.DiskMetrics{
			Device:              partition.Device,
			Mountpoint:          partition.Mountpoint,
			Filesystem:          partition.Fstype,
			TotalBytes:          usage.Total,
			UsedBytes:           usage.Used,
			FreeBytes:           usage.Free,
			UsagePercent:        usage.UsedPercent,
			InodesTotal:         usage.InodesTotal,
			InodesUsed:          usage.InodesUsed,
			InodesFree:          usage.InodesFree,
			InodesUsagePercent:  usage.InodesUsedPercent,
		}

		diskMetrics = append(diskMetrics, diskMetric)
	}

	return diskMetrics, nil
}

// collectNetworkMetrics gathers network interface statistics
func (m *MetricsService) collectNetworkMetrics() ([]*pb.NetworkMetrics, error) {
	ioCounters, err := net.IOCounters(true)
	if err != nil {
		return nil, fmt.Errorf("failed to get network IO counters: %w", err)
	}

	var networkMetrics []*pb.NetworkMetrics

	for _, counter := range ioCounters {
		// Skip loopback interface
		if counter.Name == "lo" {
			continue
		}

		netMetric := &pb.NetworkMetrics{
			Interface:   counter.Name,
			BytesSent:   counter.BytesSent,
			BytesRecv:   counter.BytesRecv,
			PacketsSent: counter.PacketsSent,
			PacketsRecv: counter.PacketsRecv,
			ErrIn:       counter.Errin,
			ErrOut:      counter.Errout,
			DropIn:      counter.Dropin,
			DropOut:     counter.Dropout,
		}

		networkMetrics = append(networkMetrics, netMetric)
	}

	return networkMetrics, nil
}

// collectThermalMetrics gathers thermal information (especially important for Raspberry Pi)
func (m *MetricsService) collectThermalMetrics() (*pb.ThermalMetrics, error) {
	temps, err := host.SensorsTemperatures()
	if err != nil {
		return nil, fmt.Errorf("failed to get temperature sensors: %w", err)
	}

	var zones []*pb.ThermalZone

	for _, temp := range temps {
		zone := &pb.ThermalZone{
			Name:               temp.SensorKey,
			TemperatureCelsius: temp.Temperature,
			CriticalTemp:       temp.High,
			Status:             "ok", // Could be enhanced with actual status
		}

		zones = append(zones, zone)
	}

	return &pb.ThermalMetrics{
		Zones: zones,
	}, nil
}

// collectLoadMetrics gathers system load averages
func (m *MetricsService) collectLoadMetrics() (*pb.LoadMetrics, error) {
	loadAvg, err := load.Avg()
	if err != nil {
		return nil, fmt.Errorf("failed to get load averages: %w", err)
	}

	return &pb.LoadMetrics{
		Load1:  loadAvg.Load1,
		Load5:  loadAvg.Load5,
		Load15: loadAvg.Load15,
	}, nil
}

// collectProcessMetrics gathers process statistics
func (m *MetricsService) collectProcessMetrics() (*pb.ProcessMetrics, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to get processes: %w", err)
	}

	metrics := &pb.ProcessMetrics{
		Total: uint32(len(processes)),
	}

	// Count processes by status
	for _, proc := range processes {
		status, err := proc.Status()
		if err != nil {
			continue // Skip processes we can't read
		}

		// gopsutil returns status as []string, we need the first element
		if len(status) == 0 {
			continue
		}

		switch status[0] {
		case "R": // Running
			metrics.Running++
		case "S", "D", "I": // Sleeping (interruptible, uninterruptible, idle)
			metrics.Sleeping++
		case "T": // Stopped
			metrics.Stopped++
		case "Z": // Zombie
			metrics.Zombie++
		}
	}

	return metrics, nil
}