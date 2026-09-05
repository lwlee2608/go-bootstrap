# Kubernetes CI/CD

Use this reference only for Kubernetes (GCR, GKE, Helm). It replaces the Kubernetes build/deploy and release-test policies in `reference.md`; Railway and Overwatcher stay unchanged. Consume the external release-manifest actions, not local helper scripts or a new manifest repository. Releases are stored in the application repository.

## Prerequisites

- Replace both `<release-manifest-sha>` placeholders with the same **verified released commit** of `lwlee2608/release-manifest` before generating the workflow. Verify that commit's action contracts; do not invent a tag or SHA. `resolve` outputs `build`, `retained`, and a draft `manifest`; `publish` outputs `release`, `created`, `url`, and the complete published `manifest`. Deployment refs come from that published record, not a caller-side join of build outputs.
- Inspect the service directories, server Makefile/test target and `VERSION`, Dockerfiles, web `packageManager`, and lockfile. Resolve all `<...>` placeholders from the target project.
- Require an existing Helm chart. Inspect `Chart.yaml`, templates, and environment values. The example assumes the templates consume complete digest references through `server.image.ref` and `web.image.ref`; verify those keys or adapt to the chart's actual full-reference keys. If it only supports repository/tag, resolve chart adaptation separately. Do not assume the scaffold supplies `helm/`.
- Confirm chart/values paths, GCP project, cluster, zone or region, namespace, and Helm release name for each environment; these may all differ. Replace the `helm/**` filter with the actual deployment paths. Provision namespaces and registry pull access. Configure `GCR_SA_KEY` for GCR pushes and GKE deployment, GitHub Release write access, and the GitHub `production` environment (also `staging` for release-prod). For unattended CD, environment rules must not require manual approval.
- Extend fingerprints for shared build inputs, build arguments, and configuration outside each service. Pin Docker base images by digest for reproducibility. Do not add the current `github.sha` to fingerprints: unrelated commits must retain images. The server commit argument below is stable across unrelated pushes, included in its fingerprint, and passed unchanged to Docker. `VERSION` is covered by the service tree. The web receives no commit argument.

## Main-Prod Workflow

