package hooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/client"
	"github.com/stripe/stripe-go/v82/webhook"
)

// getStripeKey resolves the Stripe secret key.
// It checks the database first (via CryptoService), then falls back to the env var.
func getStripeKey(app *pocketbase.PocketBase, crypto *services.CryptoService) string {
	key := services.GetStripeSecretKey(app, crypto)
	if key != "" {
		return key
	}
	return os.Getenv("STRIPE_SECRET_KEY")
}

// getStripeWebhookSecret resolves the Stripe webhook secret.
// It checks the database first (via CryptoService), then falls back to the env var.
func getStripeWebhookSecret(app *pocketbase.PocketBase, crypto *services.CryptoService) string {
	secret := services.GetStripeWebhookSecret(app, crypto)
	if secret != "" {
		return secret
	}
	return os.Getenv("STRIPE_WEBHOOK_SECRET")
}

// urlOrigin returns scheme://host:port for an absolute http(s) URL, with the
// default port filled in, so "https://a.example" and "https://a.example:443"
// compare equal. It returns "" for anything else.
func urlOrigin(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.User != nil || u.Hostname() == "" {
		return ""
	}
	scheme := strings.ToLower(u.Scheme)
	port := u.Port()
	switch {
	case scheme == "https" && port == "":
		port = "443"
	case scheme == "http" && port == "":
		port = "80"
	case scheme != "https" && scheme != "http":
		return ""
	}
	return scheme + "://" + strings.ToLower(u.Hostname()) + ":" + port
}

// validateRedirectURL checks that a checkout success or cancel URL points at
// this site, as configured in the application URL. Stripe sends the buyer to
// these URLs, so they can't be judged against the request's Host or
// X-Forwarded-Host headers: whoever sends the request chooses those. Scheme,
// host and port must all match, so plain HTTP is only accepted when the owner
// configured an HTTP address themselves, as on a self-hosted LAN install.
// With no application URL configured, nothing is accepted.
func validateRedirectURL(redirectURL string, publicOrigin string) bool {
	want := urlOrigin(publicOrigin)
	return want != "" && urlOrigin(redirectURL) == want
}

