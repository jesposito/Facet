# Development guide

How to run Facet from source, make changes, and check them. For running Facet as an owner, use the [install guide](SETUP.md) instead. For proposing changes, see [Contributing](../CONTRIBUTING.md).

## Requirements

| Tool | Version | Why |
|------|---------|-----|
| Linux | any recent distribution, or WSL2 | The `make dev` supervisor reads `/proc`, so it only runs on Linux |
| Python | 3 | Runs the development supervisor and its tests |
| Go | 1.27.3, as in [`backend/go.mod`](../backend/go.mod) | Backend |
| Node.js | 24, as in [`.nvmrc`](../.nvmrc). `package.json` accepts `>=24 <25`. | Frontend |
| [Air](https://github.com/air-verse/air) | `go install github.com/air-verse/air@v1.67.4` | Rebuilds the backend when Go files change |
| A C compiler (`gcc`) | | The PDF text extractor used for résumé import ([go-fitz](https://github.com/gen2brain/go-fitz)) is built with cgo. It bundles its own MuPDF library, so you don't need MuPDF installed. |
| GNU coreutils, curl, Make, Bash | | Used by the scripts |
| Pandoc and LaTeX | optional | Only for testing AI résumé PDFs |

On macOS, the individual `make backend` and `make frontend` targets may work with GNU coreutils installed, but `make dev` won't. Linux or WSL2 is the supported path.

## First run

```bash
git clone https://github.com/jesposito/Facet.git
cd Facet
make dev
```

The first run downloads Go modules and npm packages, so it takes a while. Then open **http://127.0.0.1:5173**, go to `/admin`, and sign in with the account and password printed in the backend log. `make seed-dev` sets `SEED_DATA=dev`, which uses the fixed local-only password `changeme123` instead; Facet makes you change it before the admin APIs answer.

You don't need a `.env` file. If one exists, `make dev` loads it first, so values such as `ADMIN_EMAILS` or AI keys apply to your development instance too.

## What `make dev` does

`make dev` runs [`scripts/start-dev.sh`](../scripts/start-dev.sh), which sets development defaults and hands over to the Python supervisor, [`scripts/dev-server.py`](../scripts/dev-server.py):

| Setting | Default in development |
|---------|------------------------|
| `DATA_DIR` | `pb_data/` in the checkout |
| `ENCRYPTION_KEY` | a fixed development-only key |
| `SEED_DATA` | `minimal`: just the first account |

The supervisor then:

1. takes a lock for this checkout, so you can't start two sessions of the same checkout by accident
2. checks that ports 8090 and 5173 are free on `127.0.0.1`, and stops with a message if not
3. starts the backend through [`scripts/dev-backend.sh`](../scripts/dev-backend.sh), which runs `go mod tidy` only when `go.mod` or `go.sum` changed, then Air. Backend output goes to `tmp/backend.log`.
4. waits up to three minutes for `http://127.0.0.1:8090/api/health`
5. starts the frontend through [`scripts/dev-frontend.sh`](../scripts/dev-frontend.sh), which runs `npm ci` only when `package-lock.json` changed, then Vite on `127.0.0.1:5173`. Frontend output stays in your terminal.

Both servers listen on loopback only, so they aren't reachable from other machines. Vite forwards `/api` and `/_` to the backend.

To stop, press Ctrl+C, or run `make stop` from another terminal. `make stop` only stops the session this checkout's supervisor started. It never kills unrelated processes on those ports, and it's safe to run when nothing is running. If something else holds port 8090 or 5173, stop it yourself.

Your data in `pb_data/` survives restarts.

| Address | What |
|---------|------|
| http://127.0.0.1:5173 | The site, with hot reload |
| http://127.0.0.1:5173/admin | The admin dashboard |
| http://127.0.0.1:8090/api/health | Backend health |
| http://127.0.0.1:8090/_/ | PocketBase dashboard (always available in development) |

The PocketBase dashboard needs a superuser, which Facet never creates. Make one from the `backend/` folder:

```bash
cd backend
ENCRYPTION_KEY='dev-only-key-do-not-use-in-production!!' \
  go run . superuser upsert admin@localhost.dev 'a-local-password' --dir=../pb_data
```

Some browser tests expect this superuser ([browser tests](../frontend/tests/README.md#superuser-tests)).

Air deletes the `tmp/` folder, including `tmp/backend.log`, when the session stops. Copy the log first if you need it afterwards.

## Make targets

From the [Makefile](../Makefile):

| Target | What it does | Touches your data? |
|--------|--------------|--------------------|
| `make dev` (or `dev-up`) | Start backend and frontend as above | No |
| `make stop` (or `kill`) | Stop this checkout's `make dev` session | No |
| `make backend` / `make frontend` | Run one side on its own, without the supervisor | No |
| `make dev-reset` | Clear the dependency hashes and `.svelte-kit`, so the next `make dev` reinstalls | **No**, `pb_data/` is kept |
| `make seed-dev` | Stop, **delete `pb_data/`**, ask about sign-in options, **overwrite `.env`**, and start with the development profile | **Yes, destructive** |
| `make seed-clear` | Stop and **delete `pb_data/`** | **Yes, destructive** |
| `make backup` | Archive a stopped development `pb_data/` (and `uploads/` if present) into `backups/` | Reads only |
| `make test` | Script regressions, Go tests, Svelte type checks | No |
| `make test-scripts` / `test-backend` / `test-frontend` | One part of `make test` | No |
| `make lint` | `go vet`, then Prettier check and ESLint | No |
| `make fmt` | `gofmt -w` and Prettier write | Rewrites source files |
| `make clean` | Remove build output. Refuses while `make dev` is running. | No |
| `make deps` | Download Go modules and install npm packages | No |
| `make gen-key` | Print a new random encryption key | No |
| `make build` (or `docker-build`) | Build the production image as `facet:latest` | No |
| `make docker-run` | Run that image with `./data` and `./uploads` (needs `ENCRYPTION_KEY` set) | Uses `./data` |
| `make prod` / `prod-logs` / `prod-down` | Production Compose up, logs, down | Uses Compose volumes |
| `make dev-docker` / `dev-down` / `dev-logs` | Docker-based development (see below) | Uses `pb_data/` |

`make seed-dev` replaces your whole `.env` with the handful of values it asks about. Copy anything else you need out of `.env` first.

## Seed data

| `SEED_DATA` | Result on an empty database |
|-------------|-----------------------------|
| `minimal` or unset | One account: the first `ADMIN_EMAILS` address, then `DEV_ADMIN_EMAIL`, then `admin@example.com`, with a random password printed once in the log |
| `dev` | The same account with the fixed password `changeme123`, plus the maintainer's own profile, used for development and screenshots. Never use it on a reachable instance. |

Seeding only runs when the database has no accounts, so changing `SEED_DATA` on an existing `pb_data/` does nothing. Demo data is separate and loaded from the admin header toggle ([Features](FEATURES.md#demo-mode)).

## Docker-based development

`make dev-docker` runs [`docker-compose.dev.yml`](../docker-compose.dev.yml): a Go container with Air for the backend and a Node container for Vite, sharing `pb_data/` with native development. It defaults `SEED_DATA` to `dev`.

Inside the containers, both services set `FACET_DEV_HOST=0.0.0.0` so their ports can be published. Compose publishes them on your machine's loopback only, at `127.0.0.1:8090` and `127.0.0.1:5173`. The two containers sit on Compose's default network, and the frontend sets `POCKETBASE_URL=http://backend:8090`. Vite's `/api` and `/_` proxy and SvelteKit's server code both use that address. Native development leaves `POCKETBASE_URL` unset, so the proxy uses `http://127.0.0.1:8090`. `POCKETBASE_URL` has no `VITE_` prefix, so it stays on the server and never reaches the browser bundle. Vite refuses to start if `POCKETBASE_URL` isn't a plain `http(s)` origin, such as one with a path or credentials in it.

## Tests and checks

| Command | Covers |
|---------|--------|
| `make test-scripts` | [`scripts/ci/test-dev-server.py`](../scripts/ci/test-dev-server.py) (supervisor safety) and [`scripts/ci/test-push-validated-release.py`](../scripts/ci/test-push-validated-release.py) (release script) |
| `make test-backend` | `go test -v ./...` in `backend/` |
| `make test-frontend` | `npm run check`: SvelteKit sync and `svelte-check` type checking |
| `make lint` | `go vet ./...`, `prettier --check .` and `eslint .` in `frontend/` |
| `cd frontend && npm run i18n:validate` | Translation key parity and parameters |
| `cd frontend && npm run test:dependencies` | Tests for the vendored `braces` and `deepmerge` forks |

`make test` doesn't run the browser tests. Those need running servers and are described in [frontend/tests/README.md](../frontend/tests/README.md). CI runs everything, including browser smoke tests against a freshly built app ([Contributing](../CONTRIBUTING.md#what-ci-checks)).

## Where things are

```text
backend/            Go + PocketBase
  main.go           startup, hook registration, CLI commands
  hooks/            HTTP routes and record hooks, one file per feature
  services/         shared logic (crypto, AI, email, storage, webhooks, ...)
  migrations/       schema changes, applied automatically
frontend/           SvelteKit app
  src/routes/       public pages and /admin
  src/components/   public/, admin/, shared/
  src/lib/          client, stores, i18n, view editor, experiences
  src/locales/      translations
  tests/            Playwright browser tests
docker/             production Dockerfile, Caddyfile, start script, Caddy build
scripts/            dev scripts, CI scripts, i18n validators
docs/               these pages
```

[Architecture](ARCHITECTURE.md) explains how the pieces connect.

## Adding a migration

Migrations live in [`backend/migrations`](../backend/migrations) as Go files named `<unix timestamp>_<description>.go`. They run automatically on startup in both development and production, so:

- Make each one safe to run against existing data, and idempotent where you can (check whether a field exists before adding it).
- Give it a working down function where practical. Facet never runs down migrations automatically, so owners rely on backups to roll back.
- Lock new collections' access rules. Direct collection access should need sign-in, or be backend-only, with public data served through custom endpoints.
- Think about demo mode. If the collection holds content that views display, it probably needs a `demo_` twin.

## PocketBase notes

These have caught contributors before.

**Record hooks must call `e.Next()`.** In `BindFunc` handlers, return `e.Next()` to continue the chain. Returning `nil` silently stops the operation.

```go
app.OnRecordCreate("my_collection").BindFunc(func(e *core.RecordEvent) error {
    if err := validate(e.Record); err != nil {
        return err // abort with an error
    }
    return e.Next() // continue
})
```

**Hidden fields aren't populated in record events.** Fields marked `Hidden: true` aren't available in `OnRecordCreate`/`OnRecordUpdate`. Use the `...Request` hooks and read the request body, as [`hooks/ai.go`](../backend/hooks/ai.go) does to encrypt provider keys.

**Writes from some request contexts.** Several places in Facet use direct SQL inserts with `app.DB().NewQuery(...)` because `app.Save()` didn't persist records when called from certain `GET` handlers and `OnServe` hooks (see the comments in [`hooks/seed.go`](../backend/hooks/seed.go) and [`hooks/ai.go`](../backend/hooks/ai.go)). If a save seems to succeed but nothing is stored, check the context it runs in.

**Collection rules don't apply to server code.** `app.FindRecordsByFilter()` and friends run with full authority. Any endpoint that returns data to visitors must apply visibility and draft rules itself.

**JS SDK.** The frontend uses PocketBase JS SDK 0.28 with a 0.40 backend. Some code still reads `pb.authStore.model`, the older name for `pb.authStore.record`. Use `record` in new code.

## SvelteKit notes

**Links back to `/` from the posts and talks index pages** use `data-sveltekit-reload`, forcing a full page load. Client-side navigation to the root page has failed there in the past. If you find the cause, remove the workaround and update this note.

**Static files beat routes.** A file in `frontend/static/` is served instead of a route at the same path. `robots.txt` is a route ([`src/routes/robots.txt/+server.ts`](../frontend/src/routes/robots.txt/+server.ts)) so it can include your sitemap address; adding a `static/robots.txt` would silently replace it.

## AI writing assistant component

[`AIContentHelper.svelte`](../frontend/src/components/admin/AIContentHelper.svelte) adds the rewrite and critique menu to a field. It hides itself when no AI provider is available.

```svelte
<AIContentHelper
  fieldType="description"
  content={description}
  context={{ role: title, company, location }}
  on:apply={(e) => (description = e.detail.content)}
/>
```

| Prop | Type | Purpose |
|------|------|---------|
| `content` | string | Current field text |
| `fieldType` | string | What the text is, such as `description`, `bullets`, `summary` or `headline`. Shapes the prompt. |
| `context` | object of strings | Related form values to give the model context |
| `disabled` | boolean | Disable the button |
| `size` | `'sm'` or `'md'` | Button size |

It emits `apply` with `{ content }` when the owner accepts a suggestion. Prompts live in [`hooks/ai.go`](../backend/hooks/ai.go). Keep the plain-writing rules in them, and never let a prompt invite the model to invent facts.

## Debugging tips

- Backend logs during `make dev`: `tail -f tmp/backend.log`.
- Air's build errors: `tmp/build-errors.log`.
- Logging: use `app.Logger()` with structured fields, and log enough context on errors to diagnose them later. Avoid logging secrets, tokens or personal data.
- A fresh database: `make stop`, then move `pb_data/` aside (`mv pb_data pb_data.old`) and `make dev`. Moving rather than deleting means you can go back.
