package services

import (
	"time"
)

type TestimonialService struct {
	crypto *CryptoService
}

func NewTestimonialService(crypto *CryptoService) *TestimonialService {
	return &TestimonialService{crypto: crypto}
}

type TestimonialRequestValidation struct {
	Valid         bool   `json:"valid"`
	RequestID     string `json:"request_id,omitempty"`
	Label         string `json:"label,omitempty"`
	CustomMessage string `json:"custom_message,omitempty"`
	RecipientName string `json:"recipient_name,omitempty"`
	Error         string `json:"error,omitempty"`
}

type TestimonialSubmission struct {
	RequestToken           string `json:"request_token"`
	Content                string `json:"content"`
	Relationship           string `json:"relationship,omitempty"`
	Project                string `json:"project,omitempty"`
	AuthorName             string `json:"author_name"`
	AuthorTitle            string `json:"author_title,omitempty"`
	AuthorCompany          string `json:"author_company,omitempty"`
	AuthorWebsite          string `json:"author_website,omitempty"`
	VerificationMethod     string `json:"verification_method,omitempty"`
	VerificationIdentifier string `json:"verification_identifier,omitempty"`
}

func (s *TestimonialService) GenerateToken() (string, error) {
	return s.crypto.GenerateToken(32)
}

func (s *TestimonialService) HMACToken(token string) string {
	return s.crypto.HMACToken(token)
}

func (s *TestimonialService) TokenPrefix(token string) string {
	if len(token) < TokenPrefixLength {
		return token
	}
	return token[:TokenPrefixLength]
}

func (s *TestimonialService) ValidateTokenHMAC(token, storedHMAC string) bool {
	return s.crypto.ValidateTokenHMAC(token, storedHMAC)
}

func (s *TestimonialService) IsRequestExpired(expiresAt *time.Time) bool {
	if expiresAt == nil {
		return false
	}
	return time.Now().After(*expiresAt)
}

func (s *TestimonialService) CanUseRequest(useCount int, maxUses *int) bool {
	if maxUses == nil || *maxUses <= 0 {
		return true
	}
	return useCount < *maxUses
}

func (s *TestimonialService) GenerateEmailVerificationToken() (string, error) {
	return s.crypto.GenerateToken(32)
}

func (s *TestimonialService) EmailVerificationExpiry() time.Time {
	return time.Now().Add(15 * time.Minute)
}
