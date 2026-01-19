---
layout: home
title: Pi-Controller
nav_order: 1
description: Comprehensive Raspberry Pi cluster management with GPIO-as-a-Service
---

{: .fs-9 }

Comprehensive Raspberry Pi cluster management with GPIO-as-a-Service.
{: .fs-6 .fw-300 }

[Get Started](./quickstart.html){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
[View on GitHub](https://github.com/yourusername/pi-controller){: .btn .fs-5 .mb-4 .mb-md-0 }

---

## What is Pi-Controller?

Pi-Controller is a **single-binary** cluster management platform designed specifically for Raspberry Pi enthusiasts and homelabbers. It unifies node discovery, distributed consensus, and hardware control into a unified platform.

Unlike managing individual Pis with SSH scripts or heavy automation tools, Pi-Controller creates a **self-organizing cluster** where every node is aware of the others, and you can control them all through a single API.

### Key Features

- **Single Binary**: No complex dependencies. One binary runs everywhere.
- **Auto-Discovery**: Zero-configuration node detection using mDNS.
- **Raft Clustering**: Distributed state consistency. If one node dies, the cluster survives.
- **GPIO-as-a-Service**: Control pins, PWM, and I2C across your cluster via REST, gRPC, or WebSocket.
- **Optional Kubernetes**: Seamlessly deploy K3s and manage GPIO via Kubernetes Custom Resources (CRDs).
- **Portable Mode**: Manage your cluster remotely from your laptop.

## How It Works

Pi-Controller uses a peer-to-peer architecture. Every node runs the same software. When deployed, they form a **Raft consensus cluster**, ensuring that state (like node configurations and user accounts) is replicated and consistent.

You can interact with the cluster via:

1. **CLI**: `pi-controller` command-line tool.
2. **Web UI**: A modern dashboard for visualizing your cluster.
3. **API**: REST and gRPC endpoints for custom integrations.

[Read the Architecture Guide](./architecture/index.html){: .btn .btn-outline }

## Documentation Structure

- **[Quick Start](./quickstart.html)**: Get your first cluster running in minutes.
- **[Architecture](./architecture/index.html)**: Deep dive into Raft, Discovery, and the internal design.
- **[Operations](./operations/installation.html)**: Installation, Security, TLS, and Maintenance.
- **[Reference](./reference/configuration.html)**: API specs, Configuration options, and CLI commands.
- **[Development](./development/setup.html)**: Guide for contributors and developers building on Pi-Controller.
