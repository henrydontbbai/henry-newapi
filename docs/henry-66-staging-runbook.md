# Henry /66 Staging Runbook

This runbook defines the target staging shape for `henry-newapi` on the Debian guest inside `/66`.

## Purpose

- Treat `/66` as a **preprod/staging** environment, not a disposable one-shot smoke target.
- Reuse the repository root `docker-compose.yml`, but pin staging onto a tracked override file: `docker-compose.henry-staging.override.yml`.
- Build the application from the current repository source instead of depending on `calciumion/new-api:latest`.
- Keep the runtime guest-local and loopback-only on `127.0.0.1:13000:3000`.

## Current live audit fact (`2026-07-04`)

- The authoritative LAN Device Ops backend exists at `C:\Users\HHPC\Documents\Codex\ssh-lan-device-codex`.
- The latest live read-only report is `C:\Users\HHPC\Documents\Codex\ssh-lan-device-codex\outputs\windows-66-readonly-2026-07-04T03-00-37+08-00.json`.
- The current `/66` classification is `reachable-but-not-deployed`.
- Current live facts from that audit:
  - host health remained normal
  - host `sshd` is running
  - guest loopback SSH `127.0.0.1:22222` is reachable and listening
  - neither `127.0.0.1:13000/api/status` nor `127.0.0.1:3000/api/status` returned a usable body

This means the current blocker is no longer "missing backend" or "unknown host reachability." The blocker is the staging HTTP runtime behind the already-live host/guest entry path. Do not treat the machine as staged until a later write-scope phase restores a healthy `/api/status`.

## Classification rules for the live `/66` audit

Use the device-ops backend to classify `/66` using these exact outcomes:

- `offline`
  - host SSH cannot be reached, or the host cannot reach guest `127.0.0.1:22222`
- `reachable-but-not-deployed`
  - host/guest entry works, but the guest lacks a usable workspace, Docker/Compose, or a running staging stack
- `staging-running`
  - guest stack is up from the tracked staging compose path, `/api/status` is healthy on `127.0.0.1:13000`, and a restart preserves health and data
- `staging-drifted`
  - guest is reachable, but the stack shape diverges from this runbook (wrong image source, wrong port exposure, unexpected host dependency, or stale mutable state)

## Tracked files used for staging

- Base compose: `docker-compose.yml`
- Staging override: `docker-compose.henry-staging.override.yml`
- Guest-local secret env file (untracked): `.env.henry-staging.local`

The override file fixes the staging contract:

- local-source image build through `Dockerfile`
- `new-api` exposed only on `127.0.0.1:13000:3000`
- `redis:7-alpine`
- `postgres:16-alpine`
- persistent guest-local data paths:
  - `./data-staging`
  - `./logs-staging`
  - named volume `henry_staging_pg_data`

## Guest-local secret file

Create this file only on the guest and do not commit it:

```bash
cd /home/newapi/henry-newapi
cat > .env.henry-staging.local <<'EOF'
HENRY_STAGING_POSTGRES_PASSWORD=change-me-postgres
HENRY_STAGING_REDIS_PASSWORD=change-me-redis
HENRY_STAGING_TZ=Asia/Shanghai
EOF
chmod 600 .env.henry-staging.local
```

## Read-only audit commands (once host/guest entry works)

Run these before changing anything:

```bash
cd /home/newapi/henry-newapi

pwd
ls -la
test -f docker-compose.henry-staging.override.yml && echo tracked_override_present
test -x /home/newapi/go-1.25.1/bin/go && /home/newapi/go-1.25.1/bin/go version || true
docker --version || true
docker compose version || true
docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'
ss -ltnp | grep -E '13000|3000|22222' || true
wget -q -O - http://127.0.0.1:13000/api/status || true
wget -q -O - http://127.0.0.1:13000/api/setup || true
```

Use the results to decide whether the state is `reachable-but-not-deployed`, `staging-running`, or `staging-drifted`.

## Current interpretation of the latest audit

