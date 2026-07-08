package hooks

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"

	"facet/services"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
	"github.com/yuin/goldmark"
)

// emailRegex validates email addresses for newsletter subscription.
// Note: commentEmailRegex also exists in comments.go in the same package — both are intentional.
var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// hrefSrcRegex matches href="..." and src="..." attributes for protocol allowlist enforcement.
var hrefSrcRegex = regexp.MustCompile(`(?i)((?:href|src)\s*=\s*["'])([^"']*)(["'])`)

// stripDangerousProtocols enforces an allowlist on href/src attribute values.
// Only http://, https://, mailto:, fragment (#), and empty strings are allowed.
func stripDangerousProtocols(htmlContent string) string {
	return hrefSrcRegex.ReplaceAllStringFunc(htmlContent, func(match string) string {
		parts := hrefSrcRegex.FindStringSubmatch(match)
		if len(parts) < 4 {
			return match
		}
		val := strings.TrimSpace(parts[2])
		if val == "" || strings.HasPrefix(val, "#") ||
			strings.HasPrefix(strings.ToLower(val), "http://") ||
			strings.HasPrefix(strings.ToLower(val), "https://") ||
			strings.HasPrefix(strings.ToLower(val), "mailto:") {
			return match
		}
		return parts[1] + "#" + parts[3]
	})
}

// sanitizeCSVField prefixes dangerous characters to prevent CSV formula injection.
func sanitizeCSVField(field string) string {
	if len(field) == 0 {
		return field
	}
	switch field[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + field
	}
	return field
}

// linkHrefRegex matches <a href="..."> links for click-tracking rewriting.
var linkHrefRegex = regexp.MustCompile(`(<a\s[^>]*href=")([^"]+)(")`)

// rewriteLinksForTracking replaces <a href="..."> links in newsletter HTML with click-tracking URLs.
// Skips unsubscribe links to ensure they always work directly.
func rewriteLinksForTracking(app core.App, htmlContent, sendID, token, unsubscribeURL string) string {
	baseURL := strings.TrimRight(app.Settings().Meta.AppURL, "/")
	return linkHrefRegex.ReplaceAllStringFunc(htmlContent, func(match string) string {
		parts := linkHrefRegex.FindStringSubmatch(match)
		if len(parts) < 4 {
			return match
		}
		href := parts[2]
		// Skip empty hrefs, unsubscribe links, anchors, and non-HTTP schemes
		if href == "" || href == unsubscribeURL || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "mailto:") ||
			strings.HasPrefix(href, "tel:") || strings.HasPrefix(href, "sms:") || strings.HasPrefix(href, "facetime:") {
			return match
		}
		// Skip relative URLs — they'll break after base64 encoding
		if !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
			return match
		}
		encoded := base64.URLEncoding.EncodeToString([]byte(href))
		trackURL := fmt.Sprintf("%s/api/public/newsletter/click/%s/%s/%s", baseURL, sendID, token, encoded)
		return parts[1] + trackURL + parts[3]
	})
}

