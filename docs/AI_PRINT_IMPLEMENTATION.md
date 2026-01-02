# AI Print Implementation Plan

**Status**: Research Complete, Ready for Implementation
**Last Updated**: 2026-01-01

This document captures the research findings and implementation plan for Phase 4.2: AI Print.

---

## Executive Summary

AI Print generates professional resumes/CVs from view data by:
1. Serializing view data to JSON
2. Sending to AI provider with resume-optimized prompt
3. Converting AI's markdown response to DOCX/PDF via Pandoc
4. Storing files in PocketBase and serving for download

---

## Research Findings

### 1. View Data Structure

The existing `/api/view/{slug}/data` endpoint (see `backend/hooks/view.go:124-347`) returns:

```json
{
  "id": "view_id",
  "slug": "recruiter",
  "name": "For Recruiters",
  "visibility": "public",
  "hero_headline": "Optional override",
  "hero_summary": "Optional override",
  "cta_text": "Contact Me",
  "cta_url": "mailto:...",
  "accent_color": "indigo",
  "profile": {
    "name": "John Doe",
    "headline": "Software Engineer",
    "location": "San Francisco, CA",
    "summary": "...",
    "contact_email": "...",
    "contact_links": [...],
    "avatar_url": "/api/files/...",
    "hero_image_url": "/api/files/..."
  },
  "sections": {
    "experience": [...],
    "projects": [...],
    "education": [...],
    "skills": [...],
    "certifications": [...]
  },
  "section_order": ["experience", "education", "skills", "projects"],
  "section_layouts": {"experience": "timeline", ...},
  "section_widths": {"experience": "full", ...}
}
```

**Key Insight**: We can reuse the existing view data fetching logic. The `collectExportData` function in `export.go` shows the pattern for gathering data.

### 2. PocketBase File Storage

