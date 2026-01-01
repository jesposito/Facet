package hooks

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"facet/services"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterAIHooks registers AI-related API endpoints
func RegisterAIHooks(app *pocketbase.PocketBase, ai *services.AIService, crypto *services.CryptoService) {
	// Register API endpoints on serve
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Check AI status endpoint - tells frontend if AI is available
		// Also auto-configures from environment on first call if needed
		se.Router.GET("/api/ai/status", func(e *core.RequestEvent) error {
			// First check if any providers exist
			allProviders, _ := app.FindRecordsByFilter("ai_providers", "1=1", "", 1, 0)

			// If no providers exist at all, try to auto-configure from environment
			if len(allProviders) == 0 {
				autoConfigureFromEnv(app, crypto)
			}

			providers, err := app.FindRecordsByFilter("ai_providers", "is_active = true", "", 1, 0)
			available := err == nil && len(providers) > 0

			var defaultProvider map[string]interface{}
			if available {
				for _, p := range providers {
					if p.GetBool("is_default") {
						defaultProvider = map[string]interface{}{
							"id":    p.Id,
							"name":  p.GetString("name"),
							"type":  p.GetString("type"),
							"model": p.GetString("model"),
						}
						break
					}
				}
				// If no default, use first available
				if defaultProvider == nil {
					p := providers[0]
					defaultProvider = map[string]interface{}{
						"id":    p.Id,
						"name":  p.GetString("name"),
						"type":  p.GetString("type"),
						"model": p.GetString("model"),
					}
				}
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"available":        available,
				"provider_count":   len(providers),
				"default_provider": defaultProvider,
			})
		})

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

			provider, err := getProviderFromRecord(record, crypto)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
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

		// General-purpose AI content improvement endpoint
		se.Router.POST("/api/ai/improve", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			}

			var req struct {
				ContentType string            `json:"content_type"` // headline, summary, description, bullets, experience, project, education
				Content     string            `json:"content"`      // Current content (can be empty)
				Context     map[string]string `json:"context"`      // Additional context like title, company, role, etc.
				Action      string            `json:"action"`       // improve, generate, expand, shorten
				ProviderID  string            `json:"provider_id"`  // Optional, uses default if not specified
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			// Get provider (specified or default)
			provider, err := getActiveProvider(app, crypto, req.ProviderID)
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}

			// Build the improvement prompt
			prompt := buildImprovementPrompt(req.ContentType, req.Content, req.Context, req.Action)

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			result, err := ai.ImproveContent(ctx, provider, prompt)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"improved_content": result,
				"provider":         provider.Name,
			})
		}).Bind(apis.RequireAuth())

		// Enrich project content with AI (existing endpoint)
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
			provider, err := getActiveProvider(app, crypto, req.ProviderID)
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
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
	// NOTE: Use OnRecordCreateRequest to access raw request body (hidden fields not auto-populated)
	// Must call e.Next() to continue the hook chain in PocketBase v0.23+
	app.OnRecordCreateRequest("ai_providers").BindFunc(func(e *core.RecordRequestEvent) error {
		log.Printf("[AI] OnRecordCreateRequest hook triggered for ai_providers")
		if err := encryptProviderKeyFromRequest(e, crypto); err != nil {
			log.Printf("[AI] Error in encryptProviderKeyFromRequest: %v", err)
			return err
		}
		log.Printf("[AI] Hook completed successfully, calling e.Next()")
		log.Printf("[AI] Record before save: name=%s, type=%s, model=%s, api_key_encrypted_len=%d",
			e.Record.GetString("name"),
			e.Record.GetString("type"),
			e.Record.GetString("model"),
			len(e.Record.GetString("api_key_encrypted")))
		if err := e.Next(); err != nil {
			log.Printf("[AI] e.Next() returned error: %v", err)
			log.Printf("[AI] Error type: %T", err)
			// Try to get more details from the error
			if apiErr, ok := err.(*apis.ApiError); ok {
				log.Printf("[AI] ApiError: status=%d, message=%s, data=%v", apiErr.Code, apiErr.Message, apiErr.RawData())
			}
			return err
		}
		log.Printf("[AI] Record created successfully with ID: %s", e.Record.Id)
		return nil
	})

	app.OnRecordUpdateRequest("ai_providers").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := encryptProviderKeyFromRequest(e, crypto); err != nil {
			return err
		}
		return e.Next()
	})
}

