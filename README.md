# Facet

**Every side of you. Your way.**

A self-hosted personal profile platform that puts you in control. Own your data, choose what each audience sees, and skip the tracking.

Think LinkedIn meets personal portfolio, except you hold all the cards: the data lives in your SQLite database, you decide who sees what, and analytics are off by default (opt-in only if you want them).

---

## What Is This For?

**If you want to:**
- Show recruiters your employment history without broadcasting it to your boss
- Share conference-specific work at events without exposing client projects
- Keep a professional presence online without feeding the LinkedIn algorithm
- Import your GitHub projects without copying and pasting 47 README files
- Upload your resume and have AI extract all your experience, skills, and education automatically
- Actually own your professional identity
- Make up fake people like some kind of weirdie 

**Then Facet might be for you.**

---

## Try It Out (Demo Mode)

Not sure if Facet is for you? After signing in, you'll see a **Demo toggle** at the top of the admin panel. Toggle it on to instantly load The Doctor's hilarious profile - a time-traveling alien trying to pass as a normal developer.

**What you'll see:**
- 5 different views showcasing different professional personas
- 4 extensive blog posts with technical humor (~2000 words each)
- Conference talks, projects, work experience, and more
- All features working: views, content types, multiple layouts

**Demo mode features:**
- One-click toggle on/off (top of admin panel)
- Loads comprehensive example content instantly
- Your original data is backed up and restored when you toggle off
- Shows you what a complete Facet profile looks like
- Perfect for exploring features before building your own

Toggle off to restore your original data, or keep the demo data as your starting point.

---

## The 30-Second Version

One Docker container. One command. You get:

- A profile with all the usual sections (experience, projects, education, skills, etc.)
- Multiple "views" (different versions of your profile for different audiences)
- Privacy controls (public, unlisted with share links, password-protected, or private)
- GitHub import that pulls in your repos (with optional AI summaries)
- RSS feed for your blog posts, iCal export for your talks
- No tracking by default, no ads, no engagement metrics (analytics are opt-in)
- Your data in SQLite, your uploads in a folder, both easy to backup

One port exposed. One volume to backup. That's it.

---

## Why Facet Is Secure

Your professional identity deserves better than hoping a platform protects it. Facet takes security seriously:

**You Control the Data**
- SQLite database you own (no cloud dependency)
- AES-256-GCM encryption for API keys and tokens
- Bcrypt password hashing (cost 12)
- Everything runs on your hardware

**Privacy by Design**
- No tracking by default (Google Analytics is opt-in if you want it)
- No third-party scripts unless you enable them
- No data mining
- Email allowlist for admin access

**Battle-Tested Security**
- Full security review completed and issues addressed
- 11-layer path traversal protection
- DOMPurify XSS prevention
- Rate limiting on sensitive endpoints
- Deny-by-default access control

**Transparent and Auditable**
- Open source (you can read every line)
- 25+ E2E security tests
- Full security documentation
- No proprietary black boxes

**Share Links That Don't Leak**
- HMAC-SHA256 hashed tokens (raw tokens never stored)
- Expiration dates and use limits
- One-click revocation
- Works correctly behind reverse proxies

**Contact Protection**
- Four-tier protection (CSS obfuscation, click-to-reveal, CAPTCHA-ready)
- Per-view visibility controls
- Anti-bot measures

Facet isn't just private. It's designed to be verifiably secure. Read the full security model: [docs/SECURITY.md](docs/SECURITY.md)

---

## Quick Start

```bash
# Clone it
git clone https://github.com/jesposito/Facet.git
cd Facet

# Generate an encryption key (required for API keys and tokens)
openssl rand -hex 32

# Copy the example config
cp .env.example .env

# Edit .env:
# - Set ENCRYPTION_KEY (required)
# - Set ADMIN_EMAILS to your email
# - (Optional) Add OAuth credentials (GOOGLE_CLIENT_ID/SECRET or GITHUB_CLIENT_ID/SECRET)

# Run it
docker-compose up -d
```

