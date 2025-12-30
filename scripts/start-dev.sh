#!/bin/bash
# me.yaml Development Startup Script
set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo ""
echo "  me.yaml - Your profile, expressed as data."
echo ""

# Environment
export DATA_DIR="${DATA_DIR:-$PROJECT_ROOT/pb_data}"
export ENCRYPTION_KEY="${ENCRYPTION_KEY:-dev-only-key-do-not-use-in-production!!}"
export SEED_DATA="true"

mkdir -p "$DATA_DIR"

# Start backend
echo "Starting backend..."
cd "$PROJECT_ROOT/backend"
go run . serve --http=0.0.0.0:8090 &
BACKEND_PID=$!
cd "$PROJECT_ROOT"

# Wait for backend
echo "Waiting for backend..."
for i in {1..30}; do
    if curl -s http://localhost:8090/api/health > /dev/null 2>&1; then
        break
    fi
    sleep 1
done

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
