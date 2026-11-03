# Testimonials: technical reference

How the testimonials feature is built. For how to use it, see [Features → Testimonials](../FEATURES.md#testimonials). The source is [`backend/hooks/testimonials.go`](../../backend/hooks/testimonials.go), [`backend/services/testimonial.go`](../../backend/services/testimonial.go) and the migrations named `*testimonial*` in [`backend/migrations`](../../backend/migrations).

## What it does

- Owners create **request links** and send them to people they've worked with.
- Recipients submit a testimonial through a public form, with no account.
- Submitters can optionally **verify their email address** with a link that expires in 15 minutes. Email is the only verification method.
- Submissions arrive as pending. The owner approves, rejects, edits, features or deletes them.
- Approved testimonials can be grouped into lists and shown in views as a wall, a carousel or featured cards.

## Data model

### `testimonials`

| Group | Fields |
|-------|--------|
| Content | `content`, `relationship`, `project` |
| Author | `author_name`, `author_title`, `author_company`, `author_photo` (file, 5 MB), `author_website` |
| Verification | `verification_method` (`none` or `email`), `verification_identifier`, `verification_data`, `verified_at` |
| Workflow | `status` (`pending`, `approved`, `rejected`), `request_id`, `submitted_at`, `approved_at`, `rejected_at`, `rejection_reason` |
| Display | `featured`, `sort_order`, `list`, `visibility`, `view_visibility` |

### `testimonial_requests`

Follows the same security model as share links.

| Group | Fields |
|-------|--------|
| Security | `token_hash` (HMAC-SHA256 of the full token), `token_prefix` (first 12 characters, for lookup) |
| Customisation | `label`, `custom_message`, `recipient_name`, `recipient_email`, `list` |
| Limits | `expires_at`, `max_uses`, `use_count` |
| State | `is_active` |

### `email_verification_tokens`

`testimonial_id`, `email`, `token_hash`, `expires_at` (15 minutes after creation), `verified_at`.

## Endpoints

**Owner (signed in):**

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/api/testimonials/requests` | Create a request link. The raw token is returned once. |
| `GET` | `/api/testimonials/requests` | List request links |
| `DELETE` | `/api/testimonials/requests/{id}` | Delete a request link |
| `POST` | `/api/testimonials/requests/{id}/regenerate` | Issue a new token for a request link |
| `GET` | `/api/testimonials/lists` | Testimonial lists in use |
| `GET` | `/api/testimonials` | List testimonials, filterable by status |
| `PATCH` | `/api/testimonials/{id}` | Edit content, status, featured flag and display settings |
| `POST` | `/api/testimonials/{id}/approve` | Approve |
| `POST` | `/api/testimonials/{id}/reject` | Reject |
| `DELETE` | `/api/testimonials/{id}` | Delete |
| `GET` | `/api/testimonials/pending-count` | Count for the sidebar badge |

**Public:**

| Method | Path | Rate limit | Purpose |
|--------|------|------------|---------|
| `GET` | `/api/testimonials/request/{token}` | Moderate | Validate a request link and return the form context |
| `POST` | `/api/testimonials/submit` | Strict | Submit a testimonial |
| `POST` | `/api/testimonials/verify/email` | Strict | Send a verification email |
| `GET` | `/api/testimonials/verify/email/{token}` | — | Confirm the email address |
| `GET` | `/api/public/testimonials` | — | Approved, public testimonials |

The public pages are `/testimonial/<token>` (the form) and `/testimonial/verify/<token>` (the verification landing page).

## Email

Both emails need SMTP. Without it, Facet logs the skipped email and carries on.

- **New submission notification.** Sent in the background to each address in `ADMIN_EMAILS`, or to the accounts in `users` if that's empty. It includes the author details, the first 300 characters of the testimonial, the request link's label, and a button to the review page. Links use `APP_URL`, falling back to the request's forwarded host.
- **Verification email.** Sent to the address the submitter enters. The link expires after 15 minutes.

Both are translated into the site's default language ([email translations](../../backend/services/email_i18n.go)).

## Display in views

Each view's testimonials section chooses a layout (`wall`, `carousel` or `featured`) and optionally a hand-picked, ordered set of testimonials. With nothing picked, it shows all approved testimonials whose visibility is public. Per-view visibility switches on each testimonial can hide it from particular views.

## Security

- Request tokens are never stored raw: only an HMAC of the token, keyed from `ENCRYPTION_KEY`, plus a 12-character prefix for lookup.
- Submission and verification endpoints use the strict rate limit. Request link lookups use the moderate limit.
- Only approved testimonials with public visibility are served publicly.

## Not built

GitHub, Twitter and LinkedIn verification were designed but aren't implemented. The `verification_method` field has values for them, but only `none` and `email` are ever set.
