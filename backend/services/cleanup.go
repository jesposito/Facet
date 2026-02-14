package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
)

// CleanupResult holds the count of records deleted per collection.
type CleanupResult struct {
	ExpiredShareTokens       int
	RevokedShareTokens       int
	ExpiredVerificationTokens int
	VerifiedTokens           int
	FailedExports            int
	StuckExports             int
	OrphanedUploads          int
}

// Total returns the total number of records cleaned up.
func (r CleanupResult) Total() int {
	return r.ExpiredShareTokens + r.RevokedShareTokens +
		r.ExpiredVerificationTokens + r.VerifiedTokens +
		r.FailedExports + r.StuckExports + r.OrphanedUploads
}

// RunCleanup deletes stale records from share_tokens, email_verification_tokens,
// view_exports, and orphaned uploads collections. It returns a summary of what was cleaned.
func RunCleanup(app *pocketbase.PocketBase) CleanupResult {
	result := CleanupResult{}

	result.ExpiredShareTokens = cleanupExpiredShareTokens(app)
	result.RevokedShareTokens = cleanupRevokedShareTokens(app)
	result.ExpiredVerificationTokens = cleanupExpiredVerificationTokens(app)
	result.VerifiedTokens = cleanupOldVerifiedTokens(app)
	result.FailedExports = cleanupFailedExports(app)
	result.StuckExports = cleanupStuckExports(app)
	result.OrphanedUploads = cleanupOrphanedUploads(app)

	return result
}

// cleanupExpiredShareTokens deletes share tokens past their expiry date.
func cleanupExpiredShareTokens(app *pocketbase.PocketBase) int {
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
		return 0
	}

	deleted := 0
	for _, record := range records {
		if err := app.Delete(record); err != nil {
			app.Logger().Warn("cleanup: failed to delete expired share token",
				"record_id", record.Id, "error", err)
		} else {
			deleted++
		}
	}
	return deleted
}

// cleanupRevokedShareTokens deletes share tokens that were revoked more than 30 days ago.
func cleanupRevokedShareTokens(app *pocketbase.PocketBase) int {
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
		return 0
	}

	deleted := 0
	for _, record := range records {
		if err := app.Delete(record); err != nil {
			app.Logger().Warn("cleanup: failed to delete revoked share token",
				"record_id", record.Id, "error", err)
		} else {
			deleted++
		}
	}
	return deleted
}

// cleanupExpiredVerificationTokens deletes email verification tokens past their expiry.
func cleanupExpiredVerificationTokens(app *pocketbase.PocketBase) int {
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
		return 0
	}

	deleted := 0
	for _, record := range records {
		if err := app.Delete(record); err != nil {
			app.Logger().Warn("cleanup: failed to delete expired verification token",
				"record_id", record.Id, "error", err)
		} else {
			deleted++
		}
	}
	return deleted
}

// cleanupOldVerifiedTokens deletes email verification tokens that were used (verified)
// more than 7 days ago.
func cleanupOldVerifiedTokens(app *pocketbase.PocketBase) int {
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
		return 0
	}

	deleted := 0
	for _, record := range records {
		if err := app.Delete(record); err != nil {
			app.Logger().Warn("cleanup: failed to delete old verified token",
				"record_id", record.Id, "error", err)
		} else {
			deleted++
		}
	}
	return deleted
}

// cleanupFailedExports deletes view exports that failed more than 7 days ago.
func cleanupFailedExports(app *pocketbase.PocketBase) int {
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
		return 0
	}

	deleted := 0
	for _, record := range records {
		if err := app.Delete(record); err != nil {
			app.Logger().Warn("cleanup: failed to delete failed export",
				"record_id", record.Id, "error", err)
		} else {
			deleted++
		}
	}
	return deleted
}

// cleanupStuckExports deletes view exports stuck in pending/processing for over 1 hour.
func cleanupStuckExports(app *pocketbase.PocketBase) int {
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
		return 0
	}

	deleted := 0
	for _, record := range records {
		if err := app.Delete(record); err != nil {
			app.Logger().Warn("cleanup: failed to delete stuck export",
				"record_id", record.Id, "error", err)
		} else {
			deleted++
		}
	}
	return deleted
}

// cleanupOrphanedUploads deletes upload records that are not referenced by any content.
// An upload is orphaned if:
// 1. Created more than 24 hours ago (grace period for in-progress editing)
// 2. Not referenced by any *_library_url field in any collection
// 3. Not referenced via external_media mirror
func cleanupOrphanedUploads(app *pocketbase.PocketBase) int {
	cutoff := time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02 15:04:05.000Z")

	deleted := 0
	page := 1
	perPage := 50

	// Collections and their library_url fields to check.
	// MAINTENANCE: Update this map when adding new collections with *_library_url fields.
	collectionsToCheck := map[string][]string{
		"profile":        {"hero_image_library_url", "avatar_library_url"},
		"experience":     {"company_logo_library_url"},
		"education":      {"institution_logo_library_url"},
		"projects":       {"cover_image_library_url"},
		"posts":          {"cover_image_library_url"},
		"talks":          {"cover_image_library_url"},
		"views":          {"hero_image_library_url"},
		"site_settings":  {"favicon_library_url"},
		"custom_content": {"cover_image_library_url"},
		"testimonials":   {"author_photo_library_url"},
	}

	for {
		// Query uploads in batches
		records, err := app.FindRecordsByFilter(
			"uploads",
			"created < {:cutoff}",
			"",
			perPage,
			(page-1)*perPage,
			dbx.Params{"cutoff": cutoff},
		)
		if err != nil {
			app.Logger().Warn("cleanup: failed to query orphaned uploads", "error", err)
			return deleted
		}

		// No more records to process
		if len(records) == 0 {
			break
		}

		for _, record := range records {
			// Get the file URL
			files := record.GetStringSlice("file")
			if len(files) == 0 {
				continue
			}
			filename := files[0]
			fileURL := fmt.Sprintf("/api/files/%s/%s/%s", record.Collection().Id, record.Id, filename)

			// Check if this upload is referenced anywhere
			isReferenced := false

			// Check all collections for library_url references
			for collName, fields := range collectionsToCheck {
				for _, fieldName := range fields {
					matches, err := app.FindRecordsByFilter(
						collName,
						fmt.Sprintf("%s = {:url}", fieldName),
						"",
						1,
						0,
						dbx.Params{"url": fileURL},
					)
					if err != nil {
						// Collection might not exist yet (e.g., testimonials), skip silently
						continue
					}
					if len(matches) > 0 {
						isReferenced = true
						break
					}
				}
				if isReferenced {
					break
				}
			}

			// Also check external_media for mirrors
			if !isReferenced {
				mirrors, err := app.FindRecordsByFilter(
					"external_media",
					"url ~ {:url}",
					"",
					10,
					0,
					dbx.Params{"url": fileURL},
				)
				if err == nil {
					for _, mirror := range mirrors {
						// Verify suffix match to prevent false positives from
						// PocketBase's case-insensitive substring matching
						if strings.HasSuffix(mirror.GetString("url"), fileURL) {
							isReferenced = true
							break
						}
					}
				}
			}

			// If not referenced anywhere, delete it
			if !isReferenced {
				if err := app.Delete(record); err != nil {
					app.Logger().Warn("cleanup: failed to delete orphaned upload",
						"record_id", record.Id, "file", filename, "error", err)
				} else {
					app.Logger().Debug("cleanup: deleted orphaned upload",
						"record_id", record.Id, "file", filename)
					deleted++
				}
			}
		}

		// Move to next page
		page++
	}

	return deleted
}
