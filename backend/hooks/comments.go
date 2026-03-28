package hooks

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"facet/services"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// commentEmailRegex validates email addresses for comment submissions.
var commentEmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// urlRegex matches http:// and https:// URLs for link spam detection.
var urlRegex = regexp.MustCompile(`https?://`)

// validContentTypes for comment threads.
var validContentTypes = map[string]bool{
	"posts":          true,
	"projects":       true,
	"custom_content": true,
	"lessons":        true,
}

// validReportReasons for comment reports.
var validReportReasons = map[string]bool{
	"spam":       true,
	"harassment": true,
	"off_topic":  true,
	"other":      true,
}

// validCommentStatuses for admin status updates.
var validCommentStatuses = map[string]bool{
	"approved": true,
	"rejected": true,
	"spam":     true,
}

// gravatarURL computes a Gravatar URL from an email address.
// Returns empty string if email is empty.
func gravatarURL(email string) string {
	if email == "" {
		return ""
	}
	email = strings.ToLower(strings.TrimSpace(email))
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(email)))
	// Gravatar uses MD5 but we use SHA256 of the trimmed lowercase email
	// per the Gravatar spec: https://docs.gravatar.com/api/avatars/hash/
	// Actually Gravatar supports SHA256 hashes as well.
	return fmt.Sprintf("https://www.gravatar.com/avatar/%s?d=mp&s=40", hash)
}

// hashIP creates a daily-salted SHA-256 hash of the IP for GDPR compliance.
func hashIP(ip string) string {
	dailySalt := time.Now().UTC().Format("2006-01-02")
	h := sha256.Sum256([]byte(ip + dailySalt))
	return fmt.Sprintf("%x", h)
}

// getCommentIP extracts the client IP from the request (X-Forwarded-For or RemoteAddr).
func getCommentIP(e *core.RequestEvent) string {
	if xff := e.Request.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(e.Request.RemoteAddr)
	if err != nil {
		return e.Request.RemoteAddr
	}
	return host
}

// validateContentType checks if the given content type is allowed for comments.
func validateContentType(contentType string) bool {
	return validContentTypes[contentType]
}

// getSiteSettings fetches the first site_settings record, returning nil if not found.
func getSiteSettings(app core.App) *core.Record {
	records, err := app.FindRecordsByFilter("site_settings", "id != ''", "", 1, 0)
	if err != nil || len(records) == 0 {
		return nil
	}
	return records[0]
}

