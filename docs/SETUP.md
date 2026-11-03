# Install Facet and sign in

This page gets one Facet instance running and gets you into the admin dashboard. It covers the published Docker image (the usual choice), building from source, and Unraid. Once it works on your own network, [Putting Facet on the internet](SELF-HOSTING-GUIDE.md) covers domains, HTTPS and Google/GitHub sign-in.

## What you need

- A machine that can run Docker: a Linux server, NAS, Raspberry Pi (64-bit OS), Mac or Windows with Docker Desktop. Published images are built for `linux/amd64` and `linux/arm64` ([workflow](../.github/workflows/docker-publish.yml)).
- Docker with the Compose plugin (`docker compose`). The older `docker-compose` command also works if that's what you have.
- Some disk space. The image is fairly large because it bundles Pandoc and LaTeX for AI resume PDFs ([Dockerfile](../docker/Dockerfile)). Your own data starts small and grows with uploads.

You don't need Go, Node.js or a database server. Everything runs inside one container.

## What runs inside the container

One container runs three processes ([start script](../docker/start.sh)):

- **Caddy** listens on port `8080` and is the only thing exposed.
- **SvelteKit** (Node.js) renders pages on internal port `3000`.
- **PocketBase** (the Go backend, with SQLite) serves the API on internal port `8090`.

Two folders hold everything you care about:

| Container path | What's in it | Why it matters |
|----------------|--------------|----------------|
| `/data` | SQLite database, settings, built-in backup archives, and `.encryption_key` if Facet generated one | Small but critical. Losing it loses your site. |
| `/uploads` | Uploaded images, documents and generated files | Can grow large. Back it up alongside `/data`. |

## Option 1: published image with Docker Compose (recommended)

The `docker-compose.yml` at the root of this repository builds the image from source. To use the published image instead, create a folder with two files.

```bash
mkdir -p ~/facet && cd ~/facet
```

**`docker-compose.yml`**

```yaml
services:
  facet:
    image: ghcr.io/jesposito/facet:3
    container_name: facet
    restart: unless-stopped
    ports:
      - "${PORT:-8080}:8080"
    volumes:
      - ${DATA_PATH:-./data}:/data
      - ${UPLOADS_PATH:-./uploads}:/uploads
    environment:
      # Core
      ENCRYPTION_KEY: ${ENCRYPTION_KEY:-}
      APP_URL: ${APP_URL:-http://localhost:8080}
      ADMIN_EMAILS: ${ADMIN_EMAILS:-}
      TRUST_PROXY: ${TRUST_PROXY:-false}
      TRUSTED_PROXIES: ${TRUSTED_PROXIES:-127.0.0.1/32}
      ADMIN_ENABLED: ${ADMIN_ENABLED:-false}
      # Optional sign-in providers
      GOOGLE_CLIENT_ID: ${GOOGLE_CLIENT_ID:-}
      GOOGLE_CLIENT_SECRET: ${GOOGLE_CLIENT_SECRET:-}
      GITHUB_CLIENT_ID: ${GITHUB_CLIENT_ID:-}
      GITHUB_CLIENT_SECRET: ${GITHUB_CLIENT_SECRET:-}
      # Optional email
      SMTP_HOST: ${SMTP_HOST:-}
      SMTP_PORT: ${SMTP_PORT:-}
      SMTP_USERNAME: ${SMTP_USERNAME:-}
      SMTP_PASSWORD: ${SMTP_PASSWORD:-}
      SMTP_TLS: ${SMTP_TLS:-}
      SMTP_AUTH_METHOD: ${SMTP_AUTH_METHOD:-}
      SMTP_SENDER_NAME: ${SMTP_SENDER_NAME:-}
      SMTP_SENDER_ADDRESS: ${SMTP_SENDER_ADDRESS:-}
      # Optional AI provider (or add one in the admin UI instead)
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY:-}
      OPENAI_API_KEY: ${OPENAI_API_KEY:-}
      OLLAMA_BASE_URL: ${OLLAMA_BASE_URL:-}
      OLLAMA_MODEL: ${OLLAMA_MODEL:-}
      # Optional built-in backup schedule
      BACKUP_HOUR: ${BACKUP_HOUR:-}
      BACKUP_MAX_COUNT: ${BACKUP_MAX_COUNT:-}
```

The environment block lists each variable on purpose. Avoid `env_file: .env` here: `.env` contains `PORT`, and the container already uses `PORT` internally for the SvelteKit server, so passing yours through would make it collide with Caddy. Every variable is explained in the [configuration reference](CONFIGURATION.md).

**`.env`** (in the same folder)

```env
# The address you'll type into your browser, with no trailing slash.
APP_URL=http://localhost:8080

# Your email. This becomes the first account and the only one allowed to sign in.
ADMIN_EMAILS=you@example.com

# Where your data lives on the host.
DATA_PATH=./data
UPLOADS_PATH=./uploads
```

Then start it:

```bash
docker compose up -d
docker compose logs -f facet
```

The log prints a banner with the version, the storage location and the default login. When you see `Facet is running!`, press Ctrl+C to stop following the log (the container keeps running) and open `http://localhost:8080`.

