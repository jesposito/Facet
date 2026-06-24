# Contact Protection & Social Links Feature

**Status:** Implemented (CAPTCHA/Turnstile tier planned, not yet shipped)

---

## Overview

Facet provides granular per-view control over contact information and social links with anti-scraping protection to prevent bot harvesting while maintaining accessibility.

Contact methods are stored in the `contact_methods` collection and managed from **Admin > Contacts** (`/admin/contacts`). Each method has a protection tier and per-view visibility.

### Protection Tiers

Four protection tiers are available per contact method:

1. **None** - Display the value directly (use for public profiles like GitHub).
2. **CSS Obfuscation** - Hide the value from basic scrapers using CSS tricks and decoy characters.
3. **Click-to-Reveal** - Require user interaction before revealing the value.
4. **CAPTCHA** - Cloudflare Turnstile verification. **Planned: this tier is defined but not yet shipped.** Treat the Turnstile sections below as the design for that work.

Per-view visibility lets you choose which contact methods appear on each view.

## Business Requirements

### User Stories

**As a portfolio owner, I want to:**
1. Display different contact methods on different views (professional vs personal)
2. Protect my email/phone from scrapers and AI bots
3. Make it easy for legitimate humans to contact me
4. Control which social links appear on each view
5. Have confidence my contact info won't end up in spam databases

**As a visitor, I want to:**
1. Easily find and use contact information
2. Not be frustrated by excessive CAPTCHAs or verification
3. Have contact info work with screen readers
4. Be able to copy/paste contact details

### Success Metrics

- **Security**: 90%+ reduction in scraping attempts
- **Conversion**: <5% drop in contact button clicks
- **Accessibility**: 100% WCAG 2.1 AA compliance
- **Performance**: <50ms overhead per page load

---

## Technical Design

### 1. Database Schema

**New Collection: `contact_methods`**

```go
// backend/migrations/1736000000_add_contact_methods.go

type ContactMethod struct {
    ID              string                 // PocketBase auto-generated
    UserID          string                 // FK to users
    Type            string                 // email, phone, linkedin, github, twitter, etc.
    Value           string                 // The actual contact info
    Label           string                 // Display label (optional)
    Icon            string                 // Icon name/URL (optional)
    ProtectionLevel string                 // none, obfuscation, click_to_reveal, captcha
    ViewVisibility  map[string]bool        // {"view_id": true/false}
    IsPrimary       bool                   // Primary contact for this type
    SortOrder       int                    // Display order
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

// Supported Types
const ContactTypes = [
    "email",
    "phone",
    "linkedin",
    "github",
    "twitter",
    "facebook",
    "instagram",
    "whatsapp",
    "telegram",
    "discord",
    "youtube",
    "twitch",
    "tiktok",
    "mastodon",
    "website",
    "other"
]

// Protection Levels
const ProtectionLevels = [
    "none",           // Display directly (for public profiles like GitHub)
    "obfuscation",    // CSS tricks + decoy characters
    "click_to_reveal", // Reveal on user interaction
    "captcha"         // Turnstile verification required
]
```

**Migration Plan:**
1. Create `contact_methods` collection
2. Migrate existing `profile.contact_email` → `contact_methods` (type=email, protection=click_to_reveal)
3. Migrate existing `profile.contact_links` → `contact_methods` (extract type from URL, protection=none)
4. Add indexes on `user_id`, `type`, `is_primary`

### 2. Protection Techniques

#### Level 1: CSS Obfuscation
**Use case:** LinkedIn, GitHub (semi-public anyway)
**Effectiveness:** ⭐⭐⭐⭐ Against basic scrapers
**Accessibility:** ✅ Fully accessible

```html
<a href="https://linkedin.com/in/username">
  linked<span style="display:none">REMOVE</span>in.com/in/<span style="display:none">SPAM</span>username
</a>
```

#### Level 2: Click-to-Reveal
**Use case:** Work email, phone numbers
**Effectiveness:** ⭐⭐⭐⭐⭐ Blocks automated bots
**Accessibility:** ✅ Keyboard accessible, screen reader friendly

```svelte
<!-- Before click -->
<button aria-label="Click to reveal email">
  📧 Click to reveal email
</button>

<!-- After click -->
<a href="mailto:contact@example.com">
  contact@example.com
</a>
<button aria-label="Copy email">Copy</button>
```

#### Level 3: Turnstile CAPTCHA (Planned, not yet shipped)
**Use case:** Primary email on high-traffic sites
**Effectiveness:** ⭐⭐⭐⭐⭐ Blocks all bots
**Accessibility:** ✅ Invisible challenge, fully accessible

