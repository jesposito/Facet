# Facet Development Container
# https://containers.dev/implementors/json_reference/
#
# This Dockerfile is optimized for fast startup:
# - Go modules are pre-downloaded during build (cached layer)
# - Air is pre-installed
# - Runtime dependency installation is handled by scripts with hash caching

FROM mcr.microsoft.com/devcontainers/go:1.23

# Build-time only: use goproxy.cn as fallback for restricted environments
# (proxy.golang.org redirects to storage.googleapis.com which may be blocked)
ARG GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
# Note: GOSUMDB=off was removed to allow gopls installation at runtime

# Install air for Go hot reloading
# Ref: https://github.com/air-verse/air/releases
RUN go install github.com/air-verse/air@v1.61.7

# Install Pandoc and LaTeX for AI Print feature (PDF/DOCX generation)
RUN apt-get update && apt-get install -y --no-install-recommends \
    pandoc \
    texlive-latex-base \
    texlive-fonts-recommended \
    texlive-latex-extra \
    lmodern \
    && rm -rf /var/lib/apt/lists/*

# Pre-download Go modules for faster first startup
# This layer is cached and only rebuilt when go.mod/go.sum change
WORKDIR /tmp/go-cache
COPY backend/go.mod backend/go.sum* ./
RUN go mod download 2>/dev/null || true

# Fix permissions for module cache (vscode user runs as non-root)
RUN mkdir -p /go/pkg/mod/cache /go/pkg/sumdb \
    && chmod -R 777 /go/pkg || true

# Reset working directory
WORKDIR /workspaces

# Note: Node.js is installed via devcontainer features, not here
# Note: npm dependencies are installed by scripts/dev-frontend.sh with hash caching
