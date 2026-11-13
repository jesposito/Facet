# PocketBase

Checked **2026-11-14**.

| Component                                  | Locked | Latest | Status                 |
| ------------------------------------------ | ------ | ------ | ---------------------- |
| `github.com/pocketbase/pocketbase`         | 0.40.6 | 0.40.6 | current                |
| `pocketbase` JavaScript SDK                | 0.28.2 | 0.29.0 | minor update available |
| `github.com/pocketbase/dbx`                | 1.12.0 | 1.12.0 | current                |
| `github.com/pocketbase/ozzo-validation/v4` | 4.3.0  | 4.3.0  | current                |

Facet uses PocketBase as a Go framework rather than a separately deployed service. `backend/main.go` creates the app, registers hooks and CLI commands, and runs migrations automatically. Public content is returned through custom endpoints that enforce Facet's visibility rules.

The browser and SvelteKit server share the client in `frontend/src/lib/pocketbase.ts`. New code should use `pb.authStore.record`; that is the SDK API used by the current client. The backend is 0.40 while the JS SDK is 0.28, so do not assume matching version numbers are required or that raw backend collection APIs belong in public frontend code.

SDK 0.29.0 automatically clears `pb.authStore` when a token expires or is externally invalidated, including after a 401 response. `pb.autoClearAuthStore(false)` disables that behaviour. Review the change and test login, expired-token and 401 flows before updating; see `pocketbase-js.md`.

Development serves the backend on port 8090. The Vite proxy forwards `/api` and `/_`; production Caddy controls whether the PocketBase dashboard at `/_/` is available.

Sources: [`backend/go.mod`](../backend/go.mod), [`backend/main.go`](../backend/main.go), [`frontend/package-lock.json`](../frontend/package-lock.json), [`frontend/src/lib/pocketbase.ts`](../frontend/src/lib/pocketbase.ts), [`docs/ARCHITECTURE.md`](../docs/ARCHITECTURE.md), [`docs/DEV.md`](../docs/DEV.md), [Go module proxy: PocketBase](https://proxy.golang.org/github.com/pocketbase/pocketbase/@latest), [Go module proxy: dbx](https://proxy.golang.org/github.com/pocketbase/dbx/@latest), [Go module proxy: ozzo-validation](https://proxy.golang.org/github.com/pocketbase/ozzo-validation/v4/@latest), [npm registry: PocketBase JS](https://registry.npmjs.org/pocketbase/latest), [official 0.29.0 package tarball and changelog](https://registry.npmjs.org/pocketbase/-/pocketbase-0.29.0.tgz), all checked 2026-11-14.
