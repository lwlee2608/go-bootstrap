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

VERSION=$(curl -sI "https://github.com/$REPO/releases/latest" | grep -i "^location:" | sed 's/.*tag\///' | tr -d '\r\n')
if [ -z "$VERSION" ]; then
    echo "Error: Could not determine latest version"
    exit 1
fi

EXT="tar.gz"
[ "$OS" = "windows" ] && EXT="zip"

URL="https://github.com/$REPO/releases/download/${VERSION}/${BINARY}_${VERSION#v}_${OS}_${ARCH}.${EXT}"

echo "Downloading $BINARY $VERSION for $OS/$ARCH..."

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -sL "$URL" -o "$TMPDIR/archive.$EXT"

if [ "$EXT" = "zip" ]; then
    unzip -q "$TMPDIR/archive.$EXT" -d "$TMPDIR"
else
    tar -xzf "$TMPDIR/archive.$EXT" -C "$TMPDIR"
fi

mkdir -p "$INSTALL_DIR"
cp "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
chmod +x "$INSTALL_DIR/$BINARY"

echo "Installed $BINARY to $INSTALL_DIR/$BINARY"

if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo "Add $INSTALL_DIR to your PATH to use $BINARY"
fi
