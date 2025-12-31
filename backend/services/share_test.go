package services

import (
	"testing"
	"time"
)

func TestTokenPrefixLength(t *testing.T) {
	if TokenPrefixLength != 12 {
		t.Fatalf("Expected TokenPrefixLength to be 12, got %d", TokenPrefixLength)
	}
}

func TestTokenPrefix(t *testing.T) {
	crypto := NewCryptoService("test-encryption-key-32-chars-ok!")
	share := NewShareService(crypto)

	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{"normal token", "abcdefghijklmnopqrstuvwxyz", "abcdefghijkl"},
		{"exact length", "abcdefghijkl", "abcdefghijkl"},
		{"short token", "short", "short"},
		{"empty token", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix := share.TokenPrefix(tt.token)
			if prefix != tt.expected {
				t.Errorf("TokenPrefix(%q) = %q, want %q", tt.token, prefix, tt.expected)
			}
		})
	}
}

func TestShareTokenGeneration(t *testing.T) {
	crypto := NewCryptoService("test-encryption-key-32-chars-ok!")
	share := NewShareService(crypto)

	token1, err := share.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	token2, err := share.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Tokens should be unique
	if token1 == token2 {
		t.Error("Generated tokens should be unique")
	}

	// Tokens should have reasonable length (base64 of 32 bytes = ~43 chars)
	if len(token1) < 40 {
		t.Errorf("Token length = %d, want at least 40", len(token1))
	}
}

func TestShareTokenHashValidation(t *testing.T) {
	crypto := NewCryptoService("test-encryption-key-32-chars-ok!")
	share := NewShareService(crypto)

	token, _ := share.GenerateToken()
	hash := share.HMACToken(token)

	// Same token should validate
	if !share.ValidateTokenHMAC(token, hash) {
		t.Error("ValidateTokenHMAC() should return true for matching token")
	}

	// Different token should not validate
	otherToken, _ := share.GenerateToken()
	if share.ValidateTokenHMAC(otherToken, hash) {
		t.Error("ValidateTokenHMAC() should return false for different token")
	}

	// Wrong hash should not validate
	if share.ValidateTokenHMAC(token, "wrong-hash") {
		t.Error("ValidateTokenHMAC() should return false for wrong hash")
	}
}

func TestHMACWithDifferentKeys(t *testing.T) {
	crypto1 := NewCryptoService("first-encryption-key-32-chars!!")
	crypto2 := NewCryptoService("second-encryption-key-32-chars!")
	share1 := NewShareService(crypto1)
	share2 := NewShareService(crypto2)

	token := "same-token"
	hash1 := share1.HMACToken(token)
	hash2 := share2.HMACToken(token)

	// Same token with different keys should produce different hashes
	if hash1 == hash2 {
		t.Error("Same token with different keys should produce different hashes")
	}

	// Cross-validation should fail
	if share2.ValidateTokenHMAC(token, hash1) {
		t.Error("Token hashed with different key should not validate")
	}
}

func TestTokenPrefixConsistency(t *testing.T) {
	crypto := NewCryptoService("test-encryption-key-32-chars-ok!")
	share := NewShareService(crypto)

	// Generate a token and extract prefix
	token, _ := share.GenerateToken()
	prefix := share.TokenPrefix(token)

	// Prefix should be the first 12 chars of the token
	if token[:TokenPrefixLength] != prefix {
		t.Error("Prefix should be first 12 chars of token")
	}

	// Multiple calls should return same prefix
	prefix2 := share.TokenPrefix(token)
	if prefix != prefix2 {
		t.Error("TokenPrefix should be deterministic")
	}
}

func TestTokenExpiration(t *testing.T) {
	crypto := NewCryptoService("test-encryption-key-32-chars-ok!")
	share := NewShareService(crypto)

	// No expiration
	if share.IsTokenExpired(nil) {
		t.Error("IsTokenExpired(nil) should return false")
	}

	// Future expiration
	future := time.Now().Add(1 * time.Hour)
	if share.IsTokenExpired(&future) {
		t.Error("IsTokenExpired(future) should return false")
	}

	// Past expiration
	past := time.Now().Add(-1 * time.Hour)
	if !share.IsTokenExpired(&past) {
		t.Error("IsTokenExpired(past) should return true")
	}
}

func TestTokenUsageLimit(t *testing.T) {
	crypto := NewCryptoService("test-encryption-key-32-chars-ok!")
	share := NewShareService(crypto)

	// No limit
	if !share.CanUseToken(100, nil) {
		t.Error("CanUseToken(100, nil) should return true")
	}

	// Zero limit (unlimited)
	zero := 0
	if !share.CanUseToken(100, &zero) {
		t.Error("CanUseToken(100, 0) should return true")
	}

	// Under limit
	limit := 5
	if !share.CanUseToken(3, &limit) {
		t.Error("CanUseToken(3, 5) should return true")
	}

	// At limit
	if share.CanUseToken(5, &limit) {
		t.Error("CanUseToken(5, 5) should return false")
	}

	// Over limit
	if share.CanUseToken(10, &limit) {
		t.Error("CanUseToken(10, 5) should return false")
	}
}

