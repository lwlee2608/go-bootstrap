---
name: genesis-cicd
description: Use when adding GitHub Actions CI/CD to a project scaffolded with genesis (services/<app>-server Go API, services/<app>-web pnpm React app). Asks which CD target (Railway, Kubernetes, VM + Overwatcher) and which branch model (main = production, or main = staging + release = production), then writes one ci.yml wired with needs so GitHub renders a single pipeline graph.
user-invocable: true
disable-model-invocation: true
argument-hint: "[railway | kubernetes | overwatcher] [main-prod | release-prod] [sticky-disk]"
---

# genesis-cicd

Adds one `.github/workflows/ci.yml` to a genesis monorepo. CI is the same for every project; the tail depends on the answers below. Templates live in [reference.md](reference.md).

```
changes ──┬──▶ server ──┐
          └──▶ web ─────┴──▶ build-image ──▶ deploy (kubernetes only)
```

## Question flow

1. **Read the arguments first.** `railway`/`kubernetes`/`overwatcher` is the CD target; `main-prod`/`release-prod` is the branch model; `sticky-disk` opts into Blacksmith's paid Docker cache. Ask only what is missing.
2. **Ask the rest in ONE AskUserQuestion call:**
   - CD target: `railway (Recommended)` | `kubernetes` | `overwatcher` (VM running Docker Compose, deployed by the Overwatcher GitHub App)
   - Branch model: `main = production (Recommended)` | `main = staging, release = production`
   - Docker layer cache, only for `kubernetes`/`overwatcher`: `GitHub Actions cache (Recommended)` — free | `Blacksmith sticky disk` — $0.50/GB/month, also persists `RUN --mount=type=cache`
3. **Confirm in one sentence, then write.** Example: "kubernetes, main→staging / release→prod, GCR, GHA cache — proceed?"

## Rules

1. **Read the target project first.** Confirm service dir names, `make test` in the server Makefile, `packageManager: pnpm@…` plus `pnpm-lock.yaml` in the web. Never leave `project-00`.

2. **One workflow file, jobs joined by `needs:`.** A second workflow chained via `workflow_run` breaks the Actions graph, checks out the default-branch tip unless you pass `head_sha`, and fires for both the push and PR run of the same commit.

3. **One test job per service, filtered by a `changes` job (`dorny/paths-filter`).** `server` runs only when `services/<app>-server/**` changed, same for `web`. Never use workflow-level `paths:` for this: a workflow that never starts never reports a status check, so required PR checks hang, whereas a skipped job reports as passing. Keep `paths-ignore: ['docs/**', '**.md']` so doc-only pushes cost nothing.

4. **Gate `build-image` on push to a release branch with `always() && needs.*.result != 'failure' && != 'cancelled'`.** `skipped` is not `success`, so plain `needs` fails closed whenever a test job was filtered out. Always build both images even if one service was untouched — deploys pin `image.tag=${{ github.sha }}` for both, and the unchanged image is a cache hit.

5. **Branch model:**
   - `main-prod`: `push: branches: [main]` + `pull_request`.
   - `release-prod`: `push: branches: [main, release]` + `pull_request`. Test jobs also get `github.ref != 'refs/heads/release'` — a push to `release` is a merge of `main` that already ran every suite on the same tree.

6. **CD target:**
   - `railway`: no `build-image`, no deploy job. Railway builds from the Dockerfile on push; under `release-prod` point its production environment at `release` and staging at `main`.
   - `kubernetes`: images to GCR (same service-account secret pushes and deploys). `deploy` (staging, on `main`) and `deploy-prod` (on `release`, `environment: production`) run `helm upgrade -i … --set image.tag=${{ github.sha }} --atomic`. Under `main-prod` only `deploy-prod` exists, on `main`.
   - `overwatcher`: images to GHCR, no deploy job. Overwatcher matches `workflow_run` of `ci.yml` by filename on the service's branch. Push inside `build-image`, the last job, so `workflow_run(success)` never fires before the image lands. Tell the user to set the service's Workflow field to `ci.yml`.

7. **Tag images `sha` + `latest`, pass `VERSION` and `COMMIT_SHA` build-args.** Deploys pin the sha; `latest` is mutable.

8. **Blacksmith runners:** `blacksmith-4vcpu-ubuntu-2404` for `server` and `build-image`, `blacksmith-2vcpu-ubuntu-2404` elsewhere. Keep upstream `actions/setup-go` / `actions/setup-node` with `cache` on; Blacksmith accelerates the GitHub cache API transparently and its `useblacksmith/setup-*` wrappers are deprecated.

9. **Docker layer cache: `type=gha`, one scope per image, unless the user chose `sticky-disk`.** `useblacksmith/setup-docker-builder` + `useblacksmith/build-push-action` keep the builder on a sticky disk billed per GB; never add them for "speed" without the user opting in, and never mix the two approaches in one job.

10. **Never commit secrets.** GHCR uses `GITHUB_TOKEN` with `packages: write`; GCR/GKE need a service-account JSON the user adds in repo settings.

## Verification procedure

1. `actionlint .github/workflows/ci.yml`, or `python3 -c 'import yaml,sys;yaml.safe_load(open(sys.argv[1]))' .github/workflows/ci.yml`.
2. `grep -n project-00 .github/workflows/ci.yml` returns nothing.
3. Run the same steps locally: `go vet ./... && make test`; `pnpm install --frozen-lockfile && pnpm lint && pnpm build`.
4. On the Actions tab the run is one graph: `changes` → `server`/`web` → `build-image` (→ deploy). On a PR, `build-image` shows skipped, not missing.
5. A server-only push skips `web`; a README-only push starts no run.

## Common mistakes to watch for

- **Gating `build-image` on `needs` alone** — pushes images from PR branches, and fails closed when a test job is skipped.
- **Using workflow-level `paths:` instead of the `changes` job** — required PR checks never report and block merging.
- **Writing a deploy job for Railway or Overwatcher** — both deploy on their own; a helm/ssh job would fight them.
- **`npm ci` for the web** — ignores `pnpm-lock.yaml`.
- **Sharing one `type=gha` scope between server and web** — each build evicts the other's layers.
