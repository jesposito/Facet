FROM mcr.microsoft.com/devcontainers/go:1.22

# Pre-download Go modules for faster startup
WORKDIR /tmp/go-cache
COPY backend/go.mod ./
COPY backend/go.sum* ./
RUN go mod download || true

# Fix permissions so vscode user can use the module cache
RUN chmod -R 777 /go/pkg/mod/cache || true

# Reset working directory
WORKDIR /workspaces
