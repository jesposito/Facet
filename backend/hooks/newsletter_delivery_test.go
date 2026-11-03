package hooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestNewsletterResumeSkipsAcceptedAndUncertainRecipients(t *testing.T) {
	f := newAudienceFixture(t)
	send := f.create(t, "newsletter_sends", map[string]any{
		"subject": "Resume", "body_html": "<p>Hello</p>", "body_markdown": "Hello",
		"status": "interrupted", "recipient_count": 3, "list_id": f.listA.Id,
	})
	accepted := f.create(t, "newsletter_recipients", map[string]any{"send_id": send.Id, "subscriber_id": f.subscriberID(t, "evil@example.com"), "state": "accepted"})
	pending := f.create(t, "newsletter_recipients", map[string]any{"send_id": send.Id, "subscriber_id": f.subscriberID(t, "zoe@example.com"), "state": "pending"})
	uncertain := f.create(t, "newsletter_recipients", map[string]any{"send_id": send.Id, "subscriber_id": f.subscriberID(t, "puppets@example.com"), "state": "uncertain"})
	_ = accepted

	if err := resumeNewsletter(f.pb, f.crypto, send.Id); err != nil {
		t.Fatal(err)
	}
	f.waitForSends(t)
	if got := strings.Join(keys(f.drain()), ","); got != "zoe@example.com" {
		t.Fatalf("resume mailed %v, want only pending recipient", got)
	}
	for id, state := range map[string]string{accepted.Id: "accepted", pending.Id: "accepted", uncertain.Id: "uncertain"} {
		r, err := f.app.FindRecordById("newsletter_recipients", id)
		if err != nil || r.GetString("state") != state {
			t.Fatalf("recipient %s state = %q, %v; want %s", id, r.GetString("state"), err, state)
		}
	}
}

