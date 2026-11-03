# API, webhooks and integrations

Facet has three ways for other software to work with your site:

- the **`/api/v1` REST API**, using API keys you create, for scripts and integrations that read or change content
- **webhooks**, which Facet sends to your URL when something happens
- **feeds**, the public RSS and calendar files

Everything else under `/api/` serves Facet's own pages and admin dashboard. Those endpoints aren't a stable interface and may change between releases without notice.

## REST API (`/api/v1`)

### Create a key

1. In the admin, open **API Keys**.
2. Create a key with a label (for example "Static site mirror"), the scopes it needs, and optionally allowed origins and an expiry date. The form groups read and write scopes separately, starts with only `read:profile` ticked, and warns you when you tick a write scope ([form](../frontend/src/routes/admin/api/+page.svelte)).
3. Copy the key. It starts with `facet_` and is shown **once**. Store it like a password.

Send it as a bearer token on every request:

```bash
curl https://facet.example.com/api/v1/posts \
  -H "Authorization: Bearer facet_..."
```

An expiry date on its own, such as `2026-12-31`, means the key works until the end of that day in UTC. A date that's invalid or already past is refused with `400`.

Revoking a key in **API Keys** takes effect on the next request. Facet stores only a keyed hash of each key, plus a short prefix so you can tell keys apart ([source](../backend/hooks/api_keys.go)). You can't change a key's scopes or origins after creating it: revoke it and create a new one.

### Scopes

| Scope | Allows |
|-------|--------|
| `read:profile` | `GET /api/v1/profile` |
| `read:posts` | `GET /api/v1/posts` |
| `read:projects` | `GET /api/v1/projects` |
| `read:skills` | `GET /api/v1/skills` |
| `read:experience` | `GET /api/v1/experience` |
| `write:profile` | `PATCH /api/v1/profile` |
| `write:posts` | `POST /api/v1/posts`, `PATCH` and `DELETE /api/v1/posts/{id}` |
| `write:projects` | the same for projects |
| `write:skills` | the same for skills |
| `write:experience` | the same for experience |

Read and write scopes are independent. A key with `write:posts` can't list posts unless it also has `read:posts`.

The form calls an admin endpoint you can also use directly while signed in as the owner, with your session token as the bearer:

```bash
curl -X POST https://facet.example.com/api/admin/api-keys \
  -H "Authorization: Bearer <owner session token>" \
  -H "Content-Type: application/json" \
  -d '{"label":"Blog sync","scopes":["read:posts","write:posts"],"allowed_origins":["https://blog.example.com"],"expires_at":"2026-12-31"}'
```

The response contains the full key once, as the form does. Unknown scopes get `400`. The key list in the admin shows every key, with its scopes and any allowed origins.

### Allowed origins

Allowed origins stop a key from being used by someone else's web page. If a key's list is empty, any origin may use it. If the list has entries:

- requests without an `Origin` header are allowed, which covers curl, servers and scripts
- a browser request is allowed only when its `Origin` exactly matches an entry
- `null`, malformed values, subdomains of a listed site and requests with more than one `Origin` header get `403`

Write each entry as a scheme and host, such as `https://blog.example.com`, with a port only if it isn't the default. Facet tidies entries when you save, so `HTTPS://Blog.Example.com:443/` is stored as `https://blog.example.com`. Wildcards, paths, queries, user names and non-http(s) schemes are refused with `400`, and a key can list at most 20 origins.

Origin restrictions do not replace authentication: non-browser clients can choose their own `Origin` header, so a leaked key still works from a script. PocketBase supplies CORS response headers and answers browser preflight requests. The key's allowlist is checked separately on the authenticated API request; a successful preflight does not grant access.

### Reading

List endpoints return `{"data": [...]}` with up to 200 items. They include only items that are **public and not drafts**:

| Endpoint | Fields returned |
|----------|-----------------|
| `GET /api/v1/posts` | `id`, `title`, `slug`, `excerpt`, `content`, `tags`, `visibility`, `is_draft`, `access_tier`, `published_at`, `created`, `updated` |
| `GET /api/v1/projects` | `id`, `title`, `slug`, `summary`, `description`, `experience`, `tech_stack`, `links`, `url`, `categories`, `visibility`, `is_draft`, `is_featured`, `sort_order`, `access_tier`, `created`, `updated` |
| `GET /api/v1/skills` | `id`, `name`, `category`, `proficiency`, `level`, `visibility`, `sort_order` |
| `GET /api/v1/experience` | `id`, `company`, `title`, `location`, `start_date`, `end_date`, `is_current`, `description`, `bullets`, `skills`, `visibility`, `is_draft`, `sort_order` |
| `GET /api/v1/profile` | `name`, `headline`, `summary`, `location`, `updated` |

