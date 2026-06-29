# Henry NewAPI Realtime Channel Health Gate Plan

## Summary

This repository is Henry's fork of `QuantumNous/new-api` for a minimal NewAPI core modification. The goal is not to replace the NAS watcher. The watcher remains responsible for long-period probing, recovery, health summaries, and operations visibility.

The NewAPI core change should only add a request-path health gate:

- before selecting or reusing a channel, skip channels that are in short runtime cooldown;
- after an upstream/provider-side failure, mark that channel in short cooldown immediately;
- after success, record success and clear lightweight runtime failure state;
- preserve existing channel `status`, `priority`, `weight`, keys, providers, model routing, AiMaMi behavior, and NAS watcher behavior.

The first implementation should ship as observe-only or disabled-by-default, then move to enforcement after local and NAS dry-run evidence is available.

## Current Source Findings

The current branch is based on upstream `main`. The relevant source paths are:

- `service/channel_select.go`: `CacheGetRandomSatisfiedChannel` selects a channel for retry-aware relay paths.
- `model/channel_cache.go`: `GetRandomSatisfiedChannel` handles memory-cache channel selection by model, request path, priority, and weight.
- `model/ability.go`: `GetChannel` handles non-memory-cache database selection and must get equivalent filtering.
- `controller/relay.go`: normal relay and task relay both retry and call `processChannelError` after channel failures.
- `middleware/distributor.go`: initial distribution may use channel affinity before random selection, so affinity-selected channels must also pass the health gate.
- `service/channel.go`: `ShouldDisableChannel` and `DisableChannel` implement existing hard-disable/status-oriented behavior and should not be reused as the runtime short-cooldown layer.
- `model/log.go`: `RecordErrorLog` already records channel id, request path, status code, error type/code, and admin info; the runtime gate should not depend on log polling for request-path behavior.

Important behavior already present:

- Normal relay retries in `controller/relay.go` call `getChannel` again on each retry.
- Task relay has a separate retry loop and also calls `processChannelError`.
- `model.GetRandomSatisfiedChannel` chooses candidates by exact/normalized model and request path, then by descending priority and weight.
- Channel affinity can select a channel before random selection, which can otherwise keep traffic stuck on a bad channel.
- Memory-cache and DB selection paths both exist, so v1 must keep behavior consistent in both modes.

## Implementation Plan

### Runtime Health State

Add a small runtime health module, preferably in `service` unless implementation needs lower-level model access:

- store per-channel state keyed by channel id;
- fields: `cooldown_until`, `reason`, `last_error_at`, `last_success_at`, `failure_count`, `success_count`, `last_status_code`;
- use in-memory state for v1;
- if Redis is enabled, Redis persistence can be a later phase, not required for v1;
- default mode: disabled or observe-only through one explicit option or environment-backed setting.

Do not add database migrations for v1.

### Channel Selection Gate

Add one shared predicate:

```text
IsChannelRuntimeHealthy(channel_id, now) -> healthy, reason
```

Apply it in all selection paths:

- memory-cache path in `model/channel_cache.go`;
- DB path in `model/ability.go`;
- affinity path in `middleware/distributor.go` before accepting a preferred channel.

Filtering rule:

- if the gate is disabled, return original behavior;
- if observe-only, log/record what would have been skipped but still return original behavior;
- if enforce mode, remove channels whose `cooldown_until` is in the future;
- if filtering removes every candidate in the current candidate set, fail open and use the original candidates to avoid a self-inflicted outage.

Do not change priority or weight. Filtering happens before the existing priority/weight random choice.

### Failure And Success Updates

Hook failure updates in `controller/relay.go`:

- normal relay: after `processChannelError` receives a channel failure;
- task relay: same `processChannelError` path for non-local task errors;
- classify only provider/upstream-side failures;
- ignore local/client/request validation errors and skip-retry user quota errors.

Hook success updates:

- normal relay: immediately before returning on `newAPIError == nil`;
- task relay: when `taskErr == nil` after successful submit;
- record success for the channel used by the successful attempt.

V1 cooldown classes:

```text
short: timeout/status=000, 408, high demand, temporary upstream overload
medium: 429, concurrency limit, too many pending requests, 503
long: quota/balance/group disabled/subscription not found/migrated
ignored: client cancel, invalid request, user quota/preconsume, local request body errors
```

Exact durations should be conservative and configurable. Suggested starting defaults:

```text
short = 60s
medium = 180s
long = 1800s
```

### Watcher Boundary

Keep these outside NewAPI core:

- probe-only recovery;
- channel status changes for long-term quarantine;
- health summary generation;
- slow-channel governance;
- paygo nudge;
- reason tag backfill;
- NAS deployment and cron monitoring.

The NewAPI core only makes immediate request-path routing decisions. It should not become a replacement operations watcher.

## Test Plan

Minimum tests before implementation is considered ready:

- unit test error classification: 429, concurrency limit, pending requests, 503, timeout, quota, group disabled, subscription not found, client cancel, invalid request;
- unit test runtime health state: mark failure, cooldown active, cooldown expires, success clears lightweight failure state;
- selection test for memory-cache path: unhealthy candidate is skipped while priority/weight behavior remains intact for healthy candidates;
- selection test for DB path: unhealthy ability candidate is skipped equivalently;
- affinity test: preferred unhealthy channel is not accepted in enforce mode and affinity cache can fall through to normal selection;
- fail-open test: if all candidates are unhealthy, selection still returns an original candidate instead of failing with no channel;
- relay-level test or focused integration test: a failed attempt marks cooldown and the next retry does not reuse that channel when another candidate exists.

Suggested local commands after implementation:

```bash
go test ./service ./model ./controller ./middleware
```

If `go` is unavailable on the machine, report that explicitly and run static/source checks only.

## Rollout And Acceptance

Rollout sequence:

1. Add runtime state and classification tests with gate disabled.
2. Add observe-only mode and confirm logs/metrics show expected skip decisions.
3. Enable enforce mode in local/staging only.
4. Compare user-facing errors, retry behavior, and watcher summary before any NAS production change.
5. Only after a separate authorization, build/deploy the forked NewAPI image.

Acceptance criteria:

- repository remains a fork with `origin` pointing to `henrydontbbai/henry-newapi` and `upstream` pointing to `QuantumNous/new-api`;
- work happens on `codex/realtime-channel-health-gate`, not directly on `main`;
- no core implementation changes are made during the planning phase;
- the implementation plan is based on current source files and covers memory-cache selection, DB selection, retry failure handling, success handling, and channel affinity;
- the plan explicitly keeps watcher responsibilities out of NewAPI core;
- upstream push is disabled locally to reduce accidental writes to the official repository.

## Non-Goals

- Do not modify AiMaMi, provider routing, model routing, proxy, relay config, or global Codex/user configuration.
- Do not change channel keys, priority, weight, or status for this v1 runtime gate.
- Do not migrate NAS watcher logic into NewAPI.
- Do not deploy, restart services, or touch NAS without separate current-turn authorization.
