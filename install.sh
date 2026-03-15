#!/bin/sh
set -e

REPO="ngavilan-dogfy/woffuk-cli"
INSTALL_DIR="/usr/local/bin"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  darwin|linux) ;;
  *)            echo "Unsupported OS: $OS"; exit 1 ;;
esac

BINARY="woffuk-${OS}-${ARCH}"

# Get latest release tag
echo "Finding latest release..."
TAG=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -1 | cut -d'"' -f4)

if [ -z "$TAG" ]; then
  echo "Error: could not find latest release"
  exit 1
fi

URL="https://github.com/${REPO}/releases/download/${TAG}/${BINARY}"

echo "Downloading woffuk ${TAG} for ${OS}/${ARCH}..."
curl -sL "$URL" -o /tmp/woffuk

chmod +x /tmp/woffuk

echo "Installing to ${INSTALL_DIR}/woffuk (may need sudo)..."
if [ -w "$INSTALL_DIR" ]; then
  mv /tmp/woffuk "${INSTALL_DIR}/woffuk"
else
  sudo mv /tmp/woffuk "${INSTALL_DIR}/woffuk"
fi

echo ""
echo "✓ woffuk ${TAG} installed successfully!"
echo ""
echo "Next steps:"
echo "  1. Install gh CLI: https://cli.github.com"
echo "  2. Login to GitHub: gh auth login"
echo "  3. Run the setup:   woffuk setup"
echo ""
