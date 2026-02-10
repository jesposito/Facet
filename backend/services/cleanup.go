package services

import (
	"time"

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
}

// Total returns the total number of records cleaned up.
func (r CleanupResult) Total() int {
	return r.ExpiredShareTokens + r.RevokedShareTokens +
		r.ExpiredVerificationTokens + r.VerifiedTokens +
		r.FailedExports + r.StuckExports
}

// RunCleanup deletes stale records from share_tokens, email_verification_tokens,
// and view_exports collections. It returns a summary of what was cleaned.
func RunCleanup(app *pocketbase.PocketBase) CleanupResult {
	result := CleanupResult{}

	result.ExpiredShareTokens = cleanupExpiredShareTokens(app)
	result.RevokedShareTokens = cleanupRevokedShareTokens(app)
	result.ExpiredVerificationTokens = cleanupExpiredVerificationTokens(app)
	result.VerifiedTokens = cleanupOldVerifiedTokens(app)
	result.FailedExports = cleanupFailedExports(app)
	result.StuckExports = cleanupStuckExports(app)

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
