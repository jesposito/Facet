# Resume Upload & Parsing Design

**Status**: Proposed
**Date**: 2026-01-04
**Goal**: Reverse of resume generation - upload PDF/DOCX resume and extract data into Facet profile

---

## Problem Statement

Users have existing resumes in PDF or DOCX format. Manually copying data into Facet is tedious and error-prone. We need an intelligent import system that:

1. Extracts text from uploaded resumes
2. Uses AI to parse unstructured resume data into Facet's structured schema
3. **Creates a NEW VIEW** with parsed data (keeps existing profile/views safe)
4. Provides a review/edit workflow before publishing
5. Handles edge cases (ambiguous data, formatting variations, etc.)

## Key Design Decision

**Resume uploads create a NEW VIEW, not modify existing profile/views.**

This approach:
- ✅ Keeps default profile and other views completely safe
- ✅ Allows user to review before publishing
- ✅ User can later merge content if desired
- ✅ View starts as private/draft by default
- ✅ Simpler workflow: upload → review → publish

---

## Architecture Overview

```
┌─────────────┐     ┌──────────────┐     ┌───────────────┐     ┌──────────────┐
│  User       │────▶│  Upload      │────▶│  AI Parser    │────▶│  Create      │
│  Uploads    │     │  PDF/DOCX    │     │  Extract Data │     │  NEW VIEW    │
│  Resume     │     │  Extract Text│     │  to JSON      │     │  (Draft)     │
└─────────────┘     └──────────────┘     └───────────────┘     └──────────────┘
                                                                       │
                                                                       ▼
                                                                ┌──────────────┐
                                                                │  Review/Edit │
                                                                │  View in     │
                                                                │  Admin UI    │
                                                                └──────────────┘
                                                                       │
                                                                       ▼
                                                                ┌──────────────┐
                                                                │  User        │
                                                                │  Publishes   │
                                                                │  View        │
                                                                └──────────────┘
```

**Flow Details**:
1. User uploads resume PDF/DOCX
2. Backend extracts text and parses with AI
3. Backend **creates a new VIEW** called "Resume Import - [date]"
4. All parsed data is added as NEW RECORDS linked to this view
5. View is initially `visibility="private"` and `is_active=false`
6. User reviews view at `/admin/views/[new-view-id]`
7. User can edit any section directly in the view editor
8. When ready, user activates and publishes the view

---

## Library Selection

### PDF Text Extraction
**Choice**: `go-fitz` (wrapper around MuPDF)
- **Pros**: Mature, handles complex PDFs, extracts both text and layout
- **Cons**: CGo dependency (but acceptable for server deployment)
- **Alternative**: `pdfcpu` (pure Go, but less robust for text extraction)

### DOCX Text Extraction
**Choice**: `go-docx` (fumiama/go-docx) - Open Source
- **Pros**: Free, reads .docx well, extracts text and formatting
- **Cons**: Less feature-complete than UniOffice
- **Alternative**: `unioffice` (commercial, free tier available)
- **Fallback**: `docxlib` (simpler, good for basic extraction)

---

## Data Flow

### 1. Upload Endpoint

**POST `/api/resume/upload`**

```go
type ResumeUploadRequest struct {
    File         multipart.File
    ProviderID   string   // AI provider to use
    ViewName     string   // Optional custom view name (default: "Resume Import - [date]")
}

type ResumeUploadResponse struct {
    ViewID       string   // ID of newly created view
    ViewSlug     string   // Slug for navigation (/admin/views/[slug])
    ItemCounts   map[string]int // Section → count created
    Warnings     []string // Ambiguous items flagged for review
    Status       string   // "success" | "partial" | "failed"
}
```

**Process**:
1. Validate file (PDF or DOCX, max 5MB)
2. Extract raw text (preserving some structure/formatting)
3. Send to AI parser to get structured JSON
4. Create NEW VIEW record with name from request or "Resume Import - Jan 4, 2026"
5. Create records in each collection (experience, education, skills, etc.)
6. Link all records to the new view via `sections` JSON
7. Set view: `visibility="private"`, `is_active=false`, `is_default=false`
8. Return view ID and stats for redirect to view editor

---

### 2. AI Parsing Strategy

**Prompt Design** (sent to configured AI provider):

