#!/bin/bash

# Pocket Prompt Installation Script
# This script installs pocket-prompt CLI tool

set -e

REPO="dpshade/pocket-prompt"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="pocket-prompt"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Convert architecture names
case $ARCH in
  x86_64)
    ARCH="amd64"
    ;;
  arm64|aarch64)
    ARCH="arm64"
    ;;
  armv7l)
    ARCH="armv7"
    ;;
  *)
    echo -e "${RED}Unsupported architecture: $ARCH${NC}"
    exit 1
    ;;
esac

echo -e "${GREEN}🚀 Installing Pocket Prompt CLI${NC}"
echo "Detected OS: $OS"
echo "Detected Architecture: $ARCH"

# Check if Go is installed for fallback
if command -v go &> /dev/null; then
    echo -e "${YELLOW}Go detected. You can also install with: go install github.com/$REPO@latest${NC}"
fi

# Try to install via Go first (recommended)
if command -v go &> /dev/null; then
    echo -e "${GREEN}Installing via Go...${NC}"
    go install github.com/$REPO@latest
    
    # Check if $GOPATH/bin or $GOBIN is in PATH
    GOBIN_PATH=$(go env GOPATH)/bin
    if [[ ":$PATH:" != *":$GOBIN_PATH:"* ]]; then
        echo -e "${YELLOW}⚠️  Note: Add $GOBIN_PATH to your PATH to use pocket-prompt from anywhere${NC}"
        echo "Add this line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        echo "export PATH=\$PATH:$GOBIN_PATH"
    fi
    
    echo -e "${GREEN}✅ Installation complete!${NC}"
    echo "Run 'pocket-prompt --init' to get started"
    exit 0
fi

# Fallback: try to download from releases (if releases exist)
echo -e "${YELLOW}Go not found. Checking for pre-built releases...${NC}"

# Get latest release info
RELEASE_URL="https://api.github.com/repos/$REPO/releases/latest"
DOWNLOAD_URL=$(curl -s $RELEASE_URL | grep "browser_download_url.*${OS}_${ARCH}" | cut -d '"' -f 4)

if [ -z "$DOWNLOAD_URL" ]; then
    echo -e "${RED}❌ No pre-built binary found for $OS/$ARCH${NC}"
    echo -e "${YELLOW}Please install Go and run: go install github.com/$REPO@latest${NC}"
    exit 1
fi

echo "Downloading from: $DOWNLOAD_URL"

# Create temporary directory
TMP_DIR=$(mktemp -d)
cd $TMP_DIR

# Download and extract
curl -L -o "$BINARY_NAME.tar.gz" "$DOWNLOAD_URL"
tar -xzf "$BINARY_NAME.tar.gz"

# Install binary
if [ ! -d "$INSTALL_DIR" ]; then
    echo -e "${YELLOW}Creating $INSTALL_DIR directory...${NC}"
    sudo mkdir -p "$INSTALL_DIR"
fi

echo "Installing to $INSTALL_DIR/$BINARY_NAME..."
sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"

# Cleanup
cd - > /dev/null
rm -rf $TMP_DIR

echo -e "${GREEN}✅ Installation complete!${NC}"
echo "Run 'pocket-prompt --init' to get started"