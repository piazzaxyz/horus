#!/usr/bin/env bash
# install.sh - Install QAITOR on Linux/macOS
set -euo pipefail

BINARY_NAME="qaitor"
INSTALL_DIR="${HOME}/.local/bin"
BUILD_DIR="bin"

echo "QAITOR Installer"
echo "================"

# Check for Go
if ! command -v go &>/dev/null; then
    echo "ERROR: Go is not installed. Please install Go 1.22+ from https://go.dev/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "Go version: ${GO_VERSION}"

# Build
echo "Building QAITOR..."
go build -ldflags "-s -w" -o "${BUILD_DIR}/${BINARY_NAME}" .
echo "Build successful: ${BUILD_DIR}/${BINARY_NAME}"

# Create install dir if needed
mkdir -p "${INSTALL_DIR}"

# Copy binary
cp "${BUILD_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

echo ""
echo "Installed to: ${INSTALL_DIR}/${BINARY_NAME}"
echo ""

# Check if dir is in PATH
if [[ ":${PATH}:" != *":${INSTALL_DIR}:"* ]]; then
    echo "NOTE: ${INSTALL_DIR} is not in your PATH."
    echo "Add this to your ~/.bashrc or ~/.zshrc:"
    echo ""
    echo '  export PATH="${HOME}/.local/bin:${PATH}"'
    echo ""
    echo "Then reload your shell: source ~/.bashrc"
else
    echo "Ready! Run 'qaitor' to start."
fi
