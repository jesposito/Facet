# Facet Roadmap

**Last Updated:** 2026-01-01

This roadmap outlines the feature development plan for Facet (formerly Me.yaml), organized into logical phases. Each phase is independently valuable and builds toward a complete personal profile platform.

**Important**: This roadmap contains no time estimates. Each phase represents a coherent set of features, not a sprint or deadline. Phases should be completed in order, as later phases depend on earlier ones.

---

## ✅ Rebrand Me.yaml → Facet (Complete)

**Purpose**: Rename the project from "Me.yaml" to "Facet" to improve market positioning, broaden appeal, and align the name with the core "Views as Facets" feature.

**Status**: ✅ Complete

### Strategic Rationale

| Aspect | Me.yaml | Facet |
|--------|---------|-------|
| **Audience** | Developers only | Developers + non-technical professionals |
| **Clarity** | Requires YAML knowledge | Intuitive English word |
| **Memorability** | Punctuation, file extension | Single word, easy to spell |
| **Domain availability** | Limited | More options (facet.app, getfacet.io, etc.) |
| **Feature alignment** | None | "Views = Facets" built into name |
| **Word of mouth** | "Check out my me-dot-yaml" | "Check out my Facet" |

### Pre-Flight Checklist

Before starting technical work:

- [ ] **Secure domain name** (facet.app, getfacet.io, facetprofile.com, etc.)
- [ ] **Check trademark availability** ("Facet" is common—ensure no conflicts in software/SaaS)
- [ ] **Secure social handles** (@facet, @getfacet on Twitter/X, GitHub org, etc.)
- [ ] **Decide GitHub strategy**: Rename existing repo (recommended) vs. create new repo
- [ ] **Notify any existing users** (if applicable)

### Phase 1: Documentation & Frontend (Low Risk)

All find-and-replace operations on user-facing text. No breaking changes.

#### 1.1 Documentation Files

| File | Occurrences | Pattern |
|------|-------------|---------|
| `README.md` | 13 | "Me.yaml" → "Facet", "me.yaml" → "facet" |
| `DESIGN.md` | 14 | "Me.yaml" → "Facet" |
| `ROADMAP.md` | 3 | "Me.yaml" → "Facet" |
| `ARCHITECTURE.md` | 1 | "Me.yaml" → "Facet" |
| `agent-instructions.md` | 2 | "Me.yaml" → "Facet" |
| `docs/SETUP.md` | 7 | "Me.yaml" → "Facet", URLs |
| `docs/DEV.md` | 10 | "Me.yaml" → "Facet" |
| `docs/SECURITY.md` | 5 | "Me.yaml" → "Facet" |
| `docs/UPGRADE.md` | 1 | "Me.yaml" → "Facet" |
| `RESEARCH.md` | Multiple | "OwnProfile" → "Facet" (legacy) |

**Tasks:**
- [ ] Global find-replace "Me.yaml" → "Facet" in all `.md` files
- [ ] Global find-replace "me.yaml" → "facet" in URLs/examples
- [ ] Update LICENSE copyright: "me.yaml Contributors" → "Facet Contributors"
- [ ] Review and update tagline: "You, human-readable" → new tagline TBD

#### 1.2 Frontend Page Titles (22 files)

All `<svelte:head><title>` tags containing "Me.yaml":

| Route | Current Title |
|-------|---------------|
| `/` | `... \| Me.yaml` |
| `/admin` | `Dashboard \| Me.yaml` |
| `/admin/login` | `Sign In \| Me.yaml`, "Sign in to Me.yaml" |
| `/admin/profile` | `Edit Profile \| Me.yaml` |
| `/admin/experience` | `Experience \| Me.yaml Admin` |
| `/admin/education` | `Education \| Me.yaml Admin` |
| `/admin/certifications` | `Certifications \| Me.yaml Admin` |
| `/admin/skills` | `Skills \| Me.yaml Admin` |
| `/admin/projects` | `Projects \| Me.yaml Admin` |
| `/admin/posts` | `Posts \| Me.yaml Admin` |
| `/admin/talks` | `Talks \| Me.yaml Admin` |
| `/admin/import` | `Import from GitHub \| Me.yaml` |
| `/admin/views` | `Views \| Me.yaml` |
| `/admin/views/new` | `Create View \| Me.yaml` |
| `/admin/views/[id]` | `Edit View \| Me.yaml` |
| `/admin/tokens` | `Share Tokens \| Me.yaml` |
| `/admin/settings` | `Settings \| Me.yaml` |
| `/admin/review/[id]` | `Review Import \| Me.yaml` |
| `/s/[token]` | `Shared Link \| Me.yaml` |

**Tasks:**
- [ ] Find-replace `| Me.yaml` → `| Facet` in all route `+page.svelte` files
- [ ] Update login page text: "Sign in to Me.yaml" → "Sign in to Facet"

#### 1.3 Frontend Components

| Component | Location | Change |
|-----------|----------|--------|
| `AdminHeader.svelte` | Line 33 | Brand text in header |
| `Footer.svelte` | Lines 13, 47 | Footer branding, "Powered by" |

**Tasks:**
- [ ] Update `frontend/src/components/admin/AdminHeader.svelte`
- [ ] Update `frontend/src/components/public/Footer.svelte`

#### 1.4 Script Comments & Headers

| File | Change |
|------|--------|
| `scripts/start-dev.sh` | Header comments |
| `scripts/dev-backend.sh` | Header comments |
| `scripts/dev-frontend.sh` | Header comments |
| `scripts/seed.js` | File header comment |
| `Makefile` | Header comment |
| `docker/start.sh` | Startup messages ("Starting Me.yaml...") |
| `docker/Caddyfile` | Header comment |

**Tasks:**
- [ ] Update all script headers and comments
- [ ] Update startup/status messages in `docker/start.sh`

---

### Phase 2: Configuration & Build System (Medium Risk)

Changes to build configuration, Docker, and development environment.

#### 2.1 Package Names

| File | Current | New |
|------|---------|-----|
| `frontend/package.json` | `"name": "me-yaml-frontend"` | `"name": "facet-frontend"` |

**Tasks:**
- [ ] Update `frontend/package.json` name field
- [ ] Run `npm install` to update package-lock.json

#### 2.2 Docker Configuration

| File | Changes Needed |
|------|----------------|
| `docker-compose.yml` | Service name `me-yaml` → `facet` |
| `docker-compose.dev.yml` | Container names: `meyaml-backend-dev` → `facet-backend-dev`, etc. |
| `docker-compose.dev.yml` | Volume names: `meyaml-go-cache` → `facet-go-cache`, etc. |
| `docker/Dockerfile` | Binary name `me-yaml` → `facet` |
| `docker/Dockerfile` | User name `meyaml` → `facet` |
| `docker/Dockerfile.dev` | Header comment |

**Tasks:**
- [ ] Update `docker-compose.yml`: service name, container name
- [ ] Update `docker-compose.dev.yml`: all container and volume names (7 occurrences)
- [ ] Update `docker/Dockerfile`: binary name, user name
- [ ] Update `docker/Dockerfile.dev`: header comment
- [ ] Test Docker build: `docker build -t facet:latest ./docker`

#### 2.3 Development Environment

| File | Changes Needed |
|------|----------------|
| `.air.toml` | Binary path `./tmp/me-yaml` → `./tmp/facet` |
| `backend/.air.toml` | Binary path `./tmp/me-yaml` → `./tmp/facet` |
| `.devcontainer/devcontainer.json` | Volume names, container labels |
| `.devcontainer/Dockerfile` | Header comment |
| `.env.example` | Domain examples `meyaml.yourdomain.com` → `facet.yourdomain.com` |

**Tasks:**
- [ ] Update `.air.toml` binary paths (2 files)
- [ ] Update `.devcontainer/devcontainer.json` volume names and labels
- [ ] Update `.devcontainer/Dockerfile` header
- [ ] Update `.env.example` domain examples

#### 2.4 Makefile

| Line | Change |
|------|--------|
| Header | Comment update |
| ~81 | `pkill -9 -f "ownprofile"` → `pkill -9 -f "facet"` |
| ~113, 117 | Docker image tag `me-yaml:latest` → `facet:latest` |
| ~183 | Additional Docker references |

**Tasks:**
- [ ] Update Makefile header comment
- [ ] Update process kill command
- [ ] Update Docker image tags
- [ ] Test: `make dev`, `make build`, `make docker-build`

