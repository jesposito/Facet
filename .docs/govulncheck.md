# govulncheck

Checked **2026-11-11**.

CI runs `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...` independently in `backend/` and `docker/caddy/`. Version **1.9.0** is the latest release reported by the official Go module proxy.

The standalone host binary is absent, so this inventory did not rerun the scans locally. [Security run 42754669718](https://github.com/jesposito/Facet/actions/runs/42754669718) on `07de104` passed both the backend and proxy reachable-vulnerability steps. The older Caddy failure through klauspost/compress was fixed by PR #528.

Sources: [`.github/workflows/security-tests.yml`](../.github/workflows/security-tests.yml), [Go module proxy latest](https://proxy.golang.org/golang.org/x/vuln/@latest), [current CI run](https://github.com/jesposito/Facet/actions/runs/42754669718), checked 2026-11-11.
