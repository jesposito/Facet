package hooks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// fakeAI answers like an OpenAI-compatible provider and keeps each prompt.
type fakeAI struct {
	mu      sync.Mutex
	prompts []string
	server  *httptest.Server
}

func newFakeAI(t *testing.T) *fakeAI {
	t.Helper()
	ai := &fakeAI{}
	ai.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		ai.mu.Lock()
		if len(req.Messages) > 0 {
			ai.prompts = append(ai.prompts, req.Messages[0].Content)
		}
		ai.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"# Ada Example\n\nBuilds things."}}]}`))
	}))
	t.Cleanup(ai.server.Close)
	return ai
}

func (a *fakeAI) calls() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.prompts...)
}

func (f *securityFixture) aiProvider(t *testing.T, name, baseURL string, isDefault bool) *core.Record {
	t.Helper()
	key, err := f.crypto.Encrypt("test-key")
	if err != nil {
		t.Fatal(err)
	}
	return f.create(t, "ai_providers", map[string]any{
		"name": name, "type": "custom", "api_key_encrypted": key, "base_url": baseURL,
		"model": "test-model", "is_active": true, "is_default": isDefault,
	})
}

type resumeFixture struct {
	*securityFixture
	handler  http.Handler
	ai       *fakeAI
	views    map[string]*core.Record
	password string
	other    *core.Record
}

func newResumeFixture(t *testing.T) *resumeFixture {
	t.Helper()
	f := newSecurityFixture(t)
	RegisterClientIPMiddleware(f.pb)
	RegisterResumeHooks(f.pb, f.crypto, f.share)

	ai := newFakeAI(t)
	f.aiProvider(t, "Default", ai.server.URL, true)
	other := newFakeAI(t)
	otherProvider := f.aiProvider(t, "Other", other.server.URL, false)

	f.create(t, "profile", map[string]any{"name": "Ada Example", "headline": "Engineer", "visibility": "public", "contact_email": "ada@example.com"})
	testimonial := f.create(t, "testimonials", map[string]any{
		"content": "Ada shipped it.", "author_name": "Grace", "status": "approved", "visibility": "public",
		"verification_method": "email", "verification_identifier": "referee@private.example",
	})
	post := f.create(t, "posts", map[string]any{
		"title": "Paid deep dive", "slug": "paid-deep-dive", "visibility": "public",
		"access_tier": "paid", "price": 500,
		"content": "FREE OPENING\n\nPAID SECOND PARAGRAPH",
	})
	sections := []map[string]any{
		{"section": "testimonials", "enabled": true, "items": []string{testimonial.Id}},
		{"section": "posts", "enabled": true, "items": []string{post.Id}},
	}
	view := func(slug, visibility string, active bool) *core.Record {
		return f.create(t, "views", map[string]any{"name": slug, "slug": slug, "visibility": visibility, "is_active": active, "sections": sections})
	}
	views := map[string]*core.Record{
		"public":   view("public-view", "public", true),
		"password": view("password-view", "password", true),
		"unlisted": view("unlisted-view", "unlisted", true),
		"private":  view("private-view", "private", true),
		"inactive": view("inactive-view", "public", false),
	}
	f.create(t, "share_tokens", map[string]any{"view_id": views["unlisted"].Id, "token_hash": f.share.HMACToken("valid-share"), "is_active": true, "max_uses": 1, "use_count": 0})
	passwordJWT, _, err := f.crypto.GenerateViewAccessJWT(views["password"].Id, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	return &resumeFixture{securityFixture: f, handler: f.serve(t), ai: ai, views: views, password: passwordJWT, other: otherProvider}
}

func generate(t *testing.T, h http.Handler, slug string, headers map[string]string, body string) *httptest.ResponseRecorder {
	t.Helper()
	if body == "" {
		body = `{"format":"docx"}`
	}
	return do(t, h, testRequest{method: http.MethodPost, url: "/api/view/" + slug + "/generate", body: body, headers: headers})
}

func TestResumeGenerationFollowsViewAccess(t *testing.T) {
	r := newResumeFixture(t)
	_, adminToken := r.createAdmin(t, "owner@example.com", "a-strong-password", nil)

	for _, tc := range []struct {
		name    string
		slug    string
		headers map[string]string
		want    int
	}{
		{"password view without its password", "password-view", nil, http.StatusUnauthorized},
		{"unlisted view without a share token", "unlisted-view", nil, http.StatusUnauthorized},
		{"unlisted view with a wrong share token", "unlisted-view", map[string]string{"X-Share-Token": "guess"}, http.StatusUnauthorized},
		{"private view", "private-view", nil, http.StatusNotFound},
		{"inactive view", "inactive-view", nil, http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertStatus(t, generate(t, r.handler, tc.slug, tc.headers, ""), tc.want)
		})
	}
	if n := len(r.ai.calls()); n != 0 {
		t.Fatalf("refused requests still reached the AI provider %d times", n)
	}

	if _, err := exec.LookPath("pandoc"); err != nil {
		t.Skip("pandoc not installed; the successful generations need it")
	}
	for _, tc := range []struct {
		name    string
		slug    string
		headers map[string]string
	}{
		{"public view", "public-view", nil},
		{"password view with its token", "password-view", map[string]string{"X-Password-Token": r.password}},
		{"unlisted view with its share token", "unlisted-view", map[string]string{"X-Share-Token": "valid-share"}},
		{"owner on a private view", "private-view", map[string]string{"Authorization": adminToken}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := generate(t, r.handler, tc.slug, tc.headers, "")
			assertStatus(t, rec, http.StatusOK)
			assertContains(t, rec, `"status":"completed"`)
		})
	}

	// Generating from an unlisted view doesn't spend the visitor's token,
	// which here had a single use.
	token, err := r.app.FindFirstRecordByFilter("share_tokens", "view_id = {:v}", map[string]any{"v": r.views["unlisted"].Id})
	if err != nil {
		t.Fatal(err)
	}
	if token.GetInt("use_count") != 0 {
		t.Fatalf("generation used up the share token: use_count=%d", token.GetInt("use_count"))
	}
}

// The AI prompt only holds what the visitor could read on the page.
func TestResumePromptUsesThePublicProjection(t *testing.T) {
	if _, err := exec.LookPath("pandoc"); err != nil {
		t.Skip("pandoc not installed")
	}
	r := newResumeFixture(t)

	rec := generate(t, r.handler, "public-view", nil, `{"format":"docx","provider_id":"`+r.other.Id+`"}`)
	assertStatus(t, rec, http.StatusOK)

	prompts := r.ai.calls()
	if len(prompts) != 1 {
		t.Fatalf("default provider got %d calls; visitors must not pick another provider", len(prompts))
	}
	prompt := prompts[0]
	for _, want := range []string{"Ada Example", "Ada shipped it.", "FREE OPENING"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt is missing public content %q", want)
		}
	}
	for _, secret := range []string{"referee@private.example", "PAID SECOND PARAGRAPH"} {
		if strings.Contains(prompt, secret) {
			t.Errorf("prompt leaked %q:\n%s", secret, prompt)
		}
	}
}

func TestResumeLimitFollowsTheClientBehindCaddy(t *testing.T) {
	r := newResumeFixture(t)
	viaCaddy := func(port int, client string) testRequest {
		return testRequest{
			method:     http.MethodPost,
			url:        "/api/view/no-such-view/generate",
			body:       `{"format":"docx"}`,
			remoteAddr: fmt.Sprintf("127.0.0.1:%d", 40000+port),
			headers:    map[string]string{"X-Forwarded-For": client},
		}
	}
	// Each request arrives on a new connection, as browsers often do.
	for i := 0; i < 5; i++ {
		assertStatus(t, do(t, r.handler, viaCaddy(i, "203.0.113.40")), http.StatusNotFound)
	}
	assertStatus(t, do(t, r.handler, viaCaddy(5, "203.0.113.40")), http.StatusTooManyRequests)
	assertStatus(t, do(t, r.handler, viaCaddy(6, "203.0.113.41")), http.StatusNotFound)
}
