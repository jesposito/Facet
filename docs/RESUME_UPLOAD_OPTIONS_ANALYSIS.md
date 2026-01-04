# Resume Upload Design Options Analysis

**Date**: 2026-01-04
**Empirica Session**: aa34d76e-f858-4663-b295-0bc041c65838

## Design Question

Where should parsed resume data go?

---

## Option A: Direct Import to Main Profile

**Flow**: Upload → Parse → Add/Replace items in main collections → Done

**Pros**:
- ✅ Simplest user flow (one step)
- ✅ Data immediately in main profile
- ✅ No orphaned records

**Cons**:
- ❌ **DESTRUCTIVE** - Can't undo easily
- ❌ **RISKY** - Corrupts existing profile if parsing fails or is incorrect
- ❌ **NO REVIEW** - User can't preview before committing
- ❌ Merge vs Replace logic complex and error-prone
- ❌ User loses control over what gets imported

**Verdict**: ❌ **REJECTED** - Too risky, doesn't align with Facet's philosophy of user control

---

## Option B: Create Isolated VIEW with New Records

**Flow**: Upload → Parse → Create NEW VIEW → All data linked only to that view → User reviews

**Pros**:
- ✅ **SAFE** - Doesn't touch existing profile/views
- ✅ **ISOLATED** - Can delete view and all records together
- ✅ Easy to review before publishing
- ✅ Clear mental model: "This is the resume import view"

**Cons**:
- ⚠️ Creates "orphan" records only visible in that view
- ⚠️ User can't easily merge content into other views
- ⚠️ If user wants content in main profile, needs manual copy/paste
- ⚠️ Records are tied to view lifecycle (delete view → lose all data)

**Use Cases**:
- ✅ User wants a "resume view" separate from their main profile
- ✅ Importing someone else's resume (e.g., for team management)
- ❌ User wants to enrich their existing profile with resume data

**Verdict**: ✅ **GOOD** but not ideal for primary use case

---

## Option C: Import to Main Collections as Private/Draft

**Flow**: Upload → Parse → Create records in main collections → Set `visibility="private"` by default → User creates view(s) manually

**Process**:
1. Upload resume
2. Parse and create records in:
   - `experience` (visibility=private, is_draft=false)
   - `education` (visibility=private)
   - `skills` (visibility=private)
   - etc.
3. Show import summary: "Created 4 experience items, 2 education items..."
4. Redirect to `/admin/experience` (or import summary page)
5. User reviews each section:
   - Edit items inline
   - Change visibility to public when ready
   - Delete items they don't want
6. User creates views and selects which items to include

**Pros**:
- ✅ **FLEXIBLE** - Content in main collections, reusable across views
- ✅ **SAFE** - Private by default, doesn't show anywhere until user approves
- ✅ **GRADUAL** - User can publish items one-by-one or all at once
- ✅ **REUSABLE** - Same content can be in multiple views
- ✅ **ALIGNED** with Facet's existing mental model (collections → views)
- ✅ No orphaned data - records live in main collections
- ✅ User keeps full control

**Cons**:
- ⚠️ Two-step process (import → create view)
- ⚠️ Slightly more complex flow
- ⚠️ User might be confused: "Where did my resume go?"

**Mitigation**:
- Show clear post-import summary with next steps
- Add "Create View from Imported Content" button
- Banner on admin pages: "You have 10 private items from resume import"

**Use Cases**:
- ✅ User wants to add resume data to existing profile
- ✅ User wants to create multiple views with resume content
- ✅ User wants to gradually review and publish content
- ✅ User wants to mix resume data with manually entered data

**Verdict**: ✅ **BEST** - Most flexible, safest, aligns with Facet's design

---

## Epistemic Assessment

| Criterion | Option A | Option B | Option C |
|-----------|----------|----------|----------|
| **Safety** | 0.2 (risky) | 0.9 (very safe) | 0.9 (very safe) |
| **Flexibility** | 0.3 (inflexible) | 0.5 (limited reuse) | 0.9 (max flexibility) |
| **User Control** | 0.3 (auto-import) | 0.7 (isolated control) | 0.9 (granular control) |
| **Complexity** | 0.9 (simple) | 0.7 (moderate) | 0.5 (complex) |
| **Alignment with Facet** | 0.4 (contradicts) | 0.7 (isolated model) | 0.9 (perfect fit) |
| **Undo-ability** | 0.2 (hard to undo) | 0.9 (delete view) | 0.8 (delete items) |
| **TOTAL** | **2.3** | **4.7** | **5.0** ⭐ |

