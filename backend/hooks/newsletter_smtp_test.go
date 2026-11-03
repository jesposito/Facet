package hooks

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"net/textproto"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

// fakeSMTP is a minimal in-process SMTP server on loopback. rcpt and final
// are the replies to RCPT TO and to the end of DATA; dropAfterData closes
// the connection instead of answering the end of DATA.
func fakeSMTP(t *testing.T, rcpt, final string, dropAfterData bool) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				r := bufio.NewReader(conn)
				w := func(s string) { fmt.Fprintf(conn, "%s\r\n", s) }
				w("220 fake ESMTP")
				inData := false
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}
					line = strings.TrimRight(line, "\r\n")
					if inData {
						if line == "." {
							inData = false
							if dropAfterData {
								return
							}
							w(final)
						}
						continue
					}
					switch cmd := strings.ToUpper(line); {
					case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
						w("250 fake")
					case strings.HasPrefix(cmd, "MAIL"):
						w("250 ok")
					case strings.HasPrefix(cmd, "RCPT"):
						w(rcpt)
					case cmd == "DATA":
						w("354 go")
						inData = true
					case cmd == "QUIT":
						w("221 bye")
						return
					default:
						w("250 ok")
					}
				}
			}(conn)
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port
}

func smtpSendError(t *testing.T, port int) error {
	t.Helper()
	client := &mailer.SMTPClient{Host: "127.0.0.1", Port: port}
	return client.Send(&mailer.Message{
		From:    mail.Address{Address: "site@facet.example"},
		To:      []mail.Address{{Address: "reader@example.com"}},
		Subject: "Hi",
		HTML:    "<p>Hi</p>",
	})
}

func TestSMTPFailureStateUsesTypedReplies(t *testing.T) {
	if err := smtpSendError(t, fakeSMTP(t, "250 ok", "250 queued", false)); err != nil {
		t.Fatalf("accepted message returned %v", err)
	}
	for name, tc := range map[string]struct {
		rcpt, final string
		drop        bool
		want        string
	}{
		"4xx at RCPT":     {"450 mailbox busy", "250 queued", false, "failed"},
		"5xx at RCPT":     {"550 no such user", "250 queued", false, "failed"},
		"4xx after DATA":  {"250 ok", "451 try later", false, "failed"},
		"5xx after DATA":  {"250 ok", "554 rejected", false, "failed"},
		"drop after DATA": {"250 ok", "", true, "uncertain"},
	} {
		err := smtpSendError(t, fakeSMTP(t, tc.rcpt, tc.final, tc.drop))
		if err == nil {
			t.Fatalf("%s: no error", name)
		}
		if got := smtpFailureState(err); got != tc.want {
			t.Fatalf("%s: state %q for %T %v, want %q", name, got, err, err, tc.want)
		}
	}

	// Nothing listens on a closed port: the dial fails before any SMTP.
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	if got := smtpFailureState(smtpSendError(t, port)); got != "failed" {
		t.Fatalf("refused dial = %q, want failed", got)
	}

	// Text that merely looks like an SMTP code proves nothing.
	for _, msg := range []string{"550 user unknown", "smtp: 421 closing", "write: broken pipe after 4 bytes", "EOF"} {
		if got := smtpFailureState(errors.New(msg)); got != "uncertain" {
			t.Fatalf("untyped %q = %q, want uncertain", msg, got)
		}
	}
	if got := smtpFailureState(fmt.Errorf("wrapped: %w", &textproto.Error{Code: 452, Msg: "full"})); got != "failed" {
		t.Fatalf("wrapped typed 452 = %q", got)
	}
	if got := smtpFailureState(&textproto.Error{Code: 250, Msg: "odd"}); got != "uncertain" {
		t.Fatalf("non-negative typed reply = %q", got)
	}
	if got := smtpFailureState(&net.OpError{Op: "write", Err: errors.New("reset")}); got != "uncertain" {
		t.Fatalf("mid-session write error = %q", got)
	}
}

