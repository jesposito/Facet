# Contributing to Facet

Thanks for wanting to help. Bug reports, documentation fixes, translations and code are all welcome.

## Before you start

- Search the [issues](https://github.com/jesposito/Facet/issues) to see whether someone's already on it.
- For anything bigger than a bug fix, open an issue or start a [Discord](https://discord.gg/XD8eUudnmf) conversation first, so we can agree on the approach before you spend the time.
- Read [Design principles](docs/DESIGN.md). Facet is deliberately single-owner, self-hosted and quiet, and changes that pull against that are hard to accept.
- **Found a security problem?** Don't open a public issue. Use **Report a vulnerability** on the [Security tab](https://github.com/jesposito/Facet/security).

Translations have their own guide: [CONTRIBUTING_TRANSLATIONS.md](CONTRIBUTING_TRANSLATIONS.md).

## Set up

Follow the [development guide](docs/DEV.md). In short, on Linux with Go 1.27.3, Node.js 24, Python 3, a C compiler and Air:

```bash
make dev
```

## Make your change

- **Branch from `main`.** Keep each pull request to one fix or feature.
- **Match the code around you.** Backend features usually mean a file in `backend/hooks/` and, if the logic is shared, `backend/services/`. Frontend work follows the existing Svelte 5 and Tailwind patterns.
- **Schema changes go in a new migration** in `backend/migrations/`. Migrations run automatically on every owner's instance, so they must be safe on existing data ([guidance](docs/DEV.md#adding-a-migration)).
- **Public data goes through endpoints that apply visibility rules.** Never loosen collection rules to make something public ([Architecture](docs/ARCHITECTURE.md#access-control-model)).
- **Add or update tests** for behaviour you change: Go tests next to the code, Playwright tests in `frontend/tests/`.
- **Add translation keys to `en.json`**, then to the other locale files (an English placeholder is acceptable if you can't translate).
- **Update the docs** in the same pull request if you change behaviour, configuration or commands. Describe what the code does, and say plainly when something is only partly built.
- **Let the release workflow write version entries in `CHANGELOG.md`.** Add operator-facing migration notices under `### Upgrade notes` in `## [Unreleased]`; the workflow moves those notes into the release.

## Check your change

```bash
make test    # script regressions, Go tests, Svelte type checks
make lint    # go vet, Prettier and ESLint
make fmt     # format Go and frontend code
```

If you touched the UI, run the relevant browser tests too ([frontend/tests/README.md](frontend/tests/README.md)). If you touched translations, run `npm run i18n:validate` in `frontend/`.

## Open the pull request

- **Title it with a conventional prefix**, because the release workflow uses the merged title to choose the next version: `fix: …` makes a patch release, `feat: …` a minor release, and a `!` after the type (as in `feat!: …`) or `BREAKING CHANGE` in the title a major one.
- **Describe what changed and why**, how you tested it, and any follow-up work. The description becomes the source for the changelog entry.
- Link related issues.

## What CI checks

| Workflow | Runs on | What it does |
|----------|---------|--------------|
| [Security and tests](.github/workflows/security-tests.yml) | Pull requests and pushes to `main`, weekly | Script regressions, Go tests, `govulncheck` for the backend and the Caddy build, `npm audit`, vendored-dependency tests, `svelte-check`, a production build, and browser smoke tests against a fresh database ([script](scripts/ci/run-browser-smoke.sh)) |
| [Supply chain](.github/workflows/supply-chain.yml) | Dependency and Docker changes | Go modules are tidy, `package-lock.json` matches `package.json` and is reproducible, and Docker base images are pinned by digest |
| [i18n validation](.github/workflows/i18n-check.yml) | Translation and frontend changes | Key parity and placeholder checks |
| [Build and publish](.github/workflows/docker-publish.yml) | Pull requests to `main` and some branches | Builds both architectures, scans with Trivy, and smoke-tests the packaged image |
| [Published image security](.github/workflows/published-image-security.yml) | Weekly | Re-scans the published image for fixable high and critical issues |

`main` requires the "Backend tests and security" and "Frontend checks and smoke" jobs to pass.

## Releases and published images

Releases are automatic ([auto-tag workflow](.github/workflows/auto-tag.yml)). When a pull request is merged into `main`:

1. The workflow picks the version bump from the pull request title.
2. It writes a `CHANGELOG.md` entry and updates `LATEST_VERSION`. It first tries to group merged pull requests into "Bugs Fixed", "New Features" and "Other Changes" with a language model (GitHub Models, falling back to OpenAI if a key is configured), keeps only entries whose pull request numbers really exist, and falls back to sorting by title keywords if that doesn't work.
3. Before pushing to `main`, it validates that exact commit on a temporary branch with the full test workflow ([script](scripts/ci/push-validated-release.sh)).
4. It tags the commit (`vX.Y.Z`) and triggers the image build.
5. The image is built for `linux/amd64` and `linux/arm64`, scanned, signed with cosign, given SBOM and provenance attestations, and published to `ghcr.io/jesposito/facet` as `X.Y.Z`, `X.Y`, `X` and `latest`.

Owners see the new version in the admin's update notice, which reads `LATEST_VERSION` from `main`.

## Supply-chain maintenance

- **Docker base images** in `docker/Dockerfile` are pinned as `tag@sha256:digest`. Update the tag and digest together from the registry, then run `bash scripts/check-docker-base-pins.sh`. `docker/Dockerfile.dev` is development-only and may carry the `supply-chain: allow-floating-base` marker. Never add that marker to the release Dockerfile.
- **The Caddy proxy** is built from [`docker/caddy`](docker/caddy) with the same Go toolchain as the backend, rather than using a distribution package with an older embedded Go runtime. Keep the Go version in `backend/go.mod`, `docker/caddy/go.mod` and the Docker build images in step.
- **Dependabot** opens one weekly grouped pull request across npm, both Go modules, Docker and GitHub Actions, with security updates grouped per ecosystem ([config](.github/dependabot.yml)). Routine npm updates stay within major versions. Framework and compiler major upgrades are planned separately. `@types/node` follows the Node.js version, and TypeScript stays below 7 until `svelte-check` supports it.
- **Vendored forks.** `frontend/vendor/guarded-braces` is an MIT-licensed fork of `braces` 3.0.3 that limits nesting and expansion work, because the upstream advisory GHSA-vfj7-8cjw-p6xm has no upstream fix. `frontend/vendor/guarded-deepmerge` carries `deepmerge` 4.3.1 with a filter for prototype-poisoning keys, for advisory GHSA-gcqx-w48m-w2h4 ([notes](frontend/vendor/guarded-deepmerge/README.md)). Their provenance files and upstream licences are kept beside the source. After any dependency refresh, run `npm run test:dependencies` in `frontend/`. Remove a fork once the upstream dependency chain has an equivalent fix.
- **Tailwind's selector parser** is overridden to the patched `postcss-selector-parser` 7.x. The production build and glob tests cover compatibility.

## Questions

Ask in [Discord](https://discord.gg/XD8eUudnmf) or open an issue. Thanks again.
