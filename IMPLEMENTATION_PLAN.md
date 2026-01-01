# Facet Implementation Plan

## Phase 2: Milestones & Checklists

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
- [x] GitHub importer service:
  - [x] Fetch repo metadata
  - [x] Fetch README (raw)
  - [x] Fetch languages
  - [x] Fetch topics
  - [x] Create ImportProposal
- [x] AI enrichment service:
  - [x] OpenAI provider
  - [x] Anthropic provider
  - [x] Ollama provider
  - [x] Custom endpoint provider
  - [x] Encryption for stored keys
  - [x] Test connection endpoint
- [x] Share token service:
  - [x] Generate tokens
  - [x] Validate tokens
  - [x] Track usage
- [x] Password protection service:
  - [x] Hash passwords
  - [x] Validate passwords
  - [x] Issue session cookies

### Milestone 4: Frontend - Public Site ✅ Complete
- [x] Set up SvelteKit project
- [x] Create layout component with:
  - [x] SEO meta tags
  - [x] Open Graph tags
  - [x] Responsive navigation
- [x] Implement public routes:
  - [x] `/` - Main profile page
  - [x] `/[slug]` - View page (LinkedIn-style canonical URLs)
  - [x] `/s/[token]` - Share token landing
  - [x] `/projects/[slug]` - Project detail
  - [x] `/posts/[slug]` - Blog post
- [x] Create public components:
  - [x] ProfileHero
  - [x] ExperienceSection
  - [x] ProjectsSection
  - [x] EducationSection
  - [x] SkillsSection
  - [x] TalksSection
  - [x] CertificationsSection
  - [x] PostsSection
  - [x] PasswordPrompt
  - [x] Footer
- [x] Implement dark/light theme (ThemeToggle)
- [x] Add loading states and error pages
- [x] Full accessibility audit (a11y) - Phase 9.2

### Milestone 5: Frontend - Admin Dashboard ✅ Complete
- [x] Set up admin layout with sidebar
- [x] Implement OAuth login flow
- [x] Create admin routes:
  - [x] `/admin` - Dashboard overview
  - [x] `/admin/profile` - Edit profile
  - [x] `/admin/experience` - CRUD experience
  - [x] `/admin/projects` - CRUD projects
  - [x] `/admin/education` - CRUD education
  - [x] `/admin/certifications` - CRUD certs
  - [x] `/admin/skills` - CRUD skills
  - [x] `/admin/posts` - CRUD posts
  - [x] `/admin/talks` - CRUD talks
  - [ ] `/admin/media` - Media library (deferred to Phase 7)
  - [x] `/admin/settings` - AI providers
- [x] Create admin components:
  - [x] AdminHeader
  - [x] AdminSidebar
  - [x] Toast notifications
  - [ ] DataTable (sortable, filterable) - inline in pages
  - [ ] Reusable FormField components - inline in pages
  - [ ] DragReorder - deferred

### Milestone 6: Views & Share Tokens ✅ Complete
- [x] Admin UI for views:
  - [x] `/admin/views` - List views
  - [x] `/admin/views/new` - Create view
  - [x] `/admin/views/[id]` - Edit view
  - [x] Section selector with drag ordering (svelte-dnd-action)
  - [x] Item picker per section with drag reordering
  - [x] Override fields (headline, summary, CTA)
  - [x] Default view management
- [x] Share token management UI:
  - [x] `/admin/tokens` - Full token list page
  - [x] Generate token with name, expiry, max uses
  - [x] Copy shareable URL to clipboard
  - [x] Revoke token with confirmation
  - [x] Token status badges (active, expired, revoked, max uses reached)
  - [x] Usage stats (use count, last used timestamp)
  - [ ] View access log (deferred to Phase 8 - Audit)
- [x] Public view rendering:
  - [x] Apply section filters
  - [x] Apply item filters
  - [x] Apply overrides
  - [x] Handle password views

### Milestone 7: GitHub Importer UI ✅ Complete
- [x] `/admin/import` - Import wizard:
  - [x] Enter repo URL
  - [x] Fetch preview
  - [x] Choose AI enrichment (optional)
  - [x] Create proposal
