# Production container base images

Checked **2026-11-16** against `main` at `43bf199` (`v3.8.11`).

All three production bases use `tag@sha256:digest`, and every pin matches Docker Hub's current multi-architecture tag digest. The Debian refresh from [PR #537](https://github.com/jesposito/Facet/pull/537) is released in v3.8.11; no base-digest update is pending. Digest drift alone would not prove a vulnerability, so future refreshes should use the normal image verification path.

| Base                     | Facet digest prefix | Current tag digest                                                        | Docker Hub last updated |
| ------------------------ | ------------------- | ------------------------------------------------------------------------- | ----------------------- |
| `golang:1.27.4-bookworm` | `4fa7c968…`         | `sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea` | 2026-11-10              |
| `node:24-alpine`         | `116b800c…`         | `sha256:116b800c3754b440ee0aa3ed2f20f5bf3d99a554bfd9adeea09ea8017f6e85f5` | 2026-11-11              |
| `debian:bookworm-slim`   | `734a72b3…`         | `sha256:734a72b3d6142a71993b4bc580ca01d8ca687f77a2c422d96a961d3376b179bc` | 2026-11-05              |

The selected Node image's source revision sets `NODE_VERSION=24.22.3`. **Fixed in v3.8.10 through [PR #536](https://github.com/jesposito/Facet/pull/536):** Facet's Node builder pin selects that digest and the final-image rebuild installs the same security patch through NodeSource. The lead verified Node 24.22.3 on both published architectures for v3.8.10 and again for v3.8.11. [Published-image scan 43422619540](https://github.com/jesposito/Facet/actions/runs/43422619540) passed on v3.8.11. See `node.md`.

[PR #533](https://github.com/jesposito/Facet/pull/533), merged as **`25fe89f`** and released in **v3.8.8**, updated both Go modules and the development and production Go tag+digest pairs. `Facet-zl8` is closed.

Refresh each tag and digest together, then run `bash scripts/check-docker-base-pins.sh`, the multi-architecture build, image smoke test and Trivy scan. [PR #528](https://github.com/jesposito/Facet/pull/528), merged as `fde26de`, added an explicit APT upgrade for the CVE-2026-109518 fix. Its disposable image check found apt and libapt-pkg6.0 at 2.6.2, and both architecture scans and packaged smoke tests passed.

Sources: [`docker/Dockerfile`](../docker/Dockerfile), [`docker/Dockerfile.dev`](../docker/Dockerfile.dev), Docker Hub API for [Go 1.27.4](https://hub.docker.com/v2/repositories/library/golang/tags/1.27.4-bookworm), [Node](https://hub.docker.com/v2/repositories/library/node/tags/24-alpine) and [Debian](https://hub.docker.com/v2/repositories/library/debian/tags/bookworm-slim), [Node 24.22.3 release](https://nodejs.org/en/blog/release/v24.22.3), [Debian CVE tracker](https://security-tracker.debian.org/tracker/CVE-2026-109518), [PR #533](https://github.com/jesposito/Facet/pull/533), [PR #537](https://github.com/jesposito/Facet/pull/537), [v3.8.11 published-image scan](https://github.com/jesposito/Facet/actions/runs/43422619540), checked 2026-11-16.
