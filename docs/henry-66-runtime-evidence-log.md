# Henry /66 Runtime Evidence Log

This file records the latest trusted `/66` evidence for the routing probe/restore slice and now also records how that evidence relates to the current mainline.

## Current status summary

### Known current state (`2026-07-04`)

- The LAN Device Ops backend is available and was used from `C:\Users\HHPC\Documents\Codex\ssh-lan-device-codex`.
- A fresh `/66` live read-only refresh was completed on `2026-07-04T03:00:37+08:00`.
- Latest live report: `C:\Users\HHPC\Documents\Codex\ssh-lan-device-codex\outputs\windows-66-readonly-2026-07-04T03-00-37+08-00.json`.
- Latest live classification: `reachable-but-not-deployed`.
- Current live host-side facts:
  - host SSH read-only refresh succeeded
  - host health remained `healthy`
  - host `sshd` is running
  - guest SSH loopback `127.0.0.1:22222` is reachable and has a listener
  - no usable `/api/status` body was returned on `127.0.0.1:13000` or `127.0.0.1:3000`
- The historically verified routing slice is already merged on `main`.
- Relevant merged commits:
  - `8cfec81f feat: add native routing policy baseline and /66 verification docs`
  - `2c4e9354 feat: close probe restore loop and refresh /66 verification evidence`
  - `d12c0630 fix: clear pending restore state on routing success`
- Later engineering closure also landed on `main`, including green `Go CI`, `Frontend CI`, and `Release Dry Run` on `d95aa5ca`.

### Current blocker state

- `/66` is no longer blocked on missing audit tooling; it has a current live classification.
- The read-only classification is still `reachable-but-not-deployed`: guest SSH exists, but there is still no usable HTTP status endpoint on `13000` or `3000`.
- The current write-scope blocker is now narrower and confirmed: `/66` host runtime is blocked from WSL2 recovery on the current Windows Server 2022 build, so guest staging redeploy cannot proceed yet.
- There is still no newer accepted guest-local targeted Go rerun or guest smoke snapshot than the historical `2026-06-30` evidence below.

### Historical capture note

At the time the `2026-06-30` evidence was collected, the slice had not yet been committed/pushed/submitted. That historical note is preserved below for context only; it is no longer the current repository state.

## 2026-07-04 live read-only staging audit

- Status: completed
- Audit backend: `C:\Users\HHPC\Documents\Codex\ssh-lan-device-codex`
- Inventory path used by that backend: `C:\Users\HHPC\Documents\Codex\ssh-lan-device-codex\configs\devices.local.yaml`
- Latest accepted live report: `C:\Users\HHPC\Documents\Codex\ssh-lan-device-codex\outputs\windows-66-readonly-2026-07-04T03-00-37+08-00.json`
- Latest accepted live report SHA256: `9CF3E536F3C319A5EF5FAFA091BD27B4158652A316C4B4049379AA4303D4C6F4`
- Latest compare command: `compare-readonly --device 66 --reports-dir outputs --json`

### Captured evidence

- `doctor --json` succeeded; only the local terminal encoding warning remained (`stdout_encoding=gbk`), and JSON data stayed parseable.
- `validate` succeeded against the local device inventory.
- `status --device 66 --refresh --summary zh-ascii` succeeded and reported:
  - host status `OK`
  - host health reason `healthy`
  - staging classification `reachable-but-not-deployed`
  - `22222 reachable=True`
  - `listener=True`
  - `api_port=-`
  - `api_success=False`
- The refreshed live report recorded:
  - `guest_ssh_reachable=true`
  - `guest_ssh_listener=true`
  - `host_sshd_running=true`
  - `api_status_port=null`
  - `api_status_http_ok=false`
  - `api_status_success=false`
  - `api_setup=null`
- `compare-readonly --json` succeeded after the backend JSON emission fix and showed `changed_count=0` between the `2026-07-04T02:46:16+08:00` and `2026-07-04T03:00:37+08:00` snapshots.
- The compare output also confirmed that the persistent failure is isolated to the staging HTTP probes, not host health or guest SSH reachability.

### Live audit conclusion

