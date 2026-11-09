# golang-jwt

Checked **2026-11-10**.

Facet directly requires `github.com/golang-jwt/jwt/v4` **v4.5.2**, the latest v4 release. The separate v5 module path is at **v5.3.1** and already appears indirectly in the backend graph. Keep import paths and API changes explicit during any v5 migration; do not treat it as an in-place patch.

Sources: [`backend/go.mod`](../backend/go.mod), [Go proxy v4 latest](https://proxy.golang.org/github.com/golang-jwt/jwt/v4/@latest), [Go proxy v5 latest](https://proxy.golang.org/github.com/golang-jwt/jwt/v5/@latest), checked 2026-11-10.