func TestNewsletterConcurrentRecipientClaimHasOneWinner(t *testing.T) {
	f := newAudienceFixture(t)
	send := f.create(t, "newsletter_sends", map[string]any{"subject": "Claim", "status": "sending"})
	recipient := f.create(t, "newsletter_recipients", map[string]any{"send_id": send.Id, "subscriber_id": f.subscriberID(t, "zoe@example.com"), "state": "pending"})
	var winners atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, err := claimNewsletterRecipient(f.app, recipient.Id)
			if err != nil {
				t.Error(err)
			}
			if claimed {
				winners.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := winners.Load(); got != 1 {
		t.Fatalf("claim winners = %d, want 1", got)
	}
}

func TestNewsletterSMTPOutagesPreserveSubscriberConsent(t *testing.T) {
	f := newAudienceFixture(t)
	f.app.OnMailerSend().BindFunc(func(e *core.MailerEvent) error {
		return errors.New("535 authentication failed")
	})
	subID := f.subscriberID(t, "puppets@example.com")
	for i := 0; i < 4; i++ {
		res := f.post(t, "/api/admin/newsletter/send", fmt.Sprintf(`{"subject":"Outage %d","body_markdown":"Hello","list_id":"puppeteers","resend":true}`, i))
		if res["_status"] != float64(http.StatusOK) {
			t.Fatalf("send %d = %v", i, res)
		}
		f.waitForSends(t)
	}
	sub, err := f.app.FindRecordById("subscribers", subID)
	if err != nil {
		t.Fatal(err)
	}
	if sub.GetString("status") != "active" || sub.GetInt("bounce_count") != 0 {
		t.Fatalf("SMTP errors changed subscriber: status=%q bounce_count=%d", sub.GetString("status"), sub.GetInt("bounce_count"))
	}
}

func TestNewsletterStartupMarksOnlyInflightUncertain(t *testing.T) {
	f := newAudienceFixture(t)
	send := f.create(t, "newsletter_sends", map[string]any{"subject": "Restart", "status": "sending", "active_lock": newsletterActiveLock})
	inflight := f.create(t, "newsletter_recipients", map[string]any{"send_id": send.Id, "subscriber_id": f.subscriberID(t, "evil@example.com"), "state": "in-flight"})
	pending := f.create(t, "newsletter_recipients", map[string]any{"send_id": send.Id, "subscriber_id": f.subscriberID(t, "zoe@example.com"), "state": "pending"})
	if err := recoverInterruptedNewsletters(f.app); err != nil {
		t.Fatal(err)
	}
	for id, state := range map[string]string{inflight.Id: "uncertain", pending.Id: "pending"} {
		r, _ := f.app.FindRecordById("newsletter_recipients", id)
		if got := r.GetString("state"); got != state {
			t.Fatalf("state = %q, want %q", got, state)
		}
	}
	if got := f.app.TestMailer.TotalSend(); got != 0 {
		t.Fatalf("startup sent %d emails", got)
	}
}

func TestNewsletterStartupKeepsLegacySendHistory(t *testing.T) {
	f := newAudienceFixture(t)
	bodyHTML, err := markdownToHTML("Old news")
	if err != nil {
		t.Fatal(err)
	}
	// Recorded before the recipient ledger: counters only, no audience.
	legacy := f.create(t, "newsletter_sends", map[string]any{
		"subject": "Old", "body_html": bodyHTML, "status": "sending", "active_lock": newsletterActiveLock,
		"recipient_count": 5, "sent_count": 3, "failed_count": 2,
	})
	for i := 0; i < 2; i++ {
		if err := recoverInterruptedNewsletters(f.app); err != nil {
			t.Fatalf("recovery run %d: %v", i+1, err)
		}
		r, err := f.app.FindRecordById("newsletter_sends", legacy.Id)
		if err != nil {
			t.Fatal(err)
		}
		if r.GetString("status") != "interrupted" || r.GetString("active_lock") != "" ||
			r.GetInt("sent_count") != 3 || r.GetInt("failed_count") != 2 || r.GetInt("recipient_count") != 5 {
			t.Fatalf("run %d: status=%q lock=%q sent=%d failed=%d recipients=%d", i+1, r.GetString("status"),
				r.GetString("active_lock"), r.GetInt("sent_count"), r.GetInt("failed_count"), r.GetInt("recipient_count"))
		}
	}
	if n, _ := f.app.CountRecords("newsletter_recipients", dbx.HashExp{"send_id": legacy.Id}); n != 0 {
		t.Fatalf("recovery invented %d recipient outcomes", n)
	}
	if got := f.app.TestMailer.TotalSend(); got != 0 {
		t.Fatalf("startup sent %d emails", got)
	}

	res := f.post(t, "/api/admin/newsletter/send", `{"subject":"Old","body_markdown":"Old news","list_id":"puppeteers","request_key":"legacy-restart-1"}`)
	if res["_status"] != float64(http.StatusConflict) || res["code"] != "already_sent" || res["send_id"] != legacy.Id {
		t.Fatalf("repeat after recovering a legacy send = %v", res)
	}
	f.assertResumable(t, legacy.Id, 0)
}

func TestNewsletterLegacyFailedSendCannotResume(t *testing.T) {
	f := newAudienceFixture(t)
	legacy := f.create(t, "newsletter_sends", map[string]any{
		"subject": "Old failure", "body_html": "<p>Hi</p>", "status": "failed", "recipient_count": 2, "failed_count": 2,
	})
	f.assertResumable(t, legacy.Id, 0)
	res := f.post(t, "/api/admin/newsletter/sends/"+legacy.Id+"/resume", `{}`)
	if res["_status"] != float64(http.StatusBadRequest) {
		t.Fatalf("resume of a send without a ledger = %v", res)
	}
	if r, _ := f.app.FindRecordById("newsletter_sends", legacy.Id); r.GetString("status") != "failed" || r.GetString("active_lock") != "" {
		t.Fatalf("refused resume changed the send: status=%q lock=%q", r.GetString("status"), r.GetString("active_lock"))
	}
	if got := len(f.drain()); got != 0 {
		t.Fatalf("refused resume sent %d emails", got)
	}
}

func TestNewsletterResumableCountIsPendingAndFailedLedgerRows(t *testing.T) {
	f := newAudienceFixture(t)
	send := f.create(t, "newsletter_sends", map[string]any{"subject": "Ledger", "status": "interrupted", "recipient_count": 5})
	other := f.create(t, "newsletter_sends", map[string]any{"subject": "Done", "status": "sent", "recipient_count": 1, "sent_count": 1})
	for email, state := range map[string]string{
		"evil@example.com": "pending", "zoe@example.com": "failed", "pending@example.com": "failed",
		"gone@example.com": "accepted", "bounced@example.com": "uncertain", "puppets@example.com": "skipped",
	} {
		f.create(t, "newsletter_recipients", map[string]any{"send_id": send.Id, "subscriber_id": f.subscriberID(t, email), "state": state})
	}
	f.create(t, "newsletter_recipients", map[string]any{"send_id": other.Id, "subscriber_id": f.subscriberID(t, "zoe@example.com"), "state": "accepted"})

	// The list endpoint counts every listed send in one call like this.
	counts, err := newsletterResumableCounts(f.app, []string{send.Id, other.Id})
	if err != nil {
		t.Fatal(err)
	}
	if len(counts) != 1 || counts[send.Id] != 3 {
		t.Fatalf("resumable counts = %v, want only %s: 3", counts, send.Id)
	}
	f.assertResumable(t, send.Id, 3)
	f.assertResumable(t, other.Id, 0)
}

func TestNewsletterByEmailDoesNotMutateSubscription(t *testing.T) {
	n := newNewsletterFixture(t, true)
	n.create(t, "subscribers", map[string]any{"email": "reader@example.com", "status": "active"})
	rec := do(t, n.handler, testRequest{method: http.MethodPost, url: "/api/public/unsubscribe/by-email", body: `{"email":"reader@example.com"}`})
	assertStatus(t, rec, http.StatusOK)
	if got := n.status(t, "reader@example.com"); got != "active" {
		t.Fatalf("email-only request changed status to %q", got)
	}
}

func TestNewsletterFailedConfirmationAllowsBoundedRetry(t *testing.T) {
	n := newNewsletterFixture(t, true)
	n.app.OnMailerSend().BindFunc(func(e *core.MailerEvent) error {
		return errors.New("temporary SMTP outage")
	})
	n.subscribe(t, "retry@example.com")
	sub, _ := n.app.FindFirstRecordByFilter("subscribers", "email = {:email}", dbx.Params{"email": "retry@example.com"})
	if got := sub.GetString("confirmation_sent_at"); got != "" {
		t.Fatalf("failed submission recorded accepted timestamp %q", got)
	}
	n.subscribe(t, "retry@example.com")
	if got := n.app.TestMailer.TotalSend(); got != 0 {
		t.Fatalf("immediate retry escaped cooldown, sent %d", got)
	}
	sub.Set("confirmation_attempted_at", time.Now().Add(-2*subscribeConfirmationAttemptCooldown).UTC().Format(time.RFC3339))
	if err := n.app.Save(sub); err != nil {
		t.Fatal(err)
	}
	firstHash := sub.GetString("confirm_token_hash")
	n.subscribe(t, "retry@example.com")
	sub, _ = n.app.FindRecordById("subscribers", sub.Id)
	if sub.GetString("confirm_token_hash") == "" || sub.GetString("confirm_token_hash") == firstHash {
		t.Fatal("bounded retry after a failed confirmation did not issue a new token")
	}
	if got := sub.GetString("confirmation_sent_at"); got != "" {
		t.Fatalf("failed retry recorded accepted timestamp %q", got)
	}
}

func TestNewsletterConcurrentConfirmationRetriesClaimOnce(t *testing.T) {
	n := newNewsletterFixture(t, true)
	n.create(t, "subscribers", map[string]any{"email": "race@example.com", "status": "pending"})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := do(t, n.handler, testRequest{method: http.MethodPost, url: "/api/public/home/subscribe", body: `{"email":"race@example.com"}`})
			if rec.Code != http.StatusOK {
				t.Errorf("status %d", rec.Code)
			}
		}()
	}
	wg.Wait()
	if got := n.app.TestMailer.TotalSend(); got != 1 {
		t.Fatalf("racing signups sent %d confirmations, want 1", got)
	}
}

