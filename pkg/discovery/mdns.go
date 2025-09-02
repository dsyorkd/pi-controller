// Package discovery provides node discovery services for Pi Controller
package discovery

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Node represents a discovered node
type Node struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	IPAddress   string            `json:"ip_address"`
	Port        int               `json:"port"`
	ServiceType string            `json:"service_type"`
	TXTRecords  map[string]string `json:"txt_records"`
	LastSeen    time.Time         `json:"last_seen"`
	Capabilities []string         `json:"capabilities"`
}

// NodeEventType represents the type of node discovery event
type NodeEventType string

const (
	NodeDiscovered NodeEventType = "discovered"
	NodeUpdated    NodeEventType = "updated"
	NodeLost       NodeEventType = "lost"
)

// NodeEvent represents a node discovery event
type NodeEvent struct {
	Type NodeEventType `json:"type"`
	Node Node          `json:"node"`
}

// NodeEventHandler is a function type for handling node events
type NodeEventHandler func(event NodeEvent)

// Config represents discovery service configuration
type Config struct {
	Enabled         bool     `yaml:"enabled" mapstructure:"enabled"`
	Method          string   `yaml:"method" mapstructure:"method"`
	Interface       string   `yaml:"interface" mapstructure:"interface"`
	Port            int      `yaml:"port" mapstructure:"port"`
	Interval        string   `yaml:"interval" mapstructure:"interval"`
	Timeout         string   `yaml:"timeout" mapstructure:"timeout"`
	StaticNodes     []string `yaml:"static_nodes" mapstructure:"static_nodes"`
	ServiceName     string   `yaml:"service_name" mapstructure:"service_name"`
	ServiceType     string   `yaml:"service_type" mapstructure:"service_type"`
}

// DefaultConfig returns default discovery configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:     true,
		Method:      "mdns",
		Port:        9091,
		Interval:    "30s",
		Timeout:     "5s",
		ServiceName: "pi-controller",
		ServiceType: "_pi-controller._tcp",
	}
}

// Service provides node discovery functionality
type Service struct {
	config        *Config
	logger        *logrus.Entry
	mu            sync.RWMutex
	nodes         map[string]*Node
	eventHandlers []NodeEventHandler
	running       bool
	stopChan      chan struct{}
	interval      time.Duration
	timeout       time.Duration
}

// NewService creates a new discovery service
func NewService(config *Config, logger *logrus.Logger) (*Service, error) {
	if config == nil {
		config = DefaultConfig()
	}

	interval, err := time.ParseDuration(config.Interval)
	if err != nil {
		return nil, fmt.Errorf("invalid interval: %w", err)
	}

	timeout, err := time.ParseDuration(config.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout: %w", err)
	}

	service := &Service{
		config:   config,
		logger:   logger.WithField("component", "discovery"),
		nodes:    make(map[string]*Node),
		interval: interval,
		timeout:  timeout,
		stopChan: make(chan struct{}),
	}

	return service, nil
}

// Start starts the discovery service
func (s *Service) Start(ctx context.Context) error {
	if !s.config.Enabled {
		s.logger.Info("Discovery service disabled")
		return nil
	}

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("discovery service is already running")
	}
	s.running = true
	s.mu.Unlock()

	s.logger.WithFields(logrus.Fields{
		"method":   s.config.Method,
		"interval": s.config.Interval,
	}).Info("Starting discovery service")

	// Load static nodes if configured
	if len(s.config.StaticNodes) > 0 {
		s.loadStaticNodes()
	}

	// Start discovery based on method
	switch s.config.Method {
	case "mdns":
		go s.runMDNSDiscovery(ctx)
	case "scan":
		go s.runNetworkScan(ctx)
	case "static":
		s.logger.Info("Using static node discovery only")
	default:
		return fmt.Errorf("unsupported discovery method: %s", s.config.Method)
	}

	// Start cleanup routine
	go s.runCleanupRoutine(ctx)

	return nil
}

// Stop stops the discovery service
func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.logger.Info("Stopping discovery service")
	close(s.stopChan)
	s.running = false

	return nil
}

// AddEventHandler adds an event handler for node discovery events
func (s *Service) AddEventHandler(handler NodeEventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventHandlers = append(s.eventHandlers, handler)
}

// GetNodes returns all currently discovered nodes
func (s *Service) GetNodes() []Node {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]Node, 0, len(s.nodes))
	for _, node := range s.nodes {
		nodes = append(nodes, *node)
	}

	return nodes
}

// GetNode returns a specific node by ID
func (s *Service) GetNode(id string) (*Node, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	node, exists := s.nodes[id]
	if !exists {
		return nil, false
	}

	return &Node{
		ID:           node.ID,
		Name:         node.Name,
		IPAddress:    node.IPAddress,
		Port:         node.Port,
		ServiceType:  node.ServiceType,
		TXTRecords:   node.TXTRecords,
		LastSeen:     node.LastSeen,
		Capabilities: node.Capabilities,
	}, true
}

