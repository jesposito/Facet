package hooks

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"

	"facet/internal/textlimit"
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
// rl limits the public signup and confirmation routes.
func RegisterNewsletterHooks(app *pocketbase.PocketBase, crypto *services.CryptoService, rl *services.RateLimitService, planConfig *services.PlanConfig) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := recoverInterruptedNewsletters(app); err != nil {
			log.Printf("[newsletter] Failed to recover interrupted sends: %v", err)
		}
		return se.Next()
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		// POST /api/public/{slug}/subscribe — public newsletter signup
		//
		// Signing up only asks: the address owner gets a confirmation link
		// and is subscribed when they follow it. Someone who unsubscribed
		// stays unsubscribed until they confirm again themselves. The reply
		// is the same in every case, so the form doesn't reveal who is on
		// the list.
		se.Router.POST("/api/public/{slug}/subscribe", RateLimitMiddleware(rl, "moderate")(func(e *core.RequestEvent) error {
			slug := e.Request.PathValue("slug")

			// Honeypot check — bots fill hidden fields, humans don't
			if honeypot := e.Request.FormValue("website_url"); honeypot != "" {
				return e.JSON(http.StatusOK, map[string]string{"status": "confirmation_required"})
			}

			// Parse request body
			var req struct {
				Email  string `json:"email"`
				Name   string `json:"name"`
				Source string `json:"source"`
				// List is the id or slug of the list to join. Empty means
				// the default list.
				List string `json:"list"`
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
			// The owner's switch closes signup. Unsubscribing is never gated.
			if !services.IsFeatureEnabled(app, services.FeatureNewsletter, true) {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "newsletter is turned off"})
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

			// The list is checked before anything is stored. Asking for a
			// list that isn't open to the public is an error, never a quiet
			// signup to some other list.
			list, err := resolveSignupList(app, req.List)
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "this newsletter list isn't open for signup", "code": "list_unavailable"})
			}

			// Source defaults to "newsletter"
			source := req.Source
			if source == "" {
				source = "newsletter"
			}

			// Check for existing subscriber
			existing, err := app.FindRecordsByFilter(
				"subscribers",
				"email = {:email}",
				"",
				1,
				0,
				map[string]interface{}{"email": req.Email},
			)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to subscribe"})
			}

			confirmationRequired := map[string]string{"status": "confirmation_required"}

			var sub *core.Record
			if len(existing) > 0 {
				sub = existing[0]
				// Already on this list: nothing to confirm, and no email.
				// An active reader asking for another list still confirms
				// that list from their inbox first.
				if sub.GetString("status") == "active" && isListMember(app, sub.Id, list.Id) {
					return e.JSON(http.StatusOK, confirmationRequired)
				}
				// Unsubscribed, bounced or still pending keep their status
				// until the owner confirms.
			} else {
				collection, err := app.FindCollectionByNameOrId("subscribers")
				if err != nil {
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "subscribers collection not found"})
				}
				sub = core.NewRecord(collection)
				sub.Set("email", req.Email)
				sub.Set("name", textlimit.Runes(req.Name, 200))
				sub.Set("source", textlimit.Runes(source, 100))
				sub.Set("view_id", view.Id)
				sub.Set("status", "pending")
			}

			now := time.Now().UTC()
			if attemptedAt, err := time.Parse(time.RFC3339, sub.GetString("confirmation_attempted_at")); err == nil &&
				now.Sub(attemptedAt) < subscribeConfirmationAttemptCooldown {
				return e.JSON(http.StatusOK, confirmationRequired)
			}
			if sentAt, err := time.Parse(time.RFC3339, sub.GetString("confirmation_sent_at")); err == nil &&
				now.Sub(sentAt) < subscribeConfirmationCooldown {
				return e.JSON(http.StatusOK, confirmationRequired)
			}

			canSend := newsletterReady(app)
			// Without mail no new link can be issued, and the list an
			// existing link confirms must not change under it.
			if len(existing) > 0 && !canSend {
				return e.JSON(http.StatusOK, confirmationRequired)
			}
			var rawToken, tokenHash string
			if canSend {
				rawToken, err = crypto.GenerateToken(32)
				if err != nil {
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to subscribe"})
				}
				tokenHash = crypto.HMACToken(rawToken)
			}

			if len(existing) > 0 {
				// Only the signup's own fields are written, never the whole
				// record read above, so a confirmation or unsubscribe that
				// lands meanwhile is not undone. A new token is issued in the
				// same statement that claims the attempt, and only while both
				// cooldowns allow it.
				result, err := app.DB().NewQuery(`UPDATE subscribers SET confirmation_attempted_at = {:now},
					confirm_token_hash = {:hash}, pending_list_id = {:list}
					WHERE id = {:id}
					AND (confirmation_attempted_at = '' OR confirmation_attempted_at <= {:attempt_before})
					AND (confirmation_sent_at = '' OR confirmation_sent_at <= {:accepted_before})`).
					Bind(dbx.Params{
						"now": now.Format(time.RFC3339), "hash": tokenHash, "list": list.Id, "id": sub.Id,
						"attempt_before":  now.Add(-subscribeConfirmationAttemptCooldown).Format(time.RFC3339),
						"accepted_before": now.Add(-subscribeConfirmationCooldown).Format(time.RFC3339),
					}).Execute()
				if err != nil {
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to subscribe"})
				}
				claimed, err := result.RowsAffected()
				if err != nil || claimed != 1 {
					return e.JSON(http.StatusOK, confirmationRequired)
				}
			} else {
				if canSend {
					sub.Set("confirm_token_hash", tokenHash)
					sub.Set("confirmation_attempted_at", now.Format(time.RFC3339))
				}
				sub.Set("pending_list_id", list.Id)
				if err := app.Save(sub); err != nil {
					// A simultaneous first signup for the same address got
					// there first. Its confirmation is the one that goes out;
					// this request leaves that record alone.
					if winner, findErr := app.FindFirstRecordByData("subscribers", "email", req.Email); findErr == nil && winner.Id != sub.Id {
						return e.JSON(http.StatusOK, confirmationRequired)
					}
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to subscribe"})
				}
			}

			if canSend {
				if err := sendSubscribeConfirmation(app, sub, rawToken); err != nil {
					// The link may still have arrived; it stays valid from
					// confirmation_attempted_at.
					app.Logger().Warn("newsletter: confirmation email failed", "error", err, "subscriber_id", sub.Id)
				} else {
					// Scoped to the token just issued. If the link was already
					// used or replaced, no row matches and nothing changes.
					if _, err := app.DB().NewQuery(`UPDATE subscribers SET confirmation_sent_at = {:now}
						WHERE id = {:id} AND confirm_token_hash = {:hash}`).
						Bind(dbx.Params{"now": time.Now().UTC().Format(time.RFC3339), "id": sub.Id, "hash": tokenHash}).
						Execute(); err != nil {
						app.Logger().Warn("newsletter: failed to record accepted confirmation", "error", err, "subscriber_id", sub.Id)
					}
				}
			} else {
				app.Logger().Info("newsletter: email not configured, signup left pending", "subscriber_id", sub.Id)
			}

			return e.JSON(http.StatusOK, confirmationRequired)
		}))

		// GET /api/public/newsletter/confirm/{token} — the link in the
		// confirmation email. Subscribes the address and sends the welcome
		// email.
		se.Router.GET("/api/public/newsletter/confirm/{token}", RateLimitMiddleware(rl, "moderate")(func(e *core.RequestEvent) error {
			token := e.Request.PathValue("token")
			if token == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid or expired link"})
			}
			sub, err := app.FindFirstRecordByFilter(
				"subscribers",
				"confirm_token_hash = {:hash}",
				map[string]any{"hash": crypto.HMACToken(token)},
			)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "invalid or expired link"})
			}
			// A link lives from when it was issued, recorded before the
			// SMTP attempt, so one that arrived despite a dropped reply still
			// works. Links issued before that was recorded fall back to the
			// accepted time.
			issued := sub.GetString("confirmation_attempted_at")
			if issued == "" {
				issued = sub.GetString("confirmation_sent_at")
			}
			issuedAt, err := time.Parse(time.RFC3339, issued)
			if err != nil || time.Since(issuedAt) > subscribeConfirmationLifetime {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "invalid or expired link"})
			}

			// Join the list the signup asked for. Links sent before lists
			// were recorded on signups carry none and join the default list.
			listID := sub.GetString("pending_list_id")
			if listID == "" {
				if def, err := resolveSignupList(app, ""); err == nil {
					listID = def.Id
				}
			} else if list, err := app.FindRecordById("newsletter_lists", listID); err != nil || !list.GetBool("is_active") {
				// The list was removed or closed after the link went out.
				return e.JSON(http.StatusNotFound, map[string]string{"error": "this newsletter list is no longer open"})
			}

			// The update is conditional on the token and list read above, so
			// a link replaced or used meanwhile confirms nothing, and no other
			// field of the stale record is written back.
			err = app.RunInTransaction(func(txApp core.App) error {
				result, err := txApp.DB().NewQuery(`UPDATE subscribers SET status = 'active',
					confirm_token_hash = '', pending_list_id = '', bounce_count = 0
					WHERE id = {:id} AND confirm_token_hash = {:hash} AND pending_list_id = {:list}`).
					Bind(dbx.Params{
						"id": sub.Id, "hash": sub.GetString("confirm_token_hash"), "list": sub.GetString("pending_list_id"),
					}).Execute()
				if err != nil {
					return err
				}
				if n, err := result.RowsAffected(); err != nil {
					return err
				} else if n != 1 {
					return errNewsletterConfirmStale
				}
				if listID == "" {
					return nil
				}
				return ensureListMembership(txApp, sub.Id, listID)
			})
			if errors.Is(err, errNewsletterConfirmStale) {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "invalid or expired link"})
			}
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to subscribe"})
			}
			// A link sent before the newsletter was switched off still
			// records the reader's consent, but no welcome mail goes out
			// while the newsletter is off.
			if services.IsFeatureEnabled(app, services.FeatureNewsletter, true) {
				sendWelcomeEmail(app, crypto, sub.Id)
			}
			return e.Redirect(http.StatusFound, "/?newsletter=confirmed")
		}))

		// GET /api/public/newsletter/lists — the lists a visitor may join
		// from the signup form: active, and the default or marked public.
		// Private lists and subscriber counts are never listed.
		se.Router.GET("/api/public/newsletter/lists", func(e *core.RequestEvent) error {
			if planConfig.IsManaged() && !planConfig.HasFeature("newsletter") {
				return respondForbidden(e, "newsletter not available on current plan")
			}
			if !services.IsFeatureEnabled(app, services.FeatureNewsletter, true) {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "newsletter is turned off"})
			}
			records, err := app.FindRecordsByFilter(
				"newsletter_lists", "is_active = true && (is_default = true || is_public = true)", "-is_default,name", 0, 0,
			)
			if err != nil {
				records = []*core.Record{}
			}
			items := make([]map[string]any, 0, len(records))
			for _, r := range records {
				items = append(items, map[string]any{
					"id":          r.Id,
					"slug":        r.GetString("slug"),
					"name":        r.GetString("name"),
					"description": r.GetString("description"),
					"is_default":  r.GetBool("is_default"),
				})
			}
			return e.JSON(http.StatusOK, map[string]any{"data": items})
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

			records, err := app.FindRecordsByFilter("subscribers", filter, subscriberSort(app), perPage, offset, params)
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

			records, err := app.FindRecordsByFilter("subscribers", filter, subscriberSort(app), 0, 0, params)
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
			return respondJSON(e, http.StatusOK, map[string]any{
				"configured": smtpConfigured(app),
				"ready":      newsletterReady(app),
				"tested":     false,
				"delivery":   "unknown",
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

			if !services.IsFeatureEnabled(app, services.FeatureNewsletter, true) {
				return respondForbidden(e, "the newsletter feature is turned off")
			}

			var req struct {
				Subject      string `json:"subject"`
				BodyMarkdown string `json:"body_markdown"`
				TestEmail    string `json:"test_email"`
				Template     string `json:"template"`
				ListID       string `json:"list_id"`
				Personalize  bool   `json:"personalize"`
				// Resend is the explicit "send this again" action. Without
				// it, a newsletter matching one already sent to the same
				// audience is refused.
				Resend bool `json:"resend"`
				// RequestKey identifies one Send click. Retrying with the
				// same key returns the original send instead of starting
				// another; a deliberate resend uses a new key.
				RequestKey string `json:"request_key"`
			}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			}

			req.Subject = strings.TrimSpace(req.Subject)
			req.BodyMarkdown = strings.TrimSpace(req.BodyMarkdown)
			req.RequestKey = strings.TrimSpace(req.RequestKey)
			if req.Subject == "" || req.BodyMarkdown == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "subject and body_markdown are required"})
			}
			if req.RequestKey != "" && !newsletterRequestKeyPattern.MatchString(req.RequestKey) {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request_key"})
			}

			// Convert markdown to HTML
			bodyHTML, err := markdownToHTML(req.BodyMarkdown)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to render markdown"})
			}

			audience, err := selectNewsletterAudience(app, req.ListID)
			if err != nil {
				return respondNewsletterAudienceError(e, err)
			}
			personalization := newNewsletterPersonalization(app, req.Personalize, audience)
			bodyHTML, err = prepareNewsletterLinks(bodyHTML, personalization)
			if err != nil {
				return respondNewsletterLinkError(e, err)
			}

			// Test email mode
			if req.TestEmail != "" {
				if !emailRegex.MatchString(req.TestEmail) {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid test email"})
				}

				if !newsletterReady(app) {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "SMTP sender and absolute APP_URL must be configured"})
				}

				testSenderName := app.Settings().Meta.SenderName
				if profile, err := app.FindFirstRecordByFilter("profile", "1=1", nil); err == nil && profile != nil {
					if customName := profile.GetString("newsletter_sender_name"); customName != "" {
						testSenderName = customName
					}
				}
				// A personalized test shows what the first recipient of the
				// selected audience would get, the same sample as the preview.
				sampleName, sampleEmail := newsletterSampleRecipient(app, audience, req.TestEmail)
				testBody := personalization.HTML(bodyHTML, sampleName, sampleEmail)
				testHTML := RenderNewsletterTemplate(req.Template, testBody, "#", testSenderName)
				message := &mailer.Message{
					From: mail.Address{
						Address: app.Settings().Meta.SenderAddress,
						Name:    testSenderName,
					},
					To:      []mail.Address{{Address: req.TestEmail}},
					Subject: "[TEST] " + personalization.Subject(req.Subject, sampleName, sampleEmail),
					HTML:    testHTML,
				}

				if sendErr := app.NewMailClient().Send(message); sendErr != nil {
					app.Logger().Warn("newsletter: test email failed", "error", sendErr)
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to send test email"})
				}

				return e.JSON(http.StatusOK, map[string]string{"status": "test_sent"})
			}

			// Full send mode
			if !newsletterReady(app) {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "SMTP sender and absolute APP_URL must be configured"})
			}
			if _, err := newsletterSendInterval(); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}

			subscriberIDs := audience.SubscriberIDs
			if len(subscriberIDs) == 0 {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "no active subscribers"})
			}

			// Create send record
			sendCollection, err := app.FindCollectionByNameOrId("newsletter_sends")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "newsletter_sends collection not found"})
			}

			audienceKey := audience.key()
			contentHash := newsletterContentHash(req.Subject, bodyHTML)

			sendRecord := core.NewRecord(sendCollection)
			sendRecord.Set("subject", req.Subject)
			sendRecord.Set("body_markdown", req.BodyMarkdown)
			sendRecord.Set("body_html", bodyHTML)
			sendRecord.Set("status", "sending")
			sendRecord.Set("recipient_count", len(subscriberIDs))
			sendRecord.Set("sent_count", 0)
			sendRecord.Set("failed_count", 0)
			sendRecord.Set("request_key", req.RequestKey)
			sendRecord.Set("audience_key", audienceKey)
			sendRecord.Set("list_id", audience.listID())
			sendRecord.Set("list_name", audience.listName())
			sendRecord.Set("content_hash", contentHash)
			sendRecord.Set("active_lock", newsletterActiveLock)
			sendRecord.Set("pending_count", len(subscriberIDs))
			sendRecord.Set("template_slug", req.Template)
			sendRecord.Set("personalize", req.Personalize)

			// The checks and the new record share a transaction, and the
			// unique indexes on request_key and active_lock back them up in
			// the database, so two racing requests can't both start a send.
			var previous *core.Record
			err = app.RunInTransaction(func(txApp core.App) error {
				if req.RequestKey != "" {
					if prior, err := txApp.FindFirstRecordByFilter(
						"newsletter_sends", "request_key = {:key}", dbx.Params{"key": req.RequestKey},
					); err == nil && prior != nil {
						previous = prior
						if prior.GetString("content_hash") != contentHash || prior.GetString("audience_key") != audienceKey {
							return errNewsletterRequestKeyReused
						}
						return errNewsletterRequestReplayed
					}
				}
				activeSendCount, err := txApp.CountRecords("newsletter_sends",
					dbx.NewExp("status = 'sending' OR active_lock != ''"))
				if err != nil {
					return err
				}
				if activeSendCount > 0 {
					return errNewsletterSendInProgress
				}
				if !req.Resend {
					if prior := findPreviousNewsletterSend(txApp, req.Subject, bodyHTML, contentHash, audienceKey); prior != nil {
						previous = prior
						return errNewsletterAlreadySent
					}
				}
				if err := txApp.Save(sendRecord); err != nil {
					return err
				}
				return createNewsletterRecipients(txApp, sendRecord.Id, subscriberIDs)
			})
			if err != nil && previous == nil && !errors.Is(err, errNewsletterSendInProgress) {
				// A unique index refused the insert: another request with
				// this key, or another send, got there first.
				if req.RequestKey != "" {
					if prior, findErr := app.FindFirstRecordByFilter(
						"newsletter_sends", "request_key = {:key}", dbx.Params{"key": req.RequestKey},
					); findErr == nil && prior != nil {
						previous = prior
						err = errNewsletterRequestReplayed
						if prior.GetString("content_hash") != contentHash || prior.GetString("audience_key") != audienceKey {
							err = errNewsletterRequestKeyReused
						}
					}
				}
				if previous == nil {
					if n, _ := app.CountRecords("newsletter_sends", dbx.NewExp("active_lock != ''")); n > 0 {
						err = errNewsletterSendInProgress
					}
				}
			}
			switch {
			case errors.Is(err, errNewsletterRequestReplayed):
				// A retry of a request that already started: report that
				// send, and send nothing new.
				return e.JSON(http.StatusOK, map[string]any{
					"status":          previous.GetString("status"),
					"send_id":         previous.Id,
					"recipient_count": previous.GetInt("recipient_count"),
					"list_name":       previous.GetString("list_name"),
					"replayed":        true,
				})
			case errors.Is(err, errNewsletterRequestKeyReused):
				return e.JSON(http.StatusConflict, map[string]string{
					"error":   "this request key was already used for a different newsletter",
					"code":    "request_key_reused",
					"send_id": previous.Id,
				})
			case errors.Is(err, errNewsletterSendInProgress):
				return e.JSON(http.StatusConflict, map[string]string{"error": "a newsletter send is already in progress", "code": "send_in_progress"})
			case errors.Is(err, errNewsletterAlreadySent):
				return e.JSON(http.StatusConflict, map[string]string{
					"error":   "this newsletter has already been sent to this audience; confirm a resend to send it again",
					"code":    "already_sent",
					"send_id": previous.Id,
				})
			case err != nil:
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create send record"})
			}

			// Spawn bounded background sender (pass IDs, not mutable pointers).
			goSendNewsletter(app, crypto, sendRecord.Id, req.Subject, bodyHTML, req.Template, subscriberIDs, personalization)

			return e.JSON(http.StatusOK, map[string]interface{}{
				"status":          "sending",
				"send_id":         sendRecord.Id,
				"recipient_count": len(subscriberIDs),
				"list_name":       audience.listName(),
			})
		})

		se.Router.POST("/api/admin/newsletter/sends/{id}/resume", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			if planConfig.IsManaged() && !planConfig.HasFeature("newsletter") {
				return respondForbidden(e, "newsletter not available on current plan")
			}
			if err := resumeNewsletter(app, crypto, e.Request.PathValue("id")); err != nil {
				switch {
				case errors.Is(err, errNewsletterFeatureOff):
					return respondForbidden(e, err.Error())
				case errors.Is(err, errNewsletterSendInProgress):
					return e.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
				case errors.Is(err, sql.ErrNoRows):
					return e.JSON(http.StatusNotFound, map[string]string{"error": "send not found"})
				}
				return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}
			return e.JSON(http.StatusOK, map[string]string{"status": "sending", "send_id": e.Request.PathValue("id")})
		})

		// POST /api/admin/newsletter/preview — the audience a send would
		// reach and what its first recipient would see. The compose page
		// uses this for both the confirmation count and the personalized
		// preview, so they match the send exactly.
		se.Router.POST("/api/admin/newsletter/preview", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			if planConfig.IsManaged() && !planConfig.HasFeature("newsletter") {
				return respondForbidden(e, "newsletter not available on current plan")
			}

			var req struct {
				Subject      string `json:"subject"`
				BodyMarkdown string `json:"body_markdown"`
				ListID       string `json:"list_id"`
				Personalize  bool   `json:"personalize"`
			}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			}

			audience, err := selectNewsletterAudience(app, req.ListID)
			if err != nil {
				return respondNewsletterAudienceError(e, err)
			}
			personalization := newNewsletterPersonalization(app, req.Personalize, audience)
			bodyHTML, err := markdownToHTML(strings.TrimSpace(req.BodyMarkdown))
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to render markdown"})
			}
			// The preview refuses the same links the send would, so the
			// problem shows up while writing.
			bodyHTML, err = prepareNewsletterLinks(bodyHTML, personalization)
			if err != nil {
				return respondNewsletterLinkError(e, err)
			}
			sampleName, sampleEmail := newsletterSampleRecipient(app, audience, "")

			return e.JSON(http.StatusOK, map[string]any{
				"recipient_count": len(audience.SubscriberIDs),
				"list_name":       audience.listName(),
				"sample_email":    sampleEmail,
				"subject":         personalization.Subject(strings.TrimSpace(req.Subject), sampleName, sampleEmail),
				"body_html":       stripDangerousProtocols(personalization.HTML(bodyHTML, sampleName, sampleEmail)),
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

			records, err := app.FindRecordsByFilter("newsletter_sends", "id != ''", "-created,-id", 50, 0)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load send history"})
			}
			ids := make([]string, len(records))
			for i, r := range records {
				ids[i] = r.Id
			}
			resumable, err := newsletterResumableCounts(app, ids)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load send history"})
			}

			sends := make([]map[string]interface{}, 0, len(records))
			for _, r := range records {
				sends = append(sends, map[string]interface{}{
					"id":              r.Id,
					"subject":         r.GetString("subject"),
					"status":          r.GetString("status"),
					"recipient_count": r.GetInt("recipient_count"),
					"list_id":         r.GetString("list_id"),
					"list_name":       r.GetString("list_name"),
					"sent_count":      r.GetInt("sent_count"),
					"failed_count":    r.GetInt("failed_count"),
					"pending_count":   r.GetInt("pending_count"),
					"uncertain_count": r.GetInt("uncertain_count"),
					"skipped_count":   r.GetInt("skipped_count"),
					"resumable_count": resumable[r.Id],
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

			if err := deleteNewsletterSend(app, e.Request.PathValue("id")); err != nil {
				switch {
				case errors.Is(err, sql.ErrNoRows):
					return e.JSON(http.StatusNotFound, map[string]string{"error": "send record not found"})
				case errors.Is(err, errNewsletterSendInProgress):
					return e.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
				}
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

			if err := unsubscribeSubscriber(app, sub.Id); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to unsubscribe"})
			}

			log.Printf("[newsletter] Subscriber %s unsubscribed via token", sub.GetString("email"))
			return e.JSON(http.StatusOK, map[string]string{"status": "unsubscribed"})
		})

		// Address-only requests cannot prove inbox ownership. Signed links in
		// newsletter footers and the RFC 8058 route remain the mutation paths.
		se.Router.POST("/api/public/unsubscribe/by-email", RateLimitMiddleware(rl, "moderate")(func(e *core.RequestEvent) error {
			return e.JSON(http.StatusOK, map[string]string{
				"status":  "confirmation_required",
				"message": "Use the signed unsubscribe link from a newsletter email.",
			})
		}))

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

			if err := unsubscribeSubscriber(app, sub.Id); err != nil {
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
				recordNewsletterEvent(app, crypto, sendID, token, "open", "")
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
			parsed, err := url.Parse(targetURL)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid url"})
			}
			// Only follow links that were in this newsletter. Otherwise the
			// route is an open redirect on the site's own domain.
			send, err := app.FindRecordById("newsletter_sends", sendID)
			if err != nil || !newsletterLinksTo(send.GetString("body_html"), targetURL) {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "link not found"})
			}
			if token != "" {
				recordNewsletterEvent(app, crypto, sendID, token, "click", targetURL)
			}
			// The link was taken from HTML, where & is written &amp;.
			return e.Redirect(http.StatusFound, html.UnescapeString(targetURL))
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
			resumable, err := newsletterResumableCounts(app, []string{send.Id})
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load send statistics"})
			}
			return e.JSON(http.StatusOK, map[string]interface{}{
				"send_id":         send.Id,
				"open_count":      send.GetInt("open_count"),
				"click_count":     send.GetInt("click_count"),
				"total_sent":      send.GetInt("sent_count"),
				"accepted_count":  send.GetInt("sent_count"),
				"failed_count":    send.GetInt("failed_count"),
				"pending_count":   send.GetInt("pending_count"),
				"uncertain_count": send.GetInt("uncertain_count"),
				"skipped_count":   send.GetInt("skipped_count"),
				"resumable_count": resumable[send.Id],
			})
		})

		return se.Next()
	})
}

