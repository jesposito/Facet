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

By default, the database and uploads are stored at `./data` on the host (mapped to `/data` in the container). Set `DATA_PATH=/absolute/host/path` in your `.env` if you want them elsewhere, and ensure that path is persisted/backed up.

### 4. Access Facet

- **Public profile**: `http://localhost:8080`
- **Admin dashboard**: `http://localhost:8080/admin`

### 5. First Login

For development, a default admin account is created automatically:

| Email | Password |
|-------|----------|
| `admin@example.com` | `changeme123` |

**Security Note**: On your first login with the default password, you'll be prompted to change it immediately. The modal cannot be dismissed until you set a secure password.

**Resetting Admin Password**: If you forget your password or need to reset it to default:

```bash
# From your Docker container
docker exec -it facet /app/backend/facet reset-admin-password

# Or specify a different email
docker exec -it facet /app/backend/facet reset-admin-password admin@yourdomain.com
```

This resets the password to `changeme123` and the user will be prompted to change it on next login.

### 6. Try Demo Mode (Optional)

Not sure where to start? After logging in, toggle **Demo Mode** ON at the top of the admin panel to instantly load The Doctor's hilarious profile showcasing all features:

- 5 different views (recruiter, conference, consulting, personal, academic)
- 4 extensive blog posts with technical humor
- Projects, experience, talks, certifications, and more
- All features demonstrated: media embeds, different layouts, privacy controls

**Your data is safe:** When you toggle demo mode ON, your original data is backed up. Toggle it OFF to restore your data exactly as it was (or keep the demo data as your starting point if you prefer).

**Perfect for:**
- Exploring what a complete profile looks like
- Understanding how views and privacy controls work
- Seeing examples of well-written content
- Learning the interface before building your own profile

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

Facet works great on Unraid! You can install it via Community Applications or manually with Docker Compose.

### Method 1: Community Applications (Recommended)

1. **Install from Community Applications**
   - Open Unraid WebUI
   - Go to Apps tab
   - Search for "Facet"
   - Click Install

2. **Configure Required Settings**
   - **Port**: Leave as 8080 (or change if needed)
   - **Data Path**: Default `/mnt/user/appdata/facet` (recommended)
   - **Encryption Key**: Generate with `openssl rand -hex 32` in Unraid terminal
     - SSH into Unraid: `ssh root@tower`
     - Run: `openssl rand -hex 32`
     - Copy the output
   - **Admin Emails**: Your email address (e.g., `you@gmail.com`)
   - **App URL**:
     - Local only: `http://tower.local:8080`
     - With Cloudflare Tunnel: `https://facet.yourdomain.com`

3. **Optional: Enable OAuth Login**
   - Expand "Show more settings"
   - Enter Google or GitHub OAuth credentials (see OAuth section above)

4. **Start the Container**
   - Click "Apply"
   - Wait for container to start

5. **Access Facet**
   - Local: `http://tower:8080`
   - Or your configured domain

### Method 2: Docker Compose

1. **Prepare Directory**
   ```bash
   mkdir -p /mnt/user/appdata/facet
   cd /mnt/user/appdata/facet
   ```

2. **Create .env file**
   ```bash
   # Generate encryption key
   openssl rand -hex 32

   # Create .env file
   cat > .env << 'EOF'
   ENCRYPTION_KEY=paste-generated-key-here
   DATA_PATH=/mnt/user/appdata/facet
   APP_URL=http://tower.local:8080
   TRUST_PROXY=false
   ADMIN_EMAILS=your@email.com
   PORT=8080
   EOF
   ```

3. **Create docker-compose.yml**
   ```bash
   wget https://raw.githubusercontent.com/jesposito/Facet/main/docker-compose.yml
   ```

4. **Start Container**
   ```bash
   docker-compose up -d
   ```

5. **Check Logs**
   ```bash
   docker-compose logs -f facet
   ```

### Unraid + Cloudflare Tunnel (Secure Remote Access)

For secure remote access without opening ports:

1. **Install Cloudflare Tunnel on Unraid**
   - Install "cloudflared" from Community Applications
   - Or follow [Cloudflare's Unraid guide](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/)

2. **Create Tunnel**
   ```bash
   cloudflared tunnel create facet
   ```

3. **Configure Tunnel**
   - Point tunnel to `http://172.17.0.x:8080` (your Facet container IP)
   - Get container IP: `docker inspect facet | grep IPAddress`

4. **Update Facet Settings**
   In Unraid WebUI or .env:
   ```env
   APP_URL=https://facet.yourdomain.com
   TRUST_PROXY=true
   ```

5. **Add DNS Record**
   - In Cloudflare dashboard, add CNAME for `facet.yourdomain.com` → your tunnel

### Unraid Backup

**Method 1: CA Backup Plugin (Recommended)**
1. Install "CA Backup / Restore Appdata" from Community Applications
2. Add `/mnt/user/appdata/facet` to backup list
3. Schedule regular backups

**Method 2: Manual Backup**
```bash
# Backup
cd /mnt/user/appdata
tar -czvf facet-backup-$(date +%Y%m%d).tar.gz facet/

# Restore
tar -xzvf facet-backup-20241203.tar.gz
```

**What to back up:**
- `/mnt/user/appdata/facet/pb_data/` - Your database and PocketBase data
- `/mnt/user/appdata/facet/data/pb_data/storage` - Your uploads
- `.env` file - Your configuration

### Unraid Troubleshooting

**Container won't start:**
```bash
# Check logs
docker logs facet

# Check if port is in use
netstat -tulpn | grep 8080

# Verify encryption key is set
docker exec facet env | grep ENCRYPTION_KEY
```

**Can't access from other devices:**
- Check Unraid firewall settings
- Verify container is running: `docker ps | grep facet`
- Check container network: `docker inspect facet | grep IPAddress`

**Share tokens not working behind reverse proxy:**
- Ensure `TRUST_PROXY=true` is set
- Check reverse proxy forwards `X-Forwarded-*` headers
- Verify `APP_URL` matches your actual domain

**Database locked errors:**
- Stop container: `docker stop facet`
- Check for zombie processes: `lsof | grep data.db`
- Start container: `docker start facet`

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
