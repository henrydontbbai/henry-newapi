# Henry NewAPI Submit Package Boundary

This file defines the current submit-package boundary for the probe/restore increment on top of the already-pushed native-routing baseline.

## Include

- Existing native-routing baseline source and tests already tracked for this slice.
- Current increment source:
  - `controller/routing_policy_probe.go`
  - `relay/channel/api_request.go`
  - `routingpolicy/runtime.go`
  - `service/http_client.go`
- Current increment tests:
  - `service/routing_policy_test.go`
  - `service/http_client_test.go`
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
  - Keep as the master status/spec doc for this slice; it should distinguish the pushed baseline from the current unsubmitted increment.
- `docs/henry-66-acceptance-matrix.md`
  - Keep as the current-snapshot gate summary, not a mixed historical checklist.
- `docs/henry-66-runtime-evidence-log.md`
  - Keep as the captured evidence log for this snapshot; append future rerun facts instead of silently reusing old smoke claims.

## Verification Rule

- If only Markdown docs in this package change, no `/66` rerun is required.
- If any `go` source or test file changes, repackage and rerun the targeted `/66` Go verification.
- If runtime wiring or guest smoke behavior changes, rerun both targeted `/66` Go verification and guest Docker Compose smoke.

## Current Evidence Snapshot

- `/66` Debian guest targeted Go verification for the current snapshot: passed
- Current package SHA256: `70C5737104CF86F894E17DFB0333E95FBCF8BECE002F9868BFF744534FD679C9`
- `/66` guest Docker Compose smoke for the current snapshot: passed
- Current-snapshot read-only audit acceptance: passed with notes
- No commit, push, PR, or deployment performed for this increment

## Suggested Submit Message

- `feat: close probe restore loop and refresh /66 verification evidence`