// unsubscribeSubscriber writes only the status of a subscriber whose signed
// link was already verified, so a confirmation token or consent recorded
// since the record was read is kept.
func unsubscribeSubscriber(app core.App, id string) error {
	_, err := app.DB().NewQuery(`UPDATE subscribers SET status = 'unsubscribed' WHERE id = {:id}`).
		Bind(dbx.Params{"id": id}).Execute()
	return err
}

var (
	errNewsletterSendInProgress   = errors.New("a newsletter send is already in progress")
	errNewsletterAlreadySent      = errors.New("this newsletter has already been sent")
	errNewsletterRequestReplayed  = errors.New("this send request was already accepted")
	errNewsletterRequestKeyReused = errors.New("request key used for a different newsletter")
	errNewsletterFeatureOff       = errors.New("the newsletter feature is turned off")
	errNewsletterConfirmStale     = errors.New("confirmation link was used or replaced")
)

// newsletterActiveLock is the active_lock value while a send runs. A partial
// unique index allows only one row to hold it.
const newsletterActiveLock = "sending"

// findPreviousNewsletterSend returns a send that already delivered, or may
// have delivered, the same newsletter to the same audience, or nil. A send
// whose only outcomes are uncertain counts: its server may have accepted
// those messages.
//
// Sends recorded before audiences were stored have a blank audience_key and
// content_hash. Nobody knows who those went to, so they're matched by
// subject and body against any audience: the owner confirms a resend rather
// than risk mailing the same people twice.
func findPreviousNewsletterSend(app core.App, subject, bodyHTML, contentHash, audienceKey string) *core.Record {
	prior, err := app.FindFirstRecordByFilter(
		"newsletter_sends",
		"(status = 'sent' || sent_count > 0 || uncertain_count > 0) && ("+
			"(content_hash = {:hash} && audience_key = {:audience}) || "+
			"(audience_key = '' && subject = {:subject} && body_html = {:body}))",
		dbx.Params{"hash": contentHash, "audience": audienceKey, "subject": subject, "body": bodyHTML},
	)
	if err != nil {
		return nil
	}
	return prior
}

