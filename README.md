# Pi-Controller

> Single-binary Raspberry Pi cluster management with optional Kubernetes integration.

[![CI Status](https://github.com/yourusername/pi-controller/workflows/CI/badge.svg)](https://github.com/yourusername/pi-controller/actions)
[![Documentation](https://img.shields.io/badge/docs-pi--controller.io-blue)](https://pi-controller.io)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

**Pi-Controller** is a comprehensive platform for managing Raspberry Pi clusters. It combines automated discovery, Raft-based distributed consensus, and GPIO-as-a-Service into a single, dependency-free binary.

## 🚀 Quick Start

**Install on Raspberry Pi:**

```bash
curl -sSL https://pi-controller.io/install.sh | bash
sudo systemctl enable --now pi-controller
```

**Install on Laptop (Portable Mode):**

```bash
PORTABLE_MODE=true curl -sSL https://pi-controller.io/install.sh | bash
```

[View the Full Quick Start Guide](docs/quickstart.md)

## 📚 Documentation

Complete documentation is available at [pi-controller.io](https://pi-controller.io) (or in the `docs/` folder).

- **[Getting Started](docs/quickstart.md)**: Set up your first cluster.
- **[Architecture](docs/architecture/index.md)**: Deep dive into the system design.
- **[Operations](docs/operations/installation.md)**: Installation, Security, and K8s integration.
- **[API Reference](docs/reference/rest-api.md)**: REST, gRPC, and WebSocket APIs.
- **[Contributing](CONTRIBUTING.md)**: Guide for developers.

## ✨ Features

- **Single Binary**: No complex dependencies (Python, Ruby, etc. not required).
- **Auto-Discovery**: Nodes find each other automatically via mDNS.
- **Raft Clustering**: High-availability consensus engine.
- **GPIO Control**: Control hardware remotely via API or Kubernetes CRDs.
- **Kubernetes Ready**: Optional one-click K3s deployment.

## 🛠️ Building from Source

```bash
git clone https://github.com/yourusername/pi-controller.git
cd pi-controller
make build
```

See [Development Setup](docs/development/setup.md) for more details.

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details on our workflow and code standards.

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.