// RegisterPurchaseHooks registers content purchase endpoints
func RegisterPurchaseHooks(app *pocketbase.PocketBase, crypto *services.CryptoService, rl *services.RateLimitService) {
	purchaseService := services.NewPurchaseService(crypto)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Check if Stripe is configured and ready for content purchases.
		// Used by admin UI to determine whether "paid" access tier should be available.
		se.Router.GET("/api/stripe/ready", func(e *core.RequestEvent) error {
			stripeKey := getStripeKey(app, crypto)
			if stripeKey == "" {
				return e.JSON(http.StatusOK, map[string]any{
					"ready":   false,
					"reason":  "stripe_not_configured",
					"message": "Stripe secret key is not set. Configure Stripe in your settings to accept payments.",
				})
			}

			if ready, reason := purchaseDeliveryReady(app); !ready {
				return e.JSON(http.StatusOK, map[string]any{
					"ready":   false,
					"reason":  "email_not_configured",
					"message": reason,
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

			// Course sales are Facet Cloud-only. Refuse before any lookup,
			// coupon claim, purchase record or Stripe call, whatever the
			// request claims or the environment says.
			if req.ContentType == "courses" && !services.CourseSalesAvailable {
				return courseSalesUnavailable(e)
			}

			// Redirect URLs must be on this site's configured address
			publicOrigin := publicAppURL(app)
			if publicOrigin == "" {
				app.Logger().Error("Checkout refused: the application URL is not set, so redirect URLs can't be checked.")
				return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "purchases are temporarily unavailable"})
			}
			if !validateRedirectURL(req.SuccessURL, publicOrigin) {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid success_url"})
			}
			if !validateRedirectURL(req.CancelURL, publicOrigin) {
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
				// Take the use before granting anything, so a burst of
				// requests can't all redeem the coupon's last use.
				if !claimCouponUse(app, couponCode) {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid coupon: Coupon usage limit reached"})
				}
				accessToken, err := purchaseService.GenerateAccessToken()
				if err != nil {
					releaseCouponUse(app, couponCode)
					app.Logger().Error("Failed to generate access token for free coupon", "error", err)
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
				}

				collection, err := app.FindCollectionByNameOrId("purchases")
				if err != nil {
					releaseCouponUse(app, couponCode)
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "purchases collection not found"})
				}

				purchase := core.NewRecord(collection)
				purchase.Set("buyer_email", fmt.Sprintf("coupon-%s@noreply.facet.run", accessToken[:12]))
				purchase.Set("content_type", req.ContentType)
				purchase.Set("content_id", req.ContentID)
				purchase.Set("stripe_payment_intent_id", "")
				purchase.Set("stripe_checkout_session_id", fmt.Sprintf(couponSessionPrefix+"%s_%s_%s", couponCode, req.ContentID, accessToken[:8]))
				purchase.Set("amount", 0)
				purchase.Set("currency", "usd")
				purchase.Set("status", "completed")
				purchase.Set("access_token", accessToken)

				if err := app.Save(purchase); err != nil {
					releaseCouponUse(app, couponCode)
					app.Logger().Error("Failed to save coupon purchase", "error", err)
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to grant access"})
				}

				// Return the access URL so the frontend can redirect
				// This token is safe to hand back directly: its buyer email is
				// made up for this redemption, so it unlocks nothing else.
				return e.JSON(http.StatusOK, map[string]string{
					"status":       "authorized",
					"checkout_url": fmt.Sprintf("/api/content/access/%s", accessToken),
				})
			}

			stripeKey := getStripeKey(app, crypto)
			if stripeKey == "" {
				return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Stripe not configured"})
			}

			// Buyers get their access link by email. Without email, refuse
			// before taking payment rather than after.
			if ready, _ := purchaseDeliveryReady(app); !ready {
				app.Logger().Error("Paid checkout refused: purchase emails can't be sent. Configure SMTP, the sender address and the application URL.")
				return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "purchases are temporarily unavailable"})
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
				SuccessURL: stripe.String(withCheckoutSessionID(req.SuccessURL)),
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
				app.Logger().Error("Failed to create Stripe session", "code", stripeErrorCode(err))
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create checkout session"})
			}

			return e.JSON(http.StatusOK, map[string]string{"checkout_url": sess.URL})
		}))

		// Checkout return page: report a session's status and email the
		// buyer their access link once it's paid. Never sets a cookie.
		// Rate limited: strict tier (10/min)
		se.Router.POST("/api/content/checkout/confirm", RateLimitMiddleware(rl, "strict")(
			handleCheckoutConfirm(app, purchaseService, func() string { return getStripeKey(app, crypto) })))

		// Stripe webhook endpoint
		// Rate limited: normal tier (60/min) to prevent abuse
		se.Router.POST("/api/content/webhook", RateLimitMiddleware(rl, "normal")(func(e *core.RequestEvent) error {
			webhookSecret := getStripeWebhookSecret(app, crypto)

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

			// A completed checkout only grants access once Stripe reports the
			// money as paid. Delayed methods (bank debits, vouchers) complete
			// first with payment_status "unpaid" and settle later through
			// checkout.session.async_payment_succeeded.
			switch event.Type {
			case "checkout.session.completed", "checkout.session.async_payment_succeeded":
				var sess stripe.CheckoutSession
				if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid event data"})
				}

				if sess.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
					app.Logger().Info("Checkout session not paid yet, access not granted",
						"session_id", sess.ID, "event", event.Type, "payment_status", sess.PaymentStatus)
					return e.JSON(http.StatusOK, map[string]string{"status": "awaiting_payment"})
				}

				if sess.Metadata["content_type"] == "" || sess.Metadata["content_id"] == "" {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "missing content metadata"})
				}

				// A repeated event finds the purchase already recorded and
				// only retries the email if it hasn't gone out yet. A failed
				// send answers with an error, so Stripe retries the event.
				unlock := lockSession(sess.ID)
				defer unlock()

				purchase, err := recordPaidSession(app, purchaseService, &sess)
				switch {
				case errors.Is(err, errSessionMismatch):
					app.Logger().Error("Checkout session metadata does not match its purchase", "session_id", sess.ID)
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "session does not match purchase"})
				case errors.Is(err, errSessionNoEmail):
					app.Logger().Error("Webhook missing customer email", "session_id", sess.ID)
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "missing customer email"})
				case err != nil:
					app.Logger().Error("Failed to save purchase", "session_id", sess.ID)
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save purchase"})
				}

				if purchase.GetString("status") != "completed" {
					return e.JSON(http.StatusOK, map[string]string{"status": "already_processed"})
				}
				alreadySent := !purchase.GetDateTime("access_email_sent_at").IsZero()
				if _, err := deliverPurchaseAccess(app, purchase, false); err != nil {
					if errors.Is(err, errDeliveryUnconfigured) {
						return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "purchase email delivery not configured"})
					}
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to send access email"})
				}
				if alreadySent {
					return e.JSON(http.StatusOK, map[string]string{"status": "already_processed"})
				}
				return e.JSON(http.StatusOK, map[string]string{"status": "success"})

			case "checkout.session.async_payment_failed":
				var sess stripe.CheckoutSession
				if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid event data"})
				}
				app.Logger().Warn("Delayed checkout payment failed, access not granted", "session_id", sess.ID)
				return e.JSON(http.StatusOK, map[string]string{"status": "payment_failed"})
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

				status, err := applyChargeRefund(app, paymentIntentID, &charge)
				switch {
				case errors.Is(err, errRefundPurchaseNotFound):
					// The checkout may not be recorded yet. Answering with an
					// error makes Stripe retry, by which time it will be.
					app.Logger().Warn("Purchase not found for refund", "payment_intent", paymentIntentID)
					return e.JSON(http.StatusNotFound, map[string]string{"error": "purchase not found"})
				case err != nil:
					app.Logger().Error("Failed to update purchase status", "error", err)
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update purchase"})
				}
				return e.JSON(http.StatusOK, map[string]string{"status": status})
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

			// A paid access link only ever goes to the buyer's inbox, so
			// following it proves that email and unlocks everything bought
			// with it. A free coupon link is handed straight back to whoever
			// redeemed it, proves no inbox, and so unlocks only its own item.
			buyerEmail := purchase.GetString("buyer_email")
			var jwtToken string
			var expiresAt time.Time
			if isCouponRedemption(purchase) {
				jwtToken, expiresAt, err = purchaseService.GeneratePurchaseJWT([]string{purchase.Id}, buyerEmail)
			} else {
				jwtToken, expiresAt, err = purchaseService.GenerateBuyerJWT(buyerEmail)
			}
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
			return validatePaidContent(app, crypto, e)
		})
		app.OnRecordUpdateRequest(coll).BindFunc(func(e *core.RecordRequestEvent) error {
			return validatePaidContent(app, crypto, e)
		})
	}
}

