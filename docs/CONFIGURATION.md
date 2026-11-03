# Configuration reference

Facet reads its configuration from environment variables when the container starts. Most day-to-day settings (site appearance, SMTP, AI providers, Stripe, webhooks) can also be changed in the admin dashboard, and some can only be changed there.

After changing a variable, recreate the container so it picks up the new value:

```bash
docker compose up -d
```

`docker compose restart` reuses the old environment and won't apply your change.

## Does the variable reach the container?

Docker Compose reads `.env` to fill in `${VARIABLES}` in the Compose file. The container itself only sees variables listed under `environment:`.

- The [Compose file in the install guide](SETUP.md#option-1-published-image-with-docker-compose-recommended) lists every variable on this page.
- The repository's own [`docker-compose.yml`](../docker-compose.yml) passes through the core, proxy, OAuth, SMTP, AI, Stripe, backup, `PUID`/`PGID` and tuning variables, plus `SEED_DATA`. It doesn't pass `FACET_FEATURE_COURSES`, `FACET_MANAGED` or `PORT` (which only sets the published host port). Add any variable that's missing to its `environment:` list.

To check what the container actually received:

```bash
docker exec facet printenv | sort
```

That prints secrets too, so don't paste the output anywhere public.

## Core settings

### `APP_URL`

Your public address, such as `https://facet.example.com` or `http://192.168.1.20:8080`. No trailing slash. The bundled Compose file defaults to `http://localhost:8080`; set it explicitly for other installation methods.

Facet uses it for OAuth callbacks, share links, sitemap and robots URLs, and email links. It must match the address people actually type into the browser. If it doesn't, sign-in redirects and some form submissions fail.

The frontend derives the request origin from `PROTOCOL_HEADER` and `HOST_HEADER`, supplied by Caddy. The current adapter does not use the legacy `ORIGIN` variable. Keep the frontend port private and configure your outer proxy to preserve the public host and scheme.

`PUBLIC_APP_URL` is an optional override used only for robots.txt and sitemap links. It takes precedence over `APP_URL` for those routes and is not passed through by the bundled Compose file. Most installations should set only `APP_URL`.

Facet also logs a warning at startup when `APP_URL` doesn't start with `https://`, unless it's empty, `http://localhost:8080` or `http://localhost:5173` ([source](../backend/hooks/security.go)). The warning is informational: Facet still runs.

### `ADMIN_EMAILS`

A comma-separated list of email addresses allowed to sign in, for example `you@example.com,backup@example.com`. Default: empty.

- The first address becomes the email of the account Facet creates on first start.
- With a list set, password and OAuth sign-in are refused for any other address.
- With an empty list, any account in the `users` table can sign in with a password, and OAuth sign-in is refused once one account exists.

Set it before your instance is reachable by anyone else. [Security](SECURITY.md#who-can-sign-in) has the details.

### `ENCRYPTION_KEY`

The key used to encrypt stored secrets. At least 32 characters. Generate one with `openssl rand -hex 32` (or `make gen-key` in a checkout). Default: empty, in which case Facet reads `/data/.encryption_key` or creates it on first start.

Don't change this once Facet has stored secrets. A different key can't decrypt the existing AI keys, GitHub token, Stripe secrets or two-factor secrets, and existing share links, testimonial request links and API keys stop matching. [The encryption key](SETUP.md#the-encryption-key) explains the lookup order.

### `TRUST_PROXY`

Working out a visitor's real address takes two steps, and two variables control them ([client IP](../backend/services/clientip.go), [Caddyfile](../docker/Caddyfile)):

1. **Caddy, at the edge of the container**, decides which address the request came from. It believes `X-Forwarded-For` only from peers listed in `TRUSTED_PROXIES`, reading right to left and stopping at the first untrusted address. It then hands the backend a single, cleaned address.
2. **The backend** believes that address only when the request comes from loopback (Caddy, or a proxy on the same host in a native install), and only the rightmost entry. `TRUST_PROXY=false` turns this off, so every request is keyed on the connecting peer.

Headers a visitor can set themselves, such as `CF-Connecting-IP` and `X-Real-IP`, are never used.

| Variable | Default | Set it to |
|----------|---------|-----------|
| `TRUST_PROXY` | `true` in `docker-compose.yml` and `.env.example`; unset means trusted | `false` only to ignore forwarded addresses entirely, which puts every visitor into Caddy's single rate-limit bucket |
| `TRUSTED_PROXIES` | `127.0.0.1/32` | Space-separated CIDRs of your reverse proxy or tunnel, for example the Docker network of your Cloudflare Tunnel or Nginx Proxy Manager container (`172.18.0.0/16`). Never `0.0.0.0/0` or `::/0`. |

With the default `TRUSTED_PROXIES`, Caddy takes every connection at face value. Behind a proxy that means every visitor looks like the proxy and shares one rate-limit bucket, so set it when you add a proxy. The client address also feeds view deduplication and the access log.

Neither variable affects how links are built. That's what `APP_URL` is for.

### `PORT`

The **host** port mapped to Caddy's port 8080 in the Compose files. Default: `8080`. Change it if 8080 is already taken, for example `PORT=8081`, and update `APP_URL` to match.

Inside the container, `PORT` means the SvelteKit server's port (`3000`, set by the [Dockerfile](../docker/Dockerfile)). Never pass your `.env` `PORT` into the container.

### `DATA_PATH` and `UPLOADS_PATH`

Host folders mounted at `/data` and `/uploads` by the Compose files. Defaults: `./data` and `./uploads`. Use absolute paths if you keep data on another disk, for example `/mnt/user/appdata/facet`.

### `PUID` and `PGID`

The user and group IDs the app runs as inside the container. Default: `1000` for both. At startup the container changes ownership of `/app`, `/data` and `/uploads` to this user, then drops root ([start script](../docker/start.sh)). Set them to match the owner of your host folders, which is common on Unraid and Synology.

### `ADMIN_ENABLED`

Set to `true` to expose PocketBase's own database dashboard at `/_/`. Default: `false`, which makes Caddy answer `/_/` with 404 ([Caddyfile](../docker/Caddyfile)). This dashboard is separate from Facet's `/admin` and gives raw access to every table. Turn it on only while you need it. It needs a PocketBase superuser account. [Recovery](UPGRADE.md#use-the-pocketbase-dashboard) shows how to create one.

## Sign-in providers

| Variable | Purpose |
|----------|---------|
| `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` | Enable "Sign in with Google" |
| `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET` | Enable "Sign in with GitHub" |

A provider is enabled only when both its ID and secret are set; Facet logs a warning and skips it if one is missing ([source](../backend/hooks/oauth.go)). When at least one provider is configured, Facet writes the providers and `APP_URL` into PocketBase's settings at every start. If you remove the variables later, the previously saved providers stay enabled until you change them in the PocketBase dashboard.

The redirect URL to register with Google or GitHub is `<APP_URL>/api/oauth2-redirect`. [Putting Facet on the internet](SELF-HOSTING-GUIDE.md#sign-in-with-google-or-github) walks through it.

## Email (SMTP)

| Variable | Default | Purpose |
|----------|---------|---------|
| `SMTP_HOST` | (none) | Mail server. Setting this turns the rest on. |
| `SMTP_PORT` | `587` | Mail server port |
| `SMTP_USERNAME` | (none) | Login name |
| `SMTP_PASSWORD` | (none) | Password or app password |
| `SMTP_TLS` | Follows the port | Unset: implicit TLS on port 465, STARTTLS (when the server offers it) on every other port, including 587. `true`/`1` forces implicit TLS; any other value turns it off. Facet logs a warning for `SMTP_TLS=true` on port 587. |
| `SMTP_AUTH_METHOD` | `PLAIN` | Authentication method passed to PocketBase |
| `SMTP_SENDER_NAME` | `Facet` | Display name on outgoing mail |
| `SMTP_SENDER_ADDRESS` | (none) | From address |
| `APP_NAME` | (none) | Application name PocketBase uses in emails |

You can configure the same settings in **Settings → Site Settings** instead. When `SMTP_HOST` is set, Facet copies these values into PocketBase's mail settings at every start ([source](../backend/hooks/smtp.go)), overwriting what you changed in the dashboard. Username, password and sender values you leave blank in the environment keep their stored values. If nothing differs from what's stored, nothing is saved. Use one method or the other.

`APP_URL` and `APP_NAME` are applied the same way even without SMTP. PocketBase won't store a blank app URL and falls back to `http://localhost:8090`, so set `APP_URL` on any instance that sends email or takes payments.

Email is used for testimonial notifications and verification, newsletter sends, and tests sent from the admin.

SMTP delivery has a 30-second deadline covering connection, TLS negotiation and the SMTP exchange. A stalled server returns a send error when that deadline expires. Facet keeps a small [local transport patch](../backend/third_party/mailyak-v3/FACET_PATCH.md) until the upstream library provides this behavior.

## AI providers

All optional. You can add providers in **Settings → Integrations** instead, which also lets you use any OpenAI-compatible endpoint.

| Variable | Creates |
|----------|---------|
| `ANTHROPIC_API_KEY` | A provider named "Claude (Auto)" |
| `OPENAI_API_KEY` | A provider named "OpenAI (Auto)" |
| `OLLAMA_BASE_URL` | A provider named "Ollama (Auto)" pointing at your Ollama server |
| `OLLAMA_MODEL` | The Ollama model to use. Default: `llama3.2` |

Facet creates these providers the first time the admin checks AI status **and no provider exists yet** ([source](../backend/hooks/ai.go)). After that the variables are ignored: changing or removing them doesn't change the saved provider. Edit providers in the dashboard instead. [AI features](AI_FEATURES.md) covers models and what data is sent.

## Payments

| Variable | Purpose |
|----------|---------|
| `STRIPE_SECRET_KEY` | Fallback Stripe secret key |
| `STRIPE_WEBHOOK_SECRET` | Fallback Stripe webhook signing secret |

Keys saved in **Settings → Payments** take precedence over these ([source](../backend/hooks/purchase.go)). See [Newsletter, payments and courses](AUDIENCE.md#payments-stripe) for what's built.

## Built-in backups

| Variable | Default | Purpose |
|----------|---------|---------|
| `BACKUP_HOUR` | `2` | Hour of day (UTC, 0–23) for the automatic backup |
| `BACKUP_MAX_COUNT` | `10` | How many backup archives to keep before deleting the oldest |

See [Upgrades, backups and recovery](UPGRADE.md#built-in-backups) for what these archives contain. They don't include uploads.

## Features and tuning

| Variable | Default | Purpose |
|----------|---------|---------|
| `FACET_FEATURE_COURSES` | `true` | Courses are on for self-hosted sites. `false` or `0` turns off the catalogue, new enrolments and course checkout; people who already bought or enrolled keep access. A value that isn't a boolean keeps the default ([source](../backend/services/plan.go)). Not passed through by the repository's `docker-compose.yml`. |
| `FACET_MANAGED` | `false` | Compatibility switch for managed hosting. Unknown features default to off in this mode. It isn't a supported way to run self-hosted Facet, and it doesn't make this repository a cloud product. |
| `NEWSLETTER_MAX_CONCURRENT_SENDS` | `4` | Limits how many newsletter sends run at once ([source](../backend/hooks/newsletter_concurrency.go)) |
| `VIEW_DEDUP_WINDOW` | `30m` | Go duration (for example `30m`) during which repeat hits from the same visitor don't increase view counts |
| `CONTENT_RECOVERY_DISABLED` | (unset) | Set to `1` to skip a one-time repair of rich-text content at startup ([source](../backend/hooks/content_recovery.go)) |

## Set by the image

You normally leave these alone. The [Dockerfile](../docker/Dockerfile) and [start script](../docker/start.sh) set them.

| Variable | Value in the image | Purpose |
|----------|--------------------|---------|
| `DATA_DIR` | `/data` | PocketBase data directory |
| `UPLOADS_DIR` | `/uploads` | Where uploaded files are stored. `/data/storage` is a symlink to it. |
| `POCKETBASE_URL` | `http://localhost:8090` | How the SvelteKit server reaches the backend |
| `PORT`, `HOST` | `3000`, `0.0.0.0` | SvelteKit server address inside the container |
| `PROTOCOL_HEADER`, `HOST_HEADER` | `x-forwarded-proto`, `x-forwarded-host` | Headers SvelteKit trusts for the original scheme and host |
| `NODE_ENV` | `production` | |
| `FACET_VERSION`, `BUILD_DATE`, `GIT_COMMIT` | build arguments | Shown in the startup banner and **About Facet** |

## Development only

These are for working on Facet, not running it. See the [development guide](DEV.md).

| Variable | Purpose |
|----------|---------|
| `SEED_DATA` | `minimal` (default) creates only the first account. `dev` loads the maintainer's development profile. |
| `DEV_ADMIN_EMAIL` | Email for the seeded account when `ADMIN_EMAILS` is empty |
| `DEV_MODE` | `true` skips the HTTPS startup warning |
| `VITE_POCKETBASE_URL` | Backend URL for a Vite dev server on another host or port |
| `VITE_DISABLE_LIVE_PREVIEW` | Turns off the admin live-preview pane in development builds |
| `PLAYWRIGHT_BASE_URL`, `API_BASE_URL`, `ADMIN_EMAIL`, `ADMIN_PASSWORD`, `PB_SUPERUSER_EMAIL`, `PB_SUPERUSER_PASSWORD` | Browser test settings. See [frontend/tests/README.md](../frontend/tests/README.md). |

`FACET_MANAGED` and `FACET_DEMO_MODE` exist for compatibility with the hosted service and have no use in a self-hosted install.

## Variables that don't exist

Older versions of these docs mentioned `ENCRYPTION_KEY_AUTOPROVISION` and `ALLOWED_ORIGINS`. Facet doesn't read either. Key generation happens automatically without a flag, and there is no global CORS setting.