The first YAML block is a complete `.github/workflows/ci.yml`: `main` publishes and deploys the `production` track. Use the second block only for release-prod. Do not add trigger-level path filters.

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
  SERVER_IMAGE: gcr.io/<gcp-project>/<app>-server
  WEB_IMAGE: gcr.io/<gcp-project>/<app>-web

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
              - '{services/<app>-server/**,services/<app>-web/**,.github/workflows/ci.yml,helm/**}'
              - '!**/*.md'
              - '!**/docs/**'

  resolve:
    needs: changes
    if: github.event_name == 'push' && github.ref == 'refs/heads/main' && needs.changes.result == 'success'
    runs-on: blacksmith-2vcpu-ubuntu-2404
    permissions:
      contents: read
    env:
      TRACK: production
    outputs:
      track: ${{ steps.inputs.outputs.track }}
      build: ${{ steps.resolve.outputs.build }}
      manifest: ${{ steps.resolve.outputs.manifest }}
      server_commit: ${{ steps.inputs.outputs.server_commit }}
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - id: inputs
        shell: bash
        run: |
          set -euo pipefail
          server_commit=$(git log -1 --format=%H -- services/<app>-server .github/workflows/ci.yml)
          server_inputs=$(git rev-parse HEAD:services/<app>-server HEAD:.github/workflows/ci.yml)
          server_fp=$(printf '%s\n' "$server_inputs" "$server_commit" | sha256sum | cut -d ' ' -f1)
          web_fp=$(git rev-parse HEAD:services/<app>-web HEAD:.github/workflows/ci.yml | sha256sum | cut -d ' ' -f1)
          images=$(jq -cn --arg sr "$SERVER_IMAGE" --arg wr "$WEB_IMAGE" --arg sf "$server_fp" --arg wf "$web_fp" \
            '[{name:"server",repository:$sr,fingerprint:$sf},{name:"web",repository:$wr,fingerprint:$wf}]')
          printf 'track=%s\nserver_commit=%s\nimages=%s\n' "$TRACK" "$server_commit" "$images" >> "$GITHUB_OUTPUT"
      - id: resolve
        uses: lwlee2608/release-manifest/resolve@<release-manifest-sha>
        with:
          track: ${{ env.TRACK }}
          images: ${{ steps.inputs.outputs.images }}

  server:
    needs: [changes, resolve]
    if: |
      !cancelled() && needs.changes.result == 'success' &&
      ((github.event_name == 'pull_request' && needs.changes.outputs.server == 'true') ||
       (github.event_name == 'push' && needs.resolve.result == 'success' && contains(fromJSON(needs.resolve.outputs.build), 'server')))
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
    needs: [changes, resolve]
    if: |
      !cancelled() && needs.changes.result == 'success' &&
      ((github.event_name == 'pull_request' && needs.changes.outputs.web == 'true') ||
       (github.event_name == 'push' && needs.resolve.result == 'success' && contains(fromJSON(needs.resolve.outputs.build), 'web')))
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
    needs: [changes, resolve, server, web]
    if: |
      !cancelled() && github.event_name == 'push' &&
      needs.changes.result == 'success' && needs.resolve.result == 'success' &&
      (needs.resolve.outputs.build != '[]' || needs.changes.outputs.deploy == 'true' || github.ref == 'refs/heads/release') &&
      ((contains(fromJSON(needs.resolve.outputs.build), 'server') && needs.server.result == 'success') ||
       (!contains(fromJSON(needs.resolve.outputs.build), 'server') && needs.server.result == 'skipped')) &&
      ((contains(fromJSON(needs.resolve.outputs.build), 'web') && needs.web.result == 'success') ||
       (!contains(fromJSON(needs.resolve.outputs.build), 'web') && needs.web.result == 'skipped'))
    runs-on: blacksmith-4vcpu-ubuntu-2404
    permissions:
      contents: read
    outputs:
      images: ${{ steps.refs.outputs.images }}
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
        if: needs.resolve.outputs.build != '[]'
      - uses: docker/login-action@v3
        if: needs.resolve.outputs.build != '[]'
        with:
          registry: gcr.io
          username: _json_key
          password: ${{ secrets.GCR_SA_KEY }}
      - id: version
        if: contains(fromJSON(needs.resolve.outputs.build), 'server')
        run: echo "value=$(cat services/<app>-server/VERSION)" >> "$GITHUB_OUTPUT"
      - id: server
        if: contains(fromJSON(needs.resolve.outputs.build), 'server')
        uses: docker/build-push-action@v6
        with:
          context: services/<app>-server
          push: true
          provenance: false
          tags: ${{ env.SERVER_IMAGE }}:build-${{ github.run_id }}-${{ github.run_attempt }}
          build-args: |
            VERSION=${{ steps.version.outputs.value }}
            COMMIT_SHA=${{ needs.resolve.outputs.server_commit }}
          cache-from: type=gha,scope=server
          cache-to: type=gha,scope=server,mode=max
      - id: web
        if: contains(fromJSON(needs.resolve.outputs.build), 'web')
        uses: docker/build-push-action@v6
        with:
          context: services/<app>-web
          push: true
          provenance: false
          tags: ${{ env.WEB_IMAGE }}:build-${{ github.run_id }}-${{ github.run_attempt }}
          cache-from: type=gha,scope=web
          cache-to: type=gha,scope=web,mode=max
      - id: refs
        shell: bash
        env:
          SERVER_DIGEST: ${{ steps.server.outputs.digest }}
          WEB_DIGEST: ${{ steps.web.outputs.digest }}
        run: |
          set -euo pipefail
          server_new= web_new=
          if [[ -n "$SERVER_DIGEST" ]]; then server_new="$SERVER_IMAGE@$SERVER_DIGEST"; fi
          if [[ -n "$WEB_DIGEST" ]]; then web_new="$WEB_IMAGE@$WEB_DIGEST"; fi
          images=$(jq -cn --arg server "$server_new" --arg web "$web_new" \
            '{server:$server,web:$web} | with_entries(select(.value != ""))')
          printf 'images=%s\n' "$images" >> "$GITHUB_OUTPUT"

  publish:
    needs: [resolve, build-image]
    if: |
      !cancelled() && github.event_name == 'push' &&
      needs.resolve.result == 'success' && needs.build-image.result == 'success'
    runs-on: blacksmith-2vcpu-ubuntu-2404
    permissions:
      contents: write
    concurrency:
      group: release-manifest-${{ needs.resolve.outputs.track }}
      cancel-in-progress: false
    outputs:
      release: ${{ steps.publish.outputs.release }}
      manifest: ${{ steps.publish.outputs.manifest }}
    steps:
      - id: publish
        uses: lwlee2608/release-manifest/publish@<release-manifest-sha>
        with:
          track: ${{ needs.resolve.outputs.track }}
          manifest: ${{ needs.resolve.outputs.manifest }}
          images: ${{ needs.build-image.outputs.images }}

  deploy:
    needs: [resolve, publish]
    if: |
      !cancelled() && github.event_name == 'push' &&
      needs.resolve.result == 'success' && needs.publish.result == 'success'
    runs-on: blacksmith-2vcpu-ubuntu-2404
    environment: ${{ needs.resolve.outputs.track }}
    concurrency:
      group: kubernetes-<app>-${{ needs.resolve.outputs.track }}
      cancel-in-progress: false
    env:
      GKE_PROJECT: <prod-gcp-project>
      GKE_CLUSTER: <prod-cluster>
      GKE_LOCATION: <prod-zone-or-region>
      NAMESPACE: <prod-namespace>
      RELEASE: <prod-helm-release>
      CHART: ./helm
      VALUES: helm/prod-values.yaml
      SERVER_REF: ${{ fromJSON(needs.publish.outputs.manifest).images.server.ref }}
      WEB_REF: ${{ fromJSON(needs.publish.outputs.manifest).images.web.ref }}
    steps:
      - uses: actions/checkout@v4
      - uses: google-github-actions/auth@v2
        with:
          credentials_json: ${{ secrets.GCR_SA_KEY }}
      - uses: google-github-actions/get-gke-credentials@v2
        with:
          cluster_name: ${{ env.GKE_CLUSTER }}
          location: ${{ env.GKE_LOCATION }}
          project_id: ${{ env.GKE_PROJECT }}
      - uses: azure/setup-helm@v4
      - run: |
          test -n "$SERVER_REF"
          test -n "$WEB_REF"
          helm upgrade -i "$RELEASE" "$CHART" -n "$NAMESPACE" --values "$VALUES" \
            --set-string "server.image.ref=$SERVER_REF" \
            --set-string "web.image.ref=$WEB_REF" \
            --atomic --timeout 15m