---

### Phase 3: Backend & API (Higher Risk)

Changes that affect runtime behavior, tokens, and data formats.

#### 3.1 Go Module Namespace (CRITICAL)

The Go module is currently named `ownprofile` (legacy from original name). This should be updated to `facet`.

| File | Current Import | New Import |
|------|----------------|------------|
| `backend/go.mod` | `module ownprofile` | `module facet` |
| `backend/main.go` | `"ownprofile/hooks"`, etc. | `"facet/hooks"`, etc. |
| `backend/hooks/password.go` | `"ownprofile/services"` | `"facet/services"` |
| `backend/hooks/ratelimit.go` | `"ownprofile/services"` | `"facet/services"` |
| `backend/hooks/share.go` | `"ownprofile/services"` | `"facet/services"` |
| `backend/hooks/ai.go` | `"ownprofile/services"` | `"facet/services"` |
| `backend/hooks/view.go` | `"ownprofile/services"` | `"facet/services"` |

**Tasks:**
- [ ] Update `backend/go.mod`: `module ownprofile` → `module facet`
- [ ] Update `backend/main.go`: 3 import path changes
- [ ] Update `backend/hooks/*.go`: 5 files with import changes
- [ ] Run `cd backend && go mod tidy`
- [ ] Run `cd backend && go build ./...` to verify
- [ ] Run `cd backend && go test ./...` to verify all tests pass

#### 3.2 JWT Token Issuer (BACKWARDS COMPATIBLE ✅)

| File | Line | Current | New |
|------|------|---------|-----|
| `backend/services/crypto.go` | 21 | `JWTIssuer = "me.yaml"` | `JWTIssuer = "facet"` + `JWTIssuerLegacy = "me.yaml"` |
| `backend/services/crypto_test.go` | 303-308 | Test assertion | Verify both constants |

**✅ No Breaking Change**: Implemented backwards compatibility - the validation logic accepts both `"facet"` (new) and `"me.yaml"` (legacy) issuers. Existing password-protected view sessions remain valid after upgrade.

**Tasks:**
- [x] Update `JWTIssuer` constant in `crypto.go`
- [x] Add `JWTIssuerLegacy` constant for backwards compatibility
- [x] Update `ValidateViewAccessJWT` to accept both issuers
- [x] Update test assertions in `crypto_test.go`

#### 3.3 Export Metadata

| File | Line | Current | New |
|------|------|---------|-----|
| `backend/hooks/export.go` | 81 | `App: "Me.yaml"` | `App: "Facet"` |
| `backend/hooks/export_test.go` | Multiple | Test assertions | Update assertions |

**Impact**: New exports will show `"app": "Facet"` in metadata. Old exports remain valid.

**Tasks:**
- [ ] Update `export.go` metadata field
- [ ] Update `export_test.go` assertions (5 occurrences)

#### 3.4 Seed Data

| File | Line | Current | New |
|------|------|---------|-----|
| `backend/hooks/seed.go` | 534-542 | Sample project "Me.yaml" | Sample project "Facet" |

**Tasks:**
- [ ] Update seed data project title and GitHub URL
- [ ] Consider if seed data should reference the new repo URL

---

### Phase 4: External References (Strategic Decisions Required)

#### 4.1 GitHub Repository Rename

**Recommended approach**: Rename existing repo in GitHub settings.

GitHub will automatically redirect:
- `github.com/jesposito/me.yaml` → `github.com/jesposito/facet`
- `git clone https://github.com/jesposito/me.yaml.git` will still work (redirects)

**Steps:**
1. Go to repo Settings → General → Repository name
2. Change `me.yaml` to `facet`
3. Update local git remote: `git remote set-url origin https://github.com/jesposito/facet.git`

**Tasks:**
- [ ] Rename GitHub repository
- [ ] Update local git remote URL
- [ ] Update `README.md` clone URLs (2 occurrences)
- [ ] Update `docs/SETUP.md` clone URLs (2 occurrences)
- [ ] Update any CI/CD configuration referencing repo URL

#### 4.2 Domain & Deployment Examples

| File | Location | Change |
|------|----------|--------|
| `.env.example` | Lines 20, 56, 62, 69, 75 | `meyaml.yourdomain.com` → `facet.yourdomain.com` |
| `docs/SETUP.md` | Lines 150-152 | Traefik routing examples |

**Tasks:**
- [ ] Update all domain examples in `.env.example`
- [ ] Update Traefik/Caddy configuration examples in docs

#### 4.3 Brand Assets

| Asset | Location | Action |
|-------|----------|--------|
| Favicon | `frontend/static/favicon.png` | Design new favicon for "Facet" |
| OG Image | (if exists) | Update social sharing image |

**Tasks:**
- [ ] Design new favicon (can be simple "F" or faceted gem icon)
- [ ] Replace `frontend/static/favicon.png`
- [ ] Create OG image for social sharing (optional)

---

### Phase 5: Verification & Testing

#### 5.1 Build Verification

```bash
# Backend
cd backend && go mod tidy && go build ./... && go test ./...

# Frontend
cd frontend && npm install && npm run check && npm run build

# Docker
docker build -t facet:latest ./docker
docker run --rm facet:latest --help

# Full stack
make dev  # Should start without errors
```

**Tasks:**
- [ ] Backend compiles with no errors
- [ ] Backend tests pass (especially crypto and export tests)
- [ ] Frontend builds with no errors
- [ ] Frontend svelte-check passes
- [ ] Docker image builds successfully
- [ ] Development environment starts correctly
- [ ] Production Docker container runs correctly

#### 5.2 Functional Verification

- [ ] Admin login works
- [ ] All admin pages load with correct titles ("... | Facet")
- [ ] Public profile renders correctly
- [ ] Views render correctly
- [ ] Password-protected views work (new tokens)
- [ ] Share tokens work
- [ ] Export generates with `"app": "Facet"` metadata
- [ ] Print stylesheet works
- [ ] GitHub import works

#### 5.3 Search & Verify No Orphaned References

```bash
# Run these after all changes to verify no orphaned references
grep -ri "me\.yaml" --include="*.go" --include="*.ts" --include="*.svelte" --include="*.md"
grep -ri "meyaml" --include="*.go" --include="*.ts" --include="*.svelte" --include="*.yml" --include="*.json"
grep -ri "me-yaml" --include="*.go" --include="*.ts" --include="*.svelte" --include="*.yml" --include="*.json"
grep -ri "ownprofile" --include="*.go"
```

**Tasks:**
- [ ] Run verification greps
- [ ] Address any remaining occurrences
- [ ] Commit final cleanup

---

### Phase 6: Announcement & Migration

#### 6.1 Documentation Updates

- [ ] Update README with "Facet (formerly Me.yaml)" for SEO continuity
- [ ] Add migration note to UPGRADE.md for existing users
- [ ] Update any external documentation or wikis

#### 6.2 Migration Guide for Existing Users

Add to `docs/UPGRADE.md`:

```markdown
## Upgrading from Me.yaml to Facet

### Breaking Changes
- **Export metadata**: New exports will show `"app": "Facet"` instead of `"app": "Me.yaml"`

### Non-Breaking Changes
- **JWT sessions preserved**: Password-protected view sessions remain valid (legacy issuer accepted)

### Docker Users
If upgrading from `me-yaml` container:
1. Stop existing container
2. Rename volume (optional): `docker volume create facet_data && docker run --rm -v me-yaml_data:/from -v facet_data:/to alpine cp -av /from/. /to/`
3. Pull new image: `docker pull jesposito/facet:latest`
4. Start with new container name

### No Data Migration Required
- Database schema unchanged
- All content, views, tokens preserved
- Only cosmetic/branding changes
```

**Tasks:**
- [ ] Write migration guide in UPGRADE.md
- [ ] Test upgrade path from "Me.yaml" Docker container

---

### Summary: Complete Checklist

**Pre-Flight (Do First):**
- [x] Secure domain
- [x] Check trademarks
- [x] Secure social handles

**Phase 1 - Documentation & Frontend:**
- [x] 11 documentation files
- [x] 22 page title updates
- [x] 2 component updates
- [x] 7 script comment updates

**Phase 2 - Configuration:**
- [x] package.json
- [x] 4 Docker files
- [x] 4 dev environment files
- [x] Makefile

