# Getting Started with Development

This guide will walk you through setting up your local development environment for Pi-Controller.

## Prerequisites

Before you begin, ensure you have the following tools installed on your system:

*   **Go**: Version 1.21 or later.
*   **Make**: A standard `make` utility.
*   **Docker**: For building and running containerized versions of the application.
*   **Git**: For version control.

## 1. Clone the Repository

Start by cloning the project repository to your local machine:

```sh
git clone https://github.com/dsyorkd/pi-controller.git
cd pi-controller
```

## 2. Install Dependencies

The project uses Go Modules to manage dependencies. You can install them using the provided Makefile target:

```sh
make deps
```
This command will download and tidy the Go modules required for the project.

## 3. Build the Binaries

You can build the `pi-controller` and `pi-agent` binaries for your local system architecture with a single command:

```sh
make build
```
The compiled binaries will be placed in the `build/` directory.

## 4. Run the Controller

To run the main control plane locally for development, use the `run-controller` target:

```sh
make run-controller
```
This will start the `pi-controller` server. By default, it will look for a configuration file and use development settings.

## 5. Run Tests

To ensure everything is working correctly, run the test suite:

```sh
make test
```
This will execute all unit and integration tests for the project.

## Useful Makefile Targets

The `Makefile.mk` contains many helpful targets for development:

*   `make help`: Shows a list of all available targets and their descriptions.
*   `make build-all`: Cross-compiles the binaries for all supported platforms (Linux, macOS, Windows).
*   `make lint`: Runs the `golangci-lint` linter to check for code quality issues.
*   `make test-coverage`: Runs tests and generates an HTML coverage report.
*   `make docker`: Builds the Docker images for the controller and agent.
*   `make clean`: Removes build artifacts.