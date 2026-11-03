package hooks

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"facet/services"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

func TestPublicEndpointsOmitEmailAddresses(t *testing.T) {
	f := newSecurityFixture(t)
	plan := &services.PlanConfig{Features: services.PlanFeatures{Courses: true}}
	RegisterTestimonialHooks(f.pb, services.NewTestimonialService(f.crypto), f.rl)
	RegisterCourseHooks(f.pb, plan, f.crypto, f.rl)

	f.create(t, "testimonials", map[string]any{
		"content": "Great to work with", "author_name": "Reviewer", "relationship": "colleague",
		"verification_method": "email", "verification_identifier": "reviewer@example.com",
		"status": "approved", "visibility": "public",
	})
	course := f.create(t, "courses", map[string]any{"title": "Course", "slug": "course", "access_tier": "free", "visibility": "public"})
	f.create(t, "course_certificates", map[string]any{"course": course.Id, "buyer_email": "student@example.com", "certificate_id": "CERT-42", "student_name": "Sam Student", "issued_at": time.Now()})
	h := f.serve(t)

	rec := do(t, h, testRequest{url: "/api/public/testimonials"})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"verification_method":"email"`)
	assertNotContains(t, rec, "reviewer@example.com")

	rec = do(t, h, testRequest{url: "/api/certificates/verify/CERT-42"})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, "Sam Student")
	assertNotContains(t, rec, "student@example.com")
}

const testWebhookSecret = "whsec_test_secret"

func stripeEvent(t *testing.T, eventType, paymentStatus, sessionID, postID string) (string, string) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"id": "evt_" + sessionID + "_" + eventType, "object": "event", "api_version": stripe.APIVersion, "type": eventType,
		"data": map[string]any{"object": map[string]any{
			"id": sessionID, "object": "checkout.session", "payment_status": paymentStatus,
			"amount_total": 500, "currency": "usd", "payment_intent": "pi_" + sessionID,
			"customer_details": map[string]any{"email": "buyer@example.com"},
			"metadata":         map[string]any{"content_type": "posts", "content_id": postID},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{Payload: payload, Secret: testWebhookSecret, Timestamp: time.Now()})
	return string(payload), signed.Header
}

func TestCheckoutGrantsAccessOnlyOncePaid(t *testing.T) {
	t.Setenv("STRIPE_WEBHOOK_SECRET", testWebhookSecret)
	f := newSecurityFixture(t)
	// Paid access links go out by email, so the site needs a mailer.
	settings := f.app.Settings()
	settings.SMTP.Enabled = true
	settings.Meta.AppURL = "https://facet.example"
	settings.Meta.SenderAddress = "site@facet.example"
	if err := f.app.Save(settings); err != nil {
		t.Fatal(err)
	}
	RegisterPurchaseHooks(f.pb, f.crypto, f.rl)
	post := f.create(t, "posts", map[string]any{"title": "Paid", "slug": "paid", "visibility": "public", "access_tier": "paid", "price": 500})
	h := f.serve(t)

	send := func(eventType, paymentStatus, sessionID string) {
		t.Helper()
		body, signature := stripeEvent(t, eventType, paymentStatus, sessionID, post.Id)
		rec := do(t, h, testRequest{method: http.MethodPost, url: "/api/content/webhook", body: body, headers: map[string]string{"Stripe-Signature": signature}})
		assertStatus(t, rec, http.StatusOK)
	}
	purchasesFor := func(sessionID string) int {
		records, _ := f.app.FindRecordsByFilter("purchases", "stripe_checkout_session_id = {:id}", "", 0, 0, map[string]any{"id": sessionID})
		return len(records)
	}

	// A bank debit completes the checkout before the money arrives.
	send("checkout.session.completed", "unpaid", "cs_delayed")
	if purchasesFor("cs_delayed") != 0 {
		t.Fatal("unpaid checkout granted access")
	}
	send("checkout.session.async_payment_succeeded", "paid", "cs_delayed")
	if purchasesFor("cs_delayed") != 1 {
		t.Fatal("settled payment didn't grant access")
	}
	send("checkout.session.async_payment_succeeded", "paid", "cs_delayed")
	if purchasesFor("cs_delayed") != 1 {
		t.Fatal("a repeated event created a second purchase")
	}

	send("checkout.session.completed", "unpaid", "cs_failed")
	send("checkout.session.async_payment_failed", "unpaid", "cs_failed")
	if purchasesFor("cs_failed") != 0 {
		t.Fatal("failed payment granted access")
	}

	// Card payments are paid when the checkout completes.
	send("checkout.session.completed", "paid", "cs_card")
	if purchasesFor("cs_card") != 1 {
		t.Fatal("paid checkout didn't grant access")
	}

	body, _ := stripeEvent(t, "checkout.session.completed", "paid", "cs_forged", post.Id)
	rec := do(t, h, testRequest{method: http.MethodPost, url: "/api/content/webhook", body: body, headers: map[string]string{"Stripe-Signature": "t=1,v1=forged"}})
	assertStatus(t, rec, http.StatusBadRequest)
	if purchasesFor("cs_forged") != 0 {
		t.Fatal("unsigned event granted access")
	}
}
