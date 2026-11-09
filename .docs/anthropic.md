# Anthropic API

Checked **2026-11-10**.

Facet calls `https://api.anthropic.com/v1/messages` with `anthropic-version: 2023-06-01`. Current model discovery calls `/v1/models?limit=100`, allowing the provider to return models available to the configured key instead of relying only on a static list.

## Current migration branch

Commit `a11b7ea` replaces the retired presets. The default is **`claude-sonnet-5-5`**. Fallback options are **`claude-sonnet-5-5`**, **`claude-opus-5-5`** and **`claude-haiku-5-5`**. Anthropic's current model overview lists all three in its current lineup; its deprecation page marks them active. The response parser handles thinking blocks by extracting text blocks.

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

Partner-operated platforms can have different retirement schedules. Keep checking the official lifecycle page and the Models API rather than treating the new static fallbacks as permanent. `Facet-c7g` tracks the wider AI and release-automation migration.

Sources: [`backend/hooks/ai.go`](../backend/hooks/ai.go), [`backend/services/ai.go`](../backend/services/ai.go), [`frontend/src/routes/admin/settings/integrations/+page.svelte`](../frontend/src/routes/admin/settings/integrations/+page.svelte), [official model overview](https://docs.anthropic.com/en/docs/about-claude/models/overview), [official model deprecations](https://docs.anthropic.com/en/docs/about-claude/model-deprecations), all checked 2026-11-10.
