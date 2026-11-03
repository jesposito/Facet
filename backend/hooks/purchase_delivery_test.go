package hooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stripe/stripe-go/v82"
)

const deliveryWebhookSecret = "whsec_test_delivery"

// deliveryFixture is a site that sells one post, with Stripe and the mailer
// replaced by stubs. Nothing here talks to Stripe or a mail server.
type deliveryFixture struct {
	*securityFixture
	h        http.Handler
	post     *core.Record
	other    *core.Record
	mu       sync.Mutex
	sessions map[string]*stripe.CheckoutSession
	fetches  atomic.Int64
	failMail atomic.Bool
}

func newDeliveryFixture(t *testing.T, smtp bool) *deliveryFixture {
	t.Helper()
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_stub")
	t.Setenv("STRIPE_WEBHOOK_SECRET", deliveryWebhookSecret)

	f := newSecurityFixture(t)
	settings := f.app.Settings()
	settings.SMTP.Enabled = smtp
	settings.SMTP.Host = "smtp.example"
	settings.Meta.AppURL = "https://facet.example"
	settings.Meta.SenderAddress = "site@facet.example"
	if err := f.app.Save(settings); err != nil {
		t.Fatal(err)
	}
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	RegisterPurchaseHooks(f.pb, f.crypto, f.rl)
	RegisterCourseHooks(f.pb, &services.PlanConfig{Features: services.PlanFeatures{Courses: true}}, f.crypto, f.rl)

	d := &deliveryFixture{securityFixture: f, sessions: map[string]*stripe.CheckoutSession{}}
	d.post = f.create(t, "posts", map[string]any{"title": "Paid post", "slug": "paid-post", "content": paidPostBody, "visibility": "public", "access_tier": "paid", "price": 500, "published_at": "2026-01-01 00:00:00.000Z"})
	d.other = f.create(t, "posts", map[string]any{"title": "Victim's post", "slug": "victim-post", "content": "One.\n\nTwo.\n\nVICTIM ENDING", "visibility": "public", "access_tier": "paid", "price": 900, "published_at": "2026-01-02 00:00:00.000Z"})

	f.app.OnMailerSend().BindFunc(func(e *core.MailerEvent) error {
		if d.failMail.Load() {
			return errors.New("smtp: 451 temporary failure")
		}
		return e.Next()
	})

	original := fetchCheckoutSession
	fetchCheckoutSession = func(key, id string) (*stripe.CheckoutSession, error) {
		d.fetches.Add(1)
		if key != "sk_test_stub" {
			return nil, errors.New("unexpected key")
		}
		d.mu.Lock()
		defer d.mu.Unlock()
		sess, ok := d.sessions[id]
		if !ok {
			return nil, &stripe.Error{HTTPStatusCode: http.StatusNotFound, Code: stripe.ErrorCodeResourceMissing}
		}
		copy := *sess
		return &copy, nil
	}
	t.Cleanup(func() { fetchCheckoutSession = original })

	d.h = f.serve(t)
	return d
}

func (d *deliveryFixture) session(id, email, status string, payment stripe.CheckoutSessionPaymentStatus, meta map[string]string) *stripe.CheckoutSession {
	if meta == nil {
		meta = map[string]string{"content_type": "posts", "content_id": d.post.Id}
	}
	sess := &stripe.CheckoutSession{
		ID:              id,
		Status:          stripe.CheckoutSessionStatus(status),
		PaymentStatus:   payment,
		Metadata:        meta,
		CustomerDetails: &stripe.CheckoutSessionCustomerDetails{Email: email},
		PaymentIntent:   &stripe.PaymentIntent{ID: "pi_" + id, Status: stripe.PaymentIntentStatusSucceeded},
		AmountTotal:     500,
		Currency:        "usd",
	}
	d.mu.Lock()
	d.sessions[id] = sess
	d.mu.Unlock()
	return sess
}

func (d *deliveryFixture) paid(id, email string) *stripe.CheckoutSession {
	return d.session(id, email, "complete", stripe.CheckoutSessionPaymentStatusPaid, nil)
}

