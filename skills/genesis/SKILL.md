---
name: genesis
description: Use when scaffolding a full-stack project — a Go backend, a React/Vite web frontend, Dockerfiles, docker-compose, a Makefile, sqlc. Refers to reference/project-00 in github.com/lwlee2608/genesis as the canonical template.
user-invocable: true
disable-model-invocation: true
argument-hint: "[server | web | docker-compose | Dockerfile | Makefile | sqlc]"
---

# genesis Reference Template

When the user asks to scaffold a Go backend, a React/Vite web frontend, Dockerize a service, add a docker-compose stack, add a Makefile, or set up sqlc, read the `reference/project-00` template in [`lwlee2608/genesis`](https://github.com/lwlee2608/genesis) and refer to it as the canonical template. Do not write these files from scratch or from memory.

The template is a monorepo with two services:
- `services/project-00-server` — Go API (cmd layout, `internal/api/http`, `internal/db` with migrations + sqlc, Makefile, Dockerfile)
- `services/project-00-web` — React/Vite/TypeScript app (eslint, nginx + entrypoint, Dockerfile)
- root `docker-compose.yml` (dev: Postgres only) and `docker-compose.prod.yml` (full stack)

## Structure

```
reference/project-00/
├── docker-compose.yml          # dev stack: Postgres only
├── docker-compose.prod.yml     # full stack: postgres + server + web
├── .env.example
└── services/
    ├── project-00-server/      # Go API
    │   ├── cmd/project-00/      # main, config, logger
    │   ├── internal/
    │   │   ├── api/http/        # router, handlers, dto, middleware
    │   │   └── db/              # migrations, queries, sqlc (generated)
    │   ├── Dockerfile
    │   ├── Makefile
    │   └── sqlc.yaml
    └── project-00-web/         # React/Vite/TS app
        ├── src/                # main.tsx, App.tsx
        ├── Dockerfile          # build + nginx serve
        ├── nginx.conf
        └── entrypoint.sh
```

## Arguments

The argument (e.g. "Add the server", "Set up sqlc", "Add the web Dockerfile") scopes the work — copy only the matching files, not the whole template. If no argument is given, ask which parts to scaffold.

## Rules

1. **Clone the live repo, never work from memory.** Source of truth is `https://github.com/lwlee2608/genesis` (branch `main`), under `reference/project-00`. Clone it once with `git clone --depth=1 https://github.com/lwlee2608/genesis /tmp/genesis-template`, then `ls` and `Read` files under `/tmp/genesis-template/reference/project-00`. This lets you see the full monorepo layout (`services/*-server/cmd`, `internal/api`, `internal/db`, `services/*-web/src`) — that structure is part of the template too, not just the individual files. The repo evolves; reproducing contents from memory causes drift.

2. **Adapt, don't blind-copy.** The template hardcodes the placeholder name `project-00` throughout — directory names (`services/project-00-server`, `services/project-00-web`), the Go module `github.com/lwlee2608/project-00`, the binary and `cmd/project-00/` path, `APP := project-00` in the Makefile, the `project-00`/`project-00-web` Docker image and container names, and the `project_00` Postgres database/user (underscore form). Replace all with the target project's name, keeping the underscore variant for Postgres identifiers. Otherwise keep the template's structure and conventions.

For deploying the result to Railway, use `/genesis-railway`.
