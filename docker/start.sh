#!/bin/bash
set -e

echo "Starting Facet..."

# Start PocketBase backend (internal only, bound to localhost)
./facet serve --http=127.0.0.1:8090 --dir=/data &
BACKEND_PID=$!

# Wait for backend
for i in $(seq 1 30); do
    if wget -q --spider http://127.0.0.1:8090/api/health 2>/dev/null; then
        break
    fi
    sleep 1
done

# Start SvelteKit frontend (internal only)
cd frontend
node build/index.js &
FRONTEND_PID=$!
cd ..

# Start Caddy reverse proxy (single public entry point)
caddy run --config ./Caddyfile &
CADDY_PID=$!

echo ""
echo "Facet is running on port 8080"
if [ "$ADMIN_ENABLED" = "true" ]; then
    echo "PocketBase admin available at /_/"
fi
echo ""

# Handle shutdown
trap "kill $CADDY_PID $FRONTEND_PID $BACKEND_PID 2>/dev/null" EXIT
wait -n $BACKEND_PID $FRONTEND_PID $CADDY_PID
