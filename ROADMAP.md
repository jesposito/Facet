# Me.yaml Roadmap

**Last Updated:** 2025-12-31

This roadmap outlines the feature development plan for Me.yaml, organized into logical phases. Each phase is independently valuable and builds toward a complete personal profile platform.

**Important**: This roadmap contains no time estimates. Each phase represents a coherent set of features, not a sprint or deadline. Phases should be completed in order, as later phases depend on earlier ones.

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
- [ ] Preview pane showing live result — Deferred

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
- [ ] Preview of how homepage will look — Deferred to 2.2

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

## Phase 4: Export & Print (Partial)

**Purpose**: Enable offline access and traditional resume formats.

### Features

#### 4.1 Resume PDF Generation (Deferred)
- [ ] Server-side PDF generation from view content — Deferred (browser print sufficient)
- [x] Clean, ATS-friendly layout — Via print stylesheet
- [x] Include/exclude sections based on view config — Via view system
- [ ] Download button on admin — Deferred

#### 4.2 Print Stylesheet ✅ Complete
- [x] Optimized CSS for printing
- [x] Page breaks at section boundaries
- [x] Hide navigation and UI controls
- [x] Print button on public pages
- [x] ATS-friendly typography (serif body, sans-serif headers)
- [x] Force light mode colors
- [x] Display URLs after links
- [x] Proper page margins

#### 4.3 Data Export (Deferred)
- [ ] Export all data as JSON — Deferred
- [ ] Export as YAML (for backup) — Deferred
- [ ] Include uploaded files in archive — Deferred

#### 4.4 View Snapshot (Deferred)
- [ ] Generate static HTML of a view — Deferred
- [ ] Self-contained (inline CSS/images) — Deferred
- [ ] Useful for offline sharing — Deferred

### Prerequisites
- Phase 3 complete ✅

### Completed
- Print stylesheet enables browser-based PDF generation via Print (Ctrl+P / Cmd+P)
- Print button added to homepage and all view pages
- Clean, professional output suitable for resumes

### Deferred Items
Server-side PDF generation and data export are deferred as browser print provides sufficient functionality for the primary use case (resume generation). These may be revisited based on user feedback.

### Risks
- ~~PDF generation may require headless browser or Go library~~ — Mitigated by browser print
- Static snapshot may break with complex layouts

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

## Phase 6: Theming & Customization

**Purpose**: Allow visual customization without code changes.

### Features

#### 6.1 Color Themes
- [ ] Light/dark mode toggle
- [ ] Accent color picker
- [ ] Preview in admin

#### 6.2 Layout Options
- [ ] Section layout presets
- [ ] Hero image position options
- [ ] Avatar placement options

#### 6.3 Custom CSS
- [ ] Admin textarea for custom CSS
- [ ] Scoped to public pages only
- [ ] Syntax validation

#### 6.4 Theme Presets
- [ ] Bundled themes (minimal, professional, creative)
- [ ] One-click apply
- [ ] Reset to default

### Prerequisites
- Phase 5 complete

### Risks
- Custom CSS can break layout
- Need good preview system

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

### Self-Hosting Improvements
- One-line install script
- Docker Compose with Caddy reverse proxy
- Kubernetes Helm chart
- Unraid app template

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

---

*This roadmap is a living document. Update it as priorities evolve.*
