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

	"golang.org/x/crypto/bcrypt"
)

// CryptoService handles encryption/decryption of sensitive data
type CryptoService struct {
	key     []byte
	hmacKey []byte
}

// NewCryptoService creates a new crypto service with the given key
func NewCryptoService(key string) *CryptoService {
	// Derive separate keys for encryption and HMAC
	encKey := sha256.Sum256([]byte(key + ":encryption"))
	hmacKey := sha256.Sum256([]byte(key + ":hmac"))
	return &CryptoService{
		key:     encKey[:],
		hmacKey: hmacKey[:],
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
func (c *CryptoService) GenerateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
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
