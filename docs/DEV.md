# Facet Development Guide

This guide covers development setup for Facet, a PocketBase + SvelteKit application.

## Quick Start (Codespaces)

1. Click "Open in Codespaces" from GitHub
2. Wait for the devcontainer to build (~2 min first time, ~15s thereafter)
3. Services start automatically via `postStartCommand`
4. Open the forwarded ports when prompted:
   - **Frontend**: http://localhost:5173
   - **PocketBase Admin**: http://localhost:8090/_/

**Default credentials** (dev only, auto-created on first run):

| Admin Panel | URL | Email | Password |
|-------------|-----|-------|----------|
| Facet Admin | http://localhost:5173/admin | `admin@example.com` | `changeme123` |
| PocketBase Admin | http://localhost:8090/_/ | `admin@localhost.dev` | `admin123` |

If credentials don't work, reset the database: `rm -rf pb_data && ./scripts/start-dev.sh`

## Quick Start (Local)

### Prerequisites

- Go 1.23+
- Node.js 20+
- [Air](https://github.com/air-verse/air) for Go hot reload

```bash
# Install air
go install github.com/air-verse/air@v1.61.7
```

### Running Locally

```bash
# Start everything with hot reload
make dev

# Or start services individually
make backend   # Start backend with air
make frontend  # Start frontend with Vite HMR
```

### Using Docker Compose

```bash
# Start development environment
make dev-docker

# View logs
make dev-logs

# Stop
make dev-down
```

## Ports and URLs

| Service | Port | URL | Description |
|---------|------|-----|-------------|
| Frontend | 5173 | http://localhost:5173 | SvelteKit dev server with HMR |
| Backend API | 8090 | http://localhost:8090 | PocketBase API |
| PB Admin | 8090 | http://localhost:8090/_/ | PocketBase admin UI |

## Project Structure

```
Facet/
├── backend/           # Go + PocketBase hooks
│   ├── hooks/         # PocketBase event hooks
│   ├── services/      # Business logic
│   ├── migrations/    # Database migrations
│   └── main.go        # Entry point
├── frontend/          # SvelteKit application
│   ├── src/
│   │   ├── routes/    # SvelteKit routes
│   │   ├── params/    # Param matchers (slug validation)
│   │   └── lib/       # Shared components
│   └── package.json
├── scripts/           # Development scripts
│   ├── start-dev.sh   # Start all services
│   ├── dev-backend.sh # Backend with caching
│   └── dev-frontend.sh# Frontend with caching
├── docker/            # Docker configurations
├── pb_data/           # PocketBase data (gitignored)
└── docs/              # Documentation
```

## Development Best Practices

### Verbose Logging for In-Development Features

**REQUIREMENT**: All new features MUST include verbose debug logging during development.

When implementing new features, especially those involving:
- API integrations (AI providers, external services)
- Data transformations (encryption, parsing, serialization)
- Multi-step workflows (import pipelines, export generation)
- PocketBase hooks and middleware

Add logging at each step of the flow:

```go
// ✅ GOOD - Verbose logging during development
func processData(input string) (string, error) {
    log.Println("[DEBUG] processData called with input len:", len(input))

    transformed, err := transform(input)
    if err != nil {
        log.Println("[ERROR] transform failed:", err)
        return "", err
    }
    log.Println("[DEBUG] transform succeeded, output len:", len(transformed))

    result, err := validate(transformed)
    if err != nil {
        log.Println("[ERROR] validate failed:", err)
        return "", err
    }
    log.Println("[DEBUG] validate succeeded")

    return result, nil
}
```

**Why this matters**: Silent failures are the hardest bugs to track. During the AI integration work, we spent hours debugging issues that would have been immediately obvious with proper logging:

- API keys not being received (PocketBase hidden field issue)
- JSON parsing failures (AI returning arrays instead of strings)
- Hook chain not continuing (`e.Next()` missing)

**When to remove**: After a feature is stable and well-tested, verbose logging can be reduced. Keep error logging permanently.

### Feature Development Checklist

Before marking a feature complete:

- [ ] Verbose logging added at each step of the flow
- [ ] Error cases logged with context (input values, state)
- [ ] Tested with real data (not just happy path)
- [ ] Edge cases documented (e.g., "AI may return array or string")
- [ ] Troubleshooting notes added to DEV.md if gotchas discovered

## URL Routing Model

Facet uses a LinkedIn-style URL structure for public profiles:

### Public Routes

| Route | Purpose | Example |
|-------|---------|---------|
| `/` | Default profile view | Homepage |
| `/<slug>` | Named view | `/recruiter`, `/investor` |
| `/s/<token>` | Share link entry | Validates token, sets cookie, redirects to `/<slug>` |
| `/v/<slug>` | Legacy route | 301 redirects to `/<slug>` |

### Default View

The homepage (`/`) renders the "default view", determined by:

1. A view with `is_default=true` AND `is_active=true` AND `visibility='public'`
2. Fallback: The first public active view (by creation date)
3. Fallback: Legacy homepage aggregation (all public content)

Only one view can be marked as default at a time (enforced by backend hook).

### Reserved Slugs

These slugs are protected and cannot be used for views:

```
admin, api, s, v, projects, posts, talks,
_app, _, assets, static,
favicon.ico, robots.txt, sitemap.xml,
health, healthz, ready,
login, logout, auth, oauth, callback,
home, index, default, profile
```

Protection is enforced at:
- **Frontend**: `src/params/slug.ts` param matcher
- **Backend**: Views collection create/update hooks

### Share Link Flow

```
1. User receives: /s/<token>
2. Server validates token (POST /api/share/validate)
3. Sets httpOnly cookie (me_share_token, SameSite=Lax)
4. 302 redirect to /<slug>
5. Token NOT in final URL (security: no history/referer leakage)
```

## Development Workflow

### Hot Reload

Both frontend and backend support hot reload:

- **Frontend**: Vite HMR automatically refreshes on `.svelte`, `.ts`, `.css` changes
- **Backend**: Air watches `.go` files and rebuilds automatically

### Optimized Startup

The dev scripts use **lockfile hash caching** to skip unnecessary installs:

```bash
# First run: installs dependencies, saves hash
[frontend] Installing dependencies (node_modules missing)...

# Subsequent runs: skips install if lockfile unchanged
[frontend] Dependencies up to date (skipping npm install)
```

To force a fresh install:
```bash
make dev-reset  # Clears all caches
make dev        # Reinstalls everything
```

## Common Tasks

### Running Tests

```bash
make test           # All tests
make test-backend   # Go tests only
make test-frontend  # SvelteKit checks only
```

### Linting and Formatting

```bash
make lint  # Run linters
make fmt   # Format code
```

### Building for Production

```bash
make build  # Build Docker image
```

## Codespaces Networking Limitations

Some Codespaces environments have network restrictions that block access to `storage.googleapis.com`, which is used by the default Go module proxy (`proxy.golang.org`).

**Symptoms:**
- `go mod tidy` times out with DNS lookup errors for `storage.googleapis.com`
- `go build` fails with "missing go.sum entry" errors
- Downloads hang indefinitely

**Solution (already configured):**

The devcontainer is configured to use `goproxy.cn` as a fallback proxy:

```bash
# Set in devcontainer.json containerEnv
GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
GOSUMDB=off
```

If you're running outside the devcontainer and encounter these issues:

```bash
# Set environment variables
export GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
export GOSUMDB=off

# Then run your go commands
go mod tidy
go build ./...
```

**Alternative: Vendor dependencies (offline-first)**

For truly offline development, you can vendor all dependencies:

```bash
# In backend/
go mod vendor

# Build with vendored deps
go build -mod=vendor ./...
```

Note: Vendoring adds ~50MB to the repository but guarantees zero network dependencies after clone.

## Troubleshooting

### "air: command not found"

Install air globally:
```bash
go install github.com/air-verse/air@v1.61.7
```

### "go: cannot find main module"

This occurs when air runs from the wrong directory. The root `.air.toml` is configured to handle this. Ensure you're running from the project root:
```bash
cd /path/to/Facet
make dev
```

### Port Already in Use

Stop any existing services:
```bash
# Find process using port
lsof -i :8090
lsof -i :5173

# Kill it
kill <PID>

# Or use make
make dev-down
```

### Slow Startup in Codespaces

If startup is slow (>30s), check:

1. **Named volumes exist**: The devcontainer uses named volumes for node_modules and Go modules
2. **Hash files are valid**: Check `frontend/node_modules/.lockfile-hash` and `backend/.gomod-hash`
3. **Force reset**: `make dev-reset && make dev`

### File Watching Not Working

In Codespaces, file watching may need polling. The devcontainer is configured with appropriate `files.watcherExclude` settings. If issues persist:

1. Check that `node_modules`, `pb_data`, and `.git` are excluded from watching
2. Try restarting the terminal

### Database Issues

Reset the database:
```bash
rm -rf pb_data
make dev  # Will recreate with seed data
```

### PocketBase `app.Save()` Silently Fails in GET Handlers

**Symptoms:**
- `app.Save(record)` returns `nil` (success) but record is not persisted
- Record gets an ID assigned, suggesting save worked
- But database shows no new row (check with SQL logs or direct query)
- Only happens in GET request handlers, works fine in POST handlers

**Cause:**
PocketBase v0.23+ has internal transaction/context behavior that prevents writes during GET requests. The `app.Save()` call appears to succeed but silently skips the actual INSERT.

**Solution:**
Use direct SQL inserts via `app.DB().NewQuery()`:

```go
import "github.com/pocketbase/dbx"

// Instead of app.Save(record), use:
query := `INSERT INTO my_table (id, name, value)
          VALUES ({:id}, {:name}, {:value})`

_, err := app.DB().NewQuery(query).Bind(dbx.Params{
    "id":    "unique-id",
    "name":  "example",
    "value": 123,
}).Execute()
```

**Important notes:**
- Don't include `created` or `updated` columns - PocketBase manages these automatically
- Use `dbx.Params{}` map for parameter binding (not variadic args)
- This workaround is used in `backend/hooks/ai.go` for AI provider auto-configuration

**Alternative approaches that DON'T work:**
- `app.RunInTransaction()` - same silent failure
- `app.SaveNoValidate()` - same behavior
- Goroutine with delay - race conditions
- `OnBootstrap` hook - different context, still fails

See: `backend/hooks/ai.go:240-280` for a working implementation.

### PocketBase Record Hooks Must Call `e.Next()`

**Symptoms:**
- Record create/update appears to succeed (API returns 200 with record data)
- But the record is not actually persisted to the database
- Listing records shows empty results
- Getting record by ID returns 404

**Cause:**
In PocketBase v0.23+, record hooks using `BindFunc` must explicitly call `e.Next()` to continue the hook chain and complete the operation. Without this call, the record modification process silently stops.

**Solution:**
Always return `e.Next()` at the end of your hook handlers:

```go
// ✅ CORRECT - calls e.Next() to continue
app.OnRecordCreate("my_collection").BindFunc(func(e *core.RecordEvent) error {
    // Your logic here
    if err := doSomething(e.Record); err != nil {
        return err  // Return error to abort
    }
    return e.Next()  // REQUIRED: continue the hook chain
})

// ❌ WRONG - returns nil instead of e.Next()
app.OnRecordCreate("my_collection").BindFunc(func(e *core.RecordEvent) error {
    doSomething(e.Record)
    return nil  // This silently aborts the save!
})
```

**Applies to these hook types:**
- `OnRecordCreate`
- `OnRecordUpdate`
- `OnRecordDelete`
- `OnRecordAuthWithPasswordRequest`
- `OnRecordAuthWithOAuth2Request`
- Any other `BindFunc` hooks

**Note:** Router hooks (like `OnServe`) also need `se.Next()`, but for different reasons (to continue the middleware chain).

See: `backend/hooks/ai.go:201-214` and `backend/hooks/view.go:1232-1267` for correct implementations.

### PocketBase Hidden Fields Not Accessible in Record Hooks

**Symptoms:**
- `record.GetString("my_hidden_field")` returns empty string in hooks
- Field data was sent in the request body
- Field is defined with `Hidden: true` in the schema

**Cause:**
In PocketBase v0.23+, fields marked as `Hidden: true` are not auto-populated into the record before hooks run. The `OnRecordCreate` and `OnRecordUpdate` hooks only receive fields that aren't hidden.

**Solution:**
Use `OnRecordCreateRequest` / `OnRecordUpdateRequest` hooks instead, which have access to the raw request body:

```go
// ❌ WRONG - hidden fields not available in RecordEvent
app.OnRecordCreate("ai_providers").BindFunc(func(e *core.RecordEvent) error {
    apiKey := e.Record.GetString("api_key")  // Always empty for hidden fields!
    return e.Next()
})

// ✅ CORRECT - access request body directly
app.OnRecordCreateRequest("ai_providers").BindFunc(func(e *core.RecordRequestEvent) error {
    info, _ := e.RequestInfo()
    if apiKey, ok := info.Body["api_key"].(string); ok {
        // Process the hidden field value
        e.Record.Set("api_key_encrypted", encrypt(apiKey))
    }
    return e.Next()
})
```

**Note:** This is used in `backend/hooks/ai.go` to encrypt API keys before saving to `api_key_encrypted`.

### AI Integration

Facet supports AI-powered content enrichment using configurable providers (OpenAI, Anthropic, Ollama, or custom).

**AI Integration Points:**

| Feature | Endpoint | Description |
|---------|----------|-------------|
| GitHub Import Enrichment | `/api/ai/enrich` | Generates summaries, bullets, and tags from README |
| Content Improvement | `/api/ai/improve` | Improves headlines, summaries, descriptions, etc. |
| Connection Test | `/api/ai/test/{id}` | Tests if an AI provider is configured correctly |

**AI Response Parsing:**

The AI may return JSON with varying types for certain fields. The parser in `backend/services/ai.go` handles this flexibly:

- `case_study` can be string OR array (converted to bullet points)
- Arrays are parsed item-by-item to handle mixed types
- Markdown code blocks are stripped before parsing

**AI Prompt Guidelines:**

All AI prompts include these style rules to avoid AI-sounding language:

```
IMPORTANT WRITING STYLE RULES:
- Write like a human, not an AI. Be direct and natural.
- NEVER use em dashes (—). Use commas, periods, or "and" instead.
- NEVER use words like "delve", "leverage", "utilize", "spearheaded", "synergy", "cutting-edge"
- Avoid corporate buzzwords and marketing speak
- Use simple, clear language over fancy vocabulary
```

See: `backend/services/ai.go:buildPrompt()` and `backend/hooks/ai.go:buildImprovementPrompt()`

**Lessons Learned (AI Debugging):**

These issues caused significant debugging time and should be avoided in future AI work:

1. **PocketBase Hidden Fields Block API Input**
   - Fields with `Hidden: true` are NOT received in API requests
   - We thought `Hidden` only affected API responses, but it blocks input too
   - Solution: Use `OnRecordCreateRequest` hooks to access raw request body
   - See: `backend/hooks/ai.go` for the correct pattern

2. **AI Models Return Inconsistent Types**
   - Asked for `case_study: string`, got `case_study: ["bullet 1", "bullet 2"]`
   - Never assume AI will follow schema exactly
   - Solution: Use flexible parsing with type switches (see `parseEnrichmentResponse`)

3. **Silent Failures in Hook Chains**
   - Missing `e.Next()` causes request to hang or fail silently
   - No error message, no log output, just broken functionality
   - Solution: ALWAYS call `e.Next()` in BindFunc hooks

4. **Encryption Without Verification**
   - API key was "encrypted" but we never logged that it was received
   - Added logging revealed the key was never making it to the hook
   - Solution: Log input AND output at each transformation step

**AI Print Implementation Considerations:**

When implementing AI Print (Phase 4.2), watch for these potential issues:

| Concern | Risk | Mitigation |
|---------|------|------------|
| View data serialization | Large views may exceed token limits | Truncate sections, log payload size |
| Resume prompt formatting | AI may not follow markdown structure | Validate markdown before Pandoc |
| Pandoc conversion | May fail silently on malformed input | Log input/output, capture stderr |
| File storage | PocketBase file upload may fail | Log file size, verify upload success |
| Provider selection | User may not have AI configured | Check status before showing "Generate" button |
| Timeout handling | Large resumes may exceed 60s timeout | Increase timeout, show progress indicator |

**Required Logging for AI Print:**

```go
// Example of required verbose logging
func (h *ViewHooks) generateResume(ctx context.Context, viewID string) error {
    log.Printf("[AI-PRINT] Starting resume generation for view: %s", viewID)

    viewData, err := h.getViewData(viewID)
    if err != nil {
        log.Printf("[AI-PRINT] Failed to get view data: %v", err)
        return err
    }
    log.Printf("[AI-PRINT] View data retrieved, sections: %d, total size: %d bytes",
        len(viewData.Sections), len(viewData.ToJSON()))

    aiResponse, err := h.ai.GenerateResume(ctx, provider, viewData)
    if err != nil {
        log.Printf("[AI-PRINT] AI generation failed: %v", err)
        return err
    }
    log.Printf("[AI-PRINT] AI response received, markdown length: %d", len(aiResponse))

    // ... continue with Pandoc, file storage, etc.
}
```

### PocketBase API 400 Errors with Sort Parameters

**Symptoms:**
- `ClientResponseError 400: Something went wrong while processing your request`
- Error occurs on `getList()` calls with `sort` parameter
- Manual fetch without sort works, but SDK calls with sort fail

**Cause:**
PocketBase collections in this setup do NOT have automatic `created` or `updated` fields. Attempting to sort by these non-existent fields causes a 400 error.

**Solution:**
Use fields that actually exist on the collection:

```javascript
// ❌ Wrong - 'created' field doesn't exist
pb.collection('posts').getList(1, 100, { sort: '-created' })

// ✅ Correct - use existing fields or '-id' (time-ordered)
pb.collection('posts').getList(1, 100, { sort: '-published_at' })
pb.collection('views').getList(1, 50, { sort: '-id' })
```

**Debugging tip:** Check what fields exist on a collection:
```javascript
pb.collection('posts').getList(1, 1).then(d => {
  console.log('Fields:', Object.keys(d.items[0]));
});
```

**Note:** PocketBase record IDs are time-ordered (like ULIDs), so `sort: '-id'` gives newest-first ordering and always works.

### SvelteKit Client-Side Navigation 404 on Root Route

**Symptoms:**
- Clicking a link to `/` (root route) results in a 404 error
- Full page loads to `/` work correctly (e.g., opening in new tab, browser refresh)
- Server-side logs show the page loading successfully, but browser shows 404
- Only affects client-side navigation (SvelteKit's internal routing)

**Cause:**
SvelteKit's client-side navigation to the root route (`/`) can fail in certain configurations, particularly when:
- The root page has a complex server load function with multiple API calls
- There are parameterized routes like `[slug=slug]` that might interfere
- The root page uses default view resolution with fallback logic

This appears to be related to how SvelteKit handles client-side data fetching for the root route. The server-side load function executes correctly (visible in server logs), but the client-side navigation fails to render the page.

**What works vs. what doesn't:**

| Navigation Type | Example | Works? |
|-----------------|---------|--------|
| Full page load | Browser refresh, `target="_blank"` link | ✅ Yes |
| Direct URL entry | Typing `/` in address bar | ✅ Yes |
| Client-side navigation | Regular `<a href="/">` link | ❌ 404 |
| Client-side with reload | `<a href="/" data-sveltekit-reload>` | ✅ Yes |

**Solution:**
Use `data-sveltekit-reload` attribute on links that navigate to the root route:

```svelte
<!-- ❌ Client-side navigation - may 404 -->
<a href="/">Back to Profile</a>

<!-- ✅ Forces full page load - always works -->
<a href="/" data-sveltekit-reload>Back to Profile</a>
```

**Implementation locations:**
- `frontend/src/routes/posts/+page.svelte` - Back to Profile button
- `frontend/src/routes/talks/+page.svelte` - Back to Profile button

**Debugging:**
Navigation events are logged to the browser console:
- `[NAVIGATION] Before navigate:` - shows source and destination
- `[NAVIGATION] After navigate:` - confirms successful navigation
- `[ROOT PAGE CLIENT] Page mounted` - confirms root page rendered

These logs are added via `beforeNavigate` and `afterNavigate` hooks in `+layout.svelte`.

**Note:** This is a workaround, not a root cause fix. The `data-sveltekit-reload` attribute causes a full page reload instead of client-side navigation, which has slightly more overhead but works reliably. If you identify the root cause of the client-side navigation failure, please update this documentation.

## VS Code Tasks

The project includes VS Code tasks (`.vscode/tasks.json`):

- `Ctrl+Shift+B`: Run default build task (dev:up)
- `Ctrl+Shift+P` → "Tasks: Run Task" for all available tasks

Available tasks:
- `dev:up` - Start all services
- `dev:backend` - Start backend only
- `dev:frontend` - Start frontend only
- `dev:reset` - Clear caches
- `test` - Run all tests
- `build:docker` - Build production image

## Dependency Versions

### PocketBase Version Compatibility

**Critical:** The frontend SDK version must be compatible with the backend PocketBase version.

| Component | Version | Notes |
|-----------|---------|-------|
| Backend (Go) | PocketBase v0.23.4 | Set in `backend/go.mod` |
| Frontend SDK | pocketbase ^0.21.0 | Set in `frontend/package.json` |

**Why this matters:**
- SDK v0.22+ renamed `authStore.model` → `authStore.record`
- SDK v0.26+ is designed for PocketBase v0.34+ and uses incompatible request formats
- Using mismatched versions causes 400 errors on authenticated requests

**If upgrading PocketBase backend:**
1. Check the [PocketBase JS SDK releases](https://github.com/pocketbase/js-sdk/releases) for compatible SDK version
2. Update `frontend/package.json` to match
3. If upgrading past v0.22, change `authStore.model` → `authStore.record` in:
   - `frontend/src/lib/pocketbase.ts`
   - `frontend/src/routes/admin/login/+page.svelte`

### Other Key Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| SvelteKit | ^2.0.0 | Frontend framework |
| Svelte | ^4.2.0 | Component framework |
| Vite | ^5.0.0 | Build tool |
| Tailwind CSS | ^3.4.0 | Styling |

## Environment Variables

Copy `.env.example` to `.env` and customize:

```bash
cp .env.example .env
```

Key variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `ENCRYPTION_KEY` | (required in prod) | AES-256-GCM key for AI tokens |
| `SEED_DATA` | — | Seed mode: `dev` for dev profile, unset for no seeding |
| `DATA_DIR` | `./pb_data` | PocketBase data directory |
| `LOG_LEVEL` | `info` | Logging verbosity |

## Seed Data

Facet has two ways to load sample data:

### Development Seed (Jedidiah Esposito)

For development and testing, use `make seed-dev` to load real-world profile data. The script now:
- Lets you choose auth mode: password-only, Google, GitHub, or both.
- Defaults APP_URL to your Codespace URL (if present) or localhost, with an option to override.
- Reuses any existing `.env` values for APP_URL/ADMIN_EMAILS and Google/GitHub creds; only prompts for missing fields.
- Writes the chosen values into `.env` and starts the dev stack with those env vars.

- **Role**: Front-End Lead | Product Engineering Lead
- **Experience**: NZ Police, Ryman Healthcare, Okta, Amazon, ChefSteps
- **Projects**: Facet, MCP Servers, Voice Assistant, Home Infrastructure
- **Skills**: SvelteKit, TypeScript, MCP, LLMs, Agile
- **View**: `/front-end-lead` with LinkedIn CTA

```bash
# Load dev seed data
make seed-dev

# Example: enable both Google and GitHub during the prompt; the script will store
# APP_URL/ADMIN_EMAILS and the provider IDs/secrets into .env for reuse.

# Just clear database (no restart)
make seed-clear
```

### Homepage visibility toggle

From **Admin > Settings**, you can turn off the public homepage (`/`) while keeping shared views accessible via their direct URLs. When disabled, `/`, `/posts`, and `/talks` show a private landing message (customizable in the settings panel). Use this when you want to share specific views without exposing a public homepage.

### Demo Data (Admin UI)

New users can load demo data via **Admin > Settings > Demo Data**. This loads a fun
Arthurian-themed profile (Merlin Ambrosius, Chief Wizard) to demonstrate all features.

The demo data can be loaded and cleared at any time from the admin settings page.
This is useful for:
- Seeing what a complete profile looks like
- Testing views and layouts
- Demonstrating the platform to others

| `SEED_DATA` env | Behavior |
|-----------------|----------|
| `dev` | Auto-seeds Jedidiah Esposito profile (development) |
| (unset) | No auto-seeding (production default) |

**Note:** The `SEED_DATA=demo` option has been removed. Demo data is now managed via the admin UI.

## Architecture Overview

```
┌─────────────────┐     ┌─────────────────┐
│   SvelteKit     │────▶│   PocketBase    │
│   (Frontend)    │ API │   (Backend)     │
│   Port 5173     │     │   Port 8090     │
└─────────────────┘     └─────────────────┘
                              │
                              ▼
                        ┌───────────┐
                        │  SQLite   │
                        │ (pb_data) │
                        └───────────┘
```

For detailed architecture, see [ARCHITECTURE.md](../ARCHITECTURE.md).

## Development Phases

### Phase A: Identity Polish (Complete)

*Tag: `phase-identity-polish`*

Focused exclusively on language, voice, and identity. No functional changes.

Changes:
- All page titles updated from "X | Admin" to "X | Facet"
- Removed "Admin" badge from header
- Login copy: "Sign in to manage your profile"
- Footer fallback: "OwnProfile" → "Facet"
- Import button: "Create Import Proposal" → "Review & Import"
- Removed "(admin only)" from visibility dropdown

### Phase B: First-Run Warmth (Complete)

*Tag: `phase-first-run-warmth`*

Improved empty states and first-run experience. No wizards or progress trackers.

Changes:
- Dashboard: "This is your space" welcome message when empty (replaces 0/0/0/0 stats)
- Activity: "Nothing here yet — and that's okay" empty state
- Profile hero: Removed "?" avatar fallback when no profile
- Profile hero: Removed "Welcome" heading when no name set
- Views: Descriptive empty state explaining what views do
- Contact links: "Add links to help people reach you"
- AI providers: Gentle guidance about optional enrichment

### Phase C: Visual Calm (Complete)

*Tag: `phase-visual-calm`*

Consistent icons, refined button styles, and improved tagline. No emoji icons in UI.

Changes:
- Created `$lib/icons.ts` with consistent SVG icons (check, x, info, warning, eye, trash, copy, lock, download, star, gitFork, brain, zap, toggleOn, toggleOff)
- Replaced all emoji icons across admin pages with SVG icons
- Added `btn-danger` and `btn-danger-ghost` styles for destructive actions
- Login tagline: "A simple home for your story." (replaces tech-focused copy)
- Improved hover states on destructive action buttons

Files modified:
- `frontend/src/lib/icons.ts` (new)
- `frontend/src/app.css`
- `frontend/src/components/shared/Toast.svelte`
- `frontend/src/routes/admin/views/+page.svelte`
- `frontend/src/routes/admin/import/+page.svelte`
- `frontend/src/routes/admin/review/[id]/+page.svelte`
- `frontend/src/routes/admin/settings/+page.svelte`
- `frontend/src/routes/admin/profile/+page.svelte`
- `frontend/src/routes/admin/login/+page.svelte`
