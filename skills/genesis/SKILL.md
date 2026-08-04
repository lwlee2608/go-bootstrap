---
name: genesis
description: Use when scaffolding a full-stack project — a Go backend, a React/Vite web frontend, Dockerfiles, docker-compose, a Makefile, sqlc — or deploying the stack to Railway. Refers to reference/project-00 in github.com/lwlee2608/genesis as the canonical template.
user-invocable: true
disable-model-invocation: true
argument-hint: [server | web | docker-compose | Dockerfile | Makefile | sqlc | railway]
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
    │   ├── railway.json        # Railway build/deploy config
    │   └── sqlc.yaml
    └── project-00-web/         # React/Vite/TS app
        ├── src/                # main.tsx, App.tsx
        ├── Dockerfile          # build + nginx serve
        ├── nginx.conf
        ├── railway.json        # Railway build/deploy config
        └── entrypoint.sh
```

## Arguments

The argument (e.g. "Add the server", "Set up sqlc", "Add the web Dockerfile") scopes the work — copy only the matching files, not the whole template. If no argument is given, ask which parts to scaffold.

## Rules

1. **Clone the live repo, never work from memory.** Source of truth is `https://github.com/lwlee2608/genesis` (branch `main`), under `reference/project-00`. Clone it once with `git clone --depth=1 https://github.com/lwlee2608/genesis /tmp/genesis-template`, then `ls` and `Read` files under `/tmp/genesis-template/reference/project-00`. This lets you see the full monorepo layout (`services/*-server/cmd`, `internal/api`, `internal/db`, `services/*-web/src`) — that structure is part of the template too, not just the individual files. The repo evolves; reproducing contents from memory causes drift.

2. **Adapt, don't blind-copy.** The template hardcodes the placeholder name `project-00` throughout — directory names (`services/project-00-server`, `services/project-00-web`), the Go module `github.com/lwlee2608/project-00`, the binary and `cmd/project-00/` path, `APP := project-00` in the Makefile, the `project-00`/`project-00-web` Docker image and container names, and the `project_00` Postgres database/user (underscore form). Replace all with the target project's name, keeping the underscore variant for Postgres identifiers. Otherwise keep the template's structure and conventions.

## Railway deploy

The stack maps to one Railway **project** with three services, mirroring `docker-compose.prod.yml`. Each app service builds from its own subdirectory's `Dockerfile`.

```
Railway project: <app>
├── Postgres        # database plugin (has its own volume)
├── <app>-server    # builds services/<app>-server/Dockerfile
└── <app>-web       # builds services/<app>-web/Dockerfile, gets the public domain
        web ──▶ server ──▶ Postgres   (private networking)
```

**Linking is per-subdirectory.** The Railway CLI keys link state by absolute directory (in `~/.railway/config.json`), so each `services/<app>-*` subdir links to its *own* Railway service within the same project and environment.

### Config as code (`railway.json`)

Builder, watch paths, health check, and restart policy come from each service's `railway.json` (see the template) — config in code overrides dashboard settings. Only **Source**, **Root Directory**, and the **Config as Code** path can't be set there; env vars stay in `railway variables set` so `${{...}}` refs resolve.

Gotcha: the config path does *not* follow Root Directory — set it to the repo-root path `services/<app>-server/railway.json`. `railway up` from the subdir needs no setting; that subdir is the upload root.

### From scratch

Only `railway login` needs a human (browser, or device code on headless); the rest is scriptable. Create the project and all services first, **in order** — `railway link` can only target a service that already exists, and a reference variable resolves to empty until the service it points at exists: **Postgres before the server** (whose `DB_URL` needs `${{Postgres.DATABASE_URL}}`), then **server before web** (whose `BACKEND_URL` needs `${{<app>-server.RAILWAY_PRIVATE_DOMAIN}}`).

```sh
railway login                              # one-time

# create the project + services (run from the repo root)
railway init --name <app>                  # creates the project, links this dir
railway add --database postgres            # FIRST — exposes DATABASE_URL; ${{Postgres.DATABASE_URL}} is empty until this exists
railway add --service <app>-server         # before web — web references ${{<app>-server.RAILWAY_PRIVATE_DOMAIN}}
railway add --service <app>-web

# link + deploy each service from its own subdir (each already has its railway.json)
cd services/<app>-server
railway link                               # pick the project, environment, and <app>-server service
railway variables set 'DB_URL=${{Postgres.DATABASE_URL}}'   # append ?sslmode=disable if pgx needs it
railway variables set PORT=8080            # pin the target port the healthcheck probes
railway variables set 'HTTP_PORT=${{PORT}}' # the server reads HTTP_PORT, not PORT
railway up                                 # uploads this dir; railway.json here picks the Dockerfile builder

cd ../<app>-web
railway link                               # same project/environment, pick the <app>-web service
railway variables set 'BACKEND_URL=http://${{<app>-server.RAILWAY_PRIVATE_DOMAIN}}:8080'
railway up
railway domain                             # generate the public URL for web
```

The `${{...}}` are Railway reference variables that wire services together — single-quote them so the shell passes them through literally. The web `entrypoint.sh` auto-derives `DNS_RESOLVER` from the container's resolver, so that never needs setting.

**The server's port needs both variables.** Its config keys map to env with `.` → `_` (`http.port` → `HTTP_PORT`), so Railway's injected `PORT` alone is ignored and the server would stay on its `application.yml` default while the healthcheck probes elsewhere — failing every deploy. Pinning `PORT=8080` also keeps the web service's `BACKEND_URL=...:8080` correct. The web service needs no such setting: nginx already listens on `${PORT}`.

Notes:
- **Non-interactive linking:** `railway link` prompts for selections; pass `-p <project> -e production -s <service>` to script it.
- **CI / no browser:** set `RAILWAY_TOKEN=<project-token>` instead of `railway login` — the only otherwise non-scriptable step.
- **`railway up` can bootstrap by itself** on a cold run (creates the project + service and deploys; add `-y` to skip prompts). The explicit flow above is preferred because it fixes the service *names*, which the reference variables depend on.
- **Auto-deploy on push** (instead of `railway up`): connect each service to GitHub — see [Connecting GitHub](#connecting-github-auto-deploy) below.

### Connecting GitHub (auto-deploy)

A service created via the CLI is **not connected to GitHub** — it only redeploys on `railway up`. Both the BE (`<app>-server`) and FE (`<app>-web`) services start disconnected and must be wired up. Once connected, `railway.json` supplies the builder, watch paths, health check, and restart policy, so only three things remain in the dashboard (Service → Settings):

1. **Source** — connect the GitHub repo (`<owner>/<repo>`).
2. **Root Directory** — set to the service's subdir: `services/<app>-server` for the BE, `services/<app>-web` for the FE.
3. **Config as Code** — set to the repo-root path of that service's file: `services/<app>-server/railway.json` (BE) or `services/<app>-web/railway.json` (FE). Required — this setting does *not* inherit the Root Directory.

CLI equivalent for the source connect (from the linked subdir): `railway service source connect --repo <owner>/<repo> --branch main`. Root Directory and the config file path have no CLI or config-as-code equivalent.
