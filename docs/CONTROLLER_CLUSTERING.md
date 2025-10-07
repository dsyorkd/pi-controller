---
layout: default
title: Clustering & Raft
parent: Architecture
nav_order: 2
---

# Pi-Controller Clustering Architecture
{: .no_toc }

High-availability clustering using HashiCorp Raft consensus
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Overview

This document describes the high-availability clustering architecture for pi-controller binaries, enabling multiple controller instances to work together for fault tolerance and load distribution.

## Architecture Goals

1. **High Availability**: Survive single controller failures
2. **Data Consistency**: Maintain consistent state across controllers
3. **Automatic Failover**: Seamless transition between controller instances
4. **Load Distribution**: Distribute API requests across healthy controllers
5. **Simple Deployment**: Easy to configure and operate

## Clustering Modes

### 1. Active-Passive (Recommended for Start)
- One active controller, others standby
- Fast failover with leader election
- Simple state management
- Lower resource usage

### 2. Active-Active (Future Enhancement)
- All controllers serve requests
- Distributed state synchronization
- Higher throughput
- More complex implementation

## Component Design

### Controller Cluster Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Controller Cluster                           │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │  Controller 1   │  │  Controller 2   │  │  Controller 3   │  │
│  │   (Leader)      │  │   (Follower)    │  │   (Follower)    │  │
│  ├─────────────────┤  ├─────────────────┤  ├─────────────────┤  │
│  │ Leader Election │  │ Leader Election │  │ Leader Election │  │
│  │ State Manager   │  │ State Manager   │  │ State Manager   │  │
│  │ Health Monitor  │  │ Health Monitor  │  │ Health Monitor  │  │
│  │ API Server      │  │ API Server      │  │ API Server      │  │
│  │ SQLite (RW)     │  │ SQLite (RO)     │  │ SQLite (RO)     │  │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘  │
│           │                    │                     │           │
│           └────────────────────┼─────────────────────┘           │
│                                │                                 │
│  ┌─────────────────────────────▼───────────────────────────────┐ │
│  │              Consensus Layer (etcd/Raft)                    │ │
│  │  • Leader election coordination                             │ │
│  │  • Distributed configuration                                │ │
│  │  • Cluster membership tracking                              │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Core Components

#### 1. Leader Election Module

**Purpose**: Ensure only one controller is active (leader)

**Implementation Options**:
- **Option A**: etcd-based (recommended)
  - Leverages existing K3s embedded etcd
  - Battle-tested in production
  - Built-in lease management

- **Option B**: Raft consensus library
  - Self-contained, no external dependencies
  - More control over implementation
  - Requires custom integration

**Features**:
- Automatic leader election on startup
- Leader lease with TTL (30 seconds default)
- Automatic failover on leader failure
- Split-brain prevention

```go
type LeaderElector interface {
    // Run starts the leader election loop
    Run(ctx context.Context) error

    // IsLeader returns true if this controller is the leader
    IsLeader() bool

    // GetLeader returns the current leader ID
    GetLeader() (string, error)

    // OnBecomeLeader sets callback for when this controller becomes leader
    OnBecomeLeader(callback func())

    // OnLoseLeadership sets callback for when this controller loses leadership
    OnLoseLeadership(callback func())
}
```

#### 2. State Synchronization

**Purpose**: Replicate SQLite data across controller instances

**Strategies**:

**Option A: WAL Shipping (Recommended for SQLite)**
```
Leader Controller (192.168.1.10)
├── SQLite DB with WAL mode enabled
├── WAL file watcher
└── WAL shipper → Sends WAL segments to followers

Follower Controllers (192.168.1.11, 192.168.1.12)
├── SQLite DB (read-only)
├── WAL receiver
└── WAL replayer → Applies WAL segments
```

**Features**:
- Near real-time replication
- Minimal overhead
- Transaction-level consistency
- No schema changes required

**Implementation**:
```go
type StateReplicator interface {
    // Start begins replication process
    Start(ctx context.Context) error

    // Replicate sends state updates to followers
    Replicate(walSegment []byte) error

    // Receive applies state updates from leader
    Receive(walSegment []byte) error

    // GetReplicationLag returns current lag in seconds
    GetReplicationLag() time.Duration
}
```

**Option B: Snapshot + Incremental Sync**
- Full database snapshot periodically
- Incremental changes via gRPC streaming
- Higher network overhead
- Simpler implementation

#### 3. Health Monitoring

**Purpose**: Detect controller failures and trigger failover

