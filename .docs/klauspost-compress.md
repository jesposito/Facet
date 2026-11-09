# klauspost/compress

Checked **2026-11-10**.

The Caddy graph selects **v1.20.1**. GHSA-fpmh-frq8-7rj8 / GO-2026-6865 covers a crafted zstd frame that can panic decoding. The **v1.20.2** release explicitly fixes that advisory; **v1.20.4** is latest.

Scheduled Facet CI reported reachable `huff0` and `fse` call paths, so this was an action item rather than an unreviewed package match. Caddy 2.11.9 still selects 1.20.1, meaning the Caddy bump alone did not fix it. [PR #528](https://github.com/jesposito/Facet/pull/528), merged as `fde26de`, explicitly selects the minimum fixed 1.20.2 release. Local proxy `govulncheck`, full security CI and both architecture image scans passed; `Facet-cyi` is closed.

Sources: [`docker/caddy/go.mod`](../docker/caddy/go.mod), [security advisory](https://github.com/klauspost/compress/security/advisories/GHSA-fpmh-frq8-7rj8), [v1.20.2 release](https://github.com/klauspost/compress/releases/tag/v1.20.2), [Go module proxy latest](https://proxy.golang.org/github.com/klauspost/compress/@latest), [failing CI run](https://github.com/jesposito/Facet/actions/runs/42568077807), checked 2026-11-10.