- [x] `/admin/review/[id]` - Review UI:
  - [x] Field-by-field review
  - [x] Per-field controls (apply/ignore/lock/edit)
  - [x] Apply all / Reject all buttons
  - [x] Show AI-generated vs fetched labels
- [ ] `/admin/sources` - List sources (not separate page)
- [ ] Refresh button on existing projects

### Milestone 8: AI Provider Settings ✅ Complete
- [x] `/admin/settings` - AI providers:
  - [x] Add provider form (type selector)
  - [x] API key input (write-only display)
  - [x] Base URL input (for Ollama/custom)
  - [x] Model selector
  - [x] Test connection button
  - [x] Set default provider
  - [x] Delete provider
- [x] Enrichment options in import:
  - [x] Provider selector
  - [x] Privacy level options
  - [ ] Preview estimated tokens

### Milestone 9: Docker & Deployment ✅ Complete
- [x] Create production Dockerfile:
  - [x] Multi-stage build
  - [x] Build Go backend
  - [x] Build SvelteKit
  - [x] Copy Caddy config
  - [x] Final minimal image
- [x] Create development Dockerfile
- [x] Create docker-compose.yml
- [x] Create docker-compose.dev.yml
- [x] Create .env.example with all vars
- [ ] Test behind reverse proxy (documented)

### Milestone 10: Documentation & Polish (Partial)
- [x] README.md with overview
- [x] DESIGN.md - Comprehensive design document
- [x] ARCHITECTURE.md - Technical architecture
- [x] ROADMAP.md - Feature roadmap
- [ ] SETUP.md - Detailed installation guide
- [ ] UPGRADE.md - Version upgrade process
- [x] DEV.md - Development setup (if exists)
- [x] Seed data for demo
- [ ] Final testing pass
- [ ] Performance check

### Milestone 11: Testing (Partial)
- [x] Backend tests:
  - [x] Encryption service tests (crypto_test.go)
  - [x] Share token validation tests (share_test.go)
  - [x] Rate limiting tests (ratelimit_test.go)
  - [x] Routing tests (routing_test.go)
  - [x] Visibility tests (visibility_test.go)
  - [x] Collection rules tests (rules_test.go)
  - [ ] GitHub API mock tests
  - [ ] AI provider mock tests
- [ ] Frontend tests:
  - [ ] Component unit tests
  - [ ] View access logic tests
  - [ ] Form validation tests
- [ ] Integration tests:
  - [ ] OAuth flow (mocked)
  - [ ] Import pipeline
  - [ ] Review and apply flow

---

## Current Status Summary

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
| 12. Print & Export | 🟡 Partial | Print stylesheet + data export complete, AI print deferred |
| 13. Visual Layout | 🟡 Partial | Layout presets (6.1) + Live preview (6.2) + Section widths (6.3) + Accent color (6.5) + Per-view theming (6.6) complete |

---

## Remaining Work

### High Priority
(None - core functionality complete)

### Medium Priority
1. ~~Drag-drop section/item reordering for views (Phase 2.2 continued)~~ ✅ Complete
2. ~~Live preview pane for view editor (Phase 6.2)~~ ✅ Complete
3. Media library (Phase 7)
4. Additional frontend tests

### Low Priority
5. SETUP.md and UPGRADE.md documentation
6. AI provider mock tests
7. Integration tests
8. View access log / audit logging (Phase 8)
9. ~~Data export (JSON/YAML) - Phase 4.4~~ ✅ Complete
10. ~~Section width & columns (Phase 6.3)~~ ✅ Complete
11. Mobile preview mode (Phase 6.2.2)

---

## Phase 2.2: Drag-Drop Section & Item Reordering ✅ Complete

This phase enables users to customize the order of sections and items within views using drag-and-drop.

### Implementation
- [x] Install `svelte-dnd-action` library for Svelte drag-drop
- [x] Update view editor (`/admin/views/[id]`) with section drag handles
- [x] Update view editor with item drag handles within expanded sections
- [x] Update view create page (`/admin/views/new`) with same functionality
- [x] Backend returns `section_order` array in view data API response
- [x] Public view renders sections dynamically based on order

### Technical Details
- Section order stored as array position in `sections` JSON field (no schema change needed)
- Item order within sections preserved by array position in `items` field
- Backend iterates sections in order and returns `section_order` array
- Frontend uses `{#each effectiveSectionOrder}` to render sections dynamically
- Drag handles visible on all draggable elements
- Smooth flip animation (200ms) during drag operations

