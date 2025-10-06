# Pi-Controller Architecture (Revised)

## Overview

Pi-Controller is a **single-binary, Raft-based cluster management system** for Raspberry Pi devices. It provides automated discovery, cluster formation, and optional Kubernetes integration with GPIO-as-a-Service capabilities.

**Key Principle**: ONE binary (`pi-controller`) runs on all nodes with identical capabilities. No master/agent distinction - only Raft leader/followers for consensus.

---

## Core Architecture

### Single Binary Model

```
┌─────────────────────────────────────────────────┐
│              PI-CONTROLLER BINARY                │
│                                                  │
│  Components (all nodes run same):               │
│  • REST API (port 8080)                         │
│  • gRPC Server (port 9090)                      │
│  • WebSocket Server (port 8081)                 │
│  • Raft Member (port 9091)                      │
│  • SQLite Database (with WAL)                   │
│  • Local GPIO Manager                           │
│  • mDNS Advertiser/Discoverer                   │
│  • Certificate Authority Client                 │
└─────────────────────────────────────────────────┘
```

**No Agents**: There is no separate `pi-agent` binary. Every node runs the same `pi-controller` binary.

---

## Deployment Modes

### 1. Portable Mode (Laptop/Workstation)

Run pi-controller from a laptop or control plane host to provision and manage remote Raspberry Pis.

**Characteristics**:
- Does NOT participate in Raft cluster
- Acts as secure ingress point for cluster management
- Can run temporarily (exit after provisioning) or persistently (management interface)
- Communicates with cluster via gRPC/REST APIs

**Use Cases**:
- Initial cluster deployment from workstation
- Secure management console for existing cluster
- CI/CD integration for cluster operations

**Example**:
```bash
# Discover available Pis
$ pi-controller discover --scan --interface=en0

# Install pi-controller on discovered nodes and bootstrap Raft cluster
$ pi-controller install \
  --nodes=192.168.1.10,192.168.1.11,192.168.1.12 \
  --bootstrap \
  --ssh-user=pi \
  --ssh-key=~/.ssh/id_rsa

# (Optional) Install Kubernetes
$ pi-controller kubernetes install \
  --distribution=k3s \
  --config=cluster.yaml
```

### 2. On-Device Mode (Raspberry Pi)

Pi-controller runs as a systemd service on each Raspberry Pi node.

**Characteristics**:
- Participates in Raft cluster for consensus
- Exposes all API endpoints (REST, gRPC, WebSocket)
- Manages local GPIO pins
- Replicates cluster state via Raft
- Auto-discovers peers via mDNS

**Installation**:
```bash
# Manual installation
curl -sSL https://install.pi-controller.io | bash

# Or installed via portable mode provisioner
# (systemd service auto-configured)

# Start service
sudo systemctl enable --now pi-controller

# Join existing cluster (via config or API)
sudo pi-controller join --cluster=192.168.1.10:9091
```

---

## Raft Clustering

### Consensus & State Replication

All on-device pi-controller instances form a Raft cluster for distributed consensus.

**What Gets Replicated** (via Raft):
- ✅ Node registry (discovered nodes)
- ✅ GPIO device configurations
- ✅ Cluster metadata
- ✅ User accounts and permissions
- ❌ GPIO readings (too verbose, local-only)

**Cluster Formation**:
1. First node bootstraps Raft cluster (becomes initial leader)
2. Subsequent nodes join via `JoinMember` API
3. Automatic leader election on failure
4. Quorum maintained (N/2 + 1)

**Tuning for Raspberry Pi**:
```yaml
cluster:
  heartbeat_timeout: 1s       # Fast heartbeats for low latency
  election_timeout: 1s        # Quick leader election
  snapshot_interval: 30m      # Periodic state snapshots
  snapshot_threshold: 8192    # Snapshot after 8K log entries
  max_append_entries: 64      # Conservative batch size
```

---

## Node Discovery

### Automatic Discovery (mDNS)

Pi-controller advertises itself via mDNS and discovers peers on the local network.

