package hooks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase/core"
)

// foreignAuthHeaders signs in an account from an auth collection other than
// the owner's. Its token is valid, so a refusal comes from the owner check and
// not from the check for a missing token.
func (f *securityFixture) foreignAuthHeaders(t *testing.T) map[string]string {
	t.Helper()
	visitors := core.NewAuthCollection("visitors")
	if err := f.app.Save(visitors); err != nil {
		t.Fatal(err)
	}
	visitor := newAuthRecord(visitors, "visitor@example.test", "a-strong-password")
	if err := f.app.Save(visitor); err != nil {
		t.Fatal(err)
	}
	token, err := visitor.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	return map[string]string{"Authorization": token}
}

func TestForeignAuthCollectionCannotManageShareLinks(t *testing.T) {
	r := newShareRegenFixture(t)
	foreign := r.f.foreignAuthHeaders(t)
	old := r.token(t, "old-link", r.view.Id, map[string]any{"max_uses": 0})
	generateBody := fmt.Sprintf(`{"view_id":%q}`, r.view.Id)

	for _, tc := range []struct{ name, url, body string }{
		{"generate", "/api/share/generate", generateBody},
		{"revoke", "/api/share/revoke/" + old.Id, ""},
		{"regenerate", "/api/share/regenerate/" + old.Id, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, r.h, testRequest{method: http.MethodPost, url: tc.url, body: tc.body, headers: foreign})
			assertStatus(t, rec, http.StatusForbidden)
			assertNotContains(t, rec, `"token"`)
			r.assertUntouched(t, old)
		})
	}
	r.assertLink(t, "old-link", true)

	// The owner can still do all three.
	issued := decodeIssuedToken(t, r.generate(t, generateBody))
	r.assertLink(t, issued.Token, true)

	replaced := decodeIssuedToken(t, r.regenerate(t, old.Id))
	r.assertLink(t, replaced.Token, true)
	r.assertLink(t, "old-link", false)

	rec := do(t, r.h, testRequest{method: http.MethodPost, url: "/api/share/revoke/" + replaced.ID, headers: r.auth})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"status":"revoked"`)
	r.assertLink(t, replaced.Token, false)
}

func TestForeignAuthCollectionIsAVisitorToResumeGeneration(t *testing.T) {
	r := newResumeFixture(t)
	foreign := r.foreignAuthHeaders(t)
	_, adminToken := r.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	owner := map[string]string{"Authorization": adminToken}

	for _, slug := range []string{"inactive-view", "private-view"} {
		t.Run(slug, func(t *testing.T) {
			assertStatus(t, generate(t, r.handler, slug, foreign, ""), http.StatusNotFound)
		})
	}
	if n := len(r.ai.calls()); n != 0 {
		t.Fatalf("refused requests still reached the AI provider %d times", n)
	}

	// A slug that doesn't exist exercises the limit without pandoc or the AI.
	fromOneClient := func(headers map[string]string) testRequest {
		return testRequest{
			method:     http.MethodPost,
			url:        "/api/view/no-such-view/generate",
			body:       `{"format":"docx"}`,
			headers:    headers,
			remoteAddr: "203.0.113.50:4000",
		}
	}
	for i := 0; i < 5; i++ {
		assertStatus(t, do(t, r.handler, fromOneClient(foreign)), http.StatusNotFound)
	}
	assertStatus(t, do(t, r.handler, fromOneClient(foreign)), http.StatusTooManyRequests)

	// The owner skips the limit, even from the client that used it up.
	for i := 0; i < 6; i++ {
		assertStatus(t, do(t, r.handler, fromOneClient(owner)), http.StatusNotFound)
	}

	// The owner still reaches an inactive view. Without pandoc the request
	// stops at the pandoc check, which comes after the view lookup.
	want := http.StatusServiceUnavailable
	if _, err := exec.LookPath("pandoc"); err == nil {
		want = http.StatusOK
	}
	assertStatus(t, generate(t, r.handler, "inactive-view", owner, ""), want)
}

// failingPandoc puts a stand-in pandoc on PATH. It passes the install check and
// then fails the conversion, which runs after the AI provider has answered, so
// these tests need neither pandoc nor a LaTeX install.
func failingPandoc(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\n[ \"$1\" = --version ] && exit 0\nexit 1\n"
	if err := os.WriteFile(filepath.Join(dir, "pandoc"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestResumeProviderChoiceStaysWithTheOwner(t *testing.T) {
	failingPandoc(t)
	r := newResumeFixture(t)
	foreign := r.foreignAuthHeaders(t)
	_, adminToken := r.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	body := `{"format":"docx","provider_id":"` + r.other.Id + `"}`

	// Conversion fails on purpose. The AI calls and the export show who ran.
	assertStatus(t, generate(t, r.handler, "public-view", foreign, body), http.StatusInternalServerError)
	if n := len(r.ai.calls()); n != 1 {
		t.Fatalf("default provider got %d calls; only the owner may pick another provider", n)
	}

	assertStatus(t, generate(t, r.handler, "public-view", map[string]string{"Authorization": adminToken}, body), http.StatusInternalServerError)
	if n := len(r.ai.calls()); n != 1 {
		t.Fatalf("default provider got %d calls; the owner's choice was ignored", n)
	}
	chosen, err := r.app.FindRecordsByFilter("view_exports", "ai_provider = {:id}", "", 10, 0, map[string]any{"id": r.other.Id})
	if err != nil || len(chosen) != 1 {
		t.Fatalf("exports from the owner's provider = %d, err %v; want 1", len(chosen), err)
	}
}

type publishedDelivery struct {
	Event string `json:"event"`
	Data  struct {
		ID         string `json:"id"`
		Collection string `json:"collection"`
		Title      string `json:"title"`
		Slug       string `json:"slug"`
	} `json:"data"`
}

// newWebhookFixture subscribes one active webhook to post.published. Tests that
// use it must not call f.serve, which would start the delivery workers.
// Dispatch saves each delivery before queueing it, so the rows show what would
// be sent while nothing leaves the process.
func newWebhookFixture(t *testing.T) *securityFixture {
	t.Helper()
	f := newSecurityFixture(t)
	RegisterWebhookHooks(f.pb, f.rl)
	f.create(t, "webhooks", map[string]any{
		"label": "Announcer", "url": "https://example.com/hook", "secret": "test-secret",
		"events": []string{services.EventPostPublished}, "is_active": true,
	})
	return f
}

func publishedDeliveries(t *testing.T, f *securityFixture) []publishedDelivery {
	t.Helper()
	records, err := f.app.FindRecordsByFilter("webhook_deliveries", "event_name = {:event}", "", 100, 0, map[string]any{"event": services.EventPostPublished})
	if err != nil {
		t.Fatal(err)
	}
	deliveries := make([]publishedDelivery, 0, len(records))
	for _, record := range records {
		var delivery publishedDelivery
		if err := json.Unmarshal([]byte(record.GetString("request_body")), &delivery); err != nil {
			t.Fatal(err)
		}
		deliveries = append(deliveries, delivery)
	}
	return deliveries
}

func TestPostPublishedFollowsPublicNonDraftUpdates(t *testing.T) {
	for _, tc := range []struct {
		collection string
		fields     map[string]any
	}{
		{"posts", map[string]any{"title": "Post", "slug": "webhook-post", "content": "x", "visibility": "private"}},
		{"talks", map[string]any{"title": "Talk", "slug": "webhook-talk", "event": "Conf", "date": "2026-02-01 00:00:00.000Z", "description": "x", "visibility": "private"}},
		{"custom_content", map[string]any{"title": "Custom", "visibility": "private"}},
	} {
		t.Run(tc.collection, func(t *testing.T) {
			f := newWebhookFixture(t)
			record := f.create(t, tc.collection, tc.fields)
			title := record.GetString("title")

			want := 0
			for _, step := range []struct {
				name   string
				fields map[string]any
				sends  bool
			}{
				{"private save", map[string]any{"is_draft": false}, false},
				{"public draft", map[string]any{"visibility": "public", "is_draft": true}, false},
				{"unlisted", map[string]any{"visibility": "unlisted", "is_draft": false}, false},
				{"no visibility", map[string]any{"visibility": ""}, false},
				{"made public", map[string]any{"visibility": "public"}, true},
				{"public edit", map[string]any{"title": title + " edited"}, true},
				{"made private again", map[string]any{"visibility": "private"}, false},
			} {
				setFields(t, f, record, step.fields)
				if step.sends {
					want++
				}
				if got := publishedDeliveries(t, f); len(got) != want {
					t.Fatalf("after %s: %d deliveries, want %d: %+v", step.name, len(got), want, got)
				}
			}

			titles := map[string]bool{}
			for _, d := range publishedDeliveries(t, f) {
				if d.Event != services.EventPostPublished || d.Data.ID != record.Id || d.Data.Collection != tc.collection || d.Data.Slug != record.GetString("slug") {
					t.Fatalf("delivery doesn't describe the item: %+v", d)
				}
				titles[d.Data.Title] = true
			}
			if !titles[title] || !titles[title+" edited"] {
				t.Fatalf("delivered titles = %v", titles)
			}
		})
	}
}

func TestPrivateItemGrantDoesNotAnnounceTheItem(t *testing.T) {
	f := newWebhookFixture(t)
	view := f.create(t, "views", map[string]any{"name": "Portfolio", "slug": "portfolio", "visibility": "public", "is_active": true})
	post := f.create(t, "posts", map[string]any{"title": "Private post", "slug": "private-post", "content": "x", "visibility": "private"})
	talk := f.create(t, "talks", map[string]any{"title": "Private talk", "slug": "private-talk", "event": "Conf", "date": "2026-02-01 00:00:00.000Z", "description": "x", "visibility": "private"})
	public := f.create(t, "posts", map[string]any{"title": "Public post", "slug": "public-post", "content": "x", "visibility": "public"})
	setViewSection(t, f, view, `{"section":"posts","enabled":true,"items":`+idList(t, post.Id)+`},{"section":"talks","enabled":true,"items":`+idList(t, talk.Id)+`}`)

	// What POST /api/view/{id}/private-items runs, without starting a server.
	granted, err := grantSelectedPrivateRecords(f.app, view.Id)
	if err != nil || granted != 2 {
		t.Fatalf("granted %d, err %v; want 2", granted, err)
	}
	if !hasOverride(t, f, "posts", post.Id, view.Id) || !hasOverride(t, f, "talks", talk.Id, view.Id) {
		t.Fatal("a selected private item wasn't granted")
	}
	if got := publishedDeliveries(t, f); len(got) != 0 {
		t.Fatalf("granting private items sent %+v", got)
	}

	// The same webhook does hear about a public item.
	setFields(t, f, public, map[string]any{"title": "Public post, edited"})
	got := publishedDeliveries(t, f)
	if len(got) != 1 || got[0].Data.ID != public.Id {
		t.Fatalf("public update sent %+v, want one delivery for %s", got, public.Id)
	}
}

// Each row is an owner-only Facet endpoint. A token from another auth
// collection and an anonymous request are both refused before the handler
// changes anything. Routes with the middleware answer the foreign token with
// 403, and routes that check inside the handler answer 401. owner is 0 where
// the test only checks the refusal, for example because the owner's call would
// reach an outside service.
func TestOwnerEndpointsRefuseOtherAuthCollections(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterResumeHooks(f.pb, f.crypto, f.share)
	RegisterResumeUploadHooks(f.pb, f.crypto)
	RegisterAIHooks(f.pb, services.NewAIService(f.crypto), f.crypto)
	RegisterSiteSettingsHooks(f.pb, f.crypto)
	RegisterMediaHooks(f.pb, "")
	RegisterBackupHooks(f.pb)
	RegisterPasswordHooks(f.pb, f.crypto, f.rl)
	RegisterDuplicateHooks(f.pb)
	RegisterDemoHandlers(f.pb)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	RegisterTOTPHooks(f.pb, f.crypto, f.rl)
	RegisterPasswordChangeEndpoint(f.pb, f.rl)

	foreign := f.foreignAuthHeaders(t)
	_, adminToken := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	owner := map[string]string{"Authorization": adminToken}

	// Nothing listens on this address. No row sends the owner to it.
	provider := f.aiProvider(t, "Default", "http://127.0.0.1:1", true)
	view := f.create(t, "views", map[string]any{"name": "Facet", "slug": "facet", "visibility": "public", "is_active": true})
	locked := f.create(t, "views", map[string]any{"name": "Locked", "slug": "locked", "visibility": "public", "is_active": true})
	export := f.create(t, "view_exports", map[string]any{"view": view.Id, "format": "docx", "status": "completed"})
	post := f.create(t, "posts", map[string]any{"title": "Post", "slug": "post", "content": "x", "visibility": "public"})
	h := f.serve(t)

	rows := []struct {
		name, method, url, body string
		foreign, owner          int
	}{
		{"list exports", http.MethodGet, "/api/view/facet/exports", "", http.StatusForbidden, http.StatusOK},
		{"delete an export", http.MethodDelete, "/api/view/facet/exports/" + export.Id, "", http.StatusForbidden, http.StatusOK},
		{"list AI models", http.MethodPost, "/api/ai/models", `{"type":"openai"}`, http.StatusForbidden, 0},
		{"test an AI provider", http.MethodPost, "/api/ai/test/" + provider.Id, "", http.StatusForbidden, 0},
		{"AI rewrite", http.MethodPost, "/api/ai/rewrite", `{"content":"x"}`, http.StatusForbidden, 0},
		{"read encryption status", http.MethodGet, "/api/system/encryption-status", "", http.StatusForbidden, http.StatusOK},
		{"read SMTP settings", http.MethodGet, "/api/smtp-settings", "", http.StatusForbidden, http.StatusOK},
		{"save SMTP settings", http.MethodPut, "/api/smtp-settings", `{"host":"smtp.foreign.test"}`, http.StatusForbidden, 0},
		{"read Stripe settings", http.MethodGet, "/api/stripe-settings", "", http.StatusForbidden, http.StatusOK},
		{"upload a favicon", http.MethodPost, "/api/site-settings/favicon", "", http.StatusUnauthorized, 0},
		{"remove the favicon", http.MethodDelete, "/api/site-settings/favicon", "", http.StatusUnauthorized, http.StatusOK},
		{"set a Facet password", http.MethodPost, "/api/password/set", fmt.Sprintf(`{"view_id":%q,"password":"a-view-password"}`, locked.Id), http.StatusForbidden, http.StatusOK},
		{"read Facet memberships", http.MethodGet, "/api/admin/view-memberships?collection=posts", "", http.StatusForbidden, 0},
		{"duplicate a post", http.MethodPost, "/api/admin/duplicate/posts/" + post.Id, "", http.StatusUnauthorized, http.StatusCreated},
		{"import a resume", http.MethodPost, "/api/resume/import", `{"filename":"cv.pdf","visibility":"private","parsed":{"experience":[{"company":"Acme","title":"Engineer"}]}}`, http.StatusForbidden, http.StatusOK},
		{"read storage info", http.MethodGet, "/api/media/storage-info", "", http.StatusForbidden, http.StatusOK},
		{"add external media", http.MethodPost, "/api/media/external", `{"url":"https://youtu.be/example"}`, http.StatusForbidden, 0},
		{"start a backup", http.MethodPost, "/api/admin/backup", "", http.StatusForbidden, 0},
		{"restore a backup", http.MethodPost, "/api/admin/backup/restore", `{"filename":"example.zip"}`, http.StatusForbidden, 0},
		{"read backup status", http.MethodGet, "/api/admin/backup/status", "", http.StatusForbidden, http.StatusOK},
		{"read demo status", http.MethodGet, "/api/demo/status", "", http.StatusUnauthorized, http.StatusOK},
		{"load demo data", http.MethodPost, "/api/demo/enable", "", http.StatusUnauthorized, 0},
		{"clear demo data", http.MethodPost, "/api/demo/restore", "", http.StatusUnauthorized, 0},
	}

	// state is everything the rows above could change.
	state := func(t *testing.T) string {
		t.Helper()
		summary := ""
		for _, collection := range []string{"view_exports", "posts", "experience", "external_media", "demo_profile"} {
			n, err := f.app.CountRecords(collection)
			if err != nil {
				t.Fatal(err)
			}
			summary += fmt.Sprintf("%s=%d ", collection, n)
		}
		backups, err := services.ListBackups(f.app)
		if err != nil {
			t.Fatal(err)
		}
		currentView, err := f.app.FindRecordById("views", locked.Id)
		if err != nil {
			t.Fatal(err)
		}
		currentProvider, err := f.app.FindRecordById("ai_providers", provider.Id)
		if err != nil {
			t.Fatal(err)
		}
		return summary + fmt.Sprintf("backups=%d hash=%q ai_test=%q smtp=%q", len(backups), currentView.GetString("password_hash"), currentProvider.GetString("test_status"), f.app.Settings().SMTP.Host)
	}

	before := state(t)
	for _, tc := range rows {
		t.Run(tc.name, func(t *testing.T) {
			request := testRequest{method: tc.method, url: tc.url, body: tc.body, headers: foreign}
			assertStatus(t, do(t, h, request), tc.foreign)
			request.headers = nil
			assertStatus(t, do(t, h, request), http.StatusUnauthorized)
			if got := state(t); got != before {
				t.Fatalf("refused request changed %q to %q", before, got)
			}
		})
	}

	// Public routes keep working, and a foreign token gets the visitor's answer.
	for name, headers := range map[string]map[string]string{"anonymous": nil, "foreign token": foreign} {
		t.Run(name, func(t *testing.T) {
			rec := do(t, h, testRequest{url: "/api/ai/status", headers: headers})
			assertStatus(t, rec, http.StatusOK)
			assertContains(t, rec, `"available":true`)
			assertNotContains(t, rec, "default_provider")
			assertStatus(t, do(t, h, testRequest{url: "/api/site-settings", headers: headers}), http.StatusOK)
		})
	}

	// Facet account security belongs to owner accounts. Native authentication
	// for an operator's other collection is separate from these owner routes.
	assertStatus(t, do(t, h, testRequest{url: "/api/totp/status", headers: foreign}), http.StatusForbidden)

	rec := do(t, h, testRequest{url: "/api/ai/status", headers: owner})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"name":"Default"`)

	for _, tc := range rows {
		if tc.owner != 0 {
			assertStatus(t, do(t, h, testRequest{method: tc.method, url: tc.url, body: tc.body, headers: owner}), tc.owner)
		}
	}
	if _, err := f.app.FindRecordById("view_exports", export.Id); err == nil {
		t.Fatal("the owner's delete left the export in place")
	}
	for collection, want := range map[string]int64{"posts": 2, "experience": 1} {
		if n, err := f.app.CountRecords(collection); err != nil || n != want {
			t.Fatalf("%s count = %d, err %v; want %d", collection, n, err, want)
		}
	}
	protected, err := f.app.FindRecordById("views", locked.Id)
	if err != nil {
		t.Fatal(err)
	}
	if protected.GetString("password_hash") == "" {
		t.Fatal("the owner's password wasn't saved")
	}

	// PocketBase superusers count as the owner too.
	superusers, err := f.app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		t.Fatal(err)
	}
	root := newAuthRecord(superusers, "root@example.com", "a-strong-password")
	if err := f.app.Save(root); err != nil {
		t.Fatal(err)
	}
	rootToken, err := root.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	for _, url := range []string{"/api/view/facet/exports", "/api/demo/status"} {
		assertStatus(t, do(t, h, testRequest{url: url, headers: map[string]string{"Authorization": rootToken}}), http.StatusOK)
	}
}

