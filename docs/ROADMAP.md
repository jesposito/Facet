# Facet Roadmap

**Last Updated:** 2026-01-05

This roadmap reflects current implementation status and planned work, ordered chronologically by phase. Completed items remain for context; upcoming items are listed under each phase.

---

## Current Status Snapshot
- ✅ Rebrand complete; branding, assets, and metadata reflect Facet.
- ✅ Core flows: views, share tokens/passwords, GitHub import, AI enrichment (optional), admin CRUD, public pages, print stylesheet.
- ✅ View editor with overrides/reordering; per-view theming; accent colors; media library with orphan detection and cleanup.
- ✅ Media optimization (thumb/srcset) live on posts/projects/homepage; view membership badges in admin lists.
- ✅ External media embeds complete: uploads, external links, public rendering on projects/posts/talks, bulk delete.
- ✅ SEO & Error UX complete: custom 404/500 pages, canonical URLs, comprehensive Open Graph/Twitter Cards, JSON-LD, sitemap, robots.txt.
- ✅ E2E Testing: Playwright test suite with 100% pass rate on public tests (12/12), 25+ total tests covering public APIs, SEO, error pages, media, admin flows, security (96% overall pass rate).
- ✅ Security review complete: Full security review completed with all identified issues addressed.
- ✅ Critical security fixes: XSS prevention (DOMPurify sanitization) and path traversal protection (11-layer validation with symlink detection) implemented and tested.
- ✅ Contact protection & social links (Phase 11): Complete with contact_methods collection, admin CRUD, per-view visibility, and 4-tier protection levels.
- ✅ AI Writing Assistant (Phase 12): Complete with 5 tone options, critique mode, mobile-responsive, integrated across all content forms.
- ✅ AI Resume Generation (Phase 4): Complete with PDF/DOCX export, multiple formats/styles, AI provider integration.
- ✅ README rewrite: Comprehensive, user-focused documentation for visitors, site owners, and developers with security highlights and accurate feature descriptions.
- ✅ docker-compose.yml enhancement: Extensively commented with Unraid-specific guidance, troubleshooting, and backup instructions.
- ✅ **Demo Mode (Phase 10):** Demo toggle in admin panel with The Doctor's hilarious profile showcasing all features. Data backup/restore when toggling on/off.
- ✅ **Demo Media System (Phase 14):** Profile avatar, project covers, and blog post covers with professional SVG graphics (60KB total). Demo mode now visually complete.
- ✅ **First-Run Experience (Phase 13):** Welcome page, feature highlights, demo integration, Unraid Community Apps template, comprehensive SETUP.md.
- ✅ **Resume Upload & AI Parsing (Phase 15):** Upload PDF/DOCX resumes, AI extraction to Facet data, smart deduplication, file storage with hash-based duplicate prevention.
- 🔜 **Next Up:** Security headers, debug logging cleanup, 2FA, audit logging hooks, Performance tuning.

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

## Phase 4: Export & Print System (✅ Complete)
- ✅ Print stylesheet + print button on public views
- ✅ JSON/YAML export endpoint `/api/export` (admin)
- ✅ AI print/resume generation: Full implementation with PDF/DOCX output, multiple styles, AI provider integration
  - Backend: `/api/view/{slug}/generate` endpoint
  - Frontend: AI Resume modal with format/style/length options
  - Streaming support and error handling
  - Works with OpenAI, Anthropic, and Ollama

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

## Phase 8: Security & Hardening (✅ Core Complete, 🔜 Enhancements Planned)
- ✅ **Security Review Complete** - Full security review completed with all critical issues addressed
- ✅ **XSS Prevention** - DOMPurify sanitization with iframe whitelisting implemented and tested
- ✅ **Path Traversal Protection** - Complete rewrite with 11-layer validation, symlink detection, defense-in-depth
- ✅ **Security Test Suite** - Comprehensive tests for XSS, path traversal, input validation (tests/security.spec.ts)
- ✅ Audit logs database schema prepared (migration ready)
- ✅ HTTPS enforcement check (warns in production)
- 🔜 **Planned Enhancements:**
  - Security headers (CSP, X-Frame-Options, Permissions Policy)
  - Audit log implementation (hooks for admin actions)
  - 2FA (TOTP + backup codes)
  - Session listing/revoke/expiry
  - Remove debug logging from production code

