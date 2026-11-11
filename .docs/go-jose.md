# go-jose

Checked **2026-11-11**.

Caddy's graph selects `github.com/go-jose/go-jose/v3` **v3.0.5**. On current `main`, Facet's local replacement backports RSA bounds checks for **GHSA-mgcc-cww5-fp2q**, fixed upstream in v4.1.6.

[PR #533](https://github.com/jesposito/Facet/pull/533) expands the replacement to cover the other two v3 denial-of-service advisories:

- **GHSA-4w3g-jfj2-cw2q** (CBC-HMAC empty-ciphertext panic), backported from upstream commit [`74114c2`](https://github.com/go-jose/go-jose/commit/74114c2fa22669a2b2a296746eaa82e15566fd54), fixed on the v4 line in 4.1.5;
- **GHSA-vvm8-gf78-28g4** (excessive JSON nesting), backported from upstream commit [`a31af25`](https://github.com/go-jose/go-jose/commit/a31af25836222f4c45d366d6e8f9b1049dd9935a), also fixed in v4.1.5.

The new CBC-HMAC and JSON-depth regression tests each fail against unpatched v3.0.5. PR #533 also changes the CI proxy step to run `go test ./... github.com/go-jose/go-jose/v3/...`, so the replacement's own tests run alongside Caddy's tests. Once that PR merges, the local replacement covers all three advisories.

OSV will continue to match the upstream **v3.0.5** package version even with all three fixes backported. That package/version match cannot inspect Facet's `replace` target or prove reachability. Facet's default `docker/Caddyfile` serves plain HTTP on `:8080` and enables neither `acme_server` nor `tls internal`; the affected code remains compiled into Caddy, but these PKI paths are not reached by the default configuration. Self-hosted configurations can enable them.

Latest go-jose/v4 is **4.1.9**. Keep the replacement while the graph still contains v3, and review all three advisories during the smallstep migration.

Sources: [PR #533](https://github.com/jesposito/Facet/pull/533), [`docker/caddy/third_party/go-jose-v3/FACET-PATCH.md`](../docker/caddy/third_party/go-jose-v3/FACET-PATCH.md), [`docker/Caddyfile`](../docker/Caddyfile), [RSA advisory](https://github.com/advisories/GHSA-mgcc-cww5-fp2q), [CBC-HMAC advisory](https://github.com/advisories/GHSA-4w3g-jfj2-cw2q), [deep-JSON advisory](https://github.com/advisories/GHSA-vvm8-gf78-28g4), [Go proxy v4 latest](https://proxy.golang.org/github.com/go-jose/go-jose/v4/@latest), code and tests inspected from PR #533 on 2026-11-11.
