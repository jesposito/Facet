# Me.yaml Security

This document describes security features, authentication flows, and known limitations.

## Encryption

### Master Encryption Key

All sensitive data is encrypted using AES-256-GCM. The encryption key is derived from the `ENCRYPTION_KEY` environment variable.

**Requirements:**
- Minimum 32 characters (256 bits)
- Generate with: `openssl rand -hex 32`
- Application **will not start** without a valid key

**What's encrypted:**
- AI provider API keys (`ai_providers.api_key_encrypted`)
- GitHub tokens (`settings.github_token`)

**Key derivation:**
The master key is stretched into separate keys for different purposes:
- Encryption key: `SHA256(master + ":encryption")`
- HMAC key: `SHA256(master + ":hmac")`
- JWT signing key: `SHA256(master + ":jwt")`

## Authentication

### Admin Authentication

Admin users authenticate via:
1. **OAuth2** (Google, GitHub) - preferred
2. **Password** - fallback

Access is controlled by the `ADMIN_EMAILS` environment variable (comma-separated list).

### View Access Levels

| Visibility | Access Control | HTTP Response |
|------------|----------------|---------------|
| `public` | Anyone | 200 OK |
| `unlisted` | Valid share token required | 200 with token, 404 without |
| `password` | Valid password JWT required | 200 with JWT, password prompt without |
| `private` | Admin only | 404 (not 401 - prevents discovery) |

### URL Routing Model

Me.yaml uses LinkedIn-style canonical URLs:

| Route | Purpose | Notes |
|-------|---------|-------|
| `/` | Default public profile | Renders view with `is_default=true` |
| `/<slug>` | Named view (canonical) | e.g., `/recruiter`, `/investor` |
| `/s/<token>` | Share link entry | Sets cookie, redirects to `/<slug>` |
| `/v/<slug>` | Legacy route | 301 redirects to `/<slug>` |

### Reserved Slug Protection

View slugs cannot collide with system routes. These are protected:

```
admin, api, s, v, _app, _, assets, static,
favicon.ico, robots.txt, sitemap.xml,
health, healthz, ready, login, logout,
auth, oauth, callback, home, index, default, profile
```

**Enforcement layers:**
1. **Frontend param matcher**: `src/params/slug.ts` - invalid slugs don't route
2. **Backend hook**: Returns HTTP 400 when creating/updating views with reserved slugs

## Password-Protected Views

Password-protected views use signed JWTs for access control.

### Flow

```
1. Client: POST /api/view/{slug}/access
   Response: { "requires_password": true, "id": "..." }

2. Client: POST /api/password/check
   Body: { "view_id": "...", "password": "..." }
   Response: { "access_token": "<JWT>", "expires_in": 3600 }

3. Client: GET /api/view/{slug}/data
   Header: Authorization: Bearer <JWT>
   Response: { view data }
```

### JWT Structure

**Algorithm:** HS256

**Claims:**
| Claim | Description |
|-------|-------------|
| `vid` | View ID |
| `iss` | Issuer: `me.yaml` |
| `aud` | Audience: `view-access` |
| `iat` | Issued at timestamp |
| `exp` | Expiration timestamp |
| `jti` | Unique token ID (for audit) |

**Lifetime:** 1 hour

### Token Transport

Tokens can be sent via:
1. `Authorization: Bearer <token>` (preferred)
2. `X-Password-Token: <token>` (legacy/UI convenience)

### Security Properties

- Tokens are signed with HMAC-SHA256
- Signature validation is required
- Expiry is enforced
- Issuer and audience are validated
- View ID in token must match requested view
- Tokens cannot be used for a different view than issued

### Limitations

- **No revocation:** Tokens are valid until expiry
- **No refresh:** Client must re-authenticate after expiry
- **Stateless:** Server doesn't track issued tokens

## Share Tokens

Share tokens provide access to unlisted views. They are required for any `visibility=unlisted` view.

### Share Link Flow

The recommended way to share unlisted views is via `/s/<token>` URLs:

```
1. Admin: POST /api/share/generate (authenticated)
   Body: { "view_id": "...", "name": "For recruiters", "expires_at": null, "max_uses": 0 }
   Response: { "id": "...", "token": "<raw-token>", "name": "..." }
   ⚠️ Raw token is returned ONLY ONCE - store it securely

2. Admin shares URL: https://example.com/s/<token>

3. User visits /s/<token>:
   - Server validates token (POST /api/share/validate)
   - Sets httpOnly cookie (me_share_token, SameSite=Lax)
   - 302 redirect to /<slug> (canonical URL)
   - Token is NOT in the final URL

4. User's browser requests /<slug>:
   - SvelteKit reads token from cookie
   - Sends X-Share-Token header to backend
   - Backend validates and returns view data
```

This flow ensures:
- Token never appears in browser history
- Token never leaks via Referer headers
- Clean canonical URLs are displayed
- Cookie is httpOnly (no JavaScript access)

### Direct API Flow (for programmatic access)

```
1. GET /api/view/{slug}/access
   Response: { "requires_token": true, "id": "..." }

2. GET /api/view/{slug}/data
   Header: X-Share-Token: <raw-token>         (RECOMMENDED)
   -- or --
   Header: Authorization: Bearer <raw-token>  (alternative)
   Response: { view data }
```

### Storage Architecture

Share tokens use a two-part storage strategy for O(1) lookup:

1. **token_prefix** (first 12 chars): Stored in plaintext for indexed queries
2. **token_hash**: HMAC-SHA256 of the full token

**Lookup algorithm:**
```
1. Extract prefix from provided token (first 12 chars)
2. Query: SELECT * FROM share_tokens WHERE token_prefix = ? AND is_active = true
3. For each candidate (typically 1), verify full HMAC
4. Validate: view_id matches, not expired, under max_uses
```

This achieves O(1) database lookup instead of O(n) scanning, while maintaining security.

### Security Properties

- **HMAC storage:** Raw tokens never stored; DB leak doesn't reveal usable tokens
- **Constant-time comparison:** Prevents timing attacks on HMAC verification
- **Prefix is non-secret:** The 12-char prefix is a lookup optimization only; security relies entirely on HMAC verification of the full token and the underlying 256-bit token randomness
- **View-bound:** Each token is tied to a specific view ID
- **Expiry support:** Tokens can have optional expiration dates
- **Usage limits:** Tokens can have optional max usage counts
- **Revocation:** Admin can deactivate tokens at any time
- **Non-leaky errors:** All validation failures return the same generic error to prevent oracle attacks

### Token Properties

| Property | Description |
|----------|-------------|
| Length | 32 bytes, URL-safe base64 encoded (~43 chars) |
| Prefix | First 12 characters stored for indexed lookup |
| HMAC | Full token hashed with server's HMAC key |
| Expiry | Optional, enforced server-side |
| Max uses | Optional, 0 = unlimited |
| Use count | Tracked per-token |

### Token Transport

Tokens can be sent via:
1. `Authorization: Bearer <token>` — **RECOMMENDED** for API clients
2. `X-Share-Token: <token>` — Alternative header for programmatic access
3. `?token=<token>` — **LEGACY/COMPAT** for shareable links only

> ⚠️ **Security Warning:** Query parameter tokens (`?token=...`) are logged in server access logs, stored in browser history, and may leak via HTTP Referer headers. Use header-based transport whenever possible. Consider the query parameter method only for human-shareable links where header transport is impractical.

### Limitations

- **No token refresh:** Expired tokens require admin to generate new one
- **Prefix collision:** Rare but possible; mitigated by HMAC verification
- **No per-use logging:** Usage count tracked, but not individual accesses
- **URL token leakage:** Tokens in query strings may leak (see warning above)

## Collection Access Control

### Deny-by-Default Model

All PocketBase collections require authentication for direct access via `/api/collections/{name}/records`. This prevents bypassing visibility and draft rules.

**Public data flows through custom API endpoints:**
- `/api/view/{slug}/access` — Returns view metadata (visibility, requirements)
- `/api/view/{slug}/data` — Returns view content with visibility rules enforced

These endpoints use server-side database calls that bypass collection rules, allowing them to serve public content while maintaining access control.

### Collection Categories

