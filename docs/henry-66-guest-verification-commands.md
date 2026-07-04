# Henry /66 Guest Verification Commands

This file records the exact guest-side verification sequence for the current probe/restore increment. It is a command checklist, not proof that verification already passed.

## Preconditions

- Target: Debian guest inside `/66`, not the Windows host.
- Current host-side guest SSH mapping: `127.0.0.1:22222 -> guest:22`.
- Source package path on local Windows: `C:\Users\HHPC\Documents\Codex\henry-newapi\henry-newapi-src-current.tar.gz`.
- Guest workspace: `/home/newapi/henry-newapi`.
- Do not use Windows host MySQL `3306` or Python service `5000`.
- Known guest Go path: `/home/newapi/go-1.25.1/bin/go`.

## Host-side packaging commands

Before uploading to the guest, regenerate the package from the current local workspace and record the SHA256 from that exact file:

```powershell
if (Test-Path C:\Users\HHPC\Documents\Codex\henry-newapi\henry-newapi-src-current.tar.gz) { Remove-Item C:\Users\HHPC\Documents\Codex\henry-newapi\henry-newapi-src-current.tar.gz }
tar -czf C:\Users\HHPC\Documents\Codex\henry-newapi\henry-newapi-src-current.tar.gz --exclude=.git --exclude=web/node_modules --exclude=web/default/dist --exclude=web/classic/dist -C C:\Users\HHPC\Documents\Codex\henry-newapi .
Get-FileHash C:\Users\HHPC\Documents\Codex\henry-newapi\henry-newapi-src-current.tar.gz -Algorithm SHA256
```

Acceptance:

- upload only the package generated in this step
- record the actual SHA256 value in the current `/66` evidence note
- do not reuse a historical SHA256 from an earlier guest rerun
- make sure the package includes `docker-compose.henry-staging.yml` before upload

## Guest Commands

```bash
set -euo pipefail

export PATH=/home/newapi/go-1.25.1/bin:$PATH

cd /home/newapi
mkdir -p henry-newapi
sha256sum /home/newapi/henry-newapi-src-current.tar.gz
tar -xzf /home/newapi/henry-newapi-src-current.tar.gz -C /home/newapi/henry-newapi --strip-components=1
cd /home/newapi/henry-newapi

mkdir -p web/default/dist web/classic/dist
printf '<!doctype html><html><body>placeholder</body></html>' > web/default/dist/index.html
printf '<!doctype html><html><body>placeholder</body></html>' > web/classic/dist/index.html

go version
go test ./service -run 'TestGetHttpClientForContext|TestCloneHTTPClientWithConnectTimeoutPreservesExistingDialContext|TestRunRoutingAutomationOnceProbe' -count=1 -v
go test ./service -run 'DetectRoutingChannelRole|RoutingStatusCodeMapping|ChannelSupportsRequestPath|ChannelRuntimeHealthCooldown|RunRoutingAutomationOnce|ChannelAffinity|TestGetHttpClientForContext|TestCloneHTTPClientWithConnectTimeoutPreservesExistingDialContext' -count=1 -v
go test ./middleware -run 'SpecificChannel|AffinityChannel' -count=1 -v
go test ./controller -run 'RoutingAutomation|ShouldRetry|LockedTaskChannel' -count=1 -v
go test ./model -run 'RoutingPolicy|GetRandomSatisfiedChannelSkipsUnhealthyAndFailOpens|GetChannelSkipsUnhealthyAndFailOpens' -count=1 -v
go test ./relay/channel -run TestDoesNotExist -count=1
```

## Acceptance Notes

- Passing these commands proves compile-level viability and current-snapshot targeted behavior for the touched backend slices.
- The `sha256sum` output above must match the host-side SHA256 recorded from the package generated for this exact rerun.
- It does not by itself prove current-snapshot Docker Compose smoke, setup flow, or probe success against real upstream channels.
- The next runtime step is the tracked staging bring-up from `docs/henry-66-staging-runbook.md` using `docker-compose.henry-staging.yml`.
