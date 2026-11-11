# GitHub Models

Checked **2026-11-11**.

**GitHub Models was fully retired on 2026-07-30.** GitHub's official documentation says the playground, model catalog, inference API and bring-your-own-key support are no longer available.

The baseline `f3d7a14` release workflow attempted `https://models.inference.ai.azure.com/chat/completions` with `gpt-4o-mini` before trying OpenAI and deterministic keyword categorisation. [PR #531](https://github.com/jesposito/Facet/pull/531), merged as `74c3e31` and released in v3.8.6, removed that retired call. Changelog generation now tries OpenAI when a key is available and otherwise uses keywords, while retaining validation of returned PR numbers. `Facet-c7g` is closed.

GitHub Models was separate from GitHub Copilot. The retired endpoint should not be restored under a Copilot label.

Sources: [`.github/workflows/auto-tag.yml`](../.github/workflows/auto-tag.yml), [official GitHub Models retirement notice](https://docs.github.com/en/github-models), [PR #531](https://github.com/jesposito/Facet/pull/531), baseline diff and current workflow inspected 2026-11-11.
