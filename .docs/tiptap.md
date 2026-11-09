# Tiptap editor

Checked **2026-11-10**.

All direct Tiptap packages are locked at **3.32.2**. The npm registry reports **3.33.0** for each, so a coordinated minor update is available.

| Package                                                        | Purpose in Facet                  |
| -------------------------------------------------------------- | --------------------------------- |
| `@tiptap/core`                                                 | Editor core and extension types   |
| `@tiptap/starter-kit`                                          | Base editing extensions           |
| `@tiptap/markdown`                                             | Markdown input/output integration |
| `@tiptap/pm`                                                   | ProseMirror packages              |
| `@tiptap/extension-code-block-lowlight`                        | Highlighted code blocks           |
| `@tiptap/extension-image`                                      | Images                            |
| `@tiptap/extension-link`                                       | Links                             |
| `@tiptap/extension-placeholder`                                | Empty-field placeholders          |
| `@tiptap/extension-table` and table row/header/cell extensions | Table editing                     |

`frontend/src/components/admin/MarkdownEditor.svelte` lazy-loads the editor and its extensions. Keep all Tiptap packages on the same release to avoid mixing core and extension versions. After an update, check Markdown round trips, links, images, tables, code blocks, XSS handling and the public rendering path, then run the dependency and browser tests.

Sources: [`frontend/package.json`](../frontend/package.json), [`frontend/package-lock.json`](../frontend/package-lock.json), [`frontend/src/components/admin/MarkdownEditor.svelte`](../frontend/src/components/admin/MarkdownEditor.svelte), and the npm registry `/latest` endpoint for each package under [`@tiptap`](https://www.npmjs.com/org/tiptap), all checked 2026-11-10. Example primary registry record: [`@tiptap/core` 3.33.0](https://registry.npmjs.org/%40tiptap%2Fcore/latest).
