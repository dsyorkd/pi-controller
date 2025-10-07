---
layout: home
title: Home
nav_order: 0
---

# Pi-Controller Documentation
{: .fs-9 }

Comprehensive Kubernetes management platform for Raspberry Pi clusters
{: .fs-6 .fw-300 }

[Get Started](#getting-started){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
[View on GitHub](https://github.com/dsyorkd/pi-controller){: .btn .fs-5 .mb-4 .mb-md-0 }

---

## Overview

Pi-Controller is a single-binary platform that provides:

- **Automated Discovery**: Find and register Raspberry Pi nodes automatically via mDNS
- **K3s Management**: Deploy and manage lightweight Kubernetes clusters
- **GPIO-as-a-Service**: Control hardware through Kubernetes Custom Resources
- **High Availability**: Raft consensus for fault-tolerant cluster operations
- **Zero Dependencies**: Embedded SQLite, runs anywhere

## Quick Start

```bash
# Download and install
curl -L https://github.com/dsyorkd/pi-controller/releases/latest/download/pi-controller-linux-arm64 -o pi-controller
chmod +x pi-controller

# Start the controller
./pi-controller start

# Access the dashboard
open http://localhost:8080
```

## Documentation Structure

### Getting Started
Start here to set up your development environment and understand basic concepts.

### System Overview
High-level architecture and component overview.

### Architecture
Deep dive into technical architecture:
- **Control Plane**: Core management and API gateway
- **Clustering & Raft**: High-availability consensus protocol

### Deployment
Deployment strategies and migration guides:
- **Portable Mode**: Client-only mode for remote management
- **Raft Migration**: Zero-downtime migration from systemd to DaemonSet

### Configuration
Unified configuration system for all deployment modes.

### Security
Comprehensive security implementation:
- **Security Implementation**: Best practices and measures
- **TLS Setup**: Secure communication configuration
- **Certificate Authority**: PKI infrastructure

### API Reference
Complete API documentation:
- **REST API**: RESTful HTTP interface
- **gRPC API**: High-performance RPC communication
- **WebSocket API**: Real-time event streaming

### Advanced Topics
Specialized features and integrations:
- **Node Discovery**: Automatic node detection
- **MCP Server Integration**: AI-assisted operations
- **CI/CD Requirements**: Pipeline container specifications

## Key Features

### Single Binary Architecture
One simple binary for all nodes - no separate agent/controller binaries. Simplified deployment and maintenance.

### Portable Mode
Run pi-controller from your laptop/workstation to manage remote clusters without joining the Raft cluster.

### Raft Clustering
Built-in high availability using HashiCorp Raft for distributed consensus and state replication.

### GPIO Control
Manage hardware through Kubernetes CRDs - no custom controller needed.

## Support

- **GitHub Issues**: [Report bugs and request features](https://github.com/dsyorkd/pi-controller/issues)
- **Documentation**: You're reading it!
- **Source Code**: [github.com/dsyorkd/pi-controller](https://github.com/dsyorkd/pi-controller)

---

{: .fs-3 }