// RegisterCommentHooks registers comment-related API endpoints for threaded discussions.
func RegisterCommentHooks(app *pocketbase.PocketBase, planConfig *services.PlanConfig) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		// ========================================
		// Public Comment Endpoints
		// ========================================

		// GET /api/comments/{content_type}/{content_id} — List approved comments for content
		se.Router.GET("/api/comments/{content_type}/{content_id}", func(e *core.RequestEvent) error {
			// Feature gate
			if !planConfig.HasFeature("discussions") {
				return respondNotFound(e, "Comments are not enabled")
			}

			contentType := e.Request.PathValue("content_type")
			contentID := e.Request.PathValue("content_id")

			if !validateContentType(contentType) {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid content type"})
			}

			// Pagination
			page := 1
			perPage := 50
			if p, err := strconv.Atoi(e.Request.URL.Query().Get("page")); err == nil && p > 0 {
				page = p
			}
			if pp, err := strconv.Atoi(e.Request.URL.Query().Get("per_page")); err == nil && pp > 0 {
				perPage = pp
			}
			if perPage > 100 {
				perPage = 100
			}
			offset := (page - 1) * perPage

			// Query approved comments
			filter := "content_type = {:type} && content_id = {:id} && status = 'approved'"
			params := map[string]interface{}{
				"type": contentType,
				"id":   contentID,
			}

			records, err := app.FindRecordsByFilter("comments", filter, "created", perPage, offset, params)
			if err != nil {
				records = []*core.Record{}
			}

			// Count total
			totalItems64, _ := app.CountRecords("comments",
				dbx.NewExp("content_type = {:type}", dbx.Params{"type": contentType}),
				dbx.NewExp("content_id = {:id}", dbx.Params{"id": contentID}),
				dbx.NewExp("status = 'approved'"),
			)
			totalItems := int(totalItems64)

			// Build response — never expose author_email, ip_hash, or user_agent
			comments := make([]map[string]interface{}, 0, len(records))
			for _, r := range records {
				comment := map[string]interface{}{
					"id":             r.Id,
					"content_type":   r.GetString("content_type"),
					"content_id":     r.GetString("content_id"),
					"parent_id":      r.GetString("parent_id"),
					"author_name":    r.GetString("author_name"),
					"body":           r.GetString("body"),
					"is_admin_reply": r.GetBool("is_admin_reply"),
					"is_pinned":      r.GetBool("is_pinned"),
					"status":         r.GetString("status"),
					"created":        r.GetString("created"),
				}

				// Compute gravatar URL from author_email (not exposed)
				if avatar := gravatarURL(r.GetString("author_email")); avatar != "" {
					comment["avatar_url"] = avatar
				}

				comments = append(comments, comment)
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"comments": comments,
				"total":    totalItems,
				"page":     page,
				"per_page": perPage,
			})
		})

		// POST /api/comments/{content_type}/{content_id} — Submit a new comment
		se.Router.POST("/api/comments/{content_type}/{content_id}", func(e *core.RequestEvent) error {
			// Feature gate
			if !planConfig.HasFeature("discussions") {
				return respondNotFound(e, "Comments are not enabled")
			}

			contentType := e.Request.PathValue("content_type")
			contentID := e.Request.PathValue("content_id")

			if !validateContentType(contentType) {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid content type"})
			}

			// Honeypot check - if the hidden field is filled, it's a bot
			if honeypot := e.Request.FormValue("website_url"); honeypot != "" {
				// Silently accept but don't save (bot doesn't know it was caught)
				return respondJSON(e, http.StatusOK, map[string]any{
					"id":     "fake-" + time.Now().Format("20060102150405"),
					"status": "pending",
				})
			}

			// Verify content exists
			contentRecord, err := app.FindRecordById(contentType, contentID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "content not found"})
			}

			// Verify comments_enabled on the content record
			if !contentRecord.GetBool("comments_enabled") {
				return e.JSON(http.StatusForbidden, map[string]string{"error": "comments are disabled for this content"})
			}

			// Check auto-close
			settings := getSiteSettings(app)
			if settings != nil {
				autoCloseDays := settings.GetInt("comments_auto_close_days")
				if autoCloseDays > 0 {
					createdStr := contentRecord.GetString("created")
					if created, err := time.Parse("2006-01-02 15:04:05.000Z", createdStr); err == nil {
						if time.Since(created) > time.Duration(autoCloseDays)*24*time.Hour {
							return e.JSON(http.StatusForbidden, map[string]string{"error": "comments are closed for this content"})
						}
					}
				}
			}

			// Parse request body
			var req struct {
				AuthorName  string `json:"author_name"`
				AuthorEmail string `json:"author_email"`
				Body        string `json:"body"`
				ParentID    string `json:"parent_id"`
			}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			}

			// Validate required fields
			req.AuthorName = strings.TrimSpace(req.AuthorName)
			req.Body = strings.TrimSpace(req.Body)
			req.AuthorEmail = strings.TrimSpace(req.AuthorEmail)

			if req.AuthorName == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "author_name is required"})
			}
			if len(req.AuthorName) > 100 {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "author_name must be 100 characters or less"})
			}
			if req.Body == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "body is required"})
			}
			if len(req.Body) > 5000 {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "body must be 5000 characters or less"})
			}

			// Validate email format if provided
			if req.AuthorEmail != "" && !commentEmailRegex.MatchString(req.AuthorEmail) {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid email format"})
			}

			// Validate parent_id if set
			if req.ParentID != "" {
				parentComment, err := app.FindRecordById("comments", req.ParentID)
				if err != nil {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "parent comment not found"})
				}
				// Parent must belong to the same content
				if parentComment.GetString("content_type") != contentType || parentComment.GetString("content_id") != contentID {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "parent comment belongs to different content"})
				}
				// Single-level nesting only: parent must not itself be a reply
				if parentComment.GetString("parent_id") != "" {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "replies to replies are not supported"})
				}
			}

			// IP handling
			clientIP := getCommentIP(e)
			ipHash := hashIP(clientIP)

			// User agent (truncate to 500)
			userAgent := e.Request.Header.Get("User-Agent")
			if len(userAgent) > 500 {
				userAgent = userAgent[:500]
			}

			// Duplicate detection: same ip_hash and body in last 5 minutes
			fiveMinAgo := time.Now().UTC().Add(-5 * time.Minute).Format("2006-01-02 15:04:05")
			dupes, _ := app.FindRecordsByFilter(
				"comments",
				"ip_hash = {:ip_hash} && body = {:body} && created > {:since}",
				"",
				1,
				0,
				map[string]interface{}{
					"ip_hash": ipHash,
					"body":    req.Body,
					"since":   fiveMinAgo,
				},
			)
			if len(dupes) > 0 {
				return e.JSON(http.StatusTooManyRequests, map[string]string{"error": "duplicate comment detected, please wait before posting again"})
			}

			// Determine status via auto-moderation
			status := "approved"

			// Blocked word filter
			if settings != nil {
				blockedWordsStr := settings.GetString("comments_blocked_words")
				if blockedWordsStr != "" {
					words := strings.Split(blockedWordsStr, ",")
					bodyLower := strings.ToLower(req.Body)
					for _, w := range words {
						w = strings.TrimSpace(strings.ToLower(w))
						if w == "" {
							continue
						}
						// Whole-word match using word boundary check
						pattern := `(?i)\b` + regexp.QuoteMeta(w) + `\b`
						if matched, _ := regexp.MatchString(pattern, bodyLower); matched {
							status = "spam"
							break
						}
					}
				}
			}

			// Link spam: more than 2 URLs
			if status != "spam" {
				urlMatches := urlRegex.FindAllStringIndex(req.Body, -1)
				if len(urlMatches) > 2 {
					status = "spam"
				}
			}

			// Require approval setting
			if status != "spam" && settings != nil && settings.GetBool("comments_require_approval") {
				status = "pending"
			}

			// Create the comment record
			collection, err := app.FindCollectionByNameOrId("comments")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "comments collection not found"})
			}

			record := core.NewRecord(collection)
			record.Set("content_type", contentType)
			record.Set("content_id", contentID)
			record.Set("parent_id", req.ParentID)
			record.Set("author_name", req.AuthorName)
			record.Set("author_email", req.AuthorEmail)
			record.Set("body", req.Body)
			record.Set("ip_hash", ipHash)
			record.Set("user_agent", userAgent)
			record.Set("status", status)
			record.Set("is_admin_reply", false)
			record.Set("is_pinned", false)

			if err := app.Save(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save comment"})
			}

			// Return created comment (include status so frontend can show moderation state)
			response := map[string]interface{}{
				"id":             record.Id,
				"content_type":   contentType,
				"content_id":     contentID,
				"parent_id":      req.ParentID,
				"author_name":    req.AuthorName,
				"body":           req.Body,
				"status":         status,
				"is_admin_reply": false,
				"is_pinned":      false,
				"created":        record.GetString("created"),
			}

			if avatar := gravatarURL(req.AuthorEmail); avatar != "" {
				response["avatar_url"] = avatar
			}

			return e.JSON(http.StatusCreated, response)
		})

		// POST /api/comments/{comment_id}/report — Report a comment
		se.Router.POST("/api/comments/{comment_id}/report", func(e *core.RequestEvent) error {
			commentID := e.Request.PathValue("comment_id")

			// Verify the comment exists
			_, err := app.FindRecordById("comments", commentID)
			if err != nil {
				// Return 200 even if not found to not leak info
				return e.JSON(http.StatusOK, map[string]string{"status": "received"})
			}

			var req struct {
				Reason string `json:"reason"`
				Detail string `json:"detail"`
			}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			}

			req.Reason = strings.TrimSpace(req.Reason)
			req.Detail = strings.TrimSpace(req.Detail)

			if !validReportReasons[req.Reason] {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "reason must be one of: spam, harassment, off_topic, other"})
			}
			if len(req.Detail) > 1000 {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "detail must be 1000 characters or less"})
			}

			collection, err := app.FindCollectionByNameOrId("comment_reports")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "comment_reports collection not found"})
			}

			report := core.NewRecord(collection)
			report.Set("comment_id", commentID)
			report.Set("reason", req.Reason)
			report.Set("detail", req.Detail)
			report.Set("status", "open")

			if err := app.Save(report); err != nil {
				// Return 200 even on failure to not leak info
				return e.JSON(http.StatusOK, map[string]string{"status": "received"})
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "received"})
		})

		// ========================================
		// Admin Comment Endpoints
		// ========================================

		// GET /api/admin/comments — List comments with filters
		se.Router.GET("/api/admin/comments", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			if !planConfig.HasFeature("discussions") {
				return respondForbidden(e, "discussions not available on current plan")
			}

			statusFilter := strings.TrimSpace(e.Request.URL.Query().Get("status"))
			contentTypeFilter := strings.TrimSpace(e.Request.URL.Query().Get("content_type"))

			page := 1
			perPage := 50
			if p, err := strconv.Atoi(e.Request.URL.Query().Get("page")); err == nil && p > 0 {
				page = p
			}
			if pp, err := strconv.Atoi(e.Request.URL.Query().Get("per_page")); err == nil && pp > 0 && pp <= 100 {
				perPage = pp
			}
			offset := (page - 1) * perPage

			filter := "id != ''"
			params := map[string]interface{}{}
			var countExprs []dbx.Expression

			if statusFilter != "" {
				filter += " && status = {:status}"
				params["status"] = statusFilter
				countExprs = append(countExprs, dbx.NewExp("status = {:status}", dbx.Params{"status": statusFilter}))
			}
			if contentTypeFilter != "" {
				filter += " && content_type = {:content_type}"
				params["content_type"] = contentTypeFilter
				countExprs = append(countExprs, dbx.NewExp("content_type = {:content_type}", dbx.Params{"content_type": contentTypeFilter}))
			}

			records, err := app.FindRecordsByFilter("comments", filter, "-created", perPage, offset, params)
			if err != nil {
				records = []*core.Record{}
			}

			totalItems64, _ := app.CountRecords("comments", countExprs...)
			totalItems := int(totalItems64)

			comments := make([]map[string]interface{}, 0, len(records))
			for _, r := range records {
				// Count open reports for this comment
				reportCount64, _ := app.CountRecords("comment_reports",
					dbx.NewExp("comment_id = {:cid}", dbx.Params{"cid": r.Id}),
					dbx.NewExp("status = 'open'"),
				)

				// Look up content title for admin display
				contentTitle := ""
				contentType := r.GetString("content_type")
				contentID := r.GetString("content_id")
				if contentID != "" && contentType != "" {
					if contentRecord, err := app.FindRecordById(contentType, contentID); err == nil {
						contentTitle = contentRecord.GetString("title")
						if contentTitle == "" {
							contentTitle = contentRecord.GetString("name")
						}
					}
				}

				comments = append(comments, map[string]interface{}{
					"id":             r.Id,
					"content_type":   contentType,
					"content_id":     contentID,
					"content_title":  contentTitle,
					"parent_id":      r.GetString("parent_id"),
					"author_name":    r.GetString("author_name"),
					"author_email":   r.GetString("author_email"),
					"body":           r.GetString("body"),
					"status":         r.GetString("status"),
					"is_admin_reply": r.GetBool("is_admin_reply"),
					"is_pinned":      r.GetBool("is_pinned"),
					"report_count":   int(reportCount64),
					"created":        r.GetString("created"),
				})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"data":       comments,
				"page":       page,
				"perPage":    perPage,
				"totalItems": totalItems,
				"totalPages": (totalItems + perPage - 1) / perPage,
			})
		})

		// PATCH /api/admin/comments/{id} — Update comment status or pin
		se.Router.PATCH("/api/admin/comments/{id}", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			if !planConfig.HasFeature("discussions") {
				return respondForbidden(e, "discussions not available on current plan")
			}

			id := e.Request.PathValue("id")
			record, err := app.FindRecordById("comments", id)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "comment not found"})
			}

			var req struct {
				Status   *string `json:"status"`
				IsPinned *bool   `json:"is_pinned"`
			}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			}

			if req.Status != nil {
				if !validCommentStatuses[*req.Status] {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "status must be one of: approved, rejected, spam"})
				}
				record.Set("status", *req.Status)
			}
			if req.IsPinned != nil {
				record.Set("is_pinned", *req.IsPinned)
			}

			if err := app.Save(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update comment"})
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "updated"})
		})

		// DELETE /api/admin/comments/{id} — Delete comment and its replies
		se.Router.DELETE("/api/admin/comments/{id}", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			if !planConfig.HasFeature("discussions") {
				return respondForbidden(e, "discussions not available on current plan")
			}

			id := e.Request.PathValue("id")
			record, err := app.FindRecordById("comments", id)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "comment not found"})
			}

			// Delete all replies (where parent_id = this comment)
			replies, _ := app.FindRecordsByFilter(
				"comments",
				"parent_id = {:parent_id}",
				"",
				0,
				0,
				map[string]interface{}{"parent_id": id},
			)
			for _, reply := range replies {
				// Delete reports for the reply
				replyReports, _ := app.FindRecordsByFilter(
					"comment_reports",
					"comment_id = {:cid}",
					"",
					0,
					0,
					map[string]interface{}{"cid": reply.Id},
				)
				for _, rr := range replyReports {
					if err := app.Delete(rr); err != nil {
						app.Logger().Warn("comments: failed to delete reply report", "id", rr.Id, "error", err)
					}
				}
				if err := app.Delete(reply); err != nil {
					app.Logger().Warn("comments: failed to delete reply", "id", reply.Id, "error", err)
				}
			}

			// Delete reports for the comment itself
			reports, _ := app.FindRecordsByFilter(
				"comment_reports",
				"comment_id = {:cid}",
				"",
				0,
				0,
				map[string]interface{}{"cid": id},
			)
			for _, rpt := range reports {
				if err := app.Delete(rpt); err != nil {
					app.Logger().Warn("comments: failed to delete report", "id", rpt.Id, "error", err)
				}
			}

			// Delete the comment
			if err := app.Delete(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete comment"})
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
		})

		// POST /api/admin/comments/{content_type}/{content_id} — Admin reply
		se.Router.POST("/api/admin/comments/{content_type}/{content_id}", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			if !planConfig.HasFeature("discussions") {
				return respondForbidden(e, "discussions not available on current plan")
			}

			contentType := e.Request.PathValue("content_type")
			contentID := e.Request.PathValue("content_id")

			if !validateContentType(contentType) {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid content type"})
			}

			// Verify content exists
			_, err := app.FindRecordById(contentType, contentID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "content not found"})
			}

			var req struct {
				Body     string `json:"body"`
				ParentID string `json:"parent_id"`
			}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			}

			req.Body = strings.TrimSpace(req.Body)
			if req.Body == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "body is required"})
			}

			// Determine admin author name from auth record
			authorName := "Admin"
			if e.Auth != nil {
				if name := e.Auth.GetString("name"); name != "" {
					authorName = name
				} else if email := e.Auth.GetString("email"); email != "" {
					authorName = email
				}
			}

			// Validate parent_id if set
			if req.ParentID != "" {
				parentComment, err := app.FindRecordById("comments", req.ParentID)
				if err != nil {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "parent comment not found"})
				}
				if parentComment.GetString("content_type") != contentType || parentComment.GetString("content_id") != contentID {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "parent comment belongs to different content"})
				}
				if parentComment.GetString("parent_id") != "" {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "replies to replies are not supported"})
				}
			}

			collection, err := app.FindCollectionByNameOrId("comments")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "comments collection not found"})
			}

			record := core.NewRecord(collection)
			record.Set("content_type", contentType)
			record.Set("content_id", contentID)
			record.Set("parent_id", req.ParentID)
			record.Set("author_name", authorName)
			record.Set("body", req.Body)
			record.Set("status", "approved")
			record.Set("is_admin_reply", true)
			record.Set("is_pinned", false)

			if err := app.Save(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save comment"})
			}

			return e.JSON(http.StatusCreated, map[string]interface{}{
				"id":             record.Id,
				"content_type":   contentType,
				"content_id":     contentID,
				"parent_id":      req.ParentID,
				"author_name":    authorName,
				"body":           req.Body,
				"status":         "approved",
				"is_admin_reply": true,
				"is_pinned":      false,
				"created":        record.GetString("created"),
			})
		})

		// GET /api/admin/comment-reports — List comment reports
		se.Router.GET("/api/admin/comment-reports", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			if !planConfig.HasFeature("discussions") {
				return respondForbidden(e, "discussions not available on current plan")
			}

			statusFilter := strings.TrimSpace(e.Request.URL.Query().Get("status"))
			if statusFilter == "" {
				statusFilter = "open"
			}

			page := 1
			perPage := 50
			if p, err := strconv.Atoi(e.Request.URL.Query().Get("page")); err == nil && p > 0 {
				page = p
			}
			if pp, err := strconv.Atoi(e.Request.URL.Query().Get("per_page")); err == nil && pp > 0 && pp <= 100 {
				perPage = pp
			}
			offset := (page - 1) * perPage

			filter := "status = {:status}"
			params := map[string]interface{}{"status": statusFilter}

			records, err := app.FindRecordsByFilter("comment_reports", filter, "-created", perPage, offset, params)
			if err != nil {
				records = []*core.Record{}
			}

			totalItems64, _ := app.CountRecords("comment_reports",
				dbx.NewExp("status = {:status}", dbx.Params{"status": statusFilter}),
			)
			totalItems := int(totalItems64)

			reports := make([]map[string]interface{}, 0, len(records))
			for _, r := range records {
				report := map[string]interface{}{
					"id":         r.Id,
					"comment_id": r.GetString("comment_id"),
					"reason":     r.GetString("reason"),
					"detail":     r.GetString("detail"),
					"status":     r.GetString("status"),
					"created":    r.GetString("created"),
				}

				// Include comment details if available
				commentID := r.GetString("comment_id")
				if comment, err := app.FindRecordById("comments", commentID); err == nil {
					report["comment"] = map[string]interface{}{
						"author_name":  comment.GetString("author_name"),
						"body":         comment.GetString("body"),
						"content_type": comment.GetString("content_type"),
						"content_id":   comment.GetString("content_id"),
						"status":       comment.GetString("status"),
					}
				}

				reports = append(reports, report)
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"data":       reports,
				"page":       page,
				"perPage":    perPage,
				"totalItems": totalItems,
				"totalPages": (totalItems + perPage - 1) / perPage,
			})
		})

		// PATCH /api/admin/comment-reports/{id} — Update report status
		se.Router.PATCH("/api/admin/comment-reports/{id}", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			if !planConfig.HasFeature("discussions") {
				return respondForbidden(e, "discussions not available on current plan")
			}

			id := e.Request.PathValue("id")
			record, err := app.FindRecordById("comment_reports", id)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "report not found"})
			}

			var req struct {
				Status string `json:"status"`
			}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			}

			if req.Status != "resolved" && req.Status != "dismissed" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "status must be 'resolved' or 'dismissed'"})
			}

			record.Set("status", req.Status)
			if err := app.Save(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update report"})
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "updated"})
		})

		// POST /api/admin/comments/bulk-action — Bulk actions on comments
		se.Router.POST("/api/admin/comments/bulk-action", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			if !planConfig.HasFeature("discussions") {
				return respondForbidden(e, "discussions not available on current plan")
			}

			var req struct {
				IDs    []string `json:"ids"`
				Action string   `json:"action"`
			}
			if err := e.BindBody(&req); err != nil || len(req.IDs) == 0 {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "ids array and action are required"})
			}

			validActions := map[string]bool{
				"approve": true,
				"reject":  true,
				"delete":  true,
				"spam":    true,
			}
			if !validActions[req.Action] {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "action must be one of: approve, reject, delete, spam"})
			}

			affected := 0
			for _, id := range req.IDs {
				record, err := app.FindRecordById("comments", id)
				if err != nil {
					continue
				}

				if req.Action == "delete" {
					// Delete replies
					replies, _ := app.FindRecordsByFilter(
						"comments",
						"parent_id = {:parent_id}",
						"",
						0,
						0,
						map[string]interface{}{"parent_id": id},
					)
					for _, reply := range replies {
						if err := app.Delete(reply); err != nil {
							app.Logger().Warn("comments: bulk delete reply failed", "id", reply.Id, "error", err)
						}
					}
					// Delete reports
					reports, _ := app.FindRecordsByFilter(
						"comment_reports",
						"comment_id = {:cid}",
						"",
						0,
						0,
						map[string]interface{}{"cid": id},
					)
					for _, rpt := range reports {
						if err := app.Delete(rpt); err != nil {
							app.Logger().Warn("comments: bulk delete report failed", "id", rpt.Id, "error", err)
						}
					}
					if err := app.Delete(record); err == nil {
						affected++
					}
				} else {
					statusMap := map[string]string{
						"approve": "approved",
						"reject":  "rejected",
						"spam":    "spam",
					}
					record.Set("status", statusMap[req.Action])
					if err := app.Save(record); err == nil {
						affected++
					}
				}
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"status":   "completed",
				"affected": affected,
				"message":  fmt.Sprintf("Applied '%s' to %d of %d comments", req.Action, affected, len(req.IDs)),
			})
		})

		return se.Next()
	})
}
