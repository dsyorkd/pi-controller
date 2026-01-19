# Installation Guide

This guide covers installing pi-controller on your Raspberry Pi cluster or workstation.

## Quick Install

```bash
curl -sSL https://pi-controller.io/install.sh | bash
```

### Installation Options

Customize using environment variables:

```bash
# Install specific version
VERSION=v1.0.0 curl -sSL https://pi-controller.io/install.sh | bash

# Install to custom directory
INSTALL_DIR=$HOME/.local/bin curl -sSL https://pi-controller.io/install.sh | bash

# Force portable mode
PORTABLE_MODE=true curl -sSL https://pi-controller.io/install.sh | bash

# Skip systemd setup
SETUP_SYSTEMD=false curl -sSL https://pi-controller.io/install.sh | bash
```

## Deployment Modes

### Portable Mode (Laptop/Workstation)

Manage remote Raspberry Pis from your laptop without joining the cluster:

```bash
PORTABLE_MODE=true curl -sSL https://pi-controller.io/install.sh | bash
```

**Usage:**

```bash
# Discover nodes
pi-controller discover --scan

# Provision cluster
pi-controller install \
  --nodes=192.168.1.10,192.168.1.11,192.168.1.12 \
  --bootstrap \
  --ssh-user=pi

# Control GPIO remotely
pi-controller gpio set --node=pi-1 --pin=17 --value=high
```

### On-Device Mode (Raspberry Pi)

Run as a systemd service on each Pi, forming a Raft cluster:

```bash
curl -sSL https://pi-controller.io/install.sh | bash
sudo systemctl enable --now pi-controller
```

## Manual Installation

Download binaries from [releases](https://github.com/yourusername/pi-controller/releases):

**Raspberry Pi (ARM64):**

```bash
wget https://github.com/yourusername/pi-controller/releases/latest/download/pi-controller-linux-arm64
chmod +x pi-controller-linux-arm64
sudo mv pi-controller-linux-arm64 /usr/local/bin/pi-controller
```

**Linux (AMD64):**

```bash
wget https://github.com/yourusername/pi-controller/releases/latest/download/pi-controller-linux-amd64
chmod +x pi-controller-linux-amd64
sudo mv pi-controller-linux-amd64 /usr/local/bin/pi-controller
```

**macOS (Apple Silicon):**

```bash
wget https://github.com/yourusername/pi-controller/releases/latest/download/pi-controller-darwin-arm64
chmod +x pi-controller-darwin-arm64
sudo mv pi-controller-darwin-arm64 /usr/local/bin/pi-controller
```

## Cluster Setup

### First Node (Bootstrap)

Edit `/etc/pi-controller/config.yaml`:

```yaml
cluster:
  enabled: true
  node_id: "pi-1"
  bind_addr: "0.0.0.0:9091"
  bootstrap: true  # Only true on first node
```

Restart:

```bash
sudo systemctl restart pi-controller
```

### Additional Nodes

Edit `/etc/pi-controller/config.yaml`:

```yaml
cluster:
  enabled: true
  node_id: "pi-2"  # Unique per node
  bind_addr: "0.0.0.0:9091"
  bootstrap: false
  join_addresses:
    - "192.168.1.10:9091"  # First node's IP
```

Restart:

```bash
sudo systemctl restart pi-controller
```

### Verify Cluster

```bash
pi-controller cluster status
```

Expected output:

```
Node ID: pi-1
Role: leader
Cluster Size: 3
Members:
  - pi-1 (leader) 192.168.1.10:9091
  - pi-2 (follower) 192.168.1.11:9091
  - pi-3 (follower) 192.168.1.12:9091
```

## Configuration

See [config/config.example.yaml](config/config.example.yaml) for complete options.

**Minimal config:**

```yaml
app:
  name: "pi-controller"
  data_dir: "/var/lib/pi-controller"

api:
  port: 8080

grpc:
  port: 9090

cluster:
  enabled: true
  node_id: "pi-1"
  bind_addr: "0.0.0.0:9091"

gpio:
  enabled: true

discovery:
  enabled: true
  method: "mdns"
```

## Network Discovery

Scan for nodes using mDNS:

```bash
pi-controller discover --scan

# Specific interface
pi-controller discover --scan --interface=eth0
```

## Kubernetes Integration (Optional)

Deploy K3s on your cluster:

```bash
pi-controller kubernetes install \
  --distribution=k3s \
  --server-nodes=pi-1,pi-2,pi-3
```

This automatically:

- Installs K3s in HA mode
- Deploys pi-controller as DaemonSet
- Registers GPIO CRDs
- Migrates cluster state
- Removes systemd services

## Troubleshooting

**Check service:**

```bash
sudo systemctl status pi-controller
sudo journalctl -u pi-controller -f
```

**Debug logging:**

```yaml
log:
  level: "debug"
  format: "text"
```

**Common issues:**

- **Cluster won't form:** Check firewall (port 9091), unique node IDs, only first node has `bootstrap: true`
- **GPIO not working:** Run as root or add user to `gpio` group
- **Discovery not working:** Enable mDNS, check firewall (UDP 5353)

## Upgrading

```bash
VERSION=v1.2.0 curl -sSL https://pi-controller.io/install.sh | bash
```

## Uninstallation

```bash
sudo systemctl stop pi-controller
sudo systemctl disable pi-controller
sudo rm /usr/local/bin/pi-controller
sudo rm /etc/systemd/system/pi-controller.service
sudo rm -rf /etc/pi-controller /var/lib/pi-controller
sudo systemctl daemon-reload
```

## Next Steps

- [Architecture Overview](../architecture/index.md)
- [API Documentation](../reference/rest-api.md)
- [Configuration Reference](../../config/config.example.yaml)
- [Development Guide](../development/setup.md)
- [Contributing](../../CONTRIBUTING.md)

## Support

- **Documentation:** [pi-controller.io](https://pi-controller.io)
- **GitHub Issues:** [github.com/yourusername/pi-controller/issues](https://github.com/yourusername/pi-controller/issues)
