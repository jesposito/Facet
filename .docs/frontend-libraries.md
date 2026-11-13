# Frontend runtime libraries

Checked **2026-11-14**. Versions below come from `frontend/package-lock.json`; latest versions come from each package's official npm registry `/latest` record on the same date.

| Package                | Locked | Latest | Use / status                              |
| ---------------------- | ------ | ------ | ----------------------------------------- |
| `marked`               | 18.1.3 | 18.2.0 | Markdown parsing; minor update available  |
| `isomorphic-dompurify` | 4.6.0  | 4.7.0  | HTML sanitisation; minor update available |
| `qrcode`               | 1.5.4  | 1.5.4  | QR generation; current                    |
| `svelte-i18n`          | 4.0.1  | 4.0.1  | Interface translations; current           |
| `svelte-dnd-action`    | 0.9.80 | 0.9.80 | Drag-and-drop reordering; current         |
| `pocketbase`           | 0.28.2 | 0.29.0 | API client; minor update available        |

Marked and DOMPurify sit on content-rendering boundaries. Updating them requires the stored-XSS and rich-content browser tests, not only a type check. Translation changes require `npm run i18n:validate`. Drag-and-drop changes should retain keyboard reordering coverage.

Registry sources: [Marked](https://registry.npmjs.org/marked/latest), [isomorphic-dompurify](https://registry.npmjs.org/isomorphic-dompurify/latest), [qrcode](https://registry.npmjs.org/qrcode/latest), [svelte-i18n](https://registry.npmjs.org/svelte-i18n/latest), [svelte-dnd-action](https://registry.npmjs.org/svelte-dnd-action/latest), [PocketBase JS](https://registry.npmjs.org/pocketbase/latest). Repository sources: [`frontend/package.json`](../frontend/package.json), [`frontend/package-lock.json`](../frontend/package-lock.json), [`frontend/tests/security.spec.ts`](../frontend/tests/security.spec.ts), [`frontend/tests/README.md`](../frontend/tests/README.md). All checked 2026-11-14.
