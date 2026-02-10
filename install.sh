#!/bin/sh
set -e

REPO="ethanrcohen/ddcli"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  darwin) OS="darwin" ;;
  linux)  OS="linux" ;;
  *)      echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)       echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Get latest release tag
TAG=$(curl -sI "https://github.com/$REPO/releases/latest" | grep -i "^location:" | sed 's|.*/tag/||' | tr -d '\r\n')
if [ -z "$TAG" ]; then
  echo "Failed to determine latest release" >&2
  exit 1
fi

URL="https://github.com/$REPO/releases/download/$TAG/ddcli_${OS}_${ARCH}.tar.gz"

echo "Downloading ddcli $TAG ($OS/$ARCH)..."
TMPDIR=$(mktemp -d)
curl -sL "$URL" | tar xz -C "$TMPDIR"

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMPDIR/ddcli" "$INSTALL_DIR/ddcli"
else
  echo "Installing to $INSTALL_DIR (requires sudo)..."
  sudo mv "$TMPDIR/ddcli" "$INSTALL_DIR/ddcli"
fi

rm -rf "$TMPDIR"
echo "Installed ddcli $TAG to $INSTALL_DIR/ddcli"