// RegisterNewsletterHooks registers all newsletter-related routes and hooks.
// Note: rl is accepted for future rate limiting use but not applied to routes in this upstream version.
func RegisterNewsletterHooks(app *pocketbase.PocketBase, crypto *services.CryptoService, rl *services.RateLimitService, planConfig *services.PlanConfig) {
	// Clean up stale "sending" records on startup.
	// Any send still in "sending" status after 10 minutes is assumed to have crashed.
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		staleThreshold := time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339)
		staleSends, err := app.FindRecordsByFilter(
			"newsletter_sends",
			"status = 'sending' && created < {:threshold}",
			"",
			0,
			0,
			map[string]interface{}{"threshold": staleThreshold},
		)
		if err == nil {
			for _, sr := range staleSends {
				sr.Set("status", "failed")
				sr.Set("sent_at", time.Now().UTC().Format(time.RFC3339))
				if saveErr := app.Save(sr); saveErr != nil {
					log.Printf("[newsletter] Failed to mark stale send %s as failed: %v", sr.Id, saveErr)
				} else {
					log.Printf("[newsletter] Marked stale send %s as failed (startup cleanup)", sr.Id)
				}
			}
		}
		return se.Next()
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		// POST /api/public/{slug}/subscribe — public newsletter signup
		se.Router.POST("/api/public/{slug}/subscribe", func(e *core.RequestEvent) error {
			slug := e.Request.PathValue("slug")

			// Honeypot check — bots fill hidden fields, humans don't
			if honeypot := e.Request.FormValue("website_url"); honeypot != "" {
				return e.JSON(http.StatusOK, map[string]string{"status": "subscribed"})
			}

			// Parse request body
			var req struct {
				Email  string `json:"email"`
				Name   string `json:"name"`
				Source string `json:"source"`
			}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			// Validate email
			req.Email = strings.ToLower(strings.TrimSpace(req.Email))
			if req.Email == "" || !emailRegex.MatchString(req.Email) {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid email address"})
			}

			// Feature gate: check newsletter feature for managed instances
			if planConfig.IsManaged() && !planConfig.HasFeature("newsletter") {
				return respondForbidden(e, "newsletter not available on current plan")
			}

			// Verify view exists by slug
			records, err := app.FindRecordsByFilter(
				"views",
				"slug = {:slug} && is_active = true",
				"",
				1,
				0,
				map[string]interface{}{"slug": slug},
			)
			if err != nil || len(records) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "view not found"})
			}
			view := records[0]

			// Source defaults to "newsletter"
			source := req.Source
			if source == "" {
				source = "newsletter"
			}

			// Check for existing subscriber
			existing, _ := app.FindRecordsByFilter(
				"subscribers",
				"email = {:email}",
				"",
				1,
				0,
				map[string]interface{}{"email": req.Email},
			)

			if len(existing) > 0 {
				sub := existing[0]
				if sub.GetString("status") == "unsubscribed" {
					// Re-subscribe
					sub.Set("status", "active")
					sub.Set("source", source)
					sub.Set("view_id", view.Id)
					if req.Name != "" {
						sub.Set("name", req.Name)
					}
					if err := app.Save(sub); err != nil {
						return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to resubscribe"})
					}
					// Send welcome email on re-subscribe (SMTP graceful degradation handled inside)
					sendWelcomeEmail(app, crypto, sub.Id)
				}
				// Already active — return success silently
				return e.JSON(http.StatusOK, map[string]string{"status": "subscribed"})
			}

			// Create new subscriber
			collection, err := app.FindCollectionByNameOrId("subscribers")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "subscribers collection not found"})
			}

			record := core.NewRecord(collection)
			record.Set("email", req.Email)
			record.Set("name", req.Name)
			record.Set("source", source)
			record.Set("view_id", view.Id)
			record.Set("status", "active")

			if err := app.Save(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to subscribe"})
			}

			// Send welcome email to new subscriber (SMTP graceful degradation handled inside)
			sendWelcomeEmail(app, crypto, record.Id)

			return e.JSON(http.StatusOK, map[string]string{"status": "subscribed"})
		})

		// ========================================
		// Admin Subscriber Management Endpoints
		// ========================================

		// GET /api/admin/subscribers — List subscribers with search, status filter, pagination
		se.Router.GET("/api/admin/subscribers", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			if planConfig.IsManaged() && !planConfig.HasFeature("newsletter") {
				return respondForbidden(e, "newsletter not available on current plan")
			}

			search := strings.TrimSpace(e.Request.URL.Query().Get("search"))
			status := strings.TrimSpace(e.Request.URL.Query().Get("status"))
			pageStr := e.Request.URL.Query().Get("page")
			perPageStr := e.Request.URL.Query().Get("perPage")

			page := parsePositiveInt(pageStr, 1, 0)
			perPage := parsePositiveInt(perPageStr, 50, 200)

			filter := "id != ''"
			params := map[string]interface{}{}
			var countExprs []dbx.Expression

			if status != "" && (status == "active" || status == "unsubscribed" || status == "bounced") {
				filter += " && status = {:status}"
				params["status"] = status
				countExprs = append(countExprs, dbx.NewExp("status = {:status}", dbx.Params{"status": status}))
			}

			if search != "" {
				filter += " && (email ~ {:search} || name ~ {:search})"
				params["search"] = search
				countExprs = append(countExprs, dbx.NewExp("(email LIKE {:search} OR name LIKE {:search})", dbx.Params{"search": "%" + search + "%"}))
			}

			offset := (page - 1) * perPage

			records, err := app.FindRecordsByFilter("subscribers", filter, "-created", perPage, offset, params)
			if err != nil {
				records = []*core.Record{}
			}

			totalItems64, _ := app.CountRecords("subscribers", countExprs...)
			totalItems := int(totalItems64)

			subs := make([]map[string]interface{}, 0, len(records))
			for _, r := range records {
				subs = append(subs, map[string]interface{}{
					"id":      r.Id,
					"email":   r.GetString("email"),
					"name":    r.GetString("name"),
					"source":  r.GetString("source"),
					"status":  r.GetString("status"),
					"created": r.GetString("created"),
				})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"data":       subs,
				"page":       page,
				"perPage":    perPage,
				"totalItems": totalItems,
				"totalPages": (totalItems + perPage - 1) / perPage,
			})
		})

		// GET /api/admin/subscribers/stats — Source breakdown and growth stats
		se.Router.GET("/api/admin/subscribers/stats", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			if planConfig.IsManaged() && !planConfig.HasFeature("newsletter") {
				return respondForbidden(e, "newsletter not available on current plan")
			}

			type sourceCount struct {
				Source string `json:"source"`
				Count  int    `json:"count"`
			}
			var sources []sourceCount
			rows, err := app.DB().
				NewQuery("SELECT COALESCE(NULLIF(source, ''), 'unknown') as source, COUNT(*) as count FROM subscribers WHERE status = 'active' GROUP BY source ORDER BY count DESC").
				Rows()
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var sc sourceCount
					if err := rows.Scan(&sc.Source, &sc.Count); err == nil {
						sources = append(sources, sc)
					}
				}
			}

			var totalActive int
			app.DB().NewQuery("SELECT COUNT(*) FROM subscribers WHERE status = 'active'").Row(&totalActive)
			var totalUnsubscribed int
			app.DB().NewQuery("SELECT COUNT(*) FROM subscribers WHERE status = 'unsubscribed'").Row(&totalUnsubscribed)
			var totalBounced int
			app.DB().NewQuery("SELECT COUNT(*) FROM subscribers WHERE status = 'bounced'").Row(&totalBounced)

			type dailyCount struct {
				Date  string `json:"date"`
				Count int    `json:"count"`
			}
			var growth []dailyCount
			growthRows, err := app.DB().
				NewQuery("SELECT DATE(created) as date, COUNT(*) as count FROM subscribers WHERE created >= datetime('now', '-30 days') GROUP BY DATE(created) ORDER BY date ASC").
				Rows()
			if err == nil {
				defer growthRows.Close()
				for growthRows.Next() {
					var dc dailyCount
					if err := growthRows.Scan(&dc.Date, &dc.Count); err == nil {
						growth = append(growth, dc)
					}
				}
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"sources":            sources,
				"total_active":       totalActive,
				"total_unsubscribed": totalUnsubscribed,
				"total_bounced":      totalBounced,
				"growth":             growth,
			})
		})

		// GET /api/admin/subscribers/export — CSV download with optional filters
		se.Router.GET("/api/admin/subscribers/export", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			if planConfig.IsManaged() && !planConfig.HasFeature("newsletter") {
				return respondForbidden(e, "newsletter not available on current plan")
			}

			search := strings.TrimSpace(e.Request.URL.Query().Get("search"))
			status := strings.TrimSpace(e.Request.URL.Query().Get("status"))

			filter := "id != ''"
			params := map[string]interface{}{}

			if status != "" && (status == "active" || status == "unsubscribed" || status == "bounced") {
				filter += " && status = {:status}"
				params["status"] = status
			}
			if search != "" {
				filter += " && (email ~ {:search} || name ~ {:search})"
				params["search"] = search
			}

			records, err := app.FindRecordsByFilter("subscribers", filter, "-created", 0, 0, params)
			if err != nil {
				records = []*core.Record{}
			}

			// Empty CSV guard — return JSON instead of empty CSV
			if len(records) == 0 {
				return e.JSON(http.StatusOK, map[string]string{
					"status":  "empty",
					"message": "No subscribers match the current filters",
				})
			}

			e.Response.Header().Set("Content-Type", "text/csv")
			e.Response.Header().Set("Content-Disposition", "attachment; filename=subscribers.csv")

			w := csv.NewWriter(e.Response)
			w.Write([]string{"email", "name", "source", "status", "created"})
			for _, r := range records {
				w.Write([]string{
					sanitizeCSVField(r.GetString("email")),
					sanitizeCSVField(r.GetString("name")),
					sanitizeCSVField(r.GetString("source")),
					sanitizeCSVField(r.GetString("status")),
					sanitizeCSVField(r.GetString("created")),
				})
			}
			w.Flush()
			return nil
		})

		// DELETE /api/admin/subscribers/{id} — Delete single subscriber
		se.Router.DELETE("/api/admin/subscribers/{id}", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			if planConfig.IsManaged() && !planConfig.HasFeature("newsletter") {
				return respondForbidden(e, "newsletter not available on current plan")
			}

			id := e.Request.PathValue("id")
			record, err := app.FindRecordById("subscribers", id)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "subscriber not found"})
			}

			if err := app.Delete(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete subscriber"})
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
		})

		// DELETE /api/admin/subscribers/bulk — Bulk delete subscribers
		se.Router.DELETE("/api/admin/subscribers/bulk", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			if planConfig.IsManaged() && !planConfig.HasFeature("newsletter") {
				return respondForbidden(e, "newsletter not available on current plan")
			}

			var req struct {
				IDs []string `json:"ids"`
			}
			if err := e.BindBody(&req); err != nil || len(req.IDs) == 0 {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "ids array is required"})
			}

			deleted := 0
			missing := 0
			err := app.RunInTransaction(func(txApp core.App) error {
				for _, id := range req.IDs {
					record, err := txApp.FindRecordById("subscribers", id)
					if err != nil {
						missing++
						continue
					}
					if err := txApp.Delete(record); err != nil {
						return fmt.Errorf("delete subscriber %s: %w", id, err)
					}
					deleted++
				}
				return nil
			})
			if err != nil {
				app.Logger().Error("bulk subscriber delete failed; transaction rolled back", "requested", len(req.IDs), "error", err)
				return e.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error":   "bulk delete failed; no subscribers were removed",
					"message": err.Error(),
				})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"status":  "deleted",
				"deleted": deleted,
				"missing": missing,
				"message": fmt.Sprintf("Deleted %d of %d subscribers (%d already absent)", deleted, len(req.IDs), missing),
			})
		})

		registerNewsletterSubscriberImportRoute(se, app, planConfig)

		// ========================================
		// Newsletter Compose & Send Endpoints
		// ========================================

		// GET /api/admin/newsletter/smtp-status — Check if SMTP is configured
		se.Router.GET("/api/admin/newsletter/smtp-status", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			configured := app.Settings().SMTP.Host != ""
			return respondJSON(e, http.StatusOK, map[string]any{
				"configured": configured,
				"host":       app.Settings().SMTP.Host,
			})
		})

		// GET /api/admin/newsletter/settings — Get newsletter sender settings
		se.Router.GET("/api/admin/newsletter/settings", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			if planConfig.IsManaged() && !planConfig.HasFeature("newsletter") {
				return respondForbidden(e, "newsletter not available on current plan")
			}

			profile, err := app.FindFirstRecordByFilter("profile", "1=1", nil)
			if err != nil {
				return e.JSON(http.StatusOK, map[string]string{
					"sender_name": "",
					"reply_to":    "",
				})
			}

			return e.JSON(http.StatusOK, map[string]string{
				"sender_name": profile.GetString("newsletter_sender_name"),
				"reply_to":    profile.GetString("newsletter_reply_to"),
			})
		})

		// PUT /api/admin/newsletter/settings — Update newsletter sender settings
		se.Router.PUT("/api/admin/newsletter/settings", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			if planConfig.IsManaged() && !planConfig.HasFeature("newsletter") {
				return respondForbidden(e, "newsletter not available on current plan")
			}

			var req struct {
				SenderName string `json:"sender_name"`
				ReplyTo    string `json:"reply_to"`
			}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			}

			profile, err := app.FindFirstRecordByFilter("profile", "1=1", nil)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "profile not found"})
			}

			profile.Set("newsletter_sender_name", req.SenderName)
			profile.Set("newsletter_reply_to", req.ReplyTo)

			if err := app.Save(profile); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save settings"})
			}

			return e.JSON(http.StatusOK, map[string]string{
				"sender_name": profile.GetString("newsletter_sender_name"),
				"reply_to":    profile.GetString("newsletter_reply_to"),
			})
		})

		// GET /api/admin/newsletter/templates — List built-in email templates
		se.Router.GET("/api/admin/newsletter/templates", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			return e.JSON(http.StatusOK, BuiltInTemplates)
		})

		// POST /api/admin/newsletter/send — Send newsletter (or test email)
		se.Router.POST("/api/admin/newsletter/send", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			if planConfig.IsManaged() && !planConfig.HasFeature("newsletter") {
				return respondForbidden(e, "newsletter not available on current plan")
			}

			var req struct {
				Subject      string `json:"subject"`
				BodyMarkdown string `json:"body_markdown"`
				TestEmail    string `json:"test_email"`
				Template     string `json:"template"`
			}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			}

			req.Subject = strings.TrimSpace(req.Subject)
			req.BodyMarkdown = strings.TrimSpace(req.BodyMarkdown)
			if req.Subject == "" || req.BodyMarkdown == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "subject and body_markdown are required"})
			}

			// Convert markdown to HTML
			bodyHTML, err := markdownToHTML(req.BodyMarkdown)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to render markdown"})
			}

			// Test email mode
			if req.TestEmail != "" {
				if !emailRegex.MatchString(req.TestEmail) {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid test email"})
				}

				if app.Settings().SMTP.Host == "" {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "SMTP not configured"})
				}

				testSenderName := app.Settings().Meta.SenderName
				if profile, err := app.FindFirstRecordByFilter("profile", "1=1", nil); err == nil && profile != nil {
					if customName := profile.GetString("newsletter_sender_name"); customName != "" {
						testSenderName = customName
					}
				}
				testHTML := RenderNewsletterTemplate(req.Template, bodyHTML, "#", testSenderName)
				message := &mailer.Message{
					From: mail.Address{
						Address: app.Settings().Meta.SenderAddress,
						Name:    testSenderName,
					},
					To:      []mail.Address{{Address: req.TestEmail}},
					Subject: "[TEST] " + req.Subject,
					HTML:    testHTML,
				}

				if sendErr := app.NewMailClient().Send(message); sendErr != nil {
					app.Logger().Warn("newsletter: test email failed", "error", sendErr)
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to send test email"})
				}

				return e.JSON(http.StatusOK, map[string]string{"status": "test_sent"})
			}

			// Full send mode
			if app.Settings().SMTP.Host == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "SMTP not configured"})
			}

			// Concurrent send guard - check if another send is already in progress
			activeSendCount, _ := app.CountRecords("newsletter_sends", dbx.NewExp("status = 'sending'"))
			if activeSendCount > 0 {
				return e.JSON(http.StatusConflict, map[string]string{"error": "a newsletter send is already in progress"})
			}

			// Count active subscribers
			activeSubscribers, err := app.FindRecordsByFilter("subscribers", "status = 'active'", "", 0, 0)
			if err != nil || len(activeSubscribers) == 0 {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "no active subscribers"})
			}

			// Collect subscriber IDs for goroutine (don't pass mutable pointers)
			subscriberIDs := make([]string, 0, len(activeSubscribers))
			for _, sub := range activeSubscribers {
				subscriberIDs = append(subscriberIDs, sub.Id)
			}

			// Create send record
			sendCollection, err := app.FindCollectionByNameOrId("newsletter_sends")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "newsletter_sends collection not found"})
			}

			sendRecord := core.NewRecord(sendCollection)
			sendRecord.Set("subject", req.Subject)
			sendRecord.Set("body_html", bodyHTML)
			sendRecord.Set("status", "sending")
			sendRecord.Set("recipient_count", len(subscriberIDs))
			sendRecord.Set("sent_count", 0)
			sendRecord.Set("failed_count", 0)

			if err := app.Save(sendRecord); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create send record"})
			}

			// Spawn bounded background sender (pass IDs, not mutable pointers).
			goSendNewsletter(app, crypto, sendRecord.Id, req.Subject, bodyHTML, req.Template, subscriberIDs)

			return e.JSON(http.StatusOK, map[string]interface{}{
				"status":          "sending",
				"send_id":         sendRecord.Id,
				"recipient_count": len(subscriberIDs),
			})
		})

		// GET /api/admin/newsletter/sends — List send history
		se.Router.GET("/api/admin/newsletter/sends", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			if planConfig.IsManaged() && !planConfig.HasFeature("newsletter") {
				return respondForbidden(e, "newsletter not available on current plan")
			}

			records, err := app.FindRecordsByFilter("newsletter_sends", "id != ''", "-created", 50, 0)
			if err != nil {
				records = []*core.Record{}
			}

			sends := make([]map[string]interface{}, 0, len(records))
			for _, r := range records {
				sends = append(sends, map[string]interface{}{
					"id":              r.Id,
					"subject":         r.GetString("subject"),
					"status":          r.GetString("status"),
					"recipient_count": r.GetInt("recipient_count"),
					"sent_count":      r.GetInt("sent_count"),
					"failed_count":    r.GetInt("failed_count"),
					"open_count":      r.GetInt("open_count"),
					"click_count":     r.GetInt("click_count"),
					"sent_at":         r.GetString("sent_at"),
					"created":         r.GetString("created"),
				})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{"data": sends})
		})

		// DELETE /api/admin/newsletter/sends/{id} — Delete send record
		se.Router.DELETE("/api/admin/newsletter/sends/{id}", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			if planConfig.IsManaged() && !planConfig.HasFeature("newsletter") {
				return respondForbidden(e, "newsletter not available on current plan")
			}

			id := e.Request.PathValue("id")
			record, err := app.FindRecordById("newsletter_sends", id)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "send record not found"})
			}

			if err := app.Delete(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete send record"})
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
		})

		// ========================================
		// Public Unsubscribe Endpoints
		// ========================================

		// GET /api/public/unsubscribe/{token} — Validate unsubscribe token
		se.Router.GET("/api/public/unsubscribe/{token}", func(e *core.RequestEvent) error {
			token := e.Request.PathValue("token")
			if token == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "token required"})
			}

			sub, err := findSubscriberByToken(app, crypto, token)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "invalid or expired token"})
			}

			// Mask email: j***@example.com
			email := sub.GetString("email")
			masked := maskEmail(email)

			return e.JSON(http.StatusOK, map[string]interface{}{
				"email":  masked,
				"status": sub.GetString("status"),
			})
		})

		// POST /api/public/unsubscribe — Execute unsubscribe
		se.Router.POST("/api/public/unsubscribe", func(e *core.RequestEvent) error {
			var req struct {
				Token string `json:"token"`
			}
			if err := e.BindBody(&req); err != nil || req.Token == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "token required"})
			}

			sub, err := findSubscriberByToken(app, crypto, req.Token)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "invalid or expired token"})
			}

			if sub.GetString("status") == "unsubscribed" {
				return e.JSON(http.StatusOK, map[string]string{"status": "already_unsubscribed"})
			}

			sub.Set("status", "unsubscribed")
			if err := app.Save(sub); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to unsubscribe"})
			}

			log.Printf("[newsletter] Subscriber %s unsubscribed via token", sub.GetString("email"))
			return e.JSON(http.StatusOK, map[string]string{"status": "unsubscribed"})
		})

		// POST /api/public/unsubscribe/by-email — Fallback unsubscribe by email (CAN-SPAM compliance).
		// Always returns success to prevent email enumeration.
		se.Router.POST("/api/public/unsubscribe/by-email", func(e *core.RequestEvent) error {
			var req struct {
				Email string `json:"email"`
			}
			if err := e.BindBody(&req); err != nil {
				// Always return success — anti-enumeration
				return e.JSON(http.StatusOK, map[string]string{"status": "received"})
			}

			req.Email = strings.ToLower(strings.TrimSpace(req.Email))
			if req.Email == "" || !emailRegex.MatchString(req.Email) {
				return e.JSON(http.StatusOK, map[string]string{"status": "received"})
			}

			// Look up subscriber silently
			existing, _ := app.FindRecordsByFilter(
				"subscribers",
				"email = {:email}",
				"",
				1,
				0,
				map[string]interface{}{"email": req.Email},
			)

			if len(existing) > 0 && existing[0].GetString("status") == "active" {
				existing[0].Set("status", "unsubscribed")
				if err := app.Save(existing[0]); err != nil {
					log.Printf("[newsletter] Failed to unsubscribe by email %s: %v", req.Email, err)
				} else {
					log.Printf("[newsletter] Subscriber %s unsubscribed via email fallback", req.Email)
				}
			}

			// Always return success — anti-enumeration
			return e.JSON(http.StatusOK, map[string]string{"status": "received"})
		})

		// POST /api/public/unsubscribe/one-click/{token} — RFC 8058 one-click unsubscribe
		se.Router.POST("/api/public/unsubscribe/one-click/{token}", func(e *core.RequestEvent) error {
			token := e.Request.PathValue("token")
			if token == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "token required"})
			}

			sub, err := findSubscriberByToken(app, crypto, token)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "invalid or expired token"})
			}

			if sub.GetString("status") == "unsubscribed" {
				return e.JSON(http.StatusOK, map[string]string{"status": "already_unsubscribed"})
			}

			sub.Set("status", "unsubscribed")
			if err := app.Save(sub); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to unsubscribe"})
			}

			log.Printf("[newsletter] Subscriber %s unsubscribed via one-click (RFC 8058)", sub.GetString("email"))
			return e.JSON(http.StatusOK, map[string]string{"status": "unsubscribed"})
		})

		// GET /api/public/newsletter/open/{send_id}/{token} — Tracking pixel (1x1 transparent GIF)
		se.Router.GET("/api/public/newsletter/open/{send_id}/{token}", func(e *core.RequestEvent) error {
			sendID := e.Request.PathValue("send_id")
			token := e.Request.PathValue("token")
			pixelData, _ := base64.StdEncoding.DecodeString("R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7")
			if sendID != "" && token != "" {
				go recordNewsletterEvent(app, crypto, sendID, token, "open", "")
			}
			e.Response.Header().Set("Content-Type", "image/gif")
			e.Response.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			_, _ = e.Response.Write(pixelData)
			return nil
		})

		// GET /api/public/newsletter/click/{send_id}/{token}/{url} — Click tracking redirect
		se.Router.GET("/api/public/newsletter/click/{send_id}/{token}/{url...}", func(e *core.RequestEvent) error {
			sendID := e.Request.PathValue("send_id")
			token := e.Request.PathValue("token")
			encodedURL := e.Request.PathValue("url")
			targetURL := ""
			if decoded, err := base64.URLEncoding.DecodeString(encodedURL); err == nil {
				targetURL = string(decoded)
			}
			if targetURL == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid url"})
			}
			// Validate redirect URL scheme — prevent open redirect
			parsed, err := url.Parse(targetURL)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid url"})
			}
			if sendID != "" && token != "" {
				go recordNewsletterEvent(app, crypto, sendID, token, "click", targetURL)
			}
			return e.Redirect(http.StatusFound, targetURL)
		})

		// GET /api/admin/newsletter/sends/{id}/stats — Send engagement statistics
		se.Router.GET("/api/admin/newsletter/sends/{id}/stats", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			sendID := e.Request.PathValue("id")
			send, err := app.FindRecordById("newsletter_sends", sendID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "send not found"})
			}
			return e.JSON(http.StatusOK, map[string]interface{}{
				"send_id":     send.Id,
				"open_count":  send.GetInt("open_count"),
				"click_count": send.GetInt("click_count"),
				"total_sent":  send.GetInt("recipient_count"),
			})
		})

		return se.Next()
	})
}

