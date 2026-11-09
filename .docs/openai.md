# OpenAI and compatible APIs

Checked **2026-11-10**.

Facet uses the chat-completions shape at `https://api.openai.com/v1/chat/completions` by default and allows a custom base URL.

## Current migration branch

The current branch selects **`gpt-6-luna`** for the application default, settings fallback and release changelog workflow. Model discovery ranks `gpt-6` IDs first. The retired GitHub Models attempt has been removed; release automation tries OpenAI when a key exists and otherwise uses deterministic keyword categorisation.

For `gpt-6-luna`, commit `f761697` explicitly sets `reasoning_effort: "none"` and retains `temperature: 0.7`; bounded requests use `max_completion_tokens`. Other recognised reasoning-model prefixes (`gpt-5`, `gpt-6`, `o1`, `o3`, `o4`) omit temperature and use `max_completion_tokens`. Custom non-reasoning models retain `temperature: 0.7` and legacy `max_tokens` for OpenAI-compatible providers. The release workflow sets `reasoning_effort: "none"`, `temperature: 0.3` and `max_completion_tokens: 2000`. Empty responses now produce an explicit error, including when the token limit is reached before an answer.

## Baseline deprecation findings

The `f3d7a14` baseline used `gpt-4o-mini` as the application fallback and `gpt-4o` for automatic environment configuration. Its settings UI listed `gpt-4o`, `gpt-4o-mini`, `gpt-4-turbo`, `gpt-3.5-turbo`, `o1` and `o1-mini`.

OpenAI's official deprecation schedule reports:

- `gpt-4-turbo`, `gpt-3.5-turbo` and `o1` shut down on **2026-10-23**;
- `o1-mini` shut down on **2025-10-27**;
- dated GPT-4o and GPT-4o-mini snapshots are scheduled for **2027-01-14**, with `gpt-6-luna` named as replacement.

The baseline used aliases rather than dated GPT-4o snapshots. An authenticated Models API remains the way to verify an alias's target and `shutdown_date`; the deprecation page alone does not prove when a particular alias stops resolving. `Facet-c7g` tracks the migration.

## GPT-6 Luna request compatibility

OpenAI's model page supports Chat Completions and `reasoning_effort` values `none`, `low`, `medium` (default), `high`, `xhigh` and `max`.

- At the default `medium` effort, **omit `temperature`**, `top_p`, `top_logprobs` and Chat Completions `logprobs`.
- Temperature may be retained when explicitly using `reasoning_effort: "none"`; the migration guide restricts the removal requirement to efforts other than `none`.
- The Chat Completions reference marks **`max_tokens` deprecated in favour of `max_completion_tokens`**. The newer parameter includes visible output and reasoning tokens.
- OpenAI recommends the Responses API for new work. GPT-6 Luna remains available through Chat Completions; function calling there requires `reasoning_effort: "none"`.

Sources: [`backend/services/ai.go`](../backend/services/ai.go), [`backend/hooks/ai.go`](../backend/hooks/ai.go), [`frontend/src/routes/admin/settings/integrations/+page.svelte`](../frontend/src/routes/admin/settings/integrations/+page.svelte), [`.github/workflows/auto-tag.yml`](../.github/workflows/auto-tag.yml), [official OpenAI deprecations](https://platform.openai.com/docs/deprecations), [GPT-6 Luna model](https://developers.openai.com/api/docs/models/gpt-6-luna), [GPT-6 migration parameters](https://developers.openai.com/api/docs/guides/latest-model#update-api-and-model-parameters), [Chat Completions reference](https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create/), all checked 2026-11-10.
