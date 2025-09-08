package discovery

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/mdns"
	"github.com/sirupsen/logrus"
)

// AdvertiserConfig represents mDNS advertiser configuration
type AdvertiserConfig struct {
	ServiceName string            `yaml:"service_name" mapstructure:"service_name"`
	ServiceType string            `yaml:"service_type" mapstructure:"service_type"`
	Domain      string            `yaml:"domain" mapstructure:"domain"`
	Port        int               `yaml:"port" mapstructure:"port"`
	HostName    string            `yaml:"hostname" mapstructure:"hostname"`
	TXTRecords  map[string]string `yaml:"txt_records" mapstructure:"txt_records"`
	TTL         time.Duration     `yaml:"ttl" mapstructure:"ttl"`
	Interface   string            `yaml:"interface" mapstructure:"interface"`
}

// DefaultAdvertiserConfig returns default advertiser configuration
func DefaultAdvertiserConfig() *AdvertiserConfig {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "pi-controller-agent"
	}

	return &AdvertiserConfig{
		ServiceName: hostname,
		ServiceType: "_pi-controller._tcp",
		Domain:      "local",
		Port:        9091,
		HostName:    hostname,
		TXTRecords: map[string]string{
			"version":      "1.0.0",
			"capabilities": "gpio,monitoring",
		},
		TTL: 3600 * time.Second, // 1 hour
	}
}

// Advertiser provides mDNS advertisement functionality for Pi Agents
type Advertiser struct {
	config   *AdvertiserConfig
	logger   *logrus.Entry
	server   *mdns.Server
	mu       sync.Mutex
	running  bool
	stopChan chan struct{}
}

// NewAdvertiser creates a new mDNS advertiser
func NewAdvertiser(config *AdvertiserConfig, logger *logrus.Logger) *Advertiser {
	if config == nil {
		config = DefaultAdvertiserConfig()
	}

	return &Advertiser{
		config:   config,
		logger:   logger.WithField("component", "mdns-advertiser"),
		stopChan: make(chan struct{}),
	}
}

// Start starts advertising the service via mDNS
func (a *Advertiser) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return fmt.Errorf("advertiser is already running")
	}

	// Get the primary IP address
	ip, err := a.getPrimaryIP()
	if err != nil {
		return fmt.Errorf("failed to get primary IP: %w", err)
	}

	// Prepare TXT records as string slice
	txtRecords := make([]string, 0, len(a.config.TXTRecords))
	for key, value := range a.config.TXTRecords {
		if value != "" {
			txtRecords = append(txtRecords, key+"="+value)
		} else {
			txtRecords = append(txtRecords, key)
		}
	}

	// Create service configuration
	service, err := mdns.NewMDNSService(
		a.config.ServiceName,
		a.config.ServiceType,
		a.config.Domain,
		a.config.HostName,
		a.config.Port,
		[]net.IP{ip},
		txtRecords,
	)
	if err != nil {
		return fmt.Errorf("failed to create mDNS service: %w", err)
	}

	// Create and configure the mDNS server
	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return fmt.Errorf("failed to create mDNS server: %w", err)
	}

	a.server = server
	a.running = true

	a.logger.WithFields(logrus.Fields{
		"service_name": a.config.ServiceName,
		"service_type": a.config.ServiceType,
		"port":         a.config.Port,
		"ip_address":   ip.String(),
		"txt_records":  len(txtRecords),
	}).Info("Started mDNS advertising")

	// Start monitoring for shutdown in a goroutine
	go a.monitorShutdown(ctx)

	return nil
}

// Stop stops the mDNS advertiser
func (a *Advertiser) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	a.logger.Info("Stopping mDNS advertising")

	// Signal shutdown
	close(a.stopChan)

	// Shutdown the mDNS server
	if a.server != nil {
		if err := a.server.Shutdown(); err != nil {
			a.logger.WithError(err).Error("Failed to shutdown mDNS server")
			return err
		}
		a.server = nil
	}

	a.running = false
	a.logger.Info("Stopped mDNS advertising")

	return nil
}

// UpdateTXTRecords updates the advertised TXT records
func (a *Advertiser) UpdateTXTRecords(records map[string]string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		// Just update the config if not running
		a.config.TXTRecords = records
		return nil
	}

	// Update the config
	a.config.TXTRecords = records

	// Restart advertising with new records
	if err := a.stopInternal(); err != nil {
		return fmt.Errorf("failed to stop for TXT record update: %w", err)
	}

	if err := a.startInternal(); err != nil {
		return fmt.Errorf("failed to restart after TXT record update: %w", err)
	}

	a.logger.WithField("records", len(records)).Info("Updated TXT records")
	return nil
}

// IsRunning returns whether the advertiser is currently running
func (a *Advertiser) IsRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.running
}

