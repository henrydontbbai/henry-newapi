# Henry NewAPI Native Routing Strategy - Current Mainline and /66 Staging Gap

## Summary

As of `2026-07-04`, the routing-policy work that was validated on `/66` is no longer a pending local slice. The relevant code has already landed on `main`, and the repository now has stable CI coverage for the surrounding areas.

### Current known state

- The native routing baseline is on `main` via `8cfec81f feat: add native routing policy baseline and /66 verification docs`.
- The probe/restore closure slice is on `main` via `2c4e9354 feat: close probe restore loop and refresh /66 verification evidence`.
- The restore-state cleanup follow-up is on `main` via `d12c0630 fix: clear pending restore state on routing success`.
- The engineering closure around these changes is on `main` and green, including `Go CI`, `Frontend CI`, and `Release Dry Run` on commit `d95aa5ca`.

### Current unknown state

- The latest trusted `/66` runtime evidence is still the historical `2026-06-30` Debian guest verification snapshot.
- `/66` has not yet been re-audited as a living preprod/staging environment after the later mainline and CI closure work.
- This local Windows machine still does not provide the trusted Go/Docker runtime used for `/66` verification; the trusted runtime target remains the Debian guest inside `/66`.

## Mainline Routing Scope Already Landed

The following runtime capabilities are already part of the current branch history and must be treated as shipped code, not as a pending increment:

- native settings for mode, role detection, subscription policy, probe policy, paygo nudge, paygo hard-failure, and slow-channel policy
- runtime cooldown / success / failure tracking
- request-path health gating across cache / DB / affinity / retry / task relay / `specific_channel_id` / locked-channel reuse
- request-context connect-timeout propagation for routing probes and request-path HTTP client selection
- restore lifecycle state such as waiting, failed-hold, restored, next-probe timing, and last probe/restore markers
- follow-up `MarkSuccess` cleanup for stale restore fields without wiping `LastProbeAt` and `LastRestoreAt`

The merged probe/restore slice still maps to the same source/tests that were verified on `/66`:

- `controller/routing_policy_probe.go`
- `relay/channel/api_request.go`
- `routingpolicy/runtime.go`
- `service/http_client.go`
- `service/routing_policy_test.go`
- `service/http_client_test.go`

## Historical /66 Verification Record

The real verification target for the routing slice remains the Debian guest inside `/66`, not the Windows host itself.

Historical verified facts from `2026-06-30`:

- host-side guest SSH mapping: `127.0.0.1:22222 -> guest:22`
- guest Go version: `go1.25.1 linux/amd64`
- current package SHA256 at that time: `70C5737104CF86F894E17DFB0333E95FBCF8BECE002F9868BFF744534FD679C9`
- targeted Go slices passed for:
  - `service` minimal probe/http-client slice
  - `service` broader routing slice
  - `middleware`
  - `controller`
  - `model`
  - `relay/channel` compile-level check
- guest smoke passed with:
  - guest-local `postgres:16-alpine` and `redis:7-alpine`
  - guest-built `./new-api`
  - loopback-only `127.0.0.1:13000 -> 3000`
  - successful `GET /api/status`
  - disposable root/admin setup completion evidence

That snapshot is still the latest trusted runtime evidence, but it should now be read as historical evidence for code that is already on `main`, not as proof that `/66` is currently online.

## Current /66 Staging Gap

What remains is operational, not feature-design work:

- confirm whether `/66` is currently reachable as a host/guest chain
- classify `/66` as `offline`, `reachable-but-not-deployed`, `staging-running`, or `staging-drifted`
- replace the old disposable smoke-only path with a repeatable staging runbook and tracked override file (`docs/henry-66-staging-runbook.md`, `docker-compose.henry-staging.override.yml`)
- keep the first recovered staging run in the safest mode:
  - `routing_policy_setting.mode=observe`
  - `probe_policy.active_probe_enabled=false`
- avoid Windows-host dependencies (`3306`, `5000`, old `23000`) and keep loopback-only guest exposure

## Verification Rule

From this point forward:

- if only Markdown docs or staging/release runbook notes change, no `/66` rerun is required
- if any `go` source or test file changes, rerun targeted `/66` Go verification
- if runtime wiring, Compose wiring, or guest smoke behavior changes, rerun targeted `/66` Go verification and guest Docker Compose smoke

## Next Iteration Order

The next implementation round should stay narrow and follow this order:

1. align `/66` docs with the current mainline state
2. close `/66` as a repeatable preprod/staging environment
3. move real release workflows to fork-owned targets before any side-effectful release verification
4. then resume routing hardening in this order:
   - direct request-path tests for `probe -> testChannel -> doRequest -> GetHttpClientForContext`
   - proxy-path coverage for the same timeout propagation path
   - stronger connect-timeout cancellation semantics for custom dialers and SOCKS-backed paths
   - broader restore-state matrix coverage, especially slow-channel failed-hold and multi-snapshot timing edges
