package services

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
)

// CleanupResult holds the count of records deleted per collection.
type CleanupResult struct {
	ExpiredShareTokens        int
	RevokedShareTokens        int
	ExpiredVerificationTokens int
	VerifiedTokens            int
	FailedExports             int
	StuckExports              int
	OrphanedUploads           int

	// Failed lists the steps (by their JSON count name, e.g. "failed_exports")
	// that hit an error. A failed step's count is still the number of records
	// it actually deleted before or around the error.
	Failed []string
}

// OK reports whether every cleanup step finished without an error.
func (r CleanupResult) OK() bool {
	return len(r.Failed) == 0
}

// Total returns the total number of records cleaned up.
func (r CleanupResult) Total() int {
	return r.ExpiredShareTokens + r.RevokedShareTokens +
		r.ExpiredVerificationTokens + r.VerifiedTokens +
		r.FailedExports + r.StuckExports + r.OrphanedUploads
}

// RunCleanup deletes stale records from share_tokens, email_verification_tokens,
// view_exports, and orphaned uploads collections. It returns a summary of what was cleaned.
// A failing step is logged and recorded in Failed; the remaining steps still run.
func RunCleanup(app *pocketbase.PocketBase) CleanupResult {
	result := CleanupResult{}

	steps := []struct {
		name  string
		count *int
		run   func(*pocketbase.PocketBase) (int, error)
	}{
		{"expired_share_tokens", &result.ExpiredShareTokens, cleanupExpiredShareTokens},
		{"revoked_share_tokens", &result.RevokedShareTokens, cleanupRevokedShareTokens},
		{"expired_verification_tokens", &result.ExpiredVerificationTokens, cleanupExpiredVerificationTokens},
		{"verified_tokens", &result.VerifiedTokens, cleanupOldVerifiedTokens},
		{"failed_exports", &result.FailedExports, cleanupFailedExports},
		{"stuck_exports", &result.StuckExports, cleanupStuckExports},
		{"orphaned_uploads", &result.OrphanedUploads, cleanupOrphanedUploads},
	}
	for _, step := range steps {
		deleted, err := step.run(app)
		*step.count = deleted
		if err != nil {
			app.Logger().Error("cleanup: step failed",
				"step", step.name, "deleted", deleted, "error", err)
			result.Failed = append(result.Failed, step.name)
		}
	}

	return result
}

// cleanupExpiredShareTokens deletes share tokens past their expiry date.
func cleanupExpiredShareTokens(app *pocketbase.PocketBase) (int, error) {
	records, err := app.FindRecordsByFilter(
		"share_tokens",
		"expires_at != '' && expires_at < {:now}",
		"",
		500,
		0,
		map[string]any{"now": time.Now().UTC().Format("2006-01-02 15:04:05.000Z")},
	)
	if err != nil {
		app.Logger().Warn("cleanup: failed to query expired share tokens", "error", err)
		return 0, err
	}

	deleted := 0
	var deleteErr error
	for _, record := range records {
		if err := app.Delete(record); err != nil {
			app.Logger().Warn("cleanup: failed to delete expired share token",
				"record_id", record.Id, "error", err)
			if deleteErr == nil {
				deleteErr = fmt.Errorf("delete %s: %w", record.Id, err)
			}
		} else {
			deleted++
		}
	}
	return deleted, deleteErr
}

// cleanupRevokedShareTokens deletes share tokens that were revoked more than 30 days ago.
func cleanupRevokedShareTokens(app *pocketbase.PocketBase) (int, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -30).Format("2006-01-02 15:04:05.000Z")
	records, err := app.FindRecordsByFilter(
		"share_tokens",
		"is_active = false && created < {:cutoff}",
		"",
		500,
		0,
		map[string]any{"cutoff": cutoff},
	)
	if err != nil {
		app.Logger().Warn("cleanup: failed to query revoked share tokens", "error", err)
		return 0, err
	}

	deleted := 0
	var deleteErr error
	for _, record := range records {
		if err := app.Delete(record); err != nil {
			app.Logger().Warn("cleanup: failed to delete revoked share token",
				"record_id", record.Id, "error", err)
			if deleteErr == nil {
				deleteErr = fmt.Errorf("delete %s: %w", record.Id, err)
			}
		} else {
			deleted++
		}
	}
	return deleted, deleteErr
}

