# Upgrading Facet

## Standard Upgrade

### 1. Backup First

Always backup before upgrading:

```bash
# Stop the container
docker-compose down

# Backup data directory
tar -czvf facet-backup-$(date +%Y%m%d).tar.gz ./data

# Keep backup somewhere safe!
```

### 2. Pull Latest Version

```bash
# If using docker-compose with build
git pull
docker-compose build

# If using pre-built images
docker-compose pull
```

### 3. Check Release Notes

Before starting, check the [release notes](https://github.com/jesposito/Facet/releases) for:
- Breaking changes
- Required migration steps
- New environment variables

### 4. Start New Version

```bash
docker-compose up -d
```

### 5. Verify

- Check the dashboard loads: `http://localhost:8080/admin`
- Check public profile: `http://localhost:8080`
- Check logs for errors: `docker-compose logs -f`

---

## Rollback

If something goes wrong:

```bash
# Stop new version
docker-compose down

# Restore backup
rm -rf ./data/*
tar -xzvf facet-backup-YYYYMMDD.tar.gz

# Start with the previous version.
# If using pre-built images, pin the previous tag in docker-compose.yml
# (e.g. ghcr.io/jesposito/facet:v2.21.0) and re-pull:
docker-compose pull
docker-compose up -d

# Or, if building from source, check out the previous tag and rebuild:
git checkout v2.21.0
docker-compose build
docker-compose up -d
```

---

## Major Version Upgrades

Facet is currently on the 2.x line (latest: v2.22.1). Minor releases within a major version are designed to upgrade in place with `docker compose pull` and a restart. A future major bump (2.x → 3.x) may include breaking changes — when that happens, always:

1. Read the full changelog
2. Test in a staging environment first
3. Have a rollback plan
4. Schedule during low-traffic time

---

## Database Migrations

Facet runs migrations automatically on boot (PocketBase automigrate is enabled). You do **not** need to run any manual migration command. When you pull a new version and restart:

1. New collections and fields are applied on startup
2. Existing data is preserved
3. No manual `migrate up` step is required — just `docker compose pull` then restart

Because migrations apply on boot, the safe upgrade flow is simply: back up `./data`, pull the new image, and restart the container.

---

## Configuration Changes

When new environment variables are added:

1. Check `.env.example` for new variables
2. Add them to your `.env` file
3. Most new variables have sensible defaults
