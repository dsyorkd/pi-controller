# Portable Mode

## Overview

Portable mode allows pi-controller to run as a client-only management interface without participating in the Raft cluster. This is useful for:

- **Remote Management**: Run pi-controller from a laptop/workstation to manage a cluster
- **Provisioning**: Deploy and configure clusters from a control plane host
- **CI/CD Integration**: Automate cluster operations from build servers
- **Temporary Operations**: Run commands without permanently joining the cluster

## Enabling Portable Mode

### Option 1: Environment Variable (Recommended)

```bash
export PI_CONTROLLER_MODE=portable
pi-controller
```

The environment variable automatically:

- Sets `cluster.portable = true`
- Disables Raft participation (`cluster.enabled = false`)
- Allows all API operations as a client

### Option 2: Configuration File

Edit your `pi-controller.yaml`:

```yaml
cluster:
  enabled: false
  portable: true
```

## Usage Examples

### Remote Cluster Management

```bash
# From your laptop, manage a remote cluster
export PI_CONTROLLER_MODE=portable

# Discover nodes on the network
pi-controller discover --scan --interface=en0

# List nodes in the remote cluster
pi-controller nodes list

# Create a new GPIO device
pi-controller gpio create \
    --node=1 \
    --pin=17 \
    --direction=output \
    --name="LED"

# Control GPIO remotely
pi-controller gpio write --id=1 --value=1
```

### Initial Cluster Provisioning

```bash
# Run from control plane host
export PI_CONTROLLER_MODE=portable

# Discover Raspberry Pis
pi-controller discover --scan

# Install pi-controller on discovered nodes
pi-controller install \
    --nodes=192.168.1.10,192.168.1.11,192.168.1.12 \
    --bootstrap \
    --ssh-user=pi \
    --ssh-key=~/.ssh/id_rsa

# Verify cluster health
pi-controller cluster status
```

### Kubernetes Installation

```bash
# From laptop in portable mode
export PI_CONTROLLER_MODE=portable

# Install K3s on existing cluster
pi-controller kubernetes install \
    --distribution=k3s \
    --config=cluster.yaml

# Monitor installation progress
pi-controller kubernetes status
```

### CI/CD Pipeline Integration

```yaml
# .github/workflows/deploy.yml
name: Deploy to Pi Cluster

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup pi-controller
        run: |
          curl -sSL https://install.pi-controller.io | bash
          export PI_CONTROLLER_MODE=portable

      - name: Deploy application
        run: |
          pi-controller kubernetes apply -f deployment.yaml
```

## How It Works

### Architecture in Portable Mode

```
┌─────────────────┐                 ┌──────────────────────┐
│   Laptop/CI     │                 │   Pi Cluster         │
│                 │                 │                      │
│  pi-controller  │────────gRPC────▶│  ┌────┐  ┌────┐    │
│  (portable)     │                 │  │Pi-1│  │Pi-2│    │
│                 │                 │  └────┘  └────┘    │
│  • No Raft      │                 │     ▲       ▲       │
│  • Client only  │                 │     └─ Raft ─┘      │
│  • Full API     │                 │                      │
└─────────────────┘                 └──────────────────────┘
```

### What Gets Disabled

When running in portable mode:

- ❌ **Raft Initialization**: No Raft cluster participation
- ❌ **Leader Election**: Doesn't vote or become leader
- ❌ **State Replication**: Doesn't replicate database changes
- ❌ **Health Checking**: No cluster health monitoring
- ❌ **Discovery Broadcasting**: Doesn't advertise via mDNS

### What Remains Enabled

- ✅ **REST API**: Full API available for cluster management
- ✅ **gRPC Client**: Communicates with cluster nodes
- ✅ **Database**: Local SQLite for caching (not replicated)
- ✅ **Discovery Client**: Can discover nodes via mDNS
- ✅ **WebSocket**: Real-time event streaming from cluster

## Configuration Details

### Portable Mode Configuration

```yaml
# Minimal portable mode config
app:
  name: pi-controller
  environment: production
  data_dir: /tmp/pi-controller  # Temporary data directory

api:
  host: 0.0.0.0
  port: 8080

grpc:
  host: 0.0.0.0
  port: 9090

cluster:
  enabled: false      # No Raft participation
  portable: true      # Client-only mode

discovery:
  mdns:
    enabled: true     # Can discover nodes
    domain: local
    service: _pi-controller._tcp
```

### Environment Variables

```bash
# Required
export PI_CONTROLLER_MODE=portable

# Optional overrides
export PI_CONTROLLER_API_PORT=8080
export PI_CONTROLLER_API_HOST=0.0.0.0
export PI_CONTROLLER_DATA_DIR=/tmp/pi-controller
export PI_CONTROLLER_LOG_LEVEL=info
```

## Security Considerations

### Authentication

Portable mode still requires authentication when accessing cluster APIs:

```bash
# Login to cluster
pi-controller login \
    --server=https://cluster.example.com \
    --username=admin

# Token saved to ~/.pi-controller/token
# Subsequent commands use saved token
```

### Network Access

Portable mode requires network access to cluster nodes:

- **REST API**: Port 8080 (HTTPS recommended)
- **gRPC**: Port 9090 (mTLS recommended)
- **Discovery**: UDP 5353 (mDNS, optional)

### Best Practices

1. **Use HTTPS**: Always use TLS for production clusters

   ```bash
   export PI_CONTROLLER_API_TLS_CERT=/path/to/cert.pem
   export PI_CONTROLLER_API_TLS_KEY=/path/to/key.pem
   ```

2. **Limit Permissions**: Use viewer/operator roles, not admin

   ```bash
   pi-controller user create \
       --username=ci-bot \
       --role=operator
   ```

3. **Token Rotation**: Regularly rotate access tokens

   ```bash
   pi-controller token rotate
   ```

4. **Network Isolation**: Use VPN or bastion host for remote access

   ```bash
   ssh -L 8080:localhost:8080 pi@bastion
   export PI_CONTROLLER_API_HOST=localhost
   ```

## Troubleshooting

### Cannot Connect to Cluster

```bash
# Check network connectivity
ping 192.168.1.10

# Verify API is accessible
curl -k https://192.168.1.10:8080/health

# Test gRPC connection
pi-controller cluster ping --node=192.168.1.10
```

### Authentication Failed

```bash
# Re-login to cluster
pi-controller login \
    --server=https://cluster.example.com \
    --username=admin

# Verify token
pi-controller whoami
```

### Discovery Not Working

```bash
# Enable debug logging
export PI_CONTROLLER_LOG_LEVEL=debug

# Check mDNS interface
pi-controller discover --scan --interface=en0 --debug

# Use manual node addition if mDNS unavailable
pi-controller nodes add \
    --ip=192.168.1.10 \
    --name=pi-1
```

## Comparison: Portable vs On-Device

| Feature | Portable Mode | On-Device Mode |
|---------|--------------|----------------|
| Raft Participation | ❌ No | ✅ Yes |
| Can Manage Cluster | ✅ Yes (client) | ✅ Yes (member) |
| Receives Replicated State | ❌ No | ✅ Yes |
| Can Be Leader | ❌ No | ✅ Yes |
| Broadcasts mDNS | ❌ No | ✅ Yes |
| Persistent Storage | 🔶 Optional | ✅ Required |
| Resource Usage | 🟢 Low | 🟡 Medium |
| Use Case | Management, CI/CD | Cluster member |

## Example Workflows

### Laptop → Cluster Management

```bash
# 1. Enable portable mode
export PI_CONTROLLER_MODE=portable

# 2. Start pi-controller on laptop
pi-controller &

# 3. Open web UI in browser
open http://localhost:8080

# 4. Manage cluster via UI or CLI
pi-controller nodes list
pi-controller gpio list
```

### Workstation → Provisioning

```bash
# 1. Discover Pis on network
export PI_CONTROLLER_MODE=portable
pi-controller discover --scan

# 2. Install on discovered nodes
pi-controller install \
    --nodes=192.168.1.10,192.168.1.11,192.168.1.12 \
    --bootstrap

# 3. Install Kubernetes (optional)
pi-controller kubernetes install

# 4. Exit portable mode (or keep running)
# Cluster now runs independently
```

### CI Server → Automated Deployments

```bash
#!/bin/bash
# deploy.sh - CI deployment script

# Enable portable mode
export PI_CONTROLLER_MODE=portable

# Start pi-controller in background
pi-controller &
sleep 5

# Deploy application
pi-controller kubernetes apply -f app.yaml

# Wait for rollout
pi-controller kubernetes rollout status deployment/app

# Run tests
pi-controller kubernetes exec pod/test -- ./run-tests.sh

# Cleanup
pkill pi-controller
```

## Advanced Usage

### Multi-Cluster Management

```bash
# Portable mode can manage multiple clusters

# Cluster 1 (production)
export PI_CONTROLLER_CLUSTER_1=https://prod.example.com
pi-controller --server=$PI_CONTROLLER_CLUSTER_1 nodes list

# Cluster 2 (staging)
export PI_CONTROLLER_CLUSTER_2=https://staging.example.com
pi-controller --server=$PI_CONTROLLER_CLUSTER_2 nodes list
```

### Headless Automation

```bash
# No user interaction required
export PI_CONTROLLER_MODE=portable
export PI_CONTROLLER_TOKEN=$(cat /secure/token)

# Automated operations
pi-controller gpio write --id=1 --value=1
pi-controller nodes update --id=5 --status=maintenance
```

## See Also

- [Installation Guide](./installation.md)
- [Cluster Management](./cluster-management.md)
- [Provisioning Guide](./provisioning.md)
- [CI/CD Integration](./ci-cd.md)
