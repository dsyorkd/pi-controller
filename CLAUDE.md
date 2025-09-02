# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 1. Project Overview

**pi-controller** is a comprehensive Kubernetes management platform designed specifically for Raspberry Pi clusters. The system provides automated discovery, provisioning, and lifecycle management of K3s clusters while offering GPIO-as-a-Service capabilities through Kubernetes Custom Resources (CRDs).

The core philosophy is to provide a simple, single-binary deployment with zero external dependencies, making it ideal for homelabs and small-scale IoT deployments while being robust enough for enterprise use.

## 2. Architecture

The system consists of two main components:

- **Control Plane (`pi-controller`)**: A Go binary that runs on master nodes. It manages cluster state, node discovery, provisioning, and exposes REST, gRPC, and WebSocket APIs. It uses an embedded SQLite database for state persistence.
- **Node Agent (`pi-agent`)**: A lightweight Go binary that runs as a DaemonSet on every node in the cluster. It provides hardware access (GPIO, I2C, etc.), system monitoring, and communicates with the Control Plane via gRPC.

For more details, refer to `ARCHITECTURE.md`.

## 3. Key Technologies

- **Backend**: Go (Golang)
- **Kubernetes Distribution**: K3s
- **API**: REST, gRPC, WebSockets
- **Database**: SQLite (embedded)
- **Hardware Control**: Kubernetes CRDs (`GPIOPin`, `PWMController`, etc.)
- **Build System**: Make

## 4. Development Workflow

The project uses a `Makefile.mk` to streamline common development tasks.

### Setup

- Install dependencies: `make deps`
- Generate protobuf code (if `proto` files change): `make proto`

### Building

- Build all binaries for your local OS/Arch: `make build`
- Build for a specific target (e.g., Raspberry Pi): `make build-linux-arm64`
- Build for all supported platforms: `make build-all`

### Running Locally

- Run the main control plane: `make run-controller`
- Run the node agent: `make run-agent`

### Testing

- Run all tests (unit, integration, security): `make test-all`
- Run only unit tests: `make test-unit`
- Run security-focused tests: `make test-security`
- Generate a test coverage report: `make test-coverage`

### Code Quality

- Format code: `make fmt`
- Run linter (`golangci-lint`): `make lint`
- Run `go vet`: `make vet`

## 5. Testing Strategy

The project has a comprehensive testing framework. Refer to `TESTING_FRAMEWORK_SUMMARY.md` for full details.

- **Unit Tests**: Located alongside the source code (`*_test.go`).
- **Integration Tests**: `test/integration/`
- **Security Tests**: `test/security/`
- **Key Goal**: Maintain high test coverage (target >80%) and ensure all new features include relevant tests. All security vulnerabilities must have a corresponding test case.

## 6. Coding Style & Conventions

- **Formatting**: Code must be formatted with `gofmt`. Run `make fmt` before committing.
- **Linting**: Code must pass `golangci-lint` checks as defined in `.golangci.yml`. Run `make lint` to check.
- **Error Handling**: Errors should be handled explicitly. Do not discard errors with `_`. Use the `errors` package for wrapping and adding context.
- **Comments**: Public functions and structs should have clear, concise comments explaining their purpose.

## 7. Project Rules & Contribution Guidelines

To ensure consistency and quality, all contributions must adhere to the following rules.

### 7.1. Issue Tracking

All work, including new features, bug fixes, and chores, must be tracked in **GitHub Issues**. Before starting any work, please ensure an issue exists. If not, create one detailing the task.

### 7.2. Branching Strategy (GitFlow)

We follow the GitFlow branching model to manage our development lifecycle.

- **`master`**: This branch contains production-ready code. Direct commits are forbidden. Merges only happen from `release` or `hotfix` branches.
- **`develop`**: This is the main development branch where all completed features are merged. All feature branches should be created from `develop`.
- **`feature/<issue-id>-short-description`**: For new feature development. Branched from `develop` and merged back into `develop`.
  - *Example*: `feature/PI-42-add-node-discovery`
- **`bugfix/<issue-id>-short-description`**: For non-critical bug fixes. Branched from `develop` and merged back into `develop`.
  - *Example*: `bugfix/PI-55-fix-api-panic`
- **`release/<version>`**: Used to prepare for a new production release. Branched from `develop`. Allows for final bug fixes and preparation before merging into `master`.
  - *Example*: `release/v1.1.0`
- **`hotfix/<version>`**: For critical production bugs that must be fixed immediately. Branched from `master` and merged back into both `master` and `develop`.
  - *Example*: `hotfix/v1.0.1`

### 7.3. Commit Messages (Conventional Commits)

We use the Conventional Commits specification for our commit messages. This helps with automated changelog generation and makes the project history easier to read.

The format is: `<type>(<scope>): <subject>`

- **`<type>`**: Must be one of `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `build`, `ci`.
- **`<scope>`** (optional): The part of the codebase affected (e.g., `api`, `provisioner`, `gpio`).
- **`<subject>`**: A concise description of the change.

**Examples:**

```
feat(api): add endpoint for node registration
fix(provisioner): correct ssh connection timeout issue
docs(readme): update installation instructions
test(gpio): add unit tests for pin validation
```

### 7.4. Pull Requests (PRs)

All code changes must be submitted via a Pull Request.

1. **Target Branch**: PRs for features and bugfixes should target the `develop` branch.
2. **Title Format**: The PR title must be clear and follow this format:
   `[<Issue-ID>] <Type>(<scope>): <Description>`
  - *Example*: `[PI-42] feat(discovery): implement mDNS node discovery service`
3. **Description**: The PR description is critical and must clearly explain:
  - **What** was changed.
  - **Why** this change is necessary (linking to the business value or bug).
  - **How** the change was implemented and any architectural decisions made.
4. **CI Checks**: All automated CI checks (linting, tests, build) must pass. A PR cannot be merged if any checks are failing.
5. **Code Review**:
  - At least **one** approval from another developer is required before merging.
  - Address all review comments before requesting a final review or merge.

## Task Master AI Instructions

**Import Task Master's development workflow commands and guidelines, treat as if import is in the main CLAUDE.md file.**
@./.taskmaster/CLAUDE.md
