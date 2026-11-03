package hooks

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// createTestAPIKey stores a key the way POST /api/admin/api-keys does and
// returns the full secret plus its record.
func (f *securityFixture) createTestAPIKey(t *testing.T, label string, scopes ...string) (string, *core.Record) {
	t.Helper()
	fullKey, err := generateAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	key := f.create(t, "api_keys", map[string]any{
		"label":      label,
		"key_prefix": extractKeyPrefix(fullKey),
		"key_hash":   hashAPIKey(f.crypto, fullKey),
		"scopes":     scopes,
		"is_active":  true,
	})
	// Mark the key as recently used so the middleware skips its background
	// last_used_at write, which could otherwise outlive the test database.
	lastUsedMu.Lock()
	lastUsedCache[key.Id] = time.Now()
	lastUsedMu.Unlock()
	return fullKey, key
}

func auditLogs(t *testing.T, f *securityFixture) []*core.Record {
	t.Helper()
	records, err := f.app.FindRecordsByFilter("audit_logs", "id != ''", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	return records
}

func assertAuditCount(t *testing.T, f *securityFixture, want int) {
	t.Helper()
	if got := len(auditLogs(t, f)); got != want {
		t.Fatalf("audit log count = %d, want %d", got, want)
	}
}

// assertAPIKeyAudit finds the single audit record for this write, checks it,
// and makes sure neither the secret nor its hash was recorded.
func assertAPIKeyAudit(t *testing.T, f *securityFixture, fullKey string, key *core.Record, action, resourceType, resourceID, label string) {
	t.Helper()
	logs, err := f.app.FindRecordsByFilter(
		"audit_logs",
		"action = {:action} && resource_type = {:type} && resource_id = {:id}",
		"", 0, 0,
		map[string]any{"action": action, "type": resourceType, "id": resourceID},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("found %d %s audit logs for %s/%s, want 1", len(logs), action, resourceType, resourceID)
	}
	entry := logs[0]

	if got := entry.GetString("status"); got != "success" {
		t.Fatalf("status = %q, want success", got)
	}
	prefix := key.GetString("key_prefix")
	if got, want := entry.GetString("user_email"), "API key: "+key.GetString("label")+" ("+prefix+")"; got != want {
		t.Fatalf("user_email = %q, want %q", got, want)
	}

	var metadata map[string]any
	if err := entry.UnmarshalJSONField("metadata", &metadata); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"via":            "api_key",
		"api_key_id":     key.Id,
		"api_key_label":  key.GetString("label"),
		"api_key_prefix": prefix,
		"label":          label,
	}
	for k, v := range want {
		if metadata[k] != v {
			t.Fatalf("metadata[%q] = %v, want %v (metadata: %v)", k, metadata[k], v, metadata)
		}
	}

	raw, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{fullKey, fullKey[keyDisplayPrefixLen:], key.GetString("key_hash")} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("audit log contains API key secret material: %s", raw)
		}
	}
}

