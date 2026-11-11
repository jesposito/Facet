# External services overview

Checked **2026-11-11** against repository commit `07de104` (`v3.8.7`) and the official primary sources linked below. The AI migration described as pending in older notes is merged on `main`.

## Current findings

- **No immediate AI model replacement is required by the checked lifecycle pages.** Facet defaults to `claude-sonnet-5-5` and `gpt-6-luna`; both remain available. Anthropic lists its three configured Claude 5.5 presets as active, and OpenAI does not list GPT-6 Luna in its deprecation schedule.
- **Anthropic has a newer API version.** Facet sends `anthropic-version: 2023-06-01`; the current version is `2026-10-21`. Anthropic recommends the latest version but its version history announces no removal date for `2023-06-01`. Facet does not send the `temperature`, `top_p` or `top_k` parameters removed by the new version. Review and test the upgrade rather than treating it as an emergency.
- **GitHub REST has a newer version with a long migration window.** Facet explicitly uses `2022-11-28` for repository and latest-release requests; its languages and README requests omit the header and currently default to that version. GitHub supports `2022-11-28` until **2028-03-10** and lists `2026-03-10` as current. Plan and test the version upgrade before the support deadline.
- **Ollama and Stripe have available release information but no forced repository change today.** Ollama v0.40.16 is the latest upstream release, but Facet does not pin an Ollama server. stripe-go v82.5.1 remains current on Facet's selected v82 module path; v87.1.0 exists on a separate major path and requires a deliberate migration review.

## Provider checks

### Anthropic

Facet calls `/v1/messages` and discovers models through `/v1/models?limit=100`. The application default is `claude-sonnet-5-5`, with `claude-opus-5-5` and `claude-haiku-5-5` as settings fallbacks. Anthropic's lifecycle page marks all three active, with tentative retirement dates no sooner than:

- `claude-opus-5-5`: **2027-09-22**;
- `claude-sonnet-5-5`: **2027-09-28**;
- `claude-haiku-5-5`: **2027-10-07**.

