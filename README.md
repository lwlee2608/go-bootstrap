# Genesis

A TUI tool for bootstrapping new Go projects.

This repo has two parts:

1. **CLI** — creates the project skeleton for a new project
2. **Skills** (`skills/`) — Claude Code skills to apply additional features using `reference/project-00` as the template: `genesis` (server, web, Docker, sqlc, Railway) and `genesis-auth` (tiered authentication)

Typical workflow: start a new project with the CLI, then use the skill to add features.

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/lwlee2608/genesis/main/scripts/install.sh | bash
```

Or build from source:

```bash
make install
```

This will install the binary to `~/.local/bin/genesis`. Make sure `~/.local/bin` is in your PATH.

## Usage

```bash
cd /path/to/new/project
genesis
```