func TestHMACConstantTimeComparison(t *testing.T) {
	crypto := NewCryptoService("test-encryption-key-32-chars-ok!")
	share := NewShareService(crypto)

	token := "test-token-12345"
	hash := share.HMACToken(token)

	// Test that validation uses constant-time comparison by verifying behavior
	// (actual timing verification would require more complex testing)

	// Correct token should validate
	if !share.ValidateTokenHMAC(token, hash) {
		t.Error("Correct token should validate")
	}

	// Completely wrong token should not validate
	if share.ValidateTokenHMAC("completely-different", hash) {
		t.Error("Wrong token should not validate")
	}

	// Token with only first char different should not validate
	wrongFirst := "Xest-token-12345"
	if share.ValidateTokenHMAC(wrongFirst, hash) {
		t.Error("Token with different first char should not validate")
	}

	// Token with only last char different should not validate
	wrongLast := "test-token-12346"
	if share.ValidateTokenHMAC(wrongLast, hash) {
		t.Error("Token with different last char should not validate")
	}
}

func TestTokenPrefixWithRealToken(t *testing.T) {
	crypto := NewCryptoService("test-encryption-key-32-chars-ok!")
	share := NewShareService(crypto)

	// Generate a real token
	token, err := share.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	prefix := share.TokenPrefix(token)

	// Prefix length should be exactly TokenPrefixLength
	if len(prefix) != TokenPrefixLength {
		t.Errorf("Prefix length = %d, want %d", len(prefix), TokenPrefixLength)
	}

	// Prefix should be a substring of the token
	if token[:TokenPrefixLength] != prefix {
		t.Error("Prefix should match first TokenPrefixLength chars of token")
	}

	// HMAC should still work correctly with the full token
	hash := share.HMACToken(token)
	if !share.ValidateTokenHMAC(token, hash) {
		t.Error("Full token HMAC validation should work")
	}
}

// TestPrefixLookupStrategy verifies the prefix-based lookup design:
// - Prefix is deterministic and extractable from any token
// - Same prefix means potential match (requires HMAC verification)
// - Different prefix means definite non-match (can skip HMAC check)
func TestPrefixLookupStrategy(t *testing.T) {
	crypto := NewCryptoService("test-encryption-key-32-chars-ok!")
	share := NewShareService(crypto)

	// Generate multiple tokens
	tokens := make([]string, 10)
	prefixes := make(map[string][]string) // prefix -> tokens with that prefix

	for i := 0; i < 10; i++ {
		token, err := share.GenerateToken()
		if err != nil {
			t.Fatalf("GenerateToken() error = %v", err)
		}
		tokens[i] = token
		prefix := share.TokenPrefix(token)
		prefixes[prefix] = append(prefixes[prefix], token)
	}

	// Most prefixes should be unique (collision is astronomically rare with 12 base64 chars)
	// With 10 tokens and ~72 bits of prefix entropy, collisions are virtually impossible
	uniquePrefixes := len(prefixes)
	if uniquePrefixes < 10 {
		t.Logf("Warning: Got %d unique prefixes from 10 tokens (collision detected)", uniquePrefixes)
		// This is not an error - just very unlikely
	}

	// For each token, verify:
	// 1. Its prefix can be used to locate it
	// 2. HMAC verification confirms the match
	for _, token := range tokens {
		prefix := share.TokenPrefix(token)
		hash := share.HMACToken(token)

		// The prefix matches
		if len(prefix) != TokenPrefixLength {
			t.Errorf("Prefix length mismatch for token")
		}

		// HMAC verification works
		if !share.ValidateTokenHMAC(token, hash) {
			t.Error("HMAC verification failed for matching token")
		}

		// A token with different suffix but same prefix would fail HMAC
		// (simulating a prefix collision scenario)
		fakeToken := prefix + "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
		if share.ValidateTokenHMAC(fakeToken, hash) {
			t.Error("HMAC should fail for fake token with same prefix")
		}
	}
}

// TestPrefixIsNonSecretLookupHelper verifies that prefix alone is insufficient
// for authentication - HMAC verification is required for security
func TestPrefixIsNonSecretLookupHelper(t *testing.T) {
	crypto := NewCryptoService("test-encryption-key-32-chars-ok!")
	share := NewShareService(crypto)

	token, _ := share.GenerateToken()
	prefix := share.TokenPrefix(token)
	hash := share.HMACToken(token)

	// Knowing the prefix is NOT sufficient to authenticate
	// An attacker with the prefix cannot generate a valid token
	for i := 0; i < 100; i++ {
		// Generate random suffix attempts
		attackToken := prefix + "attack-attempt-" + string(rune('0'+i))
		if share.ValidateTokenHMAC(attackToken, hash) {
			t.Fatal("CRITICAL: Prefix-only attack succeeded - this should never happen")
		}
	}

	// Only the exact original token validates
	if !share.ValidateTokenHMAC(token, hash) {
		t.Error("Original token should validate")
	}
}

// TestLegacyTokenWithoutPrefix simulates tokens created before prefix migration
func TestLegacyTokenWithoutPrefix(t *testing.T) {
	crypto := NewCryptoService("test-encryption-key-32-chars-ok!")
	share := NewShareService(crypto)

	// Simulate a legacy token (HMAC exists but no prefix stored)
	legacyToken, _ := share.GenerateToken()
	legacyHash := share.HMACToken(legacyToken)

	// Legacy tokens can still be validated via HMAC comparison
	// (the hooks layer handles the fallback query)
	if !share.ValidateTokenHMAC(legacyToken, legacyHash) {
		t.Error("Legacy token should still validate via HMAC")
	}

	// Prefix extraction still works for new lookups
	prefix := share.TokenPrefix(legacyToken)
	if len(prefix) != TokenPrefixLength {
		t.Error("Prefix extraction should work on legacy tokens")
	}
}
