# Henry /66 Audit Agent Brief

Use this brief only after current-snapshot `/66` Go verification evidence and current-snapshot guest Docker Compose smoke evidence both exist.

This brief is for the `2026-06-30` probe/restore increment, not for the older pushed baseline snapshot alone.

## Scope

- Review only the current workspace, `docs/henry-66-acceptance-matrix.md`, `docs/henry-66-runtime-evidence-log.md`, and captured current-snapshot `/66` command outputs.
- Validate the current increment as a minimum-risk VM guest verification and smoke test.
- Treat the Windows `/66` host as a VM carrier only; NewAPI must run inside the Debian guest.

## Forbidden Actions

- Do not edit files.
- Do not use SSH, Docker, browser control, desktop control, Git writes, GitHub writes, or deployment tools.
- Do not inspect secrets, `.env`, credentials, private keys, cookies, or session files.
- Do not expand scope into AiMaMi, provider, model routing, proxy, Codex, local relay, or global user configuration.

## Required Checks

- Confirm the current-snapshot Go rerun covered:
  - the minimal probe/http-client service slice;
  - the broader routing-policy service slice;
  - middleware, controller, model, and relay/channel compile-level checks.
- Confirm the current-snapshot package SHA matches the evidence recorded in `docs/henry-66-runtime-evidence-log.md`.
- Confirm the current-snapshot guest Docker Compose smoke used guest-local services and did not depend on Windows host MySQL `3306` or Python `5000`.
- Accept a guest-local fallback compose file if Docker Hub pull failures forced the smoke away from the default `postgres:15` / `redis:latest` tags, as long as the fallback still used guest-local services only and preserved the same isolation contract.
- Confirm `/api/status` was checked on guest loopback `127.0.0.1:13000`.
- Confirm root/admin setup was completed or the exact blocker was recorded.
- Confirm no public exposure, production database, real provider key, or host service mutation was introduced.
- Confirm the evidence is strong enough to update each gate in `docs/henry-66-acceptance-matrix.md`.

## Failure Rule

If current-snapshot smoke evidence is still missing, the verdict must not be `pass`.

At that point, the audit should return `fail` or `pass-with-notes` with an explicit blocker that current-snapshot smoke has not yet been rerun and older baseline smoke evidence is not enough.

## Output Format

Return a concise acceptance report:

- Overall verdict: `pass`, `pass-with-notes`, or `fail`
- Gate results: one line per matrix gate, with `pass`, `weak`, or `fail`
- Blocking issues: exact blockers, if any
- Evidence gaps: exact missing evidence, if any
- Recommendation: whether the main agent may summarize the current increment as complete