**Phase 3 - Backend:**
- [x] Go module rename (8 files)
- [x] JWT issuer constant (with legacy compatibility)
- [x] Export metadata
- [x] Seed data
- [x] All tests passing

**Phase 4 - External:**
- [x] GitHub repo rename
- [x] Update clone URLs in docs
- [x] Domain examples
- [ ] New favicon (deferred - current is acceptable)

**Phase 5 - Verification:**
- [x] All builds pass
- [x] All tests pass
- [x] Manual functional testing
- [x] Grep verification

**Phase 6 - Announcement:**
- [x] Migration guide
- [ ] Announce rebrand (optional - repo is already named Facet)

**✅ Rebrand Complete**

### Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Go module rename breaks build | ~~High~~ None (dev-only) | Not user-facing; build verified, tests pass |
| JWT issuer change breaks sessions | ~~Medium~~ None | Implemented legacy issuer support; sessions preserved |
| Orphaned "Me.yaml" references | Low | Grep verification step |
| GitHub redirect expires | Low | Redirects last indefinitely for most operations |
| SEO impact | Low | Keep "(formerly Me.yaml)" in README for transition period |

---

## Implementation Status Summary

| Milestone | Status | Notes |
|-----------|--------|-------|
| 1. Scaffold | ✅ Complete | All files in place |
| 2. Backend Core | ✅ Complete | PocketBase with all collections |
| 3. Backend Hooks | ✅ Complete | GitHub, AI, share, password |
| 4. Public Site | ✅ Complete | All routes and components |
| 5. Admin Dashboard | ✅ Complete | All CRUD pages |
| 6. Views & Tokens | ✅ Complete | Full token management UI |
| 7. GitHub Importer | ✅ Complete | Import and review working |
| 8. AI Settings | ✅ Complete | Full provider management |
| 9. Docker | ✅ Complete | Production-ready |
| 10. Documentation | 🟡 Partial | Core docs done |
| 11. Testing | 🟡 Partial | Backend tests exist |
| 12. Print & Export | 🟡 Partial | Print stylesheet + data export complete, AI print in progress |
| 13. Visual Layout | ✅ Complete | Layout presets, live preview, section widths, accent colors, per-view theming |

### Remaining Work

**Medium Priority:**
1. Media library (Phase 7)
2. Additional frontend tests
3. AI Print completion (Phase 4.3)

**Low Priority:**
1. AI provider mock tests
2. Integration tests
3. View access log / audit logging (Phase 8)

---

## Development Milestones (Detailed Checklists)

### Milestone 1: Repository Scaffold ✅ Complete
- [x] Initialize Git repository
- [x] Create directory structure (backend/, frontend/, docker/)
- [x] Create go.mod for backend
- [x] Create package.json for frontend
- [x] Create Makefile with common commands
- [x] Create .env.example
- [ ] Set up linting (golangci-lint, eslint, prettier)

### Milestone 2: Backend - PocketBase Core ✅ Complete
- [x] Set up PocketBase as Go framework
- [x] Define all collections via migrations
- [x] Configure OAuth providers (Google, GitHub)
- [x] Set up collection rules (admin-only for most)
- [x] Implement encryption service (AES-256-GCM)
- [x] Add custom `/api/health` endpoint
- [x] Test collection CRUD via API

### Milestone 3: Backend - Custom Hooks ✅ Complete
- [x] GitHub importer service (fetch repo metadata, README, languages, topics, create ImportProposal)
- [x] AI enrichment service (OpenAI, Anthropic, Ollama, custom providers, encrypted keys)
- [x] Share token service (generate, validate, track usage)
- [x] Password protection service (hash, validate, session cookies)

### Milestone 4: Frontend - Public Site ✅ Complete
- [x] Set up SvelteKit project
- [x] Create layout with SEO, Open Graph, responsive navigation
- [x] Implement public routes (/, /[slug], /s/[token], /projects/[slug], /posts/[slug])
- [x] Create all public components (ProfileHero, ExperienceSection, ProjectsSection, etc.)
- [x] Implement dark/light theme
- [x] Add loading states and error pages
- [x] Full accessibility audit

### Milestone 5: Frontend - Admin Dashboard ✅ Complete
- [x] Set up admin layout with sidebar
- [x] Implement OAuth login flow
- [x] Create admin routes (/admin, /admin/profile, /admin/experience, etc.)
- [x] Create admin components (AdminHeader, AdminSidebar, Toast)
- [ ] `/admin/media` - Media library (deferred to Phase 7)

### Milestone 6: Views & Share Tokens ✅ Complete
- [x] Admin UI for views (list, create, edit, section selector, item picker, overrides)
- [x] Share token management UI (generate, copy URL, revoke, status badges, usage stats)
- [x] Public view rendering (section filters, item filters, overrides, password handling)

### Milestone 7: GitHub Importer UI ✅ Complete
- [x] `/admin/import` - Import wizard
- [x] `/admin/review/[id]` - Review UI with per-field controls

### Milestone 8: AI Provider Settings ✅ Complete
- [x] `/admin/settings` - AI providers (add, test, set default, delete)
- [x] Enrichment options in import (provider selector, privacy levels)

### Milestone 9: Docker & Deployment ✅ Complete
- [x] Production Dockerfile (multi-stage build)
- [x] Development Dockerfile
- [x] docker-compose.yml and docker-compose.dev.yml
- [x] .env.example with all vars

### Milestone 10: Documentation & Polish (Partial)
- [x] README.md, DESIGN.md, ARCHITECTURE.md, ROADMAP.md
- [x] DEV.md - Development setup
- [x] Seed data for demo
- [ ] Final testing pass
- [ ] Performance check

### Milestone 11: Testing (Partial)
- [x] Backend tests (crypto, share token, rate limiting, routing, visibility, collection rules)
- [ ] GitHub API mock tests
- [ ] AI provider mock tests
- [ ] Frontend tests (component, view access, form validation)
- [ ] Integration tests (OAuth flow, import pipeline, review flow)

---

## Phase 0: Foundation Stabilization (Complete)

**Purpose**: Ensure the existing foundation is solid before adding new features.

### Features
- [x] Core routing model (/, /<slug>, /s/<token>)
- [x] Views with visibility controls
- [x] Share token generation and validation
- [x] Password-protected views with JWT
- [x] GitHub import pipeline
- [x] AI enrichment (optional)
- [x] Admin dashboard with CRUD for all content types
- [x] Admin navigation grouped into labeled sections for clarity (Overview, All About You, Faceted Views & Sharing, AI & Imports, System) with collapsible sidebar
- [x] Rate limiting on sensitive endpoints
- [x] Reserved slug protection (frontend + backend)

### Bugs Fixed
- [x] TypeScript errors in review page (null checks, param validation)
- [x] A11y warnings (label → span for non-form controls)

### Prerequisites
None (this is the starting phase)

### Risks
- PocketBase is pre-v1.0; breaking changes possible on upgrade
- Current test coverage is basic; more integration tests needed

---

## Phase 1: Content Completeness (Complete)

**Purpose**: Fill in missing content types and their public-facing pages.

### Features

#### 1.1 Project Detail Pages (Complete)
- [x] Route: `/projects/<slug>`
- [x] Full project page with description, tech stack, media gallery
- [x] Links to GitHub, demo, etc.
- [ ] Related projects (same categories) — Deferred to Phase 2.5
- [x] Meta tags for sharing (Open Graph)

#### 1.2 Posts/Blog System (Complete)
- [x] Route: `/posts/<slug>`
- [x] Markdown rendering with syntax highlighting
- [x] Cover images
- [x] Tags with filtering
- [x] Previous/next navigation
- [x] Admin: Full CRUD for posts
- [ ] Rich text editor — Deferred (basic markdown sufficient)

#### 1.3 Talks Section (Complete)
- [x] Public display in profile
- [x] Embedded video players (YouTube, Vimeo)
- [x] Slides embed/download
- [x] Admin: Full CRUD for talks

#### 1.4 Certifications Section (Complete)
- [x] Public display with verification links
- [x] Expiry date handling (shows expired/expiring soon badges)
- [x] Grouping by issuer
- [x] Admin: Full CRUD for certifications

### Prerequisites
- Phase 0 complete

### Risks
- Adding new routes requires updating reserved slug list
- Markdown rendering security (XSS prevention)

---

## Phase 1.5: Content Discovery & Navigation (Complete)

**Purpose**: Improve discoverability of posts and talks by adding index pages and navigation tabs.

