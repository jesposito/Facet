# OwnProfile Implementation Plan

## Phase 2: Milestones & Checklists

### Milestone 1: Repository Scaffold
- [x] Initialize Git repository
- [ ] Create directory structure:
  ```
  ownprofile/
  ├── backend/              # Go + PocketBase
  │   ├── main.go
  │   ├── hooks/
  │   ├── services/
  │   └── migrations/
  ├── frontend/             # SvelteKit
  │   ├── src/
  │   │   ├── routes/
  │   │   ├── lib/
  │   │   └── components/
  │   ├── static/
  │   └── package.json
  ├── docker/
  │   ├── Dockerfile
  │   ├── Dockerfile.dev
  │   └── Caddyfile
  ├── docker-compose.yml
  ├── docker-compose.dev.yml
  ├── .env.example
  ├── README.md
  └── Makefile
  ```
- [ ] Create go.mod for backend
- [ ] Create package.json for frontend
- [ ] Create Makefile with common commands
- [ ] Set up linting (golangci-lint, eslint, prettier)

### Milestone 2: Backend - PocketBase Core
- [ ] Set up PocketBase as Go framework
- [ ] Define all collections via migrations
- [ ] Configure OAuth providers (Google, GitHub)
- [ ] Set up collection rules (admin-only for most)
- [ ] Implement encryption service (AES-256-GCM)
- [ ] Add custom `/api/health` endpoint
- [ ] Test collection CRUD via API

### Milestone 3: Backend - Custom Hooks
- [ ] GitHub importer service:
  - [ ] Fetch repo metadata
  - [ ] Fetch README (raw)
  - [ ] Fetch languages
  - [ ] Fetch topics
  - [ ] Create ImportProposal
- [ ] AI enrichment service:
  - [ ] OpenAI provider
  - [ ] Anthropic provider
  - [ ] Ollama provider
  - [ ] Custom endpoint provider
  - [ ] Encryption for stored keys
  - [ ] Test connection endpoint
- [ ] Share token service:
  - [ ] Generate tokens
  - [ ] Validate tokens
  - [ ] Track usage
- [ ] Password protection service:
  - [ ] Hash passwords
  - [ ] Validate passwords
  - [ ] Issue session cookies

### Milestone 4: Frontend - Public Site
- [ ] Set up SvelteKit project
- [ ] Create layout component with:
  - [ ] SEO meta tags
  - [ ] Open Graph tags
  - [ ] Responsive navigation
- [ ] Implement public routes:
  - [ ] `/` - Main profile page
  - [ ] `/v/[slug]` - View page
  - [ ] `/s/[token]` - Share token landing
  - [ ] `/p/[slug]` - Project detail
  - [ ] `/post/[slug]` - Blog post
- [ ] Create public components:
  - [ ] ProfileHero
  - [ ] ExperienceList
  - [ ] ProjectGrid
  - [ ] EducationList
  - [ ] SkillsSection
  - [ ] ContactSection
  - [ ] PasswordPrompt
- [ ] Implement dark/light theme
- [ ] Add loading states and error pages
- [ ] Ensure accessibility (a11y)

### Milestone 5: Frontend - Admin Dashboard
- [ ] Set up admin layout with sidebar
- [ ] Implement OAuth login flow
- [ ] Create admin routes:
  - [ ] `/admin` - Dashboard overview
  - [ ] `/admin/profile` - Edit profile
  - [ ] `/admin/experience` - CRUD experience
  - [ ] `/admin/projects` - CRUD projects
  - [ ] `/admin/education` - CRUD education
  - [ ] `/admin/certifications` - CRUD certs
  - [ ] `/admin/skills` - CRUD skills
  - [ ] `/admin/posts` - CRUD posts
  - [ ] `/admin/talks` - CRUD talks
  - [ ] `/admin/media` - Media library
  - [ ] `/admin/settings` - AI providers
- [ ] Create admin components:
  - [ ] DataTable (sortable, filterable)
  - [ ] FormField (text, textarea, date, file, JSON)
  - [ ] VisibilitySelector
  - [ ] DraftToggle
  - [ ] DragReorder
  - [ ] MediaUploader
  - [ ] MarkdownEditor
  - [ ] Toast notifications

