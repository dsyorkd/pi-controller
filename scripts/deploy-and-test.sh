#!/bin/bash
# Deploy and Test Script
# ======================
# Wrapper script for running Ansible deployment and tests.
# Can be used locally or in CI.
#
# Usage:
#   ./scripts/deploy-and-test.sh                    # Deploy and test
#   ./scripts/deploy-and-test.sh --reset            # Reset cluster first
#   ./scripts/deploy-and-test.sh --deploy-only      # Skip tests
#   ./scripts/deploy-and-test.sh --local-binary     # Use local build

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
ANSIBLE_DIR="${PROJECT_ROOT}/ansible"

# Defaults
RESET_FIRST=false
RUN_TESTS=true
USE_LOCAL_BINARY=false
INVENTORY_FILE="${ANSIBLE_DIR}/inventory/hosts.yml"
VERBOSE=""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

usage() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS]

Deploy pi-controller to the Pi cluster and run integration tests.

Options:
    --reset             Reset cluster before deploying
    --deploy-only       Skip running tests
    --local-binary      Use locally built binary (build/pi-controller-linux-arm64)
    --inventory FILE    Use custom inventory file
    -v, --verbose       Verbose Ansible output
    -h, --help          Show this help message

Examples:
    # Standard deploy and test
    $(basename "$0")

    # Reset cluster, deploy fresh, and test
    $(basename "$0") --reset --local-binary

    # Just deploy without running tests
    $(basename "$0") --deploy-only --local-binary

Environment Variables:
    PI_SSH_KEY          Path to SSH private key for Pi access
    PI_HOSTS            Comma-separated list of Pi hostnames (optional)
EOF
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --reset)
            RESET_FIRST=true
            shift
            ;;
        --deploy-only)
            RUN_TESTS=false
            shift
            ;;
        --local-binary)
            USE_LOCAL_BINARY=true
            shift
            ;;
        --inventory)
            INVENTORY_FILE="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE="-vvv"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Verify Ansible is installed
if ! command -v ansible-playbook &> /dev/null; then
    log_error "Ansible is not installed. Install with: pip install ansible"
    exit 1
fi

# Verify inventory exists
if [[ ! -f "${INVENTORY_FILE}" ]]; then
    log_error "Inventory file not found: ${INVENTORY_FILE}"
    exit 1
fi

# Build extra vars
EXTRA_VARS=""

if [[ "${USE_LOCAL_BINARY}" == "true" ]]; then
    BINARY_PATH="${PROJECT_ROOT}/build/pi-controller-linux-arm64"
    if [[ ! -f "${BINARY_PATH}" ]]; then
        log_warn "Local binary not found. Building..."
        (cd "${PROJECT_ROOT}" && make build-linux-arm64)
    fi
    EXTRA_VARS="-e pi_controller_local_binary=${BINARY_PATH}"
    log_info "Using local binary: ${BINARY_PATH}"
fi

if [[ "${RUN_TESTS}" == "false" ]]; then
    EXTRA_VARS="${EXTRA_VARS} -e run_tests=false"
fi

cd "${ANSIBLE_DIR}"

# Cleanup function
cleanup() {
    local exit_code=$?
    if [[ $exit_code -ne 0 ]]; then
        log_warn "Deployment failed. Releasing lock..."
        ansible-playbook playbooks/release-lock.yml \
            -i "${INVENTORY_FILE}" ${VERBOSE} || true
    fi
    exit $exit_code
}
trap cleanup EXIT

# Reset cluster if requested
if [[ "${RESET_FIRST}" == "true" ]]; then
    log_info "Resetting cluster..."
    ansible-playbook playbooks/reset-cluster.yml \
        -i "${INVENTORY_FILE}" \
        -e force_reset=true \
        ${VERBOSE}
fi

# Deploy and test
log_info "Deploying pi-controller and running tests..."
ansible-playbook playbooks/deploy-test.yml \
    -i "${INVENTORY_FILE}" \
    "${EXTRA_VARS}" \
    "${VERBOSE}"

log_info "Deployment and testing complete!"
