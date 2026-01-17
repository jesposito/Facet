# Facet Roadmap

**Last Updated:** 2026-01-17

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
- ✅ **Admin UX Overhaul (Phase 16):** Accessible confirm dialogs, bulk operations across all content types, improved navigation, visibility badges.
- ✅ **Bulk Operations:** Select multiple items → change visibility, delete in bulk. Available on projects, posts, talks, experience, education, skills, certifications, awards.
- ✅ **Custom Domain Support:** Self-hosted architecture supports any domain via reverse proxy (Nginx, Traefik, Cloudflare Tunnel, etc.).
- ✅ **Mobile UX Overhaul (Phase 16.5):** Complete responsive redesign of admin panel - overlay sidebar, touch targets, bottom sheet modals, form stacking, overflow prevention.
- ✅ **UX Improvements (Phase 17.1-17.2):** Setup Wizard for new users, Contextual Help on all admin pages.
- ✅ **Quick Share to Social (Phase 18.1):** Native Web Share API with social platform fallbacks (LinkedIn, Twitter/X, Reddit, Email).
- 🔜 **Next Up:** Phase 18.2 View Analytics Dashboard, Phase 18.3 QR Codes, Phase 19 Developer Platform.

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
- ✅ Resume upload & AI parsing (PDF/DOCX to Facet data)
- 🔜 Scheduled/cron refresh: planned
- 🔜 Additional sources: LinkedIn, JSON Resume (deferred - see "Tracking Upstream Dependencies")

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

## Phase 8: Security & Hardening (✅ Complete)
- ✅ **Security Review Complete** - Full security review completed with all critical issues addressed
- ✅ **XSS Prevention** - DOMPurify sanitization with iframe whitelisting implemented and tested
- ✅ **Path Traversal Protection** - Complete rewrite with 11-layer validation, symlink detection, defense-in-depth
- ✅ **Security Test Suite** - Comprehensive tests for XSS, path traversal, input validation (tests/security.spec.ts)
- ✅ **Password Security** - First-time password change enforcement, CLI reset tool, bcrypt validation
- ✅ HTTPS enforcement check (warns in production)
- ✅ **Security Headers** - Comprehensive headers implemented via Caddy (docker/Caddyfile) and PocketBase built-in middleware
- ✅ **HTTP Timeouts** - All AI and GitHub API calls have proper timeouts (30-120s)
- ✅ **Rate Limiting** - Proper mutex synchronization, no race conditions
- ✅ **File Validation** - 5MB limits with MIME type validation
- ✅ **SQL Injection Protection** - All queries use parameterized filters (`{:slug}`, `{:id}`)

## Phase 9: Polish & Performance (✅ Complete)
- ✅ SEO: JSON-LD, sitemap, robots.txt, canonical URLs, Open Graph/Twitter Cards
- ✅ Error UX: custom 404/500 with self-deprecating humor and SVG illustrations
- ✅ Dark mode: 1000+ dark mode classes for comprehensive theming
- ✅ Responsive design: Mobile-first with 124 breakpoint usages
- ✅ Accessibility: 195 aria/role attributes across components

## Phase 10: Demo & Showcase Mode (✅ Complete)
**Purpose:** One-click demo showing all Facet features with hilarious content.

**Implemented:**
- ✅ **The Doctor's Profile:** Laugh-out-loud funny demo showcasing EVERY feature
- ✅ **One-Click Demo Toggle:** In admin header, toggle on/off instantly
- ✅ **Data Preservation:** Original data backed up and restored when demo disabled
- ✅ **Feature Showcase:** All content types, views, visibility levels, theming

See [backend/hooks/demo.go](../backend/hooks/demo.go) and [AdminHeader.svelte](../frontend/src/components/admin/AdminHeader.svelte).

---

## Phase 11: Contact Protection & Social Links (✅ Complete)
**Purpose:** Granular per-view contact control with anti-scraping protection
- ✅ Create `contact_methods` collection with view-specific visibility
- ✅ Implement CSS obfuscation and click-to-reveal components
- ✅ Contact methods admin page with full CRUD
- ✅ Per-view visibility controls
- ✅ Protection level selector (none/obfuscation/click-to-reveal/captcha)
- ✅ Public rendering in views with ContactMethodsList component

See [CONTACT_PROTECTION.md](CONTACT_PROTECTION.md) for complete spec.

---

