#!/bin/sh
# Facet Frontend Development Script
# Optimized startup: only installs dependencies when lockfile changes
#
# Usage: ./scripts/dev-frontend.sh
# POSIX sh so it also runs in the node:24-alpine dev container, which has no bash.
# Ref: https://docs.npmjs.com/cli/v10/commands/npm-ci

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
FRONTEND_DIR="$PROJECT_ROOT/frontend"

cd "$FRONTEND_DIR"

# Hash file location (inside node_modules so it's cleared on clean install)
HASH_FILE="$FRONTEND_DIR/node_modules/.lockfile-hash"
LOCKFILE="$FRONTEND_DIR/package-lock.json"

# Calculate current lockfile hash
get_lockfile_hash() {
    if [ -f "$LOCKFILE" ]; then
        sha256sum "$LOCKFILE" 2>/dev/null | cut -d' ' -f1
    else
        echo "no-lockfile"
    fi
}

# Check if we need to install
needs_install() {
    # No node_modules at all
    if [ ! -d "$FRONTEND_DIR/node_modules" ]; then
        echo "node_modules missing"
        return 0
    fi

    # No hash file (first time or was cleaned)
    if [ ! -f "$HASH_FILE" ]; then
        echo "hash file missing"
        return 0
    fi

    # Hash mismatch
    current_hash=$(get_lockfile_hash)
    cached_hash=$(cat "$HASH_FILE" 2>/dev/null || echo "")

    if [ "$current_hash" != "$cached_hash" ]; then
        echo "lockfile changed"
        return 0
    fi

    return 1
}

# Vite receives the host as an argument; accept only hostname or IPv4
# characters. Docker dev sets 0.0.0.0 so the published port reaches Vite.
FACET_DEV_HOST="${FACET_DEV_HOST:-127.0.0.1}"
case "$FACET_DEV_HOST" in
    *[!A-Za-z0-9.-]* | -* | .* | *. | *..*)
        echo "[frontend] ERROR: FACET_DEV_HOST must be a hostname or IPv4 address"
        exit 1
        ;;
esac

echo "[frontend] Checking dependencies..."

if reason=$(needs_install); then
    echo "[frontend] Installing dependencies ($reason)..."

    # Prefer npm ci for reproducible installs if lockfile exists
    if [ -f "$LOCKFILE" ]; then
        npm ci --loglevel=warn
    else
        npm install --loglevel=warn
    fi

    # Save the hash after successful install
    get_lockfile_hash > "$HASH_FILE"
    echo "[frontend] Dependencies installed and hash cached"
else
    echo "[frontend] Dependencies up to date (skipping npm install)"
fi

# Ensure SvelteKit types are generated
if [ ! -d "$FRONTEND_DIR/.svelte-kit" ]; then
    echo "[frontend] Generating SvelteKit types..."
    npx svelte-kit sync
fi

echo "[frontend] Starting Vite dev server on $FACET_DEV_HOST:5173..."
exec npm run dev -- --host "$FACET_DEV_HOST" --port 5173 --strictPort
