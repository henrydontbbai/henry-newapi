# Henry /66 Guest Verification Commands

This file records the exact guest-side verification sequence for the current probe/restore increment. It is a command checklist, not proof that verification already passed.

## Preconditions

- Target: Debian guest inside `/66`, not the Windows host.
- Current host-side guest SSH mapping: `127.0.0.1:22222 -> guest:22`.
- Source package to upload from local Windows: `C:\Users\HHPC\Documents\Codex\henry-newapi\henry-newapi-src-current.tar.gz`.
- Expected current package SHA256: `70C5737104CF86F894E17DFB0333E95FBCF8BECE002F9868BFF744534FD679C9`.
- Guest workspace: `/home/newapi/henry-newapi`.
- Do not use Windows host MySQL `3306` or Python service `5000`.
- Known guest Go path: `/home/newapi/go-1.25.1/bin/go`.

## Guest Commands

```bash
set -euo pipefail

export PATH=/home/newapi/go-1.25.1/bin:$PATH

cd /home/newapi
mkdir -p henry-newapi
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
- It does not by itself prove current-snapshot Docker Compose smoke, setup flow, or probe success against real upstream channels.
- Because this increment changes request-path runtime wiring, the next step remains the isolated guest smoke checklist in `docs/henry-66-guest-smoke-checklist.md` with loopback-only exposure at `127.0.0.1:13000`.