func webhookRequest(t *testing.T, eventType string, sess *stripe.CheckoutSession) testRequest {
	t.Helper()
	raw, err := json.Marshal(sess)
	if err != nil {
		t.Fatal(err)
	}
	event, _ := json.Marshal(map[string]any{
		"id": "evt_" + sess.ID, "object": "event", "type": eventType,
		"api_version": stripe.APIVersion, "data": map[string]any{"object": json.RawMessage(raw)},
	})
	ts := time.Now().Unix()
	mac := hmac.New(sha256.New, []byte(deliveryWebhookSecret))
	fmt.Fprintf(mac, "%d.%s", ts, event)
	return testRequest{
		method:  http.MethodPost,
		url:     "/api/content/webhook",
		body:    string(event),
		headers: map[string]string{"Stripe-Signature": fmt.Sprintf("t=%d,v1=%s", ts, hex.EncodeToString(mac.Sum(nil)))},
	}
}

func (d *deliveryFixture) webhook(t *testing.T, eventType string, sess *stripe.CheckoutSession) *httptest.ResponseRecorder {
	t.Helper()
	return do(t, d.h, webhookRequest(t, eventType, sess))
}

func (d *deliveryFixture) confirm(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := do(t, d.h, testRequest{method: http.MethodPost, url: "/api/content/checkout/confirm", body: body})
	// The return page proves nothing about the inbox, so it never signs
	// anyone in.
	if c := learnerCookie(rec); c != "" {
		t.Fatalf("confirm set a purchase cookie: %s", c)
	}
	return rec
}

var accessLinkPattern = regexp.MustCompile(`https://facet\.example(/api/content/access/[A-Za-z0-9_-]+)`)

func (d *deliveryFixture) accessLink(t *testing.T, to string) string {
	t.Helper()
	msg := d.app.TestMailer.LastMessage()
	if len(msg.To) != 1 || msg.To[0].Address != to {
		t.Fatalf("last email went to %v, want %s", msg.To, to)
	}
	m := accessLinkPattern.FindStringSubmatch(msg.Text)
	if m == nil {
		t.Fatalf("no access link in email:\n%s", msg.Text)
	}
	return m[1]
}

func (d *deliveryFixture) purchaseCount(t *testing.T) int {
	t.Helper()
	n, err := d.app.CountRecords("purchases")
	if err != nil {
		t.Fatal(err)
	}
	return int(n)
}

func (d *deliveryFixture) readPost(t *testing.T, slug, cookie string) *httptest.ResponseRecorder {
	t.Helper()
	return do(t, d.h, testRequest{url: "/api/post/" + slug, headers: map[string]string{"Cookie": cookie}})
}

