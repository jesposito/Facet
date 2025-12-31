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
| `unlisted` | Anyone with the link (share tokens planned) |
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

Share tokens provide access to unlisted views.

### Storage

Share tokens are stored as HMAC-SHA256 hashes:
- Raw token is returned to user once
- Server stores only the HMAC
- Constant-time comparison prevents timing attacks

### Properties

- Cryptographically random (32 bytes)
- Optional expiration date
- Optional usage limit
- Can be revoked by admin

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
