# Henry NewAPI /66 Acceptance Matrix

This matrix tracks the acceptance state for the current `2026-06-30` probe/restore snapshot, not just the earlier baseline snapshot.

## Current Artifact

- Source package: `henry-newapi-src-current.tar.gz`
- Latest local SHA256 evidence: `henry-newapi-src-current.tar.gz.sha256`
- Latest uploaded SHA256 for the current snapshot: `70C5737104CF86F894E17DFB0333E95FBCF8BECE002F9868BFF744534FD679C9`
- Local host limitation: no `go`, no `docker`
- Remote target: Debian guest inside `/66`, not the Windows host
- Current host-side guest SSH mapping: `127.0.0.1:22222 -> guest:22`
- Verified guest Go path: `/home/newapi/go-1.25.1/bin/go`

## Gate Matrix

| Gate | Required evidence | Current status |
| --- | --- | --- |
| Source package excludes unsafe local state | package listing excludes `.git`, `.env*`, `node_modules`, old tarballs, screenshots | Passed for current package |
| Current-snapshot targeted Go service probe/http-client slice | `/66` guest output for `TestGetHttpClientForContext`, `TestCloneHTTPClientWithConnectTimeoutPreservesExistingDialContext`, `TestRunRoutingAutomationOnceProbe` | Passed on `/66` Debian guest with Go `1.25.1` |
| Current-snapshot targeted Go service broader routing slice | `/66` guest output for the corrected service rerun including routing, cooldown, affinity, probe/restore coverage | Passed on `/66` Debian guest with Go `1.25.1` |
| Current-snapshot targeted Go middleware tests | `/66` guest output for `go test ./middleware -run 'SpecificChannel\|AffinityChannel' -count=1 -v` | Passed on `/66` Debian guest with Go `1.25.1` |
| Current-snapshot targeted Go controller tests | `/66` guest output for `go test ./controller -run 'RoutingAutomation\|ShouldRetry\|LockedTaskChannel' -count=1 -v` | Passed on `/66` Debian guest with Go `1.25.1` |
| Current-snapshot targeted Go model tests | `/66` guest output for `go test ./model -run 'RoutingPolicy\|GetRandomSatisfiedChannelSkipsUnhealthyAndFailOpens\|GetChannelSkipsUnhealthyAndFailOpens' -count=1 -v` | Passed on `/66` Debian guest with Go `1.25.1` |
| Relay/channel compile-level check | `/66` guest output for `go test ./relay/channel -run TestDoesNotExist -count=1` | Passed on `/66` Debian guest |
| Current-snapshot guest-only Docker smoke | guest output from `docs/henry-66-guest-smoke-checklist.md` showing compose up and `/api/status` on `127.0.0.1:13000` | Passed on `/66` Debian guest with loopback-only `127.0.0.1:13000->3000` |
| Current-snapshot no Windows-host DB/service dependency | guest smoke showing compose Postgres/Redis only and no host `3306` / `5000` dependency | Passed on `/66`; host `23000` was also unreachable during the accepted isolated smoke |
| Current-snapshot root/admin setup gate | guest smoke either completes setup or records the exact setup blocker | Passed with disposable guest-only setup completion evidence |
| Current-snapshot routing observe safety | smoke or runtime evidence from this snapshot that observe-mode startup does not break guest runtime shape | Passed for the default guest startup path; `routing_policy_setting.mode` remains `observe` by default and current smoke stayed healthy |
| Probe-off safety | targeted tests and current guest runtime confirm probe-gated disables do not run when probe is off | Passed at Go-test level |
| Current-snapshot audit acceptance | read-only audit review after current-snapshot smoke exists | Passed with notes: evidence is sufficient for current increment closure; host SSH remained flaky during later readback but did not weaken the accepted Go rerun or smoke evidence |

## Ready-To-Run Guest Inputs

- Go verification commands: `docs/henry-66-guest-verification-commands.md`
- Smoke checklist: `docs/henry-66-guest-smoke-checklist.md`
- Evidence log: `docs/henry-66-runtime-evidence-log.md`

## Current Conclusion

- Current-snapshot targeted Go rerun is complete and passed.
- Current-snapshot guest smoke is complete and captured as refreshed `/66` evidence.
- Current-snapshot read-only audit acceptance is complete.
- Audit notes: the accepted evidence set is strong enough for current increment closure; the only standing caution is intermittent `/66` host SSH instability during later readback, not a gap in the accepted Go rerun or smoke evidence.
- No commit, push, PR, or deployment has been performed for this increment.