- `/66` is currently reachable as a Windows staging host.
- The guest SSH path is still present through the host loopback mapping.
- `/66` is **not** currently evidenced as a running staging stack because no usable `/api/status` response exists on `127.0.0.1:13000` or `127.0.0.1:3000`.
- Treat the environment as `reachable-but-not-deployed` until a later write-scope repair or redeploy phase produces a healthy `/api/status` result.

## 2026-07-04 host-side WSL recovery attempt

- Status: blocked
- Goal: upgrade `/66` from the old in-box WSL path to a WSL2-capable runtime so a new Debian guest can host staging.
- Existing rollback point preserved:
  - `C:\WSL\exports\debian-20260704.tar`
  - size `3353948160`
  - last write `2026-07-04 11:06:52 +08:00`
- Official package used:
  - file `wsl.2.7.10.0.x64.msi`
  - source `https://github.com/microsoft/WSL/releases/download/2.7.10/wsl.2.7.10.0.x64.msi`
  - verified SHA256 `1A62F90A43C03CC5BDA47DFD0B6FAF496AC70FD4389190518120A4F84FC895CF`

### Captured evidence

- The old host entry still used `C:\Windows\System32\wsl.exe` version `10.0.20348.1`.
- Installing the official MSI succeeded and added a new runtime at `C:\Program Files\WSL\wsl.exe`.
- The new runtime reports:
  - WSL version `2.7.10.0`
  - kernel version `6.18.33.2-2`
  - Windows version `10.0.20348.169`
- The new runtime can answer `--version`, which proves the MSI installed correctly.
- However, the recovery-critical commands still failed:
  - `C:\Program Files\WSL\wsl.exe --set-default-version 2`
  - `C:\Program Files\WSL\wsl.exe -l -v`
- The failure was `Wsl/WSL_E_OS_NOT_SUPPORTED`, and the runtime itself pointed to:
  - `https://aka.ms/store-wsl-kb-winserver2022`
  - `https://aka.ms/wslinstall`
- The existing distros remained stuck at version `1`:
  - `Debian`
  - `DebianWSL2`

### Recovery conclusion

- The official Store-style WSL runtime is now installed on `/66`, but it still reports `Wsl/WSL_E_OS_NOT_SUPPORTED` for WSL2 operations on this Windows Server build.
- No WSL2 distro was created.
- No guest Docker / Compose recovery work started.
- No new staging `/api/status` evidence exists.
- Treat the host as `host runtime blocked` until the required Windows Server 2022 update path behind `aka.ms/store-wsl-kb-winserver2022` is applied.

## Historical local package used for the 2026-06-30 rerun

- Local package file used for that run: `henry-newapi-src-current.tar.gz`
- Local SHA256 sidecar used for that run: `henry-newapi-src-current.tar.gz.sha256`
- Uploaded guest artifact name: `henry-newapi-src-current.tar.gz`
- Uploaded guest SHA256 sidecar name: `henry-newapi-src-current.tar.gz.sha256`
- Package SHA256 for that run: `70C5737104CF86F894E17DFB0333E95FBCF8BECE002F9868BFF744534FD679C9`
- Guest upload path used: `/home/newapi/henry-newapi-src-current.tar.gz`
- Guest workspace used: `/home/newapi/henry-newapi`
- Host-side guest SSH mapping used: `127.0.0.1:22222 -> guest:22`

## 2026-06-30 historical targeted Go verification

- Status: passed
- Target: `/66` Debian guest via host-side mapping `127.0.0.1:22222 -> guest:22`
- Required command source: `docs/henry-66-guest-verification-commands.md`

### Captured evidence

- Guest SHA256 matched the uploaded package SHA sidecar for `70C5737104CF86F894E17DFB0333E95FBCF8BECE002F9868BFF744534FD679C9`.
- `go version` reported `go1.25.1 linux/amd64`.
- Minimal probe/restore slice passed:
  - `go test ./service -run 'TestGetHttpClientForContext|TestCloneHTTPClientWithConnectTimeoutPreservesExistingDialContext|TestRunRoutingAutomationOnceProbe' -count=1 -v`
