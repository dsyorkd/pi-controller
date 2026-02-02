# Race Conditions and State Reset Strategy

## Overview

This document identifies potential race condition risks in pi-controller and defines comprehensive state reset procedures to restore Raspberry Pi devices to their pre-installation state.

---

## 1. Race Condition Risk Areas

### 1.1 Raft Cluster State Transitions

**Risk**: Multiple nodes attempting leadership election simultaneously or concurrent node join/leave operations.

**Locations**:

- `internal/clustering/raft_cluster.go`: Leadership changes, node membership
- `observeLeadership()`: State transition callbacks

**Specific Scenarios**:

```go
// RACE CONDITION: Leadership callback execution
// Problem: Multiple goroutines may execute callbacks simultaneously
go c.observeLeadership()

// Solution: Proper locking in callbacks
c.callbackMu.RLock()
if c.becomeLeaderCallback != nil {
    go c.becomeLeaderCallback()
}
c.callbackMu.RUnlock()
```

**Mitigation Strategies**:

1. ✅ **Already Implemented**: `leaderMu` RWMutex for leader state
2. ✅ **Already Implemented**: `callbackMu` for callback registration
3. ⚠️ **Needs Enhancement**: Add timeout protection for callback execution
4. ⚠️ **Needs Enhancement**: Prevent callback re-entry during rapid leader changes

**Recommended Improvements**:

```go
// Add callback execution tracking
type RaftCluster struct {
    // ... existing fields
    callbackExecuting sync.Map // nodeID -> bool
    callbackTimeout   time.Duration
}

func (c *RaftCluster) observeLeadership() {
    for isLeader := range c.raft.LeaderCh() {
        c.leaderMu.Lock()
        previousState := c.isLeader
        c.isLeader = isLeader
        c.leaderMu.Unlock()

        if isLeader && !previousState {
            // Check if callback is already executing
            if _, loaded := c.callbackExecuting.LoadOrStore(c.config.ControllerID, true); !loaded {
                go func() {
                    defer c.callbackExecuting.Delete(c.config.ControllerID)

                    ctx, cancel := context.WithTimeout(context.Background(), c.callbackTimeout)
                    defer cancel()

                    c.callbackMu.RLock()
                    if c.becomeLeaderCallback != nil {
                        c.becomeLeaderCallback()
                    }
                    c.callbackMu.RUnlock()
                }()
            }
        }
    }
}
```

### 1.2 GPIO Pin Access

**Risk**: Concurrent operations on the same GPIO pin causing hardware damage or inconsistent state.

**Locations**:

- `pkg/gpio/controller.go`: Pin state management
- `internal/agent/gpio_service.go`: gRPC service handlers

**Specific Scenarios**:

```go
// RACE CONDITION: Pin state modification
// Problem: Multiple API calls modifying same pin simultaneously
type Controller struct {
    activePins map[int]*PinState  // Protected by mutex
    mutex      sync.RWMutex
    opMutex    sync.Mutex         // For operation counting
}
```

**Current Protection**:

- ✅ `mutex` for `activePins` map access
- ✅ `opMutex` for concurrent operation limiting
- ✅ `MaxConcurrentOps` configuration

**Missing Protection**:

- ⚠️ Per-pin mutex for hardware operations
- ⚠️ Operation queue for same-pin requests
- ⚠️ Hardware state verification after concurrent access

**Recommended Improvements**:

```go
type PinState struct {
    Pin        int
    Direction  PinDirection
    Value      bool
    LastAccess time.Time
    mu         sync.Mutex  // Per-pin lock
}

func (c *Controller) SetPinValue(ctx context.Context, pin int, value bool) error {
    // Get pin state with read lock
    c.mutex.RLock()
    state, exists := c.activePins[pin]
    c.mutex.RUnlock()

    if !exists {
        return fmt.Errorf("pin %d not initialized", pin)
    }

    // Lock the specific pin for hardware operation
    state.mu.Lock()
    defer state.mu.Unlock()

    // Perform hardware operation
    if err := c.impl.SetPinValue(pin, value); err != nil {
        return err
    }

    // Verify hardware state matches desired state
    actualValue, err := c.impl.GetPinValue(pin)
    if err != nil {
        return fmt.Errorf("failed to verify pin state: %w", err)
    }

    if actualValue != value {
        c.logger.WithFields(logrus.Fields{
            "pin":      pin,
            "expected": value,
            "actual":   actualValue,
        }).Error("Pin state mismatch after write")
        return fmt.Errorf("pin state verification failed")
    }

    state.Value = value
    state.LastAccess = time.Now()
    return nil
}
```

### 1.3 SSH Connection Pool

**Risk**: Connection pool exhaustion, race in connection acquisition/release.

**Locations**:

- `internal/provisioner/ssh_client.go`: Connection pooling logic

**Specific Scenarios**:

```go
// RACE CONDITION: Pool exhaustion and cleanup
func (c *SSHClient) getConnection(ctx context.Context) (*SSHConnection, error) {
    c.poolMutex.Lock()
    defer c.poolMutex.Unlock()

    // PROBLEM: Lock held during slow operations (connection testing)
    for _, conn := range c.pool {
        conn.mutex.Lock()
        if !conn.inUse && time.Since(conn.lastUsed) < c.config.IdleTimeout {
            if err := c.testConnection(conn.client); err == nil {  // Slow operation
                conn.inUse = true
                conn.lastUsed = time.Now()
                conn.mutex.Unlock()
                return conn, nil
            }
            _ = conn.client.Close()
        }
        conn.mutex.Unlock()
    }
}
```

**Issues**:

- ⚠️ `poolMutex` held during connection testing (network I/O)
- ⚠️ Potential deadlock between `poolMutex` and `conn.mutex`
- ⚠️ No priority queue for waiting requests

**Recommended Improvements**:

```go
// Use connection channels for better concurrency
type SSHClient struct {
    config      SSHClientConfig
    connChan    chan *SSHConnection
    logger      logger.Interface
    authMethods []ssh.AuthMethod
    wg          sync.WaitGroup
}

func (c *SSHClient) getConnection(ctx context.Context) (*SSHConnection, error) {
    select {
    case conn := <-c.connChan:
        // Test connection outside of critical section
        if err := c.testConnection(conn.client); err == nil {
            conn.inUse = true
            conn.lastUsed = time.Now()
            return conn, nil
        }
        // Connection dead, create new one
        conn.client.Close()
        return c.createNewConnection(ctx)
    case <-ctx.Done():
        return nil, ctx.Err()
    case <-time.After(c.config.Timeout):
        return nil, fmt.Errorf("connection acquisition timeout")
    }
}

func (c *SSHClient) releaseConnection(conn *SSHConnection) {
    conn.inUse = false
    conn.lastUsed = time.Now()

    select {
    case c.connChan <- conn:
        // Connection returned to pool
    default:
        // Pool full, close connection
        conn.client.Close()
    }
}
```

### 1.4 Kubernetes Migration (systemd → DaemonSet)

**Risk**: Critical state transition without quorum loss, port conflicts, dual operation.

**Location**: Migration logic needs to be implemented

**Architecture Issue**: Per `index.md` lines 427-453:

```
BEFORE: systemd on ports 9091 (Raft)
DURING: systemd (9091) + DaemonSet (9092) ← RACE WINDOW
AFTER:  DaemonSet (9091) only
```

**Race Scenarios**:

1. **Port Binding Race**:

   ```
   T1: systemd releases port 9091
   T2: DaemonSet pod tries to bind port 9091
   T3: Another process grabs port 9091
   Result: DaemonSet fails to start
   ```

2. **Quorum Loss During Migration**:

   ```
   T1: systemd stops on Node-1 (was leader)
   T2: Raft election starts
   T3: DaemonSet not yet running on Node-1
   Result: Temporary loss of leader, operations blocked
   ```

3. **State Divergence**:

   ```
   T1: systemd receives GPIO command
   T2: systemd replicates to Raft cluster
   T3: systemd shuts down before state sync
   T4: DaemonSet starts with stale snapshot
   Result: Lost GPIO state updates
   ```

**Mitigation Strategy**:

```go
type MigrationOrchestrator struct {
    nodes          []*NodeInfo
    k8sClient      *k8s.Client
    raftCluster    *clustering.RaftCluster
    logger         logger.Interface
    migrationState *MigrationState
    mu             sync.RWMutex
}

type MigrationState struct {
    Phase          MigrationPhase
    NodesCompleted map[string]bool
    Snapshot       []byte
    RollbackData   *RollbackData
    StartTime      time.Time
}

type MigrationPhase int
const (
    PhasePreMigration MigrationPhase = iota
    PhaseDualRun           // systemd + DaemonSet both running
    PhaseQuorumTransfer    // Transfer Raft quorum
    PhaseSystemdShutdown   // Graceful systemd shutdown
    PhasePortHandoff       // Port ownership transfer
    PhaseDaemonSetTakeover // DaemonSet assumes control
    PhaseComplete
)

func (m *MigrationOrchestrator) MigrateToKubernetes(ctx context.Context) error {
    // Phase 1: Pre-migration validation
    if err := m.validatePreMigration(ctx); err != nil {
        return fmt.Errorf("pre-migration validation failed: %w", err)
    }

    // Phase 2: Deploy DaemonSet with temporary ports
    if err := m.deployDaemonSetWithTempPorts(ctx); err != nil {
        return fmt.Errorf("DaemonSet deployment failed: %w", err)
    }

    // Phase 3: Wait for DaemonSet cluster to achieve quorum
    if err := m.waitForDaemonSetQuorum(ctx); err != nil {
        return m.rollback(ctx, fmt.Errorf("DaemonSet quorum failed: %w", err))
    }

    // Phase 4: Snapshot systemd Raft state
    snapshot, err := m.snapshotSystemdRaft(ctx)
    if err != nil {
        return m.rollback(ctx, fmt.Errorf("state snapshot failed: %w", err))
    }
    m.migrationState.Snapshot = snapshot

    // Phase 5: Transfer state to DaemonSet cluster
    if err := m.transferStateToDaemonSet(ctx, snapshot); err != nil {
        return m.rollback(ctx, fmt.Errorf("state transfer failed: %w", err))
    }

    // Phase 6: Verify state synchronization
    if err := m.verifyStateSync(ctx); err != nil {
        return m.rollback(ctx, fmt.Errorf("state verification failed: %w", err))
    }

    // Phase 7: Graceful systemd shutdown (one node at a time)
    for _, node := range m.nodes {
        if err := m.shutdownSystemdOnNode(ctx, node); err != nil {
            return m.rollback(ctx, fmt.Errorf("systemd shutdown failed on %s: %w", node.Name, err))
        }

        // Verify DaemonSet maintains quorum after each shutdown
        if err := m.verifyDaemonSetQuorum(ctx); err != nil {
            return m.rollback(ctx, fmt.Errorf("quorum lost after shutting down %s: %w", node.Name, err))
        }
    }

    // Phase 8: Reconfigure DaemonSet to use standard ports
    if err := m.reconfigureDaemonSetPorts(ctx); err != nil {
        return m.rollback(ctx, fmt.Errorf("port reconfiguration failed: %w", err))
    }

    // Phase 9: Remove systemd services permanently
    if err := m.removeSystemdServices(ctx); err != nil {
        // Non-fatal: log warning but continue
        m.logger.WithError(err).Warn("Failed to remove systemd services, manual cleanup required")
    }

    // Phase 10: Final validation
    if err := m.validatePostMigration(ctx); err != nil {
        return fmt.Errorf("post-migration validation failed: %w", err)
    }

    m.logger.Info("Kubernetes migration completed successfully")
    return nil
}

func (m *MigrationOrchestrator) shutdownSystemdOnNode(ctx context.Context, node *NodeInfo) error {
    // Step 1: Mark node as migrating
    if err := m.markNodeMigrating(ctx, node); err != nil {
        return err
    }

    // Step 2: Drain Raft leadership if this node is leader
    if m.isRaftLeader(node) {
        if err := m.transferRaftLeadership(ctx, node); err != nil {
            return fmt.Errorf("leadership transfer failed: %w", err)
        }

        // Wait for leadership to stabilize
        time.Sleep(5 * time.Second)
    }

    // Step 3: Stop accepting new GPIO operations
    if err := m.stopGPIOOperations(ctx, node); err != nil {
        return err
    }

    // Step 4: Wait for in-flight operations to complete
    if err := m.waitForOperationsComplete(ctx, node); err != nil {
        return err
    }

    // Step 5: Final state sync to DaemonSet
    if err := m.syncFinalState(ctx, node); err != nil {
        return err
    }

    // Step 6: Stop systemd service gracefully
    if err := m.stopSystemdService(ctx, node); err != nil {
        return err
    }

    // Step 7: Disable systemd service
    if err := m.disableSystemdService(ctx, node); err != nil {
        return err
    }

    return nil
}

func (m *MigrationOrchestrator) rollback(ctx context.Context, cause error) error {
    m.logger.WithError(cause).Error("Migration failed, initiating rollback")

    // Stop DaemonSet pods
    if err := m.k8sClient.DeleteDaemonSet(ctx, "pi-controller", "pi-controller"); err != nil {
        m.logger.WithError(err).Error("Failed to delete DaemonSet during rollback")
    }

    // Ensure systemd services are running
    for _, node := range m.nodes {
        if err := m.ensureSystemdRunning(ctx, node); err != nil {
            m.logger.WithError(err).WithField("node", node.Name).Error("Failed to restart systemd during rollback")
        }
    }

    // Restore Raft state if we have a snapshot
    if m.migrationState.Snapshot != nil {
        if err := m.restoreRaftSnapshot(ctx, m.migrationState.Snapshot); err != nil {
            m.logger.WithError(err).Error("Failed to restore Raft snapshot during rollback")
        }
    }

    return cause
}
```

### 1.5 mDNS Discovery and Node Registration

**Risk**: Duplicate node registration, split-brain scenarios.

**Locations**:

- `pkg/discovery/`: mDNS advertisement and discovery

**Scenarios**:

1. Multiple nodes discover each other simultaneously
2. Node rejoins after network partition
3. Stale mDNS entries after node failure

**Recommended Protection**:

```go
type DiscoveryService struct {
    knownNodes    map[string]*NodeInfo
    nodesMutex    sync.RWMutex
    registryLock  sync.Mutex  // For node registration
    lastSeen      map[string]time.Time
    cleanupTicker *time.Ticker
}

func (d *DiscoveryService) RegisterNode(node *NodeInfo) error {
    // Use mutex to prevent duplicate registration
    d.registryLock.Lock()
    defer d.registryLock.Unlock()

    d.nodesMutex.Lock()
    defer d.nodesMutex.Unlock()

    // Check if node already exists
    if existing, exists := d.knownNodes[node.ID]; exists {
        // Compare timestamps to determine if this is a newer registration
        if node.LastSeen.After(existing.LastSeen) {
            d.knownNodes[node.ID] = node
            d.lastSeen[node.ID] = time.Now()
            return nil
        }
        return fmt.Errorf("node %s already registered with newer timestamp", node.ID)
    }

    d.knownNodes[node.ID] = node
    d.lastSeen[node.ID] = time.Now()
    return nil
}

// Background cleanup of stale nodes
func (d *DiscoveryService) cleanupStaleNodes() {
    d.nodesMutex.Lock()
    defer d.nodesMutex.Unlock()

    now := time.Now()
    for nodeID, lastSeen := range d.lastSeen {
        if now.Sub(lastSeen) > 5*time.Minute {
            delete(d.knownNodes, nodeID)
            delete(d.lastSeen, nodeID)
            d.logger.WithField("node_id", nodeID).Info("Removed stale node")
        }
    }
}
```

### 1.6 Database Concurrent Access

**Risk**: SQLite concurrent write conflicts, WAL corruption.

**Locations**:

- `internal/storage/`: Database operations
- Raft log application to SQLite

**SQLite Concurrency Limits**:

- Multiple readers OK
- Single writer at a time
- WAL mode enables better concurrency

**Protection Strategy**:

```go
type Database struct {
    db          *sql.DB
    writeMutex  sync.Mutex  // Serialize writes
    raftApplied chan struct{}
}

func (d *Database) executeWrite(ctx context.Context, query string, args ...interface{}) error {
    d.writeMutex.Lock()
    defer d.writeMutex.Unlock()

    // Use transaction with retry logic
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        tx, err := d.db.BeginTx(ctx, nil)
        if err != nil {
            return err
        }

        _, err = tx.ExecContext(ctx, query, args...)
        if err != nil {
            tx.Rollback()

            // Check if it's a lock error
            if strings.Contains(err.Error(), "database is locked") {
                time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
                continue
            }
            return err
        }

        if err := tx.Commit(); err != nil {
            return err
        }

        return nil
    }

    return fmt.Errorf("write failed after %d retries", maxRetries)
}
```

---

## 2. State Reset and Cleanup Strategy

### 2.1 Complete Uninstall Procedure

Goal: Restore Raspberry Pi to pre-installation state, removing ALL traces of pi-controller.

