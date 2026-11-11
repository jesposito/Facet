# Technology documentation

Current as of **2026-11-11** at commit `28a4daad19804775f88d4561c2721af3119637c9` (`v3.8.8`). The initial inventory baseline was `f3d7a149bc9f21d82843efca6e98daf2171dffbf` (`v3.8.3`). Versions come from the lockfiles and module manifests, not the partially stale local `frontend/node_modules`. Latest versions come from official registries and vendor releases. Every technology has a linked note with sources and a check date.

## Act on this

- **Do not take smallstep/crypto 0.92.0 as a routine update.** A compatibility build failed against certificates 0.30.2 after the go-jose v4 API change. Keep 0.91.0 and Facet's local v3 patch until certificates supports it. All three known go-jose v3 advisories are covered by local backports, and the proxy CI step runs the replacement's regression tests alongside Caddy's tests. OSV continues matching upstream v3.0.5 by version; it does not inspect those backports. The default plain-HTTP Caddyfile does not reach these PKI paths, but the code is compiled in. Read `smallstep-crypto.md` and `go-jose.md`.
- **Update the Docker host through Ubuntu's supported channel.** Host Engine 29.1.3 is behind 29.9.3, which includes engine, Swarm and BuildKit security fixes. Compose 2.40.3 has a new major, 5.7.0; review that separately. Read `docker.md` and `docker-compose.md`.
- **Schedule tested service API migrations.** Anthropic 2026-10-21 and GitHub REST 2026-03-10 are current, while Facet's selected versions remain supported with no immediate shutdown. GitHub supports REST 2022-11-28 until **2028-03-10**. Ollama v0.40.16 is latest upstream, but each owner administers their own server. Read `external-services.md`.
- **Use the lockfile for local checks.** `npm audit` reports zero findings for the locked frontend graph. The existing `node_modules` is older than the lockfile for several packages, including ESLint, Marked and PocketBase JS. Run `npm ci` in `frontend/` when a clean local install is needed.
- **Other compatible updates are available:** x/image 0.48.0, SQLite 1.62.0, SvelteKit 3.2.0, adapter-node 6.0.2, Tiptap 3.33.1, Marked 18.2.0, Vite 8.3.7, Playwright 1.65.0 and several frontend patches. Update deliberately and run the checks in `architecture.md`.
- **Keep local security mitigations.** Facet carries guarded `braces` and `deepmerge` forks, a go-jose v3 backport and an SMTP timeout patch. Read `security-advisories.md`.
- **Major migrations remain intentional:** Tailwind 4, TypeScript 7 after Svelte tooling supports it, Node 26, npm 12, Stripe API majors and cosign-installer 4. `Facet-ehg` tracks the frontend compatibility project.

