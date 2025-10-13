# Contributing to Pi-Controller

Thank you for your interest in contributing to pi-controller! This document provides guidelines and setup instructions for local development.

## Table of Contents

- [Development Setup](#development-setup)
- [Pre-commit Hooks](#pre-commit-hooks)
- [Running Tests](#running-tests)
- [Building](#building)
- [Code Standards](#code-standards)
- [Security Guidelines](#security-guidelines)
- [Pull Request Process](#pull-request-process)

## Development Setup

### Prerequisites

- Go 1.25.1 or later
- Docker (for building images)
- Make
- Protocol Buffers compiler (`protoc`)
- Python 3.8+ (for pre-commit hooks)

### Initial Setup

1. Clone the repository:

```bash
git clone https://github.com/dsyorkd/pi-controller.git
cd pi-controller
```

2. Install dependencies:

```bash
make deps
```

3. Generate protobuf files:

```bash
make proto
```

4. Build the project:

```bash
make build
```

## Pre-commit Hooks

We use pre-commit hooks to catch issues early and reduce CI runner load. This runs linting, formatting, and basic security checks locally before you commit.

### Installation

1. Install pre-commit framework:

```bash
# Using pip
pip install pre-commit

# Or using homebrew (macOS)
brew install pre-commit
```

2. Install the git hooks:

```bash
pre-commit install
```

3. (Optional) Run hooks on all files to verify setup:

```bash
pre-commit run --all-files
```

### What Gets Checked

The pre-commit hooks run the following checks:

#### File Checks

- Trailing whitespace removal
- End-of-file fixer
- YAML syntax validation
- Large file prevention
- Merge conflict detection
- Private key detection

#### Go Code

- `go fmt` - Code formatting
- `go-imports` - Import organization
- `go-mod-tidy` - Dependency cleanup
- `go-vet` - Static analysis
- `go-build` - Compilation check
- `golangci-lint` - Comprehensive linting (see `.golangci.yml`)
- `gosec` - Security scanning

#### Security

- `detect-secrets` - Secret detection (AWS keys, tokens, etc.)
- Custom forbidden pattern check (blocks `fmt.Print`, `panic`, `TODO SECURITY`)

#### Documentation & Configuration

- `markdownlint` - Markdown linting
- `yamllint` - YAML linting (GitHub Actions workflows)
- `hadolint` - Dockerfile linting
- `shellcheck` - Shell script analysis

#### Protocol Buffers

- `buf lint` - Protobuf linting
- Auto-generation of protobuf files

#### Custom Hooks

- **go-test-changed**: Runs tests only for modified Go packages
- **check-forbidden-patterns**: Blocks problematic code patterns
- **proto-generated**: Auto-generates protobuf files before commit
- **security-tests** (manual): Run full security test suite

### Skipping Hooks

Sometimes you may need to skip hooks (use sparingly):

```bash
# Skip all hooks
SKIP=pre-commit git commit -m "message"

# Skip specific hook
SKIP=golangci-lint git commit -m "message"

# Skip multiple hooks
SKIP=golangci-lint,gosec git commit -m "message"
```

### Running Specific Hooks

```bash
# Run a specific hook
pre-commit run golangci-lint --all-files

# Run only on staged files
pre-commit run

# Update hook repositories
pre-commit autoupdate
```

### Troubleshooting

#### Hooks are slow

- The first run will be slower as it downloads and caches tools
- Subsequent runs are much faster
- Use `SKIP=` for quick commits during development (but CI will still check)

#### golangci-lint failures

- Check `.golangci.yml` for configuration
- Run `golangci-lint run` locally to see detailed output
- Some checks can be disabled for specific lines with `//nolint:linter-name`

#### detect-secrets failures

- False positives can be added to `.secrets.baseline`
- Run `detect-secrets scan --baseline .secrets.baseline` to update baseline
- Audit baseline regularly to ensure no real secrets are ignored

#### Protobuf generation fails

- Ensure `protoc` is installed: `brew install protobuf`
- Run `make proto` manually to verify

## Running Tests

### Unit Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests for specific package
go test ./internal/gpio/...
```

### Security Tests

```bash
# Run full security test suite
make test-security-verbose
```

### Integration Tests

```bash
# Run integration tests (requires cluster)
make test-integration
```

## Building

### Local Build

```bash
# Build binary
make build

# Build with race detector
make build-race
```

### Docker Build

```bash
# Build Docker image
make docker-build

# Build multi-arch images (requires buildx)
make docker-buildx
```

## Code Standards

### Go Code

1. **Formatting**: Use `gofmt` and `goimports`
2. **Imports**: Organize with standard, third-party, then local
3. **Error Handling**: Always check errors, use meaningful error messages
4. **Context**: Pass `context.Context` as first parameter for I/O operations
5. **Logging**: Use structured logging (no `fmt.Print` in production code)
6. **Testing**: Write tests for new functionality, maintain >80% coverage

### Commit Messages

Follow conventional commits:

```
feat: add GPIO monitoring endpoint
fix: resolve race condition in controller loop
docs: update API documentation
test: add unit tests for GPIO handler
refactor: simplify certificate rotation logic
chore: update dependencies
```

### Code Comments

- Add package-level comments for exported packages
- Document exported functions, types, and constants
- Use `//nolint:linter-name` with justification for linter exceptions
- Keep comments up-to-date with code changes

## Security Guidelines

### Secrets Management

- **Never** commit secrets, tokens, or private keys
- Use environment variables or secret management systems
- Add test fixtures to `.secrets.baseline` if needed
- Rotate secrets immediately if accidentally committed

### Dependencies

- Run `go mod tidy` regularly
- Update dependencies with security patches promptly
- Review dependency licenses (no GPL, AGPL, or commercial licenses)

### TLS/Certificates

- Use strong cipher suites (TLS 1.2+)
- Validate certificates in production
- Rotate certificates regularly
- Use proper key sizes (RSA 2048+, ECDSA P-256+)

### Input Validation

- Validate all external inputs
- Use prepared statements for database queries
- Sanitize user-provided data
- Implement rate limiting for APIs

## Pull Request Process

1. **Create a feature branch** from `develop`:

   ```bash
   git checkout -b feature/my-feature develop
   ```

2. **Make your changes**:
   - Follow code standards
   - Add tests for new functionality
   - Update documentation

3. **Run pre-commit checks**:

   ```bash
   pre-commit run --all-files
   ```

4. **Commit your changes**:

   ```bash
   git add .
   git commit -m "feat: add my feature"
   ```

5. **Push and create PR**:

   ```bash
   git push origin feature/my-feature
   ```

   - Open PR from your branch to `develop`
   - Fill out PR template
   - Link related issues

6. **CI Checks**:
   - All workflows must pass (build, test, security)
   - Address any linting or security findings
   - Maintain or improve code coverage

7. **Code Review**:
   - Address reviewer feedback
   - Keep commits clean and logical
   - Squash commits if requested

8. **Merge**:
   - PRs are merged by maintainers
   - Delete feature branch after merge

## Getting Help

- **Issues**: Open a GitHub issue for bugs or feature requests
- **Discussions**: Use GitHub discussions for questions
- **Security**: Report security issues privately to maintainers

## License

By contributing, you agree that your contributions will be licensed under the project's license.
