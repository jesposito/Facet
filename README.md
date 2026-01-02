# Facet

**Every side of you. Your way.**

Own your profile. Structure your story. Share it on your terms.

A self-hosted, privacy-respecting personal profile platform. Think LinkedIn profile, but you own everything: the data, the hosting, the rules.

---

## Why Facet?

| Problem | Facet Solution |
|---------|------------------|
| LinkedIn owns your professional identity | **You own everything** — data lives in SQLite you control |
| One profile for all audiences | **Views** — curated versions for recruiters, clients, conferences |
| No control over who sees what | **Privacy controls** — public, unlisted, or password-protected |
| Can't import your work easily | **GitHub import** — pull in projects with one click |
| Platform tracks everything | **Zero tracking** — no analytics, no engagement metrics |

**One container. One port. One volume to backup.**

---

## Quick Links

| I want to... | Go here |
|--------------|---------|
| **Try it out** | [Quick Start](#quick-start) |
| **Learn what it does** | [Features](#features) |
| **Self-host it** | [Setup Guide](docs/SETUP.md) |
| **Contribute or develop** | [Developer Guide](#for-developers) |
| **Understand the architecture** | [Architecture](ARCHITECTURE.md) |
| **Read the design philosophy** | [Design Document](DESIGN.md) |

---

# For Users

Everything you need to run your own Facet instance.

## Quick Start

```bash
# Clone the repository
git clone https://github.com/jesposito/Facet.git
cd Facet

# Generate encryption key (required)
openssl rand -hex 32

# Configure
cp .env.example .env
# Edit .env: add your ENCRYPTION_KEY

# Run
docker-compose up -d
```

Open `http://localhost:8080` — you're live.

### First Login

| Environment | Email | Password |
|-------------|-------|----------|
| Development | `admin@example.com` | `changeme123` |
| Production | Your email in `ADMIN_EMAILS` | Your password |

**OAuth:** Google/GitHub via environment variables. See [Setup Guide](docs/SETUP.md).

---

## Features

### Your Profile, Your Way

- **Experience, projects, education, skills, certifications, awards** — all the professional sections you'd expect
- **Posts & talks** — share your writing and speaking engagements (RSS feed, iCal for talks)
- **Views & theming** — per-view section curation, overrides, accent colors, and custom CSS
- **Media library** — browse/search/delete uploads with orphan detection and responsive thumbnails
- **View membership cues** — admin lists show which views a project/post belongs to (with links to edit)
- **Exports** — JSON/YAML export, print-ready stylesheet, AI print/resume (beta)
- **Markdown everywhere** — rich content without a heavy editor

### Views: Different Faces for Different Audiences

Create tailored versions of your profile:

- **Recruiter view** — emphasize employment history and skills
- **Conference view** — highlight talks and projects
- **Consulting view** — focus on case studies and expertise
- **Personal view** — the stuff your friends care about

Each view can:
- Show/hide sections and specific items
- Override your headline and summary
- Include a custom call-to-action button
- Have different visibility settings

### Privacy Controls

| Visibility | Who can access |
|------------|----------------|
| **Public** | Anyone |
| **Unlisted** | Only people with a share link |
| **Password** | Anyone who knows the password |
| **Private** | Only you (admin) |

### Share Links

Generate private URLs for specific views:
- Set expiration dates
- Limit total uses
- Revoke instantly
- Clean URLs (token never visible in address bar)

### Import & Enrichment

**GitHub Import:**
- Fetch repository metadata, languages, and README
- Preview before importing
- Lock fields you've customized
- Refresh anytime

**AI Enrichment (optional):**
- OpenAI, Anthropic, or Ollama
- Generates summaries from READMEs
- You review everything before publishing
- Your API keys, encrypted at rest

---

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ENCRYPTION_KEY` | Yes | — | 32-byte hex key (`openssl rand -hex 32`) |
| `PORT` | No | `8080` | Public port |
| `APP_URL` | No | `http://localhost:8080` | Your public URL (for OAuth callbacks) |
| `TRUST_PROXY` | No | `false` | Set `true` behind reverse proxy |
| `ADMIN_EMAILS` | No | — | Comma-separated email allowlist for OAuth |
| `ADMIN_ENABLED` | No | `false` | Enable PocketBase admin at `/_/` |
| `DATA_PATH` | No | `./data` | Database and uploads directory |

For detailed setup instructions including OAuth, reverse proxy, and Unraid configuration, see the **[Setup Guide](docs/SETUP.md)**.

---

## Backup & Restore

Everything lives in one directory:

```bash
# Backup
docker-compose down
tar -czvf backup.tar.gz ./data
docker-compose up -d

# Restore
tar -xzvf backup.tar.gz
docker-compose up -d
```

For upgrade procedures, see **[Upgrade Guide](docs/UPGRADE.md)**.

---

## Architecture Overview

```
+--------------------------------------------+
|              Docker Container              |
|                                            |
|   :8080 -> Caddy -+-> /api/*  -> PocketBase|
|                   +-> /*      -> SvelteKit |
|                                            |
|   +------------------------------------+   |
|   |   SQLite + Uploads (/data)         |   |
+--------------------------------------------+
```

- **Single port** exposed (8080)
- **Single volume** to backup (`/data`)
- **Caddy** routes requests internally
- **PocketBase** handles API and auth
- **SvelteKit** serves the frontend

For complete technical details, see **[Architecture](ARCHITECTURE.md)**.

---

# For Developers

Everything you need to contribute to Facet.

## Development Setup

### Option 1: Codespaces (Fastest)

1. Click "Open in Codespaces" from GitHub
2. Wait for devcontainer to build (~2 min first time)
3. Services start automatically
4. Frontend: http://localhost:5173
5. API: http://localhost:8090

### Option 2: Local Development

**Prerequisites:**
- Go 1.23+
- Node.js 20+
- [Air](https://github.com/air-verse/air) for Go hot reload

```bash
# Install air
go install github.com/air-verse/air@v1.61.7

# Start everything with hot reload
make dev

# Or start services individually
make backend   # Backend with air
make frontend  # Frontend with Vite HMR
```

### Option 3: Docker Compose

```bash
make dev-docker   # Start development environment
make dev-logs     # View logs
make dev-down     # Stop
```

### Development Ports

| Service | Port | URL |
|---------|------|-----|
| Frontend (Vite) | 5173 | http://localhost:5173 |
| Backend API | 8090 | http://localhost:8090 |
| PocketBase Admin | 8090 | http://localhost:8090/_/ |

### Admin Access (Development)

Run `make seed-dev` (or `./scripts/start-dev.sh` which seeds on first run) to set your admin email/password and optional OAuth env vars. The seed prompts you; no hard-coded defaults are shipped in production.

For complete development documentation, see **[Development Guide](docs/DEV.md)**.

---

## Project Structure

```
Facet/
├── backend/                 # Go + PocketBase
│   ├── hooks/               # Custom event handlers
│   ├── services/            # Business logic
│   ├── migrations/          # Database schema
│   └── main.go              # Entry point
│
├── frontend/                # SvelteKit + TypeScript
│   ├── src/routes/          # Page routes
│   ├── src/components/      # UI components
│   └── src/lib/             # Shared utilities
│
├── docker/                  # Production Docker config
├── scripts/                 # Development scripts
└── docs/                    # Documentation
```

---

## Tech Stack

**Backend:**
- Go 1.23 — backend language
- [PocketBase](https://pocketbase.io/) v0.23.4 — Go-based backend framework
- SQLite — embedded database
- AES-256-GCM — encryption for sensitive data

**Frontend:**
- [SvelteKit](https://kit.svelte.dev/) v2.0 — full-stack web framework
- [Tailwind CSS](https://tailwindcss.com/) v3.4 — utility-first styling
- TypeScript — type safety

**Infrastructure:**
- Docker — containerization
- Caddy — internal reverse proxy
- Multi-stage builds — optimized production images

---

## Common Tasks

```bash
make dev          # Start development environment
make test         # Run all tests
make build        # Build production Docker image
make lint         # Run linters
make fmt          # Format code
make seed-dev     # Load development seed data
make dev-reset    # Clear caches and restart
```

---

## Seed Data

**For development:** Use `make seed-dev` to load a real-world test profile.

**For demos:** Use `make seed-dev` to load a rich sample profile; you’ll be prompted for admin email/password and optional OAuth envs.

---

## URL Routing

| Route | Purpose |
|-------|---------|
| `/` | Default public view |
| `/<slug>` | Named view (e.g., `/recruiter`) |
| `/s/<token>` | Share link entry point |
| `/projects/<slug>` | Project detail page |
| `/posts/<slug>` | Blog post page |
| `/admin/*` | Admin dashboard |

---

## Security Model

- **AES-256-GCM** encryption for API keys and tokens
- **HMAC-SHA256** share tokens (raw tokens never stored)
- **JWT** for password-protected views
- **Rate limiting** on sensitive endpoints
- **Deny-by-default** collection access

For complete security documentation, see **[Security Guide](docs/SECURITY.md)**.

---

# Documentation

| Document | Description |
|----------|-------------|
| **[Setup Guide](docs/SETUP.md)** | Installation, OAuth, reverse proxy, Unraid |
| **[Development Guide](docs/DEV.md)** | Local setup, project structure, troubleshooting |
| **[Security Guide](docs/SECURITY.md)** | Encryption, auth flows, rate limiting |
| **[Upgrade Guide](docs/UPGRADE.md)** | Upgrade and rollback procedures |
| **[Architecture](ARCHITECTURE.md)** | Technical system design and data model |
| **[Design Document](DESIGN.md)** | Vision, principles, and detailed specifications |
| **[Roadmap](ROADMAP.md)** | Feature development phases |

> ⚠️ **IMPORTANT FOR CONTRIBUTORS:** The [ROADMAP.md](ROADMAP.md) must be kept up-to-date with development progress. When completing features, mark them as done in the roadmap. When adding new features, add them to the appropriate phase. The roadmap is the source of truth for what is implemented vs. planned.

---

## Roadmap Highlights

**Complete:**
- Core profile and content management
- Views with custom ordering and overrides
- Share token management
- GitHub import with AI enrichment
- Print-optimized CSS for resume export
- Per-section layout presets
- Live preview in view editor
- Per-view theming with accent colors
- Data export (JSON/YAML)
- Content discovery (Posts & Talks index pages)

**In Progress:**
- AI-powered PDF resume generation

**Planned:**
- Scheduled GitHub sync
- Media library
- Additional integrations

See **[Roadmap](ROADMAP.md)** for the full development plan.

---

## License

MIT

---

*Your profile, your data, your rules.*
