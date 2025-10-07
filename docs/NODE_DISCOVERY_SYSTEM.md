---
layout: default
title: Node Discovery
parent: Advanced Topics
nav_order: 1
---

# Node Discovery and Registration System
{: .no_toc }

Automatic node detection and cluster membership
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Overview

The pi-controller supports multiple methods for discovering and registering Raspberry Pi nodes, providing flexibility for different network configurations and deployment scenarios. The system distinguishes between different node types and discovery methods, enabling intelligent management of both controller nodes (for Raft clustering) and agent nodes (for workload execution).

## Node Types

### NodeType Enum

```go
const (
    NodeTypeController  // Node running pi-controller (can join Raft cluster)
    NodeTypeAgent       // Node running pi-agent only (managed worker)
    NodeTypeGeneric     // Generic Raspberry Pi (no pi-controller/pi-agent detected)
    NodeTypeUnknown     // Type not yet determined
)
```

### Type Characteristics

**Controller Nodes:**
- Run the full pi-controller binary
- Can join the Raft cluster for high availability
- Support both cluster management and workload execution
- Typically deployed on 3-5 nodes for production HA

**Agent Nodes:**
- Run only the lightweight pi-agent binary
- Managed by controller nodes
- Execute workloads and provide hardware access (GPIO, sensors)
- Unlimited scaling potential

**Generic Nodes:**
- Raspberry Pi detected on the network
- No pi-controller or pi-agent installed yet
- Available for provisioning

## Discovery Methods

### DiscoveryMethod Enum

```go
const (
    DiscoveryMethodMDNS            // mDNS service discovery (automatic)
    DiscoveryMethodDHCP            // DHCP lease scanning (automatic)
    DiscoveryMethodNetworkScan     // Network scanning (automatic)
    DiscoveryMethodManual          // User manual entry (manual)
    DiscoveryMethodRaftCluster     // Discovered via Raft membership (automatic)
    DiscoveryMethodAPI             // Registered via API (manual)
)
```

## Discovery Flows

### Flow 1: mDNS Discovery of Controllers

**Scenario:** User installs pi-controller on two Raspberry Pis

```
┌──────────────────────────────────────────────────────────────┐
│ 1. User installs pi-controller on Pi-1 and Pi-2             │
│    - Pi-1: 192.168.1.10                                     │
│    - Pi-2: 192.168.1.11                                     │
└──────────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────────┐
│ 2. Both controllers start mDNS broadcasting                  │
│    - Service: _pi-controller._tcp                           │
│    - TXT Records:                                           │
│      * type=controller                                      │
│      * version=v1.0.0                                       │
│      * arch=arm64                                           │
│      * model=Raspberry Pi 4                                 │
└──────────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────────┐
│ 3. Pi-1 discovers Pi-2 via mDNS                              │
│    - Creates node entry:                                    │
│      * discovery_method: mdns                               │
│      * node_type: controller                                │
│      * controller_version: v1.0.0                           │
│      * status: discovered                                   │
└──────────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────────┐
│ 4. User visits Pi-1's web UI                                 │
│    - Sees Pi-1 (local controller)                           │
│    - Sees Pi-2 (discovered controller)                      │
│    - Both marked as "controller" type                       │
└──────────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────────┐
│ 5. User forms Raft cluster                                   │
│    - Clicks "Create Cluster" on Pi-1                        │
│    - Selects Pi-2 to join                                   │
│    - Pi-1 becomes leader                                    │
│    - Both controllers now in sync                           │
└──────────────────────────────────────────────────────────────┘
```

### Flow 2: mDNS Discovery of Generic Nodes

**Scenario:** Raspberry Pi with no pi-controller/agent installed

```
┌──────────────────────────────────────────────────────────────┐
│ 1. Plain Raspberry Pi boots on network                      │
│    - IP: 192.168.1.20                                       │
│    - Running standard Raspberry Pi OS                       │
│    - Broadcasting standard mDNS services                    │
└──────────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────────┐
│ 2. Controller detects generic Raspberry Pi                   │
│    - Creates node entry:                                    │
│      * discovery_method: mdns                               │
│      * node_type: generic                                   │
│      * status: discovered                                   │
└──────────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────────┐
│ 3. User sees node in web UI as "Discovered" type            │
│    - Can provision pi-agent remotely                        │
│    - Can provision full pi-controller                       │
│    - Node transitions to "agent" or "controller" type       │
└──────────────────────────────────────────────────────────────┘
```

### Flow 3: Manual Node Entry

**Scenario:** Node on different subnet or network

