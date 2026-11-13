# postcss-selector-parser override

Checked **2026-11-14**.

Facet's `main` lockfile resolves `postcss-selector-parser` **7.1.7**. Latest is **7.2.1**. The upstream 7.1.8 changelog says it parses four more kinds of crafted input in linear time, closing CPU exhaustion vulnerability **GHSA-67xj-gfww-g738**, and replaces `cssesc` with an internal linear-time copy. The public advisory URL returned 404 during this check, so affected-range details and Facet's exposure remain unverified; the clean npm audit does not settle them.

**Fixed in [PR #537](https://github.com/jesposito/Facet/pull/537):** the lockfile selects 7.2.1 under the existing override and drops `cssesc`. This is a development-only dependency used by Tailwind, `@tailwindcss/typography` and `eslint-plugin-svelte`; the production-only npm tree is empty. `Facet-8qm` tracks the update. Lint, `svelte-check`, the production frontend build and guarded-dependency regression tests passed under Node 24.22.3.

Sources: [`frontend/package.json`](../frontend/package.json), [`frontend/package-lock.json`](../frontend/package-lock.json), [`CONTRIBUTING.md`](../CONTRIBUTING.md), [npm registry latest](https://registry.npmjs.org/postcss-selector-parser/latest), [official 7.2.1 changelog](https://github.com/postcss/postcss-selector-parser/blob/v7.2.1/CHANGELOG.md), [named advisory](https://github.com/advisories/GHSA-67xj-gfww-g738), [PR #537](https://github.com/jesposito/Facet/pull/537), checked 2026-11-14.
