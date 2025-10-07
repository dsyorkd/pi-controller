---
layout: default
title: MCP Server Integration
parent: Advanced Topics
nav_order: 2
---

# Pi-Controller MCP Server Design
{: .no_toc }

Model Context Protocol integration for AI-assisted operations
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Executive Summary

This document outlines the design for `pi-controller-mcp`, a TypeScript-based Model Context Protocol (MCP) server that provides AI assistants with comprehensive cluster management capabilities for Raspberry Pi K3s clusters.

## 1. Repository Structure

**New Repository:** `pi-controller-mcp`

```
pi-controller-mcp/
├── src/
│   ├── index.ts                    # MCP server entry point
│   ├── config.ts                   # Configuration management
│   ├── client/
│   │   ├── pi-controller-client.ts # REST API client
│   │   └── auth.ts                 # Authentication handling
│   ├── tools/
│   │   ├── cluster.ts              # Cluster management tools
│   │   ├── node.ts                 # Node management tools
│   │   ├── gpio.ts                 # GPIO control tools
│   │   ├── deployment.ts           # K8s deployment tools
│   │   ├── ca.ts                   # Certificate authority tools
│   │   └── provisioning.ts         # Provisioning tools
│   ├── resources/
│   │   ├── cluster-status.ts       # Cluster state resources
│   │   ├── node-info.ts            # Node information resources
│   │   ├── gpio-state.ts           # GPIO state resources
│   │   └── metrics.ts              # System metrics resources
│   ├── types/
│   │   └── pi-controller.ts        # Type definitions
│   └── utils/
│       ├── validation.ts           # Input validation
│       └── error-handler.ts        # Error handling
├── package.json
├── tsconfig.json
├── README.md
└── .env.example
```

## 2. MCP Tools Design

### 2.1 Cluster Management Tools

#### `create_cluster`
Create a new cluster definition.

```typescript
{
  name: "create_cluster",
  description: "Create a new K3s cluster definition",
  inputSchema: {
    type: "object",
    properties: {
      name: { type: "string", description: "Cluster name" },
      description: { type: "string", description: "Cluster description" },
      k3s_version: { type: "string", description: "K3s version (e.g., 'v1.28.5+k3s1')" }
    },
    required: ["name"]
  }
}
```

**API Mapping:** `POST /api/v1/clusters`

#### `list_clusters`
List all clusters with optional filtering.

```typescript
{
  name: "list_clusters",
  description: "List all K3s clusters",
  inputSchema: {
    type: "object",
    properties: {
      limit: { type: "number", default: 50 },
      offset: { type: "number", default: 0 }
    }
  }
}
```

**API Mapping:** `GET /api/v1/clusters`

#### `get_cluster_status`
Get detailed cluster status including nodes and health.

```typescript
{
  name: "get_cluster_status",
  description: "Get detailed status of a cluster including nodes, workloads, and health",
  inputSchema: {
    type: "object",
    properties: {
      cluster_id: { type: "number", description: "Cluster ID" }
    },
    required: ["cluster_id"]
  }
}
```

**API Mapping:** `GET /api/v1/clusters/{id}/status`

#### `provision_cluster`
Provision a K3s cluster on physical nodes.

```typescript
{
  name: "provision_cluster",
  description: "Provision a K3s cluster on Raspberry Pi nodes (async operation)",
  inputSchema: {
    type: "object",
    properties: {
      cluster_id: { type: "number" },
      master_node_id: { type: "number", description: "Master node ID" },
      worker_node_ids: { type: "array", items: { type: "number" } },
      ssh_config: {
        type: "object",
        properties: {
          username: { type: "string", default: "pi" },
          port: { type: "number", default: 22 },
          private_key_path: { type: "string" },
          password: { type: "string" },
          use_agent: { type: "boolean", default: false }
        }
      }
    },
    required: ["cluster_id", "master_node_id", "ssh_config"]
  }
}
```

**API Mapping:** `POST /api/v1/clusters/{id}/provision`