**Components**:
```go
type HealthChecker interface {
    // Check performs health check on this controller
    Check() HealthStatus

    // Monitor starts health monitoring of other controllers
    Monitor(ctx context.Context) error

    // GetClusterHealth returns health status of all controllers
    GetClusterHealth() []ControllerHealth
}

type HealthStatus struct {
    Healthy        bool
    LeaderElection bool   // Can participate in leader election
    APIServer      bool   // API server responding
    Database       bool   // Database accessible
    Replication    bool   // Replication working
    LastCheck      time.Time
    Message        string
}
```

**Health Checks**:
1. **Leader Election Participation**: Can acquire/renew lease
2. **API Server**: HTTP endpoint responding
3. **Database**: SQLite queries succeeding
4. **Replication**: WAL sync lag < threshold
5. **System Resources**: CPU, memory, disk within limits

**Check Intervals**:
- Self health check: Every 5 seconds
- Peer health check: Every 10 seconds
- Leader lease renewal: Every 10 seconds (30s TTL)

#### 4. Cluster Membership

**Purpose**: Track which controllers are part of the cluster

```go
type MembershipManager interface {
    // Join adds this controller to the cluster
    Join(ctx context.Context, clusterEndpoints []string) error

    // Leave removes this controller from the cluster
    Leave(ctx context.Context) error

    // GetMembers returns all cluster members
    GetMembers() []ClusterMember

    // RemoveMember removes a failed member
    RemoveMember(memberID string) error
}

type ClusterMember struct {
    ID            string
    Address       string
    Role          MemberRole  // leader, follower
    Status        MemberStatus // healthy, degraded, failed
    JoinedAt      time.Time
    LastSeenAt    time.Time
}
```

## Data Flow

### Normal Operation (Leader Active)

```
Client Request
    │
    ▼
Load Balancer (HAProxy/VIP)
    │
    ├──► Controller 1 (Leader) ──► SQLite (Read/Write) ──► Response
    │
    ├──► Controller 2 (Follower) ─┐
    │                              ├──► Redirect to Leader
    └──► Controller 3 (Follower) ─┘
```

### Leader Failure Scenario

```
1. Leader (Controller 1) fails
   │
   ▼
2. Lease expires (30 seconds)
   │
   ▼
3. Followers detect leader loss
   │
   ▼
4. New leader election triggered
   │
   ▼
5. Controller 2 wins election
   │
   ▼
6. Controller 2 becomes leader
   │
   ├──► Promotes SQLite to read-write
   ├──► Starts serving API requests
   └──► Updates cluster state
   │
   ▼
7. Controller 3 syncs with new leader
   │
   ▼
8. System operational (total downtime: 30-45 seconds)
```

## Configuration

### Clustering Configuration

```yaml
# config.yaml
cluster:
  # Enable controller clustering
  enabled: true

  # Unique controller ID (auto-generated if not set)
  controllerId: "controller-01"

  # Cluster name (all controllers must have same name)
  clusterName: "pi-controller-cluster"

  # Initial cluster members (for bootstrapping)
  initialMembers:
    - "https://192.168.1.10:9091"
    - "https://192.168.1.11:9091"
    - "https://192.168.1.12:9091"

  # Leader election configuration
  leaderElection:
    # Backend: etcd or raft
    backend: "etcd"

    # Lease TTL (seconds)
    leaseTTL: 30

    # Lease renewal interval (seconds)
    renewalInterval: 10

    # etcd endpoints (if using etcd backend)
    etcdEndpoints:
      - "https://192.168.1.10:2379"
      - "https://192.168.1.11:2379"
      - "https://192.168.1.12:2379"

  # State replication configuration
  replication:
    # Method: wal-shipping or snapshot
    method: "wal-shipping"

    # WAL shipping interval (milliseconds)
    walShippingInterval: 1000

    # Maximum replication lag before warning (seconds)
    maxLag: 10

    # Snapshot interval (if using snapshot method)
    snapshotInterval: "1h"

  # Health check configuration
  healthCheck:
    # Self health check interval (seconds)
    selfCheckInterval: 5

    # Peer health check interval (seconds)
    peerCheckInterval: 10

    # Unhealthy threshold (failed checks before marking unhealthy)
    unhealthyThreshold: 3

  # Communication configuration
  communication:
    # Inter-controller gRPC port
    grpcPort: 9091

    # TLS configuration
    tls:
      enabled: true
      certFile: "/etc/pi-controller/tls/controller-cert.pem"
      keyFile: "/etc/pi-controller/tls/controller-key.pem"
      caFile: "/etc/pi-controller/tls/ca-cert.pem"
```

