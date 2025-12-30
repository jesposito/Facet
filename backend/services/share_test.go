package services

import (
	"testing"
	"time"
)

func TestShareTokenGeneration(t *testing.T) {
	share := NewShareService()

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
	share := NewShareService()

	token, _ := share.GenerateToken()
	hash := share.HashToken(token)

	// Same token should validate
	if !share.ValidateTokenHash(token, hash) {
		t.Error("ValidateTokenHash() should return true for matching token")
	}

	// Different token should not validate
	otherToken, _ := share.GenerateToken()
	if share.ValidateTokenHash(otherToken, hash) {
		t.Error("ValidateTokenHash() should return false for different token")
	}

	// Wrong hash should not validate
	if share.ValidateTokenHash(token, "wrong-hash") {
		t.Error("ValidateTokenHash() should return false for wrong hash")
	}
}

func TestTokenExpiration(t *testing.T) {
	share := NewShareService()

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
	share := NewShareService()

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
