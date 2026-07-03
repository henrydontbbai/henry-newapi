# Henry NewAPI /66 Reverification Boundary

This file no longer describes a pending submit package. As of `2026-07-04`, the `2026-06-30` routing probe/restore slice is already part of `main`. The purpose of this file is now to preserve the verified boundary of that historical slice and to define when `/66` must be rerun again.

## Historical merged slice

The verified routing slice that was historically packaged and rerun on `/66` corresponds to:

- `controller/routing_policy_probe.go`
- `relay/channel/api_request.go`
- `routingpolicy/runtime.go`
- `service/http_client.go`
- `service/routing_policy_test.go`
- `service/http_client_test.go`

Mainline outcomes tied to that slice:

- `8cfec81f feat: add native routing policy baseline and /66 verification docs`
- `2c4e9354 feat: close probe restore loop and refresh /66 verification evidence`
- `d12c0630 fix: clear pending restore state on routing success`

## Keep tracked

Keep these tracked docs as the canonical record for the historical `/66` verification and for future rerun rules:

- `docs/henry-realtime-channel-health-gate-plan.md`
- `docs/henry-66-acceptance-matrix.md`
- `docs/henry-66-runtime-evidence-log.md`
- `docs/henry-66-guest-verification-commands.md`
- `docs/henry-66-guest-smoke-checklist.md`
- `docs/henry-66-audit-agent-brief.md`
- `docs/henry-submit-package-plan.md`

## Hold locally

Do not commit transient transfer or inspection artifacts:

- `henry-newapi-src-current.tar.gz`
- `henry-newapi-src-current.tar.gz.sha256`
- `henry-newapi-src.tar.gz`
- `tmp-current-guest-screen.png`
- any future one-off package, screenshot, or guest-only residue generated only for transfer/audit work

## Current state as of 2026-07-04

- the historical `/66` targeted Go verification passed
- the historical `/66` guest smoke passed
- the historical read-only audit acceptance passed with notes
- the code tied to that evidence is already merged on `main`
- the current unknown is not code inclusion; it is whether `/66` is presently online and behaving as a staging environment
- there is no remaining submit-boundary or Git-authorization task for the old slice itself

## Verification rule

- If only Markdown docs or runbook notes change, no `/66` rerun is required.
- If any `go` source or test file changes, rerun the targeted `/66` Go verification.
- If runtime wiring, Compose wiring, or guest smoke behavior changes, rerun both the targeted `/66` Go verification and guest Docker Compose smoke.

## Watch points for future reruns

- Keep `docs/henry-realtime-channel-health-gate-plan.md` as the master status/spec doc for routing plus `/66` staging gaps.
- Keep `docs/henry-66-acceptance-matrix.md` as the historical gate summary plus current known/unknown state.
- Keep `docs/henry-66-runtime-evidence-log.md` append-only for future `/66` reruns; do not silently reuse old smoke claims as current runtime proof.
