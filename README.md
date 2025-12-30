# me.yaml

**A self-hosted, human-readable profile and portfolio.**

> *Your profile, expressed as data. One identity. Many views. A profile you actually own.*

---

## What is this?

`me.yaml` is a self-hosted alternative to LinkedIn profiles and portfolio sites. It gives you complete ownership of your professional identity—your experience, projects, skills, and story—stored as data you control.

**Key features:**

- **Self-hosted**: One container, one profile. Your data stays on your server.
- **Multiple Views**: Create curated versions for different audiences (recruiters, clients, collaborators).
- **Share Tokens**: Generate private links for job applications without making your profile public.
- **GitHub Import**: Pull project data directly from your repos, with optional AI enrichment.
- **Beautiful Public Site**: Fast, clean, accessible profile pages that just work.
- **Simple Admin**: CRUD everything, drag to reorder, publish when ready.

```
┌─────────────────────────────────────────────────┐
│                   me.yaml                       │
├─────────────────────────────────────────────────┤
│  Public Profile    │    Admin Dashboard        │
│  ────────────────  │    ───────────────────    │
│  • Experience      │    • CRUD everything      │
│  • Projects        │    • Drag to reorder      │
│  • Education       │    • Import from GitHub   │
│  • Skills          │    • Create Views         │
│  • Writing         │    • Generate share links │
└─────────────────────────────────────────────────┘
```

---

## Quick Start

### 1. Clone and configure

```bash
git clone https://github.com/yourusername/me.yaml.git
cd me.yaml
cp .env.example .env
```

### 2. Generate an encryption key

```bash
# Generate a secure key for encrypting API tokens
openssl rand -hex 32
```

Add this to your `.env` file as `ENCRYPTION_KEY`.

### 3. Start the container

```bash
docker-compose up -d
```

### 4. Set up admin access

1. Visit `http://localhost:8090/_/` (PocketBase admin)
2. Create your superuser account
3. Configure OAuth providers (Google, GitHub) in Settings → Auth Providers
4. Visit `http://localhost:8080/admin` to log in

That's it. Start building your profile.

---

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ENCRYPTION_KEY` | Yes | - | 32-byte hex key for encrypting API tokens |
| `PORT` | No | `8080` | Port for the main web interface |
| `ADMIN_PORT` | No | `8090` | Port for PocketBase admin (can be disabled) |
| `APP_URL` | No | `http://localhost:8080` | Public URL for OAuth redirects |
| `TRUST_PROXY` | No | `false` | Set to `true` behind a reverse proxy |
| `DATA_PATH` | No | `./data` | Path to store database and uploads |
| `LOG_LEVEL` | No | `info` | Logging level: debug, info, warn, error |

### For Unraid Users

```env
DATA_PATH=/mnt/user/appdata/me-yaml
PORT=8080
TRUST_PROXY=true
APP_URL=https://profile.yourdomain.com
ENCRYPTION_KEY=<your-generated-key>
```

Works great with:
- **Cloudflare Tunnel**: Point to `localhost:8080`
- **Nginx Proxy Manager**: Standard reverse proxy setup
- **Traefik**: Add labels to docker-compose.yml

