#!/bin/bash
set -e

FACET_VERSION=${FACET_VERSION:-dev}
BUILD_DATE=${BUILD_DATE:-unknown}
GIT_COMMIT=${GIT_COMMIT:-unknown}

echo ""
echo "========================================"
echo "  ███████╗ █████╗  ██████╗███████╗████████╗"
echo "  ██╔════╝██╔══██╗██╔════╝██╔════╝╚══██╔══╝"
echo "  █████╗  ███████║██║     █████╗     ██║   "
echo "  ██╔══╝  ██╔══██║██║     ██╔══╝     ██║   "
echo "  ██║     ██║  ██║╚██████╗███████╗   ██║   "
echo "  ╚═╝     ╚═╝  ╚═╝ ╚═════╝╚══════╝   ╚═╝   "
echo "========================================"
echo "  Version: $FACET_VERSION"
if [ "$BUILD_DATE" != "unknown" ]; then
    echo "  Built:   $BUILD_DATE"
fi
if [ "$GIT_COMMIT" != "unknown" ]; then
    echo "  Commit:  ${GIT_COMMIT:0:7}"
fi
echo "========================================"
echo ""

PUID=${PUID:-1000}
PGID=${PGID:-1000}

echo "[Config] PUID=$PUID PGID=$PGID"
echo "[Config] Data directory: /data"
echo "[Config] Uploads directory: /uploads"

if [ "$(id -u)" = "0" ]; then
    groupmod -o -g "$PGID" facet 2>/dev/null || groupadd -o -g "$PGID" facet
    usermod -o -u "$PUID" -g facet facet 2>/dev/null || useradd -o -u "$PUID" -g facet -s /bin/bash -m facet
    
    chown -R facet:facet /app /data /uploads
    
    echo "[Config] Switching to user facet (PUID=$PUID, PGID=$PGID)"
    exec gosu facet "$0" "$@"
fi

DATA_DIR="/data"
export UPLOADS_DIR="/uploads"  # Export so Go app can use it for direct storage
KEY_FILE="$DATA_DIR/.encryption_key"
# PocketBase --dir=/data means storage is at /data/storage (not /data/pb_data/storage)
STORAGE_PATH="$DATA_DIR/storage"

if [ -n "$ENCRYPTION_KEY" ]; then
    export ENCRYPTION_KEY_SOURCE="env"
    echo "[Config] Using ENCRYPTION_KEY from environment"
elif [ -f "$KEY_FILE" ]; then
    export ENCRYPTION_KEY=$(cat "$KEY_FILE")
    export ENCRYPTION_KEY_SOURCE="file"
    echo "[Config] Loaded ENCRYPTION_KEY from $KEY_FILE"
else
    export ENCRYPTION_KEY=$(openssl rand -hex 32)
    export ENCRYPTION_KEY_SOURCE="auto"
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

# Storage setup - gracefully handle existing files
STORAGE_BACKUP="$DATA_DIR/.storage-backup"