```html
<div class="cf-turnstile"
     data-sitekey="YOUR_KEY"
     data-callback="revealContact"
     data-size="invisible">
</div>
```

### 3. Additional Security Layers

#### robots.txt - Block AI Crawlers

```txt
# Block AI training bots
User-agent: GPTBot
Disallow: /

User-agent: ClaudeBot
Disallow: /

User-agent: Google-Extended
Disallow: /

User-agent: CCBot
Disallow: /

User-agent: anthropic-ai
Disallow: /

User-agent: Bytespider
Disallow: /

# Allow legitimate search engines
User-agent: Googlebot
Allow: /
```

#### Rate Limiting

```go
// 10 contact reveals per 10 minutes per device fingerprint
var contactRevealLimiter = NewRateLimiter(10, 10*time.Minute)

// Track reveals for monitoring
type ContactReveal struct {
    ContactType string
    Fingerprint string
    IPAddress   string
    UserAgent   string
    Timestamp   time.Time
}
```

#### Honeypot Detection

```html
<!-- Hidden field that bots will fill -->
<input type="text"
       name="website"
       class="honeypot"
       tabindex="-1"
       aria-hidden="true"
       autocomplete="off">
```

---

## Implementation Plan

### Phase 1: Foundation (Week 1)

**Backend:**
- [ ] Create migration for `contact_methods` collection
- [ ] Create backend API endpoints:
  - `POST /api/contact-methods` - Create
  - `GET /api/contact-methods` - List (admin only)
  - `PATCH /api/contact-methods/:id` - Update
  - `DELETE /api/contact-methods/:id` - Delete
  - `GET /api/view/:id/contacts` - Get contacts for view (respects visibility)
- [ ] Implement rate limiting middleware
- [ ] Add contact reveal tracking

**Frontend:**
- [ ] Create base components:
  - `ObfuscatedLink.svelte` - CSS obfuscation
  - `ClickToReveal.svelte` - Click-to-reveal pattern
  - `ProtectedContact.svelte` - Wrapper component
- [ ] Update `ProfileHero.svelte` to use new components
- [ ] Add `robots.txt` with AI bot blocking

**Testing:**
- [ ] Test CSS obfuscation with scraper tools
- [ ] Test keyboard accessibility
- [ ] Test screen reader compatibility

### Phase 2: Admin UI (Week 2)

**Features:**
- [ ] Contact methods management page (`/admin/contacts`)
- [ ] Add/edit/delete contact methods
- [ ] Per-view visibility toggles
- [ ] Protection level selector
- [ ] Drag-and-drop reordering
- [ ] Preview of how contacts appear on each view
- [ ] Bulk import from existing `contact_links`

**Components:**
- [ ] `ContactMethodEditor.svelte` - Edit single contact
- [ ] `ContactMethodList.svelte` - List with drag-drop
- [ ] `ViewVisibilityMatrix.svelte` - Grid showing which contacts appear on which views
- [ ] `ProtectionLevelPicker.svelte` - Choose protection strategy

### Phase 3: Advanced Protection (Week 3)

**Features:**
- [ ] Cloudflare Turnstile integration
- [ ] Device fingerprinting for rate limiting
- [ ] Canary tokens for scraping detection
- [ ] Analytics dashboard showing:
  - Contact reveal attempts
  - Bot detection events
  - Conversion rates
  - Honeypot triggers

**Optional Enhancements:**
- [ ] A/B test different protection levels
- [ ] Auto-adjust protection based on traffic patterns
- [ ] Email alias generation (contact+view123@domain.com)
- [ ] Temporary contact links (expire after 7 days)

---

## UI/UX Design

### Admin: Contact Management

