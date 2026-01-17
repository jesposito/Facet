package hooks

import (
	"net/http"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterTestimonialHooks(app *pocketbase.PocketBase, testimonial *services.TestimonialService, rl *services.RateLimitService) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		se.Router.POST("/api/testimonials/requests", func(e *core.RequestEvent) error {
			var req struct {
				Label          string  `json:"label"`
				CustomMessage  string  `json:"custom_message"`
				RecipientName  string  `json:"recipient_name"`
				RecipientEmail string  `json:"recipient_email"`
				ExpiresAt      *string `json:"expires_at"`
				MaxUses        int     `json:"max_uses"`
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			rawToken, err := testimonial.GenerateToken()
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
			}

			tokenHMAC := testimonial.HMACToken(rawToken)
			tokenPrefix := testimonial.TokenPrefix(rawToken)

			collection, err := app.FindCollectionByNameOrId("testimonial_requests")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "collection not found"})
			}

			record := core.NewRecord(collection)
			record.Set("token_hash", tokenHMAC)
			record.Set("token_prefix", tokenPrefix)
			record.Set("label", req.Label)
			record.Set("custom_message", req.CustomMessage)
			record.Set("recipient_name", req.RecipientName)
			record.Set("recipient_email", req.RecipientEmail)
			record.Set("is_active", true)
			record.Set("use_count", 0)

			if req.ExpiresAt != nil && *req.ExpiresAt != "" {
				expiresAt, err := time.Parse("2006-01-02T15:04", *req.ExpiresAt)
				if err != nil {
					expiresAt, err = time.Parse(time.RFC3339, *req.ExpiresAt)
				}
				if err != nil {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid expiration date format"})
				}
				record.Set("expires_at", expiresAt)
			}

			if req.MaxUses > 0 {
				record.Set("max_uses", req.MaxUses)
			}

			if err := app.Save(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save request"})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"id":    record.Id,
				"token": rawToken,
				"label": req.Label,
			})
		}).Bind(apis.RequireAuth())

		se.Router.GET("/api/testimonials/requests", func(e *core.RequestEvent) error {
			records, err := app.FindRecordsByFilter(
				"testimonial_requests",
				"1=1",
				"-created",
				100,
				0,
			)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch requests"})
			}

			var result []map[string]interface{}
			for _, r := range records {
				result = append(result, map[string]interface{}{
					"id":              r.Id,
					"label":           r.GetString("label"),
					"custom_message":  r.GetString("custom_message"),
					"recipient_name":  r.GetString("recipient_name"),
					"recipient_email": r.GetString("recipient_email"),
					"expires_at":      r.GetDateTime("expires_at"),
					"max_uses":        r.GetInt("max_uses"),
					"use_count":       r.GetInt("use_count"),
					"is_active":       r.GetBool("is_active"),
					"created":         r.GetDateTime("created"),
				})
			}

			return e.JSON(http.StatusOK, result)
		}).Bind(apis.RequireAuth())

		se.Router.DELETE("/api/testimonials/requests/{id}", func(e *core.RequestEvent) error {
			id := e.Request.PathValue("id")
			record, err := app.FindRecordById("testimonial_requests", id)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "request not found"})
			}

			if err := app.Delete(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete request"})
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
		}).Bind(apis.RequireAuth())

		se.Router.GET("/api/testimonials/request/{token}", RateLimitMiddleware(rl, "moderate")(func(e *core.RequestEvent) error {
			token := e.Request.PathValue("token")
			invalidResponse := services.TestimonialRequestValidation{
				Valid: false,
				Error: "invalid token",
			}

			tokenHMAC := testimonial.HMACToken(token)
			record, err := app.FindFirstRecordByFilter(
				"testimonial_requests",
				"token_hash = {:hash} && is_active = true",
				map[string]interface{}{"hash": tokenHMAC},
			)
			if err != nil || record == nil {
				return e.JSON(http.StatusOK, invalidResponse)
			}

			expiresAt := record.GetDateTime("expires_at")
			if !expiresAt.IsZero() && time.Now().After(expiresAt.Time()) {
				return e.JSON(http.StatusOK, invalidResponse)
			}

			useCount := record.GetInt("use_count")
			maxUses := record.GetInt("max_uses")
			if maxUses > 0 && useCount >= maxUses {
				return e.JSON(http.StatusOK, invalidResponse)
			}

			profile, _ := app.FindFirstRecordByFilter("profile", "1=1", nil)
			var profileName, profileHeadline, profileAvatar string
			if profile != nil {
				profileName = profile.GetString("name")
				profileHeadline = profile.GetString("headline")
				if avatar := profile.GetString("avatar"); avatar != "" {
					profileAvatar = "/api/files/profile/" + profile.Id + "/" + avatar
				}
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"valid":            true,
				"request_id":       record.Id,
				"label":            record.GetString("label"),
				"custom_message":   record.GetString("custom_message"),
				"recipient_name":   record.GetString("recipient_name"),
				"profile_name":     profileName,
				"profile_headline": profileHeadline,
				"profile_avatar":   profileAvatar,
			})
		}))

		se.Router.POST("/api/testimonials/submit", RateLimitMiddleware(rl, "strict")(func(e *core.RequestEvent) error {
			var req services.TestimonialSubmission
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			if req.AuthorName == "" || req.Content == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "author_name and content are required"})
			}

			var requestRecord *core.Record
			if req.RequestToken != "" {
				tokenHMAC := testimonial.HMACToken(req.RequestToken)
				requestRecord, _ = app.FindFirstRecordByFilter(
					"testimonial_requests",
					"token_hash = {:hash} && is_active = true",
					map[string]interface{}{"hash": tokenHMAC},
				)

				if requestRecord != nil {
					expiresAt := requestRecord.GetDateTime("expires_at")
					if !expiresAt.IsZero() && time.Now().After(expiresAt.Time()) {
						requestRecord = nil
					}

					if requestRecord != nil {
						useCount := requestRecord.GetInt("use_count")
						maxUses := requestRecord.GetInt("max_uses")
						if maxUses > 0 && useCount >= maxUses {
							requestRecord = nil
						}
					}
				}
			}

			collection, err := app.FindCollectionByNameOrId("testimonials")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "collection not found"})
			}

			record := core.NewRecord(collection)
			record.Set("content", req.Content)
			record.Set("relationship", req.Relationship)
			record.Set("project", req.Project)
			record.Set("author_name", req.AuthorName)
			record.Set("author_title", req.AuthorTitle)
			record.Set("author_company", req.AuthorCompany)
			record.Set("author_website", req.AuthorWebsite)
			record.Set("verification_method", "none")
			record.Set("status", "pending")
			record.Set("submitted_at", time.Now())
			record.Set("featured", false)
			record.Set("sort_order", 0)

			if requestRecord != nil {
				record.Set("request_id", requestRecord.Id)

				requestRecord.Set("use_count", requestRecord.GetInt("use_count")+1)
				app.Save(requestRecord)
			}

			if err := app.Save(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save testimonial"})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"id":     record.Id,
				"status": "pending",
			})
		}))

		se.Router.GET("/api/testimonials", func(e *core.RequestEvent) error {
			status := e.Request.URL.Query().Get("status")

			var filter string
			var params map[string]interface{}

			if status != "" {
				filter = "status = {:status}"
				params = map[string]interface{}{"status": status}
			} else {
				filter = "1=1"
				params = nil
			}

			records, err := app.FindRecordsByFilter(
				"testimonials",
				filter,
				"-created",
				100,
				0,
				params,
			)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch testimonials"})
			}

			var result []map[string]interface{}
			for _, r := range records {
				result = append(result, map[string]interface{}{
					"id":                      r.Id,
					"content":                 r.GetString("content"),
					"relationship":            r.GetString("relationship"),
					"project":                 r.GetString("project"),
					"author_name":             r.GetString("author_name"),
					"author_title":            r.GetString("author_title"),
					"author_company":          r.GetString("author_company"),
					"author_website":          r.GetString("author_website"),
					"author_photo":            r.GetString("author_photo"),
					"verification_method":     r.GetString("verification_method"),
					"verification_identifier": r.GetString("verification_identifier"),
					"verified_at":             r.GetDateTime("verified_at"),
					"status":                  r.GetString("status"),
					"submitted_at":            r.GetDateTime("submitted_at"),
					"approved_at":             r.GetDateTime("approved_at"),
					"featured":                r.GetBool("featured"),
					"sort_order":              r.GetInt("sort_order"),
					"created":                 r.GetDateTime("created"),
				})
			}

			return e.JSON(http.StatusOK, result)
		}).Bind(apis.RequireAuth())

		se.Router.POST("/api/testimonials/{id}/approve", func(e *core.RequestEvent) error {
			id := e.Request.PathValue("id")
			record, err := app.FindRecordById("testimonials", id)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "testimonial not found"})
			}

			record.Set("status", "approved")
			record.Set("approved_at", time.Now())

			if err := app.Save(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to approve"})
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "approved"})
		}).Bind(apis.RequireAuth())

		se.Router.POST("/api/testimonials/{id}/reject", func(e *core.RequestEvent) error {
			id := e.Request.PathValue("id")
			record, err := app.FindRecordById("testimonials", id)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "testimonial not found"})
			}

			var req struct {
				Reason string `json:"reason"`
			}
			e.BindBody(&req)

			record.Set("status", "rejected")
			record.Set("rejected_at", time.Now())
			record.Set("rejection_reason", req.Reason)

			if err := app.Save(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to reject"})
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "rejected"})
		}).Bind(apis.RequireAuth())

		se.Router.PATCH("/api/testimonials/{id}", func(e *core.RequestEvent) error {
			id := e.Request.PathValue("id")
			record, err := app.FindRecordById("testimonials", id)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "testimonial not found"})
			}

			var req struct {
				Content   *string `json:"content"`
				Featured  *bool   `json:"featured"`
				SortOrder *int    `json:"sort_order"`
				Status    *string `json:"status"`
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			if req.Content != nil {
				record.Set("content", *req.Content)
			}
			if req.Featured != nil {
				record.Set("featured", *req.Featured)
			}
			if req.SortOrder != nil {
				record.Set("sort_order", *req.SortOrder)
			}
			if req.Status != nil {
				record.Set("status", *req.Status)
				if *req.Status == "approved" {
					record.Set("approved_at", time.Now())
				}
			}

			if err := app.Save(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update"})
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "updated"})
		}).Bind(apis.RequireAuth())

		se.Router.DELETE("/api/testimonials/{id}", func(e *core.RequestEvent) error {
			id := e.Request.PathValue("id")
			record, err := app.FindRecordById("testimonials", id)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "testimonial not found"})
			}

			if err := app.Delete(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete"})
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
		}).Bind(apis.RequireAuth())

		se.Router.GET("/api/testimonials/pending-count", func(e *core.RequestEvent) error {
			records, err := app.FindRecordsByFilter(
				"testimonials",
				"status = 'pending'",
				"",
				0,
				0,
			)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to count"})
			}

			return e.JSON(http.StatusOK, map[string]int{"count": len(records)})
		}).Bind(apis.RequireAuth())

		se.Router.GET("/api/public/testimonials", func(e *core.RequestEvent) error {
			records, err := app.FindRecordsByFilter(
				"testimonials",
				"status = 'approved'",
				"sort_order, -created",
				50,
				0,
			)
			if err != nil {
				return e.JSON(http.StatusOK, []interface{}{})
			}

			var result []map[string]interface{}
			for _, r := range records {
				item := map[string]interface{}{
					"id":                      r.Id,
					"content":                 r.GetString("content"),
					"relationship":            r.GetString("relationship"),
					"author_name":             r.GetString("author_name"),
					"author_title":            r.GetString("author_title"),
					"author_company":          r.GetString("author_company"),
					"verification_method":     r.GetString("verification_method"),
					"verification_identifier": r.GetString("verification_identifier"),
					"featured":                r.GetBool("featured"),
				}

				if photo := r.GetString("author_photo"); photo != "" {
					item["author_photo"] = "/api/files/testimonials/" + r.Id + "/" + photo
				}

				result = append(result, item)
			}

			return e.JSON(http.StatusOK, result)
		})

		se.Router.POST("/api/testimonials/verify/email", RateLimitMiddleware(rl, "strict")(func(e *core.RequestEvent) error {
			var req struct {
				TestimonialID string `json:"testimonial_id"`
				Email         string `json:"email"`
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			if req.TestimonialID == "" || req.Email == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "testimonial_id and email required"})
			}

			testimonialRecord, err := app.FindRecordById("testimonials", req.TestimonialID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "testimonial not found"})
			}

			if testimonialRecord.GetString("verification_method") != "none" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "testimonial already verified"})
			}

			rawToken, err := testimonial.GenerateToken()
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
			}

			tokenHMAC := testimonial.HMACToken(rawToken)

			collection, err := app.FindCollectionByNameOrId("email_verification_tokens")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "collection not found"})
			}

			record := core.NewRecord(collection)
			record.Set("testimonial_id", req.TestimonialID)
			record.Set("email", req.Email)
			record.Set("token_hash", tokenHMAC)
			record.Set("expires_at", testimonial.EmailVerificationExpiry())

			if err := app.Save(record); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save verification token"})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"status":             "verification_sent",
				"verification_token": rawToken,
			})
		}))

		se.Router.GET("/api/testimonials/verify/email/{token}", func(e *core.RequestEvent) error {
			token := e.Request.PathValue("token")

			tokenHMAC := testimonial.HMACToken(token)
			verificationRecord, err := app.FindFirstRecordByFilter(
				"email_verification_tokens",
				"token_hash = {:hash}",
				map[string]interface{}{"hash": tokenHMAC},
			)
			if err != nil || verificationRecord == nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid or expired token"})
			}

			expiresAt := verificationRecord.GetDateTime("expires_at")
			if !expiresAt.IsZero() && time.Now().After(expiresAt.Time()) {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "token expired"})
			}

			if !verificationRecord.GetDateTime("verified_at").IsZero() {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "already verified"})
			}

			testimonialID := verificationRecord.GetString("testimonial_id")
			testimonialRecord, err := app.FindRecordById("testimonials", testimonialID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "testimonial not found"})
			}

			email := verificationRecord.GetString("email")
			testimonialRecord.Set("verification_method", "email")
			testimonialRecord.Set("verification_identifier", email)
			testimonialRecord.Set("verified_at", time.Now())

			if err := app.Save(testimonialRecord); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to verify"})
			}

			verificationRecord.Set("verified_at", time.Now())
			app.Save(verificationRecord)

			return e.JSON(http.StatusOK, map[string]string{"status": "verified"})
		})

		return se.Next()
	})
}