if [ -d "$UPLOADS_DIR" ]; then
    if [ -L "$STORAGE_PATH" ]; then
        # Symlink exists - verify it points to uploads
        CURRENT_TARGET=$(readlink "$STORAGE_PATH")
        if [ "$CURRENT_TARGET" = "$UPLOADS_DIR" ]; then
            echo "[Storage] Using uploads directory: $UPLOADS_DIR"
        else
            rm "$STORAGE_PATH"
            ln -s "$UPLOADS_DIR" "$STORAGE_PATH"
            echo "[Storage] Using uploads directory: $UPLOADS_DIR"
        fi

        # Self-healing: a previous migration may have moved files into
        # $STORAGE_BACKUP without successfully copying them to $UPLOADS_DIR.
        # Recover them now so the symlink resolves to a populated tree.
        if [ -d "$STORAGE_BACKUP" ] && [ "$(ls -A "$STORAGE_BACKUP" 2>/dev/null)" ]; then
            UPLOAD_FILE_COUNT=$(find "$UPLOADS_DIR" -type f 2>/dev/null | wc -l)
            BACKUP_FILE_COUNT=$(find "$STORAGE_BACKUP" -type f 2>/dev/null | wc -l)
            if [ "$BACKUP_FILE_COUNT" -gt 0 ] && [ "$UPLOAD_FILE_COUNT" -lt "$BACKUP_FILE_COUNT" ]; then
                echo "[Storage] Recovering from $STORAGE_BACKUP ($BACKUP_FILE_COUNT files backed up, $UPLOAD_FILE_COUNT in active uploads)"
                if cp -an "$STORAGE_BACKUP"/* "$UPLOADS_DIR"/ 2>/dev/null; then
                    NEW_COUNT=$(find "$UPLOADS_DIR" -type f 2>/dev/null | wc -l)
                    echo "[Storage] Recovered files — /uploads now has $NEW_COUNT files (was $UPLOAD_FILE_COUNT)"
                else
                    echo "[Storage] WARNING: copy from $STORAGE_BACKUP failed — check permissions"
                fi
            fi
        fi
    elif [ -d "$STORAGE_PATH" ] && [ "$(ls -A $STORAGE_PATH 2>/dev/null)" ]; then
        # Existing files in old location - migrate gracefully
        OLD_COUNT=$(find "$STORAGE_PATH" -type f ! -name "*.attrs" 2>/dev/null | wc -l)
        echo "[Storage] Found $OLD_COUNT files in legacy location"
        echo "[Storage] Copying to uploads directory (keeping backup)..."

        # Copy files to uploads directory.
        # Use -n (no-clobber) so an interrupted previous run can be resumed
        # safely without overwriting good files in $UPLOADS_DIR.
        if cp -an "$STORAGE_PATH"/* "$UPLOADS_DIR"/ 2>/dev/null; then
            COPIED_COUNT=$(find "$UPLOADS_DIR" -type f ! -name "*.attrs" 2>/dev/null | wc -l)
            EXPECTED_COUNT=$(find "$STORAGE_PATH" -type f ! -name "*.attrs" 2>/dev/null | wc -l)
            if [ "$COPIED_COUNT" -lt "$EXPECTED_COUNT" ]; then
                echo "[Storage] WARNING: only $COPIED_COUNT of $EXPECTED_COUNT files reached uploads — keeping legacy storage in place, NOT moving to backup"
                # Bail out before the mv so we never strand files in a backup
                # the user can't tell is the source of truth.
                exit 1
            fi
            echo "[Storage] Copied $COPIED_COUNT files to uploads directory"

            # Move old storage to backup (not delete)
            if [ -d "$STORAGE_BACKUP" ]; then
                # Backup already exists from previous migration - remove old storage
                rm -rf "$STORAGE_PATH"
            else
                mv "$STORAGE_PATH" "$STORAGE_BACKUP"
                echo "[Storage] Original files backed up to: $STORAGE_BACKUP"
            fi

            # Create symlink
            ln -s "$UPLOADS_DIR" "$STORAGE_PATH"
            echo "[Storage] Using uploads directory: $UPLOADS_DIR"
            echo "[Storage] (backup at $STORAGE_BACKUP can be removed once verified)"
        else
            echo "[Storage] Could not copy files (check permissions/space)"
            echo "[Storage] Using legacy location: $STORAGE_PATH"
        fi
    elif [ -d "$STORAGE_PATH" ]; then
        # Empty directory - replace with symlink
        rmdir "$STORAGE_PATH" 2>/dev/null
        ln -s "$UPLOADS_DIR" "$STORAGE_PATH"
        echo "[Storage] Using uploads directory: $UPLOADS_DIR"
    else
        # Fresh install - create symlink
        ln -s "$UPLOADS_DIR" "$STORAGE_PATH"
        echo "[Storage] Using uploads directory: $UPLOADS_DIR"
    fi
else
    # No /uploads mount - use default location
    echo "[Storage] No /uploads mount detected"
    echo "[Storage] Using default location: $STORAGE_PATH"
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

# Determine admin email (mirrors logic in seed.go getSeedAdminEmail)
# Priority: 1) First email from ADMIN_EMAILS, 2) DEV_ADMIN_EMAIL, 3) admin@example.com
ADMIN_LOGIN_EMAIL="admin@example.com"
if [ -n "$ADMIN_EMAILS" ]; then
    # Get first email from comma-separated list, trim whitespace
    FIRST_EMAIL=$(echo "$ADMIN_EMAILS" | cut -d',' -f1 | xargs)
    if [ -n "$FIRST_EMAIL" ]; then
        ADMIN_LOGIN_EMAIL="$FIRST_EMAIL"
    fi
elif [ -n "$DEV_ADMIN_EMAIL" ]; then
    ADMIN_LOGIN_EMAIL="$DEV_ADMIN_EMAIL"
fi

echo "  Default login:"
echo "    Email:    $ADMIN_LOGIN_EMAIL"
echo "    Password: changeme123"
echo ""
if [ "$ADMIN_ENABLED" = "true" ]; then
    echo "  PocketBase Admin: http://localhost:8080/_/"
    echo ""
fi
echo "----------------------------------------"
echo "  Configuration"
echo "----------------------------------------"
if [ -n "$APP_URL" ]; then
    echo "  APP_URL:       $APP_URL"
fi
if [ -n "$ADMIN_EMAILS" ]; then
    echo "  ADMIN_EMAILS:  $ADMIN_EMAILS"
else
    echo "  ADMIN_EMAILS:  (any authenticated user)"
fi
echo "  TRUST_PROXY:   ${TRUST_PROXY:-true}"
echo "  ADMIN_ENABLED: ${ADMIN_ENABLED:-false}"
if [ -n "$GOOGLE_CLIENT_ID" ]; then
    echo "  OAuth:         Google enabled"
fi
if [ -n "$GITHUB_CLIENT_ID" ]; then
    echo "  OAuth:         GitHub enabled"
fi
if [ -n "$ANTHROPIC_API_KEY" ]; then
    echo "  AI Provider:   Anthropic"
elif [ -n "$OPENAI_API_KEY" ]; then
    echo "  AI Provider:   OpenAI"
elif [ -n "$OLLAMA_BASE_URL" ]; then
    echo "  AI Provider:   Ollama (${OLLAMA_MODEL:-llama3.2})"
fi
echo ""
echo "========================================"
echo ""

trap "kill $CADDY_PID $FRONTEND_PID $BACKEND_PID 2>/dev/null" EXIT
wait -n $BACKEND_PID $FRONTEND_PID $CADDY_PID