func TestOnlyOwnerVisitsSkipTheViewCount(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	view := f.create(t, "views", map[string]any{"name": "Portfolio", "slug": "portfolio", "visibility": "public", "is_active": true})
	foreign := f.foreignAuthHeaders(t)
	foreign["User-Agent"] = browserUA
	_, adminToken := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	h := f.serve(t)
	setFeatures(t, f, map[string]bool{"analytics": true})
	viewCount := func() int {
		t.Helper()
		record, err := f.app.FindRecordById("views", view.Id)
		if err != nil {
			t.Fatal(err)
		}
		return record.GetInt("view_count")
	}

	owner := map[string]string{"Authorization": adminToken, "User-Agent": browserUA}
	assertStatus(t, do(t, h, testRequest{url: "/api/view/portfolio/data", headers: owner}), http.StatusOK)
	time.Sleep(300 * time.Millisecond)
	if n := viewCount(); n != 0 {
		t.Fatalf("view_count = %d after the owner's visit, want 0", n)
	}

	assertStatus(t, do(t, h, testRequest{url: "/api/view/portfolio/data", headers: foreign}), http.StatusOK)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && viewCount() < 1 {
		time.Sleep(20 * time.Millisecond)
	}
	if n := viewCount(); n != 1 {
		t.Fatalf("view_count = %d after a foreign account's visit, want 1", n)
	}
}