func TestOwnerSMTPGuardBlocksSendmailFallback(t *testing.T) {
	n := newNewsletterFixture(t, false)
	RegisterSMTPEnvConfig(n.pb)
	msg := &mailer.Message{From: mail.Address{Address: "site@facet.example"}, To: []mail.Address{{Address: "r@example.com"}}, Subject: "x", HTML: "x"}

	for name, mutate := range map[string]func(s *core.Settings){
		"smtp disabled": func(s *core.Settings) { s.SMTP.Enabled = false; s.SMTP.Host = "smtp.example" },
		"missing host":  func(s *core.Settings) { s.SMTP.Enabled = true; s.SMTP.Host = " " },
		"no sender":     func(s *core.Settings) { s.SMTP.Enabled = true; s.SMTP.Host = "smtp.example"; s.Meta.SenderAddress = "" },
	} {
		// Live settings, unsaved: PocketBase validation would refuse some
		// of these, but the guard must not rely on that.
		settings := n.app.Settings()
		settings.Meta.SenderAddress = "site@facet.example"
		mutate(settings)
		if err := n.app.NewMailClient().Send(msg); !errors.Is(err, errOwnerSMTPUnavailable) {
			t.Fatalf("%s: send err = %v, want guard refusal", name, err)
		}
		if got := n.app.TestMailer.TotalSend(); got != 0 {
			t.Fatalf("%s: %d messages escaped the guard", name, got)
		}
	}

	settings := n.app.Settings()
	settings.SMTP.Enabled, settings.SMTP.Host, settings.Meta.SenderAddress = true, "smtp.example", "site@facet.example"
	if err := n.app.NewMailClient().Send(msg); err != nil {
		t.Fatalf("configured owner SMTP refused: %v", err)
	}
	if got := n.app.TestMailer.TotalSend(); got != 1 {
		t.Fatalf("configured owner SMTP sent %d, want 1", got)
	}
}

func TestOwnerSMTPGuardChecksTheActualClient(t *testing.T) {
	n := newNewsletterFixture(t, true)
	RegisterSMTPEnvConfig(n.pb)
	msg := &mailer.Message{From: mail.Address{Address: "site@facet.example"}, To: []mail.Address{{Address: "r@example.com"}}, Subject: "x", HTML: "x"}

	// Settings are ready, but the client in hand was made under other
	// settings. The terminal callback stands in for the real transport.
	for name, client := range map[string]mailer.Mailer{
		"stale sendmail":        &mailer.Sendmail{},
		"stale blank-host smtp": &mailer.SMTPClient{Host: " "},
	} {
		reached := false
		err := n.app.OnMailerSend().Trigger(&core.MailerEvent{App: n.app, Mailer: client, Message: msg}, func(*core.MailerEvent) error {
			reached = true
			return nil
		})
		if !errors.Is(err, errOwnerSMTPUnavailable) || reached {
			t.Fatalf("%s: err = %v, transport reached = %v", name, err, reached)
		}
	}

	reached := false
	err := n.app.OnMailerSend().Trigger(&core.MailerEvent{App: n.app, Mailer: &mailer.SMTPClient{Host: "smtp.example"}, Message: msg}, func(*core.MailerEvent) error {
		reached = true
		return nil
	})
	if err != nil || !reached {
		t.Fatalf("configured SMTP client: err = %v, transport reached = %v", err, reached)
	}
}

func TestNewsletterReadyNeedsAHostName(t *testing.T) {
	n := newNewsletterFixture(t, true)
	settings := n.app.Settings()
	for _, appURL := range []string{"http://:8080", "https://:443/", "http://@:8080"} {
		settings.Meta.AppURL = appURL
		if newsletterReady(n.app) {
			t.Fatalf("app URL %q without a host name was ready", appURL)
		}
	}
	settings.Meta.AppURL = "https://facet.example"
	if !newsletterReady(n.app) {
		t.Fatal("valid app URL was rejected")
	}
}

