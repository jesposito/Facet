# Media System (Storage, API, and Embeds)

This doc explains how Facet’s media pipeline works and how to extend or touch it without breaking things.

## Storage layout
- Files are stored under `pb_data/storage/<collectionId>/<recordId>/<filename>`.
- Primary file fields today:
  - `profile.hero_image`, `profile.avatar`
  - `experience.media`, `projects.media`, `projects.cover_image`
  - `education.media`, `certifications` (none), `posts.cover_image`
  - `talks` (no files), `views` (no files)
  - `view_exports.file`
- Orphans: anything in `storage` that isn’t referenced by a file field above.

## Collections
- `external_media`: link-based entries for embeds. Fields: `url` (required), `title`, `mime`, `thumbnail_url`.
- `media_refs` relation (multi-select) is added to `projects`, `posts`, `talks` to attach external media.
- `uploads`: generic files added via the Media Library upload form (single `file` field plus optional title/mime).

## API endpoints
- `GET /api/media` (auth required):
  - Builds referenced items by scanning file fields (see above).
  - Merges external_media entries (normalized via `mediaembed.Normalize`).
  - Optionally includes orphans via `includeOrphans=1` or `orphans=1`.
  - Returns stats (referenced/orphan/storage size and counts).
- `POST /api/media/external` / `DELETE /api/media/external/{id}`: manage `external_media` entries.
- `DELETE /api/media`: delete a file from a record, or delete an orphan via `relative_path`.
- `POST /api/media/bulk-delete`: delete multiple orphans by relative paths.

## Normalization (backend/services/mediaembed)
Recognizes providers and builds `provider`, `embed_url`, `thumbnail_url`, `mime`:
- YouTube (watch/embed/short), Vimeo, Loom
- SoundCloud, Spotify
- CodePen, Figma
- Immich: treated as link card; inline only if direct image/video URL is detected.
- Direct image/video/pdf URLs
- Fallback: link card

## Admin UI
- Media Library: lists uploads + external entries, shows storage/orphan stats, bulk orphan delete, and supports uploading files directly into the `uploads` collection.
- Projects/Posts forms: multi-select of media options (uploads or external entries) stored in `media_refs`.
  - Talks picker is planned but not wired yet.

## Public rendering (current state)
- Cover images use responsive thumb/large URLs.
- Media refs are stored but not yet rendered on public project/post/talk pages (follow-up required).

## Common failure modes & how to avoid them
- **Missing external_media collection**: migrations must run; deleting `pb_data` and reseeding applies all migrations.
- **/api/media 400s**: usually due to missing collections or bad paths. Keep `collectMediaItems` and `collectExternalMediaItems` tolerant; avoid early `BadRequest` when scans partially fail.
- **Auth failures**: `/api/media` requires a valid user token (`users` collection). Use seeded creds (`egerthe@gmail.com` / `changeme123`) in dev.
- **Stale data after schema changes**: clear `pb_data` and rerun `SEED_DATA=dev make seed-dev`.

## When changing media code
1) Run `go test ./...` and `npm run check`.
2) Verify `/api/media` with a fresh token.
3) Do not remove file fields listed above without updating collectors.
4) If adding providers, extend `mediaembed.Normalize` and keep it offline (no network calls).