```bash
#!/bin/bash
# /usr/local/bin/pi-controller-uninstall.sh

set -e

echo "Pi-Controller Complete Uninstall"
echo "================================="
echo ""
echo "WARNING: This will remove ALL pi-controller data and configuration."
echo "This action CANNOT be undone."
echo ""
read -p "Type 'UNINSTALL' to confirm: " confirm

if [ "$confirm" != "UNINSTALL" ]; then
    echo "Uninstall cancelled."
    exit 1
fi

# Function to log actions
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# 1. Stop all running services
log "Stopping pi-controller services..."
systemctl stop pi-controller.service 2>/dev/null || true
systemctl disable pi-controller.service 2>/dev/null || true

# 2. Reset GPIO pins to safe state
log "Resetting GPIO pins to safe state..."
if command -v pi-controller &> /dev/null; then
    pi-controller gpio reset-all --safe-mode || log "Warning: GPIO reset failed"
fi

# 3. Leave Raft cluster gracefully
log "Leaving Raft cluster..."
if command -v pi-controller &> /dev/null; then
    pi-controller cluster leave --force || log "Warning: Cluster leave failed"
fi

# 4. Remove Kubernetes resources (if installed)
log "Checking for Kubernetes resources..."
if command -v kubectl &> /dev/null; then
    kubectl delete daemonset pi-controller -n pi-controller 2>/dev/null || true
    kubectl delete namespace pi-controller 2>/dev/null || true
    kubectl delete crd gpiopins.pi-controller.io 2>/dev/null || true
    kubectl delete crd pwmcontrollers.pi-controller.io 2>/dev/null || true
    kubectl delete crd i2cdevices.pi-controller.io 2>/dev/null || true
    log "Kubernetes resources removed"
fi

# 5. Remove systemd service files
log "Removing systemd service files..."
rm -f /etc/systemd/system/pi-controller.service
rm -f /etc/systemd/system/pi-controller.service.d/*
rmdir /etc/systemd/system/pi-controller.service.d 2>/dev/null || true
systemctl daemon-reload

# 6. Remove binaries
log "Removing binaries..."
rm -f /usr/local/bin/pi-controller
rm -f /usr/bin/pi-controller

# 7. Remove data directories
log "Removing data directories..."
rm -rf /var/lib/pi-controller
rm -rf /var/lib/raft
rm -rf /opt/pi-controller

# 8. Remove configuration
log "Removing configuration files..."
rm -rf /etc/pi-controller
rm -f /etc/default/pi-controller

# 9. Remove logs
log "Removing log files..."
rm -rf /var/log/pi-controller

# 10. Remove user and group
log "Removing pi-controller user and group..."
userdel pi-controller 2>/dev/null || true
groupdel pi-controller 2>/dev/null || true

# 11. Remove TLS certificates
log "Removing TLS certificates..."
rm -rf /etc/pi-controller/certs
rm -rf /var/lib/pi-controller/pki

# 12. Remove mDNS service registration
log "Removing mDNS service registration..."
if command -v avahi-daemon &> /dev/null; then
    rm -f /etc/avahi/services/pi-controller.service
    systemctl reload avahi-daemon 2>/dev/null || true
fi

# 13. Remove firewall rules (if ufw is used)
log "Removing firewall rules..."
if command -v ufw &> /dev/null; then
    ufw delete allow 8080/tcp comment 'pi-controller REST API' 2>/dev/null || true
    ufw delete allow 9090/tcp comment 'pi-controller gRPC' 2>/dev/null || true
    ufw delete allow 8081/tcp comment 'pi-controller WebSocket' 2>/dev/null || true
    ufw delete allow 9091/tcp comment 'pi-controller Raft' 2>/dev/null || true
fi

# 14. Remove iptables rules (alternative to ufw)
log "Removing iptables rules..."
iptables -D INPUT -p tcp --dport 8080 -j ACCEPT 2>/dev/null || true
iptables -D INPUT -p tcp --dport 9090 -j ACCEPT 2>/dev/null || true
iptables -D INPUT -p tcp --dport 8081 -j ACCEPT 2>/dev/null || true
iptables -D INPUT -p tcp --dport 9091 -j ACCEPT 2>/dev/null || true

# 15. Remove kernel modules (if any were loaded)
log "Removing kernel modules..."
# Pi-controller doesn't load custom modules, but placeholder for future

# 16. Verify GPIO pins are in safe state
log "Verifying GPIO pin state..."
# Read all GPIO pins and ensure they're not driving voltage
for pin in {0..27}; do
    # Skip critical system pins
    if [[ $pin -eq 0 || $pin -eq 1 || $pin -eq 14 || $pin -eq 15 ]]; then
        continue
    fi

    # Attempt to set pin to input mode (safe state)
    echo "$pin" > /sys/class/gpio/export 2>/dev/null || true
    echo "in" > /sys/class/gpio/gpio${pin}/direction 2>/dev/null || true
    echo "$pin" > /sys/class/gpio/unexport 2>/dev/null || true
done

# 17. Remove scheduled cron jobs (if any)
log "Removing cron jobs..."
crontab -u pi-controller -r 2>/dev/null || true

# 18. Clean up package manager (if installed via package)
log "Cleaning up package manager..."
if command -v apt-get &> /dev/null; then
    apt-get purge -y pi-controller 2>/dev/null || true
    apt-get autoremove -y 2>/dev/null || true
fi

# 19. Create uninstall report
REPORT_FILE="/tmp/pi-controller-uninstall-report-$(date +%Y%m%d-%H%M%S).txt"
log "Creating uninstall report at $REPORT_FILE..."

cat > "$REPORT_FILE" <<EOF
Pi-Controller Uninstall Report
==============================
Date: $(date)
Hostname: $(hostname)

Services Stopped:
- pi-controller.service

Files Removed:
- /usr/local/bin/pi-controller
- /var/lib/pi-controller/
- /etc/pi-controller/
- /var/log/pi-controller/

Network Configuration:
- Firewall rules removed (ports 8080, 9090, 8081, 9091)
- mDNS service deregistered

Kubernetes Resources:
- DaemonSet removed (if existed)
- CRDs removed (if existed)

GPIO State:
- All pins reset to input mode (safe state)

Raft Cluster:
- Node gracefully left cluster

Post-Uninstall Actions Required:
1. Verify no processes listening on ports 8080, 9090, 8081, 9091
2. Manually remove any custom configurations
3. Reboot recommended to ensure clean state

To verify complete removal:
  sudo find / -name "*pi-controller*" 2>/dev/null

EOF

log "Uninstall complete!"
log "Report saved to: $REPORT_FILE"
log ""
log "IMPORTANT: Please reboot your Raspberry Pi to ensure all changes take effect."
log "  sudo reboot"

exit 0
```

