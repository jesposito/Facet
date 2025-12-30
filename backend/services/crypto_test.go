package services

import (
	"testing"
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

func TestHashToken(t *testing.T) {
	crypto := NewCryptoService("")

	token := "my-share-token"
	hash1 := crypto.HashToken(token)
	hash2 := crypto.HashToken(token)

	// Same token should produce same hash
	if hash1 != hash2 {
		t.Error("Same token should produce same hash")
	}

	// Different token should produce different hash
	hash3 := crypto.HashToken("different-token")
	if hash1 == hash3 {
		t.Error("Different tokens should produce different hashes")
	}
}
