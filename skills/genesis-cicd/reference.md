# genesis-cicd reference

## Base ci.yml (main-prod, GHCR)

```yaml
name: ci

on:
  push:
    branches: [main]
    paths-ignore: ['docs/**', '**.md']
  pull_request:
    paths-ignore: ['docs/**', '**.md']

concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

permissions:
  contents: read
  packages: write

env:
  SERVER_IMAGE: ghcr.io/${{ github.repository_owner }}/<app>-server
  WEB_IMAGE: ghcr.io/${{ github.repository_owner }}/<app>-web

jobs:
  server:
    runs-on: ubuntu-latest
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
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: services/<app>-web
    steps:
      - uses: actions/checkout@v4
      - uses: pnpm/action-setup@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: pnpm
          cache-dependency-path: services/<app>-web/pnpm-lock.yaml
      - run: pnpm install --frozen-lockfile
      - run: pnpm lint
      - run: pnpm build

  build-image:
    needs: [server, web]
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
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
          tags: ${{ env.SERVER_IMAGE }}:latest,${{ env.SERVER_IMAGE }}:${{ github.sha }}
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
          tags: ${{ env.WEB_IMAGE }}:latest,${{ env.WEB_IMAGE }}:${{ github.sha }}
          cache-from: type=gha,scope=web
          cache-to: type=gha,scope=web,mode=max
```

## release-prod variant

Replace the trigger, add the skip on test jobs, and widen the `build-image` gate.

```yaml
on:
  push:
    branches: [main, release]
    paths-ignore: ['docs/**', '**.md']
  pull_request:
    paths-ignore: ['docs/**', '**.md']

jobs:
  server:
    if: github.ref != 'refs/heads/release'
    # ...
  web:
    if: github.ref != 'refs/heads/release'
    # ...

  build-image:
    needs: [server, web]
    if: |
      always() &&
      github.event_name == 'push' &&
      (github.ref == 'refs/heads/main' || github.ref == 'refs/heads/release') &&
      needs.server.result != 'failure' && needs.server.result != 'cancelled' &&
      needs.web.result != 'failure' && needs.web.result != 'cancelled'
    # ...
```

## GCR variant

```yaml
env:
  SERVER_IMAGE: gcr.io/<gcp-project>/<app>-server
  WEB_IMAGE: gcr.io/<gcp-project>/<app>-web
# ...
      - uses: docker/login-action@v3
        with:
          registry: gcr.io
          username: _json_key
          password: ${{ secrets.GCR_SA_KEY }}
```

## CD tail: kubernetes (GKE + Helm)

`deploy` exists only under `release-prod`. Under `main-prod` keep `deploy-prod` with `github.ref == 'refs/heads/main'`.

```yaml
env:
  RELEASE: <app>
  NAMESPACE: <namespace>
  GKE_PROJECT: <gcp-project>
  GKE_ZONE: <zone>

jobs:
  deploy:
    needs: build-image
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
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
    if: github.event_name == 'push' && github.ref == 'refs/heads/release'
    runs-on: ubuntu-latest
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

1. GitHub App subscribed to `workflow_run` with Actions read permission.
2. In Overwatcher, set the service's Workflow field to `ci.yml` and its branch (`main`, or `release` for prod under `release-prod`).
3. Service `tag` matches what `build-image` pushes: `latest` or the commit sha.

## CD tail: railway

No job. `build-image` is optional. Under `release-prod`, set the Railway production environment's branch to `release` and staging to `main`.

## Optional store job (integration tests against Postgres)

Only if the server has `//go:build integration` tests. Add to `build-image`'s `needs`. Match the image in the repo's `docker-compose.yml`.

```yaml
  store:
    runs-on: ubuntu-latest
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