### 2.2 GPIO Safe State Reset

```go
// internal/gpio/reset.go

package gpio

import (
    "context"
    "fmt"
    "time"
)

// ResetAllPinsToSafe resets all GPIO pins to a safe input state
func (c *Controller) ResetAllPinsToSafe(ctx context.Context) error {
    c.logger.Info("Resetting all GPIO pins to safe state")

    c.mutex.Lock()
    defer c.mutex.Unlock()

    var errors []error

    // Iterate through all active pins
    for pinNum, state := range c.activePins {
        // Skip critical system pins
        if isCriticalPin(pinNum) {
            c.logger.WithField("pin", pinNum).Debug("Skipping critical system pin")
            continue
        }

        // Reset pin to input mode with pull-down
        if err := c.resetPinSafe(ctx, pinNum, state); err != nil {
            c.logger.WithError(err).WithField("pin", pinNum).Error("Failed to reset pin")
            errors = append(errors, fmt.Errorf("pin %d: %w", pinNum, err))
            continue
        }

        c.logger.WithField("pin", pinNum).Info("Pin reset to safe state")
    }

    // Clear active pins map
    c.activePins = make(map[int]*PinState)

    if len(errors) > 0 {
        return fmt.Errorf("failed to reset %d pins: %v", len(errors), errors)
    }

    c.logger.Info("All GPIO pins reset successfully")
    return nil
}

func (c *Controller) resetPinSafe(ctx context.Context, pin int, state *PinState) error {
    // Step 1: If output, set to LOW before changing direction
    if state.Direction == DirectionOut {
        if err := c.impl.SetPinValue(pin, false); err != nil {
            return fmt.Errorf("failed to set pin LOW: %w", err)
        }
        time.Sleep(10 * time.Millisecond) // Brief delay for hardware to settle
    }

    // Step 2: Change to input direction
    if err := c.impl.SetPinDirection(pin, DirectionIn); err != nil {
        return fmt.Errorf("failed to set direction to input: %w", err)
    }

    // Step 3: Enable pull-down resistor
    if err := c.impl.SetPinPull(pin, PullDown); err != nil {
        return fmt.Errorf("failed to set pull-down: %w", err)
    }

    // Step 4: Release the pin
    if err := c.impl.ReleasePin(pin); err != nil {
        return fmt.Errorf("failed to release pin: %w", err)
    }

    return nil
}

func isCriticalPin(pin int) bool {
    for _, critical := range CriticalSystemPins {
        if pin == critical {
            return true
        }
    }
    return false
}

// SavePinStates creates a snapshot of current GPIO state for recovery
func (c *Controller) SavePinStates() (*GPIOSnapshot, error) {
    c.mutex.RLock()
    defer c.mutex.RUnlock()

    snapshot := &GPIOSnapshot{
        Timestamp: time.Now(),
        Pins:      make(map[int]PinState),
    }

    for pinNum, state := range c.activePins {
        snapshot.Pins[pinNum] = *state
    }

    return snapshot, nil
}

// RestorePinStates restores GPIO state from a snapshot
func (c *Controller) RestorePinStates(ctx context.Context, snapshot *GPIOSnapshot) error {
    c.logger.WithField("timestamp", snapshot.Timestamp).Info("Restoring GPIO state from snapshot")

    c.mutex.Lock()
    defer c.mutex.Unlock()

    var errors []error

    for pinNum, state := range snapshot.Pins {
        if err := c.restorePin(ctx, pinNum, &state); err != nil {
            errors = append(errors, fmt.Errorf("pin %d: %w", pinNum, err))
        }
    }

    if len(errors) > 0 {
        return fmt.Errorf("failed to restore %d pins: %v", len(errors), errors)
    }

    return nil
}

type GPIOSnapshot struct {
    Timestamp time.Time
    Pins      map[int]PinState
}
```

### 2.3 Raft State Cleanup

