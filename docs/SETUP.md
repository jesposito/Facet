# Facet Setup Guide

This guide walks you through setting up Facet for the first time.

## Prerequisites

- Docker and Docker Compose
- A domain (optional, recommended for production)
- ~512MB RAM

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/jesposito/Facet.git
cd Facet
```

### 2. Configure Environment

```bash
cp .env.example .env
```

Edit `.env` and set at minimum:

```env
# Generate this with: openssl rand -hex 32
ENCRYPTION_KEY=your-32-byte-hex-key-here

# Your admin email (for login)
ADMIN_EMAILS=you@example.com
```

### 3. Start the Container

```bash
docker-compose up -d
```

### 4. Access Facet

- **Public profile**: `http://localhost:8080`
- **Admin dashboard**: `http://localhost:8080/admin`

### 5. First Login

For development, a default admin account is created automatically:

| Email | Password |
|-------|----------|
| `admin@example.com` | `changeme123` |

**Important**: Change this password or set up your own authentication for production.

---

## Authentication Options

### Password Login (Default)

Password authentication works out of the box. Add your email to `ADMIN_EMAILS` to authorize your account:

```env
ADMIN_EMAILS=you@example.com
```

### OAuth Login (Google/GitHub)

Configure OAuth without opening the PocketBase admin UI by setting environment variables:

```env
# Required when enabling OAuth
APP_URL=https://facet.yourdomain.com

# Google
GOOGLE_CLIENT_ID=your-client-id
GOOGLE_CLIENT_SECRET=your-client-secret

# GitHub
GITHUB_CLIENT_ID=your-client-id
GITHUB_CLIENT_SECRET=your-client-secret
```

On startup, Facet will automatically enable the providers you configure and the login page will only show available buttons.

#### Getting OAuth Credentials

**Google OAuth:**
1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project (or select existing)
3. Go to APIs & Services → Credentials
4. Click "Create Credentials" → "OAuth 2.0 Client ID"
5. Choose "Web application"
6. Add authorized redirect URI: `https://yourdomain.com/api/oauth2-redirect`
7. Save your Client ID and Client Secret

**GitHub OAuth:**
1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. Click "New OAuth App"
3. Fill in:
   - Application name: Facet
   - Homepage URL: https://yourdomain.com
   - Authorization callback URL: `https://yourdomain.com/api/oauth2-redirect`
4. Save your Client ID and Client Secret

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ENCRYPTION_KEY` | Yes | — | 32-byte hex key (`openssl rand -hex 32`) |
| `PORT` | No | `8080` | Public port |
| `APP_URL` | No | `http://localhost:8080` | Your public URL |
| `TRUST_PROXY` | No | `false` | Set `true` behind reverse proxy |
| `ADMIN_EMAILS` | No | — | Comma-separated email allowlist |
| `DATA_PATH` | No | `./data` | Database and uploads directory |

---

## Reverse Proxy Setup

### Cloudflare Tunnel

1. Install cloudflared on your server
2. Create a tunnel: `cloudflared tunnel create facet`
3. Configure tunnel to point to `localhost:8080`
4. Set in `.env`:
   ```env
   TRUST_PROXY=true
   APP_URL=https://profile.yourdomain.com
   ```

### Nginx Proxy Manager

1. Add a new proxy host
2. Domain: profile.yourdomain.com
3. Forward to: your-server-ip:8080
4. Enable SSL
5. Set in `.env`:
   ```env
   TRUST_PROXY=true
   APP_URL=https://profile.yourdomain.com
   ```

### Traefik

Add labels to `docker-compose.yml`:

```yaml
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.facet.rule=Host(`profile.yourdomain.com`)"
  - "traefik.http.routers.facet.entrypoints=websecure"
  - "traefik.http.services.facet.loadbalancer.server.port=8080"
```

---

## Unraid Setup

### Using Docker Compose

1. Place files in `/mnt/user/appdata/facet/`
2. Edit `.env`:
   ```env
   ENCRYPTION_KEY=your-key-here
   DATA_PATH=/mnt/user/appdata/facet/data
   TRUST_PROXY=true
   APP_URL=https://profile.yourdomain.com
   ADMIN_EMAILS=you@gmail.com
   ```
3. Run: `docker-compose up -d`

### Community App (Coming Soon)

An Unraid Community App template is planned for easier installation.

---

## First Steps After Setup

1. **Edit your profile**: Go to Admin → Profile
2. **Add experience**: Admin → Experience → Add
3. **Import projects**: Admin → Import → Enter GitHub repo
4. **Create views**: Admin → Views → Create view for specific audiences
5. **Generate share links**: Admin → Views → (select view) → Share Tokens

---

## Backup

Everything lives in one directory:

```bash
# Stop, backup, restart
docker-compose down
tar -czvf facet-backup-$(date +%Y%m%d).tar.gz ./data
docker-compose up -d
```

---

## Troubleshooting

### "Connection refused" errors

Check that the container started:
```bash
docker-compose logs facet
```

### OAuth redirects failing

Ensure `APP_URL` in `.env` matches your actual domain and OAuth redirect URIs.

### Uploads not persisting

Check volume permissions:
```bash
ls -la ./data/
```

The container runs as UID 1000. Ensure the data directory is writable.

### Need to reset everything

```bash
docker-compose down
rm -rf ./data/*
docker-compose up -d
```

Then set up admin account again.