**mDNS Service**:
- Service Type: `_pi-controller._tcp`
- Port: 9091 (Raft)
- TXT Records:
  - `version=<version>` - Pi-controller version
  - `arch=<architecture>` - CPU architecture (arm64, etc.)
  - `model=<model>` - Raspberry Pi model
  - `node_id=<unique-id>` - Node identifier

**Discovery Process**:
1. Pi-controller starts → Advertises via mDNS
2. Listens for other pi-controller instances
3. Auto-registers discovered nodes in database
4. Can be invited to join Raft cluster

### Manual Discovery

Nodes can be added manually via API or CLI:

```bash
# Add node manually
$ pi-controller nodes add --ip=192.168.1.20 --name=pi-remote

# Invite to join Raft cluster
$ pi-controller cluster invite --node-id=<id>
```

---

## GPIO Management

### State Model (Simplified)

GPIO devices represent **desired state** without reservations or ownership.

**GPIODevice Model**:
```go
type GPIODevice struct {
    ID          uint
    Name        string
    NodeID      uint           // Which Pi this pin is on
    PinNumber   int            // Physical pin number
    Direction   GPIODirection  // input/output
    PullMode    GPIOPullMode   // none/up/down
    DeviceType  GPIODeviceType // digital/analog/pwm/spi/i2c
    Status      GPIOStatus     // active/inactive/error
    Value       int            // Current/desired value
    Config      GPIOConfig     // Device-specific config (JSON)
}
```

**No Reservations**: Removed reservation system. GPIO state is simply:
- **Desired value**: What it should be
- **Current value**: What it actually is (from hardware)

### GPIO Access Pattern

Since all nodes run the same binary, GPIO access works via direct gRPC:

```
User/API → Raft Leader → Direct gRPC to Target Pi → Local GPIO Hardware
```

**Example Flow**:
1. User requests GPIO pin 17 on Pi-2 be set to HIGH
2. Request hits Pi-1 (current Raft leader)
3. Pi-1 updates GPIO device state in Raft-replicated database
4. Pi-1 makes gRPC call to Pi-2's gRPC server
5. Pi-2 sets physical pin 17 to HIGH
6. Pi-2 returns current value
7. State persisted in replicated database

**Reading GPIO**:
- Readings NOT replicated (too verbose)
- Direct gRPC call to target node
- Stored locally in each node's SQLite database
- API exposes readings from specific node

---

## Kubernetes Integration (Optional)

### K3s/K8s Provisioning

Users can optionally deploy Kubernetes (K3s, K0s, etc.) on the cluster via Web UI or CLI.

**Provisioning**:
```bash
$ pi-controller kubernetes install \
  --distribution=k3s \
  --server-nodes=pi-1,pi-2,pi-3 \
  --agent-nodes=pi-4,pi-5 \
  --config=cluster.yaml
```

**What Happens**:
1. Pi-controller provisions K3s cluster (servers + agents)
2. Deploys pi-controller as **DaemonSet** on all K8s nodes
3. Registers CRDs: `GPIOPin`, `PWMController`, `I2CDevice`
4. Deploys operator to reconcile CRDs
5. **Migrates Raft state** from systemd cluster → DaemonSet cluster
6. **Disables systemd service** to prevent port conflicts
7. **Removes systemd service** cleanly

### DaemonSet Migration

Critical transition: systemd Raft cluster → Kubernetes DaemonSet Raft cluster

**Migration Process** (without losing quorum):
1. DaemonSet pods start on all nodes (different ports temporarily)
2. DaemonSet cluster initializes with bootstrapped state from systemd cluster
3. DaemonSet cluster achieves quorum
4. Systemd services gracefully shut down one-by-one
5. Systemd services disabled and removed
6. DaemonSet cluster takes over on standard ports

**Post-Migration**:
- Pi-controller runs exclusively as K8s DaemonSet
- No systemd service (removed)
- Raft cluster maintained, now managed by K8s
- CRD-based GPIO management available

---

## Web UI (kubes-aura)

The Web UI is maintained in a **separate repository**: `kubes-aura`

**Integration**:
- Connects to pi-controller via gRPC client
- Manages:
  - Cluster nodes
  - GPIO devices
  - Kubernetes deployments (if installed)
  - User accounts
  - Monitoring dashboards

