package hooks

import (
	"context"
	"net/http"
	"time"

	"ownprofile/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterAIHooks registers AI-related API endpoints
func RegisterAIHooks(app *pocketbase.PocketBase, ai *services.AIService, crypto *services.CryptoService) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Test AI provider connection
		se.Router.POST("/api/ai/test/{id}", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			}

			providerID := e.Request.PathValue("id")
			record, err := app.FindRecordById("ai_providers", providerID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "provider not found"})
			}

			// Decrypt API key
			apiKeyEnc := record.GetString("api_key_encrypted")
			apiKey, err := crypto.Decrypt(apiKeyEnc)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to decrypt API key"})
			}

			provider := &services.AIProvider{
				ID:      record.Id,
				Type:    record.GetString("type"),
				APIKey:  apiKey,
				BaseURL: record.GetString("base_url"),
				Model:   record.GetString("model"),
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err = ai.TestConnection(ctx, provider)
			if err != nil {
				record.Set("test_status", "error")
				record.Set("last_test", time.Now())
				app.Save(record)
				return e.JSON(http.StatusOK, map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				})
			}

			record.Set("test_status", "success")
			record.Set("last_test", time.Now())
			app.Save(record)

			return e.JSON(http.StatusOK, map[string]interface{}{
				"success": true,
			})
		}).Bind(apis.RequireAuth())

		// Enrich project content with AI
		se.Router.POST("/api/ai/enrich", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			}

			var req struct {
				ProviderID  string         `json:"provider_id"`
				Title       string         `json:"title"`
				Description string         `json:"description"`
				README      string         `json:"readme"`
				Languages   map[string]int `json:"languages"`
				Topics      []string       `json:"topics"`
				PrivacyMode string         `json:"privacy_mode"`
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			// Get provider
			record, err := app.FindRecordById("ai_providers", req.ProviderID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "provider not found"})
			}

			if !record.GetBool("is_active") {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "provider is not active"})
			}

			// Decrypt API key
			apiKeyEnc := record.GetString("api_key_encrypted")
			apiKey, err := crypto.Decrypt(apiKeyEnc)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to decrypt API key"})
			}

			provider := &services.AIProvider{
				ID:      record.Id,
				Type:    record.GetString("type"),
				APIKey:  apiKey,
				BaseURL: record.GetString("base_url"),
				Model:   record.GetString("model"),
			}

			enrichReq := &services.EnrichmentRequest{
				Title:       req.Title,
				Description: req.Description,
				README:      req.README,
				Languages:   req.Languages,
				Topics:      req.Topics,
				PrivacyMode: req.PrivacyMode,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			result, err := ai.EnrichProject(ctx, provider, enrichReq)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}

			return e.JSON(http.StatusOK, result)
		}).Bind(apis.RequireAuth())

		return se.Next()
	})

	// Hook to encrypt API keys before saving
	app.OnRecordCreate("ai_providers").BindFunc(func(e *core.RecordEvent) error {
		return encryptProviderKey(e.Record, crypto)
	})

	app.OnRecordUpdate("ai_providers").BindFunc(func(e *core.RecordEvent) error {
		return encryptProviderKey(e.Record, crypto)
	})
}

func encryptProviderKey(record *core.Record, crypto *services.CryptoService) error {
	// Check if there's a new API key to encrypt
	apiKey := record.GetString("api_key")
	if apiKey != "" && apiKey != "********" {
		encrypted, err := crypto.Encrypt(apiKey)
		if err != nil {
			return err
		}
		record.Set("api_key_encrypted", encrypted)
		record.Set("api_key", "") // Clear plaintext
	}
	return nil
}
