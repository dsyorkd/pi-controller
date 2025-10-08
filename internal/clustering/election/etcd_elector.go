package election

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// EtcdElector implements LeaderElector using etcd
type EtcdElector struct {
	config *Config

	// etcd client and session
	client   *clientv3.Client
	session  *concurrency.Session
	election *concurrency.Election

	// State
	isLeader bool
	leaderID string
	term     uint64
	mu       sync.RWMutex

	// Callbacks
	becomeLeaderCallback func()
	loseLeaderCallback   func()
	callbackMu           sync.RWMutex

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	stopCh chan struct{}
}

// NewEtcdElector creates a new etcd-based leader elector
func NewEtcdElector(config *Config) (*EtcdElector, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if config.ControllerID == "" {
		return nil, fmt.Errorf("ControllerID is required")
	}

	if len(config.EtcdEndpoints) == 0 {
		return nil, fmt.Errorf("EtcdEndpoints are required")
	}

	// Build etcd client config
	clientConfig := clientv3.Config{
		Endpoints:   config.EtcdEndpoints,
		DialTimeout: 5 * time.Second,
	}

	// Configure TLS if specified
	if config.EtcdTLS != nil {
		tlsConfig, err := buildTLSConfig(config.EtcdTLS)
		if err != nil {
			return nil, fmt.Errorf("failed to build TLS config: %w", err)
		}
		clientConfig.TLS = tlsConfig
	}

	// Create etcd client
	client, err := clientv3.New(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &EtcdElector{
		config: config,
		client: client,
		ctx:    ctx,
		cancel: cancel,
		stopCh: make(chan struct{}),
	}, nil
}

// Run starts the leader election loop
func (e *EtcdElector) Run(ctx context.Context) error {
	// Create session with lease TTL
	session, err := concurrency.NewSession(
		e.client,
		concurrency.WithTTL(e.config.LeaseTTL),
		concurrency.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to create etcd session: %w", err)
	}
	e.session = session
	defer session.Close()

	// Create election object
	e.election = concurrency.NewElection(session, e.config.LeadershipKey)

	// Campaign for leadership
	go e.campaign(ctx)

	// Observe leadership changes
	go e.observe(ctx)

	// Wait for context cancellation
	<-ctx.Done()
	close(e.stopCh)

	return nil
}

// campaign attempts to become the leader
func (e *EtcdElector) campaign(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		default:
			// Campaign for leadership (blocks until we become leader or context is cancelled)
			if err := e.election.Campaign(ctx, e.config.ControllerID); err != nil {
				if err == context.Canceled {
					return
				}
				// Election failed, retry after interval
				time.Sleep(time.Duration(e.config.RetryInterval) * time.Second)
				continue
			}

			// We became the leader!
			e.mu.Lock()
			wasLeader := e.isLeader
			e.isLeader = true
			e.leaderID = e.config.ControllerID
			e.term++
			e.mu.Unlock()

			if !wasLeader {
				// Call the become leader callback
				e.callbackMu.RLock()
				if e.becomeLeaderCallback != nil {
					go e.becomeLeaderCallback()
				}
				e.callbackMu.RUnlock()
			}

			// Keep campaigning to maintain leadership
			// Campaign will return when we lose leadership
		}
	}
}

// observe watches for leadership changes
func (e *EtcdElector) observe(ctx context.Context) {
	observeChan := e.election.Observe(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case resp := <-observeChan:
			if resp.Kvs == nil || len(resp.Kvs) == 0 {
				continue
			}

			leaderID := string(resp.Kvs[0].Value)

			e.mu.Lock()
			previousLeaderID := e.leaderID
			wasLeader := e.isLeader

			// Update leader info
			e.leaderID = leaderID
			e.isLeader = (leaderID == e.config.ControllerID)

			// Detect leadership loss
			if wasLeader && !e.isLeader {
				e.mu.Unlock()

				// Call the lose leadership callback
				e.callbackMu.RLock()
				if e.loseLeaderCallback != nil {
					go e.loseLeaderCallback()
				}
				e.callbackMu.RUnlock()
			} else {
				e.mu.Unlock()
			}

			// Log leadership change
			if previousLeaderID != leaderID {
				// Leadership changed
			}
		}
	}
}

// IsLeader returns true if this controller is the leader
func (e *EtcdElector) IsLeader() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.isLeader
}

// GetLeader returns the current leader ID
func (e *EtcdElector) GetLeader() (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.leaderID, nil
}

// GetTerm returns the current election term
func (e *EtcdElector) GetTerm() uint64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.term
}

// OnBecomeLeader sets callback for when this controller becomes leader
func (e *EtcdElector) OnBecomeLeader(callback func()) {
	e.callbackMu.Lock()
	defer e.callbackMu.Unlock()
	e.becomeLeaderCallback = callback
}

// OnLoseLeadership sets callback for when this controller loses leadership
func (e *EtcdElector) OnLoseLeadership(callback func()) {
	e.callbackMu.Lock()
	defer e.callbackMu.Unlock()
	e.loseLeaderCallback = callback
}

// Stop gracefully stops the leader election
func (e *EtcdElector) Stop() error {
	// Cancel context to stop all goroutines
	e.cancel()

	// Resign from leadership if we're the leader
	if e.isLeader && e.election != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := e.election.Resign(ctx); err != nil {
			return fmt.Errorf("failed to resign leadership: %w", err)
		}
	}

	// Close session
	if e.session != nil {
		if err := e.session.Close(); err != nil {
			return fmt.Errorf("failed to close session: %w", err)
		}
	}

	// Close etcd client
	if e.client != nil {
		if err := e.client.Close(); err != nil {
			return fmt.Errorf("failed to close etcd client: %w", err)
		}
	}

	return nil
}

// buildTLSConfig creates a TLS configuration from the provided config
func buildTLSConfig(config *TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12, // Enforce TLS 1.2 minimum for security
	}

	// Load client certificate and key
	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client cert/key: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificate
	if config.CAFile != "" {
		caCert, err := os.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA cert")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}