## Phase 9: Polish & Performance (✅ Complete)
- ✅ SEO: JSON-LD, sitemap, robots.txt, canonical URLs, Open Graph/Twitter Cards
- ✅ Error UX: custom 404/500 with self-deprecating humor and SVG illustrations
- 🔜 Performance/Lighthouse tuning: lazy loading, bundle/db optimization (planned)

## Phase 10: Demo & Showcase Mode (✅ Complete)
**Purpose:** One-click demo showing all Facet features with hilarious content.

**Implemented:** Simplified, better approach than original plan!
- ✅ **The Doctor's Profile:** Laugh-out-loud funny demo showcasing EVERY feature
- ✅ **One-Click Demo Login:** `/api/demo/login` endpoint + welcome page button
- ✅ **No Data Pollution:** Demo uses separate account, user's data untouched
- ✅ **Feature Showcase:** Experience, Projects, Skills, Education, Certs, Custom View

### Demo Content Strategy
**Profile:** The Doctor
- Name: "The Doctor"
- Headline: "Time Lord, Adventurer, Earth's Protector (Sometimes)"
- Location: "The TARDIS (usually parked near Earth)"
- Summary: Witty, self-aware description of being a time-traveling problem solver

**Views (6+ Different Facets):**
1. **Default/Public** - "The Official Record"
   - Formal, professional, downplayed
   - "Freelance Consultant specializing in Impossible Problems"
   - Shows: basic experience, selected projects, professional photo

2. **Recruiter** - "For Earth's Defense"
   - Emphasis on leadership, crisis management, team building
   - 900+ years of experience (with humor about age discrimination)
   - Skills: Crisis Management, Alien Diplomacy, Improvisation
   - Password-protected (password: "jiggery-pokery")

3. **Conference** - "Speaking Circuit"
   - All talks and presentations
   - "Fixing Time Paradoxes" (TechConf 2024)
   - "Why You Shouldn't Cross Your Own Timeline" (DevCon)
   - Shows media embeds, slides, videos

4. **Consulting** - "The Real Resume"
   - Unlisted (share token required)
   - Case studies: Stopped Dalek invasion, resolved Cyberman uprising
   - Client testimonials (funny quotes from companions)

5. **Personal/Companions** - "For Friends"
   - Password-protected (password: "allonsy")
   - Hobbies, interests, fish custard recipes
   - Less formal, more personality
   - Custom CSS for fun styling

6. **Academic** - "The Scholar"
   - Education section (Prydonian Academy, etc.)
   - Certifications in temporal mechanics
   - Published papers on wibbly-wobbly timey-wimey stuff

**Content Examples:**

**Experience:**
- "Time Lord Academy" (900+ years ago - present)
- "Freelance Problem Solver" (various timelines)
- Each with bullets showing feature richness

**Projects:**
- "TARDIS Redesign" (GitHub import example)
- "Sonic Screwdriver v14" (with tech stack, media)
- "Defeating the Daleks" (case study format)

**Posts:**
- "Why Time Travel Isn't All It's Cracked Up To Be"
- "10 Things I've Learned in 900 Years" (with AI writing in different tones)
- Show tags, cover images, markdown formatting

**Talks:**
- "Time and Relative Dimensions in Space" (iCal export example)
- With slides URL, video embed, event details

**Awards:**
- "Saved Earth (Again)" - Earth Defense Force
- "Best Use of a Sonic Screwdriver" - Maker Faire 2023

### Feature Showcase
Every view demonstrates:
- ✅ Different visibility levels (public, unlisted, password, private)
- ✅ Custom theming and accent colors per view
- ✅ Share tokens (with expiration, use limits)
- ✅ Media library (images, YouTube embeds, external links)
- ✅ GitHub import (fictional TARDIS project)
- ✅ AI Writing Assistant samples (show different tones on same content)
- ✅ Contact protection (different levels per view)
- ✅ Custom CSS (one view has playful styling)
- ✅ RSS feed and iCal export
- ✅ Drag-and-drop reordering
- ✅ Field locking on imported projects

### Implementation Details
- **Demo Mode Toggle** in admin dashboard (off by default)
- **Data Preservation:**
  - On enable: Snapshot current user data before loading demo
  - On disable: Restore user's original data exactly as it was
  - If user edits during demo: preserve those edits OR restore to empty (user choice)
- **Backend API:**
  - `/api/demo/enable` - Loads Doctor persona
  - `/api/demo/disable` - Restores user data
  - Database backup/restore mechanism (export to JSON, store in temporary table)
- **UI Indicators:**
  - Banner at top of admin when demo mode active
  - "Exploring Demo Mode" message
  - Clear "Exit Demo Mode" button