- Broader rerun passed:
  - `go test ./service -run 'DetectRoutingChannelRole|RoutingStatusCodeMapping|ChannelSupportsRequestPath|ChannelRuntimeHealthCooldown|RunRoutingAutomationOnce|ChannelAffinity|TestGetHttpClientForContext|TestCloneHTTPClientWithConnectTimeoutPreservesExistingDialContext' -count=1 -v`
  - `go test ./middleware -run 'SpecificChannel|AffinityChannel' -count=1 -v`
  - `go test ./controller -run 'RoutingAutomation|ShouldRetry|LockedTaskChannel' -count=1 -v`
  - `go test ./model -run 'RoutingPolicy|GetRandomSatisfiedChannelSkipsUnhealthyAndFailOpens|GetChannelSkipsUnhealthyAndFailOpens' -count=1 -v`
  - `go test ./relay/channel -run TestDoesNotExist -count=1`

### Historical notes

- The first minimal rerun exposed an outdated restore-timing test assumption, not a runtime failure.
- The local test update aligned old restore tests with the new `ProbeRetrySeconds` / `ActiveProbeIntervalSeconds` scheduling behavior before the corrected rerun passed.

## 2026-06-30 historical guest smoke

- Status: passed
- Required command source: `docs/henry-66-guest-smoke-checklist.md`
- Actual smoke date: `2026-06-30`
- Runtime target: guest-local Docker Compose inside `/66` Debian guest

### Captured evidence

- The first smoke attempt exposed a guest runtime dependency on pulling `postgres:15` and `redis:latest` from Docker Hub; guest diagnostics showed normal routing and DNS, but the Docker daemon timed out while awaiting registry headers.
- The accepted rerun stayed guest-local and used a disposable full smoke compose file `docker-compose.henry-smoke.full.yml` with:
  - current guest-built `./new-api` bind-mounted to `/new-api:ro`
  - cached guest-local `postgres:16-alpine` and `redis:7-alpine` images
  - loopback-only exposure `127.0.0.1:13000 -> 3000`
- Rendered compose config was captured at `/home/newapi/henry-newapi-compose-rendered.yml` and copied back to the `/66` host as `C:\Users\administrator\Downloads\henry-newapi\henry-newapi-compose-rendered.yml`.
- `docker compose -f docker-compose.henry-smoke.full.yml ps` showed:
  - `new-api` exposed only on `127.0.0.1:13000->3000/tcp`
  - `postgres` on guest-internal `5432/tcp`
  - `redis` on guest-internal `6379/tcp`
- `http://127.0.0.1:13000/api/status` returned `success=true` on that snapshot.
- Disposable root/admin setup was completed through `POST /api/setup`, and follow-up `GET /api/setup` plus `GET /api/status` showed `setup=true`.
- Windows host `http://127.0.0.1:23000/api/status` was unreachable during the accepted isolated smoke, confirming the older NAT-forwarded path was not reused for that snapshot.
- No Windows host MySQL `3306` or Python `5000` dependency was introduced.
- After evidence capture, the guest smoke stack was rolled back with `docker compose -f docker-compose.henry-smoke.full.yml down -v`.

## Historical merged probe/restore slice

The historically rerun routing slice corresponds to:

- `controller/routing_policy_probe.go`
- `relay/channel/api_request.go`
- `routingpolicy/runtime.go`
- `service/http_client.go`
- `service/routing_policy_test.go`
- `service/http_client_test.go`

Those changes are now part of `main` and are no longer a pending local-only increment.

## Remaining boundaries

- The trusted runtime evidence still belongs to the `/66` Debian guest, not this local Windows machine.
- The accepted `2026-06-30` evidence remains valid as historical proof for the merged routing slice.
- The latest open operational question is no longer basic reachability; it is how to clear the `/66` host WSL2 blocker so guest staging can be restored behind the already-reachable host/guest entry path.
- If any `go` source or test changes again, the package must be re-synced and rerun on `/66`.
- If runtime wiring or guest smoke behavior changes again, both targeted Go verification and guest smoke must be rerun.

## Evidence refresh checklist for the next `/66` rerun

The next accepted `/66` rerun should again capture:

- rendered compose config path and loopback-only exposure evidence
- compose `ps` result for `new-api`, `postgres`, and `redis`
- `http://127.0.0.1:13000/api/status` success payload
- setup completion evidence for the guest stack, or the exact blocker
- confirmation that host `3306`, `5000`, and the old `23000` runtime path are not part of the accepted runtime dependency chain
