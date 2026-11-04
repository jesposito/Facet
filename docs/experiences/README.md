# Project experiences

A project page normally shows a title, summary and description. A **project experience** turns it into something richer: a case study, a short video course, or a step-by-step code walkthrough, built from sections you arrange yourself.

## Creating one

In a project's editor, below the description, choose **Start a case study**, **Start a video course** or **Start a code walkthrough**. Each opens the same section editor with a different starting template.

Each section has a title, an introduction, a colour tone, and a one- or two-column layout. Inside a section you can add:

- **Text**, with the same rich-text formatting as elsewhere
- **Images**, with captions
- **Video chapters**: a playlist of videos with optional captions and transcripts
- **Code steps**: source code shown as text, one step at a time, with a copy button
- **Disclosures**: expandable "show more" panels
- **A story activity**: a fill-in-the-blanks story that visitors complete with their own words

Move sections and blocks with the move buttons, which also work from the keyboard.

The preview uses the same renderer as the public page. You can switch it to a 390-pixel mobile width and toggle light and dark mode independently of the admin theme. Save the project as usual. Local draft recovery covers the experience too.

Switching the experience off keeps everything you've written and shows the classic description again. Switch it back on whenever you like.

## Video

Use files from your [Media Library](../MEDIA.md) for videos and WebVTT caption files. The usual upload size limits and the site's Content-Security-Policy still apply. You can save an external HTTPS video URL, but the browser may refuse to play it if the site's policy doesn't allow that source. This feature doesn't loosen the policy.

A YouTube, Vimeo or Bunny Stream page isn't a video file, so Facet shows a **Watch Video** link that opens the page in a new tab, with the chapter's poster behind the link and its transcript below. The provider's player isn't embedded and caption files aren't used. A video file hosted by one of those providers still gets the normal player. If a video only plays on approved domains, its page may refuse visitors who arrive from this link, so upload those videos to your Media Library instead.

Nothing plays automatically. Switching chapters pauses the previous video, and transcripts stay readable even if a video won't play.

## Story activities

Visitors type words into the fields you define, and Facet fills them into your story template as plain text. They can read it in a calm or dramatic style, with optional pauses, and stop, reset or print it. Read-aloud uses the visitor's own browser speech, which some browsers don't offer. What visitors type never leaves their browser: it isn't sent to Facet or stored. Speech stops when they leave the page.

## Good to know

- An experience follows its project's visibility and draft settings. A private or draft project's experience is never shown to visitors.
- Export and duplication include the experience. Media usage, replacement and unused-file cleanup look inside it, so files it uses aren't reported as unused.
- The server checks every experience when it's saved, including through the [API](../API.md) and PocketBase's own record endpoints. Anything malformed is rejected with an error, and anything unreadable on a page falls back to the classic project view.
- Existing projects need no changes. The feature added one optional field.

## For contributors

The data format is described in the [experience contract](contract.md) and defined in [`frontend/src/lib/experiences/types.ts`](../../frontend/src/lib/experiences/types.ts). Server-side validation is in [`backend/hooks/project_experience.go`](../../backend/hooks/project_experience.go). Browser coverage is in [`frontend/tests/project-experiences.spec.ts`](../../frontend/tests/project-experiences.spec.ts).
