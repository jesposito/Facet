#!/bin/bash
# Me.yaml Development Startup Script
# Starts both frontend and backend with hot reloading
#
# Usage: ./scripts/start-dev.sh
#
# This script:
# 1. Uses lockfile-hash caching to skip unnecessary installs
# 2. Starts backend with air for Go hot reload
# 3. Starts frontend with Vite HMR
# 4. Waits for backend to be ready before starting frontend

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

echo ""
echo "  Me.yaml — You, human-readable."
echo ""

# Ensure scripts are executable
chmod +x "$SCRIPT_DIR/dev-backend.sh" "$SCRIPT_DIR/dev-frontend.sh" 2>/dev/null || true

# Environment
export DATA_DIR="${DATA_DIR:-$PROJECT_ROOT/pb_data}"
export ENCRYPTION_KEY="${ENCRYPTION_KEY:-dev-only-key-do-not-use-in-production!!}"
export SEED_DATA="true"

mkdir -p "$DATA_DIR"
mkdir -p "$PROJECT_ROOT/tmp"

# Log file for capturing backend output
BACKEND_LOG="$PROJECT_ROOT/tmp/backend.log"

# Start backend with output captured to log
echo "[startup] Starting backend..."
"$SCRIPT_DIR/dev-backend.sh" 2>&1 | tee "$BACKEND_LOG" &
BACKEND_PID=$!

# Wait for backend health (max 180 seconds for first build)
# First build compiles PocketBase + all dependencies, which takes time
echo "[startup] Waiting for backend (first build may take 2-3 minutes)..."
READY=false
for i in {1..180}; do
    if curl -s http://localhost:8090/api/health > /dev/null 2>&1; then
        READY=true
        break
    fi
    # Check if backend process died
    if ! kill -0 $BACKEND_PID 2>/dev/null; then
        echo "[startup] ERROR: Backend process exited unexpectedly"
        exit 1
    fi
    # Show progress every 30 seconds
    if (( i % 30 == 0 )); then
        echo "[startup] Still waiting... ($i seconds)"
    fi
    sleep 1
done

if [ "$READY" = false ]; then
    echo "[startup] ERROR: Backend failed to start after 180 seconds"

    # Check for common errors in the log
    if grep -q "Collection name must be unique" "$BACKEND_LOG" 2>/dev/null; then
        echo ""
        echo "[startup] DETECTED: Migration error - duplicate collections"
        echo "[startup] This can happen if pb_data has stale data from a previous run."
        echo "[startup] To fix, run: rm -rf pb_data && ./scripts/start-dev.sh"
        echo ""
    fi

    if grep -q "missing go.sum entry" "$BACKEND_LOG" 2>/dev/null; then
        echo ""
        echo "[startup] DETECTED: Missing go.sum entries"
        echo "[startup] To fix, run: cd backend && go mod tidy"
        echo ""
    fi

    kill $BACKEND_PID 2>/dev/null || true
    exit 1
fi

echo "[startup] Backend ready!"

# Start frontend
echo "[startup] Starting frontend..."
"$SCRIPT_DIR/dev-frontend.sh" &
FRONTEND_PID=$!

echo ""
echo "  Ready! (with hot reload)"
echo ""
echo "  Frontend:  http://localhost:5173  (auto-reloads on save)"
echo "  API:       http://localhost:8090  (auto-rebuilds on save)"
echo "  PB Admin:  http://localhost:8090/_/"
echo ""
echo "  Press Ctrl+C to stop all services"
echo ""

# Cleanup handler
cleanup() {
    echo ""
    echo "[startup] Shutting down..."
    kill $FRONTEND_PID 2>/dev/null || true
    kill $BACKEND_PID 2>/dev/null || true
    # Kill any child processes
    pkill -P $$ 2>/dev/null || true
    exit 0
}

trap cleanup SIGINT SIGTERM EXIT

# Wait for either process to exit
wait -n $BACKEND_PID $FRONTEND_PID 2>/dev/null || true

# If we get here, one of them died
echo "[startup] A service exited unexpectedly"
cleanup
