# go-jose

Checked **2026-11-10**.

Caddy's graph selects `github.com/go-jose/go-jose/v3` **v3.0.5**. Facet's local replacement backports RSA bounds checks for **GHSA-mgcc-cww5-fp2q**, fixed upstream in v4.1.6.

Two other v3 denial-of-service advisories are not covered by that patch: **GHSA-4w3g-jfj2-cw2q** and **GHSA-vvm8-gf78-28g4**, both fixed on the v4 line in 4.1.5. Do not describe the local replacement as a complete v3 security fix. Latest go-jose/v4 is **4.1.9**. Keep the replacement while the graph still contains v3, and review all three advisories during the smallstep migration.

Sources: [`docker/caddy/third_party/go-jose-v3/FACET-PATCH.md`](../docker/caddy/third_party/go-jose-v3/FACET-PATCH.md), [RSA advisory](https://github.com/advisories/GHSA-mgcc-cww5-fp2q), [CBC-HMAC advisory](https://github.com/advisories/GHSA-4w3g-jfj2-cw2q), [deep-JSON advisory](https://github.com/advisories/GHSA-vvm8-gf78-28g4), [Go proxy v4 latest](https://proxy.golang.org/github.com/go-jose/go-jose/v4/@latest), checked 2026-11-10.
