package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"io"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

// JWT constants
const (
	JWTIssuer       = "facet"
	JWTIssuerLegacy = "me.yaml" // Accept tokens from before rebrand
	JWTAudience     = "view-access"
)

// ViewAccessClaims represents the JWT claims for password-protected view access
type ViewAccessClaims struct {
	ViewID string `json:"vid"`
	jwt.RegisteredClaims
}

// CryptoService handles encryption/decryption of sensitive data
type CryptoService struct {
	key     []byte
	hmacKey []byte
	jwtKey  []byte
}

// NewCryptoService creates a new crypto service with the given key
func NewCryptoService(key string) *CryptoService {
	// Derive separate keys for encryption, HMAC, and JWT signing
	encKey := sha256.Sum256([]byte(key + ":encryption"))
	hmacKey := sha256.Sum256([]byte(key + ":hmac"))
	jwtKey := sha256.Sum256([]byte(key + ":jwt"))
	return &CryptoService{
		key:     encKey[:],
		hmacKey: hmacKey[:],
		jwtKey:  jwtKey[:],
	}
}

// Encrypt encrypts plaintext using AES-256-GCM
func (c *CryptoService) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func (c *CryptoService) Decrypt(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// HashPassword hashes a password using bcrypt
func (c *CryptoService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword compares a password with a hash
func (c *CryptoService) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateToken generates a cryptographically secure random token
// Uses URL-safe base64 encoding WITHOUT padding to avoid issues with = in URLs
func (c *CryptoService) GenerateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// HMACToken creates an HMAC-SHA256 of a token for secure storage
// Uses server secret as key, so DB leak doesn't allow offline verification
func (c *CryptoService) HMACToken(token string) string {
	h := hmac.New(sha256.New, c.hmacKey)
	h.Write([]byte(token))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// ValidateTokenHMAC compares a token against stored HMAC using constant-time comparison
func (c *CryptoService) ValidateTokenHMAC(token, storedHMAC string) bool {
	expectedHMAC := c.HMACToken(token)
	return subtle.ConstantTimeCompare([]byte(expectedHMAC), []byte(storedHMAC)) == 1
}

// GenerateViewAccessJWT creates a signed JWT for password-protected view access
// Returns the token string and expiration time
func (c *CryptoService) GenerateViewAccessJWT(viewID string, duration time.Duration) (string, time.Time, error) {
	// Generate random JWT ID for audit/future revocation
	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", time.Time{}, err
	}
	jti := base64.URLEncoding.EncodeToString(jtiBytes)

	now := time.Now()
	expiresAt := now.Add(duration)

	claims := ViewAccessClaims{
		ViewID: viewID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    JWTIssuer,
			Audience:  jwt.ClaimStrings{JWTAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(c.jwtKey)
	if err != nil {
		return "", time.Time{}, err
	}

	return signedToken, expiresAt, nil
}

// ValidateViewAccessJWT validates a JWT and returns the view ID if valid
// Returns an error if the token is invalid, expired, or tampered with
func (c *CryptoService) ValidateViewAccessJWT(tokenString string) (string, error) {
	if tokenString == "" {
		return "", errors.New("token required")
	}

	token, err := jwt.ParseWithClaims(tokenString, &ViewAccessClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return c.jwtKey, nil
	})

	if err != nil {
		return "", errors.New("invalid token")
	}

	claims, ok := token.Claims.(*ViewAccessClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token")
	}

	// Validate issuer (accept both current and legacy for backwards compatibility)
	if claims.Issuer != JWTIssuer && claims.Issuer != JWTIssuerLegacy {
		return "", errors.New("invalid issuer")
	}

	// Validate audience
	if !claims.VerifyAudience(JWTAudience, true) {
		return "", errors.New("invalid audience")
	}

	// Validate expiry (already checked by jwt library, but explicit)
	if claims.ExpiresAt == nil || claims.ExpiresAt.Before(time.Now()) {
		return "", errors.New("token expired")
	}

	// Validate view ID exists
	if claims.ViewID == "" {
		return "", errors.New("missing view ID")
	}

	return claims.ViewID, nil
}

// JWTShareVisitAudience marks a share visit proof. A separate audience keeps a
// visit proof from passing as a password view token, and the reverse.
const JWTShareVisitAudience = "share-visit"

// ShareVisitClaims identify one visit admitted through a share token. TokenID
// and TokenHash pin the proof to the share_tokens record that was spent.
type ShareVisitClaims struct {
	ViewID    string `json:"vid"`
	TokenID   string `json:"tid"`
	TokenHash string `json:"th"`
	jwt.RegisteredClaims
}

// ShareVisitTokenFingerprint condenses a stored token hash for a visit proof,
// so the proof doesn't carry the stored hash itself.
func (c *CryptoService) ShareVisitTokenFingerprint(tokenHash string) string {
	h := hmac.New(sha256.New, c.jwtKey)
	h.Write([]byte("share-visit:" + tokenHash))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// GenerateShareVisitProof signs a proof that a visit spent a use of a share
// token. It expires at expiresAt.
func (c *CryptoService) GenerateShareVisitProof(viewID, tokenID, tokenHash string, expiresAt time.Time) (string, error) {
	if viewID == "" || tokenID == "" || tokenHash == "" {
		return "", errors.New("missing share visit fields")
	}
	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", err
	}
	now := time.Now()
	claims := ShareVisitClaims{
		ViewID:    viewID,
		TokenID:   tokenID,
		TokenHash: c.ShareVisitTokenFingerprint(tokenHash),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    JWTIssuer,
			Audience:  jwt.ClaimStrings{JWTShareVisitAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        base64.RawURLEncoding.EncodeToString(jtiBytes),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(c.jwtKey)
}

// ValidateShareVisitProof checks a proof's signature, audience and expiry. The
// caller must still check the share token record it names.
func (c *CryptoService) ValidateShareVisitProof(tokenString string) (*ShareVisitClaims, error) {
	if tokenString == "" {
		return nil, errors.New("proof required")
	}
	claims := &ShareVisitClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("invalid signing method")
		}
		return c.jwtKey, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid proof")
	}
	if claims.Issuer != JWTIssuer || !claims.VerifyAudience(JWTShareVisitAudience, true) {
		return nil, errors.New("invalid proof")
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.After(time.Now()) {
		return nil, errors.New("proof expired")
	}
	if claims.ViewID == "" || claims.TokenID == "" || claims.TokenHash == "" {
		return nil, errors.New("invalid proof")
	}
	return claims, nil
}
