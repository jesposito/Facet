# go-jose

Checked **2026-11-11**.

Caddy's graph selects `github.com/go-jose/go-jose/v3` **v3.0.5**. Facet's local replacement carries backports for all three known v3 denial-of-service advisories:

- **GHSA-mgcc-cww5-fp2q** (RSA JWK CPU denial of service), backported from upstream commit [`a0111f8`](https://github.com/go-jose/go-jose/commit/a0111f82742fa92bceec9ede40845e73319e6230), fixed on the v4 line in 4.1.6;
- **GHSA-4w3g-jfj2-cw2q** (CBC-HMAC empty-ciphertext panic), backported from upstream commit [`74114c2`](https://github.com/go-jose/go-jose/commit/74114c2fa22669a2b2a296746eaa82e15566fd54), fixed in v4.1.5;
- **GHSA-vvm8-gf78-28g4** (excessive JSON nesting), backported from upstream commit [`a31af25`](https://github.com/go-jose/go-jose/commit/a31af25836222f4c45d366d6e8f9b1049dd9935a), also fixed in v4.1.5.

[PR #533](https://github.com/jesposito/Facet/pull/533) merged as **`25fe89f`** and was released in **v3.8.8**. The CBC-HMAC and JSON-depth regression tests each fail against unpatched v3.0.5. The CI proxy step runs `go test ./... github.com/go-jose/go-jose/v3/...`, so the replacement's own tests run alongside Caddy's tests before `govulncheck`.

OSV continues to match the upstream **v3.0.5** package version even though all three fixes are backported. That package/version match cannot inspect Facet's `replace` target or prove reachability. Facet's default `docker/Caddyfile` serves plain HTTP on `:8080` and enables neither `acme_server` nor `tls internal`; the affected code remains compiled into Caddy, but these PKI paths are not reached by the default configuration. Self-hosted configurations can enable them.

Latest go-jose/v4 is **4.1.9**. Keep the replacement while the graph still contains v3, and review all three advisories during the smallstep migration.

Sources: [PR #533](https://github.com/jesposito/Facet/pull/533), [`docker/caddy/third_party/go-jose-v3/FACET-PATCH.md`](../docker/caddy/third_party/go-jose-v3/FACET-PATCH.md), [`docker/Caddyfile`](../docker/Caddyfile), [`.github/workflows/security-tests.yml`](../.github/workflows/security-tests.yml), [RSA advisory](https://github.com/advisories/GHSA-mgcc-cww5-fp2q), [CBC-HMAC advisory](https://github.com/advisories/GHSA-4w3g-jfj2-cw2q), [deep-JSON advisory](https://github.com/advisories/GHSA-vvm8-gf78-28g4), [Go proxy v4 latest](https://proxy.golang.org/github.com/go-jose/go-jose/v4/@latest), merged code and tests inspected on 2026-11-11.
