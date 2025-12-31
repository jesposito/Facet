package services

import (
	"time"
)

// TokenPrefixLength is the number of characters to store for indexed lookup
// 12 chars of base64 = ~72 bits of entropy, sufficient for narrowing to 1-2 candidates
const TokenPrefixLength = 12

// ShareService handles share token operations
type ShareService struct {
	crypto *CryptoService
}

// NewShareService creates a new share service with crypto for HMAC
func NewShareService(crypto *CryptoService) *ShareService {
	return &ShareService{crypto: crypto}
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
	return s.crypto.GenerateToken(32)
}

// HMACToken creates an HMAC of a token for secure storage
// DB leaks won't allow offline verification without the server secret
func (s *ShareService) HMACToken(token string) string {
	return s.crypto.HMACToken(token)
}

// ValidateTokenHMAC compares a token against stored HMAC using constant-time comparison
func (s *ShareService) ValidateTokenHMAC(token, storedHMAC string) bool {
	return s.crypto.ValidateTokenHMAC(token, storedHMAC)
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

// TokenPrefix extracts the prefix from a raw token for indexed lookup
// This is stored unencrypted for O(1) database queries
func (s *ShareService) TokenPrefix(token string) string {
	if len(token) < TokenPrefixLength {
		return token
	}
	return token[:TokenPrefixLength]
}