func respondNewsletterLinkError(e *core.RequestEvent, err error) error {
	code := "link_placeholder"
	if errors.Is(err, errPerRecipientLink) {
		code = "per_recipient_link"
	}
	return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": code})
}

// subscriberSort orders subscribers newest first when the collection has a
// created field. Some installs' subscribers collection doesn't, and sorting
// on a missing field makes the whole query fail and the list look empty.
func subscriberSort(app core.App) string {
	if c, err := app.FindCachedCollectionByNameOrId("subscribers"); err == nil && c.Fields.GetByName("created") != nil {
		return "-created"
	}
	return ""
}

func respondNewsletterAudienceError(e *core.RequestEvent, err error) error {
	switch {
	case errors.Is(err, errNewsletterListNotFound):
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "newsletter list not found"})
	case errors.Is(err, errNewsletterListInactive):
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "newsletter list is inactive"})
	}
	return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load recipients"})
}

// newsletterSampleRecipient returns the first recipient of an audience for
// previews. fallbackEmail is used when the audience is empty.
func newsletterSampleRecipient(app core.App, audience *newsletterAudience, fallbackEmail string) (name, email string) {
	if audience != nil && len(audience.SubscriberIDs) > 0 {
		if sub, err := app.FindRecordById("subscribers", audience.SubscriberIDs[0]); err == nil {
			return sub.GetString("name"), sub.GetString("email")
		}
	}
	return "", fallbackEmail
}

