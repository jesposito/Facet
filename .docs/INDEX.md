# Technology documentation

Current as of **2026-11-10**, initially inventoried at commit `f3d7a149bc9f21d82843efca6e98daf2171dffbf` (`v3.8.3`). Versions come from the lockfiles and module manifests, not the partially stale local `frontend/node_modules`. Latest versions come from official registries and vendor releases. Every technology has a linked note with sources and a check date.

## Act on this

- **Finish and merge the AI migration.** The `f3d7a14` baseline contained only retired Anthropic presets, retired OpenAI models and a call to GitHub Models, which shut down on **2026-07-30**. The current migration branch selects active Claude 5.5 models and `gpt-6-luna`, removes GitHub Models and uses the required GPT-6 request parameters. `Facet-c7g` tracks verification and merge. Read `anthropic.md`, `openai.md` and `github-models.md`.
- **Caddy and image fixes are merged.** Baseline CI failed reachable `govulncheck` on `klauspost/compress` 1.20.1. [PR #528](https://github.com/jesposito/Facet/pull/528), merged as `fde26de`, selects Caddy 2.11.9 and compress 1.20.2, refreshes base pins and fixes the APT findings. Full security CI and both architecture image scans and packaged smoke tests passed. `Facet-cyi` is closed. Read `caddy.md`, `apt.md` and `security-advisories.md`.
- **Publish the refreshed runtime image.** The original published-image scan found `apt` and `libapt-pkg6.0` vulnerable to CVE-2026-109518. PR #528 refreshes all three base pins; a disposable image check found both packages at fixed **2.6.2**, and PR scans passed. Read `apt.md` and `container-base-images.md`.
- **Resolve the ARM64 GHCR publication denial.** Push run 42706080159 failed only the ARM64 registry upload with `permission_denied: write_package`; AMD64 publication succeeded. This is not an account-wide loss of access. `Facet-dcy` remains open until both architecture digests and the signed manifest publish. Read `ghcr.md`.
- **Do not take smallstep/crypto 0.92.0 as a routine update.** A compatibility build failed against certificates 0.30.2 after the go-jose v4 API change. Keep 0.91.0 and Facet's local v3 patch until certificates supports it. The local patch covers only the RSA advisory, not all go-jose v3 findings. Read `smallstep-crypto.md` and `go-jose.md`.
- **Update the Docker host through Ubuntu's supported channel.** Host Engine 29.1.3 is behind 29.9.3, which includes engine, Swarm and BuildKit security fixes. Compose 2.40.3 has a new major, 5.7.0; review that separately. Read `docker.md` and `docker-compose.md`.
- **Use the lockfile for local checks.** `npm audit` reports zero findings for the locked frontend graph. The existing `node_modules` is older than the lockfile for several packages, including ESLint, Marked and PocketBase JS. Run `npm ci` in `frontend/` when a clean local install is needed.
- **Other compatible updates are available:** x/image 0.48.0, SQLite 1.62.0, Tiptap 3.33.0, Vite 8.3.7, Playwright 1.65.0 and several frontend patches. Update deliberately and run the checks in `architecture.md`.
- **Keep local security mitigations.** Facet carries guarded `braces` and `deepmerge` forks, a go-jose v3 backport and an SMTP timeout patch. Read `security-advisories.md`.
- **Major migrations remain intentional:** Tailwind 4, TypeScript 7 after Svelte tooling supports it, Node 26, npm 12, Stripe API majors and cosign-installer 4. `Facet-ehg` tracks the frontend compatibility project.

## Core inventory

All rows were checked **2026-11-10**. “Current” means the selected version matches the cited upstream source; it does not prove compatibility or deployed configuration.

| Technology           | Installed / selected                   | Latest released                 | Status                                             | File                      |
| -------------------- | -------------------------------------- | ------------------------------- | -------------------------------------------------- | ------------------------- |
| Go                   | 1.27.3                                 | 1.27.3                          | current stable; host toolchain absent              | `go.md`                   |
| Node.js              | 24 runtime; host 24.22.1               | 24.22.2; 26.13.0 LTS            | host patch behind; Node 24 supported to 2028-04-30 | `node.md`                 |
| npm                  | host 11.19.1                           | 12.2.2; Node 24 bundles 11.19.1 | current for Node 24; new major available           | `npm.md`                  |
| Docker Engine        | host 29.1.3                            | 29.9.3                          | update available, security fixes                   | `docker.md`               |
| Docker Compose       | host 2.40.3                            | 5.7.0                           | update available, new major                        | `docker-compose.md`       |
| Air                  | 1.67.4 selected                        | 1.67.4                          | current; host binary absent                        | `air.md`                  |
| PocketBase Go        | 0.40.6                                 | 0.40.6                          | current                                            | `pocketbase.md`           |
| PocketBase JS        | 0.28.2                                 | 0.28.2                          | current                                            | `pocketbase-js.md`        |
| SQLite               | 1.61.1                                 | 1.62.0                          | minor update available                             | `sqlite.md`               |
| Caddy                | 2.11.9 on main (`fde26de`)             | 2.11.9                          | current; security checks passed                    | `caddy.md`                |
| Svelte               | 5.57.3                                 | 5.57.3                          | current                                            | `svelte.md`               |
| SvelteKit            | 3.1.1                                  | 3.1.1                           | current                                            | `sveltekit.md`            |
| adapter-node         | 6.0.1                                  | 6.0.1                           | current                                            | `adapter-node.md`         |
| Vite                 | 8.3.5                                  | 8.3.7                           | patch updates available                            | `vite.md`                 |
| TypeScript           | 6.0.3                                  | 7.0.2                           | new major intentionally blocked                    | `typescript.md`           |
| Tailwind CSS         | 3.4.19                                 | 4.3.5                           | planned major migration                            | `tailwindcss.md`          |
| Tiptap packages      | 3.32.2                                 | 3.33.0                          | minor update available                             | `tiptap.md`               |
| Marked               | 18.1.3                                 | 18.1.3                          | current                                            | `marked.md`               |
| isomorphic-dompurify | 4.6.0                                  | 4.7.0                           | minor update available                             | `isomorphic-dompurify.md` |
| ESLint               | 10.14.0                                | 10.14.0                         | current                                            | `eslint.md`               |
| Playwright           | 1.64.1                                 | 1.65.0                          | minor update available                             | `playwright.md`           |
| Fontsource packages  | 5.3.0 or 5.3.1                         | same                            | current                                            | `fonts.md`                |
| GitHub Actions       | current major tags; Trivy pinned SHA   | see action notes                | cosign-installer 4 available; most tags float      | `github-actions.md`       |
| Anthropic            | Claude 5.5 presets in migration branch | managed service                 | updated; merge pending                             | `anthropic.md`            |
| OpenAI               | `gpt-6-luna` in migration branch       | managed service                 | updated; merge pending                             | `openai.md`               |
| GitHub Models        | removed from auto-tag in migration     | retired 2026-07-30              | retired call removed                               | `github-models.md`        |
| Ollama               | local `/api/generate` and `/api/tags`  | installation-specific           | no hosted version selected                         | `ollama.md`               |
| Stripe               | stripe-go/v82 82.5.1                   | 82.5.1 on v82; 87.1.0 on v87    | current on selected major                          | `stripe.md`               |
| Debian Bookworm      | refreshed base pins in PR #528         | Bookworm LTS to 2028-06-30      | fixed apt verified; publication pending            | `debian.md`               |

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

- The inventory covers direct dependencies, referenced services and security-relevant transitive modules. It does not enumerate every transitive package.
- No deployed Facet instance, provider account or production host configuration was inspected. Registry and release status is a point-in-time snapshot.
- Local Go tests and `govulncheck` were not run because Go and the standalone scanner are absent on this host. CI evidence is cited instead. The frontend lockfile audit returned zero findings.
- Latest scheduled CI on the baseline commit passed frontend checks, backend tests and backend `govulncheck`, but failed the Caddy vulnerability step. The published-image scan's two APT findings are documented above.
- `architecture.md` maps the repository and canonical verification commands. Product behaviour remains documented in `docs/`; this directory tracks technology versions and upstream references.
