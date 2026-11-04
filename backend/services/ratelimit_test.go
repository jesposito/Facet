package services

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitService_Allow(t *testing.T) {
	svc := NewRateLimitService()

	// Create a mock request
	req := httptest.NewRequest("POST", "/api/password/check", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	// Strict tier: 5/min with burst of 3
	// First 3 requests should be allowed (burst)
	for i := 0; i < 3; i++ {
		allowed, _ := svc.Allow(req, "strict")
		if !allowed {
			t.Errorf("Request %d should be allowed (within burst)", i+1)
		}
	}

	// 4th request should be denied (burst exhausted, need to wait for refill)
	allowed, retryAfter := svc.Allow(req, "strict")
	if allowed {
		t.Error("4th request should be denied (burst exhausted)")
	}
	if retryAfter < 1 {
		t.Error("Retry-After should be at least 1 second")
	}
}

func TestRateLimitService_DifferentIPsNotShared(t *testing.T) {
	svc := NewRateLimitService()

	// Two different IPs
	req1 := httptest.NewRequest("POST", "/api/password/check", nil)
	req1.RemoteAddr = "192.168.1.100:12345"

	req2 := httptest.NewRequest("POST", "/api/password/check", nil)
	req2.RemoteAddr = "192.168.1.200:12345"

	// Exhaust burst for IP1
	for i := 0; i < 5; i++ {
		svc.Allow(req1, "strict")
	}

	// IP1 should be rate limited
	allowed1, _ := svc.Allow(req1, "strict")
	if allowed1 {
		t.Error("IP1 should be rate limited")
	}

	// IP2 should still have its full burst available
	allowed2, _ := svc.Allow(req2, "strict")
	if !allowed2 {
		t.Error("IP2 should not be rate limited (separate bucket)")
	}
}

func TestRateLimitService_DifferentTiers(t *testing.T) {
	svc := NewRateLimitService()

	req := httptest.NewRequest("POST", "/api/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	// Exhaust strict tier burst (3)
	for i := 0; i < 4; i++ {
		svc.Allow(req, "strict")
	}

	// Strict tier should be exhausted
	allowed, _ := svc.Allow(req, "strict")
	if allowed {
		t.Error("Strict tier should be exhausted")
	}

	// Normal tier should still be available (separate bucket)
	allowed, _ = svc.Allow(req, "normal")
	if !allowed {
		t.Error("Normal tier should still be available")
	}
}

func TestClientIP(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		trustProxy string
		want       string
	}{
		{
			name:       "direct client ignores every proxy header",
			remoteAddr: "198.51.100.7:4000",
			headers: map[string]string{
				"X-Forwarded-For":  "203.0.113.50",
				"CF-Connecting-IP": "203.0.113.100",
				"X-Real-IP":        "203.0.113.101",
			},
			trustProxy: "true",
			want:       "198.51.100.7",
		},
		{
			name:       "Caddy on loopback supplies the client",
			remoteAddr: "127.0.0.1:51234",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.50"},
			trustProxy: "true",
			want:       "203.0.113.50",
		},
		{
			name:       "only the entry the nearest proxy wrote counts",
			remoteAddr: "127.0.0.1:51234",
			headers:    map[string]string{"X-Forwarded-For": "6.6.6.6, 203.0.113.50"},
			trustProxy: "true",
			want:       "203.0.113.50",
		},
		{
			name:       "client-set headers are ignored behind Caddy too",
			remoteAddr: "127.0.0.1:51234",
			headers: map[string]string{
				"X-Forwarded-For":  "203.0.113.50",
				"CF-Connecting-IP": "6.6.6.6",
				"X-Real-IP":        "6.6.6.7",
			},
			trustProxy: "true",
			want:       "203.0.113.50",
		},
		{
			name:       "native loopback request without a proxy",
			remoteAddr: "127.0.0.1:51234",
			trustProxy: "",
			want:       "127.0.0.1",
		},
		{
			name:       "garbage forwarded value falls back to the peer",
			remoteAddr: "[::1]:51234",
			headers:    map[string]string{"X-Forwarded-For": "not-an-ip"},
			trustProxy: "true",
			want:       "::1",
		},
		{
			name:       "TRUST_PROXY=false ignores the header",
			remoteAddr: "127.0.0.1:51234",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.50"},
			trustProxy: "false",
			want:       "127.0.0.1",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TRUST_PROXY", tc.trustProxy)
			req := httptest.NewRequest("POST", "/api/test", nil)
			req.RemoteAddr = tc.remoteAddr
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			if got := ClientIP(req); got != tc.want {
				t.Fatalf("ClientIP = %q, want %q", got, tc.want)
			}
		})
	}
}