```
┌──────────────────────────────────────────────────────────────┐
│ 1. User has Pi on remote network                            │
│    - IP: 10.0.5.50 (different subnet)                       │
│    - Not reachable via mDNS                                 │
└──────────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────────┐
│ 2. User manually adds node via web UI                       │
│    - Enters IP: 10.0.5.50                                   │
│    - Enters name: "remote-pi"                               │
│    - Optionally specifies if controller/agent              │
└──────────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────────┐
│ 3. System creates node entry                                 │
│    - discovery_method: manual                               │
│    - node_type: generic (or user-specified)                │
│    - Attempts to probe for type                            │
└──────────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────────┐
│ 4. Node appears in web UI                                    │
│    - Shows as manually added                                │
│    - Available for provisioning                             │
└──────────────────────────────────────────────────────────────┘
```

### Flow 4: Raft Cluster Discovery

**Scenario:** Node joins via Raft membership

```
┌──────────────────────────────────────────────────────────────┐
│ 1. Administrator calls Raft join API                        │
│    POST /api/v1/raft/members                                │
│    {                                                        │
│      "node_id": "ctrl-3",                                   │
│      "address": "192.168.1.12:9091"                         │
│    }                                                        │
└──────────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────────┐
│ 2. Leader adds node to Raft cluster                         │
│    - Raft consensus for membership change                  │
│    - All controllers updated                                │
└──────────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────────┐
│ 3. System auto-creates node entry if not exists             │
│    - discovery_method: raft_cluster                         │
│    - node_type: controller                                  │
│    - Extracted from Raft member info                       │
└──────────────────────────────────────────────────────────────┘
```

## API Endpoints

### List Nodes with Filtering

```bash
# Get all nodes
GET /api/v1/nodes

# Get only controller nodes
GET /api/v1/nodes?node_type=controller

# Get nodes discovered via mDNS
GET /api/v1/nodes?discovery_method=mdns

# Get generic (unprovisioned) nodes
GET /api/v1/nodes?node_type=generic

# Get manually added nodes
GET /api/v1/nodes?discovery_method=manual

# Combine filters
GET /api/v1/nodes?node_type=controller&discovery_method=mdns
```

### Create Node Manually

```bash
POST /api/v1/nodes
Content-Type: application/json

{
  "name": "remote-pi",
  "ip_address": "10.0.5.50",
  "discovery_method": "manual",
  "node_type": "generic",
  "role": "worker",
  "cpu_cores": 4,
  "memory": 4294967296
}
```

Response:
```json
{
  "data": {
    "id": 5,
    "name": "remote-pi",
    "ip_address": "10.0.5.50",
    "status": "discovered",
    "role": "worker",
    "discovery_method": "manual",
    "discovered_at": "2025-10-05T23:45:00Z",
    "node_type": "generic",
    "architecture": "",
    "model": "",
    "created_at": "2025-10-05T23:45:00Z"
  }
}
```

## Web UI Integration

### Node List View

The web UI displays nodes with visual indicators:

```
┌─────────────────────────────────────────────────────────────┐
│ Available Nodes                                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  🎮 pi-controller-1   192.168.1.10   [Controller] [mDNS]   │
│     Status: Active    Cluster: Leader                      │
│     Version: v1.0.0                                        │
│                                                             │
│  🎮 pi-controller-2   192.168.1.11   [Controller] [mDNS]   │
│     Status: Active    Cluster: Follower                    │
│     Version: v1.0.0                                        │
│                                                             │
│  🤖 pi-worker-1       192.168.1.15   [Agent] [mDNS]        │
│     Status: Ready     Cluster: default                     │
│     GPIO: Available                                        │
│                                                             │
│  📍 pi-generic-1      192.168.1.20   [Generic] [mDNS]      │
│     Status: Discovered                                     │
│     [ Provision pi-agent ] [ Provision pi-controller ]     │
│                                                             │
│  ✏️  remote-pi        10.0.5.50      [Generic] [Manual]    │
│     Status: Discovered                                     │
│     [ Provision pi-agent ] [ Test Connection ]            │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Filter Options

```
Filter by Type:  [ All | Controllers | Agents | Generic ]
Filter by Method: [ All | mDNS | Manual | Network Scan ]
Filter by Status: [ All | Active | Discovered | Failed ]
```

### Actions by Node Type

**Controller Nodes:**
- View cluster status
- Join to Raft cluster
- Remove from cluster
- View logs
- SSH access

**Agent Nodes:**
- View GPIO devices
- Assign to K8s cluster
- View workloads
- Configure

**Generic Nodes:**
- Provision pi-agent
- Provision pi-controller
- Test connectivity
- Remove

## Database Schema

```sql
-- Nodes table with discovery tracking
CREATE TABLE nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    ip_address TEXT NOT NULL,
    mac_address TEXT UNIQUE,
    status TEXT DEFAULT 'discovered' NOT NULL,
    role TEXT DEFAULT 'worker' NOT NULL,

    -- Discovery Information
    discovery_method TEXT NOT NULL DEFAULT 'manual',
    discovered_at DATETIME,
    node_type TEXT NOT NULL DEFAULT 'generic',
    controller_version TEXT,
    agent_port INTEGER DEFAULT 0,

    -- Hardware Information
    architecture TEXT,
    model TEXT,
    serial_number TEXT,
    cpu_cores INTEGER,
    memory INTEGER,

    -- Timestamps
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    last_seen DATETIME
);
```

## mDNS TXT Record Format

### pi-controller Broadcasting

```
Service: _pi-controller._tcp.local.
Port: 8080