// smtpConfigured reports whether the site can send email. PocketBase's
// default settings carry a placeholder host, so the host alone doesn't say
// SMTP was set up; the Enabled switch does.
func smtpConfigured(app core.App) bool {
	smtp := app.Settings().SMTP
	return smtp.Enabled && smtp.Host != ""
}

const (
	subscribeConfirmationAttemptCooldown = time.Minute
	// subscribeConfirmationCooldown starts only after SMTP accepts the message.
	subscribeConfirmationCooldown = 24 * time.Hour
	// subscribeConfirmationLifetime is how long a confirmation link works.
	subscribeConfirmationLifetime = 7 * 24 * time.Hour
)

// sendSubscribeConfirmation emails the link that completes a signup.
func sendSubscribeConfirmation(app core.App, sub *core.Record, rawToken string) error {
	senderName := app.Settings().Meta.SenderName
	if profile, err := app.FindFirstRecordByFilter("profile", "1=1", nil); err == nil && profile != nil {
		if customName := profile.GetString("newsletter_sender_name"); customName != "" {
			senderName = customName
		} else if name := profile.GetString("name"); name != "" {
			senderName = name
		}
	}
	link := fmt.Sprintf("%s/api/public/newsletter/confirm/%s", publicAppURL(app), url.PathEscape(rawToken))
	safeSender := html.EscapeString(senderName)

	inner := fmt.Sprintf(
		`<p style="color: #374151; font-size: 16px; line-height: 1.6;">Someone asked to subscribe this address to <strong>%s</strong>'s newsletter.</p>
<p style="color: #374151; font-size: 16px; line-height: 1.6;"><a href="%s" style="color: #2563eb;">Confirm your subscription</a></p>
<p style="color: #6B7280; font-size: 14px; line-height: 1.5;">If this wasn't you, ignore this email. You won't be subscribed or hear from us again.</p>`,
		safeSender, html.EscapeString(link),
	)

	message := &mailer.Message{
		From: mail.Address{
			Address: app.Settings().Meta.SenderAddress,
			Name:    senderName,
		},
		To:      []mail.Address{{Address: sub.GetString("email")}},
		Subject: fmt.Sprintf("Confirm your subscription to %s's newsletter", senderName),
		HTML:    EmailLayout(inner),
		Text: fmt.Sprintf("Someone asked to subscribe this address to %s's newsletter.\n\nConfirm your subscription: %s\n\nIf this wasn't you, ignore this email. You won't be subscribed or hear from us again.\n",
			senderName, link),
	}
	return app.NewMailClient().Send(message)
}

