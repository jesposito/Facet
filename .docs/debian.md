# Debian Bookworm runtime

Checked **2026-11-10**.

Facet's production runtime uses `debian:bookworm-slim` pinned by digest. Debian 12 Bookworm is now oldstable; Debian 13.7 Trixie is current stable. Bookworm LTS support continues until **2028-06-30**, after full support ended in July 2026.

Bookworm receives security updates through its security/LTS channels. The original published Facet image had apt findings for CVE-2026-109518; Bookworm security apt **2.6.2** is fixed. [PR #528](https://github.com/jesposito/Facet/pull/528), merged as `fde26de`, refreshes the base pins, and a disposable image check found both affected packages at 2.6.2. Both architecture image scans and packaged smoke tests passed; ARM64 publication remains blocked by `Facet-dcy`.

Sources: [`docker/Dockerfile`](../docker/Dockerfile), [official Debian releases](https://www.debian.org/releases/), [Debian security tracker](https://security-tracker.debian.org/tracker/CVE-2026-109518), [published-image scan](https://github.com/jesposito/Facet/actions/runs/42568088543), checked 2026-11-10.
