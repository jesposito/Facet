# Troubleshooting

Start with the logs. Most problems announce themselves there.

```bash
docker compose logs --tail=200 facet
```

The startup banner shows the version, where data and uploads are stored, how the encryption key was found, and the main settings. Lines start with `[Config]`, `[Storage]`, `[Backend]`, `[Frontend]` or `[Proxy]`, so you can tell which part is talking.

Two quick health checks:

```bash
curl -s http://localhost:8080/health      # frontend and backend together
curl -s http://localhost:8080/api/health  # backend only
```

## The container won't start or keeps restarting

**`FATAL: ENCRYPTION_KEY must be at least 32 characters`.** You've set `ENCRYPTION_KEY` to something short. Generate a proper one with `openssl rand -hex 32`, or remove the variable to let Facet manage its own key. If Facet has already stored secrets under the old key, see [Lost encryption key](UPGRADE.md#lost-encryption-key).

**Permission errors on `/data` or `/uploads`.** The container changes ownership of both folders to `PUID:PGID` (default `1000:1000`) at startup. That can fail on some network shares (NFS with root squashing, SMB mounts). Set `PUID` and `PGID` to a user that can write there, or keep `DATA_PATH` on a local disk. SQLite also needs file locking that works, which some network filesystems don't provide reliably.

**`[Storage] WARNING: only N of M files reached uploads`.** An older install is moving files from `/data/storage` to `/uploads` and the copy came up short, usually because the disk is full or permissions are wrong. Facet stops rather than risk losing files. Fix the space or permissions and restart; the copy resumes without overwriting anything.

**Port already in use.** Something else is using port 8080 on the host. Set `PORT=8081` (or another free port) in `.env`, update `APP_URL` to match, and run `docker compose up -d`.

**It exits a few seconds after starting.** If any of the three internal processes stops, the container stops. Read the last 50 lines of the log for the first error. During startup, `[Startup] Facet did not start` names the part to look at: `[Backend]` if PocketBase never answered `/api/health`, or `[Frontend]` and `[Proxy]` if `/health` through Caddy never answered. Each check waits 30 seconds by default. On a slow machine, such as a Raspberry Pi doing its first migration, set `FACET_HEALTH_TIMEOUT=90` (seconds) in the container's environment. After startup, a line like `[Frontend] ERROR: process exited (status 1); shutting down` tells you which process stopped first.

## I can't sign in

**I don't know the email.** It's the first address that was in `ADMIN_EMAILS` when Facet first started, or `admin@example.com` if that was empty. The startup banner shows the address it expects.

**"Admin access denied. Your email is not authorized."** The account's email isn't in `ADMIN_EMAILS`. Add it (lists are comma-separated) and run `docker compose up -d`.

**"Admin access denied. Configure ADMIN_EMAILS environment variable."** You tried Google or GitHub sign-in with `ADMIN_EMAILS` empty. OAuth sign-in needs `ADMIN_EMAILS` set to the email on your Google or GitHub account.

