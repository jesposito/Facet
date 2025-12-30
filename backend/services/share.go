package services

import (
	"time"
)

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
