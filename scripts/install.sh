#!/usr/bin/env bash
#
# Pi-Controller Quick Install Script
#
# Usage:
#   curl -sSL https://pi-controller.io/install.sh | bash
#
# Options (via environment variables):
#   VERSION=v1.0.0        - Specific version to install (default: latest)
#   INSTALL_DIR=/path     - Installation directory (default: /usr/local/bin or ~/bin)
#   PORTABLE_MODE=true    - Install in portable mode (default: auto-detect)
#   SETUP_SYSTEMD=true    - Setup systemd service (default: true on Linux)
#   GITHUB_REPO=owner/repo - GitHub repository (default: yourusername/pi-controller)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GITHUB_REPO="${GITHUB_REPO:-yourusername/pi-controller}"
VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-}"
PORTABLE_MODE="${PORTABLE_MODE:-auto}"
SETUP_SYSTEMD="${SETUP_SYSTEMD:-auto}"
BINARY_NAME="pi-controller"

# Runtime variables
DETECTED_OS=""
DETECTED_ARCH=""
DOWNLOAD_URL=""
TEMP_DIR=""
NEEDS_SUDO=false

# Utility functions
log() {
    echo -e "${GREEN}[INFO]${NC} $*"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

fatal() {
    error "$*"
    exit 1
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Detect operating system
detect_os() {
    local os
    os="$(uname -s)"

    case "$os" in
        Linux*)
            DETECTED_OS="linux"
            ;;
        Darwin*)
            DETECTED_OS="darwin"
            ;;
        MINGW* | MSYS* | CYGWIN*)
            DETECTED_OS="windows"
            ;;
        *)
            fatal "Unsupported operating system: $os"
            ;;
    esac

    log "Detected OS: $DETECTED_OS"
}

# Detect architecture
detect_arch() {
    local arch
    arch="$(uname -m)"

    case "$arch" in
        x86_64 | amd64)
            DETECTED_ARCH="amd64"
            ;;
        aarch64 | arm64)
            DETECTED_ARCH="arm64"
            ;;
        armv7* | armv6*)
            DETECTED_ARCH="arm"
            ;;
        *)
            fatal "Unsupported architecture: $arch"
            ;;
    esac

    log "Detected architecture: $DETECTED_ARCH"
}

# Determine installation directory
determine_install_dir() {
    if [ -n "$INSTALL_DIR" ]; then
        log "Using specified install directory: $INSTALL_DIR"
        return
    fi

    # Try /usr/local/bin first (requires sudo)
    if [ -w "/usr/local/bin" ]; then
        INSTALL_DIR="/usr/local/bin"
        NEEDS_SUDO=false
    elif command_exists sudo; then
        INSTALL_DIR="/usr/local/bin"
        NEEDS_SUDO=true
        warn "Will need sudo to install to $INSTALL_DIR"
    else
        # Fallback to user's home bin
        INSTALL_DIR="$HOME/bin"
        NEEDS_SUDO=false
        warn "Installing to user directory: $INSTALL_DIR"

        # Add to PATH if not already there
        if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
            warn "Add $INSTALL_DIR to your PATH by adding this to your shell profile:"
            echo "    export PATH=\"\$PATH:$INSTALL_DIR\""
        fi
    fi

    log "Installation directory: $INSTALL_DIR"
}

# Determine if running in portable mode
determine_mode() {
    if [ "$PORTABLE_MODE" != "auto" ]; then
        return
    fi

    # Auto-detect: portable mode on non-Linux or if not a Raspberry Pi
    if [ "$DETECTED_OS" != "linux" ]; then
        PORTABLE_MODE="true"
        log "Auto-detected portable mode (non-Linux system)"
    elif [ ! -f "/proc/device-tree/model" ]; then
        PORTABLE_MODE="true"
        log "Auto-detected portable mode (not a Raspberry Pi)"
    else
        local model
        model="$(cat /proc/device-tree/model 2>/dev/null || echo '')"
        if [[ "$model" =~ "Raspberry Pi" ]]; then
            PORTABLE_MODE="false"
            log "Auto-detected on-device mode (Raspberry Pi detected)"
        else
            PORTABLE_MODE="true"
            log "Auto-detected portable mode"
        fi
    fi
}

