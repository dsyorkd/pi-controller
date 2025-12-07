# Pi Controller Ansible Infrastructure

Ansible playbooks and roles for deploying, testing, and managing pi-controller on Raspberry Pi clusters.

## Purpose

This Ansible infrastructure provides:

1. **Deployment** - Install pi-controller binary to Pi nodes
2. **Cleanup/Recovery** - Reset cluster state when pi-controller fails
3. **Race Condition Prevention** - Lock mechanism for CI to prevent concurrent test runs
4. **K3s Rollback** - Clean K3s installations that pi-controller can't recover from

> **Note:** pi-controller handles K3s installation itself. Ansible only handles cleanup/rollback of failed K3s installations.

## Quick Start

### Prerequisites

```bash
# Install Ansible
pip install ansible

# Verify installation
ansible --version
```

### Configure Inventory

Edit `inventory/hosts.yml` with your Pi hosts:

```yaml
all:
  children:
    pis:
      hosts:
        pi-node-1:
          ansible_host: 192.168.1.101
        pi-node-2:
          ansible_host: 192.168.1.102
      vars:
        ansible_user: pi
        ansible_ssh_private_key_file: ~/.ssh/pi_key
    bootstrap:
      hosts:
        pi-node-1:
    workers:
      hosts:
        pi-node-2:
```

### Deploy and Test

```bash
# Using the helper script (recommended)
./scripts/deploy-and-test.sh --local-binary

# Or directly with Ansible
cd ansible
ansible-playbook playbooks/deploy-test.yml \
  -e pi_controller_local_binary=../build/pi-controller-linux-arm64
```

## Playbooks

| Playbook | Purpose |
|----------|---------|
| `site.yml` | Full deployment (prepare nodes + install pi-controller) |
| `deploy-test.yml` | Deploy + run integration tests (with locking) |
| `reset-cluster.yml` | **DESTRUCTIVE** - Clean all cluster state |
| `release-lock.yml` | Emergency release of test lock |

### Common Usage

```bash
cd ansible

# Full fresh deployment
ansible-playbook playbooks/site.yml

# Deploy local build and run tests
ansible-playbook playbooks/deploy-test.yml \
  -e pi_controller_local_binary=../build/pi-controller-linux-arm64

# Deploy without tests
ansible-playbook playbooks/deploy-test.yml \
  -e run_tests=false \
  -e pi_controller_local_binary=../build/pi-controller-linux-arm64

# Reset everything and start fresh
ansible-playbook playbooks/reset-cluster.yml -e force_reset=true

# Force release a stale lock
ansible-playbook playbooks/release-lock.yml
```

## Roles

### cluster-lock

Prevents race conditions during CI by maintaining a lock file on the bootstrap node.

```yaml
- include_role:
    name: cluster-lock
  vars:
    lock_action: acquire  # or 'release'
    lock_reason: "my-test-run"
```

### pi-base

Prepares Pi nodes: packages, GPIO permissions, networking, time sync.

### pi-controller

Installs and configures the pi-controller service.

Variables:

- `pi_controller_local_binary` - Path to local binary (for dev/CI)
- `pi_controller_version` - Version to download from releases
- `cluster_bootstrap` - Set `true` for the first node

### k3s

**Cleanup only** - removes K3s installations when pi-controller can't recover.

## Directory Structure

```
ansible/
├── ansible.cfg              # Ansible configuration
├── inventory/
│   ├── hosts.yml           # Cluster inventory
│   └── group_vars/
│       └── all.yml         # Global variables
├── playbooks/
│   ├── site.yml            # Main deployment
│   ├── deploy-test.yml     # Deploy + test with locking
│   ├── reset-cluster.yml   # Full cleanup
│   └── release-lock.yml    # Emergency lock release
└── roles/
    ├── cluster-lock/       # Distributed locking
    ├── pi-base/            # OS preparation
    ├── pi-controller/      # Service management
    └── k3s/                # K3s cleanup
```

## CI/CD Integration

The GitHub Actions workflow (`.github/workflows/pi-integration-test.yml`) automatically:

1. Builds the ARM64 binary
2. Deploys to the Pi cluster
3. Runs integration tests
4. Cleans up on failure

The `concurrency` block ensures only one test run can use the cluster at a time:

```yaml
concurrency:
  group: pi-cluster
  cancel-in-progress: false
```

### Required Secrets

| Secret | Description |
|--------|-------------|
| `PI_SSH_KEY` | SSH private key for Pi access |
| `PI_HOST` | Pi host IP (if using port forwarding) |

## Troubleshooting

### Lock is stuck

```bash
ansible-playbook playbooks/release-lock.yml
```

### K3s won't uninstall

```bash
# Run the k3s role directly
ansible pis -m include_role -a name=k3s
```

### Check cluster health

```bash
ansible bootstrap -m uri \
  -a "url=http://127.0.0.1:8080/health"
```

### View pi-controller logs

```bash
ansible pis -m shell \
  -a "journalctl -u pi-controller --no-pager -n 50"
```
