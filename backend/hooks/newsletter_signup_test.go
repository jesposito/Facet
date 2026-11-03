package hooks

import (
	"encoding/base64"
	"net/http"
	"regexp"
	"testing"
	"time"

	"facet/services"
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

func (n *newsletterFixture) waitForEmails(t *testing.T, want int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if int(n.app.TestMailer.TotalSend()) >= want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("sent %d emails, want %d", n.app.TestMailer.TotalSend(), want)
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
	n.waitForEmails(t, 1)
	path := n.confirmPath(t, "reader@example.com")

	rec := do(t, n.handler, testRequest{url: path})
	assertStatus(t, rec, http.StatusFound)
	if got := n.status(t, "reader@example.com"); got != "active" {
		t.Fatalf("status after confirming = %q, want active", got)
	}
	n.waitForEmails(t, 2) // welcome email

	// The link only works once.
	assertStatus(t, do(t, n.handler, testRequest{url: path}), http.StatusNotFound)

	// Signing up again while subscribed sends nothing.
	sent := n.app.TestMailer.TotalSend()
	n.subscribe(t, "reader@example.com")
	time.Sleep(50 * time.Millisecond)
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
	n.waitForEmails(t, 1)
	path := n.confirmPath(t, "left@example.com")

	// Repeating the form doesn't flood their inbox.
	for i := 0; i < 3; i++ {
		n.subscribe(t, "left@example.com")
	}
	time.Sleep(50 * time.Millisecond)
	if got := n.app.TestMailer.TotalSend(); got != 1 {
		t.Fatalf("sent %d confirmation emails, want 1", got)
	}

	// If they follow the link themselves, they're back.
	assertStatus(t, do(t, n.handler, testRequest{url: path}), http.StatusFound)
	if got := n.status(t, "left@example.com"); got != "active" {
		t.Fatalf("status after the owner confirmed = %q", got)
	}
	n.waitForEmails(t, 2)
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
