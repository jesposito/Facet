package services

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"time"
)

// ShareService handles share token operations
type ShareService struct{}

// NewShareService creates a new share service
func NewShareService() *ShareService {
	return &ShareService{}
}

// ShareTokenValidation represents the result of token validation
type ShareTokenValidation struct {
	Valid     bool   `json:"valid"`
	ViewID    string `json:"view_id"`
	ViewSlug  string `json:"view_slug"`
	Error     string `json:"error,omitempty"`
}

// GenerateToken generates a new share token (32 bytes, URL-safe base64)
func (s *ShareService) GenerateToken() (string, error) {
	crypto := &CryptoService{}
	return crypto.GenerateToken(32)
}

// HashToken creates a SHA-256 hash of a token for storage
func (s *ShareService) HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// ValidateTokenHash compares a token against a stored hash using constant-time comparison
func (s *ShareService) ValidateTokenHash(token, storedHash string) bool {
	tokenHash := s.HashToken(token)
	return subtle.ConstantTimeCompare([]byte(tokenHash), []byte(storedHash)) == 1
}

// IsTokenExpired checks if a token has expired
func (s *ShareService) IsTokenExpired(expiresAt *time.Time) bool {
	if expiresAt == nil {
		return false // No expiration
	}
	return time.Now().After(*expiresAt)
}

// CanUseToken checks if a token has remaining uses
func (s *ShareService) CanUseToken(useCount int, maxUses *int) bool {
	if maxUses == nil || *maxUses <= 0 {
		return true // Unlimited uses
	}
	return useCount < *maxUses
}