// loadStaticNodes loads nodes from static configuration
func (s *Service) loadStaticNodes() {
	for i, nodeAddr := range s.config.StaticNodes {
		host, portStr, err := net.SplitHostPort(nodeAddr)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"address": nodeAddr,
				"error":   err,
			}).Warn("Invalid static node address")
			continue
		}

		port, err := strconv.Atoi(portStr)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"address": nodeAddr,
				"error":   err,
			}).Warn("Invalid static node port")
			continue
		}

		node := &Node{
			ID:          fmt.Sprintf("static-%d", i),
			Name:        fmt.Sprintf("static-node-%d", i),
			IPAddress:   host,
			Port:        port,
			ServiceType: "static",
			TXTRecords:  make(map[string]string),
			LastSeen:    time.Now(),
			Capabilities: []string{"gpio", "monitoring"},
		}

		s.nodes[node.ID] = node
		s.emitEvent(NodeEvent{
			Type: NodeDiscovered,
			Node: *node,
		})

		s.logger.WithFields(logrus.Fields{
			"id":         node.ID,
			"ip_address": node.IPAddress,
			"port":       node.Port,
		}).Info("Loaded static node")
	}
}

// runMDNSDiscovery runs mDNS-based discovery
func (s *Service) runMDNSDiscovery(ctx context.Context) {
	s.logger.Info("Starting mDNS discovery")
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.performMDNSDiscovery()
		}
	}
}

// performMDNSDiscovery performs a single mDNS discovery cycle
func (s *Service) performMDNSDiscovery() {
	// TODO: Implement actual mDNS discovery
	// This would typically use libraries like github.com/hashicorp/mdns
	// For now, this is a mock implementation

	s.logger.Debug("Performing mDNS discovery scan")

	// Mock discovery: simulate finding nodes
	mockNodes := []Node{
		{
			ID:        "mdns-pi-001",
			Name:      "raspberrypi-001",
			IPAddress: "192.168.1.101",
			Port:      s.config.Port,
			ServiceType: s.config.ServiceType,
			TXTRecords: map[string]string{
				"version":      "1.0.0",
				"capabilities": "gpio,monitoring",
				"model":        "Raspberry Pi 4",
			},
			LastSeen:    time.Now(),
			Capabilities: []string{"gpio", "monitoring"},
		},
		{
			ID:        "mdns-pi-002",
			Name:      "raspberrypi-002",
			IPAddress: "192.168.1.102",
			Port:      s.config.Port,
			ServiceType: s.config.ServiceType,
			TXTRecords: map[string]string{
				"version":      "1.0.0",
				"capabilities": "gpio,monitoring",
				"model":        "Raspberry Pi 3",
			},
			LastSeen:    time.Now(),
			Capabilities: []string{"gpio", "monitoring"},
		},
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, node := range mockNodes {
		existingNode, exists := s.nodes[node.ID]
		if exists {
			// Update existing node
			existingNode.LastSeen = node.LastSeen
			s.emitEvent(NodeEvent{
				Type: NodeUpdated,
				Node: node,
			})
		} else {
			// New node discovered
			s.nodes[node.ID] = &node
			s.emitEvent(NodeEvent{
				Type: NodeDiscovered,
				Node: node,
			})

			s.logger.WithFields(logrus.Fields{
				"id":         node.ID,
				"name":       node.Name,
				"ip_address": node.IPAddress,
			}).Info("Discovered new node via mDNS")
		}
	}
}

// runNetworkScan runs network scan-based discovery
func (s *Service) runNetworkScan(ctx context.Context) {
	s.logger.Info("Starting network scan discovery")
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.performNetworkScan()
		}
	}
}

// performNetworkScan performs a network scan for discovery
func (s *Service) performNetworkScan() {
	s.logger.Debug("Performing network scan")

	// TODO: Implement actual network scanning
	// This would typically scan common subnets for the service port
	// For now, this is a placeholder

	// Get local network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get network interfaces")
		return
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				s.logger.WithFields(logrus.Fields{
					"interface": iface.Name,
					"network":   ipnet.String(),
				}).Debug("Scanning network")

				// TODO: Scan the network for nodes
				// This would involve:
				// 1. Generating IP ranges from the network
				// 2. Port scanning for the service port
				// 3. Attempting to connect and identify pi-controller agents
			}
		}
	}
}

// runCleanupRoutine runs the cleanup routine to remove stale nodes
func (s *Service) runCleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(s.interval * 2) // Run cleanup less frequently
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.cleanupStaleNodes()
		}
	}
}

// cleanupStaleNodes removes nodes that haven't been seen recently
func (s *Service) cleanupStaleNodes() {
	s.mu.Lock()
	defer s.mu.Unlock()

	staleThreshold := time.Now().Add(-s.interval * 3) // Consider stale after 3 intervals
	staleCandidates := make([]string, 0)

	for id, node := range s.nodes {
		// Don't cleanup static nodes
		if node.ServiceType == "static" {
			continue
		}

		if node.LastSeen.Before(staleThreshold) {
			staleCandidates = append(staleCandidates, id)
		}
	}

	for _, id := range staleCandidates {
		node := s.nodes[id]
		delete(s.nodes, id)

		s.emitEvent(NodeEvent{
			Type: NodeLost,
			Node: *node,
		})

		s.logger.WithFields(logrus.Fields{
			"id":        id,
			"last_seen": node.LastSeen,
		}).Info("Removed stale node")
	}
}

// emitEvent emits a node event to all registered handlers
func (s *Service) emitEvent(event NodeEvent) {
	for _, handler := range s.eventHandlers {
		go func(h NodeEventHandler) {
			defer func() {
				if r := recover(); r != nil {
					s.logger.WithField("panic", r).Error("Event handler panicked")
				}
			}()
			h(event)
		}(handler)
	}
}