// autoConfigureFromEnv creates AI providers from environment variables
// Called from request context where saves work correctly
func autoConfigureFromEnv(app *pocketbase.PocketBase, crypto *services.CryptoService) {
	isFirstProvider := true

	// Check for Anthropic
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		if err := createProviderFromEnv(app, crypto, "anthropic", "Claude (Auto)", apiKey, "", "claude-sonnet-4-20250514", isFirstProvider); err != nil {
			log.Printf("[AI] Failed to auto-configure Anthropic: %v", err)
		} else {
			log.Printf("[AI] Auto-configured Anthropic Claude provider from environment")
			isFirstProvider = false
		}
	}

	// Check for OpenAI
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		if err := createProviderFromEnv(app, crypto, "openai", "OpenAI (Auto)", apiKey, "", "gpt-4o", isFirstProvider); err != nil {
			log.Printf("[AI] Failed to auto-configure OpenAI: %v", err)
		} else {
			log.Printf("[AI] Auto-configured OpenAI provider from environment")
			isFirstProvider = false
		}
	}

	// Check for Ollama
	if baseURL := os.Getenv("OLLAMA_BASE_URL"); baseURL != "" {
		model := os.Getenv("OLLAMA_MODEL")
		if model == "" {
			model = "llama3.2"
		}
		if err := createProviderFromEnv(app, crypto, "ollama", "Ollama (Auto)", "", baseURL, model, isFirstProvider); err != nil {
			log.Printf("[AI] Failed to auto-configure Ollama: %v", err)
		} else {
			log.Printf("[AI] Auto-configured Ollama provider from environment")
		}
	}
}

// createProviderFromEnv creates a single AI provider from environment config
// Uses direct SQL insert since app.Save() doesn't work in GET request handlers
func createProviderFromEnv(app *pocketbase.PocketBase, crypto *services.CryptoService, providerType, name, apiKey, baseURL, model string, isDefault bool) error {
	// Generate a unique ID
	id := fmt.Sprintf("%s%d", providerType[:3], time.Now().UnixNano()%1000000000)

	// Encrypt API key if provided
	var encryptedKey string
	if apiKey != "" {
		var err error
		encryptedKey, err = crypto.Encrypt(apiKey)
		if err != nil {
			return fmt.Errorf("encrypt: %w", err)
		}
	}

	// Use raw SQL insert since app.Save() doesn't work in GET handlers
	isDefaultInt := 0
	if isDefault {
		isDefaultInt = 1
	}

	query := `INSERT INTO ai_providers (id, name, type, model, base_url, api_key_encrypted, is_active, is_default)
	          VALUES ({:id}, {:name}, {:type}, {:model}, {:base_url}, {:api_key_encrypted}, 1, {:is_default})`

	_, err := app.DB().NewQuery(query).Bind(dbx.Params{
		"id":                id,
		"name":              name,
		"type":              providerType,
		"model":             model,
		"base_url":          baseURL,
		"api_key_encrypted": encryptedKey,
		"is_default":        isDefaultInt,
	}).Execute()
	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}

	log.Printf("[AI] Created provider via direct SQL: %s (id: %s)", name, id)
	return nil
}

// getActiveProvider gets a provider by ID or returns the default active provider
func getActiveProvider(app *pocketbase.PocketBase, crypto *services.CryptoService, providerID string) (*services.AIProvider, error) {
	var record *core.Record
	var err error

	if providerID != "" {
		record, err = app.FindRecordById("ai_providers", providerID)
		if err != nil {
			return nil, fmt.Errorf("provider not found")
		}
	} else {
		// Find default provider
		providers, err := app.FindRecordsByFilter("ai_providers", "is_default = true && is_active = true", "", 1, 0)
		if err != nil || len(providers) == 0 {
			// Fallback to any active provider
			providers, err = app.FindRecordsByFilter("ai_providers", "is_active = true", "", 1, 0)
			if err != nil || len(providers) == 0 {
				return nil, fmt.Errorf("no AI provider configured. Add one in Settings or set ANTHROPIC_API_KEY environment variable")
			}
		}
		record = providers[0]
	}

	if !record.GetBool("is_active") {
		return nil, fmt.Errorf("provider is not active")
	}

	return getProviderFromRecord(record, crypto)
}

// getProviderFromRecord creates an AIProvider from a database record
func getProviderFromRecord(record *core.Record, crypto *services.CryptoService) (*services.AIProvider, error) {
	apiKeyEnc := record.GetString("api_key_encrypted")
	apiKey, err := crypto.Decrypt(apiKeyEnc)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt API key")
	}

	return &services.AIProvider{
		ID:      record.Id,
		Name:    record.GetString("name"),
		Type:    record.GetString("type"),
		APIKey:  apiKey,
		BaseURL: record.GetString("base_url"),
		Model:   record.GetString("model"),
	}, nil
}