### Files Changed
- `frontend/src/routes/admin/views/[id]/+page.svelte` - Section and item drag-drop
- `frontend/src/routes/admin/views/new/+page.svelte` - Section and item drag-drop
- `frontend/src/routes/[slug=slug]/+page.svelte` - Dynamic section rendering
- `frontend/src/routes/[slug=slug]/+page.server.ts` - Pass sectionOrder to page
- `backend/hooks/view.go` - Return section_order in API response
- `frontend/package.json` - Added svelte-dnd-action dependency

---

## Phase 2.2: Item-Level Overrides ✅ Complete

This phase enables audience-specific framing of the same content. Key use case: career pivoters who need to present the same job differently to different audiences.

### Backend Changes
- [x] Update view data endpoint to merge itemConfig.overrides with source records
- [x] Add validation for overridable fields per collection type
- [x] `serializeRecordsWithOverrides()` function applies overrides
- [x] `getOverridableFields()` defines allowed fields per section

### Frontend Changes
- [x] Add "Customize" button on each selected item in view editor
- [x] Create override editor modal with:
  - [x] Original value display (collapsible)
  - [x] Override input fields (text, textarea, array)
  - [x] "Reset to original" button per field
- [x] Add override indicator badges showing count
- [x] Public view automatically shows overridden values (backend applies)

### Overridable Fields
| Collection | Fields |
|------------|--------|
| Experience | title, description, bullets |
| Projects | title, summary, description |
| Education | degree, field, description |
| Talks | title, description |

### Not Overridable (Factual Data)
- Company names, dates, institutions
- URLs, credential IDs
- Skills, certifications (include/exclude only)

---

## Phase 4.2: Print Stylesheet ✅ Complete

This phase enables browser-based PDF generation for resumes and profile printing.

### Implementation
- [x] Comprehensive print stylesheet in `app.css`
- [x] Print button on homepage (`/`)
- [x] Print button on all view pages (`/[slug]`)
- [x] Hidden UI elements during print (theme toggle, print button, CTA banner)

### Print Optimizations
- [x] ATS-friendly typography (Georgia for body, Helvetica for headers)
- [x] Force light mode colors for all elements
- [x] Page breaks avoided inside sections and articles
- [x] Proper page margins (0.75in, letter size)
- [x] URLs displayed after link text
- [x] Reduced padding and margins for space efficiency
- [x] Cards rendered without shadows, with subtle borders
- [x] Hero section simplified (no background images/gradients)

### Usage
Users can generate PDFs by:
1. Navigate to any public view or homepage
2. Click the print button (printer icon) in the top-right
3. Use browser's Print dialog (Ctrl+P / Cmd+P)
4. Select "Save as PDF" as the destination

### Current Status
Phase 4.1 (Simple Print) is complete. See ROADMAP.md Phase 4 for the full two-tier design.

---

## Phase 4.3: AI Print (Planned)

This phase adds AI-powered document generation for professional resumes.

### Overview
- Send view data to AI for content optimization
- AI returns formatted markdown
- Pandoc converts to DOCX and PDF
- Generated files stored in `view_exports` collection

### Key Components
- [ ] `view_exports` collection (migration)
- [ ] Resume prompt template
- [ ] `/api/view/{slug}/generate` endpoint
- [ ] Pandoc integration (Docker or binary)
- [ ] Generate Resume button in view editor
- [ ] Download buttons for generated files

### See ROADMAP.md Phase 4.2 for full specification.

---

## Phase 9.2: Accessibility Audit ✅ Complete

This phase improves accessibility to ensure the application is usable by all users, including those using assistive technologies.

### CSS Utilities Added
- [x] `.sr-only` - Visually hidden but accessible to screen readers
- [x] `.skip-link` - Skip navigation link visible on focus

### Skip Navigation
- [x] Skip link added to main layout (visible on focus)
- [x] `#main-content` ID added to main content areas

### ARIA Improvements
- [x] `aria-hidden="true"` on all decorative SVG icons
- [x] `aria-label` on interactive elements (theme toggle, sidebar toggle, print button)
- [x] `aria-current="page"` on active navigation items
- [x] `aria-expanded` and `aria-controls` on sidebar toggle
- [x] `aria-label` on navigation landmark regions
- [x] `aria-live="polite"` on toast notification container
- [x] `role="status"` on loading indicators

