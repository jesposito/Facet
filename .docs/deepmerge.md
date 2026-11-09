# deepmerge / guarded-deepmerge

Checked **2026-11-10**.

Facet replaces upstream `deepmerge` with local **`@facet/guarded-deepmerge` 4.3.1-facet.1**, based on upstream 4.3.1. It rejects prototype-control keys on both inputs for GHSA-gcqx-w48m-w2h4. Upstream latest remains 4.3.1; do not remove the filter as part of a no-op-looking version update.

Run `cd frontend && npm run test:dependencies` after dependency refreshes.

Sources: [`frontend/vendor/guarded-deepmerge/package.json`](../frontend/vendor/guarded-deepmerge/package.json), [`frontend/vendor/guarded-deepmerge/README.md`](../frontend/vendor/guarded-deepmerge/README.md), [npm upstream latest](https://registry.npmjs.org/deepmerge/latest), checked 2026-11-10.
