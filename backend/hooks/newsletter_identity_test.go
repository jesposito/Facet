package hooks

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// ── Send identity and idempotency ─────────────────────────────────────

func TestNewsletterRequestKeyRetryReturnsTheOriginalSend(t *testing.T) {
	f := newAudienceFixture(t)
	body := `{"subject":"Weekly","body_markdown":"News","list_id":"puppeteers","request_key":"key-weekly-0001"}`

	first := f.post(t, "/api/admin/newsletter/send", body)
	if first["_status"] != float64(200) {
		t.Fatalf("first send = %v", first)
	}
	f.waitForSends(t)
	if got := len(f.drain()); got != 1 {
		t.Fatalf("first send delivered %d emails", got)
	}

	retry := f.post(t, "/api/admin/newsletter/send", body)
	if retry["_status"] != float64(200) || retry["replayed"] != true || retry["send_id"] != first["send_id"] {
		t.Fatalf("retry with the same key = %v", retry)
	}
	time.Sleep(200 * time.Millisecond)
	if got := len(f.drain()); got != 0 {
		t.Fatalf("retry delivered %d emails", got)
	}

	// The same key for different content is a client bug, not a retry.
	reused := f.post(t, "/api/admin/newsletter/send", `{"subject":"Other","body_markdown":"News","list_id":"puppeteers","request_key":"key-weekly-0001"}`)
	if reused["_status"] != float64(409) || reused["code"] != "request_key_reused" {
		t.Fatalf("key reused for other content = %v", reused)
	}

	// Without resend, a new key for the same newsletter and list is refused.
	dup := f.post(t, "/api/admin/newsletter/send", strings.Replace(body, "key-weekly-0001", "key-weekly-0002", 1))
	if dup["_status"] != float64(409) || dup["code"] != "already_sent" {
		t.Fatalf("new key, same newsletter = %v", dup)
	}

	// An explicit resend uses a new key and goes out.
	again := f.post(t, "/api/admin/newsletter/send", `{"subject":"Weekly","body_markdown":"News","list_id":"puppeteers","request_key":"key-weekly-0003","resend":true}`)
	if again["_status"] != float64(200) || again["replayed"] == true {
		t.Fatalf("explicit resend = %v", again)
	}
	f.waitForSends(t)
	if got := len(f.drain()); got != 1 {
		t.Fatalf("explicit resend delivered %d emails", got)
	}
	if n, _ := f.app.CountRecords("newsletter_sends"); n != 2 {
		t.Fatalf("%d send records, want 2", n)
	}
}