### Component Updates
| Component | Changes |
|-----------|---------|
| AdminSidebar | aria-current, aria-hidden on icons, sr-only labels when collapsed |
| AdminHeader | aria-expanded, aria-controls, aria-label on toggle |
| ThemeToggle | Dynamic aria-label based on current theme |
| ProfileHero | aria-hidden on decorative elements, aria-label on contact links |
| Toast container | role="region", aria-live for notifications |
| Admin layout | role="status" on loading state |
| Public pages | id="main-content" for skip link target |

### Testing
- [x] `svelte-check` passes with 0 errors and 0 warnings
- [x] Focus styles already present in app.css
- [x] Keyboard navigation works throughout application

---

## Phase 6.1: Per-Section Layout Presets ✅ Complete

This phase enables users to customize how each section renders with curated layout presets.

### Overview
Users can now select different layout styles for each section in the view editor. This provides visual variety without requiring design expertise, using guardrails (curated presets) instead of freeform controls.

### Available Layouts

| Section | Layouts | Default |
|---------|---------|---------|
| Experience | default (cards), timeline, compact | default |
| Projects | grid-3, grid-2, list, featured | grid-3 |
| Education | default, timeline | default |
| Certifications | grouped, grid, timeline | grouped |
| Skills | grouped, cloud, bars, flat | grouped |
| Posts | grid-3, grid-2, list, featured | grid-3 |
| Talks | default, cards, list | default |

### Implementation

#### Backend Changes
- [x] Add `section_layouts` map to `/api/view/{slug}/data` response
- [x] Extract layout from sections JSON for each enabled section
- [x] Add `getDefaultLayout()` helper function

#### Frontend Type Changes
- [x] Add `layout?: SectionLayout` to `ViewSection` interface
- [x] Add `VALID_LAYOUTS` constant with section-to-layouts mapping
- [x] Add `getSectionLayout()` helper function

#### View Editor Changes
- [x] Add layout dropdown to section headers (when section enabled)
- [x] Include layout in saved sections data
- [x] Updated both `/admin/views/[id]` and `/admin/views/new`

#### Section Component Changes
- [x] ExperienceSection: default, timeline, compact variants
- [x] ProjectsSection: grid-3, grid-2, list, featured variants
- [x] SkillsSection: grouped, cloud, bars, flat variants
- [x] EducationSection: default, timeline variants
- [x] CertificationsSection: layout prop added (grouped default)
- [x] PostsSection: grid-3, grid-2, list support
- [x] TalksSection: layout prop added (default)

#### Public View Changes
- [x] Page receives `sectionLayouts` from API
- [x] Pass layout prop to each section component
- [x] Components render appropriate variant based on layout

### Files Changed
- `backend/hooks/view.go` - Add section_layouts to response, getDefaultLayout()
- `frontend/src/lib/pocketbase.ts` - Add layout field, VALID_LAYOUTS constant
- `frontend/src/routes/admin/views/[id]/+page.svelte` - Layout dropdown UI
- `frontend/src/routes/admin/views/new/+page.svelte` - Layout dropdown UI
- `frontend/src/routes/[slug=slug]/+page.svelte` - Pass layouts to sections
- `frontend/src/routes/[slug=slug]/+page.server.ts` - Include sectionLayouts
- `frontend/src/components/public/ExperienceSection.svelte` - 3 layouts
- `frontend/src/components/public/ProjectsSection.svelte` - 4 layouts
- `frontend/src/components/public/SkillsSection.svelte` - 4 layouts
- `frontend/src/components/public/EducationSection.svelte` - 2 layouts
- `frontend/src/components/public/CertificationsSection.svelte` - layout prop
- `frontend/src/components/public/PostsSection.svelte` - layout prop + grid variants
- `frontend/src/components/public/TalksSection.svelte` - layout prop

### Usage
1. Navigate to View Editor (/admin/views/[id])
2. Enable a section (toggle on)
3. Layout dropdown appears next to the expand button
4. Select desired layout preset
5. Save view - layout is applied on public view

---

## Phase 1.5: Content Discovery & Navigation ✅ Complete

