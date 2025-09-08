# CI/CD Image Requirements for Pi-Controller Repository

## Base System Requirements

### Operating System

- **Base**: Ubuntu LTS (20.04 or 22.04)
- **Architecture Support**: Multi-architecture support (linux/amd64, linux/arm64)
- **Package Manager**: apt (Ubuntu)

### Core System Tools

- **Make**: Build automation and task runner
- **Git**: Version control and repository operations
- **curl**: HTTP requests and downloads
- **sudo**: Elevated permissions for package installation
- **bash/sh**: Shell scripting support
- **tar/gzip**: Archive extraction

## Programming Language & Runtime

### Go Environment

- **Go Version**: 1.24.x (as specified in workflows and go.mod)
- **Go Toolchain**: 1.24.6+
- **CGO**: Disabled by default (`CGO_ENABLED=0`)
- **Go Modules**: Full module support
- **Go Cache**: Support for build and module caching (`~/.cache/go-build`, `~/go/pkg/mod`)

### Node.js Environment

- **Node.js Version**: 20.x LTS or 22.x LTS
- **Package Manager**: npm (latest stable) or yarn
- **TypeScript**: ~5.8.3 (as specified in web/package.json)
- **Node Cache**: Support for npm/yarn caching (`~/.npm`, `~/.yarn`)

### Go Tools & Dependencies

```bash
# Core Go tools (auto-installed by workflows)
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Node.js Tools & Dependencies

```bash
# Global tools that may be needed
npm install -g typescript@~5.8.3
npm install -g prettier@^3.6.2
npm install -g eslint@^9.33.0

# Project dependencies (installed via package.json)
# React 19.x, Vite 7.x, Vitest 3.x, etc.
```

## Build & Development Tools

### Protocol Buffers

- **protoc**: Protocol Buffer compiler
- **protoc-gen-go**: Go Protocol Buffer plugin
- **protoc-gen-go-grpc**: Go gRPC plugin

### Code Quality & Linting

#### Go Tools

- **golangci-lint**: v1.54 (comprehensive Go linter)
- **go fmt**: Code formatting
- **go vet**: Static analysis tool

#### Frontend Tools

- **ESLint**: ^9.33.0 (JavaScript/TypeScript linting)
- **Prettier**: ^3.6.2 (Code formatting)
- **TypeScript Compiler**: ~5.8.3 (Type checking and compilation)

### Build Tools

#### Frontend Build System

- **Vite**: ^7.1.2 (Fast build tool and dev server)
- **Vite React Plugin**: ^5.0.0 (React support for Vite)
- **TypeScript**: Project references and composite builds support

## Testing & Security Tools

### Security Scanning Tools

```bash
# Vulnerability scanners
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/google/osv-scanner/cmd/osv-scanner@latest

# SBOM generation
go install github.com/anchore/syft/cmd/syft@latest

# Secrets scanning
# TruffleHog (specific version to avoid update issues)
TRUFFLEHOG_VERSION="3.63.2"

# Static analysis
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
```

### Python Tools (for some security tools)

- **Python 3**: For detect-secrets and other security tools
- **pip**: Python package manager
- **detect-secrets**: Additional secrets detection

## Testing Infrastructure

### Test Types Supported

#### Backend Testing

- **Unit Tests**: Fast, isolated component tests
- **Integration Tests**: Database and API integration
- **Security Tests**: Vulnerability and penetration testing
- **API Tests**: REST API endpoint validation
- **GPIO Tests**: Hardware simulation tests
- **Benchmark Tests**: Performance testing

#### Frontend Testing

- **Unit Tests**: Component and utility function tests
- **Integration Tests**: Component interaction testing
- **UI Tests**: User interface behavior validation
- **E2E Tests**: End-to-end user workflow testing (if implemented)

### Test Coverage

#### Backend Coverage

- **Coverage Tools**: Built-in Go coverage tools
- **Coverage Reports**: Codecov integration
- **Coverage Formats**: `.out` files, JSON, XML

#### Frontend Coverage

- **Coverage Tools**: Vitest coverage (^3.2.4)
- **Test Runner**: Vitest with jsdom environment
- **Testing Libraries**: React Testing Library, Jest DOM
- **Coverage Formats**: HTML, JSON, text reports

### Testing Frameworks & Libraries

#### Frontend Testing Stack

- **Vitest**: ^3.2.4 (Test runner and framework)
- **React Testing Library**: ^16.3.0 (Component testing utilities)
- **Jest DOM**: ^6.8.0 (Custom DOM matchers)
- **User Event**: ^14.6.1 (User interaction simulation)
- **jsdom**: ^26.1.0 (DOM environment for Node.js)

## Build Capabilities

### Multi-Architecture Builds

```yaml
# Supported build targets
- linux/amd64 (primary)
- linux/arm64 (primary) 
- linux/arm/v7 (Raspberry Pi)
- linux/arm/v6 (older Pi models)
- darwin/amd64 (optional)
- darwin/arm64 (optional)
- windows/amd64 (optional)
```

### Build Artifacts

- **Binaries**: pi-controller, pi-agent, pi-web
- **Frontend Assets**: Built React application (dist/ folder)
- **Static Assets**: HTML, CSS, JavaScript bundles
- **SBOM Files**: Software Bill of Materials (JSON/text)
- **Coverage Reports**: Test coverage data (backend and frontend)
- **Security Reports**: Vulnerability scan results

## Performance & Optimization

### Caching Requirements

- **Go Build Cache**: `~/.cache/go-build`
- **Go Module Cache**: `~/go/pkg/mod`
- **Node.js Cache**: `~/.npm` or `~/.yarn/cache`
- **Vite Cache**: `node_modules/.vite`
- **TypeScript Cache**: `.tsbuildinfo` files
- **Dependency Caching**: Efficient module download caching

### Timeout Specifications

- **Validation Jobs**: 10 minutes
- **Test Jobs**: 15 minutes
- **Build Jobs**: 15-20 minutes
- **Security Scans**: 10-15 minutes

## Environment Variables

### Required Environment Variables

```bash
# Go Environment
GO_VERSION="1.24"
GOLANGCI_LINT_VERSION="v1.54"
CGO_ENABLED=0
TRUFFLEHOG_NO_UPDATE=true  # Prevent auto-update issues

