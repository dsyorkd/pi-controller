# Pi-Controller System Architecture

## Executive Summary

Pi-Controller is a comprehensive Kubernetes management platform designed specifically for Raspberry Pi clusters. The system provides automated discovery, provisioning, and lifecycle management of K3s clusters while offering GPIO-as-a-Service capabilities through Kubernetes Custom Resources.

## 1. System Overview

### Core Philosophy
- **Single Binary Deployment**: Control plane runs as one Go binary with embedded web UI
- **Zero Dependencies**: No external automation tools (Ansible, Terraform, etc.)
- **Pi-Native**: Optimized for ARM64/ARMv7 hardware constraints
- **Kubernetes-First**: GPIO and hardware control through standard K8s APIs
- **Homelab-Friendly**: Simple deployment with enterprise-grade scalability

### Architecture Principles
- **Distributed Control**: Every Pi can act as control plane node
- **Resilient Communication**: Hybrid gRPC/REST with automatic failover
- **Hardware Abstraction**: GPIO operations via Kubernetes CRDs
- **State Reconciliation**: Continuous drift detection and correction
- **Event-Driven**: Real-time updates via WebSocket streams

## 2. System Components

### 2.1 Control Plane Components

```
┌─────────────────────────────────────────────────────────────────┐
│                    Pi-Controller Control Plane                  │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────┐ │
│  │ Discovery   │  │ Provisioner │  │ Cluster     │  │ GPIO    │ │
│  │ Service     │  │ Engine      │  │ Manager     │  │ CRD     │ │
│  │             │  │             │  │             │  │ Manager │ │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────┘ │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────┐ │
│  │ Web         │  │ CLI         │  │ MCP         │  │ Event   │ │
│  │ Frontend    │  │ Interface   │  │ Server      │  │ Bus     │ │
│  │             │  │             │  │             │  │         │ │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────┘ │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │ State       │  │ Certificate │  │ Backup      │              │
│  │ Database    │  │ Manager     │  │ Manager     │              │
│  │ (SQLite)    │  │             │  │             │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Node Components (DaemonSet)

```
┌─────────────────────────────────────────────────────────────────┐
│                    Pi-Controller Node Agent                     │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────┐ │
│  │ System      │  │ GPIO        │  │ Hardware    │  │ Health  │ │
│  │ Monitor     │  │ Controller  │  │ Monitor     │  │ Check   │ │
│  │             │  │             │  │             │  │ Agent   │ │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────┘ │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │ gRPC        │  │ Metrics     │  │ Log         │              │
│  │ Server      │  │ Collector   │  │ Collector   │              │
│  │             │  │             │  │             │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└─────────────────────────────────────────────────────────────────┘
```

## 3. Detailed Component Specifications

### 3.1 Discovery Service
**Purpose**: Automatic Pi node discovery and network mapping
**Technology**: mDNS, network scanning, DHCP lease parsing

**Key Features**:
- Multi-protocol discovery (mDNS, network scan, manual registration)
- Hardware capability detection (GPIO pins, I2C buses, SPI interfaces)
- Network topology mapping with bandwidth testing
- Automatic cluster membership management

**Implementation Details**:
```go
type DiscoveryService struct {
    mdnsClient     *mdns.Client
    networkScanner *NetworkScanner
    nodeRegistry   *NodeRegistry
    capabilities   *CapabilityDetector
}

type DiscoveredNode struct {
    ID           string
    IPAddress    net.IP
    MACAddress   net.HardwareAddr
    Architecture string
    Capabilities NodeCapabilities
    LastSeen     time.Time
    Status       NodeStatus
}
```

### 3.2 Provisioner Engine
**Purpose**: Automated K3s cluster bootstrapping and node joining
**Technology**: SSH, K3s installation scripts, certificate management

**Key Features**:
- Zero-touch K3s installation with custom configurations
- High-availability control plane setup (embedded etcd)
- Automatic node joining with secure token management
- Custom CNI and CSI driver installation
- Pi-optimized K3s configurations (memory limits, storage)

**Implementation Details**:
```go
type ProvisionerEngine struct {
    sshPool        *SSHConnectionPool
    certManager    *CertificateManager
    k3sInstaller   *K3sInstaller
    configRenderer *ConfigTemplateRenderer
}

