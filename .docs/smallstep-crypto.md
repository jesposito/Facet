# smallstep/crypto

Checked **2026-11-10**.

The Caddy module selects `go.step.sm/crypto` **v0.91.0**. Latest is **v0.92.0**, but it is not a compatible drop-in update for the current graph.

A 2026-11-10 compatibility build with smallstep/certificates 0.30.2 failed because `jwt.Builder.CompactSerialize` is undefined and `jose.Expected.Audience` is missing. Version 0.92.0 selects go-jose/v4 4.1.9, while Caddy 2.11.9 and certificates 0.30.2 still select go-jose/v3. Keep 0.91.0 and Facet's local v3 patch until certificates supports the new API. Go 1.26 or newer is required by 0.92.0; Facet's Go 1.27.3 satisfies that part.

Sources: [`docker/caddy/go.mod`](../docker/caddy/go.mod), [0.92.0 release](https://github.com/smallstep/crypto/releases/tag/v0.92.0), [0.92.0 go.mod](https://proxy.golang.org/go.step.sm/crypto/@v/v0.92.0.mod), [Caddy 2.11.9 go.mod](https://proxy.golang.org/github.com/caddyserver/caddy/v2/@v/v2.11.9.mod), compatibility build result reported in `Facet-cyi` work on 2026-11-10.
