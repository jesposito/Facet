# Media library

Facet keeps images, documents and video files you upload, plus links to media hosted elsewhere, in one **Media Library**. This page explains how to use it and, further down, how it works for anyone changing the code.

## For owners

### Uploading

You can add files in two ways:

- directly in the **Media Library**, then attach them wherever you need them
- from a field on an item, such as a project's cover image or your profile avatar

Size limits depend on where the file goes ([migrations](../backend/migrations)):

| Where | Limit |
|-------|-------|
| Media Library uploads | 20 MB |
| Profile hero image, project and post covers, item media | 10 MB |
| Avatar, company and institution logos, testimonial photos | 5 MB |
| Favicon | 512 KB |

The built-in proxy also caps any single request at 25 MB, and a reverse proxy in front of Facet may have a lower limit of its own ([Putting Facet on the internet](SELF-HOSTING-GUIDE.md#option-b-a-reverse-proxy-you-run)).

Facet makes smaller copies of images for faster pages. iPhone HEIC photos are converted to JPEG for those copies when the `heif-convert` tool is available, which the Docker image includes.

### External media

Instead of uploading, you can add a link to media hosted elsewhere. Facet recognises these and shows them as embedded players or previews ([source](../backend/services/mediaembed)):

- YouTube, Vimeo and Loom videos
- SoundCloud and Spotify audio
- CodePen pens and Figma files
- direct links to images, video files and PDFs

Anything else becomes a link card. When you add a link from a supported service, Facet asks that service for the title and thumbnail. Embedded players load from the provider's site when a visitor opens the page, which means the provider sees that visit ([Security](SECURITY.md#what-leaves-your-server)).

Immich share links are treated as link cards, unless the URL points straight at an image or video file.

### Organising

The library lets you search, filter, tag files, rename how they're displayed, add alt text and other details, replace a file while keeping its uses, see where each item is used, and find recently used items quickly.

### Cleaning up

Files can be left behind when you delete or change the content that used them. The library shows these **unused files** (orphans) separately with total storage used, and you can delete them one at a time or in bulk. Deleting a file can't be undone, so take a [backup](UPGRADE.md#back-up-your-data) first if you're unsure.

After restoring a built-in backup, files uploaded since that backup become orphans, because the restored database doesn't know about them.

### Where files are stored

| Install | Location |
|---------|----------|
| Docker | `/uploads` in the container (your `UPLOADS_PATH` on the host). `/data/storage` is a link to it. |
| Native development | `pb_data/storage` in the checkout |

Back up the uploads folder together with `/data` ([Backups](UPGRADE.md#back-up-your-data)).

## For contributors

### How it fits together

- Uploaded files live in PocketBase file fields on their records: profile images, experience, project, education and custom-content media, covers, logos, testimonial photos, the `uploads` collection used by the Media Library, generated résumés in `view_exports`, and others. PocketBase stores them under `<storage>/<collectionId>/<recordId>/<filename>`.
- The `external_media` collection holds links (`url`, `title`, `mime`, `thumbnail_url` and metadata). Files uploaded to `uploads` are mirrored into `external_media` so the library can treat everything the same way.
- Projects, posts and talks attach library items through the `media_refs` relation. Several collections also store a library item's URL directly in `*_library_url` fields, such as the profile's `hero_image_library_url`.
- Project experience documents can contain media URLs nested inside their JSON. Usage lookups, replacement and orphan detection inspect those too ([services/media.go](../backend/services/media.go)).

### Automatic cleanup

Shortly after startup and then every 24 hours, a cleanup job deletes `uploads` records older than 24 hours that nothing refers to ([services/cleanup.go](../backend/services/cleanup.go)). It counts a reference from any `*_library_url` field in [services/library_refs.go](../backend/services/library_refs.go) (this includes profile, project, post, talk, view, custom-content, testimonial and course images), from a project experience, or from an `external_media` record whose URL ends with the upload's URL. An upload with such a mirror is kept even when no content uses it. If a lookup fails, the run stops there and keeps the remaining uploads, so a database fault is never treated as "unused". If you add a new `*_library_url` field, add it to that list too.

The same job removes expired and long-revoked share links, used and expired email verification tokens, and failed or stuck résumé exports.

### Endpoints

All of these need a signed-in owner, apart from thumbnails ([hooks/media.go](../backend/hooks/media.go)).

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/media` | List referenced files and external items with storage statistics. Add `includeOrphans=1` to include unused files. |
| `GET` | `/api/media/recent` | Recently used items |
| `GET` | `/api/media/usage/{id}` | Where an external media item is used |
| `GET` | `/api/media/storage-info` | Storage totals |
| `GET` | `/api/media/fetch-metadata?url=` | Look up title and thumbnail from a supported provider |
| `GET` | `/api/media/thumb/{collection}/{record}/{filename}` | Resized image |
| `POST` | `/api/media/external` | Add an external item |
| `DELETE` | `/api/media/external/{id}` | Remove an external item and its references |
| `DELETE` | `/api/media` | Remove a file from a record, or an orphan by `relative_path` |
| `POST` | `/api/media/bulk-delete` | Delete orphans: `{"orphans": ["path", ...]}` returns `{"deleted": n, "failed": m, "errors": [...]}` |
| `POST` | `/api/media/replace` | Replace a file and update references |
| `PATCH` | `/api/media/display-name`, `/api/media/metadata` | Rename, edit alt text and details |
| `POST` | `/api/media/bulk-tag`, `/api/media/mark-used` | Tagging and recently-used tracking |

Files themselves are served by PocketBase at `/api/files/...`, which Caddy caches for a day.

### When you change media code

- Run `make test`. `mediaembed.Normalize` must stay offline (no network calls), so URL recognition works without internet access. Network lookups belong in `FetchMetadata`.
- Bulk delete must keep rejecting path traversal. `frontend/tests/media-management.spec.ts` covers it.
- If you add a file field to a collection, make sure the media collectors and orphan detection see it, or its files will be listed as orphans and could be deleted.
- Check `/api/media` with a fresh owner token after changing collections.
