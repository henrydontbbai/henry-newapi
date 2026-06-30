# Henry NewAPI /66 Acceptance Matrix

This matrix records the current acceptance status for the `/66` VM-host validation and minimum-risk guest smoke. It summarizes already-captured evidence; the detailed proof remains in the paired runtime evidence log.

## Current Artifact

- Source package: `henry-newapi-src-current.tar.gz`
- Latest local SHA256 evidence: `henry-newapi-src-current.tar.gz.sha256`
- Latest uploaded SHA256: `8EADF43034A5DF7AD1DA853EF37A5DA4E863BFD126C60D7CB0484FB9BFD689E6`
- Local host limitation: no `go`, no `gofmt`, no `docker`
- Remote target: Debian guest inside `/66`, not the Windows host
- Verified guest Go path: `/home/newapi/go-1.25.1/bin/go`

## Gate Matrix

| Gate | Required evidence | Current status |
| --- | --- | --- |
| Source package excludes unsafe local state | package listing excludes `.git`, `.env*`, `node_modules`, old tarballs, screenshots | Current local package prepared; package check passed |
| Targeted Go settings tests | `/66` guest output for `go test ./setting/operation_setting -run RoutingPolicy -count=1 -v` | Passed on `/66` Debian guest with Go `1.25.1` |
| Targeted Go service channel-affinity tests | `/66` guest output for earlier `go test ./service -run 'RoutingPolicy\|ChannelAffinity' -count=1 -v` | Passed on `/66` Debian guest with Go `1.25.1` |
| Targeted Go service routing-policy tests | `/66` guest output for `go test ./service -run 'DetectRoutingChannelRole\|RoutingStatusCodeMapping\|ChannelSupportsRequestPath\|ChannelRuntimeHealthCooldown\|RunRoutingAutomationOnce\|ChannelAffinity' -count=1 -v` | Pass on `/66` Debian guest with Go `1.25.1` |
| Targeted Go middleware tests | `/66` guest output for `go test ./middleware -run 'SpecificChannel\|AffinityChannel' -count=1 -v` | Passed on `/66` Debian guest with Go `1.25.1` |
| Targeted Go controller tests | `/66` guest output for `go test ./controller -run 'RoutingAutomation\|ShouldRetry\|LockedTaskChannel' -count=1 -v` | Passed on `/66` Debian guest with Go `1.25.1` |
| Targeted Go model tests | `/66` guest output for `go test ./model -run 'RoutingPolicy\|GetRandomSatisfiedChannelSkipsUnhealthyAndFailOpens\|GetChannelSkipsUnhealthyAndFailOpens' -count=1 -v` | Passed on `/66` Debian guest with Go `1.25.1` |
| Guest-only Docker smoke | guest output from `docs/henry-66-guest-smoke-checklist.md` showing compose up and `/api/status` on `127.0.0.1:13000` | Pass with notes: final corrected run stayed loopback-only |
| No Windows-host DB/service dependency | guest smoke uses compose Postgres/Redis and does not touch host `3306` or `5000` | Pass |
| Root/admin setup gate | guest smoke either completes setup or reports the exact setup blocker | Pass with notes: `/api/status` returned `setup: false`, so the guest smoke did not need root/admin setup interaction |
| Routing observe safety | guest smoke or test evidence shows observe mode does not mutate channel status | Pass |
| Probe-off safety | targeted tests and guest runtime confirm probe-gated disables do not run when probe is off | Pass |
| Audit-agent acceptance | read-only audit review of current code and `/66` evidence after tests/smoke | Pass with notes: main-thread read-only audit completed; spawned subagents hit usage limits before returning a verdict |

## Ready-To-Run Guest Inputs

- Go verification commands: `docs/henry-66-guest-verification-commands.md`
- Smoke checklist: `docs/henry-66-guest-smoke-checklist.md`
- Evidence log: `docs/henry-66-runtime-evidence-log.md`

## Completion Notes

- Corrected targeted Go verification has been rerun on `/66` and passed.
- Guest Docker Compose smoke has been run and passed with loopback-only exposure.
- Final audit-agent acceptance completed in the main thread; spawned subagents were blocked by usage limits and did not provide an additional verdict.
- No commit, push, PR, or deployment has been performed.
