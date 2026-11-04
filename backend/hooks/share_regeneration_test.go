package hooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

const shareFutureExpiry = "2035-01-15 14:30:00.000Z"

type shareRegenFixture struct {
	*shareVisitFixture
	auth map[string]string
}

type issuedShareToken struct {
	ID    string `json:"id"`
	Token string `json:"token"`
	Name  string `json:"name"`
}

func newShareRegenFixture(t *testing.T) *shareRegenFixture {
	t.Helper()
	s := newShareVisitFixture(t)
	_, adminToken := s.f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	return &shareRegenFixture{shareVisitFixture: s, auth: map[string]string{"Authorization": adminToken}}
}

func (r *shareRegenFixture) regenerate(t *testing.T, id string) *httptest.ResponseRecorder {
	t.Helper()
	return do(t, r.h, testRequest{method: http.MethodPost, url: "/api/share/regenerate/" + id, headers: r.auth})
}

func (r *shareRegenFixture) generate(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	return do(t, r.h, testRequest{method: http.MethodPost, url: "/api/share/generate", body: body, headers: r.auth})
}

func (r *shareRegenFixture) validate(t *testing.T, raw string) *httptest.ResponseRecorder {
	t.Helper()
	rec := do(t, r.h, testRequest{method: http.MethodPost, url: "/api/share/validate", body: fmt.Sprintf(`{"token":%q}`, raw)})
	assertStatus(t, rec, http.StatusOK)
	return rec
}

func (r *shareRegenFixture) assertLink(t *testing.T, raw string, want bool) {
	t.Helper()
	rec := r.validate(t, raw)
	if got := strings.Contains(rec.Body.String(), `"valid":true`); got != want {
		t.Fatalf("link valid = %v, want %v: %s", got, want, rec.Body.String())
	}
}

