# Upgrading me.yaml

## Standard Upgrade

### 1. Backup First

Always backup before upgrading:

```bash
# Stop the container
docker-compose down

# Backup data directory
tar -czvf me-yaml-backup-$(date +%Y%m%d).tar.gz ./data

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

Before starting, check the [release notes](https://github.com/yourusername/me-yaml/releases) for:
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
tar -xzvf me-yaml-backup-YYYYMMDD.tar.gz

# Start with old version (if you have it)
# Or: checkout previous git tag and rebuild
git checkout v1.0.0
docker-compose build
docker-compose up -d
```

---

## Major Version Upgrades

Major versions (1.x → 2.x) may include breaking changes. Always:

1. Read the full changelog
2. Test in a staging environment first
3. Have a rollback plan
4. Schedule during low-traffic time

---

## Database Migrations

PocketBase handles schema migrations automatically. When you upgrade:

1. New collections/fields are added automatically
2. Existing data is preserved
3. Migrations run on first startup

If you need to run migrations manually:

```bash
docker-compose exec ownprofile ./ownprofile migrate up
```

---

## Configuration Changes

When new environment variables are added:

1. Check `.env.example` for new variables
2. Add them to your `.env` file
3. Most new variables have sensible defaults