```go
// internal/clustering/cleanup.go

package clustering

import (
    "fmt"
    "os"
    "path/filepath"
)

// CleanupRaftState removes all Raft data from the node
func (c *RaftCluster) CleanupRaftState() error {
    c.logger.Info("Cleaning up Raft state")

    // Step 1: Ensure Raft is shut down
    if c.raft != nil {
        future := c.raft.Shutdown()
        if err := future.Error(); err != nil {
            c.logger.WithError(err).Warn("Error during Raft shutdown")
        }
    }

    // Step 2: Close transport
    if c.transport != nil {
        if err := c.transport.Close(); err != nil {
            c.logger.WithError(err).Warn("Error closing transport")
        }
    }

    // Step 3: Remove data directory
    if err := os.RemoveAll(c.config.DataDir); err != nil {
        return fmt.Errorf("failed to remove Raft data directory: %w", err)
    }

    c.logger.WithField("data_dir", c.config.DataDir).Info("Raft state cleaned up successfully")
    return nil
}

// CreateStateSnapshot creates a complete snapshot of Raft state for backup
func (c *RaftCluster) CreateStateSnapshot() (*RaftSnapshot, error) {
    // Trigger Raft snapshot
    future := c.raft.Snapshot()
    if err := future.Error(); err != nil {
        return nil, fmt.Errorf("failed to create Raft snapshot: %w", err)
    }

    // Copy snapshot files
    snapshotDir := filepath.Join(c.config.DataDir, "snapshots")
    files, err := os.ReadDir(snapshotDir)
    if err != nil {
        return nil, fmt.Errorf("failed to read snapshot directory: %w", err)
    }

    snapshot := &RaftSnapshot{
        Timestamp: time.Now(),
        DataDir:   c.config.DataDir,
        Files:     make(map[string][]byte),
    }

    for _, file := range files {
        if file.IsDir() {
            continue
        }

        data, err := os.ReadFile(filepath.Join(snapshotDir, file.Name()))
        if err != nil {
            return nil, fmt.Errorf("failed to read snapshot file %s: %w", file.Name(), err)
        }

        snapshot.Files[file.Name()] = data
    }

    return snapshot, nil
}

type RaftSnapshot struct {
    Timestamp time.Time
    DataDir   string
    Files     map[string][]byte
}
```

### 2.4 Database State Reset

```go
// internal/storage/reset.go

package storage

import (
    "context"
    "fmt"
)

// ResetDatabase drops all tables and recreates schema
func (s *Storage) ResetDatabase(ctx context.Context) error {
    s.logger.Warn("Resetting database - all data will be lost")

    // Step 1: Create backup before reset
    backup, err := s.CreateBackup(ctx)
    if err != nil {
        return fmt.Errorf("failed to create backup before reset: %w", err)
    }

    s.logger.WithField("backup_path", backup).Info("Created backup before reset")

    // Step 2: Drop all tables
    tables := []string{
        "gpio_devices",
        "nodes",
        "users",
        "cluster_config",
        "audit_log",
    }

    for _, table := range tables {
        if _, err := s.db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", table)); err != nil {
            return fmt.Errorf("failed to drop table %s: %w", table, err)
        }
    }

    // Step 3: Re-run migrations to recreate schema
    if err := s.RunMigrations(ctx); err != nil {
        return fmt.Errorf("failed to recreate schema: %w", err)
    }

    s.logger.Info("Database reset complete")
    return nil
}

// CreateBackup creates a complete database backup
func (s *Storage) CreateBackup(ctx context.Context) (string, error) {
    backupPath := fmt.Sprintf("/var/backups/pi-controller-db-%s.db", time.Now().Format("20060102-150405"))

    // Use SQLite backup API
    query := fmt.Sprintf("VACUUM INTO '%s'", backupPath)
    if _, err := s.db.ExecContext(ctx, query); err != nil {
        return "", fmt.Errorf("failed to create backup: %w", err)
    }

    return backupPath, nil
}
```

---

## 3. Pre-Installation State Capture

Before installation, capture the system state to enable complete restoration:

```go
// internal/provisioner/state_capture.go

package provisioner

type SystemState struct {
    Timestamp        time.Time
    Hostname         string
    NetworkConfig    *NetworkState
    GPIOConfig       *GPIOState
    Services         []ServiceState
    InstalledPackages []string
    FirewallRules    []string
    CronJobs         []string
    Users            []string
    Groups           []string
}

func CaptureSystemState(ctx context.Context) (*SystemState, error) {
    state := &SystemState{
        Timestamp: time.Now(),
    }

    // Capture hostname
    hostname, err := os.Hostname()
    if err != nil {
        return nil, fmt.Errorf("failed to get hostname: %w", err)
    }
    state.Hostname = hostname

    // Capture network configuration
    networkState, err := captureNetworkState(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to capture network state: %w", err)
    }
    state.NetworkConfig = networkState

    // Capture GPIO state
    gpioState, err := captureGPIOState(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to capture GPIO state: %w", err)
    }
    state.GPIOConfig = gpioState

    // Capture running services
    services, err := captureServices(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to capture services: %w", err)
    }
    state.Services = services

    // Save state to file
    stateFile := "/var/lib/pi-controller/pre-install-state.json"
    if err := saveStateToFile(state, stateFile); err != nil {
        return nil, fmt.Errorf("failed to save state: %w", err)
    }

    return state, nil
}

func RestoreSystemState(ctx context.Context, stateFile string) error {
    state, err := loadStateFromFile(stateFile)
    if err != nil {
        return fmt.Errorf("failed to load state: %w", err)
    }

    // Restore network configuration
    if err := restoreNetworkState(ctx, state.NetworkConfig); err != nil {
        return fmt.Errorf("failed to restore network: %w", err)
    }

    // Restore GPIO configuration
    if err := restoreGPIOState(ctx, state.GPIOConfig); err != nil {
        return fmt.Errorf("failed to restore GPIO: %w", err)
    }

    // Restore services
    if err := restoreServices(ctx, state.Services); err != nil {
        return fmt.Errorf("failed to restore services: %w", err)
    }

    return nil
}
```

