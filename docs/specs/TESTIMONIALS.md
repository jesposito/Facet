# Technical Specification: Testimonials Feature

## Overview
The Testimonials feature allows Facet users to collect, manage, and display professional endorsements on their profile. It includes a frictionless public submission flow, verification options (Email, GitHub, Twitter), an admin review workflow, and integration with the Views system for curated display.

## Goals
1. **Low Friction**: Easy for people to leave testimonials without creating an account.
2. **Verification**: Multiple ways to prove authenticity (Email magic link, OAuth).
3. **Admin Control**: Comprehensive approve/reject workflow with editing capabilities.
4. **View Integration**: Curate specific testimonials for specific audience views.
5. **Premium Display**: Beautiful, responsive layouts (Wall of Love, Carousel, Featured).

---

## Data Model

### Table: `testimonials`
Stores the actual testimonial content and metadata.
- `id`: `string` (PocketBase ID)
- `profile_id`: `string` (FK to profile)
- **Content**
  - `content`: `string` (the testimonial text, supports markdown)
  - `relationship`: `string` (enum: `client`, `colleague`, `manager`, `report`, `mentor`, `other`)
  - `project`: `string` (optional - what they worked on together)
- **Author Info**
  - `author_name`: `string`
  - `author_title`: `string` (optional, e.g., "CTO at TechCorp")
  - `author_company`: `string` (optional)
  - `author_photo`: `file` (optional - uploaded or from OAuth)
  - `author_website`: `string` (optional)
- **Verification**
  - `verification_method`: `string` (enum: `none`, `email`, `github`, `twitter`, `linkedin`)
  - `verification_identifier`: `string` (email address, username, or profile URL)
  - `verification_data`: `json` (store OAuth profile data for display)
  - `verified_at`: `datetime`
- **Workflow**
  - `status`: `string` (enum: `pending`, `approved`, `rejected`)
  - `request_id`: `string` (FK to `testimonial_requests`, optional)
  - `submitted_at`: `datetime`
  - `approved_at`: `datetime`
  - `rejected_at`: `datetime`
  - `rejection_reason`: `string` (optional, internal note)
- **Display**
  - `featured`: `boolean` (default: `false`)
  - `sort_order`: `int` (for manual ordering)
- **Metadata**
  - `created`: `datetime`
  - `updated`: `datetime`

### Table: `testimonial_requests`
Manages the invitation links sent to recipients. Follows the `share_tokens` security model.
- `id`: `string` (PocketBase ID)
- `profile_id`: `string` (FK to profile)
- **Security**
  - `token_hash`: `string` (HMAC-SHA256 of full token)
  - `token_prefix`: `string` (first 12 chars for O(1) lookup, indexed)
- **Customization**
  - `label`: `string` (internal label, e.g., "Project X Clients")
  - `custom_message`: `string` (optional - shown on submission form)
  - `recipient_name`: `string` (optional - pre-fill)
  - `recipient_email`: `string` (optional)
- **Constraints**
  - `expires_at`: `datetime` (optional)
  - `max_uses`: `int` (optional, null = unlimited)
  - `use_count`: `int` (default: 0)
- **State**
  - `is_active`: `boolean` (default: `true`)
  - `created`: `datetime`
  - `updated`: `datetime`

### Table: `email_verification_tokens`
Ephemeral tokens for the magic link flow.
- `id`: `string`
- `testimonial_id`: `string` (FK to `testimonials`)
- `email`: `string`
- `token_hash`: `string`
- `expires_at`: `datetime` (15 minutes)
- `verified_at`: `datetime`
- `created`: `datetime`

---

## API Endpoints

### Admin APIs (Auth Required)
- `POST   /api/testimonials/requests` - Create new request link
- `GET    /api/testimonials/requests` - List all request links
- `PATCH  /api/testimonials/requests/:id` - Update request
- `DELETE /api/testimonials/requests/:id` - Delete request link
- `GET    /api/testimonials` - List all testimonials (with status filters)
- `PATCH  /api/testimonials/:id` - Update testimonial (status, featured, content)
- `POST   /api/testimonials/:id/approve` - Approve testimonial
- `POST   /api/testimonials/:id/reject` - Reject testimonial

### Public APIs (No Auth)
- `GET    /api/testimonials/request/:token` - Validate request token, get profile context
- `POST   /api/testimonials/submit` - Submit new testimonial
- `POST   /api/testimonials/verify/email` - Trigger magic link
- `GET    /api/testimonials/verify/email/:token` - Verify via magic link
- `GET    /api/testimonials/verify/:provider` - OAuth initiate (github, twitter)
- `GET    /api/testimonials/verify/:provider/callback` - OAuth callback

---

## Frontend Architecture (Svelte 5)

### Admin UI (`frontend/src/routes/admin/testimonials/`)
- `+page.svelte`: Main management interface using the "List + Inline Form" pattern seen in Projects.
- `TestimonialCard.svelte`: Interactive card with Approve/Reject/Edit actions.
- `RequestLinkManager.svelte`: UI for generating and managing invitation links.
- **Sidebar Integration**: Add a top-level "Testimonials" item in `AdminSidebar.svelte`.
  - Use a `PendingBadge.svelte` to show the count of `status='pending'` items.

