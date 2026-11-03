package hooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// ---------------------------------------------------------------------------
// last_used_at
// ---------------------------------------------------------------------------

// The old code loaded the key, set last_used_at and called Save. Save writes
// every column, so a revocation that landed between that load and that save
// was undone. This hook revokes the key the moment anything tries to Save it
// for a last-use update, which is exactly that window. With the targeted
// update, Save is never called and the revocation sticks.
func TestTouchAPIKeyLastUsedDoesNotRestoreRevokedKey(t *testing.T) {
	f := newSecurityFixture(t)
	_, key := f.createTestAPIKey(t, "Race", "read:posts", "write:posts")
	expiry := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Millisecond)
	key.Set("expires_at", expiry)
	if err := f.app.Save(key); err != nil {
		t.Fatal(err)
	}

	saves := 0
	f.app.OnRecordUpdate("api_keys").BindFunc(func(e *core.RecordEvent) error {
		saves++
		// An admin revokes the key and cuts its scopes while the stale copy
		// is mid-save.
		if _, err := e.App.NonconcurrentDB().NewQuery(
			`UPDATE api_keys SET is_active = FALSE, scopes = '["read:posts"]' WHERE id = {:id}`,
		).Bind(map[string]any{"id": e.Record.Id}).Execute(); err != nil {
			t.Fatal(err)
		}
		return e.Next()
	})

	used := time.Now().Add(-time.Second)
	if err := touchAPIKeyLastUsed(f.app, key.Id, used); err != nil {
		t.Fatal(err)
	}
	if saves != 0 {
		t.Fatalf("last-use update went through Save %d times; it must update only last_used_at", saves)
	}

	// Apply the admin change the ordinary way, then touch again.
	fresh, err := f.app.FindRecordById("api_keys", key.Id)
	if err != nil {
		t.Fatal(err)
	}
	fresh.Set("is_active", false)
	fresh.Set("scopes", []string{"read:posts"})
	if err := f.app.Save(fresh); err != nil {
		t.Fatal(err)
	}
	saves = 0
	if err := touchAPIKeyLastUsed(f.app, key.Id, used); err != nil {
		t.Fatal(err)
	}

	got, err := f.app.FindRecordById("api_keys", key.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got.GetBool("is_active") {
		t.Fatal("last-use update re-activated a revoked key")
	}
	if scopes := decodeJSONStringSlice(got.Get("scopes")); !reflect.DeepEqual(scopes, []string{"read:posts"}) {
		t.Fatalf("scopes = %v, want the reduced [read:posts]", scopes)
	}
	if !got.GetDateTime("expires_at").Time().Equal(expiry) {
		t.Fatalf("expires_at = %v, want %v", got.GetDateTime("expires_at"), expiry)
	}
	if !got.GetDateTime("last_used_at").Time().Equal(used.UTC().Truncate(time.Millisecond)) {
		t.Fatalf("last_used_at = %v, want %v", got.GetDateTime("last_used_at"), used)
	}
}

func TestUpdateLastUsedAsyncDebouncesAndRetriesAfterDBError(t *testing.T) {
	f := newSecurityFixture(t)
	_, key := f.createTestAPIKey(t, "Debounce", "read:posts")
	clear := func() {
		lastUsedMu.Lock()
		delete(lastUsedCache, key.Id)
		lastUsedMu.Unlock()
	}
	cached := func() bool {
		lastUsedMu.Lock()
		defer lastUsedMu.Unlock()
		_, ok := lastUsedCache[key.Id]
		return ok
	}
	t.Cleanup(clear)

	// First use writes.
	clear()
	updateLastUsedAsync(f.app, key.Id)
	got, _ := f.app.FindRecordById("api_keys", key.Id)
	first := got.GetDateTime("last_used_at")
	if first.IsZero() {
		t.Fatal("first use did not record last_used_at")
	}

	// A second use inside the window doesn't write again.
	if _, err := f.app.NonconcurrentDB().NewQuery(`UPDATE api_keys SET last_used_at = '' WHERE id = {:id}`).
		Bind(map[string]any{"id": key.Id}).Execute(); err != nil {
		t.Fatal(err)
	}
	updateLastUsedAsync(f.app, key.Id)
	got, _ = f.app.FindRecordById("api_keys", key.Id)
	if !got.GetDateTime("last_used_at").IsZero() {
		t.Fatal("use inside the debounce window wrote last_used_at again")
	}

	// A failed write releases the slot so the next request retries.
	clear()
	if _, err := f.app.NonconcurrentDB().NewQuery(`ALTER TABLE api_keys RENAME TO api_keys_moved`).Execute(); err != nil {
		t.Fatal(err)
	}
	updateLastUsedAsync(f.app, key.Id)
	if cached() {
		t.Fatal("failed write left the key in the debounce cache, so no retry for a minute")
	}
	if _, err := f.app.NonconcurrentDB().NewQuery(`ALTER TABLE api_keys_moved RENAME TO api_keys`).Execute(); err != nil {
		t.Fatal(err)
	}
	updateLastUsedAsync(f.app, key.Id)
	got, _ = f.app.FindRecordById("api_keys", key.Id)
	if got.GetDateTime("last_used_at").IsZero() {
		t.Fatal("retry after the failed write did not record last_used_at")
	}
}

