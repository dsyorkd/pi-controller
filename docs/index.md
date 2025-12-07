# Pi-Controller

Comprehensive Raspberry Pi cluster management with GPIO-as-a-Service.

## Quick Install

```bash
curl -sSL https://pi-controller.io/install.sh | bash
```

## What is Pi-Controller?

Pi-Controller is a **single-binary** cluster management platform for Raspberry Pi that provides:

- **Automated Discovery** - mDNS-based node detection, zero configuration
- **Raft Clustering** - Distributed consensus for high availability
- **GPIO-as-a-Service** - Control GPIO pins across your cluster via REST/gRPC/WebSocket
- **Optional Kubernetes** - Deploy K3s with CRD-based GPIO management
- **Portable Mode** - Manage clusters from your laptop without joining
- **Zero Dependencies** - Embedded SQLite, built-in CA, fully self-contained

## Architecture

Pi-Controller uses a **single-binary model** - all nodes run identical code. There are no separate control plane or agent binaries, only Raft leader/followers for consensus.

### Deployment Modes

**Portable Mode** (Laptop/Workstation):

```bash
PORTABLE_MODE=true curl -sSL https://pi-controller.io/install.sh | bash
```

Manage your Raspberry Pis remotely without joining the cluster.

**On-Device Mode** (Raspberry Pi):

```bash
curl -sSL https://pi-controller.io/install.sh | bash
sudo systemctl enable --now pi-controller
```

Run as a systemd service on each Pi, forming a distributed Raft cluster.

[Read full architecture →](../ARCHITECTURE.md)

## Getting Started

### 1. Install on Raspberry Pis

On each Raspberry Pi in your cluster:

```bash
curl -sSL https://pi-controller.io/install.sh | bash
sudo systemctl enable --now pi-controller
```

### 2. Configure First Node

Edit `/etc/pi-controller/config.yaml` on your first Pi:

```yaml
cluster:
  enabled: true
  node_id: "pi-1"
  bootstrap: true  # Only on first node
```

Restart:

```bash
sudo systemctl restart pi-controller
```

### 3. Join Additional Nodes

On subsequent Pis, edit `/etc/pi-controller/config.yaml`:

```yaml
cluster:
  enabled: true
  node_id: "pi-2"  # Unique per node
  bootstrap: false
  join_addresses:
    - "192.168.1.10:9091"  # First node's IP
```

Restart:

```bash
sudo systemctl restart pi-controller
```

### 4. Verify Cluster

```bash
pi-controller cluster status
```

## Features

### GPIO Control

Control GPIO pins across your entire cluster:

```bash
# Set pin high
pi-controller gpio set --node=pi-1 --pin=17 --value=high

# Read pin value
pi-controller gpio get --node=pi-1 --pin=17

# Configure PWM
pi-controller gpio pwm --node=pi-2 --pin=18 --frequency=1000 --duty-cycle=50
```

### Auto-Discovery

Discover Raspberry Pis on your network automatically:

```bash
pi-controller discover --scan
```

### Kubernetes Integration

Optionally deploy K3s for advanced orchestration:

```bash
pi-controller kubernetes install \
  --distribution=k3s \
  --server-nodes=pi-1,pi-2,pi-3
```

Includes:

- GPIO Custom Resource Definitions (CRDs)
- Kubernetes operator for GPIO management
- Seamless migration from systemd to DaemonSet

### REST API

```bash
# Get cluster status
curl http://pi-controller:8080/api/v1/cluster/status

# Control GPIO
curl -X POST http://pi-controller:8080/api/v1/gpio/pins/17 \
  -d '{"node_id": "pi-1", "value": 1}'

# List nodes
curl http://pi-controller:8080/api/v1/nodes
```

### WebSocket Events

Real-time streaming of cluster events:

```javascript
const ws = new WebSocket('ws://pi-controller:8081/ws');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Cluster event:', data);
};
```

## Use Cases

- **Home Automation** - Control lights, sensors, and actuators
- **IoT Projects** - Distributed sensor networks
- **Edge Computing** - Kubernetes at the edge with GPIO
- **Learning Platform** - Explore distributed systems and Kubernetes
- **Homelab Clusters** - Self-hosted services with hardware control

## Documentation

### Installation & Setup

- [Installation Guide](../INSTALLATION.md) - Install pi-controller
- [Configuration Reference](../config/config.example.yaml) - All configuration options

### Architecture & Design

- [Architecture Overview](../ARCHITECTURE.md) - System design and philosophy
- [Raft Clustering](../docs/CONTROLLER_CLUSTERING.md) - Consensus and replication

### API Reference

- [REST API](rest.md) - HTTP endpoints
- [gRPC API](grpc.md) - gRPC services
- [WebSocket API](websocket.md) - Real-time events

### Development

- [Getting Started](getting-started.md) - Development setup
- [Contributing](../CONTRIBUTING.md) - How to contribute
- [Testing Guide](../TESTING_FRAMEWORK_SUMMARY.md) - Testing framework

## Community & Support

- **Documentation**: [pi-controller.io](https://pi-controller.io)
- **GitHub**: [github.com/yourusername/pi-controller](https://github.com/yourusername/pi-controller)
- **Issues**: [Report bugs or request features](https://github.com/yourusername/pi-controller/issues)
- **Discussions**: [Join the conversation](https://github.com/yourusername/pi-controller/discussions)

## License

[Your License Here]

---

**Pi-Controller** - Built with ❤️ for the Raspberry Pi community
