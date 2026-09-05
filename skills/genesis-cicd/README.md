# genesis-cicd

Creates one `.github/workflows/ci.yml` for a genesis monorepo, with jobs connected through `needs:` so GitHub Actions shows a single pipeline graph. Choose Kubernetes, Overwatcher, or Railway.

## Kubernetes: test, build, publish, deploy

```text
                               ┌────────┐    ┌────────────────────┐
                            ┌─▶│ server │───▶│ build-server-image │──┐
┌─────────┐    ┌─────────┐  │  └────────┘    └────────────────────┘  │  ┌─────────┐    ┌────────┐
│ changes │───▶│ resolve │──┤                                        ├─▶│ publish │───▶│ deploy │
└─────────┘    └─────────┘  │  ┌────────┐    ┌────────────────────┐  │  └─────────┘    └────────┘
                            └─▶│  web   │───▶│  build-web-image   │──┘
                               └────────┘    └────────────────────┘
```

The diagram shows the main flow; direct dependencies on `changes` and `resolve` used for gating and outputs are omitted for readability.

- **changes**: detect affected services and deployment inputs.
- **resolve**: compare build fingerprints with the selected track's published manifest; request only images that need rebuilding.
- **server / web**: run each requested service's checks independently.
- **build-server-image / build-web-image**: push images to GCR. Each starts as soon as its own checks pass, without waiting for the other service.
- **publish**: wait for all requested builds to succeed, then save a release manifest as a GitHub Release asset in the application repository. Retain existing digests for unchanged images.
- **deploy**: run Helm against GKE using both complete image digest references from the published manifest. Uses the selected GitHub environment, including the production gate.

**What runs when:**

- Pull requests: `changes` and affected service checks only; `resolve`, image builds, publication, and deployment skip.
- First push without a published baseline: test and build both images, then publish and deploy.
- Server-only or web-only push: test and build that image; retain the other image's digest.
- Helm-only push: skip both image builds; publish and deploy retained digests.
- Unrelated root-docs-only main push with no unpublished image changes: skip builds, publication, and deployment. Docs inside a service build context can require a rebuild.
- Failed requested checks or builds: block publication and deployment. A later push still detects changes not yet published.
- Failed Helm deployment: the manifest is already published; rerun the failed deployment job with its original refs rather than expecting a no-op main push to retry it.

Requires a digest-capable Helm chart, GCR/GKE configuration, credentials, and a verified commit pin for the external `lwlee2608/release-manifest` actions. The skill resolves these prerequisites before generating the workflow; the canonical genesis scaffold does not include a Helm chart.

## Overwatcher: test and build; deploy externally

```text
                ┌────────┐    ┌────────────────────┐
             ┌─▶│ server │───▶│ build-server-image │
┌─────────┐  │  └────────┘    └────────────────────┘
│ changes │──┤
└─────────┘  │  ┌────────┐    ┌────────────────────┐
             └─▶│  web   │───▶│  build-web-image   │
                └────────┘    └────────────────────┘
```

Outside GitHub Actions (requires both image-build jobs to succeed):

```text
┌──────────────────────────┐    ┌─────────────┐    ┌─────────────────────┐
│   Successful push run    │───▶│ Overwatcher │───▶│ VM / Docker Compose │
└──────────────────────────┘    └─────────────┘    └─────────────────────┘
```

`server -> build-server-image` and `web -> build-web-image` are independent paths. Each image job depends directly on `changes` and its own checks, not the other service. Both images are built and pushed to GHCR on relevant eligible pushes, or every release push, so both SHA tags exist for deployment. Checks may intentionally skip under the reference path-filter/release policy. A failed suite or image build does not block the other image path.

There is **no deploy job**. Configure Overwatcher's Workflow field as `ci.yml`, select the deployment branch, and verify that the integration requires a successful push run **and both `build-server-image` and `build-web-image` to succeed**. Both deployed images must use that run's `head_sha` tags, not mutable branch tags or shared `latest`. A partial build must not deploy.

## Railway: test only; build and deploy externally

```text
                ┌────────┐
             ┌─▶│ server │
┌─────────┐  │  └────────┘
│ changes │──┤
└─────────┘  │  ┌────────┐
             └─▶│  web   │
                └────────┘
```

Outside GitHub Actions:

```text
┌──────┐    ┌──────────────────────┐    ┌──────────────────┐    ┌────────┐
│ push │───▶│ Railway: Wait for CI │───▶│ Dockerfile build │───▶│ deploy │
└──────┘    └──────────────────────┘    └──────────────────┘    └────────┘
```

There is **no image-build or deploy job** in Actions. Railway builds and deploys independently. Configure **Wait for CI**, environment branches, and service watch paths in Railway; skipped GitHub jobs alone do not suppress Railway deployments.

## Branch models

```text
main-prod

┌────────┐    ┌────────────┐
│  main  │───▶│ production │
└────────┘    └────────────┘

release-prod

┌────────┐    ┌────────────┐
│  main  │───▶│  staging   │
└───┬────┘    └────────────┘
    │
    │  merge into release
    ▼
┌────────┐    ┌────────────┐
│release │───▶│ production │
└────────┘    └────────────┘
```

For Kubernetes, merging into `release` automatically resolves against the **production** baseline, tests/builds requested images from the release commit, and deploys production. It does not promote staging images. A release push with no new images still publishes and deploys retained production digests.

For Railway and Overwatcher, release checks may skip only with a protected, already-tested main-tree promotion flow; otherwise release pushes run all suites. Configure external production deployments to track `release` and staging deployments to track `main`.

## Usage and options

```text
/genesis-cicd kubernetes release-prod
/genesis-cicd overwatcher main-prod
/genesis-cicd railway main-prod
```

Missing choices are prompted. Kubernetes and Overwatcher default to GitHub Actions Docker layer caching; add `sticky-disk` only to opt into paid Blacksmith caching. Workflows use Blacksmith runners.

If the server has integration-tagged database tests, an optional `store` job gates only `build-server-image` for Kubernetes and Overwatcher; the web image build remains independent.

See [SKILL.md](SKILL.md) for generation rules, [kubernetes.md](kubernetes.md) for the full Kubernetes workflow, and [reference.md](reference.md) for Railway/Overwatcher templates and configuration checklists.