```

## Release-Prod Overlay

Recursively merge this YAML mapping into the first block: merge mapping keys, replace scalar values and sequences, and preserve omitted keys. In particular, replace `on.push.branches`, not append to it. This is an overlay, not a second workflow; no steps or jobs are removed. Use a YAML 1.2 parser so `on` remains a string key.

`main` resolves against `staging`; merging `main` into `release` triggers a push that resolves against `production` and automatically deploys production. Checkouts stay on the triggering commit (the release-branch commit for production). Changed production images are tested and built **from release**, never promoted from staging or resolved with `source-release`. Unchanged images may retain only the production baseline's digests. Keep the exact requested-test gates; do not apply the base reference's release-test skipping policy.

```yaml
on:
  push:
    branches: [main, release]
jobs:
  resolve:
    if: |
      github.event_name == 'push' && needs.changes.result == 'success' &&
      (github.ref == 'refs/heads/main' || github.ref == 'refs/heads/release')
    env:
      TRACK: ${{ github.ref == 'refs/heads/release' && 'production' || 'staging' }}
  deploy:
    env:
      GKE_PROJECT: ${{ needs.resolve.outputs.track == 'production' && '<prod-gcp-project>' || '<staging-gcp-project>' }}
      GKE_CLUSTER: ${{ needs.resolve.outputs.track == 'production' && '<prod-cluster>' || '<staging-cluster>' }}
      GKE_LOCATION: ${{ needs.resolve.outputs.track == 'production' && '<prod-zone-or-region>' || '<staging-zone-or-region>' }}
      NAMESPACE: ${{ needs.resolve.outputs.track == 'production' && '<prod-namespace>' || '<staging-namespace>' }}
      RELEASE: ${{ needs.resolve.outputs.track == 'production' && '<prod-helm-release>' || '<staging-helm-release>' }}
      VALUES: ${{ needs.resolve.outputs.track == 'production' && 'helm/prod-values.yaml' || 'helm/staging-values.yaml' }}
