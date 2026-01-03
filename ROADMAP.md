# Facet Roadmap

**Last Updated:** 2026-01-03

This roadmap reflects current implementation status and planned work, ordered chronologically by phase. Completed items remain for context; upcoming items are listed under each phase.

---

## Current Status Snapshot
- ✅ Rebrand complete; branding, assets, and metadata reflect Facet.
- ✅ Core flows: views, share tokens/passwords, GitHub import, AI enrichment (optional), admin CRUD, public pages, print stylesheet.
- ✅ View editor with overrides/reordering; per-view theming; accent colors; media library with orphan detection and cleanup.
- ✅ Media optimization (thumb/srcset) live on posts/projects/homepage; view membership badges in admin lists.
- ✅ External media embeds complete: uploads, external links, public rendering on projects/posts/talks, bulk delete.
- ✅ SEO & Error UX complete: custom 404/500 pages, canonical URLs, comprehensive Open Graph/Twitter Cards, JSON-LD, sitemap, robots.txt.
- ✅ E2E Testing: Playwright test suite with 90%+ coverage of public APIs, SEO features, error pages, and media management (25 tests, 96% pass rate).
- ✅ Security audit complete: Full codebase audit documented in [SECURITY_AUDIT.md](docs/SECURITY_AUDIT.md) (1 HIGH, 3 MEDIUM, 2 LOW severity issues) with prioritized remediation roadmap.
- ✅ Critical security fixes: XSS prevention (DOMPurify sanitization) and path traversal protection (11-layer validation with symlink detection) implemented and tested.
- ✅ Contact protection & social links (Phase 11): Complete with contact_methods collection, admin CRUD, per-view visibility, and 4-tier protection levels.
- ✅ AI Writing Assistant (Phase 12): Complete with 5 tone options, critique mode, mobile-responsive, integrated across all content forms.
- 🔜 Planned: Resume upload & AI parsing, Security headers, debug logging cleanup, 2FA, audit logging hooks, Performance tuning, Demo Mode (Phase 10).

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

## Phase 7: Media Management (✅ Complete)
- 7.1 Media library: ✅ `/admin/media` listing, filters, search, delete; orphan detection
- 7.2 Image optimization: ✅ thumbnails + responsive srcsets for posts/projects/homepage
- 7.3 Cleanup UX: ✅ orphan detection + storage usage stats + bulk delete endpoint
- 7.4 External media: ✅ link-based entries (URL/title/mime/thumbnail) listed alongside uploads; deletion supported; media_refs on projects/posts/talks
- 7.5 Public rendering: ✅ Projects, Posts, and Talks pages render media_refs (YouTube, Vimeo, images, videos, link cards)
- 7.6 Upload mirroring: ✅ Uploaded files automatically mirrored to external_media for unified media_refs
- ℹ️ Media stability note: `/api/media` depends on file fields + `external_media`; run migrations or reseed (`rm -rf pb_data && SEED_DATA=dev make seed-dev`) after schema changes; see docs/MEDIA.md for details.

## Phase 8: Security & Audit (🟡 In Progress)
- ✅ **Security Audit Complete** - Full codebase audit documented in [SECURITY_AUDIT.md](docs/SECURITY_AUDIT.md)
- ✅ **XSS Prevention** - DOMPurify sanitization with iframe whitelisting implemented and tested
- ✅ **Path Traversal Protection** - Complete rewrite with 11-layer validation, symlink detection, defense-in-depth
- ✅ **Security Test Suite** - Comprehensive tests for XSS, path traversal, input validation (tests/security.spec.ts)
- ✅ Audit logs database schema prepared (migration ready)
- ✅ HTTPS enforcement check (warns in production)
- 🔜 **Remaining fixes:**
  - Remove debug logging from production code (deferred - lower priority)
  - Re-enable security headers (CSP, X-Frame-Options, etc.)
- 🔜 **Planned:**
  - Audit log implementation (hooks)
  - Security headers (CSP, Permissions Policy)
  - 2FA (TOTP + backup codes)
  - Session listing/revoke/expiry

## Phase 9: Polish & Performance (✅ Complete)
- ✅ SEO: JSON-LD, sitemap, robots.txt, canonical URLs, Open Graph/Twitter Cards
- ✅ Error UX: custom 404/500 with self-deprecating humor and SVG illustrations
- 🔜 Performance/Lighthouse tuning: lazy loading, bundle/db optimization (planned)

