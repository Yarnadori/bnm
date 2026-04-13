#!/usr/bin/env bash
set -euo pipefail

REPO="Yarnadori/bnm"
INSTALL_DIR="/usr/local/bin"
BINARY="bnm"

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux)  GOOS="linux"  ;;
  Darwin) GOOS="darwin" ;;
  *)
    echo "Unsupported OS: $OS"
    echo "Please download the binary manually from https://github.com/$REPO/releases"
    exit 1
    ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64 | amd64) GOARCH="amd64" ;;
  arm64 | aarch64) GOARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    echo "Please download the binary manually from https://github.com/$REPO/releases"
    exit 1
    ;;
esac

ASSET_NAME="${BINARY}-${GOOS}-${GOARCH}"

# Determine version
if [ -n "${1:-}" ]; then
  VERSION="$1"
else
  VERSION="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
fi

if [ -z "$VERSION" ]; then
  echo "Failed to determine the latest version."
  exit 1
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$ASSET_NAME"

echo "Installing bnm $VERSION ($GOOS/$GOARCH)..."
echo "Downloading from: $DOWNLOAD_URL"

TMP_FILE="$(mktemp)"
trap 'rm -f "$TMP_FILE"' EXIT

curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE"
chmod +x "$TMP_FILE"

# Install (may require sudo)
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP_FILE" "$INSTALL_DIR/$BINARY"
else
  echo "Installing to $INSTALL_DIR requires elevated permissions."
  sudo mv "$TMP_FILE" "$INSTALL_DIR/$BINARY"
fi

echo "bnm $VERSION installed to $INSTALL_DIR/$BINARY"
echo "Run 'bnm --help' or 'bnm' to get started."
