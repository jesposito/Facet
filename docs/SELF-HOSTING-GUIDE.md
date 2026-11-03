# Putting Facet on the internet

Once Facet works on your own network ([install guide](SETUP.md)), you'll probably want it at a proper address like `https://facet.example.com`. This page covers the three common ways to do that, plus Google and GitHub sign-in.

Facet doesn't handle HTTPS certificates itself. Its built-in Caddy listens on plain HTTP port 8080 and expects something in front of it to handle TLS: a tunnel, a reverse proxy, or a load balancer.

## Before you open the door

Check these first. Each takes a minute, and together they close the easy ways in.

- `ADMIN_EMAILS` is set to your address ([why](SECURITY.md#who-can-sign-in)).
- You've signed in and changed the first-run password from the startup log (or, on an install from an older release, replaced `changeme123`).
- Two-factor sign-in is on (**Account & Security**).
- `ADMIN_ENABLED` is `false`, so PocketBase's raw database dashboard isn't public.
- You have a backup plan for both `/data` and `/uploads` ([backups](UPGRADE.md#back-up-your-data)).

## The settings that change

Whatever method you pick, you'll end up changing the same values in `.env`:

```env
APP_URL=https://facet.example.com
TRUST_PROXY=true
TRUSTED_PROXIES=172.18.0.0/16
```

Set `TRUSTED_PROXIES` to the address range your proxy or tunnel connects from, such as its Docker network. Caddy only believes forwarded client addresses from those ranges, so visitors can't fake their address, but if it doesn't match your proxy, every visitor shares one rate-limit bucket. Never use `0.0.0.0/0`.

Then recreate the container with `docker compose up -d`. The bundled frontend trusts Caddy’s forwarded host and protocol headers; your outer proxy must preserve the public address.

It's still good practice to keep port 8080 away from the internet when a proxy sits in front. If the proxy runs on the same machine, publish the port on loopback only:

```yaml
    ports:
      - "127.0.0.1:${PORT:-8080}:8080"
```

If the proxy runs elsewhere, use your firewall to allow port 8080 only from the proxy. [Configuration](CONFIGURATION.md#trust_proxy) explains the risk.

## Option A: Cloudflare Tunnel

A tunnel makes an outbound connection from your machine to Cloudflare, so you don't open any ports on your router. You need a domain whose DNS is managed by Cloudflare.

1. Install `cloudflared` using [Cloudflare's instructions](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/). On Unraid, there's a Cloudflared container in Community Applications.
2. Sign in and create a tunnel:
   ```bash
   cloudflared tunnel login
   cloudflared tunnel create facet
   ```
3. Point the tunnel at Facet. In `~/.cloudflared/config.yml`:
   ```yaml
   tunnel: facet
   credentials-file: /home/YOUR_USER/.cloudflared/TUNNEL_ID.json

   ingress:
     - hostname: facet.example.com
       service: http://localhost:8080
     - service: http_status:404
   ```
   If `cloudflared` runs in its own container, use Facet's container name or IP instead of `localhost`, for example `http://facet:8080` on a shared Docker network.
4. Create the DNS record and start the tunnel:
   ```bash
   cloudflared tunnel route dns facet facet.example.com
   cloudflared tunnel run facet
   ```
5. Set `APP_URL=https://facet.example.com` and `TRUST_PROXY=true`, then `docker compose up -d`.

Cloudflare adds a `CF-Connecting-IP` header, which Facet uses for per-visitor rate limits.

## Option B: a reverse proxy you run

Use this if you already run Nginx Proxy Manager, Caddy, Traefik or nginx for other services. Point your domain's DNS at the proxy, let it obtain a certificate, and forward everything to Facet's port 8080.

Two things trip people up:

- **Upload size.** Facet accepts request bodies up to 25 MB ([Caddyfile](../docker/Caddyfile)). Many proxies default to 1 MB, which makes image and résumé uploads fail with a 413 error. Raise your proxy's limit to at least 25 MB.
- **Forwarded headers.** Pass the original host and scheme (`X-Forwarded-Host`, `X-Forwarded-Proto`) and the client IP (`X-Real-IP` or `X-Forwarded-For`). Most proxies do this by default.

### Nginx Proxy Manager

Add a proxy host for your domain with scheme `http`, the Facet host or container name, and port `8080`. On the SSL tab, request a certificate and turn on **Force SSL**. On the Advanced tab, add:

```nginx
client_max_body_size 25m;
```

### Caddy

```caddy
facet.example.com {
    reverse_proxy localhost:8080
}
```

Caddy gets a certificate automatically and sends the forwarded headers. Its default body size is unlimited.

### Traefik

Add labels to the Facet service in your Compose file, adjusted for your entrypoint and certificate resolver names:

```yaml
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.facet.rule=Host(`facet.example.com`)"
      - "traefik.http.routers.facet.entrypoints=websecure"
      - "traefik.http.routers.facet.tls.certresolver=letsencrypt"
      - "traefik.http.services.facet.loadbalancer.server.port=8080"
```

### nginx

```nginx
server {
    listen 443 ssl;
    server_name facet.example.com;
    # ssl_certificate and ssl_certificate_key lines go here

    client_max_body_size 25m;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

## Option C: stay on your network

If Facet only needs to work at home, skip all of this. Set `APP_URL` to the address you use, such as `http://192.168.1.20:8080` or `http://tower.local:8080`, and leave `TRUST_PROXY=false`.

Without HTTPS, passwords travel unencrypted across your network, and Facet logs a reminder about that at startup. On a home network that's usually an accepted trade-off. Don't forward port 8080 from your router to the internet without HTTPS in front.

## A note on HSTS

Facet's Caddy sends `Strict-Transport-Security: max-age=31536000; includeSubDomains` on every response ([Caddyfile](../docker/Caddyfile)). Browsers ignore this header on plain HTTP, so it doesn't affect a LAN install. Once a browser has visited Facet over HTTPS at `facet.example.com`, it will insist on HTTPS for that hostname **and every subdomain of it** for a year. That's normally what you want. Keep it in mind if you serve Facet from a bare domain whose subdomains still need plain HTTP.

## Sign in with Google or GitHub

OAuth is optional. Password sign-in keeps working alongside it.

1. Make sure `APP_URL` is your final public address. The redirect URL is built from it.
2. Create the OAuth app:
   - **Google:** in the [Google Cloud Console](https://console.cloud.google.com/), go to **APIs & Services → Credentials → Create credentials → OAuth client ID**, choose **Web application**, and add the authorised redirect URI `https://facet.example.com/api/oauth2-redirect`.
   - **GitHub:** in [Developer settings → OAuth Apps](https://github.com/settings/developers), create a new app with homepage `https://facet.example.com` and callback URL `https://facet.example.com/api/oauth2-redirect`.
3. Add the ID and secret to `.env`:
   ```env
   GOOGLE_CLIENT_ID=...
   GOOGLE_CLIENT_SECRET=...
   GITHUB_CLIENT_ID=...
   GITHUB_CLIENT_SECRET=...
   ```
   If you use the repository's `docker-compose.yml`, also add these four names to its `environment:` list. That file doesn't pass them through by default ([why](CONFIGURATION.md#does-the-variable-reach-the-container)).
4. Recreate the container (`docker compose up -d`). The log should include `OAuth env config: providers enabled`.
5. Make sure the email on your Google or GitHub account is listed in `ADMIN_EMAILS`. OAuth sign-in is refused for any other address, and refused entirely when `ADMIN_EMAILS` is empty and an account already exists ([source](../backend/hooks/auth.go)).

## Check that it worked

From another network (your phone on mobile data is perfect):

- `https://facet.example.com/` shows your public page.
- `https://facet.example.com/health` returns JSON with `"ok": true`.
- `https://facet.example.com/_/` returns 404 (unless you've deliberately set `ADMIN_ENABLED=true`).
- Signing in at `/admin` works, and share links you create start with your real domain.

If something's off, [Troubleshooting](TROUBLESHOOTING.md) covers the usual causes.