func TestNewsletterDeleteRefusesActiveSendAndClearsOwnLedger(t *testing.T) {
	f := newAudienceFixture(t)
	zoe := f.subscriberID(t, "zoe@example.com")
	active := f.create(t, "newsletter_sends", map[string]any{"subject": "Active", "status": "sending", "active_lock": newsletterActiveLock})
	done := f.create(t, "newsletter_sends", map[string]any{"subject": "Done", "status": "sent"})
	other := f.create(t, "newsletter_sends", map[string]any{"subject": "Other", "status": "sent"})
	for _, s := range []*core.Record{active, done, other} {
		f.create(t, "newsletter_recipients", map[string]any{"send_id": s.Id, "subscriber_id": zoe, "state": "accepted"})
	}
	del := func(id string) int {
		return do(t, f.handler, testRequest{method: http.MethodDelete, url: "/api/admin/newsletter/sends/" + id, headers: map[string]string{"Authorization": f.adminToken}}).Code
	}
	receipts := func(id string) int {
		n, err := f.app.CountRecords("newsletter_recipients", dbx.HashExp{"send_id": id})
		if err != nil {
			t.Fatal(err)
		}
		return int(n)
	}

	if code := del(active.Id); code != http.StatusConflict {
		t.Fatalf("delete active send = %d, want 409", code)
	}
	if s, err := f.app.FindRecordById("newsletter_sends", active.Id); err != nil || s.GetString("active_lock") != newsletterActiveLock {
		t.Fatalf("active send after refused delete: %v", err)
	}
	if receipts(active.Id) != 1 {
		t.Fatal("refused delete removed the active send's receipts")
	}

	if code := del(done.Id); code != http.StatusOK {
		t.Fatalf("delete finished send = %d, want 200", code)
	}
	if _, err := f.app.FindRecordById("newsletter_sends", done.Id); err == nil {
		t.Fatal("finished send survived delete")
	}
	if receipts(done.Id) != 0 {
		t.Fatal("finished send's receipts were orphaned")
	}
	if _, err := f.app.FindRecordById("newsletter_sends", other.Id); err != nil || receipts(other.Id) != 1 {
		t.Fatalf("unrelated send or its receipts were touched: %v", err)
	}
	if code := del(done.Id); code != http.StatusNotFound {
		t.Fatalf("repeat delete = %d, want 404", code)
	}
}

func TestNewsletterStopsWhenOutcomeCannotBeSaved(t *testing.T) {
	f := newAudienceFixture(t)
	var once sync.Once
	f.app.OnMailerSend().BindFunc(func(e *core.MailerEvent) error {
		err := e.Next()
		// The outcome row vanishes while SMTP is accepting the message.
		once.Do(func() {
			_, _ = f.app.DB().NewQuery("DELETE FROM newsletter_recipients WHERE state = 'in-flight'").Execute()
		})
		return err
	})
	res := f.post(t, "/api/admin/newsletter/send", `{"subject":"Persist","body_markdown":"Hello","list_id":"`+f.listA.Id+`"}`)
	if res["_status"] != float64(http.StatusOK) {
		t.Fatalf("send = %v", res)
	}
	f.waitForSends(t)
	if got := len(f.drain()); got != 1 {
		t.Fatalf("sent %d emails after progress stopped saving, want 1", got)
	}
	send, err := f.app.FindFirstRecordByFilter("newsletter_sends", "subject = 'Persist'")
	if err != nil {
		t.Fatal(err)
	}
	if send.GetString("status") != "interrupted" || send.GetInt("pending_count") != 1 || send.GetString("active_lock") != "" {
		t.Fatalf("send = status %q pending %d lock %q", send.GetString("status"), send.GetInt("pending_count"), send.GetString("active_lock"))
	}
}

func TestNewsletterSMTPFailuresSetRecipientStates(t *testing.T) {
	f := newAudienceFixture(t)
	f.app.OnMailerSend().BindFunc(func(e *core.MailerEvent) error {
		if e.Message.To[0].Address == "evil@example.com" {
			return &textproto.Error{Code: 550, Msg: "no such user"}
		}
		return errors.New("550 looks like a refusal but is only text")
	})
	res := f.post(t, "/api/admin/newsletter/send", `{"subject":"States","body_markdown":"Hello","list_id":"`+f.listA.Id+`"}`)
	if res["_status"] != float64(http.StatusOK) {
		t.Fatalf("send = %v", res)
	}
	f.waitForSends(t)
	send, _ := f.app.FindFirstRecordByFilter("newsletter_sends", "subject = 'States'")
	if send.GetInt("failed_count") != 1 || send.GetInt("uncertain_count") != 1 || send.GetInt("sent_count") != 0 || send.GetString("status") != "interrupted" {
		t.Fatalf("counts failed=%d uncertain=%d sent=%d status=%q", send.GetInt("failed_count"), send.GetInt("uncertain_count"), send.GetInt("sent_count"), send.GetString("status"))
	}
}