This phase improves content discovery by adding navigation tabs and dedicated index pages for posts and talks.

### Overview
Previously, posts and talks were only visible at the bottom of the profile page with no easy way to browse them. This phase adds:
- Profile navigation tabs for quick section jumping
- Dedicated index pages for posts (`/posts`) and talks (`/talks`)
- Individual talk detail pages (`/talks/[slug]`)

### Features Implemented

#### Profile Navigation Tabs
- [x] `ProfileNav.svelte` component with sticky positioning
- [x] Smooth scroll to sections on the same page
- [x] Links to `/posts` and `/talks` index pages
- [x] Active section highlighting based on scroll position
- [x] Horizontal scrolling on mobile for many tabs
- [x] Hidden during print

#### Posts Index Page (`/posts`)
- [x] Grid layout with post cards
- [x] Cover image thumbnails (or gradient placeholder)
- [x] Tag-based filtering
- [x] Links to individual posts
- [x] SEO meta tags

#### Talks Index Page (`/talks`)
- [x] List layout with video thumbnails (YouTube)
- [x] Year-based filtering
- [x] Event, date, and location metadata
- [x] Links to video and slides
- [x] Links to individual talk pages (when slug exists)

#### Individual Talk Pages (`/talks/[slug]`)
- [x] Video embed (YouTube/Vimeo)
- [x] Full description with Markdown support
- [x] Event, date, location metadata
- [x] Links to slides and external video
- [x] Previous/next talk navigation
- [x] Profile context in footer

#### Backend Changes
- [x] Migration to add `slug` field to talks collection
- [x] `/api/talk/{slug}` endpoint for individual talk data
- [x] Previous/next talk navigation in API response

#### Admin UI Updates
- [x] Slug field in talks admin form
- [x] Auto-generate slug from title
- [x] Link to public talk page in list view

### Files Added
- `backend/migrations/1735600005_add_talks_slug.go`
- `frontend/src/components/public/ProfileNav.svelte`
- `frontend/src/routes/posts/+page.server.ts`
- `frontend/src/routes/posts/+page.svelte`
- `frontend/src/routes/talks/+page.server.ts`
- `frontend/src/routes/talks/+page.svelte`
- `frontend/src/routes/talks/[slug]/+page.server.ts`
- `frontend/src/routes/talks/[slug]/+page.svelte`

### Files Modified
- `frontend/src/lib/pocketbase.ts` - Added slug to Talk interface
- `frontend/src/routes/+page.svelte` - Added ProfileNav component
- `frontend/src/routes/admin/talks/+page.svelte` - Added slug field to form
- `backend/hooks/view.go` - Added `/api/talk/{slug}` endpoint

---

## Phase 6.2: Live Preview Pane ✅ Complete

This phase adds a side-by-side live preview in the view editor for immediate visual feedback when customizing views.

### Overview
Users can now see how their view will look while editing, without needing to save and navigate to the public page. The preview updates instantly as changes are made to sections, layouts, items, and overrides.

### Features
- [x] Split-pane layout: editor (~60%) and preview (~40%)
- [x] Preview updates reactively on any editor change
- [x] Preview reuses actual public section components (not mockups)
- [x] Toggle button to hide/show preview for more editor space
- [x] Responsive layout: side-by-side on desktop, stacked on mobile
- [x] Profile data loaded and displayed in preview hero
- [x] Hero overrides (headline, summary) shown in preview
- [x] CTA banner shown when configured
- [x] Section layouts applied in real-time
- [x] Item-level overrides reflected in preview

### Implementation

#### New Component
- `frontend/src/components/admin/ViewPreview.svelte`
  - Reuses all public section components (ExperienceSection, ProjectsSection, etc.)
  - Accepts editor state as props (profile, sections, sectionOrder, sectionItems, etc.)
  - Filters items based on enabled sections and selected items
  - Applies item-level overrides before rendering
  - Scaled-down styling for compact preview display

#### View Editor Changes
- [x] Import ViewPreview component
- [x] Add profile data loading on mount
- [x] Add `showPreview` toggle state (default: true)
- [x] Split layout with editor-pane and preview-pane
- [x] Toggle button with eye icon in header
- [x] Pass all editor state to ViewPreview component