## Phase 12: AI Writing Assistant (✅ Complete)
**Purpose:** Intelligent content rewriting and feedback across all text fields
- ✅ **Multi-tone rewriting:** Executive, Professional, Technical, Conversational, Creative
- ✅ **Critique mode:** Inline feedback with [bracketed suggestions]
- ✅ **Anti-AI guidelines:** Strict rules to avoid AI-sounding language
- ✅ **Integrated everywhere:** Experience, Projects, Profile, Education, Posts, Talks
- ✅ **Mobile-responsive:** Optimized for all screen sizes
- ✅ **Context-aware:** Uses form fields for better results

See [AI_WRITING_ASSISTANT.md](AI_WRITING_ASSISTANT.md) for complete documentation.

---

## Phase 13: First-Run Experience & Unraid Support (✅ Complete)
**Purpose:** Make installation and onboarding seamless for all users.
- ✅ Welcome screen when no profile exists
- ✅ "Try Demo" button for one-click access to The Doctor's profile
- ✅ Unraid Community Apps template
- ✅ Comprehensive SETUP.md with Cloudflare Tunnel walkthrough

---

## Phase 14: Demo Media System (✅ Complete)
**Purpose:** Add visual richness to demo mode with images and media
- ✅ Demo Assets Directory with themed SVG graphics (60KB total)
- ✅ Profile avatar, project covers, blog post covers
- ✅ Automatic cleanup when demo disabled

---

## Phase 15: Resume Upload & AI Parsing (✅ Complete)
**Purpose:** Upload existing resumes and automatically populate Facet profile
- ✅ PDF/DOCX parsing with AI extraction
- ✅ Smart deduplication system
- ✅ File storage and tracking

---

## Phase 16: Admin UX Overhaul (✅ Complete)
**Purpose:** Improve admin experience with modern, accessible UI patterns

**Completed:**
- ✅ **Accessible Confirm Dialogs** - Replaced native `confirm()` with styled, accessible modal dialogs
- ✅ **Bulk Operations** - Select multiple items, change visibility, bulk delete across 8 content types
- ✅ **Visibility Badges** - Clear indicators showing public/unlisted/private status in admin lists
- ✅ **Improved Navigation** - Better admin menu structure and organization
- ✅ **Dialog System** - Consistent modal patterns across the app
- ✅ **View Hero Images** - Fixed display issues in admin and demo mode

**Bug Status (Previously Listed as Phase 16):**
The original Phase 16 bug list has been verified - most items were already fixed:
- ✅ HTTP timeouts - Configured (30-120s on all external calls)
- ✅ Race conditions - Proper mutex usage in rate limiting
- ✅ File validation - 5MB limits with MIME type checks
- ✅ SQL injection - Parameterized queries throughout
- ✅ Error handling - All Save() calls checked (except 1 non-critical stat update)
- ⚠️ Slug uniqueness - DB enforces via unique index, but no suffix on collision (minor)

---

## Phase 16.5: Mobile UX Overhaul (✅ Complete)
**Purpose:** Make the admin panel fully responsive and touch-friendly on mobile devices.

**Implemented:**
- ✅ **Overlay Sidebar:** Converts to drawer overlay on mobile (<1024px), hidden by default, closes on navigation
- ✅ **Touch Targets:** All buttons meet 44px minimum touch target size per Apple/Google guidelines
- ✅ **Form Stacking:** Link inputs and form groups stack vertically on mobile
- ✅ **View Editor Redesign:** Simplified headers, larger drag handles, collapsible preview, reduced padding
- ✅ **Bottom Sheet Modals:** ConfirmDialog and PasswordChangeModal use native-feeling bottom sheets on mobile
- ✅ **Overflow Prevention:** Multi-layer defense with overflow-x: hidden on html/body, min-w-0 on flex items
- ✅ **Responsive Grids:** All grid-cols-2 converted to grid-cols-1 sm:grid-cols-2

**Files Modified:**
- `admin/+layout.svelte` - Mobile detection, sidebar overlay logic
- `AdminSidebar.svelte` - Overlay mode with z-index layering
- `ConfirmDialog.svelte`, `PasswordChangeModal.svelte` - Bottom sheet pattern
- `views/[id]/+page.svelte`, `views/new/+page.svelte` - Mobile-optimized view editor
- 10+ admin pages - Touch targets, form stacking, responsive grids
- `app.css`, `app.html` - Global overflow prevention

See [MOBILE_UX_PLAN.md](MOBILE_UX_PLAN.md) for detailed implementation plan.

---

## Phase 17: UX Improvements (✅ Complete)
**Purpose:** Make Facet easier to understand and use