func TestSMTPRelayUnavailableUsesTypedErrors(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	for name, err := range map[string]error{
		"guard":       fmt.Errorf("send: %w", errOwnerSMTPUnavailable),
		"refused":     smtpSendError(t, port),
		"dns":         &net.DNSError{Err: "no such host", Name: "smtp.invalid"},
		"421 closing": &textproto.Error{Code: 421, Msg: "closing"},
		"454 auth":    &textproto.Error{Code: 454, Msg: "temporary auth failure"},
		"530 auth":    &textproto.Error{Code: 530, Msg: "auth required"},
		"534 auth":    &textproto.Error{Code: 534, Msg: "mechanism too weak"},
		"535 auth":    fmt.Errorf("auth: %w", &textproto.Error{Code: 535, Msg: "bad credentials"}),
	} {
		if !smtpRelayUnavailable(err) {
			t.Fatalf("%s (%T %v) not treated as relay-wide", name, err, err)
		}
	}
	for name, err := range map[string]error{
		"550 user":        &textproto.Error{Code: 550, Msg: "no such user"},
		"551 user":        &textproto.Error{Code: 551, Msg: "not local"},
		"553 mailbox":     &textproto.Error{Code: 553, Msg: "bad mailbox"},
		"452 full":        &textproto.Error{Code: 452, Msg: "full"},
		"text 535":        errors.New("535 authentication failed"),
		"text dial":       errors.New("dial tcp: connection refused"),
		"write op":        &net.OpError{Op: "write", Err: errors.New("reset")},
		"drop after DATA": smtpSendError(t, fakeSMTP(t, "250 ok", "", true)),
	} {
		if smtpRelayUnavailable(err) {
			t.Fatalf("%s (%T %v) treated as relay-wide", name, err, err)
		}
	}
}

func TestNewsletterPausesOnRelayFailureAndResumesUnsentOnly(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	closedPort := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	for name, relayErr := range map[string]func() error{
		"535 auth":     func() error { return &textproto.Error{Code: 535, Msg: "bad credentials"} },
		"dial refused": func() error { return smtpSendError(t, closedPort) },
	} {
		t.Run(name, func(t *testing.T) {
			f := newAudienceFixture(t)
			var broken atomic.Bool
			broken.Store(true)
			f.app.OnMailerSend().BindFunc(func(e *core.MailerEvent) error {
				if broken.Load() {
					return relayErr()
				}
				return e.Next()
			})
			res := f.post(t, "/api/admin/newsletter/send", `{"subject":"Relay","body_markdown":"Hello","list_id":"`+f.listA.Id+`"}`)
			if res["_status"] != float64(http.StatusOK) {
				t.Fatalf("send = %v", res)
			}
			f.waitForSends(t)
			if got := len(f.drain()); got != 1 {
				t.Fatalf("broken relay got %d SMTP attempts, want 1", got)
			}
			send, err := f.app.FindFirstRecordByFilter("newsletter_sends", "subject = 'Relay'")
			if err != nil {
				t.Fatal(err)
			}
			if send.GetString("status") != "interrupted" || send.GetInt("failed_count") != 1 ||
				send.GetInt("pending_count") != 1 || send.GetInt("uncertain_count") != 0 || send.GetString("active_lock") != "" {
				t.Fatalf("paused send = status %q failed %d pending %d uncertain %d lock %q", send.GetString("status"),
					send.GetInt("failed_count"), send.GetInt("pending_count"), send.GetInt("uncertain_count"), send.GetString("active_lock"))
			}
			for _, email := range []string{"evil@example.com", "zoe@example.com"} {
				sub, err := f.app.FindFirstRecordByData("subscribers", "email", email)
				if err != nil || sub.GetString("status") != "active" || sub.GetInt("bounce_count") != 0 {
					t.Fatalf("%s consent changed by relay failure: %v", email, err)
				}
			}

			// Receipts that were or may have been delivered must never resend.
			for email, state := range map[string]string{"acc@example.com": "accepted", "unc@example.com": "uncertain"} {
				sub := f.create(t, "subscribers", map[string]any{"email": email, "name": state, "status": "active"})
				f.create(t, "newsletter_recipients", map[string]any{"send_id": send.Id, "subscriber_id": sub.Id, "state": state})
			}
			broken.Store(false)
			rec := do(t, f.handler, testRequest{method: http.MethodPost, url: "/api/admin/newsletter/sends/" + send.Id + "/resume", headers: map[string]string{"Authorization": f.adminToken}})
			assertStatus(t, rec, http.StatusOK)
			f.waitForSends(t)
			if got := strings.Join(keys(f.drain()), ","); got != "evil@example.com,zoe@example.com" {
				t.Fatalf("resume mailed %q, want only the failed and pending recipients", got)
			}
			send, _ = f.app.FindRecordById("newsletter_sends", send.Id)
			if send.GetInt("sent_count") != 3 || send.GetInt("uncertain_count") != 1 || send.GetInt("pending_count") != 0 ||
				send.GetInt("failed_count") != 0 || send.GetString("active_lock") != "" {
				t.Fatalf("resumed send = sent %d uncertain %d pending %d failed %d lock %q", send.GetInt("sent_count"),
					send.GetInt("uncertain_count"), send.GetInt("pending_count"), send.GetInt("failed_count"), send.GetString("active_lock"))
			}
		})
	}
}