## Deployment Scenarios

### Scenario 1: 3-Node Cluster (Recommended)

**Hardware**:
- 3x Raspberry Pi 4 (4GB+ RAM)
- Each running pi-controller + K3s server

**Setup**:
```bash
# Node 1 (Bootstrap)
pi-controller server \
  --cluster-init \
  --cluster-id=controller-01 \
  --cluster-members=https://192.168.1.10:9091,https://192.168.1.11:9091,https://192.168.1.12:9091

# Node 2
pi-controller server \
  --cluster-join \
  --cluster-id=controller-02 \
  --cluster-members=https://192.168.1.10:9091

# Node 3
pi-controller server \
  --cluster-join \
  --cluster-id=controller-03 \
  --cluster-members=https://192.168.1.10:9091
```

**Benefits**:
- Survives 1 node failure
- Maintains quorum with 2/3 nodes
- Optimal for small deployments

### Scenario 2: 5-Node Cluster (High Availability)

**Hardware**:
- 5x Raspberry Pi 4
- Can survive 2 node failures
- Better load distribution

**Quorum**: 3/5 nodes required

### Scenario 3: 2-Node Cluster + Witness (Budget)

**Hardware**:
- 2x Raspberry Pi 4 (controllers)
- 1x Raspberry Pi Zero (witness node, no storage)

**Features**:
- Witness node only participates in leader election
- Cheaper than full 3-node cluster
- Can survive 1 controller failure

## API Changes

### Cluster Status Endpoint

```http
GET /api/v1/cluster/status
```

**Response**:
```json
{
  "clusterId": "pi-controller-cluster",
  "members": [
    {
      "id": "controller-01",
      "address": "https://192.168.1.10:9091",
      "role": "leader",
      "status": "healthy",
      "joinedAt": "2025-01-30T10:00:00Z",
      "lastSeenAt": "2025-01-30T10:30:15Z"
    },
    {
      "id": "controller-02",
      "address": "https://192.168.1.11:9091",
      "role": "follower",
      "status": "healthy",
      "joinedAt": "2025-01-30T10:01:00Z",
      "lastSeenAt": "2025-01-30T10:30:16Z"
    },
    {
      "id": "controller-03",
      "address": "https://192.168.1.12:9091",
      "role": "follower",
      "status": "healthy",
      "joinedAt": "2025-01-30T10:02:00Z",
      "lastSeenAt": "2025-01-30T10:30:14Z"
    }
  ],
  "leader": "controller-01",
  "quorum": true,
  "replicationLag": {
    "controller-02": "150ms",
    "controller-03": "200ms"
  }
}
```

### Leader Transfer Endpoint (Admin Only)

```http
POST /api/v1/cluster/leader/transfer
```

**Request**:
```json
{
  "targetController": "controller-02"
}
```

**Purpose**: Gracefully transfer leadership (for maintenance)

## Load Balancing

### HAProxy Configuration

```
global
  daemon
  maxconn 4096

defaults
  mode http
  timeout connect 5000ms
  timeout client 50000ms
  timeout server 50000ms

# API Load Balancer (distributes to any healthy controller)
frontend pi_controller_api
  bind *:8080 ssl crt /etc/ssl/certs/pi-controller.pem
  default_backend pi_controller_backends

backend pi_controller_backends
  balance roundrobin
  option httpchk GET /health
  http-check expect status 200

  # All controllers can serve read requests
  server controller-01 192.168.1.10:8080 check ssl verify none
  server controller-02 192.168.1.11:8080 check ssl verify none
  server controller-03 192.168.1.12:8080 check ssl verify none

# Write requests redirected to leader by application logic
```

### Keepalived VIP Configuration

```
vrrp_instance PI_CONTROLLER {
  state MASTER
  interface eth0
  virtual_router_id 51
  priority 100
  advert_int 1

  authentication {
    auth_type PASS
    auth_pass secret123
  }

  virtual_ipaddress {
    192.168.1.100/24
  }
}
```

**Access**: `https://192.168.1.100:8080` (VIP always points to healthy controller)

## Failure Scenarios & Recovery

### Scenario 1: Leader Crashes

**Timeline**:
1. Leader process crashes (t=0s)
2. Lease expires (t=30s)
3. Followers detect leader loss (t=30s)
4. Election starts (t=30s)
5. New leader elected (t=35s)
6. New leader promoted (t=40s)
7. System operational (t=45s)

**Impact**: 30-45 seconds of API unavailability for writes

