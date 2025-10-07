---
layout: default
title: Raft Migration
parent: Deployment
nav_order: 2
---

# Raft State Migration: systemd → DaemonSet
{: .no_toc }

Zero-downtime migration from systemd to Kubernetes deployment
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Overview

When Kubernetes (K3s) is installed on a cluster running pi-controller via systemd, we need to migrate the Raft cluster from systemd-managed processes to K8s DaemonSet-managed pods without losing quorum or causing downtime.

## Challenge

- **Raft quorum requirement**: N/2 + 1 nodes must be available
- **Port conflicts**: Can't run both systemd and DaemonSet on same ports simultaneously
- **State continuity**: Must preserve Raft log, snapshots, and cluster membership
- **Zero downtime**: No interruption to cluster operations

## Migration Strategy

### Phase 1: Pre-Migration Preparation

**Snapshot Current State**
- Create Raft snapshot on current leader
- Backup SQLite databases from all nodes
- Record current Raft cluster configuration (members, leader)

**Deploy DaemonSet (Different Ports)**
```yaml
# DaemonSet configuration
ports:
  - raft: 9092 (temp, normally 9091)
  - grpc: 9093 (temp, normally 9090)
  - rest: 8082 (temp, normally 8080)
```

### Phase 2: State Transfer & Bootstrap

**Transfer State to DaemonSet Pods**
- Copy Raft snapshots to each DaemonSet pod's persistent volume
- Copy SQLite database to each pod
- Preserve node IDs and membership info

**Bootstrap New Raft Cluster**
- Initialize Raft with transferred state
- All DaemonSet pods form new cluster on temporary ports
- Verify new cluster achieves quorum

**Verify Parallel Operation**
- systemd cluster: ports 9091/9090/8080 (still serving traffic)
- DaemonSet cluster: ports 9092/9093/8082 (initialized, ready)

### Phase 3: Traffic Cutover

**Update Kubernetes Service**
```yaml
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
      targetPort: 8082  # Points to DaemonSet temp port
    - name: grpc
      port: 9090
      targetPort: 9093
```

**Gradual Cutover**
- Update ingress/service to point to DaemonSet pods
- Health check confirms DaemonSet cluster is healthy
- New requests go to DaemonSet cluster

### Phase 4: Systemd Shutdown

```bash
# On each node
sudo systemctl stop pi-controller
sudo systemctl disable pi-controller
```

### Phase 5: Port Migration

**Update DaemonSet to Standard Ports**
```yaml
ports:
  - raft: 9091 (standard)
  - grpc: 9090 (standard)
  - rest: 8080 (standard)
```

**Rolling Restart**
- K8s performs rolling restart to apply new port config
- Raft cluster maintains quorum during rolling update

### Phase 6: Cleanup

```bash
# Remove systemd artifacts
sudo rm /etc/systemd/system/pi-controller.service
sudo systemctl daemon-reload

# Archive old data
sudo mv /var/lib/pi-controller /var/lib/pi-controller.systemd.backup
```

## CLI Commands

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
pi-controller migrate cutover --confirm

# Complete migration
pi-controller migrate complete --cleanup
```

## Safety Mechanisms

### Pre-flight Checks
- Verify K3s is installed and healthy
- Confirm sufficient resources for DaemonSet
- Validate network connectivity between pods
- Check persistent volume availability

### Rollback Plan
```bash
# Emergency rollback
pi-controller migrate rollback \
    --restore-from=/var/lib/pi-controller.systemd.backup
```

### Health Monitoring
- Continuous Raft quorum checks during migration
- Alert if any phase takes longer than expected
- Automatic pause if errors detected

## Configuration

```yaml
# migration.yaml
migration:
  temporary_ports:
    raft: 9092
    grpc: 9093
    rest: 8082

  standard_ports:
    raft: 9091
    grpc: 9090
    rest: 8080

  backup_path: /var/lib/pi-controller/migration
  archive_path: /var/lib/pi-controller.systemd.backup

  timeouts:
    snapshot_timeout: 30s
    transfer_timeout: 2m
    cutover_timeout: 1m

  safety:
    require_confirmation: true
    auto_rollback_on_error: true
    preserve_systemd_backup: true
```

## Testing Strategy

### Integration Tests

1. **3-Node Cluster Migration**
   - Start with 3-node systemd cluster
   - Execute full migration
   - Verify Raft quorum maintained
   - Confirm zero data loss

2. **Failure Scenarios**
   - DaemonSet pod fails during migration
   - Network partition during cutover
   - Verify rollback works correctly

### Manual Test Plan

```bash
# Setup initial cluster
pi-controller cluster create --nodes=pi-1,pi-2,pi-3

# Add test data
pi-controller nodes create --name=test-node
pi-controller gpio create --node=1 --pin=17

# Install K3s and migrate
pi-controller kubernetes install --distribution=k3s
pi-controller migrate prepare
pi-controller migrate transfer
pi-controller migrate cutover --confirm

# Verify data preserved
pi-controller nodes list
pi-controller gpio list
```

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Quorum loss during cutover | High | Use temporary ports for parallel clusters |
| Data corruption | High | Snapshots + checksum verification |
| Port conflicts | Medium | Graduated port migration |
| Network partition | Medium | Health checks + automatic pause |
| Resource constraints | Medium | Pre-flight validation |

## Success Criteria

- ✅ Zero downtime during migration
- ✅ Raft quorum maintained throughout
- ✅ All data preserved
- ✅ No manual intervention after initiation
- ✅ Rollback available at each phase
- ✅ Complete in under 5 minutes (3-node cluster)