func TestAPIV1WritesAreAudited(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterAPIV1WriteHooks(f.pb, f.crypto, f.rl)
	h := f.serve(t)

	fullKey, key := f.createTestAPIKey(t, "Zapier", "write:skills", "write:profile")
	auth := map[string]string{"Authorization": "Bearer " + fullKey}
	baseline := len(auditLogs(t, f))

	// Create. "id" is not a writable field, so the caller can't choose it.
	rec := do(t, h, testRequest{
		method:  http.MethodPost,
		url:     "/api/v1/skills",
		body:    `{"id":"chosenbycaller1","name":"Go","category":"Languages"}`,
		headers: auth,
	})
	assertStatus(t, rec, http.StatusCreated)
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.ID == "chosenbycaller1" {
		t.Fatalf("created id = %q; id should be server-assigned", created.ID)
	}
	assertAuditCount(t, f, baseline+1)
	assertAPIKeyAudit(t, f, fullKey, key, "create", "skills", created.ID, "Go (Languages)")

	// Update.
	rec = do(t, h, testRequest{
		method:  http.MethodPatch,
		url:     "/api/v1/skills/" + created.ID,
		body:    `{"name":"Golang"}`,
		headers: auth,
	})
	assertStatus(t, rec, http.StatusOK)
	assertAuditCount(t, f, baseline+2)
	assertAPIKeyAudit(t, f, fullKey, key, "update", "skills", created.ID, "Golang (Languages)")

	// Delete. The label comes from the record as it was before deletion.
	rec = do(t, h, testRequest{
		method:  http.MethodDelete,
		url:     "/api/v1/skills/" + created.ID,
		headers: auth,
	})
	assertStatus(t, rec, http.StatusNoContent)
	assertAuditCount(t, f, baseline+3)
	assertAPIKeyAudit(t, f, fullKey, key, "delete", "skills", created.ID, "Golang (Languages)")

	// Profile singleton update.
	profiles, err := f.app.FindRecordsByFilter("profile", "id != ''", "", 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	var profile *core.Record
	if len(profiles) > 0 {
		profile = profiles[0]
	} else {
		profile = f.create(t, "profile", map[string]any{"name": "Before"})
	}
	baseline = len(auditLogs(t, f))
	rec = do(t, h, testRequest{
		method:  http.MethodPatch,
		url:     "/api/v1/profile",
		body:    `{"name":"Jordan Example"}`,
		headers: auth,
	})
	assertStatus(t, rec, http.StatusOK)
	assertAuditCount(t, f, baseline+1)
	assertAPIKeyAudit(t, f, fullKey, key, "update", "profile", profile.Id, "Jordan Example")
}

func TestAPIV1FailedWritesAreNotAuditedAsSuccess(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterAPIV1WriteHooks(f.pb, f.crypto, f.rl)
	h := f.serve(t)

	fullKey, _ := f.createTestAPIKey(t, "Limited", "write:skills", "write:profile")
	auth := map[string]string{"Authorization": "Bearer " + fullKey}
	existing := f.create(t, "skills", map[string]any{"name": "Existing"})

	profiles, err := f.app.FindRecordsByFilter("profile", "id != ''", "", 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) == 0 {
		f.create(t, "profile", map[string]any{"name": "Before"})
	}

	baseline := len(auditLogs(t, f))

	cases := []struct {
		name string
		req  testRequest
		want int
	}{
		{"missing key", testRequest{method: http.MethodPost, url: "/api/v1/skills", body: `{"name":"Go"}`}, http.StatusUnauthorized},
		{"wrong key", testRequest{method: http.MethodPost, url: "/api/v1/skills", body: `{"name":"Go"}`,
			headers: map[string]string{"Authorization": "Bearer " + APIKeyPrefix + strings.Repeat("0", 64)}}, http.StatusUnauthorized},
		{"missing scope", testRequest{method: http.MethodPost, url: "/api/v1/posts", body: `{"title":"Hi"}`, headers: auth}, http.StatusForbidden},
		{"invalid JSON", testRequest{method: http.MethodPost, url: "/api/v1/skills", body: `{`, headers: auth}, http.StatusBadRequest},
		{"validation failure on create", testRequest{method: http.MethodPost, url: "/api/v1/skills", body: `{"category":"No name"}`, headers: auth}, http.StatusBadRequest},
		{"validation failure on update", testRequest{method: http.MethodPatch, url: "/api/v1/skills/" + existing.Id, body: `{"name":""}`, headers: auth}, http.StatusBadRequest},
		{"update missing record", testRequest{method: http.MethodPatch, url: "/api/v1/skills/doesnotexist123", body: `{"name":"Go"}`, headers: auth}, http.StatusNotFound},
		{"delete missing record", testRequest{method: http.MethodDelete, url: "/api/v1/skills/doesnotexist123", headers: auth}, http.StatusNotFound},
		{"profile validation failure", testRequest{method: http.MethodPatch, url: "/api/v1/profile", body: `{"name":""}`, headers: auth}, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, tc.req)
			assertStatus(t, rec, tc.want)
			assertAuditCount(t, f, baseline)
		})
	}

	// The failed update left the record alone.
	reloaded, err := f.app.FindRecordById("skills", existing.Id)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.GetString("name") != "Existing" {
		t.Fatalf("failed update changed the record: %q", reloaded.GetString("name"))
	}
}