### Current State Analysis

**Posts:**
- [x] Individual post pages at `/posts/[slug]`
- [x] Posts section displays on profile with cards
- [x] Admin CRUD complete
- [x] Visibility settings (public/unlisted/private) and draft status
- [x] View limiting via section selection already works
- [x] Index page at `/posts` to browse all posts
- [x] Navigation tabs to jump directly to posts section

**Talks:**
- [x] Talks section displays on profile with embedded videos
- [x] Admin CRUD complete
- [x] Visibility settings and draft status
- [x] View limiting via section selection already works
- [x] Individual talk pages at `/talks/[slug]`
- [x] Index page at `/talks` to browse all talks
- [x] Navigation tabs to jump directly to talks section

### Features

#### 1.5.1 Profile Navigation Tabs (Complete)

Add a navigation bar that appears when the profile has multiple content types (posts, talks, projects).

**Behavior:**
- Navigation tabs appear below the hero section
- Only show tabs for sections that have visible content
- Clicking a tab smooth-scrolls to that section
- Sticky on scroll (implemented)

**Tabs to show (when content exists):**
- Experience
- Projects
- Education
- Certifications
- Skills
- Posts (links to /posts index)
- Talks (links to /talks index)

**Implementation:**
- [x] Create `ProfileNav.svelte` component
- [x] Compute visible sections from page data
- [x] Add smooth-scroll behavior with anchor links
- [x] Make nav sticky on scroll
- [x] Hide on print

#### 1.5.2 Posts Index Page (Complete)

Add `/posts` route to browse all published posts.

**Features:**
- Grid layout of post cards
- Filter by tag (query param: `/posts?tag=go`)
- Sort by date (newest first default)
- Meta tags for SEO
- Link back to profile

**Implementation:**
- [x] Create `/posts/+page.svelte` route
- [x] Create `/posts/+page.server.ts` to fetch all public, non-draft posts
- [x] Add tag filter UI
- [x] Update reserved slugs (already in place)
- [ ] Add pagination (deferred - not needed for small collections)

#### 1.5.3 Talks Index Page (Complete)

Add `/talks` route to browse all talks.

**Features:**
- List layout of talk entries with video thumbnails
- Filter by year
- Sort by date (newest first default)
- Meta tags for SEO
- Link back to profile

**Implementation:**
- [x] Create `/talks/+page.svelte` route
- [x] Create `/talks/+page.server.ts` to fetch all public, non-draft talks
- [x] Add year filter UI
- [x] Update reserved slugs (already in place)

#### 1.5.4 Individual Talk Pages (Complete)

Add `/talks/[slug]` route for detailed talk view.

**Features:**
- Full talk detail page similar to posts
- Video embed (YouTube/Vimeo, full width)
- Slides link
- Event details and description (markdown rendered)
- Previous/next talk navigation
- Meta tags for SEO (Open Graph video support)
- Back to talks list link

**Implementation:**
- [x] Add `slug` field to talks collection (migration 1735600005)
- [x] Create `/talks/[slug]/+page.svelte` route
- [x] Create `/talks/[slug]/+page.server.ts`
- [x] Add `/api/talk/{slug}` backend endpoint
- [x] Add admin UI for talk slug (auto-generate from title)
- [x] Update talk cards to link to detail page when slug exists

### View Limiting Considerations

**Already Working:**
- Views can enable/disable posts and talks sections
- Views can select specific posts/talks to include
- Visibility (public/unlisted/private) filters content correctly
- Draft status filters content correctly

**To Consider:**
- Index pages (`/posts`, `/talks`) should only show public, non-draft items
- Index pages are NOT view-scoped (they show all public content)
- Individual pages (`/posts/[slug]`, `/talks/[slug]`) respect visibility settings
- Views continue to work as curated collections

### Prerequisites
- Phase 1 complete ✅

### Risks
- Adding `/posts` and `/talks` routes already reserved as slugs ✅
- Talks need slug field added (migration required)
- Navigation tabs add visual complexity

---

## Phase 2: View System Enhancement (Current)

**Purpose**: Make views more powerful and easier to manage.

### Features

#### 2.1 View Editor Core (Complete)
- [x] View editor page (`/admin/views/[id]`)
- [x] View create page (`/admin/views/new`)
- [x] Per-section toggle controls (enable/disable sections)
- [x] Per-section item selection with checkboxes
- [x] Hero overrides (custom headline, summary per view)
- [x] CTA configuration (button text and URL)
- [x] Visibility settings (public, unlisted, password, private)
- [x] Drag-and-drop section ordering (svelte-dnd-action)
- [x] Preview pane showing live result — Implemented in Phase 6.2

#### 2.2 Section & Item Customization (Complete)
- [x] Drag-and-drop section reordering
- [x] Drag-and-drop item reordering within sections
- [x] **Item-level field overrides** ✅ Complete
- [ ] Custom section headings per view — Deferred
- [ ] Show/hide section titles — Deferred
- [ ] Section layout options (list, grid, compact) — Deferred

##### Item-Level Overrides ✅ Complete

Enable per-view customization of individual items without modifying source records:

| Collection | Overridable Fields |
|------------|-------------------|
| Experience | title, description, bullets |
| Projects | title, summary, description |
| Education | degree, field, description |
| Talks | title, description |

**Use Case**: Career pivoter has one job record but presents it differently:
- "UX Designer" view → emphasizes user research, prototyping
- "Instructional Designer" view → emphasizes learning design, curriculum

**Implementation** (Complete):
- [x] "Customize" button on selected items in view editor
- [x] Override editor modal with original value preview
- [x] Override count badges on items with customizations
- [x] Backend merges overrides when serving view data

#### 2.3 Default View Management (Complete)
- [x] Clear UI for setting default view (checkbox in editor)
- [x] Default view badge in views list
- [x] Only one view can be default (enforced)
- [ ] Warning when changing default — Minor, deferred
- [x] Preview of how homepage will look — Implemented in Phase 6.2

#### 2.4 View Analytics (Minimal)
- [ ] View count per view (opt-in)
- [ ] Last accessed timestamp
- [ ] No PII collected

### Prerequisites
- Phase 1 complete

### Risks
- ~~Drag-drop complexity; may need library (svelte-dnd-action)~~ — Resolved: svelte-dnd-action installed and working
- View config schema changes require migration

---

## Phase 3: Share Token Management (Complete)

**Purpose**: Full control over share tokens with admin UI.

### Features

#### 3.1 Token Management Page (Complete)
- [x] Route: `/admin/tokens`
- [x] List all tokens grouped by view with status, usage, expiry
- [x] Create new token (name, expiry, max uses)
- [x] Copy token URL to clipboard
- [x] Revoke/delete tokens with confirmation
- [x] Status badges (active, expired, revoked, max uses reached)

#### 3.2 Token Analytics (Partial)
- [x] Use count display
- [x] Last used timestamp
- [ ] Usage history (recent accesses) — Deferred to Phase 8

#### 3.3 Batch Operations
- [ ] Revoke all tokens for a view — Deferred
- [ ] Expire all tokens older than X days — Deferred
- [ ] Export token list (for auditing) — Deferred

#### 3.4 Token QR Codes
- [ ] Generate QR code for share URL — Deferred
- [ ] Download as PNG — Deferred
- [ ] Useful for physical sharing (business cards, posters)

### Prerequisites
- Phase 2 complete (views are stable) ✅

### Risks
- QR generation may need external library
- Usage history requires new audit collection

---

## Phase 4: Export & Print System

**Purpose**: Enable professional resume/CV generation with two tiers: simple browser print and AI-powered document generation.

### Design Philosophy

Two-tier approach addresses different needs:
1. **Simple Print**: Fast, works offline, user controls final formatting via browser
2. **AI Print**: Professional quality, AI optimizes content and formatting for target role/industry

### Features

#### 4.1 Simple Print ✅ Complete

Browser-based printing optimized for resumes. Zero setup required.

- [x] Optimized print stylesheet in `app.css`
- [x] Page breaks at section boundaries
- [x] Hide navigation, theme toggle, footer
- [x] Print button on public pages
- [x] ATS-friendly typography (Helvetica headers, Georgia body)
- [x] Force light mode colors
- [x] Display URLs after links
- [x] Proper page margins (letter size, 0.5in × 0.6in)

**Usage**: Navigate to any view → Click print button → Browser Print dialog (Ctrl+P) → Save as PDF