func TestNewsletterSendIntervalValidationAndInjectedPacing(t *testing.T) {
	t.Setenv("NEWSLETTER_SEND_INTERVAL_MS", "invalid")
	if _, err := newsletterSendInterval(); err == nil {
		t.Fatal("invalid pacing was accepted")
	}
	for _, bad := range []string{"-1", "3600001", "1.5"} {
		t.Setenv("NEWSLETTER_SEND_INTERVAL_MS", bad)
		if _, err := newsletterSendInterval(); err == nil {
			t.Fatalf("pacing %q was accepted", bad)
		}
	}
	t.Setenv("NEWSLETTER_SEND_INTERVAL_MS", "3600000")
	if d, err := newsletterSendInterval(); err != nil || d != time.Hour {
		t.Fatalf("hourly pacing = %v, %v", d, err)
	}
	t.Setenv("NEWSLETTER_SEND_INTERVAL_MS", "")
	if d, err := newsletterSendInterval(); err != nil || d != time.Second {
		t.Fatalf("default pacing = %v, %v; want 1s", d, err)
	}
	f := newAudienceFixture(t)
	t.Setenv("NEWSLETTER_SEND_INTERVAL_MS", "25")
	var paced []time.Duration
	oldPace := newsletterPace
	newsletterPace = func(d time.Duration) { paced = append(paced, d) }
	t.Cleanup(func() { newsletterPace = oldPace })
	res := f.post(t, "/api/admin/newsletter/send", `{"subject":"Pacing","body_markdown":"Hello","list_id":"`+f.listA.Id+`"}`)
	if res["_status"] != float64(http.StatusOK) {
		t.Fatalf("send = %v", res)
	}
	f.waitForSends(t)
	if len(paced) != 1 || paced[0] != 25*time.Millisecond {
		t.Fatalf("pacing calls = %v, want one 25ms pause for two recipients", paced)
	}
}