| Category | Collections | Direct Access |
|----------|-------------|---------------|
| Content | profile, experience, projects, education, certifications, skills, posts, talks, views | Auth required |
| Sensitive | share_tokens, sources, ai_providers, import_proposals, settings | Auth required |
| Auth | users | Managed by PocketBase |

### Why This Matters

Without these restrictions, an attacker could:
1. Enumerate all records via `/api/collections/projects/records`
2. Access draft content (`is_draft=true`)
3. Access private content (`visibility=private`)
4. Bypass share token requirements for unlisted views

With deny-by-default:
1. Public visitors use `/api/view/{slug}/data` which enforces visibility
2. Only authenticated admins can access raw collection data
3. Visibility and draft filtering is guaranteed

### Security Boundary

**PocketBase collection API rules protect HTTP access only.** Internal application queries (via `app.FindRecordsByFilter()` and similar methods) run with server authority and bypass these rules by design.

Custom API endpoints (e.g., `/api/view/{slug}/data`) are responsible for enforcing visibility and draft rules in application code. This separation is intentional: collection rules block external enumeration, while application code handles business logic.

### Authenticated Access

Authenticated users (admin OAuth allowlist) can still:
- Use the admin dashboard to manage content
- Access collections directly via PocketBase API
- Use the `/_/` admin UI (if enabled)

## API Security

### Rate Limiting