#### 4.2 AI Print ✅ Complete

AI-powered document generation that creates polished, professionally formatted resumes.

> **📖 Full Documentation:** See [docs/AI_FEATURES.md](docs/AI_FEATURES.md) for complete AI feature documentation including API reference, provider setup, and troubleshooting.

**How It Works:**

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  View Data  │ ──▶ │   AI API    │ ──▶ │   Pandoc    │ ──▶ │  DOCX/PDF   │
│  (JSON)     │     │  (Optimize) │     │  (Convert)  │     │  (Storage)  │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
```

1. **Collect**: Gather complete view data (profile, sections, overrides)
2. **Optimize**: Send to AI with resume formatting prompt
3. **Structure**: AI returns optimized markdown with resume-specific formatting
4. **Convert**: [Pandoc](https://pandoc.org/MANUAL.html) converts markdown → DOCX and PDF
5. **Store**: Files saved to PocketBase, linked to view
6. **Download**: User downloads from view editor or public page

**Schema Changes:**

```typescript
// New collection: view_exports
interface ViewExport {
  id: string;
  view: string;           // Relation to views
  format: 'pdf' | 'docx';
  file: string;           // PocketBase file field
  ai_provider?: string;   // Relation to ai_providers (null for non-AI)
  generated_at: string;
  generation_config?: {
    target_role?: string;     // "Software Engineer at FAANG"
    style?: 'chronological' | 'functional' | 'hybrid';
    length?: 'one-page' | 'two-page' | 'full';
    emphasis?: string[];      // ["leadership", "technical"]
  };
}

// Addition to ViewSection (future)
interface ViewSection {
  // ... existing fields
  ai_instructions?: string;  // Per-section AI guidance
}
```

**AI Prompt Strategy:**

The AI receives:
- Complete view data as structured JSON
- User's target role/industry (optional)
- Resume style preferences
- Length constraints

The AI returns:
- Optimized markdown formatted for Pandoc
- Suggestions applied (better action verbs, quantified achievements)
- Content prioritized for target role
- Consistent formatting throughout

**Implementation Tasks:**

- [x] Add `view_exports` collection via migration
- [x] Create resume prompt template (stored in backend)
- [x] Add `/api/view/{slug}/generate` endpoint
- [x] Integrate Pandoc in Docker image
- [x] Add "AI Resume" option to print dropdown on public views
- [x] Add generation config modal (format, style, length)
- [x] Auto-download generated files
- [x] Rate limiting for public access (5/hour per IP)
- [x] Target role uses view's hero_headline (profile owner configured)
- [ ] Add reference DOCX template for consistent styling — Deferred
- [ ] Add "Regenerate" button with spinner in admin — Deferred
- [ ] Show generation timestamp and AI provider used in admin — Deferred

**UX Flow:**

1. User visits public view page (or root page with default view)
2. Clicks print dropdown, selects "AI Resume"
3. Modal appears with options:
   - Format: PDF / Word Document
   - Style: Chronological / Functional / Hybrid
   - Length: One page / Two pages / Full
   - Target role: Automatically uses view's hero_headline
4. Clicks "Generate"
5. Loading state shows progress
6. On success, file auto-downloads

**Error Handling:**

- No AI provider configured → Show setup prompt with link to /admin/settings
- AI API failure → Show error, suggest retry
- Pandoc failure → Log error, notify user
- File too large → Warn user, suggest shorter view

#### 4.3 Document Templates

Pre-designed templates for consistent, professional output.

- [ ] Default resume template (clean, ATS-friendly)
- [ ] Academic CV template (publications, research focus)
- [ ] Creative template (for design roles)
- [ ] Template selection in generation config

**Technical Approach:**
- Templates are reference DOCX files with styles defined
- Pandoc uses `--reference-doc` flag to apply template styling
- Templates stored in `backend/templates/` directory

#### 4.4 Data Export ✅ Complete

Export all data for backup or migration.

- [x] Export all data as JSON
- [x] Export as YAML (human-readable backup)
- [ ] Include uploaded files in ZIP archive — Deferred
- [ ] Import from backup (restore) — Deferred

#### 4.5 Static Snapshot

Generate self-contained HTML for offline sharing.

- [ ] Generate static HTML of a view
- [ ] Inline all CSS and base64 images
- [ ] Single file output for email attachment

### Prerequisites
- Phase 3 complete ✅
- AI providers configured (for AI Print)
- Pandoc available in Docker image (for document conversion)

### Technical Requirements

**Pandoc Integration:**

Option A: Include Pandoc in Docker image
```dockerfile
# Add to production Dockerfile
RUN apt-get update && apt-get install -y pandoc
```

Option B: Use [pandoc/latex Docker image](https://hub.docker.com/r/pandoc/latex) as sidecar
```yaml
# docker-compose.yml
services:
  pandoc:
    image: pandoc/latex
    volumes:
      - ./temp:/data
