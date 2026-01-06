# Self-Hosting Guide for Beginners

New to self-hosting? This guide will walk you through getting Facet up and running, even if you've never self-hosted anything before.

**Time required:** 15-30 minutes depending on your setup.

---

## What is Self-Hosting?

Self-hosting means running applications on your own hardware instead of using someone else's servers. When you self-host Facet:

- **Your data stays on your machine** - no third parties have access
- **No monthly fees** - just your electricity and internet
- **Full control** - customize anything, export anytime, no vendor lock-in
- **Privacy by default** - no tracking, no analytics unless you want them

---

## What You'll Need

### Hardware (pick one)

| Option | Cost | Difficulty | Notes |
|--------|------|------------|-------|
| **Old laptop or PC** | Free (reuse) | Easy | Just needs to stay on |
| **Raspberry Pi 4/5** | ~$60-100 | Easy | Low power, quiet, great starter |
| **Unraid/Synology NAS** | $200-500+ | Easiest | Built-in Docker, app stores |
| **Cloud VPS** | $5-10/month | Medium | DigitalOcean, Linode, Hetzner |

### Software

- **Docker** - containers make installation easy (pre-installed on most NAS systems)
- **A terminal** - for running commands (or use your NAS's web UI)

### Optional (but recommended)

- **Domain name** - ~$10/year (Namecheap, Cloudflare, Porkbun)
- **Cloudflare account** - free, provides secure remote access

---

## Choose Your Path

### Path A: Unraid or Synology (Easiest - 5 minutes)

If you have an Unraid or Synology NAS, Facet is in the app store:

**Unraid:**
1. Open the Unraid WebUI
2. Go to **Apps** tab
3. Search for "**Facet**"
4. Click **Install**
5. Fill in your email for `ADMIN_EMAILS`
6. Click **Apply**

**Synology:**
1. Open **Container Manager** (or Docker package)
2. Search Docker Hub for `ghcr.io/jesposito/facet`
3. Download and configure with port 8080 and a `/data` volume

That's it! Access Facet at `http://your-nas-ip:8080`.

---

### Path B: Any Computer with Docker (10 minutes)

Works on Linux, Mac, Windows, Raspberry Pi, or any cloud VPS.

**Step 1: Install Docker**

If you don't have Docker:
- **Linux:** `curl -fsSL https://get.docker.com | sh`
- **Mac/Windows:** Download [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- **Raspberry Pi:** `curl -fsSL https://get.docker.com | sh`

**Step 2: Create a folder for Facet**

```bash
mkdir ~/facet && cd ~/facet
```

**Step 3: Create your configuration**

```bash
# Generate an encryption key (required)
openssl rand -hex 32
```

Copy that key, then create a `.env` file:

```bash
cat > .env << 'EOF'
# Paste your generated key here
ENCRYPTION_KEY=paste-your-key-here

# Your email (for admin login)
ADMIN_EMAILS=you@example.com

# Leave these as-is for now
TRUST_PROXY=false
EOF
```

**Step 4: Download and run Facet**

```bash
# Download the docker-compose file
curl -O https://raw.githubusercontent.com/jesposito/Facet/main/docker-compose.yml

# Start Facet
docker compose up -d
```

**Step 5: Open Facet**

Go to `http://localhost:8080` (or your server's IP address).

Default login: `admin@example.com` / `changeme123`

You'll be prompted to change the password on first login.

---

## Making Facet Accessible From Anywhere

Right now, Facet only works on your local network. To access it from anywhere (phone, work, coffee shop), you need to expose it to the internet securely.

### Option 1: Cloudflare Tunnel (Recommended - Free & Secure)

Cloudflare Tunnels let you expose Facet without opening any ports on your router. No port forwarding, no firewall changes, free SSL certificate.

**What you need:**
- A domain name (point it to Cloudflare's nameservers)
- A free Cloudflare account

**Step 1: Install cloudflared**

```bash
# Linux/Mac
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 -o cloudflared
chmod +x cloudflared
sudo mv cloudflared /usr/local/bin/

# Or on Unraid: install the "Cloudflared" container from Community Apps
```

**Step 2: Authenticate with Cloudflare**

```bash
cloudflared tunnel login
```

This opens a browser window. Select your domain.

**Step 3: Create a tunnel**

```bash
cloudflared tunnel create facet
```

**Step 4: Configure the tunnel**

Create `~/.cloudflared/config.yml`:

```yaml
tunnel: facet
credentials-file: /home/YOUR_USER/.cloudflared/TUNNEL_ID.json

ingress:
  - hostname: facet.yourdomain.com
    service: http://localhost:8080
  - service: http_status:404
```

**Step 5: Add DNS record**

```bash
cloudflared tunnel route dns facet facet.yourdomain.com
```

**Step 6: Start the tunnel**

```bash
cloudflared tunnel run facet
```

**Step 7: Update Facet config**

Edit your `.env` file:

```bash
APP_URL=https://facet.yourdomain.com
TRUST_PROXY=true
```

Restart Facet:

```bash
docker compose down && docker compose up -d
```

Now visit `https://facet.yourdomain.com` from anywhere!

---

### Option 2: Nginx Proxy Manager (More Control)

NPM gives you a nice web UI for managing reverse proxies and SSL certificates. Good if you're running multiple self-hosted apps.

**Step 1: Install Nginx Proxy Manager**

Add this to a new `docker-compose.yml` (or add to your existing stack):

```yaml
services:
  npm:
    image: 'jc21/nginx-proxy-manager:latest'
    restart: unless-stopped
    ports:
      - '80:80'
      - '443:443'
      - '81:81'  # Admin UI
    volumes:
      - ./npm-data:/data
      - ./npm-letsencrypt:/etc/letsencrypt
```

```bash
docker compose up -d
```

Access NPM admin at `http://your-server-ip:81`

Default login: `admin@example.com` / `changeme`

**Step 2: Point your domain to your server**

In your domain's DNS settings, add an A record:
- **Name:** `facet` (or whatever subdomain you want)
- **Value:** Your server's public IP address

**Step 3: Add a Proxy Host in NPM**

1. Click **Hosts** → **Proxy Hosts** → **Add Proxy Host**
2. **Domain Names:** `facet.yourdomain.com`
3. **Scheme:** `http`
4. **Forward Hostname:** `host.docker.internal` (or your server's local IP)
5. **Forward Port:** `8080`
6. **SSL tab:** Request a new SSL certificate, enable Force SSL

**Step 4: Update Facet config**

Edit your `.env`:

```bash
APP_URL=https://facet.yourdomain.com
TRUST_PROXY=true
```

Restart Facet:

```bash
docker compose down && docker compose up -d
```

---

### Option 3: Local Network Only

If you only need Facet at home, no extra setup needed. Just use:

- `http://your-server-ip:8080`
- Or `http://your-hostname.local:8080`

This won't work outside your home network, but it's the simplest option.

---

## You Did It!

You're now self-hosting your own profile platform. Here's what to do next:

1. **Log in** at `/admin` with your credentials
2. **Try Demo Mode** - toggle it on to see a full example profile
3. **Build your profile** - add experience, projects, skills
4. **Create views** - different versions for different audiences
5. **Generate share links** - give recruiters access without making everything public

---

## Common Questions

### Is this secure?

Yes. Facet uses:
- AES-256-GCM encryption for sensitive data
- Bcrypt password hashing
- No tracking or analytics by default
- All traffic over HTTPS (if you set up Cloudflare or NPM)

See [SECURITY.md](SECURITY.md) for the full security model.

### What if I break something?

Your data is in one folder (`./data`). To backup:

```bash
tar -czvf facet-backup.tar.gz ./data
```

To restore, just extract that tarball. That's it.

### How do I update Facet?

```bash
docker compose pull
docker compose down && docker compose up -d
```

### What does this cost?

- **Facet:** Free (MIT licensed)
- **Hardware:** Whatever you already have, or ~$5/month for a VPS
- **Domain:** ~$10/year (optional)
- **Cloudflare:** Free tier is plenty
- **SSL:** Free via Cloudflare or Let's Encrypt

Most people self-host for $0-15/month total.

### Where do I get help?

- [GitHub Issues](https://github.com/jesposito/Facet/issues) - bug reports and feature requests
- [README](../README.md) - full documentation
- [Setup Guide](SETUP.md) - detailed configuration options

---

## Next Steps

- [Full Setup Guide](SETUP.md) - OAuth, advanced configuration
- [Developer Guide](DEV.md) - contribute to Facet
- [Security Documentation](SECURITY.md) - understand the security model
