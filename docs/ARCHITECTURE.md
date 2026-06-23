# Facet Architecture Document

> **See Also**: For the complete design document including vision, principles, and detailed specifications, see [DESIGN.md](DESIGN.md). For the feature roadmap, see [ROADMAP.md](ROADMAP.md).

## Technical Architecture

### Technology Stack

| Layer | Technology | Version |
|-------|------------|---------|
| Backend | Go + PocketBase | Go 1.25.0, PocketBase v0.23.4 |
| Frontend | SvelteKit + Svelte | SvelteKit 2.5, Svelte 5.55 |
| Styling | Tailwind CSS | 3.4 |
| Build | Vite + TypeScript | Vite 7.3, TypeScript 5.5 |
| Reverse proxy | Caddy (internal) | — |
| Database | SQLite (via PocketBase) | — |

Current Facet version: **v2.22.1**. Default admin login (self-hosted): `admin@example.com` / `changeme123` (change on first login).

A managed/hosted version of Facet is available at [get-facet.com](https://get-facet.com); this repository and document focus on the **self-hosted** deployment.

### System Diagram

```
                                    ┌─────────────────┐
                                    │  Cloudflare     │
                                    │  Tunnel / NPM   │
                                    └────────┬────────┘
                                             │
                                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        Docker Container                              │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                     Caddy (Internal)                            ││
│  │  ├── /api/*      → PocketBase :8090                             ││
│  │  ├── /_/*        → PocketBase :8090 (Admin UI)                  ││
│  │  └── /*          → SvelteKit :3000                              ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                      │
│  ┌─────────────────────┐    ┌──────────────────────────────────────┐│
│  │    SvelteKit        │    │         PocketBase (Go)              ││
│  │    :3000            │    │         :8090                        ││
│  │                     │    │                                      ││
│  │  Public Pages:      │    │  Collections:                        ││
│  │  ├── /              │◄───┤  ├── profile, experience, projects   ││
│  │  ├── /:slug         │    │  ├── education, certifications       ││
│  │  ├── /s/:token      │    │  ├── awards, skills, posts, talks    ││
│  │  ├── /projects/:slug│    │  ├── custom_content, contact_methods ││
│  │  ├── /posts/:slug   │    │  ├── views, share_tokens             ││
│  │  ├── /talks/:slug   │    │  ├── testimonials, testimonial_reqs  ││
│  │  └── /testimonial/* │    │  ├── comments, comment_reports       ││
│  │                     │    │  ├── sources, ai_providers           ││
│  │  Admin Pages:       │    │  ├── import_proposals, resume_imports││
│  │  ├── /admin/*       │───►│  ├── media, external_media           ││
│  │  ├── /admin/alerts  │    │  ├── api_keys, webhooks, deliveries  ││
│  │  └── 30+ routes     │    │  ├── system_alerts, audit_logs       ││
│  │                     │    │  ├── newsletter_lists, subscribers   ││
│  │                     │    │  └── site_settings, admin_tags       ││
│  └─────────────────────┘    │                                      ││
│                             │  Custom Go Hooks:                    ││
│                             │  ├── /api/github/* (import/refresh)  ││
│                             │  ├── /api/ai/* (test/enrich/write)   ││
│                             │  ├── /api/view/* (access/data/gen)   ││
│                             │  ├── /api/testimonials/* (14 routes) ││
│                             │  ├── /api/v1/* (public REST + keys)  ││
│                             │  ├── /api/admin/webhooks, alerts     ││
│                             │  ├── /api/demo/* (enable/disable)    ││
│                             │  └── /api/resume/parse               ││
│                             │                                      ││
│                             │  Auth: OAuth (Google, GitHub)        ││
│                             └──────────────────────────────────────┘│
│                                            │                         │
│                             ┌──────────────┴───────────────┐        │
│                             │     /data (Volume)           │        │
│                             │     ├── pb_data/             │        │
│                             │     │   └── data.db          │        │
│                             │     └── uploads/             │        │
│                             └──────────────────────────────┘        │
└─────────────────────────────────────────────────────────────────────┘
```

### Data Model

#### Collections Schema

```sql
-- Profile (singleton - one record per instance)
profile {
  id                 TEXT PRIMARY KEY
  name               TEXT NOT NULL
  headline           TEXT
  location           TEXT
  summary            TEXT        -- Markdown
  hero_image         FILE
  avatar             FILE
  contact_email      TEXT
  contact_links      JSON        -- [{type: "github", url: "..."}, ...]
  visibility         TEXT        -- "public" | "unlisted" | "private"
  created            DATETIME
  updated            DATETIME
}

-- Experience (work history)
experience {
  id                 TEXT PRIMARY KEY
  company            TEXT NOT NULL
  title              TEXT NOT NULL
  location           TEXT
  start_date         DATE
  end_date           DATE        -- NULL = current
  description        TEXT        -- Markdown
  bullets            JSON        -- ["Achieved X...", ...]
  skills             JSON        -- ["Go", "Docker", ...]
  media              FILE[]
  visibility         TEXT        -- "public" | "unlisted" | "private" | "password"
  password_hash      TEXT        -- For password-protected
  is_draft           BOOLEAN     -- Draft vs published
  sort_order         INTEGER
  created            DATETIME
  updated            DATETIME
}

-- Projects (portfolio items)
projects {
  id                 TEXT PRIMARY KEY
  title              TEXT NOT NULL
  slug               TEXT UNIQUE
  summary            TEXT        -- Short description
  description        TEXT        -- Full markdown content
  tech_stack         JSON        -- ["Go", "Docker", ...]
  links              JSON        -- [{type: "github", url: "..."}, ...]
  media              FILE[]
  cover_image        FILE
  categories         JSON        -- ["web", "devtools", ...]
  visibility         TEXT
  password_hash      TEXT
  is_draft           BOOLEAN
  is_featured        BOOLEAN
  sort_order         INTEGER

  -- Import tracking
  source_id          RELATION -> sources
  field_locks        JSON        -- {"title": true, "summary": false, ...}
  last_sync          DATETIME

  created            DATETIME
  updated            DATETIME
}

-- Education
education {
  id                 TEXT PRIMARY KEY
  institution        TEXT NOT NULL
  degree             TEXT
  field              TEXT
  start_date         DATE
  end_date           DATE
  description        TEXT
  visibility         TEXT
  is_draft           BOOLEAN
  sort_order         INTEGER
  created            DATETIME
  updated            DATETIME
}

-- Certifications
certifications {
  id                 TEXT PRIMARY KEY
  name               TEXT NOT NULL
  issuer             TEXT
  issue_date         DATE
  expiry_date        DATE
  credential_id      TEXT
  credential_url     TEXT
  visibility         TEXT
  is_draft           BOOLEAN
  sort_order         INTEGER
  created            DATETIME
  updated            DATETIME
}

-- Skills (grouped by category)
skills {
  id                 TEXT PRIMARY KEY
  name               TEXT NOT NULL
  category           TEXT        -- "Languages", "Frameworks", etc.
  proficiency        TEXT        -- "expert" | "proficient" | "familiar"
  visibility         TEXT
  sort_order         INTEGER
  created            DATETIME
  updated            DATETIME
}

-- Posts (blog/writing)
posts {
  id                 TEXT PRIMARY KEY
  title              TEXT NOT NULL
  slug               TEXT UNIQUE
  excerpt            TEXT
  content            TEXT        -- Markdown
  cover_image        FILE
  tags               JSON
  visibility         TEXT
  is_draft           BOOLEAN
  published_at       DATETIME
  created            DATETIME
  updated            DATETIME
}

-- Talks (speaking engagements)
talks {
  id                 TEXT PRIMARY KEY
  title              TEXT NOT NULL
  event              TEXT
  event_url          TEXT
  date               DATE
  location           TEXT
  description        TEXT
  slides_url         TEXT
  video_url          TEXT
  visibility         TEXT
  is_draft           BOOLEAN
  sort_order         INTEGER
  created            DATETIME
  updated            DATETIME
}

-- Views (curated versions)
views {
  id                 TEXT PRIMARY KEY
  name               TEXT NOT NULL
  slug               TEXT UNIQUE NOT NULL
  description        TEXT        -- Internal note
  visibility         TEXT        -- "public" | "unlisted" | "private" | "password"
  password_hash      TEXT

  -- Overrides
  hero_headline      TEXT        -- Override profile headline
  hero_summary       TEXT        -- Override profile summary
  cta_text           TEXT        -- e.g., "Download Resume"
  cta_url            TEXT

  -- Section configuration
  sections           JSON        -- [{section: "experience", enabled: true, items: ["id1", "id2"]}, ...]

  -- Metadata
  is_active          BOOLEAN
  created            DATETIME
  updated            DATETIME
}

-- Share Tokens (unlisted access)
share_tokens {
  id                 TEXT PRIMARY KEY
  view_id            RELATION -> views
  token_hash         TEXT UNIQUE  -- Store hash, not raw token
  name               TEXT         -- "Sent to Company X"
  expires_at         DATETIME
  max_uses           INTEGER
  use_count          INTEGER DEFAULT 0
  is_active          BOOLEAN
  last_used_at       DATETIME
  created            DATETIME
  updated            DATETIME
}

-- Sources (GitHub repos, etc.)
sources {
  id                 TEXT PRIMARY KEY
  type               TEXT         -- "github"
  identifier         TEXT         -- "owner/repo"
  project_id         RELATION -> projects

  -- GitHub-specific
  github_token       TEXT         -- Encrypted PAT (optional)

  -- Sync state
  last_sync          DATETIME
  sync_status        TEXT         -- "success" | "error" | "pending"
  sync_log           TEXT         -- Last sync details

  created            DATETIME
  updated            DATETIME
}

-- AI Providers (BYO tokens)
ai_providers {
  id                 TEXT PRIMARY KEY
  name               TEXT         -- User-friendly name
  type               TEXT         -- "openai" | "anthropic" | "ollama" | "custom"

  -- Encrypted credentials
  api_key_encrypted  TEXT
  base_url           TEXT         -- For Ollama/custom
  model              TEXT         -- e.g., "gpt-4", "claude-3-opus"

  -- Settings
  is_default         BOOLEAN
  is_active          BOOLEAN

  -- Metadata
  last_test          DATETIME
  test_status        TEXT

  created            DATETIME
  updated            DATETIME
}

-- Import Proposals (pending review)
import_proposals {
  id                 TEXT PRIMARY KEY
  source_id          RELATION -> sources
  project_id         RELATION -> projects  -- NULL for new projects

  -- Proposed changes
  proposed_data      JSON         -- Full proposed project data
  diff               JSON         -- Field-by-field diff
  ai_enriched        BOOLEAN

  -- Review state
  status             TEXT         -- "pending" | "applied" | "rejected"
  applied_fields     JSON         -- Which fields were applied

  created            DATETIME
  updated            DATETIME
}

-- Awards (recognition and achievements)
awards {
  id                 TEXT PRIMARY KEY
  title              TEXT NOT NULL
  issuer             TEXT
  date               DATE
  description        TEXT
  logo               FILE
  visibility         TEXT         -- "public" | "unlisted" | "private"
  is_draft           BOOLEAN
  sort_order         INTEGER
  created            DATETIME
  updated            DATETIME
}

-- Contact Methods (protected contact information)
contact_methods {
  id                 TEXT PRIMARY KEY
  type               TEXT NOT NULL  -- "email" | "phone" | "linkedin" | "github" | etc.
  value              TEXT NOT NULL
  label              TEXT           -- Display label
  protection_level   TEXT           -- "none" | "obfuscated" | "click_reveal" | "captcha"
  view_visibility    JSON           -- {view_id: boolean} for per-view visibility
  visibility         TEXT
  sort_order         INTEGER
  created            DATETIME
  updated            DATETIME
}

-- Custom Content (user-defined content sections)
custom_content {
  id                 TEXT PRIMARY KEY
  title              TEXT NOT NULL
  slug               TEXT UNIQUE
  content            TEXT           -- Markdown
  icon               TEXT           -- Icon identifier
  visibility         TEXT
  is_draft           BOOLEAN
  sort_order         INTEGER
  created            DATETIME
  updated            DATETIME
}

-- Testimonials (social proof from third parties)
testimonials {
  id                 TEXT PRIMARY KEY
  request_id         RELATION -> testimonial_requests
  name               TEXT NOT NULL
  title              TEXT
  company            TEXT
  relationship       TEXT           -- "client" | "colleague" | "manager" | etc.
  content            TEXT NOT NULL
  rating             INTEGER        -- Optional 1-5 rating
  status             TEXT           -- "pending" | "approved" | "rejected"
  featured           BOOLEAN
  verified           BOOLEAN
  verified_via       TEXT           -- "email" | "github" | "linkedin"
  avatar_url         TEXT
  created            DATETIME
  updated            DATETIME
}

-- Testimonial Requests (shareable links to collect testimonials)
testimonial_requests {
  id                 TEXT PRIMARY KEY
  token_hash         TEXT UNIQUE    -- HMAC-SHA256 hash (raw never stored)
  token_prefix       TEXT           -- First 12 chars for lookup
  name               TEXT           -- Admin label
  custom_message     TEXT           -- Optional message to show submitters
  expires_at         DATETIME
  max_uses           INTEGER
  use_count          INTEGER DEFAULT 0
  is_active          BOOLEAN
  created            DATETIME
  updated            DATETIME
}

-- Email Verification Tokens (for testimonial verification)
email_verification_tokens {
  id                 TEXT PRIMARY KEY
  testimonial_id     RELATION -> testimonials
  token_hash         TEXT UNIQUE
  email              TEXT
  expires_at         DATETIME       -- 15-minute expiration
  verified_at        DATETIME
  created            DATETIME
}

-- Resume Imports (uploaded resume tracking)
resume_imports {
  id                 TEXT PRIMARY KEY
  filename           TEXT
  file_hash          TEXT           -- SHA256 for duplicate detection
  file_size          INTEGER
  mime_type          TEXT
  parsed_data        JSON           -- Extracted structured data
  status             TEXT           -- "pending" | "parsed" | "error"
  error_message      TEXT
  created            DATETIME
  updated            DATETIME
}

-- Site Settings (global configuration singleton)
site_settings {
  id                 TEXT PRIMARY KEY
  homepage_enabled   BOOLEAN
  landing_page_message TEXT
  custom_css         TEXT
  ga_measurement_id  TEXT           -- Google Analytics
  hide_login_button  BOOLEAN
  hide_demo_toggle   BOOLEAN
  site_nav_enabled   BOOLEAN
  site_nav_items     JSON           -- Navigation buttons config
  skills_category_order JSON        -- Custom ordering
  homepage_sections  JSON           -- Per-section config
  homepage_section_order JSON       -- Section display order
  created            DATETIME
  updated            DATETIME
}

-- External Media (linked media from external sources)
external_media {
  id                 TEXT PRIMARY KEY
  url                TEXT NOT NULL
  title              TEXT
  mime_type          TEXT
  thumbnail_url      TEXT
  provider           TEXT           -- "youtube" | "vimeo" | "image" | etc.
  created            DATETIME
  updated            DATETIME
}

-- Admin Tags (for organizing content)
admin_tags {
  id                 TEXT PRIMARY KEY
  name               TEXT NOT NULL UNIQUE
  color              TEXT
  created            DATETIME
  updated            DATETIME
}

-- API Keys (scoped access to the public REST API at /api/v1/*)
api_keys {
  id                 TEXT PRIMARY KEY
  label              TEXT
  key_prefix         TEXT           -- Indexed lookup prefix
  key_hash           TEXT           -- HMAC hash; raw key shown once
  scopes             JSON           -- ["read:profile", "write:posts", ...]
  allowed_origins    JSON           -- Optional CORS allowlist
  expires_at         DATETIME
  last_used_at       DATETIME
  is_active          BOOLEAN
  created            DATETIME
  updated            DATETIME
}

-- Webhooks (outbound event notifications)
webhooks {
  id                 TEXT PRIMARY KEY
  label              TEXT
  url                TEXT           -- SSRF-protected target
  secret             TEXT           -- HMAC-SHA256 signing secret (auto-generated)
  events             JSON           -- Subscribed event names
  is_active          BOOLEAN
  failure_count      INTEGER        -- Auto-disabled after repeated failures
  last_delivered_at  DATETIME
  last_failure_at    DATETIME
  last_error         TEXT
  created            DATETIME
  updated            DATETIME
}

-- Webhook Deliveries (delivery log for each attempt)
webhook_deliveries {
  id                 TEXT PRIMARY KEY
  webhook_id         RELATION -> webhooks
  event_name         TEXT
  request_body       TEXT
  response_status    INTEGER
  response_body      TEXT
  attempt_count      INTEGER
  succeeded          BOOLEAN
  created            DATETIME
}

-- System Alerts (admin inbox surfaced at /admin/alerts)
system_alerts {
  id                 TEXT PRIMARY KEY
  severity           TEXT           -- "info" | "warning" | "error"
  event              TEXT
  title              TEXT
  details            TEXT
  metadata           JSON
  acknowledged_at    DATETIME
  created            DATETIME
  updated            DATETIME
}

-- Comments (threaded, with moderation queue)
comments {
  id                 TEXT PRIMARY KEY
  content_type       TEXT           -- "post" | "project" | etc.
  content_id         TEXT
  parent_id          TEXT           -- For threaded replies
  author_name        TEXT
  author_email       TEXT
  body               TEXT
  status             TEXT           -- "pending" | "approved" | "spam" | "rejected"
  is_pinned          BOOLEAN
  is_admin_reply     BOOLEAN
  ip_hash            TEXT
  user_agent         TEXT
  created            DATETIME
  updated            DATETIME
}

-- Comment Reports (user-flagged comments)
comment_reports {
  id                 TEXT PRIMARY KEY
  comment            RELATION -> comments
  reason             TEXT
  detail             TEXT
  reporter_ip_hash   TEXT
  status             TEXT
  created            DATETIME
  updated            DATETIME
}

-- Newsletter Lists (multi-list / segments)
newsletter_lists {
  id                 TEXT PRIMARY KEY
  name               TEXT NOT NULL
  slug               TEXT NOT NULL
  description        TEXT
  sender_name        TEXT
  reply_to           TEXT
  welcome_subject    TEXT
  welcome_html       TEXT
  is_default         BOOLEAN
  is_active          BOOLEAN
  subscriber_count   INTEGER
  created            DATETIME
  updated            DATETIME
}

-- Subscribers + list memberships
subscribers {
  id                 TEXT PRIMARY KEY
  email              TEXT
  status             TEXT
  -- ... confirmation / unsubscribe tracking
}

subscriber_list_memberships {
  id                 TEXT PRIMARY KEY
  subscriber_id      RELATION -> subscribers
  list_id            RELATION -> newsletter_lists
  created            DATETIME
}

-- Audit Logs (access / admin action history)
audit_logs {
  id                 TEXT PRIMARY KEY
  event              TEXT
  metadata           JSON
  created            DATETIME
}
```

> The schema above covers the self-hosted profile platform. The codebase also pins the PocketBase database version and runs **automigrate on boot** so a fresh self-hosted container provisions all collections automatically.

### Route Map

#### Public Routes (SvelteKit)

| Route | Description | Auth |
|-------|-------------|------|
| `GET /` | Default public view (homepage) | Based on view.visibility |
| `GET /:slug` | Named view (canonical) | Based on view.visibility |
| `GET /v/:slug` | Legacy view route | 301 redirect to `/:slug` |
| `GET /s/:token` | Share token access | Token validation, redirects to `/:slug` |
| `GET /projects/:slug` | Project detail page | Based on project.visibility |
| `GET /posts` | Blog post listing | Public |
| `GET /posts/:slug` | Blog post page | Based on post.visibility |
| `GET /talks` | Talks listing | Public |
| `GET /talks/:slug` | Talk detail page | Based on talk.visibility |
| `GET /testimonial/:token` | Submit testimonial form | Token validation |
| `GET /testimonial/verify/:token` | Email verification | Token validation |
| `GET /rss.xml` | RSS feed for posts | Public |
| `GET /talks.ics` | iCal export for talks | Public |
| `GET /sitemap.xml` | Dynamic sitemap | Public |
| `GET /robots.txt` | Robots configuration | Public |

#### Admin Routes (SvelteKit)

| Route | Description | Auth |
|-------|-------------|------|
| `GET /admin` | Admin dashboard | OAuth required |
| `GET /admin/alerts` | System alerts inbox | OAuth required |
| `GET /admin/homepage` | Homepage configuration | OAuth required |
| `GET /admin/contacts` | Contact methods | OAuth required |
| `GET /admin/experience` | Manage experience | OAuth required |
| `GET /admin/projects` | Manage projects | OAuth required |
| `GET /admin/education` | Manage education | OAuth required |
| `GET /admin/certifications` | Manage certifications | OAuth required |
| `GET /admin/awards` | Manage awards | OAuth required |
| `GET /admin/skills` | Manage skills | OAuth required |
| `GET /admin/custom` | Custom content sections | OAuth required |
| `GET /admin/import` | GitHub import & Resume AI | OAuth required |
| `GET /admin/posts` | Manage posts | OAuth required |
| `GET /admin/talks` | Manage talks | OAuth required |
| `GET /admin/testimonials` | Manage testimonials | OAuth required |
| `GET /admin/testimonials/requests` | Testimonial request links | OAuth required |
| `GET /admin/views` | Manage views/facets | OAuth required |
| `GET /admin/views/new` | Create new view | OAuth required |
| `GET /admin/views/:id` | Edit specific view | OAuth required |
| `GET /admin/tokens` | Manage share tokens | OAuth required |
| `GET /admin/media` | Media library | OAuth required |
| `GET /admin/review/:id` | Review import proposal | OAuth required |
| `GET /admin/settings` | Main settings | OAuth required |
| `GET /admin/settings/account` | Account & security | OAuth required |
| `GET /admin/settings/appearance` | Appearance settings | OAuth required |
| `GET /admin/settings/general` | General settings | OAuth required |
| `GET /admin/settings/analytics` | Analytics settings | OAuth required |
| `GET /admin/settings/integrations` | AI providers & integrations | OAuth required |
| `GET /admin/settings/tags` | Admin tags | OAuth required |
| `GET /admin/settings/about` | About Facet | OAuth required |
| `GET /admin/login` | Admin login | None |

#### API Routes (PocketBase + Custom Hooks)

| Route | Method | Description | Auth |
|-------|--------|-------------|------|
| `/api/collections/*` | * | PocketBase CRUD | API rules |
| `/api/health` | GET | Health check | None |
| `/api/default-view` | GET | Get default view slug | None |
| `/api/view/:slug/access` | GET | Check view access | Token/Password |
| `/api/view/:slug/data` | GET | Get view content | Token/Password |
| `/api/view/:slug/generate` | POST | Generate AI resume | OAuth |
| `/api/homepage` | GET | Homepage data | None |
| `/api/share/validate` | POST | Validate share token | None |
| `/api/password/check` | POST | Validate view password | None |

**GitHub Import:**

| Route | Method | Description | Auth |
|-------|--------|-------------|------|
| `/api/github/repos` | GET | List user's GitHub repos | OAuth |
| `/api/github/preview` | POST | Preview repo before import | OAuth |
| `/api/github/import` | POST | Start import from GitHub | OAuth |
| `/api/github/refresh/:id` | POST | Refresh source from GitHub | OAuth |
| `/api/proposals/:id/apply` | POST | Apply import proposal | OAuth |
| `/api/proposals/:id/reject` | POST | Reject import proposal | OAuth |

**AI Features:**

| Route | Method | Description | Auth |
|-------|--------|-------------|------|
| `/api/ai/test/:id` | POST | Test AI provider connection | OAuth |
| `/api/ai/enrich` | POST | AI content enrichment | OAuth |
| `/api/ai/write` | POST | AI writing assistant | OAuth |
| `/api/resume/parse` | POST | Parse uploaded resume | OAuth |

**Testimonials:**

| Route | Method | Description | Auth |
|-------|--------|-------------|------|
| `/api/testimonials` | GET | List testimonials | OAuth |
| `/api/testimonials/requests` | GET | List request links | OAuth |
| `/api/testimonials/requests` | POST | Create request link | OAuth |
| `/api/testimonials/requests/:id` | DELETE | Delete request link | OAuth |
| `/api/testimonials/request/:token` | GET | Validate request token | None |
| `/api/testimonials/submit` | POST | Submit testimonial | None |
| `/api/testimonials/:id/approve` | POST | Approve testimonial | OAuth |
| `/api/testimonials/:id/reject` | POST | Reject testimonial | OAuth |
| `/api/testimonials/:id` | PATCH | Update testimonial | OAuth |
| `/api/testimonials/:id` | DELETE | Delete testimonial | OAuth |
| `/api/testimonials/pending-count` | GET | Get pending count | OAuth |
| `/api/public/testimonials` | GET | Public approved list | None |
| `/api/testimonials/verify/email` | POST | Send verification email | None |
| `/api/testimonials/verify/email/:token` | GET | Complete verification | None |

**Demo Mode:**

| Route | Method | Description | Auth |
|-------|--------|-------------|------|
| `/api/demo/enable` | POST | Enable demo mode | OAuth |
| `/api/demo/disable` | POST | Disable demo mode | OAuth |
| `/api/demo/status` | GET | Check demo status | OAuth |

**Public REST API (v1):**

Versioned, read/write API authenticated with scoped API keys (`Authorization: Bearer <key>`). Scopes are `read:*` and `write:*` for `profile`, `posts`, `projects`, `skills`, and `experience`.

| Route | Method | Description | Scope |
|-------|--------|-------------|-------|
| `/api/v1/profile` | GET | Read profile | `read:profile` |
| `/api/v1/posts` | GET / POST | List / create posts | `read:posts` / `write:posts` |
| `/api/v1/posts/:id` | PATCH / DELETE | Update / delete post | `write:posts` |
| `/api/v1/projects` | GET / POST | List / create projects | `read:projects` / `write:projects` |
| `/api/v1/projects/:id` | PATCH / DELETE | Update / delete project | `write:projects` |
| `/api/v1/skills` | GET / POST | List / create skills | `read:skills` / `write:skills` |
| `/api/v1/skills/:id` | PATCH / DELETE | Update / delete skill | `write:skills` |
| `/api/v1/experience` | GET / POST | List / create experience | `read:experience` / `write:experience` |
| `/api/v1/experience/:id` | PATCH / DELETE | Update / delete experience | `write:experience` |

**Webhooks & System Alerts (admin):**

| Route | Method | Description | Auth |
|-------|--------|-------------|------|
| `/api/admin/webhooks` | GET / POST | List / create webhook (auto HMAC secret) | OAuth |
| `/api/admin/webhooks/:id` | PATCH / DELETE | Update / delete webhook | OAuth |
| `/api/admin/system-alerts` | GET | List alerts (filterable) | OAuth |
| `/api/admin/system-alerts/:id/acknowledge` | POST | Acknowledge one alert | OAuth |
| `/api/admin/system-alerts/acknowledge-all` | POST | Acknowledge all | OAuth |
| `/api/admin/system-alerts/:id` | DELETE | Remove alert | OAuth |

Webhooks sign each delivery with **HMAC-SHA256**, validate targets against **SSRF** protections, retry on failure, auto-disable after repeated failures, and record every attempt in `webhook_deliveries`.

### Security Model

#### Authentication

```
┌─────────────────────────────────────────────────────────────┐
│                    Authentication Flow                       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Admin Access:                                               │
│  ┌─────────┐    ┌──────────────┐    ┌─────────────────────┐ │
│  │ /admin  │───►│ OAuth Check  │───►│ Google OR GitHub    │ │
│  └─────────┘    └──────────────┘    │ (configured in PB)  │ │
│                        │            └─────────────────────┘ │
│                        ▼                                     │
│                 ┌──────────────┐                            │
│                 │ PocketBase   │                            │
│                 │ Auth Token   │                            │
│                 └──────────────┘                            │
│                                                              │
│  Public Access:                                              │
│  ┌─────────┐    ┌──────────────┐    ┌─────────────────────┐ │
│  │ /       │───►│ Visibility   │───►│ public: allow       │ │
│  │ /v/slug │    │ Check        │    │ unlisted: 404       │ │
│  │ /s/tok  │    └──────────────┘    │ private: 404        │ │
│  └─────────┘           │            │ password: prompt    │ │
│                        ▼            └─────────────────────┘ │
│                 ┌──────────────┐                            │
│                 │ Share Token  │                            │
│                 │ (if /s/tok)  │                            │
│                 └──────────────┘                            │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

#### Token Encryption

```go
// Encryption approach for stored API tokens

func encryptToken(plaintext string) (string, error) {
    key := os.Getenv("ENCRYPTION_KEY") // 32 bytes

    block, _ := aes.NewCipher([]byte(key))
    gcm, _ := cipher.NewGCM(block)

    nonce := make([]byte, gcm.NonceSize())
    io.ReadFull(rand.Reader, nonce)

    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptToken(encrypted string) (string, error) {
    // Only called server-side, never exposed to browser
    // ...
}
```

#### Share Token Security

1. Generate 32-byte random token
2. Store SHA-256 hash in database
3. Return raw token once to user
4. Validate by hashing incoming token and comparing
5. Use timing-safe comparison
6. Log all access attempts

#### Password Protection

1. Hash passwords with bcrypt (cost 12)
2. Store hash in `*_hash` field
3. Client submits password via POST
4. Server validates and issues short-lived session token
5. Session token stored in httpOnly cookie

### Import Pipeline

```
┌─────────────────────────────────────────────────────────────────────┐
│                       GitHub Import Pipeline                         │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  1. SOURCE CREATION                                                  │
│  ┌─────────────────┐                                                │
│  │ Admin selects   │    ┌──────────────────┐                        │
│  │ GitHub repo     │───►│ Create Source    │                        │
│  │ owner/repo      │    │ record           │                        │
│  └─────────────────┘    └────────┬─────────┘                        │
│                                  │                                   │
│  2. FETCH DATA                   ▼                                   │
│  ┌──────────────────────────────────────────────────────┐           │
│  │ GitHub API Calls:                                     │           │
│  │ ├── GET /repos/{owner}/{repo}        → metadata       │           │
│  │ ├── GET /repos/{owner}/{repo}/readme → README         │           │
│  │ ├── GET /repos/{owner}/{repo}/languages → languages   │           │
│  │ └── GET /repos/{owner}/{repo}/topics → topics         │           │
│  └──────────────────────────────────────────────────────┘           │
│                                  │                                   │
│  3. OPTIONAL AI ENRICHMENT       ▼                                   │
│  ┌──────────────────────────────────────────────────────┐           │
│  │ If user opts in:                                      │           │
│  │ ├── Send metadata + README (or summary) to AI         │           │
│  │ ├── Request: summary, bullets, tags, case study outline│          │
│  │ ├── Guardrails:                                       │           │
│  │ │   ├── "Do not invent metrics or statistics"         │           │
│  │ │   ├── "Stay factual, avoid marketing language"      │           │
│  │ │   └── "Use neutral, professional tone"              │           │
│  │ └── Return enriched proposal                          │           │
│  └──────────────────────────────────────────────────────┘           │
│                                  │                                   │
│  4. CREATE PROPOSAL              ▼                                   │
│  ┌──────────────────────────────────────────────────────┐           │
│  │ ImportProposal record created:                        │           │
│  │ ├── proposed_data: full project JSON                  │           │
│  │ ├── diff: field-by-field comparison (if update)       │           │
│  │ ├── ai_enriched: boolean                              │           │
│  │ └── status: "pending"                                 │           │
│  └──────────────────────────────────────────────────────┘           │
│                                  │                                   │
│  5. REVIEW UI                    ▼                                   │
│  ┌──────────────────────────────────────────────────────┐           │
│  │ Admin reviews each field:                             │           │
│  │ ├── [✓] Apply  - Use proposed value                   │           │
│  │ ├── [✗] Ignore - Keep current value                   │           │
│  │ ├── [🔒] Lock  - Apply + prevent future updates       │           │
│  │ └── [✎] Edit   - Modify before applying               │           │
│  └──────────────────────────────────────────────────────┘           │
│                                  │                                   │
│  6. APPLY CHANGES                ▼                                   │
│  ┌──────────────────────────────────────────────────────┐           │
│  │ POST /api/proposals/:id/apply                         │           │
│  │ ├── Create or update Project                          │           │
│  │ ├── Update field_locks on Project                     │           │
│  │ ├── Update Source.last_sync                           │           │
│  │ └── Mark proposal as "applied"                        │           │
│  └──────────────────────────────────────────────────────┘           │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Error Handling Approach

| Layer | Strategy |
|-------|----------|
| **PocketBase Hooks** | Return structured JSON errors with codes |
| **SvelteKit Server** | Catch errors, log to console, return user-friendly messages |
| **SvelteKit Client** | Toast notifications for actions, inline errors for forms |
| **API Calls** | Retry with exponential backoff (network), immediate fail (4xx) |
| **GitHub API** | Respect rate limits, queue requests, cache responses |
| **AI Providers** | Timeout after 30s, fallback to non-enriched data |

### Logging Approach

| Event Type | Logged Where | Retention |
|------------|--------------|-----------|
| Auth events | PocketBase logs | 30 days |
| API errors | Container stdout | Docker logging |
| Import operations | Source.sync_log | Permanent |
| Share token access | Dedicated audit log | 90 days |
| AI requests | Container stdout | Docker logging |

Configuration via environment:
- `LOG_LEVEL`: debug, info, warn, error
- `LOG_FORMAT`: json, text