**Deployment Options**:
1. Standalone (separate host)
2. K8s Deployment (if K8s installed)
3. Embedded in pi-controller (future consideration)

---

## Security Model

### Authentication & Authorization

- **JWT-based authentication** with role-based access control (RBAC)
- **Roles**: `viewer`, `operator`, `admin`
- **HTTPS mandatory** in production
- **mTLS** for inter-node Raft communication

### Certificate Authority

Pi-controller includes embedded CA for certificate management:
- **Local CA** (default): Self-hosted PKI
- **Vault PKI** (optional): Integration with HashiCorp Vault
- Auto-generates certificates for nodes
- Handles renewal and revocation

### GPIO Safety

- **Pin validation**: Restricted pins (I2C, UART) cannot be configured
- **Hardware protection**: Rate limiting prevents rapid switching damage
- **Audit logging**: All GPIO operations logged

---

## Database & State

### SQLite with Raft Replication

Each pi-controller instance maintains:
- **SQLite database** (local storage)
- **Raft log** (replicated state changes)
- **WAL mode** (write-ahead logging for SD card protection)

**Replication Strategy**:
- Raft replicates: Node configs, GPIO devices, cluster metadata, users
- Local-only: GPIO readings, logs, temporary state

**SD Card Optimization**:
- WAL mode reduces write amplification
- Batched GPIO readings (5-second windows)
- Periodic snapshots to limit log growth
- A2-rated SD cards recommended

---

## API Endpoints

### REST API (Port 8080)

```
/api/v1/nodes          - Node management
/api/v1/clusters       - Cluster operations
/api/v1/gpio           - GPIO device management
/api/v1/ca             - Certificate authority
/api/v1/raft           - Raft cluster management
/api/v1/auth           - Authentication
/health                - Health check
/ready                 - Readiness probe
```

### gRPC Server (Port 9090)

- Node-to-node communication
- GPIO hardware operations
- Cluster coordination
- Used by Web UI (kubes-aura)

### WebSocket Server (Port 8081)

- Real-time event streaming
- GPIO value changes
- Node status updates
- Cluster events

---

## Deployment Architecture Diagram

```
┌──────────────────────────────────────────────────────────────┐
│                   DEPLOYMENT OPTIONS                         │
└──────────────────────────────────────────────────────────────┘

OPTION 1: STANDALONE (No Kubernetes)
═══════════════════════════════════════

  Laptop (Portable Mode)           Raspberry Pi Cluster
  ┌────────────────┐               ┌──────────┬──────────┬──────────┐
  │ pi-controller  │───────────────│  Pi-1    │  Pi-2    │  Pi-3    │
  │ (client mode)  │   SSH/gRPC    │          │          │          │
  │                │               │ systemd  │ systemd  │ systemd  │
  │ • Provision    │               │ Raft     │ Raft     │ Raft     │
  │ • Discover     │               │ SQLite   │ SQLite   │ SQLite   │
  │ • Manage       │               │ GPIO     │ GPIO     │ GPIO     │
  └────────────────┘               └──────────┴──────────┴──────────┘
                                          ▲         ▲         ▲
                                          └─── Raft Cluster ──┘


OPTION 2: WITH KUBERNETES (K3s)
════════════════════════════════

  Web UI (kubes-aura)              Kubernetes Cluster (K3s)
  ┌────────────────┐               ┌──────────────────────────────┐
  │ gRPC Client    │───────────────│  DaemonSet: pi-controller    │
  │                │               │  ┌────┐  ┌────┐  ┌────┐     │
  │ • Dashboard    │               │  │Pi-1│  │Pi-2│  │Pi-3│     │
  │ • GPIO Control │               │  └────┘  └────┘  └────┘     │
  │ • K8s Mgmt     │               │    ▲       ▲       ▲         │
  └────────────────┘               │    └── Raft Cluster ─┘      │
                                    │                             │
                                    │  CRDs:                      │
                                    │  • GPIOPin                  │
                                    │  • PWMController            │
                                    │  • I2CDevice                │
                                    └─────────────────────────────┘

  systemd services: REMOVED (no port conflicts)
```