#### View Create Page Changes
- [x] Same changes applied to `/admin/views/new`
- [x] Preview works during initial view creation
- [x] All sections enabled by default for new views

### Files Changed
- `frontend/src/components/admin/ViewPreview.svelte` (new)
- `frontend/src/routes/admin/views/[id]/+page.svelte` - Split layout, preview toggle, profile loading
- `frontend/src/routes/admin/views/new/+page.svelte` - Split layout, preview toggle, profile loading

### Technical Details
- Preview rendered in same page (not iframe) for simplicity
- Svelte reactive bindings update preview instantly (no debouncing needed)
- CSS scales down components for compact display
- Responsive: preview appears above editor on mobile screens
- Profile avatar URL resolved for preview display

### Usage
1. Navigate to View Editor (/admin/views/[id] or /admin/views/new)
2. Preview appears on the right side (or top on mobile)
3. Make any change: enable/disable sections, change layouts, select items, add overrides
4. See changes reflected immediately in preview
5. Click "Hide/Show Preview" button to toggle preview visibility
6. Click "Open in Tab" to see full-size public view in new tab

---

## Phase 4.4: Data Export ✅ Complete

This phase enables users to export their complete profile data for backup or migration.

### Overview
Users can download their entire profile in JSON or YAML format from the admin settings page. The export includes all content (profile, experience, projects, education, certifications, skills, posts, talks, and views) but excludes sensitive data like password hashes and internal IDs.

### Features
- [x] Export all profile data as JSON
- [x] Export all profile data as YAML
- [x] Download as file with timestamp filename
- [x] Admin-only access (requires authentication)
- [x] Metadata includes version and export timestamp

### Backend Implementation
- `GET /api/export?format=json` - Returns JSON export file
- `GET /api/export?format=yaml` - Returns YAML export file
- Requires authentication via `apis.RequireAuth()` middleware

### Export Schema
```json
{
  "meta": {
    "version": "1.0.0",
    "exported_at": "2026-01-01T12:00:00Z",
    "app": "Me.yaml"
  },
  "profile": { ... },
  "experience": [ ... ],
  "projects": [ ... ],
  "education": [ ... ],
  "certifications": [ ... ],
  "skills": [ ... ],
  "posts": [ ... ],
  "talks": [ ... ],
  "views": [ ... ]
}
```

### Files Added/Modified
- `backend/hooks/export.go` (new) - Export endpoint and data collection
- `backend/hooks/export_test.go` (new) - Unit tests for export functionality
- `backend/main.go` - Register export hooks
- `backend/go.mod` - Added gopkg.in/yaml.v3 dependency
- `frontend/src/routes/admin/settings/+page.svelte` - Export buttons UI

### Security Considerations
- Export requires admin authentication
- Password hashes are stripped from view exports
- Internal record IDs are included for reference but not sensitive
- Files are served with proper Content-Disposition headers for download

### Usage
1. Navigate to Admin > Settings
2. Scroll to "Data Export" section
3. Click "Download YAML" or "Download JSON"
4. File downloads with timestamped filename (e.g., `me-yaml-export-2026-01-01.yaml`)

### Future Enhancements (Deferred)
- [ ] Include uploaded media files in ZIP archive
- [ ] Import/restore from backup file
- [ ] Export specific views only

---

## Phase 6.3: Section Width & Columns ✅ Complete

This phase enables sections to share horizontal space, allowing side-by-side layouts for more compact and professional presentations.

### Overview
Users can now set each section's width (full, half, or third) in the view editor. Consecutive sections with compatible widths render side-by-side on desktop, automatically stacking on mobile for responsive design.

### Features
- [x] Width selector dropdown in view editor (both create and edit)
- [x] Visual width indicator icons (column preview)
- [x] CSS Grid layout on public view pages
- [x] Responsive collapse to full-width on mobile (< 768px)
- [x] Live preview reflects width settings in real-time
- [x] Print stylesheet supports side-by-side layout

### Width Options
| Width | Grid Span | Use Case |
|-------|-----------|----------|
| Full | 6 columns | Default - section takes entire row |
| Half | 3 columns | Side-by-side pairs (Skills + Certifications) |
| Third | 2 columns | Triplets (rarely needed) |

### Implementation

