# Pi Controller CI/CD Setup

This directory contains the GitHub Actions workflows and configuration for the Pi Controller project's continuous integration and deployment pipeline.

## Workflows Overview

### 🔄 [CI Pipeline](workflows/ci.yml)
**Triggers:** Push, PR to `main/develop/master`  
**Runners:** Self-hosted Ubuntu, Raspberry Pi 5

**Jobs:**
- **validate**: Code formatting, linting, vet checks
- **test**: Unit, integration, API, and security tests
- **build**: Multi-architecture binary builds
- **hardware-test**: Hardware-in-the-loop testing on actual Pi
- **migration-test**: Database migration validation
- **performance-test**: Race detection, benchmarks, fuzzing
- **comprehensive-test**: Full test suite execution
- **status-check**: Overall CI status verification

### 🚀 [Release Pipeline](workflows/release.yml)
**Triggers:** Git tags (`v*`), releases, manual dispatch  
**Runners:** Self-hosted Ubuntu, Raspberry Pi 5

**Features:**
- Multi-platform binary builds (Linux, macOS, Windows)
- ARM variants (ARMv6, ARMv7, ARM64)
- Docker multi-arch images
- Hardware validation on Raspberry Pi
- GitHub release creation with assets
- Systemd service files and installation scripts

### 🔒 [Security Scanning](workflows/security.yml)
**Triggers:** Push, PR, daily schedule (3 AM UTC)  
**Runners:** Self-hosted Ubuntu

**Scans:**
- **Dependency vulnerabilities** (govulncheck, nancy)
- **Static code analysis** (gosec, staticcheck, golangci-lint)
- **CodeQL analysis** (GitHub's semantic code analysis)
- **Container scanning** (Trivy for Docker images)
- **Secrets detection** (TruffleHog, detect-secrets)
- **License compliance** (go-licenses)

### 🏗️ [Multi-Architecture Build](workflows/build-multiarch.yml)
**Triggers:** Code changes, manual dispatch  
**Runners:** Self-hosted Ubuntu, Raspberry Pi 5

**Platforms:**
- **Primary:** Linux (amd64, arm64, armv7, armv6)
- **Secondary:** macOS (Intel, Apple Silicon), Windows
- **BSD:** FreeBSD, OpenBSD, NetBSD
- **Experimental:** MIPS, PowerPC, s390x, RISC-V

### 🔧 [Hardware Testing](workflows/hardware-testing.yml)
**Triggers:** Daily schedule (2 AM UTC), hardware code changes  
**Runners:** Raspberry Pi 5 Model B

**Tests:**
- GPIO pin functionality and state transitions
- Performance benchmarks on Pi hardware
- Controller + Agent integration testing
- Hardware-specific feature validation

### 🤖 [Dependabot Auto-merge](workflows/dependabot-auto-merge.yml)
**Triggers:** Dependabot PRs  
**Features:**
- Auto-merge minor and patch updates after CI passes
- Manual review required for major updates
- Automatic approval for safe updates

## Self-Hosted Runners

### Ubuntu Runner
**Tags:** `[self-hosted, ubuntu]`  
**Purpose:** Primary CI/CD workload, builds, tests, security scanning  
**Specifications:** Optimized for CPU-intensive tasks, Docker builds

### Raspberry Pi Runner
**Tags:** `[self-hosted, raspberry-pi, 5-Model-B]`  
**Purpose:** Hardware-in-the-loop testing, ARM builds  
**Hardware:** Raspberry Pi 5 Model B  
**Use Cases:**
- GPIO hardware testing
- ARM64/ARMv7 build validation
- Real hardware integration tests
- Performance testing on target platform

## Workflow Configuration

### Environment Variables
```yaml
GO_VERSION: "1.22"
GOLANGCI_LINT_VERSION: "v1.54" 
REGISTRY: ghcr.io
```

### Security Considerations
- All secrets managed via GitHub Secrets
- Container registry authentication
- Code scanning and vulnerability detection
- License compliance checking
- No hardcoded credentials or tokens

### Artifact Management
- **Build artifacts:** 7-day retention
- **Test results:** 7-30 day retention based on type
- **Security reports:** 30-day retention
- **Hardware logs:** 3-7 day retention

## Usage Examples

### Manual Workflow Dispatch
```bash
# Trigger release build
gh workflow run release.yml -f version=v1.2.0 -f prerelease=false

# Run specific hardware tests
gh workflow run hardware-testing.yml -f test_suite=gpio -f gpio_pins=18,19,20

# Build experimental architectures
gh workflow run build-multiarch.yml -f include_experimental=true
```

### Monitoring CI/CD Status
```bash
# Check workflow status
gh run list --workflow=ci.yml

# View workflow logs
gh run view --log

# Download artifacts
gh run download <run-id>
```

## Runner Maintenance

### Ubuntu Runner Setup
```bash
# Install runner software
curl -o actions-runner-linux-x64-2.311.0.tar.gz -L https://github.com/actions/runner/releases/download/v2.311.0/actions-runner-linux-x64-2.311.0.tar.gz
tar xzf ./actions-runner-linux-x64-2.311.0.tar.gz

# Configure runner
./config.sh --url https://github.com/dsyorkd/pi-controller --token <TOKEN> --labels self-hosted,ubuntu

# Install as service
sudo ./svc.sh install
sudo ./svc.sh start
```

### Raspberry Pi Runner Setup
```bash
# Install runner (ARM64)
curl -o actions-runner-linux-arm64-2.311.0.tar.gz -L https://github.com/actions/runner/releases/download/v2.311.0/actions-runner-linux-arm64-2.311.0.tar.gz
tar xzf ./actions-runner-linux-arm64-2.311.0.tar.gz

# Configure with Pi-specific labels
./config.sh --url https://github.com/dsyorkd/pi-controller --token <TOKEN> --labels self-hosted,raspberry-pi,5-Model-B

# Install as service
sudo ./svc.sh install
sudo ./svc.sh start
```

## Cost Optimization

### Runner Usage Strategy
- **Ubuntu runner** for all standard CI tasks (saves GitHub Actions credits)
- **Raspberry Pi runner** only for hardware-specific testing
- Concurrent job limits to prevent resource contention
- Artifact cleanup to minimize storage costs

### Workflow Optimization
- Path-based triggers to avoid unnecessary runs
- Concurrency groups to cancel outdated workflows
- Matrix builds to parallelize cross-platform compilation
- Conditional job execution based on changes

## Troubleshooting

### Common Issues

**Runner Offline**
```bash
# Check runner service
sudo systemctl status actions.runner.pi-controller.service

# Restart runner
sudo systemctl restart actions.runner.pi-controller.service
```

**Build Failures**
```bash
# Check disk space
df -h

# Clean build cache
make clean-all
docker system prune -af
```

**Hardware Test Failures**
```bash
# Check GPIO permissions
ls -la /sys/class/gpio/
groups $USER

# Verify Pi hardware
cat /proc/device-tree/model
```

### Monitoring Commands
```bash
# View runner logs
sudo journalctl -u actions.runner.pi-controller.service -f

# Monitor system resources
htop
iostat -x 1

# Check network connectivity
ping github.com
curl -I https://api.github.com
```

## Security Notes

- Runners isolated in secure network
- Regular security updates applied
- No sensitive data stored on runners
- All artifacts encrypted in transit
- Access logs monitored and retained

## Contributing

When modifying workflows:
1. Test changes in a fork first
2. Use workflow_dispatch for testing
3. Monitor resource usage impact
4. Update this documentation
5. Follow the existing patterns and naming conventions

For questions or issues with the CI/CD setup, please open an issue or contact @dsyorkd.