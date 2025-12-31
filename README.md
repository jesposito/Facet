# Me.yaml

**You, human-readable.**

Own your profile. Structure your story. Share it on your terms.

A self-hosted profile and portfolio. One container, one port, one volume.

---

## What is this?

Own your professional identity. Experience, projects, skills, and story stored as data you control.

```
+--------------------------------------------------+
|                    Me.yaml                       |
+--------------------------------------------------+
|  Public Profile    |    Admin Dashboard         |
|  ----------------  |    --------------------    |
|  - Experience      |    - Edit everything       |
|  - Projects        |    - Import from GitHub    |
|  - Education       |    - Create Views          |
|  - Skills          |    - Generate share links  |
+--------------------------------------------------+
```

---

## Quick Start

```bash
git clone https://github.com/yourusername/me.yaml.git
cd me.yaml

# Generate encryption key
openssl rand -hex 32

# Configure
cp .env.example .env
# Edit .env: set ENCRYPTION_KEY

# Run
docker-compose up -d
```

Visit `http://localhost:8080`

### First Login

| Environment | Email | Password |
|-------------|-------|----------|
| Development | `admin@example.com` | `changeme123` |
| Production | Set via OAuth or `ADMIN_EMAILS` | — |

In development mode, a demo admin account is created automatically.
**Change the password immediately** if deploying beyond localhost.

OAuth (Google/GitHub) is the recommended authentication method for production.

---

## Features

### Available Now

**Public Profile**
- Experience, projects, education, skills sections
- Clean, fast, accessible pages
- Responsive design

**Views**
- Create curated versions for different audiences
- Override headline and summary per view
- Show/hide sections
- Visibility: public, unlisted, password-protected

**Share Tokens**
- Generate private links (`/s/abc123...`)
- Set expiration dates
- Limit total uses
- Revoke instantly

**GitHub Import**
- Fetch repo metadata (description, languages, topics)
- Preview before importing
- Review changes field-by-field
- Lock fields you've customized

**AI Enrichment** (optional)
- OpenAI, Anthropic, or Ollama
- Generates summaries from README
- Never auto-publishes (you review first)
- Keys encrypted at rest

**Security**
- Admin email allowlist
- HMAC-secured share tokens
- Single public port (8080)
- PocketBase admin disabled by default

### Planned

- [ ] Scheduled GitHub sync
- [ ] Me.yaml export/import
- [ ] Theme customization
- [ ] Resume PDF export

---

## Architecture

Single container with internal Caddy reverse proxy:

```
+--------------------------------------------+
|              Docker Container              |
|                                            |
|   :8080 -> Caddy -+-> /api/*  -> PocketBase|
|                   |                        |
|                   +-> /*      -> SvelteKit |
|                                            |
|   Internal only:                           |
|   - PocketBase :8090 (localhost)           |
|   - SvelteKit  :3000 (localhost)           |
|                                            |
|   +------------------------------------+   |
|   |   SQLite + Uploads (/data)         |   |
|   +------------------------------------+   |
+--------------------------------------------+
```

One port exposed. One volume to backup.

---

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ENCRYPTION_KEY` | Yes | - | 32-byte hex key (`openssl rand -hex 32`) |
| `PORT` | No | `8080` | Public port |
| `APP_URL` | No | `http://localhost:8080` | Your public URL |
| `TRUST_PROXY` | No | `false` | Set `true` behind reverse proxy |
| `ADMIN_EMAILS` | No | - | Comma-separated allowlist |
| `ADMIN_ENABLED` | No | `false` | Enable PocketBase admin at `/_/` |
| `DATA_PATH` | No | `./data` | Database and uploads |

### Unraid + Cloudflare Tunnel

```env
PORT=8080
DATA_PATH=/mnt/user/appdata/me-yaml
APP_URL=https://me.yourdomain.com
TRUST_PROXY=true
ADMIN_EMAILS=you@gmail.com
ADMIN_ENABLED=false
```

Point Cloudflare Tunnel to `http://container-ip:8080`.

---

## Development

### Codespaces (One-Click)

Open in GitHub Codespaces. Includes demo profile and admin account.

### Local

```bash
make dev
```

Development uses separate ports for hot reload:
- http://localhost:5173 (Vite dev server with HMR)
- API calls proxy to backend automatically

Production uses single port 8080 for everything.

### Default Credentials (Development Only)

| Login | Email | Password |
|-------|-------|----------|
| Me.yaml Admin | `admin@example.com` | `changeme123` |
| PocketBase Admin | `admin@localhost.dev` | `admin123` |

The PocketBase admin panel (`/_/`) is disabled by default in production.
Set `ADMIN_ENABLED=true` only for debugging.

---

## Backup

```bash
docker-compose down
tar -czvf backup.tar.gz ./data
docker-compose up -d
```

---

## License

MIT

---

*Your profile, your data, your rules.*
