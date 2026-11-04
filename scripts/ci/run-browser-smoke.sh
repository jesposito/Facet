#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
repo="$PWD"
scratch=$(mktemp -d /tmp/facet-smoke-XXXXXX)
mkdir -p "$scratch/logs" "$scratch/uploads" "$scratch/data"
ln -s "$scratch/uploads" "$scratch/data/storage"
pids=()
cleanup() {
  status=$?
  trap - EXIT
  for ((i=${#pids[@]}-1; i>=0; i--)); do
    kill "${pids[$i]}" 2>/dev/null || true
  done
  for pid in "${pids[@]}"; do wait "$pid" 2>/dev/null || true; done
  rm -f "$scratch/owner-auth.json"
  echo "Smoke-test logs: $scratch/logs"
  exit "$status"
}
trap cleanup EXIT

SMOKE_HOST=${SMOKE_HOST:-127.0.0.1}
SMOKE_PROXY_PORT=${SMOKE_PROXY_PORT:-18080}
SMOKE_BACKEND_PORT=${SMOKE_BACKEND_PORT:-18090}
SMOKE_FRONTEND_PORT=${SMOKE_FRONTEND_PORT:-13000}
for port in "$SMOKE_PROXY_PORT" "$SMOKE_BACKEND_PORT" "$SMOKE_FRONTEND_PORT"; do
  if [[ ! "$port" =~ ^[0-9]+$ ]] || ((port < 1 || port > 65535)); then
    echo "Invalid smoke-test port: $port" >&2
    exit 1
  fi
done
if [[ "$SMOKE_PROXY_PORT" == "$SMOKE_BACKEND_PORT" || "$SMOKE_PROXY_PORT" == "$SMOKE_FRONTEND_PORT" || "$SMOKE_BACKEND_PORT" == "$SMOKE_FRONTEND_PORT" ]]; then
  echo "Smoke-test proxy, backend, and frontend ports must be distinct" >&2
  exit 1
fi
for port in "$SMOKE_PROXY_PORT" "$SMOKE_BACKEND_PORT" "$SMOKE_FRONTEND_PORT"; do
  if (exec 3<>"/dev/tcp/$SMOKE_HOST/$port") 2>/dev/null; then
    echo "Smoke-test port $port on $SMOKE_HOST is already in use" >&2
    exit 1
  fi
done

# All credentials and data belong exclusively to this disposable local fixture.
export ENCRYPTION_KEY=facet-ci-only-encryption-key-0000000000000000000000000000000000000
export ADMIN_EMAILS=smoke@example.com ADMIN_EMAIL=smoke@example.com ADMIN_PASSWORD=ci-only-facet-password-20261023
export SEED_DATA=dev UPLOADS_DIR="$scratch/uploads"
export POCKETBASE_URL="http://$SMOKE_HOST:$SMOKE_BACKEND_PORT"
export API_BASE_URL="$POCKETBASE_URL" PLAYWRIGHT_BASE_URL="http://$SMOKE_HOST:$SMOKE_PROXY_PORT"
export ORIGIN="$PLAYWRIGHT_BASE_URL" HOST="$SMOKE_HOST" PORT="$SMOKE_FRONTEND_PORT"
export APP_URL="$PLAYWRIGHT_BASE_URL"
export PROTOCOL_HEADER=x-forwarded-proto HOST_HEADER=x-forwarded-host ADDRESS_HEADER=x-facet-client-ip
export SMOKE_HOST SMOKE_PROXY_PORT SMOKE_BACKEND_PORT SMOKE_FRONTEND_PORT

(cd backend && go build -buildvcs=false -o "$scratch/facet" .)
start_backend() {
  STRIPE_SECRET_KEY=sk_test_facet_disposable_placeholder \
    "$scratch/facet" serve --http="$SMOKE_HOST:$SMOKE_BACKEND_PORT" --dir="$scratch/data" >> "$scratch/logs/backend.log" 2>&1 &
  backend_pid=$!
}
start_backend
pids+=("$backend_pid")
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

# Complete the real first-login password change, then authenticate once for all test workers.
FACET_OWNER_AUTH_FILE="$scratch/owner-auth.json" node --input-type=module <<'JS'
import { writeFile } from 'node:fs/promises';

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
const ownerAuth = await fetch(`${process.env.API_BASE_URL}/api/collections/users/auth-with-password`, {
  method: 'POST', headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ identity: process.env.ADMIN_EMAIL, password: process.env.ADMIN_PASSWORD })
});
if (!ownerAuth.ok) throw new Error(`Fixture owner authentication failed: ${ownerAuth.status}`);
const owner = await ownerAuth.json();
await writeFile(process.env.FACET_OWNER_AUTH_FILE, JSON.stringify({
  token: owner.token,
  record: owner.record,
  origin: new URL(process.env.PLAYWRIGHT_BASE_URL).origin,
  apiOrigin: new URL(process.env.API_BASE_URL).origin
}), { mode: 0o600 });
JS
export FACET_OWNER_AUTH_FILE="$scratch/owner-auth.json"

# Each browser file starts with fresh in-memory rate buckets. The database,
# owner session and production limits remain unchanged across the run.
restart_backend() {
  kill "$backend_pid"
  wait "$backend_pid" 2>/dev/null || true
  start_backend
  pids[0]="$backend_pid"
  for attempt in {1..60}; do
    if curl -fsS "$POCKETBASE_URL/api/health" >/dev/null 2>&1; then return 0; fi
    if ! kill -0 "$backend_pid" 2>/dev/null; then
      echo "Smoke-test backend exited during restart" >&2
      return 1
    fi
    sleep 1
  done
  echo "Smoke-test backend did not become ready after restart" >&2
  return 1
}

cd frontend
run_tests() {
  local status=0 result=0 index=0 spec
  if (($# > 0)); then
    COMMERCE_FIXTURE=1 npx playwright test --output=test-results/focused "$@" || status=$?
  else
    COMMERCE_FIXTURE=1 npx playwright test --project=public-api --output=test-results/public-api || status=$?
    COMMERCE_FIXTURE=1 npx playwright test --project=source-unit --output=test-results/source-unit || result=$?
    if ((status == 0)); then status=$result; fi

    npx playwright test --list --project=chromium --reporter=json > "$scratch/browser-list.json"
    node --input-type=module - "$scratch/browser-list.json" > "$scratch/browser-files" <<'JS'
import { readFile } from 'node:fs/promises';
import { relative, isAbsolute } from 'node:path';
const report = JSON.parse(await readFile(process.argv[2], 'utf8'));
const files = [...new Set(report.suites.map((suite) => suite.file))];
if (files.length === 0) throw new Error('No browser test files discovered');
for (const file of files) {
  const local = relative(process.cwd(), file);
  if (isAbsolute(local) || local.startsWith('..') || !local.endsWith('.spec.ts')) {
    throw new Error('Unexpected browser test file');
  }
  console.log(local);
}
JS
    local -a browser_files
    mapfile -t browser_files < "$scratch/browser-files"
    for spec in "${browser_files[@]}"; do
      restart_backend || return 1
      index=$((index + 1))
      result=0
      COMMERCE_FIXTURE=1 npx playwright test "$spec" --project=chromium \
        --output="test-results/browser-$index" || result=$?
      if ((status == 0)); then status=$result; fi
    done
  fi
  return "$status"
}
run_tests "$@"
