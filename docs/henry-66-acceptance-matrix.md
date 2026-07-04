# Henry NewAPI /66 Acceptance Matrix

This matrix preserves the accepted `2026-06-30` `/66` verification snapshot and explains how that snapshot relates to the current repository state.

## Current status summary (`2026-07-04`)

### Known current state

- The routing baseline plus probe/restore closure work is already on `main`.
- The live `/66` host audit path is restored through `C:\Users\HHPC\Documents\Codex\ssh-lan-device-codex`.
- A fresh `/66` read-only refresh completed on `2026-07-04T03:00:37+08:00`.
- The current live classification is `reachable-but-not-deployed`.
- Current live facts:
  - host health `OK`
  - host `sshd` running
  - guest loopback SSH `127.0.0.1:22222` reachable with a listener
  - no usable `/api/status` body on `127.0.0.1:13000` or `127.0.0.1:3000`
- Relevant merged commits:
  - `8cfec81f feat: add native routing policy baseline and /66 verification docs`
  - `2c4e9354 feat: close probe restore loop and refresh /66 verification evidence`
  - `d12c0630 fix: clear pending restore state on routing success`
- Mainline engineering closure is green, including `Go CI`, `Frontend CI`, and `Release Dry Run`.

### Current blocker state

- `/66` is no longer blocked on missing live-audit tooling.
- The latest trusted guest-local Go/smoke evidence is still the historical `2026-06-30` snapshot below.
- The read-only state is still `reachable-but-not-deployed`: the SSH entry path exists, but the expected staging HTTP surface is not up.
- The current write-scope blocker is now host-side and more precise: the official WSL 2.7.10 runtime installed on `/66`, but WSL2 commands still fail with `Wsl/WSL_E_OS_NOT_SUPPORTED` on Windows Server 2022 build `10.0.20348.169`.

## Current live audit gate (`2026-07-04`)

| Gate | Required evidence | Current live status |
| --- | --- | --- |
| Host read-only audit path | `doctor --json`, `validate`, `status --device 66 --refresh` succeed from the LAN Device Ops backend | Passed via `C:\Users\HHPC\Documents\Codex\ssh-lan-device-codex` |
| Host health | latest host report completes with host status `OK` and health reason `healthy` | Passed |
| Host WSL2-capable runtime | official WSL runtime can run `--set-default-version 2` and `-l -v` without `Wsl/WSL_E_OS_NOT_SUPPORTED` | Failed; `C:\Program Files\WSL\wsl.exe` 2.7.10.0 installed, but WSL2 operations are blocked on Windows Server 2022 build `10.0.20348.169` |
| Guest SSH loopback path | `127.0.0.1:22222` reachable and listening | Passed |
| Staging HTTP status | usable `/api/status` on `127.0.0.1:13000` or `127.0.0.1:3000` | Failed; no usable body returned |
| Current classification | one of `offline`, `reachable-but-not-deployed`, `staging-running`, `staging-drifted` | `reachable-but-not-deployed` |
| Live report traceability | external live report path plus SHA256 | `windows-66-readonly-2026-07-04T03-00-37+08-00.json` / `9CF3E536F3C319A5EF5FAFA091BD27B4158652A316C4B4049379AA4303D4C6F4` |
| Stable classification compare | `compare-readonly --device 66 --reports-dir outputs --json` succeeds and preserves current staging facts | Passed; `changed_count=0` between the two `2026-07-04` snapshots |

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
- Staging runbook: `docs/henry-66-staging-runbook.md`

## Current conclusion

- The historical targeted Go rerun is complete and passed.
- The historical guest smoke is complete and remains the latest accepted guest runtime proof.
- That historically verified code is now merged on `main`.
- `/66` now has a current live classification, and it is **`reachable-but-not-deployed`** rather than unknown.
- The remaining gap is no longer audit entry; it is first clearing the `/66` host WSL2 runtime blocker, then restoring a healthy guest staging `/api/status` surface.