Resolved since the initial inventory: the AI migration merged in [PR #531](https://github.com/jesposito/Facet/pull/531) and `Facet-c7g` is closed. Caddy, compress and APT fixes merged in [PR #528](https://github.com/jesposito/Facet/pull/528). Go 1.27.4 and all three go-jose backports merged in [PR #533](https://github.com/jesposito/Facet/pull/533) as `25fe89f`, were released in v3.8.8 and closed `Facet-zl8`. The intermittent GHCR denial required no configuration change; later releases published both architectures and `Facet-dcy` is closed.

## Core inventory

All rows were checked **2026-11-11**. “Current” means the selected version matches the cited upstream source; it does not prove compatibility or deployed configuration.

| Technology           | Installed / selected                      | Latest released                 | Status                                             | File                      |
| -------------------- | ----------------------------------------- | ------------------------------- | -------------------------------------------------- | ------------------------- |
| Go                   | 1.27.4                                    | 1.27.4                          | current; host toolchain absent                     | `go.md`                   |
| Node.js              | 24 runtime; host 24.22.1                  | 24.22.2; 26.13.0 LTS            | host patch behind; Node 24 supported to 2028-04-30 | `node.md`                 |
| npm                  | host 11.19.1                              | 12.2.2; Node 24 bundles 11.19.1 | current for Node 24; new major available           | `npm.md`                  |
| Docker Engine        | host 29.1.3                               | 29.9.3                          | update available, security fixes                   | `docker.md`               |
| Docker Compose       | host 2.40.3                               | 5.7.0                           | update available, new major                        | `docker-compose.md`       |
| Air                  | 1.67.4 selected                           | 1.67.4                          | current; host binary absent                        | `air.md`                  |
| PocketBase Go        | 0.40.6                                    | 0.40.6                          | current                                            | `pocketbase.md`           |
| PocketBase JS        | 0.28.2                                    | 0.28.2                          | current                                            | `pocketbase-js.md`        |
| SQLite               | 1.61.1                                    | 1.62.0                          | minor update available                             | `sqlite.md`               |
| Caddy                | 2.11.9                                    | 2.11.9                          | current; security checks passed                    | `caddy.md`                |
| Svelte               | 5.57.3                                    | 5.57.3                          | current                                            | `svelte.md`               |
| SvelteKit            | 3.1.1                                     | 3.2.0                           | minor update available                             | `sveltekit.md`            |
| adapter-node         | 6.0.1                                     | 6.0.2                           | patch update available                             | `adapter-node.md`         |
| Vite                 | 8.3.5                                     | 8.3.7                           | patch updates available                            | `vite.md`                 |
| TypeScript           | 6.0.3                                     | 7.0.2                           | new major intentionally blocked                    | `typescript.md`           |
| Tailwind CSS         | 3.4.19                                    | 4.3.5                           | planned major migration                            | `tailwindcss.md`          |
| Tiptap packages      | 3.32.2                                    | 3.33.1                          | minor update available                             | `tiptap.md`               |
| Marked               | 18.1.3                                    | 18.2.0                          | minor update available                             | `marked.md`               |
| isomorphic-dompurify | 4.6.0                                     | 4.7.0                           | minor update available                             | `isomorphic-dompurify.md` |
| ESLint               | 10.14.0                                   | 10.14.0                         | current                                            | `eslint.md`               |
| Playwright           | 1.64.1                                    | 1.65.0                          | minor update available                             | `playwright.md`           |
| Fontsource packages  | 5.3.0 or 5.3.1                            | same                            | current                                            | `fonts.md`                |
| GitHub Actions       | current major tags; Trivy pinned SHA      | see action notes                | cosign-installer 4 available; most tags float      | `github-actions.md`       |
| Anthropic            | Claude 5.5 presets on `main`              | managed service                 | active; no immediate replacement required          | `anthropic.md`            |
| OpenAI               | `gpt-6-luna` on `main`                    | managed service                 | active; no deprecation listed                      | `openai.md`               |
| GitHub Models        | removed from release automation           | retired 2026-07-30              | retired call removed in PR #531                    | `github-models.md`        |
| Ollama               | local `/api/generate` and `/api/tags`     | 0.40.16 upstream                | server unpinned; owner-specific comparison needed  | `ollama.md`               |
| Stripe               | stripe-go/v82 82.5.1                      | 82.5.1 on v82; 87.1.0 on v87    | current on selected major                          | `stripe.md`               |
| Debian Bookworm      | digest-pinned base; APT upgraded at build | Bookworm LTS to 2028-06-30      | fixed apt verified; published-image scan passed    | `debian.md`               |

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
- No deployed Facet instance, provider account or production host configuration was inspected. Registry and release status is a point-in-time snapshot. External-service lifecycle checks were static and unauthenticated.
- Local Go tests and `govulncheck` were not run because Go and the standalone scanner are absent on this host. [Security run 42941611668](https://github.com/jesposito/Facet/actions/runs/42941611668) on `28a4daa` (`v3.8.8`) passed frontend checks, backend tests and both reachable Go vulnerability steps.
- The frontend lockfile audit returned zero findings across 465 dependencies. The OSV package/version query covered 196 Go entries and matched imaging and go-jose/v3; it is not reachability analysis and does not inspect the local go-jose backports.
- [Published-image scan 42724230846](https://github.com/jesposito/Facet/actions/runs/42724230846) passed. [Release run 42941554396](https://github.com/jesposito/Facet/actions/runs/42941554396) built and scanned both architectures for v3.8.8, then published and verified the signed manifest.
- `architecture.md` maps the repository and canonical verification commands. Product behaviour remains documented in `docs/`; this directory tracks technology versions and upstream references.
