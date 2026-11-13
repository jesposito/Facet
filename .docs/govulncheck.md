# govulncheck

Checked **2026-11-14**.

CI runs `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...` independently in `backend/` and `docker/caddy/`. Version **1.9.0** is the latest release reported by the official Go module proxy.

The proxy step runs `go test ./... github.com/go-jose/go-jose/v3/...` before `govulncheck`, covering the local replacement's regression tests as well as Caddy's own tests. [PR #533](https://github.com/jesposito/Facet/pull/533) merged this command as `25fe89f` in v3.8.8. The vulnerability-scan command is unchanged.

The standalone host binary is absent, so this inventory did not rerun the scans locally. [Security run 43135526868](https://github.com/jesposito/Facet/actions/runs/43135526868) on `795ad77` (`v3.8.10`) passed both the backend and proxy reachable-vulnerability steps. The older Caddy failure through klauspost/compress was fixed by PR #528.

Sources: [`.github/workflows/security-tests.yml`](../.github/workflows/security-tests.yml), [Go module proxy latest](https://proxy.golang.org/golang.org/x/vuln/@latest), [current CI run](https://github.com/jesposito/Facet/actions/runs/43135526868), checked 2026-11-14.