```
┌─────────────────────────────────────────────────────┐
│ Contact Methods                          + Add New  │
├─────────────────────────────────────────────────────┤
│                                                     │
│ ⚡ Primary Email                                    │
│ contact@example.com                                 │
│ Protection: Click to Reveal                         │
│ Visible on: Professional, Personal (2/5 views)     │
│ [Edit] [Delete]                                     │
│                                                     │
│ 💼 LinkedIn                                         │
│ linkedin.com/in/username                            │
│ Protection: CSS Obfuscation                         │
│ Visible on: All views                              │
│ [Edit] [Delete]                                     │
│                                                     │
│ 📱 Phone                                            │
│ +1 (555) 123-4567                                  │
│ Protection: Click to Reveal + CAPTCHA              │
│ Visible on: Personal only (1/5 views)              │
│ [Edit] [Delete]                                     │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### Public: Contact Display

**Professional View:**
```
┌─────────────────────────────────────────┐
│ Contact Me                              │
├─────────────────────────────────────────┤
│                                         │
│ [📧 Click to reveal email]              │
│                                         │
│ 💼 LinkedIn                             │
│ linkedin.com/in/username                │
│                                         │
│ 🐙 GitHub                               │
│ github.com/username                     │
│                                         │
└─────────────────────────────────────────┘
```

**Personal View:**
```
┌─────────────────────────────────────────┐
│ Get in Touch                            │
├─────────────────────────────────────────┤
│                                         │
│ [📧 Click to reveal email]              │
│                                         │
│ [📱 Click to reveal phone]              │
│                                         │
│ 💬 WhatsApp                             │
│ wa.me/1555123456                        │
│                                         │
│ 📘 Facebook                             │
│ facebook.com/username                   │
│                                         │
└─────────────────────────────────────────┘
```

---

## API Endpoints

### GET `/api/view/:slug/contacts`

**Description:** Get contacts visible on a specific view

**Response:**
```json
{
  "contacts": [
    {
      "id": "abc123",
      "type": "email",
      "label": "Work Email",
      "protection_level": "click_to_reveal",
      "is_primary": true,
      "sort_order": 1
      // Note: `value` is NOT returned for click_to_reveal/captcha
    },
    {
      "id": "def456",
      "type": "linkedin",
      "value": "https://linkedin.com/in/username",
      "label": "LinkedIn Profile",
      "protection_level": "obfuscation",
      "is_primary": false,
      "sort_order": 2
    }
  ]
}
```

### POST `/api/contact-methods/:id/reveal`

**Description:** Reveal a protected contact (after user interaction)

**Request:**
```json
{
  "fingerprint": "abc123def456",
  "turnstile_token": "optional-if-captcha-protected"
}
```

**Response:**
```json
{
  "value": "contact@example.com",
  "type": "email",
  "label": "Work Email"
}
```

**Rate Limiting:** 10 requests per 10 minutes per fingerprint

---

## Accessibility Compliance

### WCAG 2.1 AA Requirements

**✅ Keyboard Accessible**
- All contact reveal buttons support Tab, Enter, Space
- Focus indicators visible (2px outline)
- Logical tab order

**✅ Screen Reader Compatible**
- Proper ARIA labels: `aria-label="Click to reveal email address"`
- State changes announced: `aria-live="polite"`
- Hidden decorative content: `aria-hidden="true"` on decoy spans

**✅ Color Contrast**
- Contact buttons: 4.5:1 minimum
- Links: 4.5:1 minimum
- Focus indicators: 3:1 minimum

**✅ Responsive**
- Touch targets ≥44×44px
- Works on mobile devices
- Text scales to 200%

### Testing Checklist

- [ ] Test with NVDA screen reader (Windows)
- [ ] Test with JAWS screen reader (Windows)
- [ ] Test with VoiceOver (macOS/iOS)
- [ ] Keyboard-only navigation test
- [ ] Color contrast audit (Lighthouse)
- [ ] axe DevTools audit (0 violations)

---

## Security Considerations

### Threat Model

**Threats:**
1. **Email scraping bots** - Harvest emails for spam
2. **AI training crawlers** - Ingest data for LLM training
3. **Mass scraping tools** - Collect contacts at scale
4. **Manual copying** - Humans copying data (acceptable)

**Non-Threats:**
- Determined attackers with browser automation (acceptable risk)
- Government agencies with legal authority (out of scope)

### Defense in Depth

**Layer 1: robots.txt**
- Blocks polite bots (GPTBot, ClaudeBot, etc.)
- Easy to implement, low overhead
- Doesn't stop malicious bots

**Layer 2: CSS Obfuscation**
- Stops basic scrapers that don't apply CSS
- No UX impact
- Sophisticated scrapers can bypass

**Layer 3: Click-to-Reveal**
- Requires JavaScript execution + user interaction
- Blocks automated bots
- Small UX friction (acceptable)

**Layer 4: Rate Limiting**
- Prevents mass scraping
- Based on device fingerprint + IP
- Doesn't affect normal users

**Layer 5: Turnstile (Optional, planned)**
- Strongest protection
- Cloudflare's invisible CAPTCHA
- Use sparingly (high-value contacts only)
- Not yet shipped; planned enhancement

### Privacy Considerations

**GDPR Compliance:**
- ✅ It's YOUR contact info (not user data)
- ✅ Legitimate interest basis (people need to contact you)
- ✅ No consent banner required
- ⚠️ If collecting visitor emails → need consent

**Data Minimization:**
- Only store necessary contact info
- Don't log visitor data excessively
- Auto-delete old reveal logs (>90 days)

---

## Testing Strategy

### Manual Testing

**Scraping Test:**
```bash
# Test if email is scrapable
curl https://yoursite.com/v/professional | grep -oE '\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b'

