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

		// AI content rewrite with tone options
		se.Router.POST("/api/ai/rewrite", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			}

			var req struct {
				Content   string            `json:"content"`
				FieldType string            `json:"field_type"`
				Context   map[string]string `json:"context"`
				Tone      string            `json:"tone"` // executive, professional, technical, conversational, creative
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			if req.Content == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "content is required"})
			}

			// Get default provider
			provider, err := getActiveProvider(app, crypto, "")
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}

			// Build rewrite prompt with tone
			prompt := buildRewritePrompt(req.Content, req.FieldType, req.Context, req.Tone)

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			result, err := ai.ImproveContent(ctx, provider, prompt)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"content":  result,
				"tone":     req.Tone,
				"provider": provider.Name,
			})
		}).Bind(apis.RequireAuth())

		// AI content critique with inline feedback
		se.Router.POST("/api/ai/critique", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			}

			var req struct {
				Content   string            `json:"content"`
				FieldType string            `json:"field_type"`
				Context   map[string]string `json:"context"`
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			if req.Content == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "content is required"})
			}

			// Get default provider
			provider, err := getActiveProvider(app, crypto, "")
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}

			// Build critique prompt
			prompt := buildCritiquePrompt(req.Content, req.FieldType, req.Context)

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			result, err := ai.ImproveContent(ctx, provider, prompt)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"content":  result,
				"provider": provider.Name,
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
		if err := encryptProviderKeyFromRequest(e, crypto); err != nil {
			log.Printf("[AI] Provider create: encryption error: %v", err)
			return err
		}
		if err := e.Next(); err != nil {
			log.Printf("[AI] Provider create failed: %v", err)
			return err
		}
		return nil
	})

	app.OnRecordUpdateRequest("ai_providers").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := encryptProviderKeyFromRequest(e, crypto); err != nil {
			log.Printf("[AI] Provider update: encryption error: %v", err)
			return err
		}
		if err := e.Next(); err != nil {
			log.Printf("[AI] Provider update failed: %v", err)
			return err
		}
		return nil
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
			isFirstProvider = false
		}
	}

	// Check for OpenAI
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		if err := createProviderFromEnv(app, crypto, "openai", "OpenAI (Auto)", apiKey, "", "gpt-4o", isFirstProvider); err != nil {
			log.Printf("[AI] Failed to auto-configure OpenAI: %v", err)
		} else {
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

// buildRewritePrompt creates a tone-specific rewrite prompt
func buildRewritePrompt(content, fieldType string, ctx map[string]string, tone string) string {
	var sb strings.Builder

	sb.WriteString(`You are a professional writing assistant helping to rewrite portfolio content.

CRITICAL STYLE RULES - NEVER VIOLATE:
- Write like a human, not an AI. Be direct and natural.
- NEVER use em dashes (—). Use commas, periods, semicolons, or "and" instead.
- NEVER use these AI-sounding words: delve, leverage, utilize, spearheaded, synergy, cutting-edge,
  comprehensive, robust, streamline, optimize, revolutionize, game-changing, innovative (unless
  genuinely describing innovation), state-of-the-art, paradigm, holistic, seamless
- Avoid passive voice. Use active, strong verbs.
- Don't start sentences with "This project..." or "I was responsible for..."
- Vary sentence structure and length
- Be specific and concrete, not vague and abstract

`)

	// Add tone-specific instructions
	switch tone {
	case "executive":
		sb.WriteString(`TONE: Executive/C-Suite
- Focus on strategic impact, leadership, and business outcomes
- Emphasize ROI, revenue impact, cost savings, market position
- Use terminology like: "directed," "architected strategy," "drove," "delivered"
- Quantify everything possible (%, $, time saved, team size)
- 3rd person perspective preferred ("Led team of..." not "I led...")
- Example: "Led strategic initiative" NOT "Worked on project"

`)
	case "technical":
		sb.WriteString(`TONE: Technical/Engineering
- Focus on technologies, architectures, methodologies, technical challenges
- Be specific about tech stack, frameworks, design patterns
- Emphasize technical depth: "Implemented distributed caching using Redis" NOT "Made things faster"
- Use precise technical terms but avoid unnecessary jargon
- Explain the "how" and "why" of technical decisions
- First person active voice is fine ("Built," "Designed," "Implemented")

`)
	case "conversational":
		sb.WriteString(`TONE: Conversational/Approachable
- Use first person ("I built," "I designed")
- More personality, less corporate
- Contractions are okay (I'm, we're, it's)
- Shorter sentences, more direct communication
- Sound human and genuine, like explaining to a colleague
- Still professional, just more relaxed and personable

`)
	case "creative":
		sb.WriteString(`TONE: Creative/Portfolio Style
- Storytelling approach - what was the challenge, solution, impact?
- Emphasis on innovation, creativity, user experience
- More descriptive and engaging language
- Paint a picture of the work and its impact
- Show passion and craft, not just competence
- Balance creativity with credibility

`)
	default: // professional
		sb.WriteString(`TONE: Professional/Standard Resume
- Balanced, achievement-focused, industry-appropriate
- Clear action verbs: "Built," "Created," "Improved," "Managed"
- Quantify achievements where possible
- Professional but not stuffy, clear but not casual
- This is the safe, widely-appropriate choice

`)
	}

	// Add context if provided
	if len(ctx) > 0 {
		sb.WriteString("\nContext for this content:\n")
		for k, v := range ctx {
			if v != "" {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
			}
		}
		sb.WriteString("\n")
	}

	// Add field-specific guidance
	switch fieldType {
	case "headline":
		sb.WriteString("Rewrite this HEADLINE (one line, under 100 characters):\n\n")
	case "summary":
		sb.WriteString("Rewrite this SUMMARY (2-4 sentences capturing key professional value):\n\n")
	case "description":
		sb.WriteString("Rewrite this DESCRIPTION (1-3 paragraphs of detailed content):\n\n")
	case "bullets":
		sb.WriteString("Rewrite these BULLET POINTS (return as bullet list with • prefix):\n\n")
	default:
		sb.WriteString("Rewrite this content:\n\n")
	}

	sb.WriteString(content)
	sb.WriteString("\n\nReturn ONLY the rewritten content with no explanations, no preamble, no meta-commentary.")

	return sb.String()
}

// buildCritiquePrompt creates a prompt for inline feedback
func buildCritiquePrompt(content, fieldType string, ctx map[string]string) string {
	var sb strings.Builder

	sb.WriteString(`You are a professional writing coach providing inline feedback on portfolio content.

Your task: Return the EXACT original text with constructive feedback inserted in [square brackets].

FEEDBACK GUIDELINES:
- Be specific and actionable
- Point out vague language: [Too generic - what specific technology/approach?]
- Request quantification: [Can you quantify this? How much/many? What timeframe?]
- Flag buzzwords: [Avoid "leverage" - say "used" or be more specific]
- Suggest stronger verbs: [Passive voice - try "Built" or "Created" instead]
- Note missing context: [What was the impact? Who benefited?]
- Keep feedback brief and in brackets

EXAMPLE:
Original: "Worked on improving the application's performance using various techniques"
With feedback: "Worked on [Weak verb - try 'optimized' or 'redesigned'] improving the application's performance [By how much? 2x? 50% faster?] using various techniques [Too vague - name specific techniques like caching, query optimization, etc.]"

DO NOT:
- Rewrite the content
- Remove any original text
- Add feedback at the end
- Explain your feedback outside of brackets

`)

	// Add context if provided
	if len(ctx) > 0 {
		sb.WriteString("\nContext:\n")
		for k, v := range ctx {
			if v != "" {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Content to critique:\n\n")
	sb.WriteString(content)
	sb.WriteString("\n\nReturn the original text with inline [feedback in brackets].")

	return sb.String()
}

// encryptProviderKeyFromRequest reads api_key from request body and encrypts it
// The api_key field needs to be received from the request to encrypt, then cleared
func encryptProviderKeyFromRequest(e *core.RecordRequestEvent, crypto *services.CryptoService) error {
	// Access the request info to get the body data
	info, err := e.RequestInfo()
	if err != nil {
		return nil // No request info available
	}

	apiKeyRaw, ok := info.Body["api_key"]
	if !ok {
		return nil // No api_key in request
	}

	apiKey, ok := apiKeyRaw.(string)
	if !ok || apiKey == "" || apiKey == "********" {
		return nil // Empty or masked key, skip encryption
	}

	encrypted, err := crypto.Encrypt(apiKey)
	if err != nil {
		return fmt.Errorf("encrypt api key: %w", err)
	}

	e.Record.Set("api_key_encrypted", encrypted)
	e.Record.Set("api_key", "") // Clear plaintext from record

	return nil
}
