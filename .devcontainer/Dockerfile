FROM mcr.microsoft.com/devcontainers/go:1.22

# Install air for hot reloading
RUN go install github.com/air-verse/air@latest

# Pre-download Go modules for faster startup
WORKDIR /tmp/go-cache
COPY backend/go.mod ./
COPY backend/go.sum* ./
RUN go mod download || true

# Fix permissions so vscode user can use the module cache
RUN mkdir -p /go/pkg/mod/cache && chmod -R 777 /go/pkg || true

# Reset working directory
WORKDIR /workspaces
