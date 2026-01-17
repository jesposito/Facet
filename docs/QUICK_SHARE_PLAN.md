# Quick Share to Social - Phase 18.1 Design

**Status:** Planning  
**Priority:** High  
**Effort:** Medium  

## Research Summary

Based on deep research into Web Share API, social platform URLs, and UX best practices.

---

## Core Principle: Progressive Enhancement

```
User clicks Share
       │
       ▼
┌──────────────────┐
│ Web Share API    │  ← 75% of users (great on mobile)
│ available?       │
└────────┬─────────┘
         │ Yes                    │ No
         ▼                        ▼
┌──────────────────┐    ┌──────────────────┐
│ Native OS share  │    │ Custom dropdown  │
│ sheet opens      │    │ with platforms   │
└──────────────────┘    └──────────────────┘
```

**Why this matters:**
- Native share on mobile = best UX (user's preferred apps)
- Desktop fallback = Copy Link + key platforms
- No third-party SDKs = privacy, performance, no tracking

---

## Implementation Architecture

### 1. ShareButton Component

```svelte
<!-- ShareButton.svelte -->
<script lang="ts">
  interface Props {
    url: string;
    title: string;
    text?: string;
    class?: string;
  }
</script>
```

**Features:**
- Single button that adapts to environment
- Shows native share icon on supported browsers
- Shows dropdown on unsupported browsers
- Always includes "Copy Link" option

### 2. Placement Locations

| Location | When to Show | UI Treatment |
|----------|--------------|--------------|
| **View pages** (`/[slug]`) | Always | Inline with print button (top right) |
| **Post detail** (`/posts/[slug]`) | Always | Below post title |
| **Project detail** (`/projects/[slug]`) | Always | Below project title |
| **Talk detail** (`/talks/[slug]`) | Always | Below talk title |
| **Admin view editor** | When saved | Header actions area |

### 3. Platform Selection (Fallback Mode)

**Primary (always visible):**
- **Copy Link** - Universal, fast, privacy-respecting

**Secondary (3 platforms max):**
- **LinkedIn** - Key audience for professional profiles
- **Twitter/X** - Tech community, quick sharing
- **Email** - Universal fallback

**Why NOT Facebook/WhatsApp/etc:**
- Facebook sharer broken on iOS Safari since Oct 2025
- WhatsApp requires phone number or opens picker (mobile-only UX)
- Telegram is niche
- Less is more - cognitive load from too many options

---

## Platform URL Formats (2025)

### LinkedIn
```
https://www.linkedin.com/sharing/share-offsite/?url={encodedUrl}
```
- Only `url` param works (title/summary deprecated)
- Relies on Open Graph meta tags for preview
- Facet already has OG tags ✓

### Twitter/X
```
https://twitter.com/intent/tweet?text={encodedText}&url={encodedUrl}
```
- `text` - Pre-filled tweet text
- `url` - URL to share (auto-shortened)
- 280 char limit total

### Email
```
mailto:?subject={encodedSubject}&body={encodedBody}
```
- Opens default mail client
- `%0A` for line breaks in body
- Universal fallback

---

## Component Design

### Public Page Share (Views, Posts, Projects, Talks)

```
┌─────────────────────────────────────┐
│  [Print ▾] [Share ↗]  [Theme ◐]    │  ← Top right, inline
└─────────────────────────────────────┘
```

**On click (Web Share supported):**
- Native share sheet opens
- User picks destination

**On click (fallback):**
```
┌─────────────────────────────┐
│  Share                      │
├─────────────────────────────┤
│  🔗 Copy Link              │  ← Primary, always first
│  ─────────────────────────  │
│  in LinkedIn               │
│  𝕏 Twitter/X               │
│  ✉️ Email                   │
└─────────────────────────────┘
```

### Copy Link Feedback

```
Before:  [🔗 Copy Link]
During:  [Copying...]
After:   [✓ Copied!]  ← 2 seconds, then revert
```

---

## What Gets Shared

### View Pages
```javascript
{
  url: 'https://facet.example.com/recruiter',
  title: 'John Doe - Recruiter View',
  text: 'Check out my professional profile'
}
```

### Blog Posts
```javascript
{
  url: 'https://facet.example.com/posts/my-article',
  title: 'My Article Title',
  text: 'My Article Title by John Doe'
}
```

### Projects
```javascript
{
  url: 'https://facet.example.com/projects/my-project',
  title: 'My Project - John Doe',
  text: 'Check out my project: My Project'
}
```

### Talks
```javascript
{
  url: 'https://facet.example.com/talks/my-talk',
  title: 'My Talk - John Doe',
  text: 'Watch my talk: My Talk'
}
```

---

## Technical Implementation

### 1. Share Utility (`frontend/src/lib/share.ts`)

```typescript
interface ShareData {
  url: string;
  title: string;
  text?: string;
}

interface ShareResult {
  method: 'native' | 'clipboard' | 'external';
  success: boolean;
  cancelled?: boolean;
}

// Check if Web Share API is available
export function canUseNativeShare(): boolean {
  return (
    typeof navigator !== 'undefined' &&
    'share' in navigator &&
    typeof window !== 'undefined' &&
    window.isSecureContext
  );
}

// Native share (Web Share API)
export async function nativeShare(data: ShareData): Promise<ShareResult> {
  if (!canUseNativeShare()) {
    return { method: 'native', success: false };
  }
  
  try {
    await navigator.share({
      url: data.url,
      title: data.title,
      text: data.text
    });
    return { method: 'native', success: true };
  } catch (error: any) {
    if (error.name === 'AbortError') {
      return { method: 'native', success: false, cancelled: true };
    }
    throw error;
  }
}

// Copy to clipboard with fallback
export async function copyToClipboard(text: string): Promise<boolean> {
  if (navigator.clipboard) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {}
  }
  
  // Fallback for older browsers
  const textarea = document.createElement('textarea');
  textarea.value = text;
  textarea.style.position = 'fixed';
  textarea.style.left = '-999999px';
  document.body.appendChild(textarea);
  textarea.select();
  try {
    document.execCommand('copy');
    return true;
  } catch {
    return false;
  } finally {
    document.body.removeChild(textarea);
  }
}

// Generate platform share URLs
export function getShareUrls(data: ShareData) {
  const encodedUrl = encodeURIComponent(data.url);
  const encodedTitle = encodeURIComponent(data.title);
  const encodedText = encodeURIComponent(data.text || data.title);
  
  return {
    linkedin: `https://www.linkedin.com/sharing/share-offsite/?url=${encodedUrl}`,
    twitter: `https://twitter.com/intent/tweet?text=${encodedText}&url=${encodedUrl}`,
    email: `mailto:?subject=${encodedTitle}&body=${encodeURIComponent((data.text || '') + '\n\n' + data.url)}`
  };
}
```

### 2. ShareButton Component (`frontend/src/components/shared/ShareButton.svelte`)

Key features:
- Uses Web Share API when available
- Falls back to dropdown with Copy Link + platforms
- 44px touch targets on mobile
- Keyboard accessible (Tab, Enter, Space, Esc)
- ARIA labels for screen readers
- Success feedback on copy

### 3. Files to Create/Modify

**New files:**
- `frontend/src/lib/share.ts` - Share utility functions
- `frontend/src/components/shared/ShareButton.svelte` - Reusable component

**Modified files:**
- `frontend/src/routes/[slug=slug]/+page.svelte` - Add to view pages
- `frontend/src/routes/posts/[slug]/+page.svelte` - Add to post pages
- `frontend/src/routes/projects/[slug]/+page.svelte` - Add to project pages
- `frontend/src/routes/talks/[slug]/+page.svelte` - Add to talk pages

---

## Accessibility Requirements

1. **Keyboard Navigation:**
   - Tab to focus
   - Enter/Space to activate
   - Esc to close dropdown
   - Arrow keys to navigate dropdown

2. **ARIA Attributes:**
   ```html
   <button
     aria-label="Share this page"
     aria-haspopup="true"
     aria-expanded={isOpen}
   >
   ```

3. **Screen Reader Announcements:**
   - "Share button"
   - "Link copied to clipboard" (on copy)
   - Platform names in dropdown

4. **Focus Management:**
   - Return focus to button after dropdown closes
   - Trap focus within dropdown when open

---

## Mobile Considerations

1. **Touch Targets:** 44px minimum (Apple/Google guidelines)
2. **Web Share API:** Primary path on mobile (native experience)
3. **Bottom Sheet:** If dropdown needed, use bottom sheet pattern on mobile
4. **Thumb Zone:** Button placement in easy reach

---

## Analytics (Optional, Privacy-First)

If tracking is desired (opt-in via settings):

```javascript
// Track share events locally
function trackShare(method: string, contentType: string) {
  // Only if analytics enabled in settings
  if (!analyticsEnabled) return;
  
  // Local tracking only, no external services
  fetch('/api/analytics/share', {
    method: 'POST',
    body: JSON.stringify({ method, contentType, timestamp: Date.now() })
  });
}
```

Track:
- Share method (native, linkedin, twitter, email, copy)
- Content type (view, post, project, talk)
- Success/cancel

---

## Security

1. **External links:** `target="_blank" rel="noopener noreferrer"`
2. **No third-party SDKs:** Only intent URLs, no Facebook/LinkedIn SDKs
3. **No tracking pixels:** Privacy-respecting implementation
4. **HTTPS required:** Web Share API only works on secure contexts (Facet is already HTTPS)

---

## Testing Checklist

- [ ] Web Share API works on iOS Safari
- [ ] Web Share API works on Android Chrome
- [ ] Fallback dropdown works on Firefox Desktop
- [ ] Copy Link works and shows feedback
- [ ] LinkedIn share opens with correct URL
- [ ] Twitter share opens with text and URL
- [ ] Email opens default mail client
- [ ] Keyboard navigation works
- [ ] Screen reader announces correctly
- [ ] Touch targets are 44px+ on mobile
- [ ] Dropdown closes on outside click
- [ ] Dropdown closes on Esc key

---

## Implementation Order

1. **Create share utility** (`lib/share.ts`)
2. **Create ShareButton component**
3. **Add to view pages** (highest value)
4. **Add to post/project/talk pages**
5. **Test across browsers/devices**
6. **Polish animations and feedback**

---

## What Makes This "Really Awesome"

1. **Native first:** Mobile users get their familiar share sheet
2. **Zero friction:** One tap to share or copy
3. **Privacy respecting:** No third-party tracking
4. **Professional focus:** LinkedIn + Twitter for target audience
5. **Universal fallback:** Copy Link always works
6. **Great feedback:** Clear visual confirmation
7. **Accessible:** Works for everyone
8. **Fast:** No external SDK loading

---

## Future Enhancements (Not in Scope)

- QR code generation (Phase 18.3)
- Share analytics dashboard
- Custom share messages per view
- Schedule shares (social media posting)
