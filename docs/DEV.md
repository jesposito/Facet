# Me.yaml Development Guide

This guide covers development setup for Me.yaml, a PocketBase + SvelteKit application.

## Quick Start (Codespaces)

1. Click "Open in Codespaces" from GitHub
2. Wait for the devcontainer to build (~2 min first time, ~15s thereafter)
3. Services start automatically via `postStartCommand`
4. Open the forwarded ports when prompted:
   - **Frontend**: http://localhost:5173
   - **PocketBase Admin**: http://localhost:8090/_/

**Default credentials** (dev only, auto-created on first run):

| Admin Panel | URL | Email | Password |
|-------------|-----|-------|----------|
| Me.yaml Admin | http://localhost:5173/admin | `admin@example.com` | `changeme123` |
| PocketBase Admin | http://localhost:8090/_/ | `admin@localhost.dev` | `admin123` |

If credentials don't work, reset the database: `rm -rf pb_data && ./scripts/start-dev.sh`

## Quick Start (Local)

### Prerequisites

- Go 1.23+
- Node.js 20+
- [Air](https://github.com/air-verse/air) for Go hot reload

```bash
# Install air
go install github.com/air-verse/air@v1.61.7
```

### Running Locally

```bash
# Start everything with hot reload
make dev

# Or start services individually
make backend   # Start backend with air
make frontend  # Start frontend with Vite HMR
```

### Using Docker Compose

```bash
# Start development environment
make dev-docker

# View logs
make dev-logs

# Stop
make dev-down
```

## Ports and URLs

| Service | Port | URL | Description |
|---------|------|-----|-------------|
| Frontend | 5173 | http://localhost:5173 | SvelteKit dev server with HMR |
| Backend API | 8090 | http://localhost:8090 | PocketBase API |
| PB Admin | 8090 | http://localhost:8090/_/ | PocketBase admin UI |

## Project Structure

```
Me.yaml/
├── backend/           # Go + PocketBase hooks
│   ├── hooks/         # PocketBase event hooks
│   ├── services/      # Business logic
│   ├── migrations/    # Database migrations
│   └── main.go        # Entry point
├── frontend/          # SvelteKit application
│   ├── src/
│   │   ├── routes/    # SvelteKit routes
│   │   └── lib/       # Shared components
│   └── package.json
├── scripts/           # Development scripts
│   ├── start-dev.sh   # Start all services
│   ├── dev-backend.sh # Backend with caching
│   └── dev-frontend.sh# Frontend with caching
├── docker/            # Docker configurations
├── pb_data/           # PocketBase data (gitignored)
└── docs/              # Documentation
```

## Development Workflow

### Hot Reload

Both frontend and backend support hot reload:

- **Frontend**: Vite HMR automatically refreshes on `.svelte`, `.ts`, `.css` changes
- **Backend**: Air watches `.go` files and rebuilds automatically

### Optimized Startup

The dev scripts use **lockfile hash caching** to skip unnecessary installs:

```bash
# First run: installs dependencies, saves hash
[frontend] Installing dependencies (node_modules missing)...

# Subsequent runs: skips install if lockfile unchanged
[frontend] Dependencies up to date (skipping npm install)
```

To force a fresh install:
```bash
make dev-reset  # Clears all caches
make dev        # Reinstalls everything
```

## Common Tasks

### Running Tests

```bash
make test           # All tests
make test-backend   # Go tests only
make test-frontend  # SvelteKit checks only
```

### Linting and Formatting

```bash
make lint  # Run linters
make fmt   # Format code
```

### Building for Production

```bash
make build  # Build Docker image
```

## Codespaces Networking Limitations

Some Codespaces environments have network restrictions that block access to `storage.googleapis.com`, which is used by the default Go module proxy (`proxy.golang.org`).

**Symptoms:**
- `go mod tidy` times out with DNS lookup errors for `storage.googleapis.com`
- `go build` fails with "missing go.sum entry" errors
- Downloads hang indefinitely

**Solution (already configured):**

The devcontainer is configured to use `goproxy.cn` as a fallback proxy:

```bash
# Set in devcontainer.json containerEnv
GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
GOSUMDB=off
```

If you're running outside the devcontainer and encounter these issues:

```bash
# Set environment variables
export GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
export GOSUMDB=off

# Then run your go commands
go mod tidy
go build ./...
```

**Alternative: Vendor dependencies (offline-first)**

For truly offline development, you can vendor all dependencies:

```bash
# In backend/
go mod vendor

# Build with vendored deps
go build -mod=vendor ./...
```

Note: Vendoring adds ~50MB to the repository but guarantees zero network dependencies after clone.

## Troubleshooting

### "air: command not found"

Install air globally:
```bash
go install github.com/air-verse/air@v1.61.7
```

### "go: cannot find main module"

This occurs when air runs from the wrong directory. The root `.air.toml` is configured to handle this. Ensure you're running from the project root:
```bash
cd /path/to/Me.yaml
make dev
```

### Port Already in Use

Stop any existing services:
```bash
# Find process using port
lsof -i :8090
lsof -i :5173

# Kill it
kill <PID>

# Or use make
make dev-down
```

### Slow Startup in Codespaces

If startup is slow (>30s), check:

1. **Named volumes exist**: The devcontainer uses named volumes for node_modules and Go modules
2. **Hash files are valid**: Check `frontend/node_modules/.lockfile-hash` and `backend/.gomod-hash`
3. **Force reset**: `make dev-reset && make dev`

### File Watching Not Working

In Codespaces, file watching may need polling. The devcontainer is configured with appropriate `files.watcherExclude` settings. If issues persist:

1. Check that `node_modules`, `pb_data`, and `.git` are excluded from watching
2. Try restarting the terminal

### Database Issues

Reset the database:
```bash
rm -rf pb_data
make dev  # Will recreate with seed data
```

## VS Code Tasks

The project includes VS Code tasks (`.vscode/tasks.json`):

- `Ctrl+Shift+B`: Run default build task (dev:up)
- `Ctrl+Shift+P` → "Tasks: Run Task" for all available tasks

Available tasks:
- `dev:up` - Start all services
- `dev:backend` - Start backend only
- `dev:frontend` - Start frontend only
- `dev:reset` - Clear caches
- `test` - Run all tests
- `build:docker` - Build production image

## Environment Variables

Copy `.env.example` to `.env` and customize:

```bash
cp .env.example .env
```

Key variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `ENCRYPTION_KEY` | (required in prod) | AES-256-GCM key for AI tokens |
| `SEED_DATA` | `true` in dev | Load demo data on startup |
| `DATA_DIR` | `./pb_data` | PocketBase data directory |
| `LOG_LEVEL` | `info` | Logging verbosity |

## Architecture Overview

```
┌─────────────────┐     ┌─────────────────┐
│   SvelteKit     │────▶│   PocketBase    │
│   (Frontend)    │ API │   (Backend)     │
│   Port 5173     │     │   Port 8090     │
└─────────────────┘     └─────────────────┘
                              │
                              ▼
                        ┌───────────┐
                        │  SQLite   │
                        │ (pb_data) │
                        └───────────┘
```

For detailed architecture, see [ARCHITECTURE.md](../ARCHITECTURE.md).

## Development Phases

### Phase A: Identity Polish (Complete)

*Tag: `phase-identity-polish`*

Focused exclusively on language, voice, and identity. No functional changes.

Changes:
- All page titles updated from "X | Admin" to "X | Me.yaml"
- Removed "Admin" badge from header
- Login copy: "Sign in to manage your profile"
- Footer fallback: "OwnProfile" → "Me.yaml"
- Import button: "Create Import Proposal" → "Review & Import"
- Removed "(admin only)" from visibility dropdown

### Phase B: First-Run Warmth (Planned)

Empty states, gentle guidance for new users.

### Phase C: Visual Calm (Planned)

Icon consistency, spacing, hierarchy refinements.