#### `scale_cluster`
Scale cluster to specified node count.

```typescript
{
  name: "scale_cluster",
  description: "Scale a cluster by adding or removing worker nodes",
  inputSchema: {
    type: "object",
    properties: {
      cluster_id: { type: "number" },
      node_count: { type: "number", minimum: 1 },
      ssh_config: { type: "object" }
    },
    required: ["cluster_id", "node_count"]
  }
}
```

**API Mapping:** `POST /api/v1/clusters/{id}/scale`

#### `delete_cluster`
Delete a cluster definition.

```typescript
{
  name: "delete_cluster",
  description: "Delete a cluster (must be deprovisioned first)",
  inputSchema: {
    type: "object",
    properties: {
      cluster_id: { type: "number" }
    },
    required: ["cluster_id"]
  }
}
```

**API Mapping:** `DELETE /api/v1/clusters/{id}`

### 2.2 Node Management Tools

#### `discover_nodes`
List all discovered nodes with filtering.

```typescript
{
  name: "discover_nodes",
  description: "List all discovered Raspberry Pi nodes",
  inputSchema: {
    type: "object",
    properties: {
      cluster_id: { type: "number", description: "Filter by cluster" },
      status: { type: "string", enum: ["discovered", "provisioning", "ready", "error"] },
      role: { type: "string", enum: ["master", "worker"] },
      include_gpio: { type: "boolean", default: false }
    }
  }
}
```

**API Mapping:** `GET /api/v1/nodes`

#### `get_node_info`
Get detailed node information.

```typescript
{
  name: "get_node_info",
  description: "Get detailed information about a specific node",
  inputSchema: {
    type: "object",
    properties: {
      node_id: { type: "number" },
      include_gpio: { type: "boolean", default: false }
    },
    required: ["node_id"]
  }
}
```

**API Mapping:** `GET /api/v1/nodes/{id}`

#### `register_node`
Manually register a node.

```typescript
{
  name: "register_node",
  description: "Manually register a Raspberry Pi node",
  inputSchema: {
    type: "object",
    properties: {
      name: { type: "string" },
      ip_address: { type: "string" },
      architecture: { type: "string", enum: ["arm64", "armv7"] },
      cluster_id: { type: "number" },
      role: { type: "string", enum: ["master", "worker"] }
    },
    required: ["name", "ip_address", "architecture"]
  }
}
```

**API Mapping:** `POST /api/v1/nodes`

#### `provision_node`
Provision K3s on a single node.

```typescript
{
  name: "provision_node",
  description: "Provision K3s on a single node",
  inputSchema: {
    type: "object",
    properties: {
      node_id: { type: "number" },
      role: { type: "string", enum: ["master", "worker"] },
      join_token: { type: "string", description: "For worker nodes" },
      ssh_config: { type: "object" }
    },
    required: ["node_id", "role"]
  }
}
```

**API Mapping:** `POST /api/v1/nodes/{id}/provision`

#### `deprovision_node`
Remove K3s from a node.

```typescript
{
  name: "deprovision_node",
  description: "Remove K3s from a node and return to discovered state",
  inputSchema: {
    type: "object",
    properties: {
      node_id: { type: "number" },
      ssh_config: { type: "object" }
    },
    required: ["node_id"]
  }
}
```

**API Mapping:** `POST /api/v1/nodes/{id}/deprovision`

### 2.3 GPIO Control Tools

#### `list_gpio_devices`
List all GPIO devices.

```typescript
{
  name: "list_gpio_devices",
  description: "List all GPIO devices across all nodes",
  inputSchema: {
    type: "object",
    properties: {
      node_id: { type: "number", description: "Filter by node" },
      limit: { type: "number", default: 50 },
      offset: { type: "number", default: 0 }
    }
  }
}
```

**API Mapping:** `GET /api/v1/gpio`

#### `create_gpio_device`
Register a new GPIO device.