```

Option C: Shell exec to host Pandoc (if installed)
```go
cmd := exec.Command("pandoc", "-f", "markdown", "-o", "output.docx", "input.md")
```

**Recommended**: Option A for simplicity, Option B for full LaTeX support (better PDF quality)

### Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| AI returns poorly formatted content | Validate markdown structure, fallback to simple format |
| Pandoc not available | Graceful degradation to browser print |
| Large documents timeout | Set reasonable limits, show progress |
| Template styling inconsistent | Test templates thoroughly, provide preview |
| AI costs | Show estimated cost, require confirmation for long docs |

### Research References
- [Pandoc User's Guide](https://pandoc.org/MANUAL.html) - Comprehensive conversion documentation
- [Pandoc Docker Images](https://hub.docker.com/r/pandoc/latex) - Pre-built containers with LaTeX
- [Simple Markdown Resume Workflow](https://sdsawtelle.github.io/blog/output/simple-markdown-resume-with-pandoc-and-wkhtmltopdf.html) - End-to-end example
- [LaTeX Résumé AI](https://medium.com/institute-for-applied-computational-science/latex-r%C3%A9sum%C3%A9-ai-an-ai-powered-cv-creation-tool-and-natural-language-document-editor-7cbfe52f846f) - AI-powered CV creation approach

---

## Phase 5: Import System Expansion

**Purpose**: Support more import sources beyond GitHub.

### Features

#### 5.1 LinkedIn Import
- [ ] Manual JSON upload (LinkedIn data export)
- [ ] Map to experience, education, skills
- [ ] Proposal-based review (same as GitHub)

#### 5.2 JSON Resume Import
- [ ] Import from JSON Resume format
- [ ] Bi-directional: export to JSON Resume

#### 5.3 Scheduled Sync
- [ ] Cron-based GitHub refresh
- [ ] Configurable interval (daily, weekly, monthly)
- [ ] Auto-create proposals for review
- [ ] Email notification (optional)

#### 5.4 Credential & Badge Import
- [ ] Credly badge import (via public profile URL or API)
- [ ] Acclaim/Pearson badges support
- [ ] Auto-map to certifications collection
- [ ] Badge image/logo import
- [ ] Verification URL extraction
- [ ] Periodic refresh for expiry updates
- [ ] Other badge platforms as demand emerges

#### 5.5 Source Management UI
- [ ] List all sources with sync status
- [ ] Manual refresh button
- [ ] Unlink source from project
- [ ] View sync history/logs

### Prerequisites
- Phase 4 complete

### Risks
- LinkedIn JSON format may change
- Scheduled sync requires background job system

---

## Phase 6: Visual Layout System

**Purpose**: Enable per-section layout customization with guardrails that prevent bad design choices. Inspired by [SharePoint's flexible sections](https://learn.microsoft.com/en-us/sharepoint/dev/design/layout-patterns) but simpler - curated presets rather than freeform editing.

### Design Principles

1. **Guardrails for Non-Designers**: Only offer layouts proven to look good for each content type
2. **Progressive Disclosure**: Defaults work without configuration; advanced options are optional
3. **Responsive by Default**: All layouts must work on mobile - users can't break responsiveness
4. **Instant Feedback**: Changes should preview immediately or with minimal friction

### Features

#### 6.1 Per-Section Layout Presets (Phase A - Foundation) ✅ Complete

Add a `layout` field to each section in the view editor. Each section type has its own curated set of valid layouts.

**Schema Change:**
```typescript
interface ViewSection {
  section: string;
  enabled: boolean;
  items?: string[];
  layout?: SectionLayout;      // NEW: 'default' | 'compact' | 'timeline' | etc.
  layoutOptions?: {            // NEW: Future extensibility
    columns?: 2 | 3;
    showImages?: boolean;
  };
  itemConfig?: Record<string, ItemConfig>;
}
```

**Layout Options by Section:**

| Section | Available Layouts | Default | Notes |
|---------|-------------------|---------|-------|
| Experience | `default`, `timeline`, `compact` | default | Timeline emphasizes career progression |
| Projects | `grid-3`, `grid-2`, `list`, `featured` | grid-3 | Featured shows 1 large + grid |
| Education | `default`, `timeline` | default | Timeline connects education visually |
| Certifications | `grouped`, `grid`, `timeline` | grouped | Grouped = by issuer (current) |
| Skills | `grouped`, `cloud`, `bars`, `flat` | grouped | Cloud = size by proficiency |
| Posts | `grid-3`, `grid-2`, `list`, `featured` | grid-3 | Same as projects |
| Talks | `default`, `cards`, `list` | default | Default embeds video |

**Implementation:**
- [x] Add `layout` field to ViewSection type in `pocketbase.ts`
- [x] Add `VALID_LAYOUTS` constant mapping section → allowed layouts
- [x] Add layout dropdown in view editor (in section header when expanded)
- [x] Backend passes layout through in `/api/view/:slug/data` response
- [x] Update section components to accept `layout` prop
- [x] Implement 2-3 layout variants per section (start with most valuable)

**UX Flow:**
1. User expands section in view editor
2. Sees "Layout" dropdown next to section toggle (default: "Default")
3. Options filtered to valid layouts for that section type
4. Selection saves with view config
5. Public view renders with selected layout

#### 6.2 Live Preview Pane (Phase B - Feedback) ✅ Complete

Add side-by-side preview in the view editor for immediate visual feedback.

- [x] Split-pane layout: editor left (~60%), preview right (~40%)
- [x] Preview updates on any change (reactive Svelte bindings)
- [x] Preview uses actual section components (not mockups)
- [x] Toggle button to hide preview for more editor space
- [x] Mobile preview mode (preview shown at mobile width) — Complete (Phase 6.2.2)

**Implementation Details:**
- `ViewPreview.svelte` component reuses public section components
- Reactive updates via Svelte props (no debouncing needed)
- Preview rendered in same page (not iframe) for simplicity
- Responsive layout: side-by-side on desktop, stacked on mobile
- Preview scales down content for compact display
- Desktop/Mobile toggle buttons in preview header (Phase 6.2.2)
- Mobile preview constrains to 375px with phone frame styling
- Section widths collapse to full-width in mobile mode

#### 6.3 Section Width & Columns (Phase C - Complete) ✅

Enable sections to share horizontal space (side-by-side layouts).

**Width Options:**
- `full` - 100% width (current default)
- `half` - 50% width (pairs with another half)
- `third` - 33% width (triplets)

**Implementation:**
- [x] Width dropdown in view editor (both create and edit pages)
- [x] Visual column indicator icons showing layout
- [x] CSS Grid with 6-column structure on public view
- [x] Responsive collapse to full-width on mobile (< 768px)
- [x] Live preview reflects width settings in real-time
- [x] Backend returns `section_widths` map in API response

**Schema Addition:**
```typescript
interface ViewSection {
  // ... existing fields
  width?: 'full' | 'half' | 'third';  // Added
}
```

**Example:**
```
[Experience: full]     → Full width row
[Skills: half][Certs: half]  → Side-by-side row
[Projects: full]       → Full width row
```

#### 6.4 Visual WYSIWYG Editor (Phase D - Future)

Full drag-and-drop editing directly in the preview pane.

- [ ] Drag sections to reorder in preview
- [ ] Resize handles on section edges
- [ ] Drop zones between sections
- [ ] Inline editing of section titles
- [ ] Mobile/tablet/desktop preview breakpoints

**Deferred Rationale:** This requires significant interaction layer complexity. The phased approach (A→B→C) delivers 80% of the value with 20% of the complexity. WYSIWYG can be added later when the foundation is solid.

### Color & Theme Customization

#### 6.5 Accent Color (Curated Palette) ✅ Complete

Enable users to customize their profile's accent color via Admin Settings. Uses a **curated palette approach** (not freeform color picker) to maintain design guardrails and accessibility.

**Design Philosophy:**
- Curated palette prevents ugly/inaccessible color combinations
- All colors are pre-tested for WCAG contrast compliance
- Simple UI with visual preview
- Global setting (not per-view) for simplicity

**Curated Color Palette:**

| Name | Hex | CSS Variable | Use Case |
|------|-----|--------------|----------|
| **Sky** (default) | `#0ea5e9` | `--accent-sky` | Tech, software, professional |
| **Indigo** | `#6366f1` | `--accent-indigo` | Creative, design, consulting |
| **Emerald** | `#10b981` | `--accent-emerald` | Finance, sustainability, health |
| **Rose** | `#f43f5e` | `--accent-rose` | Marketing, creative, personal branding |
| **Amber** | `#f59e0b` | `--accent-amber` | Education, construction, energy |
| **Slate** | `#64748b` | `--accent-slate` | Minimal, monochrome, conservative |

**What Accent Color Affects:**
- Primary buttons (`.btn-primary`)
- Links and hover states
- Profile hero gradient tint
- Badges and tag highlights
- Focus outlines for accessibility
- Active states and selections

**UI Design (Admin → Settings):**

```
┌─────────────────────────────────────────────────────────────┐
│ Appearance                                                   │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                              │
│ Accent Color                                                 │
│ Choose a color for buttons, links, and highlights.          │
│                                                              │
│ [Sky ●] [Indigo ●] [Emerald ●] [Rose ●] [Amber ●] [Slate ●] │
│    ✓                                                         │
│                                                              │
│ Preview:                                                     │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │  [Primary Button]  [Secondary]  Link Example            │ │
│ └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

**Technical Implementation:**

1. **Schema Change:**
```typescript
// Add to profile collection (or new site_settings collection)
interface SiteSettings {
  accent_color: 'sky' | 'indigo' | 'emerald' | 'rose' | 'amber' | 'slate';
}
```

2. **CSS Custom Properties:** Inject variables in `+layout.svelte`:
```css
:root {
  --color-primary-50: var(--accent-50);
  --color-primary-500: var(--accent-500);
  --color-primary-600: var(--accent-600);
  /* ... full scale 50-950 */
}
```

3. **Color Scale Generation:** Each accent color has a full Tailwind-style scale (50-950) pre-defined in a constants file.

4. **Component Updates:** Migrate hardcoded `primary-*` classes to use CSS variables where dynamic theming is needed.

**Implementation Tasks:**

- [x] Add `accent_color` field to profile collection (migration)
- [x] Create color palette constants file with full scales
- [x] Add "Appearance" section to Admin Settings page
- [x] Create color swatch selector component with visual feedback
- [x] Add live preview showing button/link appearance
- [x] Inject CSS custom properties in root layout based on setting
- [x] Update `app.css` component classes to use CSS variables
- [x] Test all 6 colors across light and dark modes
- [x] Verify WCAG contrast ratios for all combinations

**Out of Scope (Intentionally):**
- ❌ Freeform color picker (guardrails philosophy)
- ❌ Custom font selection (deferred)

#### 6.6 Per-View Theming & Presets ✅ Complete

Enable different views to have different visual styles. A recruiter view might use professional Indigo, while a speaking/conference view uses energetic Rose.

**Per-View Accent Color Override:**

Each view can optionally override the global accent color. This enables:
- **Recruiter view** → Indigo (professional, corporate)
- **Speaking view** → Rose (energetic, memorable)
- **Portfolio view** → Emerald (creative, fresh)
- **Default view** → Uses global setting

**Schema Change:**
```typescript
interface ViewSection {
  // ... existing fields
}

