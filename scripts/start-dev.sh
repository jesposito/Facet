#!/usr/bin/env bash
# Start this checkout's development services and stop them together on exit.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
if [[ -f "$PROJECT_ROOT/.env" ]]; then
  set -a
  source "$PROJECT_ROOT/.env"
  set +a
fi
export DATA_DIR="${DATA_DIR:-$PROJECT_ROOT/pb_data}"
export ENCRYPTION_KEY="${ENCRYPTION_KEY:-dev-only-key-do-not-use-in-production!!}"
export SEED_DATA="${SEED_DATA:-minimal}"
exec python3 "$SCRIPT_DIR/dev-server.py"
