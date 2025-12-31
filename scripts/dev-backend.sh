#!/bin/bash
# Me.yaml Backend Development Script
# Optimized startup: only downloads modules when go.mod/go.sum change
#
# Usage: ./scripts/dev-backend.sh
# Ref: https://go.dev/ref/mod#go-mod-download

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKEND_DIR="$PROJECT_ROOT/backend"

cd "$PROJECT_ROOT"

# Environment setup
export DATA_DIR="${DATA_DIR:-$PROJECT_ROOT/pb_data}"
export ENCRYPTION_KEY="${ENCRYPTION_KEY:-dev-only-key-do-not-use-in-production!!}"
export SEED_DATA="${SEED_DATA:-true}"

# Ensure directories exist
mkdir -p "$DATA_DIR"
mkdir -p "$PROJECT_ROOT/tmp"

# Hash file location
HASH_FILE="$BACKEND_DIR/.gomod-hash"

# Calculate combined hash of go.mod and go.sum
get_gomod_hash() {
    local hash=""
    if [[ -f "$BACKEND_DIR/go.mod" ]]; then
        hash=$(sha256sum "$BACKEND_DIR/go.mod" 2>/dev/null | cut -d' ' -f1)
    fi
    if [[ -f "$BACKEND_DIR/go.sum" ]]; then
        local sum_hash=$(sha256sum "$BACKEND_DIR/go.sum" 2>/dev/null | cut -d' ' -f1)
        hash="${hash}-${sum_hash}"
    fi
    echo "$hash"
}

# Check if we need to download modules
needs_download() {
    # No hash file
    if [[ ! -f "$HASH_FILE" ]]; then
        echo "first run"
        return 0
    fi

    # Hash mismatch
    local current_hash=$(get_gomod_hash)
    local cached_hash=$(cat "$HASH_FILE" 2>/dev/null || echo "")

    if [[ "$current_hash" != "$cached_hash" ]]; then
        echo "go.mod/go.sum changed"
        return 0
    fi

    return 1
}

echo "[backend] Checking Go modules..."

if reason=$(needs_download); then
    echo "[backend] Downloading modules ($reason)..."
    cd "$BACKEND_DIR"
    go mod download
    cd "$PROJECT_ROOT"

    # Save hash after successful download
    get_gomod_hash > "$HASH_FILE"
    echo "[backend] Modules downloaded and hash cached"
else
    echo "[backend] Modules up to date (skipping go mod download)"
fi

# Check if air is available
if ! command -v air &> /dev/null; then
    echo "[backend] ERROR: air not found. Install with: go install github.com/air-verse/air@latest"
    exit 1
fi

echo "[backend] Starting with hot reload (air)..."
echo "[backend] Watching: $BACKEND_DIR"
echo "[backend] Output: $PROJECT_ROOT/tmp/me-yaml"

# Run air from project root with explicit config
exec air -c "$PROJECT_ROOT/.air.toml"
