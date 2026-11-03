package hooks

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"sync"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/client"
)

// How a paid buyer gets their content.
//
// Stripe collects the buyer's email but doesn't prove they own it, and
// purchases, course progress and certificates are all keyed by email. So the
// access link, which signs the visitor in as that email, is only ever sent to
// the inbox. The checkout return page can ask for the status of a session and
// have the link sent (or sent again), but it never receives a token or cookie
// itself. Someone who checks out with another person's address pays for that
// person and learns nothing about them.

// fetchCheckoutSession asks Stripe for a checkout session with its payment
// intent expanded. Tests replace it so they never call Stripe.
var fetchCheckoutSession = func(stripeKey, sessionID string) (*stripe.CheckoutSession, error) {
	sc := &client.API{}
	sc.Init(stripeKey, nil)
	params := &stripe.CheckoutSessionParams{}
	params.AddExpand("payment_intent")
	return sc.CheckoutSessions.Get(sessionID, params)
}

// accessEmailResendCooldown limits how often the return page can resend an
// access link, so a session ID can't be used to flood the buyer's inbox.
const accessEmailResendCooldown = 2 * time.Minute

var (
	errSessionNotPaid       = errors.New("checkout session is not paid")
	errSessionMismatch      = errors.New("checkout session does not match purchase")
	errSessionNoEmail       = errors.New("checkout session has no customer email")
	errDeliveryUnconfigured = errors.New("purchase email delivery is not configured")
	errDeliveryFailed       = errors.New("purchase email could not be sent")
)

var checkoutSessionIDPattern = regexp.MustCompile(`^cs_[A-Za-z0-9_]{1,250}$`)

// sessionLocks serialises work on one checkout session, so a webhook and the
// return page arriving together record one purchase and send one email.
var sessionLocks = struct {
	mu    sync.Mutex
	locks map[string]*sessionLock
}{locks: map[string]*sessionLock{}}

type sessionLock struct {
	mu   sync.Mutex
	refs int
}

func lockSession(id string) func() {
	sessionLocks.mu.Lock()
	l := sessionLocks.locks[id]
	if l == nil {
		l = &sessionLock{}
		sessionLocks.locks[id] = l
	}
	l.refs++
	sessionLocks.mu.Unlock()

	l.mu.Lock()
	return func() {
		l.mu.Unlock()
		sessionLocks.mu.Lock()
		l.refs--
		if l.refs == 0 {
			delete(sessionLocks.locks, id)
		}
		sessionLocks.mu.Unlock()
	}
}

// purchaseDeliveryReady reports whether this site can email access links.
// Paid checkout is refused when it can't, so nobody pays and then waits for
// an email that will never come.
func purchaseDeliveryReady(app core.App) (bool, string) {
	if !app.Settings().SMTP.Enabled {
		return false, "Email (SMTP) is not configured. Paid content is delivered by email, so set up SMTP in Settings before selling."
	}
	if strings.TrimSpace(app.Settings().Meta.SenderAddress) == "" {
		return false, "The email sender address is not set. Set it in Settings so purchase emails can be sent."
	}
	if publicAppURL(app) == "" {
		return false, "The site URL is not set. Set the application URL in Settings so purchase emails link back to this site."
	}
	return true, ""
}

func sessionBuyerEmail(sess *stripe.CheckoutSession) string {
	if sess.CustomerDetails != nil && strings.TrimSpace(sess.CustomerDetails.Email) != "" {
		return strings.TrimSpace(sess.CustomerDetails.Email)
	}
	return strings.TrimSpace(sess.CustomerEmail)
}

// recordPaidSession returns the purchase for a paid checkout session,
// creating it the first time. The caller must hold lockSession(sess.ID). An
// existing purchase must be for the item named in the session's metadata.
func recordPaidSession(app core.App, purchaseService *services.PurchaseService, sess *stripe.CheckoutSession) (*core.Record, error) {
	if sess.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
		return nil, errSessionNotPaid
	}
	contentType := sess.Metadata["content_type"]
	contentID := sess.Metadata["content_id"]
	if contentType == "" || contentID == "" {
		return nil, errSessionMismatch
	}

	existing, err := app.FindFirstRecordByFilter("purchases", "stripe_checkout_session_id = {:sid}", map[string]any{"sid": sess.ID})
	if err == nil {
		if existing.GetString("content_type") != contentType || existing.GetString("content_id") != contentID {
			return nil, errSessionMismatch
		}
		return existing, nil
	}

	buyerEmail := sessionBuyerEmail(sess)
	if buyerEmail == "" {
		return nil, errSessionNoEmail
	}

	accessToken, err := purchaseService.GenerateAccessToken()
	if err != nil {
		return nil, err
	}
	collection, err := app.FindCollectionByNameOrId("purchases")
	if err != nil {
		return nil, err
	}

	purchase := core.NewRecord(collection)
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
		return nil, err
	}

	app.Logger().Info("Purchase created", "purchase_id", purchase.Id, "session_id", sess.ID)
	if couponCode := sess.Metadata["coupon_code"]; couponCode != "" {
		IncrementCouponUses(app, couponCode)
	}
	return purchase, nil
}

