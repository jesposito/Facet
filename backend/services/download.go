package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// DownloadURLExpiry is the duration a signed download URL remains valid
	DownloadURLExpiry = 15 * time.Minute
)

// DownloadService handles HMAC-signed download URLs
type DownloadService struct {
	cryptoService *CryptoService
}

// NewDownloadService creates a new download service
func NewDownloadService(crypto *CryptoService) *DownloadService {
	return &DownloadService{cryptoService: crypto}
}

// GenerateSignedURL creates an HMAC-signed download token with 15-minute expiry.
// Returns the token string and the expiration time.
func (s *DownloadService) GenerateSignedURL(contentType, contentID, filename string) (string, time.Time) {
	expiry := time.Now().Add(DownloadURLExpiry)
	expiryUnix := strconv.FormatInt(expiry.Unix(), 10)

	// Build the payload: contentType:contentID:filename:expiryUnix
	payload := fmt.Sprintf("%s:%s:%s:%s", contentType, contentID, filename, expiryUnix)

	// Generate HMAC of the payload
	mac := hmac.New(sha256.New, s.cryptoService.key)
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	// Token format: base64url(contentType:contentID:filename:expiryUnix:hmac)
	token := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", payload, sig)))

	return token, expiry
}

// ValidateSignedURL checks the HMAC signature and expiry of a download token.
// Returns the contentType, contentID, and filename if valid.
func (s *DownloadService) ValidateSignedURL(token string) (contentType, contentID, filename string, err error) {
	// Decode the token
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", "", "", errors.New("invalid token encoding")
	}

	// Split into parts: contentType:contentID:filename:expiryUnix:hmac
	parts := strings.SplitN(string(decoded), ":", 5)
	if len(parts) != 5 {
		return "", "", "", errors.New("invalid token format")
	}

	contentType = parts[0]
	contentID = parts[1]
	filename = parts[2]
	expiryStr := parts[3]
	providedSig := parts[4]

	// Validate expiry
	expiryUnix, err := strconv.ParseInt(expiryStr, 10, 64)
	if err != nil {
		return "", "", "", errors.New("invalid expiry")
	}

	if time.Now().Unix() > expiryUnix {
		return "", "", "", errors.New("token expired")
	}

	// Recompute HMAC and compare
	payload := fmt.Sprintf("%s:%s:%s:%s", contentType, contentID, filename, expiryStr)
	mac := hmac.New(sha256.New, s.cryptoService.key)
	mac.Write([]byte(payload))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(providedSig), []byte(expectedSig)) {
		return "", "", "", errors.New("invalid signature")
	}

	return contentType, contentID, filename, nil
}
