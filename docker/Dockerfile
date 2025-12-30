# OwnProfile Production Dockerfile
# Multi-stage build for minimal final image

# ============================================
# Stage 1: Build Go backend
# ============================================
FROM golang:1.22-alpine AS backend-builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy Go modules
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy source
COPY backend/ ./

# Build
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /ownprofile .

# ============================================
# Stage 2: Build SvelteKit frontend
# ============================================
FROM node:20-alpine AS frontend-builder

WORKDIR /build

# Copy package files
COPY frontend/package*.json ./
RUN npm ci

# Copy source
COPY frontend/ ./

# Build
ENV NODE_ENV=production
RUN npm run build

# ============================================
# Stage 3: Final runtime image
# ============================================
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    nodejs \
    npm

# Create non-root user
RUN addgroup -g 1000 ownprofile && \
    adduser -u 1000 -G ownprofile -s /bin/sh -D ownprofile

WORKDIR /app

# Copy backend binary
COPY --from=backend-builder /ownprofile ./ownprofile

# Copy frontend build
COPY --from=frontend-builder /build/build ./frontend/build
COPY --from=frontend-builder /build/package.json ./frontend/
COPY --from=frontend-builder /build/node_modules ./frontend/node_modules

# Copy startup script
COPY docker/start.sh ./start.sh
RUN chmod +x ./start.sh

# Create data directory
RUN mkdir -p /data && chown -R ownprofile:ownprofile /data /app

# Environment variables
ENV DATA_DIR=/data \
    NODE_ENV=production \
    POCKETBASE_URL=http://localhost:8090 \
    PORT=3000 \
    HOST=0.0.0.0

# Expose ports
EXPOSE 8090 3000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8090/api/health || exit 1

# Switch to non-root user
USER ownprofile

# Data volume
VOLUME ["/data"]

# Start both services
CMD ["./start.sh"]
