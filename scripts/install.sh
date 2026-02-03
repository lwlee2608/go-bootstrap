#!/bin/sh
set -e

REPO="lwlee2608/go-bootstrap"
BINARY="genesis"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *) echo "unsupported" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *) echo "unsupported" ;;
    esac
}

OS=$(detect_os)
ARCH=$(detect_arch)

if [ "$OS" = "unsupported" ] || [ "$ARCH" = "unsupported" ]; then
    echo "Error: Unsupported OS or architecture"
    exit 1
fi

# Check for required tools
if ! command -v go >/dev/null 2>&1; then
    echo "Error: Go is not installed. Please install Go 1.23 or later."
    exit 1
fi

VERSION=$(curl -sI "https://github.com/$REPO/releases/latest" | grep -i "^location:" | sed 's/.*tag\///' | tr -d '\r\n')
if [ -z "$VERSION" ]; then
    echo "Error: Could not determine latest version"
    exit 1
fi

echo "Installing $BINARY $VERSION for $OS/$ARCH..."

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# Download source tarball
echo "Downloading source code..."
curl -sL "https://github.com/$REPO/tarball/${VERSION}" -o "$TMPDIR/source.tar.gz"

# Extract
tar -xzf "$TMPDIR/source.tar.gz" -C "$TMPDIR"

# Find the extracted directory (format: username-repo-shortsha)
SRCDIR=$(find "$TMPDIR" -maxdepth 1 -type d -name "*-${REPO#*/}-*" | head -1)

if [ -z "$SRCDIR" ]; then
    echo "Error: Could not find extracted source directory"
    exit 1
fi

# Build
echo "Building $BINARY..."
cd "$SRCDIR"
go build -ldflags "-X main.AppVersion=${VERSION}" -o "$TMPDIR/$BINARY" ./cmd/genesis

# Install
mkdir -p "$INSTALL_DIR"
cp "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
chmod +x "$INSTALL_DIR/$BINARY"

echo "Installed $BINARY to $INSTALL_DIR/$BINARY"

if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo "Add $INSTALL_DIR to your PATH to use $BINARY"
fi