### Milestone 6: Views & Share Tokens
- [ ] Admin UI for views:
  - [ ] `/admin/views` - List views
  - [ ] `/admin/views/new` - Create view
  - [ ] `/admin/views/[id]` - Edit view
  - [ ] Section selector with drag ordering
  - [ ] Item picker per section
  - [ ] Override fields (headline, summary, CTA)
- [ ] Share token management:
  - [ ] Generate token button
  - [ ] Copy shareable URL
  - [ ] Set expiration
  - [ ] Set max uses
  - [ ] Revoke token
  - [ ] View access log
- [ ] Public view rendering:
  - [ ] Apply section filters
  - [ ] Apply item filters
  - [ ] Apply overrides
  - [ ] Handle password views

### Milestone 7: GitHub Importer UI
- [ ] `/admin/sources` - List sources
- [ ] `/admin/import` - Import wizard:
  - [ ] Step 1: Enter repo URL or select from list
  - [ ] Step 2: Fetch preview
  - [ ] Step 3: Choose AI enrichment (optional)
  - [ ] Step 4: Review proposal
- [ ] `/admin/review/[id]` - Review UI:
  - [ ] Side-by-side diff view
  - [ ] Per-field controls (apply/ignore/lock/edit)
  - [ ] Apply all / Reject all buttons
  - [ ] Show AI-generated vs fetched labels
- [ ] Refresh button on existing projects
- [ ] Sync status indicators

### Milestone 8: AI Provider Settings
- [ ] `/admin/settings/ai` - AI providers:
  - [ ] Add provider form (type selector)
  - [ ] API key input (write-only display)
  - [ ] Base URL input (for Ollama/custom)
  - [ ] Model selector
  - [ ] Test connection button
  - [ ] Set default provider
  - [ ] Delete provider
- [ ] Enrichment options in import:
  - [ ] Provider selector
  - [ ] Privacy level (full README, summary only, none)
  - [ ] Preview estimated tokens

### Milestone 9: Docker & Deployment
- [ ] Create production Dockerfile:
  - [ ] Multi-stage build
  - [ ] Build Go backend
  - [ ] Build SvelteKit
  - [ ] Copy Caddy config
  - [ ] Final minimal image
- [ ] Create development Dockerfile:
  - [ ] Hot reload for Go
  - [ ] Hot reload for SvelteKit
- [ ] Create docker-compose.yml:
  - [ ] Production config
  - [ ] Volume mounts
  - [ ] Environment variables
- [ ] Create docker-compose.dev.yml:
  - [ ] Development overrides
  - [ ] Source mounts
- [ ] Create .env.example with all vars
- [ ] Test build and run
- [ ] Test behind reverse proxy

### Milestone 10: Documentation & Polish
- [ ] README.md:
  - [ ] Project overview
  - [ ] Quick start
  - [ ] Screenshots/ASCII art
  - [ ] Configuration reference
  - [ ] Reverse proxy examples (NPM, Cloudflare)
- [ ] SETUP.md:
  - [ ] Detailed installation
  - [ ] OAuth setup guide
  - [ ] First run setup
- [ ] UPGRADE.md:
  - [ ] Version upgrade process
  - [ ] Backup procedures
- [ ] DEVELOPMENT.md:
  - [ ] Dev environment setup
  - [ ] Architecture overview
  - [ ] Contributing guide
- [ ] Seed data for demo
- [ ] Final testing pass
- [ ] Performance check

### Milestone 11: Testing
- [ ] Backend tests:
  - [ ] Encryption service tests
  - [ ] Share token validation tests
  - [ ] Password hashing tests
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

## Implementation Order

Execute milestones in this order for optimal progress:

1. **Milestone 1** - Scaffold (foundation)
2. **Milestone 2** - Backend core (database ready)
3. **Milestone 4** - Public frontend (visible progress)
4. **Milestone 5** - Admin frontend (full CRUD)
5. **Milestone 3** - Backend hooks (GitHub/AI)
6. **Milestone 6** - Views & tokens (key feature)
7. **Milestone 7** - Importer UI (GitHub integration)
8. **Milestone 8** - AI settings (enrichment)
9. **Milestone 9** - Docker (deployment)
10. **Milestone 10** - Docs (polish)
11. **Milestone 11** - Tests (quality assurance)
