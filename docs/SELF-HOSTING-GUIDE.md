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

Create a `.env` file:

```bash
cat > .env << 'EOF'
# Your email (for admin login)
ADMIN_EMAILS=you@example.com

# Leave these as-is for now
TRUST_PROXY=false

# Optional: Set your own encryption key. If not set, one is auto-generated
# and saved to /data/.encryption_key. Make sure /data is in your backups.
# ENCRYPTION_KEY=your-key-here
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

## Going Further: Power-User Features

Once you have Facet running, these features unlock automation, monitoring, and integration with the rest of your stack. None of them require an account anywhere — everything runs on your own box.

### API Keys (Build Things Around Your Profile)

Want a static site to mirror your latest blog posts? A cron job that pipes new GitHub repos into your portfolio? A small script that publishes drafts from your terminal? The public API at `/api/v1/*` is built for that.

**Create a key:**

1. Log in to `/admin`
2. Go to **API** in the sidebar
3. Click **New API Key**
4. Give it a label (e.g. "Static site mirror"), pick scopes, optionally set an expiry
5. Copy the `facet_...` key — it is shown **once** and never again

**Read your posts from a script:**

```bash
curl https://facet.example.com/api/v1/posts \
  -H "Authorization: Bearer facet_a1b2c3..."
```

**Publish a draft from a script:**

```bash
curl -X POST https://facet.example.com/api/v1/posts \
  -H "Authorization: Bearer facet_a1b2c3..." \
  -H "Content-Type: application/json" \
  -d '{"title":"Hello","slug":"hello","content":"Markdown body","is_draft":true}'
```

**Available scopes:**

| Scope | Lets the key do this |
|-------|----------------------|
| `read:profile` | `GET /api/v1/profile` |
| `read:posts` | `GET /api/v1/posts` |
| `read:projects` | `GET /api/v1/projects` |
| `read:skills` | `GET /api/v1/skills` |
| `read:experience` | `GET /api/v1/experience` |
| `write:profile` | `PATCH /api/v1/profile` |
| `write:posts` | `POST/PATCH/DELETE /api/v1/posts[/{id}]` |
| `write:projects` | `POST/PATCH/DELETE /api/v1/projects[/{id}]` |
| `write:skills` | `POST/PATCH/DELETE /api/v1/skills[/{id}]` |
| `write:experience` | `POST/PATCH/DELETE /api/v1/experience[/{id}]` |

Read and write are **independent** — granting `write:posts` does not imply `read:posts`. Request both if your tool needs to read after writing.

Keys can be revoked from the same page; revocation is a soft flag so the audit trail survives.

### Webhooks (Get Notified When Things Happen)

Webhooks let Facet POST a JSON envelope to a URL of your choice when something happens. Use them to fan content out to Slack/Discord, trigger a deploy when you publish a post, or pipe new comments into your moderation queue.

**Available events:**

| Event | Fired when |
|-------|------------|
| `post.published` | A post / talk / custom content row transitions out of draft |
| `comment.created` | A new comment is created on a public item |
| `newsletter.subscribed` | A new subscriber confirms (status becomes `active`) |

**Add a webhook:**

1. Go to **Admin → Settings → Webhooks**
2. Click **New Webhook**
3. Give it a label, paste the receiver URL, pick events
4. Copy the auto-generated `secret` — shown **once**, used to verify signatures

**Payload shape:**

```json
{
  "event": "post.published",
  "timestamp": "2026-05-16T12:00:00Z",
  "data": {
    "id": "abc123",
    "collection": "posts",
    "title": "Hello",
    "slug": "hello"
  }
}
```

**Verifying signatures (Node.js):**

```js
import crypto from 'node:crypto';
const expected = 'sha256=' + crypto
  .createHmac('sha256', WEBHOOK_SECRET)
  .update(rawBody)
  .digest('hex');
// Compare in constant time
const ok = crypto.timingSafeEqual(
  Buffer.from(expected),
  Buffer.from(req.headers['x-facet-signature']),
);
```

**What you can trust:**

- The receiver URL is validated **at dial time**: hostnames are resolved and rejected if they point at private / loopback / link-local / CGNAT / cloud-metadata IPs. You cannot accidentally aim a webhook at `localhost` or `169.254.169.254`.
- Delivery retries with exponential backoff (1m, 5m, 30m, 2h, 12h — 6 total attempts).
- After 10 consecutive failures the endpoint is auto-disabled to stop piling up dead deliveries. Toggle it back on in the UI to reset the counter.
- Every attempt is logged in **Deliveries** with status code, response body, and attempt count.

**Testing without waiting for a real event:**

- **Send test** on a webhook fires a synthetic delivery and shows the response inline.
- The **dispatch test event** picker (above the webhooks list) fires a realistic synthetic payload for any registered event to every active subscriber so you can validate your receiver before going live.

### System Alerts (Know When Something Breaks)

Background failures used to be invisible until you went looking for them. Now they land at `/admin/alerts`. The sidebar link is hidden today because no emitters are wired yet — the page works, the inbox just stays empty until backup / SMTP / webhook code paths start calling `CreateSystemAlert`.

What shows up here:

- Failed backups, SMTP errors, repeated webhook delivery failures
- Security warnings (e.g., login lockouts)
- Anything backend code emits via `CreateSystemAlert`

Three severities: `info` (FYI), `warning` (look soon), `critical` (look now). You can filter by severity or by acknowledged / unacknowledged, ack alerts one at a time or all at once, and delete alerts you no longer want around.

The alert subsystem is intentionally lenient — if writing an alert fails, the originating operation never breaks. Better to lose a notification than to lose a backup.

### Newsletter Lists (Run More Than One List)

A fresh install seeds a single "Newsletter" list so the existing subscribe flow keeps working. When one list isn't enough — say, separate "Engineering posts" and "Talks I'm giving" lists — head to **Admin → Newsletter → Lists** and create as many as you want.

**Per-list settings:**

- Name, slug, description
- Custom sender name and reply-to address
- Custom welcome email subject + HTML body
- Active toggle (pauses new subscribes without deleting members)
- "Default" flag (exactly one list is default; the swap is atomic)

**Sending a broadcast:**

1. **Admin → Newsletter → Compose**
2. Pick which lists to send to
3. Compose subject + body, preview the rendered email
4. Send a test to yourself
5. Dispatch the broadcast

Self-hosted has no list cap (`cap: 0` in the API response means unlimited). Subscriber counts on each list are kept in sync automatically by hooks on the underlying membership join table.

### AI Features Are Bring-Your-Own-Key

All AI features in Facet — resume parsing, the writing assistant, GitHub project enrichment, AI Print resume generation — use **your own** API credentials. Nothing is metered, billed, or rate-limited by a Facet service. Configure providers under **Admin → Settings → Integrations**.

Supported providers: OpenAI, Anthropic, and any OpenAI-compatible endpoint (including a local Ollama instance, so you can keep everything on-prem). Provider keys are encrypted at rest with AES-256-GCM.

If you skip this step entirely, the AI panels just don't appear in the UI. Facet works fully without AI.

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