// cleanupExpiredVerificationTokens deletes email verification tokens past their expiry.
func cleanupExpiredVerificationTokens(app *pocketbase.PocketBase) (int, error) {
	records, err := app.FindRecordsByFilter(
		"email_verification_tokens",
		"expires_at != '' && expires_at < {:now}",
		"",
		500,
		0,
		map[string]any{"now": time.Now().UTC().Format("2006-01-02 15:04:05.000Z")},
	)
	if err != nil {
		app.Logger().Warn("cleanup: failed to query expired verification tokens", "error", err)
		return 0, err
	}

	deleted := 0
	var deleteErr error
	for _, record := range records {
		if err := app.Delete(record); err != nil {
			app.Logger().Warn("cleanup: failed to delete expired verification token",
				"record_id", record.Id, "error", err)
			if deleteErr == nil {
				deleteErr = fmt.Errorf("delete %s: %w", record.Id, err)
			}
		} else {
			deleted++
		}
	}
	return deleted, deleteErr
}

// cleanupOldVerifiedTokens deletes email verification tokens that were used (verified)
// more than 7 days ago.
func cleanupOldVerifiedTokens(app *pocketbase.PocketBase) (int, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -7).Format("2006-01-02 15:04:05.000Z")
	records, err := app.FindRecordsByFilter(
		"email_verification_tokens",
		"verified_at != '' && verified_at < {:cutoff}",
		"",
		500,
		0,
		map[string]any{"cutoff": cutoff},
	)
	if err != nil {
		app.Logger().Warn("cleanup: failed to query old verified tokens", "error", err)
		return 0, err
	}

	deleted := 0
	var deleteErr error
	for _, record := range records {
		if err := app.Delete(record); err != nil {
			app.Logger().Warn("cleanup: failed to delete old verified token",
				"record_id", record.Id, "error", err)
			if deleteErr == nil {
				deleteErr = fmt.Errorf("delete %s: %w", record.Id, err)
			}
		} else {
			deleted++
		}
	}
	return deleted, deleteErr
}

// cleanupFailedExports deletes view exports that failed more than 7 days ago.
func cleanupFailedExports(app *pocketbase.PocketBase) (int, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -7).Format("2006-01-02 15:04:05.000Z")
	records, err := app.FindRecordsByFilter(
		"view_exports",
		"status = 'failed' && created < {:cutoff}",
		"",
		500,
		0,
		map[string]any{"cutoff": cutoff},
	)
	if err != nil {
		app.Logger().Warn("cleanup: failed to query failed exports", "error", err)
		return 0, err
	}

	deleted := 0
	var deleteErr error
	for _, record := range records {
		if err := app.Delete(record); err != nil {
			app.Logger().Warn("cleanup: failed to delete failed export",
				"record_id", record.Id, "error", err)
			if deleteErr == nil {
				deleteErr = fmt.Errorf("delete %s: %w", record.Id, err)
			}
		} else {
			deleted++
		}
	}
	return deleted, deleteErr
}

// cleanupStuckExports deletes view exports stuck in pending/processing for over 1 hour.
func cleanupStuckExports(app *pocketbase.PocketBase) (int, error) {
	cutoff := time.Now().UTC().Add(-1 * time.Hour).Format("2006-01-02 15:04:05.000Z")
	records, err := app.FindRecordsByFilter(
		"view_exports",
		"(status = 'pending' || status = 'processing') && created < {:cutoff}",
		"",
		500,
		0,
		map[string]any{"cutoff": cutoff},
	)
	if err != nil {
		app.Logger().Warn("cleanup: failed to query stuck exports", "error", err)
		return 0, err
	}

	deleted := 0
	var deleteErr error
	for _, record := range records {
		if err := app.Delete(record); err != nil {
			app.Logger().Warn("cleanup: failed to delete stuck export",
				"record_id", record.Id, "error", err)
			if deleteErr == nil {
				deleteErr = fmt.Errorf("delete %s: %w", record.Id, err)
			}
		} else {
			deleted++
		}
	}
	return deleted, deleteErr
}