```typescript
{
  name: "create_gpio_device",
  description: "Register a GPIO device for control",
  inputSchema: {
    type: "object",
    properties: {
      node_id: { type: "number" },
      pin_number: { type: "number", minimum: 1, maximum: 40 },
      name: { type: "string" },
      mode: { type: "string", enum: ["input", "output", "pwm"] },
      description: { type: "string" }
    },
    required: ["node_id", "pin_number", "name", "mode"]
  }
}
```

**API Mapping:** `POST /api/v1/gpio`

#### `read_gpio_pin`
Read current GPIO pin value.

```typescript
{
  name: "read_gpio_pin",
  description: "Read the current value of a GPIO pin",
  inputSchema: {
    type: "object",
    properties: {
      gpio_id: { type: "number" }
    },
    required: ["gpio_id"]
  }
}
```

**API Mapping:** `POST /api/v1/gpio/{id}/read`

#### `write_gpio_pin`
Write value to GPIO pin.

```typescript
{
  name: "write_gpio_pin",
  description: "Write a value to a GPIO pin (HIGH=1, LOW=0)",
  inputSchema: {
    type: "object",
    properties: {
      gpio_id: { type: "number" },
      value: { type: "number", enum: [0, 1] }
    },
    required: ["gpio_id", "value"]
  }
}
```

**API Mapping:** `POST /api/v1/gpio/{id}/write`

#### `reserve_gpio_pin`
Reserve GPIO pin for exclusive use.

```typescript
{
  name: "reserve_gpio_pin",
  description: "Reserve a GPIO pin for exclusive use by a client",
  inputSchema: {
    type: "object",
    properties: {
      gpio_id: { type: "number" },
      client_id: { type: "string" },
      duration_seconds: { type: "number", default: 300 }
    },
    required: ["gpio_id", "client_id"]
  }
}
```

**API Mapping:** `POST /api/v1/gpio/{id}/reserve`

#### `release_gpio_pin`
Release GPIO pin reservation.

```typescript
{
  name: "release_gpio_pin",
  description: "Release a GPIO pin reservation",
  inputSchema: {
    type: "object",
    properties: {
      gpio_id: { type: "number" },
      client_id: { type: "string" }
    },
    required: ["gpio_id", "client_id"]
  }
}
```

**API Mapping:** `POST /api/v1/gpio/{id}/release`

#### `get_gpio_readings`
Get historical GPIO readings.

```typescript
{
  name: "get_gpio_readings",
  description: "Get historical readings for a GPIO device",
  inputSchema: {
    type: "object",
    properties: {
      gpio_id: { type: "number" },
      limit: { type: "number", default: 100 },
      offset: { type: "number", default: 0 }
    },
    required: ["gpio_id"]
  }
}
```

**API Mapping:** `GET /api/v1/gpio/{id}/readings`

### 2.4 Deployment Tools

#### `deploy_pod`
Deploy a Kubernetes pod.

```typescript
{
  name: "deploy_pod",
  description: "Deploy a Kubernetes pod to a cluster",
  inputSchema: {
    type: "object",
    properties: {
      cluster_id: { type: "number" },
      pod_spec: {
        type: "object",
        description: "Kubernetes pod specification"
      }
    },
    required: ["cluster_id", "pod_spec"]
  }
}
```

**API Mapping:** `POST /api/v1/deployments/pods`

#### `get_pod`
Get pod information.

```typescript
{
  name: "get_pod",
  description: "Get information about a specific pod",
  inputSchema: {
    type: "object",
    properties: {
      cluster_id: { type: "number" },
      pod_name: { type: "string" }
    },
    required: ["cluster_id", "pod_name"]
  }
}
```

**API Mapping:** `GET /api/v1/deployments/clusters/{cluster_id}/pods/{name}`

#### `delete_pod`
Delete a pod.

```typescript
{
  name: "delete_pod",
  description: "Delete a pod from a cluster",
  inputSchema: {
    type: "object",
    properties: {
      cluster_id: { type: "number" },
      pod_name: { type: "string" }
    },
    required: ["cluster_id", "pod_name"]
  }
}
```