// newsletterLinksTo reports whether a sent newsletter contains a tracked
// link to target. The click route only redirects to those, so its URLs
// can't be used to bounce visitors to an arbitrary site.
func newsletterLinksTo(bodyHTML, target string) bool {
	for _, match := range linkHrefRegex.FindAllStringSubmatch(stripDangerousProtocols(bodyHTML), -1) {
		if len(match) >= 3 && match[2] == target {
			return true
		}
	}
	return false
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

// errDuplicateNewsletterEvent rolls back an event the subscriber already
// recorded for this send.
var errDuplicateNewsletterEvent = errors.New("duplicate newsletter event")

// recordNewsletterEvent records an open or click event and bumps the
// matching count on the send. It runs inside the request, so nothing touches
// the database after the response or the app has finished. The event insert
// and the count update share one transaction: the unique index on
// (send_id, subscriber_id, event_type) lets only the first event through, and
// the count moves only when that insert commits. Two clicks arriving together
// therefore count once.
func recordNewsletterEvent(app core.App, crypto *services.CryptoService, sendID, token, eventType, targetURL string) {
	// Validate eventType to prevent SQL injection via countField concatenation
	countField, ok := newsletterCountFieldFor(eventType)
	if !ok {
		log.Printf("[newsletter] Invalid event type: %s", eventType)
		return
	}

	// Resolve subscriber from token
	sub, err := findSubscriberByToken(app, crypto, token)
	if err != nil {
		return // invalid token — silently ignore
	}

	eventsCollection, err := app.FindCollectionByNameOrId("newsletter_events")
	if err != nil {
		log.Printf("[newsletter] Events collection not found: %v", err)
		return
	}

	// Map open/click to stored enum values
	storedEventType := "opened"
	if eventType == "click" {
		storedEventType = "clicked"
	}

	err = app.RunInTransaction(func(txApp core.App) error {
		event := core.NewRecord(eventsCollection)
		event.Set("send_id", sendID)
		event.Set("subscriber_id", sub.Id)
		event.Set("event_type", storedEventType)
		if targetURL != "" {
			event.Set("url", targetURL)
		}
		if err := txApp.Save(event); err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return errDuplicateNewsletterEvent
			}
			return fmt.Errorf("save event: %w", err)
		}
		_, err := txApp.DB().NewQuery("UPDATE newsletter_sends SET " + countField + " = " + countField + " + 1 WHERE id = {:id}").
			Bind(dbx.Params{"id": sendID}).
			Execute()
		if err != nil {
			return fmt.Errorf("update send %s: %w", countField, err)
		}
		return nil
	})
	if err != nil && !errors.Is(err, errDuplicateNewsletterEvent) {
		log.Printf("[newsletter] Failed to record %s event: %v", eventType, err)
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

// sendWelcomeEmail sends a branded welcome email to a new subscriber.
// It runs inside the confirmation request, the same way the confirmation
// email itself is sent, so it finishes before the response does and never
// outlives the app. Failures are logged; the subscription stands either way.
// Includes List-Unsubscribe headers for native unsubscribe support in Gmail/Apple Mail.
// Gracefully skips if SMTP is not configured.
func sendWelcomeEmail(app *pocketbase.PocketBase, crypto *services.CryptoService, subID string) {
	// SMTP graceful degradation — self-hosted may not have SMTP configured
	if !smtpConfigured(app) {
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
}

// sendNewsletter sends a newsletter to all active subscribers in a goroutine.
// Accepts subscriber IDs (not pointers) to avoid goroutine capturing mutable records.
// Updates the send record with progress as it sends.
// Gracefully skips if SMTP is not configured.
func sendNewsletter(app *pocketbase.PocketBase, crypto *services.CryptoService, sendID, subject, bodyHTML, templateSlug string, subscriberIDs []string, personalization *newsletterPersonalization) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[newsletter] Send panic for %s: %v", sendID, r)
			if err := markInflightUncertain(app, sendID, "sender stopped after the SMTP attempt began"); err == nil {
				_ = syncNewsletterProgress(app, sendID, true)
			}
		}
	}()
	if !newsletterReady(app) {
		_ = syncNewsletterProgress(app, sendID, true)
		return
	}
	interval, err := newsletterSendInterval()
	if err != nil {
		_ = syncNewsletterProgress(app, sendID, true)
		return
	}
	safeBodyHTML := stripDangerousProtocols(bodyHTML)
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

	if err := deliverNewsletterRecipients(app, crypto, sendID, subject, safeBodyHTML, templateSlug, fromName, replyTo, interval, personalization); err != nil {
		log.Printf("[newsletter] Send %s stopped: %v", sendID, err)
		// Whatever was in flight when progress stopped saving may have been
		// submitted. If this write fails too, the active lock stays so no
		// second send can start, and startup recovery repeats this step.
		if markErr := markInflightUncertain(app, sendID, "progress could not be saved after the SMTP attempt"); markErr != nil {
			log.Printf("[newsletter] Failed to mark in-flight recipients uncertain for %s: %v", sendID, markErr)
			return
		}
	}
	if err := syncNewsletterProgress(app, sendID, true); err != nil {
		log.Printf("[newsletter] Failed to finish send record %s: %v", sendID, err)
	}
}

