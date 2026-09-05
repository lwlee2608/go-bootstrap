# genesis-cicd reference

## Base ci.yml (main-prod, GHCR)

```yaml
name: ci

on:
  push:
    branches: [main]
  pull_request:

concurrency:
  group: ${{ github.workflow }}-${{ github.event_name == 'pull_request' && github.ref || github.run_id }}
  cancel-in-progress: ${{ github.event_name == 'pull_request' }}

permissions:
  contents: read

env:
  SERVER_IMAGE: ghcr.io/${{ github.repository_owner }}/<app>-server
  WEB_IMAGE: ghcr.io/${{ github.repository_owner }}/<app>-web

jobs:
  changes:
    runs-on: blacksmith-2vcpu-ubuntu-2404
    permissions:
      contents: read
      pull-requests: read
    outputs:
      server: ${{ steps.filter.outputs.server }}
      web: ${{ steps.filter.outputs.web }}
      deploy: ${{ steps.filter.outputs.deploy }}
    steps:
      - uses: actions/checkout@v4
      - id: filter
        uses: dorny/paths-filter@v3
        with:
          base: ${{ github.ref }}
          predicate-quantifier: every
          filters: |
            server:
              - '{services/<app>-server/**,.github/workflows/ci.yml}'
              - '!**/*.md'
              - '!**/docs/**'
            web:
              - '{services/<app>-web/**,.github/workflows/ci.yml}'
              - '!**/*.md'
              - '!**/docs/**'
            deploy:
              - '{services/<app>-server/**,services/<app>-web/**,.github/workflows/ci.yml}'
              - '!**/*.md'
              - '!**/docs/**'

  server:
    needs: changes
    if: needs.changes.outputs.server == 'true'
    runs-on: blacksmith-4vcpu-ubuntu-2404
    defaults:
      run:
        working-directory: services/<app>-server
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: services/<app>-server/go.mod
          cache-dependency-path: services/<app>-server/go.sum
      - run: go vet ./...
      - run: make test

  web:
    needs: changes
    if: needs.changes.outputs.web == 'true'
    runs-on: blacksmith-2vcpu-ubuntu-2404
    defaults:
      run:
        working-directory: services/<app>-web
    steps:
      - uses: actions/checkout@v4
      - uses: pnpm/action-setup@v4
        with:
          package_json_file: services/<app>-web/package.json
      - uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: pnpm
          cache-dependency-path: services/<app>-web/pnpm-lock.yaml
      - run: pnpm install --frozen-lockfile
      - run: pnpm lint
      - run: pnpm build

  build-image:
    needs: [changes, server, web]
    if: |
      !cancelled() &&
      github.event_name == 'push' && github.ref == 'refs/heads/main' &&
      needs.changes.result == 'success' && needs.changes.outputs.deploy == 'true' &&
      needs.server.result != 'failure' && needs.server.result != 'cancelled' &&
      needs.web.result != 'failure' && needs.web.result != 'cancelled'
    runs-on: blacksmith-4vcpu-ubuntu-2404
    permissions:
      contents: read
      packages: write
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - id: version
        run: echo "value=$(cat services/<app>-server/VERSION)" >> "$GITHUB_OUTPUT"

      - uses: docker/build-push-action@v6
        with:
          context: services/<app>-server
          push: true
          provenance: false
          tags: ${{ env.SERVER_IMAGE }}:${{ github.ref_name }},${{ env.SERVER_IMAGE }}:${{ github.sha }}
          build-args: |
            VERSION=${{ steps.version.outputs.value }}
            COMMIT_SHA=${{ github.sha }}
          cache-from: type=gha,scope=server
          cache-to: type=gha,scope=server,mode=max

      - uses: docker/build-push-action@v6
        with:
          context: services/<app>-web
          push: true
          provenance: false
          tags: ${{ env.WEB_IMAGE }}:${{ github.ref_name }},${{ env.WEB_IMAGE }}:${{ github.sha }}
          cache-from: type=gha,scope=web
          cache-to: type=gha,scope=web,mode=max
```

## release-prod variant

Replace the trigger and job conditions below, preserving the remaining base configuration. This release-test skip requires a protected, promotion-only flow from an already-tested `main` tree. If that guarantee is absent, use `if: needs.changes.outputs.server == 'true' || github.ref == 'refs/heads/release'` for `server` (and the equivalent for `web` and optional `store`) so release pushes run all suites.