**API Mapping:** `DELETE /api/v1/deployments/clusters/{cluster_id}/pods/{name}`

### 2.5 Certificate Authority Tools

#### `initialize_ca`
Initialize the Certificate Authority.

```typescript
{
  name: "initialize_ca",
  description: "Initialize the Certificate Authority for the cluster",
  inputSchema: {
    type: "object",
    properties: {
      organization: { type: "string" },
      country: { type: "string" },
      validity_days: { type: "number", default: 3650 }
    },
    required: ["organization", "country"]
  }
}
```

**API Mapping:** `POST /api/v1/ca/initialize`

#### `issue_certificate`
Issue a new certificate.

```typescript
{
  name: "issue_certificate",
  description: "Issue a new certificate for a node or service",
  inputSchema: {
    type: "object",
    properties: {
      common_name: { type: "string" },
      certificate_type: { type: "string", enum: ["node", "service", "user"] },
      validity_days: { type: "number", default: 365 },
      dns_names: { type: "array", items: { type: "string" } },
      ip_addresses: { type: "array", items: { type: "string" } }
    },
    required: ["common_name", "certificate_type"]
  }
}
```

**API Mapping:** `POST /api/v1/ca/certificates`

#### `list_certificates`
List all certificates.

```typescript
{
  name: "list_certificates",
  description: "List all issued certificates",
  inputSchema: {
    type: "object",
    properties: {
      status: { type: "string", enum: ["active", "expired", "revoked"] }
    }
  }
}
```

**API Mapping:** `GET /api/v1/ca/certificates`

#### `revoke_certificate`
Revoke a certificate.

```typescript
{
  name: "revoke_certificate",
  description: "Revoke a certificate",
  inputSchema: {
    type: "object",
    properties: {
      certificate_id: { type: "number" },
      reason: { type: "string" }
    },
    required: ["certificate_id"]
  }
}
```

**API Mapping:** `POST /api/v1/ca/certificates/{id}/revoke`

## 3. MCP Resources Design

Resources provide read-only context about cluster state.

### 3.1 Cluster Resources

#### `cluster://{cluster_id}/status`
Real-time cluster status and health.

```typescript
{
  uri: "cluster://{cluster_id}/status",
  name: "Cluster Status",
  description: "Current cluster state, node count, and health metrics",
  mimeType: "application/json"
}
```

**Response:**
```json
{
  "cluster_id": 1,
  "name": "homelab",
  "status": "ready",
  "nodes": {
    "total": 5,
    "ready": 5,
    "masters": 1,
    "workers": 4
  },
  "health": {
    "status": "healthy",
    "api_server": "available",
    "etcd": "healthy"
  },
  "version": "v1.28.5+k3s1"
}
```

#### `cluster://{cluster_id}/nodes`
List of nodes in cluster with health.

```typescript
{
  uri: "cluster://{cluster_id}/nodes",
  name: "Cluster Nodes",
  description: "Detailed node information for cluster",
  mimeType: "application/json"
}
```

### 3.2 Node Resources

#### `node://{node_id}/info`
Detailed node information.

```typescript
{
  uri: "node://{node_id}/info",
  name: "Node Information",
  description: "Hardware specs, OS info, and capabilities",
  mimeType: "application/json"
}
```

**Response:**
```json
{
  "id": 1,
  "name": "pi-master-1",
  "ip_address": "192.168.1.10",
  "architecture": "arm64",
  "role": "master",
  "status": "ready",
  "hardware": {
    "model": "Raspberry Pi 4 Model B",
    "cpu_cores": 4,
    "memory_mb": 8192,
    "gpio_pins": 40
  }
}
```

#### `node://{node_id}/metrics`
Real-time node metrics.

```typescript
{
  uri: "node://{node_id}/metrics",
  name: "Node Metrics",
  description: "CPU, memory, temperature, and network metrics",
  mimeType: "application/json"
}
```

