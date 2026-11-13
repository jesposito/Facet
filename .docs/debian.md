# Debian Bookworm runtime

Checked **2026-11-14**.

Facet's production runtime uses `debian:bookworm-slim` pinned by digest. Debian 12 Bookworm is now oldstable; Debian 13.7 Trixie is current stable. Bookworm LTS support continues until **2028-06-30**, after full support ended in July 2026.

The v3.8.10 Dockerfile selects digest `sha256:7ff6ba31…`, while the current `bookworm-slim` tag selects `sha256:734a72b3…`. [PR #537](https://github.com/jesposito/Facet/pull/537) refreshes the pin to the current digest. Digest drift alone does not establish a vulnerability; run the multi-architecture build, smoke and Trivy checks before release.

Bookworm receives security updates through its security/LTS channels. The original published Facet image had apt findings for CVE-2026-109518; Bookworm security apt **2.6.2** is fixed. [PR #528](https://github.com/jesposito/Facet/pull/528), merged as `fde26de`, added an explicit APT upgrade to the Dockerfile, and a disposable image check found both affected packages at 2.6.2. Both architecture image scans and packaged smoke tests passed. The current [published-image scan 43138170223](https://github.com/jesposito/Facet/actions/runs/43138170223) also passed on v3.8.10.

Sources: [`docker/Dockerfile`](../docker/Dockerfile), [Docker Hub Bookworm tag](https://hub.docker.com/v2/repositories/library/debian/tags/bookworm-slim), [official Debian releases](https://www.debian.org/releases/), [Debian security tracker](https://security-tracker.debian.org/tracker/CVE-2026-109518), [original published-image scan](https://github.com/jesposito/Facet/actions/runs/42568088543), [current published-image scan](https://github.com/jesposito/Facet/actions/runs/43138170223), [PR #537](https://github.com/jesposito/Facet/pull/537), checked 2026-11-14.
