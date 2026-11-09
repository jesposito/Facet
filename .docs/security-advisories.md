# Dependency and image security status

Checked **2026-11-10**.

## Current results

- `npm audit --json` returned **zero vulnerabilities** across 465 dependencies for the locked frontend graph. The audit does not establish that the older local `node_modules` matches the lockfile.
- An OSV package/version query covered 211 direct and indirect Go module entries. It matched imaging 1.6.2, go-jose/v3 3.0.5 and klauspost/compress 1.20.1. Package matching is not reachability analysis.
- [Scheduled CI run 42568077807](https://github.com/jesposito/Facet/actions/runs/42568077807) on `f3d7a14` passed frontend checks, backend tests and backend `govulncheck`. It failed Caddy `govulncheck` with reachable `huff0.ReadTable`, `huff0.Compress4X`, `huff0.Compress1X`, `fse.Decompress` and `fse.Compress` paths. See `caddy.md`.
- [Published-image scan 42568088543](https://github.com/jesposito/Facet/actions/runs/42568088543) failed with exactly two HIGH findings: `apt` and `libapt-pkg6.0`, both for CVE-2026-109518. The Debian tracker marks Bookworm security version 2.6.2 fixed.
- [PR #528](https://github.com/jesposito/Facet/pull/528), merged as `fde26de`, selects Caddy 2.11.9 and compress 1.20.2, refreshes base pins and retains smallstep/crypto 0.91.0 with the local go-jose patch. Local proxy checks, the full security workflow and both architecture image scans and packaged smoke tests passed. A disposable image contained apt/libapt 2.6.2. `Facet-cyi` is closed; final ARM64 publication awaits the GHCR permission fix in `Facet-dcy`.
- Local Go tests and scans were not run by the librarian because the host lacks Go and standalone `govulncheck`; the separate proxy implementation agent ran the PR checks listed above.

## Advisories and mitigations

| Dependency                     | Finding                                                                                            | Current mitigation / action                                                                                                                           |
| ------------------------------ | -------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| `klauspost/compress` 1.20.1    | GHSA-fpmh-frq8-7rj8 / GO-2026-6865: crafted zstd input can panic decoding; fixed 1.20.2            | **Reachable in Caddy CI.** Select 1.20.2 or newer, preferably current 1.20.4 after tests; `Facet-cyi`                                                 |
| `disintegration/imaging` 1.6.2 | GHSA-q7pp-wcgr-pffx: crafted TIFF panic; no patched upstream release                               | TIFF thumbnail generation is disabled and backend reachable scan passed. Keep `Facet-aqv` open; do not call the upstream issue fixed                  |
| go-jose/v3 3.0.5               | GHSA-mgcc-cww5-fp2q: RSA JWK CPU denial of service                                                 | Local replacement backports v4.1.6 RSA bounds checks; retain while v3 remains                                                                         |
| go-jose/v3 3.0.5               | GHSA-4w3g-jfj2-cw2q and GHSA-vvm8-gf78-28g4: separate denial-of-service issues                     | **Not covered by the local RSA backport.** Assess reachability and remove the v3 import chain when compatible                                         |
| `braces` upstream 3.0.3        | GHSA-vfj7-8cjw-p6xm                                                                                | Local `@facet/guarded-braces` 3.0.3-facet.1 bounds nesting and expansion; run `npm run test:dependencies`                                             |
| `deepmerge` upstream 4.3.1     | GHSA-gcqx-w48m-w2h4                                                                                | Local `@facet/guarded-deepmerge` 4.3.1-facet.1 rejects prototype-control keys; run the same regression command                                        |
| Debian Bookworm apt            | CVE-2026-109518: on-path package swap after verification, requiring redirects and non-zero retries | Debian tracker marks Bookworm security apt **2.6.2** fixed. Verify the actual rebuilt image package version; do not infer it from a base digest alone |

## Local patches

- `frontend/vendor/guarded-braces` and `guarded-deepmerge` retain upstream licences and provenance. Remove them only when the consuming dependency chain provides equivalent protection.
- `docker/caddy/third_party/go-jose-v3` is based on v3.0.5 with the RSA bounds fix and an explicit Unicode conversion bound. Its tests run from the parent `docker/caddy` module.
- `backend/third_party/mailyak-v3` adds SMTP connection/exchange deadlines. It is unrelated to these advisories but is another replacement to review during upgrades.

Sources: local `npm audit --json` (2026-11-10), [`frontend/vendor/guarded-braces/package.json`](../frontend/vendor/guarded-braces/package.json), [`frontend/vendor/guarded-deepmerge/package.json`](../frontend/vendor/guarded-deepmerge/package.json), [`docker/caddy/third_party/go-jose-v3/FACET-PATCH.md`](../docker/caddy/third_party/go-jose-v3/FACET-PATCH.md), [compress advisory](https://github.com/klauspost/compress/security/advisories/GHSA-fpmh-frq8-7rj8), [compress 1.20.2 release](https://github.com/klauspost/compress/releases/tag/v1.20.2), [imaging advisory](https://github.com/advisories/GHSA-q7pp-wcgr-pffx), [go-jose CBC-HMAC advisory](https://github.com/advisories/GHSA-4w3g-jfj2-cw2q), [go-jose deep-JSON advisory](https://github.com/advisories/GHSA-vvm8-gf78-28g4), [go-jose RSA advisory](https://github.com/advisories/GHSA-mgcc-cww5-fp2q), [Debian CVE-2026-109518 tracker](https://security-tracker.debian.org/tracker/CVE-2026-109518), all checked 2026-11-10.