# Should return nothing for protected emails
```

**Rate Limiting Test:**
```bash
# Attempt 20 reveals (should block after 10)
for i in {1..20}; do
  curl -X POST https://yoursite.com/api/contact-methods/abc123/reveal \
    -H "Content-Type: application/json" \
    -d '{"fingerprint":"test123"}'
done
```

### Automated Testing

```typescript
// frontend/tests/contact-protection.spec.ts
test('email is obfuscated in HTML', async ({ page }) => {
  await page.goto('/v/professional');

  const html = await page.content();
  expect(html).not.toContain('contact@example.com');
  expect(html).toContain('Click to reveal');
});

test('click-to-reveal works with keyboard', async ({ page }) => {
  await page.goto('/v/professional');

  await page.keyboard.press('Tab'); // Focus reveal button
  await page.keyboard.press('Enter'); // Activate

  await expect(page.locator('a[href^="mailto:"]')).toBeVisible();
});

test('rate limiting blocks excessive requests', async ({ request }) => {
  // Make 11 requests (limit is 10)
  for (let i = 0; i < 11; i++) {
    const res = await request.post('/api/contact-methods/abc123/reveal', {
      data: { fingerprint: 'test' }
    });

    if (i < 10) {
      expect(res.ok()).toBeTruthy();
    } else {
      expect(res.status()).toBe(429); // Too Many Requests
    }
  }
});
```

---

## Performance Benchmarks

**Targets:**
- Page load: <50ms overhead for contact protection
- Click-to-reveal: <100ms to reveal
- Fingerprinting: <10ms to generate
- Rate limit check: <5ms

**Monitoring:**
```typescript
// Track performance
const start = performance.now();
await revealContact();
const duration = performance.now() - start;

analytics.track('contact_reveal_performance', {
  duration_ms: duration,
  protection_level: 'click_to_reveal'
});
```

---

## Documentation

### User Guide

**For Portfolio Owners:**
- How to add contact methods
- Choosing protection levels
- Setting per-view visibility
- Best practices for balancing security & conversion

**For Developers:**
- API reference
- Component documentation
- Security implementation guide
- Testing guide

### Admin Help Text

```
Protection Levels:

• None: Display directly (use for public profiles)
• CSS Obfuscation: Hide from basic scrapers (good for LinkedIn)
• Click to Reveal: Require user interaction (best for email/phone)
• CAPTCHA: Maximum protection (use sparingly - impacts UX)

Tip: Start with "Click to Reveal" for emails and "None" for social profiles.
```

---

## Future Enhancements

**Phase 4: Advanced Features**
- [ ] Machine learning bot detection
- [ ] Contact form builder (alternative to direct display)
- [ ] Temporary contact links with expiration
- [ ] Email alias generation per view
- [ ] WebAuthn/Passkey verification for premium contacts
- [ ] Analytics dashboard for contact engagement
- [ ] A/B testing framework for protection levels

**Phase 5: Integrations**
- [ ] SendGrid integration for contact forms
- [ ] Cloudflare Bot Management
- [ ] Google reCAPTCHA v3 (alternative to Turnstile)
- [ ] Plausible/Fathom analytics integration

---

## Open Questions

1. Should we allow custom contact types beyond the predefined list?
2. Do we want a global "protection level override" for paranoid users?
3. Should we add a "trust score" that auto-adjusts protection based on visitor behavior?
4. Do we want to support QR codes for contact info (useful for print versions)?
5. Should there be a fallback contact form when all direct contacts are hidden?

---

## References

- [WCAG 2.1 Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)
- [Cloudflare Turnstile Docs](https://developers.cloudflare.com/turnstile/)
- [GDPR Compliance](https://gdpr.eu/)
- [AI Bot User Agents](https://darkvisitors.com/agents)
- [Email Obfuscation Techniques](https://www.emailtooltester.com/en/email-obfuscator-tool/)

---

**Last Updated:** 2026-06-23
**Status:** Implemented — `contact_methods` collection, per-view visibility, and the None / CSS Obfuscation / Click-to-Reveal tiers ship today. The CAPTCHA (Cloudflare Turnstile) tier is planned and not yet shipped; sections describing Turnstile describe the intended design.
