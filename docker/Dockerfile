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
FROM golang:1.27.3-bookworm@sha256:972b4110ba0f47539a975c31e52c64765e1ff19f2d74469ae4a595c6676e80a5 AS backend-builder

ARG FACET_VERSION=dev
ARG BUILD_DATE=unknown
ARG GIT_COMMIT=unknown

WORKDIR /build
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    && rm -rf /var/lib/apt/lists/*

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w -X facet/buildinfo.Version=${FACET_VERSION} -X facet/buildinfo.SHA=${GIT_COMMIT} -X facet/buildinfo.BuildDate=${BUILD_DATE}" \
    -o /facet .

# ============================================
# Stage 2: Build SvelteKit frontend
# ============================================
FROM node:24-alpine@sha256:06d7adf3e0ef61a13f1d28682726366f97a15aa1888c934d3ecec86c33918b53 AS frontend-builder

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
FROM golang:1.27.3-bookworm@sha256:972b4110ba0f47539a975c31e52c64765e1ff19f2d74469ae4a595c6676e80a5 AS caddy-builder
WORKDIR /caddy-build
COPY docker/caddy/go.mod docker/caddy/go.sum ./
RUN go mod download
COPY docker/caddy/main.go ./
RUN CGO_ENABLED=0 go build -trimpath -o /caddy .

# ============================================
# Stage 3: Final runtime image
# ============================================
FROM debian:bookworm-slim@sha256:7ff6ba31527b9e336162fa061746c366c4b2032f4d2cff2078bea43f78efe3f8

RUN apt-get update && apt-get install -y --no-install-recommends \
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
COPY --from=backend-builder /facet ./facet

# Copy demo assets for demo mode
COPY --from=backend-builder /build/seeds/demo_assets ./backend/seeds/demo_assets

# Copy frontend build
COPY --from=frontend-builder /build/build ./frontend/build
COPY --from=frontend-builder /build/package.json ./frontend/
COPY --from=frontend-builder /build/node_modules ./frontend/node_modules

# Copy Caddy config and startup script
COPY docker/Caddyfile ./Caddyfile
COPY docker/start.sh ./start.sh
RUN chmod +x ./start.sh

# Create data and uploads directories (ownership set at runtime via PUID/PGID)
RUN mkdir -p /data /uploads

# Re-declare build args in final stage
ARG FACET_VERSION=dev
ARG BUILD_DATE=unknown
ARG GIT_COMMIT=unknown

# Environment - PUID/PGID follow linuxserver.io convention for Unraid compatibility
ENV DATA_DIR=/data \
    NODE_ENV=production \
    POCKETBASE_URL=http://localhost:8090 \
    PORT=3000 \
    HOST=0.0.0.0 \
    PROTOCOL_HEADER=x-forwarded-proto \
    HOST_HEADER=x-forwarded-host \
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
