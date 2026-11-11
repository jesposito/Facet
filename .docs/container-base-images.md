# Production container base images

Checked **2026-11-11**.

All three production bases use `tag@sha256:digest`. Docker Hub's current multi-architecture tag digests differ from the pins in `docker/Dockerfile` on `main` at `07de104`. Digest drift alone does not prove a vulnerability; refreshing is how a rebuild selects current base contents.

| Base                     | Facet digest prefix | Current tag digest                                                        | Docker Hub tag updated |
| ------------------------ | ------------------- | ------------------------------------------------------------------------- | ---------------------- |
| `golang:1.27.3-bookworm` | `972b4110…`         | `sha256:dc0963f32053e4bc451d24d87e3892d42d01021b61f093e98eb95c0352bb84b0` | 2026-11-05             |
| `node:24-alpine`         | `06d7adf3…`         | `sha256:a34ab6d7f6f4813f4d0c5469d74685604b962fbdbf5793a7bc7b4998ecb867c8` | 2026-11-05             |
| `debian:bookworm-slim`   | `7ff6ba31…`         | `sha256:734a72b3d6142a71993b4bc580ca01d8ca687f77a2c422d96a961d3376b179bc` | 2026-11-05             |

Go **1.27.4** supersedes the selected builder patch and includes security fixes. The official `golang:1.27.4-bookworm` tag currently resolves to `sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea`. [PR #533](https://github.com/jesposito/Facet/pull/533) updates the Go tag+digest pairs; `Facet-zl8` tracks the upgrade.

Refresh each tag and digest together, then run `bash scripts/check-docker-base-pins.sh`, the multi-architecture build, image smoke test and Trivy scan. [PR #528](https://github.com/jesposito/Facet/pull/528), merged as `fde26de`, added an explicit APT upgrade for the CVE-2026-109518 fix; it did not replace these three base digests. Its disposable image check found apt and libapt-pkg6.0 at 2.6.2, and both architecture scans and packaged smoke tests passed. Later release runs published both architectures successfully.

Sources: [`docker/Dockerfile`](../docker/Dockerfile), Docker Hub API for [Go 1.27.3](https://hub.docker.com/v2/repositories/library/golang/tags/1.27.3-bookworm), [Go 1.27.4](https://hub.docker.com/v2/repositories/library/golang/tags/1.27.4-bookworm), [Node](https://hub.docker.com/v2/repositories/library/node/tags/24-alpine) and [Debian](https://hub.docker.com/v2/repositories/library/debian/tags/bookworm-slim), [Debian CVE tracker](https://security-tracker.debian.org/tracker/CVE-2026-109518), [PR #533](https://github.com/jesposito/Facet/pull/533), checked 2026-11-11.
