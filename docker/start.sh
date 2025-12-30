#!/bin/sh
set -e

echo "Starting OwnProfile..."

# Start PocketBase backend in background
echo "Starting backend on port 8090..."
./ownprofile serve --http=0.0.0.0:8090 --dir=/data &
BACKEND_PID=$!

# Wait for backend to be ready
echo "Waiting for backend..."
for i in $(seq 1 30); do
    if wget --no-verbose --tries=1 --spider http://localhost:8090/api/health 2>/dev/null; then
        echo "Backend is ready!"
        break
    fi
    sleep 1
done

# Start SvelteKit frontend
echo "Starting frontend on port ${PORT:-3000}..."
cd frontend
node build/index.js &
FRONTEND_PID=$!

echo "OwnProfile is running!"
echo "  - Backend:  http://localhost:8090"
echo "  - Frontend: http://localhost:${PORT:-3000}"
echo "  - Admin UI: http://localhost:8090/_/"

# Handle shutdown
trap "kill $BACKEND_PID $FRONTEND_PID 2>/dev/null" EXIT

# Wait for either process to exit
wait -n $BACKEND_PID $FRONTEND_PID
