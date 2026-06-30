# Henry /66 Guest Verification Commands

This file records the exact guest-side verification sequence for the current native-routing branch. It is a command checklist, not proof that verification already passed.

## Preconditions

- Target: Debian guest inside `/66`, not the Windows host.
- Source package to upload from local Windows: `C:\Users\HHPC\Documents\Codex\henry-newapi\henry-newapi-src-current.tar.gz`.
- Guest workspace: `/home/newapi/henry-newapi`.
- Do not use Windows host MySQL `3306` or Python service `5000`.
- Known guest Go path from the first verification run: `/home/newapi/go-1.25.1/bin/go`.

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
go test ./setting/operation_setting -run RoutingPolicy -count=1 -v
go test ./service -run 'DetectRoutingChannelRole|RoutingStatusCodeMapping|ChannelSupportsRequestPath|ChannelRuntimeHealthCooldown|RunRoutingAutomationOnce|ChannelAffinity' -count=1 -v
go test ./middleware -run 'SpecificChannel|AffinityChannel' -count=1 -v
go test ./controller -run 'RoutingAutomation|ShouldRetry|LockedTaskChannel' -count=1 -v
go test ./model -run 'RoutingPolicy|GetRandomSatisfiedChannelSkipsUnhealthyAndFailOpens|GetChannelSkipsUnhealthyAndFailOpens' -count=1 -v
```

## Acceptance Notes

- Passing these commands proves compile-level viability for the touched backend slices.
- It does not by itself prove Docker Compose smoke, setup flow, root/admin initialization, or probe success against real upstream channels.
- After these pass, the next step is the isolated guest smoke checklist in `docs/henry-66-guest-smoke-checklist.md` with default `observe` and `probe_policy.active_probe_enabled=false`.
