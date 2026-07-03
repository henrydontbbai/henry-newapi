# Henry /66 Runtime Evidence Log

This file records the latest trusted `/66` evidence for the routing probe/restore slice and now also records how that evidence relates to the current mainline.

## Current status summary

### Known current state (`2026-07-04`)

- The trusted runtime evidence date is still `2026-06-30`.
- The historically verified routing slice is already merged on `main`.
- Relevant merged commits:
  - `8cfec81f feat: add native routing policy baseline and /66 verification docs`
  - `2c4e9354 feat: close probe restore loop and refresh /66 verification evidence`
  - `d12c0630 fix: clear pending restore state on routing success`
- Later engineering closure also landed on `main`, including green `Go CI`, `Frontend CI`, and `Release Dry Run` on `d95aa5ca`.

### Current unknown state

- `/66` has not yet been re-audited as a live preprod/staging environment after the mainline and CI closure work.
- There is no newer accepted `/66` runtime snapshot than the `2026-06-30` guest verification and smoke evidence.

### Historical capture note

At the time the `2026-06-30` evidence was collected, the slice had not yet been committed/pushed/submitted. That historical note is preserved below for context only; it is no longer the current repository state.

## Historical local package used for the 2026-06-30 rerun

- Local package file used for that run: `henry-newapi-src-current.tar.gz`
- Local SHA256 sidecar used for that run: `henry-newapi-src-current.tar.gz.sha256`
- Uploaded guest artifact name: `henry-newapi-src-current.tar.gz`
- Uploaded guest SHA256 sidecar name: `henry-newapi-src-current.tar.gz.sha256`
- Package SHA256 for that run: `70C5737104CF86F894E17DFB0333E95FBCF8BECE002F9868BFF744534FD679C9`
- Guest upload path used: `/home/newapi/henry-newapi-src-current.tar.gz`
- Guest workspace used: `/home/newapi/henry-newapi`
- Host-side guest SSH mapping used: `127.0.0.1:22222 -> guest:22`

## 2026-06-30 historical targeted Go verification

- Status: passed
- Target: `/66` Debian guest via host-side mapping `127.0.0.1:22222 -> guest:22`
- Required command source: `docs/henry-66-guest-verification-commands.md`

### Captured evidence

- Guest SHA256 matched the uploaded package SHA sidecar for `70C5737104CF86F894E17DFB0333E95FBCF8BECE002F9868BFF744534FD679C9`.
- `go version` reported `go1.25.1 linux/amd64`.
- Minimal probe/restore slice passed:
  - `go test ./service -run 'TestGetHttpClientForContext|TestCloneHTTPClientWithConnectTimeoutPreservesExistingDialContext|TestRunRoutingAutomationOnceProbe' -count=1 -v`
- Broader rerun passed:
  - `go test ./service -run 'DetectRoutingChannelRole|RoutingStatusCodeMapping|ChannelSupportsRequestPath|ChannelRuntimeHealthCooldown|RunRoutingAutomationOnce|ChannelAffinity|TestGetHttpClientForContext|TestCloneHTTPClientWithConnectTimeoutPreservesExistingDialContext' -count=1 -v`
  - `go test ./middleware -run 'SpecificChannel|AffinityChannel' -count=1 -v`
  - `go test ./controller -run 'RoutingAutomation|ShouldRetry|LockedTaskChannel' -count=1 -v`
  - `go test ./model -run 'RoutingPolicy|GetRandomSatisfiedChannelSkipsUnhealthyAndFailOpens|GetChannelSkipsUnhealthyAndFailOpens' -count=1 -v`
  - `go test ./relay/channel -run TestDoesNotExist -count=1`

### Historical notes

- The first minimal rerun exposed an outdated restore-timing test assumption, not a runtime failure.
- The local test update aligned old restore tests with the new `ProbeRetrySeconds` / `ActiveProbeIntervalSeconds` scheduling behavior before the corrected rerun passed.

## 2026-06-30 historical guest smoke

- Status: passed
- Required command source: `docs/henry-66-guest-smoke-checklist.md`
- Actual smoke date: `2026-06-30`
- Runtime target: guest-local Docker Compose inside `/66` Debian guest

### Captured evidence

- The first smoke attempt exposed a guest runtime dependency on pulling `postgres:15` and `redis:latest` from Docker Hub; guest diagnostics showed normal routing and DNS, but the Docker daemon timed out while awaiting registry headers.
- The accepted rerun stayed guest-local and used a disposable full smoke compose file `docker-compose.henry-smoke.full.yml` with:
  - current guest-built `./new-api` bind-mounted to `/new-api:ro`
  - cached guest-local `postgres:16-alpine` and `redis:7-alpine` images
  - loopback-only exposure `127.0.0.1:13000 -> 3000`
- Rendered compose config was captured at `/home/newapi/henry-newapi-compose-rendered.yml` and copied back to the `/66` host as `C:\Users\administrator\Downloads\henry-newapi\henry-newapi-compose-rendered.yml`.
- `docker compose -f docker-compose.henry-smoke.full.yml ps` showed:
  - `new-api` exposed only on `127.0.0.1:13000->3000/tcp`
  - `postgres` on guest-internal `5432/tcp`
  - `redis` on guest-internal `6379/tcp`
- `http://127.0.0.1:13000/api/status` returned `success=true` on that snapshot.
- Disposable root/admin setup was completed through `POST /api/setup`, and follow-up `GET /api/setup` plus `GET /api/status` showed `setup=true`.
- Windows host `http://127.0.0.1:23000/api/status` was unreachable during the accepted isolated smoke, confirming the older NAT-forwarded path was not reused for that snapshot.
- No Windows host MySQL `3306` or Python `5000` dependency was introduced.
- After evidence capture, the guest smoke stack was rolled back with `docker compose -f docker-compose.henry-smoke.full.yml down -v`.

## Historical merged probe/restore slice

The historically rerun routing slice corresponds to:

- `controller/routing_policy_probe.go`
- `relay/channel/api_request.go`
- `routingpolicy/runtime.go`
- `service/http_client.go`
- `service/routing_policy_test.go`
- `service/http_client_test.go`

Those changes are now part of `main` and are no longer a pending local-only increment.

## Remaining boundaries

- The trusted runtime evidence still belongs to the `/66` Debian guest, not this local Windows machine.
- The accepted `2026-06-30` evidence remains valid as historical proof for the merged routing slice.
- The latest open operational question is whether `/66` is currently reachable and staged, not whether the old routing slice landed in Git.
- If any `go` source or test changes again, the package must be re-synced and rerun on `/66`.
- If runtime wiring or guest smoke behavior changes again, both targeted Go verification and guest smoke must be rerun.

## Evidence refresh checklist for the next `/66` rerun

The next accepted `/66` rerun should again capture:

- rendered compose config path and loopback-only exposure evidence
- compose `ps` result for `new-api`, `postgres`, and `redis`
- `http://127.0.0.1:13000/api/status` success payload
- setup completion evidence for the guest stack, or the exact blocker
- confirmation that host `3306`, `5000`, and the old `23000` runtime path are not part of the accepted runtime dependency chain