#### Type Changes
- `ViewSection.width?: 'full' | 'half' | 'third'` added to interface
- `VALID_WIDTHS` constant with labels for dropdown
- `SectionWidth` type alias for width values

#### Backend Changes
- Extract `width` from sections JSON in view data endpoint
- Return `section_widths` map in API response
- Default to `"full"` when width not specified

#### View Editor Changes
- Width dropdown appears next to layout dropdown (when section enabled)
- Visual column indicator shows current width setting
- Width saved in sections configuration

#### Public View Changes
- Grid container with 6-column layout
- Sections wrapped in divs with width classes
- CSS handles column spanning and responsive collapse

#### ViewPreview Changes
- Same grid layout as public view
- Width classes applied in real-time preview

### Files Changed
- `frontend/src/lib/pocketbase.ts` - Added SectionWidth type, VALID_WIDTHS
- `frontend/src/routes/admin/views/[id]/+page.svelte` - Width dropdown + indicator
- `frontend/src/routes/admin/views/new/+page.svelte` - Width dropdown + indicator
- `frontend/src/routes/[slug=slug]/+page.svelte` - Grid layout + width classes
- `frontend/src/routes/[slug=slug]/+page.server.ts` - Pass sectionWidths
- `frontend/src/components/admin/ViewPreview.svelte` - Grid layout support
- `backend/hooks/view.go` - Extract and return section_widths

### CSS Grid Structure
```css
.sections-grid {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 1.5rem;
}

.section-full { grid-column: span 6; }
.section-half { grid-column: span 3; }
.section-third { grid-column: span 2; }

@media (max-width: 768px) {
  .section-half, .section-third {
    grid-column: span 6; /* Stack on mobile */
  }
}
```

### Usage
1. Navigate to View Editor (/admin/views/[id])
2. Enable desired sections
3. Select width from dropdown (Full/Half/Third)
4. Observe visual indicator showing column layout
5. Preview pane shows sections side-by-side
6. Save view - layout applies on public page
7. Example: Set Skills to "Half" and Certifications to "Half" for side-by-side display

---

## Phase 6.5: Accent Color (Curated Palette) ✅ Complete

This phase enables users to customize their profile's accent color using a curated 6-color palette, providing personalization while maintaining design guardrails.

### Overview
Users can now select an accent color from the Admin Settings page. The color applies globally to buttons, links, badges, and focus states across the entire profile. Uses CSS custom properties for runtime theming without page reload.

### Curated Color Palette

| Name | Hex | Use Case |
|------|-----|----------|
| **Sky** (default) | `#0ea5e9` | Tech, software, professional |
| **Indigo** | `#6366f1` | Creative, design, consulting |
| **Emerald** | `#10b981` | Finance, sustainability, health |
| **Rose** | `#f43f5e` | Marketing, creative, personal branding |
| **Amber** | `#f59e0b` | Education, construction, energy |
| **Slate** | `#64748b` | Minimal, monochrome, conservative |

### Implementation

#### Backend Changes
- [x] Migration `1735600006_add_accent_color.go` adds `accent_color` field to profile collection
- [x] `/api/homepage` endpoint includes `accent_color` in profile response
- [x] `/api/view/{slug}/data` endpoint includes `accent_color` in profile response

#### Frontend Type Changes
- [x] Added `accent_color` field to `Profile` interface in `pocketbase.ts`
- [x] Created `frontend/src/lib/colors.ts` with:
  - `AccentColor` type
  - `ACCENT_COLORS` constant with full color scales (50-950)
  - `ACCENT_COLOR_LIST` for UI iteration
  - `generateAccentCssVariables()` helper function

#### Tailwind Configuration
- [x] Updated `tailwind.config.js` to use CSS custom properties for primary colors
- [x] All `primary-*` colors now reference `var(--color-primary-*)` variables

#### CSS Variables
- [x] Added default CSS custom properties in `app.css` `:root` selector
- [x] Default values match Sky color palette (existing behavior preserved)

#### Root Layout Changes
- [x] `+layout.svelte` fetches profile accent color on mount
- [x] `applyAccentColor()` function updates CSS custom properties
- [x] Listens for `accent-color-changed` custom event for real-time updates
- [x] Updates `theme-color` meta tag for browser chrome