### 17.1 Guided Setup Wizard (✅ Complete)
**Priority:** High | **Effort:** Medium

First-time users get a 3-step wizard instead of facing 22 admin pages:
- Step 1: Basic profile (name, headline, summary)
- Step 2: Create first facet from templates (Recruiter, Portfolio, Consulting, Speaker)
- Step 3: Review and launch

**Features:**
- Auto-opens for new users (missing profile data or no views)
- Respects password change flow (waits until complete)
- Sidebar auto-refreshes when facet created
- Skip option with permanent dismissal (localStorage)
- Never shows in demo mode

**Files:**
- `frontend/src/lib/stores/setupWizard.ts` - State management, templates
- `frontend/src/components/admin/SetupWizard.svelte` - Main modal
- `frontend/src/components/admin/wizard/Step*.svelte` - Step components
- `frontend/src/routes/admin/+layout.svelte` - Integration

### 17.2 Contextual Help on Admin Pages (✅ Complete)
**Priority:** High | **Effort:** Low

Each admin page now has a collapsible help section explaining:
- What the page does
- Why users would use it
- How it connects to views/facets

**Features:**
- Collapsible `PageHelp` component with localStorage persistence
- Contextual tips for all 16 admin pages
- "Learn more" links to documentation where applicable

**Files:**
- `frontend/src/components/admin/PageHelp.svelte` - Reusable component
- All admin pages updated with contextual help

---

## Phase 18: Sharing & Analytics (🟡 In Progress)
**Purpose:** Make sharing easier and provide insight into profile views

### 18.1 Quick Share to Social (✅ Complete)
**Priority:** High | **Effort:** Low

One-click sharing to major platforms with native OS integration.

**Implementation:**
- `frontend/src/lib/share.ts` - Share utility with Web Share API detection
- `frontend/src/components/shared/ShareButton.svelte` - Reusable component

**Features:**
- **Web Share API** - Native sharing on mobile/Chrome/Safari (75% browser support)
- **Fallback dropdown** - Copy Link, LinkedIn, Twitter/X, Reddit, Email
- **Share token URLs** - When viewing via `/s/[token]`, shares the token URL
- **Unlisted warning** - Shows "This view is unlisted. Share links may expire."
- **Accessibility** - 44px touch targets, keyboard navigation, ARIA labels
- **Copy feedback** - "Copied!" indicator for 1.5 seconds

**Share URL formats:**
- LinkedIn: `https://www.linkedin.com/sharing/share-offsite/?url={url}`
- Twitter: `https://twitter.com/intent/tweet?url={url}&text={title}`
- Reddit: `https://reddit.com/submit?url={url}&title={title}`
- Email: `mailto:?subject={title}&body={text}%0A%0A{url}`

**Where it shows:**
- View pages (`/[slug]`) - Always visible
- Post pages (`/posts/[slug]`) - Only if `visibility === 'public'`
- Project pages (`/projects/[slug]`) - Only if `visibility === 'public'`
- Talk pages (`/talks/[slug]`) - Only if `visibility === 'public'`

**Platform decisions:**
- ✅ Copy Link (primary, always visible)
- ✅ LinkedIn (professional network)
- ✅ Twitter/X (tech community)
- ✅ Reddit (communities)
- ✅ Email (universal)
- ❌ Facebook (broken on iOS Safari)
- ❌ Bluesky (niche, user preference)
- ❌ Instagram (no web share URL)

### 18.2 View Analytics Dashboard
**Priority:** High | **Effort:** Medium

The data already exists (`use_count`, `last_used_at`). Surface it!

**Implementation:**
- New page: `/admin/analytics`
- Dashboard showing:
  - Total views per view (bar chart)
  - Share token usage stats
  - Most viewed content
  - Recent activity timeline
- Use existing PocketBase data, no external tracking
- Privacy-respecting: all data stays local

**Backend:**
- New endpoint: `GET /api/analytics/summary`
- Aggregates from views, share_tokens, and minimal access logs

### 18.3 QR Code for Views
**Priority:** Medium | **Effort:** Low

Generate QR codes for any view or share link.

**Implementation:**
- Use `qrcode` npm package (lightweight, no external deps)
- Add "QR Code" button next to share link
- Modal shows:
  - QR code image (SVG)
  - Download as PNG button
  - Print button

**Use cases:**
- Business cards
- Conference badges
- Print resumes with QR linking to full profile

---

## Phase 19: Developer Platform (🔜 Planned)
**Purpose:** Enable integrations and extensibility