// The frontend's share check reaches the backend over loopback, carrying the
// visitor's address in X-Forwarded-For and no X-Internal. Each visitor has to
// get their own quota, and TRUST_PROXY=false has to keep opting out of that.
func TestRateLimitService_ForwardedClientsHaveSeparateQuotas(t *testing.T) {
	check := func(svc *RateLimitService, client string) bool {
		req := httptest.NewRequest("POST", "/api/share/validate", nil)
		req.RemoteAddr = "127.0.0.1:51234"
		req.Header.Set("X-Forwarded-For", client)
		allowed, _ := svc.Allow(req, "moderate")
		return allowed
	}
	spend := func(svc *RateLimitService, client string) {
		t.Helper()
		for i := 0; i < TierModerate.Burst; i++ {
			if !check(svc, client) {
				t.Fatalf("request %d refused before the burst was used", i)
			}
		}
		if check(svc, client) {
			t.Fatal("request past the burst was allowed")
		}
	}

	t.Setenv("TRUST_PROXY", "true")
	svc := NewRateLimitService()
	spend(svc, "203.0.113.10")
	if !check(svc, "203.0.113.11") {
		t.Fatal("a second visitor was limited by the first")
	}

	t.Setenv("TRUST_PROXY", "false")
	svc = NewRateLimitService()
	spend(svc, "203.0.113.10")
	if check(svc, "203.0.113.11") {
		t.Fatal("TRUST_PROXY=false still separated forwarded visitors")
	}
}