func TestNewsletterRecipientRefusalDoesNotStopOthers(t *testing.T) {
	f := newAudienceFixture(t)
	f.app.OnMailerSend().BindFunc(func(e *core.MailerEvent) error {
		if e.Message.To[0].Address == "evil@example.com" {
			return &textproto.Error{Code: 550, Msg: "no such user"}
		}
		return e.Next()
	})
	res := f.post(t, "/api/admin/newsletter/send", `{"subject":"Refusal","body_markdown":"Hello","list_id":"`+f.listA.Id+`"}`)
	if res["_status"] != float64(http.StatusOK) {
		t.Fatalf("send = %v", res)
	}
	f.waitForSends(t)
	if got := len(f.drain()); got != 2 {
		t.Fatalf("recipient refusal allowed %d attempts, want 2", got)
	}
	send, _ := f.app.FindFirstRecordByFilter("newsletter_sends", "subject = 'Refusal'")
	if send.GetInt("failed_count") != 1 || send.GetInt("sent_count") != 1 || send.GetInt("pending_count") != 0 || send.GetString("status") != "partial" {
		t.Fatalf("counts failed=%d sent=%d pending=%d status=%q", send.GetInt("failed_count"), send.GetInt("sent_count"), send.GetInt("pending_count"), send.GetString("status"))
	}
	if sub, err := f.app.FindFirstRecordByData("subscribers", "email", "evil@example.com"); err != nil || sub.GetString("status") != "active" {
		t.Fatalf("550 changed subscriber consent: %v", err)
	}
}

func TestNewsletterResumeRespectsFeatureSwitchAndOneActiveSend(t *testing.T) {
	f := newAudienceFixture(t)
	send := f.create(t, "newsletter_sends", map[string]any{"subject": "Resume", "status": "interrupted", "list_id": f.listA.Id})
	f.create(t, "newsletter_recipients", map[string]any{"send_id": send.Id, "subscriber_id": f.subscriberID(t, "zoe@example.com"), "state": "pending"})

	setFeatures(t, f.securityFixture, map[string]bool{"newsletter": false})
	rec := do(t, f.handler, testRequest{method: http.MethodPost, url: "/api/admin/newsletter/sends/" + send.Id + "/resume", headers: map[string]string{"Authorization": f.adminToken}})
	assertStatus(t, rec, http.StatusForbidden)
	if err := resumeNewsletter(f.pb, f.crypto, send.Id); !errors.Is(err, errNewsletterFeatureOff) {
		t.Fatalf("resume with feature off = %v", err)
	}
	setFeatures(t, f.securityFixture, map[string]bool{"newsletter": true})

	// Hold the single sender slot so racing resumes can't finish early.
	sendSemaphore <- struct{}{}
	var ok, conflict int
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := do(t, f.handler, testRequest{method: http.MethodPost, url: "/api/admin/newsletter/sends/" + send.Id + "/resume", headers: map[string]string{"Authorization": f.adminToken}})
			mu.Lock()
			defer mu.Unlock()
			switch rec.Code {
			case http.StatusOK:
				ok++
			case http.StatusConflict:
				conflict++
			default:
				t.Errorf("resume status %d: %s", rec.Code, rec.Body.String())
			}
		}()
	}
	wg.Wait()
	<-sendSemaphore
	f.waitForSends(t)
	if ok != 1 || conflict != 5 {
		t.Fatalf("racing resumes ok=%d conflict=%d, want 1 and 5", ok, conflict)
	}
	if got := len(f.drain()); got != 1 {
		t.Fatalf("racing resumes sent %d emails, want 1", got)
	}
}

