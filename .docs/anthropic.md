# Anthropic API

Checked **2026-11-11**.

Facet calls `https://api.anthropic.com/v1/messages` with `anthropic-version: 2023-06-01`. Current model discovery calls `/v1/models?limit=100`, allowing the provider to return models available to the configured key instead of relying only on a static list.

Anthropic's current API version is **2026-10-21**. Its version history announces no removal date for 2023-06-01, and Facet does not send the `temperature`, `top_p` or `top_k` parameters removed by the newer version. Review and test an API-version migration; there is no announced shutdown forcing an immediate change.

## Current main

[PR #531](https://github.com/jesposito/Facet/pull/531), merged as `74c3e31` and released in v3.8.6, includes the replacement presets first introduced in `a11b7ea`. The default is **`claude-sonnet-5-5`**. Fallback options are **`claude-sonnet-5-5`**, **`claude-opus-5-5`** and **`claude-haiku-5-5`**. Anthropic's current model overview lists all three in its current lineup and its deprecation page marks them active. The response parser handles thinking blocks by extracting text blocks. `Facet-c7g` is closed.

The lifecycle page lists tentative retirement dates no earlier than 2027-09-22 for Opus 5.5, 2027-09-28 for Sonnet 5.5 and 2027-10-07 for Haiku 5.5. These dates apply to Anthropic-operated platforms; partner-operated platforms can differ. Keep checking the official lifecycle page and Models API rather than treating the static fallbacks as permanent.

## Retired IDs in the baseline

The `f3d7a14` baseline offered only the following retired model IDs on Anthropic-operated platforms. Requests to retired models fail.

| Baseline model ID            | Retirement date | Official recommended replacement at inventory time |
| ---------------------------- | --------------- | -------------------------------------------------- |
| `claude-sonnet-4-20250514`   | 2026-06-15      | `claude-sonnet-5-5`                                |
| `claude-opus-4-20250514`     | 2026-06-15      | `claude-opus-4-8`                                  |
| `claude-3-5-sonnet-20241022` | 2025-10-28      | `claude-sonnet-5-5`                                |
| `claude-3-5-haiku-20241022`  | 2026-02-19      | `claude-haiku-4-5-20251001`                        |
| `claude-3-opus-20240229`     | 2026-01-05      | `claude-opus-4-8`                                  |
| `claude-3-sonnet-20240229`   | 2025-07-21      | `claude-sonnet-5-5`                                |
| `claude-3-haiku-20240307`    | 2026-04-20      | `claude-haiku-4-5-20251001`                        |

Sources: [`backend/hooks/ai.go`](../backend/hooks/ai.go), [`backend/services/ai.go`](../backend/services/ai.go), [`frontend/src/routes/admin/settings/integrations/+page.svelte`](../frontend/src/routes/admin/settings/integrations/+page.svelte), [official model overview](https://platform.claude.com/docs/en/about-claude/models/overview), [official model deprecations](https://platform.claude.com/docs/en/about-claude/model-deprecations), [API version history](https://platform.claude.com/docs/en/api/versioning), [PR #531](https://github.com/jesposito/Facet/pull/531), all checked 2026-11-11.