Open `http://localhost:8080`. You're live.

> Your data lives in `./data` by default. Back that up. If you want it somewhere else, set `DATA_PATH` in `.env`.

**First login:**
- Password login: `admin@example.com` / `changeme123`
  - You'll be prompted to change this password on first login (modal blocks access until changed)
- OAuth login: Set up Google or GitHub OAuth credentials in `.env` (see [docs/SETUP.md](docs/SETUP.md))

Full setup instructions (OAuth, reverse proxy, etc.): [docs/SETUP.md](docs/SETUP.md)

---

## Who Uses Facet?

Three kinds of people interact with Facet instances:

### 1. **Visitors** (People viewing your profile)

They see whatever you've made public or shared with them:
- Your homepage at `/` (your default view)
- Named views like `/recruiter` or `/conference`
- Share links you've sent them (`/s/abc123`)
- Individual project pages (`/projects/my-cool-app`)
- Blog posts (`/posts/my-article`)
- Talks (`/talks/my-conference-talk`)
- Your RSS feed (`/rss.xml`) if they use a feed reader
- Your talks calendar (`/talks.ics`) if they want to add events

They can't see:
- Anything marked private or unlisted (unless they have a share link)
- Content you've hidden from specific views
- Your admin dashboard
- Your actual contact info if you've protected it

### 2. **Site Owners** (That's you, running your own Facet)

You get an admin dashboard at `/admin` where you:
- Build your profile (name, headline, summary, avatar, etc.)
- Add your work history, projects, education, certifications, awards
- Upload your resume (PDF/DOCX) and have AI automatically extract everything
- Write blog posts and add speaking engagements
- Manage skills and contact methods
- Create "views" (different versions of your profile)
- Import projects from GitHub (with optional AI summaries)
- Generate share links that expire or have use limits
- Upload media or link to YouTube/Vimeo
- Configure AI providers (OpenAI, Anthropic, or local Ollama)
- Use the AI writing assistant to improve your content
- Export everything as JSON or YAML
- Print your profile or generate an AI-powered resume (PDF/DOCX)

The views system is the killer feature. You create different versions of your profile:
- **Recruiter view**: Heavy on employment, light on side projects
- **Conference view**: All your talks and relevant projects
- **Consulting view**: Case studies and client work
- **Personal view**: The stuff your friends actually care about

Each view can show/hide sections, include/exclude specific items, override your headline, have a custom call-to-action, use different colors, and have its own privacy settings.

### 3. **Developers** (Contributing to Facet or customizing it)

The codebase is:
- **Backend**: Go 1.24 with PocketBase (a lightweight backend framework)
- **Frontend**: SvelteKit 2.0 with TypeScript and Tailwind CSS
- **Database**: SQLite (embedded, single file)
- **Deployment**: Docker with Caddy reverse proxy

Local development is straightforward:
```bash
make dev          # Starts backend + frontend with hot reload
make test         # Runs Playwright E2E tests
make build        # Builds production Docker image
```

Full dev setup: [docs/DEV.md](docs/DEV.md)

---

## Key Features (The Stuff That Actually Matters)

### Content Types You Can Create

| Thing | What It Is | Public URL |
|-------|------------|------------|
| **Profile** | Your name, headline, summary, avatar, hero image | `/` (homepage) |
| **Experience** | Job history with bullets and date ranges | Embedded in views |
| **Projects** | Portfolio pieces with tech stack, links, images | `/projects/{slug}` |
| **Education** | Schools and degrees | Embedded in views |
| **Certifications** | Professional creds with expiry tracking | Embedded in views |
| **Skills** | Grouped by category with proficiency levels | Embedded in views |
| **Posts** | Blog articles in Markdown with tags | `/posts/{slug}` |
| **Talks** | Speaking engagements with slides/video URLs | `/talks/{slug}` |
| **Awards** | Recognition and achievements | Embedded in views |
| **Contact Methods** | Email, phone, social links (with protection) | Embedded in views |

