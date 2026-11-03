# Project experience contract, version 1

This is the data format behind [project experiences](README.md). The exact TypeScript types are in [`frontend/src/lib/experiences/types.ts`](../../frontend/src/lib/experiences/types.ts), and the server enforces the same rules in [`backend/hooks/project_experience.go`](../../backend/hooks/project_experience.go). If this page and the code ever disagree, the code wins. Please fix this page.

## Where it lives

`projects.experience` is an optional JSON field.

- `null` means the classic project page.
- `enabled: false` keeps the authored content but shows the classic description.
- The project's own title, summary, description, visibility and draft fields keep their usual meanings. Don't repeat the title inside the experience.

## Shape

Version 1 documents have `version`, `enabled`, `layout: "immersive"`, a `hero`, and an ordered list of `sections`.

- Tones: `paper`, `aqua`, `sand`, `green`, `pink`, `ink`.
- Each section has an `id`, `title`, `intro`, `tone`, a `layout` of `stack` or `split`, and `blocks`.
- Block types: `text`, `image`, `video` (with `lessons`), `code` (with `steps`), `disclosure` and `story`.
- Every block, section, lesson and step ID is unique across the whole document, matches `[a-zA-Z][\w-]{0,63}`, and stays the same when items are reordered.
- Story field keys are unique within their activity and at most 32 characters.

## Story templates

Templates use `{fieldKey}` for substitutions and `[[pause]]` for a pause. Substitution is plain text, with no HTML interpolation or evaluation. Unknown placeholders stay visible as typed. Visitor input never goes to a server. Pause length is 0–5000 milliseconds.

## Limits

| Limit | Value |
|-------|-------|
| Whole document | 500,000 bytes of JSON |
| Sections | 30 |
| Blocks per section | 20 |
| Lessons or steps per block | 30 |
| Story fields | 12 |
| IDs | 64 characters |
| Titles and labels | 200 characters |
| Other single-line text | 500 characters |
| Prose, code, transcripts, templates | 50,000 characters |
| URLs | 2,048 characters |

Empty strings and empty lists are valid while authoring, and the renderer shows readable empty states for them. Every object must have the expected shape and only known fields. Strings are rendered as escaped text, except section intros and text bodies, which go through the existing rich-text sanitiser.

## URLs

Image, video, poster and caption URLs may be root-relative paths (no protocol-relative `//` URLs, backslashes or control characters) or HTTPS URLs without embedded credentials. Hero actions may also be a fragment pointing at a section. Empty optional URLs are allowed. The renderer checks schemes again before using any URL.

Not allowed: arbitrary JavaScript, iframe documents, inline styles, server-side fetching of remote URLs, and access tokens.

## Rules for code that touches experiences

- Validation must cover every write path: the admin, PocketBase's own record API, `/api/v1` writes, imports and duplication.
- Apply the project's visibility and draft rules before serialising the experience. Private and draft projects must never leak it.
- Export, import and duplication must keep the document and rewrite managed media URLs using the existing media mapping.
- Media usage, replacement and unused-file detection must look inside nested experience values before deleting an upload.

The contract was written so it could be shared with Facet Cloud. Change it by evolving this version deliberately, rather than introducing a second, incompatible schema.
