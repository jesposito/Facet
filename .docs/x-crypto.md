# golang.org/x/crypto

Checked **2026-11-12**.

The backend selects pseudo-version **v0.58.1-0.20261009181718-2f1f834330b2**. The Go proxy's `@latest` endpoint now reports tagged **v0.59.0**, published 2026-11-11. Status: **minor update available**. The v0.59.0 module manifest requires Go 1.26.0, so Facet's Go 1.27.4 toolchain satisfies that requirement.

The intervening official log removes SSH DSA support and the separate openpgp module, updates the fallback root bundle to remove roots that issued compromised intermediates, and includes SSH hardening. Facet's direct imports are limited to `bcrypt` and `pbkdf2`, but Caddy also carries transitive module paths. Update deliberately and run backend and proxy tests plus reachable `govulncheck`; the release log alone does not establish that every changed path is reachable in Facet.

Sources: [`backend/go.mod`](../backend/go.mod), [Go module proxy latest](https://proxy.golang.org/golang.org/x/crypto/@latest), [v0.59.0 module manifest](https://proxy.golang.org/golang.org/x/crypto/@v/v0.59.0.mod), [official revision log](https://go.googlesource.com/crypto/+log/2f1f834330b2..v0.59.0), checked 2026-11-12.
