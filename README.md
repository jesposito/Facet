# Facet

[![Join the Facet HQ Discord](https://img.shields.io/badge/Discord-Join%20Facet%20HQ-5865F2?logo=discord&logoColor=white)](https://discord.gg/XD8eUudnmf)
[![Latest version](https://img.shields.io/github/v/tag/jesposito/Facet?color=blueviolet&label=version&sort=semver)](https://github.com/jesposito/Facet/tags)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Docker image](https://img.shields.io/badge/docker-ghcr.io%2Fjesposito%2Ffacet-2496ED?logo=docker&logoColor=white)](https://github.com/jesposito/Facet/pkgs/container/facet)
[![Build](https://img.shields.io/github/actions/workflow/status/jesposito/Facet/docker-publish.yml?branch=main&label=build)](https://github.com/jesposito/Facet/actions)
[![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Svelte 5](https://img.shields.io/badge/Svelte-5-FF3E00?logo=svelte&logoColor=white)](https://svelte.dev)
[![Self-hosted](https://img.shields.io/badge/self--hosted-friendly-2ea44f?logo=docker&logoColor=white)](docs/SETUP.md)

**Every side of you. Your way.**

Facet is a personal website you host yourself: your profile, work history, projects, writing and talks in one place, with a different **view** for each audience. A recruiter sees your employment history. A conference organiser sees your talks. A client sees case studies and testimonials. You write everything once.

Everything lives in a SQLite database and an uploads folder on your own machine. There's no Facet account and no telemetry.

## What it does

- **Views**: curated versions of your site at addresses like `/recruiter`, each with its own sections, items, wording, call to action and colour
- **Visibility you control**: public, unlisted views behind share links that can expire, password-protected views, and private items
- **Content**: experience, projects, education, certifications, awards, skills, posts, talks, testimonials and your own custom sections
- **Project experiences**: turn a project into a case study, video course or code walkthrough
- **Imports**: pull projects from GitHub, or turn an existing résumé into structured content
- **Optional AI**, using your own Anthropic, OpenAI, compatible or local Ollama provider: a writing assistant, résumé import, GitHub summaries and tailored résumé PDFs
- **Testimonials, comments and a newsletter**, moderated by you
- **Feeds and portability**: RSS, an iCal file of your talks, a sitemap, JSON/YAML export, and a scoped REST API with signed webhooks
- **Two-factor sign-in**, built-in daily backups, and interface translations including German, Elvish, Klingon and LOLcat
- **Room to make up fake people, like some kind of weirdie**: the built-in demo profile belongs to a 900-year-old Time Lord

Newer areas, notably payments, courses and the newsletter, have limits worth reading about first. The [roadmap](docs/ROADMAP.md) lists what works and what doesn't yet.

## Try it

You need Docker. This runs Facet on your own computer only:

```bash
docker run -d --name facet -p 127.0.0.1:8080:8080 \
  -v facet-data:/data -v facet-uploads:/uploads \
  ghcr.io/jesposito/facet:3
```

Facet creates an `admin@example.com` account with a random password in `/data/.admin-credentials` (mode `0600`), not in the log. Read it with `docker exec facet cat /data/.admin-credentials`, then open http://localhost:8080/admin and sign in. Change the password under **Settings**, then delete the handoff file with `docker exec facet rm /data/.admin-credentials`. The demo toggle in the admin header loads a complete sample site to explore.

To keep Facet running properly, with your own email as the login, a domain and backups, follow the [install guide](docs/SETUP.md).

## Self-hosted Facet and Facet Cloud

This repository is **self-hosted Facet**, free and MIT-licensed, and you run it. [Facet Cloud](https://get-facet.com) is a separate hosted service. It's related to this project but runs its own code, with its own features, pricing and data handling, and it isn't covered by the documentation here.

## Documentation

**Running Facet:** [Install and first login](docs/SETUP.md) · [Configuration](docs/CONFIGURATION.md) · [Putting it on the internet](docs/SELF-HOSTING-GUIDE.md) · [Upgrades, backups and recovery](docs/UPGRADE.md) · [Security and privacy](docs/SECURITY.md) · [Troubleshooting](docs/TROUBLESHOOTING.md)

**Using Facet:** [Features](docs/FEATURES.md) · [Project experiences](docs/experiences/README.md) · [Media](docs/MEDIA.md) · [Newsletter, payments and courses](docs/AUDIENCE.md) · [AI features](docs/AI_FEATURES.md) · [API and webhooks](docs/API.md)

**Working on Facet:** [Contributing](CONTRIBUTING.md) · [Development guide](docs/DEV.md) · [Architecture](docs/ARCHITECTURE.md) · [Design principles](docs/DESIGN.md) · [Translations](CONTRIBUTING_TRANSLATIONS.md)

The [documentation index](docs/README.md) lists every page.

## Built with

Go and [PocketBase](https://pocketbase.io/) with SQLite on the backend, [SvelteKit](https://svelte.dev/docs/kit) and Tailwind CSS on the frontend, and [Caddy](https://caddyserver.com/) in front, all in one container. For development on Linux, `make dev` starts everything with hot reload. See the [development guide](docs/DEV.md).

## Support the project

If Facet saves you time or helps your career, you can [buy me a coffee](https://buymeacoffee.com/jesposito). Bug reports, ideas and pull requests are just as welcome: open an [issue](https://github.com/jesposito/Facet/issues) or say hello on [Discord](https://discord.gg/XD8eUudnmf).

## Licence and credits

MIT. See [LICENSE](LICENSE). Bundled third-party code keeps its own licence notices, for example in [`frontend/vendor`](frontend/vendor).

Built by [jesposito](https://github.com/jesposito), on the shoulders of PocketBase, SvelteKit, Caddy and too much coffee.
