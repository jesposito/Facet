# Portable project experiences v1

The exact TypeScript contract is `frontend/src/lib/experiences/types.ts`. The optional JSON field on `projects` is `experience`. Null selects the classic project. `enabled:false` retains the authored content while showing the original description. The original title, summary, description, visibility and draft fields retain their meanings. Do not duplicate the title inside the experience.

Version 1 uses `layout:immersive`, six named tones, a hero and ordered sections. Sections contain the six discriminated block types in the contract. All block, section, lesson and step IDs are unique across the document and match `[a-zA-Z][\w-]{0,63}`. They remain stable through reordering. Story field keys are unique within the activity and at most 32 characters.

Story templates use `{fieldKey}` substitutions and `[[pause]]` for a pause. Substitution is plain text. No HTML interpolation or evaluation. Unknown placeholders remain visible as text. User input never goes to a server. Pause duration is 0–5000 milliseconds.

Limits: JSON at most 500,000 bytes; 30 sections; 20 blocks per section; 30 lessons or steps per block; 12 story fields. IDs 64 characters, titles/labels 200, normal single-line metadata 500, prose/code/transcript/template 50,000, URLs 2,048. Empty strings and empty lists are valid during authoring; the renderer provides readable empty states. All arrays/objects must have the expected shape and only known fields. Strings render as escaped text except body/intro prose, which goes through the existing rich-text sanitizer.

Image/video/poster/caption URLs allow root-relative paths (no protocol-relative URLs, backslashes or control characters) and HTTPS without credentials. Hero actions additionally allow a section fragment. No arbitrary JavaScript, iframe documents, styles, remote server-side fetching or access tokens. Empty optional URLs are allowed. The renderer checks schemes again before using a URL.

Server validation must cover native PocketBase writes, imports and duplicates, not only the admin UI. Project visibility rules apply before the experience is serialized. The public project detail and intentional public projection must carry this field; private/draft records must not leak it. Export/import must preserve it and rewrite managed media URLs using the existing media mapping. Media reference detection must inspect nested experience values before deleting an upload.

This is a portable proposal for Cloud coordination. Adaptation is possible before release; do not independently invent a second incompatible schema.