**Response:**
```json
{
  "cpu_usage_percent": 35.2,
  "memory_usage_percent": 42.8,
  "temperature_celsius": 48.5,
  "disk_usage_percent": 67.3,
  "network": {
    "rx_bytes": 1234567890,
    "tx_bytes": 987654321
  },
  "timestamp": "2025-01-15T10:30:00Z"
}
```

#### `node://{node_id}/gpio`
GPIO device list and states.

```typescript
{
  uri: "node://{node_id}/gpio",
  name: "GPIO Devices",
  description: "List of GPIO devices on this node with current states",
  mimeType: "application/json"
}
```

### 3.3 GPIO Resources

#### `gpio://{gpio_id}/state`
Current GPIO device state.

```typescript
{
  uri: "gpio://{gpio_id}/state",
  name: "GPIO State",
  description: "Current value and configuration of GPIO device",
  mimeType: "application/json"
}
```

**Response:**
```json
{
  "id": 15,
  "node_id": 1,
  "pin_number": 18,
  "name": "led-strip",
  "mode": "output",
  "current_value": 1,
  "last_updated": "2025-01-15T10:30:00Z",
  "reserved": false
}
```

### 3.4 System Resources

#### `system://health`
Overall system health.

```typescript
{
  uri: "system://health",
  name: "System Health",
  description: "Overall pi-controller system health and API status",
  mimeType: "application/json"
}
```

**Response:**
```json
{
  "status": "healthy",
  "database": "connected",
  "api_version": "v1.0.0",
  "uptime_seconds": 86400,
  "clusters": 3,
  "nodes": 15,
  "gpio_devices": 42
}
```

## 4. Configuration

### 4.1 User Configuration (`.mcp.json`)

```json
{
  "mcpServers": {
    "pi-controller": {
      "command": "npx",
      "args": ["-y", "pi-controller-mcp"],
      "env": {
        "PI_CONTROLLER_URL": "https://pi-controller.local:8080",
        "PI_CONTROLLER_API_KEY": "your-api-key-here",
        "PI_CONTROLLER_USERNAME": "admin",
        "PI_CONTROLLER_PASSWORD": "secure-password",
        "PI_CONTROLLER_TLS_VERIFY": "true",
        "PI_CONTROLLER_TLS_CA_CERT": "/path/to/ca.crt"
      }
    }
  }
}
```

### 4.2 Environment Variables

| Variable | Required | Description | Default |
|----------|----------|-------------|---------|
| `PI_CONTROLLER_URL` | Yes | Pi-controller API base URL | - |
| `PI_CONTROLLER_API_KEY` | No* | API key for authentication | - |
| `PI_CONTROLLER_USERNAME` | No* | Username for auth | - |
| `PI_CONTROLLER_PASSWORD` | No* | Password for auth | - |
| `PI_CONTROLLER_TLS_VERIFY` | No | Verify TLS certificates | `true` |
| `PI_CONTROLLER_TLS_CA_CERT` | No | Path to CA certificate | - |
| `PI_CONTROLLER_TIMEOUT` | No | Request timeout (ms) | `30000` |

*Either API key or username/password must be provided

## 5. Authentication & Security

### 5.1 Authentication Flow

1. **Initial Connection:**
   - Load credentials from environment variables
   - Attempt login with username/password OR API key
   - Store JWT token in memory

2. **Token Management:**
   - Include JWT in `Authorization: Bearer {token}` header
   - Monitor token expiration
   - Automatically refresh token when needed
   - Re-authenticate if refresh fails

3. **Security Features:**
   - TLS certificate verification (configurable)
   - Support for custom CA certificates
   - Secure token storage (memory only, never disk)
   - Automatic session cleanup on exit

### 5.2 RBAC Integration

Tools respect pi-controller's RBAC system:

- **viewer**: Can list and get resources
- **operator**: Can create/update resources, control GPIO
- **admin**: Can provision/deprovision, manage CA

MCP tools will return appropriate errors if user lacks permissions.

## 6. Error Handling

