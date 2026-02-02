# Development Guide

This guide covers setting up a development environment for pi-controller.

## Prerequisites

- **Go** 1.21 or later
- **Make** - Build automation
- **Git** - Version control
- **Docker** (optional) - For container builds
- **protoc** (optional) - Protocol buffer compiler (auto-installed if needed)

## Clone Repository

```bash
git clone https://github.com/yourusername/pi-controller.git
cd pi-controller
```

## Install Dependencies

Install Go dependencies and generate protobuf code:

```bash
make deps
```

This will:

- Download Go modules
- Generate protobuf code (if protoc is available)
- Tidy up dependencies

## Build

### Local Platform

Build for your current OS/architecture:

```bash
make build
```

Binaries are placed in `build/`:

- `build/pi-controller-<os>-<arch>`

### Cross-Compilation

Build for specific platforms:

```bash
make build-linux-arm64    # Raspberry Pi 3/4/5
make build-linux-amd64    # Linux x86_64
make build-darwin-arm64   # macOS Apple Silicon
make build-darwin-amd64   # macOS Intel
```

Build for all platforms:

```bash
make build-all
```

## Run Locally

### Development Mode

Start the controller with debug logging:

```bash
make run-controller
```

Or run directly:

```bash
go run ./cmd/pi-controller --log-level=debug --log-format=text
```

### Custom Configuration

Create a local config file:

```bash
cp config/config.example.yaml config/config.yaml
```

Edit `config/config.yaml` for your setup, then run:

```bash
go run ./cmd/pi-controller --config=config/config.yaml
```

## Testing

### All Tests

Run the complete test suite:

```bash
make test-all
```

This runs:

- Unit tests
- Integration tests
- Security tests
- GPIO tests
- API tests

### Specific Test Suites

```bash
make test-unit          # Unit tests only
make test-integration   # Integration tests
make test-security      # Security vulnerability tests
make test-gpio          # GPIO hardware simulation tests
make test-api           # API endpoint tests
```

### Coverage Reports

Generate HTML coverage report:

```bash
make test-coverage
```

View the report at `coverage.html`.

Check coverage threshold (80%):

```bash
make test-coverage-threshold
```

### Benchmarks

Run performance benchmarks:

```bash
make test-benchmarks
```

### Race Detection

Run tests with race detector:

```bash
make test-race
```

## Code Quality

### Format Code

Format all Go code:

```bash
make fmt
```

### Linting

Run golangci-lint:

```bash
make lint
```

Install golangci-lint if needed:

```bash
make install-lint
```

### Static Analysis

Run go vet:

```bash
make vet
```

## Docker

### Build Docker Images

Build for your platform:

```bash
make docker
```

### Multi-Architecture Builds

Set up Docker buildx:

```bash
make docker-buildx-setup
```

Build for multiple architectures:

```bash
make docker-multiarch
```

Test multi-arch builds without pushing:

```bash
make docker-multiarch-test
```

## Development Workflow

### 1. Create Feature Branch

Follow GitFlow branching model:

```bash
# Feature branch
git checkout -b feature/PI-123-add-node-discovery develop

# Bugfix branch
git checkout -b bugfix/PI-456-fix-api-panic develop
```

### 2. Make Changes

- Write code following Go best practices
- Add tests for new functionality
- Update documentation as needed

### 3. Test Your Changes

```bash
# Run all tests
make test-all

# Check code quality
make lint
make vet

# Format code
make fmt
```

### 4. Commit Changes

Use conventional commit messages:

```bash
git commit -m "feat(api): add endpoint for node registration"
git commit -m "fix(provisioner): correct ssh connection timeout"
git commit -m "docs(readme): update installation instructions"
```

### 5. Push and Create PR

```bash
git push origin feature/PI-123-add-node-discovery
```

Create a pull request on GitHub targeting `develop` branch.

## Useful Make Targets

View all available targets:

```bash
make help
```

Common targets:

- `make build` - Build for local platform
- `make test` - Run tests with coverage
- `make clean` - Remove build artifacts
- `make docker` - Build Docker images
- `make release` - Create release artifacts
- `make version` - Show version information
- `make env` - Show build environment

## Database Migrations

### Create Migration

Migrations are handled automatically, but you can test them:

```bash
make db-migrate-up      # Apply pending migrations
make db-migrate-down    # Rollback last migration
make db-migrate-status  # Show migration status
```

### Reset Database

**Warning: This destroys all data**

```bash
make db-reset
```

## Protocol Buffers

If you modify `.proto` files, regenerate code:

```bash
make proto
```

Install protoc tools if needed:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## Debugging

### Enable Debug Logging

Set log level to debug:

```bash
PI_CONTROLLER_LOG_LEVEL=debug go run ./cmd/pi-controller
```

### Use Delve Debugger

Install Delve:

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

Debug the application:

```bash
dlv debug ./cmd/pi-controller -- --log-level=debug
```

## Project Structure

```
pi-controller/
├── cmd/
│   └── pi-controller/    # Main application entry point
├── internal/             # Private application code
│   ├── api/              # API handlers (REST, gRPC, WebSocket)
│   ├── provisioner/      # Cluster provisioning logic
│   ├── services/         # Business logic services
│   └── migrations/       # Database migrations
├── pkg/                  # Public libraries
│   └── gpio/             # GPIO control library
├── proto/                # Protocol buffer definitions
├── config/               # Configuration files
├── docs/                 # Documentation
├── scripts/              # Build and utility scripts
└── test/                 # Test suites
    ├── integration/      # Integration tests
    └── security/         # Security tests
```

## Contributing

Read the [Contributing Guide](../CONTRIBUTING.md) for:

- GitFlow branching strategy
- Commit message conventions
- Pull request process
- Code review guidelines

## Common Issues

**Protobuf generation fails:**

- Install protoc: `brew install protobuf` (macOS) or download from protobuf releases
- Ensure protoc-gen-go and protoc-gen-go-grpc are installed

**Tests fail on macOS:**

- Some GPIO tests may fail on non-Linux systems (expected)
- Run with `-short` flag to skip hardware tests: `go test -short ./...`

**Linter errors:**

- Run `make fmt` to auto-format code
- Check `.golangci.yml` for linter configuration

## Next Steps

- Read [Architecture Overview](../architecture/index.md)
- Explore [API Documentation](../reference/rest-api.md)
- Review [Testing Framework](../TESTING_FRAMEWORK_SUMMARY.md)
- Check out [Example Projects](examples/)

## Resources

- **Go Documentation**: [golang.org/doc](https://golang.org/doc)
- **Protocol Buffers**: [protobuf.dev](https://protobuf.dev)
- **Raft Consensus**: [raft.github.io](https://raft.github.io)
- **K3s Documentation**: [k3s.io](https://k3s.io)
