# minimatch override

Checked **2026-11-12**.

Facet's npm override resolves `minimatch` **11.0.4** in the locked graph. Latest is **11.0.8**. Status: patch update available. This is an override rather than a direct application import; the locked copy sits under ESLint and typescript-eslint. Run lint and dependency tests with any refresh.

The official 11.0.7→11.0.8 comparison includes commits [`d70a7bb`](https://github.com/isaacs/minimatch/commit/d70a7bbc73140bd771daf5c30869b28dbc48929c) and [`784603e`](https://github.com/isaacs/minimatch/commit/784603e09641970d2ba2caf8c73a95d1015cc67d), which name **GHSA-hqmv-qgp3-jx75** and **GHSA-c859-jr93-m5j4** for cache memory use and escaped-pattern matching. Both public advisory URLs returned 404, and OSV returned no records for 11.0.4, 11.0.7 or 11.0.8 during this check. Affected versions and Facet's exposure are therefore unverified; the clean npm audit does not settle them. Update to 11.0.8 as a precaution and reassess when the advisory records are public. `Facet-0w3` tracks that work.

Sources: [`frontend/package.json`](../frontend/package.json), [`frontend/package-lock.json`](../frontend/package-lock.json), [npm registry latest](https://registry.npmjs.org/minimatch/latest), [upstream comparison](https://github.com/isaacs/minimatch/compare/v11.0.7...v11.0.8), [OSV API](https://api.osv.dev/v1/query), checked 2026-11-12.