// deliverNewsletterRecipients submits each unsent recipient of a send once.
// It returns at the first outcome it can't record, leaving that recipient in
// flight for the caller to mark uncertain.
func deliverNewsletterRecipients(app *pocketbase.PocketBase, crypto *services.CryptoService, sendID, subject, safeBodyHTML, templateSlug, fromName, replyTo string, interval time.Duration, personalization *newsletterPersonalization) error {
	recipients, err := resumableNewsletterRecipients(app, sendID)
	if err != nil {
		return fmt.Errorf("load recipients: %w", err)
	}
	finish := func(recipientID, state, message string) error {
		if err := completeNewsletterRecipient(app, recipientID, state, message); err != nil {
			return fmt.Errorf("%w: record %s for %s: %v", errNewsletterPersistence, state, recipientID, err)
		}
		if err := syncNewsletterProgress(app, sendID, false); err != nil {
			return fmt.Errorf("%w: progress: %v", errNewsletterPersistence, err)
		}
		return nil
	}
	submitted := false
	for _, recipient := range recipients {
		claimed, err := claimNewsletterRecipient(app, recipient.Id)
		if err != nil {
			return fmt.Errorf("%w: claim %s: %v", errNewsletterPersistence, recipient.Id, err)
		}
		if !claimed {
			continue
		}
		sub, err := app.FindRecordById("subscribers", recipient.GetString("subscriber_id"))
		if errors.Is(err, sql.ErrNoRows) {
			if err := finish(recipient.Id, "skipped", "subscriber no longer exists"); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			// Nothing was submitted, so the recipient goes back to pending.
			_, _ = app.DB().NewQuery(`UPDATE newsletter_recipients SET state = 'pending'
				WHERE id = {:id} AND state = 'in-flight'`).Bind(dbx.Params{"id": recipient.Id}).Execute()
			return fmt.Errorf("load subscriber: %w", err)
		}
		if sub.GetString("status") != "active" {
			if err := finish(recipient.Id, "skipped", "subscriber is no longer active"); err != nil {
				return err
			}
			continue
		}
		rawToken, err := ensureUnsubscribeToken(app, crypto, sub)
		if err != nil {
			if err := finish(recipient.Id, "failed", "unsubscribe token: "+err.Error()); err != nil {
				return err
			}
			continue
		}
		unsubscribeURL := generateUnsubscribeURL(app, rawToken)
		oneClickURL := generateOneClickUnsubscribeURL(app, rawToken)
		trackedBody := rewriteLinksForTracking(app, safeBodyHTML, sendID, rawToken, unsubscribeURL)
		recipientName, recipientEmail := sub.GetString("name"), sub.GetString("email")
		trackedBody = personalization.HTML(trackedBody, recipientName, recipientEmail)
		emailHTML := RenderNewsletterTemplate(templateSlug, trackedBody, unsubscribeURL, fromName)
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
			From:    mail.Address{Address: app.Settings().Meta.SenderAddress, Name: fromName},
			To:      []mail.Address{{Address: recipientEmail}},
			Subject: personalization.Subject(subject, recipientName, recipientEmail),
			HTML:    emailHTML,
			Headers: headers,
		}
		if submitted && interval > 0 {
			newsletterPace(interval)
		}
		submitted = true
		// SMTP failures describe the transport, not the reader, so they
		// never change subscriber status or bounce counts.
		state, errorMessage := "accepted", ""
		sendErr := app.NewMailClient().Send(message)
		if sendErr != nil {
			state = smtpFailureState(sendErr)
			errorMessage = sendErr.Error()
			log.Printf("[newsletter] SMTP submission to %s ended %s: %v", recipientEmail, state, sendErr)
		}
		if err := finish(recipient.Id, state, errorMessage); err != nil {
			return err
		}
		// A relay that is down or refuses our login would fail everyone the
		// same way. Stop here and leave the rest pending for Resume.
		if sendErr != nil && smtpRelayUnavailable(sendErr) {
			log.Printf("[newsletter] Send %s paused: owner SMTP relay unavailable", sendID)
			return nil
		}
	}
	return nil
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