// GetConfig returns the current advertiser configuration
func (a *Advertiser) GetConfig() *AdvertiserConfig {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Return a copy to prevent external modification
	configCopy := *a.config
	txtCopy := make(map[string]string)
	for k, v := range a.config.TXTRecords {
		txtCopy[k] = v
	}
	configCopy.TXTRecords = txtCopy

	return &configCopy
}

// monitorShutdown monitors for context cancellation and stops the advertiser
func (a *Advertiser) monitorShutdown(ctx context.Context) {
	select {
	case <-ctx.Done():
		a.logger.Debug("Context cancelled, stopping advertiser")
		if err := a.Stop(); err != nil {
			a.logger.WithError(err).Error("Failed to stop advertiser on context cancellation")
		}
	case <-a.stopChan:
		// Already stopped
		return
	}
}

// getPrimaryIP gets the primary non-loopback IP address
func (a *Advertiser) getPrimaryIP() (net.IP, error) {
	// If a specific interface is configured, use it
	if a.config.Interface != "" {
		iface, err := net.InterfaceByName(a.config.Interface)
		if err != nil {
			return nil, fmt.Errorf("interface %s not found: %w", a.config.Interface, err)
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return nil, fmt.Errorf("failed to get addresses for interface %s: %w", a.config.Interface, err)
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
				return ipNet.IP, nil
			}
		}
		return nil, fmt.Errorf("no IPv4 address found on interface %s", a.config.Interface)
	}

	// Get all interfaces and find the best non-loopback IP
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	var candidateIPs []net.IP

	for _, iface := range interfaces {
		// Skip down and loopback interfaces
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
				ip := ipNet.IP.To4()
				if !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
					candidateIPs = append(candidateIPs, ip)
				}
			}
		}
	}

	if len(candidateIPs) == 0 {
		return nil, fmt.Errorf("no suitable IP address found")
	}

	// Prefer private IP addresses (192.168.x.x, 10.x.x.x, 172.16-31.x.x)
	for _, ip := range candidateIPs {
		if a.isPrivateIP(ip) {
			return ip, nil
		}
	}

	// Fallback to first available IP
	return candidateIPs[0], nil
}

// isPrivateIP checks if an IP address is in private address space
func (a *Advertiser) isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	ip = ip.To4()
	if ip == nil {
		return false
	}

	// Check private IP ranges
	private := []struct {
		start, end byte
	}{
		{10, 10},   // 10.0.0.0/8
		{172, 172}, // 172.16.0.0/12 (we'll check second octet below)
		{192, 192}, // 192.168.0.0/16
	}

	for _, p := range private {
		if ip[0] >= p.start && ip[0] <= p.end {
			if ip[0] == 172 {
				// Check if second octet is in range 16-31
				return ip[1] >= 16 && ip[1] <= 31
			}
			if ip[0] == 192 {
				// Check if second octet is 168
				return ip[1] == 168
			}
			return true // 10.x.x.x
		}
	}

	return false
}

// startInternal starts the mDNS server (assumes lock is held)
func (a *Advertiser) startInternal() error {
	ip, err := a.getPrimaryIP()
	if err != nil {
		return fmt.Errorf("failed to get primary IP: %w", err)
	}

	txtRecords := make([]string, 0, len(a.config.TXTRecords))
	for key, value := range a.config.TXTRecords {
		if value != "" {
			txtRecords = append(txtRecords, key+"="+value)
		} else {
			txtRecords = append(txtRecords, key)
		}
	}

	service, err := mdns.NewMDNSService(
		a.config.ServiceName,
		a.config.ServiceType,
		a.config.Domain,
		a.config.HostName,
		a.config.Port,
		[]net.IP{ip},
		txtRecords,
	)
	if err != nil {
		return fmt.Errorf("failed to create mDNS service: %w", err)
	}

	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return fmt.Errorf("failed to create mDNS server: %w", err)
	}

	a.server = server
	a.running = true

	return nil
}

// stopInternal stops the mDNS server (assumes lock is held)
func (a *Advertiser) stopInternal() error {
	if a.server != nil {
		if err := a.server.Shutdown(); err != nil {
			return err
		}
		a.server = nil
	}
	a.running = false
	return nil
}

// SetCapabilities updates the capabilities TXT record
func (a *Advertiser) SetCapabilities(capabilities []string) error {
	records := make(map[string]string)
	for k, v := range a.config.TXTRecords {
		records[k] = v
	}

	if len(capabilities) > 0 {
		records["capabilities"] = strings.Join(capabilities, ",")
	} else {
		delete(records, "capabilities")
	}

	return a.UpdateTXTRecords(records)
}

// SetVersion updates the version TXT record
func (a *Advertiser) SetVersion(version string) error {
	records := make(map[string]string)
	for k, v := range a.config.TXTRecords {
		records[k] = v
	}

	if version != "" {
		records["version"] = version
	} else {
		delete(records, "version")
	}

	return a.UpdateTXTRecords(records)
}
