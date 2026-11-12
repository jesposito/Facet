# Production container base images

Checked **2026-11-12** against `main` at `8b6436f` (`v3.8.9`). The separate Node fix branch is noted below but is not treated as merged.

All three production bases use `tag@sha256:digest`. The Go and Debian pins match Docker Hub's current tag digests. The Node tag has moved to the 24.22.3 security image while `main` still selects the older digest. Digest drift alone does not prove a vulnerability; here the upstream tag change also has an explicit Node security release.

| Base                     | Facet digest prefix | Current tag digest                                                        | Docker Hub image created |
| ------------------------ | ------------------- | ------------------------------------------------------------------------- | ------------------------ |
| `golang:1.27.4-bookworm` | `4fa7c968…`         | `sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea` | 2026-11-10               |
| `node:24-alpine`         | `06d7adf3…`         | `sha256:116b800c3754b440ee0aa3ed2f20f5bf3d99a554bfd9adeea09ea8017f6e85f5` | 2026-11-11               |
| `debian:bookworm-slim`   | `7ff6ba31…`         | `sha256:734a72b3d6142a71993b4bc580ca01d8ca687f77a2c422d96a961d3376b179bc` | 2026-11-04               |

The current Node image's source revision sets `NODE_VERSION=24.22.3`. Commit [`e622136`](https://github.com/jesposito/Facet/commit/e622136e5ed4894c4d85e0c1dcb0c2831c4812d1) on `fix/node-24.22.3` updates Facet's Node builder pin to that digest and rebuilds the final image so NodeSource selects the same security patch at runtime. Merge and release it, then verify the packaged runtime version. See `node.md`.

[PR #533](https://github.com/jesposito/Facet/pull/533), merged as **`25fe89f`** and released in **v3.8.8**, updated both Go modules and the development and production Go tag+digest pairs. `Facet-zl8` is closed. [Release run 42946968507](https://github.com/jesposito/Facet/actions/runs/42946968507) passed both architecture builds and the merge job for v3.8.9, but shipped before the Node fix branch.

Refresh each tag and digest together, then run `bash scripts/check-docker-base-pins.sh`, the multi-architecture build, image smoke test and Trivy scan. [PR #528](https://github.com/jesposito/Facet/pull/528), merged as `fde26de`, added an explicit APT upgrade for the CVE-2026-109518 fix. Its disposable image check found apt and libapt-pkg6.0 at 2.6.2, and both architecture scans and packaged smoke tests passed.

Sources: [`docker/Dockerfile`](../docker/Dockerfile), [`docker/Dockerfile.dev`](../docker/Dockerfile.dev), Docker Hub API for [Go 1.27.4](https://hub.docker.com/v2/repositories/library/golang/tags/1.27.4-bookworm), [Node](https://hub.docker.com/v2/repositories/library/node/tags/24-alpine) and [Debian](https://hub.docker.com/v2/repositories/library/debian/tags/bookworm-slim), [current Node image source](https://raw.githubusercontent.com/nodejs/docker-node/1274f3ee37d4d0e59d308ea25b83d9e21e4db862/24/alpine3.24/Dockerfile), [Node 24.22.3 release](https://nodejs.org/en/blog/release/v24.22.3), [Debian CVE tracker](https://security-tracker.debian.org/tracker/CVE-2026-109518), [PR #533](https://github.com/jesposito/Facet/pull/533), [v3.8.9 release run](https://github.com/jesposito/Facet/actions/runs/42946968507), checked 2026-11-12.