func TestPaidCheckoutEmailsAWorkingAccessLink(t *testing.T) {
	d := newDeliveryFixture(t, true)
	sess := d.paid("cs_test_paid", "buyer@example.com")

	rec := d.webhook(t, "checkout.session.completed", sess)
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"status":"success"`)
	if d.app.TestMailer.TotalSend() != 1 {
		t.Fatalf("sent %d emails, want 1", d.app.TestMailer.TotalSend())
	}
	link := d.accessLink(t, "buyer@example.com")

	// Stripe may deliver the same event again; nothing new happens.
	rec = d.webhook(t, "checkout.session.completed", sess)
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"status":"already_processed"`)

	// The return page learns that the email went out, and nothing more.
	rec = d.confirm(t, `{"session_id":"cs_test_paid","content_type":"posts","slug":"paid-post"}`)
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"status":"confirmation_sent"`)
	assertContains(t, rec, `"email_hint":"b***@example.com"`)
	purchase, err := d.app.FindFirstRecordByFilter("purchases", "stripe_checkout_session_id = 'cs_test_paid'")
	if err != nil {
		t.Fatal(err)
	}
	assertNotContains(t, rec, purchase.GetString("access_token"))
	if d.app.TestMailer.TotalSend() != 1 || d.purchaseCount(t) != 1 {
		t.Fatalf("emails = %d, purchases = %d; want 1 and 1", d.app.TestMailer.TotalSend(), d.purchaseCount(t))
	}

	// Following the emailed link signs the buyer in and opens the post.
	rec = do(t, d.h, testRequest{url: link})
	assertStatus(t, rec, http.StatusFound)
	if loc := rec.Header().Get("Location"); loc != "/posts/paid-post" {
		t.Fatalf("redirect = %q", loc)
	}
	cookie := learnerCookie(rec)
	if cookie == "" {
		t.Fatal("access link set no cookie")
	}
	assertContains(t, d.readPost(t, "paid-post", cookie), "PAID ENDING")
	assertNotContains(t, d.readPost(t, "victim-post", cookie), "VICTIM ENDING")
}

func TestReturnPageRecordsPaymentBeforeTheWebhook(t *testing.T) {
	d := newDeliveryFixture(t, true)
	d.create(t, "coupons", map[string]any{"code": "HALF", "discount_type": "percentage", "discount_value": 50, "applicable_type": "all", "max_uses": 10, "current_uses": 0, "active": true})
	sess := d.session("cs_test_early", "early@example.com", "complete", stripe.CheckoutSessionPaymentStatusPaid,
		map[string]string{"content_type": "posts", "content_id": d.post.Id, "coupon_code": "HALF"})

	rec := d.confirm(t, `{"session_id":"cs_test_early"}`)
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"status":"confirmation_sent"`)

	rec = d.webhook(t, "checkout.session.completed", sess)
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"status":"already_processed"`)

	if d.purchaseCount(t) != 1 || d.app.TestMailer.TotalSend() != 1 {
		t.Fatalf("purchases = %d, emails = %d; want 1 and 1", d.purchaseCount(t), d.app.TestMailer.TotalSend())
	}
	coupon, _ := d.app.FindFirstRecordByFilter("coupons", "code = 'HALF'")
	if coupon.GetInt("current_uses") != 1 {
		t.Fatalf("coupon uses = %d, want 1", coupon.GetInt("current_uses"))
	}
}

func TestUnsettledPaymentsGrantNothing(t *testing.T) {
	d := newDeliveryFixture(t, true)

	delayed := d.session("cs_test_delayed", "slow@example.com", "complete", stripe.CheckoutSessionPaymentStatusUnpaid, nil)
	delayed.PaymentIntent.Status = stripe.PaymentIntentStatusProcessing
	rec := d.webhook(t, "checkout.session.completed", delayed)
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"status":"awaiting_payment"`)
	assertContains(t, d.confirm(t, `{"session_id":"cs_test_delayed"}`), `"status":"pending"`)

	d.session("cs_test_open", "open@example.com", "open", stripe.CheckoutSessionPaymentStatusUnpaid, nil)
	assertContains(t, d.confirm(t, `{"session_id":"cs_test_open"}`), `"status":"pending"`)

	declined := d.session("cs_test_declined", "slow@example.com", "complete", stripe.CheckoutSessionPaymentStatusUnpaid, nil)
	declined.PaymentIntent.Status = stripe.PaymentIntentStatusRequiresPaymentMethod
	rec = d.webhook(t, "checkout.session.async_payment_failed", declined)
	assertContains(t, rec, `"status":"payment_failed"`)
	assertContains(t, d.confirm(t, `{"session_id":"cs_test_declined"}`), `"status":"failed"`)

	d.session("cs_test_expired", "gone@example.com", "expired", stripe.CheckoutSessionPaymentStatusUnpaid, nil)
	assertContains(t, d.confirm(t, `{"session_id":"cs_test_expired"}`), `"status":"failed"`)

	if d.purchaseCount(t) != 0 || d.app.TestMailer.TotalSend() != 0 {
		t.Fatalf("purchases = %d, emails = %d; want none", d.purchaseCount(t), d.app.TestMailer.TotalSend())
	}

	// When the delayed payment settles, the buyer gets their link.
	settled := d.paid("cs_test_delayed", "slow@example.com")
	rec = d.webhook(t, "checkout.session.async_payment_succeeded", settled)
	assertContains(t, rec, `"status":"success"`)
	d.accessLink(t, "slow@example.com")
}

