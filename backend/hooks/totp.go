package hooks

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

func RegisterTOTPHooks(app *pocketbase.PocketBase, crypto *services.CryptoService, rl *services.RateLimitService) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		// GET /api/totp/status — check if 2FA is enabled for the current user
		se.Router.GET("/api/totp/status", func(e *core.RequestEvent) error {
			user := e.Auth
			if user == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
			}

			enabled := user.GetBool("totp_enabled")

			// Also check if session is verified
			verified := false
			if enabled {
				nonce := user.GetString("totp_session_nonce")
				expiresStr := user.GetString("totp_session_expires")
				if nonce != "" && expiresStr != "" {
					expires, err := time.Parse("2006-01-02 15:04:05.000Z", expiresStr)
					if err == nil && time.Now().Before(expires) {
						verified = true
					}
				}
			}

			return e.JSON(http.StatusOK, map[string]any{
				"enabled":  enabled,
				"verified": verified,
			})
		}).Bind(apis.RequireAuth())

		// POST /api/totp/begin-setup — generate TOTP secret and return QR URL
		se.Router.POST("/api/totp/begin-setup", RateLimitMiddleware(rl, "strict")(func(e *core.RequestEvent) error {
			user := e.Auth
			if user == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
			}

			if user.GetBool("totp_enabled") {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "2FA is already enabled"})
			}

			key, err := totp.Generate(totp.GenerateOpts{
				Issuer:      "Facet",
				AccountName: user.Email(),
				Period:      30,
				Digits:      otp.DigitsSix,
				Algorithm:   otp.AlgorithmSHA1,
			})
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate TOTP secret"})
			}

			encSecret, err := crypto.Encrypt(key.Secret())
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to encrypt secret"})
			}

			user.Set("totp_secret", encSecret)
			if err := app.Save(user); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save secret"})
			}

			WriteAuditLog(app, "totp_begin_setup", "auth", user.Id, user.Email(), GetIP(e), GetUserAgent(e), nil)

			return e.JSON(http.StatusOK, map[string]any{
				"secret": key.Secret(),
				"url":    key.URL(),
			})
		})).Bind(apis.RequireAuth())

		// POST /api/totp/confirm-setup — validate code and enable 2FA
		se.Router.POST("/api/totp/confirm-setup", RateLimitMiddleware(rl, "strict")(func(e *core.RequestEvent) error {
			user := e.Auth
			if user == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
			}

			var req struct {
				Code string `json:"code"`
			}
			if err := e.BindBody(&req); err != nil || req.Code == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "Verification code is required"})
			}

			encSecret := user.GetString("totp_secret")
			if encSecret == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "No TOTP setup in progress. Call begin-setup first."})
			}

			secret, err := crypto.Decrypt(encSecret)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to decrypt secret"})
			}

			if valid, _ := totp.ValidateCustom(req.Code, secret, time.Now(), totp.ValidateOpts{
				Period:    30,
				Skew:      1,
				Digits:    otp.DigitsSix,
				Algorithm: otp.AlgorithmSHA1,
			}); !valid {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid verification code"})
			}

			plaintextCodes, hashedCodes, err := generateRecoveryCodes()
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate recovery codes"})
			}

			hashesJSON, err := json.Marshal(hashedCodes)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to encode recovery codes"})
			}
			encCodes, err := crypto.Encrypt(string(hashesJSON))
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to encrypt recovery codes"})
			}

			user.Set("totp_enabled", true)
			user.Set("totp_recovery_codes", encCodes)
			if err := app.Save(user); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to enable 2FA"})
			}

			WriteAuditLog(app, "totp_enabled", "auth", user.Id, user.Email(), GetIP(e), GetUserAgent(e), nil)

			return e.JSON(http.StatusOK, map[string]any{
				"recovery_codes": plaintextCodes,
			})
		})).Bind(apis.RequireAuth())

		// POST /api/totp/verify — verify TOTP code or recovery code to establish session
		se.Router.POST("/api/totp/verify", RateLimitMiddleware(rl, "strict")(func(e *core.RequestEvent) error {
			user := e.Auth
			if user == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
			}

			if !user.GetBool("totp_enabled") {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "2FA is not enabled"})
			}

			var req struct {
				Code string `json:"code"`
			}
			if err := e.BindBody(&req); err != nil || req.Code == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "Verification code is required"})
			}

			valid := false

			if isRecoveryCode(req.Code) {
				// Re-fetch user inside transaction to prevent race condition where
				// two concurrent requests could both consume the same recovery code
				err := app.RunInTransaction(func(txApp core.App) error {
					freshUser, err := txApp.FindRecordById("users", user.Id)
					if err != nil {
						return err
					}
					encCodes := freshUser.GetString("totp_recovery_codes")
					matched, newEncCodes, err := checkRecoveryCode(req.Code, encCodes, crypto)
					if err != nil {
						return err
					}
					if !matched {
						return fmt.Errorf("no match")
					}
					freshUser.Set("totp_recovery_codes", newEncCodes)
					return txApp.Save(freshUser)
				})
				if err == nil {
					valid = true
					// Re-fetch user so subsequent save has fresh data
					if freshUser, err := app.FindRecordById("users", user.Id); err == nil {
						user = freshUser
					}
				}
			} else {
				encSecret := user.GetString("totp_secret")
				if encSecret != "" {
					secret, err := crypto.Decrypt(encSecret)
					if err == nil {
						valid, _ = totp.ValidateCustom(req.Code, secret, time.Now(), totp.ValidateOpts{
							Period:    30,
							Skew:      1,
							Digits:    otp.DigitsSix,
							Algorithm: otp.AlgorithmSHA1,
						})
					}
				}
			}

			if !valid {
				WriteAuditLog(app, "totp_verify_failed", "auth", user.Id, user.Email(), GetIP(e), GetUserAgent(e), nil)
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid verification code"})
			}

			nonce, err := generateSessionNonce()
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create session"})
			}

			expires := time.Now().Add(24 * time.Hour)
			user.Set("totp_session_nonce", nonce)
			user.Set("totp_session_expires", expires.UTC().Format("2006-01-02 15:04:05.000Z"))
			if err := app.Save(user); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save session"})
			}

			WriteAuditLog(app, "totp_verified", "auth", user.Id, user.Email(), GetIP(e), GetUserAgent(e), nil)

			return e.JSON(http.StatusOK, map[string]any{
				"verified": true,
			})
		})).Bind(apis.RequireAuth())

		// POST /api/totp/disable — disable 2FA (requires valid code)
		se.Router.POST("/api/totp/disable", RateLimitMiddleware(rl, "strict")(func(e *core.RequestEvent) error {
			user := e.Auth
			if user == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
			}

			if !user.GetBool("totp_enabled") {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "2FA is not enabled"})
			}

			var req struct {
				Code string `json:"code"`
			}
			if err := e.BindBody(&req); err != nil || req.Code == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "Verification code is required"})
			}

			valid := false
			if isRecoveryCode(req.Code) {
				encCodes := user.GetString("totp_recovery_codes")
				matched, _, err := checkRecoveryCode(req.Code, encCodes, crypto)
				if err == nil && matched {
					valid = true
				}
			} else {
				encSecret := user.GetString("totp_secret")
				if encSecret != "" {
					secret, err := crypto.Decrypt(encSecret)
					if err == nil {
						valid, _ = totp.ValidateCustom(req.Code, secret, time.Now(), totp.ValidateOpts{
							Period:    30,
							Skew:      1,
							Digits:    otp.DigitsSix,
							Algorithm: otp.AlgorithmSHA1,
						})
					}
				}
			}

			if !valid {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid verification code"})
			}

			user.Set("totp_enabled", false)
			user.Set("totp_secret", "")
			user.Set("totp_recovery_codes", "")
			user.Set("totp_session_nonce", "")
			user.Set("totp_session_expires", "")
			if err := app.Save(user); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to disable 2FA"})
			}

			WriteAuditLog(app, "totp_disabled", "auth", user.Id, user.Email(), GetIP(e), GetUserAgent(e), nil)

			return e.JSON(http.StatusOK, map[string]any{"disabled": true})
		})).Bind(apis.RequireAuth())

		// POST /api/totp/regenerate-codes — generate new recovery codes (requires valid TOTP code)
		se.Router.POST("/api/totp/regenerate-codes", RateLimitMiddleware(rl, "strict")(func(e *core.RequestEvent) error {
			user := e.Auth
			if user == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
			}

			if !user.GetBool("totp_enabled") {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "2FA is not enabled"})
			}

			var req struct {
				Code string `json:"code"`
			}
			if err := e.BindBody(&req); err != nil || req.Code == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "Verification code is required"})
			}

			encSecret := user.GetString("totp_secret")
			if encSecret == "" {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "TOTP secret not found"})
			}

			secret, err := crypto.Decrypt(encSecret)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to decrypt secret"})
			}

			if valid, _ := totp.ValidateCustom(req.Code, secret, time.Now(), totp.ValidateOpts{
				Period:    30,
				Skew:      1,
				Digits:    otp.DigitsSix,
				Algorithm: otp.AlgorithmSHA1,
			}); !valid {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid verification code"})
			}

			plaintextCodes, hashedCodes, err := generateRecoveryCodes()
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate recovery codes"})
			}

			hashesJSON, err := json.Marshal(hashedCodes)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to encode recovery codes"})
			}
			encCodes, err := crypto.Encrypt(string(hashesJSON))
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to encrypt recovery codes"})
			}

			user.Set("totp_recovery_codes", encCodes)
			if err := app.Save(user); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save recovery codes"})
			}

			WriteAuditLog(app, "totp_codes_regenerated", "auth", user.Id, user.Email(), GetIP(e), GetUserAgent(e), nil)

			return e.JSON(http.StatusOK, map[string]any{
				"recovery_codes": plaintextCodes,
			})
		})).Bind(apis.RequireAuth())

		// POST /api/totp/clear-session — invalidate TOTP session (called on logout)
		se.Router.POST("/api/totp/clear-session", func(e *core.RequestEvent) error {
			user := e.Auth
			if user == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
			}

			user.Set("totp_session_nonce", "")
			user.Set("totp_session_expires", "")
			if err := app.Save(user); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to clear session"})
			}

			return e.JSON(http.StatusOK, map[string]any{"cleared": true})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

