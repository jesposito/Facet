# Me.yaml Roadmap

**Last Updated:** 2026-01-01

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

#### 4.2 AI Print (New Feature)

AI-powered document generation that creates polished, professionally formatted resumes.

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

- [ ] Add `view_exports` collection via migration
- [ ] Create resume prompt template (stored in backend)
- [ ] Add `/api/view/{slug}/generate` endpoint
- [ ] Integrate [Pandoc Docker image](https://github.com/pandoc/dockerfiles) or binary
- [ ] Add reference DOCX template for consistent styling
- [ ] Add "Generate Resume" button in view editor
- [ ] Add generation config modal (target role, style, length)
- [ ] Add download buttons for generated files
- [ ] Add "Regenerate" button with spinner
- [ ] Show generation timestamp and AI provider used

**UX Flow:**

1. User edits view, configures sections/overrides
2. Clicks "Generate Resume" in view editor header
3. Modal appears with options:
   - Target role (text input, optional)
   - Style: Chronological / Functional / Hybrid
   - Length: One page / Two pages / Full
   - AI Provider: (dropdown of configured providers)
4. Clicks "Generate"
5. Loading state shows progress
6. On success, download buttons appear (PDF, DOCX)
7. Files also accessible from public view page (if visibility allows)

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

#### 4.4 Data Export

Export all data for backup or migration.

- [ ] Export all data as JSON
- [ ] Export as YAML (human-readable backup)
- [ ] Include uploaded files in ZIP archive
- [ ] Import from backup (restore)

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
- [ ] Mobile preview mode (preview shown at mobile width) — Deferred

**Implementation Details:**
- `ViewPreview.svelte` component reuses public section components
- Reactive updates via Svelte props (no debouncing needed)
- Preview rendered in same page (not iframe) for simplicity
- Responsive layout: side-by-side on desktop, stacked on mobile
- Preview scales down content for compact display

#### 6.3 Section Width & Columns (Phase C - Advanced)

Enable sections to share horizontal space (side-by-side layouts).

**Width Options:**
- `full` - 100% width (current default)
- `half` - 50% width (pairs with another half)
- `third` - 33% width (triplets)

**Behavior:**
- Consecutive sections with compatible widths render side-by-side
- CSS Grid handles responsive collapse (side-by-side on desktop, stacked on mobile)
- Visual indicator in editor shows which sections will pair

**Schema Addition:**
```typescript
interface ViewSection {
  // ... existing fields
  width?: 'full' | 'half' | 'third';  // NEW
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

#### 6.5 Accent Color
- [ ] Color picker in admin settings
- [ ] Applied via CSS custom properties
- [ ] Affects buttons, links, badges, highlights

#### 6.6 Theme Presets
- [ ] Bundled themes: Minimal, Professional, Creative
- [ ] One-click apply (sets colors, fonts, spacing)
- [ ] Reset to default option

#### 6.7 Custom CSS (Power Users)
- [ ] Admin textarea for custom CSS
- [ ] Scoped to public pages only (not admin)
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
- [ ] Read OAuth credentials from environment variables on startup
- [ ] Programmatically configure PocketBase auth providers
- [ ] Add `/api/auth/providers` endpoint to expose enabled methods
- [ ] Update login page to fetch available providers dynamically
- [ ] Only show OAuth buttons for configured providers
- [ ] Show password login as primary when no OAuth configured
- [ ] Add to `.env.example` with documentation

**Benefits:**
- End users never need to access PocketBase admin UI
- All configuration via environment variables / docker-compose
- Enables Unraid template with OAuth fields
- Clean "Me.yaml" branded experience throughout

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

---

*This roadmap is a living document. Update it as priorities evolve.*
