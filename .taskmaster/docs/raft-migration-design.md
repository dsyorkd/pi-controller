# Raft State Migration: systemd → DaemonSet

## Overview

When Kubernetes (K3s) is installed on a cluster running pi-controller via systemd, we need to migrate the Raft cluster from systemd-managed processes to K8s DaemonSet-managed pods without losing quorum or causing downtime.

## Challenge

- **Raft quorum requirement**: N/2 + 1 nodes must be available
- **Port conflicts**: Can't run both systemd and DaemonSet on same ports simultaneously
- **State continuity**: Must preserve Raft log, snapshots, and cluster membership
- **Zero downtime**: No interruption to cluster operations

## Migration Strategy: Parallel Cluster with State Transfer

### Phase 1: Pre-Migration Preparation

1. **Snapshot Current State**
   - Create Raft snapshot on current leader
   - Backup SQLite databases from all nodes
   - Record current Raft cluster configuration (members, leader)

2. **Deploy DaemonSet (Different Ports)**

   ```yaml
   # DaemonSet configuration
   ports:
     - raft: 9092 (temp, normally 9091)
     - grpc: 9093 (temp, normally 9090)
     - rest: 8082 (temp, normally 8080)
   ```

3. **DaemonSet Initialization**
   - Pods start with empty Raft state initially
   - Don't join existing systemd cluster yet

### Phase 2: State Transfer & Bootstrap

1. **Transfer State to DaemonSet Pods**
   - Copy Raft snapshots to each DaemonSet pod's persistent volume
   - Copy SQLite database to each pod
   - Preserve node IDs and membership info

2. **Bootstrap New Raft Cluster in DaemonSet**
   - Initialize Raft with transferred state
   - All DaemonSet pods form new cluster on ports 9092/9093/8082
   - Verify new cluster achieves quorum

3. **Verify Parallel Operation**
   - systemd cluster: ports 9091/9090/8080 (still serving traffic)
   - DaemonSet cluster: ports 9092/9093/8082 (initialized, ready)

### Phase 3: Traffic Cutover

1. **Update Load Balancer / Service**

   ```yaml
   # K8s Service for pi-controller
   apiVersion: v1
   kind: Service
   metadata:
     name: pi-controller
   spec:
     selector:
       app: pi-controller
     ports:
       - name: rest
         port: 8080
         targetPort: 8082  # Points to DaemonSet temp port initially
       - name: grpc
         port: 9090
         targetPort: 9093
       - name: raft
         port: 9091
         targetPort: 9092
   ```

2. **Gradual Cutover**
   - Update ingress/service to point to DaemonSet pods
   - Health check confirms DaemonSet cluster is healthy
   - New requests go to DaemonSet cluster

### Phase 4: Systemd Shutdown

1. **Stop systemd Services**

   ```bash
   # On each node
   sudo systemctl stop pi-controller
   sudo systemctl disable pi-controller
   ```

2. **Free Up Original Ports**
   - Ports 9091, 9090, 8080 now available

### Phase 5: Port Migration

1. **Update DaemonSet to Standard Ports**

   ```yaml
   # Update DaemonSet configuration
   ports:
     - raft: 9091 (standard)
     - grpc: 9090 (standard)
     - rest: 8080 (standard)
   ```

2. **Rolling Restart**
   - K8s performs rolling restart to apply new port config
   - Raft cluster maintains quorum during rolling update

3. **Update Service**

   ```yaml
   # Update Service to point to standard ports
   ports:
     - name: rest
       port: 8080
       targetPort: 8080
     - name: grpc
       port: 9090
       targetPort: 9090
     - name: raft
       port: 9091
       targetPort: 9091
   ```

### Phase 6: Cleanup

1. **Remove systemd Service Files**

   ```bash
   sudo rm /etc/systemd/system/pi-controller.service
   sudo systemctl daemon-reload
   ```

2. **Archive Old Data**

   ```bash
   # Move old data to archive location
   sudo mv /var/lib/pi-controller /var/lib/pi-controller.systemd.backup
   ```

## Implementation Details

### Raft Snapshot Format

```go
type MigrationSnapshot struct {
    RaftSnapshot    []byte            // Raw Raft snapshot data
    DatabaseBackup  []byte            // SQLite database dump
    ClusterConfig   ClusterConfig     // Current Raft members
    NodeID          string            // This node's ID
    Term            uint64            // Current Raft term
    Index           uint64            // Last applied index
    Timestamp       time.Time         // When snapshot was taken
}

type ClusterConfig struct {
    Leader    string              // Current leader ID
    Members   []RaftMember        // All cluster members
    Observers []string            // Non-voting members
}

type RaftMember struct {
    ID      string
    Address string
    Role    string  // "leader", "follower"
}
```

### Migration Service API

