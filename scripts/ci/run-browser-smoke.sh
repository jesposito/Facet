#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
repo="$PWD"
scratch=$(mktemp -d /tmp/facet-smoke-XXXXXX)
mkdir -p "$scratch/logs" "$scratch/uploads"
pids=()
cleanup() {
  for pid in "${pids[@]}"; do kill "$pid" 2>/dev/null || true; done
  wait 2>/dev/null || true
  echo "Smoke-test logs: $scratch/logs"
}
trap cleanup EXIT

# All credentials and data belong exclusively to this disposable local fixture.
export ENCRYPTION_KEY=facet-ci-only-encryption-key-0000000000000000000000000000000000000
export ADMIN_EMAILS=smoke@example.com ADMIN_EMAIL=smoke@example.com ADMIN_PASSWORD=ci-only-facet-password-20261023
export SEED_DATA=dev UPLOADS_DIR="$scratch/uploads"
export POCKETBASE_URL=http://127.0.0.1:18090
export API_BASE_URL="$POCKETBASE_URL" PLAYWRIGHT_BASE_URL=http://127.0.0.1:18080
export ORIGIN="$PLAYWRIGHT_BASE_URL" HOST=127.0.0.1 PORT=13000

(cd backend && go build -buildvcs=false -o "$scratch/facet" .)
"$scratch/facet" serve --http=127.0.0.1:18090 --dir="$scratch/data" > "$scratch/logs/backend.log" 2>&1 &
pids+=("$!")
node "$repo/frontend/build/index.js" > "$scratch/logs/frontend.log" 2>&1 &
pids+=("$!")
node "$repo/scripts/ci/smoke-proxy.mjs" > "$scratch/logs/proxy.log" 2>&1 &
pids+=("$!")

ready=false
for attempt in {1..60}; do
  if curl -fsS "$PLAYWRIGHT_BASE_URL/health" >/dev/null 2>&1; then ready=true; break; fi
  for pid in "${pids[@]}"; do
    if ! kill -0 "$pid" 2>/dev/null; then cat "$scratch"/logs/*.log; exit 1; fi
  done
  sleep 1
done
if [ "$ready" != true ]; then cat "$scratch"/logs/*.log; exit 1; fi

# Complete the real first-login password change before tests share the account.
node --input-type=module <<'JS'
const auth = await fetch(`${process.env.API_BASE_URL}/api/collections/users/auth-with-password`, {
  method: 'POST', headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ identity: process.env.ADMIN_EMAIL, password: 'changeme123' })
});
if (!auth.ok) throw new Error(`Fixture login failed: ${auth.status}`);
const { token } = await auth.json();
const changed = await fetch(`${process.env.API_BASE_URL}/api/auth/change-password`, {
  method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: token },
  body: JSON.stringify({ currentPassword: 'changeme123', newPassword: process.env.ADMIN_PASSWORD })
});
if (!changed.ok) throw new Error(`Fixture password change failed: ${changed.status}`);
JS

cd frontend
npx playwright test tests/public-api.spec.ts tests/admin-flows.spec.ts tests/booking-shortcodes.spec.ts tests/dependency-security.spec.ts