type ProvisioningPlan struct {
    ClusterID      string
    ControlPlanes  []NodeSpec
    Workers        []NodeSpec
    Configuration  K3sConfig
    NetworkConfig  NetworkConfig
    StorageConfig  StorageConfig
}
```

### 3.3 Cluster Manager
**Purpose**: K8s cluster lifecycle management and state reconciliation
**Technology**: Kubernetes client-go, custom controllers

**Key Features**:
- Multi-cluster state synchronization
- Workload placement optimization based on Pi capabilities
- Resource quota management per cluster
- Automated rolling updates and maintenance
- Cross-cluster networking setup

**Implementation Details**:
```go
type ClusterManager struct {
    k8sClients    map[string]kubernetes.Interface
    stateStore    *StateDatabase
    reconciler    *StateReconciler
    eventBus      *EventBus
}

type ClusterState struct {
    ID            string
    Name          string
    Nodes         []NodeState
    Workloads     []WorkloadState
    Resources     ResourceUsage
    Health        ClusterHealth
    LastReconcile time.Time
}
```

### 3.4 GPIO CRD Manager
**Purpose**: Kubernetes-native GPIO control via Custom Resources
**Technology**: Kubernetes Custom Resource Definitions, controller-runtime

**Key Features**:
- GPIO pin state management through CRDs
- PWM, SPI, I2C interface controls
- Hardware interrupt handling
- GPIO state persistence and recovery
- Multi-tenant GPIO resource isolation

### 3.5 Node Agent (DaemonSet)
**Purpose**: Host-level monitoring and hardware control
**Technology**: gRPC server, system monitoring, GPIO libraries

**Key Features**:
- Real-time system metrics collection
- GPIO pin direct hardware access
- Hardware health monitoring (temperature, voltage)
- Log aggregation and forwarding
- Secure communication with control plane

## 4. API Design

### 4.1 REST API Endpoints

#### Cluster Management
```
GET    /api/v1/clusters                    # List all clusters
POST   /api/v1/clusters                    # Create new cluster
GET    /api/v1/clusters/{id}               # Get cluster details
PUT    /api/v1/clusters/{id}               # Update cluster
DELETE /api/v1/clusters/{id}               # Delete cluster

GET    /api/v1/clusters/{id}/nodes         # List cluster nodes
POST   /api/v1/clusters/{id}/nodes         # Add node to cluster
DELETE /api/v1/clusters/{id}/nodes/{node}  # Remove node from cluster
```

#### Node Management
```
GET    /api/v1/nodes                       # List discovered nodes
GET    /api/v1/nodes/{id}                  # Get node details
PUT    /api/v1/nodes/{id}                  # Update node configuration
POST   /api/v1/nodes/{id}/provision        # Provision node
POST   /api/v1/nodes/{id}/deprovision      # Deprovision node
```

#### GPIO Resources
```
GET    /api/v1/gpio                        # List GPIO resources
POST   /api/v1/gpio                        # Create GPIO resource
GET    /api/v1/gpio/{id}                   # Get GPIO state
PUT    /api/v1/gpio/{id}                   # Update GPIO state
DELETE /api/v1/gpio/{id}                   # Delete GPIO resource
```

### 4.2 gRPC Services

#### Node Agent Communication
```protobuf
service NodeAgent {
    rpc GetSystemMetrics(Empty) returns (SystemMetrics);
    rpc ControlGPIO(GPIORequest) returns (GPIOResponse);
    rpc ExecuteCommand(CommandRequest) returns (CommandResponse);
    rpc StreamLogs(LogFilter) returns (stream LogEntry);
    rpc HealthCheck(Empty) returns (HealthStatus);
}

