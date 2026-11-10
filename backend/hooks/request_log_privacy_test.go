package hooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/logger"
)

func TestRedactLogURL(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"/posts/hello?from=portfolio&page=2", "/posts/hello?from=portfolio&page=2"},
		{"/api/testimonials/requests?status=pending", "/api/testimonials/requests?status=pending"},
		{"/s/SENTINEL?keep=yes", "/s/[REDACTED]?keep=yes"},
		{"/s/SENTINEL/__data.json", "/s/[REDACTED]"},
		{"/api/downloads/SENTINEL", "/api/downloads/[REDACTED]"},
		{"/api/content/access/SENTINEL", "/api/content/access/[REDACTED]"},
		{"/api/public/newsletter/confirm/SENTINEL", "/api/public/newsletter/confirm/[REDACTED]"},
		{"/api/public/newsletter/open/send1/SENTINEL", "/api/public/newsletter/open/[REDACTED]"},
		{"/api/public/newsletter/click/send1/SENTINEL/encoded-url", "/api/public/newsletter/click/[REDACTED]"},
		{"/api/public/unsubscribe/one-click/SENTINEL", "/api/public/unsubscribe/[REDACTED]"},
		{"/unsubscribe/SENTINEL", "/unsubscribe/[REDACTED]"},
		{"/api/testimonials/request/SENTINEL", "/api/testimonials/request/[REDACTED]"},
		{"/api/testimonials/verify/email/SENTINEL", "/api/testimonials/verify/email/[REDACTED]"},
		{"/testimonial/verify/SENTINEL", "/testimonial/[REDACTED]"},
		{"/pitch?t=SENTINEL&keep=ok&t=SENTINEL", "/pitch?t=[REDACTED]&keep=ok&t=[REDACTED]"},
		{"/api/files/c/r/a.pdf?thumb=100x100&token=SENTINEL", "/api/files/c/r/a.pdf?thumb=100x100&token=[REDACTED]"},
		{"/api/courses/intro/enroll/verify?token=SENTINEL", "/api/courses/intro/enroll/verify?token=[REDACTED]"},
		{"/api/oauth2-redirect?code=SENTINEL&state=SENTINEL", "/api/oauth2-redirect?code=[REDACTED]&state=[REDACTED]"},
		{"/checkout/return?type=post&session_id=SENTINEL", "/checkout/return?type=post&session_id=[REDACTED]"},
		{"/%73/SENTINEL?keep=ok", "[REDACTED_PATH]?keep=ok"},
		{"/s%2FSENTINEL", "[REDACTED_PATH]"},
		{"/a/../s/SENTINEL", "[REDACTED_PATH]"},
		{"//s/SENTINEL", "[REDACTED_PATH]"},
		{"/pitch?%74=SENTINEL&to%6Ben=SENTINEL", "/pitch?%74=[REDACTED]&to%6Ben=[REDACTED]"},
		{"https://example.test/s/SENTINEL?keep=yes", "https://example.test/s/[REDACTED]?keep=yes"},
		{"https://name:password@example.test/posts/hello?from=portfolio", "https://example.test/posts/hello?from=portfolio"},
		{"/s/%zzSENTINEL", "[REDACTED_URL]"},
		{"/s/[REDACTED]?t=[REDACTED]", "/s/[REDACTED]?t=[REDACTED]"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := redactLogURL(tc.in); got != tc.want {
				t.Errorf("redacted URL does not match expected %q", tc.want)
			}
		})
	}
}

func TestBoundedLogText(t *testing.T) {
	for _, limit := range []int{50, 2000, 3000} {
		value := boundedLogText(strings.Repeat("é", limit), limit)
		if len(value) > limit || !strings.HasSuffix(value, "...") || !json.Valid([]byte(`"`+value+`"`)) {
			t.Errorf("bounded log field at %d bytes must end on a valid character", limit)
		}
	}
}

func TestCheckoutSessionLogRef(t *testing.T) {
	const raw = "cs_test_SENSITIVE_CHECKOUT_SESSION"
	ref := checkoutSessionLogRef(raw)
	if len(ref) != 16 || strings.Contains(ref, raw) {
		t.Fatal("checkout reference must be a bounded digest")
	}
	if ref != checkoutSessionLogRef(raw) || ref == checkoutSessionLogRef(raw+"other") {
		t.Fatal("checkout reference must be stable and distinguish sessions")
	}
	if checkoutSessionLogRef("") != "" {
		t.Fatal("an empty session must have an empty reference")
	}
}

func TestRedactLogMetadataDoesNotChangeOriginalValues(t *testing.T) {
	meta := map[string]any{
		"headers": http.Header{"X-Share-Token": {"SENTINEL"}, "Referer": {"https://example.test/s/SENTINEL"}},
		"items":   []any{map[string]string{"token": "SENTINEL", "action": "keep"}},
	}
	safe := redactLogValue("", meta)
	data, err := json.Marshal(safe)
	if err != nil || strings.Contains(string(data), "SENTINEL") || !strings.Contains(string(data), "keep") {
		t.Fatal("nested metadata lost safe values or kept bearers")
	}
	if meta["headers"].(http.Header).Get("X-Share-Token") != "SENTINEL" ||
		meta["items"].([]any)[0].(map[string]string)["token"] != "SENTINEL" {
		t.Fatal("redaction mutated the application's metadata")
	}
}

