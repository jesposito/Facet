# Repository architecture and checks

Checked **2026-11-10** against the repository at `f3d7a14`.

## Shape

Facet is a self-hosted, single-owner personal site. The production container exposes Caddy on port 8080, which routes to SvelteKit on 3000 and Go/PocketBase on 8090. Content is stored in SQLite; uploads are stored separately and linked into PocketBase's storage path. Native development runs PocketBase on `127.0.0.1:8090` and Vite on `127.0.0.1:5173`, without Caddy.

| Area                              | Path                  | Role                                                              |
| --------------------------------- | --------------------- | ----------------------------------------------------------------- |
| Backend                           | `backend/`            | Go 1.27.3 and PocketBase 0.40.6 as a framework                    |
| HTTP routes and record hooks      | `backend/hooks/`      | Feature endpoints, access rules and background jobs               |
| Shared backend logic              | `backend/services/`   | AI, email, encryption, GitHub, media, backups and webhooks        |
| Schema                            | `backend/migrations/` | Go migrations applied automatically at startup                    |
| Frontend                          | `frontend/`           | SvelteKit 3, Svelte 5, TypeScript 6 and Tailwind 3                |
| Browser tests                     | `frontend/tests/`     | Playwright, with source-unit, public-api and chromium projects    |
| Container                         | `docker/`             | Multi-stage production build, Caddyfile and process supervisor    |
| Proxy module                      | `docker/caddy/`       | Caddy built from source with the same Go toolchain                |
| Development and CI scripts        | `scripts/`            | Native supervisor, browser smoke runner and validation helpers    |
| Product/contributor documentation | `docs/`               | Existing behaviour and operating guides; not replaced by `.docs/` |

The public application does not expose raw PocketBase collections. Custom backend endpoints apply visibility, draft and share-link rules. Any signed-in `users` account is treated as the owner, with `ADMIN_EMAILS` restricting sign-in when configured.

Sources: [`docs/ARCHITECTURE.md`](../docs/ARCHITECTURE.md), [`backend/main.go`](../backend/main.go), [`docker/start.sh`](../docker/start.sh), [`docker/Caddyfile`](../docker/Caddyfile), all checked 2026-11-10.

## Development commands

| Command                                    | Purpose                                          | Data warning                               |
| ------------------------------------------ | ------------------------------------------------ | ------------------------------------------ |
| `make dev`                                 | Start the supervised native backend and frontend | Preserves `pb_data/`                       |
| `make stop`                                | Stop only this checkout's supervised processes   | No data removal                            |
| `make test`                                | Script regressions, Go tests and `svelte-check`  | Does not run Playwright                    |
| `make lint`                                | `go vet`, Prettier check and ESLint              | Read-only checks                           |
| `cd frontend && npm run i18n:validate`     | Translation parity and placeholder checks        | Read-only checks                           |
| `cd frontend && npm run test:dependencies` | Guarded braces/deepmerge regression tests        | Read-only checks                           |
| `make build`                               | Build the production image as `facet:latest`     | No application data access                 |
| `make seed-dev`                            | Load the development profile                     | **Deletes `pb_data/` and rewrites `.env`** |
| `make seed-clear`                          | Clear development data                           | **Deletes `pb_data/`**                     |

The canonical full browser check is `bash scripts/ci/run-browser-smoke.sh`. It builds a fresh app, uses a temporary database and runs the source-unit, public-api and Chromium suites. Running `npm test` directly requires separately running servers and a disposable database.

Sources: [`Makefile`](../Makefile), [`docs/DEV.md`](../docs/DEV.md), [`frontend/tests/README.md`](../frontend/tests/README.md), all checked 2026-11-10.

## CI entry points

- `.github/workflows/security-tests.yml`: Go tests/race detector/vet, backend and Caddy `govulncheck`, SMTP patch tests, npm audit, guarded-fork tests, Svelte checks, lint, translations, production frontend build and browser smoke tests.
- `.github/workflows/supply-chain.yml`: Go module tidiness, npm lock reproducibility and production Docker digest pins.
- `.github/workflows/docker-publish.yml`: multi-architecture build, Trivy scan, packaged-image smoke test, signing, SBOM and provenance.
- `.github/workflows/published-image-security.yml`: scheduled rescan of the published image.
- `.github/workflows/i18n-check.yml`: translation validation on relevant changes.

Latest observed CI state on `f3d7a14`: [scheduled run 42568077807](https://github.com/jesposito/Facet/actions/runs/42568077807) (2026-11-09) passed frontend and backend tests but failed the Caddy vulnerability step. See `security-advisories.md`.

## Change conventions

- Add schema changes as new migrations; do not loosen collection rules to publish data.
- Add Go tests beside changed backend behaviour and Playwright coverage for changed browser behaviour.
- Update `docs/` with behaviour or configuration changes. Keep this `.docs/` inventory focused on technology versions and upstream references.
- Keep Node, `@types/node`, both Go modules and the Go container build images aligned with their selected majors.
- Production base images require `tag@sha256:digest`; development-only floating bases carry an explicit marker.

Sources: [`CONTRIBUTING.md`](../CONTRIBUTING.md), [`.github/dependabot.yml`](../.github/dependabot.yml), checked 2026-11-10.
