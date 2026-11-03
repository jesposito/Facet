# Facet documentation

These pages cover **self-hosted Facet**: the open-source app in this repository that you run on your own machine or server. [Facet Cloud](#facet-cloud-is-a-separate-service) is a separate hosted service and isn't documented here.

Pages are grouped by who usually needs them. Owners can stop after the first two groups.

## Running your own Facet

| Page | Read it when you want to |
|------|--------------------------|
| [Install and first login](SETUP.md) | Get Facet running with Docker, Docker Compose or Unraid, then sign in for the first time |
| [Configuration reference](CONFIGURATION.md) | Look up an environment variable, its default, and whether the bundled Compose file passes it through |
| [Putting Facet on the internet](SELF-HOSTING-GUIDE.md) | Add a domain, HTTPS, a reverse proxy or a Cloudflare Tunnel, and set up Google/GitHub sign-in |
| [Upgrades, backups and recovery](UPGRADE.md) | Upgrade safely, back up and restore, roll back, or get back in after a lockout |
| [Security, ownership and privacy](SECURITY.md) | Understand who can sign in, what's encrypted, what visitors' browsers send to third parties, and what Facet records |
| [Troubleshooting](TROUBLESHOOTING.md) | Fix a container that won't start, a login that fails, broken OAuth, missing uploads, and similar problems |

## Using Facet

| Page | What it covers |
|------|----------------|
| [Features guide](FEATURES.md) | Profile and content types, views, visibility, share links, homepage, testimonials, comments, analytics, import and export, feeds and SEO, languages, demo mode |
| [Project experiences](experiences/README.md) | Case studies, video courses and code walkthroughs inside a project page |
| [Media library](MEDIA.md) | Uploads, external embeds, storage location, orphan cleanup |
| [Contact protection](CONTACT_PROTECTION.md) | What each contact protection level does, and what it doesn't |
| [Newsletter, payments and courses](AUDIENCE.md) | Subscriber lists and broadcasts, Stripe and paid content, and the optional course tools |
| [AI features](AI_FEATURES.md) | Optional AI providers, the writing assistant, resume import, GitHub enrichment and AI resume generation |
| [API, webhooks and integrations](API.md) | The scoped `/api/v1` REST API, signed webhooks, system alerts and feeds |

## Working on Facet

| Page | What it covers |
|------|----------------|
| [Contributing](../CONTRIBUTING.md) | How to propose a change, the checks it must pass, and how releases happen |
| [Development guide](DEV.md) | Native and Docker development setup, `make` targets, seed data, PocketBase gotchas |
| [Browser tests](../frontend/tests/README.md) | Running and writing the Playwright suite |
| [Architecture](ARCHITECTURE.md) | How the container, proxy, frontend, backend and database fit together |
| [Design principles](DESIGN.md) | Why Facet works the way it does, and the trade-offs it accepts |
| [Roadmap](ROADMAP.md) | Known gaps and partially built areas |
| [Translations](../CONTRIBUTING_TRANSLATIONS.md) | Adding or improving a language |
| [Testimonials reference](specs/TESTIMONIALS.md) | Data model and endpoints behind testimonials |
| [Diagrams](diagrams/README.md) | Editable architecture and data-model diagrams |

Historical engineering records, kept for context only: [Soft Premium redesign backport plan](REDESIGN_BACKPORT_PLAN.md).

Older page names still work. [AI_WRITING_ASSISTANT.md](AI_WRITING_ASSISTANT.md) and [AI_INTEGRATION_GUIDE.md](AI_INTEGRATION_GUIDE.md) now point to their new homes.

## Facet Cloud is a separate service

Facet Cloud ([get-facet.com](https://get-facet.com)) is a hosted service related to this project. It runs its own code, with its own features, billing, data handling and support, and none of that is described here. If you use Facet Cloud, follow its documentation. If you run the code from this repository, these pages are the ones to trust.

Self-hosted Facet has no account with, and sends no data to, Facet Cloud. The outbound connections it can make are listed in [Security, ownership and privacy](SECURITY.md#what-leaves-your-server).

## A note on accuracy

These pages describe what the code in this repository does, with links to the source files where that helps. Where a feature is only partly built, the page says so. If you spot something that doesn't match what your instance does, please [open an issue](https://github.com/jesposito/Facet/issues) with the page name and what you saw.
