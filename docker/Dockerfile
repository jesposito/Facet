# Me.yaml Production Dockerfile
# Single container with Caddy reverse proxy

# ============================================
# Stage 1: Build Go backend
# ============================================
FROM golang:1.23-alpine AS backend-builder

WORKDIR /build
RUN apk add --no-cache gcc musl-dev

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /me-yaml .

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
FROM alpine:3.19

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    nodejs \
    caddy

# Create non-root user
RUN addgroup -g 1000 meyaml && \
    adduser -u 1000 -G meyaml -s /bin/sh -D meyaml

WORKDIR /app

# Copy backend binary
COPY --from=backend-builder /me-yaml ./me-yaml

# Copy frontend build
COPY --from=frontend-builder /build/build ./frontend/build
COPY --from=frontend-builder /build/package.json ./frontend/
COPY --from=frontend-builder /build/node_modules ./frontend/node_modules

# Copy Caddy config and startup script
COPY docker/Caddyfile ./Caddyfile
COPY docker/start.sh ./start.sh
RUN chmod +x ./start.sh

# Create data directory
RUN mkdir -p /data && chown -R meyaml:meyaml /data /app

# Environment
ENV DATA_DIR=/data \
    NODE_ENV=production \
    POCKETBASE_URL=http://localhost:8090 \
    PORT=3000 \
    HOST=0.0.0.0 \
    ADMIN_ENABLED=false

# Single exposed port
EXPOSE 8080

# Health check via Caddy
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

USER meyaml
VOLUME ["/data"]
CMD ["./start.sh"]
