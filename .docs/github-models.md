# GitHub Models

Checked **2026-11-10**.

**GitHub Models was fully retired on 2026-07-30.** GitHub's official documentation says the playground, model catalog, inference API and bring-your-own-key support are no longer available.

The baseline `f3d7a14` release workflow attempted `https://models.inference.ai.azure.com/chat/completions` with `gpt-4o-mini` before trying OpenAI and deterministic keyword categorisation. The current migration branch removes that retired call. Changelog generation now tries OpenAI when a key is available and otherwise uses keywords, while retaining validation of returned PR numbers.

GitHub Models was separate from GitHub Copilot. The retired endpoint should not be restored under a Copilot label. Follow-up for the remaining release-automation migration: `Facet-c7g`.

Sources: [`.github/workflows/auto-tag.yml`](../.github/workflows/auto-tag.yml), [official GitHub Models retirement notice](https://docs.github.com/en/github-models), baseline diff inspected 2026-11-10.