## Phase 10: Demo & Showcase Mode (🔜 Planned)
**Purpose:** Production-safe demo to highlight value when not in dev.
- Admin dashboard toggle (off by default, non-dev only)
- Persona: well-known, multifaceted fictional character (e.g., The Doctor) with 5 curated views mixing overlapping/specific content (tokens, password view, custom CSS, AI print, media, GitHub sample)
- Enabling: snapshot user data, seed demo dataset; clear status copy
- Disabling: restore snapshot, clean demo data, restore prior settings
- Optional telemetry hook on toggle respecting analytics settings

---

## Phase 11: Contact Protection & Social Links (✅ Complete)
**Purpose:** Granular per-view contact control with anti-scraping protection
- ✅ **Phase 1 (Week 1): Foundation**
  - ✅ Create `contact_methods` collection with view-specific visibility
  - ✅ Implement CSS obfuscation and click-to-reveal components
  - ✅ Contact methods admin page with full CRUD
  - ✅ Per-view visibility controls
  - ✅ Protection level selector (none/obfuscation/click-to-reveal/captcha)
  - ✅ Public rendering in views with ContactMethodsList component
- 🔜 **Phase 2 (Future): Advanced Protection**
  - Add robots.txt blocking AI crawlers (GPTBot, ClaudeBot, etc.)
  - Rate limiting for contact reveals (10/10min per device)
  - Cloudflare Turnstile integration for captcha level
  - Device fingerprinting
  - Analytics dashboard for reveal attempts
  - Honeypot detection

**Features:**
- Multiple contact types: email, phone, LinkedIn, GitHub, Twitter, WhatsApp, etc.
- 4-tier protection: none, CSS obfuscation, click-to-reveal, Turnstile CAPTCHA
- Full WCAG 2.1 AA accessibility compliance
- See [CONTACT_PROTECTION.md](docs/CONTACT_PROTECTION.md) for complete spec

---

## Phase 12: AI Writing Assistant (✅ Complete)
**Purpose:** Intelligent content rewriting and feedback across all text fields
- ✅ **Multi-tone rewriting:** Executive, Professional, Technical, Conversational, Creative
- ✅ **Critique mode:** Inline feedback with [bracketed suggestions]
- ✅ **Anti-AI guidelines:** Strict rules to avoid AI-sounding language (no "leverage", "delve", em-dashes, etc.)
- ✅ **Integrated everywhere:** Experience, Projects, Profile, Education, Posts, Talks
- ✅ **Mobile-responsive:** Optimized for all screen sizes
- ✅ **Context-aware:** Uses form fields (title, company, etc.) for better results

**Features:**
- 5 distinct writing tones with specific style guidelines
- Critique mode returns original text with inline `[feedback in brackets]`
- Preview modal with side-by-side comparison
- Works with OpenAI, Anthropic, and Ollama providers
- Comprehensive documentation in [AI_WRITING_ASSISTANT.md](docs/AI_WRITING_ASSISTANT.md)

---

## Cross-Cutting Backlog
- **High Priority:**
  - Testing: ✅ E2E infrastructure complete (25 Playwright tests covering public APIs, SEO, error pages, media, admin flows); 🔜 GitHub/AI provider mocks, additional coverage
  - Theme system extensions (light/dark, presets)
  - 🔜 **Resume Upload & AI Parsing:** Upload PDF/DOCX resumes, use AI to extract and populate experience/education/skills into a new view
- **Medium Priority:**
  - Import/sync: scheduled GitHub refresh, additional sources (LinkedIn/JSON Resume/Credly)
  - Custom section layouts (grids/compact), deferred view warnings, section titles/layout options
  - Security: audit log, headers, 2FA, session management
- **Low Priority:**
  - Performance/SEO/Error UX: as listed in phases 8–9
  - Content extensions: awards/publications/testimonials/custom sections; collaboration modes (read-only/suggestion) remain single-user

## Integrations
- ✅ RSS feed for posts
- ✅ iCal export for talks
- ✅ Google Analytics (opt-in)
- 🔜 Webhook notifications

## Decision Log
(unchanged; see historical entries below)
