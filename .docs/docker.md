# Docker, Compose and runtime image

Checked **2026-11-11**.

## Host tools

| Tool           | Host                                   | Latest upstream | Status                                                                          |
| -------------- | -------------------------------------- | --------------- | ------------------------------------------------------------------------------- |
| Docker Engine  | 29.1.3 (`29.1.3-0ubuntu3~24.04.2`)     | 29.9.3          | Update through the host's supported Ubuntu package channel                      |
| Docker Compose | 2.40.3 (`2.40.3+ds1-0ubuntu1~24.04.1`) | 5.7.0           | New major; review compatibility rather than replacing the distro package ad hoc |

Docker 29.9.3 includes fixes for crafted RSA-key CPU use, zstd decompression, Swarm overlay protections and BuildKit issues. That upstream release does not prove the Ubuntu host package is exploitable in Facet's configuration; it does make a supported host update prudent.

Sources: local `docker version` and `docker compose version` (2026-11-11), [Docker Engine 29 release notes](https://docs.docker.com/engine/release-notes/29/), [Docker Compose 5.7.0 release](https://github.com/docker/compose/releases/tag/v5.7.0), checked 2026-11-11.

## Image inputs

Production uses a multi-stage Dockerfile:

- `golang:1.27.3-bookworm` for backend and Caddy builds;
- `node:24-alpine` for the frontend build;
- `debian:bookworm-slim` for the runtime.

All three production bases are pinned to multi-architecture SHA-256 index digests. `scripts/check-docker-base-pins.sh` enforces that policy. Refresh a tag and digest together. Go 1.27.4 is now available with security fixes; [PR #533](https://github.com/jesposito/Facet/pull/533) updates both Go builder stages and `Facet-zl8` tracks it.

The runtime installs Node 24 from NodeSource and Pandoc, LaTeX, `libheif-examples`, curl, wget, OpenSSL and other support packages from Debian apt. Those package versions are selected at image build time rather than locked separately in the Dockerfile. Rebuilding the same pinned base can therefore still select newer apt packages.

[Published-image scan 42568088543](https://github.com/jesposito/Facet/actions/runs/42568088543) reported exactly two HIGH findings, `apt` and `libapt-pkg6.0`, for CVE-2026-109518. Debian's tracker lists Bookworm 2.6.1 as vulnerable and Bookworm security **2.6.2** as fixed. [PR #528](https://github.com/jesposito/Facet/pull/528), merged as `fde26de`, added an explicit APT upgrade; its disposable image contained both packages at 2.6.2 and both architecture scans and packaged smoke tests passed. The later [published-image scan 42724230846](https://github.com/jesposito/Facet/actions/runs/42724230846) passed, and subsequent releases published both architectures.

Development-only `node:24-alpine` and `golang:1.27.3-alpine` bases intentionally float and carry `supply-chain: allow-floating-base` markers. PR #533 updates the Go development tag to 1.27.4. `docker-compose.dev.yml` also uses floating `node:24-alpine` for its frontend container.

The production image runs Caddy, SvelteKit and PocketBase under one supervisor, exposes only port 8080, and stores persistent state in `/data` and `/uploads`.

Sources: [`docker/Dockerfile`](../docker/Dockerfile), [`docker/Dockerfile.dev`](../docker/Dockerfile.dev), [`docker/start.sh`](../docker/start.sh), [`docker-compose.dev.yml`](../docker-compose.dev.yml), [`scripts/check-docker-base-pins.sh`](../scripts/check-docker-base-pins.sh), [`CONTRIBUTING.md`](../CONTRIBUTING.md), [PR #533](https://github.com/jesposito/Facet/pull/533), all checked 2026-11-11. Exact Node and apt package versions inside a freshly built local image were not resolved during this inventory; CI image evidence is cited instead.