// cleanupOrphanedUploads deletes upload records that are not referenced by any content.
// An upload is orphaned if:
// 1. Created more than 24 hours ago (grace period for in-progress editing)
// 2. Not referenced by any LibraryURLFields entry
// 3. Not referenced by nested project experience media
// 4. Not referenced via external_media mirror
//
// The returned count is only uploads actually deleted. If a lookup fails, the
// run stops and returns the error, so a database fault is never read as
// "nothing references this upload".
func cleanupOrphanedUploads(app *pocketbase.PocketBase) (int, error) {
	if _, err := app.FindCollectionByNameOrId("uploads"); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("find uploads collection: %w", err)
	}

	cutoff := time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02 15:04:05.000Z")

	deleted := 0
	perPage := 50
	var deleteErr error

	// Walk uploads by id rather than by page offset: deleting a record would
	// otherwise shift later rows into a page that has already been read.
	lastID := ""
	for {
		records, err := app.FindRecordsByFilter(
			"uploads",
			"created < {:cutoff} && id > {:lastId}",
			"id",
			perPage,
			0,
			dbx.Params{"cutoff": cutoff, "lastId": lastID},
		)
		if err != nil {
			return deleted, fmt.Errorf("query orphaned uploads after %q: %w", lastID, err)
		}

		// No more records to process
		if len(records) == 0 {
			break
		}
		lastID = records[len(records)-1].Id

		for _, record := range records {
			// Get the file URL
			files := record.GetStringSlice("file")
			if len(files) == 0 {
				continue
			}
			filename := files[0]
			fileURL := fmt.Sprintf("/api/files/%s/%s/%s", record.Collection().Id, record.Id, filename)

			isReferenced, err := uploadIsReferenced(app, fileURL)
			if err != nil {
				return deleted, fmt.Errorf("check references for upload %s: %w", record.Id, err)
			}

			// If not referenced anywhere, delete it
			if !isReferenced {
				if err := app.Delete(record); err != nil {
					app.Logger().Warn("cleanup: failed to delete orphaned upload",
						"record_id", record.Id, "file", filename, "error", err)
					if deleteErr == nil {
						deleteErr = fmt.Errorf("delete upload %s: %w", record.Id, err)
					}
				} else {
					app.Logger().Debug("cleanup: deleted orphaned upload",
						"record_id", record.Id, "file", filename)
					deleted++
				}
			}
		}
	}

	return deleted, deleteErr
}

// uploadIsReferenced reports whether any content still points at fileURL.
// A missing optional collection (e.g. courses before its migration) is skipped;
// any other lookup error is returned so the caller keeps the upload.
func uploadIsReferenced(app *pocketbase.PocketBase, fileURL string) (bool, error) {
	for collName, fields := range LibraryURLFields {
		collection, err := app.FindCollectionByNameOrId(collName)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return false, fmt.Errorf("find collection %s: %w", collName, err)
		}
		for _, fieldName := range fields {
			if collection.Fields.GetByName(fieldName) == nil {
				continue
			}
			matches, err := app.FindRecordsByFilter(
				collName,
				fmt.Sprintf("%s = {:url}", fieldName),
				"",
				1,
				0,
				dbx.Params{"url": fileURL},
			)
			if err != nil {
				return false, fmt.Errorf("query %s.%s: %w", collName, fieldName, err)
			}
			if len(matches) > 0 {
				return true, nil
			}
		}
	}

	usage, err := findProjectExperienceMediaUsage(app, fileURL)
	if err != nil {
		return false, err
	}
	if usage.UsageCount > 0 {
		return true, nil
	}

	// Also check external_media for mirrors
	if _, err := app.FindCollectionByNameOrId("external_media"); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("find collection external_media: %w", err)
	}
	mirrors, err := app.FindRecordsByFilter(
		"external_media",
		"url ~ {:url}",
		"",
		0,
		0,
		dbx.Params{"url": fileURL},
	)
	if err != nil {
		return false, fmt.Errorf("query external_media: %w", err)
	}
	for _, mirror := range mirrors {
		// Verify suffix match to prevent false positives from
		// PocketBase's case-insensitive substring matching
		if strings.HasSuffix(mirror.GetString("url"), fileURL) {
			return true, nil
		}
	}
	return false, nil
}
