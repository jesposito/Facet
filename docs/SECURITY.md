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

| Visibility | Access Control |
|------------|----------------|
| `public` | Anyone |
| `unlisted` | Valid share token required |
| `password` | Anyone with the correct password → receives JWT |
| `private` | Admin only |

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

### Flow

```
1. Admin: POST /api/share/generate (authenticated)
   Body: { "view_id": "...", "name": "For recruiters", "expires_at": null, "max_uses": 0 }
   Response: { "id": "...", "token": "<raw-token>", "name": "..." }
   ⚠️ Raw token is returned ONLY ONCE - store it securely

2. User: GET /api/view/{slug}/access
   Response: { "requires_token": true, "id": "..." }

3. User: GET /api/view/{slug}/data?token=<raw-token>
   or: Header: Authorization: Bearer <raw-token>
   or: Header: X-Share-Token: <raw-token>
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

- **HMAC storage:** Raw tokens never stored; DB leak doesn't reveal tokens
- **Constant-time comparison:** Prevents timing attacks on HMAC verification
- **Prefix indexing:** Only reveals 12 chars (~72 bits) of token structure
- **View-bound:** Each token is tied to a specific view ID
- **Expiry support:** Tokens can have optional expiration dates
- **Usage limits:** Tokens can have optional max usage counts
- **Revocation:** Admin can deactivate tokens at any time

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
1. `Authorization: Bearer <token>` (preferred)
2. `X-Share-Token: <token>` (header alternative)
3. `?token=<token>` (query parameter for link sharing)

### Limitations

- **No token refresh:** Expired tokens require admin to generate new one
- **Prefix collision:** Rare but possible; mitigated by HMAC verification
- **No per-use logging:** Usage count tracked, but not individual accesses

## API Security

### Rate Limiting

*(Planned - not yet implemented)*

### CORS

No explicit CORS configuration - all endpoints are same-origin behind the Caddy reverse proxy.

### Security Headers

*(Planned - to be added via Caddyfile)*

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
