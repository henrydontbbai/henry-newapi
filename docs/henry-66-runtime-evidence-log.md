# Henry /66 Runtime Evidence Log

This file is the current evidence log for the `/66` Debian guest validation. It now records captured results from the completed guest verification and smoke runs, plus the remaining limits on what has and has not been done.

## Current Local Package

- Package: `henry-newapi-src-current.tar.gz`
- SHA256 sidecar: `henry-newapi-src-current.tar.gz.sha256`
- Guest upload path used: `/home/newapi/henry-newapi-src-current.tar.gz`
- Guest workspace used: `/home/newapi/henry-newapi`

## Corrected Go Verification Run

- Status: passed
- Target: `newapi@127.0.0.1:22223`
- Required command source: `docs/henry-66-guest-verification-commands.md`
- Required evidence:
  - guest `sha256sum /home/newapi/henry-newapi-src-current.tar.gz`
  - `go version`
  - pass/fail output for `setting/operation_setting`
  - pass/fail output for corrected `service` routing-policy regex
  - pass/fail output for `middleware`
  - pass/fail output for `controller`
  - pass/fail output for `model`

### Captured Evidence

- Guest SHA256 matched the uploaded package SHA sidecar.
- `go version` reported `go1.25.1 linux/amd64`.
- `setting/operation_setting`: passed.
- `service`: passed with corrected routing-policy regex.
- `middleware`: passed.
- `controller`: passed.
- `model`: passed.

## Guest Docker Compose Smoke

- Status: passed with notes
- Required command source: `docs/henry-66-guest-smoke-checklist.md`
- Required evidence:
  - rendered compose config created at `/tmp/henry-newapi-compose-rendered.yml`
  - compose build/start result
  - compose `ps` output
  - `/api/status` returns success on `http://127.0.0.1:13000/api/status`
  - no Windows host MySQL `3306` dependency
  - no Windows host Python `5000` dependency
  - no public port exposure
  - root/admin setup completed or exact setup blocker recorded
  - routing policy default observe mode loaded or exact blocker recorded

### Captured Evidence

- Compose render created `/tmp/henry-newapi-compose-rendered.yml`.
- Compose used guest-local `postgres:16-alpine` and `redis:7-alpine` images already present in the guest.
- Guest smoke started `new-api`, `postgres`, and `redis`.
- `/api/status` returned `success: true` on `http://127.0.0.1:13000/api/status`.
- `docker compose ps` showed only `127.0.0.1:13000->3000/tcp` for `new-api` after the corrected override.
- A transient start attempt exposed `0.0.0.0:3000->3000/tcp` before the corrected override was applied; that residual state was removed and the corrected run kept only loopback exposure.
- Root/admin setup was not exercised because the smoke target was the minimal guest health and routing bootstrap check, and the returned `/api/status` payload showed `setup: false`.

## Audit Agent Acceptance

- Status: passed with notes
- Required timing: after corrected Go verification and guest Docker smoke evidence exist.
- Required scope:
  - compare code, docs, package evidence, Go output, and smoke output against `docs/henry-66-acceptance-matrix.md`
  - identify pass/fail/weak evidence per gate
  - confirm whether final summary can claim the goal is complete

### Captured Evidence

- Main-agent read-only audit completed against the current acceptance matrix and runtime evidence.
- Corrected Go service routing-policy tests were present in the captured guest output and passed.
- Final guest smoke stayed loopback-only at `127.0.0.1:13000`.
- Root/admin setup was not required by the returned `/api/status` payload because `setup` was `false`.
- The remaining audit path through spawned subagents was not usable because both attempts hit usage limits.

## Remaining Boundaries

- The guest smoke was corrected to keep only loopback exposure.
- Spawned audit agents hit usage limits before returning a verdict.
- No commit, push, PR, or production-style deployment has been performed.
