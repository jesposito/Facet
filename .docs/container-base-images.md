# Production container base images

Checked **2026-11-14** against `main` at `795ad77` (`v3.8.10`).

All three production bases use `tag@sha256:digest`. The Go and Node pins match Docker Hub's current tag digests. The Debian tag points to a different digest from the v3.8.10 pin. **The Debian refresh is fixed in [PR #537](https://github.com/jesposito/Facet/pull/537), which selects `sha256:734a72b3…`.** Digest drift alone does not prove a vulnerability; publish through the normal image verification path.

| Base                     | Facet digest prefix | Current tag digest                                                        | Docker Hub image created |
| ------------------------ | ------------------- | ------------------------------------------------------------------------- | ------------------------ |
| `golang:1.27.4-bookworm` | `4fa7c968…`         | `sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea` | 2026-11-10               |
| `node:24-alpine`         | `116b800c…`         | `sha256:116b800c3754b440ee0aa3ed2f20f5bf3d99a554bfd9adeea09ea8017f6e85f5` | 2026-11-11               |
| `debian:bookworm-slim`   | `7ff6ba31…`         | `sha256:734a72b3d6142a71993b4bc580ca01d8ca687f77a2c422d96a961d3376b179bc` | 2026-11-05               |

The selected Node image's source revision sets `NODE_VERSION=24.22.3`. **Fixed in v3.8.10 through [PR #536](https://github.com/jesposito/Facet/pull/536):** Facet's Node builder pin selects that digest and the final-image rebuild installs the same security patch through NodeSource. [Release run 43135469638](https://github.com/jesposito/Facet/actions/runs/43135469638) passed both architecture builds and the merge job, and the lead verified Node 24.22.3 on both published architectures. [Published-image scan 43138170223](https://github.com/jesposito/Facet/actions/runs/43138170223) passed. See `node.md`.

[PR #533](https://github.com/jesposito/Facet/pull/533), merged as **`25fe89f`** and released in **v3.8.8**, updated both Go modules and the development and production Go tag+digest pairs. `Facet-zl8` is closed.

Refresh each tag and digest together, then run `bash scripts/check-docker-base-pins.sh`, the multi-architecture build, image smoke test and Trivy scan. [PR #528](https://github.com/jesposito/Facet/pull/528), merged as `fde26de`, added an explicit APT upgrade for the CVE-2026-109518 fix. Its disposable image check found apt and libapt-pkg6.0 at 2.6.2, and both architecture scans and packaged smoke tests passed.

Sources: [`docker/Dockerfile`](../docker/Dockerfile), [`docker/Dockerfile.dev`](../docker/Dockerfile.dev), Docker Hub API for [Go 1.27.4](https://hub.docker.com/v2/repositories/library/golang/tags/1.27.4-bookworm), [Node](https://hub.docker.com/v2/repositories/library/node/tags/24-alpine) and [Debian](https://hub.docker.com/v2/repositories/library/debian/tags/bookworm-slim), [current Node image source](https://raw.githubusercontent.com/nodejs/docker-node/1274f3ee37d4d0e59d308ea25b83d9e21e4db862/24/alpine3.24/Dockerfile), [Node 24.22.3 release](https://nodejs.org/en/blog/release/v24.22.3), [Debian CVE tracker](https://security-tracker.debian.org/tracker/CVE-2026-109518), [PR #533](https://github.com/jesposito/Facet/pull/533), [v3.8.10 release run](https://github.com/jesposito/Facet/actions/runs/43135469638), checked 2026-11-14.
