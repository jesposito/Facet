# Project designs

Project experiences can use the site's original appearance or one of three built-in designs. Choose **Page design** in the project editor. Changing designs preserves the content, media and activities already in the project.

- **Playful** suits a course or interactive demonstration: bright chapters, large type and generous media.
- **Technical** suits a product walkthrough: clear screenshots, readable code and quieter chapter colours.
- **Editorial** suits a case study: serif headings and comfortable reading space.

Standard width keeps a closer reading measure. Wide gives demonstrations and large images more space. Section headings can sit above the content or alongside their introduction. A video block can use a stacked player, a chapter sidebar or a spotlight layout with the explanation beside the player. The paper story layout puts the controls beside the resulting story. Narrow screens and narrow editor previews use the same responsive layouts.

Colours belong to each chapter so text remains legible when the surrounding site switches theme. Original experiences keep their existing light and dark palettes. Fonts are served by Facet; the font files and OFL licences are in `frontend/static/fonts/experiences`.

## Document format

These fields extend version 1. Documents without them retain the original layout.

```json
{
  "design": { "preset": "playful", "width": "wide" },
  "hero": { "imageNote": "A short note beside the picture" }
}
```

The excerpt only shows the new fields; the document still requires its existing version, enabled, layout, hero and sections fields.

A section may set `headingLayout` to `stack` or `split`. A video block may set `presentation` to `stack`, `sidebar` or `spotlight`. A story block may set it to `plain` or `paper`. `yellow` and `orange` join the existing chapter tones. An image may set `decorative: true` to render an empty alternative text. Meaningful images should retain useful descriptions.

The server validates these choices, including disabled experiences. Arbitrary CSS and font URLs are not part of the document format. Duplication and export retain the options. Export to an older Facet version may require upgrading that installation before importing the new fields.

The editor preview uses the same renderer as the published page. Custom site CSS can still override that renderer; check both the preview and public project when adding your own CSS.