# Node.js Environment
NODE_VERSION="20"  # or "22" for latest LTS
NPM_CONFIG_CACHE="/tmp/.npm"
```

### Build Variables

```bash
VERSION # Git tag/describe
COMMIT  # Git commit hash
DATE    # Build timestamp
GOOS    # Target OS
GOARCH  # Target architecture
```

## Integration Requirements

### GitHub Actions Integration

- **checkout@v4**: Repository checkout
- **setup-go@v4**: Go environment setup
- **cache@v3**: Dependency caching
- **upload-artifact@v4**: Artifact management
- **codecov-action@v3**: Coverage reporting

### Hardware Testing Considerations

- **GPIO Simulation**: Mock hardware interfaces
- **Test Pin Configuration**: Configurable GPIO pins (18,19,20,21)
- **Raspberry Pi Compatibility**: ARM64/ARMv7 build support

## Security & Compliance

### Security Tools Integration

- **SAST**: Static Application Security Testing
- **Dependency Scanning**: Vulnerability detection in dependencies
- **Secrets Detection**: Prevent credential leaks
- **SBOM Generation**: Supply chain transparency

### Compliance Features

- **Audit Logging**: Security test results
- **Vulnerability Reports**: JSON/text format outputs
- **Supply Chain Security**: SBOM and dependency tracking

## Resource Requirements

### Minimum Specifications

- **CPU**: 2-4 cores (for parallel builds)
- **Memory**: 4-8 GB RAM
- **Storage**: 20-50 GB (for caches and artifacts)
- **Network**: Reliable internet for dependency downloads

### Optimal Performance

- **CPU**: 8+ cores for matrix builds
- **Memory**: 16+ GB for large projects
- **Storage**: SSD with 100+ GB
- **Network**: High-bandwidth for faster downloads

## Implementation Notes

### Dockerfile Example Structure

```dockerfile
# Multi-stage build approach
FROM golang:1.24-alpine AS builder
# Install system dependencies
# Copy source and build

FROM alpine:3.18
# Install runtime dependencies
# Copy binaries
# Set entrypoint
```

### Key Installation Commands

```bash
# System packages
apk add --no-cache make git curl protobuf-dev

# Go tools (installed as needed)
go install [tool]@latest

# Security tools
curl -sSfL [tool-install-script] | sh
```

### Makefile Integration

The image should support all Makefile targets:

#### Backend Targets

- `make deps` - Dependency installation
- `make proto` - Protocol buffer generation
- `make fmt` - Code formatting
- `make vet` - Static analysis
- `make lint` - Comprehensive linting
- `make test-*` - Various test suites
- `make build` - Binary compilation

#### Frontend Targets (if integrated)

- `make web-deps` - Install Node.js dependencies
- `make web-build` - Build React application
- `make web-test` - Run frontend tests
- `make web-lint` - Frontend linting and formatting
- `make web-dev` - Start development server

### Frontend-Specific Commands

```bash
# Dependency management
npm install  # or yarn install

# Development
npm run dev          # Start Vite dev server
npm run build        # Production build
npm run preview      # Preview production build

# Code quality
npm run lint         # ESLint checking
npm run lint:fix     # ESLint with auto-fix
npm run format       # Prettier formatting
npm run format:check # Prettier validation

# Testing
npm run test         # Run tests with Vitest
npm run test:ui      # Run tests with UI
npm run test:coverage # Generate coverage reports
```

This comprehensive requirements list provides everything needed to create a robust CI/CD image that can handle all aspects of the pi-controller project's build, test, and security workflows.