func TestNewsletterRecipientsLoadInCreationOrder(t *testing.T) {
	f := newAudienceFixture(t)
	send := f.create(t, "newsletter_sends", map[string]any{"subject": "Order", "status": "interrupted"})
	for _, email := range []string{"zoe@example.com", "evil@example.com"} {
		f.create(t, "newsletter_recipients", map[string]any{"send_id": send.Id, "subscriber_id": f.subscriberID(t, email), "state": "pending"})
	}
	got, err := resumableNewsletterRecipients(f.app, send.Id)
	if err != nil || len(got) != 2 {
		t.Fatalf("recipients = %d, %v", len(got), err)
	}
	if got[0].GetString("created") == "" {
		t.Fatal("recipient has no created timestamp")
	}
}

// ── Confirmation links and ambiguous SMTP outcomes ────────────────────

func (n *newsletterFixture) subscriber(t *testing.T, email string) *core.Record {
	t.Helper()
	sub, err := n.app.FindFirstRecordByFilter("subscribers", "email = {:e}", dbx.Params{"e": email})
	if err != nil {
		t.Fatalf("subscriber %s: %v", email, err)
	}
	return sub
}

func (n *newsletterFixture) setConfirmationTimes(t *testing.T, id, attempted, sent string) {
	t.Helper()
	if _, err := n.app.DB().NewQuery(`UPDATE subscribers SET confirmation_attempted_at = {:a},
		confirmation_sent_at = {:s} WHERE id = {:id}`).
		Bind(dbx.Params{"a": attempted, "s": sent, "id": id}).Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNewsletterConfirmationLinkWorksAfterAmbiguousSMTPError(t *testing.T) {
	n := newNewsletterFixture(t, true)
	var delivered atomic.Pointer[mailer.Message]
	n.app.OnMailerSend().BindFunc(func(e *core.MailerEvent) error {
		if delivered.CompareAndSwap(nil, e.Message) {
			// The server took DATA, then the reply was lost.
			return errors.New("connection reset after DATA")
		}
		return e.Next()
	})
	n.subscribe(t, "dropped@example.com")
	sub := n.subscriber(t, "dropped@example.com")
	if got := sub.GetString("confirmation_sent_at"); got != "" {
		t.Fatalf("ambiguous submission recorded acceptance at %q", got)
	}
	if sub.GetString("confirmation_attempted_at") == "" || sub.GetString("confirm_token_hash") == "" {
		t.Fatal("token issuance was not saved before the SMTP attempt")
	}
	msg := delivered.Load()
	if msg == nil {
		t.Fatal("no confirmation message was attempted")
	}
	m := confirmLink.FindStringSubmatch(msg.Text)
	if m == nil {
		t.Fatalf("no confirmation link in:\n%s", msg.Text)
	}
	assertStatus(t, do(t, n.handler, testRequest{url: m[1]}), http.StatusFound)
	if got := n.status(t, "dropped@example.com"); got != "active" {
		t.Fatalf("status after confirming a delivered link = %q", got)
	}
	if got := strings.Join(n.memberOf(t, "dropped@example.com"), ","); got != "default" {
		t.Fatalf("member of %q after confirming, want default", got)
	}
}

func TestNewsletterConfirmationLifetimeNeedsAValidIssueTime(t *testing.T) {
	n := newNewsletterFixture(t, true)
	n.subscribe(t, "reader@example.com")
	path := n.confirmPath(t, "reader@example.com")
	id := n.subscriber(t, "reader@example.com").Id
	now := time.Now().UTC()
	fresh := now.Format(time.RFC3339)
	expired := now.Add(-subscribeConfirmationLifetime - time.Hour).Format(time.RFC3339)

	for _, c := range []struct{ name, attempted, sent string }{
		{"expired issue time", expired, fresh},
		{"malformed issue time does not fall back", "not-a-time", fresh},
		{"no timestamps", "", ""},
		{"expired legacy link", "", expired},
	} {
		n.setConfirmationTimes(t, id, c.attempted, c.sent)
		assertStatus(t, do(t, n.handler, testRequest{url: path}), http.StatusNotFound)
		if got := n.status(t, "reader@example.com"); got != "pending" {
			t.Fatalf("%s: status = %q", c.name, got)
		}
	}
	// Issued before confirmation_attempted_at existed: the accepted time
	// still bounds the link.
	n.setConfirmationTimes(t, id, "", fresh)
	assertStatus(t, do(t, n.handler, testRequest{url: path}), http.StatusFound)
	if got := n.status(t, "reader@example.com"); got != "active" {
		t.Fatalf("legacy link: status = %q", got)
	}
}

func TestNewsletterConfirmationDuringSubmissionIsNotUndone(t *testing.T) {
	for _, ambiguous := range []bool{false, true} {
		t.Run(fmt.Sprintf("ambiguous=%v", ambiguous), func(t *testing.T) {
			n := newNewsletterFixture(t, true)
			var confirmCode atomic.Int64
			var once atomic.Bool
			n.app.OnMailerSend().BindFunc(func(e *core.MailerEvent) error {
				if !once.CompareAndSwap(false, true) {
					return e.Next() // the welcome email
				}
				// The reader follows the link before the final SMTP reply.
				if m := confirmLink.FindStringSubmatch(e.Message.Text); m != nil {
					confirmCode.Store(int64(do(t, n.handler, testRequest{url: m[1]}).Code))
				}
				if ambiguous {
					return errors.New("connection reset after DATA")
				}
				return e.Next()
			})
			n.subscribe(t, "quick@example.com")
			if got := confirmCode.Load(); got != http.StatusFound {
				t.Fatalf("confirming during submission returned %d", got)
			}
			sub := n.subscriber(t, "quick@example.com")
			if sub.GetString("status") != "active" || sub.GetString("confirm_token_hash") != "" || sub.GetString("pending_list_id") != "" {
				t.Fatalf("confirmation undone: status=%q hash=%q pending_list=%q",
					sub.GetString("status"), sub.GetString("confirm_token_hash"), sub.GetString("pending_list_id"))
			}
			if got := strings.Join(n.memberOf(t, "quick@example.com"), ","); got != "default" {
				t.Fatalf("member of %q, want default", got)
			}
			if got := sub.GetString("confirmation_sent_at"); got != "" {
				t.Fatalf("acceptance recorded against a used token: %q", got)
			}
		})
	}
}

func TestNewsletterRetryForExistingReaderKeepsConcurrentConsent(t *testing.T) {
	n := newNewsletterFixture(t, true)
	n.create(t, "subscribers", map[string]any{"email": "retry@example.com", "status": "pending"})
	var once atomic.Bool
	var confirmCode atomic.Int64
	n.app.OnMailerSend().BindFunc(func(e *core.MailerEvent) error {
		if once.CompareAndSwap(false, true) {
			if m := confirmLink.FindStringSubmatch(e.Message.Text); m != nil {
				confirmCode.Store(int64(do(t, n.handler, testRequest{url: m[1]}).Code))
			}
		}
		return e.Next()
	})
	n.subscribe(t, "retry@example.com")
	if got := confirmCode.Load(); got != http.StatusFound {
		t.Fatalf("confirming during submission returned %d", got)
	}
	sub := n.subscriber(t, "retry@example.com")
	if sub.GetString("status") != "active" || sub.GetString("confirm_token_hash") != "" {
		t.Fatalf("existing reader's confirmation undone: status=%q hash=%q", sub.GetString("status"), sub.GetString("confirm_token_hash"))
	}
}
