# Pi-Controller

> Single-binary Raspberry Pi cluster management with optional Kubernetes integration

Pi-Controller is a comprehensive cluster management platform for Raspberry Pi that provides automated discovery, Raft-based clustering, and GPIO-as-a-Service capabilities.

## Quick Start

```bash
curl -sSL https://pi-controller.io/install.sh | bash
```

## Features

- **Single Binary** - One executable runs everywhere, no separate agents
- **Auto-Discovery** - mDNS-based automatic node detection
- **Raft Clustering** - Distributed consensus for high availability
- **GPIO Control** - Manage GPIO pins across your cluster via REST/gRPC/WebSocket APIs
- **Optional Kubernetes** - Deploy K3s with CRD-based GPIO management
- **Portable Mode** - Manage clusters remotely from your laptop
- **Zero Dependencies** - Embedded SQLite, built-in CA, self-contained

## Architecture

Pi-Controller uses a **single-binary model** where all nodes run identical code. There are no separate control plane or agent binaries - only Raft leader/followers for distributed consensus.

- **Portable Mode**: Run on your laptop to provision and manage remote Pis
- **On-Device Mode**: Run on each Pi as a systemd service, forming a Raft cluster
- **Kubernetes Mode**: Optionally deploy K3s and migrate to DaemonSet deployment

Read the [Architecture Overview](ARCHITECTURE.md) for details.

## Installation

### Raspberry Pi (On-Device Mode)

```bash
curl -sSL https://pi-controller.io/install.sh | bash
sudo systemctl enable --now pi-controller
```

### Laptop/Workstation (Portable Mode)

```bash
PORTABLE_MODE=true curl -sSL https://pi-controller.io/install.sh | bash
```

See the [Installation Guide](INSTALLATION.md) for detailed instructions.

## Usage

### Discover Nodes

```bash
pi-controller discover --scan
```

### Provision a Cluster

```bash
pi-controller install \
  --nodes=192.168.1.10,192.168.1.11,192.168.1.12 \
  --bootstrap \
  --ssh-user=pi
```

### Control GPIO

```bash
# Set pin high
pi-controller gpio set --node=pi-1 --pin=17 --value=high

# Read pin value
pi-controller gpio get --node=pi-1 --pin=17
```

### Check Cluster Status

```bash
pi-controller cluster status
```

## Documentation

- **[Installation Guide](INSTALLATION.md)** - Install and configure pi-controller
- **[Architecture](ARCHITECTURE.md)** - System design and architecture
- **[Getting Started](docs/getting-started.md)** - Development setup
- **[REST API](docs/rest.md)** - HTTP API reference
- **[WebSocket API](docs/websocket.md)** - Real-time event streaming
- **[Configuration](config/config.example.yaml)** - Configuration reference
- **[Contributing](CONTRIBUTING.md)** - Development guidelines

## API Endpoints

- **REST API**: `http://pi-controller:8080/api/v1`
- **gRPC**: `pi-controller:9090`
- **WebSocket**: `ws://pi-controller:8081/ws`
- **Raft**: `pi-controller:9091` (internal)

## Building from Source

```bash
# Clone repository
git clone https://github.com/yourusername/pi-controller.git
cd pi-controller

# Install dependencies
make deps

# Build for your platform
make build

# Build for all platforms
make build-all

# Run tests
make test-all
```

See [Getting Started](docs/getting-started.md) for development details.

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for:

- Development workflow (GitFlow)
- Commit message conventions (Conventional Commits)
- Testing requirements
- Code style guidelines

## Security

For security issues, please see [SECURITY.md](SECURITY.md).

## License

[Your License Here]

## Support

- **Documentation**: [pi-controller.io](https://pi-controller.io)
- **GitHub Issues**: [github.com/yourusername/pi-controller/issues](https://github.com/yourusername/pi-controller/issues)
- **Discussions**: [github.com/yourusername/pi-controller/discussions](https://github.com/yourusername/pi-controller/discussions)

---

**Made with ❤️ for Raspberry Pi enthusiasts**