// deliverPurchaseAccess emails the purchase's access link unless it has
// already gone out. With resend it sends again, at most once per cooldown.
// The sent time is only recorded after the mail server accepts the message,
// so a failed send is retried by the next webhook or return-page request.
func deliverPurchaseAccess(app core.App, purchase *core.Record, resend bool) (retryAfter time.Duration, err error) {
	sentAt := purchase.GetDateTime("access_email_sent_at")
	if !sentAt.IsZero() {
		if !resend {
			return 0, nil
		}
		if wait := accessEmailResendCooldown - time.Since(sentAt.Time()); wait > 0 {
			return wait, nil
		}
	}

	if ready, reason := purchaseDeliveryReady(app); !ready {
		app.Logger().Error("Purchase access email not sent: "+reason, "purchase_id", purchase.Id)
		return 0, errDeliveryUnconfigured
	}

	if err := sendPurchaseAccessEmail(app, purchase); err != nil {
		app.Logger().Error("Purchase access email failed; it will be retried", "purchase_id", purchase.Id)
		return 0, errDeliveryFailed
	}

	purchase.Set("access_email_sent_at", types.NowDateTime())
	if err := app.Save(purchase); err != nil {
		// The buyer has the email. A later retry may send it once more,
		// which is better than claiming a send that didn't happen.
		app.Logger().Error("Failed to record purchase email delivery", "purchase_id", purchase.Id)
	}
	return 0, nil
}

func purchaseItemTitle(app core.App, purchase *core.Record) string {
	if record, err := app.FindRecordById(purchase.GetString("content_type"), purchase.GetString("content_id")); err == nil {
		if title := strings.TrimSpace(record.GetString("title")); title != "" {
			return title
		}
	}
	return "your purchase"
}

func sendPurchaseAccessEmail(app core.App, purchase *core.Record) error {
	link := fmt.Sprintf("%s/api/content/access/%s", publicAppURL(app), purchase.GetString("access_token"))
	title := purchaseItemTitle(app, purchase)
	safeTitle := html.EscapeString(title)
	safeLink := html.EscapeString(link)
	message := &mailer.Message{
		From: mail.Address{
			Name:    app.Settings().Meta.SenderName,
			Address: app.Settings().Meta.SenderAddress,
		},
		To:      []mail.Address{{Address: purchase.GetString("buyer_email")}},
		Subject: fmt.Sprintf("Your access to %s", title),
		HTML: fmt.Sprintf(`<p>Thanks for buying <strong>%s</strong>.</p>
<p><a href="%s">Open your purchase</a></p>
<p>This link signs you in to everything bought with this email address, so keep it to yourself. You can use it again on another device.</p>`, safeTitle, safeLink),
		Text: fmt.Sprintf("Thanks for buying %s.\n\nOpen your purchase: %s\n\nThis link signs you in to everything bought with this email address, so keep it to yourself. You can use it again on another device.\n", title, link),
	}
	return app.NewMailClient().Send(message)
}

// checkoutSessionState sorts an unpaid session into "pending" (still open, or
// a delayed payment that hasn't settled) or "failed" (expired, cancelled or
// declined).
func checkoutSessionState(sess *stripe.CheckoutSession) string {
	if sess.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid && sess.Status == stripe.CheckoutSessionStatusComplete {
		return "paid"
	}
	switch sess.Status {
	case stripe.CheckoutSessionStatusExpired:
		return "failed"
	case stripe.CheckoutSessionStatusOpen:
		return "pending"
	}
	if sess.PaymentStatus == stripe.CheckoutSessionPaymentStatusNoPaymentRequired {
		return "failed"
	}
	if pi := sess.PaymentIntent; pi != nil {
		switch pi.Status {
		case stripe.PaymentIntentStatusCanceled, stripe.PaymentIntentStatusRequiresPaymentMethod:
			return "failed"
		}
	}
	return "pending"
}

func stripeErrorCode(err error) string {
	var se *stripe.Error
	if errors.As(err, &se) {
		return string(se.Code)
	}
	return "request_failed"
}