---

## 4. Testing Race Conditions

### 4.1 Race Detector Usage

```bash
# Build with race detector
go build -race -o pi-controller-race ./cmd/pi-controller

# Run tests with race detector
go test -race ./...

# Run integration tests with race detector
go test -race -tags=integration ./test/integration/...
```

### 4.2 Stress Testing Scenarios

```go
// test/stress/concurrent_gpio_test.go

func TestConcurrentGPIOAccess(t *testing.T) {
    controller := setupTestController(t)
    defer controller.Close()

    const (
        numGoroutines = 100
        numOperations = 1000
        targetPin     = 17
    )

    var wg sync.WaitGroup
    errors := make(chan error, numGoroutines*numOperations)

    // Spawn many goroutines accessing the same pin
    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()

            for j := 0; j < numOperations; j++ {
                value := j%2 == 0

                if err := controller.SetPinValue(context.Background(), targetPin, value); err != nil {
                    errors <- fmt.Errorf("goroutine %d op %d: %w", id, j, err)
                    return
                }

                // Verify the value was set correctly
                actualValue, err := controller.GetPinValue(context.Background(), targetPin)
                if err != nil {
                    errors <- fmt.Errorf("goroutine %d verify %d: %w", id, j, err)
                    return
                }

                if actualValue != value {
                    errors <- fmt.Errorf("goroutine %d op %d: expected %v, got %v", id, j, value, actualValue)
                    return
                }
            }
        }(i)
    }

    wg.Wait()
    close(errors)

    // Check for any errors
    var errorList []error
    for err := range errors {
        errorList = append(errorList, err)
    }

    if len(errorList) > 0 {
        t.Fatalf("Encountered %d errors during concurrent access: %v", len(errorList), errorList[0])
    }
}
```

---

## 5. Monitoring and Detection

### 5.1 Race Condition Detection Metrics

```go
// internal/metrics/race_detector.go

type RaceDetector struct {
    concurrentOps  prometheus.Gauge
    lockContention prometheus.Counter
    deadlocks      prometheus.Counter
}

func (r *RaceDetector) RecordLockAcquisition(resource string, duration time.Duration) {
    if duration > 100*time.Millisecond {
        r.lockContention.Inc()
        log.WithFields(logrus.Fields{
            "resource": resource,
            "duration": duration,
        }).Warn("High lock contention detected")
    }
}
```

---

## 6. Summary and Recommendations

### Critical Action Items

1. **Implement Migration Orchestrator** for systemd → K8s transition
2. **Add per-pin locking** for GPIO operations
3. **Enhance SSH connection pooling** with channel-based design
4. **Create pre-installation state capture** mechanism
5. **Implement comprehensive uninstall script**
6. **Add race condition stress tests**
7. **Deploy monitoring for lock contention**

### Priority Matrix

| Issue | Severity | Likelihood | Priority | Status |
|-------|----------|------------|----------|--------|
| K8s migration quorum loss | Critical | High | P0 | Not Implemented |
| GPIO concurrent access | High | Medium | P1 | Partially Protected |
| SSH pool exhaustion | Medium | Low | P2 | Current Design OK |
| mDNS duplicate registration | Low | Medium | P3 | Needs Enhancement |
| Database lock timeout | Medium | Low | P2 | WAL Helps |
| Raft leadership race | Low | Low | P4 | Already Protected |

### Testing Requirements

- [ ] Race detector on all tests
- [ ] Concurrent GPIO stress tests
- [ ] Migration rollback tests
- [ ] State restore validation
- [ ] Uninstall verification

---

## Appendix: Manual Recovery Procedures

If automated uninstall fails, use these manual recovery steps:

```bash
# 1. Emergency GPIO reset
for pin in {0..27}; do
    echo "$pin" > /sys/class/gpio/export 2>/dev/null
    echo "in" > /sys/class/gpio/gpio${pin}/direction 2>/dev/null
    echo "$pin" > /sys/class/gpio/unexport 2>/dev/null
done

# 2. Force stop all processes
pkill -9 pi-controller

# 3. Remove all data
rm -rf /var/lib/pi-controller /etc/pi-controller /var/log/pi-controller

# 4. Clean systemd
systemctl stop pi-controller
systemctl disable pi-controller
rm -f /etc/systemd/system/pi-controller.service
systemctl daemon-reload

# 5. Verify ports released
netstat -tulpn | grep -E "8080|9090|8081|9091"

# 6. Reboot for clean state
reboot
```
