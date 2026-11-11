# Production container base images

Checked **2026-11-11**.

All three production bases use `tag@sha256:digest`. The Go 1.27.4 builder pin matches Docker Hub's current tag digest. The Node and Debian tag digests have drifted from Facet's pins. Digest drift alone does not prove a vulnerability; refreshing is how a rebuild selects current base contents.

| Base                     | Facet digest prefix | Current tag digest                                                        | Docker Hub tag updated |
| ------------------------ | ------------------- | ------------------------------------------------------------------------- | ---------------------- |
| `golang:1.27.4-bookworm` | `4fa7c968…`         | `sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea` | 2026-11-10             |
| `node:24-alpine`         | `06d7adf3…`         | `sha256:a34ab6d7f6f4813f4d0c5469d74685604b962fbdbf5793a7bc7b4998ecb867c8` | 2026-11-05             |
| `debian:bookworm-slim`   | `7ff6ba31…`         | `sha256:734a72b3d6142a71993b4bc580ca01d8ca687f77a2c422d96a961d3376b179bc` | 2026-11-05             |

[PR #533](https://github.com/jesposito/Facet/pull/533), merged as **`25fe89f`** and released in **v3.8.8**, updated both Go modules and the development and production Go tag+digest pairs. `Facet-zl8` is closed. [Release run 42941554396](https://github.com/jesposito/Facet/actions/runs/42941554396) passed both architecture scans, signing and manifest publication.

Refresh each tag and digest together, then run `bash scripts/check-docker-base-pins.sh`, the multi-architecture build, image smoke test and Trivy scan. [PR #528](https://github.com/jesposito/Facet/pull/528), merged as `fde26de`, added an explicit APT upgrade for the CVE-2026-109518 fix. Its disposable image check found apt and libapt-pkg6.0 at 2.6.2, and both architecture scans and packaged smoke tests passed.

Sources: [`docker/Dockerfile`](../docker/Dockerfile), [`docker/Dockerfile.dev`](../docker/Dockerfile.dev), Docker Hub API for [Go 1.27.4](https://hub.docker.com/v2/repositories/library/golang/tags/1.27.4-bookworm), [Node](https://hub.docker.com/v2/repositories/library/node/tags/24-alpine) and [Debian](https://hub.docker.com/v2/repositories/library/debian/tags/bookworm-slim), [Debian CVE tracker](https://security-tracker.debian.org/tracker/CVE-2026-109518), [PR #533](https://github.com/jesposito/Facet/pull/533), [v3.8.8 release run](https://github.com/jesposito/Facet/actions/runs/42941554396), checked 2026-11-11.