```
You are a resume parser. Extract structured data from the resume text below.

Resume text:
"""
{extracted_text}
"""

Extract the following sections. Return valid JSON only, no explanations:

{
  "profile": {
    "name": "Full Name",
    "headline": "Professional title/headline",
    "location": "City, State/Country",
    "summary": "Professional summary or objective",
    "contact_email": "email@example.com"
  },
  "experience": [
    {
      "company": "Company Name",
      "title": "Job Title",
      "location": "City, State",
      "start_date": "YYYY-MM",
      "end_date": "YYYY-MM" or null if current,
      "description": "Brief description",
      "bullets": ["Achievement 1", "Achievement 2"],
      "skills": ["Skill1", "Skill2"]
    }
  ],
  "education": [
    {
      "institution": "University Name",
      "degree": "Degree Type",
      "field": "Field of Study",
      "start_date": "YYYY-MM",
      "end_date": "YYYY-MM",
      "description": "Honors, GPA, activities"
    }
  ],
  "skills": [
    {
      "name": "Skill Name",
      "category": "Programming|Tools|Soft Skills|etc",
      "proficiency": "expert|proficient|familiar"
    }
  ],
  "certifications": [
    {
      "name": "Certification Name",
      "issuer": "Issuing Organization",
      "issue_date": "YYYY-MM",
      "expiry_date": "YYYY-MM" or null,
      "credential_id": "ID if present",
      "credential_url": "URL if present"
    }
  ],
  "projects": [
    {
      "title": "Project Name",
      "summary": "One-sentence description",
      "description": "Detailed description",
      "tech_stack": ["Tech1", "Tech2"],
      "links": [{"type": "github|demo|other", "url": "https://..."}]
    }
  ],
  "awards": [
    {
      "title": "Award Name",
      "issuer": "Issuing Organization",
      "awarded_at": "YYYY-MM",
      "description": "Why awarded"
    }
  ],
  "talks": [
    {
      "title": "Talk Title",
      "event": "Event Name",
      "date": "YYYY-MM",
      "location": "City, State",
      "description": "Talk description"
    }
  ],
  "metadata": {
    "confidence": "high|medium|low",
    "warnings": ["Warning about ambiguous item 1", "Warning 2"],
    "notes": "Any parsing notes for the user"
  }
}

**Important parsing rules**:
1. Extract dates in YYYY-MM format (e.g., "2024-01")
2. If month is ambiguous, use "01" (January)
3. Preserve bullet points as separate array items
4. Categorize skills logically (Programming, Tools, Soft Skills, etc.)
5. Separate freelance/contract work as individual experience items
6. If confidence is low for any item, add to warnings array
7. For ambiguous proficiency levels, default to "proficient"
8. Do NOT hallucinate data - if information isn't present, omit the field
```

**AI Response Validation**:
- Verify JSON structure matches Facet schema
- Check required fields are present (name, at least one section)
- Flag items with low confidence for user review

---

### 3. Review/Edit Workflow (Frontend)

After upload completes, user is redirected to the **existing view editor** at `/admin/views/[new-view-id]`.

The view editor already has:
- ✅ Section enable/disable toggles
- ✅ Item selection and ordering
- ✅ Per-item editing
- ✅ Hero headline/summary overrides
- ✅ Visibility and activation controls

**New: Import Summary Banner**

Add a banner at the top of the view editor for newly imported views:

```
┌─────────────────────────────────────────────────────────┐
│  ℹ️  Resume Import Complete                            │
│                                                          │
│  Imported from: john_doe_resume.pdf                     │
│  Created: 4 experience, 2 education, 15 skills,         │
│           3 certifications, 2 projects                   │
│                                                          │
│  ⚠️  2 items may need review (flagged by AI parser)    │
│                                                          │
│  [View Warnings]     [Dismiss]                          │
└─────────────────────────────────────────────────────────┘
```

**Warnings Modal** (when "View Warnings" clicked):

```
┌─────────────────────────────────────────┐
│  Import Warnings                        │
├─────────────────────────────────────────┤
│  ⚠️  Experience: Freelance Consulting  │
│     Multiple clients mentioned, split   │
│     into one entry. Consider breaking   │
│     into separate items.                │
│     [Go to item →]                      │
│                                         │
│  ⚠️  Skills: JavaScript vs JS          │
│     Both found. Merged into "JavaScript"│
│     [Go to skills section →]            │
│                                         │
│  [Close]                                │
└─────────────────────────────────────────┘
```

**No separate review UI needed** - the existing view editor does everything!

**Edit Modal** (per section item):

```
┌─────────────────────────────────────────┐
│  Edit Experience                        │
├─────────────────────────────────────────┤
│  Company:     [Tech Corp         ]      │
│  Title:       [Senior Engineer   ]      │
│  Location:    [San Francisco, CA ]      │
│  Start Date:  [2020-01          ]      │
│  End Date:    [ ] Current position     │
│                                         │
│  Description:                           │
│  [Leading backend architecture...   ]  │
│                                         │
│  Bullets:                               │
│  • [Reduced latency by 40%          ]  │
│  • [Mentored 5 junior engineers     ]  │
│  • [Architected new API gateway     ]  │
│    [+ Add bullet]                       │
│                                         │
│  Skills: [python, go, kubernetes    ]  │
│                                         │
│  [Cancel]              [Save Changes]  │
└─────────────────────────────────────────┘
```

---

### 4. No Separate Import Execution Needed

Since upload immediately creates the view and all records, there's no separate "import" step.

The user workflow is:
1. Upload resume → Creates view + records
2. Review/edit in existing view editor
3. Publish view when ready (set `is_active=true`, change visibility)

**Deleting the view** cleans up all linked records (handled by existing view delete logic).

---

## Database Schema Changes