### Scenario 2: Network Partition

**Partition**: Controller 1 isolated from Controllers 2 & 3

**Behavior**:
1. Controller 1 loses majority (1/3)
2. Controller 1 steps down as leader
3. Controllers 2 & 3 maintain quorum (2/3)
4. Controller 2 or 3 becomes new leader
5. System continues with 2 controllers

**When partition heals**:
- Controller 1 rejoins as follower
- Syncs state from current leader
- Becomes available for failover

### Scenario 3: Split Brain Prevention

**Situation**: Two controllers both think they're leader

**Prevention**:
- Lease-based leadership (only one valid lease)
- Quorum requirement (majority must agree)
- Term-based leadership (higher term wins)
- Fencing (old leader detected and forced to step down)

## Implementation Phases

### Phase 1: Leader Election (MVP)
- Implement etcd-based leader election
- Add leader/follower roles
- Follower redirects to leader for writes
- Basic health checking

**Deliverables**:
- `internal/clustering/election/` package
- Leader election integration in main server
- `/api/v1/cluster/status` endpoint

### Phase 2: State Replication
- Implement WAL shipping for SQLite
- Add replication monitoring
- Automatic catchup for lagging followers

**Deliverables**:
- `internal/clustering/replication/` package
- WAL shipper and receiver
- Replication lag metrics

### Phase 3: Advanced Features
- Graceful leader transfer
- Witness node support
- Automatic member removal
- Prometheus metrics export

## Monitoring & Observability

### Key Metrics

```prometheus
# Leadership status
pi_controller_is_leader{controller_id="controller-01"} 1

# Cluster membership
pi_controller_cluster_members 3

# Replication lag
pi_controller_replication_lag_seconds{target="controller-02"} 0.15

# Election count
pi_controller_leader_elections_total 5

# Health status
pi_controller_health_status{component="api"} 1
pi_controller_health_status{component="database"} 1
pi_controller_health_status{component="replication"} 1
```

### Alerts

```yaml
# Leader election failing
- alert: ControllerNoLeader
  expr: sum(pi_controller_is_leader) == 0
  for: 1m
  severity: critical

# Replication lag high
- alert: ControllerReplicationLag
  expr: pi_controller_replication_lag_seconds > 10
  for: 5m
  severity: warning

# Cluster degraded
- alert: ControllerClusterDegraded
  expr: pi_controller_cluster_members < 3
  for: 5m
  severity: warning
```

## Migration Path

### From Single Controller to Clustered

1. **Backup existing data**:
   ```bash
   sqlite3 /var/lib/pi-controller/data.db ".backup /backup/data.db"
   ```

2. **Deploy additional controllers**:
   - Install pi-controller on 2 more nodes
   - Copy TLS certificates to all nodes

3. **Enable clustering**:
   ```yaml
   cluster:
     enabled: true
     initialMembers:
       - https://node1:9091
       - https://node2:9091
       - https://node3:9091
   ```

4. **Restart controllers**:
   ```bash
   systemctl restart pi-controller
   ```

5. **Verify cluster status**:
   ```bash
   curl https://localhost:8080/api/v1/cluster/status
   ```

6. **Configure load balancer**:
   - Point HAProxy to all 3 controllers
   - Set up VIP with keepalived

## Security Considerations

1. **mTLS Between Controllers**:
   - All inter-controller communication encrypted
   - Certificate-based authentication
   - Automatic certificate rotation

2. **Consensus Security**:
   - etcd client certificates required
   - Encrypted etcd communication
   - Strong authentication tokens

3. **Data Protection**:
   - Encrypted WAL shipping
   - Database encryption at rest
   - Secure deletion of old WAL segments

## Testing Strategy

### Unit Tests
- Leader election logic
- WAL shipping/receiving
- Health check conditions
- Membership management

### Integration Tests
- 3-node cluster formation
- Leader failover
- State synchronization
- Network partition handling

### Chaos Testing
- Random controller kills
- Network latency injection
- Disk failure simulation
- Time skew testing

## Performance Considerations

### Resource Usage
- **Memory**: +50MB per follower (for replication buffers)
- **CPU**: +5% (for health checks and replication)
- **Network**: ~100KB/s per follower (WAL shipping)
- **Disk**: +10% (for WAL retention)

### Scalability
- Tested up to 5 controllers
- Recommended max: 7 controllers (quorum 4/7)
- Beyond 7: consider separate control/data planes

---

**Document Version**: 1.0
**Last Updated**: 2025-01-30
**Status**: Design - Ready for Implementation