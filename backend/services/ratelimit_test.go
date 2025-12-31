package services

import (
	"net/http"
	"net/http/httptest"
	"os"
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

func TestRateLimitService_TrustProxy(t *testing.T) {
	// Test with TRUST_PROXY=false (default)
	os.Unsetenv("TRUST_PROXY")
	svc := NewRateLimitService()

	req := httptest.NewRequest("POST", "/api/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 10.0.0.1")
	req.Header.Set("CF-Connecting-IP", "203.0.113.100")

	// Should use RemoteAddr, not headers
	ip := svc.getClientIP(req)
	if ip != "10.0.0.1" {
		t.Errorf("Without TRUST_PROXY, should use RemoteAddr. Got: %s", ip)
	}

	// Test with TRUST_PROXY=true
	os.Setenv("TRUST_PROXY", "true")
	svc2 := NewRateLimitService()

	// Should prefer CF-Connecting-IP when available
	ip = svc2.getClientIP(req)
	if ip != "203.0.113.100" {
		t.Errorf("With TRUST_PROXY, should use CF-Connecting-IP. Got: %s", ip)
	}

	// Test X-Forwarded-For when CF-Connecting-IP is not set
	req2 := httptest.NewRequest("POST", "/api/test", nil)
	req2.RemoteAddr = "10.0.0.1:12345"
	req2.Header.Set("X-Forwarded-For", "203.0.113.50, 10.0.0.1")

	ip = svc2.getClientIP(req2)
	if ip != "203.0.113.50" {
		t.Errorf("With TRUST_PROXY and no CF-Connecting-IP, should use leftmost X-Forwarded-For. Got: %s", ip)
	}

	// Cleanup
	os.Unsetenv("TRUST_PROXY")
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