func TestIsInternalRequest(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		forwarded  string
		want       bool
	}{
		{"frontend on loopback", "127.0.0.1:40000", "", true},
		{"visitor through Caddy", "127.0.0.1:40000", "203.0.113.50", false},
		{"visitor reaching the backend directly", "198.51.100.7:40000", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/rss.xml", nil)
			req.RemoteAddr = tc.remoteAddr
			req.Header.Set("X-Internal", "true")
			if tc.forwarded != "" {
				req.Header.Set("X-Forwarded-For", tc.forwarded)
			}
			if got := IsInternalRequest(req); got != tc.want {
				t.Fatalf("IsInternalRequest = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRateLimitService_SpoofedHeadersShareOneBucket(t *testing.T) {
	t.Setenv("TRUST_PROXY", "true")
	svc := NewRateLimitService()
	for i := 0; i < TierStrict.Burst; i++ {
		req := httptest.NewRequest("POST", "/api/password/check", nil)
		req.RemoteAddr = "198.51.100.7:4000"
		req.Header.Set("CF-Connecting-IP", fmt.Sprintf("203.0.113.%d", i))
		if allowed, _ := svc.Allow(req, "strict"); !allowed {
			t.Fatalf("request %d refused before the burst was used", i)
		}
	}
	req := httptest.NewRequest("POST", "/api/password/check", nil)
	req.RemoteAddr = "198.51.100.7:4000"
	req.Header.Set("CF-Connecting-IP", "203.0.113.250")
	req.Header.Set("X-Forwarded-For", "203.0.113.251")
	if allowed, _ := svc.Allow(req, "strict"); allowed {
		t.Fatal("rotating proxy headers bought a fresh bucket")
	}
}

func TestRateLimitService_RetryAfter(t *testing.T) {
	svc := NewRateLimitService()

	req := httptest.NewRequest("POST", "/api/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	// Exhaust strict tier
	for i := 0; i < 5; i++ {
		svc.Allow(req, "strict")
	}

	// Next request should return a reasonable Retry-After value
	_, retryAfter := svc.Allow(req, "strict")

	// Strict tier: 5/min = 1 request per 12 seconds
	// Retry-After should be reasonable (between 1 and 60 seconds)
	if retryAfter < 1 || retryAfter > 60 {
		t.Errorf("Retry-After should be between 1 and 60 seconds for strict tier. Got: %d", retryAfter)
	}
}

func TestRateLimitService_Refill(t *testing.T) {
	// Create a service with a faster refill for testing
	// Note: This test may be flaky due to timing, so we use generous margins

	svc := NewRateLimitService()

	req := httptest.NewRequest("POST", "/api/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	// Use normal tier (1 req/sec, burst 10)
	// Exhaust some of the burst
	for i := 0; i < 5; i++ {
		svc.Allow(req, "normal")
	}

	// Wait a bit for refill (1 token per second)
	time.Sleep(1100 * time.Millisecond)

	// Should be allowed again (at least 1 token refilled)
	allowed, _ := svc.Allow(req, "normal")
	if !allowed {
		t.Error("Should have refilled at least 1 token after waiting")
	}
}

func TestRateLimitService_Stats(t *testing.T) {
	svc := NewRateLimitService()

	// Make requests from different IPs
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/api/test", nil)
		req.RemoteAddr = "192.168.1." + string(rune('0'+i)) + ":12345"
		svc.Allow(req, "strict")
	}

	stats := svc.Stats()
	if stats["strict"] != 5 {
		t.Errorf("Expected 5 limiters in strict tier, got %d", stats["strict"])
	}
}

// TestRateLimitService_IPv6 ensures IPv6 addresses work correctly
func TestRateLimitService_IPv6(t *testing.T) {
	svc := NewRateLimitService()

	req := httptest.NewRequest("POST", "/api/test", nil)
	req.RemoteAddr = "[2001:db8::1]:12345"

	allowed, _ := svc.Allow(req, "normal")
	if !allowed {
		t.Error("IPv6 request should be allowed")
	}

	// Verify it uses the correct IP
	ip := svc.getClientIP(req)
	if ip != "2001:db8::1" {
		t.Errorf("Expected IPv6 address, got: %s", ip)
	}
}

func TestRateLimitService_AllowWithInfo(t *testing.T) {
	svc := NewRateLimitService()

	req := httptest.NewRequest("POST", "/api/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	// First request on normal tier (burst=10)
	info := svc.AllowWithInfo(req, "normal")

	if !info.Allowed {
		t.Error("First request should be allowed")
	}
	if info.Limit != 10 {
		t.Errorf("Limit should be 10 (burst size), got %d", info.Limit)
	}
	// After consuming 1 token from burst of 10, remaining should be around 9
	// (might vary slightly due to timing, so we use a range)
	if info.Remaining < 7 || info.Remaining > 10 {
		t.Errorf("Remaining should be around 8-10 after first request, got %d", info.Remaining)
	}
	if info.Reset <= 0 {
		t.Error("Reset timestamp should be positive")
	}
	if info.RetryAfter != 0 {
		t.Errorf("RetryAfter should be 0 for allowed request, got %d", info.RetryAfter)
	}
}

func TestRateLimitService_AllowWithInfo_Denied(t *testing.T) {
	svc := NewRateLimitService()

	req := httptest.NewRequest("POST", "/api/test", nil)
	req.RemoteAddr = "192.168.1.101:12345"

	// Exhaust strict tier burst (3)
	for i := 0; i < 5; i++ {
		svc.Allow(req, "strict")
	}

	// Next request should be denied with proper info
	info := svc.AllowWithInfo(req, "strict")

	if info.Allowed {
		t.Error("Request should be denied after burst exhausted")
	}
	if info.Limit != 3 {
		t.Errorf("Limit should be 3 (strict burst size), got %d", info.Limit)
	}
	if info.Remaining != 0 {
		t.Errorf("Remaining should be 0 when denied, got %d", info.Remaining)
	}
	if info.RetryAfter < 1 {
		t.Error("RetryAfter should be at least 1 second when denied")
	}
	if info.Reset <= 0 {
		t.Error("Reset timestamp should be positive")
	}
}

func TestRateLimitService_AllowWithInfo_ResetTime(t *testing.T) {
	svc := NewRateLimitService()

	req := httptest.NewRequest("POST", "/api/test", nil)
	req.RemoteAddr = "192.168.1.102:12345"

	now := time.Now().Unix()

	// First request
	info := svc.AllowWithInfo(req, "normal")

	// Reset time should be in the future (when bucket is full again)
	if info.Reset < now {
		t.Error("Reset time should be now or in the future")
	}

	// Reset time should be reasonable (within a minute for normal tier)
	maxReset := now + 120 // Allow 2 minutes max
	if info.Reset > maxReset {
		t.Errorf("Reset time too far in future. Got %d, max expected %d", info.Reset, maxReset)
	}
}
