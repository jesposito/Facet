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
- [ ] Full accessibility audit (a11y)

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
  - [ ] Section selector with drag ordering (deferred to Phase 2.2)
  - [x] Item picker per section
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

---

## Remaining Work

### High Priority
1. **Item-level overrides for views** (Phase 2.2) - Enable per-view customization of job titles, descriptions, bullets
2. Full accessibility audit

### Medium Priority
3. Drag-drop section/item reordering for views (Phase 2.2)
4. Media library (Phase 7)
5. Additional frontend tests

### Low Priority
6. SETUP.md and UPGRADE.md documentation
7. AI provider mock tests
8. Integration tests
9. View access log / audit logging (Phase 8)

---

## Phase 2.2: Item-Level Overrides (Detailed)

This phase enables audience-specific framing of the same content. Key use case: career pivoters who need to present the same job differently to different audiences.

### Backend Changes
- [ ] Update view data endpoint to merge itemConfig.overrides with source records
- [ ] Add validation for overridable fields per collection type

### Frontend Changes
- [ ] Add "Edit for this view" button on each item in view editor
- [ ] Create override editor modal with:
  - [ ] Original value display (read-only)
  - [ ] Override input field
  - [ ] "Use original" / "Override" toggle per field
  - [ ] Reset button
- [ ] Add override indicator badges to items in view editor
- [ ] Update view preview to show overridden values

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
