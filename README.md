# Facet

[![Join the Facet HQ Discord](https://img.shields.io/badge/Discord-Join%20Facet%20HQ-5865F2?logo=discord&logoColor=white)](https://discord.gg/XD8eUudnmf)

[![Latest version](https://img.shields.io/github/v/tag/jesposito/Facet?color=blueviolet&label=version&sort=semver)](https://github.com/jesposito/Facet/tags)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Docker image](https://img.shields.io/badge/docker-ghcr.io%2Fjesposito%2Ffacet-2496ED?logo=docker&logoColor=white)](https://github.com/jesposito/Facet/pkgs/container/facet)
[![Build](https://img.shields.io/github/actions/workflow/status/jesposito/Facet/docker-publish.yml?branch=main&label=build)](https://github.com/jesposito/Facet/actions)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Svelte 5](https://img.shields.io/badge/Svelte-5.x-FF3E00?logo=svelte&logoColor=white)](https://svelte.dev)
[![Self-hosted](https://img.shields.io/badge/self--hosted-friendly-2ea44f?logo=docker&logoColor=white)](docs/SELF-HOSTING-GUIDE.md)
[![Stars](https://img.shields.io/github/stars/jesposito/Facet?style=social)](https://github.com/jesposito/Facet/stargazers)

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

## Try It Out

The fastest path is to spin up the Docker image locally:

```bash
docker run -d --rm -p 8080:8080 -v facet-data:/data \
  -e ENCRYPTION_KEY_AUTOPROVISION=true \
  ghcr.io/jesposito/facet:latest
```

Open `http://localhost:8080`, log in with `admin@example.com` / `changeme123`, change the password when prompted, and explore. See [docs/SELF-HOSTING-GUIDE.md](docs/SELF-HOSTING-GUIDE.md) for production setup (reverse proxy, OAuth, SMTP, AI keys).

> **Don't want to self-host?** A managed version is available at [get-facet.com](https://get-facet.com) — same software, hosted for you, with billing + custom domains + managed AI credits handled. Useful if you'd rather skip the Docker / DNS / backup parts.

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

# Copy the example config
cp .env.example .env

# Edit .env:
# - Set ADMIN_EMAILS to your email
# - (Optional) Set ENCRYPTION_KEY — if not set, one is auto-generated and saved to /data/.encryption_key
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
- Collect testimonials from clients and colleagues with approval workflow
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
- **Backend**: Go 1.25 with PocketBase (a lightweight backend framework)
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
| **Custom Content** | User-defined sections with Markdown | Embedded in views |
| **Testimonials** | Social proof from clients, colleagues, collaborators | Embedded in views |

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

### Quick Share to Social

Every public page has a share button that lets you:
- **Native sharing** on mobile (uses Web Share API for the native share sheet)
- **Copy link** with "Copied!" feedback
- **Share to LinkedIn, Twitter/X, Reddit, or Email** with one click

When sharing an unlisted view via token URL, the button warns that "This view is unlisted. Share links may expire." so recipients understand the link's nature.

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

Works on mobile. Context-aware (uses your form data for better results).

**Bring-your-own-key only.** Self-hosted Facet has no managed AI credits, no monthly token quotas, no platform-provided providers. You add your own OpenAI / Anthropic / Ollama key in `/admin/settings/integrations` and the backend calls those providers directly. Your API keys are encrypted at rest with AES-256-GCM. You see exactly what you're spending on whose meter.

### Contact Protection (Four Tiers)

Your email, phone, and social links can be protected:

| Level | What Happens | Use Case |
|-------|--------------|----------|
| **None** | Plain text, visible to everyone | LinkedIn, GitHub (already public) |
| **CSS Obfuscation** | Hidden with CSS, visible on hover | Light anti-bot protection |
| **Click-to-Reveal** | JavaScript toggle, user has to click | Moderate anti-bot protection |
| **CAPTCHA** | Turnstile challenge (planned) | Heavy anti-bot protection |

Plus, you can show different contact methods in different views. Example: recruiters see your email and phone, conference attendees only see your Twitter and LinkedIn.

### Testimonials System (Social Proof)

Collect and display testimonials from clients, colleagues, and collaborators to build credibility.

**How it works:**
1. Generate shareable request links with optional custom message and expiration
2. Links go to a public form where people submit testimonials (no account required)
3. Submissions are stored as "pending" for your review in the admin dashboard
4. You approve or reject testimonials before they appear publicly
5. Optional email verification adds credibility markers to testimonials
6. Approved testimonials display on your views with multiple layout options

**Features:**
- **Request link generation** - Custom messages, expiration dates, usage limits
- **Public submission form** - Name, title, company, relationship, testimonial text
- **Approval workflow** - Review all submissions before they go live
- **Email verification** - Optional verification for added credibility (15-minute tokens)
- **Admin management** - Collapsible sidebar section with pending count badge
- **Display options** - Wall layout, carousel, or featured testimonials
- **Privacy controls** - Show testimonials on specific views only
- **Security** - HMAC-SHA256 tokens, rate limiting, no raw tokens stored
- **Email notifications** - Admins receive styled HTML email when testimonials are submitted

Perfect for consultants, freelancers, job seekers, speakers, or anyone building professional credibility.

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

### Public REST API (For Scripts, Automations, and Integrations)

Need to read or write your Facet content from outside the admin UI? Facet ships a clean, scoped public API under `/api/v1/*`. Use it to mirror posts to your static site, build a custom resume page, or pipe new GitHub repos into your portfolio from CI.

**How it works:**

1. Go to **Admin → API** and create a key with the scopes you need
2. The key (`facet_...`) is shown once — store it like a password
3. Pass it as `Authorization: Bearer facet_...` on every request

**Scopes:** `read:*` and `write:*` for `profile`, `posts`, `projects`, `skills`, `experience`. Read and write are independent — granting `write:posts` does **not** imply `read:posts`.

**Example — list public posts:**
```bash
curl https://your-facet.example.com/api/v1/posts \
  -H "Authorization: Bearer facet_..."
```

**Example — create a draft post:**
```bash
curl -X POST https://your-facet.example.com/api/v1/posts \
  -H "Authorization: Bearer facet_..." \
  -H "Content-Type: application/json" \
  -d '{"title":"Hello","slug":"hello","content":"Markdown body","is_draft":true}'
```

Keys can be revoked, expired on a date, and limited to specific browser origins (for CORS-style usage). Raw keys are never stored — only the SHA-256 hash.

### Webhooks (Push Events to Other Systems)

Get notified when something happens in Facet. Configure webhook endpoints under **Admin → Settings → Webhooks** and subscribe to events.

**Events available:**
- `post.published` — a post (or talk, or custom content) transitioned out of draft
- `comment.created` — someone left a comment on a public item
- `newsletter.subscribed` — a new subscriber confirmed

**Security:**
- Every payload is signed with HMAC-SHA256 over the JSON body (`X-Facet-Signature: sha256=<hex>`)
- The signing secret is shown once at create time
- SSRF protection: hostnames are resolved at dial-time and rejected if they resolve to private/loopback/link-local/CGNAT/metadata IPs (no firing webhooks at `localhost` or `169.254.169.254`)
- Endpoints that fail 10 times in a row are auto-disabled
- Re-enabling resets the failure counter

**Testing:**
- "Send test" on each webhook fires a synthetic delivery and shows the response inline
- The **dispatch test event** picker fires a realistic synthetic payload to all active subscribers of a chosen event so you can validate your receiver
- Every attempt is logged with status code, response body, and attempt count

### System Alerts Inbox

When something goes wrong in the background — a failed backup, an SMTP error, repeated webhook delivery failures — it lands at `/admin/alerts`. Three severities (`info` / `warning` / `critical`), filterable, acknowledgable individually or all at once. Each alert can carry structured metadata for richer rendering. Backend emitters call a single `CreateSystemAlert` helper; failures to write an alert are silently logged so the alert subsystem can never break the hot path that triggered it.

The sidebar link is hidden today — the page works but the inbox stays empty until emitters are wired into backup, SMTP, and webhook code paths. Re-enable the sidebar entry in `AdminSidebar.svelte` once those are in place.

### Newsletter Lists (Segments)

Run more than one mailing list from a single Facet instance. Default install has a single "Newsletter" list seeded for you; create more as needed under **Admin → Newsletter → Lists**.

**Per-list configuration:**
- Custom name, slug, description
- Per-list sender name and reply-to address
- Per-list welcome email subject and HTML body
- Active/inactive toggle (pauses subscribes without losing existing members)
- One list is always the "default" (you can swap the flag atomically)

**Compose UI:** **Admin → Newsletter → Compose** lets you pick which lists to send to, preview the rendered email, send a test to yourself, and dispatch a broadcast. The newsletter hub at `/admin/newsletter` shows recent sends, subscriber counts, and a per-list overview.

Subscriber-count cache is kept honest by automatic recomputation hooks whenever memberships are created or deleted. The join table cascade-deletes correctly so subscribers keep their other memberships if you delete a single list.

### Comments (Threaded, Moderated, Per-Item Toggle)

Visitors can comment on any post, project, or talk where you've enabled the toggle. Threaded replies, markdown support, rate limiting, deny-by-default moderation queue at `/admin/comments`. Reports flow into the same queue so abusive content has a path. Per-item `comments_enabled` flag means you can leave the system on globally but turn it off for any single piece. All comment text passes through DOMPurify before render. No third-party services involved — comments live in your SQLite database.

### Multi-Language UI (5 Built-In Locales)

The admin and public surfaces ship in English, German, Sindarin-flavored Elvish, Klingon, and LOLcat. English is canonical; the others are translations. 100% key parity is enforced in CI (`npm run i18n:validate`) so locale drift can't ship. Adding a new language is mechanical: copy `frontend/src/locales/en.json`, translate, validate.

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
- **Go 1.25** (backend language)
- **PocketBase v0.23.4** (lightweight backend framework built on SQLite and Fiber)
- **SQLite** (embedded database, single file)
- **AES-256-GCM** (encryption for API keys and tokens)
- **Bcrypt** (password hashing)
- **JWT** (session tokens for password-protected views)

**Frontend:**
- **SvelteKit v2.0** (full-stack web framework)
- **Svelte v5.0** (component framework)
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
- 12/12 public tests passing (admin tests require credentials)

**Development:**
- **Air** (Go hot reload)
- **Vite v7.3** (frontend dev server with HMR)
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
| `ENCRYPTION_KEY` | No | Auto-generated | 32-byte hex key for encrypting API keys and tokens. If not set, one is auto-generated and saved to `/data/.encryption_key`. Generate manually with `openssl rand -hex 32` |
| `ENCRYPTION_KEY_AUTOPROVISION` | No | `false` | If `true`, the entrypoint generates `ENCRYPTION_KEY` on first boot and persists it to `/data/.encryption_key`. Convenient for self-hosted single-node setups; explicit `ENCRYPTION_KEY` wins if both are set |
| `ALLOWED_ORIGINS` | No | — | Comma-separated origin allowlist for CORS-style API responses (e.g., `https://your.site,https://api.your.site`). Used by the public REST API and admin endpoints when called from a browser |
| `PORT` | No | `8080` | Public port for the app |
| `APP_URL` | No | `http://localhost:8080` | Your public URL (needed for OAuth callbacks) |
| `PROTOCOL_HEADER` | No | `x-forwarded-proto` | SvelteKit adapter-node header for protocol detection behind a reverse proxy. Defaults to the standard header so HTTPS detection works out of the box |
| `HOST_HEADER` | No | `x-forwarded-host` | SvelteKit adapter-node header for host detection behind a reverse proxy |
| `ADMIN_EMAILS` | No | — | Comma-separated email allowlist for OAuth login |
| `TRUST_PROXY` | No | `false` | Set `true` if behind a reverse proxy (Nginx, Cloudflare, etc.) |
| `ADMIN_ENABLED` | No | `false` | Enable PocketBase admin UI at `/_/` (use for debugging only) |
| `DATA_PATH` | No | `./data` | Where to store the database and uploads |
| `GOOGLE_CLIENT_ID` | No | — | OAuth via Google |
| `GOOGLE_CLIENT_SECRET` | No | — | OAuth via Google |
| `GITHUB_CLIENT_ID` | No | — | OAuth via GitHub |
| `GITHUB_CLIENT_SECRET` | No | — | OAuth via GitHub |
| `SMTP_HOST` | No | — | SMTP server hostname (e.g., `smtp.gmail.com`) |
| `SMTP_PORT` | No | `587` | SMTP server port |
| `SMTP_USERNAME` | No | — | SMTP authentication username |
| `SMTP_PASSWORD` | No | — | SMTP authentication password |
| `SMTP_SENDER_NAME` | No | `Facet` | Display name for outgoing emails |
| `SMTP_SENDER_ADDRESS` | No | — | From address for outgoing emails |
| `ADMIN_ENABLED` | No | `false` | Enable PocketBase admin UI at `/_/` (debugging only) |
| `UPLOADS_PATH` | No | `./uploads` | Where to store uploaded files |
| `SEED_DATA` | No | — | Seed mode: `dev` for development profile |

Full setup guide (OAuth, reverse proxy, Unraid, etc.): [docs/SETUP.md](docs/SETUP.md)

---

## Backup and Restore (Super Simple)

Everything lives in two directories: `./data` (database, encryption key, settings) and `./uploads` (user-uploaded media). Both should be backed up.

**Backup:**
```bash
docker compose down
tar -czvf facet-backup-$(date +%Y%m%d).tar.gz ./data ./uploads
docker compose up -d
```

**Restore:**
```bash
docker compose down
tar -xzvf facet-backup-20260516.tar.gz
docker compose up -d
```

That's it. The tarball contains your SQLite database, your encryption key (`/data/.encryption_key` — keep it safe, encrypted data can't be recovered without it), and all uploaded files.

For PocketBase-native backups (`app.CreateBackup`) with WAL-checkpointed snapshots, use **Admin → Settings → Backups** in the UI.

For upgrade procedures: [docs/UPGRADE.md](docs/UPGRADE.md)

---

## Security (The Boring But Important Stuff)

**Authentication:**
- OAuth 2.0 (Google, GitHub)
- Email allowlist (`ADMIN_EMAILS`)
- Session tokens in httpOnly cookies
- First-time password change enforcement for default credentials
- Optional TOTP-based two-factor authentication (any authenticator app, with recovery codes)

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
- Go 1.25+
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
│   │   ├── resume.go        # Resume generation (518 lines)
│   │   ├── testimonials.go # Testimonial endpoints (700+ lines)
│   │   ├── smtp.go         # SMTP settings management
│   │   └── ...
│   ├── services/            # Reusable business logic (6K+ lines)
│   │   ├── ai.go            # AI provider integration
│   │   ├── crypto.go        # AES encryption
│   │   ├── github.go        # GitHub API
│   │   ├── email.go        # Email notifications & verification
│   │   ├── email_i18n.go   # Email translations (5 locales)
│   │   ├── testimonial.go  # Token generation & HMAC
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

**Code stats** (approximate, drifts with every release):
- ~45,000 lines across 1,400+ files
- Backend: ~19,000 lines of Go (hooks + services + migrations)
- Frontend: ~23,000 lines of Svelte/TypeScript across 5 locales
- Tests: ~3,500 lines (Go unit tests + Playwright E2E)

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
- `GET /admin/settings/account` → Account & security (password, 2FA)
- `GET /admin/settings/site` → Site settings, SMTP/email configuration
- `GET /admin/settings/integrations` → AI providers & integrations
- `GET /admin/settings/webhooks` → Webhook endpoints, secrets, delivery log
- `GET /admin/api` → API key management (create, scope, revoke)
- `GET /admin/alerts` → System alerts inbox (route is live; sidebar link is hidden until alert emitters are wired into backup / SMTP / webhook code paths)
- `GET /admin/newsletter` → Newsletter overview
- `GET /admin/newsletter/lists` → Manage newsletter lists (segments)
- `GET /admin/newsletter/compose` → Compose and send broadcasts
- (Plus routes for education, certifications, skills, posts, talks, awards, contacts, tokens)

**API routes** (via PocketBase hooks):
- `GET /api/homepage` → Fetch homepage data
- `POST /api/github/import` → Import from GitHub
- `POST /api/ai/enrich` → AI enrichment
- `GET /api/export?format=json|yaml` → Data export
- `POST /api/share/validate` → Validate share token
- `GET /api/v1/{profile,posts,projects,skills,experience}` → Public read API (requires `read:*` scope)
- `POST/PATCH/DELETE /api/v1/{posts,projects,skills,experience}` → Public write API (requires `write:*` scope)
- `PATCH /api/v1/profile` → Update the singleton profile (requires `write:profile`)
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
| [docs/SELF-HOSTING-GUIDE.md](docs/SELF-HOSTING-GUIDE.md) | Beginner guide: install, expose, API keys, webhooks, alerts, newsletter |

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
- ✅ Testimonials system with request links, approval workflow, email verification
- ✅ Email notification system for testimonial submissions (SMTP, i18n, admin alerts)
- ✅ Admin settings split into Account, Site Settings, and Integrations pages
- ✅ Custom content sections for user-defined content
- ✅ Version update notifications (checks GitHub for new releases)
- ✅ Automated changelog generation from PR descriptions
- ✅ Optional TOTP two-factor authentication with recovery codes
- ✅ Public REST API (`/api/v1/*`) with scoped API keys (read + write)
- ✅ Webhooks with HMAC signing, SSRF protection, retry + auto-disable, delivery log
- ✅ System alerts inbox (operator-facing event log)
- ✅ Newsletter lists / segments (multi-list with per-list sender + welcome email)
- ✅ Newsletter compose UI with list picker, preview, test send
- ✅ External media expanded: Loom, CodePen, Figma oEmbed providers
- ✅ Threaded comments with moderation queue, reports, per-item enable toggle
- ✅ Multi-language UI (en / de / elvish / klingon / lolcat) with CI-enforced parity
- ✅ Webhook dispatch test event picker (fire synthetic payloads to all matching subscribers)
- ✅ Automigrate on boot for self-hosted (schema changes apply on pull + restart)

**Coming Soon:**
- CAPTCHA contact protection (Cloudflare Turnstile integration)
- Scheduled GitHub sync (auto-refresh projects)
- Newsletter drip sequences (multi-step automated send chains)
- Full Liquid templating engine for per-subscriber personalization at send time (compose UI has a placeholder shim today)
- System alert emitters wired into backup, SMTP, and webhook delivery failure code paths

**Planned (Lower Priority):**
- Content Security Policy headers
- Theme system with pre-built themes
- Lucide icon system migration (consolidate inline SVGs into a single icon set)

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