```go
type MigrationService interface {
    // Phase 1: Prepare for migration
    CreateMigrationSnapshot() (*MigrationSnapshot, error)
    BackupSystemdState(destPath string) error

    // Phase 2: Transfer state
    TransferStateToK8s(snapshot *MigrationSnapshot) error
    BootstrapK8sRaftCluster(snapshot *MigrationSnapshot) error

    // Phase 3: Cutover
    ValidateK8sCluster() error
    UpdateTrafficRouting() error

    // Phase 4: Shutdown
    StopSystemdServices() error

    // Phase 5: Port migration
    ReconfigureToStandardPorts() error

    // Phase 6: Cleanup
    CleanupSystemdArtifacts() error
    ArchiveOldData() error
}
```

### CLI Commands

```bash
# Prepare for migration
pi-controller migrate prepare \
    --cluster-id=<cluster-id> \
    --backup-path=/var/lib/pi-controller/migration

# Transfer state to K8s
pi-controller migrate transfer \
    --snapshot-path=/var/lib/pi-controller/migration/snapshot.json

# Validate K8s cluster
pi-controller migrate validate

# Execute cutover
pi-controller migrate cutover \
    --confirm

# Complete migration
pi-controller migrate complete \
    --cleanup
```

### Safety Mechanisms

1. **Pre-flight Checks**
   - Verify K3s is installed and healthy
   - Confirm sufficient resources for DaemonSet
   - Validate network connectivity between pods
   - Check persistent volume availability

2. **Rollback Plan**
   - Keep systemd services stopped but not removed
   - Preserve systemd data in archive
   - Can restart systemd cluster if DaemonSet fails

   ```bash
   # Emergency rollback
   pi-controller migrate rollback \
       --restore-from=/var/lib/pi-controller.systemd.backup
   ```

3. **Health Monitoring**
   - Continuous Raft quorum checks during migration
   - Alert if any phase takes longer than expected
   - Automatic pause if errors detected

4. **Idempotency**
   - Each migration phase can be retried
   - State checks prevent duplicate operations
   - Safe to re-run commands

## Configuration

### Migration Config File

```yaml
# .taskmaster/config/migration.yaml
migration:
  # Temporary ports for DaemonSet during migration
  temporary_ports:
    raft: 9092
    grpc: 9093
    rest: 8082

  # Standard ports after migration
  standard_ports:
    raft: 9091
    grpc: 9090
    rest: 8080

  # Paths
  backup_path: /var/lib/pi-controller/migration
  archive_path: /var/lib/pi-controller.systemd.backup

  # Timeouts
  snapshot_timeout: 30s
  transfer_timeout: 2m
  cutover_timeout: 1m

  # Safety settings
  require_confirmation: true
  auto_rollback_on_error: true
  preserve_systemd_backup: true
```

## Testing Strategy

### Integration Test Scenarios

1. **3-Node Cluster Migration**
   - Start with 3-node systemd cluster
   - Execute full migration
   - Verify Raft quorum maintained
   - Confirm zero data loss

2. **5-Node Cluster Migration**
   - Larger cluster stress test
   - Verify rolling update handles multiple nodes
   - Confirm performance acceptable

3. **Failure Scenarios**
   - DaemonSet pod fails during migration
   - Network partition during cutover
   - Insufficient resources for DaemonSet
   - Verify rollback works correctly

4. **State Verification**
   - Compare SQLite data before/after
   - Verify Raft log consistency
   - Confirm GPIO device states preserved
   - Check user accounts and permissions intact

### Manual Test Plan

```bash
# 1. Setup initial systemd cluster
pi-controller cluster create --nodes=pi-1,pi-2,pi-3

# 2. Add test data
pi-controller nodes create --name=test-node
pi-controller gpio create --node=1 --pin=17 --direction=output

# 3. Install K3s
pi-controller kubernetes install --distribution=k3s

# 4. Execute migration
pi-controller migrate prepare
pi-controller migrate transfer
pi-controller migrate validate
pi-controller migrate cutover --confirm
pi-controller migrate complete

# 5. Verify data
pi-controller nodes list  # Should show test-node
pi-controller gpio list   # Should show GPIO device

# 6. Test rollback
pi-controller migrate rollback --confirm
pi-controller nodes list  # Should still show test-node
```

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Quorum loss during cutover | High | Use temporary ports to run parallel clusters |
| Data corruption during transfer | High | Take snapshots, verify checksums |
| Port conflicts | Medium | Graduated port migration strategy |
| Network partition | Medium | Health checks, automatic pause |
| Insufficient resources | Medium | Pre-flight resource validation |
| Rolling update too aggressive | Low | Configure maxUnavailable: 1 in DaemonSet |

## Success Criteria

- ✅ Zero downtime during migration
- ✅ Raft quorum maintained throughout
- ✅ All data preserved (nodes, GPIO configs, users)
- ✅ No manual intervention required after initiation
- ✅ Rollback capability available at each phase
- ✅ Complete in under 5 minutes for 3-node cluster

## Future Enhancements

1. **Automated Migration on K8s Install**
   - Detect systemd cluster during K8s installation
   - Automatically trigger migration
   - Provide progress updates via CLI

2. **Multi-Cluster Federation**
   - Support migrating multiple independent clusters
   - Federated Raft across clusters

3. **Blue-Green Migration**
   - Run both clusters simultaneously
   - Gradual traffic shifting
   - Extended parallel operation period
