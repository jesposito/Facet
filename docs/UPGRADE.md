# Upgrades, backups and recovery

This page covers keeping a self-hosted Facet safe over time: what to back up, how to restore it, how to upgrade and roll back, and how to get back in if you're locked out.

The examples assume the Compose setup from the [install guide](SETUP.md), with your files in `~/facet` and the default `./data` and `./uploads` folders. Adjust the paths if you set `DATA_PATH` or `UPLOADS_PATH` elsewhere.

## What to keep safe

| What | Where | Why it matters |
|------|-------|----------------|
| Database and settings | `/data` (host: `DATA_PATH`) | Your profile, content, views, settings, accounts. Everything except files. |
| Uploaded files | `/uploads` (host: `UPLOADS_PATH`) | Images, documents, generated résumés. The database refers to them by name. |
| Encryption key | `/data/.encryption_key`, or your `ENCRYPTION_KEY` value | Needed to read stored AI keys, the GitHub token, Stripe secrets and two-factor secrets, and to recognise existing share links and API keys |
| Your configuration | `.env` and `docker-compose.yml` | Not essential, but saves you rebuilding them |

A backup that has `/data` but not `/uploads` restores your words without your pictures. Back up both, together.

If you set `ENCRYPTION_KEY` yourself, it isn't stored in `/data`. Keep it somewhere else safe, such as a password manager, because a backup is no use without it. Your content survives a lost key, but encrypted secrets don't, and you'd need to enter them again.

## Back up your data

### The simple, reliable way: stop, archive, start

Stopping the container for a few seconds guarantees the SQLite database isn't mid-write when you copy it.

```bash
cd ~/facet
docker compose stop facet
tar -czf facet-backup-$(date +%Y%m%d-%H%M%S).tar.gz data uploads .env docker-compose.yml
docker compose start facet
```

Then copy the archive somewhere that isn't this machine: another disk, a NAS, or cloud storage. A backup on the same disk won't help when that disk dies.

The archive contains secrets (`.env` and possibly `.encryption_key`), so store it like you'd store a password.

If you used named volumes (the `docker run` option in the install guide), archive them through a throwaway container:

```bash
docker stop facet
docker run --rm -v facet-data:/data -v facet-uploads:/uploads -v "$PWD":/backup \
  debian:bookworm-slim tar -czf /backup/facet-backup.tar.gz /data /uploads
docker start facet
```

### Built-in backups

Facet also makes its own database backups ([source](../backend/hooks/backup.go)):

- **Automatically**, once a day at `BACKUP_HOUR` (default `2`, meaning 02:00 UTC). The first automatic backup happens at the next scheduled hour after the container starts.
- **On demand**, with the **Backup Now** button under **Settings → About Facet**.

Backups always go to `/data/backups/`; there is no `BACKUP_DIR` setting.

Each one is a zip archive in `/data/backups/`, named `facet_auto_…zip` or `facet_manual_…zip`. Facet keeps the newest `BACKUP_MAX_COUNT` archives (default `10`) and deletes older ones.

Built-in backups have two storage limits:

- **They don't include uploads.** The `/uploads` folder is deliberately left out. Back it up separately.
- **They live on the same disk.** They protect against a bad edit, not a dead drive. Copy `/data/backups/` off the machine as part of your routine.

Failed manual or scheduled backups raise a critical alert under **Settings → System Alerts** and write details to the container log. Repeated failures with the same event stay in one unacknowledged alert. Check **About Facet** periodically to confirm recent backups exist.

**Backup Now** reports failures and concurrent operations on the page. Retention runs after a successful backup. If pruning fails, the new archive remains valid; the page shows a warning and System Alerts records the cleanup failure. The folder may temporarily contain more than `BACKUP_MAX_COUNT` archives.

A restore checks that the archive exists before it starts, then runs in the background. A failure raises a critical alert and is logged as `backup: restore failed`. A “restore started” response is not proof of completion: wait for the app to restart and verify the restored content.