// ---------------------------------------------------------------------------
// allowed_origins
// ---------------------------------------------------------------------------

func TestCanonicalOrigin(t *testing.T) {
	good := map[string]string{
		"https://example.com":        "https://example.com",
		"HTTPS://Example.COM/":       "https://example.com",
		"https://example.com:443":    "https://example.com",
		"http://localhost:5173":      "http://localhost:5173",
		"http://example.com:80":      "http://example.com",
		"https://example.com:8443":   "https://example.com:8443",
		"http://[::1]:3000":          "http://[::1]:3000",
		"  https://app.example.org ": "https://app.example.org",
	}
	for in, want := range good {
		got, err := canonicalOrigin(in)
		if err != nil || got != want {
			t.Errorf("canonicalOrigin(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	bad := []string{
		"", "*", "https://*", "https://*.example.com", "null", "example.com",
		"ftp://example.com", "file:///etc/passwd", "javascript:alert(1)",
		"https://example.com/path", "https://example.com?x=1", "https://example.com#frag",
		"https://example.com/?", "https://user:pw@example.com", "https://", "https://example.com:",
		"https://example.com:0", "https://example.com:99999", "https://example.com:abc",
	}
	for _, in := range bad {
		if got, err := canonicalOrigin(in); err == nil {
			t.Errorf("canonicalOrigin(%q) = %q; want an error", in, got)
		}
	}
}

func TestOriginAllowed(t *testing.T) {
	req := func(origins ...string) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/posts", nil)
		for _, o := range origins {
			r.Header.Add("Origin", o)
		}
		return r
	}
	allow := []string{"https://app.example.com", "http://localhost:5173"}
	cases := []struct {
		name    string
		allowed []string
		r       *http.Request
		want    bool
	}{
		{"empty allowlist permits any origin", nil, req("https://evil.example"), true},
		{"empty allowlist permits no origin", nil, req(), true},
		{"no Origin header is a server client", allow, req(), true},
		{"exact match", allow, req("https://app.example.com"), true},
		{"case and default port normalise", allow, req("HTTPS://APP.example.com:443"), true},
		{"other port", allow, req("http://localhost:5174"), false},
		{"other scheme", allow, req("http://app.example.com"), false},
		{"subdomain is not a match", allow, req("https://evil.app.example.com"), false},
		{"suffix trick", allow, req("https://app.example.com.evil.example"), false},
		{"null origin", allow, req("null"), false},
		{"malformed origin", allow, req("not a url"), false},
		{"origin with path", allow, req("https://app.example.com/x"), false},
		{"two Origin headers", allow, req("https://evil.example", "https://app.example.com"), false},
		{"stored wildcard never matches", []string{"*"}, req("https://evil.example"), false},
		{"stored wildcard subdomain never matches", []string{"https://*.example.com"}, req("https://app.example.com"), false},
		{"unparseable stored list blocks browsers", []string{"invalid"}, req("https://app.example.com"), false},
		{"unparseable stored list still allows servers", []string{"invalid"}, req(), true},
	}
	for _, tc := range cases {
		if got := originAllowed(tc.allowed, tc.r); got != tc.want {
			t.Errorf("%s: originAllowed = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestAPIKeyMiddlewareEnforcesAllowedOrigins(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterAPIKeyHooks(f.pb, f.crypto, f.rl)
	h := f.serve(t)

	restricted, rKey := f.createTestAPIKey(t, "Browser widget", "read:skills")
	rKey.Set("allowed_origins", []string{"https://widget.example.com"})
	if err := f.app.Save(rKey); err != nil {
		t.Fatal(err)
	}
	open, _ := f.createTestAPIKey(t, "Server", "read:skills")

	// A key whose stored value isn't a list (hand-edited) must not fail open.
	broken, bKey := f.createTestAPIKey(t, "Broken", "read:skills")
	if _, err := f.app.NonconcurrentDB().NewQuery(`UPDATE api_keys SET allowed_origins = '"*"' WHERE id = {:id}`).
		Bind(map[string]any{"id": bKey.Id}).Execute(); err != nil {
		t.Fatal(err)
	}

	get := func(key, origin string) int {
		headers := map[string]string{"Authorization": "Bearer " + key}
		if origin != "" {
			headers["Origin"] = origin
		}
		return do(t, h, testRequest{url: "/api/v1/skills", headers: headers}).Code
	}
	cases := []struct {
		name   string
		key    string
		origin string
		want   int
	}{
		{"allowed origin", restricted, "https://widget.example.com", http.StatusOK},
		{"other origin", restricted, "https://evil.example", http.StatusForbidden},
		{"lookalike origin", restricted, "https://widget.example.com.evil.example", http.StatusForbidden},
		{"null origin", restricted, "null", http.StatusForbidden},
		{"malformed origin", restricted, "widget.example.com", http.StatusForbidden},
		{"no origin (server client)", restricted, "", http.StatusOK},
		{"unrestricted key from any origin", open, "https://evil.example", http.StatusOK},
		{"broken stored config from a browser", broken, "https://evil.example", http.StatusForbidden},
		{"broken stored config from a server", broken, "", http.StatusOK},
	}
	for _, tc := range cases {
		if got := get(tc.key, tc.origin); got != tc.want {
			t.Errorf("%s: status = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestCreateAPIKeyValidatesOriginsAndExpiry(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterAPIKeyHooks(f.pb, f.crypto, f.rl)
	h := f.serve(t)
	_, token := f.createAdmin(t, "owner@example.com", "correct-horse-battery", nil)
	post := func(body string) *httptest.ResponseRecorder {
		return do(t, h, testRequest{
			method:  http.MethodPost,
			url:     "/api/admin/api-keys",
			body:    body,
			headers: map[string]string{"Authorization": token},
		})
	}

	for _, body := range []string{
		`{"label":"x","scopes":["read:posts"],"allowed_origins":["*"]}`,
		`{"label":"x","scopes":["read:posts"],"allowed_origins":["https://*.example.com"]}`,
		`{"label":"x","scopes":["read:posts"],"allowed_origins":["example.com"]}`,
		`{"label":"x","scopes":["read:posts"],"allowed_origins":["https://example.com/app"]}`,
		`{"label":"x","scopes":["read:posts"],"expires_at":"next tuesday"}`,
		`{"label":"x","scopes":["read:posts"],"expires_at":"2001-01-01"}`,
		`{"label":"x","scopes":["admin:everything"]}`,
	} {
		if rec := post(body); rec.Code != http.StatusBadRequest {
			t.Errorf("POST %s: status = %d, want 400 (%s)", body, rec.Code, rec.Body.String())
		}
	}

	expiry := time.Now().AddDate(0, 1, 0).UTC().Format("2006-01-02")
	rec := post(`{"label":"Widget","scopes":["read:posts","write:posts"],` +
		`"allowed_origins":["HTTPS://Widget.Example.com/","https://widget.example.com:443"],"expires_at":"` + expiry + `"}`)
	assertStatus(t, rec, http.StatusCreated)
	var created struct {
		ID             string   `json:"id"`
		Scopes         []string `json:"scopes"`
		AllowedOrigins []string `json:"allowed_origins"`
		ExpiresAt      string   `json:"expires_at"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(created.AllowedOrigins, []string{"https://widget.example.com"}) {
		t.Fatalf("allowed_origins = %v, want one canonical origin", created.AllowedOrigins)
	}
	if !strings.HasPrefix(created.ExpiresAt, expiry+" 23:59:59") {
		t.Fatalf("expires_at = %q, want the end of %s", created.ExpiresAt, expiry)
	}
	stored, err := f.app.FindRecordById("api_keys", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := decodeJSONStringSlice(stored.Get("allowed_origins")); !reflect.DeepEqual(got, created.AllowedOrigins) {
		t.Fatalf("stored allowed_origins = %v", got)
	}
}

// ---------------------------------------------------------------------------
// v1 DTOs against the real schema
// ---------------------------------------------------------------------------

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return out
}

// readOne fetches a v1 list and returns the item with the given id.
func readOne(t *testing.T, h http.Handler, key, path, id string) map[string]any {
	t.Helper()
	rec := do(t, h, testRequest{url: path, headers: map[string]string{"Authorization": "Bearer " + key}})
	assertStatus(t, rec, http.StatusOK)
	var list struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	for _, item := range list.Data {
		if item["id"] == id {
			return item
		}
	}
	t.Fatalf("%s did not list %s: %s", path, id, rec.Body.String())
	return nil
}

func assertFields(t *testing.T, label string, got map[string]any, want map[string]any) {
	t.Helper()
	for k, v := range want {
		gotJSON, _ := json.Marshal(got[k])
		wantJSON, _ := json.Marshal(v)
		if string(gotJSON) != string(wantJSON) {
			t.Errorf("%s: %s = %s, want %s", label, k, gotJSON, wantJSON)
		}
	}
}

// Every writable field must exist in the collection schema; otherwise
// PocketBase quietly drops it and the write API pretends to accept it.
func TestAPIV1WriteFieldsExistInSchema(t *testing.T) {
	f := newSecurityFixture(t)
	for collection, fields := range apiV1WriteFields {
		c, err := f.app.FindCollectionByNameOrId(collection)
		if err != nil {
			t.Fatal(err)
		}
		for _, field := range fields {
			if c.Fields.GetByName(field) == nil {
				t.Errorf("%s.%s is writable but not in the schema", collection, field)
			}
		}
	}
}

// Mapper keys must be schema fields, system fields or a documented alias.
func TestAPIV1MappersUseSchemaFields(t *testing.T) {
	f := newSecurityFixture(t)
	aliases := map[string]bool{"skills.level": true, "projects.url": true, "experience.is_current": true}
	mappers := map[string]func(*core.Record) map[string]any{
		"posts": mapBasicPost, "projects": mapBasicProject, "skills": mapBasicSkill,
		"experience": mapBasicExperience, "profile": mapProfileWrite,
	}
	for collection, mapper := range mappers {
		c, err := f.app.FindCollectionByNameOrId(collection)
		if err != nil {
			t.Fatal(err)
		}
		for field := range mapper(core.NewRecord(c)) {
			if field == "id" || c.Fields.GetByName(field) != nil || aliases[collection+"."+field] {
				continue
			}
			t.Errorf("%s mapper returns %q, which is not a schema field or alias", collection, field)
		}
		// And every writable field is returned, so writes round-trip.
		out := mapper(core.NewRecord(c))
		for _, field := range apiV1WriteFields[collection] {
			if _, ok := out[field]; !ok {
				t.Errorf("%s.%s is writable but the response doesn't return it", collection, field)
			}
		}
	}
}

func TestAPIV1WritesRoundTripThroughReads(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterAPIKeyHooks(f.pb, f.crypto, f.rl)
	RegisterAPIV1WriteHooks(f.pb, f.crypto, f.rl)
	h := f.serve(t)
	key, _ := f.createTestAPIKey(t, "Sync",
		"read:skills", "write:skills", "read:projects", "write:projects",
		"read:experience", "write:experience", "read:posts", "write:posts")
	auth := map[string]string{"Authorization": "Bearer " + key}
	write := func(method, url, body string, want int) map[string]any {
		t.Helper()
		rec := do(t, h, testRequest{method: method, url: url, body: body, headers: auth})
		assertStatus(t, rec, want)
		return decodeBody(t, rec)
	}

	// Skills: proficiency is the field; level is its alias.
	skill := write(http.MethodPost, "/api/v1/skills",
		`{"name":"Go","category":"Languages","proficiency":"expert","visibility":"public","sort_order":2}`, http.StatusCreated)
	wantSkill := map[string]any{"name": "Go", "category": "Languages", "proficiency": "expert", "level": "expert",
		"visibility": "public", "sort_order": 2}
	assertFields(t, "skill write", skill, wantSkill)
	assertFields(t, "skill read", readOne(t, h, key, "/api/v1/skills", skill["id"].(string)), wantSkill)

	// Projects: summary, links etc. are real fields; url comes from links.
	project := write(http.MethodPost, "/api/v1/projects", `{"title":"Greenhouse monitor","slug":"greenhouse",
		"summary":"Sensors for tomatoes","description":"<p>Full story</p>","tech_stack":["Go","SQLite"],
		"links":[{"type":"demo","url":""},{"type":"github","url":"https://github.com/example/greenhouse"}],
		"categories":["iot"],"visibility":"public","is_draft":false,"is_featured":true,"sort_order":1}`, http.StatusCreated)
	wantProject := map[string]any{"title": "Greenhouse monitor", "slug": "greenhouse", "summary": "Sensors for tomatoes",
		"description": "<p>Full story</p>", "tech_stack": []string{"Go", "SQLite"},
		"links": []map[string]string{{"type": "demo", "url": ""}, {"type": "github", "url": "https://github.com/example/greenhouse"}},
		"url":   "https://github.com/example/greenhouse", "categories": []string{"iot"},
		"is_featured": true, "sort_order": 1, "visibility": "public", "is_draft": false}
	assertFields(t, "project write", project, wantProject)
	assertFields(t, "project read", readOne(t, h, key, "/api/v1/projects", project["id"].(string)), wantProject)
	if _, ok := project["status"]; ok {
		t.Error("project still returns status, which isn't a schema field")
	}

	// Experience: is_current follows end_date.
	exp := write(http.MethodPost, "/api/v1/experience", `{"company":"Playground Co","title":"Swing tester",
		"start_date":"2024-01-01 00:00:00.000Z","bullets":["Tested swings"],"skills":["Balance"],
		"visibility":"public","sort_order":0}`, http.StatusCreated)
	wantExp := map[string]any{"company": "Playground Co", "title": "Swing tester", "end_date": "",
		"is_current": true, "bullets": []string{"Tested swings"}, "skills": []string{"Balance"}}
	assertFields(t, "experience write", exp, wantExp)
	assertFields(t, "experience read", readOne(t, h, key, "/api/v1/experience", exp["id"].(string)), wantExp)
	exp = write(http.MethodPatch, "/api/v1/experience/"+exp["id"].(string),
		`{"end_date":"2025-06-30 00:00:00.000Z"}`, http.StatusOK)
	if exp["is_current"] != false {
		t.Fatalf("is_current = %v after setting end_date, want false", exp["is_current"])
	}

	// Posts: content and tags round-trip.
	post := write(http.MethodPost, "/api/v1/posts", `{"title":"Puppets","slug":"puppets","excerpt":"Short",
		"content":"<p>Long</p>","tags":["fun"],"visibility":"public","is_draft":false}`, http.StatusCreated)
	wantPost := map[string]any{"title": "Puppets", "excerpt": "Short", "content": "<p>Long</p>", "tags": []string{"fun"}}
	assertFields(t, "post write", post, wantPost)
	assertFields(t, "post read", readOne(t, h, key, "/api/v1/posts", post["id"].(string)), wantPost)
}

func TestAPIV1ReadsWithholdPaidContent(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterAPIKeyHooks(f.pb, f.crypto, f.rl)
	h := f.serve(t)
	key, _ := f.createTestAPIKey(t, "Reader", "read:posts", "read:projects")

	post := f.create(t, "posts", map[string]any{"title": "Paid post", "excerpt": "Teaser",
		"content": "<p>Paid words</p>", "visibility": "public", "access_tier": "paid", "price": 5})
	project := f.create(t, "projects", map[string]any{"title": "Paid project", "summary": "Teaser",
		"description": "<p>Paid words</p>", "experience": map[string]any{"blocks": []any{"x"}},
		"visibility": "public", "access_tier": "paid", "price": 5})

	gotPost := readOne(t, h, key, "/api/v1/posts", post.Id)
	assertFields(t, "paid post", gotPost, map[string]any{"excerpt": "Teaser", "content": "", "access_tier": "paid"})
	gotProject := readOne(t, h, key, "/api/v1/projects", project.Id)
	assertFields(t, "paid project", gotProject, map[string]any{"summary": "Teaser", "description": "", "experience": nil})
}