func generateRecoveryCodes() ([]string, []string, error) {
	codes := make([]string, 8)
	hashes := make([]string, 8)
	for i := 0; i < 8; i++ {
		b := make([]byte, 4)
		if _, err := rand.Read(b); err != nil {
			return nil, nil, err
		}
		code := fmt.Sprintf("%s-%s", hex.EncodeToString(b[:2]), hex.EncodeToString(b[2:]))
		codes[i] = code
		hash, err := bcrypt.GenerateFromPassword([]byte(code), 10)
		if err != nil {
			return nil, nil, err
		}
		hashes[i] = string(hash)
	}
	return codes, hashes, nil
}

func checkRecoveryCode(code string, encryptedCodes string, crypto *services.CryptoService) (bool, string, error) {
	if encryptedCodes == "" {
		return false, "", nil
	}
	decrypted, err := crypto.Decrypt(encryptedCodes)
	if err != nil {
		return false, "", err
	}
	var hashes []string
	if err := json.Unmarshal([]byte(decrypted), &hashes); err != nil {
		return false, "", err
	}
	// Normalize to lowercase — recovery codes are hex, users may type uppercase
	normalizedCode := strings.ToLower(code)
	for i, hash := range hashes {
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(normalizedCode)) == nil {
			hashes = append(hashes[:i], hashes[i+1:]...)
			newJSON, err := json.Marshal(hashes)
			if err != nil {
				return false, "", err
			}
			newEncrypted, err := crypto.Encrypt(string(newJSON))
			if err != nil {
				return false, "", err
			}
			return true, newEncrypted, nil
		}
	}
	return false, "", nil
}

func isRecoveryCode(code string) bool {
	return len(code) == 9 && code[4] == '-'
}

func generateSessionNonce() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