Everything supports Markdown. Most things support media (images, videos, external embeds).

### The Views System (Why This Exists)

You don't have one profile. You have multiple "views" of your profile, each tailored to an audience.

**Example:** You're looking for a new job but don't want your current employer to know. You:
1. Create a "recruiter" view with your full employment history
2. Set it to "unlisted" (not searchable, only accessible via direct link)
3. Generate a share link that expires in 30 days
4. Send that link to recruiters

Your public view (at `/`) shows whatever you want the world to see. Your boss won't stumble on your job hunt.

**Each view can:**
- Show/hide entire sections (experience, projects, posts, etc.)
- Include/exclude specific items (show this project, hide that one)
- Override your hero headline and summary
- Add a custom call-to-action button
- Use a different accent color and custom CSS
- Be public, unlisted, password-protected, or private
- Be reordered with drag-and-drop

You can have as many views as you want. One must be marked as default (shown at `/`).

### Privacy Controls (Four Levels)

| Level | Who Can Access | Use Case |
|-------|----------------|----------|
| **Public** | Anyone on the internet | Your general professional presence |
| **Unlisted** | Only people with the URL | Share with specific people without a password |
| **Password** | Anyone who knows the password | "Here's my consulting portfolio, password is TechConf2024" |
| **Private** | Only you (when logged in) | Drafts or internal notes |

Privacy applies to individual items (projects, posts, etc.) and entire views.

### Share Links (Unlisted Views with Superpowers)

For unlisted views, you can generate share links that:
- Expire after a certain date
- Limit total uses (e.g., "can be viewed 10 times")
- Track when they were last used
- Get revoked instantly if needed
- Hide the token from the URL bar (clean links like `/recruiter` instead of `/s/abc123xyz`)

You create a link, send it to someone, they click it, they see your view. No account needed. No ugly tokens in the URL.

### GitHub Import (With Optional AI Help)

Connect your GitHub account and import repositories as projects. Facet grabs:
- Repo name and description
- README content
- Languages used (with percentages)
- Topics/tags
- GitHub URL

**Optional AI enrichment:**
If you configure an AI provider (OpenAI, Anthropic, or local Ollama), Facet can:
- Generate a summary from the README
- Create bullet points highlighting key features
- Suggest tags based on content
- Clean up technical jargon

You review everything before it goes live. You can edit any field, lock fields you've customized (so they don't get overwritten on refresh), and refresh projects from GitHub anytime.

The AI won't invent metrics or hallucinate features (we've built guardrails). Your API keys are encrypted at rest with AES-256-GCM.

### Resume Upload & AI Parsing (Import Your Existing Resume)

Already have a resume? Upload it (PDF or DOCX) and let AI extract all your professional information automatically.

**What gets extracted:**
- Work experience (title, company, dates, responsibilities)
- Education (degrees, schools, dates)
- Skills (with categories and proficiency levels)
- Projects (title, description, technologies)
- Certifications (name, issuer, dates)
- Awards and speaking engagements

**Smart deduplication:**
- Skills are always deduplicated across imports ("Docker" is "Docker")
- Experience and projects dedupe within the same file only (allows faceted views)
- Education, certifications, and awards dedupe universally

**How it works:**
1. Upload your PDF or Word resume
2. AI extracts text and parses it into structured data
3. Records are created with intelligent deduplication
4. File is stored for your records (visible in media gallery)
5. Import summary shows what was created

**File handling:**
- SHA256 hash prevents accidental duplicate imports (5-minute window)
- Supports complex layouts, tables, and multi-column resumes
- Fallback XML extraction for problematic DOCX files
- Stores original file for future reference

This is the fastest way to populate your Facet profile if you already have a resume prepared.

### AI Writing Assistant (Makes You Sound Better)

Built into every text field in the admin dashboard. Two modes:

**1. Rewrite Mode (5 tones):**
- Executive (formal, C-suite focused)
- Professional (balanced, industry standard)
- Technical (methodology-driven, precise)
- Conversational (approachable, first-person)
- Creative (engaging, storytelling-focused)

Paste your rough draft, pick a tone, get a polished version.

**2. Critique Mode:**
Gives you inline feedback like:
- [This is vague. What kind of system? What scale?]
- [Weak verb. What did you actually do?]
- [Quantify this. How much faster?]
- [This sounds like AI wrote it. Be more specific.]

It won't rewrite for you, just tells you what's weak. Good for when you want to improve your own writing.

**Anti-AI rules baked in:**
- Banned words: "leverage", "delve", "synergy", "robust", "utilize"
- No em-dashes (we're not writing a novel)
- Prefer active voice, specific details, quantification

Works on mobile. Context-aware (uses your form data for better results). Supports streaming responses so you see text as it generates.

### Contact Protection (Four Tiers)

Your email, phone, and social links can be protected:

| Level | What Happens | Use Case |
|-------|--------------|----------|
| **None** | Plain text, visible to everyone | LinkedIn, GitHub (already public) |
| **CSS Obfuscation** | Hidden with CSS, visible on hover | Light anti-bot protection |
| **Click-to-Reveal** | JavaScript toggle, user has to click | Moderate anti-bot protection |
| **CAPTCHA** | Turnstile challenge (planned) | Heavy anti-bot protection |

Plus, you can show different contact methods in different views. Example: recruiters see your email and phone, conference attendees only see your Twitter and LinkedIn.

### Media Library

Upload images, videos, documents. Or add external media (YouTube, Vimeo, image URLs).

Features:
- Automatic thumbnail generation for images
- Responsive srcsets (different sizes for different screens)
- Orphan detection (finds files not used anywhere)
- Bulk cleanup (delete all orphans at once)
- Storage usage stats
- Search and filter

Media attaches to projects, posts, and talks. It shows up on public pages with lazy loading and proper alt text.

### Feeds and Exports

**RSS Feed** (`/rss.xml`):
- All your public blog posts
- Auto-discovery in browsers and feed readers
- Full post content included

**iCal Export** (`/talks.ics`):
- All your public talks as calendar events
- Import into Google Calendar, Outlook, Apple Calendar
- Includes event name, date, location, links to slides/video

**Data Export** (JSON or YAML):
- Everything: profile, experience, projects, posts, talks, views, settings
- Perfect for backups or migrating to another system
- Timestamped snapshots

**Print System**:
- Print-optimized stylesheet (works with Cmd+P or Ctrl+P)
- AI-powered resume generation (PDF/DOCX with multiple formats and styles)
- Clean, professional layout

### SEO and Discoverability

Every page gets:
- Proper `<title>` and `<meta description>` tags
- Open Graph tags (for Twitter, Facebook, Slack previews)
- JSON-LD structured data (Person, Article, WebSite schemas)
- Canonical URLs (avoid duplicate content penalties)

Plus:
- Dynamic sitemap at `/sitemap.xml`
- Robots.txt at `/robots.txt`
- Custom 404 and 500 error pages (with a sense of humor)

Search engines can index your public content. Unlisted and private stuff stays hidden.

---

## What Facet Is Not

**It's not a CMS.** If you want a blog with 17 post types and a visual page builder, use WordPress.

**It's not a social network.** There are no likes, no comments, no followers. It's a profile platform.

**It's not a resume builder.** It's more than a resume (you can export a resume from it, though).

**It's not a no-code tool.** You need to run a Docker container and edit a `.env` file. If that sounds scary, this might not be for you (yet).

**It's not LinkedIn.** There's no feed, no messaging, no "People You May Know". It's your profile, hosted by you, under your control.

---

## Tech Stack (For Developers)

**Backend:**
- **Go 1.24** (backend language)
- **PocketBase v0.23.4** (lightweight backend framework built on SQLite and Fiber)
- **SQLite** (embedded database, single file)
- **AES-256-GCM** (encryption for API keys and tokens)
- **Bcrypt** (password hashing)
- **JWT** (session tokens for password-protected views)

**Frontend:**
- **SvelteKit v2.0** (full-stack web framework)
- **Svelte v4.2** (component framework)
- **TypeScript** (type safety)
- **Tailwind CSS v3.4** (utility-first CSS)
- **Marked** (Markdown rendering)
- **DOMPurify** (XSS prevention)

**Infrastructure:**
- **Docker** (containerization)
- **Caddy** (internal reverse proxy)
- **Multi-stage builds** (optimized production images)

**Testing:**
- **Playwright** (E2E tests)
- 25+ tests covering public APIs, admin flows, security, media management
- 96% pass rate (12/12 public tests passing)

**Development:**
- **Air** (Go hot reload)
- **Vite** (frontend dev server with HMR)
- **Make** (task automation)

---

## Architecture (The 10,000-Foot View)

```
┌──────────────────────────────────────────────┐
│         Docker Container (port 8080)         │
│                                              │
│  ┌────────────────────────────────────────┐ │
│  │  Caddy (Reverse Proxy)                 │ │
│  │  /api/*  → PocketBase :8090            │ │
│  │  /*      → SvelteKit :3000             │ │
│  └────────────────────────────────────────┘ │
│                                              │
│  ┌──────────────┐      ┌─────────────────┐ │
│  │  SvelteKit   │      │   PocketBase    │ │
│  │  :3000       │◄────►│   :8090         │ │
│  │  (Frontend)  │      │   (Backend)     │ │
│  └──────────────┘      └─────────────────┘ │
│                              │               │
│                        ┌─────▼────────┐     │
│                        │  /data       │     │
│                        │  (Volume)    │     │
│                        │              │     │
│                        │  data.db     │     │
│                        │  uploads/    │     │
│                        └──────────────┘     │
└──────────────────────────────────────────────┘
```

**What happens when someone visits `/recruiter`:**

1. Browser → Caddy :8080
2. Caddy → SvelteKit :3000 (route handler)
3. SvelteKit → PocketBase API :8090 (fetch view data)
4. PocketBase → SQLite (query database)
5. Response flows back up the chain
6. SvelteKit renders HTML with data
7. Browser displays the page

**Everything runs in one container.** One port exposed (8080). One volume to backup (`/data`). That's the whole deployment.

For detailed architecture: [ARCHITECTURE.md](docs/ARCHITECTURE.md)

---

## Configuration (Environment Variables)

| Variable | Required? | Default | What It Does |
|----------|-----------|---------|--------------|
| `ENCRYPTION_KEY` | **Yes** | — | 32-byte hex key for encrypting API keys and tokens (`openssl rand -hex 32`) |
| `PORT` | No | `8080` | Public port for the app |
| `APP_URL` | No | `http://localhost:8080` | Your public URL (needed for OAuth callbacks) |
| `ADMIN_EMAILS` | No | — | Comma-separated email allowlist for OAuth login |
| `TRUST_PROXY` | No | `false` | Set `true` if behind a reverse proxy (Nginx, Cloudflare, etc.) |
| `ADMIN_ENABLED` | No | `false` | Enable PocketBase admin UI at `/_/` (use for debugging only) |
| `DATA_PATH` | No | `./data` | Where to store the database and uploads |
| `GOOGLE_CLIENT_ID` | No | — | OAuth via Google |
| `GOOGLE_CLIENT_SECRET` | No | — | OAuth via Google |
| `GITHUB_CLIENT_ID` | No | — | OAuth via GitHub |
| `GITHUB_CLIENT_SECRET` | No | — | OAuth via GitHub |

Full setup guide (OAuth, reverse proxy, Unraid, etc.): [docs/SETUP.md](docs/SETUP.md)

---

## Backup and Restore (Super Simple)

Everything lives in one directory: `./data` (or wherever `DATA_PATH` points).

**Backup:**
```bash
docker-compose down
tar -czvf facet-backup-$(date +%Y%m%d).tar.gz ./data
docker-compose up -d
```

**Restore:**
```bash
docker-compose down
tar -xzvf facet-backup-20260103.tar.gz
docker-compose up -d
```

That's it. The tarball contains your SQLite database and all uploaded files.

For upgrade procedures: [docs/UPGRADE.md](docs/UPGRADE.md)

---

## Security (The Boring But Important Stuff)

**Authentication:**
- OAuth 2.0 (Google, GitHub)
- Email allowlist (`ADMIN_EMAILS`)
- Session tokens in httpOnly cookies

**Encryption:**
- AES-256-GCM for API keys and sensitive tokens (encrypted at rest)
- Bcrypt for passwords (cost 12)
- HMAC-SHA256 for share tokens (raw tokens never stored)
- JWT for password-protected view sessions

**Access Control:**
- Deny-by-default on all database collections
- Admin-only by default
- Public content requires explicit `visibility="public"`
- Rate limiting on sensitive endpoints

**Input Validation:**
- DOMPurify for XSS prevention
- 11-layer path traversal protection
- Symlink detection
- Type validation (TypeScript + PocketBase schema)

**What We Don't Do:**
- No analytics or tracking
- No engagement metrics
- No user profiling
- Minimal server logging

Full security docs: [docs/SECURITY.md](docs/SECURITY.md)

---

## For Developers (Contributing or Customizing)

### Local Development

**Prerequisites:**
- Go 1.24+
- Node.js 20+
- [Air](https://github.com/air-verse/air) for Go hot reload (install: `go install github.com/air-verse/air@v1.61.7`)

**Start everything:**
```bash
make dev          # Starts backend (with Air) + frontend (with Vite HMR)
```

**Or start services individually:**
```bash
make backend      # Just the Go backend on :8090
make frontend     # Just the SvelteKit frontend on :5173
```

**Other useful commands:**
```bash
make test         # Run Playwright E2E tests
make build        # Build production Docker image
make lint         # Run linters
make fmt          # Format code (Go + Prettier)
make seed-dev     # Load development seed data
make dev-reset    # Clear caches and restart
```

**Development ports:**
- Frontend (Vite): http://localhost:5173
- Backend (PocketBase): http://localhost:8090
- PocketBase Admin: http://localhost:8090/_/

**First-time setup:**
Run `make seed-dev` to set your admin email/password and optional OAuth credentials. The script prompts you (no hard-coded defaults).

Full developer guide: [docs/DEV.md](docs/DEV.md)

### Project Structure

```
Facet/
├── backend/                 # Go + PocketBase
│   ├── hooks/               # Custom API endpoints (10K+ lines)
│   │   ├── view.go          # Views, RSS, iCal (1,883 lines)
│   │   ├── ai.go            # AI enrichment (688 lines)
│   │   ├── media.go         # Media management (612 lines)
│   │   └── resume.go        # Resume generation (518 lines)
│   ├── services/            # Reusable business logic (6K+ lines)
│   │   ├── ai.go            # AI provider integration
│   │   ├── crypto.go        # AES encryption
│   │   └── github.go        # GitHub API
│   ├── migrations/          # Database schema (20+ migrations)
│   └── main.go              # Entry point
│
├── frontend/                # SvelteKit + TypeScript
│   ├── src/routes/          # Page routes (77 files)
│   │   ├── [slug]/          # Public view pages
│   │   ├── admin/           # Admin dashboard
│   │   ├── projects/        # Project pages
│   │   ├── posts/           # Blog pages
│   │   └── talks/           # Talks pages
│   ├── src/components/      # UI components (30+ files)
│   └── src/lib/             # Utilities, stores, API client
│
├── frontend/tests/          # Playwright E2E tests
│   ├── public-api.spec.ts   # RSS, iCal, endpoints
│   ├── seo-and-errors.spec.ts  # SEO, error pages
│   ├── admin-flows.spec.ts  # Auth, CRUD
│   ├── media-management.spec.ts  # Uploads, orphans
│   └── security.spec.ts     # XSS, path traversal
│
├── docker/                  # Production Docker config
├── scripts/                 # Development scripts
└── docs/                    # Documentation
```

**Code stats:**
- ~38,000 lines across 1,300+ files
- Backend: ~16,000 lines of Go
- Frontend: ~21,000 lines of Svelte/TypeScript
- Tests: ~3,100 lines (Go unit tests + Playwright E2E)

### Testing

**Run all tests:**
```bash
make test
```

**Run specific test suites:**
```bash
cd frontend
npm run test:public          # Public API tests (no auth required)
npm run test -- security.spec.ts  # Just security tests
```

**Current test status:**
- ✅ Backend tests: 100% passing
- ✅ Public E2E tests: 12/12 passing
- ⚠️ Admin tests: 20 tests require `ADMIN_EMAIL` and `ADMIN_PASSWORD` in `.env`

Full testing guide: [frontend/tests/README.md](frontend/tests/README.md)

### URL Routing (For Reference)

**Public routes:**
- `GET /` → Default view (homepage)
- `GET /{slug}` → Named view (e.g., `/recruiter`)
- `GET /v/{slug}` → Legacy view route (301 redirect to `/{slug}`)
- `GET /s/{token}` → Share link (redirects to view)
- `GET /projects/{slug}` → Project detail page
- `GET /posts` → Blog index
- `GET /posts/{slug}` → Blog post
- `GET /talks` → Talks index
- `GET /talks/{slug}` → Talk detail
- `GET /rss.xml` → RSS feed
- `GET /talks.ics` → iCal export
- `GET /sitemap.xml` → Sitemap
- `GET /robots.txt` → Robots.txt

**Admin routes** (OAuth required):
- `GET /admin` → Dashboard
- `GET /admin/profile` → Edit profile
- `GET /admin/experience` → Manage jobs
- `GET /admin/projects` → Manage projects
- `GET /admin/views` → Manage views
- `GET /admin/import` → GitHub import
- `GET /admin/media` → Media library
- `GET /admin/settings` → AI providers, settings
- (Plus routes for education, certifications, skills, posts, talks, awards, contacts, tokens)

**API routes** (via PocketBase hooks):
- `GET /api/homepage` → Fetch homepage data
- `POST /api/github/import` → Import from GitHub
- `POST /api/ai/enrich` → AI enrichment
- `GET /api/export?format=json|yaml` → Data export
- `POST /api/share/validate` → Validate share token
- (Plus standard PocketBase collection endpoints)

---

## Documentation (Everything Else)

| Doc | What's In It |
|-----|--------------|
| [docs/SETUP.md](docs/SETUP.md) | Installation, OAuth setup, reverse proxy config, Unraid |
| [docs/DEV.md](docs/DEV.md) | Local development, project structure, troubleshooting |
| [docs/SECURITY.md](docs/SECURITY.md) | Encryption, auth flows, rate limiting, threat model |
| [docs/UPGRADE.md](docs/UPGRADE.md) | How to upgrade Facet and roll back if needed |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Technical system design, data model, request flow |
| [docs/DESIGN.md](docs/DESIGN.md) | Vision, principles, detailed feature specs |
| [docs/ROADMAP.md](docs/ROADMAP.md) | Development phases (what's done, what's planned) |
| [frontend/tests/README.md](frontend/tests/README.md) | How to run tests, write new tests, test structure |
| [docs/AI_FEATURES.md](docs/AI_FEATURES.md) | AI provider setup, enrichment details |
| [docs/AI_WRITING_ASSISTANT.md](docs/AI_WRITING_ASSISTANT.md) | Writing assistant tones, critique mode, implementation |
| [docs/CONTACT_PROTECTION.md](docs/CONTACT_PROTECTION.md) | Contact protection tiers, implementation details |
| [docs/MEDIA.md](docs/MEDIA.md) | Media system internals, file handling, optimization |

> **Note for contributors:** Keep [ROADMAP.md](docs/ROADMAP.md) up-to-date. When you complete a feature, mark it done. When you add a feature, add it to the roadmap. It's the source of truth for what's implemented vs. planned.

---

## Roadmap (What's Done, What's Next)

**Completed (13 phases):**
- ✅ Core profile and content management
- ✅ Views system with custom ordering, overrides, theming
- ✅ Share token management with expiration and use limits
- ✅ GitHub import with AI enrichment and field locking
- ✅ Media library with uploads, external embeds, orphan detection
- ✅ AI Writing Assistant (5 tones + critique mode)
- ✅ Contact protection (4 tiers)
- ✅ Export system (JSON/YAML)
- ✅ Print-optimized CSS
- ✅ AI-powered resume generation (PDF/DOCX with multiple formats and styles)
- ✅ Resume upload and AI parsing (PDF/DOCX → auto-extract to Facet)
- ✅ RSS feed and iCal export
- ✅ SEO (Open Graph, JSON-LD, sitemaps)
- ✅ Security review and XSS/path traversal protection
- ✅ Demo mode with comprehensive example content

**Planned:**
- Scheduled GitHub sync (auto-refresh projects)
- Security headers (CSP, X-Frame-Options)
- 2FA (TOTP + backup codes)
- Audit logging for admin actions
- Webhooks and integrations

Full roadmap: [ROADMAP.md](docs/ROADMAP.md)

---

## Common Questions

**Q: Can I use this without Docker?**
A: Not easily. The production setup uses Caddy to route requests between the backend and frontend. You could run them separately in development (`make dev`), but deployment assumes Docker.

**Q: Can I use a different database?**
A: No. PocketBase uses SQLite. It's baked into the framework. (And honestly, SQLite is perfect for this use case.)

**Q: Can I customize the design?**
A: Yes. Each view supports custom CSS. The frontend uses Tailwind, so you can modify `frontend/src/app.css` or component styles. For deeper changes, you'll need to edit Svelte components.

**Q: Can I self-host this on a Raspberry Pi?**
A: Probably? Docker runs on ARM. The SQLite database is tiny. Give it a shot and let us know.

**Q: Can I use this for a team or company?**
A: Not really. Facet is designed for individuals. There's no multi-tenancy, no user roles (other than admin vs. visitor). You could hack it, but you'd be fighting the design.

**Q: Can I migrate from LinkedIn?**
A: There's no automated LinkedIn import. You'll need to copy/paste your content or use the GitHub import for projects. (LinkedIn doesn't export cleanly.)

**Q: Can I use this without AI features?**
A: Absolutely. AI enrichment and the writing assistant are optional. If you don't configure an AI provider, those features just won't appear in the UI.

**Q: Can I contribute?**
A: Yes! Check [CONTRIBUTING.md](CONTRIBUTING.md) if it exists, or just open a PR. Bug fixes and documentation improvements are always welcome. For big features, open an issue first to discuss.

---

## License

MIT. Do whatever you want with it.

---

## Support This Project

If Facet saves you time or helps your career, consider buying me a coffee!

[![Buy Me a Coffee](https://img.shields.io/badge/Buy%20Me%20A%20Coffee-Support%20Facet-FFDD00?style=for-the-badge&logo=buy-me-a-coffee&logoColor=black)](https://buymeacoffee.com/jesposito)

Your support helps keep the project maintained and growing. Thank you! ☕

---

## Credits

Built by [jesposito](https://github.com/jesposito). Powered by [PocketBase](https://pocketbase.io/), [SvelteKit](https://kit.svelte.dev/), and too much coffee.

If you use Facet and like it, star the repo. If you find bugs, open issues. If you want a feature, open a discussion.

---

*Your profile. Your data. Your rules.*