---

## Migration Path: systemd → DaemonSet

When K8s is installed, pi-controller transitions from systemd to DaemonSet.

**Before Migration**:
```
Pi-1: systemd pi-controller (Raft leader)
Pi-2: systemd pi-controller (Raft follower)
Pi-3: systemd pi-controller (Raft follower)
```

**During Migration**:
```
Pi-1: systemd (port 9091) + DaemonSet pod (port 9092)  ← Parallel running
Pi-2: systemd (port 9091) + DaemonSet pod (port 9092)  ← State sync
Pi-3: systemd (port 9091) + DaemonSet pod (port 9092)  ← Quorum maintained
```

**After Migration**:
```
Pi-1: DaemonSet pod only (port 9091)  ← systemd disabled & removed
Pi-2: DaemonSet pod only (port 9091)  ← systemd disabled & removed
Pi-3: DaemonSet pod only (port 9091)  ← systemd disabled & removed
```

**Key Insight**: Clever migration ensures no quorum loss during transition.

---

## Configuration

### Deployment Modes

**Portable Mode** (environment variable):
```bash
PI_CONTROLLER_MODE=portable    # Client-only, no Raft
```

**On-Device Mode** (default):
```yaml
cluster:
  enabled: true                 # Participate in Raft
  controller_id: pi-1          # Unique node ID
  bind_addr: 192.168.1.10:9091 # Raft bind address
  bootstrap: true               # First node in cluster
```

### Kubernetes Integration

```yaml
kubernetes:
  enabled: true
  distribution: k3s
  config_path: /etc/pi-controller/k3s.yaml
```

---

## Key Design Decisions

1. **Single Binary**: Simplifies deployment, reduces attack surface
2. **No Agents**: All nodes equal peers, Raft determines leader
3. **Portable Mode**: Enables remote management without cluster participation
4. **Optional K8s**: Core functionality works standalone, K8s is additive
5. **Simple GPIO State**: No reservations, just desired vs. current state
6. **Raft for Coordination**: Proven consensus for distributed state
7. **mDNS Discovery**: Zero-config node discovery on local networks
8. **Embedded CA**: Self-contained PKI, no external dependencies
9. **SQLite + WAL**: SD card-friendly persistence
10. **DaemonSet Migration**: Smooth transition without quorum loss

---

## Differences from Original Architecture

### What Changed:

❌ **Removed**:
- Separate `pi-agent` binary
- NodeTypeAgent / NodeTypeController distinctions
- GPIO pin reservation system (ReservedBy, ReservationTTL)
- Agent-specific discovery logic
- Embedded Web UI (moved to kubes-aura repo)

✅ **Added/Clarified**:
- Portable mode (client-only operation)
- Single binary for all nodes
- Raft-based clustering (all nodes equal)
- DaemonSet migration strategy
- Simplified GPIO state model
- Separate Web UI repository (kubes-aura)

### Philosophy Shift:

**Before**: Master/Agent hierarchy
**After**: Peer-to-peer Raft cluster

**Before**: GPIO ownership/reservation
**After**: Simple desired/current state

**Before**: Always runs on-device
**After**: Can run portably as client

---

## Future Considerations

- **Multi-cluster Federation**: Manage multiple Raft clusters from single interface
- **Observability**: Prometheus metrics, distributed tracing, log aggregation
- **Backup & Restore**: Automated Raft snapshot backups to S3/Minio
- **Security Hardening**: Hardware security modules (HSM), TPM integration
- **Network Partitioning**: Graceful handling of split-brain scenarios
- **Auto-scaling**: Dynamic node addition/removal based on load
- **Edge Computing**: Integration with edge platforms (KubeEdge, K3s Edge)

---

## Summary

Pi-Controller provides a unified, single-binary solution for managing Raspberry Pi clusters with:
- Automatic discovery via mDNS
- Raft-based distributed consensus
- Optional Kubernetes integration
- Built-in Certificate Authority
- GPIO-as-a-Service
- Portable or on-device deployment

**Key Strength**: Simple deployment model with powerful distributed capabilities.