// withCheckoutSessionID adds Stripe's session placeholder to the success URL
// so the return page can confirm the session. Stripe needs the braces
// literally, so the value isn't URL-encoded.
func withCheckoutSessionID(successURL string) string {
	if strings.Contains(successURL, "{CHECKOUT_SESSION_ID}") {
		return successURL
	}
	fragment := ""
	if i := strings.Index(successURL, "#"); i >= 0 {
		successURL, fragment = successURL[:i], successURL[i:]
	}
	sep := "?"
	if strings.Contains(successURL, "?") {
		sep = "&"
	}
	return successURL + sep + "session_id={CHECKOUT_SESSION_ID}" + fragment
}

// handleCheckoutConfirm serves POST /api/content/checkout/confirm.
func handleCheckoutConfirm(app core.App, purchaseService *services.PurchaseService, stripeKey func() string) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var req struct {
			SessionID   string `json:"session_id"`
			ContentType string `json:"content_type"`
			Slug        string `json:"slug"`
			Resend      bool   `json:"resend"`
		}
		if err := e.BindBody(&req); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}
		req.SessionID = strings.TrimSpace(req.SessionID)
		if !checkoutSessionIDPattern.MatchString(req.SessionID) {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid session_id"})
		}

		key := stripeKey()
		if key == "" {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Stripe not configured"})
		}

		sess, err := fetchCheckoutSession(key, req.SessionID)
		if err != nil {
			var se *stripe.Error
			if errors.As(err, &se) && se.HTTPStatusCode == http.StatusNotFound {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "checkout session not found"})
			}
			app.Logger().Warn("Failed to fetch checkout session", "code", stripeErrorCode(err))
			return e.JSON(http.StatusBadGateway, map[string]string{"error": "could not check payment status, try again shortly"})
		}
		// Stripe looks sessions up by ID, but check anyway: the reply below
		// is about this ID and nothing else.
		if sess == nil || sess.ID != req.SessionID {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "checkout session not found"})
		}

		contentType := sess.Metadata["content_type"]
		contentID := sess.Metadata["content_id"]
		if contentType == "" || contentID == "" {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "checkout session not found"})
		}
		// The return page names the item it expects. A session for another
		// item is refused rather than reported on.
		if req.ContentType != "" || req.Slug != "" {
			matches := req.ContentType == contentType
			if matches && req.Slug != "" {
				record, err := app.FindRecordById(contentType, contentID)
				matches = err == nil && record.GetString("slug") == req.Slug
			}
			if !matches {
				return e.JSON(http.StatusConflict, map[string]string{"error": "checkout session is for a different item"})
			}
		}

		switch checkoutSessionState(sess) {
		case "pending":
			return e.JSON(http.StatusOK, map[string]string{"status": "pending"})
		case "failed":
			return e.JSON(http.StatusOK, map[string]string{"status": "failed"})
		}

		unlock := lockSession(sess.ID)
		defer unlock()

		purchase, err := recordPaidSession(app, purchaseService, sess)
		switch {
		case errors.Is(err, errSessionMismatch):
			app.Logger().Error("Checkout session metadata does not match its purchase", "session_id", sess.ID)
			return e.JSON(http.StatusConflict, map[string]string{"error": "checkout session is for a different item"})
		case errors.Is(err, errSessionNoEmail):
			app.Logger().Error("Paid checkout session has no customer email", "session_id", sess.ID)
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "payment has no email address; contact the site owner"})
		case err != nil:
			app.Logger().Error("Failed to record purchase", "session_id", sess.ID)
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to record purchase, try again shortly"})
		}

		if purchase.GetString("status") != "completed" {
			return e.JSON(http.StatusOK, map[string]string{"status": "refunded"})
		}

		retryAfter, err := deliverPurchaseAccess(app, purchase, req.Resend)
		switch {
		case errors.Is(err, errDeliveryUnconfigured):
			return e.JSON(http.StatusServiceUnavailable, map[string]string{
				"status": "delivery_unavailable",
				"error":  "Your payment went through, but this site can't send email yet. The site owner has been alerted; try again later or contact them.",
			})
		case err != nil:
			return e.JSON(http.StatusBadGateway, map[string]string{
				"status": "delivery_failed",
				"error":  "Your payment went through, but the access email couldn't be sent. Try again in a moment.",
			})
		}

		resp := map[string]any{
			"status":     "confirmation_sent",
			"email_hint": maskEmail(purchase.GetString("buyer_email")),
		}
		if retryAfter > 0 {
			resp["retry_after"] = int(retryAfter.Seconds()) + 1
		}
		return e.JSON(http.StatusOK, resp)
	}
}
