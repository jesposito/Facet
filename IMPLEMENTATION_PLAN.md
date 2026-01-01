# Me.yaml Implementation Plan

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
| 12. Print & Export | 🟡 Partial | Print stylesheet complete, data export deferred |

---

## Remaining Work

### High Priority
(None - core functionality complete)

### Medium Priority
1. ~~Drag-drop section/item reordering for views (Phase 2.2 continued)~~ ✅ Complete
2. Media library (Phase 7)
3. Additional frontend tests

### Low Priority
4. SETUP.md and UPGRADE.md documentation
5. AI provider mock tests
6. Integration tests
7. View access log / audit logging (Phase 8)
8. Data export (JSON/YAML) - Phase 4.3

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
