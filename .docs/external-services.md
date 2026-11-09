# External services overview

Checked **2026-11-10**. Each managed or owner-administered integration has its own note:

- `anthropic.md` — Messages API and retired Claude presets; immediate migration required.
- `openai.md` — OpenAI-compatible Chat Completions, shutdown schedule and GPT-6 Luna parameter requirements.
- `github-models.md` — retired inference service still referenced by release automation.
- `ollama.md` — owner-administered local model API.
- `stripe.md` — stripe-go v82 and newer API majors.
- `github-api.md` — repository metadata used for project imports.

SMTP transport is owner-configured through PocketBase; `backend-dependencies.md` documents Facet's local mailyak timeout patch. Document-conversion packages are covered by `docker.md`, `pandoc.md`, `latex.md` and `libheif.md`.

No external account, deployed provider configuration or live API call was inspected for this inventory. Sources and dates are recorded in each linked note.
