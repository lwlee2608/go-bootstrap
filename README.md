# Genesis

A TUI tool for bootstrapping new Go projects.

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/lwlee2608/go-bootstrap/main/scripts/install.sh | bash
```

Or build from source:

```bash
make install
```

This will install the binary to `~/.local/bin/genesis`. Make sure `~/.local/bin` is in your PATH.

## Usage

```bash
./bin/genesis -output /path/to/new/project
```

The TUI will prompt for:

1. **App name** - Used for `cmd/{appname}/main.go` and Makefile
2. **Go module name** - e.g., `github.com/user/myapp`

### Flags

| Flag       | Description                            | Default |
| ---------- | -------------------------------------- | ------- |
| `-output`  | Output directory for generated project | `.`     |
| `-version` | Print version                          |         |

## Example

```bash
# Install genesis
make install

# Generate a new project
genesis -output ~/projects/myapp

# Build and run the generated project
cd ~/projects/myapp
make build
./bin/myapp
```
