# minimatch override

Checked **2026-11-14**.

Facet's npm override resolves `minimatch` **11.0.8** in the locked graph, the latest release. **Updated in v3.8.10 through [PR #536](https://github.com/jesposito/Facet/pull/536).** This is a development-only override rather than a direct application import or production-tree dependency; the locked copy sits under ESLint and typescript-eslint.

The official 11.0.7→11.0.8 comparison includes commits [`d70a7bb`](https://github.com/isaacs/minimatch/commit/d70a7bbc73140bd771daf5c30869b28dbc48929c) and [`784603e`](https://github.com/isaacs/minimatch/commit/784603e09641970d2ba2caf8c73a95d1015cc67d), which name **GHSA-hqmv-qgp3-jx75** and **GHSA-c859-jr93-m5j4** for cache memory use and escaped-pattern matching. Both public advisory URLs returned 404, and OSV returned no records for 11.0.4, 11.0.7 or 11.0.8 during this check. Affected versions and Facet's former exposure are therefore unverified; the clean npm audit does not settle them. The v3.8.10 lockfile selects the containing fixes, and `Facet-0w3` is closed. Reassess if the advisory records become public.

Sources: [`frontend/package.json`](../frontend/package.json), [`frontend/package-lock.json`](../frontend/package-lock.json), [npm registry latest](https://registry.npmjs.org/minimatch/latest), [upstream comparison](https://github.com/isaacs/minimatch/compare/v11.0.7...v11.0.8), [OSV API](https://api.osv.dev/v1/query), checked 2026-11-14.
