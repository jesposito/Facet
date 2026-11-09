# Go and development tools

Checked **2026-11-10**.

## Go

Both `backend/go.mod` and `docker/caddy/go.mod` select **Go 1.27.3**. The production backend and Caddy builders use the same `golang:1.27.3-bookworm` digest-pinned base. The development image uses `golang:1.27.3-alpine` and sets `GOFLAGS=-tags=musl` so go-fitz links the correct bundled MuPDF archive.

The official download feed lists **Go 1.27.3** as the newest stable release, with Go 1.26.10 as the other supported line. The selected toolchain is current.

The local host does not have `go` installed. Native backend tests and vulnerability scans therefore were not run during this inventory. CI installs the version from `backend/go.mod`.

## Supporting tools

| Tool        | Selected / host                                    | Upstream status            | Notes                                                                                          |
| ----------- | -------------------------------------------------- | -------------------------- | ---------------------------------------------------------------------------------------------- |
| Air         | 1.67.4 selected; not installed on host             | 1.67.4 latest              | Installed by `Dockerfile.dev`; native setup uses `go install github.com/air-verse/air@v1.67.4` |
| Python      | 3 required; host 3.12.3                            | no project maximum         | Runs the development supervisor and script tests                                               |
| GCC         | required for cgo; host 13.3.0                      | distribution-provided      | go-fitz bundles MuPDF; no separate MuPDF installation is needed                                |
| govulncheck | CI runs `golang.org/x/vuln/cmd/govulncheck@v1.9.0` | not independently compared | Standalone binary absent on host; CI runs it for both Go modules                               |

Sources: [`backend/go.mod`](../backend/go.mod), [`docker/caddy/go.mod`](../docker/caddy/go.mod), [`docker/Dockerfile`](../docker/Dockerfile), [`docker/Dockerfile.dev`](../docker/Dockerfile.dev), [`docs/DEV.md`](../docs/DEV.md), [`.github/workflows/security-tests.yml`](../.github/workflows/security-tests.yml), [official Go download feed](https://go.dev/dl/?mode=json), [Air v1.67.4 release](https://github.com/air-verse/air/releases/tag/v1.67.4), all checked 2026-11-10. Host command results were collected on the same date.
