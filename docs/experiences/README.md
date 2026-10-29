# Project experiences

Project editors can choose **Start a case study**, **Start a video course**, or **Start a code walkthrough** below the description. Each opens the same section editor. Add text, images, video chapters, code steps, disclosures or a story activity. Sections have a colour and one- or two-column layout; move buttons work with a keyboard.

The preview uses the public renderer. Mobile width constrains its container to 390px. The light/dark switch is independent of the admin theme. Save the project normally; local draft recovery includes the experience. Switching it off preserves its content and restores the classic description.

Use the site's media library for videos and WebVTT captions. The existing upload size limit and Content Security Policy still apply. An external HTTPS URL can be saved but may be blocked by the site's media policy; this feature does not relax that policy. Transcripts stay readable when playback fails. Nothing plays automatically. Switching chapters pauses the previous video.

Code steps display source as text and offer a copy button. The story activity substitutes visitor input as plain text, supports calm/dramatic reading and optional pauses, and includes stop, reset and print controls. Read-aloud uses the visitor's browser and may be unavailable. Visitor input is not sent to Facet or stored. Speech stops when the component is removed or the page is left.

The versioned JSON contract is documented in `contract.md` and `frontend/src/lib/experiences/types.ts`. Server validation runs on record create/update, including API writes. Existing authorization gates protect the experience with its project. Export and duplication preserve the document. Media usage, replacement and orphan cleanup inspect nested media references.

Existing projects need no migration of their content. The database migration adds a nullable field. Unknown or malformed documents fall back to the classic page. Deploy the product before populating live experience documents, retain the descriptions and take a content snapshot first.