message SystemMetrics {
    double cpu_usage = 1;
    double memory_usage = 2;
    double temperature = 3;
    double disk_usage = 4;
    repeated NetworkInterface interfaces = 5;
}
```

#### Inter-Service Communication
```protobuf
service ControlPlane {
    rpc RegisterNode(NodeRegistration) returns (RegistrationResponse);
    rpc SyncClusterState(ClusterStateRequest) returns (ClusterStateResponse);
    rpc RequestProvisioning(ProvisioningRequest) returns (ProvisioningResponse);
    rpc ReportHealth(HealthReport) returns (Empty);
}
```

### 4.3 WebSocket Events

#### Real-time Updates
```json
{
    "type": "cluster.node.added",
    "timestamp": "2025-01-15T10:30:00Z",
    "cluster_id": "cluster-1",
    "data": {
        "node_id": "pi-worker-03",
        "ip_address": "192.168.1.103",
        "status": "provisioning"
    }
}

{
    "type": "gpio.state.changed",
    "timestamp": "2025-01-15T10:30:15Z",
    "resource": "led-controller",
    "data": {
        "pin": 18,
        "state": "high",
        "pwm_duty": 75
    }
}
```

## 5. Database Schema Design

### 5.1 SQLite Schema (Control Plane State)

```sql
-- Clusters table
CREATE TABLE clusters (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    k3s_version TEXT NOT NULL,
    config JSON NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Nodes table
CREATE TABLE nodes (
    id TEXT PRIMARY KEY,
    cluster_id TEXT REFERENCES clusters(id),
    hostname TEXT NOT NULL,
    ip_address TEXT NOT NULL,
    mac_address TEXT NOT NULL,
    architecture TEXT NOT NULL,
    role TEXT NOT NULL, -- control-plane, worker
    status TEXT NOT NULL, -- discovered, provisioning, ready, error
    capabilities JSON NOT NULL,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- GPIO Resources table
CREATE TABLE gpio_resources (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL REFERENCES nodes(id),
    name TEXT NOT NULL,
    namespace TEXT NOT NULL,
    pin_number INTEGER NOT NULL,
    pin_type TEXT NOT NULL, -- digital, pwm, spi, i2c
    state JSON NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(node_id, pin_number)
);

-- Events table
CREATE TABLE events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    data JSON NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Configuration table
CREATE TABLE configuration (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX idx_nodes_cluster ON nodes(cluster_id);
CREATE INDEX idx_nodes_status ON nodes(status);
CREATE INDEX idx_gpio_node ON gpio_resources(node_id);
CREATE INDEX idx_events_type ON events(type);
CREATE INDEX idx_events_timestamp ON events(timestamp);
```

## 6. GPIO Custom Resource Definitions

### 6.1 GPIO Pin CRD
```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: gpiopins.hardware.pi-controller.io
spec:
  group: hardware.pi-controller.io
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              nodeSelector:
                type: object
                additionalProperties:
                  type: string
              pin:
                type: integer
                minimum: 1
                maximum: 40
              mode:
                type: string
                enum: ["input", "output", "pwm"]
              initialState:
                type: string
                enum: ["low", "high"]
              pullResistor:
                type: string
                enum: ["none", "up", "down"]
          status:
            type: object
            properties:
              state:
                type: string
              assignedNode:
                type: string
              lastUpdated:
                type: string
                format: date-time
  scope: Namespaced
  names:
    plural: gpiopins
    singular: gpiopin
    kind: GPIOPin
```

### 6.2 PWM Controller CRD
```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: pwmcontrollers.hardware.pi-controller.io
spec:
  group: hardware.pi-controller.io
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              nodeSelector:
                type: object
              pin:
                type: integer
              frequency:
                type: integer
                minimum: 1
                maximum: 100000
              dutyCycle:
                type: integer
                minimum: 0
                maximum: 100
          status:
            type: object
            properties:
              active:
                type: boolean
              currentDutyCycle:
                type: integer
              assignedNode:
                type: string
  scope: Namespaced
  names:
    plural: pwmcontrollers
    singular: pwmcontroller
    kind: PWMController
```

### 6.3 I2C Device CRD
```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: i2cdevices.hardware.pi-controller.io
spec:
  group: hardware.pi-controller.io
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              nodeSelector:
                type: object
              bus:
                type: integer
                minimum: 0
                maximum: 1
              address:
                type: string
                pattern: "^0x[0-9A-Fa-f]{2}$"
              deviceType:
                type: string
          status:
            type: object
            properties:
              connected:
                type: boolean
              lastResponse:
                type: string
              assignedNode:
                type: string
  scope: Namespaced
  names:
    plural: i2cdevices
    singular: i2cdevice
    kind: I2CDevice
```

## 7. Deployment Architecture

### 7.1 Distribution Strategy

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Pi Master 1   │    │   Pi Master 2   │    │   Pi Master 3   │
│                 │    │                 │    │                 │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │Control Plane│ │    │ │Control Plane│ │    │ │Control Plane│ │
│ │   (Active)  │ │    │ │ (Standby)   │ │    │ │ (Standby)   │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │ Node Agent  │ │    │ │ Node Agent  │ │    │ │ Node Agent  │ │
│ │(DaemonSet)  │ │    │ │(DaemonSet)  │ │    │ │(DaemonSet)  │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │     K3s     │ │    │ │     K3s     │ │    │ │     K3s     │ │
│ │ Server Node │ │    │ │ Server Node │ │    │ │ Server Node │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────┬───────────┴───────────┬───────────┘
                     │                       │
┌─────────────────┐  │  ┌─────────────────┐  │  ┌─────────────────┐
│  Pi Worker 1    │  │  │  Pi Worker 2    │  │  │  Pi Worker N    │
│                 │  │  │                 │  │  │                 │
│ ┌─────────────┐ │  │  │ ┌─────────────┐ │  │  │ ┌─────────────┐ │
│ │ Node Agent  │ │  │  │ │ Node Agent  │ │  │  │ │ Node Agent  │ │
│ │(DaemonSet)  │ │  │  │ │(DaemonSet)  │ │  │  │ │(DaemonSet)  │ │
│ └─────────────┘ │  │  │ └─────────────┘ │  │  │ └─────────────┘ │
│ ┌─────────────┐ │  │  │ ┌─────────────┐ │  │  │ ┌─────────────┐ │
│ │     K3s     │ │  │  │ │     K3s     │ │  │  │ │     K3s     │ │
│ │ Agent Node  │ │  │  │ │ Agent Node  │ │  │  │ │ Agent Node  │ │
│ └─────────────┘ │  │  │ └─────────────┘ │  │  │ └─────────────┘ │
└─────────────────┘  │  └─────────────────┘  │  └─────────────────┘
                     │                       │
              Load Balancer / VIP
                 (keepalived)
```

### 7.2 Component Distribution

**Control Plane Nodes (Masters)**:
- Pi-Controller binary (single process)
- Embedded SQLite database (with replication)
- Web UI (embedded static files)
- MCP server
- Node Agent DaemonSet pod
- K3s server with embedded etcd

**Worker Nodes**:
- Node Agent DaemonSet pod only
- K3s agent process
- Local storage for workloads

### 7.3 Installation Methods

#### Single Command Installation
```bash
# Bootstrap first control plane node
curl -sfL https://get.pi-controller.io/install.sh | sh -s - \
  --cluster-init \
  --node-role=server \
  --web-ui-port=8080

# Join additional nodes
curl -sfL https://get.pi-controller.io/install.sh | sh -s - \
  --server https://pi-master-1:8080 \
  --token=<join-token> \
  --node-role=worker
```

#### Docker Compose (Development)
```yaml
version: '3.8'
services:
  pi-controller:
    image: pi-controller:latest
    ports:
      - "8080:8080"
      - "6443:6443"
    volumes:
      - ./data:/data
      - /var/run/docker.sock:/var/run/docker.sock
    privileged: true
    environment:
      - CLUSTER_INIT=true
      - DATA_DIR=/data
```

## 8. Security Model

### 8.1 Authentication & Authorization

#### Multi-Tier Security Architecture
```
┌─────────────────────────────────────────────────────────────────┐
│                      External Clients                          │
├─────────────────────────────────────────────────────────────────┤
│ Web UI │ CLI │ MCP Server │ External APIs                       │
└───┬─────────┬──────────┬─────────────────────────────────────┘
    │         │          │
    │ HTTPS   │ mTLS     │ JWT/OIDC
    │         │          │
┌───▼─────────▼──────────▼─────────────────────────────────────────┐
│                  Pi-Controller Gateway                          │
├─────────────────────────────────────────────────────────────────┤
│ • Certificate-based authentication                              │
│ • RBAC policy enforcement                                       │
│ • Rate limiting and DDoS protection                            │
│ • Audit logging                                                │
└─────────────────────┬───────────────────────────────────────────┘
                      │ mTLS
┌─────────────────────▼───────────────────────────────────────────┐
│                Internal Services                               │
├─────────────────────────────────────────────────────────────────┤
│ Discovery │ Provisioner │ Cluster Manager │ GPIO CRD Manager   │
└─────────────────────┬───────────────────────────────────────────┘
                      │ gRPC with mTLS
┌─────────────────────▼───────────────────────────────────────────┐
│                   Node Agents                                  │
├─────────────────────────────────────────────────────────────────┤
│ • Certificate-based node identity                              │
│ • Hardware security module integration                         │
│ • Secure GPIO access controls                                  │
└─────────────────────────────────────────────────────────────────┘
```

#### RBAC Policies
```yaml
# Cluster Administrator Role
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: pi-controller:cluster-admin
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]
- apiGroups: ["hardware.pi-controller.io"]
  resources: ["*"]
  verbs: ["*"]

# GPIO Operator Role
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: pi-controller:gpio-operator
rules:
- apiGroups: ["hardware.pi-controller.io"]
  resources: ["gpiopins", "pwmcontrollers", "i2cdevices"]
  verbs: ["get", "list", "create", "update", "patch", "delete"]

# Read-Only Monitoring Role
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: pi-controller:monitor
rules:
- apiGroups: [""]
  resources: ["nodes", "pods", "services"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["hardware.pi-controller.io"]
  resources: ["*"]
  verbs: ["get", "list", "watch"]
```

### 8.2 Communication Security

#### Certificate Management
- **Root CA**: Self-signed cluster root certificate authority
- **Node Certificates**: Unique client certificates for each Pi node
- **Service Certificates**: TLS certificates for all internal services
- **Automatic Rotation**: 30-day certificate lifecycle with auto-renewal
- **Hardware Security**: TPM/secure enclave integration where available

#### Network Security
```go
type SecurityConfig struct {
    TLS struct {
        CACertPath     string
        CertPath       string
        KeyPath        string
        MinVersion     uint16 // TLS 1.3
        CipherSuites   []uint16
    }
    
    Authentication struct {
        Method         string // "certificate", "jwt", "oidc"
        JWTSecret      string
        OIDCIssuer     string
        TokenExpiry    time.Duration
    }
    
    Authorization struct {
        EnableRBAC     bool
        PolicyFile     string
        AuditLog       string
    }
    
    Network struct {
        AllowedCIDRs   []string
        RateLimits     map[string]int
        FirewallRules  []FirewallRule
    }
}
```

### 8.3 GPIO Security Model

#### Hardware Access Control
- **Pin Reservation**: Exclusive GPIO pin access via Kubernetes leases
- **Privilege Escalation**: Controlled sudo access for hardware operations
- **Hardware Abstraction**: No direct /dev/mem access from containers
- **Safety Limits**: Voltage/current monitoring with automatic shutdown

#### Resource Isolation
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: gpio-tenant-1
  labels:
    pi-controller.io/gpio-policy: "restricted"
---
apiVersion: hardware.pi-controller.io/v1
kind: GPIOQuota
metadata:
  name: tenant-1-quota
  namespace: gpio-tenant-1
spec:
  hard:
    gpiopins: "10"
    pwmcontrollers: "2"
    i2cdevices: "5"
    power-budget: "500mW"
```

## 9. Development Project Structure

### 9.1 Repository Layout
```
pi-controller/
├── cmd/                           # Application entry points
│   ├── pi-controller/            # Main control plane binary
│   ├── pi-agent/                 # Node agent binary
│   └── pi-cli/                   # CLI tool
├── pkg/                          # Reusable packages
│   ├── api/                      # API definitions and handlers
│   │   ├── rest/                 # REST API handlers
│   │   ├── grpc/                 # gRPC service implementations
│   │   └── websocket/            # WebSocket handlers
│   ├── discovery/                # Node discovery service
│   ├── provisioner/              # Cluster provisioning engine
│   ├── cluster/                  # Cluster management
│   ├── gpio/                     # GPIO CRD controllers
│   ├── agent/                    # Node agent components
│   ├── database/                 # Database abstraction layer
│   ├── security/                 # Authentication/authorization
│   ├── config/                   # Configuration management
│   └── util/                     # Shared utilities
├── internal/                     # Private packages
│   ├── server/                   # HTTP/gRPC server setup
│   ├── storage/                  # SQLite database layer
│   ├── certificates/             # Certificate management
│   └── hardware/                 # Hardware abstraction
├── web/                          # Frontend application
│   ├── src/                      # React TypeScript source
│   ├── public/                   # Static assets
│   └── dist/                     # Built assets (embedded)
├── deploy/                       # Deployment manifests
│   ├── k8s/                      # Kubernetes manifests
│   ├── docker/                   # Docker configurations
│   └── scripts/                  # Installation scripts
├── docs/                         # Documentation
├── test/                         # Test files
│   ├── integration/              # Integration tests
│   ├── e2e/                      # End-to-end tests
│   └── fixtures/                 # Test data
├── proto/                        # Protocol buffer definitions
├── config/                       # Configuration examples
└── tools/                        # Development tools
```

### 9.2 Module Breakdown

#### Core Modules

**cmd/pi-controller (Main Binary)**
```go
package main

import (
    "github.com/pi-controller/pkg/server"
    "github.com/pi-controller/pkg/config"
    "github.com/pi-controller/internal/storage"
)

func main() {
    cfg := config.Load()
    db := storage.NewSQLite(cfg.DatabasePath)
    srv := server.New(cfg, db)
    srv.Start()
}
```

**pkg/discovery (Node Discovery)**
```go
package discovery

type Service interface {
    Start(ctx context.Context) error
    Stop() error
    DiscoveredNodes() <-chan Node
    RegisterNode(node Node) error
    UnregisterNode(nodeID string) error
}

type Node struct {
    ID           string
    Hostname     string
    IPAddress    net.IP
    MACAddress   net.HardwareAddr
    Architecture string
    Capabilities NodeCapabilities
    LastSeen     time.Time
}
```

**pkg/provisioner (Cluster Provisioning)**
```go
package provisioner

type Engine interface {
    CreateCluster(ctx context.Context, plan ClusterPlan) error
    AddNode(ctx context.Context, clusterID string, node NodeSpec) error
    RemoveNode(ctx context.Context, clusterID, nodeID string) error
    GetClusterStatus(clusterID string) (ClusterStatus, error)
}

type ClusterPlan struct {
    ID            string
    Name          string
    K3sVersion    string
    ControlPlanes []NodeSpec
    Workers       []NodeSpec
    Config        K3sConfig
}
```

**pkg/gpio (GPIO Controllers)**
```go
package gpio

type Controller interface {
    Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error)
}

type PinController struct {
    client.Client
    scheme *runtime.Scheme
    agent  AgentClient
}
```

### 9.3 Build System

#### Multi-Architecture Builds
```makefile
# Makefile
GOARCH ?= arm64
GOOS ?= linux
VERSION ?= $(shell git describe --tags --dirty)

.PHONY: build-all
build-all: build-controller build-agent build-cli

.PHONY: build-controller
build-controller:
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-ldflags "-X main.version=$(VERSION)" \
		-o bin/pi-controller-$(GOOS)-$(GOARCH) \
		./cmd/pi-controller

.PHONY: build-agent
build-agent:
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-ldflags "-X main.version=$(VERSION)" \
		-o bin/pi-agent-$(GOOS)-$(GOARCH) \
		./cmd/pi-agent

.PHONY: docker-build
docker-build:
	docker buildx build --platform linux/arm64,linux/amd64 \
		-t pi-controller:$(VERSION) \
		--push .
```

#### Docker Multi-Stage Build
```dockerfile
# Dockerfile
FROM --platform=$BUILDPLATFORM node:18-alpine AS web-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS go-builder
RUN apk add --no-cache git gcc musl-dev sqlite-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /app/web/dist ./web/dist
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -o pi-controller ./cmd/pi-controller

FROM alpine:3.18
RUN apk add --no-cache ca-certificates sqlite
COPY --from=go-builder /app/pi-controller /usr/local/bin/
EXPOSE 8080 6443
ENTRYPOINT ["/usr/local/bin/pi-controller"]
```

## 10. Performance and Scalability Considerations

### 10.1 Pi Hardware Optimization

#### Resource-Aware Scheduling
```go
type NodeCapacity struct {
    CPU       resource.Quantity  // ARM cores available
    Memory    resource.Quantity  // RAM capacity
    Storage   resource.Quantity  // SD card/USB storage
    GPIOPins  int               // Available GPIO pins
    I2CBuses  int               // Available I2C buses
    SPIBuses  int               // Available SPI buses
    PowerBudget resource.Quantity // Available power (mW)
}

type ResourceOptimizer struct {
    nodeCapacities map[string]NodeCapacity
    workloadProfiles map[string]WorkloadProfile
    scheduler *Scheduler
}
```

#### Memory Management
- **SQLite WAL Mode**: Optimized for concurrent reads
- **Connection Pooling**: Limited database connections per Pi
- **Memory Caching**: LRU cache for frequently accessed data
- **Garbage Collection Tuning**: Lower GC pressure for ARM processors

### 10.2 Network Optimization

#### Efficient Communication Patterns
- **gRPC Streaming**: Reduce connection overhead for real-time data
- **Message Batching**: Group GPIO operations to reduce network calls
- **Compression**: gRPC compression for large data transfers
- **Connection Multiplexing**: HTTP/2 for web UI and API calls

#### Network Resilience
```go
type NetworkManager struct {
    connectionPool *grpc.ClientPool
    retryPolicy   *ExponentialBackoff
    circuitBreaker *CircuitBreaker
    healthChecker *HealthChecker
}

type ConnectionPolicy struct {
    MaxRetries        int
    BackoffMultiplier float64
    MaxBackoff        time.Duration
    HealthCheckInterval time.Duration
    CircuitBreakerThreshold int
}
```

### 10.3 Scalability Architecture

#### Horizontal Scaling (1-100+ Nodes)
- **Distributed Control Plane**: Multiple control plane nodes with leader election
- **Sharded Database**: SQLite WAL with backup replication
- **Load Balancing**: HAProxy/keepalived for API endpoint distribution
- **Event Batching**: Aggregate events before processing

#### Vertical Scaling (Per-Node Optimization)
- **Resource Quotas**: Prevent resource exhaustion
- **Priority Classes**: Critical system workloads get priority
- **Node Affinity**: Pin system workloads to appropriate hardware
- **Local Storage**: Prefer local storage for performance-critical workloads

## Conclusion

This architecture provides a robust, scalable foundation for managing Raspberry Pi Kubernetes clusters with GPIO-as-a-Service capabilities. The design balances simplicity of deployment with enterprise-grade features, making it suitable for both homelab enthusiasts and small-scale IoT deployments.

Key architectural strengths:
- **Single binary deployment** reduces complexity
- **Kubernetes-native GPIO control** provides familiar APIs
- **Multi-protocol communication** ensures reliability
- **Hardware-aware scheduling** optimizes Pi resource usage
- **Comprehensive security model** protects infrastructure and hardware

The modular design allows for incremental development and testing, while the detailed specifications enable multiple development teams to work in parallel on different components.