A few fields are derived for older clients:

- `level` on skills is a copy of `proficiency`.
- `url` on projects is the first entry in `links` that has a URL, or `""`.
- `is_current` on experience is `true` when `end_date` is empty, the same rule the site uses to show "Present".

Earlier versions returned `status` on projects. Projects have no such field, so it was always `""`, and it has been removed. A client that reads `status` will now find it missing.

Paid content stays paid. When a post or project has `access_tier` set to `paid`, the API blanks the parts a buyer pays for: `content` on posts, and `description` and `experience` on projects. Titles, summaries and excerpts stay visible, as they do on the site. An API key doesn't count as a purchase.

`/api/v1/profile` returns `404` when the profile's visibility is private, matching the public site.

### Writing

Send JSON. Fields not in this list are silently ignored, which stops a key from changing owner IDs, timestamps or files ([source](../backend/hooks/api_v1_writes.go)):

| Collection | Writable fields |
|------------|-----------------|
| posts | `title`, `slug`, `excerpt`, `content`, `tags`, `visibility`, `is_draft`, `published_at` |
| projects | `title`, `slug`, `summary`, `description`, `experience`, `tech_stack`, `links`, `categories`, `visibility`, `is_draft`, `is_featured`, `sort_order` |
| skills | `name`, `category`, `proficiency`, `visibility`, `sort_order` |
| experience | `company`, `title`, `location`, `start_date`, `end_date`, `description`, `bullets`, `skills`, `visibility`, `is_draft`, `sort_order` |
| profile | `name`, `headline`, `location`, `summary`, `contact_email`, `contact_links`, `visibility` |

| Request | Success | Errors |
|---------|---------|--------|
| `POST /api/v1/<collection>` | `201` with the new record | `400` if validation fails |
| `PATCH /api/v1/<collection>/{id}` | `200` with the updated record | `404` if the ID doesn't exist, `400` on validation |
| `DELETE /api/v1/<collection>/{id}` | `204`, permanently deleted | `404` if the ID doesn't exist |

Missing, malformed or unknown keys get `401`. Revoked or expired keys, keys without the needed scope, and browser requests from an origin the key doesn't allow get `403`.

Always set `visibility` and `is_draft` when you create something. If you leave `is_draft` out, the item is created as published. If you leave `visibility` out, it's left empty, which keeps the item out of listings and feeds but still lets it appear in views that include it.

Create a draft post:

```bash
curl -X POST https://facet.example.com/api/v1/posts \
  -H "Authorization: Bearer facet_..." \
  -H "Content-Type: application/json" \
  -d '{"title":"Hello","slug":"hello","content":"Markdown body","visibility":"public","is_draft":true}'
```

Publish it later:

```bash
curl -X PATCH https://facet.example.com/api/v1/posts/RECORD_ID \
  -H "Authorization: Bearer facet_..." \
  -H "Content-Type: application/json" \
  -d '{"is_draft":false,"published_at":"2026-11-03 09:00:00.000Z"}'
```

Every write made with a key is recorded in the audit log (**Settings → Audit Log**) with the action, the collection, the record ID, and the key's label and prefix. The key itself and its hash are never written there. If the audit entry can't be saved, the write still succeeds and the failure goes to the server log ([source](../backend/hooks/audit.go)).

### Rate limit

Every `/api/v1` request, read or write, counts against a per-IP limit of about one request a second, with bursts of up to 10. The limit is checked before the key, so requests with bad keys count too. Over the limit you get `429` with a `Retry-After` header in seconds ([source](../backend/hooks/ratelimit.go)). Limits are kept in memory and reset when Facet restarts.

## Webhooks

Webhooks let Facet tell another system when something happens, for example to post to a chat channel when you publish, or to rebuild a static site. Set them up under **Webhooks** in the admin.

### Events

