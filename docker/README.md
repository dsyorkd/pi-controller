# Multi-Architecture Docker Build Guide

This directory contains Dockerfiles for building the pi-controller and pi-agent applications for multiple architectures.

## Supported Architectures

- **linux/amd64** - Standard x86_64 Linux systems
- **linux/arm64** - 64-bit ARM systems (Raspberry Pi 4, ARM-based servers)
- **linux/arm/v7** - 32-bit ARM systems (Raspberry Pi 3 and older)

## Dockerfiles

### Dockerfile.controller
Builds the pi-controller main application which includes:
- SQLite database support (requires CGO)
- REST API server
- gRPC server
- WebSocket server
- Kubernetes client functionality

### Dockerfile.agent
Builds the pi-agent application which includes:
- Hardware access capabilities (GPIO, I2C, SPI)
- System monitoring
- gRPC client for controller communication

## Multi-Architecture Build Features

### Cross-Compilation Setup
Both Dockerfiles include proper cross-compilation configuration for CGO-enabled builds:

- **AMD64**: Uses standard gcc/g++
- **ARM64**: Uses aarch64-linux-musl-gcc/g++
- **ARM32**: Uses arm-linux-musleabihf-gcc/g++

### Runtime Dependencies
Architecture-specific runtime dependencies are installed based on target platform:
- ARM architectures get additional hardware access tools (i2c-tools, spi-tools)
- All architectures get ca-certificates and tzdata

## Build Commands

### Single Architecture
```bash
# Build for specific architecture
make docker-amd64        # AMD64 only
make docker-arm64        # ARM64 only
make docker-arm          # ARM32 only
```

### Multi-Architecture
```bash
# Setup buildx (one-time)
make docker-buildx-setup

# Test multi-arch build (dry-run)
make docker-multiarch-test

# Build and push multi-arch images
make docker-multiarch
```

### Individual Components
```bash
# Build controller only
make docker-multiarch-controller

# Build agent only
make docker-multiarch-agent
```

## Image Tags

- `pi-controller:latest` - Multi-arch manifest
- `pi-controller:latest-amd64` - AMD64 specific
- `pi-controller:latest-arm64` - ARM64 specific
- `pi-controller:latest-armv7` - ARM32 specific

## Docker Registry Configuration

Configure the target registry in Makefile:
```bash
DOCKER_REGISTRY ?= localhost:5000  # Default
# or
DOCKER_REGISTRY=your-registry.com make docker-multiarch
```

## Build Arguments

All Dockerfiles support these build arguments:
- `VERSION` - Application version (from git describe)
- `COMMIT` - Git commit hash (from git rev-parse)
- `DATE` - Build timestamp (ISO 8601 format)
- `TARGETPLATFORM` - Target platform (set by buildx)
- `BUILDPLATFORM` - Build platform (set by buildx)
- `TARGETOS` - Target OS (set by buildx)
- `TARGETARCH` - Target architecture (set by buildx)

## Security Features

- Non-root user execution
- Minimal runtime image (Alpine Linux)
- Static binary linking where possible
- Health checks included
- Multi-stage builds to minimize attack surface

## Hardware Access (pi-agent only)

The pi-agent image includes hardware access capabilities:
- GPIO pin control
- I2C communication
- SPI communication
- Requires privileged mode or specific device mounts in Kubernetes

## Usage in Kubernetes

Deploy using the multi-arch images for automatic architecture selection:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pi-controller
spec:
  template:
    spec:
      containers:
      - name: pi-controller
        image: your-registry.com/pi-controller:latest
        # Kubernetes will automatically select the correct architecture
```

For the pi-agent DaemonSet, hardware access is required:

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: pi-agent
spec:
  template:
    spec:
      hostNetwork: true
      hostPID: true
      containers:
      - name: pi-agent
        image: your-registry.com/pi-agent:latest
        securityContext:
          privileged: true
        volumeMounts:
        - name: dev
          mountPath: /dev
        - name: sys
          mountPath: /sys
      volumes:
      - name: dev
        hostPath:
          path: /dev
      - name: sys
        hostPath:
          path: /sys
```

## Troubleshooting

### CGO Cross-Compilation Issues
If you encounter CGO errors, ensure:
1. Docker buildx is properly configured
2. Cross-compilation toolchains are available in the builder image
3. CGO_ENABLED=1 is set for both SQLite (controller) and hardware access (agent)

### Architecture Mismatch
If images don't run on target architecture:
1. Verify the correct platform was specified in buildx
2. Check that the manifest includes all required architectures
3. Ensure Kubernetes nodes are properly labeled with architecture

### Hardware Access Issues (pi-agent)
For GPIO/I2C access problems:
1. Verify privileged mode is enabled
2. Check device mounts are correct
3. Ensure user has proper permissions for hardware groups