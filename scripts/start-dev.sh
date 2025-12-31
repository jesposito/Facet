#!/bin/bash
# Me.yaml Development Startup Script
set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo ""
echo "  Me.yaml - Your profile, expressed as data."
echo ""

# Environment
export DATA_DIR="${DATA_DIR:-$PROJECT_ROOT/pb_data}"
export ENCRYPTION_KEY="${ENCRYPTION_KEY:-dev-only-key-do-not-use-in-production!!}"
export SEED_DATA="true"

mkdir -p "$DATA_DIR"

# Build backend first (so we see compilation progress)
echo "Building backend..."
cd "$PROJECT_ROOT/backend"
go build -o ../pb_data/me-yaml .
cd "$PROJECT_ROOT"

# Start backend
echo "Starting backend..."
"$PROJECT_ROOT/pb_data/me-yaml" serve --http=0.0.0.0:8090 --dir="$DATA_DIR" &
BACKEND_PID=$!

# Wait for backend (up to 60 seconds)
echo "Waiting for backend..."
READY=false
for i in {1..60}; do
    if curl -s http://localhost:8090/api/health > /dev/null 2>&1; then
        READY=true
        break
    fi
    sleep 1
done

if [ "$READY" = false ]; then
    echo "ERROR: Backend failed to start after 60 seconds"
    kill $BACKEND_PID 2>/dev/null || true
    exit 1
fi

echo "Backend ready!"

# Start frontend
echo "Starting frontend..."
cd "$PROJECT_ROOT/frontend"
npm run dev -- --host &
FRONTEND_PID=$!
cd "$PROJECT_ROOT"

echo ""
echo "Ready!"
echo ""
echo "  Frontend:  http://localhost:5173"
echo "  API:       http://localhost:8090"
echo "  PB Admin:  http://localhost:8090/_/"
echo ""
echo "  Demo profile: Alex Chen"
echo "  Curated view: http://localhost:5173/v/recruiters"
echo ""

cleanup() {
    kill $FRONTEND_PID 2>/dev/null || true
    kill $BACKEND_PID 2>/dev/null || true
    exit 0
}

trap cleanup SIGINT SIGTERM
wait
