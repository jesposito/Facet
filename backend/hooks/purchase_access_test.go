package hooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stripe/stripe-go/v82"
)

// signedWebhook builds a Stripe event carrying any object, signed with the
// fixture's webhook secret.
func signedWebhook(t *testing.T, id, eventType string, object any) testRequest {
	t.Helper()
	raw, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	event, _ := json.Marshal(map[string]any{
		"id": "evt_" + id, "object": "event", "type": eventType,
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

// refundEvent is a charge.refunded event for a fixture session's payment.
// refunded is the running total Stripe reports across all refunds.
func refundEvent(t *testing.T, sessionID string, refunded int64) testRequest {
	t.Helper()
	return signedWebhook(t, fmt.Sprintf("%s_refund_%d", sessionID, refunded), "charge.refunded", map[string]any{
		"id": "ch_" + sessionID, "object": "charge",
		"amount": 500, "amount_refunded": refunded, "refunded": refunded >= 500,
		"payment_intent": "pi_" + sessionID,
	})
}

func (d *deliveryFixture) purchaseStatus(t *testing.T, sessionID string) string {
	t.Helper()
	p, err := d.app.FindFirstRecordByFilter("purchases", "stripe_checkout_session_id = {:sid}", map[string]any{"sid": sessionID})
	if err != nil {
		t.Fatal(err)
	}
	return p.GetString("status")
}

func TestPartialRefundKeepsAccessAndFullRefundEndsIt(t *testing.T) {
	d := newDeliveryFixture(t, true)
	sess := d.paid("cs_test_refund", "buyer@example.com")
	assertStatus(t, d.webhook(t, "checkout.session.completed", sess), http.StatusOK)
	link := d.accessLink(t, "buyer@example.com")
	cookie := learnerCookie(do(t, d.h, testRequest{url: link}))

	// A goodwill refund of $1 leaves the buyer with their purchase.
	rec := do(t, d.h, refundEvent(t, sess.ID, 100))
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"status":"partial_refund"`)
	if s := d.purchaseStatus(t, sess.ID); s != "completed" {
		t.Fatalf("status after partial refund = %q", s)
	}
	assertContains(t, d.readPost(t, "paid-post", cookie), "PAID ENDING")

	// Refunding the rest ends access straight away.
	rec = do(t, d.h, refundEvent(t, sess.ID, 500))
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"status":"success"`)
	assertNotContains(t, d.readPost(t, "paid-post", cookie), "PAID ENDING")

	// Repeats, a late partial-refund event, and checkout retries all leave it
	// refunded.
	assertContains(t, do(t, d.h, refundEvent(t, sess.ID, 500)), `"status":"already_processed"`)
	assertContains(t, do(t, d.h, refundEvent(t, sess.ID, 100)), `"status":"already_processed"`)
	assertContains(t, d.webhook(t, "checkout.session.completed", sess), `"status":"already_processed"`)
	assertContains(t, d.webhook(t, "checkout.session.async_payment_succeeded", sess), `"status":"already_processed"`)
	assertContains(t, d.confirm(t, `{"session_id":"cs_test_refund"}`), `"status":"refunded"`)
	if s := d.purchaseStatus(t, sess.ID); s != "refunded" {
		t.Fatalf("status after retries = %q", s)
	}
	assertStatus(t, do(t, d.h, testRequest{url: link}), http.StatusNotFound)
	assertNotContains(t, d.readPost(t, "paid-post", cookie), "PAID ENDING")
}

func TestRefundBeforeCheckoutIsRetriedNotLost(t *testing.T) {
	d := newDeliveryFixture(t, true)
	sess := d.paid("cs_test_early_refund", "buyer@example.com")

	// Stripe delivered the refund first. Answering with an error makes it
	// retry, and nothing is recorded yet.
	assertStatus(t, do(t, d.h, refundEvent(t, sess.ID, 500)), http.StatusNotFound)
	if d.purchaseCount(t) != 0 {
		t.Fatal("a refund created a purchase")
	}

	assertStatus(t, d.webhook(t, "checkout.session.completed", sess), http.StatusOK)
	assertContains(t, do(t, d.h, refundEvent(t, sess.ID, 500)), `"status":"success"`)
	assertContains(t, d.webhook(t, "checkout.session.completed", sess), `"status":"already_processed"`)
	if s := d.purchaseStatus(t, sess.ID); s != "refunded" {
		t.Fatalf("status = %q, want refunded", s)
	}
}

func TestConcurrentCheckoutRetriesCannotUndoARefund(t *testing.T) {
	d := newDeliveryFixture(t, true)
	sess := d.paid("cs_test_race", "buyer@example.com")
	// The first delivery fails to send, so every retry below goes on to
	// send and record the email while the refund arrives.
	d.failMail.Store(true)
	assertStatus(t, d.webhook(t, "checkout.session.completed", sess), http.StatusInternalServerError)
	d.failMail.Store(false)

	requests := []testRequest{refundEvent(t, sess.ID, 500)}
	for i := 0; i < 8; i++ {
		requests = append(requests,
			webhookRequest(t, "checkout.session.completed", sess),
			testRequest{method: http.MethodPost, url: "/api/content/checkout/confirm", body: `{"session_id":"cs_test_race","resend":true}`})
	}
	var wg sync.WaitGroup
	for _, r := range requests {
		wg.Add(1)
		go func(r testRequest) {
			defer wg.Done()
			req := httptest.NewRequest(r.method, r.url, strings.NewReader(r.body))
			req.RemoteAddr = fmt.Sprintf("10.9.%d.%d:4000", requestCounter.Add(1)&255, requestCounter.Add(1)&255)
			req.Header.Set("Content-Type", "application/json")
			for k, v := range r.headers {
				req.Header.Set(k, v)
			}
			d.h.ServeHTTP(httptest.NewRecorder(), req)
		}(r)
	}
	wg.Wait()

	if s := d.purchaseStatus(t, sess.ID); s != "refunded" {
		t.Fatalf("status = %q after racing checkout retries, want refunded", s)
	}
}

func TestCheckoutRedirectsMustBeOnTheConfiguredSite(t *testing.T) {
	// With email off, checkout stops just after redirect validation, so a
	// 503 means the URLs were accepted and Stripe is never called.
	d := newDeliveryFixture(t, false)
	checkout := func(appURL, success, cancel string, headers map[string]string) *httptest.ResponseRecorder {
		settings := d.app.Settings()
		settings.Meta.AppURL = appURL
		if err := d.app.Save(settings); err != nil {
			t.Fatal(err)
		}
		body := fmt.Sprintf(`{"content_type":"posts","content_id":"%s","success_url":%q,"cancel_url":%q}`, d.post.Id, success, cancel)
		return do(t, d.h, testRequest{method: http.MethodPost, url: "/api/content/checkout", body: body, headers: headers})
	}
	// httptest requests arrive with Host example.com; the forwarded headers
	// are what a client or a careless proxy can set to anything.
	spoofed := map[string]string{"X-Forwarded-Host": "attacker.example", "X-Forwarded-Proto": "https"}

	for _, tc := range []struct {
		name, appURL, success, cancel string
		headers                       map[string]string
		want                          int
	}{
		{"configured origin", "https://facet.example", "https://facet.example/posts/paid-post?ok", "https://facet.example/posts/paid-post", nil, http.StatusServiceUnavailable},
		{"explicit default port", "https://facet.example", "https://facet.example:443/ok", "https://FACET.example/no", nil, http.StatusServiceUnavailable},
		{"spoofed forwarded host", "https://facet.example", "https://attacker.example/ok", "https://attacker.example/no", spoofed, http.StatusBadRequest},
		{"spoofed host cannot unlock http", "https://facet.example", "http://facet.example/ok", "http://facet.example/no", spoofed, http.StatusBadRequest},
		{"different port", "https://facet.example", "https://facet.example:8443/ok", "https://facet.example/no", nil, http.StatusBadRequest},
		{"look-alike host", "https://facet.example", "https://facet.example.attacker.example/ok", "https://facet.example/no", nil, http.StatusBadRequest},
		{"userinfo trick", "https://facet.example", "https://facet.example@attacker.example/ok", "https://facet.example/no", nil, http.StatusBadRequest},
		{"self-hosted LAN over http", "http://192.168.1.20:8090", "http://192.168.1.20:8090/ok", "http://192.168.1.20:8090/no", nil, http.StatusServiceUnavailable},
		{"LAN origin needs its port", "http://192.168.1.20:8090", "http://192.168.1.20/ok", "http://192.168.1.20:8090/no", nil, http.StatusBadRequest},
		{"https site refuses http", "https://facet.example", "http://facet.example/ok", "http://facet.example/no", nil, http.StatusBadRequest},
		{"request host is not the site", "https://facet.example", "http://example.com/ok", "http://example.com/no", nil, http.StatusBadRequest},
		// PocketBase's default when the owner never set an address.
		{"unconfigured default refuses the request host", "http://localhost:8090", "http://example.com/ok", "http://example.com/no", nil, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertStatus(t, checkout(tc.appURL, tc.success, tc.cancel, tc.headers), tc.want)
		})
	}
	if d.fetches.Load() != 0 || d.purchaseCount(t) != 0 {
		t.Fatalf("checkout validation reached Stripe or recorded a purchase")
	}

	// PocketBase won't store a blank application URL, but APP_URL may be
	// the only source. With neither, nothing is accepted.
	for _, u := range []string{"https://facet.example/ok", "http://example.com/ok", "http://localhost/ok"} {
		if validateRedirectURL(u, "") {
			t.Fatalf("%s accepted with no configured site URL", u)
		}
	}
	for _, origin := range []string{"ftp://facet.example", "facet.example", "https://"} {
		if validateRedirectURL("https://facet.example/ok", origin) {
			t.Fatalf("accepted against malformed origin %q", origin)
		}
	}
}

func TestAccessLinkUnlocksMoreThanAHundredPurchases(t *testing.T) {
	d := newDeliveryFixture(t, true)
	const n = 101
	var first, last string
	for i := 0; i < n; i++ {
		slug := fmt.Sprintf("bulk-%03d", i)
		post := d.create(t, "posts", map[string]any{"title": slug, "slug": slug, "content": "One.\n\nTwo.\n\nENDING " + slug, "visibility": "public", "access_tier": "paid", "price": 500, "published_at": "2026-01-03 00:00:00.000Z"})
		d.completedPurchase(t, "collector@example.com", "posts", post.Id)
		if i == 0 {
			first = slug
		}
		last = slug
	}
	purchase, err := d.app.FindFirstRecordByFilter("purchases", "buyer_email = 'collector@example.com'")
	if err != nil {
		t.Fatal(err)
	}

	rec := do(t, d.h, testRequest{url: "/api/content/access/" + purchase.GetString("access_token")})
	assertStatus(t, rec, http.StatusFound)
	cookie := learnerCookie(rec)
	if len(cookie) > 1024 {
		t.Fatalf("access cookie is %d bytes for %d purchases", len(cookie), n)
	}
	for _, slug := range []string{first, last} {
		assertContains(t, d.readPost(t, slug, cookie), "ENDING "+slug)
	}
	// Someone else's paid post stays locked.
	assertNotContains(t, d.readPost(t, "victim-post", cookie), "VICTIM ENDING")

	// A purchase made after the link was followed is covered too, and a
	// refund removes it again at once.
	late := d.create(t, "posts", map[string]any{"title": "Late", "slug": "late", "content": "One.\n\nTwo.\n\nLATE ENDING", "visibility": "public", "access_tier": "paid", "price": 500, "published_at": "2026-01-04 00:00:00.000Z"})
	latePurchase := d.completedPurchase(t, "collector@example.com", "posts", late.Id)
	assertContains(t, d.readPost(t, "late", cookie), "LATE ENDING")
	latePurchase.Set("status", "refunded")
	if err := d.app.Save(latePurchase); err != nil {
		t.Fatal(err)
	}
	assertNotContains(t, d.readPost(t, "late", cookie), "LATE ENDING")
}

func TestFreeCouponCookieStaysScopedToItsItem(t *testing.T) {
	d := newDeliveryFixture(t, true)
	d.create(t, "coupons", map[string]any{"code": "FREEPOST", "discount_type": "percentage", "discount_value": 100, "applicable_type": "all", "max_uses": 5, "current_uses": 0, "active": true})
	rec := do(t, d.h, testRequest{method: http.MethodPost, url: "/api/content/checkout", body: `{"content_type":"posts","content_id":"` + d.post.Id + `","success_url":"https://facet.example/ok","cancel_url":"https://facet.example/no","coupon_code":"FREEPOST"}`})
	assertStatus(t, rec, http.StatusOK)
	var free struct {
		CheckoutURL string `json:"checkout_url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &free); err != nil {
		t.Fatal(err)
	}
	rec = do(t, d.h, testRequest{url: free.CheckoutURL})
	assertStatus(t, rec, http.StatusFound)
	cookie := learnerCookie(rec)

	token := strings.TrimPrefix(cookie, "facet_purchases=")
	ids, email, err := d.purchase.ValidatePurchaseJWT(token)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || strings.HasPrefix(ids[0], "buyer:") || !strings.HasSuffix(email, "@noreply.facet.run") {
		t.Fatalf("coupon cookie = %v for %s, want one explicit purchase", ids, email)
	}

	// Another purchase under the same synthetic address is not reachable.
	d.completedPurchase(t, email, "posts", d.other.Id)
	assertContains(t, d.readPost(t, "paid-post", cookie), "PAID ENDING")
	assertNotContains(t, d.readPost(t, "victim-post", cookie), "VICTIM ENDING")
}
