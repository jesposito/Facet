package hooks

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

type audienceFixture struct {
	*newsletterFixture
	adminToken string
	listA      *core.Record
	listB      *core.Record
	mail       chan *mailer.Message
}

// newAudienceFixture builds two lists. List A has an active reader with a
// hostile name, an active reader with a UTF-8 name, plus pending,
// unsubscribed and bounced members. List B has one active reader of its own.
func newAudienceFixture(t *testing.T) *audienceFixture {
	t.Helper()
	// No pause between submissions; pacing has its own test.
	t.Setenv("NEWSLETTER_SEND_INTERVAL_MS", "0")
	n := newNewsletterFixture(t, true)
	_, adminToken := n.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	n.create(t, "profile", map[string]any{"name": "Jo's Site"})

	listA := n.create(t, "newsletter_lists", map[string]any{"name": "Gardeners & Friends", "slug": "gardeners", "is_active": true})
	listB := n.create(t, "newsletter_lists", map[string]any{"name": "Puppeteers", "slug": "puppeteers", "is_active": true})

	member := func(list *core.Record, email, name, status string) {
		sub := n.create(t, "subscribers", map[string]any{"email": email, "name": name, "status": status})
		n.create(t, "subscriber_list_memberships", map[string]any{"subscriber_id": sub.Id, "list_id": list.Id})
	}
	member(listA, "evil@example.com", `<script>alert(1)</script>"Bob"`, "active")
	member(listA, "zoe@example.com", "Zoë Ngāti", "active")
	member(listA, "pending@example.com", "Pending", "pending")
	member(listA, "gone@example.com", "Gone", "unsubscribed")
	member(listA, "bounced@example.com", "Bounced", "bounced")
	member(listB, "puppets@example.com", "Pip", "active")

	f := &audienceFixture{newsletterFixture: n, adminToken: adminToken, listA: listA, listB: listB, mail: make(chan *mailer.Message, 64)}
	n.app.OnMailerSend().BindFunc(func(e *core.MailerEvent) error {
		f.mail <- e.Message
		return e.Next()
	})
	return f
}

func (f *audienceFixture) post(t *testing.T, url, body string) map[string]any {
	t.Helper()
	rec := do(t, f.handler, testRequest{method: http.MethodPost, url: url, body: body, headers: map[string]string{"Authorization": f.adminToken}})
	out := map[string]any{"_status": float64(rec.Code)}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	out["_status"] = float64(rec.Code)
	return out
}

func (f *audienceFixture) waitForSends(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if n, _ := f.app.CountRecords("newsletter_sends", dbx.NewExp("status = 'sending'")); n == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("send never finished")
}

func (f *audienceFixture) drain() map[string]*mailer.Message {
	got := map[string]*mailer.Message{}
	for {
		select {
		case m := <-f.mail:
			got[m.To[0].Address] = m
		default:
			return got
		}
	}
}

