# Henry NewAPI Submit Package Boundary

This file defines the current submit-package boundary for the native routing strategy slice after the completed `/66` guest verification.

## Include

- Native routing source:
  - `setting/operation_setting/routing_policy_setting.go`
  - `routingpolicy/runtime.go`
  - `service/routing_policy.go`
  - `model/routing_policy_hooks.go`
  - `controller/routing_policy_probe.go`
- Routing-related wiring and integration:
  - `controller/relay.go`
  - `controller/system_task_handlers.go`
  - `middleware/distributor.go`
  - `model/ability.go`
  - `model/channel_cache.go`
  - `model/system_task.go`
  - `service/channel_affinity.go`
- In-scope tests:
  - `setting/operation_setting/routing_policy_setting_test.go`
  - `service/routing_policy_test.go`
  - `controller/relay_retry_test.go`
  - `controller/routing_system_task_test.go`
  - `middleware/routing_policy_distributor_test.go`
  - `model/routing_policy_filter_test.go`
  - `service/channel_affinity_template_test.go`
  - `service/task_billing_test.go`
- Verification and evidence docs:
  - `docs/henry-realtime-channel-health-gate-plan.md`
  - `docs/henry-66-acceptance-matrix.md`
  - `docs/henry-66-runtime-evidence-log.md`
  - `docs/henry-66-guest-verification-commands.md`
  - `docs/henry-66-guest-smoke-checklist.md`
  - `docs/henry-66-audit-agent-brief.md`
  - `docs/henry-submit-package-plan.md`

## Hold

- Local package artifacts:
  - `henry-newapi-src-current.tar.gz`
  - `henry-newapi-src-current.tar.gz.sha256`
  - `henry-newapi-src.tar.gz`
- Local screenshot artifact:
  - `tmp-current-guest-screen.png`
- Any future transient package, screenshot, or guest-only residue generated only for local transfer or inspection

## Watch

- `docs/henry-realtime-channel-health-gate-plan.md`
  - Keep as the master status/spec doc for this slice, but do not let it drift back into "unverified" wording.
- `docs/henry-66-acceptance-matrix.md`
  - Keep as a compact gate summary, not as a mixed "todo + done" checklist.
- `docs/henry-66-runtime-evidence-log.md`
  - Keep as captured evidence log; if a future rerun happens, append new facts instead of replacing the current verification history.

## Verification Rule

- If only Markdown docs in this package change, no `/66` rerun is required.
- If any `go` source or test file changes, repackage and rerun the targeted `/66` Go verification.
- If runtime wiring or guest smoke behavior changes, rerun both targeted `/66` Go verification and guest Docker Compose smoke.

## Current Evidence Snapshot

- `/66` Debian guest targeted Go verification: passed
- `/66` guest Docker Compose smoke: passed
- Exposure mode during final smoke: loopback-only `127.0.0.1:13000 -> 3000`
- No commit, push, PR, or deployment performed yet

## Suggested Submit Message

- `feat: add native routing policy baseline and /66 verification docs`