### Public Submission (`frontend/src/routes/testimonial/`)
- `[token]/+page.svelte`: The submission form.
- `[token]/success/+page.svelte`: Confirmation page.
- `verify/[token]/+page.svelte`: Landing page for magic link verification.

### Public Display (`frontend/src/components/public/`)
- `TestimonialsSection.svelte`: The modular section component.
- `TestimonialWall.svelte`: Masonry layout using CSS columns or Grid.
- `TestimonialCarousel.svelte`: Touch-friendly slider.
- `VerificationBadge.svelte`: Subtle indicator of verification method (e.g., GitHub icon).

---

## Detailed Workflows

### 1. Verification Flows

#### Email Magic Link
1. User submits testimonial and clicks "Verify with Email" on the thank-you page.
2. User enters their email address and clicks "Send Verification".
3. Backend creates `email_verification_tokens` record with HMAC-SHA256 hashed token (15-minute expiry).
4. Sends styled HTML email via `services/email.go` with verification link.
5. User clicks link → `/testimonial/verify/[token]` page loads.
6. Backend validates token hash and expiry → Updates `testimonials.verified_at` and `verification_method`.
7. Verification page shows success state.

**Prerequisites:** SMTP must be configured (via env vars or admin UI at /admin/settings/site).
**i18n:** Email subject, body, and footer are translated based on the site's configured default locale.

#### OAuth (GitHub/Twitter)
1. User clicks "Verify with GitHub".
2. System initiates OAuth flow, passing `testimonial_id` in `state`.
3. Callback exchanges code for profile info.
4. Updates `testimonials` with `verification_data` (avatar URL, profile link).

### 2. View System Integration
Extend the `ViewSection` JSON in the `views` collection:
```typescript
interface ViewTestimonialConfig {
  enabled: boolean;
  layout: 'wall' | 'carousel' | 'featured';
  max_display: number | null;
  testimonial_ids: string[]; // Ordered list for curation
}
```
In `ViewPreview.svelte`, render the testimonials based on this config, defaulting to "all approved" if `testimonial_ids` is empty.

### 3. Admin Email Notifications

When a new testimonial is submitted, all admin users receive an email notification.

**Flow:**
1. Testimonial submitted via `POST /api/testimonials/submit`.
2. Backend resolves the public base URL from the request headers (not AppURL setting).
3. `SafeGo` launches a background goroutine to send the notification (non-blocking).
4. Styled HTML email sent to all admin email addresses with:
   - Author name, title, company, and relationship
   - Content preview (first 300 characters, rune-safe truncation)
   - Request label (if submitted via a request link)
   - "Review Testimonial" button linking to `/admin/testimonials`
5. Plain-text fallback included for email clients that don't render HTML.

**Prerequisites:** SMTP must be configured. If not enabled, notification is silently skipped (logged).
**Admin discovery:** Admin emails are resolved from `ADMIN_EMAILS` env var, falling back to querying `users` collection.

---

## Security Implementation
- **Token Storage**: Never store raw tokens. Use `hmacSha256(token, ENCRYPTION_KEY)`.
- **Token Lookup**: Use the first 12 characters (`token_prefix`) for O(1) database lookups before verifying the full hash.
- **CSRF**: Standard SvelteKit form actions with CSRF protection.
- **Rate Limiting**: Enforce on `/api/testimonials/submit` and `/api/testimonials/verify/email`.

---

## File Structure

### Backend
- `backend/migrations/1737100000_create_testimonials.go`: Schema definition.
- `backend/hooks/testimonials.go`: Route registration and API logic (700+ lines, 14 endpoints).
- `backend/hooks/smtp.go`: SMTP settings management API.
- `backend/services/testimonial.go`: Token generation, HMAC validation, email verification.
- `backend/services/email.go`: Admin notification and verification email templates.
- `backend/services/email_i18n.go`: Email translations (en, de, elvish, klingon, lolcat).

### Frontend
- `frontend/src/routes/admin/testimonials/`: Admin management.
- `frontend/src/routes/testimonial/[token]/`: Submission flow.
- `frontend/src/components/admin/testimonials/`: Admin-specific components.
- `frontend/src/components/public/TestimonialsSection.svelte`: Public display.

---

## Implementation Checklist

### Phase 1: Backend
- [x] Migration for `testimonials`, `testimonial_requests`, `email_verification_tokens`.
- [x] Implement token generation utility.
- [x] CRUD hooks for admin management.
- [x] Public submission hook with validation.

### Phase 2: Admin UI
- [x] Sidebar integration with pending badge.
- [x] Testimonial list page with status filters.
- [x] Approve/Reject/Edit actions.
- [x] Request link generator.

### Phase 3: Public Flow
- [x] Submission form with Svelte 5 `$state`.
- [x] Email magic link flow.
- [ ] GitHub OAuth integration (deferred).
- [x] Success/Verification landing pages.

### Phase 4: Public Display & Views
- [x] `TestimonialsSection.svelte` with layout options.
- [x] Curation logic in View Editor.
- [x] Mobile-responsive Wall and Carousel.

---

## Style Guide Consistency
- **Colors**: Use the Facet design system (Tailwind `primary` colors, sleek dark mode).
- **Icons**: Use Heroicons (SVGs copied from existing components).
- **Feedback**: Always use the `toasts` store for success/error messages.
- **Safety**: Use `confirm` store before destructive actions (delete/reject).