### 19.1 Webhooks
**Priority:** Medium | **Effort:** Medium

Notify external services when events occur.

**Events to support:**
- `view.accessed` - Someone viewed a profile
- `share_token.used` - Share link was used
- `content.published` - New post/project published
- `profile.updated` - Profile changed

**Implementation:**
- New collection: `webhooks` (url, events[], secret, active)
- Admin page: `/admin/settings/webhooks`
- Backend: Hook into PocketBase events, POST to registered URLs
- Include HMAC signature for verification

**Payload example:**
```json
{
  "event": "view.accessed",
  "timestamp": "2026-01-17T14:30:00Z",
  "data": {
    "view_slug": "recruiter",
    "referrer": "linkedin.com"
  }
}
```

### 19.2 Public API
**Priority:** Medium | **Effort:** Medium

Documented REST API for integrations.

**Endpoints to document:**
- `GET /api/view/{slug}/data` - Get view data (respects visibility)
- `GET /api/posts` - List public posts
- `GET /api/projects` - List public projects
- `GET /api/profile` - Get basic profile info

**Implementation:**
- OpenAPI/Swagger spec in `docs/api/`
- Auto-generated from PocketBase schema + custom endpoints
- Rate limiting for public API access
- Optional API key for higher limits

### 19.3 Offline PWA Support
**Priority:** Low | **Effort:** High

Make admin dashboard work offline.

**Implementation:**
- Service worker for caching
- IndexedDB for offline data storage
- Queue mutations when offline
- Sync when back online
- Show "offline" indicator

**Scope:**
- Admin dashboard (view/edit content offline)
- Public pages (view cached profiles)
- Exclude: AI features, GitHub import

---

## Phase 20: Social Proof & Networking (🔜 Future)
**Purpose:** Build credibility through endorsements and easy application

### 20.1 Testimonials/Endorsements System
**Priority:** High | **Effort:** High

Let others vouch for you without needing accounts.

**How it works:**
1. Owner generates endorsement request link (like share tokens)
2. Link goes to simple form: name, relationship, testimonial text
3. Submission stored as "pending" for owner review
4. Owner approves/rejects in admin
5. Approved testimonials appear on designated views

**Implementation:**

**Backend:**
- New collection: `endorsements`
  - `id`, `name`, `email`, `relationship`, `content`, `status` (pending/approved/rejected)
  - `request_token`, `submitted_at`, `approved_at`
  - `views` (which views to show on)
- New collection: `endorsement_requests`
  - `id`, `token`, `expires_at`, `message` (optional intro message)
  - `max_uses`, `use_count`
- Endpoints:
  - `POST /api/endorsement/request` - Generate request link (auth required)
  - `GET /api/endorsement/submit/{token}` - Get request details (no auth)
  - `POST /api/endorsement/submit/{token}` - Submit testimonial (no auth)
  - `GET /api/endorsements` - List for admin (auth required)
  - `PATCH /api/endorsements/{id}` - Approve/reject (auth required)

**Frontend:**
- `/endorse/{token}` - Public submission form (clean, professional)
- `/admin/endorsements` - Review and manage
- `TestimonialsSection.svelte` - Display on public views

**Security:**
- Rate limit submissions (3 per token per hour)
- Spam detection (very short or repetitive content)
- Email notification to owner on new submission
- No account required for endorsers

### 20.2 "Apply with Facet" Button
**Priority:** Medium | **Effort:** Very High

Enable one-click job applications using Facet profile.

**The Challenge:**
This requires the *receiving* side (employers) to integrate. Two approaches:

**Approach A: Embeddable Widget (Simpler)**
```html
<!-- Employer adds to job posting -->
<a href="https://facet.example.com/apply?job=senior-dev&company=acme"
   class="facet-apply-btn">
  Apply with Facet
</a>
```

When clicked:
1. Applicant lands on their Facet instance
2. Selects which view to share
3. Facet generates a time-limited share link
4. Emails the link to the employer (address in URL params)

**Approach B: Facet Network (Complex)**
- Central directory of Facet instances
- Employers register to receive applications
- OAuth handshake between instances
- This is essentially building a job board

**Recommended: Start with Approach A**
- Zero employer integration required
- Works immediately
- Can evolve into Approach B later

**Implementation (Approach A):**
- `/apply` route that shows view selector
- Email template for application
- Include: view link, optional cover note, contact info
- Track sent applications in new `applications` collection

---

## Cross-Cutting Backlog