From [PocketBase Go Filesystem docs](https://pocketbase.io/docs/go-filesystem/):

```go
// Create file from bytes (our use case - Pandoc output)
f, err := filesystem.NewFileFromBytes(pdfBytes, "resume.pdf")
if err != nil {
    return err
}

// Set on record and save
record.Set("file_field", f)
err = app.Save(record) // Handles storage, validation, cleanup
```

**Key Points**:
- Use `filesystem.NewFileFromBytes()` for generated files
- `app.Save(record)` handles validation and storage automatically
- Always `Close()` filesystem resources to prevent leaks

### 3. Pandoc Integration Options

| Option | Pros | Cons | Recommendation |
|--------|------|------|----------------|
| **A: Add to Dockerfile** | Simple, single binary | +200MB image, install complexity | **Recommended for v1** |
| **B: Sidecar container** | Clean separation, flexible | More complex, networking overhead | Future optimization |
| **C: Host binary** | Zero image impact | Not portable, user must install | Not recommended |

**Implementation (Option A):**

```dockerfile
# Add to alpine stage
RUN apk add --no-cache pandoc
# Or for full LaTeX support (larger):
# RUN apk add --no-cache pandoc texlive-full
```

**Go execution pattern:**

```go
import "os/exec"

func convertMarkdownToPDF(markdown string) ([]byte, error) {
    log.Printf("[AI-PRINT] Converting markdown to PDF, input length: %d", len(markdown))

    // Write markdown to temp file
    tmpIn, err := os.CreateTemp("", "resume-*.md")
    if err != nil {
        log.Printf("[AI-PRINT] Failed to create temp input file: %v", err)
        return nil, err
    }
    defer os.Remove(tmpIn.Name())

    if _, err := tmpIn.WriteString(markdown); err != nil {
        return nil, err
    }
    tmpIn.Close()

    // Output file
    tmpOut := tmpIn.Name() + ".pdf"
    defer os.Remove(tmpOut)

    // Run Pandoc
    cmd := exec.Command("pandoc",
        tmpIn.Name(),
        "-o", tmpOut,
        "--pdf-engine=xelatex",  // prefer xelatex for font support; fallback handled in code
        "-V", "geometry:margin=0.75in",
        "-V", "fontsize=11pt",
        "-V", "mainfont=Helvetica",
    )

    var stderr bytes.Buffer
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        log.Printf("[AI-PRINT] Pandoc failed: %v, stderr: %s", err, stderr.String())
        return nil, fmt.Errorf("pandoc conversion failed: %w", err)
    }

    log.Printf("[AI-PRINT] Pandoc succeeded, reading output file")
    return os.ReadFile(tmpOut)
}
```

### 4. Resume Prompt Template

Based on [prompt engineering best practices](https://www.promptmixer.dev/blog/7-best-practices-for-ai-prompt-engineering-in-2025) and our existing prompt style:

```go
func buildResumePrompt(viewData ViewData, config GenerationConfig) string {
    return fmt.Sprintf(`You are an expert resume writer. Generate a professional resume in Markdown format.

IMPORTANT WRITING STYLE RULES:
- Write like a human, not an AI. Be direct and natural.
- NEVER use em dashes (—). Use commas, periods, or "and" instead.
- NEVER use words like "delve", "leverage", "utilize", "spearheaded", "synergy", "cutting-edge"
- Use active voice and strong action verbs
- Be concise - every word should add value
- Quantify achievements where data is provided (don't invent numbers)

TARGET CONTEXT:
- Target Role: %s
- Style: %s (chronological/functional/hybrid)
- Length: %s (one-page/two-page/full)
- Emphasis: %s

PROFILE DATA:
%s

OUTPUT REQUIREMENTS:
1. Generate clean Markdown suitable for Pandoc conversion to PDF
2. Use standard resume sections: Contact, Summary, Experience, Education, Skills, Projects (if relevant)
3. For Experience entries, convert bullets to strong achievement statements
4. Prioritize content based on target role and emphasis areas
5. Keep within length constraint

Return ONLY the Markdown content. Do not include code blocks or explanations.`,
        config.TargetRole,
        config.Style,
        config.Length,
        strings.Join(config.Emphasis, ", "),
        viewDataToString(viewData),
    )
}
```

---

## Implementation Plan

### Phase 1: Schema & Backend Foundation

#### 1.1 Database Migration

Create `backend/migrations/1735600009_view_exports.go`:

```go
// New collection: view_exports
collection := core.NewBaseCollection("view_exports")
collection.Fields.Add(&core.RelationField{
    Name:         "view",
    CollectionId: viewsCollectionId,
    Required:     true,
})
collection.Fields.Add(&core.SelectField{
    Name:     "format",
    Values:   []string{"pdf", "docx"},
    Required: true,
})
collection.Fields.Add(&core.FileField{
    Name:      "file",
    MaxSize:   10485760, // 10MB
    MimeTypes: []string{"application/pdf", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
})
collection.Fields.Add(&core.RelationField{
    Name:         "ai_provider",
    CollectionId: aiProvidersCollectionId,
})
collection.Fields.Add(&core.DateField{Name: "generated_at"})
collection.Fields.Add(&core.JSONField{Name: "generation_config"})
collection.Fields.Add(&core.TextField{Name: "status"}) // pending, processing, completed, failed
collection.Fields.Add(&core.TextField{Name: "error_message"})
```

#### 1.2 API Endpoint

Add to `backend/hooks/ai.go` or new `backend/hooks/resume.go`:

```
POST /api/view/{slug}/generate
  Body: {
    "format": "pdf" | "docx",
    "target_role": "Software Engineer",
    "style": "chronological",
    "length": "two-page",
    "emphasis": ["leadership", "technical"]
  }
  Response: {
    "export_id": "abc123",
    "status": "processing"
  }

GET /api/view/{slug}/exports
  Response: [{
    "id": "abc123",
    "format": "pdf",
    "generated_at": "2026-01-01T12:00:00Z",
    "status": "completed",
    "download_url": "/api/files/view_exports/abc123/resume.pdf"
  }]
```

### Phase 2: AI Integration

#### 2.1 Resume Generation Service

```go
// backend/services/resume.go
type ResumeService struct {
    ai     *AIService
    app    *pocketbase.PocketBase
}

type GenerationConfig struct {
    TargetRole string   `json:"target_role"`
    Style      string   `json:"style"`      // chronological, functional, hybrid
    Length     string   `json:"length"`     // one-page, two-page, full
    Emphasis   []string `json:"emphasis"`
}

func (r *ResumeService) GenerateResume(
    ctx context.Context,
    provider *AIProvider,
    viewData *ViewData,
    config *GenerationConfig,
) ([]byte, error) {
    log.Printf("[AI-PRINT] Starting generation for view, config: %+v", config)

    // 1. Build prompt
    prompt := r.buildResumePrompt(viewData, config)
    log.Printf("[AI-PRINT] Prompt built, length: %d chars", len(prompt))

    // 2. Call AI
    markdown, err := r.ai.ImproveContent(ctx, provider, prompt)
    if err != nil {
        log.Printf("[AI-PRINT] AI call failed: %v", err)
        return nil, err
    }
    log.Printf("[AI-PRINT] AI returned markdown, length: %d chars", len(markdown))

    // 3. Validate markdown structure
    if !r.validateMarkdown(markdown) {
        log.Printf("[AI-PRINT] Markdown validation failed")
        return nil, errors.New("AI returned invalid markdown structure")
    }

    // 4. Convert to PDF/DOCX
    pdf, err := r.convertToPDF(markdown)
    if err != nil {
        log.Printf("[AI-PRINT] PDF conversion failed: %v", err)
        return nil, err
    }
    log.Printf("[AI-PRINT] PDF generated, size: %d bytes", len(pdf))

    return pdf, nil
}
```

### Phase 3: Docker Integration

#### 3.1 Update Dockerfile

```dockerfile
# Add to Stage 3: Final runtime image
FROM alpine:3.19

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    nodejs \
    caddy \
    pandoc           # ADD THIS
    # For full LaTeX (better PDF quality, but larger image):
    # texlive-xetex texmf-dist-latexextra
```

**Note**: Alpine's pandoc package is ~50MB. For full LaTeX support, add texlive packages (~200MB).

### Phase 4: Frontend UI

#### 4.1 Generation Modal Component

Add to view editor (`/admin/views/[id]/+page.svelte`):

```svelte
<script>
  let showGenerateModal = false;
  let generationConfig = {
    format: 'pdf',
    target_role: '',
    style: 'chronological',
    length: 'two-page',
    emphasis: []
  };
  let generating = false;

  async function generateResume() {
    generating = true;
    try {
      const response = await fetch(`/api/view/${view.slug}/generate`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: pb.authStore.token
        },
        body: JSON.stringify(generationConfig)
      });

      if (!response.ok) throw new Error('Generation failed');

      const { export_id } = await response.json();
      // Poll for completion or show in exports list
      toasts.add('success', 'Resume generation started');
      showGenerateModal = false;
      loadExports(); // Refresh exports list
    } catch (err) {
      toasts.add('error', err.message);
    } finally {
      generating = false;
    }
  }
</script>

{#if aiAvailable}
  <button on:click={() => showGenerateModal = true}>
    Generate Resume
  </button>
{/if}

{#if showGenerateModal}
  <Modal title="Generate Resume" on:close={() => showGenerateModal = false}>
    <form on:submit|preventDefault={generateResume}>
      <label>
        Format
        <select bind:value={generationConfig.format}>
          <option value="pdf">PDF</option>
          <option value="docx">Word Document</option>
        </select>
      </label>

      <label>
        Target Role (optional)
        <input type="text" bind:value={generationConfig.target_role}
               placeholder="e.g., Software Engineer at FAANG" />
      </label>

      <label>
        Style
        <select bind:value={generationConfig.style}>
          <option value="chronological">Chronological</option>
          <option value="functional">Functional</option>
          <option value="hybrid">Hybrid</option>
        </select>
      </label>

      <label>
        Length
        <select bind:value={generationConfig.length}>
          <option value="one-page">One Page</option>
          <option value="two-page">Two Pages</option>
          <option value="full">Full</option>
        </select>
      </label>

      <button type="submit" disabled={generating}>
        {generating ? 'Generating...' : 'Generate'}
      </button>
    </form>
  </Modal>
{/if}
```

---

## Error Handling Matrix

| Error | Detection | User Message | Log Level |
|-------|-----------|--------------|-----------|
| No AI provider configured | Check before showing button | "Configure an AI provider in Settings" | INFO |
| AI API timeout | Context deadline exceeded | "AI service timed out. Try again." | WARN |
| AI returned invalid JSON | Parse error | "AI response was malformed. Try again." | ERROR |
| Pandoc not installed | exec.LookPath fails | "PDF generation unavailable" | ERROR |
| Pandoc conversion failed | Non-zero exit | "Failed to generate PDF. Check format." | ERROR |
| File too large | Size check | "Generated file exceeds limit" | WARN |
| PocketBase save failed | Save error | "Failed to save file. Try again." | ERROR |

---

## Testing Checklist

Before marking complete:

- [ ] Unit tests for prompt generation
- [ ] Unit tests for markdown validation
- [ ] Integration test: AI call with mock provider
- [ ] Integration test: Pandoc conversion with sample markdown
- [ ] Integration test: Full flow with file storage
- [ ] E2E test: Generate resume from UI
- [ ] E2E test: Download generated file
- [ ] Error case: No AI provider
- [ ] Error case: AI timeout
- [ ] Error case: Pandoc failure
- [ ] Performance test: Large view with many sections

---

## Required Logging

All logging MUST use `[AI-PRINT]` prefix for easy filtering:

```go
log.Printf("[AI-PRINT] Starting resume generation for view: %s", viewID)
log.Printf("[AI-PRINT] View data size: %d bytes, sections: %d", len(json), sectionCount)
log.Printf("[AI-PRINT] AI prompt length: %d chars", len(prompt))
log.Printf("[AI-PRINT] AI response received, length: %d chars", len(response))
log.Printf("[AI-PRINT] Markdown validation: %s", status)
log.Printf("[AI-PRINT] Pandoc command: %v", cmdArgs)
log.Printf("[AI-PRINT] Pandoc completed, output size: %d bytes", len(output))
log.Printf("[AI-PRINT] File saved to PocketBase: %s", fileID)
```

---

## Implementation Order

1. **Week 1**: Schema migration + basic API endpoint (return mock data)
2. **Week 2**: AI integration + prompt template
3. **Week 3**: Pandoc integration + file storage
4. **Week 4**: Frontend UI + error handling
5. **Week 5**: Testing + documentation

---

## References

- [PocketBase File Handling](https://pocketbase.io/docs/files-handling/)
- [PocketBase Go Filesystem](https://pocketbase.io/docs/go-filesystem/)
- [Pandoc User's Guide](https://pandoc.org/MANUAL.html)
- [Pandoc Docker Images](https://hub.docker.com/r/pandoc/latex)
- [Prompt Engineering Best Practices 2025](https://www.promptmixer.dev/blog/7-best-practices-for-ai-prompt-engineering-in-2025)
- [Go exec Package](https://pkg.go.dev/os/exec)
