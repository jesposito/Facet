# Dependency and image security status

Checked **2026-11-11**.

## Current results

- `npm audit --json` returned **zero vulnerabilities** across 465 dependencies for the locked frontend graph. None of the 63 direct dependency or override `/latest` records inspected was marked deprecated. The audit does not establish that the older local `node_modules` matches the lockfile.
- An OSV package/version query covered **196 unique direct and indirect Go module/version entries** from both module manifests. It matched imaging 1.6.2 and go-jose/v3 3.0.5. Compress 1.20.2 no longer matches its former advisory. Package matching is not reachability analysis.
- [Security run 42754669718](https://github.com/jesposito/Facet/actions/runs/42754669718) on `07de104` passed frontend checks, backend tests, backend `govulncheck` and proxy `govulncheck`. This is the current reachable-scan evidence; the older baseline Caddy failure is retained below for context.
- [Published-image scan 42568088543](https://github.com/jesposito/Facet/actions/runs/42568088543) originally failed with exactly two HIGH findings: `apt` and `libapt-pkg6.0`, both for CVE-2026-109518. The later [published-image scan 42724230846](https://github.com/jesposito/Facet/actions/runs/42724230846) passed.
- [PR #528](https://github.com/jesposito/Facet/pull/528), merged as `fde26de`, selects Caddy 2.11.9 and compress 1.20.2 and retains smallstep/crypto 0.91.0 with the local go-jose patch. The Dockerfile explicitly upgrades APT during the image build. Local proxy checks, the full security workflow and both architecture image scans and packaged smoke tests passed. `Facet-cyi` is closed.
- Local Go tests and scans were not run by the librarian because the host lacks Go and standalone `govulncheck`; CI evidence is cited instead.

## Advisories and mitigations

| Dependency                     | Finding                                                                                            | Current mitigation / action                                                                                                                               |
| ------------------------------ | -------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Go 1.27.3                      | Go 1.27.4 includes security fixes to cgo, the Go command, compiler, linker and `net/http`          | [PR #533](https://github.com/jesposito/Facet/pull/533) updates both modules and builders; `Facet-zl8`. Reachability is not inferred from the release note |
| `klauspost/compress` 1.20.1    | GHSA-fpmh-frq8-7rj8 / GO-2026-6865: crafted zstd input can panic decoding; fixed 1.20.2            | **Fixed on main:** 1.20.2 is selected and current reachable CI passed. Latest is 1.20.4; `Facet-cyi` is closed                                            |
| `disintegration/imaging` 1.6.2 | GHSA-q7pp-wcgr-pffx: crafted TIFF panic; no patched upstream release                               | TIFF thumbnail generation is disabled and backend reachable scan passed. Keep `Facet-aqv` open; do not call the upstream issue fixed                      |
| go-jose/v3 3.0.5               | GHSA-mgcc-cww5-fp2q / CVE-2026-116587: RSA JWK CPU denial of service                               | Local replacement backports v4.1.6 RSA bounds checks; retain while v3 remains                                                                             |
| go-jose/v3 3.0.5               | GHSA-4w3g-jfj2-cw2q and GHSA-vvm8-gf78-28g4: separate denial-of-service issues                     | **Not covered by the local RSA backport.** Assess reachability and remove the v3 import chain when compatible                                             |
| `braces` upstream 3.0.3        | GHSA-vfj7-8cjw-p6xm                                                                                | Local `@facet/guarded-braces` 3.0.3-facet.1 bounds nesting and expansion; run `npm run test:dependencies`                                                 |
| `deepmerge` upstream 4.3.1     | GHSA-gcqx-w48m-w2h4                                                                                | Local `@facet/guarded-deepmerge` 4.3.1-facet.1 rejects prototype-control keys; run the same regression command                                            |
| Debian Bookworm apt            | CVE-2026-109518: on-path package swap after verification, requiring redirects and non-zero retries | Bookworm security apt **2.6.2** is fixed. The Dockerfile upgrades APT and the latest published-image scan passed                                          |

OSV did not provide separate GO IDs for the three go-jose findings when queried; its records expose the GHSA IDs above and CVE-2026-116585, CVE-2026-116586 and CVE-2026-116587 respectively.

## Local patches

- `frontend/vendor/guarded-braces` and `guarded-deepmerge` retain upstream licences and provenance. Remove them only when the consuming dependency chain provides equivalent protection.
- `docker/caddy/third_party/go-jose-v3` is based on v3.0.5 with the RSA bounds fix and an explicit Unicode conversion bound. Its tests run from the parent `docker/caddy` module.
- `backend/third_party/mailyak-v3` adds SMTP connection/exchange deadlines. It is unrelated to these advisories but is another replacement to review during upgrades.

Sources: local `npm audit --json` and direct npm registry checks (2026-11-11), [OSV API](https://api.osv.dev/v1/querybatch), [`frontend/vendor/guarded-braces/package.json`](../frontend/vendor/guarded-braces/package.json), [`frontend/vendor/guarded-deepmerge/package.json`](../frontend/vendor/guarded-deepmerge/package.json), [`docker/caddy/third_party/go-jose-v3/FACET-PATCH.md`](../docker/caddy/third_party/go-jose-v3/FACET-PATCH.md), [compress advisory](https://github.com/klauspost/compress/security/advisories/GHSA-fpmh-frq8-7rj8), [compress 1.20.2 release](https://github.com/klauspost/compress/releases/tag/v1.20.2), [imaging advisory](https://github.com/advisories/GHSA-q7pp-wcgr-pffx), [go-jose CBC-HMAC advisory](https://github.com/advisories/GHSA-4w3g-jfj2-cw2q), [go-jose deep-JSON advisory](https://github.com/advisories/GHSA-vvm8-gf78-28g4), [go-jose RSA advisory](https://github.com/advisories/GHSA-mgcc-cww5-fp2q), [Debian CVE-2026-109518 tracker](https://security-tracker.debian.org/tracker/CVE-2026-109518), [Go release history](https://go.dev/doc/devel/release), all checked 2026-11-11.
