package hooks

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterDownloadHooks registers digital download endpoints
func RegisterDownloadHooks(app *pocketbase.PocketBase, crypto *services.CryptoService, rl *services.RateLimitService) {
	purchaseService := services.NewPurchaseService(crypto)
	share := services.NewShareService(crypto)
	downloadService := services.NewDownloadService(crypto)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// GET /api/content/{content_type}/{content_id}/download-urls
		// Generate signed download URLs for a content item's downloadable files.
		// Auth: purchase JWT cookie (free content or purchased paid content)
		// Rate limited: normal tier (60/min)
		se.Router.GET("/api/content/{content_type}/{content_id}/download-urls", RateLimitMiddleware(rl, "normal")(func(e *core.RequestEvent) error {
			contentType := e.Request.PathValue("content_type")
			contentID := e.Request.PathValue("content_id")

			// Validate content type
			if contentType != "posts" && contentType != "projects" && contentType != "custom_content" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid content type"})
			}

			record, err := app.FindRecordById(contentType, contentID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "content not found"})
			}
			// A private record's files follow the Facet it was opened from. The
			// link carries that Facet, so redeeming it asks the same question.
			facetID := ""
			if !isSuperuser(e) && !isRecordVisible(record) {
				facetID = privateRecordFacet(app, crypto, share, e, record, contentType, e.Request.URL.Query().Get("from"))
			}
			if status, reason := downloadAccess(app, purchaseService, e, contentType, record, facetID != ""); status != 0 {
				return e.JSON(status, map[string]string{"error": reason})
			}

			// Get downloadable files
			downloadableFiles := record.GetStringSlice("downloadable_files")
			if len(downloadableFiles) == 0 {
				return e.JSON(http.StatusOK, map[string]interface{}{
					"files": []interface{}{},
				})
			}

			// Generate signed URLs for each file
			collectionID := record.Collection().Id
			recordID := record.Id

			files := make([]map[string]interface{}, 0, len(downloadableFiles))
			for _, filename := range downloadableFiles {
				token, expiresAt := downloadService.GenerateFacetSignedURL(contentType, contentID, filename, facetID)

				// Try to get file size from storage
				fileSize := getFileSize(app, collectionID, recordID, filename)

				files = append(files, map[string]interface{}{
					"filename":     filename,
					"display_name": filename,
					"size":         fileSize,
					"url":          fmt.Sprintf("/api/downloads/%s", token),
					"expires_at":   expiresAt.Format(time.RFC3339),
				})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"files": files,
			})
		}))

		// GET /api/downloads/{token}
		// Serve a file download using a signed token. The token only names the
		// file; access is checked again here against the current record,
		// purchase and course state, as when the link was issued. Free public
		// files and visible preview lessons need no cookie; protected files
		// need the buyer's current purchase cookie as well as the link.
		// Rate limited: moderate tier (30/min) to prevent abuse
		se.Router.GET("/api/downloads/{token}", RateLimitMiddleware(rl, "moderate")(func(e *core.RequestEvent) error {
			token := e.Request.PathValue("token")
			if token == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "token required"})
			}

			// Validate the signed token
			contentType, contentID, filename, facetID, err := downloadService.ValidateSignedURL(token)
			if err != nil {
				return respondForbidden(e, "invalid or expired download link")
			}

			// Fetch the content record to verify it still exists, then check
			// the visitor may still have it. A link captured before a refund,
			// or before the item went private or draft, stops working.
			if !isDownloadableType(contentType) {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "content not found"})
			}
			record, err := app.FindRecordById(contentType, contentID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "content not found"})
			}
			// A link issued through a Facet works only while that Facet still
			// opens the record for this visitor.
			privateOpens := false
			if facetID != "" {
				view, err := app.FindRecordById(getTableName(app, "views"), facetID)
				privateOpens = err == nil && privateRecordOpensInView(app, crypto, share, e, record, contentType, view)
			}
			if status, reason := downloadAccess(app, purchaseService, e, contentType, record, privateOpens); status != 0 {
				return e.JSON(status, map[string]string{"error": reason})
			}

			// Verify the filename is still in the downloadable files. Lesson
			// attachments use the same signed links, issued by the course API.
			fileField := "downloadable_files"
			if contentType == "lessons" {
				fileField = "attachments"
			}
			downloadableFiles := record.GetStringSlice(fileField)
			found := false
			for _, f := range downloadableFiles {
				if f == filename {
					found = true
					break
				}
			}
			if !found {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "file not found"})
			}

			// Resolve file path from storage
			collectionID := record.Collection().Id
			recordID := record.Id
			filePath := resolveFilePath(app, collectionID, recordID, filename)
			if filePath == "" {
				app.Logger().Error("Download file not found in storage",
					"content_type", contentType,
					"content_id", contentID,
					"filename", filename,
				)
				return e.JSON(http.StatusNotFound, map[string]string{"error": "file not found in storage"})
			}

			// Extract buyer email from purchase cookie for logging
			buyerEmail := ""
			if purchaseToken := extractPurchaseCookie(e); purchaseToken != "" {
				_, email, err := purchaseService.ValidatePurchaseJWT(purchaseToken)
				if err == nil {
					buyerEmail = email
				}
			}

			// Log the download (in background to not block the response)
			clientIP := e.RealIP()
			go logDownload(app, contentType, contentID, filename, buyerEmail, clientIP)

			// Stream the file with Content-Disposition: attachment
			// Sanitize filename for Content-Disposition header (prevent header injection)
			safeFilename := filepath.Base(filename)
			safeFilename = strings.ReplaceAll(safeFilename, `"`, "")
			safeFilename = strings.ReplaceAll(safeFilename, `\`, "")
			e.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, safeFilename))
			e.Response.Header().Set("Cache-Control", protectedFileCacheControl)
			return e.FileFS(os.DirFS(filepath.Dir(filePath)), filepath.Base(filePath))
		}))

		// GET /api/admin/downloads/{content_type}/{content_id}
		// Download statistics for admin. Superuser only.
		se.Router.GET("/api/admin/downloads/{content_type}/{content_id}", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			contentType := e.Request.PathValue("content_type")
			contentID := e.Request.PathValue("content_id")

			// Get total download count
			logs, err := app.FindRecordsByFilter(
				"download_logs",
				"content_type = {:type} && content_id = {:id}",
				"-downloaded_at",
				500,
				0,
				map[string]interface{}{
					"type": contentType,
					"id":   contentID,
				},
			)

			totalDownloads := 0
			uniqueBuyers := make(map[string]bool)
			recentDownloads := make([]map[string]interface{}, 0)

			if err == nil {
				totalDownloads = len(logs)
				for i, log := range logs {
					email := log.GetString("buyer_email")
					if email != "" {
						uniqueBuyers[email] = true
					}

					// Include the 20 most recent downloads
					if i < 20 {
						recentDownloads = append(recentDownloads, map[string]interface{}{
							"filename":      log.GetString("filename"),
							"buyer_email":   log.GetString("buyer_email"),
							"downloaded_at": log.GetDateTime("downloaded_at"),
						})
					}
				}
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"total_downloads":  totalDownloads,
				"unique_buyers":    len(uniqueBuyers),
				"recent_downloads": recentDownloads,
			})
		})

		return se.Next()
	})
}

func isDownloadableType(contentType string) bool {
	switch contentType {
	case "posts", "projects", "custom_content", "lessons":
		return true
	}
	return false
}

// downloadAccess decides whether the request may have a record's files right
// now. Issuing links and redeeming them both ask it, so a signed link is never
// worth more than the request that redeems it. It returns 0 when access is
// allowed, otherwise the status and reason to refuse with.
//
// The site owner may see hidden records, as elsewhere, but still needs a
// purchase for paid files.
//
// privateOpens says a Facet opens this hidden record for the visitor right now.
// Paid and drip rules apply to it as they do to anyone.
func downloadAccess(app *pocketbase.PocketBase, purchaseService *services.PurchaseService, e *core.RequestEvent, contentType string, record *core.Record, privateOpens bool) (int, string) {
	if contentType == "lessons" {
		return lessonDownloadAccess(app, purchaseService, e, record)
	}

	if !isSuperuser(e) && !isRecordVisible(record) && !privateOpens {
		return http.StatusNotFound, "content not found"
	}
	if record.GetBool("drip_enabled") {
		if locked, _ := checkDripLock(app, purchaseService, e, record, contentType); locked {
			return http.StatusForbidden, "content is not yet available"
		}
	}
	if record.GetString("access_tier") == "paid" {
		if ok, _ := checkPurchaseAccess(app, purchaseService, e, contentType, record.Id); !ok {
			return http.StatusForbidden, "purchase required"
		}
	}
	return 0, ""
}

// lessonDownloadAccess applies the course page's rules to a lesson's
// attachments: the course must be visible; preview lessons are then open to
// everyone; other lessons need a purchase of a paid course, or an enrolment
// when a free course requires one, and must not be held back by drip
// scheduling or sequential access.
func lessonDownloadAccess(app *pocketbase.PocketBase, purchaseService *services.PurchaseService, e *core.RequestEvent, lesson *core.Record) (int, string) {
	module, err := app.FindRecordById("modules", lesson.GetString("module"))
	if err != nil {
		return http.StatusNotFound, "content not found"
	}
	course, err := app.FindRecordById("courses", module.GetString("course"))
	if err != nil || !courseVisibleTo(e, course) {
		return http.StatusNotFound, "content not found"
	}
	if lesson.GetBool("is_preview") {
		return 0, ""
	}

	email := ""
	var purchaseIDs []string
	if token := extractPurchaseCookie(e); token != "" {
		if ids, tokenEmail, err := purchaseService.ValidatePurchaseJWT(token); err == nil {
			purchaseIDs, email = ids, tokenEmail
		}
	}

	openToAll := course.GetString("access_tier") != "paid" && !course.GetBool("require_enrollment")
	if !openToAll {
		if ok, reason := learnerEntitled(app, purchaseService, purchaseIDs, email, course); !ok {
			return http.StatusForbidden, reason
		}
	}

	if locked, _ := checkModuleDripLock(app, course.Id, module, email); locked {
		return http.StatusForbidden, "module is not yet available"
	}
	// Sequential access follows a learner's own progress, so as on the course
	// page it only applies to someone who has identified themselves.
	if course.GetBool("sequential_access") && email != "" && !canAccessLesson(app, course.Id, lesson, email) {
		return http.StatusForbidden, "complete previous lessons first"
	}
	return 0, ""
}

// resolveFilePath finds the actual filesystem path for a PocketBase file.
// Checks primary uploads dir first, then fallback to pb_data/storage.
// Sanitises the filename to prevent directory traversal attacks.
func resolveFilePath(app *pocketbase.PocketBase, collectionID, recordID, filename string) string {
	// Sanitise filename to prevent directory traversal
	filename = filepath.Base(filename)
	if filename == "." || filename == ".." || filename == "" {
		return ""
	}

	relPath := filepath.Join(collectionID, recordID, filename)

	// Try primary storage (UPLOADS_DIR)
	uploadsDir := filepath.Clean(filepath.Join("/uploads"))
	primaryPath := filepath.Join(uploadsDir, relPath)
	absPath, err := filepath.Abs(primaryPath)
	if err == nil && strings.HasPrefix(absPath, uploadsDir+string(filepath.Separator)) {
		if _, err := os.Stat(absPath); err == nil {
			return absPath
		}
	}

	// Try fallback storage (pb_data/storage)
	fallbackBase := filepath.Join(app.DataDir(), "storage")
	fallbackPath := filepath.Join(fallbackBase, relPath)
	absPath, err = filepath.Abs(fallbackPath)
	if err == nil && strings.HasPrefix(absPath, fallbackBase+string(filepath.Separator)) {
		if _, err := os.Stat(absPath); err == nil {
			return absPath
		}
	}

	return ""
}

// getFileSize returns the size of a file in storage, or 0 if not found.
func getFileSize(app *pocketbase.PocketBase, collectionID, recordID, filename string) int64 {
	path := resolveFilePath(app, collectionID, recordID, filename)
	if path == "" {
		return 0
	}

	info, err := os.Stat(path)
	if err != nil {
		return 0
	}

	return info.Size()
}

// logDownload creates a download_logs entry. Runs in background goroutine.
func logDownload(app *pocketbase.PocketBase, contentType, contentID, filename, buyerEmail, clientIP string) {
	collection, err := app.FindCollectionByNameOrId("download_logs")
	if err != nil {
		app.Logger().Error("Failed to find download_logs collection", "error", err)
		return
	}

	logRecord := core.NewRecord(collection)
	logRecord.Set("content_type", contentType)
	logRecord.Set("content_id", contentID)
	logRecord.Set("filename", filename)

	if buyerEmail != "" {
		logRecord.Set("buyer_email", buyerEmail)
	}

	// Hash the IP for privacy
	if clientIP != "" {
		hash := sha256.Sum256([]byte(clientIP))
		logRecord.Set("ip_hash", hex.EncodeToString(hash[:]))
	}

	if err := app.Save(logRecord); err != nil {
		app.Logger().Error("Failed to log download", "error", err)
	}
}

// checkDripLock checks if content is drip-locked.
// Returns (isLocked bool, unlockDate time.Time)
func checkDripLock(app *pocketbase.PocketBase, purchaseService *services.PurchaseService, e *core.RequestEvent, record *core.Record, contentType string) (bool, time.Time) {
	// Check absolute unlock date
	unlockDate := record.GetDateTime("drip_unlock_date")
	if !unlockDate.IsZero() && unlockDate.Time().After(time.Now()) {
		return true, unlockDate.Time()
	}

	// Check relative unlock (days after purchase)
	daysAfter := record.GetInt("drip_days_after_purchase")
	if daysAfter > 0 {
		contentID := record.Id

		// Find the purchase to get purchase date
		token := extractPurchaseCookie(e)
		if token == "" {
			// No purchase cookie - if paid content, they can't access anyway
			// If free content with drip, they see it immediately (drip only applies post-purchase)
			return false, time.Time{}
		}

		purchaseIDs, buyerEmail, err := purchaseService.ValidatePurchaseJWT(token)
		if err != nil {
			return false, time.Time{}
		}

		// Find purchase for this content filtered by buyer email
		purchaseRecords, err := app.FindRecordsByFilter(
			"purchases",
			"content_type = {:type} && content_id = {:id} && buyer_email = {:email} && status = 'completed'",
			"created",
			1,
			0,
			map[string]interface{}{
				"type":  contentType,
				"id":    contentID,
				"email": buyerEmail,
			},
		)

		if err != nil || len(purchaseRecords) == 0 {
			return false, time.Time{}
		}

		purchase := purchaseRecords[0]

		// Verify this purchase belongs to the current buyer
		if !purchaseService.HasPurchaseAccess(purchaseIDs, purchase.Id) {
			return false, time.Time{}
		}

		// Check if enough days have passed since purchase
		purchaseDate := purchase.GetDateTime("created")
		if purchaseDate.IsZero() {
			return false, time.Time{}
		}

		unlockTime := purchaseDate.Time().AddDate(0, 0, daysAfter)
		if unlockTime.After(time.Now()) {
			return true, unlockTime
		}
	}

	return false, time.Time{}
}
