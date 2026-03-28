package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/client"
	"github.com/stripe/stripe-go/v82/webhook"
)

// validateRedirectURL ensures the URL has a valid scheme, host, and matches
// the request origin to prevent open redirect attacks.
func validateRedirectURL(redirectURL string, requestHost string) bool {
	u, err := url.Parse(redirectURL)
	if err != nil {
		return false
	}
	// Must be HTTPS (or HTTP for localhost dev)
	if u.Scheme != "https" && u.Scheme != "http" {
		return false
	}
	// Must have a host
	if u.Host == "" {
		return false
	}
	// Host must match the request host to prevent open redirect
	if requestHost != "" && u.Hostname() != requestHost {
		return false
	}
	return true
}

// RegisterPurchaseHooks registers content purchase endpoints
func RegisterPurchaseHooks(app *pocketbase.PocketBase, crypto *services.CryptoService, rl *services.RateLimitService) {
	purchaseService := services.NewPurchaseService(crypto)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Check if Stripe is configured and ready for content purchases.
		// Used by admin UI to determine whether "paid" access tier should be available.
		se.Router.GET("/api/stripe/ready", func(e *core.RequestEvent) error {
			stripeKey := os.Getenv("STRIPE_SECRET_KEY")
			if stripeKey == "" {
				return e.JSON(http.StatusOK, map[string]any{
					"ready":   false,
					"reason":  "stripe_not_configured",
					"message": "Stripe secret key is not set. Configure Stripe in your settings to accept payments.",
				})
			}

			return e.JSON(http.StatusOK, map[string]any{"ready": true})
		})

		// Create Stripe Checkout session for content purchase
		// Rate limited: strict tier (10/min) to prevent abuse
		se.Router.POST("/api/content/checkout", RateLimitMiddleware(rl, "strict")(func(e *core.RequestEvent) error {
			var req struct {
				ContentType string `json:"content_type"` // "posts", "projects", "custom_content", "courses", "talks"
				ContentID   string `json:"content_id"`
				SuccessURL  string `json:"success_url"`
				CancelURL   string `json:"cancel_url"`
				CouponCode  string `json:"coupon_code"`
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			// Validate content type
			if req.ContentType != "posts" && req.ContentType != "projects" && req.ContentType != "custom_content" && req.ContentType != "courses" && req.ContentType != "talks" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid content type"})
			}

			// Validate redirect URLs (must match request host to prevent open redirect)
			requestHost := e.Request.Host
			if h := e.Request.Header.Get("X-Forwarded-Host"); h != "" {
				requestHost = h
			}
			// Strip port from host for comparison
			if idx := strings.LastIndex(requestHost, ":"); idx != -1 {
				requestHost = requestHost[:idx]
			}
			if !validateRedirectURL(req.SuccessURL, requestHost) {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid success_url"})
			}
			if !validateRedirectURL(req.CancelURL, requestHost) {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid cancel_url"})
			}

			// Fetch content record
			record, err := app.FindRecordById(req.ContentType, req.ContentID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "content not found"})
			}

			// Verify content is gated (access_tier == "paid" and price > 0)
			accessTier := record.GetString("access_tier")
			price := record.GetInt("price")

			if accessTier != "paid" || price <= 0 {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "content is not gated"})
			}

			// Stripe requires minimum 50 cents (USD)
			if price > 0 && price < 50 {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "minimum price is $0.50 (50 cents)"})
			}

			// Apply coupon if provided
			couponCode := strings.TrimSpace(req.CouponCode)
			if couponCode != "" {
				couponCode = strings.ToUpper(couponCode)
				reason := validateCoupon(app, couponCode, req.ContentType, req.ContentID)
				if reason != "" {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid coupon: " + reason})
				}

				// Fetch coupon to apply discount
				couponRecords, err := app.FindRecordsByFilter(
					"coupons",
					"code = {:code}",
					"",
					1,
					0,
					map[string]interface{}{"code": couponCode},
				)
				if err == nil && len(couponRecords) > 0 {
					coupon := couponRecords[0]
					price = CalculateDiscountedPrice(
						price,
						coupon.GetString("discount_type"),
						coupon.GetFloat("discount_value"),
					)
				}

				// Ensure discounted price meets Stripe minimum
				if price > 0 && price < 50 {
					price = 50
				}
			}

			// If coupon made the price 0, grant access directly (skip Stripe)
			if price <= 0 {
				accessToken, err := purchaseService.GenerateAccessToken()
				if err != nil {
					app.Logger().Error("Failed to generate access token for free coupon", "error", err)
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
				}

				collection, err := app.FindCollectionByNameOrId("purchases")
				if err != nil {
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "purchases collection not found"})
				}

				purchase := core.NewRecord(collection)
				purchase.Set("buyer_email", fmt.Sprintf("coupon-%s@noreply.facet.run", accessToken[:12]))
				purchase.Set("content_type", req.ContentType)
				purchase.Set("content_id", req.ContentID)
				purchase.Set("stripe_payment_intent_id", "")
				purchase.Set("stripe_checkout_session_id", fmt.Sprintf("coupon_%s_%s_%s", couponCode, req.ContentID, accessToken[:8]))
				purchase.Set("amount", 0)
				purchase.Set("currency", "usd")
				purchase.Set("status", "completed")
				purchase.Set("access_token", accessToken)

				if err := app.Save(purchase); err != nil {
					app.Logger().Error("Failed to save coupon purchase", "error", err)
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to grant access"})
				}

				IncrementCouponUses(app, couponCode)

				// Return the access URL so the frontend can redirect
				return e.JSON(http.StatusOK, map[string]string{
					"checkout_url": fmt.Sprintf("/api/content/access/%s", accessToken),
				})
			}

			stripeKey := os.Getenv("STRIPE_SECRET_KEY")
			if stripeKey == "" {
				return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Stripe not configured"})
			}

			// Per-request Stripe client (avoids global stripe.Key race condition)
			sc := &client.API{}
			sc.Init(stripeKey, nil)

			// Get content title for Stripe
			title := record.GetString("title")
			if title == "" {
				title = "Content Purchase"
			}

			// Create Stripe Checkout session
			params := &stripe.CheckoutSessionParams{
				Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
				LineItems: []*stripe.CheckoutSessionLineItemParams{
					{
						PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
							Currency: stripe.String("usd"),
							ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
								Name: stripe.String(title),
							},
							UnitAmount: stripe.Int64(int64(price)),
						},
						Quantity: stripe.Int64(1),
					},
				},
				SuccessURL: stripe.String(req.SuccessURL),
				CancelURL:  stripe.String(req.CancelURL),
			}

			// Add metadata for webhook processing
			params.Metadata = map[string]string{
				"content_type": req.ContentType,
				"content_id":   req.ContentID,
			}
			if couponCode != "" {
				params.Metadata["coupon_code"] = couponCode
			}

			sess, err := sc.CheckoutSessions.New(params)
			if err != nil {
				app.Logger().Error("Failed to create Stripe session", "error", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create checkout session"})
			}

			return e.JSON(http.StatusOK, map[string]string{"checkout_url": sess.URL})
		}))

		// Stripe webhook endpoint
		// Rate limited: normal tier (60/min) to prevent abuse
		se.Router.POST("/api/content/webhook", RateLimitMiddleware(rl, "normal")(func(e *core.RequestEvent) error {
			webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")

			if webhookSecret == "" {
				return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Stripe not configured"})
			}

			// Read request body (limit to 64KB to prevent abuse)
			body, err := io.ReadAll(io.LimitReader(e.Request.Body, 65536))
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "failed to read body"})
			}

			// Verify webhook signature
			event, err := webhook.ConstructEvent(body, e.Request.Header.Get("Stripe-Signature"), webhookSecret)
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid signature"})
			}

			// Handle checkout.session.completed
			if event.Type == "checkout.session.completed" {
				var sess stripe.CheckoutSession
				if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid event data"})
				}

				// Extract metadata
				contentType, ok := sess.Metadata["content_type"]
				if !ok {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "missing content_type in metadata"})
				}
				contentID, ok := sess.Metadata["content_id"]
				if !ok {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "missing content_id in metadata"})
				}

				// Idempotency: check if we already processed this checkout session
				existing, _ := app.FindRecordsByFilter(
					"purchases",
					"stripe_checkout_session_id = {:sid}",
					"",
					1,
					0,
					map[string]interface{}{"sid": sess.ID},
				)
				if len(existing) > 0 {
					app.Logger().Info("Duplicate webhook, purchase already exists", "session_id", sess.ID)
					return e.JSON(http.StatusOK, map[string]string{"status": "already_processed"})
				}

				// Generate access token
				accessToken, err := purchaseService.GenerateAccessToken()
				if err != nil {
					app.Logger().Error("Failed to generate access token", "error", err)
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
				}

				// Create purchase record
				collection, err := app.FindCollectionByNameOrId("purchases")
				if err != nil {
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "purchases collection not found"})
				}

				purchase := core.NewRecord(collection)
				buyerEmail := ""
				if sess.CustomerDetails != nil && sess.CustomerDetails.Email != "" {
					buyerEmail = sess.CustomerDetails.Email
				}
				if buyerEmail == "" {
					app.Logger().Error("Webhook missing customer email", "session_id", sess.ID)
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "missing customer email"})
				}
				purchase.Set("buyer_email", buyerEmail)
				purchase.Set("content_type", contentType)
				purchase.Set("content_id", contentID)
				paymentIntentID := ""
				if sess.PaymentIntent != nil {
					paymentIntentID = sess.PaymentIntent.ID
				}
				purchase.Set("stripe_payment_intent_id", paymentIntentID)
				purchase.Set("stripe_checkout_session_id", sess.ID)
				purchase.Set("amount", sess.AmountTotal)
				purchase.Set("currency", sess.Currency)
				purchase.Set("status", "completed")
				purchase.Set("access_token", accessToken)

				if err := app.Save(purchase); err != nil {
					app.Logger().Error("Failed to save purchase", "error", err)
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save purchase"})
				}

				app.Logger().Info("Purchase created", "purchase_id", purchase.Id, "email", buyerEmail)

				// Increment coupon uses if a coupon was applied
				if couponCode, ok := sess.Metadata["coupon_code"]; ok && couponCode != "" {
					IncrementCouponUses(app, couponCode)
				}

				return e.JSON(http.StatusOK, map[string]string{"status": "success"})
			}

			// Handle charge.refunded
			if event.Type == "charge.refunded" {
				var charge stripe.Charge
				if err := json.Unmarshal(event.Data.Raw, &charge); err != nil {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid event data"})
				}

				// Nil guard: charge.PaymentIntent can be nil for direct charges
				paymentIntentID := ""
				if charge.PaymentIntent != nil {
					paymentIntentID = charge.PaymentIntent.ID
				}
				if paymentIntentID == "" {
					app.Logger().Error("Refund webhook missing payment intent", "charge_id", charge.ID)
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "missing payment intent"})
				}

				// Find purchase by payment intent ID
				purchaseRecords, err := app.FindRecordsByFilter(
					"purchases",
					"stripe_payment_intent_id = {:payment_intent}",
					"",
					1,
					0,
					map[string]interface{}{"payment_intent": paymentIntentID},
				)

				if err != nil || len(purchaseRecords) == 0 {
					app.Logger().Warn("Purchase not found for refund", "payment_intent", paymentIntentID)
					return e.JSON(http.StatusNotFound, map[string]string{"error": "purchase not found"})
				}

				purchase := purchaseRecords[0]

				// Idempotency: check if purchase is already refunded
				if purchase.GetString("status") == "refunded" {
					app.Logger().Info("Duplicate webhook, purchase already refunded", "purchase_id", purchase.Id)
					return e.JSON(http.StatusOK, map[string]string{"status": "already_processed"})
				}

				purchase.Set("status", "refunded")

				if err := app.Save(purchase); err != nil {
					app.Logger().Error("Failed to update purchase status", "error", err)
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update purchase"})
				}

				app.Logger().Info("Purchase refunded", "purchase_id", purchase.Id)

				return e.JSON(http.StatusOK, map[string]string{"status": "success"})
			}

			// Ignore other event types
			return e.JSON(http.StatusOK, map[string]string{"status": "ignored"})
		}))

		// Access content via token from email link
		// Rate limited: normal tier (60/min)
		se.Router.GET("/api/content/access/{token}", RateLimitMiddleware(rl, "normal")(func(e *core.RequestEvent) error {
			token := e.Request.PathValue("token")

			if token == "" {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "token required"})
			}

			// Find purchase by access token
			purchaseRecords, err := app.FindRecordsByFilter(
				"purchases",
				"access_token = {:token}",
				"",
				1,
				0,
				map[string]interface{}{"token": token},
			)

			if err != nil || len(purchaseRecords) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "invalid token"})
			}

			purchase := purchaseRecords[0]

			// Check purchase status
			if purchase.GetString("status") != "completed" {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "purchase not valid"})
			}

			// Get all purchases for this buyer email
			buyerEmail := purchase.GetString("buyer_email")
			allPurchases, err := app.FindRecordsByFilter(
				"purchases",
				"buyer_email = {:email} && status = 'completed'",
				"",
				100,
				0,
				map[string]interface{}{"email": buyerEmail},
			)

			if err != nil {
				app.Logger().Error("Failed to fetch purchases", "error", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch purchases"})
			}

			// Collect all purchase IDs
			purchaseIDs := make([]string, 0, len(allPurchases))
			for _, p := range allPurchases {
				purchaseIDs = append(purchaseIDs, p.Id)
			}

			// Generate JWT with all purchase IDs
			jwtToken, expiresAt, err := purchaseService.GeneratePurchaseJWT(purchaseIDs, buyerEmail)
			if err != nil {
				app.Logger().Error("Failed to generate JWT", "error", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
			}

			// Set purchase cookie
			http.SetCookie(e.Response, &http.Cookie{
				Name:     services.PurchaseCookieName,
				Value:    jwtToken,
				Path:     "/",
				Expires:  expiresAt,
				HttpOnly: true,
				Secure:   isRequestSecure(e),
				SameSite: http.SameSiteLaxMode,
			})

			// Determine redirect URL
			contentType := purchase.GetString("content_type")
			contentID := purchase.GetString("content_id")

			// Fetch content to get slug
			contentRecord, err := app.FindRecordById(contentType, contentID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "content not found"})
			}

			slug := contentRecord.GetString("slug")
			redirectURL := "/"

			switch contentType {
			case "posts":
				redirectURL = fmt.Sprintf("/posts/%s", slug)
			case "projects":
				redirectURL = fmt.Sprintf("/projects/%s", slug)
			case "courses":
				redirectURL = fmt.Sprintf("/courses/%s", slug)
			case "talks":
				redirectURL = fmt.Sprintf("/talks/%s", slug)
			case "custom_content":
				// Custom content doesn't have a direct URL, redirect to homepage
				redirectURL = "/"
			}

			return e.Redirect(http.StatusFound, redirectURL)
		}))

		return se.Next()
	})

	// Validate: prevent setting access_tier=paid when Stripe isn't configured.
	// This covers all content types (posts, projects, custom_content, courses).
	// Also enforces minimum price of $0.50 (Stripe minimum).
	gatedCollections := []string{"posts", "projects", "custom_content", "courses", "talks"}
	for _, coll := range gatedCollections {
		app.OnRecordCreateRequest(coll).BindFunc(func(e *core.RecordRequestEvent) error {
			return validatePaidContent(e)
		})
		app.OnRecordUpdateRequest(coll).BindFunc(func(e *core.RecordRequestEvent) error {
			return validatePaidContent(e)
		})
	}
}

