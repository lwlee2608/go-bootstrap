---
name: genesis-cicd
description: Use when adding GitHub Actions CI/CD to a project scaffolded with genesis (services/<app>-server Go API, services/<app>-web pnpm React app). Asks which CD target (Railway, Kubernetes, VM + Overwatcher) and which branch model (main = production, or main = staging + release = production), then writes one ci.yml wired with needs so GitHub renders a single pipeline graph.
user-invocable: true
disable-model-invocation: true
argument-hint: "[railway | kubernetes | overwatcher] [main-prod | release-prod] [sticky-disk]"
---

# genesis-cicd

Adds one `.github/workflows/ci.yml` to a genesis monorepo. CI is the same for every project; the CD tail depends on two answers. Templates live in [reference.md](reference.md).

```
changes ──┬──▶ server ──┐
          └──▶ web ─────┴──▶ build-image ──▶ (CD tail: none | helm deploy | Overwatcher picks it up)
```

## Question flow

1. **Read the arguments first.** `railway`/`kubernetes`/`overwatcher` answers the CD target; `main-prod`/`release-prod` answers the branch model; `sticky-disk` opts into Blacksmith's paid Docker cache. Ask only what is missing.
2. **Ask the rest in ONE AskUserQuestion call:**
   - CD target: `railway (Recommended)` | `kubernetes` | `overwatcher` (VM running Docker Compose, deployed by the Overwatcher GitHub App)
   - Branch model: `main = production (Recommended)` | `main = staging, release = production`
   - Docker layer cache (skip if no `build-image`): `GitHub Actions cache (Recommended)` — free | `Blacksmith sticky disk` — $0.50/GB/month, persists `RUN --mount=type=cache` too
3. **Confirm in one sentence, then write.** Example: "kubernetes, main→staging / release→prod, images to GCR, GHA cache — proceed?"

## Rules

1. **Read the target project first.** Confirm service dir names, `make test` in the server Makefile, `packageManager: pnpm@…` plus `pnpm-lock.yaml` in the web. Never leave `project-00`.

2. **Keep everything in one file joined by `needs:`.** A second workflow chained via `workflow_run` breaks the Actions graph, checks out the default-branch tip unless you pass `head_sha`, and fires for both the push and PR run of the same commit.

3. **One job per service, `working-directory` defaults, toolchains pinned to repo files** (`go-version-file`, lockfile `cache-dependency-path`, `pnpm/action-setup` reading `packageManager`).

4. **Filter test jobs per service with a `changes` job (`dorny/paths-filter`), never with workflow-level `paths:`.** `server` runs only when `services/<app>-server/**` changed, same for `web`. A workflow that never starts never reports a status check, so PRs with required checks hang; a skipped job reports as passing. `build-image` therefore gates on `always() && needs.*.result != 'failure' && != 'cancelled'` — `skipped` is not `success`. Keep `paths-ignore` for `docs/**` and `**.md` so doc-only pushes cost zero runner minutes.

5. **Always build both images on a release branch, even if only one service changed.** Deploys pin `image.tag=${{ github.sha }}` for both; skipping one image would need a fallback tag. The unchanged image is a cache hit and costs seconds.

6. **Branch model decides triggers:**
   - `main-prod`: `push: branches: [main]` + `pull_request`. `build-image` gated on push to `main`.
   - `release-prod`: `push: branches: [main, release]` + `pull_request`. Test jobs also get `github.ref != 'refs/heads/release'` — a push to `release` is a merge of `main` that already ran every suite on the same tree.

7. **CD target decides the tail:**
   - `railway`: no deploy job. Railway's GitHub integration builds from the Dockerfile on push; with `release-prod`, point the production environment at `release` and staging at `main`. Skip `build-image` unless the user also wants images in a registry.
   - `kubernetes`: `deploy` (staging, on `main`) and `deploy-prod` (on `release`, `environment: production`) with `needs: build-image`, GKE auth + `helm upgrade -i … --set image.tag=${{ github.sha }} --atomic`. Under `main-prod` there is only `deploy-prod`, on `main`.
   - `overwatcher`: no deploy job. Overwatcher listens for `workflow_run` of `ci.yml` on the service's branch and pulls the tag it is configured for. Push both images inside `build-image` — the last job — so `workflow_run(success)` never fires before the image lands. Tell the user to set the service's Workflow field to `ci.yml`.

8. **Tag images `sha` + `latest`, pass `VERSION` and `COMMIT_SHA` build-args.** Deploys pin the sha; `latest` is mutable and only for humans and Overwatcher services configured that way.

9. **Run on Blacksmith runners.** `blacksmith-4vcpu-ubuntu-2404` for `server` and `build-image`, `blacksmith-2vcpu-ubuntu-2404` for `web`. Keep upstream `actions/setup-go` / `actions/setup-node` with `cache` on — Blacksmith accelerates the GitHub cache API transparently, and its own `useblacksmith/setup-*` wrappers are deprecated. Runner minutes are billed per minute; no other charge.

10. **Cache Docker layers with `type=gha`, one scope per image, unless the user chose `sticky-disk`.** `docker/build-push-action` + `cache-from/to: type=gha` is free. `useblacksmith/setup-docker-builder` + `useblacksmith/build-push-action` store the builder disk on a sticky disk billed at $0.50/GB/month (auto-evicted after 7 idle days); its only edge is persisting `RUN --mount=type=cache` between runs. Never mix the two in one job.

11. **Default to GHCR.** GCR only when the user names a GCP project; it needs a service-account JSON secret.

12. **Never commit secrets.** GHCR uses `GITHUB_TOKEN` with `packages: write`; anything else is `${{ secrets.* }}` the user adds in repo settings.

## Verification procedure

1. `actionlint .github/workflows/ci.yml`, or `python3 -c 'import yaml,sys;yaml.safe_load(open(sys.argv[1]))' .github/workflows/ci.yml`.
2. `grep -n project-00 .github/workflows/ci.yml` returns nothing.
3. Run the same steps locally: `go vet ./... && make test`; `pnpm install --frozen-lockfile && pnpm lint && pnpm build`.
4. First run on the Actions tab shows `changes` fanning out to `server` and `web`, both feeding `build-image` (and deploy jobs, if any) in one graph; on a PR, `build-image` is skipped, not missing.
5. Push a server-only change: `web` shows skipped, `server` runs. Push a README-only change: no run at all.

## Common mistakes to watch for

- **Gating `build-image` on `needs` alone** — pushes images from PR branches.
- **Forgetting `always()` on `build-image`** — a skipped test job makes `needs` fail closed, so it silently never runs.
- **Using workflow-level `paths:` instead of the `changes` job** — required PR checks never report and block merging.
- **Writing a deploy job for Railway or Overwatcher** — both deploy on their own; a helm/ssh job would fight them.
- **`npm ci` for the web** — ignores `pnpm-lock.yaml`.
- **Using `useblacksmith/setup-docker-builder` "for speed" without asking** — it silently turns on a per-GB sticky-disk charge.
- **Sharing one `type=gha` scope between server and web** — each build evicts the other's layers.