---

## DECISION: Option C

**Rationale**:
1. **Safety**: Parsed data is private by default, can't break existing profile
2. **Flexibility**: Content in main collections, reusable across views
3. **Philosophy**: Aligns with Facet's "you control everything" approach
4. **Gradual**: User can review, edit, and publish at their own pace
5. **Mental Model**: Matches existing flow (create content → add to views)

**Implementation** (Updated Design):

### Upload Flow

```
POST /api/resume/upload
  ↓
Validate file (PDF/DOCX, max 5MB)
  ↓
Extract text (go-fitz for PDF, go-docx for DOCX)
  ↓
Parse with AI → Structured JSON
  ↓
Create records in main collections:
  - experience (visibility="private")
  - education (visibility="private")
  - skills (visibility="private")
  - certifications (visibility="private")
  - projects (visibility="private")
  ↓
Return import summary + record IDs
```

### Response Format

```json
{
  "status": "success",
  "imported": {
    "experience": ["id1", "id2", "id3", "id4"],
    "education": ["id5", "id6"],
    "skills": ["id7", "id8", "id9", "id10", "id11"],
    "certifications": ["id12", "id13", "id14"],
    "projects": ["id15", "id16"]
  },
  "counts": {
    "experience": 4,
    "education": 2,
    "skills": 5,
    "certifications": 3,
    "projects": 2
  },
  "warnings": [
    "Experience 'Freelance Consulting': Multiple clients mentioned, consider splitting",
    "Skills: Merged 'JavaScript' and 'JS' into 'JavaScript'"
  ]
}
```

### Post-Import UI

Redirect to `/admin/import-summary` (new page):

```
┌───────────────────────────────────────────────────────┐
│  ✅ Resume Import Successful                         │
├───────────────────────────────────────────────────────┤
│                                                        │
│  Imported from: john_doe_resume.pdf                   │
│  Created: 4 experience, 2 education, 5 skills,        │
│           3 certifications, 2 projects                 │
│                                                        │
│  ⚠️  All items are PRIVATE and won't show on your    │
│     profile until you make them visible.              │
│                                                        │
│  ⚠️  2 warnings to review                            │
│                                                        │
│  Next Steps:                                          │
│  1. Review and edit imported items                    │
│  2. Change visibility to "public" for items you want  │
│  3. Create a view to display this content             │
│                                                        │
│  ┌────────────────────────────────────────────────┐  │
│  │  Quick Actions                                 │  │
│  │  ────────────────────────────────────────────  │  │
│  │  [Review Experience (4 items) →]              │  │
│  │  [Review Education (2 items) →]               │  │
│  │  [Review Skills (5 items) →]                  │  │
│  │  [Create View from Imported Content →]        │  │
│  └────────────────────────────────────────────────┘  │
│                                                        │
│  [View All Warnings]          [Go to Dashboard]       │
└───────────────────────────────────────────────────────┘
```

### "Create View from Imported Content" Feature

When user clicks that button, create a view pre-configured with imported items:

```
POST /api/views/from-import
Request: { "import_ids": ["id1", "id2", ...] }

Response: { "view_id": "view123", "view_slug": "resume-import-jan-4" }

Redirect to: /admin/views/view123
```

The view is created with:
- Name: "Resume Import - Jan 4, 2026"
- All imported items pre-selected
- visibility="private"
- is_active=false
- User can then customize and publish

---

## Migration Path from Option B

If we started with Option B, we can add Option C behavior:

1. Upload creates VIEW (Option B)
2. Also creates records in main collections (Option C)
3. View references those records
4. User can:
   - Keep the view as-is (Option B behavior)
   - Delete the view but keep records (transition to Option C)
   - Make records public and use in other views (Option C benefit)

**Best of both worlds!**

---

## Warnings & Edge Cases (Updated)

| Warning | Handling |
|---------|----------|
| Duplicate skill names | Merge and add warning |
| Ambiguous dates | Default to YYYY-01, add warning |
| Freelance with multiple clients | One entry with warning to split |
| Missing required fields | Skip item and add to warnings |
| Parsing confidence low | Add to warnings, create item anyway (private) |

---

## Final Decision

✅ **Option C: Import to Main Collections as Private**

Next steps:
1. Update design doc to reflect Option C
2. Implement upload endpoint with this flow
3. Create import summary UI
4. Add "Create View from Import" helper
5. Update Empirica with final decision

Confidence: **HIGH** (0.85)
Reasoning: Aligns with all design principles, most flexible, lowest risk