func keys(m map[string]*mailer.Message) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestNewsletterSendsOnlyToSelectedListWithPersonalization(t *testing.T) {
	f := newAudienceFixture(t)
	body := `{"subject":"Hi {{ subscriber.name }} from {{site.name}}","body_markdown":"Dear {{ subscriber.name }} ({{ subscriber.email }}), welcome to {{ list.name }} at {{ site.url }}. Unknown {{ other.thing }} stays.","list_id":"` + f.listA.Id + `","personalize":true}`

	preview := f.post(t, "/api/admin/newsletter/preview", body)
	if preview["_status"] != float64(200) || preview["recipient_count"] != float64(2) {
		t.Fatalf("preview = %v, want 2 recipients", preview)
	}

	res := f.post(t, "/api/admin/newsletter/send", body)
	if res["_status"] != float64(200) || res["recipient_count"] != float64(2) {
		t.Fatalf("send = %v", res)
	}
	f.waitForSends(t)
	got := f.drain()

	if want := []string{"evil@example.com", "zoe@example.com"}; strings.Join(keys(got), ",") != strings.Join(want, ",") {
		t.Fatalf("mail went to %v, want %v", keys(got), want)
	}

	zoe := got["zoe@example.com"]
	if zoe.Subject != "Hi Zoë Ngāti from Jo's Site" {
		t.Fatalf("subject = %q", zoe.Subject)
	}
	for _, want := range []string{
		"Dear Zoë Ngāti (zoe@example.com)",
		"welcome to Gardeners &amp; Friends at https://facet.example.",
		"Unknown {{ other.thing }} stays.",
		"https://facet.example/unsubscribe/",
	} {
		if !strings.Contains(zoe.HTML, want) {
			t.Fatalf("zoe's email is missing %q:\n%s", want, zoe.HTML)
		}
	}

	evil := got["evil@example.com"]
	if strings.Contains(evil.HTML, "<script>") {
		t.Fatal("subscriber name injected raw HTML")
	}
	if !strings.Contains(evil.HTML, "Dear &lt;script&gt;alert(1)&lt;/script&gt;&#34;Bob&#34; (evil@example.com)") {
		t.Fatalf("escaped name missing:\n%s", evil.HTML)
	}
	if strings.Contains(evil.HTML, "zoe@example.com") || strings.Contains(zoe.HTML, "evil@example.com") {
		t.Fatal("one recipient's details leaked into another's email")
	}
	if !strings.Contains(evil.Headers["List-Unsubscribe"], "https://facet.example/api/public/unsubscribe/one-click/") {
		t.Fatalf("List-Unsubscribe = %q", evil.Headers["List-Unsubscribe"])
	}

	// List B gets its own reader only.
	res = f.post(t, "/api/admin/newsletter/send", `{"subject":"Puppets","body_markdown":"Strings","list_id":"puppeteers"}`)
	if res["recipient_count"] != float64(1) {
		t.Fatalf("list B send = %v", res)
	}
	f.waitForSends(t)
	if got := keys(f.drain()); strings.Join(got, ",") != "puppets@example.com" {
		t.Fatalf("list B mail went to %v", got)
	}
}

func TestNewsletterUnknownListNeverFallsBackToEveryone(t *testing.T) {
	f := newAudienceFixture(t)
	for _, url := range []string{"/api/admin/newsletter/send", "/api/admin/newsletter/preview"} {
		res := f.post(t, url, `{"subject":"Hi","body_markdown":"Hello","list_id":"no-such-list"}`)
		if res["_status"] != float64(400) {
			t.Fatalf("%s with an unknown list = %v, want 400", url, res)
		}
	}
	time.Sleep(100 * time.Millisecond)
	if n, _ := f.app.CountRecords("newsletter_sends"); n != 0 {
		t.Fatalf("%d send records created for an unknown list", n)
	}
	if got := f.drain(); len(got) != 0 {
		t.Fatalf("mail sent for an unknown list: %v", keys(got))
	}
}

func TestNewsletterRetryDoesNotRepeatACompletedSend(t *testing.T) {
	f := newAudienceFixture(t)
	body := `{"subject":"Weekly","body_markdown":"News","list_id":"puppeteers"}`

	first := f.post(t, "/api/admin/newsletter/send", body)
	if first["_status"] != float64(200) {
		t.Fatalf("first send = %v", first)
	}
	f.waitForSends(t)
	if got := len(f.drain()); got != 1 {
		t.Fatalf("first send delivered %d emails", got)
	}

	// The same request again is a retry, not a second newsletter.
	retry := f.post(t, "/api/admin/newsletter/send", body)
	if retry["_status"] != float64(409) || retry["code"] != "already_sent" || retry["send_id"] != first["send_id"] {
		t.Fatalf("retry = %v", retry)
	}
	time.Sleep(200 * time.Millisecond)
	if got := len(f.drain()); got != 0 {
		t.Fatalf("retry delivered %d emails", got)
	}

	// Sending it again is an explicit choice.
	again := f.post(t, "/api/admin/newsletter/send", strings.TrimSuffix(body, "}")+`,"resend":true}`)
	if again["_status"] != float64(200) {
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
