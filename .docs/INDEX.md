# Technology documentation

Current as of **2026-11-17** against `main` commit `9d4df76c9106d024967e2dd2e30bed89cab9639a` (`v3.8.12`). The initial inventory baseline was `f3d7a149bc9f21d82843efca6e98daf2171dffbf` (`v3.8.3`). Versions come from the lockfiles and module manifests, not the partially stale local `frontend/node_modules`. Latest versions come from official registries and vendor releases. Every technology has a linked note with sources and a check date.

## Act on this

- **OpenTelemetry 1.47.0 is fixed on `main` in [PR #541](https://github.com/jesposito/Facet/pull/541), merged as `4cffe0e`.** GHSA-q7gw-24fr-5p69 / CVE-2026-128085 affected the v3.8.11 proxy's `otel/sdk` 1.46.0, `otel/sdk/log` 0.22.0 and `otel/trace` 1.46.0. All seven aligned modules now select 1.47.0. Proxy tests, vet, build, Caddyfile validation, go-jose tests and reachable `govulncheck` passed. Dependabot alerts 127–129 are fixed, duplicate PRs #538–#540 are closed, and `Facet-1ak` is closed. Facet's default Caddyfile does not configure tracing, so practical reachability remains unverified. Read `caddy.md` and `security-advisories.md`.
- **The optional Google Analytics CSP fix is in [PR #542](https://github.com/jesposito/Facet/pull/542).** The shipped policy blocked Google's tag and collection requests, so the admin setting could not work. The PR adds the documented GA4 hosts without loading anything unless the owner configures and enables Analytics. Caddy validation passed; no live Google property was tested. Read `caddy.md`.
- **Node 24.22.3 and minimatch 11.0.8 remain fixed in v3.8.10 and later.** Both published architectures and the host's nvm default run Node 24.22.3. The development-only minimatch lockfile selects 11.0.8; its two named GHSA pages remain unavailable, so former exposure is unverified. ESLint 10.15.0 separately names a minimatch 11.0.1 floor for CVE-2026-118764, which Facet already exceeds. `Facet-95e` and `Facet-0w3` are closed. Read `node.md`, `minimatch.md` and `security-advisories.md`.
- **postcss-selector-parser 7.2.1 and the Debian digest refresh are fixed in v3.8.11.** PR #537 merged as `2c347ab`; `Facet-8qm` is closed. The parser is development-only and its production npm tree is empty. All three production base digests currently match their tags, and the v3.8.11 published-image scan passed. Read `postcss-selector-parser.md`, `container-base-images.md` and `debian.md`.
- **Do not take smallstep/crypto 0.92.0 as a routine update.** A compatibility build failed against certificates 0.30.2 after the go-jose v4 API change. Keep 0.91.0 and Facet's local v3 patch until certificates supports it. All three known go-jose v3 advisories are covered by local backports, and the proxy CI step runs the replacement's regression tests alongside Caddy's tests. OSV continues matching upstream v3.0.5 by version; it does not inspect those backports. Read `smallstep-crypto.md` and `go-jose.md`.
- **Keep the Docker host on Ubuntu's supported channel.** Installed `docker.io 29.1.3-0ubuntu3~24.04.2` is also the current `noble-security` candidate; Ubuntu backports rather than tracking every upstream release as a candidate. Upstream 29.10.0 is now latest, but there is no apt update to apply today. Swarm is inactive, so the two Swarm CVEs do not apply. `Facet-jjq` tracks Ubuntu's eventual fix with Jed's sudo. Compose 2.40.3 has a new major, 5.7.1, which needs a separate compatibility review. Read `docker.md` and `docker-compose.md`.
- **Schedule tested service API migrations.** Anthropic 2026-10-21 and GitHub REST 2026-03-10 are current, while Facet's selected versions remain supported with no immediate shutdown. GitHub supports REST 2022-11-28 until **2028-03-10**. Ollama v0.40.16 is latest upstream, but each owner administers their own server. No configured-model replacement is required. The unused GPT Image 2 notice is absent from today's extracted lifecycle page. Read `external-services.md`.
- **Use the lockfile for local checks.** `npm audit` reports zero findings for the locked frontend graph. The existing `node_modules` is older than the lockfile for several packages. Run `npm ci` in `frontend/` when a clean local install is needed.
- **Other compatible updates are available:** x/crypto 0.59.0, x/image 0.48.0, SQLite 1.62.0, go-fitz 1.28.3, SvelteKit 3.2.0, adapter-node 6.0.2, Tiptap 3.34.0, PocketBase JS 0.29.0, Marked 18.2.1, minimatch 11.0.9, typescript-eslint 8.73.0, Vite 8.3.8, Playwright 1.65.1, ESLint 10.15.0, PostCSS 8.5.33, Prettier 3.10.1 and several frontend patches. Update deliberately and run the checks in `architecture.md`.
- **Keep local security mitigations.** Facet carries guarded `braces` and `deepmerge` forks, a go-jose v3 backport and an SMTP timeout patch. Read `security-advisories.md`.
- **Major migrations remain intentional:** Tailwind 4, TypeScript 7 after Svelte tooling supports it, Node 26, npm 12, Stripe API majors and cosign-installer 4. `Facet-ehg` tracks the frontend compatibility project.

Resolved since the initial inventory: the AI migration merged in [PR #531](https://github.com/jesposito/Facet/pull/531) and `Facet-c7g` is closed. Caddy, compress and APT fixes merged in [PR #528](https://github.com/jesposito/Facet/pull/528). Go 1.27.4 and all three go-jose backports merged in [PR #533](https://github.com/jesposito/Facet/pull/533) as `25fe89f`, were released in v3.8.8 and closed `Facet-zl8`. Node 24.22.3 and minimatch 11.0.8 merged in [PR #536](https://github.com/jesposito/Facet/pull/536) as `04bfd3e`, were released in v3.8.10 and closed `Facet-95e` and `Facet-0w3`. The parser and Debian updates merged in [PR #537](https://github.com/jesposito/Facet/pull/537) as `2c347ab`, were released in v3.8.11 and closed `Facet-8qm`. OpenTelemetry 1.47.0 merged in [PR #541](https://github.com/jesposito/Facet/pull/541) as `4cffe0e`, was released in v3.8.12 at `9d4df76`, and closed `Facet-1ak`; both architectures and the published-image scan passed.

## Core inventory

All rows were checked **2026-11-17**. “Current” means the selected version matches the cited upstream source; it does not prove compatibility or deployed configuration.

| Technology                  | Installed / selected                      | Latest released                 | Status                                            | File                      |
| --------------------------- | ----------------------------------------- | ------------------------------- | ------------------------------------------------- | ------------------------- |
| Go                          | 1.27.4                                    | 1.27.4                          | current; host toolchain absent                    | `go.md`                   |
| Node.js                     | 24 runtime; host and image 24.22.3        | 24.22.3; 26.13.1 LTS            | current on Node 24; v3.8.11 verified              | `node.md`                 |
| npm                         | host 11.19.1                              | 12.2.2; Node 24 bundles 11.19.1 | current for Node 24; new major available          | `npm.md`                  |
| Docker Engine               | host 29.1.3                               | 29.10.0 upstream                | current Ubuntu security candidate; apt watch      | `docker.md`               |
| Docker Compose              | host 2.40.3                               | 5.7.1                           | update available, new major                       | `docker-compose.md`       |
| Air                         | 1.67.4 selected                           | 1.67.4                          | current; host binary absent                       | `air.md`                  |
| PocketBase Go               | 0.40.6                                    | 0.40.6                          | current                                           | `pocketbase.md`           |
| PocketBase JS               | 0.28.2                                    | 0.29.0                          | minor update available                            | `pocketbase-js.md`        |
| SQLite                      | 1.61.1                                    | 1.62.0                          | minor update available                            | `sqlite.md`               |
| Caddy                       | 2.11.9                                    | 2.11.9                          | current; proxy security checks passed             | `caddy.md`                |
| OpenTelemetry proxy modules | seven aligned modules at 1.47.0           | 1.47.0                          | fixed on `main`; `Facet-1ak` closed               | `caddy.md`                |
| Svelte                      | 5.57.3                                    | 5.57.3                          | current                                           | `svelte.md`               |
| SvelteKit                   | 3.1.1                                     | 3.2.0                           | minor update available                            | `sveltekit.md`            |
| adapter-node                | 6.0.1                                     | 6.0.2                           | patch update available                            | `adapter-node.md`         |
| Vite                        | 8.3.5                                     | 8.3.8                           | patch updates available                           | `vite.md`                 |
| TypeScript                  | 6.0.3                                     | 7.0.2                           | new major intentionally blocked                   | `typescript.md`           |
| Tailwind CSS                | 3.4.19                                    | 4.3.5                           | planned major migration                           | `tailwindcss.md`          |
| Tiptap packages             | 3.32.2                                    | 3.34.0                          | minor update available                            | `tiptap.md`               |
| Marked                      | 18.1.3                                    | 18.2.1                          | minor update available                            | `marked.md`               |
| isomorphic-dompurify        | 4.6.0                                     | 4.7.0                           | minor update available                            | `isomorphic-dompurify.md` |
| ESLint                      | 10.14.0                                   | 10.15.0                         | minor update available                            | `eslint.md`               |
| Playwright                  | 1.64.1                                    | 1.65.1                          | minor update available                            | `playwright.md`           |
| Fontsource packages         | 5.3.0 or 5.3.1                            | same                            | current                                           | `fonts.md`                |
| GitHub Actions              | current major tags; Trivy pinned SHA      | see action notes                | cosign-installer 4 available; most tags float     | `github-actions.md`       |
| Anthropic                   | Claude 5.5 presets on `main`              | managed service                 | active; no immediate replacement required         | `anthropic.md`            |
| OpenAI                      | `gpt-6-luna` on `main`                    | managed service                 | active; no deprecation listed                     | `openai.md`               |
| GitHub Models               | removed from release automation           | retired 2026-07-30              | retired call removed in PR #531                   | `github-models.md`        |
| Ollama                      | local `/api/generate` and `/api/tags`     | 0.40.16 upstream                | server unpinned; owner-specific comparison needed | `ollama.md`               |
| Stripe                      | stripe-go/v82 82.5.1                      | 82.5.1 on v82; 87.1.0 on v87    | current on selected major                         | `stripe.md`               |
| Debian Bookworm             | current digest pin; APT upgraded at build | Bookworm LTS to 2028-06-30      | v3.8.11 image scan passed                         | `debian.md`               |

## Per-technology notes

### Backend and data

- Framework and storage: `pocketbase.md`, `pocketbase-dbx.md`, `ozzo-validation.md`, `sqlite.md`.
- Document and media handling: `go-docx.md`, `go-fitz.md`, `imaging.md`, `x-image.md`, `pandoc.md`, `latex.md`, `libheif.md`.
- Application libraries: `whatlanggo.md`, `golang-jwt.md`, `pquerna-otp.md`, `cobra.md`, `goldmark.md`, `x-crypto.md`, `x-time.md`, `yaml-v3.md`.
- Proxy and mail: `caddy.md`, `smallstep-crypto.md`, `klauspost-compress.md`, `go-jose.md`, `mailyak.md`, `smtp.md`.
- Summary table: `backend-dependencies.md`.

### Frontend framework and runtime libraries

- `svelte.md`, `sveltekit.md`, `adapter-node.md`, `vite.md`, `vite-plugin-svelte.md`, `svelte-check.md`.
- `typescript.md`, `typescript-eslint.md`, `types-node.md`, `types-qrcode.md`.
- `tailwindcss.md`, `tailwind-typography.md`, `postcss.md`, `autoprefixer.md`, `postcss-selector-parser.md`.
- `pocketbase-js.md`, `marked.md`, `isomorphic-dompurify.md`, `qrcode.md`, `svelte-i18n.md`, `svelte-dnd-action.md`.
- `braces.md`, `deepmerge.md`, `esbuild.md`, `minimatch.md`.
- Summary tables: `frontend-libraries.md`, `frontend-tooling.md`.

### Tiptap editor

`tiptap-core.md`, `tiptap-starter-kit.md`, `tiptap-markdown.md`, `tiptap-pm.md`, `tiptap-code-block-lowlight.md`, `tiptap-image.md`, `tiptap-link.md`, `tiptap-placeholder.md`, `tiptap-table.md`, `tiptap-table-row.md`, `tiptap-table-header.md`, `tiptap-table-cell.md`. Overview: `tiptap.md`.

### Fontsource packages

`fontsource-dm-mono.md`, `fontsource-dm-sans.md`, `fontsource-fira-code.md`, `fontsource-hanken-grotesk.md`, `fontsource-ibm-plex-mono.md`, `fontsource-ibm-plex-sans.md`, `fontsource-inter.md`, `fontsource-jetbrains-mono.md`, `fontsource-lato.md`, `fontsource-lora.md`, `fontsource-merriweather.md`, `fontsource-newsreader.md`, `fontsource-nunito-sans.md`, `fontsource-playfair-display.md`, `fontsource-plus-jakarta-sans.md`, `fontsource-sora.md`, `fontsource-source-code-pro.md`, `fontsource-source-sans-3.md`, `fontsource-space-grotesk.md`. Overview: `fonts.md`.

### Test and lint tools

`playwright.md`, `axe-playwright.md`, `eslint.md`, `eslint-js.md`, `eslint-plugin-svelte.md`, `eslint-config-prettier.md`, `globals.md`, `prettier.md`, `prettier-plugin-svelte.md`, `svelte-check.md`, `typescript-eslint.md`, `govulncheck.md`.

### CI actions and release tools

`actions-checkout.md`, `actions-setup-go.md`, `actions-setup-node.md`, `actions-upload-artifact.md`, `actions-download-artifact.md`, `actions-github-script.md`, `actions-attest-build-provenance.md`, `docker-setup-buildx-action.md`, `docker-login-action.md`, `docker-metadata-action.md`, `docker-build-push-action.md`, `cosign-installer.md`, `cosign.md`, `trivy-action.md`, `ghcr.md`. Overview: `github-actions.md`.

### External services

`anthropic.md`, `openai.md`, `github-models.md`, `ollama.md`, `stripe.md`, `github-api.md`, `smtp.md`. Overview: `external-services.md`.

### Runtime and development tools

`go.md`, `node.md`, `npm.md`, `air.md`, `python.md`, `gcc.md`, `docker.md`, `docker-compose.md`, `debian.md`, `apt.md`, `container-base-images.md`.

## Scope and verification limits

- The inventory covers direct dependencies, referenced services and security-relevant transitive modules. It does not enumerate every transitive package. `daily-2026-11-17-research.md` records this check's release deltas, commands and source links.
- No deployed Facet instance, provider account or production host configuration was inspected. Registry and release status is a point-in-time snapshot. External-service lifecycle checks were static and unauthenticated.
- The host has no native Go toolchain or standalone scanner. The combined OpenTelemetry branch passed proxy tests, vet, build, Caddyfile validation, go-jose tests and `govulncheck` in the pinned Go 1.27.4 Bookworm container before merging. [Main security run 43968105029](https://github.com/jesposito/Facet/actions/runs/43968105029) on `4cffe0e` then passed backend tests, both reachable Go vulnerability steps, frontend checks and browser smoke. Those suites were not rerun locally for this watch.
- The frontend lockfile audit returned zero findings across **464 dependencies**. None of the 63 registry-resolved direct dependency or override npm names inspected was marked deprecated at its selected or latest version; the two local guarded forks were inspected separately. None of the 19 direct Go modules' latest manifests carried a deprecation notice.
- Today's OSV package/version query covered **196 Go module/version pairs** and matched only imaging and go-jose/v3; OpenTelemetry no longer matched at 1.47.0. It is not reachability analysis and does not inspect the local go-jose backports.
- [Release run 43968108359](https://github.com/jesposito/Facet/actions/runs/43968108359) completed both architectures and the merge for `v3.8.12` at `9d4df76`, and [published-image scan 43972163295](https://github.com/jesposito/Facet/actions/runs/43972163295) passed. The watcher did not rebuild the production image or run the full browser suite locally.
- `architecture.md` maps the repository and canonical verification commands. Product behaviour remains documented in `docs/`; this directory tracks technology versions and upstream references.
