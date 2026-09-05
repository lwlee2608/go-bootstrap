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
3. **Resolve target prerequisites before writing.** For Kubernetes, inspect the chart and collect missing GCP project, cluster locations, namespaces, release names, and environment values paths. The canonical scaffold has no Helm chart: if one is missing, stop and ask whether to create it as a separate prerequisite or choose another target. Do not invent paths or leave deployment placeholders.
4. **Confirm in one sentence, then write.** Example: "kubernetes, main→staging / release→prod, GCR, GHA cache — proceed?"

## Rules

1. **Read the target project first.** Confirm service dir names, `make test` in the server Makefile, `packageManager: pnpm@…` plus `pnpm-lock.yaml` in the web. Set `pnpm/action-setup`'s `package_json_file` to the service-local file; `defaults.run.working-directory` does not affect actions. For Kubernetes, validate `Chart.yaml`, environment values files, and both image repository/tag keys against the images being published. Never leave `project-00`.

2. **One workflow file, jobs joined by `needs:`.** A second workflow chained via `workflow_run` breaks the Actions graph, checks out the default-branch tip unless you pass `head_sha`, and fires for both the push and PR run of the same commit.

   Cancel superseded PR runs only. Give push runs distinct concurrency groups so an irrelevant push cannot cancel or displace a pending relevant build. Serialize Helm jobs per deployment branch without cancelling an active deploy.

3. **One test job per service, filtered by a `changes` job (`dorny/paths-filter`).** Grant `changes` `contents: read` and `pull-requests: read`; use `base: ${{ github.ref }}` so pushes compare against the previous commit on that branch. Service changes run that suite; changes to `ci.yml` run both. Exclude documentation in the job filters, not the workflow trigger: both `paths:` and `paths-ignore:` can leave required PR checks pending. `changes` also reports a `deploy` output for service and relevant CI/deployment configuration changes. Add the actual chart/values paths for Kubernetes and shared build inputs where applicable.

4. **Gate downstream work on relevance and successful prerequisites, not skipped tests alone.** `build-image` directly needs `changes` and every test job. Require `!cancelled()`, `needs.changes.result == 'success'`, a push to the selected deployment branch, and no failed/cancelled tests. Also require `needs.changes.outputs.deploy == 'true'`, except that `release-prod` always builds on a release promotion. Docs-only/unrelated changes skip downstream jobs; failed change detection never publishes. When building, build both images, even for Helm-only changes, because deploys pin both to `${{ github.sha }}`. Deploy jobs require `!cancelled()` and `needs.build-image.result == 'success'` to tolerate skipped upstream suites without accepting a skipped build.

5. **Branch model:**
   - `main-prod`: `push: branches: [main]` + `pull_request`.
   - `release-prod`: `push: branches: [main, release]` + `pull_request`. Only skip release test jobs after confirming a protected, promotion-only flow from a successfully tested `main` tree; a merge alone does not prove this. Otherwise run all suites on release pushes, regardless of path filters. Do not path-filter release pushes: even a docs-only promotion must publish images for its new SHA.

6. **CD target:**
   - `railway`: no `build-image`, no deploy job. Railway builds from the Dockerfile on push; under `release-prod` point its production environment at `release` and staging at `main`. Configure Wait for CI and service watch paths separately; skipped GitHub jobs alone do not suppress Railway deployments. See the reference checklist for promotion exceptions.
   - `kubernetes`: images to GCR (same service-account secret pushes and deploys). `deploy` (staging, on `main`) and `deploy-prod` (on `release`, `environment: production`) run `helm upgrade -i … --set image.tag=${{ github.sha }} --atomic`. Under `main-prod` only `deploy-prod` exists, on `main`.
   - `overwatcher`: images to GHCR, no deploy job. Set the service's Workflow field to `ci.yml`. Verify the integration accepts only successful push runs on the service's branch whose `build-image` job succeeded; workflow success alone also includes no-op and PR runs. If the integration cannot check the build job, report that prerequisite rather than claim no-op deployment suppression works.

7. **Tag images with the SHA and branch (`main` or `release`), never shared `latest`.** Pass server `VERSION` and `COMMIT_SHA` build-args. Kubernetes pins the SHA; Overwatcher must resolve both images from the successful run's `head_sha`. If it only supports fixed tags, resolve SHA selection as a prerequisite. Branch tags are convenience aliases, not deployment inputs: a later failed build can leave them pointing at a partially published image pair.

8. **Blacksmith runners:** `blacksmith-4vcpu-ubuntu-2404` for `server` and `build-image`, `blacksmith-2vcpu-ubuntu-2404` elsewhere. Keep upstream `actions/setup-go` / `actions/setup-node` with `cache` on; Blacksmith accelerates the GitHub cache API transparently and its `useblacksmith/setup-*` wrappers are deprecated.

9. **Docker layer cache: `type=gha`, one scope per image, unless the user chose `sticky-disk`.** `useblacksmith/setup-docker-builder` + `useblacksmith/build-push-action` keep the builder on a sticky disk billed per GB; never add them for "speed" without the user opting in, and never mix the two approaches in one job.

10. **Never commit secrets.** GHCR uses `GITHUB_TOKEN` with `packages: write` only on `build-image`; test jobs remain read-only. GCR/GKE need a service-account JSON the user adds in repo settings.

## Verification procedure

1. Run `actionlint .github/workflows/ci.yml`, configuring its allowed self-hosted runner labels for Blacksmith. A YAML parser is only a syntax fallback, not validation of Actions expressions or dependencies; report when actionlint was unavailable.
2. `grep -n project-00 .github/workflows/ci.yml` returns nothing.
3. Run the same steps locally: `go vet ./... && make test`; `pnpm install --frozen-lockfile && pnpm lint && pnpm build`.
4. On the Actions tab the run is one graph: `changes` → `server`/`web` → `build-image` (→ deploy). On a PR, `build-image` shows skipped, not missing.
5. A server-only push skips `web` but builds both images. A docs-only PR reports successful change detection and skipped suites; a docs-only/unrelated main push skips all downstream jobs. Failed/cancelled change detection or tests prevent publication and deployment.
6. Under `release-prod`, a release promotion builds and deploys even when suites are intentionally skipped. A Helm-only push builds both images and deploys. Verify no-op runs neither cancel relevant push runs nor trigger Overwatcher/Railway, and both deployed images match the successful run's SHA.

## Common mistakes to watch for

- **Accepting skipped tests without checking `changes`** - failed change detection must fail closed; unrelated changes must not publish.
- **Using workflow-level `paths:` or `paths-ignore:` on PRs** - required PR checks never report and block merging.
- **Treating workflow success as proof images were published** - no-op and PR runs can succeed with `build-image` skipped.
- **Writing a deploy job for Railway or Overwatcher** — both deploy on their own; a helm/ssh job would fight them.
- **`npm ci` for the web** — ignores `pnpm-lock.yaml`.
- **Sharing one `type=gha` scope between server and web** — each build evicts the other's layers.
