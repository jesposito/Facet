# Facet Roadmap

**Last Updated:** 2026-01-02

This roadmap reflects current implementation status and planned work, ordered chronologically by phase. Completed items remain for context; upcoming items are listed under each phase.

---

## Current Status Snapshot
- ✅ Rebrand complete; branding, assets, and metadata reflect Facet.
- ✅ Core flows: views, share tokens/passwords, GitHub import, AI enrichment (optional), admin CRUD, public pages, print stylesheet.
- ✅ View editor with overrides/reordering; per-view theming; accent colors; media library with orphan detection and cleanup.
- ✅ Media optimization (thumb/srcset) live on posts/projects/homepage; view membership badges in admin lists.
- 🟡 In progress/up next: external media embeds, storage usage polish (bulk stats in UI done), testing backlog.
- 🔜 Planned: Security/Audit (Phase 8), Performance/SEO polish (Phase 9), Demo Mode toggle/persona (Phase 10).

---

## Phase 0: Foundation Stabilization (✅ Complete)
**Purpose:** Solid, secure base.
- Core routing: `/`, `/[slug]`, `/s/<token>`
- Views with visibility controls; share tokens; password-protected views (JWT)
- GitHub import pipeline; optional AI enrichment
- Admin dashboard CRUD for profile, experience, projects, education, skills, posts, talks, certs, awards
- Rate limiting on sensitive endpoints; reserved slug protection

## Phase 1: Content Completeness (✅ Complete)
**Purpose:** All core content types with public pages.
- Projects detail `/projects/<slug>` (meta tags, media gallery)
- Posts/blog `/posts/<slug>` (markdown, tags, prev/next, cover)
- Talks section (public display, embeds, slides)
- Certifications (issuer grouping, expiry badges)

## Phase 1.5: Discovery & Navigation (✅ Complete)
**Purpose:** Make content discoverable.
- Index pages `/posts`, `/talks`; profile nav tabs
- Slugs and detail routes for posts/talks; back-navigation fixes
- Open Graph/SEO basics for content pages

## Phase 2: View System Enhancement (✅ Complete)
**Purpose:** Powerful, curated views.
- View editor create/edit pages; per-section toggles and item selection
- Drag/drop section & item reordering; overrides per item; hero/CTA overrides
- Default view management; per-view theming/accent color; preview pane
- Minimal analytics (view count, last accessed)

## Phase 3: Share Token Management (✅ Complete)
- `/admin/tokens` full CRUD with usage stats, status badges, copy URL
- Visibility and draft filters respected on shared views

## Phase 4: Export & Print System (🟡 Partial)
- ✅ Print stylesheet + print button on public views
- ✅ JSON/YAML export endpoint `/api/export` (admin)
- 🟡 AI print/resume flow: implemented but still being polished (provider selection, error handling, fonts)
- Outstanding: export metadata audit, DOCX/PDF parity checks

## Phase 5: Import System Expansion (🟡 Partial)
- ✅ GitHub import proposals/review flow
- 🟡 Scheduled/cron refresh: planned
- 🟡 Additional sources (LinkedIn/JSON Resume/Credly): planned

## Phase 6: Visual Layout & Theming (✅ Complete)
- Admin sidebar grouped with categories/collapse
- Accent color system with curated palette; per-view overrides
- Custom CSS support; live preview in settings
- View previews in editor

## Phase 7: Media Management (🟡 Partially Complete)
- 7.1 Media library: ✅ `/admin/media` listing, filters, search, delete; orphan detection
- 7.2 Image optimization: ✅ thumbnails + responsive srcsets for posts/projects/homepage
- 7.3 Cleanup UX: ✅ orphan detection + bulk delete + storage usage stats
- 7.4 External media: 🔜 embed external URLs, YouTube/Vimeo thumbs, previews

## Phase 8: Security & Audit (🔜 Planned)
- Audit log of admin actions/share token/password attempts
- Security headers (CSP, Permissions Policy, CORS hardening)
- 2FA (TOTP + backup codes)
- Session listing/revoke/expiry

## Phase 9: Polish & Performance (🔜 Planned)
- Performance/Lighthouse tuning, lazy loading, bundle/db optimization
- SEO: JSON-LD, sitemap, robots.txt, canonical URLs
- Error UX: custom 404/500, error boundaries, friendly messages

## Phase 10: Demo & Showcase Mode (🔜 Planned)
**Purpose:** Production-safe demo to highlight value when not in dev.
- Admin dashboard toggle (off by default, non-dev only)
- Persona: well-known, multifaceted fictional character (e.g., The Doctor) with 5 curated views mixing overlapping/specific content (tokens, password view, custom CSS, AI print, media, GitHub sample)
- Enabling: snapshot user data, seed demo dataset; clear status copy
- Disabling: restore snapshot, clean demo data, restore prior settings
- Optional telemetry hook on toggle respecting analytics settings

---

## Cross-Cutting Backlog
- Testing: frontend/component + integration tests; GitHub/AI provider mocks
- Theme system extensions (light/dark, presets), custom section layouts (grids/compact), deferred view warnings, section titles/layout options
- Import/sync: scheduled GitHub refresh, additional sources (LinkedIn/JSON Resume/Credly)
- Media: image optimization, storage usage, bulk orphan delete, external embeds
- Security: audit log, headers, 2FA, session management
- Performance/SEO/Error UX: as listed in phases 8–9
- Content extensions: awards/publications/testimonials/custom sections; collaboration modes (read-only/suggestion) remain single-user

## Integrations
- ✅ RSS feed for posts
- ✅ iCal export for talks
- ✅ Google Analytics (opt-in)
- 🔜 Webhook notifications

## Decision Log
(unchanged; see historical entries below)
