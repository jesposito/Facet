#!/bin/bash
# Facet Backend Development Script
# Optimized startup: only runs go mod tidy when go.mod/go.sum change
#
# Usage: ./scripts/dev-backend.sh
# Ref: https://go.dev/ref/mod#go-mod-tidy

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKEND_DIR="$PROJECT_ROOT/backend"

# Fail early if backend directory doesn't exist
if [[ ! -d "$BACKEND_DIR" ]]; then
    echo "[backend] ERROR: Backend directory not found: $BACKEND_DIR"
    exit 1
fi

if [[ ! -f "$BACKEND_DIR/go.mod" ]]; then
    echo "[backend] ERROR: go.mod not found in $BACKEND_DIR"
    exit 1
fi

echo "[backend] Project root: $PROJECT_ROOT"
echo "[backend] Backend dir: $BACKEND_DIR"

# Environment setup
export DATA_DIR="${DATA_DIR:-$PROJECT_ROOT/pb_data}"
export ENCRYPTION_KEY="${ENCRYPTION_KEY:-dev-only-key-do-not-use-in-production!!}"
export SEED_DATA="${SEED_DATA:-minimal}"
export FACET_DEV_HOST="${FACET_DEV_HOST:-127.0.0.1}"

# Air passes FACET_DEV_HOST to a shell command, so accept only hostname or
# IPv4 characters. Docker dev sets 0.0.0.0 so published ports reach the server.
case "$FACET_DEV_HOST" in
    *[!A-Za-z0-9.-]* | -* | .* | *. | *..*)
        echo "[backend] ERROR: FACET_DEV_HOST must be a hostname or IPv4 address"
        exit 1
        ;;
esac

echo "[backend] DATA_DIR: $DATA_DIR"
echo "[backend] Listening on: $FACET_DEV_HOST:8090"

# Ensure directories exist
mkdir -p "$DATA_DIR"
mkdir -p "$PROJECT_ROOT/tmp"

# Hash file location (in backend dir, gitignored)
HASH_FILE="$BACKEND_DIR/.gomod-hash"

# Calculate combined hash of go.mod and go.sum
get_gomod_hash() {
    local hash=""
    if [[ -f "$BACKEND_DIR/go.mod" ]]; then
        hash=$(sha256sum "$BACKEND_DIR/go.mod" | cut -d' ' -f1)
    fi
    if [[ -f "$BACKEND_DIR/go.sum" ]]; then
        local sum_hash=$(sha256sum "$BACKEND_DIR/go.sum" | cut -d' ' -f1)
        hash="${hash}-${sum_hash}"
    fi
    echo "$hash"
}

# Check if we need to update modules
needs_update() {
    # No hash file means first run
    if [[ ! -f "$HASH_FILE" ]]; then
        echo "first run (no hash file)"
        return 0
    fi

    # Compare current hash with cached
    local current_hash
    current_hash=$(get_gomod_hash)
    local cached_hash
    cached_hash=$(cat "$HASH_FILE" 2>/dev/null || echo "")

    if [[ "$current_hash" != "$cached_hash" ]]; then
        echo "go.mod/go.sum changed"
        return 0
    fi

    return 1
}

echo "[backend] Checking Go modules..."

if reason=$(needs_update); then
    echo "[backend] Running go mod tidy ($reason)..."
    echo "[backend] This may take a few minutes on first run..."

    # Change to backend directory where go.mod lives
    cd "$BACKEND_DIR"

    # Run go mod tidy - this downloads modules AND updates go.sum
    # Do NOT suppress stderr - we want to see any errors
    go mod tidy

    cd "$PROJECT_ROOT"

    # Save hash after successful update
    get_gomod_hash > "$HASH_FILE"
    echo "[backend] Modules updated, hash saved to $HASH_FILE"
else
    echo "[backend] Modules up to date (skipping go mod tidy)"
fi

# Check if air is available
if ! command -v air &> /dev/null; then
    echo "[backend] ERROR: air not found"
    echo "[backend] Install with: go install github.com/air-verse/air@v1.67.4"
    exit 1
fi

echo "[backend] Starting air with hot reload..."
echo "[backend] Config: $PROJECT_ROOT/.air.toml"
echo "[backend] Watching: $BACKEND_DIR"
echo "[backend] Build output: $PROJECT_ROOT/tmp/facet"
echo "[backend] PocketBase data: $DATA_DIR"
echo ""

# Run air from project root with explicit config
# Air will handle the build and restart on file changes
cd "$PROJECT_ROOT"
exec air -c "$PROJECT_ROOT/.air.toml"
