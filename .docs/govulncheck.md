# govulncheck

Checked **2026-11-10**.

CI runs `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...` independently in `backend/` and `docker/caddy/`. Version **1.9.0** is the current official release identified by the upstream research pass.

The standalone host binary is absent, so this inventory did not rerun the scans locally. Latest scheduled CI passed the backend scan and failed the Caddy scan through reachable klauspost/compress paths.

Sources: [`.github/workflows/security-tests.yml`](../.github/workflows/security-tests.yml), [official v1.9.0 release](https://github.com/golang/vuln/releases/tag/v1.9.0), [CI run 42568077807](https://github.com/jesposito/Facet/actions/runs/42568077807), checked 2026-11-10.
