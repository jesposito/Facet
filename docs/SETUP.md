# Me.yaml Setup Guide

This guide walks you through setting up Me.yaml for the first time.

## Prerequisites

- Docker and Docker Compose
- A domain (optional, but recommended for OAuth)
- ~512MB RAM

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/yourusername/me.yaml.git
cd me.yaml
```

### 2. Configure Environment

```bash
cp .env.example .env
```

Edit `.env` and set at minimum:

```env
# Generate this with: openssl rand -hex 32
ENCRYPTION_KEY=your-32-byte-hex-key-here
```

### 3. Start the Container

```bash
docker-compose up -d
```

### 4. Create Admin Account

1. Open `http://localhost:8090/_/` in your browser
2. You'll see the PocketBase installer
3. Create your superuser account (this is your admin login)

### 5. Configure OAuth (Recommended)

For secure admin login, set up OAuth:

1. Go to `http://localhost:8090/_/` → Settings → Auth providers
2. Enable Google and/or GitHub
3. Enter your OAuth credentials (see below for how to get them)

### 6. Access Your Profile

- Public profile: `http://localhost:8080`
- Admin dashboard: `http://localhost:8080/admin`
- PocketBase admin: `http://localhost:8090/_/`

---

## Getting OAuth Credentials

### Google OAuth

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project (or select existing)
3. Go to APIs & Services → Credentials
4. Click "Create Credentials" → "OAuth 2.0 Client ID"
5. Choose "Web application"
6. Add authorized redirect URI: `https://yourdomain.com/api/oauth2-redirect`
7. Copy Client ID and Client Secret to PocketBase

### GitHub OAuth

1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. Click "New OAuth App"
3. Fill in:
   - Application name: Me.yaml
   - Homepage URL: https://yourdomain.com
   - Authorization callback URL: `https://yourdomain.com/api/oauth2-redirect`
4. Copy Client ID and Client Secret to PocketBase

---

## Reverse Proxy Setup

### Cloudflare Tunnel

1. Install cloudflared on your server
2. Create a tunnel: `cloudflared tunnel create me-yaml`
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
  - "traefik.http.routers.meyaml.rule=Host(`profile.yourdomain.com`)"
  - "traefik.http.routers.meyaml.entrypoints=websecure"
  - "traefik.http.services.meyaml.loadbalancer.server.port=3000"
```

---

## Unraid Setup

### Using Docker Template

1. Go to Docker → Add Container
2. Set:
   - Name: me-yaml
   - Repository: ghcr.io/yourusername/me-yaml:latest
   - Port Mappings: 8080 → 3000, 8090 → 8090
   - Path: /mnt/user/appdata/me-yaml → /data
   - Variable: ENCRYPTION_KEY → (your key)
   - Variable: TRUST_PROXY → true

### Using Docker Compose

1. Place files in `/mnt/user/appdata/me-yaml/`
2. Edit `.env`:
   ```env
   DATA_PATH=/mnt/user/appdata/me-yaml/data
   TRUST_PROXY=true
   APP_URL=https://profile.yourdomain.com
   ```
3. Run: `docker-compose up -d`

---

## First Steps After Setup

1. **Edit your profile**: Go to Admin → Profile
2. **Add experience**: Admin → Experience → Add
3. **Import projects**: Admin → Import → Enter GitHub repo
4. **Create views**: Admin → Views → Create view for specific audiences
5. **Generate share links**: Admin → Views → (select view) → Share Tokens

---

## Troubleshooting

### "Connection refused" errors

Check that PocketBase started:
```bash
docker-compose logs ownprofile
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
