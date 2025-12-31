FROM mcr.microsoft.com/devcontainers/go:1.23

# Pre-download Go modules for faster startup
WORKDIR /tmp/go-cache
COPY backend/go.mod ./
COPY backend/go.sum* ./
RUN go mod download || true

# Reset working directory
WORKDIR /workspaces
