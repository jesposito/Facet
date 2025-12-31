FROM mcr.microsoft.com/devcontainers/go:1.23

# Allow builds in environments that block access to the default Go proxies
ENV GOPROXY=direct \
    GOSUMDB=off

# Install air for hot reloading (pinned version compatible with Go 1.23)
RUN go install github.com/air-verse/air@v1.61.7

# Pre-download Go modules for faster startup
WORKDIR /tmp/go-cache
COPY backend/go.mod ./
COPY backend/go.sum* ./
RUN go mod download || true

# Fix permissions so vscode user can use the module cache
RUN mkdir -p /go/pkg/mod/cache /go/pkg/sumdb && chmod -R 777 /go/pkg || true

# Reset working directory
WORKDIR /workspaces