// errCourseSalesUnavailable is returned wherever a course sale is refused.
const errCourseSalesUnavailable = "Selling courses is only available on Facet Cloud. Courses can still be published for free."

func courseSalesUnavailable(e *core.RequestEvent) error {
	return e.JSON(http.StatusForbidden, map[string]any{
		"error":  errCourseSalesUnavailable,
		"reason": "course_sales_unavailable",
	})
}

// courseSaleChangeBlocked reports whether a course save would put the course
// up for sale. An imported course that is already paid keeps its tier and
// price untouched (it is never silently made free), but it can't be newly
// made paid or repriced, and a new course can't be created paid.
func courseSaleChangeBlocked(record *core.Record) bool {
	if services.CourseSalesAvailable || record.Collection().Name != "courses" {
		return false
	}
	if record.GetString("access_tier") != "paid" {
		return false
	}
	original := record.Original()
	if !record.IsNew() && original != nil &&
		original.GetString("access_tier") == "paid" &&
		original.GetInt("price") == record.GetInt("price") {
		return false
	}
	return true
}

// validatePaidContent checks that Stripe is configured when access_tier=paid, and price meets minimum.
func validatePaidContent(app *pocketbase.PocketBase, crypto *services.CryptoService, e *core.RecordRequestEvent) error {
	if e.Record.Collection().Name == "courses" && !services.CourseSalesAvailable {
		if courseSaleChangeBlocked(e.Record) {
			return courseSalesUnavailable(e.RequestEvent)
		}
		// A retained imported paid course doesn't need Stripe to stay as it is.
		return e.Next()
	}
	accessTier := e.Record.GetString("access_tier")
	if accessTier != "paid" {
		return e.Next()
	}

	// Check Stripe key: DB first, then env var
	stripeKey := getStripeKey(app, crypto)
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

// couponSessionPrefix marks purchases granted by a zero-price coupon, which
// never went through Stripe.
const couponSessionPrefix = "coupon_"

func isCouponRedemption(purchase *core.Record) bool {
	return strings.HasPrefix(purchase.GetString("stripe_checkout_session_id"), couponSessionPrefix)
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

	purchase := findCompletedPurchase(app, purchaseService, purchaseIDs, buyerEmail, contentType, contentID)
	if purchase == nil {
		return false, ""
	}
	return true, purchase.Id
}

// findCompletedPurchase returns the buyer's completed purchase of one item.
// The purchase must belong to the email in the buyer's token, and the token
// must have been issued for that purchase. Every buyer has their own purchase
// record, so this has to search the buyer's purchases rather than taking the
// first sale of the item. A refunded purchase stops matching straight away.
func findCompletedPurchase(app core.App, purchaseService *services.PurchaseService, purchaseIDs []string, buyerEmail, contentType, contentID string) *core.Record {
	if buyerEmail == "" || len(purchaseIDs) == 0 {
		return nil
	}

	purchaseRecords, err := app.FindRecordsByFilter(
		"purchases",
		"content_type = {:type} && content_id = {:id} && buyer_email = {:email} && status = 'completed'",
		"",
		0,
		0,
		map[string]interface{}{
			"type":  contentType,
			"id":    contentID,
			"email": buyerEmail,
		},
	)
	if err != nil {
		return nil
	}

	for _, purchase := range purchaseRecords {
		if purchaseService.HasPurchaseAccess(purchaseIDs, purchase.Id) {
			return purchase
		}
	}
	return nil
}

// paidContentFields lists the response fields that hold the part of an item a
// buyer pays for. Titles, summaries, excerpts and cover images stay visible so
// the item can still be listed and sold. Posts keep a short preview of their
// content instead of losing it entirely. Inline and gallery media are paid
// too, both the stored references and the URLs projections expand them into.
var paidContentFields = map[string][]string{
	"posts":          withPaidMedia("content"),
	"projects":       withPaidMedia("description", "experience"),
	"talks":          withPaidMedia("description", "slides_url", "video_url"),
	"custom_content": withPaidMedia("content"),
}

func withPaidMedia(fields ...string) []string {
	return append(fields, "media", "media_refs", "media_refs_expand", "media_urls")
}

const defaultPaywallPreviewParagraphs = 3

// contentAccess reports whether the current request may see the paid part of
// a record. A nil contentAccess grants nothing, so a projection that forgets to
// pass one fails closed.
type contentAccess func(record *core.Record) bool

// requiresPurchase reports whether a record is paid content.
func requiresPurchase(record *core.Record) bool {
	if _, gated := paidContentFields[strings.TrimPrefix(record.Collection().Name, "demo_")]; !gated {
		return false
	}
	return record.GetString("access_tier") == "paid"
}

// newContentAccess checks the request's purchase token, once per record. The
// site owner always sees full content.
func newContentAccess(app core.App, purchaseService *services.PurchaseService, e *core.RequestEvent) contentAccess {
	return func(record *core.Record) bool {
		if !requiresPurchase(record) || isSuperuser(e) {
			return true
		}
		token := extractPurchaseCookie(e)
		if token == "" {
			return false
		}
		purchaseIDs, buyerEmail, err := purchaseService.ValidatePurchaseJWT(token)
		if err != nil {
			return false
		}
		contentType := strings.TrimPrefix(record.Collection().Name, "demo_")
		return findCompletedPurchase(app, purchaseService, purchaseIDs, buyerEmail, contentType, record.Id) != nil
	}
}

// gatePaidContent removes the paid fields from a public projection of record
// unless access allows them. It runs last, after any per-view overrides and
// media expansion, so neither can put a paid field back.
func gatePaidContent(record *core.Record, item map[string]interface{}, access contentAccess) {
	if !requiresPurchase(record) {
		return
	}
	item["access_tier"] = "paid"
	item["price"] = record.GetInt("price")
	if access != nil && access(record) {
		item["is_gated"] = false
		item["is_purchased"] = true
		return
	}

	item["is_gated"] = true
	item["is_purchased"] = false
	for _, field := range paidContentFields[strings.TrimPrefix(record.Collection().Name, "demo_")] {
		delete(item, field)
	}

	if strings.TrimPrefix(record.Collection().Name, "demo_") == "posts" {
		preview := record.GetInt("paywall_preview_paragraphs")
		if preview <= 0 {
			preview = defaultPaywallPreviewParagraphs
		}
		item["content"] = paywallPreview(record.GetString("content"), preview)
		item["is_preview_content"] = true
	}
}

// paywallPreview returns up to the first n paragraphs of Markdown content. It
// always holds back at least the final paragraph, so a short post is never
// given away whole just because it fits inside the preview.
func paywallPreview(content string, n int) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	var paragraphs []string
	for _, paragraph := range strings.Split(content, "\n\n") {
		if strings.TrimSpace(paragraph) != "" {
			paragraphs = append(paragraphs, paragraph)
		}
	}
	if n > len(paragraphs)-1 {
		n = len(paragraphs) - 1
	}
	if n <= 0 {
		return ""
	}
	return strings.Join(paragraphs[:n], "\n\n")
}
