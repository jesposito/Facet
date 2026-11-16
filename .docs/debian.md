# Debian Bookworm runtime

Checked **2026-11-16**.

Facet's production runtime uses `debian:bookworm-slim` pinned by digest. Debian 12 Bookworm is oldstable; Debian 13.7 Trixie is current stable. Bookworm LTS support continues until **2028-06-30**, after full support ended in July 2026.

The v3.8.11 Dockerfile selects digest `sha256:734a72b3…`, which matches the current `bookworm-slim` tag. [PR #537](https://github.com/jesposito/Facet/pull/537), merged as `2c347ab` and released in v3.8.11, refreshed the pin. No digest update is pending. Future digest drift alone would not establish a vulnerability; run the multi-architecture build, smoke and Trivy checks before release.

Bookworm receives security updates through its security/LTS channels. The original published Facet image had apt findings for CVE-2026-109518; Bookworm security apt **2.6.2** is fixed. [PR #528](https://github.com/jesposito/Facet/pull/528), merged as `fde26de`, added an explicit APT upgrade to the Dockerfile, and a disposable image check found both affected packages at 2.6.2. The current [published-image scan 43422619540](https://github.com/jesposito/Facet/actions/runs/43422619540) passed on v3.8.11.

Sources: [`docker/Dockerfile`](../docker/Dockerfile), [Docker Hub Bookworm tag](https://hub.docker.com/v2/repositories/library/debian/tags/bookworm-slim), [official Debian releases](https://www.debian.org/releases/), [Debian security tracker](https://security-tracker.debian.org/tracker/CVE-2026-109518), [current published-image scan](https://github.com/jesposito/Facet/actions/runs/43422619540), [PR #537](https://github.com/jesposito/Facet/pull/537), checked 2026-11-16.
