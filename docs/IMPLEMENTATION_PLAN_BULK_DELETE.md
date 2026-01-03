# Implementation Plan: Bulk Delete Media Endpoint

**Status:** Ready for Implementation
**Priority:** Critical
**Issue:** Bulk delete UI exists but backend endpoint missing, causing 404 errors

---

## Problem Statement

The media library admin UI ([frontend/src/routes/admin/media/+page.svelte:254](../frontend/src/routes/admin/media/+page.svelte#L254)) includes functionality to select multiple orphaned files and delete them in bulk. The UI sends a `POST` request to `/api/media/bulk-delete` with a JSON payload containing orphan file paths. However, this endpoint does not exist in the backend, resulting in 404 errors when users attempt to use this feature.

**Documentation:**
- [docs/MEDIA.md:28](MEDIA.md#L28) documents the endpoint as existing
- Frontend UI at lines 254-279 implements the client-side logic
- Selection logic at lines 224-247 allows users to select orphans

---

## Proposed Solution

Implement the `POST /api/media/bulk-delete` endpoint in [backend/hooks/media.go](../backend/hooks/media.go) following the existing patterns for media deletion.

### API Specification

**Endpoint:** `POST /api/media/bulk-delete`

**Request Body:**
```json
{
  "orphans": ["collectionId/recordId/file1.jpg", "collectionId/recordId/file2.png"]
}
```

**Response (Success):**
```json
{
  "deleted": 2,
  "failed": 0,
  "errors": []
}
```

**Response (Partial Failure):**
```json
{
  "deleted": 1,
  "failed": 1,
  "errors": [
    {
      "path": "collectionId/recordId/file2.png",
      "error": "file not found"
    }
  ]
}
```

### Implementation Details

**Location:** `backend/hooks/media.go` after the existing `DELETE /api/media` handler (around line 253)

**Key Requirements:**
1. Require authentication (`apis.RequireAuth()`)
2. Accept JSON array of relative paths
3. Validate and sanitize each path using existing `resolveStoragePath()` function
4. Delete each file individually using `os.Remove()`
5. Track successes and failures
6. Return detailed response with counts and any errors
7. Log operations for audit purposes

**Security Considerations:**
- Use existing `resolveStoragePath()` function to prevent path traversal
- Ensure paths are within storage root
- Validate that paths don't contain `..` or escape sequences
- Rate limit if deleting large batches (consider adding progress feedback)

### Code Template

```go
se.Router.POST("/api/media/bulk-delete", func(e *core.RequestEvent) error {
	var req struct {
		Orphans []string `json:"orphans"`
	}
	if err := e.BindBody(&req); err != nil {
		return apis.NewBadRequestError("invalid request body", err)
	}

	if len(req.Orphans) == 0 {
		return apis.NewBadRequestError("no orphans specified", nil)
	}

	if len(req.Orphans) > 100 {
		return apis.NewBadRequestError("maximum 100 files per request", nil)
	}

	dataDir := app.DataDir()
	storageRoot := filepath.Join(dataDir, "storage")

	deleted := 0
	failed := 0
	var errors []map[string]string

	for _, relativePath := range req.Orphans {
		target, err := resolveStoragePath(storageRoot, relativePath)
		if err != nil {
			failed++
			errors = append(errors, map[string]string{
				"path":  relativePath,
				"error": "invalid path",
			})
			continue
		}

		if err := os.Remove(target); err != nil {
			app.Logger().Warn("bulk delete: failed to delete file", "path", target, "error", err)
			failed++
			errors = append(errors, map[string]string{
				"path":  relativePath,
				"error": err.Error(),
			})
		} else {
			deleted++
			app.Logger().Info("bulk delete: deleted orphan", "path", target)
		}
	}

	response := map[string]interface{}{
		"deleted": deleted,
		"failed":  failed,
		"errors":  errors,
	}

	return e.JSON(http.StatusOK, response)
}).Bind(apis.RequireAuth())
```

---

## Testing Plan

### Unit Tests
Add to `backend/hooks/media_test.go` (create if doesn't exist):

1. Test successful bulk delete of multiple orphans
2. Test partial failure (some files exist, some don't)
3. Test path validation (reject `..` and path traversal attempts)
4. Test authentication requirement
5. Test request validation (empty array, oversized array)
6. Test response format matches specification

### Integration Tests

1. Create orphan files in test storage directory
2. Call endpoint with valid orphan paths
3. Verify files are deleted
4. Verify response counts are accurate
5. Test error handling when file deletion fails

### Manual Testing Checklist

- [ ] Create orphan files via media library
- [ ] Select multiple orphans in UI
- [ ] Click "Delete selected" button
- [ ] Verify success toast appears
- [ ] Verify files are removed from storage
- [ ] Verify media list refreshes and shows updated counts
- [ ] Test with mix of valid and invalid paths
- [ ] Test with empty selection
- [ ] Test with large selection (50+ files)

---

## Rollout Plan

1. **Implement** endpoint in `backend/hooks/media.go`
2. **Add** unit tests
3. **Test** locally with dev environment
4. **Update** MEDIA.md to change warning from "not implemented" to "implemented"
5. **Update** ROADMAP.md Phase 7.3 to mark bulk delete as ✅ complete
6. **Commit** with message: `feat: implement bulk delete media endpoint`
7. **Create PR** referencing this implementation plan

---

## Success Criteria

- [ ] Endpoint responds at `POST /api/media/bulk-delete`
- [ ] Successfully deletes multiple orphan files
- [ ] Returns accurate deleted/failed counts
- [ ] Returns error details for failures
- [ ] UI no longer shows 404 errors
- [ ] Users can successfully bulk delete orphans
- [ ] Tests pass
- [ ] Documentation updated

---

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Path traversal vulnerability | Use existing `resolveStoragePath()` validation |
| Deleting non-orphan files | Double-check only orphans are passed from UI |
| Performance with large batches | Implement 100-file limit, consider async for larger batches |
| Race condition with concurrent deletes | Document that concurrent operations may conflict |

---

## Related Code

- **Frontend UI:** [frontend/src/routes/admin/media/+page.svelte:254-279](../frontend/src/routes/admin/media/+page.svelte#L254)
- **Existing single delete:** [backend/hooks/media.go:194-252](../backend/hooks/media.go#L194)
- **Path validation:** [backend/hooks/media.go:446-460](../backend/hooks/media.go#L446)
- **Documentation:** [docs/MEDIA.md:28](MEDIA.md#L28)

---

## Estimated Effort

- **Implementation:** 1-2 hours
- **Testing:** 1 hour
- **Documentation:** 30 minutes
- **Total:** 2.5-3.5 hours
