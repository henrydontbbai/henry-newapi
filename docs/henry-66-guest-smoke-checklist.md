# Henry /66 Guest Isolated Smoke Checklist

This file is a guest-side checklist for the step after targeted Go tests pass. It is not proof that smoke testing already passed.

## Scope

- Target: Debian guest inside `/66`, not the Windows host.
- Current host-side guest SSH mapping: `127.0.0.1:22222 -> guest:22`.
- Runtime: Docker Compose inside the guest.
- Database/cache: guest-local Postgres + Redis from the project compose stack.
- Do not use `/66` Windows host MySQL `3306`.
- Do not use or stop `/66` Windows host Python service `5000`.
- Do not expose a public port in this step.

## Preconditions

- The targeted Go verification in `docs/henry-66-guest-verification-commands.md` has passed in the same guest workspace.
- Guest workspace: `/home/newapi/henry-newapi`.
- Docker Engine and Compose plugin are available in the guest.
- Any existing `new-api`, `postgres`, or `redis` containers in the guest are disposable test containers, not unrelated services.

## Host-Side Entry Note

- Connect to the Windows `/66` host first.
- From that host, reach the Debian guest through `127.0.0.1:22222 -> guest:22`.
- Do not write passwords, private keys, or other credentials into this repository while doing that handoff.

## Port Strategy

Default project compose maps guest `3000:3000`. For the first isolated smoke, prefer a guest-only high port mapping:

```bash
cat > docker-compose.henry-smoke.override.yml <<'YAML'
services:
  new-api:
    ports:
      - "127.0.0.1:13000:3000"
YAML
```

Use this override with:

```bash
docker compose -f docker-compose.yml -f docker-compose.henry-smoke.override.yml up -d --build
```

This keeps the browser/API check on the guest loopback unless a separate host port-forwarding step is explicitly authorized later.

## Image Availability Note

Default compose uses `calciumion/new-api:latest`, `postgres:15`, and `redis:latest`.

If the guest cannot pull from Docker Hub during smoke:

- do not switch to Windows host databases or host services;
- keep the smoke guest-local;
- prefer a disposable guest-local fallback using:
  - current guest-built `./new-api` bind-mounted to `/new-api:ro`;
  - cached guest-local `postgres:16-alpine`;
  - cached guest-local `redis:7-alpine`;
  - a temporary full smoke compose file such as `docker-compose.henry-smoke.full.yml`;
- preserve the same acceptance contract:
  - `127.0.0.1:13000 -> 3000`
  - guest-local Postgres/Redis only
  - `/api/status` success
  - setup completed or blocker recorded
  - no host `3306` / `5000` dependency
  - no public exposure

## Smoke Commands

```bash
set -euo pipefail

cd /home/newapi/henry-newapi

mkdir -p data logs web/default/dist web/classic/dist
printf '<!doctype html><html><body>placeholder</body></html>' > web/default/dist/index.html
printf '<!doctype html><html><body>placeholder</body></html>' > web/classic/dist/index.html

cat > docker-compose.henry-smoke.override.yml <<'YAML'
services:
  new-api:
    ports:
      - "127.0.0.1:13000:3000"
    environment:
      - ERROR_LOG_ENABLED=true
      - BATCH_UPDATE_ENABLED=true
      - TZ=Asia/Shanghai
YAML

docker compose -f docker-compose.yml -f docker-compose.henry-smoke.override.yml down || true
docker compose -f docker-compose.yml -f docker-compose.henry-smoke.override.yml config >/home/newapi/henry-newapi-compose-rendered.yml
docker compose -f docker-compose.yml -f docker-compose.henry-smoke.override.yml up -d --build

docker compose -f docker-compose.yml -f docker-compose.henry-smoke.override.yml ps

for i in $(seq 1 60); do
  if python3 - <<'PY'
import json
import sys
import urllib.request

try:
    with urllib.request.urlopen("http://127.0.0.1:13000/api/status", timeout=3) as response:
        payload = json.loads(response.read().decode("utf-8"))
    sys.exit(0 if payload.get("success") is True else 1)
except Exception:
    sys.exit(1)
PY
  then
    echo "api status ok"
    break
  fi
  sleep 2
  if [ "$i" -eq 60 ]; then
    echo "api status timeout"
    docker compose -f docker-compose.yml -f docker-compose.henry-smoke.override.yml logs --tail=200 new-api
    exit 1
  fi
done
```

## Evidence To Capture

After the current-snapshot smoke rerun, capture at least:

- rendered compose config path: `/home/newapi/henry-newapi-compose-rendered.yml`
- `docker compose ... ps`
- successful `/api/status` response body from `http://127.0.0.1:13000/api/status`
- if setup is required, the exact setup state or blocker
- if the stack fails, `docker compose ... logs --tail=200 new-api`

These outputs are the minimum needed to refresh:

- `docs/henry-66-runtime-evidence-log.md`
- `docs/henry-66-acceptance-matrix.md`
- the final read-only audit

## Manual Setup Gate

After `/api/status` is healthy:

- complete the normal NewAPI root/admin initialization if the app reports setup is required;
- do not put real production keys into this smoke stack;
- use only disposable test channels/data;
- keep `routing_policy_setting.mode=observe` for the first automation run;
- keep `probe_policy.active_probe_enabled=false` until a valid root/admin test user is confirmed.

## Routing Smoke Acceptance

Minimum acceptance for this stage:

- compose stack starts in the guest;
- `/api/status` is healthy through `127.0.0.1:13000`;
- root/admin setup path is understood and completed or explicitly reported as blocked;
- routing policy can be loaded with default `observe`;
- observe mode does not mutate channel status;
- no dependency on Windows host MySQL `3306`;
- no dependency on Windows host Python `5000`;
- no public exposure is added.

## Rollback

For a disposable smoke stack:

```bash
cd /home/newapi/henry-newapi
docker compose -f docker-compose.yml -f docker-compose.henry-smoke.override.yml down
```

Only remove volumes after confirming they are disposable guest smoke volumes:

```bash
docker compose -f docker-compose.yml -f docker-compose.henry-smoke.override.yml down -v
```

For the temporary full-fallback smoke file:

```bash
cd /home/newapi/henry-newapi
docker compose -f docker-compose.henry-smoke.full.yml down -v
```
