# Marked

Checked **2026-11-17**. Locked **18.1.3**; latest **18.2.1**, so a minor update is available. Facet uses it for Markdown parsing. The 18.2.1 release includes parser fixes and further work on residual quadratic parsing time for unmatched emphasis and strikethrough delimiters; no security advisory was established by this check. Exercise rich-content and stored-XSS tests after updates; a successful type check does not cover rendering safety.

Sources: [`frontend/package-lock.json`](../frontend/package-lock.json), [`frontend/tests/security.spec.ts`](../frontend/tests/security.spec.ts), [npm registry latest](https://registry.npmjs.org/marked/latest), [Marked 18.2.1 release](https://github.com/markedjs/marked/releases/tag/v18.2.1), checked 2026-11-17.