The `:3` tag follows the latest 3.x release. You can pin an exact release (for example `:3.6.0`) or use `:latest`. [Upgrades](UPGRADE.md#choosing-an-image-tag) explains the trade-off.

## Option 2: a single `docker run`

For a quick look on your own computer:

```bash
docker run -d --name facet -p 8080:8080 \
  -v facet-data:/data \
  -v facet-uploads:/uploads \
  -e APP_URL=http://localhost:8080 \
  -e ADMIN_EMAILS=you@example.com \
  ghcr.io/jesposito/facet:3
```

Both folders are named Docker volumes, so they survive `docker rm`. Don't add `--rm` and don't leave out the `/uploads` mount. The image declares `/uploads` as a volume, so without a named mount Docker creates an anonymous one, and anonymous volumes are easy to lose.

## Option 3: build from source with the repository's Compose file

Use this if you want to run unreleased code or change the source.

```bash
git clone https://github.com/jesposito/Facet.git
cd Facet
cp .env.example .env
# edit .env: set APP_URL and ADMIN_EMAILS at minimum
docker compose up -d --build
```

`make prod` runs `docker compose up -d` and `make prod-down` stops it ([Makefile](../Makefile)).

The repository's [`docker-compose.yml`](../docker-compose.yml) passes through most variables in `.env.example`, including OAuth, SMTP, AI, Stripe, backup and proxy settings. It doesn't pass `FACET_FEATURE_COURSES` or `FACET_MANAGED`; add them to its `environment:` list if you need them ([Configuration](CONFIGURATION.md#does-the-variable-reach-the-container)).

## Option 4: Unraid

Facet is listed in Unraid Community Applications. Open **Apps**, search for **Facet**, and install it. The template is maintained in [jesposito/unraid_templates](https://github.com/jesposito/unraid_templates); see [unraid/README.md](../unraid/README.md) for the direct template URL.

When you fill in the template:

- Map `/data` to `/mnt/user/appdata/facet` and `/uploads` to wherever you keep larger files, for example `/mnt/user/appdata/facet/uploads`.
- Set **ADMIN_EMAILS** to your email address.
- Set **APP_URL** to the address you'll use, such as `http://tower.local:8080` on your network or `https://facet.example.com` behind a tunnel.
- Set **TRUST_PROXY** to `true` only if traffic reaches Facet through a reverse proxy or tunnel. See [Configuration](CONFIGURATION.md#trust_proxy).

The container honours `PUID` and `PGID` (both default to `1000`) so files in your appdata share are owned by the user you choose. To include Facet in Unraid's appdata backups, make sure both mapped folders are covered. [Backups](UPGRADE.md#back-up-your-data) explains what each folder contains.

## First login

1. Open `/admin` on the address you set as `APP_URL`, for example `http://localhost:8080/admin`.
2. On the first start with an empty database, Facet writes the email and random password to `/data/.admin-credentials` with mode `0600`. The startup log prints the exact file path, never the password. Read the file as the container's `facet` user:
   ```bash
   docker exec -u facet facet cat /data/.admin-credentials
   ```
3. Sign in with:
   - **Email:** the first address in `ADMIN_EMAILS`. If you left `ADMIN_EMAILS` empty, it's `admin@example.com`.
   - **Password:** the one from the credentials file
4. Change the password under **Settings**, then delete the private handoff file:
   ```bash
   docker exec -u facet facet rm /data/.admin-credentials
   ```

If you lost the credentials file, run `reset-admin-password` to create a new one ([Locked out](UPGRADE.md#locked-out)).

The account is created only once, the first time Facet starts with an empty database ([seed code](../backend/hooks/seed.go)). Changing `ADMIN_EMAILS` later doesn't rename it, and the admin dashboard has no email-change screen. If you edit `ADMIN_EMAILS`, keep the account's original address in the comma-separated list, or you'll lock yourself out of password sign-in. [Recovery](UPGRADE.md#locked-out) covers what to do if that happens.

> **Set `ADMIN_EMAILS` before anyone else can reach Facet.** When it's empty, every account in Facet's `users` table can sign in with a password and has full control. [Security](SECURITY.md#who-can-sign-in) explains why.

## The encryption key

Facet encrypts AI provider keys, the GitHub token, Stripe secrets and two-factor secrets with a key you never have to type ([start script](../docker/start.sh)):

1. If `ENCRYPTION_KEY` is set, Facet uses it. It must be at least 32 characters. Generate one with `openssl rand -hex 32`.
2. Otherwise, if `/data/.encryption_key` exists, Facet reads it.
3. Otherwise, Facet generates a key and saves it to `/data/.encryption_key` with mode 600. The key itself is never printed to the log. An existing key file is never overwritten.

If `/data/.encryption_key` exists but is empty, the container stops with an error instead of quietly making a new key. Restore the file from a backup, or delete it if you really want a fresh key.

The automatic key lives inside `/data`, so a complete backup of `/data` includes it. If you lose the key, stored secrets have to be entered again and existing share links and API keys stop working. Your profile and content aren't affected ([details](UPGRADE.md#lost-encryption-key)). If you'd rather keep the key outside the data folder, set `ENCRYPTION_KEY` yourself and store it in a password manager.

## Your first ten minutes

- **Profile:** add your name, headline and photo under **Profile**.
- **Content:** fill in **Experience**, **Projects** and the other sections, or upload a résumé under **Import & AI** if you've [set up an AI provider](AI_FEATURES.md).
- **Views:** create a view for each audience under **Views**. [Features](FEATURES.md#views) explains how views work.
- **Try the demo:** the demo toggle in the admin header loads a sample profile so you can see a full site. Your own content is kept separately and comes back when you switch the demo off. While the demo is on, your public pages show the demo profile too ([Features](FEATURES.md#demo-mode)).
- **Two-factor sign-in:** turn it on in **Account & Security**.
- **Backups:** read [Upgrades, backups and recovery](UPGRADE.md) before you put real work into Facet.

## Where to go next

- [Putting Facet on the internet](SELF-HOSTING-GUIDE.md): domain, HTTPS, reverse proxy, tunnels, OAuth
- [Configuration reference](CONFIGURATION.md): every environment variable
- [Troubleshooting](TROUBLESHOOTING.md): when something doesn't start or sign-in fails