`BACKUP_HOUR` must be a whole number from `0` to `23`, and `BACKUP_MAX_COUNT` a whole number above `0`. Any other value is ignored and the default is used.

Because a built-in backup is a consistent copy taken while Facet runs, it's also the way to back up without stopping anything: create a backup in **About Facet**, then copy that archive from `/data/backups/` together with the `/uploads` folder.

### Development checkouts are different

`make backup` in a source checkout archives the **native development** database (`pb_data/` and `uploads/` in the checkout) into `backups/`. It refuses to run while `make dev` is running. It doesn't touch Docker volumes or a production instance ([Makefile](../Makefile)). Use the methods above for a real installation.

## Restore

### From your own archive

```bash
cd ~/facet
docker compose down
mv data data.before-restore
mv uploads uploads.before-restore
tar -xzf facet-backup-20261103-020000.tar.gz
docker compose up -d
docker compose logs -f facet
```

Moving the current folders aside, rather than deleting them, means you can change your mind. Remove the `.before-restore` folders once you've checked the restored site.

You don't need to fix file ownership afterwards. The container sets ownership of `/data` and `/uploads` to `PUID:PGID` every time it starts.

If the archive came from a different machine and you set `ENCRYPTION_KEY` in `.env`, make sure it's the same key that instance used.

### From a built-in backup

Go to **Settings → About Facet**, choose a backup, and confirm the restore. Facet swaps in the backed-up database and restarts its backend. Everyone is signed out, including you, and the page returns you to the login screen once the server is back.

A built-in restore rolls back the database only. Files uploaded after that backup stay in `/uploads` but nothing refers to them, and files deleted since the backup won't come back. The **Media Library** can find and clean up unreferenced files ([Media](MEDIA.md)).

To restore a built-in backup that's no longer listed (for example one you copied off the machine), put the zip file back into `/data/backups/` and refresh the page.

## Upgrade

Facet applies database changes automatically when a new version starts. There's no separate migration command ([`main.go`](../backend/main.go)).

1. **Read what's changing.** Check the [changelog](../CHANGELOG.md) or [releases](https://github.com/jesposito/Facet/releases) for anything marked as breaking or needing action.
2. **Back up**, using the stop-archive-start method above. This is your only reliable way back, because database migrations aren't undone by switching to an older image.
3. **Upgrade.**
   - Published image:
     ```bash
     docker compose pull
     docker compose up -d
     ```
   - Built from source:
     ```bash
     git fetch --tags
     git checkout v3.6.0   # or the release you want
     docker compose up -d --build
     ```
4. **Check it.** Watch `docker compose logs -f facet` for errors, open `/health`, sign in, and look at **Settings → About Facet** for the new version number.

The admin dashboard tells you when a newer release exists. It asks GitHub for the latest version number at most once an hour ([source](../backend/hooks/version.go)).

### Choosing an image tag

Release builds are published to `ghcr.io/jesposito/facet` with several tags ([workflow](../.github/workflows/docker-publish.yml)):

| Tag | Example | `docker compose pull` gives you |
|-----|---------|--------------------------------|
| Exact version | `:3.6.0` | Never changes. Upgrade by editing the tag. |
| Minor line | `:3.6` | Patch releases within 3.6 |
| Major line | `:3` | All 3.x releases, but not 4.0 |
| `latest` | `:latest` | Every release, including a future major version |

Pinning an exact version makes rollbacks easy to reason about. `:3` is a reasonable middle ground. Tags such as `:dev`, `:sha-…` and branch names come from development branches and aren't for production use.

## Roll back

If an upgrade goes wrong, go back to both the old image **and** the backup you took before upgrading. The old image alone may not understand a database the new version has already migrated.

