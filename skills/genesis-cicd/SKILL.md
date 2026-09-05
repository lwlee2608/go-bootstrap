---
name: genesis-cicd
description: Use when adding GitHub Actions CI/CD to a project scaffolded with genesis (services/<app>-server Go API, services/<app>-web pnpm React app). Asks which CD target (Railway, Kubernetes, VM + Overwatcher) and which branch model (main = production, or main = staging + release = production), then writes one ci.yml wired with needs so GitHub renders a single pipeline graph.
user-invocable: true
disable-model-invocation: true
argument-hint: "[railway | kubernetes | overwatcher] [main-prod | release-prod]"
---

# genesis-cicd

Adds one `.github/workflows/ci.yml` to a genesis monorepo. CI is the same for every project; the CD tail depends on two answers. Templates live in [reference.md](reference.md).

```
server ──┐
         ├──▶ build-image ──▶ (CD tail: none | helm deploy | Overwatcher picks it up)
web ─────┘
```

## Question flow

1. **Read the arguments first.** `railway`/`kubernetes`/`overwatcher` answers the CD target; `main-prod`/`release-prod` answers the branch model. Ask only what is missing.
2. **Ask the rest in ONE AskUserQuestion call:**
   - CD target: `railway (Recommended)` | `kubernetes` | `overwatcher` (VM running Docker Compose, deployed by the Overwatcher GitHub App)
   - Branch model: `main = production (Recommended)` | `main = staging, release = production`
3. **Confirm in one sentence, then write.** Example: "kubernetes, main→staging / release→prod, images to GCR — proceed?"

## Rules

1. **Read the target project first.** Confirm service dir names, `make test` in the server Makefile, `packageManager: pnpm@…` plus `pnpm-lock.yaml` in the web. Never leave `project-00`.

2. **Keep everything in one file joined by `needs:`.** A second workflow chained via `workflow_run` breaks the Actions graph, checks out the default-branch tip unless you pass `head_sha`, and fires for both the push and PR run of the same commit.

3. **One job per service, `working-directory` defaults, toolchains pinned to repo files** (`go-version-file`, lockfile `cache-dependency-path`, `pnpm/action-setup` reading `packageManager`).

4. **Branch model decides triggers:**
   - `main-prod`: `push: branches: [main]` + `pull_request`. `build-image` gated on push to `main`.
   - `release-prod`: `push: branches: [main, release]` + `pull_request`. Test jobs get `if: github.ref != 'refs/heads/release'` — a push to `release` is a merge of `main` that already ran every suite on the same tree. `build-image` then needs `always() && needs.*.result != 'failure' && != 'cancelled'`, because `skipped` is not `success`.

5. **CD target decides the tail:**
   - `railway`: no deploy job. Railway's GitHub integration builds from the Dockerfile on push; with `release-prod`, point the production environment at `release` and staging at `main`. Skip `build-image` unless the user also wants images in a registry.
   - `kubernetes`: `deploy` (staging, on `main`) and `deploy-prod` (on `release`, `environment: production`) with `needs: build-image`, GKE auth + `helm upgrade -i … --set image.tag=${{ github.sha }} --atomic`. Under `main-prod` there is only `deploy-prod`, on `main`.
   - `overwatcher`: no deploy job. Overwatcher listens for `workflow_run` of `ci.yml` on the service's branch and pulls the tag it is configured for. Push both images inside `build-image` — the last job — so `workflow_run(success)` never fires before the image lands. Tell the user to set the service's Workflow field to `ci.yml`.

6. **Tag images `sha` + `latest`, pass `VERSION` and `COMMIT_SHA` build-args, separate `cache-to` scopes per image.** Deploys pin the sha; `latest` is mutable and only for humans and Overwatcher services configured that way.

7. **Default to `ubuntu-latest` and GHCR.** GCR only when the user names a GCP project; it needs a service-account JSON secret. Blacksmith runners only if the org already pays for them.

8. **Never commit secrets.** GHCR uses `GITHUB_TOKEN` with `packages: write`; anything else is `${{ secrets.* }}` the user adds in repo settings.

## Verification procedure

1. `actionlint .github/workflows/ci.yml`, or `python3 -c 'import yaml,sys;yaml.safe_load(open(sys.argv[1]))' .github/workflows/ci.yml`.
2. `grep -n project-00 .github/workflows/ci.yml` returns nothing.
3. Run the same steps locally: `go vet ./... && make test`; `pnpm install --frozen-lockfile && pnpm lint && pnpm build`.
4. First run on the Actions tab shows `server` and `web` feeding `build-image` (and deploy jobs, if any) in one graph; on a PR, `build-image` is skipped, not missing.

## Common mistakes to watch for

- **Gating `build-image` on `needs` alone** — pushes images from PR branches.
- **Forgetting `always()` under `release-prod`** — tests are skipped on `release`, so `build-image` silently never runs there.
- **Writing a deploy job for Railway or Overwatcher** — both deploy on their own; a helm/ssh job would fight them.
- **`npm ci` for the web** — ignores `pnpm-lock.yaml`.
