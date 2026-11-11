# Docker Compose

Checked **2026-11-11**.

The host runs **2.40.3** from Ubuntu; official upstream latest is **5.7.0**. This is a new major. Review the migration and the host's supported packaging path rather than replacing the distro binary ad hoc. Production uses `docker-compose.yml`; development uses `docker-compose.dev.yml` with loopback-published ports and named dependency caches.

Sources: local `docker compose version`, [`docker-compose.yml`](../docker-compose.yml), [`docker-compose.dev.yml`](../docker-compose.dev.yml), [official v5.7.0 release](https://github.com/docker/compose/releases/tag/v5.7.0), checked 2026-11-11.