// validatePaidContent checks that Stripe is configured when access_tier=paid, and price meets minimum.
func validatePaidContent(e *core.RecordRequestEvent) error {
	accessTier := e.Record.GetString("access_tier")
	if accessTier != "paid" {
		return e.Next()
	}

	// Check Stripe key
	stripeKey := os.Getenv("STRIPE_SECRET_KEY")
	if stripeKey == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Cannot set access tier to \"paid\" - Stripe is not configured. Set up Stripe in Settings first.",
		})
	}

	// Validate minimum price ($0.50 = 50 cents, Stripe minimum for USD)
	price := e.Record.GetInt("price")
	if price < 50 {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Price must be at least $0.50 (Stripe minimum).",
		})
	}

	return e.Next()
}

// extractPurchaseCookie extracts the purchase JWT from cookies or Authorization header
func extractPurchaseCookie(e *core.RequestEvent) string {
	// Try cookie first
	if cookie, err := e.Request.Cookie(services.PurchaseCookieName); err == nil {
		return cookie.Value
	}

	// Try Authorization header as fallback (for API clients)
	authHeader := e.Request.Header.Get("Authorization")
	if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}

	return ""
}

// checkPurchaseAccess verifies if the request has purchase access to specific content
// Returns (hasPurchase bool, purchaseID string)
func checkPurchaseAccess(app *pocketbase.PocketBase, purchaseService *services.PurchaseService, e *core.RequestEvent, contentType, contentID string) (bool, string) {
	// Extract purchase token from cookie
	token := extractPurchaseCookie(e)
	if token == "" {
		return false, ""
	}

	// Validate JWT
	purchaseIDs, buyerEmail, err := purchaseService.ValidatePurchaseJWT(token)
	if err != nil {
		return false, ""
	}

	// Find purchase for this specific content, scoped to buyer
	purchaseRecords, err := app.FindRecordsByFilter(
		"purchases",
		"content_type = {:type} && content_id = {:id} && buyer_email = {:email} && status = 'completed'",
		"",
		1,
		0,
		map[string]interface{}{
			"type":  contentType,
			"id":    contentID,
			"email": buyerEmail,
		},
	)

	if err != nil || len(purchaseRecords) == 0 {
		return false, ""
	}

	purchase := purchaseRecords[0]

	// Check if JWT contains this purchase ID
	if purchaseService.HasPurchaseAccess(purchaseIDs, purchase.Id) {
		return true, purchase.Id
	}

	return false, ""
}

