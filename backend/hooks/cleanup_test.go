package hooks

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type cleanupResponse struct {
	Status             string   `json:"status"`
	ExpiredShareTokens int      `json:"expired_share_tokens"`
	FailedExports      int      `json:"failed_exports"`
	Total              int      `json:"total"`
	FailedSteps        []string `json:"failed_steps"`
}

func runCleanupRequest(t *testing.T, f *securityFixture, h http.Handler, token string) (*cleanupResponse, int) {
	t.Helper()
	rec := do(t, h, testRequest{
		method:  http.MethodPost,
		url:     "/api/admin/cleanup-stale-data",
		headers: map[string]string{"Authorization": token},
	})
	var body cleanupResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return &body, rec.Code
}

func (f *securityFixture) expiredShareToken(t *testing.T, raw string) *core.Record {
	t.Helper()
	view := f.create(t, "views", map[string]any{"name": "View " + raw, "slug": raw, "visibility": "unlisted", "is_active": true})
	return f.create(t, "share_tokens", map[string]any{
		"view_id":    view.Id,
		"token_hash": f.share.HMACToken(raw),
		"is_active":  true,
		"expires_at": time.Now().Add(-time.Hour),
	})
}

func TestCleanupEndpointReportsSuccess(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterCleanupHooks(f.pb)
	h := f.serve(t)
	_, token := f.createAdmin(t, "admin@example.com", "admin-password-123", nil)

	f.expiredShareToken(t, "old-share")

	body, code := runCleanupRequest(t, f, h, token)
	if code != http.StatusOK || body.Status != "ok" {
		t.Fatalf("status = %d %q, want 200 ok", code, body.Status)
	}
	if body.ExpiredShareTokens != 1 || body.Total != 1 {
		t.Fatalf("counts = %+v, want one expired share token", body)
	}
	if len(body.FailedSteps) != 0 {
		t.Fatalf("failed_steps = %v, want none", body.FailedSteps)
	}

	rec := do(t, h, testRequest{method: http.MethodPost, url: "/api/admin/cleanup-stale-data"})
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestCleanupEndpointReportsDatabaseFailure(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterCleanupHooks(f.pb)
	h := f.serve(t)
	_, token := f.createAdmin(t, "admin@example.com", "admin-password-123", nil)

	f.expiredShareToken(t, "deletable-share")
	stuck := f.expiredShareToken(t, "stuck-share")
	f.app.OnRecordDelete("share_tokens").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.Id == stuck.Id {
			return errors.New("injected delete failure")
		}
		return e.Next()
	})
	// A missing table makes both view_exports queries fail at the database.
	if _, err := f.app.DB().NewQuery("DROP TABLE view_exports").Execute(); err != nil {
		t.Fatal(err)
	}

	body, code := runCleanupRequest(t, f, h, token)
	if code != http.StatusInternalServerError || body.Status != "error" {
		t.Fatalf("status = %d %q, want 500 error", code, body.Status)
	}
	// The token that could be deleted still counts; the stuck one doesn't.
	if body.ExpiredShareTokens != 1 || body.FailedExports != 0 || body.Total != 1 {
		t.Fatalf("counts = %+v, want only the one deleted share token", body)
	}
	want := map[string]bool{"expired_share_tokens": true, "failed_exports": true, "stuck_exports": true}
	if len(body.FailedSteps) != len(want) {
		t.Fatalf("failed_steps = %v, want %v", body.FailedSteps, want)
	}
	for _, step := range body.FailedSteps {
		if !want[step] {
			t.Fatalf("unexpected failed step %q in %v", step, body.FailedSteps)
		}
	}
	if _, err := f.app.FindRecordById("share_tokens", stuck.Id); err != nil {
		t.Fatalf("share token whose delete failed is gone: %v", err)
	}
}
