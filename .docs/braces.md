# braces / guarded-braces

Checked **2026-11-10**.

Facet replaces upstream `braces` with local **`@facet/guarded-braces` 3.0.3-facet.1**, based on upstream 3.0.3. The fork bounds parser nesting and expansion work for GHSA-vfj7-8cjw-p6xm, which has no upstream fix. Upstream latest remains 3.0.3; replacing the fork with that version would remove Facet's mitigation.

Run `cd frontend && npm run test:dependencies` after dependency refreshes. Remove the fork only when the consuming chain has equivalent protection.

Sources: [`frontend/vendor/guarded-braces/package.json`](../frontend/vendor/guarded-braces/package.json), [`CONTRIBUTING.md`](../CONTRIBUTING.md), [npm upstream latest](https://registry.npmjs.org/braces/latest), checked 2026-11-10.