**No new collections needed!** Resume upload uses existing collections:
- `views` - New view record created
- `experience`, `education`, `skills`, etc. - New records linked to view
- All records have `visibility="private"` by default

**Optional: Add metadata field to `views` collection**

```js
{
  "name": "import_metadata",
  "type": "json",
  "required": false
  // Stores: { "source": "resume_upload", "filename": "resume.pdf", "warnings": [...] }
}
```

This allows showing the import banner and warnings in the view editor.

---

## Edge Cases & Handling

| Edge Case | Solution |
|-----------|----------|
| Resume has multiple jobs at same company | Split into separate experience items, preserve company name |
| Freelance/consulting with multiple clients | Create one experience item per client OR one with generic title |
| Skills listed in multiple formats | Normalize (e.g., "JavaScript" and "JS" → "JavaScript") |
| Ambiguous dates (e.g., "Summer 2020") | Default to "2020-06", add warning |
| Certifications without expiry | Set `expiry_date` to null |
| Projects without URLs | Omit `links` field |
| Resume has custom sections (e.g., "Publications") | Add to `awards` or skip with warning |
| PDF extraction fails (scanned image) | Return error: "Resume appears to be a scanned image. Please upload a text-based PDF." |
| Conflicting import mode | Warn: "Replace mode will delete X existing items. Continue?" |

---

## Error Handling

**Upload Errors**:
- File too large (>5MB): "File must be under 5MB"
- Unsupported format: "Only PDF and DOCX files are supported"
- File corrupted: "Unable to read file. Please check the file and try again"

**Parsing Errors**:
- AI provider unavailable: "AI provider is offline. Please try again later"
- AI returns invalid JSON: "Failed to parse resume. The AI may not have understood the format"
- No data extracted: "No resume data found. Please check the file content"

**Import Errors**:
- Schema validation failed: "Data validation failed for [field]. Please correct and try again"
- Database write failed: "Failed to save to database. Please try again"

---

## Security Considerations

1. **File Validation**:
   - Check MIME type (not just extension)
   - Scan for malware (if available)
   - Limit file size (5MB max)

2. **Authentication**:
   - Require authentication for upload
   - Only allow admin users to import

3. **Rate Limiting**:
   - Max 5 uploads per hour per user
   - Prevent abuse of AI parsing

4. **Data Privacy**:
   - Uploaded files stored temporarily (24h max)
   - Parsed data in `resume_imports` auto-deleted
   - User can delete import at any time

---

## Implementation Plan

### Phase 1: Backend Foundation
1. Add `go-fitz` and `go-docx` dependencies to `go.mod`
2. Create `services/resume_parser.go` with text extraction
3. Implement `POST /api/resume/upload` endpoint
4. Add `resume_imports` collection via migration
5. Create AI parsing prompt in `services/ai.go`

### Phase 2: Review UI
6. Create `/admin/resume-import` page
7. Build review UI with section expansion
8. Implement edit modals for each section type
9. Add import mode toggle (merge vs replace)

### Phase 3: Import Execution
10. Implement `POST /api/resume/import` endpoint
11. Add validation for all field types
12. Handle merge vs replace logic
13. Add cleanup cron for old imports

### Phase 4: Polish
14. Add progress indicators (upload → parse → review → import)
15. Improve error messages with actionable suggestions
16. Add toast notifications for success/errors
17. Write E2E tests for full workflow

---

## Testing Strategy

**Unit Tests**:
- PDF text extraction (various resume formats)
- DOCX text extraction
- AI prompt generation
- JSON schema validation

**Integration Tests**:
- Upload endpoint (file handling, size limits)
- Import endpoint (merge vs replace modes)
- Database writes (all collections)

**E2E Tests**:
- Full workflow: upload → review → edit → import
- Error scenarios (corrupted file, AI failure)
- Edge cases (ambiguous dates, missing fields)

**Manual Testing**:
- Test with real resumes (chronological, functional, hybrid)
- Test with edge case formatting (tables, columns, graphics)
- Test with non-English resumes (if supported)

---

## Future Enhancements

- **Multi-language support**: Parse resumes in different languages
- **OCR for scanned PDFs**: Use Tesseract for image-based PDFs
- **LinkedIn import**: Parse LinkedIn PDF export
- **Batch import**: Upload multiple resumes for team management
- **Smart deduplication**: Detect duplicate skills/experiences
- **Confidence scoring**: Show AI confidence per field
- **Suggest improvements**: AI suggests better wording for bullets

---

## Success Metrics

- **Parsing Accuracy**: >90% of fields correctly extracted
- **User Edits**: <10% of fields require manual correction
- **Completion Rate**: >80% of uploads result in successful import
- **Time Saved**: Average 15-20 minutes saved vs manual entry
- **Error Rate**: <5% of imports fail due to parsing errors

---

## References

- [Facet Resume Generation](../backend/hooks/resume.go) - Existing generation code
- [Facet Collections Schema](../backend/migrations/) - Database schema
- [go-fitz Documentation](https://github.com/gen2brain/go-fitz)
- [go-docx Documentation](https://github.com/fumiama/go-docx)
