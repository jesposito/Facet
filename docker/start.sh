#!/bin/bash
set -e

echo "========================================"
echo "  Facet - Starting up..."
echo "========================================"
echo ""

PUID=${PUID:-1000}
PGID=${PGID:-1000}

echo "[Config] Running with PUID=$PUID PGID=$PGID"

if [ "$(id -u)" = "0" ]; then
    groupmod -o -g "$PGID" facet 2>/dev/null || groupadd -o -g "$PGID" facet
    usermod -o -u "$PUID" -g facet facet 2>/dev/null || useradd -o -u "$PUID" -g facet -s /bin/bash -m facet
    
    chown -R facet:facet /app /data /uploads
    
    echo "[Config] Switching to user facet (PUID=$PUID, PGID=$PGID)"
    exec gosu facet "$0" "$@"
fi

DATA_DIR="/data"
UPLOADS_DIR="/uploads"
KEY_FILE="$DATA_DIR/.encryption_key"
STORAGE_PATH="$DATA_DIR/pb_data/storage"

mkdir -p "$DATA_DIR/pb_data"

if [ -n "$ENCRYPTION_KEY" ]; then
    echo "[Config] Using ENCRYPTION_KEY from environment"
elif [ -f "$KEY_FILE" ]; then
    export ENCRYPTION_KEY=$(cat "$KEY_FILE")
    echo "[Config] Loaded ENCRYPTION_KEY from $KEY_FILE"
else
    export ENCRYPTION_KEY=$(openssl rand -hex 32)
    echo "$ENCRYPTION_KEY" > "$KEY_FILE"
    chmod 600 "$KEY_FILE"
    echo ""
    echo "========================================"
    echo "  ENCRYPTION KEY GENERATED"
    echo "========================================"
    echo ""
    echo "  A new encryption key has been generated"
    echo "  and saved to: $KEY_FILE"
    echo ""
    echo "  Key: $ENCRYPTION_KEY"
    echo ""
    echo "  IMPORTANT: This key encrypts your API"
    echo "  tokens. Back up your /data directory!"
    echo ""
    echo "========================================"
    echo ""
fi

if [ -d "$UPLOADS_DIR" ]; then
    if [ -L "$STORAGE_PATH" ]; then
        CURRENT_TARGET=$(readlink "$STORAGE_PATH")
        if [ "$CURRENT_TARGET" = "$UPLOADS_DIR" ]; then
            echo "[Storage] Symlink already configured: $STORAGE_PATH -> $UPLOADS_DIR"
        else
            echo "[Storage] Updating symlink target..."
            rm "$STORAGE_PATH"
            ln -s "$UPLOADS_DIR" "$STORAGE_PATH"
            echo "[Storage] Symlink updated: $STORAGE_PATH -> $UPLOADS_DIR"
        fi
    elif [ -d "$STORAGE_PATH" ]; then
        if [ "$(ls -A $STORAGE_PATH 2>/dev/null)" ]; then
            echo ""
            echo "========================================"
            echo "  STORAGE MIGRATION NEEDED"
            echo "========================================"
            echo ""
            echo "  Existing uploads found in: $STORAGE_PATH"
            echo "  New uploads directory: $UPLOADS_DIR"
            echo ""
            echo "  Please manually move your files:"
            echo "    mv $STORAGE_PATH/* $UPLOADS_DIR/"
            echo "    rm -rf $STORAGE_PATH"
            echo ""
            echo "  Then restart the container."
            echo ""
            echo "========================================"
            echo ""
        else
            rmdir "$STORAGE_PATH"
            ln -s "$UPLOADS_DIR" "$STORAGE_PATH"
            echo "[Storage] Symlink created: $STORAGE_PATH -> $UPLOADS_DIR"
        fi
    else
        ln -s "$UPLOADS_DIR" "$STORAGE_PATH"
        echo "[Storage] Symlink created: $STORAGE_PATH -> $UPLOADS_DIR"
    fi
else
    echo "[Storage] Using default storage location: $STORAGE_PATH"
    mkdir -p "$STORAGE_PATH"
fi

echo ""
echo "[Backend] Starting PocketBase..."
./facet serve --http=127.0.0.1:8090 --dir=/data &
BACKEND_PID=$!

echo "[Backend] Waiting for health check..."
for i in $(seq 1 30); do
    if wget -q --spider http://127.0.0.1:8090/api/health 2>/dev/null; then
        echo "[Backend] Ready!"
        break
    fi
    sleep 1
done

echo "[Frontend] Starting SvelteKit..."
cd frontend
node build/index.js &
FRONTEND_PID=$!
cd ..

echo "[Proxy] Starting Caddy..."
caddy run --config ./Caddyfile &
CADDY_PID=$!

echo ""
echo "========================================"
echo "  Facet is running!"
echo "========================================"
echo ""
echo "  Web UI: http://localhost:8080"
echo "  Admin:  http://localhost:8080/admin"
echo ""
echo "  Default login:"
echo "    Email:    admin@example.com"
echo "    Password: changeme123"
echo ""
if [ "$ADMIN_ENABLED" = "true" ]; then
    echo "  PocketBase Admin: http://localhost:8080/_/"
    echo ""
fi
echo "========================================"
echo ""

trap "kill $CADDY_PID $FRONTEND_PID $BACKEND_PID 2>/dev/null" EXIT
wait -n $BACKEND_PID $FRONTEND_PID $CADDY_PID