interface View {
  // ... existing fields
  accent_color?: 'sky' | 'indigo' | 'emerald' | 'rose' | 'amber' | 'slate' | null;
  // null = inherit from global setting
}
```

**UI Design (View Editor):**

```
┌─────────────────────────────────────────────────────────────┐
│ View Settings                                                │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                              │
│ Accent Color                                                 │
│ ○ Use global setting (Sky)                                  │
│ ● Override for this view:                                   │
│   [Sky ●] [Indigo ●] [Emerald ●] [Rose ●] [Amber ●] [Slate] │
│              ✓                                               │
└─────────────────────────────────────────────────────────────┘
```

**Implementation Tasks:**

- [x] Add `accent_color` field to views collection (migration)
- [x] Add accent color selector to view editor
- [x] Update view data API to include accent color
- [x] Frontend applies view accent color when rendering public view
- [x] Preview pane reflects view-specific accent color

**Theme Presets (Future):**

Full theme presets that combine accent color with typography and spacing choices.

- [ ] Bundled themes: Minimal, Professional, Creative
- [ ] One-click apply (sets colors, fonts, spacing)
- [ ] Reset to default option
- [ ] Per-view theme preset selection

#### 6.7 Custom CSS (Power Users)
- [x] Admin textarea for custom CSS
- [x] Scoped to public pages only (not admin)
- [ ] Syntax validation and preview
- [ ] Warning about responsiveness risks

### Prerequisites
- Phase 2.2 complete (drag-drop reordering) ✅
- Section components already accept items prop

### Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Layout variants multiply component complexity | Use conditional rendering, not separate files |
| Users create ugly layouts | Curated presets only - no freeform |
| Preview performance with large datasets | Debounce updates, limit preview items |
| Mobile breakage | All layouts must be mobile-responsive by design |
| Schema migration | Layout field is optional, defaults to 'default' |
| Accent colors fail contrast | Pre-test all colors for WCAG AA compliance |
| CSS variable support | Modern browsers only; fallback to default sky |

### Research References
- [SharePoint Layout Patterns](https://learn.microsoft.com/en-us/sharepoint/dev/design/layout-patterns) - Grid, list, filmstrip patterns
- [SharePoint Flexible Sections](https://www.sharepointdesigns.com/blog/how-to-use-flexible-sections-in-sharepoint-pages-a-simple-guide) - 12-cell grid approach
- [Notion Portfolio Templates](https://super.so/create/how-to-create-a-portfolio-site-with-notion-and-super) - Clean section layouts

---

## Phase 7: Media Management

**Purpose**: Better handling of uploaded files.

### Features

#### 7.1 Media Library
- [ ] Route: `/admin/media`
- [ ] Grid view of all uploads
- [ ] Filter by type, date, usage

#### 7.2 Image Optimization
- [ ] Auto-generate thumbnails
- [ ] WebP conversion
- [ ] Responsive image srcsets

#### 7.3 Unused File Cleanup
- [ ] Identify orphaned files
- [ ] Bulk delete option
- [ ] Storage usage display

#### 7.4 External Media
- [ ] Embed from external URLs
- [ ] YouTube, Vimeo thumbnails
- [ ] Preview external images

### Prerequisites
- Phase 6 complete

### Risks
- Image processing may require additional Go libraries
- Storage management complexity

---

## Phase 8: Security & Audit

**Purpose**: Enhanced security features and access logging.

### Features

#### 8.1 Audit Log
- [ ] Log all admin actions
- [ ] Log share token usage
- [ ] Log password attempts
- [ ] Filterable log viewer

#### 8.2 Security Headers
- [ ] Content Security Policy
- [ ] Permissions Policy
- [ ] Enhanced CORS settings

#### 8.3 Two-Factor Auth
- [ ] TOTP for admin login
- [ ] Backup codes
- [ ] Optional per deployment

#### 8.4 Session Management
- [ ] View active sessions
- [ ] Revoke sessions
- [ ] Session expiry settings

### Prerequisites
- Phase 7 complete

### Risks
- 2FA adds complexity for single-user system
- Audit log storage may grow large

---

## Phase 9: Polish & Performance

**Purpose**: Final refinements for production quality.

### Features

#### 9.1 Performance Audit
- [ ] Lighthouse score optimization
- [ ] Image lazy loading
- [ ] Bundle size reduction
- [ ] Database query optimization

#### 9.2 Accessibility Audit ✅ Complete
- [x] Skip navigation link for keyboard users
- [x] Screen reader support (sr-only utility, aria-labels)
- [x] Keyboard navigation audit
- [x] ARIA attributes on all interactive elements
- [x] Decorative elements marked aria-hidden

#### 9.3 SEO Optimization
- [ ] Structured data (JSON-LD)
- [ ] Auto-generated sitemap
- [ ] robots.txt management
- [ ] Canonical URLs

#### 9.4 Error Handling
- [ ] Custom 404 page
- [ ] Custom 500 page
- [ ] Error boundary components
- [ ] User-friendly error messages

### Prerequisites
- All previous phases complete

### Risks
- Performance optimization is iterative
- Accessibility fixes may require structural changes

---

## Future Considerations

These are ideas that may be explored after the core roadmap is complete:

### Homepage Privacy Control

Enable users to hide their default public profile while still allowing access to specific views via tokens or direct links. This is useful for:
- Job seekers who want to share tailored views with specific recruiters
- Professionals who don't want a public presence but need shareable profile links
- Users in transition who are "setting up" their profile

#### Proposed UX

**Admin Settings Panel:**
```
┌─────────────────────────────────────────────────────────────┐
│ Profile Visibility                                          │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                             │
│ Public Homepage  [━━━━━━○    OFF]                           │
│                                                             │
│ When OFF, visitors to your root URL (/) will see a         │
│ placeholder page. Views you create can still be accessed   │
│ based on their individual visibility settings.             │
│                                                             │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ Landing Page Message (when homepage is off)             │ │
│ │ ┌─────────────────────────────────────────────────────┐ │ │
│ │ │ This profile is being set up.                       │ │ │
│ │ │                                                     │ │ │
│ │ └─────────────────────────────────────────────────────┘ │ │
│ └─────────────────────────────────────────────────────────┘ │
│                                                             │
│ Your Views:                                                 │
│ • /recruiter (unlisted) - Requires share token              │
│ • /speaking (public) - Always accessible                    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Behavior Matrix:**

| Homepage Toggle | View Visibility | Accessible At | Notes |
|-----------------|-----------------|---------------|-------|
| ON | (any) | `/` | Normal homepage behavior |
| OFF | public | `/<slug>` | Direct URL works |
| OFF | unlisted | `/<slug>?token=...` or `/s/<token>` | Token required |
| OFF | password | `/<slug>` (prompts) | Password required |
| OFF | private | (admin only) | Not public |

**Edge Cases:**

| Scenario | Behavior |
|----------|----------|
| Homepage OFF, no views exist | Show landing page at `/` |
| Homepage OFF, public view exists | `/` shows landing; `/<slug>` shows view |
| Index pages (`/posts`, `/talks`) | Follow homepage toggle (hide when OFF) |
| Individual posts/talks (`/posts/slug`) | Respect item's own visibility (not homepage toggle) |
| SEO/robots.txt | Optionally block indexing when homepage OFF |

#### Technical Implementation

**Schema Changes:**
```typescript
// New settings collection or profile extension
interface SiteSettings {
  homepage_enabled: boolean;      // Toggle for public homepage
  landing_page_message?: string;  // Custom message when OFF
  landing_page_cta_url?: string;  // Optional "Request Access" link
  block_indexing_when_private?: boolean;  // robots.txt control
}
```

**Backend Changes:**
- [ ] Add `site_settings` collection (or extend profile)
- [ ] Modify `/api/homepage` to check `homepage_enabled`
- [x] Modify `/api/homepage` to check `homepage_enabled`
- [x] Modify `/api/default-view` to return `homepage_disabled: true` when OFF
- [ ] Fix view data endpoint to show profile regardless of profile visibility when view access is authenticated (unlisted token or password JWT validated)
- [x] Add `/api/site-settings` endpoint for frontend
- [ ] Optionally serve dynamic `robots.txt` based on setting

**Frontend Changes:**
- [ ] Add prominent toggle in Admin Settings page
- [x] Add prominent toggle in Admin Settings page
- [x] Landing page component for when homepage is disabled
- [x] Custom message textarea
- [x] Hide/show `/posts` and `/talks` index pages based on toggle
- [ ] Show "Your views" summary in settings for quick reference

**UX Considerations:**
1. Toggle should be very prominent (top of Settings or Profile page)
2. Clear explanation of what "OFF" means
3. Show list of active views and their accessibility
4. Warn if turning OFF with no shareable views configured
5. Consider "Request Access" flow for landing page (link to email or form)