// applyContentGating modifies an item map to gate content if needed
// Returns the modified item with gating applied
func applyContentGating(app *pocketbase.PocketBase, purchaseService *services.PurchaseService, e *core.RequestEvent, item map[string]interface{}, collectionName string) map[string]interface{} {
	// Only apply gating to posts, projects, custom_content, courses, talks
	if collectionName != "posts" && collectionName != "projects" && collectionName != "custom_content" && collectionName != "courses" && collectionName != "talks" {
		return item
	}

	// Check if content has access_tier field
	accessTier, hasTier := item["access_tier"].(string)
	if !hasTier || accessTier != "paid" {
		// Not gated content
		return item
	}

	// Check if user has purchased access
	contentID, ok := item["id"].(string)
	if !ok {
		return item
	}

	hasPurchase, _ := checkPurchaseAccess(app, purchaseService, e, collectionName, contentID)

	if hasPurchase {
		// User has purchased - serve full content
		item["is_gated"] = false
		item["is_purchased"] = true
		return item
	}

	// User hasn't purchased - gate the content
	item["is_gated"] = true
	item["is_purchased"] = false

	// For posts: truncate to preview paragraphs instead of deleting content
	if collectionName == "posts" {
		previewParagraphs := 3 // default
		if pp, ok := item["paywall_preview_paragraphs"].(float64); ok && int(pp) > 0 {
			previewParagraphs = int(pp)
		} else if pp, ok := item["paywall_preview_paragraphs"].(int); ok && pp > 0 {
			previewParagraphs = pp
		}

		if content, ok := item["content"].(string); ok && content != "" {
			paragraphs := strings.Split(content, "\n\n")
			if len(paragraphs) > previewParagraphs {
				item["content"] = strings.Join(paragraphs[:previewParagraphs], "\n\n")
			}
			// If content has fewer paragraphs than preview limit, keep it all
		}
		item["is_preview_content"] = true
	} else {
		// For other collections: strip content entirely
		delete(item, "content")
		delete(item, "description") // For projects
	}

	return item
}