func TestFailedAccessEmailIsRetried(t *testing.T) {
	d := newDeliveryFixture(t, true)
	sess := d.paid("cs_test_retry", "retry@example.com")

	d.failMail.Store(true)
	rec := d.webhook(t, "checkout.session.completed", sess)
	assertStatus(t, rec, http.StatusInternalServerError)
	assertNotContains(t, rec, "451")
	rec = d.confirm(t, `{"session_id":"cs_test_retry"}`)
	assertStatus(t, rec, http.StatusBadGateway)
	assertContains(t, rec, `"status":"delivery_failed"`)
	if d.purchaseCount(t) != 1 || d.app.TestMailer.TotalSend() != 0 {
		t.Fatalf("purchases = %d, emails = %d; want 1 and 0", d.purchaseCount(t), d.app.TestMailer.TotalSend())
	}

	// Stripe retries the event once the mail server recovers.
	d.failMail.Store(false)
	rec = d.webhook(t, "checkout.session.completed", sess)
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"status":"success"`)
	d.accessLink(t, "retry@example.com")

	// Asking to resend straight away is held back by the cooldown.
	rec = d.confirm(t, `{"session_id":"cs_test_retry","resend":true}`)
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"retry_after"`)
	if d.app.TestMailer.TotalSend() != 1 || d.purchaseCount(t) != 1 {
		t.Fatalf("emails = %d, purchases = %d; want 1 and 1", d.app.TestMailer.TotalSend(), d.purchaseCount(t))
	}

	// After the cooldown the buyer can have the link sent again.
	purchase, _ := d.app.FindFirstRecordByFilter("purchases", "stripe_checkout_session_id = 'cs_test_retry'")
	purchase.Set("access_email_sent_at", time.Now().Add(-time.Hour))
	if err := d.app.Save(purchase); err != nil {
		t.Fatal(err)
	}
	rec = d.confirm(t, `{"session_id":"cs_test_retry","resend":true}`)
	assertContains(t, rec, `"status":"confirmation_sent"`)
	if d.app.TestMailer.TotalSend() != 2 {
		t.Fatalf("emails = %d, want 2 after resend", d.app.TestMailer.TotalSend())
	}
}

func TestSalesNeedWorkingEmail(t *testing.T) {
	d := newDeliveryFixture(t, false)

	rec := do(t, d.h, testRequest{url: "/api/stripe/ready"})
	assertContains(t, rec, `"reason":"email_not_configured"`)

	// Checkout is refused before Stripe is asked to take any money.
	rec = do(t, d.h, testRequest{method: http.MethodPost, url: "/api/content/checkout", body: `{"content_type":"posts","content_id":"` + d.post.Id + `","success_url":"https://facet.example/ok","cancel_url":"https://facet.example/no"}`})
	assertStatus(t, rec, http.StatusServiceUnavailable)

	// A payment that arrives anyway is kept, and delivered once email works.
	sess := d.paid("cs_test_nomail", "patient@example.com")
	assertStatus(t, d.webhook(t, "checkout.session.completed", sess), http.StatusServiceUnavailable)
	rec = d.confirm(t, `{"session_id":"cs_test_nomail"}`)
	assertStatus(t, rec, http.StatusServiceUnavailable)
	assertContains(t, rec, `"status":"delivery_unavailable"`)
	if d.purchaseCount(t) != 1 {
		t.Fatalf("purchases = %d, want 1", d.purchaseCount(t))
	}

	settings := d.app.Settings()
	settings.SMTP.Enabled = true
	if err := d.app.Save(settings); err != nil {
		t.Fatal(err)
	}
	assertContains(t, d.webhook(t, "checkout.session.completed", sess), `"status":"success"`)
	d.accessLink(t, "patient@example.com")
}