```bash
cd ~/facet
docker compose down
# set image: ghcr.io/jesposito/facet:<previous version> in docker-compose.yml
mv data data.failed-upgrade
mv uploads uploads.failed-upgrade
tar -xzf facet-backup-<from before the upgrade>.tar.gz
docker compose up -d
```

If you build from source, check out the previous tag instead and use `docker compose up -d --build`.

Please [open an issue](https://github.com/jesposito/Facet/issues) with the relevant log lines so the problem can be fixed.

## Locked out

All of the commands below run inside the container. They need two things the container's normal shell doesn't provide automatically: the encryption key (Facet refuses to start any command without it) and the data directory. This wrapper supplies both:

```bash
docker exec -it -u facet facet sh -c \
  'ENCRYPTION_KEY="${ENCRYPTION_KEY:-$(cat /data/.encryption_key)}" ./facet <command> --dir=/data'
```

Replace `<command>` with one of the commands below. `-u facet` runs it as the same user as the app, so any files it touches keep the right owner.

### Forgotten password

```bash
docker exec -it -u facet facet sh -c \
  'ENCRYPTION_KEY="${ENCRYPTION_KEY:-$(cat /data/.encryption_key)}" ./facet reset-admin-password you@example.com --dir=/data'
```

This gives that account a new random password and writes it to `/data/.admin-credentials` with mode `0600`; the command prints the exact path but not the password. Read it with `docker exec -u facet facet cat /data/.admin-credentials`. Every existing session for the account is signed out, including any two-factor browser session. Two-factor sign-in itself stays on, so you'll still need your authenticator. Without an email, it resets `admin@example.com` ([source](../backend/main.go)). Sign in and change the password under **Settings**, then delete the handoff file with `docker exec -u facet facet rm /data/.admin-credentials`.

### Lost your authenticator app and recovery codes

```bash
docker exec -it -u facet facet sh -c \
  'ENCRYPTION_KEY="${ENCRYPTION_KEY:-$(cat /data/.encryption_key)}" ./facet reset-2fa you@example.com --dir=/data'
```

This turns two-factor sign-in off for that account. Turn it back on from **Account & Security** after you sign in.

### "Your email is not authorised"

Your account's email isn't in `ADMIN_EMAILS`. Add it back (comma-separated lists are fine) and recreate the container with `docker compose up -d`. If you've forgotten which email the account uses, the first address that was in `ADMIN_EMAILS` when Facet first started is the likely answer. Otherwise it's `admin@example.com`.

### Use the PocketBase dashboard

For anything the commands above don't cover, such as changing an account's email, PocketBase's own dashboard gives direct database access. Use it carefully, and turn it off afterwards.

1. Create a PocketBase superuser (separate from your Facet account):
   ```bash
   docker exec -it -u facet facet sh -c \
     'ENCRYPTION_KEY="${ENCRYPTION_KEY:-$(cat /data/.encryption_key)}" ./facet superuser upsert you@example.com "a-long-unique-password" --dir=/data'
   ```
2. Set `ADMIN_ENABLED=true` and recreate the container.
3. Open `/_/`, sign in as the superuser, and make your change.
4. Set `ADMIN_ENABLED=false` and recreate the container again.

## Lost encryption key

If `/data/.encryption_key` is gone (and you didn't set `ENCRYPTION_KEY`), Facet generates a new key at the next start. Your content is fine. Anything that depended on the old key stops working:

- AI provider keys (**Settings → Integrations**): enter them again
- the GitHub token used for imports: enter it again
- Stripe keys (**Settings → Payments**): enter them again
- two-factor sign-in: reset it with `reset-2fa` above, then set it up again
- share links, testimonial request links and API keys: their stored hashes were made with the old key, so create new ones and send them out again

## Leftover folders after older upgrades

Older versions kept uploads inside `/data/storage`. On startup, the container moves them to `/uploads` and keeps a copy in `/data/.storage-backup` ([start script](../docker/start.sh)). Once you've confirmed your images display correctly, you can delete `/data/.storage-backup` to reclaim space.
