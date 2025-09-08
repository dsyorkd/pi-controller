# Docker Setup for Self-Hosted Runners

## Problem

Your GitHub Actions workflows are failing with "docker: command not found" because your self-hosted runner doesn't have Docker installed or accessible to GitHub Actions.

## Solution Options

### Option 1: Install Docker on Self-Hosted Runner (Recommended)

#### Install Docker Engine

```bash
# Update package index
sudo apt-get update

# Install packages to allow apt to use a repository over HTTPS
sudo apt-get install \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

# Add Docker's official GPG key
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

# Set up the stable repository
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Update package index again
sudo apt-get update

# Install Docker Engine
sudo apt-get install docker-ce docker-ce-cli containerd.io

# Start and enable Docker
sudo systemctl start docker
sudo systemctl enable docker
```

#### Configure Runner User Access

```bash
# Add the runner user to the docker group
sudo usermod -aG docker $USER

# Apply group membership (may need to logout/login or restart runner service)
newgrp docker

# Test Docker access
docker --version
docker run hello-world
```

#### Configure Runner Service

If using systemd service for the runner:

```bash
# Edit the runner service to ensure proper environment
sudo systemctl edit actions-runner

# Add this content:
[Service]
ExecStartPre=/bin/bash -c 'while [ ! -S /var/run/docker.sock ]; do sleep 1; done'
SupplementaryGroups=docker
```

#### Restart GitHub Actions Runner

```bash
# If using systemd
sudo systemctl restart actions-runner

# If running manually, restart the run.sh script
```

### Option 2: Use Fallback Workflows (Temporary)

If you can't install Docker immediately, use the fallback workflows:

1. **Rename current workflows** (add `.disabled` extension):
```bash
cd .github/workflows
mv ci.yml ci.yml.disabled
mv security.yml security.yml.disabled  
mv build-multiarch.yml build-multiarch.yml.disabled
```

2. **Activate fallback workflow**:
```bash
mv ci-fallback.yml ci.yml
```

### Option 3: Use GitHub-Hosted Runners

Temporarily switch to GitHub-hosted runners by changing:

```yaml
runs-on: [self-hosted, ubuntu]
```

to:

```yaml
runs-on: ubuntu-latest
```

## Verification

After installing Docker, verify the setup:

```bash
# Check Docker is running
sudo systemctl status docker

# Check runner user can access Docker
docker --version
docker info

# Test pulling the ci-image containers
docker pull ghcr.io/dsyorkd/ci-image/ci-go-npm:v1.0
docker pull ghcr.io/dsyorkd/ci-image/ci-security:v1.0
```

## Benefits of Using Containers

- **Consistency**: Same environment every time
- **Speed**: No tool installation during runs  
- **Reliability**: Pre-tested container images
- **Security**: Isolated execution environment
- **Maintenance**: Centralized toolchain updates

## Troubleshooting

### Permission Issues

```bash
# Check current user groups
groups

# Verify docker group membership
id $USER

# If docker group not shown, re-add user
sudo usermod -aG docker $USER
sudo systemctl restart actions-runner
```

### Service Issues

```bash
# Check Docker service status
sudo systemctl status docker

# Check Docker socket permissions
ls -la /var/run/docker.sock

# Restart Docker if needed
sudo systemctl restart docker
```

### Container Pull Issues

```bash
# Login to GitHub Container Registry if needed
echo "$GITHUB_TOKEN" | docker login ghcr.io -u USERNAME --password-stdin

# Test container access
docker run --rm ghcr.io/dsyorkd/ci-image/ci-go-npm:v1.0 go version
```