func TestConfirmChecksTheSession(t *testing.T) {
	d := newDeliveryFixture(t, true)
	d.paid("cs_test_meta", "buyer@example.com")

	for name, body := range map[string]string{
		"coupon purchase ID":  `{"session_id":"coupon_FREE_abc"}`,
		"empty session":       `{"session_id":""}`,
		"injection attempt":   `{"session_id":"cs_x' || 1=1"}`,
		"unknown session":     `{"session_id":"cs_test_unknown"}`,
		"other item's slug":   `{"session_id":"cs_test_meta","content_type":"posts","slug":"victim-post"}`,
		"other content type":  `{"session_id":"cs_test_meta","content_type":"talks","slug":"paid-post"}`,
		"session without IDs": `{"session_id":"cs_test_nometa"}`,
	} {
		if name == "session without IDs" {
			d.session("cs_test_nometa", "x@example.com", "complete", stripe.CheckoutSessionPaymentStatusPaid, map[string]string{})
		}
		t.Run(name, func(t *testing.T) {
			rec := d.confirm(t, body)
			if rec.Code == http.StatusOK {
				t.Fatalf("accepted: %s", rec.Body.String())
			}
		})
	}

	// A session ID already recorded against another item is refused.
	d.create(t, "purchases", map[string]any{"buyer_email": "buyer@example.com", "content_type": "posts", "content_id": d.other.Id, "stripe_checkout_session_id": "cs_test_swapped", "status": "completed", "access_token": "tok-swapped"})
	d.paid("cs_test_swapped", "buyer@example.com")
	rec := d.confirm(t, `{"session_id":"cs_test_swapped"}`)
	assertStatus(t, rec, http.StatusConflict)

	// The database also refuses a second purchase for one session.
	c, _ := d.app.FindCollectionByNameOrId("purchases")
	dupe := core.NewRecord(c)
	dupe.Load(map[string]any{"buyer_email": "buyer@example.com", "content_type": "posts", "content_id": d.post.Id, "stripe_checkout_session_id": "cs_test_swapped", "status": "completed", "access_token": "tok-dupe"})
	if err := d.app.Save(dupe); err == nil {
		t.Fatal("saved a second purchase for one checkout session")
	}

	if d.app.TestMailer.TotalSend() != 0 {
		t.Fatalf("sent %d emails for rejected sessions", d.app.TestMailer.TotalSend())
	}
	if d.purchaseCount(t) != 1 {
		t.Fatalf("purchases = %d, want only the seeded one", d.purchaseCount(t))
	}
}