// Facet account security is restricted to the owner collection. Other auth
// collections retain their native authentication without using these routes.
func TestForeignAccountCannotUseOwnerSecurityRoutes(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterTOTPHooks(f.pb, f.crypto, f.rl)
	RegisterPasswordChangeEndpoint(f.pb, f.rl)
	owner, ownerToken := f.createAdmin(t, "owner@example.com", "owner-strong-password", nil)
	collection := core.NewAuthCollection("visitors")
	if err := f.app.Save(collection); err != nil {
		t.Fatal(err)
	}
	visitor := newAuthRecord(collection, "visitor@example.test", "visitor-strong-password")
	if err := f.app.Save(visitor); err != nil {
		t.Fatal(err)
	}
	token, err := visitor.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	h := f.serve(t)
	headers := map[string]string{"Authorization": token}
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{http.MethodGet, "/api/totp/status", "", http.StatusForbidden},
		{http.MethodPost, "/api/totp/begin-setup", "{}", http.StatusForbidden},
		{http.MethodPost, "/api/totp/confirm-setup", `{"code":"123456"}`, http.StatusForbidden},
		{http.MethodPost, "/api/totp/verify", `{"code":"123456"}`, http.StatusForbidden},
		{http.MethodPost, "/api/totp/disable", `{"code":"123456"}`, http.StatusForbidden},
		{http.MethodPost, "/api/totp/regenerate-codes", `{"code":"123456"}`, http.StatusForbidden},
		{http.MethodPost, "/api/totp/clear-session", "{}", http.StatusForbidden},
		{http.MethodGet, "/api/auth/check-default-password", "", http.StatusUnauthorized},
		{http.MethodPost, "/api/auth/change-password", `{"currentPassword":"visitor-strong-password","newPassword":"another-strong-password"}`, http.StatusUnauthorized},
	} {
		t.Run(tc.path, func(t *testing.T) {
			assertStatus(t, do(t, h, testRequest{method: tc.method, url: tc.path, body: tc.body, headers: headers}), tc.status)
		})
	}
	after, err := f.app.FindRecordById("users", owner.Id)
	if err != nil {
		t.Fatal(err)
	}
	if after.TokenKey() != owner.TokenKey() || !after.ValidatePassword("owner-strong-password") || after.GetBool("totp_enabled") || after.GetString("totp_secret") != "" {
		t.Fatal("another collection changed the owner's security state")
	}
	ownerHeaders := map[string]string{"Authorization": ownerToken}
	assertStatus(t, do(t, h, testRequest{url: "/api/totp/status", headers: ownerHeaders}), http.StatusOK)
	assertStatus(t, do(t, h, testRequest{url: "/api/auth/check-default-password", headers: ownerHeaders}), http.StatusOK)
}