Do not add trigger-level path filters: a release promotion must publish its SHA even if only docs changed. On `main`, irrelevant changes still skip `build-image` through the `deploy` output.

```yaml
on:
  push:
    branches: [main, release]
  pull_request:

jobs:
  server:
    needs: changes
    if: needs.changes.outputs.server == 'true' && github.ref != 'refs/heads/release'
    # ...
  web:
    needs: changes
    if: needs.changes.outputs.web == 'true' && github.ref != 'refs/heads/release'
    # ...

  build-image:
    needs: [changes, server, web]
    if: |
      !cancelled() &&
      github.event_name == 'push' &&
      (github.ref == 'refs/heads/main' || github.ref == 'refs/heads/release') &&
      needs.changes.result == 'success' &&
      (needs.changes.outputs.deploy == 'true' || github.ref == 'refs/heads/release') &&
      needs.server.result != 'failure' && needs.server.result != 'cancelled' &&
      needs.web.result != 'failure' && needs.web.result != 'cancelled'
    # ...
```

## Sticky-disk variant of build-image (only if the user chose it)

Swap the builder and build actions; drop `cache-from`/`cache-to`. One builder holds both images' layers, so one `cache-key` is enough.

```yaml
      - uses: useblacksmith/setup-docker-builder@v2
        with:
          cache-key: <app>-images
      - uses: docker/login-action@v3
        # ...
      - uses: useblacksmith/build-push-action@v2
        with:
          context: services/<app>-server
          push: true
          provenance: false
          tags: ${{ env.SERVER_IMAGE }}:${{ github.ref_name }},${{ env.SERVER_IMAGE }}:${{ github.sha }}
          build-args: |
            VERSION=${{ steps.version.outputs.value }}
            COMMIT_SHA=${{ github.sha }}
      - uses: useblacksmith/build-push-action@v2
        with:
          context: services/<app>-web
          push: true
          provenance: false
          tags: ${{ env.WEB_IMAGE }}:${{ github.ref_name }},${{ env.WEB_IMAGE }}:${{ github.sha }}
```

## GCR variant

Replace the image names and registry login below; remove `packages: write` from `build-image` because it no longer publishes to GHCR.

```yaml
env:
  SERVER_IMAGE: gcr.io/<gcp-project>/<app>-server
  WEB_IMAGE: gcr.io/<gcp-project>/<app>-web
```

```yaml
      - uses: docker/login-action@v3
        with:
          registry: gcr.io
          username: _json_key
          password: ${{ secrets.GCR_SA_KEY }}
```

## CD tail: kubernetes (GKE + Helm)

Pair with the GCR variant so `GCR_SA_KEY` both pushes and deploys. `deploy` exists only under `release-prod`; under `main-prod` keep `deploy-prod` with `github.ref == 'refs/heads/main'`.

Prerequisites: the scaffold does not include Helm assets. Require an existing chart, inspect its `Chart.yaml` and environment values, and confirm its server/web image repositories match the published images. Resolve project, cluster, location (zone or region), namespace, and release name for each environment. If a chart is missing, stop and resolve chart creation separately before generating a deploy job. The paths and value keys below are examples: replace them with the verified chart's paths and keys, using per-job environment overrides when targets differ.

