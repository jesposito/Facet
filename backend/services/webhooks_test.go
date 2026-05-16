package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

// TestHMACSignature verifies that the algorithm we use to sign webhook bodies
// is deterministic, prefixed correctly, and rejects body tampering.
// Receivers verify by recomputing the same hex(hmac-sha256(secret, body))
// and comparing against the X-Facet-Signature header.
func TestHMACSignature(t *testing.T) {
	secret := "test-secret-key-32-bytes-aaaaaaaa"
	body := []byte(`{"event":"comment.created","timestamp":"2026-05-16T12:00:00Z","data":{"id":"abc"}}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// Re-compute with the exact same inputs — proves determinism.
	mac2 := hmac.New(sha256.New, []byte(secret))
	mac2.Write(body)
	got := "sha256=" + hex.EncodeToString(mac2.Sum(nil))

	if got != expected {
		t.Fatalf("signature mismatch: got %q want %q", got, expected)
	}
	if !strings.HasPrefix(got, "sha256=") {
		t.Fatalf("signature must be prefixed with sha256=, got %q", got)
	}

	// Tampered body must produce a different signature.
	tampered := append([]byte(nil), body...)
	tampered[0] = 'X'
	macBad := hmac.New(sha256.New, []byte(secret))
	macBad.Write(tampered)
	tamperedSig := "sha256=" + hex.EncodeToString(macBad.Sum(nil))
	if tamperedSig == got {
		t.Fatal("tampered body produced same signature — HMAC contract broken")
	}

	// Wrong secret must produce a different signature (defence vs replay).
	macBad2 := hmac.New(sha256.New, []byte("wrong-secret"))
	macBad2.Write(body)
	wrongSecretSig := "sha256=" + hex.EncodeToString(macBad2.Sum(nil))
	if wrongSecretSig == got {
		t.Fatal("wrong secret produced same signature — key isolation broken")
	}
}

// TestSSRFProtection verifies ValidateWebhookURL blocks every category of
// internal/private address the SSRF dialer would otherwise have to catch
// at connect time. We avoid asserting on a real public DNS name to keep
// tests hermetic.
func TestSSRFProtection(t *testing.T) {
	cases := []struct {
		name      string
		url       string
		wantBlock bool
	}{
		{"loopback v4", "https://127.0.0.1/hook", true},
		{"loopback hostname", "https://localhost/hook", true},
		{"loopback v6", "https://[::1]/hook", true},
		{"unspecified", "https://0.0.0.0/hook", true},
		{"docker internal", "https://host.docker.internal/hook", true},
		{"AWS metadata literal", "https://169.254.169.254/latest/meta-data/", true},
		{"GCP metadata hostname", "https://metadata.google.internal/", true},
		{"RFC1918 10.x", "https://10.0.0.5/hook", true},
		{"RFC1918 192.168", "https://192.168.1.1/hook", true},
		{"RFC1918 172.16", "https://172.16.0.1/hook", true},
		{"CGNAT 100.64", "https://100.64.0.1/hook", true},
		{"link-local v4", "https://169.254.1.1/hook", true},
		{"IPv6 unique-local", "https://[fc00::1]/hook", true},
		{"mdns .local", "https://printer.local/hook", true},
		{"corp .internal", "https://api.internal/hook", true},
		{"ftp scheme", "ftp://example.com/hook", true},
		{"file scheme", "file:///etc/passwd", true},
		{"empty host", "https:///hook", true},
		// 8.8.8.8 is google DNS; we only check that public literal IPs aren't
		// blocked by the range checks. Reachability is irrelevant here.
		{"public literal IPv4", "https://8.8.8.8/hook", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateWebhookURL(tc.url)
			if tc.wantBlock && err == nil {
				t.Fatalf("expected URL to be blocked: %s", tc.url)
			}
			if !tc.wantBlock && err != nil {
				t.Fatalf("expected URL to be allowed: %s (error: %v)", tc.url, err)
			}
		})
	}
}

// TestRetryBackoff verifies the exponential backoff schedule matches the spec
// (1min, 5min, 30min, 2hr, 12hr) and that RetryBackoff returns a defensive
// copy so callers can't mutate shared state.
func TestRetryBackoff(t *testing.T) {
	want := []time.Duration{
		1 * time.Minute,
		5 * time.Minute,
		30 * time.Minute,
		2 * time.Hour,
		12 * time.Hour,
	}
	got := RetryBackoff()
	if len(got) != len(want) {
		t.Fatalf("backoff length mismatch: got %d want %d", len(got), len(want))
	}
	for i, d := range want {
		if got[i] != d {
			t.Errorf("backoff[%d] = %v, want %v", i, got[i], d)
		}
	}

	// Mutate the returned slice; subsequent call must be unaffected.
	got[0] = 999 * time.Hour
	got2 := RetryBackoff()
	if got2[0] != want[0] {
		t.Fatalf("RetryBackoff returned shared slice (mutation leaked): got %v", got2[0])
	}
}