#### Prerequisites
- Phase 2 complete (view system)
- Phase 3 complete (token management)

#### Risks
- Users may accidentally hide their profile
- Need clear visual feedback on public vs private state
- Index pages (`/posts`, `/talks`) decision affects content discoverability

---

### View-Specific Content Curation (Posts & Talks)

Enable views to show only specific posts/talks rather than all-or-nothing. Currently, enabling the "posts" section shows ALL public posts. This feature allows curating a subset per view.

#### Use Cases

| Persona | View | Posts/Talks Selection |
|---------|------|----------------------|
| Tech Lead | `/engineering` | Only technical blog posts, architecture talks |
| Tech Lead | `/speaking` | All conference talks, exclude internal presentations |
| Career Pivoter | `/ux-designer` | UX case studies, design thinking talks |
| Career Pivoter | `/product` | Product strategy posts, PM-focused talks |

#### Current Behavior

- Views enable/disable entire posts/talks sections
- When enabled, ALL public non-draft items appear
- No way to select specific items for a view
- The `?from=viewSlug` parameter is only for back navigation, not filtering

#### Proposed Behavior

**Default**: All public posts/talks appear (backwards compatible)
**Optional**: Curate specific items per view using the existing section item selection UI

#### Schema Changes

The `sections` JSON in views already supports item selection for other content types:

```typescript
interface ViewSection {
  section: 'posts' | 'talks' | ...;
  enabled: boolean;
  items?: string[];        // Already exists! Just needs UI for posts/talks
  layout?: string;
  itemConfig?: Record<string, ItemConfig>;
}
```

The `items` array is already supported for sections like experience, projects, education. Posts and talks just need:
1. UI to select items in view editor
2. Backend filtering when serving view data

#### Implementation Tasks

**Backend:**
- [ ] Update `/api/view/:slug/data` to filter posts/talks by `items` array when present
- [ ] If `items` is empty/undefined, return all (backwards compatible default)
- [ ] Ensure posts/talks respect the same visibility/draft filters

**Frontend (View Editor):**
- [ ] Add posts selection UI in view editor (same pattern as projects/experience)
- [ ] Show post title, date, draft status in selection list
- [ ] Add talks selection UI in view editor
- [ ] Show talk title, event, date in selection list
- [ ] Drag-drop reordering within selected items

**UX Flow:**
1. User creates/edits view
2. Enables "Posts" section
3. Clicks section to expand
4. Sees list of all posts with checkboxes (like projects/experience)
5. Selects specific posts for this view
6. Saves → only selected posts appear on public view
7. Empty selection = show all (default behavior preserved)

#### Edge Cases

| Scenario | Behavior |
|----------|----------|
| View has posts section enabled, no items selected | Show all public non-draft posts |
| View has posts section enabled, 3 items selected | Show only those 3 posts |
| View has posts section enabled, selected post becomes draft | Hide from view (respects draft status) |
| View has posts section enabled, selected post deleted | Silently removed from view items list |
| Index page `/posts` | Shows ALL public posts (not view-scoped) |
| Individual post `/posts/slug` | Shows if visibility allows (not view-scoped) |

#### Navigation Behavior

When navigating FROM a view TO a post/talk, the `?from=viewSlug` parameter enables proper back navigation:

```
/my-view → /posts/my-post?from=my-view → Back button → /my-view
```

This already works with the current implementation.

#### Prerequisites
- Phase 2 complete (view section selection UI exists)
- Posts/talks collections have stable schemas

#### Risks
- Users may not realize empty selection means "show all"
- Need clear UI indicator: "Showing all posts" vs "Showing 3 selected posts"

---

### Self-Hosting Improvements

#### OAuth via Environment Variables (Priority)

Enable OAuth configuration without accessing PocketBase admin UI:

```env
# Google OAuth
GOOGLE_CLIENT_ID=your-client-id
GOOGLE_CLIENT_SECRET=your-client-secret

# GitHub OAuth
GITHUB_CLIENT_ID=your-client-id
GITHUB_CLIENT_SECRET=your-client-secret
```

**Implementation:**
- [x] Read OAuth credentials from environment variables on startup
- [x] Programmatically configure PocketBase auth providers
- [x] Update login page to fetch available providers dynamically
- [x] Only show OAuth buttons for configured providers
- [x] Show password login as primary when no OAuth configured
- [x] Add to `.env.example` with documentation
- [x] Seed/dev flow prompts for Google/GitHub/both and persists selections into `.env`

**Benefits:**
- End users never need to access PocketBase admin UI
- All configuration via environment variables / docker-compose
- Enables Unraid template with OAuth fields
- Clean "Facet" branded experience throughout

#### Distribution & Templates
- [ ] One-line install script
- [ ] Docker Compose with Caddy reverse proxy
- [ ] Kubernetes Helm chart
- [ ] Unraid Community Apps template

### Integrations
- Webhook notifications
- RSS feed for posts
- iCal export for talks
- Google Analytics (opt-in)

### Content Types
- Awards & honors section
- Publications section
- Testimonials/references
- Custom sections (user-defined)

### Collaboration
- Read-only share for proofreaders
- Suggestion mode (propose edits)
- (Note: This is NOT multi-user; it's controlled sharing)

---

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2025-12-31 | Phase 0 focus on stability | Foundation must be solid before features |
| 2025-12-31 | No time estimates | Quality over speed; single-owner app |
| 2025-12-31 | Content completeness before views | Need pages to link to before view improvements |
| 2025-12-31 | Theming after core features | Premature optimization; default theme is sufficient |
| 2025-12-31 | Phase 1 complete - certifications added | All core content types now have public display and admin CRUD |
| 2025-12-31 | Admin CRUD pages complete | All admin routes now functional: experience, projects, education, skills |
| 2025-12-31 | Phase 3 complete - token management UI | Full token list, create, copy URL, revoke, status badges, usage stats |
| 2025-12-31 | Phase 2.2 item-level overrides complete | Career pivoters can present same job differently per view; overrides stored in sections JSON |
| 2025-12-31 | Phase 4.2 print stylesheet complete | Browser-based PDF via print is sufficient; server-side PDF deferred |
| 2025-12-31 | Phase 9.2 accessibility audit complete | Skip link, aria attributes, screen reader support added; 0 svelte-check warnings |
| 2025-12-31 | Admin loading pattern standardized | All admin pages use simple `onMount(loadData)` pattern; layout handles auth gating. Fixes Codespaces race conditions. |
| 2026-01-01 | Phase 2.2 drag-drop reordering complete | svelte-dnd-action integrated for section and item reordering; section order preserved in view config and respected in public rendering |
| 2026-01-01 | Phase 6 redesigned as Visual Layout System | Phased approach: (A) per-section layout presets, (B) live preview, (C) section widths/columns, (D) WYSIWYG. Curated layouts prevent bad design; inspired by SharePoint but simpler. |
| 2026-01-01 | Phase 4 redesigned as two-tier Export & Print | Simple Print (browser, works now) + AI Print (sends view to AI, returns optimized markdown, Pandoc converts to DOCX/PDF). Stored in view_exports collection. |
| 2026-01-01 | OAuth config via env vars prioritized | End users should never see PocketBase; all config via environment variables. Login page should dynamically show only configured auth methods. Enables Unraid template distribution. |
| 2026-01-01 | Phase 1.5 added for content discovery | Posts and talks are buried at bottom of profile with no navigation. Adding: profile nav tabs, index pages (/posts, /talks), and individual talk pages (/talks/[slug]). View limiting already works via sections config. |
| 2026-01-01 | Phase 4.4 data export complete | JSON and YAML export via /api/export endpoint. Admin-only, downloads full profile data for backup/migration. Media files and import deferred. |
| 2026-01-01 | Phase 6.5 accent color design finalized | Curated palette approach (6 colors) instead of freeform picker. Maintains design guardrails while enabling personalization. Colors: Sky, Indigo, Emerald, Rose, Amber, Slate. Uses CSS custom properties for runtime theming. |
| 2026-01-01 | Phase 6.5 accent color implementation complete | Full implementation: migration, color constants, Admin Settings UI with color swatches, live preview, CSS custom properties injection. Works in light/dark modes. |
| 2026-01-01 | Phase 6.6 per-view theming complete | Views can now override global accent color. View editor has color selector with "Use global" option. Preview pane reflects view-specific color in real-time. |

---

*This roadmap is a living document. Update it as priorities evolve.*
