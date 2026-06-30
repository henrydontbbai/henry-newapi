# Henry /66 Runtime Evidence Log

This file records the current `/66` evidence for the in-flight probe/restore increment on top of the already-pushed native-routing baseline.

As of `2026-06-30`, the current snapshot has:

- completed `/66` Debian guest targeted Go rerun evidence;
- refreshed local package SHA evidence;
- completed current-snapshot guest smoke evidence;
- no commit, push, PR, or deployment for this increment.

## Current Local Package

- Local package file used for this run: `henry-newapi-src-current.tar.gz`
- Local SHA256 sidecar used for this run: `henry-newapi-src-current.tar.gz.sha256`
- Uploaded guest artifact name: `henry-newapi-src-current.tar.gz`
- Uploaded guest SHA256 sidecar name: `henry-newapi-src-current.tar.gz.sha256`
- Current package SHA256: `70C5737104CF86F894E17DFB0333E95FBCF8BECE002F9868BFF744534FD679C9`
- Guest upload path used: `/home/newapi/henry-newapi-src-current.tar.gz`
- Guest workspace used: `/home/newapi/henry-newapi`
- Current host-side guest SSH mapping: `127.0.0.1:22222 -> guest:22`

## 2026-06-30 Current-Snapshot Go Verification

- Status: passed
- Target: `/66` Debian guest via host-side mapping `127.0.0.1:22222 -> guest:22`
- Required command source: `docs/henry-66-guest-verification-commands.md`

### Captured Evidence

- Guest SHA256 matched the uploaded package SHA sidecar for `70C5737104CF86F894E17DFB0333E95FBCF8BECE002F9868BFF744534FD679C9`.
- `go version` reported `go1.25.1 linux/amd64`.
- Minimal probe/restore slice passed:
  - `go test ./service -run 'TestGetHttpClientForContext|TestCloneHTTPClientWithConnectTimeoutPreservesExistingDialContext|TestRunRoutingAutomationOnceProbe' -count=1 -v`
- Broader current-snapshot rerun passed:
  - `go test ./service -run 'DetectRoutingChannelRole|RoutingStatusCodeMapping|ChannelSupportsRequestPath|ChannelRuntimeHealthCooldown|RunRoutingAutomationOnce|ChannelAffinity|TestGetHttpClientForContext|TestCloneHTTPClientWithConnectTimeoutPreservesExistingDialContext' -count=1 -v`
  - `go test ./middleware -run 'SpecificChannel|AffinityChannel' -count=1 -v`
  - `go test ./controller -run 'RoutingAutomation|ShouldRetry|LockedTaskChannel' -count=1 -v`
  - `go test ./model -run 'RoutingPolicy|GetRandomSatisfiedChannelSkipsUnhealthyAndFailOpens|GetChannelSkipsUnhealthyAndFailOpens' -count=1 -v`
  - `go test ./relay/channel -run TestDoesNotExist -count=1`

### Notes

- The first minimal rerun exposed an outdated restore-timing test assumption, not a runtime failure.
- The local test update aligned old restore tests with the new `ProbeRetrySeconds` / `ActiveProbeIntervalSeconds` scheduling behavior before the corrected rerun passed.

## Current-Snapshot Guest Smoke

- Status: passed
- Required command source: `docs/henry-66-guest-smoke-checklist.md`
- Actual smoke date: `2026-06-30`
- Runtime target: guest-local Docker Compose inside `/66` Debian guest

### Captured Evidence

- The first smoke attempt exposed a guest runtime dependency on pulling `postgres:15` and `redis:latest` from Docker Hub; guest diagnostics showed normal routing and DNS, but the Docker daemon timed out while awaiting registry headers.
- The accepted rerun stayed guest-local and used a disposable full smoke compose file `docker-compose.henry-smoke.full.yml` with:
  - current guest-built `./new-api` bind-mounted to `/new-api:ro`;
  - cached guest-local `postgres:16-alpine` and `redis:7-alpine` images;
  - loopback-only exposure `127.0.0.1:13000 -> 3000`.
- Rendered compose config was captured at `/home/newapi/henry-newapi-compose-rendered.yml` and copied back to the `/66` host as `C:\Users\administrator\Downloads\henry-newapi\henry-newapi-compose-rendered.yml`.
- `docker compose -f docker-compose.henry-smoke.full.yml ps` showed:
  - `new-api` exposed only on `127.0.0.1:13000->3000/tcp`;
  - `postgres` on guest-internal `5432/tcp`;
  - `redis` on guest-internal `6379/tcp`.
- `http://127.0.0.1:13000/api/status` returned `success=true` on the current snapshot.
- Disposable root/admin setup was completed through `POST /api/setup`, and follow-up `GET /api/setup` plus `GET /api/status` showed `setup=true`.
- Windows host `http://127.0.0.1:23000/api/status` was unreachable during the accepted isolated smoke, confirming the older NAT-forwarded path was not reused for this snapshot.
- No Windows host MySQL `3306` or Python `5000` dependency was introduced.
- After evidence capture, the guest smoke stack was rolled back with `docker compose -f docker-compose.henry-smoke.full.yml down -v`.

## Current Increment Boundary

The current unsubmitted probe/restore increment is the following six-file set:

- `controller/routing_policy_probe.go`
- `relay/channel/api_request.go`
- `routingpolicy/runtime.go`
- `service/http_client.go`
- `service/routing_policy_test.go`
- `service/http_client_test.go`

## Remaining Boundaries

- The trusted runtime evidence remains the `/66` Debian guest, not this local Windows machine.
- Current-snapshot read-only audit acceptance is complete with notes.
- Audit note: later evidence readback from the `/66` Windows host remained intermittently flaky at the SSH transport layer, but the accepted evidence set already included matching package SHA, `go-targeted.rc = 0`, guest smoke `ps`, `status`, and disposable setup success.
- No commit, push, PR, or production-style deployment has been performed for this increment.

## Evidence Refresh Checklist After Smoke

The current snapshot now has the required smoke facts captured:

- compose render path and loopback-only exposure evidence
- compose `ps` result for `new-api`, `postgres`, and `redis`
- `http://127.0.0.1:13000/api/status` success payload
- setup completion evidence for the disposable guest stack
- confirmation that host `3306`, `5000`, and the older `23000` runtime path were not used as accepted runtime dependencies
