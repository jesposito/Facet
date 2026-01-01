# AI Features Documentation

**Last Updated:** 2026-01-01

This document provides comprehensive documentation for all AI-powered features in Facet. These features are optional and require configuration of at least one AI provider.

---

## Table of Contents

1. [Overview](#overview)
2. [AI Provider Configuration](#ai-provider-configuration)
3. [AI Print (Resume Generation)](#ai-print-resume-generation)
4. [AI Project Enrichment](#ai-project-enrichment)
5. [AI Content Improvement](#ai-content-improvement)
6. [API Reference](#api-reference)
7. [Environment Variables](#environment-variables)
8. [Troubleshooting](#troubleshooting)

---

## Overview

Facet includes optional AI-powered features that enhance content creation and resume generation. All AI features:

- Are **completely optional** - Facet works fully without any AI configuration
- Support **multiple providers** - OpenAI, Anthropic Claude, Ollama (local), or custom OpenAI-compatible APIs
- Use **encrypted API key storage** - Keys are encrypted at rest using AES-256-GCM
- Follow **privacy-conscious prompts** - AI prompts emphasize factual content without inventing information

### Supported AI Providers

| Provider | Type | API Key Required | Custom Model | Custom URL |
|----------|------|------------------|--------------|------------|
| OpenAI | `openai` | Yes | Optional | Optional |
| Anthropic Claude | `anthropic` | Yes | Optional | No |
| Ollama (local) | `ollama` | No | Optional | Required |
| Custom (OpenAI-compatible) | `custom` | Varies | Required | Required |

---

## AI Provider Configuration

### Admin UI Configuration

Navigate to **Admin > Settings > AI Providers** to configure providers through the UI.

**Adding a Provider:**

1. Click "Add Provider"
2. Fill in the required fields:
   - **Name**: Display name (e.g., "OpenAI GPT-4o")
   - **Type**: Select from openai, anthropic, ollama, or custom
   - **API Key**: Your provider's API key (encrypted before storage)
   - **Model**: Select from presets or enter custom model name
   - **Base URL**: Required for Ollama and custom providers
3. Click "Add Provider"
4. Optionally test the connection with "Test Connection"
5. Set as default if this should be the primary provider

**Model Presets by Provider:**

| Provider | Available Models |
|----------|------------------|
| OpenAI | gpt-4o, gpt-4o-mini, gpt-4-turbo, gpt-3.5-turbo, o1, o1-mini |
| Anthropic | claude-sonnet-4-20250514, claude-opus-4-20250514, claude-3-5-sonnet-20241022, claude-3-5-haiku-20241022 |
| Ollama | llama3.2, llama3.1, mistral, codellama, phi3 |
| Custom | (user-specified) |

### Auto-Configuration from Environment

Facet automatically creates AI providers from environment variables on first access if no providers exist:

```bash
# Anthropic (recommended)
ANTHROPIC_API_KEY=sk-ant-...

# OpenAI
OPENAI_API_KEY=sk-...

# Ollama (local)
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_MODEL=llama3.2  # Optional, defaults to llama3.2
```

When environment variables are detected, providers are created automatically with names like "Claude (Auto)" or "OpenAI (Auto)".

### API Key Security

- API keys are **never stored in plaintext**
- Keys are encrypted using AES-256-GCM before storage
- Encryption key is set via the `ENCRYPTION_KEY` environment variable
- The `api_key` field is marked as a "hidden" field in PocketBase (not returned in API responses)
- Only the `api_key_encrypted` field is stored

---

## AI Print (Resume Generation)

AI Print generates professionally formatted resumes (PDF or DOCX) from your view data using AI optimization.

### How It Works

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  View Data  │ ──▶ │   AI API    │ ──▶ │   Pandoc    │ ──▶ │  PDF/DOCX   │
│  (JSON)     │     │  (Optimize) │     │  (Convert)  │     │  (Download) │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
```

1. **Collect**: Gathers complete view data including profile, all sections, and item-level overrides
2. **Optimize**: Sends data to AI with resume-specific formatting prompts
3. **Convert**: AI returns optimized markdown, Pandoc converts to PDF or DOCX
4. **Download**: File is stored and download URL is returned

### Features

- **Target Role Optimization**: Uses the view's hero_headline to tailor content for the intended role
- **Resume Styles**: Chronological, Functional, or Hybrid layouts
- **Length Control**: One-page, Two-page, or Full resume options
- **Format Options**: PDF (via LaTeX) or DOCX (Microsoft Word)
- **View Overrides**: Respects item-level field overrides configured in the view editor
- **Public Access**: Recruiters can generate resumes from public/unlisted views
- **Rate Limiting**: Public users limited to 5 generations per hour per IP

### Accessing AI Print

**From Public View Pages:**
- Navigate to any public view (e.g., `/recruiter`)
- Click the Print button dropdown
- Select "AI Resume" to open the generation modal
- Target role is automatically set from the view's hero_headline

**From the Root Page:**
- Navigate to the root URL (`/`)
- Click the Print button dropdown
- Select "AI Resume" to open the generation modal
- Target role uses the default view's hero_headline or profile headline

### Generation Modal Options (Public Pages)

| Option | Values | Description |
|--------|--------|-------------|
| Format | PDF, DOCX | Output file format |
| Style | Chronological, Functional, Hybrid | Resume organization style |
| Length | One Page, Two Pages, Full | Resume length constraint |

> **Note:** On public pages, the target role is automatically set from the view's `hero_headline` (configured by the profile owner in the view editor). This ensures recruiters see the resume tailored to the role the profile owner intended, without the ability to override it.

### Requirements

AI Print requires:
- At least one active AI provider configured
- Pandoc installed (included in Docker image)
- For PDF: LaTeX (pdflatex) with texlive packages

### AI Writing Style

The AI prompt enforces professional writing standards:
- Direct, human-like writing (not AI-sounding)
- No em dashes, corporate buzzwords, or marketing speak
- Action verbs for achievements (led, built, designed, implemented)
- Quantified achievements where data is provided
- Focus on impact and results, not just responsibilities

---

## AI Project Enrichment

During GitHub import, AI can enrich project data with professional summaries and highlights.

### What Gets Generated

| Field | Description |
|-------|-------------|
| `summary` | 2-3 sentence professional project summary |
| `bullets` | 3-5 key features or achievements |
| `tags` | Relevant technology/category tags |
| `tech_highlights` | 2-3 notable technical aspects |
| `case_study` | Brief case study outline (3-4 bullet points) |

### Privacy Modes

| Mode | Data Sent to AI |
|------|-----------------|
| `full` | Title, description, README (up to 10,000 chars), languages, topics |
| `summary` | Title, description, README (first 500 chars), languages, topics |
| `none` | No AI enrichment performed |

### Usage

1. Import a project from GitHub
2. During the review step, enable AI enrichment
3. Select a privacy mode
4. AI generates enriched content for review before saving

---

## AI Content Improvement

General-purpose content improvement available throughout the admin interface.

### Content Types Supported

| Type | Output Format |
|------|---------------|
| `headline` | Single line, under 100 characters |
| `summary` | 2-4 sentences |
| `description` | 2-3 paragraphs |
| `bullets` | JSON array of 3-5 strings |
| `experience` | JSON with description + bullets |
| `project` | JSON with summary + description |
| `education` | Brief description paragraph |

### Actions

| Action | Behavior |
|--------|----------|
| `improve` | Enhance existing content to be more professional |
| `generate` | Create new content based on context provided |
| `expand` | Add more detail while maintaining accuracy |
| `shorten` | Condense content while keeping key points |

### API Usage

```bash
POST /api/ai/improve
Authorization: <your-auth-token>
Content-Type: application/json

{
  "content_type": "summary",
  "content": "I am a developer who makes things.",
  "context": {
    "name": "Jane Doe",
    "title": "Software Engineer"
  },
  "action": "improve"
}
```

---

## API Reference

### Status Endpoints

#### Check AI Availability
```
GET /api/ai/status
```
Returns whether AI features are available and the default provider.

**Response:**
```json
{
  "available": true,
  "provider_count": 1,
  "default_provider": {
    "id": "abc123",
    "name": "OpenAI GPT-4o",
    "type": "openai",
    "model": "gpt-4o-mini"
  }
}
```

#### Check AI Print Availability
```
GET /api/ai-print/status
```
Returns whether AI Print (resume generation) is available.

**Response:**
```json
{
  "available": true,
  "pandoc_installed": true,
  "ai_configured": true,
  "supported_formats": ["pdf", "docx"]
}
```

### Provider Management

#### Test Provider Connection
```
POST /api/ai/test/{id}
Authorization: required
```
Tests connectivity to an AI provider.

**Response:**
```json
{
  "success": true
}
```
or
```json
{
  "success": false,
  "error": "Invalid API key"
}
```

### Content Generation

#### Improve Content
```
POST /api/ai/improve
Authorization: required
```
Improves or generates content using AI.

**Request Body:**
```json
{
  "content_type": "summary|headline|description|bullets|experience|project|education",
  "content": "existing content to improve",
  "context": {
    "name": "...",
    "title": "...",
    "company": "..."
  },
  "action": "improve|generate|expand|shorten",
  "provider_id": "optional-provider-id"
}
```

#### Enrich Project
```
POST /api/ai/enrich
Authorization: required
```
Enriches project data from GitHub import.

**Request Body:**
```json
{
  "provider_id": "optional",
  "title": "Project Name",
  "description": "Short description",
  "readme": "Full README content",
  "languages": {"Go": 5000, "TypeScript": 3000},
  "topics": ["backend", "api"],
  "privacy_mode": "full|summary|none"
}
```

### Resume Generation

#### Generate Resume
```
POST /api/view/{slug}/generate
Authorization: optional (public for public/unlisted views)
Rate Limit: 5/hour per IP for unauthenticated users
```
Generates a resume from view data.

**Request Body:**
```json
{
  "format": "pdf|docx",
  "target_role": "Senior Software Engineer",
  "style": "chronological|functional|hybrid",
  "length": "one-page|two-page|full",
  "emphasis": ["leadership", "technical"],
  "provider_id": "optional"
}
```

**Response:**
```json
{
  "export_id": "abc123",
  "status": "completed",
  "format": "pdf",
  "download_url": "/api/files/view_exports/abc123/resume.pdf",
  "generated_at": "2026-01-01T12:00:00Z"
}
```

#### List Exports
```
GET /api/view/{slug}/exports
Authorization: required
```
Lists all generated exports for a view.

#### Delete Export
```
DELETE /api/view/{slug}/exports/{exportId}
Authorization: required
```
Deletes a generated export.

---

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `ENCRYPTION_KEY` | 32-byte key for API key encryption | Yes |
| `ANTHROPIC_API_KEY` | Auto-configures Anthropic Claude | No |
| `OPENAI_API_KEY` | Auto-configures OpenAI | No |
| `OLLAMA_BASE_URL` | Auto-configures Ollama | No |
| `OLLAMA_MODEL` | Model for Ollama (default: llama3.2) | No |

### Generating an Encryption Key

```bash
# Generate a secure 32-byte key
openssl rand -base64 32

# Or using Go
go run -e 'import "crypto/rand"; import "encoding/base64"; b := make([]byte, 32); rand.Read(b); fmt.Println(base64.StdEncoding.EncodeToString(b))'
```

---

## Troubleshooting

### "No AI provider configured"

**Cause:** No active AI providers exist in the database.

**Solution:**
1. Go to Admin > Settings > AI Providers
2. Add a provider with your API key
3. Ensure the provider is marked as "Active"

Or set environment variables for auto-configuration:
```bash
ANTHROPIC_API_KEY=sk-ant-your-key
```

### "PDF generation requires pdflatex"

**Cause:** LaTeX is not installed in the container.

**Solution:**
- Rebuild the container - the Dockerfile includes texlive packages
- Or use DOCX format instead (doesn't require LaTeX)

### "PDF generation requires LaTeX package 'lmodern.sty'"

**Cause:** The lmodern LaTeX package is not installed.

**Solution:**
1. Rebuild your container (Dockerfile includes lmodern package)
2. Or use DOCX format as an alternative

### "Connection test failed"

**Cause:** Unable to connect to the AI provider.

**Possible Solutions:**
1. Verify API key is correct
2. Check network connectivity
3. For Ollama: ensure the service is running and URL is correct
4. For custom providers: verify the base URL is correct

### "Rate limit exceeded"

**Cause:** Too many resume generations from the same IP address.

**Solution:**
- Wait for the rate limit window to reset (1 hour)
- Authenticated users bypass rate limiting
- Rate limit is 5 generations per hour per IP

### AI returns poorly formatted content

**Cause:** AI model quality or prompt interpretation issues.

**Solutions:**
1. Try a different model (e.g., gpt-4o instead of gpt-4o-mini)
2. Use Anthropic Claude for better formatting
3. Retry generation - results can vary

---

## Database Schema

### ai_providers Collection

| Field | Type | Description |
|-------|------|-------------|
| `name` | text | Display name |
| `type` | select | openai, anthropic, ollama, custom |
| `api_key` | text (hidden) | Plaintext key (cleared after encryption) |
| `api_key_encrypted` | text | Encrypted API key |
| `base_url` | url | Custom API endpoint |
| `model` | text | Model identifier |
| `is_active` | bool | Provider is available for use |
| `is_default` | bool | Default provider for AI operations |
| `test_status` | text | Last connection test result |
| `last_test` | date | Timestamp of last test |

### view_exports Collection

| Field | Type | Description |
|-------|------|-------------|
| `view` | relation | Link to views collection |
| `format` | select | pdf, docx |
| `file` | file | Generated document file |
| `ai_provider` | relation | Provider used for generation |
| `generated_at` | date | Generation timestamp |
| `generation_config` | json | Target role, style, length, emphasis |
| `status` | select | pending, processing, completed, failed |
| `error_message` | text | Error details if failed |

---

## Security Considerations

1. **API Keys**: Encrypted at rest using AES-256-GCM
2. **Authentication**: Most AI endpoints require authentication
3. **Rate Limiting**: Public resume generation is rate-limited
4. **Data Privacy**: AI prompts explicitly instruct not to invent information
5. **View Visibility**: Resume generation respects view visibility settings

---

*For questions or issues, see the main [README](../README.md) or open an issue on GitHub.*
