package hooks

import (
	"fmt"
	"net/http"
	"testing"
)

// signInAttempts sends n failed sign-ins and returns the last status.
func signInAttempts(t *testing.T, h http.Handler, n int, remoteAddr string, headers map[string]string) int {
	t.Helper()
	body := `{"identity":"owner@example.com","password":"wrong-password"}`
	last := 0
	for i := 0; i < n; i++ {
		last = do(t, h, testRequest{
			method:     http.MethodPost,
			url:        "/api/collections/users/auth-with-password",
			body:       body,
			remoteAddr: remoteAddr,
			headers:    headers,
		}).Code
	}
	return last
}

// Behind Caddy every request reaches PocketBase from 127.0.0.1. The sign-in
// limit has to follow the client address Caddy reports, or one visitor's
// failed attempts would lock everyone out.
func TestSignInLimitSeparatesVisitorsBehindCaddy(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterClientIPMiddleware(f.pb)
	RegisterAuthRateLimits(f.pb)
	h := f.serve(t)

	viaCaddy := func(client string) map[string]string {
		return map[string]string{"X-Forwarded-For": client}
	}
	if code := signInAttempts(t, h, 6, "127.0.0.1:40001", viaCaddy("203.0.113.10")); code != http.StatusTooManyRequests {
		t.Fatalf("sixth attempt from one visitor returned %d, want 429", code)
	}
	if code := signInAttempts(t, h, 1, "127.0.0.1:40002", viaCaddy("203.0.113.11")); code == http.StatusTooManyRequests {
		t.Fatal("a second visitor behind Caddy was locked out by the first")
	}
}

// A visitor who reaches the backend directly can't buy fresh sign-in
// attempts by inventing proxy headers.
func TestSignInLimitIgnoresHeadersFromDirectClients(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterClientIPMiddleware(f.pb)
	RegisterAuthRateLimits(f.pb)
	h := f.serve(t)

	last := 0
	for i := 0; i < 6; i++ {
		last = signInAttempts(t, h, 1, "198.51.100.20:5000", map[string]string{
			"X-Forwarded-For":  fmt.Sprintf("203.0.113.%d", i),
			"CF-Connecting-IP": fmt.Sprintf("192.0.2.%d", i),
			"X-Real-IP":        fmt.Sprintf("192.0.2.%d", 100+i),
		})
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("sixth attempt with rotating headers returned %d, want 429", last)
	}
}

// The first wave couldn't confirm these rule labels against PocketBase's
// source. Each route has to answer 429 once its budget is spent.
func TestEverySignInRateLimitRuleMatchesItsRoute(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterAuthRateLimits(f.pb)
	h := f.serve(t)

	for i, tc := range []struct {
		url    string
		body   string
		budget int
	}{
		{"/api/collections/users/confirm-password-reset", `{"token":"nope","password":"x1234567","passwordConfirm":"x1234567"}`, 5},
		{"/api/collections/users/request-verification", `{"email":"owner@example.com"}`, 2},
		{"/api/collections/users/request-otp", `{"email":"owner@example.com"}`, 2},
	} {
		t.Run(tc.url, func(t *testing.T) {
			addr := fmt.Sprintf("203.0.113.%d:5000", 50+i)
			last := 0
			for n := 0; n <= tc.budget; n++ {
				last = do(t, h, testRequest{method: http.MethodPost, url: tc.url, body: tc.body, remoteAddr: addr}).Code
				if n < tc.budget && last == http.StatusTooManyRequests {
					t.Fatalf("request %d was limited before the budget of %d", n+1, tc.budget)
				}
			}
			if last != http.StatusTooManyRequests {
				t.Fatalf("request %d returned %d, want 429", tc.budget+1, last)
			}
		})
	}
}

// X-Internal lets the frontend's own server skip Facet's per-IP limits. Only
// a loopback request that didn't pass through Caddy may use it.
func TestInternalHeaderOnlyWorksForTheFrontend(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterClientIPMiddleware(f.pb)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	h := f.serve(t)

	limited := func(remoteAddr string, headers map[string]string) bool {
		for i := 0; i < 15; i++ {
			if do(t, h, testRequest{url: "/rss.xml", remoteAddr: remoteAddr, headers: headers}).Code == http.StatusTooManyRequests {
				return true
			}
		}
		return false
	}

	if limited("127.0.0.1:41000", map[string]string{"X-Internal": "true"}) {
		t.Fatal("the frontend's own requests were rate limited")
	}
	if !limited("127.0.0.1:41001", map[string]string{"X-Internal": "true", "X-Forwarded-For": "203.0.113.30"}) {
		t.Fatal("a visitor behind Caddy skipped the limit with X-Internal")
	}
	if !limited("198.51.100.31:5000", map[string]string{"X-Internal": "true"}) {
		t.Fatal("a direct visitor skipped the limit with X-Internal")
	}
}
