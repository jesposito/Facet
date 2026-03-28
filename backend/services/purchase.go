package services

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// PurchaseAccessClaims represents the JWT claims for content purchase access
type PurchaseAccessClaims struct {
	PurchaseIDs []string `json:"pids"`
	BuyerEmail  string   `json:"email"`
	jwt.RegisteredClaims
}

const (
	PurchaseJWTAudience = "content-purchase"
	PurchaseJWTDuration = 30 * 24 * time.Hour // 30 days
	PurchaseCookieName  = "facet_purchases"
)

// PurchaseService handles content purchase operations
type PurchaseService struct {
	crypto *CryptoService
}

// NewPurchaseService creates a new purchase service
func NewPurchaseService(crypto *CryptoService) *PurchaseService {
	return &PurchaseService{crypto: crypto}
}

// GenerateAccessToken generates a cryptographically secure random token for purchase access links
func (p *PurchaseService) GenerateAccessToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// GeneratePurchaseJWT creates a signed JWT for purchase access with multiple purchase IDs
func (p *PurchaseService) GeneratePurchaseJWT(purchaseIDs []string, buyerEmail string) (string, time.Time, error) {
	// Generate random JWT ID for audit/future revocation
	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", time.Time{}, err
	}
	jti := base64.URLEncoding.EncodeToString(jtiBytes)

	now := time.Now()
	expiresAt := now.Add(PurchaseJWTDuration)

	claims := PurchaseAccessClaims{
		PurchaseIDs: purchaseIDs,
		BuyerEmail:  buyerEmail,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    JWTIssuer,
			Audience:  jwt.ClaimStrings{PurchaseJWTAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(p.crypto.jwtKey)
	if err != nil {
		return "", time.Time{}, err
	}

	return signedToken, expiresAt, nil
}

// ValidatePurchaseJWT validates a JWT and returns the purchase IDs and buyer email if valid
func (p *PurchaseService) ValidatePurchaseJWT(tokenString string) ([]string, string, error) {
	if tokenString == "" {
		return nil, "", errors.New("token required")
	}

	token, err := jwt.ParseWithClaims(tokenString, &PurchaseAccessClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return p.crypto.jwtKey, nil
	})

	if err != nil {
		return nil, "", errors.New("invalid token")
	}

	claims, ok := token.Claims.(*PurchaseAccessClaims)
	if !ok || !token.Valid {
		return nil, "", errors.New("invalid token")
	}

	// Validate issuer (accept both current and legacy for backwards compatibility)
	if claims.Issuer != JWTIssuer && claims.Issuer != JWTIssuerLegacy {
		return nil, "", errors.New("invalid issuer")
	}

	// Validate audience
	if !claims.VerifyAudience(PurchaseJWTAudience, true) {
		return nil, "", errors.New("invalid audience")
	}

	// Validate expiry
	if claims.ExpiresAt == nil || claims.ExpiresAt.Before(time.Now()) {
		return nil, "", errors.New("token expired")
	}

	// Validate purchase IDs exist
	if len(claims.PurchaseIDs) == 0 {
		return nil, "", errors.New("missing purchase IDs")
	}

	return claims.PurchaseIDs, claims.BuyerEmail, nil
}

// GenerateEnrollmentJWT creates a signed JWT for free course enrollment.
// It uses a synthetic purchase ID ("free:<courseId>") so the token is compatible
// with the existing ValidatePurchaseJWT flow.
func (p *PurchaseService) GenerateEnrollmentJWT(courseID, buyerEmail string) (string, time.Time, error) {
	syntheticID := "free:" + courseID
	return p.GeneratePurchaseJWT([]string{syntheticID}, buyerEmail)
}

// HasPurchaseAccess checks if a purchase ID list contains access to specific content
func (p *PurchaseService) HasPurchaseAccess(purchaseIDs []string, targetPurchaseID string) bool {
	for _, id := range purchaseIDs {
		if id == targetPurchaseID {
			return true
		}
	}
	return false
}