func TestRequestLogPrivacyPreservesRoutesAndSafeDiagnostics(t *testing.T) {
	f := newSecurityFixture(t)
	f.app.Settings().Logs.MaxDays = 7
	RegisterRequestLogPrivacy(f.pb)
	var reached int
	f.pb.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/api/downloads/{token}", func(e *core.RequestEvent) error {
			reached++
			if e.Request.PathValue("token") != "SENTINEL" || e.Request.URL.Query().Get("token") != "SENTINEL" ||
				e.Request.Header.Get("X-Share-Token") != "SENTINEL" || !strings.Contains(e.Request.Referer(), "SENTINEL") {
				t.Error("request logging changed the original route request")
			}
			return e.NoContent(http.StatusNoContent)
		})
		se.Router.GET("/privacy/benign", func(e *core.RequestEvent) error {
			e.Set(apis.RequestEventKeyLogMeta, map[string]any{"action": "keep", "token": "SENTINEL"})
			return e.NoContent(http.StatusOK)
		})
		se.Router.GET("/privacy/skip", func(e *core.RequestEvent) error {
			return e.NoContent(http.StatusOK)
		}).Bind(apis.SkipSuccessActivityLog())
		se.Router.GET("/privacy/error", func(e *core.RequestEvent) error {
			return e.BadRequestError("Failed to open /s/SENTINEL", map[string]any{"token": "SENTINEL"})
		}).Bind(apis.SkipSuccessActivityLog())
		return se.Next()
	})
	h := f.serve(t)
	for _, target := range []string{
		"/api/downloads/SENTINEL?token=SENTINEL&keep=yes",
		"/privacy/benign?tag=go&page=2",
		"/privacy/skip",
		"/privacy/error?token=SENTINEL",
	} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		req.Header.Set("X-Share-Token", "SENTINEL")
		req.Header.Set("Referer", "https://example.test/s/SENTINEL?token=SENTINEL")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
	}
	if reached != 1 {
		t.Fatal("bearer route did not receive the request exactly once")
	}
	// Also exercise structured application logs, which share the log database.
	f.app.Logger().Info("Opening /api/content/access/SENTINEL", "session_id", "SENTINEL", "headers", map[string]any{
		"X-Share-Token": "SENTINEL", "Referer": "https://example.test/s/SENTINEL",
	})
	flushRequestPrivacyLogs(t, f)
	rows := readRequestPrivacyLogs(t, f)
	var requestRows int
	for _, row := range rows {
		if strings.Contains(row.Message+row.Data, "SENTINEL") {
			t.Fatal("a bearer survived in the persisted request or application logs")
		}
		var data map[string]any
		if err := json.Unmarshal([]byte(row.Data), &data); err != nil {
			t.Fatal(err)
		}
		if data["type"] != "request" {
			continue
		}
		requestRows++
		if data["method"] != http.MethodGet {
			t.Error("request method was lost")
		}
		switch data["url"] {
		case "/api/downloads/[REDACTED]?token=[REDACTED]&keep=yes":
			if data["status"] != float64(http.StatusNoContent) {
				t.Error("bearer route status was lost")
			}
		case "/privacy/benign?tag=go&page=2":
			if data["status"] != float64(http.StatusOK) {
				t.Error("benign route status was lost")
			}
			meta, _ := data["meta"].(map[string]any)
			if meta["action"] != "keep" || meta["token"] != "[REDACTED]" {
				t.Error("safe request metadata was lost or a named token was not redacted")
			}
		case "/privacy/error?token=[REDACTED]":
			if data["status"] != float64(http.StatusBadRequest) || data["error"] != "Failed to open /s/[REDACTED]" {
				t.Error("error request diagnostics were lost")
			}
		default:
			t.Error("unexpected or duplicate activity log")
		}
	}
	if requestRows != 3 {
		t.Errorf("got %d request logs, expected 3 with the successful skip excluded", requestRows)
	}
}

func TestRequestLogPrivacyHonorsDisabledRetention(t *testing.T) {
	f := newSecurityFixture(t)
	f.app.Settings().Logs.MaxDays = 0
	RegisterRequestLogPrivacy(f.pb)
	f.pb.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/privacy/disabled", func(e *core.RequestEvent) error {
			return e.NoContent(http.StatusOK)
		})
		return se.Next()
	})
	h := f.serve(t)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/privacy/disabled?token=SENTINEL", nil))
	flushRequestPrivacyLogs(t, f)
	// Unrelated app logs (e.g. a notify-dir watcher failure on a host out of inotify instances) may exist.
	for _, row := range readRequestPrivacyLogs(t, f) {
		if strings.Contains(row.Message+row.Data, "/privacy/disabled") {
			t.Fatal("request logging must respect disabled retention")
		}
	}
}

func flushRequestPrivacyLogs(t *testing.T, f *securityFixture) {
	t.Helper()
	handler, ok := f.app.Logger().Handler().(*logger.BatchHandler)
	if !ok {
		t.Fatal("PocketBase logger is no longer a batch handler")
	}
	ctx := context.WithValue(context.Background(), logger.BlockKey, true)
	if err := handler.WriteAll(ctx); err != nil {
		t.Fatal(err)
	}
}

func readRequestPrivacyLogs(t *testing.T, f *securityFixture) []struct{ Message, Data string } {
	t.Helper()
	var rows []struct{ Message, Data string }
	if err := f.app.AuxDB().NewQuery("SELECT message, data FROM _logs").All(&rows); err != nil {
		t.Fatal(err)
	}
	return rows
}