TXT Records:
  type=controller
  version=v1.0.0
  arch=arm64
  model=Raspberry Pi 4 Model B
  node_id=ctrl-1
```

### pi-agent Broadcasting

```
Service: _pi-agent._tcp.local.
Port: 9091

TXT Records:
  type=agent
  version=v1.0.0
  arch=arm64
  model=Raspberry Pi 4 Model B
  agent_port=9091
  node_id=worker-1
```

## Example: Complete Setup Flow

### Scenario: 5 Raspberry Pis

```
Setup:
- Pi-1: 192.168.1.10 - Will run pi-controller (bootstrap)
- Pi-2: 192.168.1.11 - Will run pi-controller (join cluster)
- Pi-3: 192.168.1.15 - Will run pi-agent
- Pi-4: 192.168.1.20 - Generic, will provision later
- Pi-5: 10.0.5.50    - On remote network, manual entry
```

### Step-by-Step

**Step 1: Install pi-controller on Pi-1**
```bash
# On Pi-1
wget https://github.com/your/pi-controller/releases/download/v1.0.0/pi-controller
chmod +x pi-controller
./pi-controller --config /etc/pi-controller/config.yaml
```

**Step 2: Access Web UI**
```
Open browser: https://192.168.1.10:3000

View: Empty node list (only local controller visible)
```

**Step 3: Install pi-controller on Pi-2**
```bash
# On Pi-2
wget https://github.com/your/pi-controller/releases/download/v1.0.0/pi-controller
chmod +x pi-controller
./pi-controller --config /etc/pi-controller/config.yaml
```

**Step 4: Pi-1 discovers Pi-2 via mDNS**
```
Pi-1 Web UI now shows:
- pi-1 (local, controller, status: active)
- pi-2 (discovered, controller, status: discovered, method: mDNS)
```

**Step 5: Create Raft Cluster**
```
In Web UI on Pi-1:
1. Click "Create Cluster"
2. Select pi-2 to join
3. Submit

Result:
- Pi-1: Leader
- Pi-2: Follower
- Both controllers in sync
```

**Step 6: Pi-3 with pi-agent auto-discovered**
```bash
# On Pi-3
wget https://github.com/your/pi-controller/releases/download/v1.0.0/pi-agent
chmod +x pi-agent
./pi-agent --controller-url https://192.168.1.10:8080
```

```
Pi-1 Web UI now shows:
- pi-1 (controller, leader)
- pi-2 (controller, follower)
- pi-3 (agent, discovered via mDNS)
```

**Step 7: Pi-4 generic node auto-discovered**
```
Pi-4 boots with standard Raspberry Pi OS

Pi-1 Web UI automatically shows:
- pi-4 (generic, discovered via mDNS)
  Actions: [Provision Agent] [Provision Controller]
```

**Step 8: Manually add Pi-5**
```
In Web UI on Pi-1:
1. Click "Add Node Manually"
2. Enter:
   - Name: remote-pi
   - IP: 10.0.5.50
   - Type: Generic
3. Submit

Result:
- pi-5 added (generic, manual, status: discovered)
```

**Final State in Web UI:**
```
5 Total Nodes:
- 2 Controllers (pi-1 leader, pi-2 follower)
- 1 Agent (pi-3)
- 2 Generic (pi-4 auto-discovered, pi-5 manual)
```

## Benefits

1. **Flexibility:** Supports both automatic and manual discovery
2. **Clear Taxonomy:** Distinguishes controllers from agents from generic nodes
3. **Deployment Tracking:** Know how each node was discovered
4. **Intelligent Management:** Different actions available based on node type
5. **HA Support:** Controllers can form Raft clusters for high availability
6. **Scalability:** Unlimited agent nodes for workload execution
7. **Network Agnostic:** Works across subnets via manual entry

## Future Enhancements

- DHCP lease scanning for automatic discovery
- Network scanning (nmap integration)
- Automatic provisioning workflows
- Node templates for quick setup
- Bulk operations (provision multiple nodes)
- Health monitoring per discovery method
