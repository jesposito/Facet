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

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterAIHooks registers AI-related API endpoints
func RegisterAIHooks(app *pocketbase.PocketBase, ai *services.AIService, crypto *services.CryptoService) {
	// Auto-configure AI providers from environment variables on app start
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Try to auto-configure Anthropic from environment
		if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
			if err := ensureEnvProvider(app, crypto, "anthropic", "Anthropic (Auto)", apiKey, "", "claude-sonnet-4-20250514"); err != nil {
				log.Printf("Warning: Failed to auto-configure Anthropic provider: %v", err)
			} else {
				log.Println("AI: Auto-configured Anthropic provider from ANTHROPIC_API_KEY")
			}
		}

		// Try to auto-configure OpenAI from environment
		if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
			if err := ensureEnvProvider(app, crypto, "openai", "OpenAI (Auto)", apiKey, "", "gpt-4o-mini"); err != nil {
				log.Printf("Warning: Failed to auto-configure OpenAI provider: %v", err)
			} else {
				log.Println("AI: Auto-configured OpenAI provider from OPENAI_API_KEY")
			}
		}

		// Check AI status endpoint - tells frontend if AI is available
		se.Router.GET("/api/ai/status", func(e *core.RequestEvent) error {
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
	app.OnRecordCreate("ai_providers").BindFunc(func(e *core.RecordEvent) error {
		return encryptProviderKey(e.Record, crypto)
	})

	app.OnRecordUpdate("ai_providers").BindFunc(func(e *core.RecordEvent) error {
		return encryptProviderKey(e.Record, crypto)
	})
}

// ensureEnvProvider creates or updates an AI provider configured via environment variable
func ensureEnvProvider(app *pocketbase.PocketBase, crypto *services.CryptoService, providerType, name, apiKey, baseURL, model string) error {
	// Check if an env-configured provider of this type already exists
	providers, err := app.FindRecordsByFilter("ai_providers", fmt.Sprintf("type = '%s' && name ~ '(Auto)'", providerType), "", 1, 0)
	if err == nil && len(providers) > 0 {
		// Update existing provider
		record := providers[0]
		encrypted, err := crypto.Encrypt(apiKey)
		if err != nil {
			return err
		}
		record.Set("api_key_encrypted", encrypted)
		record.Set("model", model)
		record.Set("is_active", true)
		return app.Save(record)
	}

	// Create new provider
	collection, err := app.FindCollectionByNameOrId("ai_providers")
	if err != nil {
		return err
	}

	record := core.NewRecord(collection)
	record.Set("name", name)
	record.Set("type", providerType)
	record.Set("base_url", baseURL)
	record.Set("model", model)
	record.Set("is_active", true)

	// Check if any default exists, if not make this the default
	defaults, _ := app.FindRecordsByFilter("ai_providers", "is_default = true", "", 1, 0)
	record.Set("is_default", len(defaults) == 0)

	// Encrypt API key
	encrypted, err := crypto.Encrypt(apiKey)
	if err != nil {
		return err
	}
	record.Set("api_key_encrypted", encrypted)

	return app.Save(record)
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

	sb.WriteString("You are helping improve professional portfolio content. ")
	sb.WriteString("Be concise, professional, and factual. Do not invent information not provided.\n\n")

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