// tokenMaxAge is the maximum age of an unsubscribe token before it is rotated (90 days).
const tokenMaxAge = 90 * 24 * time.Hour

// ensureUnsubscribeToken returns a stable unsubscribe token for a subscriber.
// Uses encrypted token storage (AES-GCM) so the raw token can be recovered without regeneration.
// Tokens are reused for up to 90 days to keep unsubscribe links stable.
func ensureUnsubscribeToken(app core.App, crypto *services.CryptoService, sub *core.Record) (string, error) {
	// Try to decrypt existing token if present and fresh
	encToken := sub.GetString("unsubscribe_token_enc")
	generatedAt := sub.GetString("token_generated_at")

	if encToken != "" && generatedAt != "" {
		if ts, err := time.Parse(time.RFC3339, generatedAt); err == nil {
			if time.Since(ts) < tokenMaxAge {
				// Token is still fresh — decrypt and return
				rawToken, err := crypto.Decrypt(encToken)
				if err == nil && rawToken != "" {
					return rawToken, nil
				}
				// Decrypt failed — fall through to regenerate
				log.Printf("[newsletter] Failed to decrypt token for subscriber %s, regenerating", sub.Id)
			}
		}
	}

	// Generate new token
	oldHash := sub.GetString("unsubscribe_token_hash")

	rawToken, err := crypto.GenerateToken(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate unsubscribe token: %w", err)
	}

	tokenHash := crypto.HMACToken(rawToken)

	// Encrypt raw token for stable storage
	encrypted, err := crypto.Encrypt(rawToken)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt unsubscribe token: %w", err)
	}

	// Rotate: move current hash to prev before overwriting
	if oldHash != "" {
		sub.Set("prev_unsubscribe_token_hash", oldHash)
	}
	sub.Set("unsubscribe_token_hash", tokenHash)
	sub.Set("unsubscribe_token_enc", encrypted)
	sub.Set("token_generated_at", time.Now().UTC().Format(time.RFC3339))
	if err := app.Save(sub); err != nil {
		return "", fmt.Errorf("failed to save unsubscribe token: %w", err)
	}

	return rawToken, nil
}

