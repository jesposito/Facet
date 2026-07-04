package hooks

import (
	"strings"
	"testing"

	"facet/services"
)

func TestApiKeyGenerate(t *testing.T) {
	key, err := generateAPIKey()
	if err != nil {
		t.Fatalf("generateAPIKey returned error: %v", err)
	}
	if !strings.HasPrefix(key, APIKeyPrefix) {
		t.Fatalf("expected key to start with %q, got %q", APIKeyPrefix, key)
	}
	// "facet_" (6) + 32 random bytes hex-encoded (64) = 70 chars.
	wantLen := len(APIKeyPrefix) + apiKeyRandBytes*2
	if len(key) != wantLen {
		t.Fatalf("expected key length %d, got %d (%q)", wantLen, len(key), key)
	}

	// Two consecutive keys should never collide.
	other, err := generateAPIKey()
	if err != nil {
		t.Fatalf("generateAPIKey second call returned error: %v", err)
	}
	if key == other {
		t.Fatalf("expected unique keys, got duplicates: %q", key)
	}
}

func TestApiKeyHashRoundtrip(t *testing.T) {
	crypto := services.NewCryptoService("test-encryption-key-32-chars-ok!")
	key, err := generateAPIKey()
	if err != nil {
		t.Fatalf("generateAPIKey returned error: %v", err)
	}

	h1 := hashAPIKey(crypto, key)
	h2 := hashAPIKey(crypto, key)
	if h1 != h2 {
		t.Fatalf("hashAPIKey is not deterministic: %q != %q", h1, h2)
	}

	// HMAC-SHA256 base64 is always 44 chars with the current CryptoService helper.
	if len(h1) != 44 {
		t.Fatalf("expected hash length 44, got %d", len(h1))
	}

	// Mutating the key must change the hash (no truncation collisions).
	tampered := key + "x"
	if hashAPIKey(crypto, tampered) == h1 {
		t.Fatalf("hash collision between %q and %q", key, tampered)
	}

	otherCrypto := services.NewCryptoService("other-encryption-key-32-chars-ok")
	if hashAPIKey(otherCrypto, key) == h1 {
		t.Fatalf("hash should be keyed to this instance's encryption key")
	}

	// The hash must NEVER equal the raw key (defensive — protects against
	// accidentally storing the plaintext key in key_hash).
	if h1 == key {
		t.Fatalf("hash equals raw key; refusing to ship")
	}
}

func TestApiKeyExtractPrefix(t *testing.T) {
	key, err := generateAPIKey()
	if err != nil {
		t.Fatalf("generateAPIKey returned error: %v", err)
	}
	prefix := extractKeyPrefix(key)
	if len(prefix) != keyDisplayPrefixLen {
		t.Fatalf("expected prefix length %d, got %d (%q)", keyDisplayPrefixLen, len(prefix), prefix)
	}
	if !strings.HasPrefix(key, prefix) {
		t.Fatalf("prefix %q is not a prefix of key %q", prefix, key)
	}

	// Short input should be returned as-is, not panic.
	short := "abc"
	if got := extractKeyPrefix(short); got != short {
		t.Fatalf("expected short string returned unchanged, got %q", got)
	}
}

func TestApiKeyScopeCheck(t *testing.T) {
	tests := []struct {
		name     string
		granted  []string
		required string
		want     bool
	}{
		{"empty grant denies", nil, "read:profile", false},
		{"exact match passes", []string{"read:profile"}, "read:profile", true},
		{"non-match denies", []string{"read:posts"}, "read:profile", false},
		{"multiple scopes one match passes", []string{"read:posts", "read:profile"}, "read:profile", true},
		{"empty required is denied (defensive)", []string{"read:profile"}, "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hasScope(tc.granted, tc.required)
			if got != tc.want {
				t.Errorf("hasScope(%v, %q) = %v, want %v", tc.granted, tc.required, got, tc.want)
			}
		})
	}
}

func TestApiKeyValidScopes(t *testing.T) {
	// Self-hosted v1 exposes read:* and write:* scopes only. Any other prefix
	// (e.g. admin:*, delete:*) is a regression — those would imply privileged
	// operations that the public API surface deliberately does not offer.
	for s := range validScopes {
		if !strings.HasPrefix(s, "read:") && !strings.HasPrefix(s, "write:") {
			t.Errorf("unexpected scope prefix in validScopes: %q", s)
		}
	}
	if len(validScopes) == 0 {
		t.Fatalf("validScopes is empty; admin UI will have nothing to offer")
	}
	// Sanity: every resource that has a write scope must have a matching read
	// scope, otherwise a write-only key couldn't read back what it created.
	for s := range validScopes {
		if strings.HasPrefix(s, "write:") {
			readEquiv := "read:" + strings.TrimPrefix(s, "write:")
			if !validScopes[readEquiv] {
				t.Errorf("write scope %q has no matching read scope %q", s, readEquiv)
			}
		}
	}
}

func TestApiKeyDecodeJSONStringSlice(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  []string
	}{
		{"nil returns nil", nil, nil},
		{"string slice direct", []string{"a", "b"}, []string{"a", "b"}},
		{"any slice", []any{"a", "b"}, []string{"a", "b"}},
		{"empty slice", []any{}, []string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := decodeJSONStringSlice(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("decodeJSONStringSlice(%v) length = %d, want %d", tc.input, len(got), len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
