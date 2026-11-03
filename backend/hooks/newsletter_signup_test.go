package hooks

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase/core"
)

type newsletterFixture struct {
	*securityFixture
	handler http.Handler
}

func newNewsletterFixture(t *testing.T, smtp bool) *newsletterFixture {
	t.Helper()
	f := newSecurityFixture(t)
	settings := f.app.Settings()
	if smtp {
		settings.SMTP.Enabled = true
		settings.SMTP.Host = "smtp.example"
	}
	settings.Meta.AppURL = "https://facet.example"
	settings.Meta.SenderAddress = "site@facet.example"
	if err := f.app.Save(settings); err != nil {
		t.Fatal(err)
	}
	RegisterNewsletterHooks(f.pb, f.crypto, f.rl, &services.PlanConfig{})
	f.create(t, "views", map[string]any{"name": "Home", "slug": "home", "visibility": "public", "is_active": true})
	return &newsletterFixture{securityFixture: f, handler: f.serve(t)}
}

func (n *newsletterFixture) subscribe(t *testing.T, email string) {
	t.Helper()
	rec := do(t, n.handler, testRequest{method: http.MethodPost, url: "/api/public/home/subscribe", body: `{"email":"` + email + `","name":"Reader"}`})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"status":"confirmation_required"`)
}

func (n *newsletterFixture) status(t *testing.T, email string) string {
	t.Helper()
	sub, err := n.app.FindFirstRecordByFilter("subscribers", "email = {:e}", map[string]any{"e": email})
	if err != nil {
		t.Fatalf("subscriber %s: %v", email, err)
	}
	return sub.GetString("status")
}

// assertEmailsSent checks the mail total right after a request. Signup
// emails are sent inside the request, so there is nothing to wait for.
func (n *newsletterFixture) assertEmailsSent(t *testing.T, want int) {
	t.Helper()
	if got := int(n.app.TestMailer.TotalSend()); got != want {
		t.Fatalf("sent %d emails, want %d", got, want)
	}
}

var confirmLink = regexp.MustCompile(`https://facet\.example(/api/public/newsletter/confirm/[A-Za-z0-9_\-%=]+)`)

func (n *newsletterFixture) confirmPath(t *testing.T, email string) string {
	t.Helper()
	msg := n.app.TestMailer.LastMessage()
	if len(msg.To) != 1 || msg.To[0].Address != email {
		t.Fatalf("last email went to %v, want %s", msg.To, email)
	}
	m := confirmLink.FindStringSubmatch(msg.Text)
	if m == nil {
		t.Fatalf("no confirmation link in:\n%s", msg.Text)
	}
	return m[1]
}

func TestNewsletterSignupWaitsForConfirmation(t *testing.T) {
	n := newNewsletterFixture(t, true)

	n.subscribe(t, "reader@example.com")
	if got := n.status(t, "reader@example.com"); got != "pending" {
		t.Fatalf("status before confirming = %q, want pending", got)
	}
	n.assertEmailsSent(t, 1)
	path := n.confirmPath(t, "reader@example.com")

	rec := do(t, n.handler, testRequest{url: path})
	assertStatus(t, rec, http.StatusFound)
	if got := n.status(t, "reader@example.com"); got != "active" {
		t.Fatalf("status after confirming = %q, want active", got)
	}
	n.assertEmailsSent(t, 2) // welcome email

	// The link only works once.
	assertStatus(t, do(t, n.handler, testRequest{url: path}), http.StatusNotFound)

	// Signing up again while subscribed sends nothing.
	sent := n.app.TestMailer.TotalSend()
	n.subscribe(t, "reader@example.com")
	if n.app.TestMailer.TotalSend() != sent {
		t.Fatal("an active subscriber was emailed again")
	}
}

func TestUnsubscribedReaderIsNotResubscribedBySomeoneElse(t *testing.T) {
	n := newNewsletterFixture(t, true)
	n.create(t, "subscribers", map[string]any{"email": "left@example.com", "status": "unsubscribed", "source": "newsletter"})

	n.subscribe(t, "left@example.com")
	if got := n.status(t, "left@example.com"); got != "unsubscribed" {
		t.Fatalf("status = %q; the form alone must not resubscribe someone", got)
	}
	n.assertEmailsSent(t, 1)
	path := n.confirmPath(t, "left@example.com")

	// Repeating the form doesn't flood their inbox.
	for i := 0; i < 3; i++ {
		n.subscribe(t, "left@example.com")
	}
	if got := n.app.TestMailer.TotalSend(); got != 1 {
		t.Fatalf("sent %d confirmation emails, want 1", got)
	}

	// If they follow the link themselves, they're back.
	assertStatus(t, do(t, n.handler, testRequest{url: path}), http.StatusFound)
	if got := n.status(t, "left@example.com"); got != "active" {
		t.Fatalf("status after the owner confirmed = %q", got)
	}
	n.assertEmailsSent(t, 2)
}

func TestSimultaneousFirstSignupsSendOneConfirmation(t *testing.T) {
	n := newNewsletterFixture(t, true)

	// Both requests have looked the address up and found nothing before
	// either one inserts it.
	arrived := make(chan struct{}, 2)
	release := make(chan struct{})
	n.app.OnRecordCreate("subscribers").BindFunc(func(e *core.RecordEvent) error {
		arrived <- struct{}{}
		select {
		case <-release:
		case <-time.After(5 * time.Second):
		}
		return e.Next()
	})

	codes := make(chan *httptest.ResponseRecorder, 2)
	for _, name := range []string{"First", "Second"} {
		go func(name string) {
			codes <- do(t, n.handler, testRequest{method: http.MethodPost, url: "/api/public/home/subscribe", body: `{"email":"race@example.com","name":"` + name + `"}`})
		}(name)
	}
	for i := 0; i < 2; i++ {
		select {
		case <-arrived:
		case <-time.After(5 * time.Second):
			t.Fatal("both signups did not reach the insert")
		}
	}
	close(release)

	for i := 0; i < 2; i++ {
		rec := <-codes
		assertStatus(t, rec, http.StatusOK)
		assertContains(t, rec, `"status":"confirmation_required"`)
	}
	if count, err := n.app.CountRecords("subscribers"); err != nil || count != 1 {
		t.Fatalf("subscribers = %d (%v), want 1", count, err)
	}
	n.assertEmailsSent(t, 1)

	// The confirmation that went out still matches the stored token.
	assertStatus(t, do(t, n.handler, testRequest{url: n.confirmPath(t, "race@example.com")}), http.StatusFound)
	if got := n.status(t, "race@example.com"); got != "active" {
		t.Fatalf("status after confirming = %q, want active", got)
	}
}

func TestNewsletterSignupWithoutEmailStaysPending(t *testing.T) {
	n := newNewsletterFixture(t, false)
	n.subscribe(t, "reader@example.com")
	if got := n.status(t, "reader@example.com"); got != "pending" {
		t.Fatalf("status = %q, want pending", got)
	}
	if n.app.TestMailer.TotalSend() != 0 {
		t.Fatal("email sent without SMTP configured")
	}
}

func TestNewsletterSignupIsRateLimited(t *testing.T) {
	n := newNewsletterFixture(t, false)
	last := 0
	for i := 0; i < 8; i++ {
		last = do(t, n.handler, testRequest{
			method:     http.MethodPost,
			url:        "/api/public/home/subscribe",
			body:       `{"email":"r` + string(rune('a'+i)) + `@example.com"}`,
			remoteAddr: "203.0.113.60:5000",
		}).Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("eighth signup from one address returned %d, want 429", last)
	}
}

func TestNewsletterClickOnlyFollowsLinksInTheNewsletter(t *testing.T) {
	n := newNewsletterFixture(t, true)
	send := n.create(t, "newsletter_sends", map[string]any{
		"subject":   "Hello",
		"body_html": `<p>Read <a href="https://example.org/post?a=1&amp;b=2">this</a></p>`,
		"status":    "sent",
	})
	click := func(sendID, target string) testRequest {
		return testRequest{url: "/api/public/newsletter/click/" + sendID + "/tok/" + base64.URLEncoding.EncodeToString([]byte(target))}
	}

	rec := do(t, n.handler, click(send.Id, "https://example.org/post?a=1&amp;b=2"))
	assertStatus(t, rec, http.StatusFound)
	if loc := rec.Header().Get("Location"); loc != "https://example.org/post?a=1&b=2" {
		t.Fatalf("redirect = %q", loc)
	}

	for name, req := range map[string]testRequest{
		"another site":    click(send.Id, "https://evil.example/login"),
		"unknown send id": click("nosuchsend00000", "https://example.org/post?a=1&amp;b=2"),
	} {
		t.Run(name, func(t *testing.T) {
			rec := do(t, n.handler, req)
			assertStatus(t, rec, http.StatusNotFound)
			if rec.Header().Get("Location") != "" {
				t.Fatal("redirected anyway")
			}
		})
	}
}
