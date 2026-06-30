# Henry NewAPI Native Routing Strategy - Current State And Submit Boundary

## Summary

This branch is no longer in a planning-only phase. The native routing strategy slice has already been implemented and verified on the `/66` Debian guest.

Current verified state:

- native config entry exists: `routing_policy_setting`
- native runtime state exists: `routingpolicy` summary, state, and snapshot cache
- native scheduler entry exists: `routing_automation`
- request-path wiring is already connected across cache / DB / affinity / retry / task relay / `specific_channel_id` / locked-channel reuse
- `/66` targeted Go verification passed
- `/66` guest Docker Compose smoke passed with loopback-only exposure

What remains is submit-package cleanup, not first-time runtime proof.

## Implemented Baseline

The current routing slice already includes:

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

Current behavior contract:

- default rollout mode remains `observe`
- `observe` logs but does not mutate routing result
- `enforce` skips unhealthy channels
- all-filtered candidate sets fail open to the original set
- `specific_channel_id` does not silently reroute in enforce mode
- locked unhealthy channels fail explicitly

## /66 Verification State

The real verification target is the Debian guest inside `/66`, not the Windows host itself.

Verified facts captured in the paired evidence docs:

- guest Go version used: `go1.25.1 linux/amd64`
- targeted Go test slices passed for:
  - `setting/operation_setting`
  - `service`
  - `middleware`
  - `controller`
  - `model`
- guest Docker Compose smoke passed
- final exposure mode stayed loopback-only at `127.0.0.1:13000 -> 3000`
- guest smoke did not depend on Windows host MySQL `3306` or Python `5000`
- no commit, push, PR, or deployment has been performed

The detailed gate summary is tracked in `docs/henry-66-acceptance-matrix.md`.
The detailed captured evidence is tracked in `docs/henry-66-runtime-evidence-log.md`.

## Current Submit Boundary

The current branch still needs submit-package collection before any Git write step.

Submit boundary rules:

- include:
  - native routing source
  - in-scope routing tests
  - `/66` verification and evidence docs
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

The current state is strong enough for submit-package confirmation, but it is not a release/deploy signoff.

Open limits:

- this local Windows machine still has no `go` or `docker`
- the trusted runtime evidence belongs to the `/66` Debian guest, not this local machine
- if any `go` source or test changes again, the package must be re-synced and re-run on `/66`
- if runtime wiring or smoke behavior changes, both targeted Go verification and guest smoke must be rerun

## Verification Rule

From this point forward:

- if only Markdown docs or submit-package notes change, no `/66` rerun is required
- if any `go` source or test file changes, rerun targeted `/66` Go verification
- if runtime wiring or guest smoke behavior changes, rerun targeted `/66` Go verification and guest Docker Compose smoke