func TestNewsletterSameContentToAnotherListIsNotADuplicate(t *testing.T) {
	f := newAudienceFixture(t)
	toB := f.post(t, "/api/admin/newsletter/send", `{"subject":"Season","body_markdown":"Seeds","list_id":"puppeteers","request_key":"season-to-b-01"}`)
	if toB["_status"] != float64(200) {
		t.Fatalf("send to list B = %v", toB)
	}
	f.waitForSends(t)
	f.drain()

	toA := f.post(t, "/api/admin/newsletter/send", `{"subject":"Season","body_markdown":"Seeds","list_id":"`+f.listA.Id+`","request_key":"season-to-a-01"}`)
	if toA["_status"] != float64(200) || toA["recipient_count"] != float64(2) {
		t.Fatalf("same newsletter to list A = %v", toA)
	}
	f.waitForSends(t)
	if got := keys(f.drain()); strings.Join(got, ",") != "evil@example.com,zoe@example.com" {
		t.Fatalf("list A send reached %v", got)
	}

	sendA, err := f.app.FindRecordById("newsletter_sends", toA["send_id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if sendA.GetString("list_id") != f.listA.Id || sendA.GetString("audience_key") != "list:"+f.listA.Id ||
		sendA.GetString("request_key") != "season-to-a-01" || sendA.GetString("content_hash") == "" ||
		sendA.GetString("active_lock") != "" {
		t.Fatalf("send identity not recorded: list=%q audience=%q key=%q hash=%q lock=%q",
			sendA.GetString("list_id"), sendA.GetString("audience_key"), sendA.GetString("request_key"),
			sendA.GetString("content_hash"), sendA.GetString("active_lock"))
	}
}

func TestNewsletterLegacySendStillBlocksAnAccidentalRepeat(t *testing.T) {
	f := newAudienceFixture(t)
	bodyHTML, err := markdownToHTML("Old news")
	if err != nil {
		t.Fatal(err)
	}
	// Recorded before sends had an audience: blank audience_key and hash.
	legacy := f.create(t, "newsletter_sends", map[string]any{
		"subject": "Old", "body_html": bodyHTML, "status": "sent", "sent_count": 3,
	})

	res := f.post(t, "/api/admin/newsletter/send", `{"subject":"Old","body_markdown":"Old news","list_id":"puppeteers","request_key":"legacy-repeat-1"}`)
	if res["_status"] != float64(409) || res["code"] != "already_sent" || res["send_id"] != legacy.Id {
		t.Fatalf("repeat of a legacy send = %v", res)
	}
	res = f.post(t, "/api/admin/newsletter/send", `{"subject":"Old","body_markdown":"Old news","list_id":"puppeteers","request_key":"legacy-repeat-2","resend":true}`)
	if res["_status"] != float64(200) {
		t.Fatalf("explicit resend of a legacy send = %v", res)
	}
	f.waitForSends(t)
}

func TestNewsletterSameRequestKeyUnderLoadStartsOneSend(t *testing.T) {
	f := newAudienceFixture(t)
	release := make(chan struct{})
	f.app.OnMailerSend().BindFunc(func(e *core.MailerEvent) error {
		<-release
		return e.Next()
	})

	results := concurrently(t, f.handler, testRequest{
		method:  http.MethodPost,
		url:     "/api/admin/newsletter/send",
		body:    `{"subject":"Burst","body_markdown":"Once","list_id":"puppeteers","request_key":"burst-key-0001"}`,
		headers: map[string]string{"Authorization": f.adminToken},
	})
	close(release)
	f.waitForSends(t)

	sendIDs := map[string]bool{}
	for _, rec := range results {
		if rec.Code != http.StatusOK {
			t.Fatalf("a retry of the same request returned %d: %s", rec.Code, rec.Body.String())
		}
		var out struct {
			SendID string `json:"send_id"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		sendIDs[out.SendID] = true
	}
	if len(sendIDs) != 1 {
		t.Fatalf("parallel retries reported %d different sends", len(sendIDs))
	}
	if n, _ := f.app.CountRecords("newsletter_sends"); n != 1 {
		t.Fatalf("%d send records for one request key", n)
	}
	if got := len(f.drain()); got != 1 {
		t.Fatalf("%d emails for one request", got)
	}
}

func TestNewsletterDatabaseAllowsOneActiveSendAndOneUseOfAKey(t *testing.T) {
	f := newAudienceFixture(t)
	f.create(t, "newsletter_sends", map[string]any{"subject": "A", "status": "sending", "active_lock": newsletterActiveLock, "request_key": "db-key-000001"})

	col, err := f.app.FindCollectionByNameOrId("newsletter_sends")
	if err != nil {
		t.Fatal(err)
	}
	second := core.NewRecord(col)
	second.Load(map[string]any{"subject": "B", "status": "sending", "active_lock": newsletterActiveLock})
	if err := f.app.Save(second); err == nil {
		t.Fatal("the database accepted a second active send")
	}
	reuse := core.NewRecord(col)
	reuse.Load(map[string]any{"subject": "C", "status": "sent", "request_key": "db-key-000001"})
	if err := f.app.Save(reuse); err == nil {
		t.Fatal("the database accepted a request key twice")
	}
	// Blank keys and finished sends don't collide.
	for i := 0; i < 2; i++ {
		f.create(t, "newsletter_sends", map[string]any{"subject": "Old", "status": "sent"})
	}

	res := f.post(t, "/api/admin/newsletter/send", `{"subject":"New","body_markdown":"Body","list_id":"puppeteers"}`)
	if res["_status"] != float64(409) || res["code"] != "send_in_progress" {
		t.Fatalf("send while another holds the lock = %v", res)
	}
}

// ── Signup joins the intended list ────────────────────────────────────

func signupListFixture(t *testing.T) (*newsletterFixture, *core.Record, *core.Record, *core.Record) {
	t.Helper()
	n := newNewsletterFixture(t, true)
	def, err := n.app.FindFirstRecordByFilter("newsletter_lists", "is_default = true")
	if err != nil {
		t.Fatalf("default list: %v", err)
	}
	public := n.create(t, "newsletter_lists", map[string]any{"name": "Greenhouse", "slug": "greenhouse", "is_active": true, "is_public": true})
	private := n.create(t, "newsletter_lists", map[string]any{"name": "Family", "slug": "family", "is_active": true, "is_public": false})
	n.create(t, "newsletter_lists", map[string]any{"name": "Closed", "slug": "closed", "is_active": false, "is_public": true})
	return n, def, public, private
}

func (n *newsletterFixture) memberOf(t *testing.T, email string) []string {
	t.Helper()
	sub, err := n.app.FindFirstRecordByFilter("subscribers", "email = {:e}", dbx.Params{"e": email})
	if err != nil {
		t.Fatalf("subscriber %s: %v", email, err)
	}
	var rows []struct {
		Slug string `db:"slug"`
	}
	if err := n.app.DB().NewQuery(`SELECT l.slug AS slug FROM subscriber_list_memberships m
		INNER JOIN newsletter_lists l ON l.id = m.list_id
		WHERE m.subscriber_id = {:s} ORDER BY l.slug`).Bind(dbx.Params{"s": sub.Id}).All(&rows); err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Slug)
	}
	return out
}

func (n *newsletterFixture) subscribeTo(t *testing.T, email, list string) int {
	t.Helper()
	return do(t, n.handler, testRequest{method: http.MethodPost, url: "/api/public/home/subscribe", body: `{"email":"` + email + `","list":"` + list + `"}`}).Code
}

func TestNewsletterSignupJoinsTheDefaultListAfterConfirming(t *testing.T) {
	n, _, _, _ := signupListFixture(t)
	n.subscribe(t, "reader@example.com")
	if got := n.memberOf(t, "reader@example.com"); len(got) != 0 {
		t.Fatalf("joined %v before confirming", got)
	}
	n.assertEmailsSent(t, 1)
	assertStatus(t, do(t, n.handler, testRequest{url: n.confirmPath(t, "reader@example.com")}), http.StatusFound)
	if got := strings.Join(n.memberOf(t, "reader@example.com"), ","); got != "default" {
		t.Fatalf("member of %q after confirming, want default", got)
	}
}

func TestNewsletterSignupJoinsTheNamedPublicList(t *testing.T) {
	n, _, public, _ := signupListFixture(t)
	if code := n.subscribeTo(t, "grower@example.com", public.GetString("slug")); code != http.StatusOK {
		t.Fatalf("signup to a public list returned %d", code)
	}
	n.assertEmailsSent(t, 1)
	assertStatus(t, do(t, n.handler, testRequest{url: n.confirmPath(t, "grower@example.com")}), http.StatusFound)
	if got := strings.Join(n.memberOf(t, "grower@example.com"), ","); got != "greenhouse" {
		t.Fatalf("member of %q, want greenhouse only", got)
	}
	n.assertEmailsSent(t, 2) // welcome

	// Already on the default list? Joining another list still asks first.
	n.subscribe(t, "both@example.com")
	n.assertEmailsSent(t, 3)
	assertStatus(t, do(t, n.handler, testRequest{url: n.confirmPath(t, "both@example.com")}), http.StatusFound)
	n.assertEmailsSent(t, 4)
	before := n.app.TestMailer.TotalSend()
	sub, _ := n.app.FindFirstRecordByFilter("subscribers", "email = 'both@example.com'")
	sub.Set("confirmation_sent_at", "") // past the cooldown
	if err := n.app.Save(sub); err != nil {
		t.Fatal(err)
	}
	if code := n.subscribeTo(t, "both@example.com", public.Id); code != http.StatusOK {
		t.Fatalf("active reader joining a second list returned %d", code)
	}
	if got := strings.Join(n.memberOf(t, "both@example.com"), ","); got != "default" {
		t.Fatalf("joined %q before confirming the second list", got)
	}
	n.assertEmailsSent(t, int(before)+1)
	assertStatus(t, do(t, n.handler, testRequest{url: n.confirmPath(t, "both@example.com")}), http.StatusFound)
	if got := strings.Join(n.memberOf(t, "both@example.com"), ","); got != "default,greenhouse" {
		t.Fatalf("member of %q, want default,greenhouse", got)
	}
}

func TestNewsletterSignupRefusesListsThatArentOpen(t *testing.T) {
	n, _, _, private := signupListFixture(t)
	for _, list := range []string{private.GetString("slug"), private.Id, "closed", "no-such-list"} {
		rec := do(t, n.handler, testRequest{method: http.MethodPost, url: "/api/public/home/subscribe", body: `{"email":"x@example.com","list":"` + list + `"}`})
		assertStatus(t, rec, http.StatusBadRequest)
		assertContains(t, rec, "list_unavailable")
	}
	if _, err := n.app.FindFirstRecordByFilter("subscribers", "email = 'x@example.com'"); err == nil {
		t.Fatal("a refused signup still stored the address")
	}
	if n.app.TestMailer.TotalSend() != 0 {
		t.Fatal("a refused signup sent mail")
	}

	// The public catalogue lists only what the form accepts.
	rec := do(t, n.handler, testRequest{url: "/api/public/newsletter/lists"})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"slug":"default"`)
	assertContains(t, rec, `"slug":"greenhouse"`)
	assertNotContains(t, rec, "family")
	assertNotContains(t, rec, "closed")
	assertNotContains(t, rec, "subscriber_count")
}

func TestNewsletterConfirmRefusesAListClosedAfterSignup(t *testing.T) {
	n, _, public, _ := signupListFixture(t)
	n.subscribeTo(t, "late@example.com", "greenhouse")
	n.assertEmailsSent(t, 1)
	path := n.confirmPath(t, "late@example.com")
	public.Set("is_active", false)
	if err := n.app.Save(public); err != nil {
		t.Fatal(err)
	}
	assertStatus(t, do(t, n.handler, testRequest{url: path}), http.StatusNotFound)
	if got := n.status(t, "late@example.com"); got != "pending" {
		t.Fatalf("status = %q after confirming a closed list", got)
	}
}

// ── Links and placeholders ────────────────────────────────────────────

func TestNewsletterRefusesPerReaderPlaceholdersInLinks(t *testing.T) {
	f := newAudienceFixture(t)
	for _, md := range []string{
		// Markdown only makes a link when the address has no spaces.
		"[profile](https://shop.example/?u={{subscriber.email}})",
		"[hi](https://shop.example/{{subscriber.name}})",
	} {
		body, _ := json.Marshal(map[string]any{"subject": "Hi", "body_markdown": md, "list_id": "puppeteers", "personalize": true})
		for _, url := range []string{"/api/admin/newsletter/preview", "/api/admin/newsletter/send"} {
			res := f.post(t, url, string(body))
			if res["_status"] != float64(400) || res["code"] != "per_recipient_link" {
				t.Fatalf("%s with %q = %v", url, md, res)
			}
		}
	}
	// Without personalization a placeholder link would go out broken.
	body, _ := json.Marshal(map[string]any{"subject": "Hi", "body_markdown": "[home]({{site.url}}/blog)", "list_id": "puppeteers"})
	if res := f.post(t, "/api/admin/newsletter/send", string(body)); res["_status"] != float64(400) || res["code"] != "link_placeholder" {
		t.Fatalf("placeholder link without personalization = %v", res)
	}
	time.Sleep(100 * time.Millisecond)
	if n, _ := f.app.CountRecords("newsletter_sends"); n != 0 {
		t.Fatalf("%d sends created for refused links", n)
	}
	if got := f.drain(); len(got) != 0 {
		t.Fatalf("mail sent for refused links: %v", keys(got))
	}
}

var trackedLink = regexp.MustCompile(`https://facet\.example(/api/public/newsletter/click/[^"]+)`)

func TestNewsletterFillsSharedPlaceholdersInLinksBeforeTracking(t *testing.T) {
	f := newAudienceFixture(t)
	body, _ := json.Marshal(map[string]any{
		"subject":       "Hi {{ subscriber.name }}",
		"body_markdown": "Hello {{ subscriber.name }}, [read the blog]({{site.url}}/lists/{{list.name}}?from=mail)",
		"list_id":       "puppeteers",
		"personalize":   true,
	})
	res := f.post(t, "/api/admin/newsletter/send", string(body))
	if res["_status"] != float64(200) {
		t.Fatalf("send = %v", res)
	}
	f.waitForSends(t)
	msg := f.drain()["puppets@example.com"]
	if msg == nil {
		t.Fatal("no email for the list B reader")
	}
	if !strings.Contains(msg.HTML, "Hello Pip") {
		t.Fatalf("body text not personalized:\n%s", msg.HTML)
	}
	m := trackedLink.FindStringSubmatch(msg.HTML)
	if m == nil {
		t.Fatalf("no tracked link in:\n%s", msg.HTML)
	}
	parts := strings.Split(m[1], "/")
	target, err := base64.URLEncoding.DecodeString(parts[len(parts)-1])
	if err != nil {
		t.Fatal(err)
	}
	if want := "https://facet.example/lists/Puppeteers?from=mail"; string(target) != want {
		t.Fatalf("tracked link points at %q, want %q", target, want)
	}
	rec := do(t, f.handler, testRequest{url: m[1]})
	assertStatus(t, rec, http.StatusFound)
	if loc := rec.Header().Get("Location"); loc != "https://facet.example/lists/Puppeteers?from=mail" {
		t.Fatalf("click redirected to %q", loc)
	}
}

func TestPrepareNewsletterLinksEscapesSharedValues(t *testing.T) {
	p := &newsletterPersonalization{Enabled: true, ListName: `A&B "x"/y`, SiteName: "Jo", SiteURL: "https://facet.example"}
	out, err := prepareNewsletterLinks(`<a href="https://x.example/{{ list.name }}">x</a> <a href="https://plain.example/?a=1&amp;b=2">p</a>`, p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `href="https://x.example/A&amp;B%20%22x%22%2Fy"`) {
		t.Fatalf("list name not path-escaped: %s", out)
	}
	if !strings.Contains(out, `href="https://plain.example/?a=1&amp;b=2"`) {
		t.Fatalf("a link without placeholders changed: %s", out)
	}
}