| Event | Sent when | `data` fields |
|-------|-----------|---------------|
| `post.published` | A post, talk or custom content item is **saved while not a draft** | `id`, `collection`, `title`, `slug` |
| `comment.created` | A visitor posts a comment | `id`, `content_type`, `content_id`, `author_name`, `author_email`, `body`, `status` |
| `newsletter.subscribed` | Someone subscribes | `id`, `email`, `name`, `source` |

Two details about `post.published` ([source](../backend/hooks/webhooks.go)):

- It fires on **every save** of a published item, not only when it changes from draft to published. Fixing a typo sends it again. Use the `id` to ignore repeats.
- It fires on updates only. An item created as already published sends nothing until it's next saved.

`comment.created` includes the commenter's email address and comment text. Only send it to systems you trust with that.

### What arrives

Each delivery is an HTTP `POST` with a JSON body:

```json
{
  "event": "post.published",
  "timestamp": "2026-11-03T09:00:00Z",
  "data": {
    "id": "abc123",
    "collection": "posts",
    "title": "Hello",
    "slug": "hello"
  }
}
```

and these headers:

| Header | Value |
|--------|-------|
| `X-Facet-Event` | The event name |
| `X-Facet-Delivery` | A unique ID for this delivery |
| `X-Facet-Signature` | `sha256=` followed by the hex HMAC-SHA256 of the raw body, using the webhook's secret |

The secret is shown **once**, when you create the webhook.

### Verify the signature

Always check the signature against the **raw** request body before trusting a delivery. In Node.js:

```js
import crypto from 'node:crypto';

function isValid(rawBody, signatureHeader, secret) {
  const expected = 'sha256=' + crypto.createHmac('sha256', secret).update(rawBody).digest('hex');
  const a = Buffer.from(expected);
  const b = Buffer.from(signatureHeader ?? '');
  return a.length === b.length && crypto.timingSafeEqual(a, b);
}
```

### Retries and failures

([source](../backend/services/webhooks.go))

- A delivery succeeds when your endpoint answers with a 2xx status within 10 seconds.
- Failed deliveries are retried after 1 minute, 5 minutes, 30 minutes, 2 hours and 12 hours, so 6 attempts in all.
- After 10 failures in a row the webhook is switched off. Switching it back on resets the count.
- Every attempt is listed under the webhook's **Deliveries**, with the status code and response.
- Deliveries wait in an in-memory queue. If Facet restarts, or the queue fills up during a burst, undelivered events are lost. Don't treat webhooks as a guaranteed record of every event.

### Where webhooks can be sent

Facet resolves the destination address at the moment it connects, and refuses loopback, private, link-local, carrier-grade NAT and cloud-metadata addresses, including through redirects. This stops a webhook being used to probe your own network. It also means a receiver on your LAN or on the same machine, such as `http://192.168.1.50/hook` or `http://localhost:3000`, will be refused. Use a publicly reachable address.

### Testing

- **Send test** on a webhook sends a test delivery and shows the response.
- **Dispatch test event** sends a sample payload for a chosen event to every active webhook subscribed to it.

Test payloads are marked with `"is_test": true` and use sample field names (such as `post_id` and `excerpt`) that differ from the real events above. Build your receiver against the real field list.

## System alerts

**Settings → System Alerts** (`/admin/alerts`) lists background problems with severity, filters, acknowledgement and deletion ([source](../backend/hooks/system_alerts.go)). Failed startup content repair raises `content_recovery_failed`; backup, restore and retention failures raise `backup_failed`, `backup_restore_failed` and `backup_retention_failed`. Repeated events are combined while an alert remains unacknowledged, including concurrent failures. Content repair retries on the next start. SMTP and webhook errors still use their operation status and server logs rather than this inbox.

## Feeds

| Address | Contents |
|---------|----------|
| `/rss.xml` | RSS 2.0 feed of public, published posts |
| `/talks.ics` | iCalendar file of public, published talks, for calendar apps |
| `/sitemap.xml` | Public views, posts, projects and talks |

These need no key and are rate limited like other public pages.

## Data export

`GET /api/export?format=json` (or `format=yaml`) downloads a full content export. It needs a signed-in owner session, not an API key. [Features](FEATURES.md#exporting-your-data) describes what's included.

## Health check

`GET /health` checks the frontend and asks the backend whether it's healthy. It returns `200` with `{"ok": true, ...}` when both are fine and `503` when the backend isn't. The Docker health check uses it. `GET /api/health` is PocketBase's own backend-only check.