// buildImprovementPrompt creates a prompt for content improvement based on type and action
func buildImprovementPrompt(contentType, content string, ctx map[string]string, action string) string {
	var sb strings.Builder

	sb.WriteString(`You are helping improve professional portfolio content.
Be concise, professional, and factual. Do not invent information not provided.

IMPORTANT WRITING STYLE RULES:
- Write like a human, not an AI. Be direct and natural.
- NEVER use em dashes (—). Use commas, periods, or "and" instead.
- NEVER use words like "delve", "leverage", "utilize", "spearheaded", "synergy", "cutting-edge", "comprehensive", "robust"
- Avoid corporate buzzwords and marketing speak
- Use simple, clear language over fancy vocabulary
- Start sentences with action verbs, not "I" or "This"

`)

	// Add context
	if len(ctx) > 0 {
		sb.WriteString("Context:\n")
		for k, v := range ctx {
			if v != "" {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
			}
		}
		sb.WriteString("\n")
	}

	// Action-specific instructions
	switch action {
	case "generate":
		sb.WriteString("Generate new content based on the context provided.\n")
	case "expand":
		sb.WriteString("Expand and add more detail to the following content while maintaining accuracy.\n")
	case "shorten":
		sb.WriteString("Shorten the following content while keeping the key points.\n")
	default: // "improve"
		sb.WriteString("Improve the following content to be more professional and impactful.\n")
	}

	// Content type specific instructions
	switch contentType {
	case "headline":
		sb.WriteString("\nCreate a professional headline (one line, under 100 characters).\n")
		sb.WriteString("Format: [Role] | [Key strength/specialty] or [Role] at [Company type]\n")
	case "summary":
		sb.WriteString("\nCreate a professional summary (2-4 sentences).\n")
		sb.WriteString("Focus on expertise, experience level, and key achievements.\n")
	case "description":
		sb.WriteString("\nCreate a clear, professional description (2-3 paragraphs).\n")
		sb.WriteString("Use action verbs and quantify achievements where possible.\n")
	case "bullets":
		sb.WriteString("\nCreate 3-5 bullet points highlighting key achievements and responsibilities.\n")
		sb.WriteString("Start each with a strong action verb. Quantify results when possible.\n")
		sb.WriteString("Return as a JSON array of strings: [\"bullet1\", \"bullet2\", ...]\n")
	case "experience":
		sb.WriteString("\nCreate content for a work experience entry.\n")
		sb.WriteString("Include a brief description and 3-5 bullet points.\n")
		sb.WriteString("Return as JSON: {\"description\": \"...\", \"bullets\": [\"...\"]}\n")
	case "project":
		sb.WriteString("\nCreate content for a portfolio project.\n")
		sb.WriteString("Include a short summary and detailed description.\n")
		sb.WriteString("Return as JSON: {\"summary\": \"...\", \"description\": \"...\"}\n")
	case "education":
		sb.WriteString("\nCreate a brief description for an education entry.\n")
		sb.WriteString("Focus on relevant coursework, achievements, or thesis if provided.\n")
	}

	// Add current content if exists
	if content != "" {
		sb.WriteString("\nCurrent content to improve:\n")
		sb.WriteString(content)
		sb.WriteString("\n")
	}

	sb.WriteString("\nRespond with only the improved content, no explanations.")

	return sb.String()
}

// getMapKeys returns the keys of a map for logging
func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// encryptProviderKeyFromRequest reads api_key from request body and encrypts it
// Hidden fields aren't auto-populated into the record, so we read from request body
func encryptProviderKeyFromRequest(e *core.RecordRequestEvent, crypto *services.CryptoService) error {
	log.Printf("[AI] encryptProviderKeyFromRequest called")

	// In PocketBase v0.23+, we access the request info to get the body
	// because hidden fields (like api_key) aren't auto-populated into the record
	info, err := e.RequestInfo()
	if err != nil {
		log.Printf("[AI] No request info available: %v", err)
		return nil // No request info available
	}

	log.Printf("[AI] Request body keys: %v", getMapKeys(info.Body))

	apiKeyRaw, ok := info.Body["api_key"]
	if !ok {
		log.Printf("[AI] No api_key in request body")
		return nil // No api_key in request
	}

	apiKey, ok := apiKeyRaw.(string)
	if !ok || apiKey == "" || apiKey == "********" {
		log.Printf("[AI] api_key is empty or masked, skipping encryption")
		return nil // Empty or masked key, skip encryption
	}

	log.Printf("[AI] Encrypting api_key (length: %d)", len(apiKey))
	encrypted, err := crypto.Encrypt(apiKey)
	if err != nil {
		log.Printf("[AI] Encryption failed: %v", err)
		return err
	}

	log.Printf("[AI] Setting api_key_encrypted (length: %d)", len(encrypted))
	e.Record.Set("api_key_encrypted", encrypted)
	// Don't clear api_key here - let PocketBase handle the record as-is
	// The api_key field value will be stored but we use api_key_encrypted for actual operations
	// TODO: Consider making api_key a transient/virtual field that doesn't persist

	return nil
}
