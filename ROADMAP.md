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

#### 2.1 View Editor Overhaul
- [ ] Drag-and-drop section ordering
- [ ] Per-section item selection with checkboxes
- [ ] Preview pane showing live result
- [ ] Item reordering within sections

#### 2.2 Section Customization
- [ ] Custom section headings per view
- [ ] Show/hide section titles
- [ ] Section layout options (list, grid, compact)

#### 2.3 Default View Management
- [ ] Clear UI for setting default view
- [ ] Warning when changing default
- [ ] Preview of how homepage will look

#### 2.4 View Analytics (Minimal)
- [ ] View count per view (opt-in)
- [ ] Last accessed timestamp
- [ ] No PII collected

### Prerequisites
- Phase 1 complete

### Risks
- Drag-drop complexity; may need library (svelte-dnd-action)
- View config schema changes require migration

---

## Phase 3: Share Token Management

**Purpose**: Full control over share tokens with admin UI.

### Features

#### 3.1 Token Management Page
- [ ] Route: `/admin/tokens`
- [ ] List all tokens with status, usage, expiry
- [ ] Create new token (name, expiry, max uses)
- [ ] Copy token URL to clipboard
- [ ] Revoke/delete tokens

#### 3.2 Token Analytics
- [ ] Use count display
- [ ] Last used timestamp
- [ ] Usage history (recent accesses)

#### 3.3 Batch Operations
- [ ] Revoke all tokens for a view
- [ ] Expire all tokens older than X days
- [ ] Export token list (for auditing)

#### 3.4 Token QR Codes
- [ ] Generate QR code for share URL
- [ ] Download as PNG
- [ ] Useful for physical sharing (business cards, posters)

### Prerequisites
- Phase 2 complete (views are stable)

### Risks
- QR generation may need external library
- Usage history requires new audit collection

---

## Phase 4: Export & Print

**Purpose**: Enable offline access and traditional resume formats.

### Features

#### 4.1 Resume PDF Generation
- [ ] Generate PDF from view content
- [ ] Clean, ATS-friendly layout
- [ ] Include/exclude sections based on view config
- [ ] Download button on admin

#### 4.2 Print Stylesheet
- [ ] Optimized CSS for printing
- [ ] Page breaks at section boundaries
- [ ] Hide navigation, show content
- [ ] Print button on public pages

#### 4.3 Data Export
- [ ] Export all data as JSON
- [ ] Export as YAML (for backup)
- [ ] Include uploaded files in archive

#### 4.4 View Snapshot
- [ ] Generate static HTML of a view
- [ ] Self-contained (inline CSS/images)
- [ ] Useful for offline sharing

### Prerequisites
- Phase 3 complete

### Risks
- PDF generation may require headless browser or Go library
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

#### 5.4 Source Management UI
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

#### 9.2 Accessibility Audit
- [ ] Full WCAG 2.1 AA compliance
- [ ] Screen reader testing
- [ ] Keyboard navigation audit

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

---

*This roadmap is a living document. Update it as priorities evolve.*
