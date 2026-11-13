# Facet Production Dockerfile
# Single container with Caddy reverse proxy

# Build arguments for version info
ARG FACET_VERSION=dev
ARG BUILD_DATE=unknown
ARG GIT_COMMIT=unknown

# Base images are pinned to multi-arch index digests so a repushed tag cannot
# change build inputs silently. Refresh the tag+digest pairs together; see
# CONTRIBUTING.md for the update process.

# ============================================
# Stage 1: Build Go backend
# ============================================
FROM golang:1.27.4-bookworm@sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea AS backend-builder

ARG FACET_VERSION=dev
ARG BUILD_DATE=unknown
ARG GIT_COMMIT=unknown

WORKDIR /build
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    && rm -rf /var/lib/apt/lists/*

COPY backend/go.mod backend/go.sum ./
# PocketBase SMTP transport with bounded connection and exchange deadlines.
COPY backend/third_party/mailyak-v3 ./third_party/mailyak-v3
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w -X facet/buildinfo.Version=${FACET_VERSION} -X facet/buildinfo.SHA=${GIT_COMMIT} -X facet/buildinfo.BuildDate=${BUILD_DATE}" \
    -o /facet .

# ============================================
# Stage 2: Build SvelteKit frontend
# ============================================
FROM node:24-alpine@sha256:116b800c3754b440ee0aa3ed2f20f5bf3d99a554bfd9adeea09ea8017f6e85f5 AS frontend-builder

ARG FACET_VERSION=dev

WORKDIR /build
COPY frontend/package*.json ./
COPY frontend/vendor ./vendor
RUN npm ci

COPY frontend/ ./
COPY CHANGELOG.md ../CHANGELOG.md
ENV NODE_ENV=production
ENV FACET_VERSION=${FACET_VERSION}
RUN npm run build

# Build the proxy from pinned modules with the patched Go toolchain. The distro
# Caddy package otherwise retains the old Go runtime embedded in its binary.
FROM golang:1.27.4-bookworm@sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea AS caddy-builder
WORKDIR /caddy-build
COPY docker/caddy/go.mod docker/caddy/go.sum ./
# go.mod replaces go-jose v3 with this patched local copy.
COPY docker/caddy/third_party ./third_party
RUN go mod download
COPY docker/caddy/main.go ./
RUN CGO_ENABLED=0 go build -trimpath -o /caddy .

# ============================================
# Stage 3: Final runtime image
# ============================================
FROM debian:bookworm-slim@sha256:734a72b3d6142a71993b4bc580ca01d8ca687f77a2c422d96a961d3376b179bc

# Upgrade APT and its libapt dependency inherited from the pinned base.
RUN apt-get update && apt-get install -y --no-install-recommends \
    apt \
    ca-certificates \
    tzdata \
    curl \
    pandoc \
    texlive-latex-base \
    texlive-latex-extra \
    texlive-xetex \
    texlive-fonts-recommended \
    lmodern \
    wget \
    openssl \
    gosu \
    libheif-examples \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd -g 1000 facet \
    && useradd -u 1000 -g facet -s /bin/bash -m facet

# Install Node.js 24 from NodeSource using an explicit signed apt repo.
# Facet runs the built server with node; omit npm/npx and their bundled
# dependencies from the shipped runtime.
RUN set -eux; \
    mkdir -p /etc/apt/keyrings; \
    curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key -o /etc/apt/keyrings/nodesource.asc; \
    echo "b42e0321dabdc24e892115da705cf061167eac12a317f23d329862d0aa0a271d  /etc/apt/keyrings/nodesource.asc" | sha256sum -c -; \
    echo "deb [signed-by=/etc/apt/keyrings/nodesource.asc] https://deb.nodesource.com/node_24.x nodistro main" > /etc/apt/sources.list.d/nodesource.list; \
    apt-get update; \
    apt-get install -y nodejs; \
    rm -rf /usr/lib/node_modules/npm; \
    rm -f /usr/bin/npm /usr/bin/npx; \
    node --version; \
    test ! -e /usr/lib/node_modules/npm; \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=caddy-builder /caddy /usr/bin/caddy

# Copy backend binary
COPY --from=backend-builder --chown=root:root /facet ./facet

# Copy demo assets for demo mode
COPY --from=backend-builder --chown=root:root /build/seeds/demo_assets ./backend/seeds/demo_assets

# Copy frontend build
COPY --from=frontend-builder --chown=root:root /build/build ./frontend/build
COPY --from=frontend-builder --chown=root:root /build/package.json ./frontend/
COPY --from=frontend-builder --chown=root:root /build/node_modules ./frontend/node_modules
# Local dependency overrides are symlinks into vendor, including deepmerge.
COPY --from=frontend-builder --chown=root:root /build/vendor ./frontend/vendor
RUN node -e "require('./frontend/node_modules/deepmerge'); require('./frontend/node_modules/braces')"

# Copy Caddy config and startup script
COPY --chown=root:root docker/Caddyfile ./Caddyfile
COPY --chown=root:root docker/start.sh ./start.sh
RUN chmod +x ./start.sh

# Create data and uploads directories (ownership set at runtime via PUID/PGID)
RUN mkdir -p /data /uploads

# Re-declare build args in final stage
ARG FACET_VERSION=dev
ARG BUILD_DATE=unknown
ARG GIT_COMMIT=unknown

# The frontend believes ADDRESS_HEADER because only Caddy can reach it: it
# listens on loopback, like the backend, so other containers can't connect.
# Environment - PUID/PGID follow linuxserver.io convention for Unraid compatibility
ENV DATA_DIR=/data \
    NODE_ENV=production \
    POCKETBASE_URL=http://localhost:8090 \
    PORT=3000 \
    HOST=127.0.0.1 \
    PROTOCOL_HEADER=x-forwarded-proto \
    HOST_HEADER=x-forwarded-host \
    ADDRESS_HEADER=x-facet-client-ip \
    ADMIN_ENABLED=false \
    PUID=1000 \
    PGID=1000 \
    FACET_VERSION=${FACET_VERSION} \
    BUILD_DATE=${BUILD_DATE} \
    GIT_COMMIT=${GIT_COMMIT}

# Single exposed port
EXPOSE 8080

# Health check via Caddy and SvelteKit, with a backend probe
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

VOLUME ["/data", "/uploads"]
CMD ["./start.sh"]
