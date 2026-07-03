# Henry Release and Registry Runbook

This runbook defines the first real release verification flow for the fork-owned `henry-newapi` delivery chain.

## Ownership model

The workflows in this repository should now publish only to fork-controlled targets:

- GitHub Releases on `henrydontbbai/henry-newapi`
- GHCR images under `ghcr.io/henrydontbbai/henry-newapi`
- Gitee release sync only to a target provided by repo vars or explicit workflow inputs

No workflow in this repo should default to upstream `calciumion/new-api` or `QuantumNous/new-api` release targets.

## Required repository configuration

### GitHub / GHCR

No extra Docker Hub secret is required for the fork-owned image path.

The container workflows use:

- `ghcr.io/${GITHUB_REPOSITORY,,}` as the package name
- `packages: write`
- `${{ secrets.GITHUB_TOKEN }}` for GHCR auth

### Gitee

Before running `Sync Release to Gitee`, configure one of these:

- repo variables:
  - `GITEE_OWNER`
  - `GITEE_REPO`
- or workflow-dispatch inputs:
  - `gitee_owner`
  - `gitee_repo`

Also configure:

- repo secret `GITEE_TOKEN`

## Disposable prerelease tag flow

Use a throwaway prerelease-style tag for the first real audit. Recommended format:

- `v0.0.0-preflight.1`

Example:

```bash
git switch main
git pull --ff-only origin main
git tag v0.0.0-preflight.1
git push origin v0.0.0-preflight.1
```

That tag should trigger:

- `Release (Linux, macOS, Windows)`
- `Publish Container Image (GHCR, Multi-arch)`

## Verification order

Follow this order exactly:

1. create and push the disposable prerelease tag on the fork
2. wait for the GitHub Release workflow to finish successfully
3. verify release assets on GitHub
4. wait for the GHCR image workflow to finish successfully
5. verify GHCR version and `latest` manifests
6. dispatch `Sync Release to Gitee` only after the GitHub Release exists

## GitHub Release checks

```bash
gh release view v0.0.0-preflight.1 --repo henrydontbbai/henry-newapi
gh run list --repo henrydontbbai/henry-newapi --limit 20 --json workflowName,displayTitle,status,conclusion,event
```

Release acceptance:

- Linux, macOS, and Windows jobs all succeed
- release assets exist for all expected binaries and checksums
- no manual edits are needed on the GitHub Release page

## GHCR checks

Expected image namespace:

- `ghcr.io/henrydontbbai/henry-newapi`

Expected tags from the disposable prerelease:

- `ghcr.io/henrydontbbai/henry-newapi:v0.0.0-preflight.1`
- `ghcr.io/henrydontbbai/henry-newapi:latest`

Use either the GitHub Packages UI or `docker buildx imagetools inspect` from a machine with Docker/Buildx available.

Acceptance:

- versioned manifest exists
- `latest` manifest exists
- both amd64 and arm64 images are present

## Gitee sync checks

Dispatch manually after the GitHub Release is present:

```bash
gh workflow run sync-to-gitee.yml   --repo henrydontbbai/henry-newapi   -f tag_name=v0.0.0-preflight.1   -f gitee_owner=<your-gitee-owner>   -f gitee_repo=<your-gitee-repo>
```

Acceptance:

- workflow succeeds
- release title/body match the GitHub Release
- assets upload if release assets exist
- target Gitee release URL resolves under the fork-owned namespace

## Cleanup after the first audit

If the prerelease tag is meant only for workflow verification, clean it up after auditing:

```bash
gh release delete v0.0.0-preflight.1 --repo henrydontbbai/henry-newapi --yes
git push origin :refs/tags/v0.0.0-preflight.1
git tag -d v0.0.0-preflight.1
```

If the Gitee sync or GHCR package should also be cleaned up, do that manually in the corresponding registry UI after capturing proof of success.
