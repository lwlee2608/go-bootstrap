---
name: genesis-railway
description: Use when deploying a project scaffolded with genesis (services/<app>-server Go API, services/<app>-web React app, Postgres) to Railway — creating the Railway project, services, variables, volume, GitHub auto-deploy, and public domain via the Railway MCP server.
user-invocable: true
disable-model-invocation: true
argument-hint: "[app name]"
---

# genesis-railway

Deploys a genesis monorepo to Railway using the Railway MCP tools. The layout it expects: `services/<app>-server` (Go API with a `Dockerfile` and `/health`), `services/<app>-web` (nginx-served React app with a `Dockerfile`), and a `docker-compose.prod.yml` describing the Postgres image.

```
Railway project: <app>
├── Postgres        # image service + volume
├── <app>-server    # builds services/<app>-server/Dockerfile
└── <app>-web       # builds services/<app>-web/Dockerfile, gets the public domain
        web ──▶ server ──▶ Postgres   (private networking)
```

## Prerequisites

**Railway MCP server.** Not bundled with Claude Code. If no `mcp__railway__*` tools are available, stop and have the user install it — do not fall back to guessing CLI commands:

```sh
claude mcp add --transport http railway https://mcp.railway.com   # add -s user for all projects
```

Then restart Claude Code, run `/mcp`, and complete the Railway OAuth login.

**Railway CLI** (`npm i -g @railway/cli` or `brew install railway`), used only for `railway login`, `railway init`, `railway link`, and reading resolved variables.

## Rules

1. **Use MCP tools for every Railway mutation.** Settings, variables, volumes, sources, and domains go through `mcp__railway__*`. The CLI is for login/link only.

2. **Never add `railway.json` / `railway.toml`.** Config-as-code is deprecated and stops working 2026-12-01, and no MCP tool writes it. All build/deploy settings are set on the service via `update-service` (step 3). Its successor, `.railway/railway.ts` (`railway config pull/plan/apply`, CLI ≥ 5), is out of scope.

3. **Create services in dependency order.** A `${{Service.VAR}}` reference is empty until the referenced service exists: Postgres, then server, then web.

4. **Find required variables before creating anything.** Grep `cmd/<app>/main.go` for config checks that `os.Exit` — each needs a variable. Config keys map to env with `.` → `_` (`http.port` → `HTTP_PORT`).

5. **Match the Postgres image to `docker-compose.yml`.** If a migration runs `CREATE EXTENSION postgis`, use `postgis/postgis`; Railway's Postgres plugin lacks it.

6. **Resolve the project ID first.** If the repo root is linked, read `~/.railway/config.json` (`.projects["<abs repo path>"].project`). Otherwise `list-projects`, or `railway init --name <app>` from the repo root.

## Steps

1. **Create services** — `create-service` `Postgres` with `image: postgres:<tag>` (name it exactly `Postgres`), then `<app>-server`, then `<app>-web` (no image).
2. **Variables** via `set-variables` with `skipDeploys: true`:
   - Postgres: `POSTGRES_DB=<app_underscore>`, `POSTGRES_USER=postgres`, `POSTGRES_PASSWORD=<openssl rand -hex 24>`, `PGDATA=/var/lib/postgresql/data/pgdata` (volume root isn't empty), `DATABASE_URL=postgres://${{POSTGRES_USER}}:${{POSTGRES_PASSWORD}}@${{RAILWAY_PRIVATE_DOMAIN}}:5432/${{POSTGRES_DB}}`
   - Server: `PORT=8080`, `HTTP_PORT=${{PORT}}` (both — the server reads `HTTP_PORT`; without it the healthcheck probes the wrong port), `DB_URL=${{Postgres.DATABASE_URL}}?sslmode=disable`, `LOG_LEVEL=info`, plus anything found in rule 4 (generate secrets, report them to the user).
   - Web: `BACKEND_URL=http://${{<app>-server.RAILWAY_PRIVATE_DOMAIN}}:8080`
3. **Settings** via `update-service` on server and web: `rootDirectory: services/<app>-{server,web}`, `dockerfilePath: Dockerfile`, `watchPatterns: ["services/<app>-{server,web}/**"]` (repo-root relative; a stale pattern matches nothing, so auto-deploy silently never fires), `restartPolicyType: ON_FAILURE`, `restartPolicyMaxRetries: 1`; server also `healthcheckPath: /health`.
4. **Volume** — `create-volume` on Postgres, `mountPath: /var/lib/postgresql/data`.
5. **Source** — `connect-service-source` on server, then web, with `repo: <owner>/<repo>`, `branch: main`. This attaches GitHub auto-deploy and starts the first build, so do it after steps 2–4.
6. **Domain** — `generate-domain` on web.
7. **Link subdirs** so the CLI works from inside them (absolute paths for `cd`):
   ```sh
   cd <abs>/services/<app>-server && railway link -p <app> -e production -s <app>-server
   cd <abs>/services/<app>-web    && railway link -p <app> -e production -s <app>-web
   ```

## Verification procedure

1. `get-status` until every service reports `SUCCESS`.
2. `curl https://<web domain>/health` returns the server's health response through the nginx proxy.
3. If proxied paths 500, check `BACKEND_URL` from a linked subdir with `railway variables --json` (MCP `list-variables` redacts references). If it renders as `http://:8080`, web built before server existed — `redeploy` web.

## Common mistakes to watch for

- **Setting only `PORT`.** The server reads `HTTP_PORT`; without it the healthcheck probes the wrong port and every deploy fails.
- **Skipping `PGDATA`.** The volume root isn't empty, so Postgres refuses to init without a subdirectory.
- **Connecting the source before variables/settings.** The first build fires immediately and runs with missing config.
- **Using the Postgres plugin when migrations need postgis.** The plugin image lacks the extension; use the `postgis/postgis` image service instead.
- **Using the placeholder `project-00` in `watchPatterns` or service names.** References like `${{project-00-server.RAILWAY_PRIVATE_DOMAIN}}` resolve to empty.