```

Branch eligibility is centralized in `resolve`; build/publish/deploy all require its success and a push, so no further branch gates need changing. The base build relevance gate already admits every release push, including no-new-image promotions. Deployment environment and concurrency follow the track automatically. Keep different target settings distinct; if chart directories differ too, add a conditional `CHART` entry to the overlay.

## Behavior And Recovery

- PRs use documentation-excluding service path filters; `resolve` is skipped. Explicit `!cancelled()` lets requested PR suites run despite that skip. Pushes resolve on every eligible commit, including docs-only commits, and test exactly the requested image names.
- Fingerprints hash Git object IDs for the **whole service tree**, including docs, and the workflow blob; the server also hashes its stable commit argument. Removing documentation from these fingerprints is unsafe when Docker can copy it. A service README push intentionally rebuilds that service, even though its PR suite was filtered. Root docs outside build contexts can be a no-op. Workflow changes request both images. Include shared inputs in PR filters as well as fingerprints.
- Without a published baseline on a track, resolve requests both images. Bootstrap therefore tests, builds, publishes, and deploys both even on an otherwise irrelevant push. Thereafter a server-only change tests/builds server and retains web; the reverse applies to web. Helm-only changes publish/deploy retained refs without Docker setup or login. Unrelated main pushes skip build/publish/deploy when the build list is empty and the deploy filter is false. Release pushes still publish/deploy retained production refs in that case.
- Failed/cancelled change detection, resolve, or requested tests prevent builds and publication. A partial Docker push does not advance the manifest baseline. Rerun failed jobs to retry, or push a fix: even a later unrelated push compares against the last published baseline and requests the still-unpublished images. Unique run/attempt tags avoid collisions; deployment uses digests, never branch tags or shared `latest`.
- Publication means requested tests/builds succeeded, **not** that deployment succeeded. A failed Helm deployment leaves the release published; a later no-op main push is not an automatic retry. Rerun the failed deployment job with its original published refs rather than rebuilding an already-published run. The action accepts identical republication, rejects a different manifest for the same run, and rejects publishing an older run after a newer release. Do not assume rebuilding produces identical digests.
- Publish concurrency is repository-wide per track; use the same group in every publisher. Deploy concurrency is per target: make its group identical across workflows deploying the same cluster/namespace/release, and distinct for different targets. Both avoid cancelling active jobs, but GitHub concurrency is not a FIFO queue and can replace pending jobs. Publication does not guarantee deployment order or prevent an older deploy from running later. Verify the current deployment before retrying an old job; strict ordering needs an additional target-side freshness mechanism.

## Optional Variants

- **Store integration tests:** use the Postgres service and test steps from `reference.md`, matching the project's database configuration. Give `store` the same `needs: [changes, resolve]` and complete `if` expression as `server`. Add `store` to `build-image.needs` and add the same exact requested-server gate for `needs.store.result`: require `success` when `server` is in `resolve.build`, otherwise require `skipped`. Do not merely reject failure/cancellation, and do not skip release tests unconditionally.
- **Sticky disk:** only on explicit opt-in, replace `docker/setup-buildx-action@v3` with `useblacksmith/setup-docker-builder@v2` and `cache-key: <app>-images`; replace both build actions with `useblacksmith/build-push-action@v2`. Remove `cache-from`/`cache-to`. Preserve setup/login conditions, build step IDs and conditions, build arguments, unique tags, job outputs, and each action's `digest` output consumed by `refs`. Verify the chosen action version exposes that output. Do not mix sticky-disk and GHA layer caches.

## Verification

Compose each branch model separately, substitute verified project values and action pins, then run `actionlint` with the Blacksmith runner labels allowed. Check the gates for PRs with skipped resolve, bootstrap, each single-service change, service docs, unrelated root docs, Helm-only changes, failed/cancelled prerequisites, partial build failure followed by an unrelated push, and a no-new-images release merge. Confirm only `publish` has `contents: write`, its `images` input contains newly built refs only, and Helm receives both nonempty complete digest refs from the published `manifest` output. No local release-manifest implementation is required.
