# Henry NewAPI /66 Acceptance Matrix

This matrix preserves the accepted `2026-06-30` `/66` verification snapshot and explains how that snapshot relates to the current repository state.

## Current status summary (`2026-07-04`)

### Known current state

- The routing baseline plus probe/restore closure work is already on `main`.
- Relevant merged commits:
  - `8cfec81f feat: add native routing policy baseline and /66 verification docs`
  - `2c4e9354 feat: close probe restore loop and refresh /66 verification evidence`
  - `d12c0630 fix: clear pending restore state on routing success`
- Mainline engineering closure is green, including `Go CI`, `Frontend CI`, and `Release Dry Run`.

### Current unknown state

- The latest trusted `/66` evidence is still the historical `2026-06-30` guest snapshot below.
- `/66` has not yet been reclassified as a live preprod/staging environment after those later mainline changes.

## Historical 2026-06-30 artifact context

- Source package used for that rerun: `henry-newapi-src-current.tar.gz`
- Historical package SHA256: `70C5737104CF86F894E17DFB0333E95FBCF8BECE002F9868BFF744534FD679C9`
- Local host limitation for that rerun: no `go`, no `docker`
- Remote target: Debian guest inside `/66`, not the Windows host
- Host-side guest SSH mapping used: `127.0.0.1:22222 -> guest:22`
- Verified guest Go path: `/home/newapi/go-1.25.1/bin/go`

## Historical gate matrix (accepted on 2026-06-30)

| Gate | Required evidence | Historical status |
| --- | --- | --- |
| Source package excludes unsafe local state | package listing excludes `.git`, `.env*`, `node_modules`, old tarballs, screenshots | Passed for the historical package |
| Targeted Go service probe/http-client slice | `/66` guest output for `TestGetHttpClientForContext`, `TestCloneHTTPClientWithConnectTimeoutPreservesExistingDialContext`, `TestRunRoutingAutomationOnceProbe` | Passed on `/66` Debian guest with Go `1.25.1` |
| Targeted Go service broader routing slice | `/66` guest output for the corrected service rerun including routing, cooldown, affinity, probe/restore coverage | Passed on `/66` Debian guest with Go `1.25.1` |
| Targeted Go middleware tests | `/66` guest output for `go test ./middleware -run 'SpecificChannel\|AffinityChannel' -count=1 -v` | Passed on `/66` Debian guest with Go `1.25.1` |
| Targeted Go controller tests | `/66` guest output for `go test ./controller -run 'RoutingAutomation\|ShouldRetry\|LockedTaskChannel' -count=1 -v` | Passed on `/66` Debian guest with Go `1.25.1` |
| Targeted Go model tests | `/66` guest output for `go test ./model -run 'RoutingPolicy\|GetRandomSatisfiedChannelSkipsUnhealthyAndFailOpens\|GetChannelSkipsUnhealthyAndFailOpens' -count=1 -v` | Passed on `/66` Debian guest with Go `1.25.1` |
| Relay/channel compile-level check | `/66` guest output for `go test ./relay/channel -run TestDoesNotExist -count=1` | Passed on `/66` Debian guest |
| Guest-only Docker smoke | guest output from `docs/henry-66-guest-smoke-checklist.md` showing compose up and `/api/status` on `127.0.0.1:13000` | Passed on `/66` Debian guest with loopback-only `127.0.0.1:13000->3000` |
| No Windows-host DB/service dependency | guest smoke showing compose Postgres/Redis only and no host `3306` / `5000` dependency | Passed on `/66`; host `23000` was also unreachable during the accepted isolated smoke |
| Root/admin setup gate | guest smoke either completes setup or records the exact setup blocker | Passed with disposable guest-only setup completion evidence |
| Routing observe safety | smoke or runtime evidence from that snapshot that observe-mode startup does not break guest runtime shape | Passed for the default guest startup path; `routing_policy_setting.mode` remained `observe` |
| Probe-off safety | targeted tests and guest runtime confirm probe-gated disables do not run when probe is off | Passed at Go-test level |
| Audit acceptance | read-only audit review after smoke exists | Passed with notes: evidence was sufficient for that increment closure; host SSH later remained flaky during readback |

## Ready-to-run inputs for the next `/66` rerun

- Go verification commands: `docs/henry-66-guest-verification-commands.md`
- Smoke checklist: `docs/henry-66-guest-smoke-checklist.md`
- Evidence log: `docs/henry-66-runtime-evidence-log.md`

## Current conclusion

- The historical targeted Go rerun is complete and passed.
- The historical guest smoke is complete and remains the latest accepted `/66` runtime proof.
- That historically verified code is now merged on `main`.
- The remaining gap is operational: `/66` still needs a fresh live audit and staging classification before it can be treated as an actively maintained preprod environment.