// generateUnsubscribeURL builds a full unsubscribe URL using the app's configured URL.
func generateUnsubscribeURL(app core.App, token string) string {
	baseURL := strings.TrimRight(app.Settings().Meta.AppURL, "/")
	return fmt.Sprintf("%s/unsubscribe/%s", baseURL, token)
}

// generateOneClickUnsubscribeURL builds the RFC 8058 one-click unsubscribe POST URL.
func generateOneClickUnsubscribeURL(app core.App, token string) string {
	baseURL := strings.TrimRight(app.Settings().Meta.AppURL, "/")
	return fmt.Sprintf("%s/api/public/unsubscribe/one-click/%s", baseURL, token)
}

// recordNewsletterEvent records an open or click event, deduplicates by subscriber+send+type,
// and updates aggregate counts on the newsletter_sends record.
func recordNewsletterEvent(app core.App, crypto *services.CryptoService, sendID, token, eventType, targetURL string) {
	// Validate eventType to prevent SQL injection via countField concatenation
	if eventType != "open" && eventType != "click" {
		log.Printf("[newsletter] Invalid event type: %s", eventType)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[newsletter] Event recording panic: %v", r)
		}
	}()

	// Resolve subscriber from token
	sub, err := findSubscriberByToken(app, crypto, token)
	if err != nil {
		return // invalid token — silently ignore
	}

	subID := sub.Id

	// Create event record — rely on unique composite index for deduplication
	eventsCollection, err := app.FindCollectionByNameOrId("newsletter_events")
	if err != nil {
		log.Printf("[newsletter] Events collection not found: %v", err)
		return
	}

	event := core.NewRecord(eventsCollection)
	event.Set("send_id", sendID)
	event.Set("subscriber_id", subID)
	// Map open/click to stored enum values
	storedEventType := eventType
	if eventType == "open" {
		storedEventType = "opened"
	} else if eventType == "click" {
		storedEventType = "clicked"
	}
	event.Set("event_type", storedEventType)
	if targetURL != "" {
		event.Set("url", targetURL)
	}
	if err := app.Save(event); err != nil {
		// Unique constraint violation means duplicate — silently ignore
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return
		}
		log.Printf("[newsletter] Failed to save event: %v", err)
		return
	}

	// Atomically increment the aggregate count on the send record.
	countField, ok := newsletterCountFieldFor(eventType)
	if !ok {
		log.Printf("[newsletter] Refusing to update count for invalid event type: %s", eventType)
		return
	}
	_, err = app.DB().NewQuery("UPDATE newsletter_sends SET " + countField + " = " + countField + " + 1 WHERE id = {:id}").
		Bind(dbx.Params{"id": sendID}).
		Execute()
	if err != nil {
		log.Printf("[newsletter] Failed to update send %s count: %v", countField, err)
	}
}

