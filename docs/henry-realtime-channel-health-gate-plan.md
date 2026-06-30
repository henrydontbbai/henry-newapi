# Henry NewAPI Native Routing Strategy - Baseline Plus Current Increment

## Summary

This repository is no longer in a planning-only phase.

There are now two distinct states that must not be mixed:

1. the already-pushed native-routing baseline;
2. the current unsubmitted `2026-06-30` probe/restore increment.

Current state as of `2026-06-30`:

- native config entry exists: `routing_policy_setting`
- native runtime state exists: `routingpolicy` summary, state, and snapshot cache
- native scheduler entry exists: `routing_automation`
- request-path wiring is connected across cache / DB / affinity / retry / task relay / `specific_channel_id` / locked-channel reuse
- current probe/restore increment adds request-context connect-timeout propagation and more explicit restore lifecycle state
- current-snapshot `/66` targeted Go verification passed
- current-snapshot `/66` guest Docker Compose smoke passed on an isolated guest-only rerun

What remains is not new feature design. Audit closure is now complete; only submit-package closure and any later Git authorization remain for the current increment.

## Implemented Baseline

The pushed baseline already includes:

- native settings for mode, role detection, subscription policy, probe policy, paygo nudge, paygo hard-failure, and slow-channel policy
- native runtime cooldown / success / failure tracking
- request-path health gating for:
  - memory-cache channel selection
  - DB ability selection
  - affinity-selected channels
  - normal relay retry
  - task relay retry
  - `specific_channel_id`
  - locked-channel reuse
- native automation for:
  - slow subscription summary
  - paygo hard-failure isolate / probe restore
  - paygo nudge hold / restore
  - subscription transient isolate / probe restore

## Current Increment

The current unsubmitted increment is a narrower probe/restore closure slice:

- `controller/routing_policy_probe.go`
  - propagate probe connect-timeout through request context
- `relay/channel/api_request.go`
  - use context-aware HTTP client selection on the request path
- `service/http_client.go`
  - clone HTTP clients with context-driven connect timeout while preserving existing dialer behavior
- `routingpolicy/runtime.go`
  - expose restore lifecycle state such as waiting, failed-hold, restored, next-probe timing, and last probe/restore markers
- `service/routing_policy_test.go`
  - add restore-state and probe scheduling coverage
- `service/http_client_test.go`
  - add direct tests for context-aware HTTP client timeout behavior

This increment is code-complete enough for targeted rerun and now has refreshed smoke evidence; the remaining closure work is submit-boundary cleanup and any later Git decision.

## /66 Verification State

The real verification target remains the Debian guest inside `/66`, not the Windows host itself.

Current verified facts for this snapshot:

- current package SHA256: `70C5737104CF86F894E17DFB0333E95FBCF8BECE002F9868BFF744534FD679C9`
- host-side guest SSH mapping: `127.0.0.1:22222 -> guest:22`
- guest Go version used: `go1.25.1 linux/amd64`
- current-snapshot targeted Go slices passed for:
  - `service` minimal probe/http-client slice
  - `service` broader routing slice
  - `middleware`
  - `controller`
  - `model`
  - `relay/channel` compile-level check
- current-snapshot guest Docker Compose smoke passed with:
  - guest-local `postgres:16-alpine` and `redis:7-alpine`;
  - current guest-built `./new-api` bind-mounted into `calciumion/new-api:latest`;
  - loopback-only `127.0.0.1:13000 -> 3000`;
  - successful `GET /api/status`;
  - disposable root/admin setup completion evidence.

Earlier loopback-only smoke evidence for the pushed baseline is now superseded by current-snapshot smoke evidence for this request-path increment.

## Submit Boundary

The current branch still needs submit-package collection before any Git write step for this increment.

Submit boundary rules:

- include:
  - the six current increment files
  - the corresponding evidence refresh docs
- hold locally:
  - `henry-newapi-src-current.tar.gz`
  - `henry-newapi-src-current.tar.gz.sha256`
  - `henry-newapi-src.tar.gz`
  - `tmp-current-guest-screen.png`
- do not widen scope into:
  - frontend settings panel
  - DB migration
  - watcher / cron / NAS sidecars
  - provider or model-routing changes
  - deployment work

The concrete include / hold / watch list is tracked in `docs/henry-submit-package-plan.md`.

## Remaining Limits

The current state is strong enough to claim refreshed runtime proof for the increment and has now cleared read-only audit acceptance for this snapshot.

Open limits:

- this local Windows machine still has no `go` or `docker`
- the trusted runtime evidence belongs to the `/66` Debian guest, not this local machine
- the current increment still needs submit-boundary confirmation before any Git write step
- if any `go` source or test changes again, the package must be re-synced and re-run on `/66`
- if runtime wiring or smoke behavior changes again after that, both targeted Go verification and guest smoke must be rerun

## Verification Rule

From this point forward:

- if only Markdown docs or submit-package notes change, no `/66` rerun is required
- if any `go` source or test file changes, rerun targeted `/66` Go verification
- if runtime wiring or guest smoke behavior changes, rerun targeted `/66` Go verification and guest Docker Compose smoke

## Next Iteration Order

After the current increment is closed with refreshed smoke evidence, the next round should stay narrow and follow this order:

1. add direct request-path tests for the `probe -> testChannel -> doRequest -> GetHttpClientForContext` chain;
2. add proxy-path coverage for the same timeout propagation path;
3. harden connect-timeout cancellation semantics for custom dialers and SOCKS-backed paths;
4. expand restore-state matrix coverage, especially slow-channel failed-hold and multi-snapshot timing edge cases;
5. only after that, reconsider broader routing knobs such as `force_threshold`, `weight_degrade_enabled`, `confirm_window_minutes`, and `auto_disable_cooldown_seconds`.