#### Admin Settings UI
- [x] New "Appearance" section at top of Settings page
- [x] Color swatch selector with visual checkmark on selected color
- [x] Live preview showing button, link, and badge appearance
- [x] Instant save on color selection (no save button needed)

### What Accent Color Affects
- Primary buttons (`.btn-primary`)
- Links and hover states
- Focus outlines (accessibility)
- Badges and tag highlights
- Input focus rings
- Skip link styling

### Files Added
- `backend/migrations/1735600006_add_accent_color.go`
- `frontend/src/lib/colors.ts`

### Files Modified
- `backend/hooks/view.go` - Added accent_color to API responses
- `frontend/src/lib/pocketbase.ts` - Added accent_color to Profile interface
- `frontend/tailwind.config.js` - Changed primary colors to CSS variables
- `frontend/src/app.css` - Added CSS custom properties
- `frontend/src/routes/+layout.svelte` - Added accent color loading and application
- `frontend/src/routes/admin/settings/+page.svelte` - Added Appearance section

### Usage
1. Navigate to Admin > Settings
2. Find the "Appearance" section at the top
3. Click on any color swatch to select it
4. Preview shows how buttons and links will look
5. Color is saved immediately and applies site-wide
6. Changes take effect on all pages without refresh

### Technical Details
- Uses CSS custom properties for zero-flicker runtime theming
- Color is fetched from `/api/homepage` endpoint on initial load
- Admin settings page dispatches `accent-color-changed` event for instant updates
- All 6 colors have full Tailwind-style scales (50-950) for consistent theming
- Works correctly in both light and dark modes

---

## Phase 6.6: Per-View Theming ✅ Complete

This phase enables different views to have different accent colors, allowing users to tailor each view's visual style to its audience.

### Overview
Each view can optionally override the global accent color. This enables:
- **Recruiter view** → Indigo (professional, corporate)
- **Speaking view** → Rose (energetic, memorable)
- **Portfolio view** → Emerald (creative, fresh)
- **Default view** → Uses global profile setting

### Implementation

#### Backend Changes
- [x] Migration `1735600007_add_view_accent_color.go` adds optional `accent_color` field to views collection
- [x] `/api/view/{slug}/data` endpoint includes `accent_color` in response (null = inherit)

#### Frontend Type Changes
- [x] Added `accent_color` field to `View` interface in `pocketbase.ts`

#### View Editor UI
- [x] Accent color selector in Settings section of view editor
- [x] "Use global" option to inherit from profile setting
- [x] Color swatches for override selection
- [x] Visual feedback showing current selection
- [x] Descriptive text indicating inheritance vs override

#### Public View Rendering
- [x] `+page.svelte` (homepage) applies view accent color if present
- [x] `[slug=slug]/+page.svelte` applies view accent color if present
- [x] View accent color takes priority over profile accent color

#### Preview Pane
- [x] ViewPreview component accepts accentColor prop
- [x] Preview applies view-specific accent color via CSS custom properties
- [x] Real-time updates as user changes accent color in editor

### Files Added
- `backend/migrations/1735600007_add_view_accent_color.go`

### Files Modified
- `backend/hooks/view.go` - Added accent_color to view data response
- `frontend/src/lib/pocketbase.ts` - Added accent_color to View interface
- `frontend/src/routes/+page.server.ts` - Added accent_color to view data
- `frontend/src/routes/[slug=slug]/+page.server.ts` - Added accent_color to view data
- `frontend/src/routes/+page.svelte` - Apply view accent color on homepage
- `frontend/src/routes/[slug=slug]/+page.svelte` - Apply view accent color on public views
- `frontend/src/routes/admin/views/[id]/+page.svelte` - Added accent color selector UI
- `frontend/src/components/admin/ViewPreview.svelte` - Added accent color preview support

### Usage
1. Navigate to Admin > Views > [Edit View]
2. Find "Accent Color" in the Settings section
3. Click "Use global" to inherit from profile, or select a color to override
4. Preview pane shows the accent color in real-time
5. Save the view - accent color applies on public view
6. Each view can have its own accent color independent of others

### Technical Details
- View accent color stored as nullable field (null = inherit from profile)
- Priority: View accent color > Profile accent color > Default (sky)
- CSS custom properties applied via inline styles in preview
- Public pages apply accent color via onMount() hook
