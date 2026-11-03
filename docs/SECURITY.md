# Security, ownership and privacy

This page explains, in plain terms first and technical detail second, who controls a self-hosted Facet, what it protects, what it records about visitors, and which outside services it talks to. Where the honest answer is "less than you might assume", it says so.

## The short version

- **You own the data.** Everything lives in a SQLite database and an uploads folder on your machine. There's no Facet account, licence server or telemetry.
- **Facet is built for one owner.** Every account that can sign in has full control of the site. `ADMIN_EMAILS` decides who that is, so set it.
- **Secrets are encrypted, content isn't.** AI keys, the GitHub token, Stripe secrets and two-factor secrets are encrypted in the database. Your profile and posts are stored as normal database rows, protected by your server's security.
- **Visitors' browsers do contact third parties.** Every page loads Google Fonts, and optional features add more. See [What leaves your server](#what-leaves-your-server).
- **Facet keeps its own visit statistics.** They're stored on your server and never shared, but they aren't anonymous. See [What Facet records about visitors](#what-facet-records-about-visitors).

## Who can sign in

Facet stores sign-in accounts in PocketBase's `users` table. Any signed-in account is treated as the site owner: the backend's admin endpoints check only that the request comes from a signed-in `users` account (or a PocketBase superuser), not which one ([helpers](../backend/hooks/helpers.go)). There are no roles.

`ADMIN_EMAILS` is the gate ([source](../backend/hooks/auth.go)):

| `ADMIN_EMAILS` | Password sign-in | Google / GitHub sign-in |
|----------------|------------------|-------------------------|
| Set, e.g. `you@example.com` | Only listed addresses | Only listed addresses |
| Empty | **Any account in `users`** | Allowed only while `users` is empty, i.e. never once the first account exists |

Facet doesn't change the `users` table's own PocketBase access rules. Treat an empty `ADMIN_EMAILS` as "anyone who manages to create an account can sign in", and set it before your instance is reachable by others.

### The first account

On first start with an empty database, Facet creates one account with the first address in `ADMIN_EMAILS` (or `admin@example.com`) and a random 20-character password. The password is printed once in the startup log; it isn't stored anywhere you can read it later ([seed](../backend/hooks/seed.go)). Treat that log as sensitive until you've changed the password under **Settings**. Passwords are stored as bcrypt hashes by PocketBase.

Older releases seeded every install with the published password `changeme123`. Only `SEED_DATA=dev` still does that, for local testing. Any account still using it is treated as signed out by admin APIs until the password is changed ([session guard](../backend/hooks/admin_session.go)), so an upgraded install that never changed it has to do so before anything else works.

`reset-admin-password` gives an account a new random password, prints it once, and signs out every existing session for that account. Two-factor sign-in stays on, but any existing two-factor browser session is cleared ([source](../backend/hooks/seed.go)).

### Facet accounts and the PocketBase dashboard

Two separate sets of credentials exist, and they're easy to mix up:

- **Facet owner accounts** live in PocketBase's `users` collection. This is what `/admin`, `ADMIN_EMAILS`, the password reset command and two-factor sign-in apply to.
- **PocketBase superusers** sign in to PocketBase's own database dashboard at `/_/`, which is off unless `ADMIN_ENABLED=true`. They're created with `./facet superuser upsert` and use PocketBase's own MFA settings, not Facet's two-factor check.

A superuser can change anything in the database, so keep the dashboard off when you aren't using it.

### Two-factor sign-in

Optional TOTP two-factor sign-in works with any authenticator app. Turn it on in **Account & Security** ([source](../backend/hooks/totp.go)).

| Aspect | Behaviour |
|--------|-----------|
| Codes | 6 digits, 30-second period, ±1 step allowed for clock drift |
| Secret | Encrypted in the database. The QR code is drawn in your browser, so the secret isn't sent to a QR service. |
| Recovery codes | 8 single-use codes, stored as bcrypt hashes |
| How long a check lasts | 24 hours per browser |
| Rate limit | Strict tier on setup, verify, disable and regenerate |

This is a check in Facet's admin layer, after PocketBase has accepted your password. It protects the dashboard. If your authenticator is lost, `reset-2fa` turns it off from the server ([Recovery](UPGRADE.md#locked-out)).

Known limits: verifying on a new browser replaces the session record, so other browsers are asked again; and changing settings doesn't ask for a fresh code.

### PocketBase's own dashboard

PocketBase includes a raw database dashboard at `/_/`. Facet's Caddy hides it (404) unless `ADMIN_ENABLED=true` ([Caddyfile](../docker/Caddyfile)). It needs a separate PocketBase superuser account, which Facet never creates for you. Keep it off unless you're fixing something.

## What's encrypted

Facet derives three keys from the master `ENCRYPTION_KEY` by hashing it with SHA-256 and a purpose label: one for encryption, one for HMAC, one for signing view-access tokens ([source](../backend/services/crypto.go)). Encryption uses AES-256-GCM.

| Encrypted with AES-256-GCM | Stored as a one-way hash |
|----------------------------|--------------------------|
| AI provider API keys | Account passwords (bcrypt, by PocketBase) |
| GitHub token for imports | Two-factor recovery codes (bcrypt) |
| Stripe secret key and webhook secret | Share-link tokens (HMAC-SHA256) |
| Two-factor (TOTP) secrets | Testimonial request tokens (HMAC-SHA256) |
| | API keys (HMAC-SHA256) |
| | Password-protected view passwords (bcrypt) |

Everything else, including drafts and private items, is stored unencrypted in SQLite. Anyone who can read `/data` or a backup archive can read your content. Protect the host, the backups and the logs accordingly. The automatically generated encryption key is printed once in the container log when it's created ([start script](../docker/start.sh)).

## What visitors can see

### Visibility

Views and individual items each have a visibility setting. [Features](FEATURES.md#visibility) explains how they combine. In security terms:

| Setting | Enforced by |
|---------|-------------|
| **Public** | Visible to anyone |
| **Unlisted view** | Needs a valid share link. Without one, the view returns 404. |
| **Password view** | Needs the password. A correct password gets a signed token valid for 1 hour. |
| **Unlisted item** (post, project, talk, …) | Not secret: anyone with the item's URL can open it, and it shows in any view that includes it. Left out of the `/posts` and `/talks` index pages, RSS, the sitemap and the API. |
| **Private** | Never served to visitors |
| **Draft** | Never served to visitors |

Visitors never read the database tables directly. Facet requires sign-in for direct access to content collections, and public pages are built by custom endpoints that apply the rules above ([rules](../backend/hooks/rules.go), [views](../backend/hooks/view.go)).

### Share links

A share link (`/s/<token>`) is the key to an unlisted view ([source](../backend/hooks/share.go)).

- Tokens are 32 random bytes. Facet stores only an HMAC of the token, plus its first 12 characters for lookup, so a copy of the database doesn't contain working links.
- Each link can have an expiry date and a maximum number of uses. You can revoke it at any time.
- Opening the link validates it, stores it in an `httpOnly` cookie for 7 days, and redirects to the clean URL (`/your-view`). The token doesn't stay in the address bar or history.
- Validation failures all return the same error, and the endpoint is rate limited.

Anyone you send a link to can forward it. Use an expiry date or a use limit for anything sensitive.

### Password-protected views

The visitor's password is checked against a bcrypt hash. A correct password returns a token signed with HMAC-SHA256 (HS256), bound to that view, with issuer and audience checks and a 1-hour lifetime ([source](../backend/hooks/password.go)). Tokens can't be revoked early. Changing the password stops new sign-ins but doesn't cancel tokens already issued. The check endpoint is rate limited to stop guessing.

### Contact details

Contact protection levels change how an email address or phone number is **displayed**. The full value is still sent to the visitor's browser with the page. They deter simple scrapers but don't keep anything secret. [Contact protection](CONTACT_PROTECTION.md) has the details.

## What Facet records about visitors

### Built-in analytics

Facet records page views for the **Analytics** dashboard ([source](../backend/hooks/analytics.go)). For each recorded visit it stores:

- the view or page visited, and the path
- the referring page, with its query string removed
- the browser's user-agent string
- any `utm_source`, `utm_medium` and `utm_campaign` values
- the share link used, if any
- a hash of the visitor's IP address combined with the current date

The IP hash changes every day, so visits can't be linked across days by hash alone. It isn't anonymisation, though: the salt is just the date, so someone with the database could recover the IP addresses by trying them all. Treat the analytics table as personal data.

This logging is on by default, records stay until you delete them, and there's no switch in this version to turn it off. The **Features** setting for analytics is described in [Features](FEATURES.md#analytics).

### Other visitor data

| Feature | What's stored |
|---------|---------------|
| Comments | Name, email, comment text, IP hash, user agent |
| Testimonials | What the person submits (name, title, company, relationship, text), plus their email if they choose to verify it |
| Newsletter | Email, optional name, list memberships. Sent emails include an open-tracking image and rewritten links that record opens and clicks. |
| Downloads | A hashed IP address per download |
| Courses (off by default) | Enrolment email and progress |

### Google Analytics

If you add a Google Analytics measurement ID in **Site Settings**, Facet adds Google's tag to public pages while the Analytics switch is on ([layout](../frontend/src/routes/+layout.svelte)). With the switch off, visitors don't receive the ID at all, and turning it off mid-visit removes the script and sets Google's opt-out flag.

The Content-Security-Policy shipped in the Docker image doesn't list Google's tag or collection domains ([Caddyfile](../docker/Caddyfile)), so browsers block those requests in a standard install. Optional Google Analytics doesn't work out of the box in this version. If you loosen the policy yourself, Google receives your visitors' data under Google's terms.

<!-- SUPERVISOR RECONCILE: a CSP expansion for optional Google Analytics is awaiting the owner's explicit approval. Leave this section saying GA is blocked until that change lands in docker/Caddyfile. -->


## What leaves your server

These are all the outside connections the code makes. Most only happen when you use the related feature.

**From visitors' browsers**

| Destination | When |
|-------------|------|
| Google Fonts (`fonts.googleapis.com`, `fonts.gstatic.com`) | Every page, admin and public ([app.html](../frontend/src/app.html), [fonts](../frontend/src/lib/fonts.ts)). Google sees each visitor's IP address and user agent. |
| YouTube, Vimeo, Loom, SoundCloud, Spotify, CodePen, Figma | Pages where you've embedded their media |
| Images hosted elsewhere | Pages that use external image URLs |
| Cloudflare Web Analytics | Allowed by the CSP so Cloudflare-injected beacons work. Facet doesn't add it itself. |
| Google Analytics | Only if configured, and blocked by the default CSP (above) |

**From your server**

| Destination | When |
|-------------|------|
| `raw.githubusercontent.com` | The admin's update check, at most once an hour ([source](../backend/hooks/version.go)) |
| `api.github.com` | GitHub project imports and refreshes |
| Your AI provider (Anthropic, OpenAI, Ollama or a custom endpoint) | When you, or a visitor using AI résumé generation, trigger an AI feature. The relevant content goes with the request. See [AI features](AI_FEATURES.md#what-gets-sent-to-the-provider). |
| oEmbed endpoints (YouTube, Vimeo, Loom, SoundCloud, CodePen, Figma) | When you add external media and Facet looks up its title and thumbnail |
| Your SMTP server | Sending email |
| Stripe | Payment set-up and checkout, if configured |
| Webhook URLs you register | When subscribed events happen |
| Google / GitHub | OAuth sign-in, if configured |

## Protections against abuse

### Rate limits

Facet limits requests per client IP with a token bucket ([source](../backend/services/ratelimit.go)):

| Tier | Rate | Burst | Used for, among others |
|------|------|-------|------------------------|
| Strict | 5 per minute | 3 | View passwords, password change, two-factor endpoints, testimonial submission, coupon checks, checkout |
| Moderate | 10 per minute | 5 | Share-link validation, testimonial request lookup, webhook admin, downloads |
| Normal | 60 per minute | 10 | Public view data, posts, talks, RSS, iCal, the `/api/v1` API |

Blocked requests get HTTP 429 with a `Retry-After` header. Limits are kept in memory, so a restart resets them. Behind a proxy, set `TRUST_PROXY` correctly or every visitor shares one bucket ([Configuration](CONFIGURATION.md#trust_proxy)).

AI résumé generation by visitors has its own limit of 5 per hour, keyed on the connecting address. Inside the Docker image that address is the internal proxy's, so in practice all anonymous visitors share those 5 generations an hour.

### Response headers

Caddy adds these to every response ([Caddyfile](../docker/Caddyfile)):

| Header | Value |
|--------|-------|
| `Content-Security-Policy` | Same-origin scripts, styles, connections and forms, with listed exceptions for Google Fonts, Cloudflare's beacon, and the embed providers above. `'unsafe-inline'` is allowed for scripts and styles because of how Svelte injects them. |
| `X-Frame-Options` | `SAMEORIGIN` (the admin live preview frames your own site) |
| `X-Content-Type-Options` | `nosniff` |
| `Referrer-Policy` | `strict-origin-when-cross-origin` |
| `Permissions-Policy` | Geolocation, microphone, camera, payment and USB disabled |
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` ([what that means for you](SELF-HOSTING-GUIDE.md#a-note-on-hsts)) |
| `Server` | Removed |

Request bodies are limited to 25 MB.

### Content sanitising

Rich text is sanitised with DOMPurify before display. Comments are shown as plain text, so any HTML in them is displayed rather than run. Project experience documents are validated on the server before they're saved ([experiences](experiences/contract.md)).

### API keys

Keys for the `/api/v1` API are shown once, stored as HMAC-SHA256 hashes keyed with your encryption key, carry explicit scopes, and can expire or be revoked ([API](API.md)). A key can also list allowed origins, which blocks browser requests from other sites. That is a guard against misuse from someone else's web page, not authentication: scripts can send any `Origin` header they like ([details](API.md#allowed-origins)). Writes made with a key are recorded in the audit log under the key's label and prefix.

### Webhooks

Deliveries are signed with HMAC-SHA256. Before connecting, Facet resolves the destination and refuses loopback, private, link-local, carrier-grade NAT and cloud-metadata addresses, so a webhook can't be aimed at your internal network ([source](../backend/services/webhooks.go)).

### Audit log

Facet records administrative actions in an audit log, shown under **Settings → Audit Log** ([source](../backend/hooks/audit.go)).

### Dependency advisories

Two advisories reach Facet through libraries it doesn't control directly.

- **go-jose v3 (GHSA-mgcc-cww5-fp2q), in the bundled Caddy.** A crafted RSA key in JSON (JWK) form can keep the CPU busy for a long time. Upstream fixed it only in go-jose v4, and Caddy's certificate code still needs v3. So the image builds Caddy against a local copy of v3.0.5 with the fix backported: RSA moduli above 4096 bits are rejected, and private keys must satisfy `p × q = n`, before the expensive check runs. The patch is limited to that fix plus a stricter bound when parsing JSON `\u` escapes; the other changes in the same upstream commit aren't included. [FACET-PATCH.md](../docker/caddy/third_party/go-jose-v3/FACET-PATCH.md) records the source, checksums and full diff, and [jose_test.go](../docker/caddy/jose_test.go) fails if Caddy stops using the patched copy. Remove the copy once Caddy's dependencies move to go-jose v4.
- **imaging and malformed TIFFs (CVE-2023-36308).** The `imaging` library can crash on a palette TIFF whose pixels point past the end of its palette. Facet's thumbnail service only accepts JPEG, PNG, GIF and WebP ([source](../backend/services/thumbnail.go)), but importing `imaging` also registers a TIFF decoder, so a TIFF can still reach it. The pinned `golang.org/x/image` (v0.47.0) checks palette indices while decoding, which stops the bad file before `imaging` sees it. [thumbnail_tiff_advisory_test.go](../backend/services/thumbnail_tiff_advisory_test.go) feeds it a malformed TIFF in a separate process and fails if that crashes, so a downgrade of `x/image` would show up there. `imaging` itself is still v1.6.2 without its own fix.

## Known limitations

- **Single owner.** No roles, no separate editor or viewer accounts.
- **Unlisted items aren't private.** Only unlisted *views* need a share link.
- **Visitors can trigger AI usage.** Anyone who can open a view can generate an AI résumé from it, on your AI provider account. The endpoint applies the view's own access check, so unlisted and password views still need a share link or password ([source](../backend/hooks/resume.go)).
- **Rate limits are per process and in memory.**
- **Uploaded files** are served by PocketBase according to each collection's rules. There are no expiring signed URLs.

## Reporting a security problem

Please don't open a public issue for a vulnerability. Use **Report a vulnerability** on the repository's [Security tab](https://github.com/jesposito/Facet/security) to send a private report to the maintainer, with steps to reproduce and the version you tested.