- **Seed Data File:** `backend/seeds/demo_doctor.json`
  - Complete profile with all views, projects, posts, talks
  - Can be updated/improved without code changes

### Tone
- Self-aware and playful (knows it's a demo)
- Shows off features without being salesy
- Easter eggs for Doctor Who fans
- Professional enough to be useful as example
- Funny enough to be memorable

**✅ What Was Actually Built:** Demo mode toggle at the top of the admin panel (in AdminHeader). Toggle on to replace your data with The Doctor's hilarious profile showcasing all features. Your original data is backed up and restored when you toggle off (or you can keep the demo data as your starting point). See [backend/hooks/demo.go](backend/hooks/demo.go) and [AdminHeader.svelte](frontend/src/components/admin/AdminHeader.svelte).

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
- See [CONTACT_PROTECTION.md](CONTACT_PROTECTION.md) for complete spec

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
- Comprehensive documentation in [AI_WRITING_ASSISTANT.md](AI_WRITING_ASSISTANT.md)

---

## Phase 13: First-Run Experience & Unraid Support (🟡 Partially Complete)
**Purpose:** Make installation and onboarding seamless for all users, especially Unraid community.

### First-Run Welcome Page
- ✅ **Welcome screen** when no profile exists
  - ✅ Engaging introduction to Facet
  - ✅ Brief explanation of what users can do
  - ✅ "Try Demo" button for one-click access to The Doctor's profile
  - ✅ "Sign In to Build Your Own" as secondary CTA
  - ✅ Feature highlights with icons
- 🔜 **First login enhancements** (deferred)
  - Detect first login with default password (`changeme123`)
  - Modal prompt to change password immediately
  - Cannot access admin dashboard until password is changed
  - Smooth redirect to admin after password update

### Unraid Community Support
- ✅ **Enhanced docker-compose.yml**
  - ✅ Extensive inline comments explaining every environment variable
  - ✅ Unraid-specific guidance throughout
  - ✅ Complete troubleshooting section
  - ✅ Backup/restore instructions
- ✅ **Unraid Community Apps Template** (XML)
  - ✅ Template for Unraid Community Applications store (`unraid/facet-template.xml`)
  - ✅ Pre-filled smart defaults for typical Unraid setups
  - ✅ Helpful descriptions for each field in the WebUI
  - ✅ Automatic DATA_PATH mapping to `/mnt/user/appdata/facet`
- ✅ **Enhanced .env.example**
  - ✅ Dedicated Unraid configuration section
  - ✅ Example configurations for common setups
  - ✅ Clear "Required vs Optional" sections
  - ✅ Seed data modes documented (dev, minimal, unset)
- ✅ **Comprehensive SETUP.md updates**
  - ✅ Dedicated Unraid section with step-by-step instructions
  - ✅ Cloudflare Tunnel + Unraid walkthrough
  - ✅ Reverse proxy configuration to preserve share tokens
  - ✅ Backup strategies and troubleshooting guide

### Reverse Proxy & Token Handling
- **Documentation improvements**
  - Verify TRUST_PROXY correctly handles share tokens
  - Cloudflare-specific settings and gotchas
  - Nginx/Traefik/Swag configuration examples
  - Debug instructions for broken share links behind proxies
- **Testing**
  - E2E tests for share tokens behind simulated proxy
  - Verify X-Forwarded-Proto and X-Forwarded-Host handling
  - Test with real Cloudflare Tunnel setup

---

## Phase 14: Demo Media System (✅ Complete)
**Purpose:** Add visual richness to demo mode with images and media

Demo mode now includes professional SVG graphics that showcase the media library features and make The Doctor's profile visually complete.

**✅ Implemented Features:**
- **Demo Assets Directory:** `/backend/seeds/demo_assets/` with themed SVG graphics (60KB total)
- **File Copying Mechanism:** `loadDemoAsset()` function loads and attaches files when demo mode is enabled
- **PocketBase Integration:** Files properly attached using `filesystem.NewFileFromBytes()` API
- **Professional Graphics:** Custom SVG designs with TARDIS blue theme:
  - Profile avatar (circular "TD" design with time vortex pattern)
  - 3 project covers (TARDIS Redesign, Sonic Screwdriver, Defeating Daleks)
  - 4 blog post covers (tech stack, paradox prevention, time travel, wisdom themes)
  - 2 media gallery images (constellation map, vortex energy)
- **Automatic Cleanup:** Files are part of demo_* records, automatically removed on demo restore
- **Lightweight:** Total asset size only 60KB (well under 10MB target)

**Benefits Delivered:**
- Demo mode now visually showcases media library features
- Profile avatar appears in header and views
- Project covers make portfolio section engaging
- Blog post covers demonstrate cover image functionality
- Professional appearance that shows what a complete profile looks like

**Implementation Details:**
- SVG files stored in `/backend/seeds/demo_assets/{profile,projects,posts,media}/`
- `loadDemoAsset(path)` helper function in [demo.go](../backend/hooks/demo.go)
- Files attached during demo enable in `loadDemoDataIntoShadowTables()`
- No cleanup code needed - files auto-delete with demo_* records
- All graphics are original SVG designs, no licensing concerns

---

## Phase 15: Resume Upload & AI Parsing (✅ Complete)
**Purpose:** Allow users to upload existing resumes and automatically populate their Facet profile using AI

Users can now upload their PDF or DOCX resumes and have AI automatically extract all professional information, eliminating manual data entry.

**✅ Implemented Features:**
- **File Upload Endpoint:** `/api/resume/upload` with multipart form support
- **File Format Support:**
  - PDF parsing using `go-fitz` v1.24.15 (reliable text extraction)
  - DOCX parsing using `go-docx` v0.1.1 with XML fallback for malformed files
  - 5MB file size limit
  - File type validation (PDF, DOCX only)
- **AI Parsing Integration:**
  - Structured prompts for OpenAI, Anthropic, and Ollama
  - Extracts: experience, education, skills, projects, certifications, awards, talks
  - 8192 max_tokens to handle complex resumes
  - Confidence scores and parsing warnings
  - Error handling with user-friendly messages
- **Smart Deduplication System:**
  - Skills: Always dedupe across all imports (case-insensitive)
  - Experience/Projects: Dedupe within same file only (enables faceted resume views)
  - Education/Certifications/Awards: Always dedupe universally
  - Fixed critical bug: All queries had swapped limit/offset parameters (returned 0 results)
- **File Storage & Tracking:**
  - `resume_imports` collection stores uploaded files with SHA256 hashing
  - Duplicate detection with 5-minute prevention window
  - Import session tracking with `import_session_id` and `import_filename` fields
  - Files accessible for future media gallery display
- **Database Migrations:**
  - 1736074800: Add import tracking fields to all resume-related collections
  - 1736074900: Create `resume_imports` collection with file storage
  - 1736075000: Add `resume_import_id` relation field for cleanup
- **UX Improvements:**
  - Clear error messages with expandable technical details
  - Import summary showing counts of created/skipped records
  - AI configuration guidance (links to Admin → Settings → AI)

**Benefits Delivered:**
- Fastest way to populate Facet profile from existing resume
- No manual data entry for users with existing resumes
- Intelligent deduplication prevents duplicates across multiple imports
- Supports "faceted resume views" (same person, multiple resume versions)
- File storage enables future media gallery integration

**Technical Highlights:**
- New services: `services/resume_parser.go` (610 lines) for AI parsing logic
- New hooks: `hooks/resume_upload.go` (944 lines) for upload endpoint
- Extensive debugging with Empirica cognitive OS to fix deduplication bugs
- Comprehensive testing with multiple resume formats (PDF, DOCX, complex layouts)
- Documentation: RESUME_UPLOAD_DESIGN.md, TESTING_RESUME_UPLOAD.md, EMPIRICA_GUIDE.md

**Bug Fixes During Implementation:**
1. Critical: Swapped limit/offset parameters in all deduplication queries (returned 0 results every time)
2. PocketBase filter syntax: `:lower` modifier doesn't exist (switched to in-memory `strings.EqualFold()`)
3. Missing migration field: `import_filename` field missing from projects collection
4. AI response truncation: Increased max_tokens from 2048 to 8192
5. DOCX parsing panics: Added XML fallback for malformed files
6. Unique constraint errors: Update existing resume_imports record instead of creating duplicate

---

## Cross-Cutting Backlog
- **High Priority:**
  - Testing: ✅ E2E infrastructure complete (25 Playwright tests covering public APIs, SEO, error pages, media, admin flows); 🔜 GitHub/AI provider mocks, additional coverage
  - Theme system extensions (light/dark, presets)
  - ✅ **Resume Upload & AI Parsing (Reverse Direction):** Upload existing PDF/DOCX resumes, use AI to extract and populate experience/education/skills
    - Complete: AI resume generation (Facet → PDF/DOCX) + resume upload (existing resume → Facet data). Both directions now supported.
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