The current model overview also includes all three in its lineup. These dates apply to Anthropic-operated platforms; partner-operated platforms can differ. Sources: [`backend/services/ai.go`](../backend/services/ai.go), [`backend/hooks/ai.go`](../backend/hooks/ai.go), [official model overview](https://platform.claude.com/docs/en/about-claude/models/overview), [official deprecations](https://platform.claude.com/docs/en/about-claude/model-deprecations), and [API version history](https://platform.claude.com/docs/en/api/versioning), checked 2026-11-11.

### OpenAI and compatible APIs

Facet defaults to `gpt-6-luna` through `/v1/chat/completions`, discovers models through `/v1/models`, and permits a custom base URL. Application requests set `reasoning_effort: "none"`, retain `temperature`, and use `max_completion_tokens` for bounded GPT-6 requests. Release automation uses the same model and reasoning setting. OpenAI's model page confirms Chat Completions support and the `none` reasoning effort; its migration guide permits sampling parameters when effort is `none`.

The deprecation page does not list `gpt-6-luna`. Dated GPT-4o, GPT-4o-mini, GPT-4.1 and GPT-4.1-mini snapshots remain scheduled for shutdown on **2027-01-14**, but those models are no longer Facet defaults. OpenAI recommends the Responses API for new work; Chat Completions remains supported for GPT-6 Luna. Facet retains deprecated `max_tokens` only for custom non-reasoning providers, while recognised reasoning models use `max_completion_tokens`.

Sources: [`backend/services/ai.go`](../backend/services/ai.go), [`backend/hooks/ai.go`](../backend/hooks/ai.go), [`.github/workflows/auto-tag.yml`](../.github/workflows/auto-tag.yml), [GPT-6 Luna model documentation](https://developers.openai.com/api/docs/models/gpt-6-luna), [GPT-6 parameter guidance](https://developers.openai.com/api/docs/guides/latest-model#update-api-and-model-parameters), [Chat Completions reference](https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create/), and [official deprecations](https://platform.openai.com/docs/deprecations), checked 2026-11-11.

### Other services

| Service       | Facet configuration and current upstream status                                                                                                                                                                                                                                                                                                                                                                                  | Action                                                                                         |
| ------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| GitHub Models | No runtime or release-workflow call remains. GitHub fully retired the playground, catalog, inference API and BYOK on **2026-07-30**. [Official retirement notice](https://docs.github.com/en/github-models), checked 2026-11-11.                                                                                                                                                                                                 | Keep the retired endpoint removed; do not substitute a Copilot endpoint under the same name.   |
| Ollama        | Facet defaults to `http://localhost:11434`, model `llama3.2`, `/api/generate` and `/api/tags`, with `http://ollama:11434` as the discovery fallback. Both endpoints remain documented. [Generate](https://docs.ollama.com/api/generate) and [tags](https://docs.ollama.com/api/tags), checked 2026-11-11. Upstream's latest release is [v0.40.16](https://github.com/ollama/ollama/releases/tag/v0.40.16), published 2026-11-09. | The owner should compare their server with upstream; the repository selects no server version. |
| Stripe        | Facet selects stripe-go/v82 **82.5.1**, still the [latest v82 release](https://proxy.golang.org/github.com/stripe/stripe-go/v82/@latest). The [latest v87 release](https://proxy.golang.org/github.com/stripe/stripe-go/v87/@latest) is **87.1.0**. Stripe's [API versioning documentation](https://docs.stripe.com/api/versioning) lists `2026-10-28.endive` as current. All checked 2026-11-11.                                | Stay current on v82 or plan a tested major migration; v87 is not a drop-in patch.              |
| GitHub REST   | Facet uses `api.github.com` for imports and update checks, plus `raw.githubusercontent.com` for `LATEST_VERSION`. The selected `2022-11-28` API remains supported until **2028-03-10**; `2026-03-10` is current. [Official API versions](https://docs.github.com/en/rest/about-the-rest-api/api-versions), checked 2026-11-11.                                                                                                   | Review breaking changes and test all import/update paths before changing the header.           |
| SMTP          | Outgoing mail is owner-configured through PocketBase and enabled only when `SMTP_HOST` is set. Facet retains its local mailyak v3.6.2 timeout patch; no hosted SMTP provider or version is selected.                                                                                                                                                                                                                             | Preserve the timeout patch during PocketBase or transport upgrades.                            |

The maintainer security-advisory pages for [Ollama](https://github.com/ollama/ollama/security/advisories) and [stripe-go](https://github.com/stripe/stripe-go/security/advisories) showed no published advisories when checked on 2026-11-11. This is a point-in-time check of those two repositories, not a general vulnerability clearance.

## Detailed notes

- `anthropic.md` — Messages API, Claude presets and retirement history.
- `openai.md` — OpenAI-compatible Chat Completions, shutdown schedule and GPT-6 Luna parameters.
- `github-models.md` — retired inference service and its removal from release automation.
- `ollama.md` — owner-administered local model API.
- `stripe.md` — stripe-go v82 and newer API majors.
- `github-api.md` — repository metadata used for project imports.
- `smtp.md` — owner-configured SMTP transport.

Document-conversion packages are covered by `docker.md`, `pandoc.md`, `latex.md` and `libheif.md`. `backend-dependencies.md` and `security-advisories.md` record Facet's local SMTP transport patch.

## Verification limits

- This was a static repository review and an unauthenticated check of public official documentation, registries, releases and maintainer advisory pages. No external account, deployed provider configuration, provider usage/deprecation notice, local Ollama server, SMTP server or live API response was inspected.
- Public model documentation does not prove that a particular API key can access a model. Facet's Models API discovery and provider account notices remain necessary for account-specific status.
- Older model IDs in `backend/services/testutil.go` are labels in mock responses, not runtime presets or provider calls. Historical retirement details in the linked notes describe the old baseline.
- No compatibility or live integration tests were run for this documentation-only update. The Anthropic `2026-10-21`, GitHub `2026-03-10` and stripe-go v87 migrations were identified, not assessed as drop-in changes.
- The security statement above covers only the Ollama and stripe-go maintainer advisory pages. It does not enumerate every possible notice for Anthropic, OpenAI, GitHub, Stripe, Ollama or owner-configured SMTP providers.
