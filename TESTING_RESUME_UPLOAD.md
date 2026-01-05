# Testing Resume Upload Feature

**Branch**: `feature/resume-upload-and-parsing`
**Commit**: `75437d1` - Add resume upload & AI parsing feature

---

## Quick Test

### Prerequisites

1. **Backend running**: `cd backend && air`
2. **Frontend running**: `cd frontend && npm run dev`
3. **AI Provider configured**: Set up OpenAI, Anthropic, or Ollama in Settings
4. **Test resume file**: Have a PDF or DOCX resume ready

### Option 1: Automated Test Script

```bash
./test-resume-upload.sh ~/Downloads/your-resume.pdf
```

The script will:
- Check if backend is running
- Prompt for auth token
- Upload the resume
- Show import statistics

### Option 2: Manual cURL Test

1. **Get auth token**: Login at http://localhost:5173/admin/login, open DevTools → Application → Cookies → find `pb_auth`

2. **Upload resume**:

```bash
curl -X POST http://localhost:8090/api/resume/upload \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -F "file=@/path/to/resume.pdf" \
  -F "provider_id=" \
  | jq '.'
```

### Option 3: Test with Postman/Insomnia

**Endpoint**: `POST http://localhost:8090/api/resume/upload`

**Headers**:
- `Authorization`: `Bearer <your-token>`

**Body** (form-data):
- `file`: (select your resume PDF/DOCX)
- `provider_id`: (leave empty to use default)

---

## Expected Response

### Success (200 OK)

```json
{
  "status": "success",
  "imported": {
    "experience": ["rec1", "rec2", "rec3"],
    "education": ["rec4", "rec5"],
    "skills": ["rec6", "rec7", "rec8"],
    "certifications": ["rec9"],
    "projects": ["rec10", "rec11"]
  },
  "counts": {
    "experience": 3,
    "education": 2,
    "skills": 3,
    "certifications": 1,
    "projects": 2,
    "awards": 0,
    "talks": 0
  },
  "warnings": [
    "Experience: Freelance Consulting - Consider splitting if multiple clients involved"
  ],
  "confidence": "high",
  "filename": "john_doe_resume.pdf"
}
```

### Errors

**401 Unauthorized**:
```json
{
  "error": "Authentication required"
}
```
→ You need to login first

**400 Bad Request - File too large**:
```json
{
  "error": "File is too large. Maximum size is 5MB."
}
```

**400 Bad Request - Invalid file type**:
```json
{
  "error": "Invalid file type. Please upload a PDF or DOCX file."
}
```

**400 Bad Request - No AI provider**:
```json
{
  "error": "AI provider not configured. Please configure an AI provider in settings."
}
```

**500 Internal Server Error - Parsing failed**:
```json
{
  "error": "AI parsing failed: <details>. Please try again or use a different file."
}
```

---

## Verify Import

After successful upload, check the imported data:

1. **Visit Admin Panel**: http://localhost:5173/admin/experience
2. **Look for private items**: All imported items have `visibility = private`
3. **Review each section**:
   - `/admin/experience` - Work history
   - `/admin/education` - Schools/degrees
   - `/admin/skills` - Skills by category
   - `/admin/certifications` - Certifications
   - `/admin/projects` - Projects (if any on resume)

---

## What Happens

1. **File Upload** → Backend receives PDF/DOCX
2. **Text Extraction** → go-fitz (PDF) or go-docx (DOCX) extracts text
3. **AI Parsing** → Sends text to configured AI provider with structured prompt
4. **Data Validation** → Validates JSON response, adds warnings
5. **Record Creation** → Creates records in main collections:
   - All records: `visibility = "private"`
   - All records: `is_draft = false`
   - Generated slugs for projects/talks
6. **Response** → Returns import summary with IDs, counts, warnings

---

## Testing Checklist

- [ ] Upload PDF resume (text-based, not scanned)
- [ ] Upload DOCX resume
- [ ] Test with 6MB file (should fail: "File too large")
- [ ] Test with .txt file (should fail: "Invalid file type")
- [ ] Test without AI provider configured (should fail)
- [ ] Verify all sections imported correctly
- [ ] Verify items are private by default
- [ ] Check warnings array for quality issues
- [ ] Test with resume containing:
  - [ ] Multiple jobs
  - [ ] Education history
  - [ ] Skills in different categories
  - [ ] Certifications with expiry dates
  - [ ] Projects with GitHub links
  - [ ] Awards/honors
  - [ ] Speaking engagements

---

## Troubleshooting

### "No text found in PDF"
- Resume is likely a scanned image
- Try converting to text-based PDF first
- Or use DOCX version if available

### "AI parsing failed"
- Check AI provider is configured and active
- Verify API key is valid
- Check backend logs for details: `cd backend && air`

### "Failed to create records"
- Database migration may be missing
- Check backend logs for schema errors
- Verify collections exist: `experience`, `education`, `skills`, etc.

### Empty response
- Check if backend is running: `curl http://localhost:8090/api/health`
- Check auth token is valid
- Look at browser Network tab for error details

---

## Next Steps (Coming Soon)

Frontend UI will provide:
- ✨ Drag-and-drop upload
- 📊 Real-time progress indicator
- 🎨 Beautiful import summary page
- 🔗 "Create View from Import" button
- ⚠️ Interactive warning review

But the backend is **fully functional** right now - test it!
