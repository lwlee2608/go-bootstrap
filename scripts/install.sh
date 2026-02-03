#!/bin/sh
set -e

REPO="lwlee2608/go-bootstrap"
BINARY="genesis"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')

VERSION=$(curl -sI "https://github.com/$REPO/releases/latest" | grep -i "^location:" | sed 's/.*tag\///' | tr -d '\r\n')
if [ -z "$VERSION" ]; then
    echo "Error: Could not determine latest version"
    exit 1
fi

URL="https://github.com/$REPO/releases/download/${VERSION}/${BINARY}_${VERSION#v}_${OS}_${ARCH}.tar.gz"

echo "Downloading $BINARY $VERSION for $OS/$ARCH..."
curl -fsSL "$URL" | tar -xz -C /tmp "$BINARY"

mkdir -p "$INSTALL_DIR"
mv "/tmp/$BINARY" "$INSTALL_DIR/$BINARY"
chmod +x "$INSTALL_DIR/$BINARY"

echo "Installed $BINARY to $INSTALL_DIR/$BINARY"