func TestNewsletterReadinessRequiresSenderAndAbsoluteHTTPURL(t *testing.T) {
	n := newNewsletterFixture(t, true)
	// PocketBase refuses to save some of these, but a stored config from
	// before validation, or a live edit, can still hold them.
	settings := n.app.Settings()
	settings.Meta.SenderAddress = ""
	if newsletterReady(n.app) {
		t.Fatal("missing sender was ready")
	}
	settings.Meta.SenderAddress = "site@facet.example"
	settings.Meta.AppURL = "facet.example"
	if newsletterReady(n.app) {
		t.Fatal("relative app URL was ready")
	}
	settings.Meta.AppURL = "http://localhost:8080"
	if !newsletterReady(n.app) {
		t.Fatal("local development HTTP URL was rejected")
	}
}

func (f *audienceFixture) get(t *testing.T, url string) map[string]any {
	t.Helper()
	rec := do(t, f.handler, testRequest{url: url, headers: map[string]string{"Authorization": f.adminToken}})
	assertStatus(t, rec, http.StatusOK)
	out := map[string]any{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// assertResumable checks the resumable count the owner sees for one send,
// from the batched helper the send list uses and from the stats endpoint.
func (f *audienceFixture) assertResumable(t *testing.T, sendID string, want int) {
	t.Helper()
	counts, err := newsletterResumableCounts(f.app, []string{sendID})
	if err != nil {
		t.Fatal(err)
	}
	if counts[sendID] != want {
		t.Fatalf("resumable count for %s = %d, want %d", sendID, counts[sendID], want)
	}
	stats := f.get(t, "/api/admin/newsletter/sends/"+sendID+"/stats")
	if stats["resumable_count"] != float64(want) {
		t.Fatalf("stats for %s = %v, want resumable_count %d", sendID, stats, want)
	}
}

func (f *audienceFixture) subscriberID(t *testing.T, email string) string {
	t.Helper()
	sub, err := f.app.FindFirstRecordByFilter("subscribers", "email = {:email}", dbx.Params{"email": email})
	if err != nil {
		t.Fatal(err)
	}
	return sub.Id
}

func TestNewsletterSendHistoryListsLegacyAndLedgerSends(t *testing.T) {
	f := newAudienceFixture(t)
	// Sent before newsletter_sends had timestamps: the upgrade leaves them empty.
	legacy := f.create(t, "newsletter_sends", map[string]any{"subject": "Old", "status": "sent", "recipient_count": 2, "sent_count": 2})
	if _, err := f.app.DB().NewQuery(`UPDATE newsletter_sends SET created = '', updated = '' WHERE id = {:id}`).
		Bind(dbx.Params{"id": legacy.Id}).Execute(); err != nil {
		t.Fatal(err)
	}
	ledger := f.create(t, "newsletter_sends", map[string]any{"subject": "Ledger", "status": "interrupted", "recipient_count": 3})
	for email, state := range map[string]string{"evil@example.com": "pending", "zoe@example.com": "failed", "puppets@example.com": "accepted"} {
		f.create(t, "newsletter_recipients", map[string]any{"send_id": ledger.Id, "subscriber_id": f.subscriberID(t, email), "state": state})
	}

	res := f.get(t, "/api/admin/newsletter/sends")
	data, _ := res["data"].([]any)
	if len(data) != 2 {
		t.Fatalf("send history = %v, want both sends", res)
	}
	got := map[string]map[string]any{}
	for _, item := range data {
		send := item.(map[string]any)
		got[send["id"].(string)] = send
	}
	if first := data[0].(map[string]any); first["id"] != ledger.Id {
		t.Fatalf("first listed send = %v, want the newest, %s", first["id"], ledger.Id)
	}
	if s := got[legacy.Id]; s == nil || s["resumable_count"] != float64(0) || s["created"] != "" || s["sent_count"] != float64(2) {
		t.Fatalf("legacy send = %v", s)
	}
	if s := got[ledger.Id]; s == nil || s["resumable_count"] != float64(2) || s["created"] == "" {
		t.Fatalf("ledger send = %v", s)
	}
}

func TestNewsletterSignupWithoutMailKeepsTheIssuedLink(t *testing.T) {
	n, _, public, _ := signupListFixture(t)
	n.subscribe(t, "reader@example.com")
	n.assertEmailsSent(t, 1)
	path := n.confirmPath(t, "reader@example.com")
	sub := n.subscriber(t, "reader@example.com")
	// Past both cooldowns, still inside the link's lifetime.
	issued := time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339)
	n.setConfirmationTimes(t, sub.Id, issued, issued)
	before := n.subscriber(t, "reader@example.com")

	settings := n.app.Settings()
	settings.SMTP.Enabled = false
	if err := n.app.Save(settings); err != nil {
		t.Fatal(err)
	}
	rec := do(t, n.handler, testRequest{method: http.MethodPost, url: "/api/public/home/subscribe", body: `{"email":"reader@example.com","list":"` + public.GetString("slug") + `"}`})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"status":"confirmation_required"`)
	n.assertEmailsSent(t, 1)
	after := n.subscriber(t, "reader@example.com")
	for _, field := range []string{"status", "pending_list_id", "confirm_token_hash", "confirmation_attempted_at", "confirmation_sent_at"} {
		if after.GetString(field) != before.GetString(field) {
			t.Fatalf("%s changed from %q to %q", field, before.GetString(field), after.GetString(field))
		}
	}

	assertStatus(t, do(t, n.handler, testRequest{url: path}), http.StatusFound)
	if got := strings.Join(n.memberOf(t, "reader@example.com"), ","); got != "default" {
		t.Fatalf("old link joined %q, want default only", got)
	}
}

func TestNewsletterSignedUnsubscribeWritesOnlyStatus(t *testing.T) {
	n := newNewsletterFixture(t, true)
	var saves atomic.Int64
	n.app.OnRecordUpdate("subscribers").BindFunc(func(e *core.RecordEvent) error {
		saves.Add(1)
		return e.Next()
	})
	pending := map[string]string{
		"confirm_token_hash": "pending-confirmation-hash", "pending_list_id": "list12345678901",
		"confirmation_attempted_at": "2026-11-01T10:00:00Z", "confirmation_sent_at": "2026-11-01T10:00:01Z",
	}
	for _, tc := range []struct {
		email string
		req   testRequest
	}{
		{"link@example.com", testRequest{method: http.MethodPost, url: "/api/public/unsubscribe", body: `{"token":"link-token"}`}},
		{"oneclick@example.com", testRequest{
			method: http.MethodPost, url: "/api/public/unsubscribe/one-click/oneclick-token", body: "List-Unsubscribe=One-Click",
			headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		}},
	} {
		token := strings.SplitN(tc.email, "@", 2)[0] + "-token"
		fields := map[string]any{"email": tc.email, "status": "pending", "unsubscribe_token_hash": n.crypto.HMACToken(token)}
		for k, v := range pending {
			fields[k] = v
		}
		n.create(t, "subscribers", fields)
		saves.Store(0)

		rec := do(t, n.handler, tc.req)
		assertStatus(t, rec, http.StatusOK)
		assertContains(t, rec, `"status":"unsubscribed"`)
		rec = do(t, n.handler, tc.req)
		assertStatus(t, rec, http.StatusOK)
		assertContains(t, rec, `"status":"already_unsubscribed"`)
		if got := saves.Load(); got != 0 {
			t.Fatalf("%s saved the whole subscriber record %d times", tc.req.url, got)
		}
		after := n.subscriber(t, tc.email)
		if after.GetString("status") != "unsubscribed" {
			t.Fatalf("%s left status %q", tc.req.url, after.GetString("status"))
		}
		for k, v := range pending {
			if after.GetString(k) != v {
				t.Fatalf("%s changed %s to %q", tc.req.url, k, after.GetString(k))
			}
		}
	}

	// A link issued after the subscriber was read survives the unsubscribe.
	stale := n.create(t, "subscribers", map[string]any{"email": "stale@example.com", "status": "active"})
	if _, err := n.app.DB().NewQuery(`UPDATE subscribers SET confirm_token_hash = 'fresh-hash', pending_list_id = 'list12345678901' WHERE id = {:id}`).
		Bind(dbx.Params{"id": stale.Id}).Execute(); err != nil {
		t.Fatal(err)
	}
	if err := unsubscribeSubscriber(n.app, stale.Id); err != nil {
		t.Fatal(err)
	}
	after := n.subscriber(t, "stale@example.com")
	if after.GetString("status") != "unsubscribed" || after.GetString("confirm_token_hash") != "fresh-hash" || after.GetString("pending_list_id") != "list12345678901" {
		t.Fatalf("unsubscribe after a stale read: status=%q hash=%q list=%q",
			after.GetString("status"), after.GetString("confirm_token_hash"), after.GetString("pending_list_id"))
	}
}