func TestCheckoutWithSomeoneElsesEmailUnlocksNothing(t *testing.T) {
	d := newDeliveryFixture(t, true)
	// The victim already owns a post and has a course certificate.
	victimPurchase := d.completedPurchase(t, "victim@example.com", "posts", d.other.Id)
	course := d.create(t, "courses", map[string]any{"title": "Course", "slug": "course", "access_tier": "free", "visibility": "public"})
	d.create(t, "course_certificates", map[string]any{"course": course.Id, "buyer_email": "victim@example.com", "certificate_id": "cert-victim-0001", "course_title": "Course"})

	// The attacker pays, typing the victim's address at Stripe.
	sess := d.paid("cs_test_attacker", "victim@example.com")
	assertStatus(t, d.webhook(t, "checkout.session.completed", sess), http.StatusOK)
	rec := d.confirm(t, `{"session_id":"cs_test_attacker","resend":true}`)
	assertStatus(t, rec, http.StatusOK)
	assertNotContains(t, rec, "victim@example.com")
	assertNotContains(t, rec, victimPurchase.GetString("access_token"))
	assertNotContains(t, rec, "/api/content/access/")

	// The link went to the victim's inbox, not to the attacker.
	link := d.accessLink(t, "victim@example.com")

	// A zero-cost coupon gives the attacker a token straight away, but it
	// belongs to a new identity that owns only the redeemed item.
	d.create(t, "coupons", map[string]any{"code": "FREEALL", "discount_type": "percentage", "discount_value": 100, "applicable_type": "all", "max_uses": 5, "current_uses": 0, "active": true})
	rec = do(t, d.h, testRequest{method: http.MethodPost, url: "/api/content/checkout", body: `{"content_type":"posts","content_id":"` + d.post.Id + `","success_url":"https://facet.example/ok","cancel_url":"https://facet.example/no","coupon_code":"FREEALL"}`})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"status":"authorized"`)
	var free struct {
		CheckoutURL string `json:"checkout_url"`
	}
	json.Unmarshal(rec.Body.Bytes(), &free)
	rec = do(t, d.h, testRequest{url: free.CheckoutURL})
	assertStatus(t, rec, http.StatusFound)
	attackerCookie := learnerCookie(rec)
	_, email, err := d.purchase.ValidatePurchaseJWT(strings.TrimPrefix(attackerCookie, services.PurchaseCookieName+"="))
	if err != nil || strings.EqualFold(email, "victim@example.com") || !strings.HasSuffix(email, "@noreply.facet.run") {
		t.Fatalf("coupon identity = %q, %v", email, err)
	}
	assertContains(t, d.readPost(t, "paid-post", attackerCookie), "PAID ENDING")
	assertNotContains(t, d.readPost(t, "victim-post", attackerCookie), "VICTIM ENDING")
	assertNotContains(t, do(t, d.h, testRequest{url: "/api/courses/course/certificate", headers: map[string]string{"Cookie": attackerCookie}}), "cert-victim-0001")

	// The real owner of the inbox gets everything bought with it.
	rec = do(t, d.h, testRequest{url: link})
	victimCookie := learnerCookie(rec)
	assertContains(t, d.readPost(t, "victim-post", victimCookie), "VICTIM ENDING")
	assertContains(t, d.readPost(t, "paid-post", victimCookie), "PAID ENDING")
	assertContains(t, do(t, d.h, testRequest{url: "/api/courses/course/certificate", headers: map[string]string{"Cookie": victimCookie}}), "cert-victim-0001")
}

func TestConcurrentWebhooksAndConfirmsDeliverOnce(t *testing.T) {
	d := newDeliveryFixture(t, true)
	sess := d.paid("cs_test_burst", "burst@example.com")

	var wg sync.WaitGroup
	start := make(chan struct{})
	codes := make([]int, burst)
	for i := 0; i < burst; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			req := webhookRequest(t, "checkout.session.completed", sess)
			if i%2 == 1 {
				req = testRequest{method: http.MethodPost, url: "/api/content/checkout/confirm", body: `{"session_id":"cs_test_burst"}`}
			}
			codes[i] = do(t, d.h, req).Code
		}(i)
	}
	close(start)
	wg.Wait()

	for i, code := range codes {
		if code != http.StatusOK {
			t.Fatalf("request %d: status %d", i, code)
		}
	}
	if d.purchaseCount(t) != 1 || d.app.TestMailer.TotalSend() != 1 {
		t.Fatalf("purchases = %d, emails = %d; want 1 and 1", d.purchaseCount(t), d.app.TestMailer.TotalSend())
	}
}

func TestSuccessURLCarriesTheSessionID(t *testing.T) {
	for in, want := range map[string]string{
		"https://site.example/checkout/return?type=posts&slug=a":  "https://site.example/checkout/return?type=posts&slug=a&session_id={CHECKOUT_SESSION_ID}",
		"https://site.example/done":                               "https://site.example/done?session_id={CHECKOUT_SESSION_ID}",
		"https://site.example/done#top":                           "https://site.example/done?session_id={CHECKOUT_SESSION_ID}#top",
		"https://site.example/r?session_id={CHECKOUT_SESSION_ID}": "https://site.example/r?session_id={CHECKOUT_SESSION_ID}",
	} {
		if got := withCheckoutSessionID(in); got != want {
			t.Errorf("withCheckoutSessionID(%q) = %q, want %q", in, got, want)
		}
	}
}
