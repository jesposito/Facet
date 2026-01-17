package services

import (
	"strings"
	"testing"
	"time"
)

func TestEncryptDecrypt(t *testing.T) {
	crypto := NewCryptoService("test-encryption-key-for-testing")

	tests := []struct {
		name      string
		plaintext string
	}{
		{"empty string", ""},
		{"simple text", "hello world"},
		{"api key", "sk-1234567890abcdef"},
		{"unicode", "こんにちは世界"},
		{"long text", "This is a longer piece of text that should also encrypt and decrypt correctly without any issues at all."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := crypto.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Empty string returns empty
			if tt.plaintext == "" {
				if encrypted != "" {
					t.Errorf("Encrypt() of empty string = %v, want empty", encrypted)
				}
				return
			}

			// Encrypted should be different from plaintext
			if encrypted == tt.plaintext {
				t.Error("Encrypted text should differ from plaintext")
			}

			// Decrypt
			decrypted, err := crypto.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			// Should match original
			if decrypted != tt.plaintext {
				t.Errorf("Decrypt() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptProducesDifferentCiphertext(t *testing.T) {
	crypto := NewCryptoService("test-key")
	plaintext := "same text"

	encrypted1, _ := crypto.Encrypt(plaintext)
	encrypted2, _ := crypto.Encrypt(plaintext)

	// Due to random nonce, same plaintext should produce different ciphertext
	if encrypted1 == encrypted2 {
		t.Error("Same plaintext should produce different ciphertext due to random nonce")
	}

	// But both should decrypt to the same value
	decrypted1, _ := crypto.Decrypt(encrypted1)
	decrypted2, _ := crypto.Decrypt(encrypted2)

	if decrypted1 != decrypted2 {
		t.Error("Both ciphertexts should decrypt to the same plaintext")
	}
}

func TestPasswordHashing(t *testing.T) {
	crypto := NewCryptoService("")

	password := "my-secure-password"

	hash, err := crypto.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	// Hash should be different from password
	if hash == password {
		t.Error("Hash should differ from password")
	}

	// Should validate correctly
	if !crypto.CheckPassword(password, hash) {
		t.Error("CheckPassword() should return true for correct password")
	}

	// Should reject wrong password
	if crypto.CheckPassword("wrong-password", hash) {
		t.Error("CheckPassword() should return false for wrong password")
	}
}

func TestGenerateToken(t *testing.T) {
	crypto := NewCryptoService("")

	token1, err := crypto.GenerateToken(32)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	token2, err := crypto.GenerateToken(32)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Tokens should be different
	if token1 == token2 {
		t.Error("Generated tokens should be unique")
	}

	// Token should have reasonable length
	if len(token1) < 40 {
		t.Errorf("Token length = %d, want at least 40 (base64 of 32 bytes)", len(token1))
	}
}

func TestHMACToken(t *testing.T) {
	crypto := NewCryptoService("test-key")

	token := "my-share-token"
	hash1 := crypto.HMACToken(token)
	hash2 := crypto.HMACToken(token)

	// Same token should produce same hash
	if hash1 != hash2 {
		t.Error("Same token should produce same hash")
	}

	// Different token should produce different hash
	hash3 := crypto.HMACToken("different-token")
	if hash1 == hash3 {
		t.Error("Different tokens should produce different hashes")
	}

	// Validate constant-time comparison works
	if !crypto.ValidateTokenHMAC(token, hash1) {
		t.Error("ValidateTokenHMAC should return true for correct token")
	}
	if crypto.ValidateTokenHMAC("wrong-token", hash1) {
		t.Error("ValidateTokenHMAC should return false for wrong token")
	}
}

// JWT Tests for password-protected view access

func TestGenerateViewAccessJWT(t *testing.T) {
	crypto := NewCryptoService("test-encryption-key-32-chars-ok!")

	viewID := "test_view_123"
	duration := 1 * time.Hour

	token, expiresAt, err := crypto.GenerateViewAccessJWT(viewID, duration)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if token == "" {
		t.Fatal("Expected non-empty token")
	}

	if expiresAt.Before(time.Now()) {
		t.Fatal("Expected expiry in the future")
	}

	// Expiry should be approximately 1 hour from now
	expectedExpiry := time.Now().Add(duration)
	if expiresAt.Sub(expectedExpiry) > time.Second {
		t.Fatalf("Expected expiry around %v, got %v", expectedExpiry, expiresAt)
	}
}

func TestValidateViewAccessJWT_Valid(t *testing.T) {
	crypto := NewCryptoService("test-encryption-key-32-chars-ok!")

	viewID := "test_view_123"
	token, _, err := crypto.GenerateViewAccessJWT(viewID, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	validatedViewID, err := crypto.ValidateViewAccessJWT(token)
	if err != nil {
		t.Fatalf("Expected valid token, got error: %v", err)
	}

	if validatedViewID != viewID {
		t.Fatalf("Expected view ID %s, got %s", viewID, validatedViewID)
	}
}

func TestValidateViewAccessJWT_NoToken(t *testing.T) {
	crypto := NewCryptoService("test-encryption-key-32-chars-ok!")

	_, err := crypto.ValidateViewAccessJWT("")
	if err == nil {
		t.Fatal("Expected error for empty token")
	}
	if err.Error() != "token required" {
		t.Fatalf("Expected 'token required' error, got: %v", err)
	}
}

func TestValidateViewAccessJWT_Malformed(t *testing.T) {
	crypto := NewCryptoService("test-encryption-key-32-chars-ok!")

	_, err := crypto.ValidateViewAccessJWT("not-a-valid-jwt")
	if err == nil {
		t.Fatal("Expected error for malformed token")
	}
	if err.Error() != "invalid token" {
		t.Fatalf("Expected 'invalid token' error, got: %v", err)
	}
}

func TestValidateViewAccessJWT_Tampered(t *testing.T) {
	crypto := NewCryptoService("test-encryption-key-32-chars-ok!")

	token, _, err := crypto.GenerateViewAccessJWT("test_view", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("Expected JWT to have 3 parts, got %d", len(parts))
	}

	signature := parts[2]
	if len(signature) < 10 {
		t.Fatalf("Signature too short to modify safely: %s", signature)
	}
	tamperedSignature := "TAMPERED" + signature[8:]
	tamperedToken := parts[0] + "." + parts[1] + "." + tamperedSignature

	_, err = crypto.ValidateViewAccessJWT(tamperedToken)
	if err == nil {
		t.Fatalf("Expected error for tampered token, but token was accepted. Original: %s, Tampered: %s", token, tamperedToken)
	}
}

func TestValidateViewAccessJWT_WrongKey(t *testing.T) {
	crypto1 := NewCryptoService("first-encryption-key-32-chars!!")
	crypto2 := NewCryptoService("second-encryption-key-32-chars!")

	token, _, err := crypto1.GenerateViewAccessJWT("test_view", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Try to validate with different key (different CryptoService)
	_, err = crypto2.ValidateViewAccessJWT(token)
	if err == nil {
		t.Fatal("Expected error when validating with different key")
	}
}

func TestValidateViewAccessJWT_Expired(t *testing.T) {
	crypto := NewCryptoService("test-encryption-key-32-chars-ok!")

	// Generate token that expires immediately (negative duration)
	token, _, err := crypto.GenerateViewAccessJWT("test_view", -1*time.Second)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Wait a moment to ensure expiry
	time.Sleep(10 * time.Millisecond)

	_, err = crypto.ValidateViewAccessJWT(token)
	if err == nil {
		t.Fatal("Expected error for expired token")
	}
}

func TestValidateViewAccessJWT_WrongViewID(t *testing.T) {
	crypto := NewCryptoService("test-encryption-key-32-chars-ok!")

	// Generate tokens for two different views
	token1, _, _ := crypto.GenerateViewAccessJWT("view_1", 1*time.Hour)
	token2, _, _ := crypto.GenerateViewAccessJWT("view_2", 1*time.Hour)

	// Validate each returns correct view ID
	viewID1, err := crypto.ValidateViewAccessJWT(token1)
	if err != nil || viewID1 != "view_1" {
		t.Fatalf("Expected view_1, got %s (err: %v)", viewID1, err)
	}

	viewID2, err := crypto.ValidateViewAccessJWT(token2)
	if err != nil || viewID2 != "view_2" {
		t.Fatalf("Expected view_2, got %s (err: %v)", viewID2, err)
	}

	// Token for view_1 should not return view_2
	if viewID1 == "view_2" {
		t.Fatal("Token for view_1 should not validate as view_2")
	}
}

func TestJWTConstants(t *testing.T) {
	if JWTIssuer != "facet" {
		t.Fatalf("Expected issuer 'facet', got '%s'", JWTIssuer)
	}
	if JWTIssuerLegacy != "me.yaml" {
		t.Fatalf("Expected legacy issuer 'me.yaml', got '%s'", JWTIssuerLegacy)
	}
	if JWTAudience != "view-access" {
		t.Fatalf("Expected audience 'view-access', got '%s'", JWTAudience)
	}
}