Extend the `deploy` filter's positive pattern to include the actual chart and values paths, for example `'{services/<app>-server/**,services/<app>-web/**,.github/workflows/ci.yml,helm/**}'`. Keep `predicate-quantifier: every` and the documentation exclusions. Helm-only changes intentionally build both images so the new commit's SHA tags exist. Do not add Helm-only paths to the service test filters.

```yaml
env:
  RELEASE: <app>
  NAMESPACE: <namespace>
  GKE_PROJECT: <gcp-project>
  GKE_ZONE: <zone>

jobs:
  deploy:
    needs: build-image
    if: |
      !cancelled() && needs.build-image.result == 'success' &&
      github.event_name == 'push' && github.ref == 'refs/heads/main'
    concurrency:
      group: ${{ github.workflow }}-deploy-${{ github.ref }}
      cancel-in-progress: false
    runs-on: blacksmith-2vcpu-ubuntu-2404
    steps:
      - uses: actions/checkout@v4
      - uses: google-github-actions/auth@v2
        with:
          credentials_json: ${{ secrets.GCR_SA_KEY }}
      - uses: google-github-actions/get-gke-credentials@v2
        with:
          cluster_name: <staging-cluster>
          location: ${{ env.GKE_ZONE }}
          project_id: ${{ env.GKE_PROJECT }}
      - uses: azure/setup-helm@v4
      - run: |
          helm upgrade -i $RELEASE ./helm -n $NAMESPACE --values helm/staging-values.yaml \
            --set server.image.tag=${{ github.sha }} \
            --set web.image.tag=${{ github.sha }} \
            --atomic --timeout 15m

  deploy-prod:
    needs: build-image
    if: |
      !cancelled() && needs.build-image.result == 'success' &&
      github.event_name == 'push' && github.ref == 'refs/heads/release'
    concurrency:
      group: ${{ github.workflow }}-deploy-${{ github.ref }}
      cancel-in-progress: false
    runs-on: blacksmith-2vcpu-ubuntu-2404
    environment: production
    steps:
      - uses: actions/checkout@v4
      - uses: google-github-actions/auth@v2
        with:
          credentials_json: ${{ secrets.GCR_SA_KEY }}
      - uses: google-github-actions/get-gke-credentials@v2
        with:
          cluster_name: <prod-cluster>
          location: ${{ env.GKE_ZONE }}
          project_id: ${{ env.GKE_PROJECT }}
      - uses: azure/setup-helm@v4
      - run: |
          helm upgrade -i $RELEASE ./helm -n $NAMESPACE --values helm/prod-values.yaml \
            --set server.image.tag=${{ github.sha }} \
            --set web.image.tag=${{ github.sha }} \
            --atomic --timeout 15m
```

## CD tail: overwatcher

No job. Checklist for the user:

1. GitHub App subscribed to `workflow_run` with Actions read permission. Verify it filters completed, successful runs to `event == 'push'` on the configured branch and checks that this run's `build-image` job concluded `success`. A successful workflow with skipped builds must not deploy. If the installed integration cannot check job conclusions, resolve that prerequisite before claiming downstream no-op skipping is supported.
2. In Overwatcher, set the service's Workflow field to `ci.yml` and its branch (`main`, or `release` for prod under `release-prod`).
3. Resolve both image tags from that run's `head_sha`. If the integration only supports fixed tags, resolve SHA selection as a prerequisite before enabling deployment. Do not deploy branch aliases or shared `latest`: a later failed/cancelled build may have overwritten only one image's mutable tag, even though the triggering run succeeded.

## CD tail: railway

No `build-image`, no deploy job; the workflow ends at `server`/`web`. Under `release-prod`, set the Railway production environment's branch to `release` and staging to `main`.

Enable Railway's Wait for CI and configure per-service watch paths for source/build inputs, excluding docs and unrelated files. GitHub job skips do not control Railway's independent push trigger. For production tracking `release`, allow every promotion through the watch configuration, including docs-only promotions; do not apply staging's relevance filter there. Verify these external settings and report anything not configured rather than claiming the YAML alone gates Railway deployment.

## Optional store job (integration tests against Postgres)

Only if the server has `//go:build integration` tests. Add `store` to `build-image`'s `needs` and both its `result != 'failure'` and `result != 'cancelled'` checks. Apply the same release-test policy as `server` under `release-prod`. Match the image and database environment variable to the target project's tests and `docker-compose.yml`.

```yaml
  store:
    needs: changes
    if: needs.changes.outputs.server == 'true'
    runs-on: blacksmith-4vcpu-ubuntu-2404
    defaults:
      run:
        working-directory: services/<app>-server
    services:
      postgres:
        image: pgvector/pgvector:pg17
        env:
          POSTGRES_PASSWORD: postgres
        ports: ['5432:5432']
        options: >-
          --health-cmd "pg_isready -U postgres"
          --health-interval 5s --health-timeout 5s --health-retries 10
    env:
      TEST_DB_URL: postgres://postgres:postgres@localhost:5432/postgres
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: services/<app>-server/go.mod
          cache-dependency-path: services/<app>-server/go.sum
      - run: go test -race -tags integration ./...
```