---

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                    Docker Container                       │
│                                                          │
│  ┌────────────────────┐    ┌─────────────────────────┐  │
│  │   SvelteKit        │    │    PocketBase (Go)      │  │
│  │   Frontend         │◄──►│    Backend              │  │
│  │   :3000            │    │    :8090                │  │
│  └────────────────────┘    └─────────────────────────┘  │
│                                      │                   │
│                          ┌───────────┴────────────┐     │
│                          │   SQLite + Uploads     │     │
│                          │   /data volume         │     │
│                          └────────────────────────┘     │
└──────────────────────────────────────────────────────────┘
```

**Why this stack?**

- **PocketBase**: Single binary backend with built-in auth, file storage, and admin UI
- **SQLite**: Perfect for single-user apps—no separate database container needed
- **SvelteKit**: Fast, minimal JavaScript for a snappy public profile
- **Single container**: One thing to deploy, one volume to backup

---

## Features

### Views

Create curated versions of your profile for different audiences:

- `/v/recruiter` — Highlight leadership and impact
- `/v/developer` — Focus on technical projects
- `/v/consultant` — Emphasize client work

Each view can:
- Override your headline and summary
- Show/hide sections
- Pin specific items to the top
- Have its own visibility (public, unlisted, password-protected)

### Share Tokens

Generate private links for job applications:

```
https://yoursite.com/s/abc123xyz...
```

Share tokens can:
- Expire after a set date
- Limit total uses
- Track when they're accessed
- Be revoked instantly

### GitHub Import

Import projects directly from your repositories:

1. Enter a repo URL (`owner/repo`)
2. Preview the fetched data
3. Optionally enrich with AI (OpenAI, Anthropic, Ollama)
4. Review changes field-by-field
5. Lock fields you've customized

AI enrichment generates summaries without inventing metrics—just the facts from your README.

### AI Providers (BYO Tokens)

Bring your own API keys:

- **OpenAI**: GPT-4o, GPT-4o-mini, etc.
- **Anthropic**: Claude 3 Opus, Sonnet, Haiku
- **Ollama**: Local models (Llama, Mistral, etc.)
- **Custom**: Any OpenAI-compatible endpoint

Keys are encrypted at rest. AI never auto-publishes—you review everything first.

---

## Development

### Local Development

```bash
# Start both backend and frontend with hot reload
docker-compose -f docker-compose.dev.yml up
```

- Frontend: http://localhost:5173
- Backend/API: http://localhost:8090
- PocketBase Admin: http://localhost:8090/_/

### GitHub Codespaces

This repo includes a devcontainer configuration. Open in Codespaces and ports will be forwarded automatically.

### Project Structure

```
me.yaml/
├── backend/           # Go + PocketBase
│   ├── main.go
│   ├── hooks/         # Custom API endpoints
│   ├── services/      # Business logic
│   └── migrations/    # Database schema
├── frontend/          # SvelteKit
│   ├── src/
│   │   ├── routes/    # Pages
│   │   ├── components/
│   │   └── lib/       # Utilities
│   └── static/
├── docker/            # Container configs
├── docs/              # Documentation
└── docker-compose.yml
```

---

## Backup & Restore

### Backup

Your data lives in a single directory:

```bash
# Stop the container
docker-compose down

# Backup the data directory
tar -czvf me-yaml-backup-$(date +%Y%m%d).tar.gz ./data
```

### Restore

```bash
# Stop the container
docker-compose down

# Restore from backup
tar -xzvf me-yaml-backup-YYYYMMDD.tar.gz

# Start again
docker-compose up -d
```

---

## Multiple Profiles (Family Use)

Each container = one profile. For family members:

```yaml
# docker-compose.yml
services:
  alice:
    image: ghcr.io/yourusername/me-yaml:latest
    ports:
      - "8081:3000"
    volumes:
      - ./data-alice:/data

  bob:
    image: ghcr.io/yourusername/me-yaml:latest
    ports:
      - "8082:3000"
    volumes:
      - ./data-bob:/data
```

Each instance is completely independent with its own data.

---

## Roadmap

Future ideas (not promises):

- [ ] Scheduled GitHub sync
- [ ] Theme customization
- [ ] Resume PDF export
- [ ] ActivityPub integration
- [ ] Import from LinkedIn export

---

## License

MIT License. See [LICENSE](LICENSE).

---

## Credits

Built with:
- [PocketBase](https://pocketbase.io/) — Backend framework
- [SvelteKit](https://kit.svelte.dev/) — Frontend framework
- [Tailwind CSS](https://tailwindcss.com/) — Styling

---

*Your profile, your data, your rules.*
