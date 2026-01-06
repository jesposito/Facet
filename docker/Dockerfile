# Facet Production Dockerfile
# Single container with Caddy reverse proxy

# ============================================
# Stage 1: Build Go backend
# ============================================
# go-fitz v1.24.15 ships MuPDF libs built with glibc 2.38+ (__isoc23_strtol)
# Debian Bookworm has glibc 2.36 -> linker errors on ARM64
# Solution: Use Ubuntu 24.04 (glibc 2.39) with manually installed Go 1.24
FROM ubuntu:24.04 AS backend-builder

ARG GO_VERSION=1.24.4
ARG TARGETARCH

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/* \
    && curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz" | tar -C /usr/local -xz

ENV PATH="/usr/local/go/bin:${PATH}"
WORKDIR /build

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /facet .

# ============================================
# Stage 2: Build SvelteKit frontend
# ============================================
FROM node:20-alpine AS frontend-builder

WORKDIR /build
COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
ENV NODE_ENV=production
RUN npm run build

# ============================================
# Stage 3: Final runtime image
# ============================================
FROM ubuntu:24.04

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    curl \
    caddy \
    pandoc \
    texlive \
    texlive-latex-extra \
    texlive-fonts-extra \
    wget \
    openssl \
    gosu \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd -g 1000 facet \
    && useradd -u 1000 -g facet -s /bin/bash -m facet

# Install Node.js 20 from NodeSource (Debian's nodejs is v18, incompatible with build)
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
    apt-get install -y nodejs && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy backend binary
COPY --from=backend-builder /facet ./facet

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

# Environment - PUID/PGID follow linuxserver.io convention for Unraid compatibility
ENV DATA_DIR=/data \
    NODE_ENV=production \
    POCKETBASE_URL=http://localhost:8090 \
    PORT=3000 \
    HOST=0.0.0.0 \
    ADMIN_ENABLED=false \
    PUID=1000 \
    PGID=1000

# Single exposed port
EXPOSE 8080

# Health check via Caddy
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

VOLUME ["/data", "/uploads"]
CMD ["./start.sh"]
