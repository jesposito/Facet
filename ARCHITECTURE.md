# OwnProfile Architecture Document

## Phase 1: Complete Architecture

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
│  │  ├── /              │◄───┤  ├── profile                         ││
│  │  ├── /v/:slug       │    │  ├── experience                      ││
│  │  ├── /s/:token      │    │  ├── projects                        ││
│  │  └── /p/:project    │    │  ├── education                       ││
│  │                     │    │  ├── certifications                  ││
│  │  Admin Pages:       │    │  ├── skills                          ││
│  │  ├── /admin         │───►│  ├── posts                           ││
│  │  ├── /admin/views   │    │  ├── talks                           ││
│  │  ├── /admin/import  │    │  ├── views                           ││
│  │  └── /admin/review  │    │  ├── share_tokens                    ││
│  │                     │    │  ├── sources                         ││
│  └─────────────────────┘    │  ├── ai_providers                    ││
│                             │  ├── import_proposals                ││
│                             │  └── media                           ││
│                             │                                      ││
│                             │  Custom Go Hooks:                    ││
│                             │  ├── /api/github/import              ││
│                             │  ├── /api/github/refresh             ││
│                             │  ├── /api/ai/enrich                  ││
│                             │  ├── /api/ai/test                    ││
│                             │  ├── /api/view/access                ││
│                             │  └── /api/share/validate             ││
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
```

### Route Map

#### Public Routes (SvelteKit)

| Route | Description | Auth |
|-------|-------------|------|
| `GET /` | Main public profile | Based on profile.visibility |
| `GET /v/:slug` | View by slug | Based on view.visibility |
| `GET /s/:token` | Share token access | Token validation |
| `GET /p/:slug` | Project detail page | Based on project.visibility |
| `GET /post/:slug` | Blog post page | Based on post.visibility |
| `POST /api/password-check` | Validate password for protected content | None |

#### Admin Routes (SvelteKit)

| Route | Description | Auth |
|-------|-------------|------|
| `GET /admin` | Admin dashboard | OAuth required |
| `GET /admin/profile` | Edit profile | OAuth required |
| `GET /admin/experience` | Manage experience | OAuth required |
| `GET /admin/projects` | Manage projects | OAuth required |
| `GET /admin/education` | Manage education | OAuth required |
| `GET /admin/certifications` | Manage certs | OAuth required |
| `GET /admin/skills` | Manage skills | OAuth required |
| `GET /admin/posts` | Manage posts | OAuth required |
| `GET /admin/talks` | Manage talks | OAuth required |
| `GET /admin/views` | Manage views | OAuth required |
| `GET /admin/views/:id` | Edit specific view | OAuth required |
| `GET /admin/sources` | Manage import sources | OAuth required |
| `GET /admin/import` | GitHub import wizard | OAuth required |
| `GET /admin/review/:id` | Review import proposal | OAuth required |
| `GET /admin/settings` | AI providers, app settings | OAuth required |
| `GET /admin/media` | Media library | OAuth required |

#### API Routes (PocketBase + Custom Hooks)

| Route | Method | Description | Auth |
|-------|--------|-------------|------|
| `/api/collections/*` | * | PocketBase CRUD | API rules |
| `/api/github/repos` | GET | List user's GitHub repos | OAuth |
| `/api/github/import` | POST | Start import from GitHub | OAuth |
| `/api/github/refresh/:id` | POST | Refresh source from GitHub | OAuth |
| `/api/ai/enrich` | POST | Request AI enrichment | OAuth |
| `/api/ai/test/:id` | POST | Test AI provider connection | OAuth |
| `/api/proposals/:id/apply` | POST | Apply import proposal | OAuth |
| `/api/proposals/:id/reject` | POST | Reject import proposal | OAuth |
| `/api/share/validate` | POST | Validate share token | None |
| `/api/view/:slug/access` | GET | Check view access | Token/Password |

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