func newsletterCountFieldFor(eventType string) (string, bool) {
	switch eventType {
	case "open":
		return "open_count", true
	case "click":
		return "click_count", true
	default:
		return "", false
	}
}

// findSubscriberByToken finds a subscriber by validating an unsubscribe token against stored HMACs.
// Checks both current and previous token hashes to support token rotation.
func findSubscriberByToken(app core.App, crypto *services.CryptoService, token string) (*core.Record, error) {
	expectedHash := crypto.HMACToken(token)

	records, err := app.FindRecordsByFilter(
		"subscribers",
		"unsubscribe_token_hash = {:hash} || prev_unsubscribe_token_hash = {:hash}",
		"",
		1,
		0,
		map[string]any{"hash": expectedHash},
	)
	if err != nil || len(records) == 0 {
		return nil, fmt.Errorf("subscriber not found")
	}

	return records[0], nil
}

// sendWelcomeEmail sends a branded welcome email to a new subscriber in a goroutine.
// Accepts subscriber ID (not pointer) to avoid goroutine capturing mutable record.
// Includes List-Unsubscribe headers for native unsubscribe support in Gmail/Apple Mail.
// Gracefully skips if SMTP is not configured.
func sendWelcomeEmail(app *pocketbase.PocketBase, crypto *services.CryptoService, subID string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[newsletter] Welcome email panic: %v", r)
			}
		}()

		// SMTP graceful degradation — self-hosted may not have SMTP configured
		if app.Settings().SMTP.Host == "" {
			app.Logger().Info("newsletter: SMTP not configured, skipping welcome email", "subscriber_id", subID)
			return
		}

		// Re-fetch subscriber by ID to avoid stale data
		sub, err := app.FindRecordById("subscribers", subID)
		if err != nil {
			log.Printf("[newsletter] Failed to find subscriber %s for welcome email: %v", subID, err)
			return
		}

		// Look up profile for personalization and sender settings
		senderName := "Newsletter"
		welcomeReplyTo := ""
		profile, err := app.FindFirstRecordByFilter("profile", "1=1", nil)
		if err == nil && profile != nil {
			if customName := profile.GetString("newsletter_sender_name"); customName != "" {
				senderName = customName
			} else if name := profile.GetString("name"); name != "" {
				senderName = name
			}
			if customReplyTo := profile.GetString("newsletter_reply_to"); customReplyTo != "" {
				welcomeReplyTo = customReplyTo
			}
		}

		// Generate unsubscribe token and URL
		rawToken, err := ensureUnsubscribeToken(app, crypto, sub)
		if err != nil {
			log.Printf("[newsletter] Failed to generate unsubscribe token for %s: %v", sub.GetString("email"), err)
			return
		}
		unsubscribeURL := generateUnsubscribeURL(app, rawToken)
		oneClickURL := generateOneClickUnsubscribeURL(app, rawToken)

		recipientEmail := sub.GetString("email")
		recipientName := sub.GetString("name")
		safeSenderName := html.EscapeString(senderName)

		greeting := "Hi"
		if recipientName != "" {
			greeting = fmt.Sprintf("Hi %s", html.EscapeString(recipientName))
		}

		subject := fmt.Sprintf("Welcome to %s's newsletter", senderName)

		innerContent := fmt.Sprintf(
			`<p style="color: #374151; font-size: 16px; line-height: 1.6;">%s,</p>
<p style="color: #374151; font-size: 16px; line-height: 1.6;">Thanks for subscribing to <strong>%s</strong>'s newsletter! You'll receive updates and content directly to your inbox.</p>
<p style="color: #6B7280; font-size: 14px; line-height: 1.5;">If you didn't subscribe, you can <a href="%s" style="color: #2563eb;">unsubscribe here</a>.</p>`,
			greeting,
			safeSenderName,
			unsubscribeURL,
		)
		htmlBody := NewsletterEmailLayout(innerContent, unsubscribeURL)

		welcomeHeaders := map[string]string{
			"List-Unsubscribe":      "<" + oneClickURL + ">, <" + unsubscribeURL + ">",
			"List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
		}
		if welcomeReplyTo != "" {
			welcomeHeaders["Reply-To"] = welcomeReplyTo
		}

		message := &mailer.Message{
			From: mail.Address{
				Address: app.Settings().Meta.SenderAddress,
				Name:    senderName,
			},
			To:      []mail.Address{{Address: recipientEmail}},
			Subject: subject,
			HTML:    htmlBody,
			Headers: welcomeHeaders,
		}

		if sendErr := app.NewMailClient().Send(message); sendErr != nil {
			log.Printf("[newsletter] Failed to send welcome email to %s: %v", recipientEmail, sendErr)
		} else {
			log.Printf("[newsletter] Sent welcome email to %s", recipientEmail)
		}
	}()
}

