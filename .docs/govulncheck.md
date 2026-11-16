# govulncheck

Checked **2026-11-17**.

CI runs `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...` independently in `backend/` and `docker/caddy/`. Version **1.9.0** is the latest release reported by the official Go module proxy.

The proxy step runs `go test ./... github.com/go-jose/go-jose/v3/...` before `govulncheck`, covering the local replacement's regression tests as well as Caddy's own tests. [PR #533](https://github.com/jesposito/Facet/pull/533) merged this command as `25fe89f` in v3.8.8. The vulnerability-scan command is unchanged.

The standalone host binary and native Go toolchain are absent. For the combined OpenTelemetry 1.47.0 update, the watcher ran the same proxy `govulncheck` command in the pinned Go 1.27.4 Bookworm container; it reported **No vulnerabilities found**. [Main security run 43968105029](https://github.com/jesposito/Facet/actions/runs/43968105029) on `4cffe0e` subsequently passed both backend and proxy reachable-vulnerability steps after that update.

Sources: [`.github/workflows/security-tests.yml`](../.github/workflows/security-tests.yml), [Go module proxy latest](https://proxy.golang.org/golang.org/x/vuln/@latest), [current CI run](https://github.com/jesposito/Facet/actions/runs/43968105029), [`docker/caddy/go.mod`](../docker/caddy/go.mod), checked 2026-11-17.