### High Priority
- ✅ **Guided Setup Wizard** (Phase 17.1) - Complete
- ✅ **Contextual Help** (Phase 17.2) - Complete
- ✅ **Quick Share to Social** (Phase 18.1) - Complete
- 🔜 **View Analytics Dashboard** (Phase 18.2) - Uses existing data

### Medium Priority
- 🔜 **QR Codes** (Phase 18.3) - Quick win for sharing
- 🔜 **Webhooks** (Phase 19.1) - Enable integrations
- 🔜 **Testimonials System** (Phase 20.1) - Social proof

### Lower Priority
- 🔜 **Public API** (Phase 19.2) - Developer platform
- 🔜 **Offline PWA** (Phase 19.3) - Complex, niche use case
- 🔜 **Apply with Facet** (Phase 20.2) - Requires ecosystem adoption
- 🔜 **Theme System** - Pre-built visual themes
- 🔜 **Scheduled Publishing** - Content calendar

### Already Complete (Removed from Backlog)
- ✅ **Guided Setup Wizard** (Phase 17.1) - 3-step onboarding for new users
- ✅ **Contextual Help** (Phase 17.2) - Help text on all admin pages
- ✅ **Quick Share to Social** (Phase 18.1) - Native Web Share API + social fallbacks
- ✅ **Bulk Operations** - Implemented across 8 content types
- ✅ **Custom Domain** - Works via reverse proxy (self-hosted)
- ✅ **Resume Upload & AI Parsing** - Both directions supported
- ✅ **Admin UX** - Dialogs, navigation, visibility badges

---

## Integrations
- ✅ RSS feed for posts
- ✅ iCal export for talks
- ✅ Google Analytics (opt-in)
- ✅ GitHub import
- 🔜 Webhook notifications
- 🔜 Zapier/IFTTT support (via webhooks)

---

## Tracking Upstream Dependencies

### PocketBase TOTP Support
**Status:** Deferred pending native support

- Subscribe to: https://github.com/pocketbase/pocketbase/discussions/1208
- OAuth users already have provider 2FA
- Will implement when PocketBase adds native TOTP

### Import Source Integrations
- **LinkedIn:** API requires partnership (deferred)
- **Credly:** No public API for individuals (deferred)
- **JSON Resume:** Lower priority, resume upload covers use case

---

## Recent Changes Log

### 2026-01-17 (Quick Share to Social - v2.4.0)
- Completed Phase 18.1: Quick Share to Social
- Web Share API integration for native mobile/browser sharing
- Fallback dropdown: Copy Link, LinkedIn, Twitter/X, Reddit, Email
- Share token URL support (shares `/s/[token]` when viewing via token)
- Unlisted warning banner when sharing token URLs
- AI API key info added to setup wizard
- Accessible: 44px touch targets, keyboard nav, ARIA labels

### 2026-01-17 (Setup Wizard - v2.3.0)
- Completed Phase 17.1: Guided Setup Wizard
- 3-step onboarding for new users (profile basics, first facet, launch)
- 4 view templates: Recruiter, Portfolio, Consulting, Speaker
- Password change flow takes priority over wizard
- Sidebar auto-refreshes when wizard creates a facet

### 2026-01-17 (Contextual Help - v2.2.0)
- Completed Phase 17.2: Contextual Help on Admin Pages
- Added PageHelp component with localStorage persistence
- All 16 admin pages now have contextual help

### 2026-01-17 (Mobile UX)
- Completed Phase 16.5: Mobile UX Overhaul
- Admin sidebar converts to overlay drawer on mobile
- Touch targets increased to 44px minimum across all admin pages
- Bottom sheet pattern for modals on mobile
- View editor redesigned for mobile with collapsible preview
- Multi-layer horizontal overflow prevention
- Added auto-tag workflow for automatic version bumps on PR merge

### 2026-01-17
- Updated Phase 16 bug status - verified most were already fixed
- Added Phase 17 (UX Improvements) with Guided Setup Wizard, Contextual Help, Better Empty States
- Added Phase 18 (Sharing & Analytics) with Quick Share, Analytics Dashboard, QR Codes
- Added Phase 19 (Developer Platform) with Webhooks, Public API, Offline PWA
- Added Phase 20 (Social Proof) with Testimonials System, Apply with Facet
- Noted Bulk Operations and Custom Domain as already complete
- Reorganized Cross-Cutting Backlog by priority

### 2026-01-12
- Completed Admin UX overhaul (accessible dialogs, bulk operations, visibility badges)
- Fixed view hero image display issues
- Upgraded to Svelte 5 and Vite 7