// sendNewsletter sends a newsletter to all active subscribers in a goroutine.
// Accepts subscriber IDs (not pointers) to avoid goroutine capturing mutable records.
// Updates the send record with progress as it sends.
// Gracefully skips if SMTP is not configured.
func sendNewsletter(app *pocketbase.PocketBase, crypto *services.CryptoService, sendID, subject, bodyHTML, templateSlug string, subscriberIDs []string) {
	sentCount := 0
	failedCount := 0

	// Panic recovery - update send record on panic
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[newsletter] Send panic for %s: %v", sendID, r)
			if sendRecord, err := app.FindRecordById("newsletter_sends", sendID); err == nil {
				sendRecord.Set("sent_count", sentCount)
				sendRecord.Set("failed_count", failedCount)
				sendRecord.Set("status", "failed")
				sendRecord.Set("sent_at", time.Now().UTC().Format(time.RFC3339))
				_ = app.Save(sendRecord)
			}
		}
	}()

	// SMTP graceful degradation — self-hosted may not have SMTP configured
	if app.Settings().SMTP.Host == "" {
		app.Logger().Info("newsletter: SMTP not configured, skipping newsletter send")
		if sendRecord, err := app.FindRecordById("newsletter_sends", sendID); err == nil {
			sendRecord.Set("status", "failed")
			sendRecord.Set("sent_at", time.Now().UTC().Format(time.RFC3339))
			_ = app.Save(sendRecord)
		}
		return
	}

	// Strip dangerous protocols from user-provided HTML content
	safeBodyHTML := stripDangerousProtocols(bodyHTML)

	// Look up profile for custom sender name and reply-to
	fromName := app.Settings().Meta.SenderName
	replyTo := ""
	if profile, err := app.FindFirstRecordByFilter("profile", "1=1", nil); err == nil && profile != nil {
		if customName := profile.GetString("newsletter_sender_name"); customName != "" {
			fromName = customName
		}
		if customReplyTo := profile.GetString("newsletter_reply_to"); customReplyTo != "" {
			replyTo = customReplyTo
		}
	}

	for _, subID := range subscriberIDs {
		// Re-fetch subscriber by ID to get fresh data
		sub, err := app.FindRecordById("subscribers", subID)
		if err != nil {
			log.Printf("[newsletter] Failed to find subscriber %s: %v", subID, err)
			failedCount++
			continue
		}

		if sub.GetString("status") != "active" {
			continue
		}

		// Generate unsubscribe token for this subscriber
		rawToken, err := ensureUnsubscribeToken(app, crypto, sub)
		if err != nil {
			log.Printf("[newsletter] Failed to generate unsubscribe token for %s: %v", sub.GetString("email"), err)
			failedCount++
			continue
		}
		unsubscribeURL := generateUnsubscribeURL(app, rawToken)
		oneClickURL := generateOneClickUnsubscribeURL(app, rawToken)

		// Rewrite links for click tracking (skip unsubscribe links)
		trackedBody := rewriteLinksForTracking(app, safeBodyHTML, sendID, rawToken, unsubscribeURL)

		emailHTML := RenderNewsletterTemplate(templateSlug, trackedBody, unsubscribeURL, fromName)

		// Inject tracking pixel before closing </body> or at end
		pixelURL := fmt.Sprintf("%s/api/public/newsletter/open/%s/%s", strings.TrimRight(app.Settings().Meta.AppURL, "/"), sendID, rawToken)
		trackingPixel := fmt.Sprintf(`<img src="%s" width="1" height="1" alt="" style="display:none;" />`, pixelURL)
		if idx := strings.LastIndex(emailHTML, "</body>"); idx != -1 {
			emailHTML = emailHTML[:idx] + trackingPixel + emailHTML[idx:]
		} else {
			emailHTML += trackingPixel
		}

		headers := map[string]string{
			"List-Unsubscribe":      "<" + oneClickURL + ">, <" + unsubscribeURL + ">",
			"List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
		}
		if replyTo != "" {
			headers["Reply-To"] = replyTo
		}

		message := &mailer.Message{
			From: mail.Address{
				Address: app.Settings().Meta.SenderAddress,
				Name:    fromName,
			},
			To:      []mail.Address{{Address: sub.GetString("email")}},
			Subject: subject,
			HTML:    emailHTML,
			Headers: headers,
		}

		if sendErr := app.NewMailClient().Send(message); sendErr != nil {
			log.Printf("[newsletter] Failed to send to %s: %v", sub.GetString("email"), sendErr)
			failedCount++

			// Track consecutive bounces
			bounceCount := sub.GetInt("bounce_count") + 1
			sub.Set("bounce_count", bounceCount)
			if bounceCount >= 3 {
				sub.Set("status", "bounced")
				log.Printf("[newsletter] Auto-quarantined %s after %d consecutive failures", sub.GetString("email"), bounceCount)
			}
			if err := app.Save(sub); err != nil {
				log.Printf("[newsletter] Failed to update bounce count for %s: %v", sub.GetString("email"), err)
			}
		} else {
			sentCount++
			// Reset bounce count on successful delivery
			if sub.GetInt("bounce_count") > 0 {
				sub.Set("bounce_count", 0)
				if err := app.Save(sub); err != nil {
					log.Printf("[newsletter] Failed to reset bounce count for %s: %v", sub.GetString("email"), err)
				}
			}
		}

		// Rate limit: 100ms between sends
		time.Sleep(100 * time.Millisecond)
	}

	// Update send record with final counts
	sendRecord, err := app.FindRecordById("newsletter_sends", sendID)
	if err != nil {
		log.Printf("[newsletter] Failed to find send record %s: %v", sendID, err)
		return
	}

	sendRecord.Set("sent_count", sentCount)
	sendRecord.Set("failed_count", failedCount)
	sendRecord.Set("sent_at", time.Now().UTC().Format(time.RFC3339))
	if failedCount > 0 && sentCount == 0 {
		sendRecord.Set("status", "failed")
	} else {
		sendRecord.Set("status", "sent")
	}
	if err := app.Save(sendRecord); err != nil {
		log.Printf("[newsletter] Failed to update send record %s: %v", sendID, err)
	}

	log.Printf("[newsletter] Send %s complete: %d sent, %d failed", sendID, sentCount, failedCount)
}

// markdownToHTML converts markdown text to HTML using goldmark.
func markdownToHTML(markdown string) (string, error) {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(markdown), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// maskEmail masks an email address: "john@example.com" → "j***@example.com"
func maskEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return "***"
	}
	local := parts[0]
	if len(local) <= 1 {
		return local + "***@" + parts[1]
	}
	return string(local[0]) + "***@" + parts[1]
}
