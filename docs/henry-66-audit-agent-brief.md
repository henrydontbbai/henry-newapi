# Henry /66 Audit Agent Brief

Use this brief only after corrected `/66` Go verification and guest Docker Compose smoke evidence exist.

## Scope

- Review only the current workspace, `docs/henry-66-acceptance-matrix.md`, `docs/henry-66-runtime-evidence-log.md`, and captured `/66` command outputs.
- Validate the `/66` plan as a minimum-risk VM guest verification and smoke test.
- Treat the Windows `/66` host as a VM carrier only; NewAPI must run inside the Debian guest.

## Forbidden Actions

- Do not edit files.
- Do not use SSH, Docker, browser control, desktop control, Git writes, GitHub writes, or deployment tools.
- Do not inspect secrets, `.env`, credentials, private keys, cookies, or session files.
- Do not expand scope into AiMaMi, provider, model routing, proxy, Codex, local relay, or global user configuration.

## Required Checks

- Confirm the corrected Go command covered the routing-policy service tests, including role detection, runtime cooldown, observe/probe-off safety, paygo hard-failure, paygo nudge, subscription transient, and probe restore tests.
- Confirm settings, middleware, controller, and model targeted tests passed on the `/66` Debian guest.
- Confirm the guest Docker Compose smoke used guest-local services and did not depend on Windows host MySQL `3306` or Python `5000`.
- Confirm `/api/status` was checked on guest loopback `127.0.0.1:13000`.
- Confirm root/admin setup was completed or the exact blocker was recorded.
- Confirm no public exposure, production database, real provider key, or host service mutation was introduced.
- Confirm the evidence is strong enough to update each gate in `docs/henry-66-acceptance-matrix.md`.

## Output Format

Return a concise acceptance report:

- Overall verdict: `pass`, `pass-with-notes`, or `fail`
- Gate results: one line per matrix gate, with `pass`, `weak`, or `fail`
- Blocking issues: exact blockers, if any
- Evidence gaps: exact missing evidence, if any
- Recommendation: whether the main agent may summarize the goal as complete