### 6.1 Error Categories

```typescript
enum ErrorType {
  // Network errors
  CONNECTION_FAILED = "connection_failed",
  TIMEOUT = "timeout",

  // Authentication errors
  UNAUTHORIZED = "unauthorized",
  FORBIDDEN = "forbidden",
  TOKEN_EXPIRED = "token_expired",

  // Validation errors
  INVALID_INPUT = "invalid_input",
  RESOURCE_NOT_FOUND = "not_found",

  // Operation errors
  OPERATION_FAILED = "operation_failed",
  CONFLICT = "conflict",

  // Pi-Controller specific
  PROVISIONING_FAILED = "provisioning_failed",
  GPIO_OPERATION_FAILED = "gpio_failed"
}
```

### 6.2 Error Response Format

```typescript
interface MCPError {
  type: ErrorType;
  message: string;
  details?: {
    statusCode?: number;
    endpoint?: string;
    originalError?: string;
  };
  suggestions?: string[];
}
```

### 6.3 User-Friendly Error Messages

Examples:
- ❌ `HTTP 401 Unauthorized`
- ✅ `Authentication failed. Please check your PI_CONTROLLER_USERNAME and PI_CONTROLLER_PASSWORD in .mcp.json`

- ❌ `ECONNREFUSED`
- ✅ `Cannot connect to pi-controller at https://pi-controller.local:8080. Is the server running?`

## 7. Advanced Features

### 7.1 Real-time Updates (Future)

Consider WebSocket support for streaming resources:

```typescript
// Future feature
resource "cluster://{cluster_id}/events" {
  description: "Real-time cluster events stream"
  streaming: true
}
```

### 7.2 Batch Operations

```typescript
// Example: Provision multiple nodes in parallel
tool "provision_nodes" {
  inputSchema: {
    node_ids: number[]
    role: "master" | "worker"
    ssh_config: SSHConfig
  }
}
```

### 7.3 Context-Aware Suggestions

```typescript
// When creating GPIO device, suggest available pins
tool "create_gpio_device" {
  // Check node://{node_id}/gpio first
  // Suggest unused pins
}
```

## 8. Testing Strategy

### 8.1 Unit Tests
- Test each tool's API client calls
- Mock pi-controller responses
- Test error handling paths
- Validate input schemas

### 8.2 Integration Tests
- Test against real pi-controller instance
- Verify authentication flows
- Test RBAC enforcement
- Validate resource URIs

### 8.3 E2E Tests (with AI)
- Claude Code creates cluster
- AI provisions nodes
- AI controls GPIO
- AI deploys workload

## 9. Documentation

### 9.1 README.md Structure

```markdown
# pi-controller-mcp

MCP server for managing Raspberry Pi K3s clusters via AI.

## Quick Start
## Installation
## Configuration
## Available Tools
## Available Resources
## Examples
## Troubleshooting
## Development
```

### 9.2 Example Usage

```markdown
## Examples

### Create and provision a cluster

**User:** "Create a 3-node K3s cluster called 'homelab'"

**Claude Code will:**
1. Use `create_cluster` to create cluster definition
2. Use `discover_nodes` to find available Pi nodes
3. Use `provision_cluster` to install K3s
4. Use `cluster://{id}/status` to monitor progress

### Control GPIO from AI

**User:** "Turn on the LED connected to GPIO pin 18 on pi-worker-1"

