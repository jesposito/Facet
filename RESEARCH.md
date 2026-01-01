# Facet Research Document

> **Note**: This research was conducted when the project was named "OwnProfile" (later "Me.yaml", now "Facet"). The technical decisions remain relevant.

## Research Summary (Phase 0)

### PocketBase Evaluation

**Sources:**
- [PocketBase Official Docs](https://pocketbase.io/docs/)
- [PocketBase Authentication](https://pocketbase.io/docs/authentication/)
- [PocketBase Go Event Hooks](https://pocketbase.io/docs/go-event-hooks/)
- [Going to Production](https://pocketbase.io/docs/going-to-production/)

**Key Findings:**
1. **OAuth Support**: 30+ OAuth2 providers including Google and GitHub
2. **File Storage**: Local or S3-compatible, files in `pb_data` directory
3. **Event Hooks**: Full Go extension support with hooks for auth, records, routes
4. **Admin UI**: Built-in at `/_/` path, fully functional CRUD
5. **Database**: Embedded SQLite - perfect for single-user self-hosted
6. **Status**: Not v1.0 yet, but widely used in production

**Trade-offs:**
- ✅ Single binary, extremely easy deployment
- ✅ Built-in auth, admin, file storage
- ✅ Go extensibility for custom business logic
- ⚠️ SQLite limits horizontal scaling (not needed for single profile)
- ⚠️ Not v1.0 (acceptable risk for this use case)

### GitHub API Evaluation

**Sources:**
- [GitHub REST API - Repositories](https://docs.github.com/en/rest/repos/repos)
- [GitHub REST API - Contents](https://docs.github.com/en/rest/repos/contents)
- [Rate Limits](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api)

**Key Endpoints:**
| Endpoint | Purpose |
|----------|---------|
| `GET /repos/{owner}/{repo}` | Repo metadata (name, description, stars, forks) |
| `GET /repos/{owner}/{repo}/readme` | README content (base64 or raw) |
| `GET /repos/{owner}/{repo}/languages` | Language breakdown (bytes per language) |
| `GET /repos/{owner}/{repo}/topics` | Topic tags |

**Rate Limits:**
- Unauthenticated: 60 requests/hour
- Authenticated (PAT): 5,000 requests/hour
- Headers: `X-RateLimit-Remaining`, `X-RateLimit-Reset`

### Frontend Framework Comparison

**Sources:**
- [2025 Frontend Framework Showdown](https://leapcell.io/blog/the-2025-frontend-framework-showdown-next-js-nuxt-js-sveltekit-and-astro)
- [SvelteKit vs Next.js 2025](https://prismic.io/blog/sveltekit-vs-nextjs)

| Framework | Performance | Bundle Size | SSR/SSG | Best For |
|-----------|-------------|-------------|---------|----------|
| **Astro** | Excellent | Minimal | SSG-first | Content sites |
| **SvelteKit** | Excellent | Small | Both | Interactive apps |
| **Next.js** | Good | Larger | Both | Enterprise apps |

**Decision**: SvelteKit - compiles to minimal JS, great DX, works excellently with PocketBase, handles both public (fast) and admin (interactive) well.

### Token Encryption Best Practices

**Sources:**
- [Go Security Guide](https://dev.to/kingyou/go-security-guide-tokens-sha1-rsa-aes-encryption-4hcc)
- [API Key Management Best Practices](https://multitaskai.com/blog/api-key-management-best-practices/)

**Recommendations:**
1. Use AES-256-GCM for symmetric encryption
2. Use unique random nonces per encryption operation
3. Store encryption key in environment variable
4. Never expose decrypted tokens to browser
5. Implement envelope encryption if possible

### Share Token / Unlisted Resource Best Practices

**Sources:**
- [OWASP Session Management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [Token Best Practices - Auth0](https://auth0.com/docs/secure/tokens/token-best-practices)

**Recommendations:**
1. Use cryptographically random tokens (32+ bytes)
2. Store hashed version, compare with timing-safe function
3. Include expiration timestamps
4. Log access for audit trail
5. Allow token revocation

---

## Architecture Decision

### Selected Stack

```
┌─────────────────────────────────────────────────────────────────┐
│                           Facet                                  │
├─────────────────────────────────────────────────────────────────┤
│  Frontend: SvelteKit                                            │
│  ├── Public Profile Pages (SSR, fast, SEO-friendly)             │
│  ├── Admin Dashboard (SPA-like, interactive)                    │
│  └── Review/Diff UI (rich interactions)                         │
├─────────────────────────────────────────────────────────────────┤
│  Backend: PocketBase (Extended with Go)                         │
│  ├── Collections (profile, experience, projects, views, etc.)   │
│  ├── OAuth (Google, GitHub)                                     │
│  ├── File Storage (local, mountable volume)                     │
│  ├── Custom Hooks:                                              │
│  │   ├── GitHub Importer API                                    │
│  │   ├── AI Provider Integration                                │
│  │   ├── Token Encryption/Decryption                            │
│  │   ├── Share Token Validation                                 │
│  │   └── Password Protection                                    │
│  └── Admin API Extensions                                       │
├─────────────────────────────────────────────────────────────────┤
│  Database: SQLite (embedded, backed up with pb_data volume)     │
├─────────────────────────────────────────────────────────────────┤
│  Container: Docker                                              │
│  ├── Single container (PocketBase + SvelteKit in one)           │
│  ├── Volumes: /data (pb_data), /uploads                         │
│  └── Reverse proxy ready (X-Forwarded-* headers)                │
└─────────────────────────────────────────────────────────────────┘
```

### Why This Stack?

1. **PocketBase + Go Extensions**
   - Single binary backend
   - Built-in OAuth for Google/GitHub admin login
   - Built-in file storage for media uploads
   - SQLite is perfect for single-profile use case
   - Go hooks for custom logic (GitHub import, AI, encryption)

2. **SvelteKit Frontend**
   - Compiles to minimal JS (fast public pages)
   - Full SSR support (SEO-friendly)
   - Single framework for public + admin (maintainability)
   - Excellent DX with reactive syntax

3. **Single Container Design**
   - Simplest for Unraid users
   - One port to expose
   - One volume to backup
   - Easy reverse proxy setup

### Trade-offs Accepted

| Decision | Trade-off | Mitigation |
|----------|-----------|------------|
| SQLite | No horizontal scaling | Single-user app doesn't need it |
| Single container | Can't scale frontend separately | Not needed for personal site |
| PocketBase pre-v1.0 | Breaking changes possible | Pin version, document upgrade path |
| SvelteKit over Astro | Slightly larger bundle | Svelte bundles are still very small |
