# Features guide

Facet is a personal site you run yourself: a profile, a portfolio, writing and talks, presented differently to different audiences. This page explains each feature from the owner's side, including what it doesn't do yet.

Sign in at `/admin` to manage everything. Sidebar groups and labels are given in **bold** as they appear in English.

## Contents

- [Your content](#your-content)
- [Views](#views)
- [Visibility](#visibility)
- [Share links](#share-links)
- [Password-protected views](#password-protected-views)
- [Homepage, navigation and design](#homepage-navigation-and-design)
- [Testimonials](#testimonials)
- [Comments](#comments)
- [Analytics](#analytics)
- [Importing content](#importing-content)
- [Exporting your data](#exporting-your-data)
- [Printing and résumés](#printing-and-résumés)
- [Feeds and search engines](#feeds-and-search-engines)
- [Languages](#languages)
- [Demo mode](#demo-mode)
- [Housekeeping tools](#housekeeping-tools)
- [Features covered elsewhere](#features-covered-elsewhere)

## Your content

| Content | Admin page | Public page |
|---------|------------|-------------|
| Profile: name, headline, summary, location, avatar, hero image | **Profile** | Shown on your views |
| Experience: roles, dates, bullets, company logos | **Experience** | Inside views |
| Projects: summary, description, tech stack, links, media | **Projects** | `/projects/<slug>` |
| Education, certifications, awards | their own pages | Inside views |
| Skills, grouped by category | **Skills** | Inside views |
| Posts (articles) | **Posts** | `/posts` and `/posts/<slug>` |
| Talks: event, date, slides and video links | **Talks** | `/talks` and `/talks/<slug>` |
| Custom content: your own sections | **Custom Content** | Inside views |
| Contact methods | **Contact Methods** | Inside views |
| Testimonials | **Testimonials** | Inside views |

Most items share a few settings:

- **Visibility**: public, unlisted or private ([below](#visibility)).
- **Draft**: posts, talks, projects and experience can be saved as drafts, which visitors never see.
- **Featured**: projects and posts can be highlighted.
- **Order**: drag to reorder in most lists.
- **Media**: attach uploaded files or external embeds ([Media](MEDIA.md)).
- **Duplicate**: copy an item as a starting point for a similar one.

Projects can also carry a richer page: a case study, a video course or a code walkthrough. See [Project experiences](experiences/README.md).

## Views

A view is a version of your site for one audience. You might keep a recruiter view that leads with experience, a conference view that leads with talks, and a consulting view built around case studies, all from the same content. Edit one project and every view that includes it updates.

Each view has its own address, `/<slug>`, for example `/recruiter`. Under **Views** you can set, per view:

- which sections appear, in what order, and with which layout
- which individual items appear in each section
- per-item overrides, such as a different project summary for this audience
- a hero headline, summary, image and location that replace your profile's for this view
- a call-to-action button with its own text and link
- an accent colour
- its own visibility: public, unlisted, password or private

The live preview pane shows the view as visitors will see it while you edit.

**The default view** is what visitors see at `/`. Facet uses the view marked as default, provided it's active and public. If there isn't one, it uses the first active public view. If there are no public views at all, it builds a homepage from all your public content.

**Slugs** may contain letters, numbers, hyphens and underscores, must start with a letter or number, and can be up to 100 characters. Some words are reserved because they're already used for other pages, including `admin`, `api`, `s`, `v`, `posts`, `projects`, `talks`, `courses`, `health`, `login`, `profile`, `testimonial`, `unsubscribe` and `facet` ([full list](../frontend/src/params.ts)).

Old links of the form `/v/<slug>` redirect to `/<slug>`.

## Visibility

Visibility works at two levels, and the word "unlisted" means something different at each. This catches people out, so it's worth a careful read.

**Views** have four settings:

| View visibility | Who can open it |
|-----------------|-----------------|
| Public | Anyone. Listed in your site navigation and sitemap. |
| Unlisted | Only people with a [share link](#share-links). Without one, the address returns "not found". |
| Password | Anyone who knows the password you set ([below](#password-protected-views)) |
| Private | Only you, while signed in |

**Items** (posts, projects, talks, experience and so on) have three:

| Item visibility | What happens |
|-----------------|--------------|
| Public | Appears in any view that includes it, on index pages, in RSS, the sitemap and the API |
| Unlisted | Appears in views that include it, and anyone with its direct URL can open it. Left out of the `/posts` and `/talks` index pages, RSS, `talks.ics`, the sitemap, the all-content homepage fallback and the API ([source](../backend/hooks/view.go)). |
| Private | Never shown to visitors |

So an unlisted *view* is gated, but an unlisted *item* is only kept out of listings, feeds and search engines. Anyone who has its address, or who can see a view that includes it, can still read it. If something must stay hidden, make it private, or put it in an unlisted or password view and make sure it isn't public elsewhere.

Contact methods also have per-view switches, so recruiters can see your phone number while conference visitors see only your social links.

## Share links

Share links let you send an unlisted view to specific people. Create them in a view's editor under **Views**, or on the **Share Tokens** page (`/admin/tokens`).

- Give each link a name, so you know who you sent it to.
- Optionally set an expiry date and a maximum number of uses.
- The full link is shown **once**, when you create it. Copy it then.
- Revoke a link at any time. Revoked links are cleaned up after 30 days.

When someone opens `/s/<token>`, Facet checks the link, remembers it in a cookie for 7 days, and redirects them to the view's clean address. They don't need an account. [Security](SECURITY.md#share-links) has the technical details.

## Password-protected views

Set a view's visibility to **Password** and choose a password. Visitors see a password prompt. A correct password unlocks the view for one hour in that browser. Wrong guesses are rate limited.

## Homepage, navigation and design

**Site Settings** controls how your whole site looks and behaves, including:

- **Homepage on or off.** With the homepage off, `/`, `/posts` and `/talks` show a short message you write, while your views stay reachable at their own addresses. This suits people who only want to share specific views.
- **Homepage layout**: which sections appear on the default page and in what order (**Homepage**).
- **Navigation**: how and where the site menu appears.
- **Design**: font pack, hero layout and spacing, accent and text colours, light/dark default, avatar display.
- **Custom CSS** for the whole site. It's checked before saving.
- **Favicon** upload.
- **Call-to-action buttons** in the header and footer.
- **Hide the login button** on public pages. `/admin` still works.
- **Default language** for visitors.
- **Email (SMTP)** settings and a test-send button.

## Testimonials

Collect recommendations from clients and colleagues without them needing an account ([reference](specs/TESTIMONIALS.md)).

1. Under **Testimonials → Request Links**, create a link. You can add a personal message, the recipient's name and email, an expiry date, a use limit, and the list it should go into.
2. Send the link. The recipient fills in a form at `/testimonial/<token>` with their name, title, company, relationship and testimonial.
3. They can optionally verify their email address. Facet emails a link that's valid for 15 minutes, and verified testimonials are marked as such.
4. New submissions arrive as **pending**. Facet emails you when one comes in (if SMTP is set up).
5. Approve or reject each one under **Testimonials → Manage**. Only approved testimonials appear on your site.

You can group testimonials into lists and choose a layout (wall, carousel or featured) for each view.

## Comments

Visitors can comment on posts, projects, custom content and course lessons where you've switched comments on for that item. Talks don't take comments ([source](../backend/hooks/comments.go)). Comments are off for each item until you enable them.

- Visitors give a name and email. Their email isn't shown publicly.
- Replies are threaded. You can reply as the owner and pin comments.
- New comments are **published immediately** unless they look like spam. A comment with more than two links is marked as spam, and so is one containing a word from your blocked-words list ([source](../backend/hooks/comments.go)). The setting that holds every comment for approval exists in the database, but this version has no control for it in the admin.
- Visitors can report a comment. Reports and spam appear under **Comments**, where you can approve, reject, mark as spam or delete, one at a time or in bulk.
- Posting is rate limited and duplicate comments are rejected.

## Analytics

**Analytics** shows total and unique visitors, views over time, your most-viewed views, share-link usage, top referring sites and UTM campaigns ([source](../backend/hooks/analytics.go)). A countries panel is present but always empty, because Facet doesn't look up locations. A view counter ignores likely bots and repeat visits from the same visitor within 30 minutes.

The data stays on your server. It does include each visitor's user agent and a daily-rotating hash of their IP address. [Security](SECURITY.md#what-facet-records-about-visitors) explains exactly what's stored and its limits.

**Settings → Features** has an on/off switch for analytics, courses, newsletter and discussions ([feature gate](../backend/hooks/feature_gate.go)):

- **Analytics off** stops view counting, and stops visitors receiving your Google Analytics ID. The GA script is removed if the switch goes off mid-visit.
- **Courses off** closes the course catalogue and new enrolments. Selling courses is Facet Cloud-only, so there is no course checkout either way. Existing learners keep access ([details](AUDIENCE.md#turning-courses-off)).
- **Discussions off** closes the public comment routes.
- **Newsletter off** closes sign-up and the public list of lists. Unsubscribing always works. A confirmation link sent earlier still records consent, but no welcome email goes out while the newsletter is off ([source](../backend/hooks/newsletter.go)).

## Importing content

- **GitHub.** Under **Import & AI**, enter a repository (or a GitHub URL) to preview and import it as a project: name, description, README, languages and topics. Add a GitHub token on the same page for private repositories and higher rate limits. It's stored encrypted. With an AI provider, Facet can also draft a summary, bullets and tags. Every import becomes a proposal you review before anything changes. Fields you've edited can be **locked**, so refreshing from GitHub won't overwrite them.
- **Résumé.** Upload a PDF or Word (`.docx`) résumé, up to 10 MB, and an AI provider turns it into experience, education, skills, projects, certifications and awards. This needs an AI provider. See [AI features](AI_FEATURES.md#résumé-import).

There's no LinkedIn importer.

## Exporting your data

**Site Settings** lets you download everything as JSON or YAML (`/api/export?format=json` or `format=yaml`, signed-in only). The export includes your profile, all content collections, views and settings ([source](../backend/hooks/export.go)).

It deliberately leaves out accounts, API keys, share links, verification tokens, audit and access logs, system alerts and webhook delivery history, and strips password hashes and secrets from every record. Uploaded files aren't inside the export.

Facet can't import its own export files. Use the export as a portable copy of your content, and use [backups](UPGRADE.md#back-up-your-data) to move or restore an installation.

## Printing and résumés

Every public page has a print stylesheet, so your browser's Print command produces a clean document.

With an AI provider configured, the print menu also offers an **AI résumé**: Facet asks the AI to write a résumé from the view's content, then converts it to PDF or Word with Pandoc. Visitors can do this too, on any view they're allowed to open, using your AI provider account. [AI features](AI_FEATURES.md#ai-résumé-generation-ai-print) explains the options and costs.

## Feeds and search engines

| Address | What it is |
|---------|------------|
| `/rss.xml` | RSS feed of public, published posts |
| `/talks.ics` | Calendar file of public, published talks |
| `/sitemap.xml` | Public views, posts, projects and talks |
| `/robots.txt` | Instructions for crawlers |

Pages include titles, descriptions, Open Graph and Twitter card tags, canonical URLs, and JSON-LD structured data.

Both `/robots.txt` and `/sitemap.xml` are generated on each request ([robots source](../frontend/src/lib/server/robots.ts)). Their absolute links use `PUBLIC_APP_URL`, then `APP_URL`, and fall back to the address the request came in on when neither is set. Set `APP_URL` to your public address so search engines get the right sitemap link.

`/robots.txt` asks known AI-training crawlers (GPTBot, ClaudeBot, Google-Extended, CCBot and others) to stay away from the whole site. Every other crawler, search engines included, is allowed except on `/admin`, `/api/`, share links (`/s/`, `/v/`), `/unsubscribe`, `/testimonial/` and `/checkout`. The file ends with a `Sitemap:` line pointing at your sitemap. Blocking crawlers relies on them honouring `robots.txt`; it isn't enforced.

## Languages

The admin and public pages are available in English, German, and three playful translations: Sindarin-flavoured Elvish, Klingon and LOLcat. Set the default for visitors in **Site Settings**. Translations are checked for missing keys in CI. [Contributing translations](../CONTRIBUTING_TRANSLATIONS.md) explains how to add a language.

## Demo mode

The demo toggle in the admin header loads a complete sample site: a profile for "The Doctor", a Time Lord with 900+ years of experience, complete with views, projects, posts, talks and testimonials. It's a quick way to see what a finished Facet looks like.

Demo data lives in separate tables. Your own content isn't changed, and it comes back when you switch the demo off ([source](../backend/hooks/demo.go)). While the demo is on, **your public site shows the demo content** to everyone, so try it before you share your address, or briefly.

You can hide the toggle once you're set up, under **Site Settings → General**. From a server shell, `./facet demo enable`, `./facet demo disable` and `./facet demo status` do the same job (use the [wrapper command](UPGRADE.md#locked-out) in Docker).

## Housekeeping tools

- **Admin Tags**: private labels for organising items. Visitors never see them.
- **Audit Log**: a record of administrative actions.
- **About Facet**: version, changelog, update notice and [built-in backups](UPGRADE.md#built-in-backups).
- **Help**: in-app guidance.
- **Settings → System Alerts** (`/admin/alerts`): an inbox for content-recovery, backup, restore and retention failures ([API](API.md#system-alerts)).

## Features covered elsewhere

- [Project experiences](experiences/README.md): case studies, video courses and code walkthroughs
- [Media library](MEDIA.md): uploads, embeds, cleanup
- [Contact protection](CONTACT_PROTECTION.md)
- [Newsletter, payments and courses](AUDIENCE.md)
- [AI features](AI_FEATURES.md)
- [API, webhooks and integrations](API.md)