Me.yaml implements per-IP rate limiting using the [token bucket algorithm](https://pkg.go.dev/golang.org/x/time/rate) to protect against brute force and abuse.

#### Rate Limit Tiers

| Tier | Limit | Burst | Endpoints |
|------|-------|-------|-----------|
| Strict | 5/min | 3 | `POST /api/password/check` |
| Moderate | 10/min | 5 | `POST /api/share/validate` |
| Normal | 60/min | 10 | `GET /api/view/{slug}/access`, `GET /api/view/{slug}/data` |

#### Response Headers

When rate limited, the server returns:
- **Status:** `429 Too Many Requests`
- **Headers:**
  - `Retry-After: <seconds>` — Time until next request allowed
  - `X-RateLimit-Limit: <rate>` — The rate limit for this endpoint
  - `X-RateLimit-Remaining: 0` — No requests remaining
- **Body:** `{"error": "too many requests"}` (uniform, non-leaky)

#### Configuration

**Environment Variables:**

| Variable | Default | Description |
|----------|---------|-------------|
| `TRUST_PROXY` | `false` | Set to `true` to trust proxy headers for client IP |

**Client IP Detection (in order of priority when `TRUST_PROXY=true`):**
1. `CF-Connecting-IP` — Cloudflare's original client IP header
2. `X-Real-IP` — Common proxy header (nginx, etc.)
3. `X-Forwarded-For` — Leftmost IP from comma-separated list
4. `RemoteAddr` — Direct connection IP (fallback)

**Security Warning:** Only set `TRUST_PROXY=true` if:
- Traffic arrives exclusively through a trusted proxy (Cloudflare, nginx, etc.)
- The proxy is configured to set/overwrite these headers
- Direct connections to the server are blocked

Without proper proxy configuration, attackers can spoof their IP address.

#### Cloudflare Setup

When using Cloudflare Tunnel or proxy:
1. Set `TRUST_PROXY=true`
2. Ensure [Cloudflare IP ranges](https://www.cloudflare.com/ips/) are the only allowed source IPs
3. The server will use `CF-Connecting-IP` for rate limiting

#### Limitations

- **In-memory storage:** Rate limit state does not persist across restarts
- **Single-instance:** Each server instance has independent rate limit state
- **No distributed coordination:** In multi-instance deployments, limits apply per-instance

For production at scale, consider implementing Redis-backed rate limiting (Step 6B).

#### Verification

```bash
# Test rate limiting on password endpoint (strict tier)
for i in {1..6}; do
  curl -s -o /dev/null -w "%{http_code}\n" \
    -X POST http://localhost:8090/api/password/check \
    -H "Content-Type: application/json" \
    -d '{"view_id":"test","password":"wrong"}'
done
# Expected: 400, 400, 400, 429, 429, 429 (first 3 allowed, then rate limited)

# Check Retry-After header
curl -s -I -X POST http://localhost:8090/api/password/check \
  -H "Content-Type: application/json" \
  -d '{"view_id":"test","password":"wrong"}' | grep -i retry-after
```

### CORS

No explicit CORS configuration - all endpoints are same-origin behind the Caddy reverse proxy.

### Security Headers

Me.yaml implements security headers in two phases. Phase 5A is deployed by default; Phase 5B requires manual configuration after testing.

#### Phase 5A: Deployed Headers

These headers are safe for all deployments and applied via `docker/Caddyfile`:

| Header | Value | Purpose |
|--------|-------|---------|
| `X-Content-Type-Options` | `nosniff` | Prevents MIME type sniffing attacks |
| `X-Frame-Options` | `DENY` | Prevents clickjacking by blocking iframes |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Limits referrer leakage on cross-origin requests |
| `Permissions-Policy` | `geolocation=(), microphone=()...` | Disables unnecessary browser APIs |
| `Server` | *(removed)* | Hides server software identity |

**Intentionally Omitted:**

| Header | Reason |
|--------|--------|
| `X-XSS-Protection` | Deprecated since 2023; can introduce vulnerabilities in modern browsers |
| `Strict-Transport-Security` | TLS terminates at edge proxy (Cloudflare), not at Caddy |

#### Phase 5B: Optional Stricter Headers

These require testing before deployment and may break functionality:

| Header | Recommendation |
|--------|----------------|
| `Content-Security-Policy` | Start with `Content-Security-Policy-Report-Only` to identify violations before enforcing. SvelteKit may require `'unsafe-inline'` for styles. |
| `Content-Security-Policy: frame-ancestors 'none'` | Supersedes X-Frame-Options; add when CSP is configured |
| `Strict-Transport-Security` | Only if Caddy terminates TLS directly (not behind proxy). Use `max-age=31536000; includeSubDomains` |
| `Cross-Origin-Opener-Policy` | May break OAuth popups; test thoroughly |
| `Cross-Origin-Embedder-Policy` | May break external image loading; test thoroughly |

**References:**
- [OWASP Secure Headers Project](https://owasp.org/www-project-secure-headers/)
- [OWASP HTTP Headers Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/HTTP_Headers_Cheat_Sheet.html)
- [MDN Content-Security-Policy](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Content-Security-Policy)

#### Cloudflare Configuration

If deploying behind Cloudflare Tunnel, configure these at Cloudflare instead of Caddy:

1. **HSTS:** SSL/TLS → Edge Certificates → Enable "Always Use HTTPS" and configure HSTS
2. **Security Headers:** Rules → Transform Rules → Modify Response Headers
3. **CSP:** Consider Cloudflare's CSP reporting if using their proxy

#### Verification

Test that security headers are applied (requires Docker deployment):

```bash
# Frontend routes
curl -sI http://localhost:8080/ | grep -E '^(X-Content-Type|X-Frame|Referrer|Permissions)'

# API endpoints
curl -sI http://localhost:8080/api/health | grep -E '^(X-Content-Type|X-Frame|Referrer|Permissions)'

# PocketBase admin (when ADMIN_ENABLED=true)
curl -sI http://localhost:8080/_/ | grep -E '^(X-Content-Type|X-Frame|Referrer|Permissions)'
```

Expected output for each:
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: geolocation=(), microphone=(), camera=(), payment=(), usb=()
```

Verify `Server` header is removed:
```bash
curl -sI http://localhost:8080/ | grep -i '^server:'
# Should return nothing (header removed)
```

## File Access

Files are served via PocketBase at `/api/files/{collectionId}/{recordId}/{filename}`.

**Current state:** Files follow the collection's access rules.

**Planned:** Signed URLs with expiration for private files.

## Development vs Production

| Aspect | Development | Production |
|--------|-------------|------------|
| Encryption key | Dev-only key in docker-compose.dev.yml | **Required** via `ENCRYPTION_KEY` |
| PocketBase Admin | Enabled at `/_/` | Disabled by default (`ADMIN_ENABLED=false`) |
| Seed data | Created automatically | Not created |
| Debug logging | Enabled | Disabled |

## Reporting Security Issues

If you discover a security vulnerability, please report it responsibly by opening a private issue or contacting the maintainers directly.

Do not open public issues for security vulnerabilities.