func (r *shareRegenFixture) record(t *testing.T, id string) *core.Record {
	t.Helper()
	record, err := r.f.app.FindRecordById("share_tokens", id)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func (r *shareRegenFixture) tokensFor(t *testing.T, viewID string) []*core.Record {
	t.Helper()
	records, err := r.f.app.FindRecordsByFilter("share_tokens", "view_id = {:v}", "", 100, 0, map[string]any{"v": viewID})
	if err != nil {
		t.Fatal(err)
	}
	return records
}

// assertUntouched fails if a refused or failed replacement left any trace.
func (r *shareRegenFixture) assertUntouched(t *testing.T, before *core.Record) {
	t.Helper()
	after := r.record(t, before.Id)
	for _, field := range []string{"view_id", "token_hash", "name", "expires_at", "max_uses", "use_count", "is_active"} {
		if after.GetString(field) != before.GetString(field) {
			t.Fatalf("%s changed from %q to %q", field, before.GetString(field), after.GetString(field))
		}
	}
	if got := len(r.tokensFor(t, before.GetString("view_id"))); got != 1 {
		t.Fatalf("view has %d tokens, want only the original", got)
	}
}

func decodeIssuedToken(t *testing.T, rec *httptest.ResponseRecorder) issuedShareToken {
	t.Helper()
	assertStatus(t, rec, http.StatusOK)
	var issued issuedShareToken
	if err := json.Unmarshal(rec.Body.Bytes(), &issued); err != nil {
		t.Fatal(err)
	}
	if issued.ID == "" || issued.Token == "" {
		t.Fatalf("incomplete token response: %s", rec.Body.String())
	}
	return issued
}

func TestShareRegenerateRequiresAuthentication(t *testing.T) {
	r := newShareRegenFixture(t)
	old := r.token(t, "old-link", r.view.Id, nil)

	for name, headers := range map[string]map[string]string{
		"anonymous":    nil,
		"forged token": {"Authorization": "not-a-real-token"},
	} {
		t.Run(name, func(t *testing.T) {
			rec := do(t, r.h, testRequest{method: http.MethodPost, url: "/api/share/regenerate/" + old.Id, headers: headers})
			assertStatus(t, rec, http.StatusUnauthorized)
			r.assertUntouched(t, old)
		})
	}
}

func TestShareRegenerateReplacesLinkAndKeepsItsSettings(t *testing.T) {
	r := newShareRegenFixture(t)
	old := r.token(t, "old-link", r.view.Id, map[string]any{"name": "Sent to Acme", "expires_at": shareFutureExpiry, "max_uses": 5, "use_count": 3})

	issued := decodeIssuedToken(t, r.regenerate(t, old.Id))
	if issued.Name != "Sent to Acme" {
		t.Fatalf("response name = %q", issued.Name)
	}

	next := r.record(t, issued.ID)
	if next.GetString("view_id") != r.view.Id || next.GetString("name") != "Sent to Acme" {
		t.Fatalf("replacement lost its view or name: %v", next.PublicExport())
	}
	if got := next.GetDateTime("expires_at").String(); got != shareFutureExpiry {
		t.Fatalf("expires_at = %q, want %q", got, shareFutureExpiry)
	}
	if next.GetInt("max_uses") != 5 || next.GetInt("use_count") != 0 || !next.GetBool("is_active") {
		t.Fatalf("replacement limits are wrong: %v", next.PublicExport())
	}
	if next.GetString("token_hash") != r.f.share.HMACToken(issued.Token) || next.GetString("token_prefix") != r.f.share.TokenPrefix(issued.Token) {
		t.Fatal("replacement does not use the token HMAC and prefix scheme")
	}
	if r.record(t, old.Id).GetBool("is_active") {
		t.Fatal("old token is still active")
	}

	r.assertLink(t, issued.Token, true)
	r.assertLink(t, "old-link", false)
}

func TestShareRegenerateRollsBackAndKeepsOldLinkUsable(t *testing.T) {
	for _, tc := range []struct {
		name string
		fail func(r *shareRegenFixture, old *core.Record)
	}{
		{"new record save fails", func(r *shareRegenFixture, old *core.Record) {
			r.f.app.OnRecordCreate("share_tokens").BindFunc(func(e *core.RecordEvent) error {
				return errors.New("forced create failure")
			})
		}},
		{"old record save fails", func(r *shareRegenFixture, old *core.Record) {
			r.f.app.OnRecordUpdate("share_tokens").BindFunc(func(e *core.RecordEvent) error {
				if e.Record.Id == old.Id {
					return errors.New("forced update failure")
				}
				return e.Next()
			})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newShareRegenFixture(t)
			old := r.token(t, "old-link", r.view.Id, map[string]any{"name": "Sent to Acme", "expires_at": shareFutureExpiry, "max_uses": 5})
			tc.fail(r, old)

			rec := r.regenerate(t, old.Id)
			assertStatus(t, rec, http.StatusInternalServerError)
			assertContains(t, rec, "failed to regenerate token")
			if strings.Contains(rec.Body.String(), "forced") || strings.Contains(rec.Body.String(), `"token"`) {
				t.Fatalf("failure response leaked detail: %s", rec.Body.String())
			}

			r.assertUntouched(t, old)
			r.assertLink(t, "old-link", true)
		})
	}
}

func TestShareRegenerateConcurrentRequestsMakeOneReplacement(t *testing.T) {
	r := newShareRegenFixture(t)
	old := r.token(t, "old-link", r.view.Id, map[string]any{"max_uses": 0})

	results := concurrently(t, r.h, testRequest{method: http.MethodPost, url: "/api/share/regenerate/" + old.Id, headers: r.auth})
	if got := countStatus(results, http.StatusOK); got != 1 {
		t.Fatalf("%d of %d parallel requests replaced the link", got, burst)
	}
	if got := countStatus(results, http.StatusConflict); got != burst-1 {
		t.Fatalf("%d of %d parallel requests were told it was superseded", got, burst-1)
	}

	active := 0
	tokens := r.tokensFor(t, r.view.Id)
	for _, token := range tokens {
		if token.GetBool("is_active") {
			active++
		}
	}
	if len(tokens) != 2 || active != 1 {
		t.Fatalf("%d tokens with %d active, want 2 with 1 active", len(tokens), active)
	}

	rec := r.regenerate(t, old.Id)
	assertStatus(t, rec, http.StatusConflict)
	assertContains(t, rec, "already been replaced")
}

func TestShareExpiryFormatsRoundTripAsTheSameInstant(t *testing.T) {
	r := newShareRegenFixture(t)
	for _, tc := range []struct{ name, sent string }{
		{"PocketBase stored format", shareFutureExpiry},
		{"browser ISO instant", "2035-01-15T14:30:00.000Z"},
		{"New York offset", "2035-01-15T09:30:00-05:00"},
		{"Auckland offset", "2035-01-16T03:30:00+13:00"},
		{"no zone reads as UTC", "2035-01-15T14:30"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			issued := decodeIssuedToken(t, r.generate(t, fmt.Sprintf(`{"view_id":%q,"expires_at":%q}`, r.view.Id, tc.sent)))
			if got := r.record(t, issued.ID).GetDateTime("expires_at").String(); got != shareFutureExpiry {
				t.Fatalf("stored expires_at = %q, want %q", got, shareFutureExpiry)
			}
			replaced := decodeIssuedToken(t, r.regenerate(t, issued.ID))
			if got := r.record(t, replaced.ID).GetDateTime("expires_at").String(); got != shareFutureExpiry {
				t.Fatalf("replacement expires_at = %q, want %q", got, shareFutureExpiry)
			}
		})
	}

	before := len(r.tokensFor(t, r.view.Id))
	assertStatus(t, r.generate(t, fmt.Sprintf(`{"view_id":%q,"expires_at":"next Tuesday"}`, r.view.Id)), http.StatusBadRequest)
	if got := len(r.tokensFor(t, r.view.Id)); got != before {
		t.Fatalf("invalid expiry created a token: %d, want %d", got, before)
	}
}

