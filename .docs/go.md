# Go and development tools

Checked **2026-11-11**.

## Go

On `main` at `07de104`, both `backend/go.mod` and `docker/caddy/go.mod` select **Go 1.27.3**. The production backend and Caddy builders use the same `golang:1.27.3-bookworm` digest-pinned base. The development image uses `golang:1.27.3-alpine` and sets `GOFLAGS=-tags=musl` so go-fitz links the correct bundled MuPDF archive.

The official download feed lists **Go 1.27.4**, released 2026-11-10, as the newest stable release, with Go 1.26.11 as the other supported line. Go 1.27.4 includes security fixes to cgo, the Go command, compiler, linker and `net/http`, plus runtime and crypto bug fixes. The release note does not establish that every fixed path is reachable in Facet.

[PR #533](https://github.com/jesposito/Facet/pull/533) updates both modules and the development and production builders to Go 1.27.4. The official `golang:1.27.4-bookworm` image is available at multi-architecture digest `sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea`; `golang:1.27.4-alpine` is also available. `Facet-zl8` tracks the upgrade through merge and verification.

The local host does not have `go` installed. Native backend tests and vulnerability scans therefore were not run during this inventory. CI installs the version from `backend/go.mod`.

## Supporting tools

| Tool        | Selected / host                                    | Upstream status       | Notes                                                                                          |
| ----------- | -------------------------------------------------- | --------------------- | ---------------------------------------------------------------------------------------------- |
| Air         | 1.67.4 selected; not installed on host             | 1.67.4 latest         | Installed by `Dockerfile.dev`; native setup uses `go install github.com/air-verse/air@v1.67.4` |
| Python      | 3 required; host 3.12.3                            | no project maximum    | Runs the development supervisor and script tests                                               |
| GCC         | required for cgo; host 13.3.0                      | distribution-provided | go-fitz bundles MuPDF; no separate MuPDF installation is needed                                |
| govulncheck | CI runs `golang.org/x/vuln/cmd/govulncheck@v1.9.0` | 1.9.0 latest          | Standalone binary absent on host; CI runs it for both Go modules                               |

Sources: [`backend/go.mod`](../backend/go.mod), [`docker/caddy/go.mod`](../docker/caddy/go.mod), [`docker/Dockerfile`](../docker/Dockerfile), [`docker/Dockerfile.dev`](../docker/Dockerfile.dev), [`docs/DEV.md`](../docs/DEV.md), [`.github/workflows/security-tests.yml`](../.github/workflows/security-tests.yml), [official Go download feed](https://go.dev/dl/?mode=json), [Go release history](https://go.dev/doc/devel/release), [Go 1.27.4 Bookworm image](https://hub.docker.com/v2/repositories/library/golang/tags/1.27.4-bookworm), [PR #533](https://github.com/jesposito/Facet/pull/533), [Air module latest](https://proxy.golang.org/github.com/air-verse/air/@latest), [govulncheck module latest](https://proxy.golang.org/golang.org/x/vuln/@latest), all checked 2026-11-11. Host command results were collected on the same date.