Apply the runbook rules to the `2026-07-04` live snapshot as follows:

- not `offline`, because the host read-only refresh succeeded and the guest SSH loopback path is live
- not `staging-running`, because no usable `/api/status` response exists on `13000` or `3000`
- currently `reachable-but-not-deployed`, because the guest path exists but there is no evidenced running staging HTTP surface

Any later repair or redeploy must stay within a separate write-scope phase. This runbook section only records the current preprod classification.

## Staging bring-up procedure

Run these only after the audit either says `reachable-but-not-deployed` or after you intentionally repair a `staging-drifted` stack.

```bash
set -euo pipefail

cd /home/newapi/henry-newapi
mkdir -p data-staging logs-staging

docker compose \
  --env-file .env.henry-staging.local \
  -f docker-compose.yml \
  -f docker-compose.henry-staging.override.yml \
  config >/home/newapi/henry-newapi-staging.rendered.yml

docker compose \
  --env-file .env.henry-staging.local \
  -f docker-compose.yml \
  -f docker-compose.henry-staging.override.yml \
  up -d --build

docker compose \
  --env-file .env.henry-staging.local \
  -f docker-compose.yml \
  -f docker-compose.henry-staging.override.yml \
  ps
```

## Health and setup verification

Verify health on the guest loopback only:

```bash
python3 - <<'PY'
import json
import urllib.request

for path in ('/api/status', '/api/setup'):
    with urllib.request.urlopen(f'http://127.0.0.1:13000{path}', timeout=5) as response:
        payload = json.loads(response.read().decode('utf-8'))
    print(path, json.dumps(payload, ensure_ascii=False))
PY
```

If `/api/setup` reports setup is still required, initialize only with disposable staging credentials and no production secrets:

```bash
python3 - <<'PY'
import json
import urllib.request

payload = {
    "username": "root66",
    "password": "StagePass66",
    "confirmPassword": "StagePass66",
    "SelfUseModeEnabled": True,
    "DemoSiteEnabled": False,
}
req = urllib.request.Request(
    'http://127.0.0.1:13000/api/setup',
    data=json.dumps(payload).encode('utf-8'),
    headers={'Content-Type': 'application/json'},
    method='POST',
)
with urllib.request.urlopen(req, timeout=10) as response:
    print(response.read().decode('utf-8'))
PY
```

Then re-check:

```bash
wget -q -O - http://127.0.0.1:13000/api/status
wget -q -O - http://127.0.0.1:13000/api/setup
```

## First-run staging safety rules

During the first successful staging closure:

- keep `routing_policy_setting.mode=observe`
- keep `probe_policy.active_probe_enabled=false`
- do not load real production API keys
- use only disposable test channels and test credentials
- do not add any public exposure
- do not introduce any dependency on Windows host `3306`, `5000`, or the old `23000` path

## Restart persistence check

A staging stack is not accepted until it survives one restart and remains healthy:

```bash
docker compose \
  --env-file .env.henry-staging.local \
  -f docker-compose.yml \
  -f docker-compose.henry-staging.override.yml \
  restart

sleep 5

python3 - <<'PY'
import json
import urllib.request
with urllib.request.urlopen('http://127.0.0.1:13000/api/status', timeout=5) as response:
    payload = json.loads(response.read().decode('utf-8'))
print(json.dumps(payload, ensure_ascii=False))
if payload.get('success') is not True:
    raise SystemExit(1)
PY
```

If that succeeds and the data/setup state remains intact, classify the environment as `staging-running`.

## Rollback / cleanup

Stop the staging stack without deleting persistent state:

```bash
docker compose \
  --env-file .env.henry-staging.local \
  -f docker-compose.yml \
  -f docker-compose.henry-staging.override.yml \
  down
```

Only remove volumes and persistent data when you intentionally want a full staging reset:

```bash
docker compose \
  --env-file .env.henry-staging.local \
  -f docker-compose.yml \
  -f docker-compose.henry-staging.override.yml \
  down -v
rm -rf data-staging logs-staging
```
