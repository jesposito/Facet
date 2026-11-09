# AI features

Facet can use an AI model to help you write, to turn an existing résumé into structured content, to summarise GitHub projects, and to produce tailored résumés from a view. All of it is optional. Without an AI provider, the AI buttons simply don't appear and everything else works the same.

Self-hosted Facet is **bring your own provider**. You connect your own Anthropic, OpenAI or OpenAI-compatible account, or a local Ollama server. Facet has no AI service of its own, no credits and no quotas. Requests go straight from your server to the provider you chose, and they're billed to your account with that provider.

## Contents

- [Connect a provider](#connect-a-provider)
- [Writing assistant](#writing-assistant)
- [Résumé import](#résumé-import)
- [GitHub project enrichment](#github-project-enrichment)
- [AI résumé generation (AI Print)](#ai-résumé-generation-ai-print)
- [What gets sent to the provider](#what-gets-sent-to-the-provider)
- [Troubleshooting](#troubleshooting)
- [API reference](#api-reference)

## Connect a provider

Go to **Settings → Integrations** and add a provider.

| Type | Needs an API key | Base URL | Notes |
|------|------------------|----------|-------|
| `anthropic` | Yes | No | Claude models |
| `openai` | Yes | Optional | OpenAI models |
| `ollama` | No | Yes, e.g. `http://192.168.1.20:11434` | Runs models on your own hardware, so nothing leaves your network |
| `custom` | Usually | Yes | Any service with an OpenAI-compatible API |

The form suggests a few model names for each type. Model names change often, so use **Fetch models** to load the list your account actually offers, or type a model name yourself. **Test connection** sends a short request to confirm everything works. If you add more than one provider, mark one as the default.

Keys are encrypted before they're saved and are never sent back to the browser ([Security](SECURITY.md#whats-encrypted)).

**Ollama in Docker:** inside the Facet container, `localhost` means the container itself, not your computer. Point the base URL at your machine's network address, or at `http://host.docker.internal:11434` on Docker Desktop.

### Setting a provider with environment variables

As a shortcut, `ANTHROPIC_API_KEY`, `OPENAI_API_KEY` or `OLLAMA_BASE_URL` (with optional `OLLAMA_MODEL`) create a provider automatically the first time the admin checks AI status, as long as no provider exists yet ([source](../backend/hooks/ai.go)). They're named "Claude (Auto)", "OpenAI (Auto)" and "Ollama (Auto)". Anthropic gets `claude-sonnet-5-5` and OpenAI gets `gpt-6-luna`, which you can change in the dashboard.

After that first time the variables aren't read again, so edit the provider in the dashboard rather than the variable. The repository's `docker-compose.yml` already passes these AI variables through. Other Compose files, including the minimal installation example, need any missing names under `environment:` ([details](CONFIGURATION.md#does-the-variable-reach-the-container)).

## Writing assistant

An AI button sits beside the main text fields on the **Experience**, **Projects**, **Education**, **Posts**, **Talks**, **Custom Content**, **Courses** and **Homepage** pages. It uses the rest of the form (job title, company, technologies and so on) as context.

It has two modes.

**Rewrite** in one of five tones:

| Tone | Sounds like |
|------|-------------|
| Executive | Direct and results-focused, for senior audiences |
| Professional | Polished and businesslike. The default. |
| Technical | Precise and detailed, for specialists |
| Conversational | Friendly and personal |
| Creative | Engaging, with more storytelling |

You see the suggestion before anything changes, and you can accept, edit or discard it.

**Critique** leaves your text alone and returns it with bracketed notes where it's weak, for example `[Quantify this: how much faster?]` or `[Avoid "leverage": say what you actually did]`. It's for when you'd rather improve your own writing than have it replaced.

The prompts ask the model to write plainly: no em dashes, none of the usual AI vocabulary ("delve", "leverage", "utilize", "synergy" and friends), active verbs, and no invented facts or numbers. Models don't always comply, so read what comes back.

## Résumé import

Under **Import & AI**, upload a résumé as a PDF or Word (`.docx`) file, up to 10 MB ([source](../backend/hooks/resume_upload.go)). Facet extracts the text on your server, sends it to your AI provider, and turns the reply into:

- experience (company, title, dates, description)
- education
- skills, grouped into categories
- projects
- certifications
- awards and talks

You review what was found before importing, and Facet skips entries that duplicate ones you already have, reporting how many it skipped ([source](../backend/hooks/resume_upload.go)).

Scanned PDFs (pictures of text) don't work, because there's no text to extract.

## GitHub project enrichment

When you import a GitHub repository as a project, Facet can ask your AI provider for a summary, 3–5 highlights, tags, technical highlights and a short case-study outline. You choose how much to send ([source](../backend/services/ai.go)):

| Privacy mode | Sent to the provider |
|--------------|----------------------|
| Full | Name, description, languages, topics, and the README up to 10,000 bytes |
| Summary | The same, but only the start of the README |
| None | Nothing. No AI enrichment. |

Everything lands in a proposal for you to review before it's saved, and fields you've locked are never overwritten.

## AI résumé generation (AI Print)

The print menu on your public pages offers an **AI résumé**. Facet sends the view's content to your AI provider with résumé-writing instructions, then converts the result to PDF or Word with Pandoc ([source](../backend/services/resume.go)).

| Option | Choices |
|--------|---------|
| Format | PDF or DOCX |
| Style | Chronological, functional or hybrid |
| Length | One page, two pages, or full |

The target role comes from the view's hero headline, so a recruiter view produces a résumé aimed at the role you chose. Item overrides in the view are respected. Generated files are kept, and you can list and delete them for each view.

**Visitors can use this too.** Anyone who can open a view can generate a résumé from it, and each generation is a request on **your** AI account. The endpoint applies the same access check as the view itself, so an unlisted view needs its share link and a password view needs its password ([source](../backend/hooks/resume.go)). Anonymous use is limited to 5 generations an hour per client address ([client address](../backend/services/clientip.go)). If that cost matters to you, leave AI providers unconfigured until you need them.

PDF output needs Pandoc and LaTeX, which the Docker image includes. On a development machine without them, the option reports itself as unavailable.

## What gets sent to the provider

| Feature | Data sent |
|---------|-----------|
| Writing assistant | The text you're editing, plus related fields from the same form |
| Résumé import | The full text extracted from your résumé |
| GitHub enrichment | Repository details and README, according to the privacy mode |
| AI Print | The view's content: profile, and every section and item the view shows |

Each provider handles that data under its own terms. Ollama keeps it on your own hardware.

## Troubleshooting

**The AI buttons don't appear.** No provider is active. Add one in **Settings → Integrations** and make sure it's switched on.

**"Connection test failed".** Check the key, the model name, and for Ollama or custom providers the base URL. From inside Docker, `localhost` won't reach a service on your host.

**PDF generation fails.** Check the container log for lines starting with `[AI-PRINT]`. Facet retries with a second LaTeX engine before giving up. DOCX is a useful fallback.

**Rate limit messages.** Either your provider is throttling you (see its dashboard), or anonymous visitors have used the 5 hourly AI Print generations.

**Odd or poorly formatted text.** Smaller local models struggle with the structured replies Facet asks for. Try a larger model, then edit the result.

## API reference

All endpoints need a signed-in owner, except the two status checks and AI Print generation.

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/ai/status` | Whether an active provider exists (also creates one from environment variables if none exist) |
| `POST` | `/api/ai/models` | List models available from a provider |
| `POST` | `/api/ai/test/{id}` | Test a saved provider |
| `POST` | `/api/ai/rewrite` | Writing assistant rewrite |
| `POST` | `/api/ai/critique` | Writing assistant critique |
| `POST` | `/api/ai/improve` | Improve, generate, expand or shorten a field |
| `POST` | `/api/ai/enrich` | GitHub project enrichment |
| `POST` | `/api/resume/upload`, `/api/resume/import` | Résumé import |
| `GET` | `/api/ai-print/status` | Whether AI Print is available (Pandoc and a provider) |
| `POST` | `/api/view/{slug}/generate` | Generate an AI résumé (public, rate limited) |
| `GET` | `/api/view/{slug}/exports` | List generated résumés for a view |
| `DELETE` | `/api/view/{slug}/exports/{exportId}` | Delete a generated résumé |

For working on the writing-assistant component itself, see the [development guide](DEV.md#ai-writing-assistant-component).