# Get latest version from GitHub
get_latest_version() {
    if [ "$VERSION" != "latest" ]; then
        log "Using specified version: $VERSION"
        return
    fi

    log "Fetching latest version from GitHub..."

    if command_exists curl; then
        VERSION=$(curl -sSL "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command_exists wget; then
        VERSION=$(wget -qO- "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        fatal "curl or wget is required"
    fi

    if [ -z "$VERSION" ]; then
        fatal "Failed to fetch latest version"
    fi

    log "Latest version: $VERSION"
}

# Construct download URL
construct_download_url() {
    local binary_file="${BINARY_NAME}-${DETECTED_OS}-${DETECTED_ARCH}"

    # Add .exe extension for Windows
    if [ "$DETECTED_OS" = "windows" ]; then
        binary_file="${binary_file}.exe"
    fi

    DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${binary_file}"
    log "Download URL: $DOWNLOAD_URL"
}

# Download binary
download_binary() {
    TEMP_DIR=$(mktemp -d)
    local temp_binary="$TEMP_DIR/$BINARY_NAME"

    log "Downloading pi-controller..."

    if command_exists curl; then
        curl -sSL -o "$temp_binary" "$DOWNLOAD_URL" || fatal "Download failed"
    elif command_exists wget; then
        wget -qO "$temp_binary" "$DOWNLOAD_URL" || fatal "Download failed"
    else
        fatal "curl or wget is required"
    fi

    # Make executable
    chmod +x "$temp_binary"

    log "Download complete"
}

# Install binary
install_binary() {
    local temp_binary="$TEMP_DIR/$BINARY_NAME"
    local target="$INSTALL_DIR/$BINARY_NAME"

    log "Installing to $target..."

    # Create install directory if needed
    if [ ! -d "$INSTALL_DIR" ]; then
        if [ "$NEEDS_SUDO" = true ]; then
            sudo mkdir -p "$INSTALL_DIR"
        else
            mkdir -p "$INSTALL_DIR"
        fi
    fi

    # Install binary
    if [ "$NEEDS_SUDO" = true ]; then
        sudo cp "$temp_binary" "$target"
        sudo chmod +x "$target"
    else
        cp "$temp_binary" "$target"
        chmod +x "$target"
    fi

    log "Binary installed successfully"
}

# Setup systemd service (Linux only, on-device mode)
setup_systemd() {
    if [ "$DETECTED_OS" != "linux" ]; then
        return
    fi

    if [ "$PORTABLE_MODE" = "true" ]; then
        log "Skipping systemd setup (portable mode)"
        return
    fi

    if [ "$SETUP_SYSTEMD" = "false" ]; then
        log "Skipping systemd setup (disabled)"
        return
    fi

    if ! command_exists systemctl; then
        warn "systemctl not found, skipping systemd setup"
        return
    fi

    log "Setting up systemd service..."

    local service_file="/etc/systemd/system/pi-controller.service"
    local config_dir="/etc/pi-controller"
    local data_dir="/var/lib/pi-controller"

    # Create config and data directories
    sudo mkdir -p "$config_dir"
    sudo mkdir -p "$data_dir"

    # Create systemd service file
    sudo tee "$service_file" >/dev/null <<EOF
[Unit]
Description=Pi-Controller - Raspberry Pi Cluster Management
Documentation=https://github.com/$GITHUB_REPO
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root
ExecStart=$INSTALL_DIR/$BINARY_NAME
Restart=on-failure
RestartSec=5s
LimitNOFILE=65536

# Environment
Environment="PI_CONTROLLER_MODE=on-device"
Environment="PI_CONTROLLER_CONFIG=$config_dir/config.yaml"
Environment="PI_CONTROLLER_DATA_DIR=$data_dir"

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$data_dir
ReadOnlyPaths=$config_dir

[Install]
WantedBy=multi-user.target
EOF

    # Create default config if it doesn't exist
    if [ ! -f "$config_dir/config.yaml" ]; then
        log "Creating default configuration..."
        sudo tee "$config_dir/config.yaml" >/dev/null <<EOF
# Pi-Controller Configuration
app:
  name: "pi-controller"
  environment: "production"
  data_dir: "$data_dir"
  debug: false

database:
  path: "pi-controller.db"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: "5m"

api:
  host: "0.0.0.0"
  port: 8080

grpc:
  host: "0.0.0.0"
  port: 9090

cluster:
  enabled: true
  bind_addr: "0.0.0.0:9091"

log:
  level: "info"
  format: "json"

gpio:
  enabled: true
  mock_mode: false

discovery:
  enabled: true
  method: "mdns"
EOF
    fi

    # Reload systemd
    sudo systemctl daemon-reload

    log "Systemd service created"
    warn "To enable and start the service, run:"
    echo "    sudo systemctl enable --now pi-controller"
    echo "    sudo systemctl status pi-controller"
}

# Cleanup
cleanup() {
    if [ -n "$TEMP_DIR" ] && [ -d "$TEMP_DIR" ]; then
        rm -rf "$TEMP_DIR"
    fi
}

# Main installation flow
main() {
    log "Pi-Controller Installation Script"
    echo ""

    # Cleanup on exit
    trap cleanup EXIT

    # Detection
    detect_os
    detect_arch
    determine_install_dir
    determine_mode

    # Download
    get_latest_version
    construct_download_url
    download_binary

    # Install
    install_binary
    setup_systemd

    # Success message
    echo ""
    log "Installation complete!"
    echo ""

    if [ "$PORTABLE_MODE" = "true" ]; then
        echo -e "${BLUE}Running in PORTABLE MODE${NC}"
        echo "Get started with:"
        echo "  $BINARY_NAME --help"
        echo "  $BINARY_NAME discover --scan"
    else
        echo -e "${BLUE}Running in ON-DEVICE MODE${NC}"
        echo "Enable and start the service:"
        echo "  sudo systemctl enable --now pi-controller"
        echo "  sudo systemctl status pi-controller"
    fi
    echo ""
    echo "For more information, visit:"
    echo "  https://github.com/$GITHUB_REPO"
}

# Run main
main "$@"