**Forgotten password, or lost two-factor device.** See [Locked out](UPGRADE.md#locked-out) for the `reset-admin-password` and `reset-2fa` commands.

**Changing `.env` had no effect.** The repository’s `docker-compose.yml` passes through most operator settings, while the minimal install example includes only common settings. Confirm that the name appears under `environment:` in the Compose file you use. Check selected non-secret values with `docker exec facet printenv APP_URL PUBLIC_APP_URL TRUST_PROXY TRUSTED_PROXIES` ([configuration checks](CONFIGURATION.md#does-the-variable-reach-the-container)). Use `docker compose up -d`, rather than `docker compose restart`, after changing environment settings.

## Google or GitHub sign-in fails

- **`redirect_uri_mismatch` or a similar error from Google or GitHub.** The redirect URL registered with the provider must be exactly `<APP_URL>/api/oauth2-redirect`, with the same scheme (`https`), host and port Facet uses.
- **The sign-in button doesn't appear.** Both the client ID and secret must reach the container. Look for `OAuth env config: providers enabled` in the log, or a warning about partial credentials.
- **It works locally but not through your domain.** `APP_URL` still points at your local address. Change it to the public one and recreate the container.

## Pages load but something's wrong

**Share links or emails contain `localhost`.** `APP_URL` isn't set to your public address, or the container didn't receive it.

**Newsletter links point to the wrong address.** Those use PocketBase's Application URL setting ([how to fix](AUDIENCE.md#before-you-start)).

**Entering a view password fails with "Cross-site POST form submissions are forbidden".** The public browser origin differs from the host or scheme forwarded to SvelteKit. Check that your outer proxy preserves the public Host and X-Forwarded-Proto headers and that only Caddy can reach the frontend port. Keep `APP_URL` set to the same public address for links and payment redirects.

**Everyone gets "too many requests".** Visitors are sharing one rate-limit bucket because Facet can't see their real IP addresses. Behind a reverse proxy or tunnel, set `TRUST_PROXY=true` and make sure port 8080 isn't reachable directly ([details](CONFIGURATION.md#trust_proxy)).

**My site suddenly shows "The Doctor".** Demo mode is on. Switch it off with the toggle in the admin header. Your own content comes back.

**Someone found an unlisted post.** Unlisted posts and talks stay off the `/posts` and `/talks` index pages, feeds and the sitemap, but anyone with the link can open them, and they show in any view that includes them. Make it private if it must stay hidden ([Visibility](FEATURES.md#visibility)).

**Google Analytics shows nothing.** The default Content-Security-Policy blocks Google's script ([details](SECURITY.md#google-analytics)).

## Uploads

**Uploads fail with a 413 error.** A reverse proxy in front of Facet has a smaller body limit than Facet's 25 MB. Raise it ([examples](SELF-HOSTING-GUIDE.md#option-b-a-reverse-proxy-you-run)). Individual fields have their own limits too ([Media](MEDIA.md#uploading)).

**Uploads disappear after recreating the container.** `/uploads` isn't mapped to a host folder or named volume. Check the `volumes:` section and that `UPLOADS_PATH` points where you expect. The startup log shows `[Storage] Using uploads directory: /uploads` when the mapping is in place.

**Images are missing after a restore.** A built-in backup doesn't contain uploads. Restore the uploads folder from your own backup as well ([Restore](UPGRADE.md#restore)).

## Email

**Nothing is sent.** Set up SMTP and use the test button in **Site Settings**. If you configure SMTP with `SMTP_*` variables, they overwrite the dashboard settings each time Facet starts. Self-hosted Facet has no other way to send email.

**The test says sent, but nothing arrives.** "Sent" means your SMTP server accepted the message. Check the spam folder, your provider's logs, and the SPF, DKIM and DMARC records for the sender's domain ([Newsletter](AUDIENCE.md#before-you-start)).

**The newsletter says SMTP isn't ready.** SMTP is configured but the sender address or `APP_URL` is missing. Fill both in and reload the page.

**Gmail rejects the password.** Gmail needs an [app password](https://myaccount.google.com/apppasswords), which requires two-step verification on the account.

## AI

See [AI features → Troubleshooting](AI_FEATURES.md#troubleshooting).

## "Database is locked"

SQLite allows one writer at a time. Brief locks are normal and resolve themselves. Persistent errors usually mean two Facet containers are using the same `/data` folder, or the folder is on a network filesystem with unreliable locking. Run one container per data folder, on local storage.

## Starting completely fresh

To discard everything and start over, stop Facet and **move** the old folders aside rather than deleting them:

```bash
docker compose down
mv data data.old
mv uploads uploads.old
docker compose up -d
```

Facet creates a new database and first account. When you're sure you don't need the old data, delete the `.old` folders.

## Still stuck?

Search the [issues](https://github.com/jesposito/Facet/issues), or open a new one with your Facet version (from **About Facet** or the startup banner), how you installed it, what you expected, what happened, and the relevant log lines. Remove email addresses, keys and tokens from anything you paste.
