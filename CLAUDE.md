# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
make build      # Compile to bin/genesis
make test       # Run all tests (go test -v ./...)
make run        # Run without building
make install    # Build and install to ~/.local/bin/genesis
make clean      # Remove build artifacts and test cache
```

Run a single test:
```bash
go test -v ./internal/scaffold -run TestGenerate
```

Version is injected at compile time via `-ldflags "-X main.AppVersion=$(VERSION)"`.

## Architecture

Genesis is a TUI tool for bootstrapping new Go projects. The flow is:

```
cmd/genesis/main.go
    ├─> internal/git      # Auto-detect app name (cwd) and module (git remote)
    ├─> internal/tui      # BubbleTea prompts for user confirmation
    └─> internal/scaffold # Generate files from embedded templates
```

**Key packages:**
- `internal/git` - Parses git remote URL (SSH/HTTPS) to suggest module name
- `internal/tui` - BubbleTea state machine: inputAppName → inputModuleName → done
- `internal/scaffold` - Renders `templates/*.tmpl` files using Go's text/template

Templates are embedded at compile time via `//go:embed templates/*` in scaffold.go.

## Testing

- Unit tests use table-driven patterns and `t.TempDir()` for isolation
- Integration test in `scaffold_integration_test.go` generates a project, runs `make build`, and verifies the binary executes correctly