func TestShareGenerateRefusesPastExpiryWithoutSaving(t *testing.T) {
	r := newShareRegenFixture(t)
	before := len(r.tokensFor(t, r.view.Id))
	for _, deadline := range []string{"2000-01-01T00:00:00Z", time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)} {
		rec := r.generate(t, fmt.Sprintf(`{"view_id":%q,"expires_at":%q}`, r.view.Id, deadline))
		assertStatus(t, rec, http.StatusBadRequest)
		assertContains(t, rec, "future")
	}
	if got := len(r.tokensFor(t, r.view.Id)); got != before {
		t.Fatalf("past expiry created a token: %d, want %d", got, before)
	}
}

func TestShareRegenerateRefusesLinksItCannotReplace(t *testing.T) {
	r := newShareRegenFixture(t)
	expired := r.token(t, "expired-link", r.view.Id, map[string]any{"expires_at": time.Now().Add(-time.Minute)})
	rec := r.regenerate(t, expired.Id)
	assertStatus(t, rec, http.StatusConflict)
	assertContains(t, rec, "expired")
	r.assertUntouched(t, expired)

	revoked := r.token(t, "revoked-link", r.other.Id, map[string]any{"is_active": false})
	assertStatus(t, r.regenerate(t, revoked.Id), http.StatusConflict)
	r.assertUntouched(t, revoked)

	assertStatus(t, r.regenerate(t, "missingtoken000"), http.StatusNotFound)
}

func TestClosedFacetsGrantNoShareAccessAndSpendNoUses(t *testing.T) {
	r := newShareRegenFixture(t)
	for _, tc := range []struct {
		name   string
		fields map[string]any
	}{
		{"private", map[string]any{"visibility": "private", "is_active": true}},
		{"inactive", map[string]any{"visibility": "unlisted", "is_active": false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fields := map[string]any{"name": tc.name, "slug": tc.name + "-facet"}
			for k, v := range tc.fields {
				fields[k] = v
			}
			view := r.f.create(t, "views", fields)
			raw := tc.name + "-link"
			old := r.token(t, raw, view.Id, map[string]any{"max_uses": 0})

			rec := r.validate(t, raw)
			if want := r.validate(t, "no-such-link").Body.String(); rec.Body.String() != want {
				t.Fatalf("validate response %s differs from the generic %s", rec.Body.String(), want)
			}
			if rec.Header().Get(shareVisitHeader) != "" {
				t.Fatal("refused validation issued a visit proof")
			}
			if got := do(t, r.h, testRequest{url: "/api/view/" + tc.name + "-facet/data", headers: map[string]string{"X-Share-Token": raw}}); got.Code == http.StatusOK {
				t.Fatal("share token opened a closed Facet")
			}

			assertStatus(t, r.regenerate(t, old.Id), http.StatusConflict)
			assertStatus(t, r.generate(t, fmt.Sprintf(`{"view_id":%q}`, view.Id)), http.StatusConflict)
			r.assertUntouched(t, old)
		})
	}
}

func TestPublicFacetShareTokensKeepWorking(t *testing.T) {
	r := newShareRegenFixture(t)
	view := r.f.create(t, "views", map[string]any{"name": "Open", "slug": "open-facet", "visibility": "public", "is_active": true})

	issued := decodeIssuedToken(t, r.generate(t, fmt.Sprintf(`{"view_id":%q}`, view.Id)))
	r.assertLink(t, issued.Token, true)
}

func TestShareGenerateBoundsMaxUses(t *testing.T) {
	r := newShareRegenFixture(t)
	for _, value := range []string{"-1", "1000001", "1.5"} {
		assertStatus(t, r.generate(t, fmt.Sprintf(`{"view_id":%q,"max_uses":%s}`, r.view.Id, value)), http.StatusBadRequest)
	}
	if got := len(r.tokensFor(t, r.view.Id)); got != 0 {
		t.Fatalf("rejected max_uses created %d tokens", got)
	}
	for _, value := range []string{"0", "1000000"} {
		decodeIssuedToken(t, r.generate(t, fmt.Sprintf(`{"view_id":%q,"max_uses":%s}`, r.view.Id, value)))
	}
}
