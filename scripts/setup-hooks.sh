#!/bin/bash
# Setup script for git hooks in goimg-datalayer
#
# This script installs the pre-commit hook to enforce linting.
# Run this script once after cloning the repository.
#
# Usage: ./scripts/setup-hooks.sh
#        make install-hooks

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
HOOKS_DIR="$PROJECT_ROOT/.git/hooks"

echo "==> Setting up git hooks for goimg-datalayer..."
echo ""

# Ensure we're in a git repository
if [ ! -d "$PROJECT_ROOT/.git" ]; then
    echo "ERROR: Not a git repository. Run this script from the project root."
    exit 1
fi

# Create hooks directory if it doesn't exist
mkdir -p "$HOOKS_DIR"

# Install pre-commit hook
echo "Installing pre-commit hook..."
cp "$SCRIPT_DIR/pre-commit" "$HOOKS_DIR/pre-commit"
chmod +x "$HOOKS_DIR/pre-commit"
echo "  -> Installed: .git/hooks/pre-commit"

# Check if pre-commit framework is available
if command -v pre-commit &> /dev/null; then
    echo ""
    echo "==> pre-commit framework detected, installing hooks..."
    cd "$PROJECT_ROOT"
    pre-commit install
    pre-commit install --hook-type commit-msg
    echo "  -> pre-commit framework hooks installed"
else
    echo ""
    echo "NOTE: pre-commit framework not installed."
    echo "For full hook support, install it with:"
    echo "  pip install pre-commit"
    echo "  pre-commit install"
    echo ""
    echo "The basic pre-commit hook has been installed and will run go fmt, go vet,"
    echo "and golangci-lint on staged files."
fi

# Verify golangci-lint is available
if ! command -v golangci-lint &> /dev/null; then
    echo ""
    echo "WARNING: golangci-lint not found."
    echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
fi

echo ""
echo "==> Git hooks setup complete!"
echo ""
echo "The following checks will run before each commit:"
echo "  - go fmt (code formatting)"
echo "  - go vet (static analysis)"
echo "  - golangci-lint (comprehensive linting)"
echo ""
echo "To bypass hooks in emergencies: git commit --no-verify"
echo "(Use sparingly and only when absolutely necessary)"