**Claude Code will:**
1. Use `discover_nodes` to find pi-worker-1
2. Use `list_gpio_devices` to find GPIO device on pin 18
3. Use `write_gpio_pin` to set value to 1
```

## 10. Implementation Phases

### Phase 1: Core Infrastructure (Week 1)
- [ ] Project scaffolding
- [ ] API client with authentication
- [ ] Basic cluster tools (list, get, create)
- [ ] Basic resources (cluster status)
- [ ] Error handling

### Phase 2: Node & Provisioning (Week 2)
- [ ] Node management tools
- [ ] Provisioning tools
- [ ] Node resources
- [ ] SSH configuration handling

### Phase 3: GPIO Control (Week 3)
- [ ] GPIO management tools
- [ ] GPIO read/write operations
- [ ] GPIO resources
- [ ] Pin reservation system

### Phase 4: Advanced Features (Week 4)
- [ ] Deployment tools
- [ ] CA management tools
- [ ] Batch operations
- [ ] Enhanced error messages

### Phase 5: Polish & Release
- [ ] Comprehensive testing
- [ ] Documentation
- [ ] npm package publishing
- [ ] Example workflows

## 11. Success Metrics

### Developer Experience
- Time to first cluster: < 5 minutes
- Command success rate: > 95%
- Error message clarity: User can fix without docs

### AI Experience
- Tool discovery: AI can find right tool for task
- Resource context: AI can access cluster state
- Error recovery: AI can retry with corrections

### System Performance
- Tool response time: < 2 seconds (90th percentile)
- Resource fetch time: < 1 second (90th percentile)
- Token refresh: Transparent to user

## 12. Future Enhancements

1. **Multi-cluster Management**
   - Manage multiple pi-controller instances
   - Cross-cluster operations
   - Unified resource view

2. **Advanced GPIO**
   - PWM control
   - I2C/SPI device support
   - Interrupt handling

3. **Monitoring Integration**
   - Prometheus metrics export
   - Alert configuration
   - Dashboard generation

4. **GitOps Integration**
   - Sync cluster state with Git
   - Declarative cluster management
   - Drift detection

## 13. Related Projects

- **kubes-aura**: Web UI for pi-controller (separate repo)
- **pi-agent**: Node agent running on each Pi
- **pi-controller**: Main control plane (this project)

## 14. Contributing

See `CONTRIBUTING.md` for development workflow, coding standards, and PR process.

## 15. License

MIT License - Same as pi-controller

---

## Appendix A: Complete Tool List

### Cluster Management (6 tools)
- `create_cluster` - Create cluster definition
- `list_clusters` - List all clusters
- `get_cluster_status` - Get cluster details
- `provision_cluster` - Provision K3s cluster
- `scale_cluster` - Scale cluster nodes
- `delete_cluster` - Delete cluster

### Node Management (5 tools)
- `discover_nodes` - List discovered nodes
- `get_node_info` - Get node details
- `register_node` - Manually register node
- `provision_node` - Provision single node
- `deprovision_node` - Remove K3s from node

### GPIO Control (8 tools)
- `list_gpio_devices` - List GPIO devices
- `create_gpio_device` - Register GPIO device
- `read_gpio_pin` - Read pin value
- `write_gpio_pin` - Write pin value
- `reserve_gpio_pin` - Reserve pin
- `release_gpio_pin` - Release pin
- `get_gpio_readings` - Get historical readings
- `delete_gpio_device` - Remove GPIO device

### Deployment (3 tools)
- `deploy_pod` - Deploy Kubernetes pod
- `get_pod` - Get pod info
- `delete_pod` - Delete pod

### Certificate Authority (4 tools)
- `initialize_ca` - Initialize CA
- `issue_certificate` - Issue new cert
- `list_certificates` - List all certs
- `revoke_certificate` - Revoke cert

**Total: 26 tools**

## Appendix B: Complete Resource List

### Cluster Resources (2 resources)
- `cluster://{cluster_id}/status` - Cluster status
- `cluster://{cluster_id}/nodes` - Cluster nodes

### Node Resources (3 resources)
- `node://{node_id}/info` - Node information
- `node://{node_id}/metrics` - Node metrics
- `node://{node_id}/gpio` - GPIO devices

### GPIO Resources (1 resource)
- `gpio://{gpio_id}/state` - GPIO state

### System Resources (1 resource)
- `system://health` - System health

**Total: 7 resources**

---

**Document Version:** 1.0
**Last Updated:** 2025-01-15
**Author:** Pi-Controller Team
**Review Status:** Ready for Implementation
