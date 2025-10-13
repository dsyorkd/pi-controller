#!/usr/bin/env bash
#
# Pre-commit hooks installation script
# This script installs and configures pre-commit hooks for the pi-controller project
#

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

# Check if running in pi-controller directory
if [[ ! -f ".pre-commit-config.yaml" ]]; then
    error "Error: .pre-commit-config.yaml not found"
    error "Please run this script from the pi-controller root directory"
    exit 1
fi

info "Setting up pre-commit hooks for pi-controller..."
echo

# Check Python version
info "Checking Python version..."
if command -v python3 &> /dev/null; then
    PYTHON_VERSION=$(python3 --version 2>&1 | grep -oE '[0-9]+\.[0-9]+' | head -1)
    PYTHON_MAJOR=$(echo "$PYTHON_VERSION" | cut -d'.' -f1)
    PYTHON_MINOR=$(echo "$PYTHON_VERSION" | cut -d'.' -f2)

    if [[ "$PYTHON_MAJOR" -gt 3 ]] || [[ "$PYTHON_MAJOR" -eq 3 && "$PYTHON_MINOR" -ge 8 ]]; then
        success "Python 3 found: $(python3 --version)"
    else
        error "Python 3.8 or higher is required, found: $(python3 --version)"
        exit 1
    fi
else
    error "Python 3 not found. Please install Python 3.8 or higher."
    exit 1
fi
echo

# Check if pre-commit is already installed
if command -v pre-commit &> /dev/null; then
    success "pre-commit is already installed: $(pre-commit --version)"
else
    info "Installing pre-commit..."

    # Try pip install
    if command -v pip3 &> /dev/null; then
        if pip3 install pre-commit; then
            success "pre-commit installed successfully via pip3"
        else
            error "Failed to install pre-commit via pip3"
            info "You can try installing manually:"
            info "  pip3 install pre-commit"
            info "  or"
            info "  brew install pre-commit (macOS with Homebrew)"
            exit 1
        fi
    else
        error "pip3 not found. Please install pip3 or use Homebrew:"
        info "  brew install pre-commit (macOS with Homebrew)"
        exit 1
    fi
fi
echo

# Install git hooks
info "Installing git hooks..."
if pre-commit install; then
    success "Git hooks installed successfully"
else
    error "Failed to install git hooks"
    exit 1
fi
echo

# Install commit-msg hook (optional, for future use)
info "Installing commit-msg hook..."
if pre-commit install --hook-type commit-msg; then
    success "Commit-msg hook installed"
else
    warning "Failed to install commit-msg hook (optional)"
fi
echo

# Install pre-push hook (optional, for security tests)
info "Installing pre-push hook..."
if pre-commit install --hook-type pre-push; then
    success "Pre-push hook installed"
else
    warning "Failed to install pre-push hook (optional)"
fi
echo

# Optional: Run all hooks on existing files
read -p "$(echo -e "${YELLOW}?${NC} Run pre-commit on all files? (recommended, may take a few minutes) [y/N]: ")" -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    info "Running pre-commit on all files..."
    echo
    if pre-commit run --all-files; then
        success "All checks passed!"
    else
        warning "Some checks failed. This is normal on first run."
        warning "Please review the output and fix any issues."
        warning "You can run 'pre-commit run --all-files' again after fixing."
    fi
else
    info "Skipping initial run. Hooks will run on your next commit."
fi
echo

# Generate detect-secrets baseline if not exists
if [[ ! -f ".secrets.baseline" ]] || [[ ! -s ".secrets.baseline" ]]; then
    info "Generating detect-secrets baseline..."
    if command -v detect-secrets &> /dev/null; then
        if detect-secrets scan --baseline .secrets.baseline; then
            success "Secrets baseline generated"
        else
            warning "Failed to generate secrets baseline (will be generated on first commit)"
        fi
    else
        info "detect-secrets not yet installed (will be installed by pre-commit on first run)"
    fi
fi
echo

# Print next steps
success "Pre-commit hooks setup complete!"
echo
info "Next steps:"
echo "  1. Make changes to your code"
echo "  2. Stage files: git add <files>"
echo "  3. Commit: git commit -m 'your message'"
echo "     The hooks will run automatically before the commit"
echo
info "Useful commands:"
echo "  • Run hooks manually:        pre-commit run --all-files"
echo "  • Run specific hook:         pre-commit run golangci-lint"
echo "  • Skip hooks (use sparingly): SKIP=hook-name git commit"
echo "  • Update hooks:              pre-commit autoupdate"
echo "  • Uninstall hooks:           pre-commit uninstall"
echo
info "For more information, see CONTRIBUTING.md"
echo
success "Happy coding! 🚀"
