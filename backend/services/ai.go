package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AIService handles AI provider interactions
type AIService struct {
	crypto *CryptoService
	client *http.Client
}

// AIProvider represents an AI provider configuration
type AIProvider struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"` // openai, anthropic, ollama, custom
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
	IsActive bool   `json:"is_active"`
}

// ContentImprovementResult represents the result of content improvement
type ContentImprovementResult struct {
	Content string `json:"content"`
}

// EnrichmentRequest represents a request to enrich project data
type EnrichmentRequest struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	README      string         `json:"readme"`
	Languages   map[string]int `json:"languages"`
	Topics      []string       `json:"topics"`
	PrivacyMode string         `json:"privacy_mode"` // full, summary, none
}

// EnrichmentResult represents AI-enriched project data
type EnrichmentResult struct {
	Summary        string   `json:"summary"`
	Bullets        []string `json:"bullets"`
	Tags           []string `json:"tags"`
	CaseStudy      string   `json:"case_study"`
	TechHighlights []string `json:"tech_highlights"`
}

// NewAIService creates a new AI service
func NewAIService(crypto *CryptoService) *AIService {
	return &AIService{
		crypto: crypto,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// EnrichProject enriches project data using AI
func (a *AIService) EnrichProject(ctx context.Context, provider *AIProvider, req *EnrichmentRequest) (*EnrichmentResult, error) {
	prompt := a.buildPrompt(req)

	switch provider.Type {
	case "openai", "custom":
		return a.callOpenAI(ctx, provider, prompt)
	case "anthropic":
		return a.callAnthropic(ctx, provider, prompt)
	case "ollama":
		return a.callOllama(ctx, provider, prompt)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", provider.Type)
	}
}

// TestConnection tests if a provider is configured correctly
func (a *AIService) TestConnection(ctx context.Context, provider *AIProvider) error {
	testPrompt := "Respond with exactly: OK"

	switch provider.Type {
	case "openai", "custom":
		_, err := a.callOpenAIRaw(ctx, provider, testPrompt)
		return err
	case "anthropic":
		_, err := a.callAnthropicRaw(ctx, provider, testPrompt)
		return err
	case "ollama":
		_, err := a.callOllamaRaw(ctx, provider, testPrompt)
		return err
	default:
		return fmt.Errorf("unsupported provider type: %s", provider.Type)
	}
}

// ImproveContent improves content using the AI provider with a custom prompt
func (a *AIService) ImproveContent(ctx context.Context, provider *AIProvider, prompt string) (string, error) {
	switch provider.Type {
	case "openai", "custom":
		return a.callOpenAIRaw(ctx, provider, prompt)
	case "anthropic":
		return a.callAnthropicRaw(ctx, provider, prompt)
	case "ollama":
		return a.callOllamaRaw(ctx, provider, prompt)
	default:
		return "", fmt.Errorf("unsupported provider type: %s", provider.Type)
	}
}

func (a *AIService) buildPrompt(req *EnrichmentRequest) string {
	var sb strings.Builder

	sb.WriteString(`You are helping create a professional portfolio entry for a software project. Generate content that is:
- Factual and based only on provided information
- Professional and neutral in tone
- Free of invented metrics, statistics, or claims not supported by the data
- Concise and impactful

Project Information:
`)

	sb.WriteString(fmt.Sprintf("Title: %s\n", req.Title))
	if req.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", req.Description))
	}

	if len(req.Languages) > 0 {
		var langs []string
		for lang := range req.Languages {
			langs = append(langs, lang)
		}
		sb.WriteString(fmt.Sprintf("Languages: %s\n", strings.Join(langs, ", ")))
	}

	if len(req.Topics) > 0 {
		sb.WriteString(fmt.Sprintf("Topics: %s\n", strings.Join(req.Topics, ", ")))
	}

	if req.README != "" && req.PrivacyMode == "full" {
		// Truncate README for prompt
		readme := req.README
		if len(readme) > 10000 {
			readme = readme[:10000] + "\n[Truncated]"
		}
		sb.WriteString(fmt.Sprintf("\nREADME:\n%s\n", readme))
	} else if req.README != "" && req.PrivacyMode == "summary" {
		// Just use first 500 chars as summary
		readme := req.README
		if len(readme) > 500 {
			readme = readme[:500] + "..."
		}
		sb.WriteString(fmt.Sprintf("\nREADME Summary:\n%s\n", readme))
	}

	sb.WriteString(`
Generate a JSON response with the following structure:
{
  "summary": "A 2-3 sentence professional summary of the project",
  "bullets": ["3-5 key features or achievements, based on actual project content"],
  "tags": ["relevant technology/category tags"],
  "tech_highlights": ["2-3 notable technical aspects"],
  "case_study": "A brief outline for a case study (3-4 bullet points)"
}

IMPORTANT: Only include information that can be derived from the provided data. Do not invent features, metrics, or claims.`)

	return sb.String()
}

func (a *AIService) callOpenAI(ctx context.Context, provider *AIProvider, prompt string) (*EnrichmentResult, error) {
	response, err := a.callOpenAIRaw(ctx, provider, prompt)
	if err != nil {
		return nil, err
	}

	return a.parseEnrichmentResponse(response)
}

func (a *AIService) callOpenAIRaw(ctx context.Context, provider *AIProvider, prompt string) (string, error) {
	baseURL := "https://api.openai.com/v1"
	if provider.BaseURL != "" {
		baseURL = strings.TrimSuffix(provider.BaseURL, "/")
	}

	model := provider.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return "", err
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

func (a *AIService) callAnthropic(ctx context.Context, provider *AIProvider, prompt string) (*EnrichmentResult, error) {
	response, err := a.callAnthropicRaw(ctx, provider, prompt)
	if err != nil {
		return nil, err
	}

	return a.parseEnrichmentResponse(response)
}

func (a *AIService) callAnthropicRaw(ctx context.Context, provider *AIProvider, prompt string) (string, error) {
	model := provider.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	reqBody := map[string]interface{}{
		"model":      model,
		"max_tokens": 2048,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", provider.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Anthropic API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var anthropicResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return "", err
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("no response from Anthropic")
	}

	return anthropicResp.Content[0].Text, nil
}

func (a *AIService) callOllama(ctx context.Context, provider *AIProvider, prompt string) (*EnrichmentResult, error) {
	response, err := a.callOllamaRaw(ctx, provider, prompt)
	if err != nil {
		return nil, err
	}

	return a.parseEnrichmentResponse(response)
}

func (a *AIService) callOllamaRaw(ctx context.Context, provider *AIProvider, prompt string) (string, error) {
	baseURL := provider.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	model := provider.Model
	if model == "" {
		model = "llama3.2"
	}

	reqBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp struct {
		Response string `json:"response"`
	}

	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return "", err
	}

	return ollamaResp.Response, nil
}

func (a *AIService) parseEnrichmentResponse(response string) (*EnrichmentResult, error) {
	// Try to extract JSON from response
	response = strings.TrimSpace(response)

	// Handle markdown code blocks
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	// Find JSON object in response
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start >= 0 && end > start {
		response = response[start : end+1]
	}

	// Use flexible parsing to handle AI returning arrays for some fields
	var rawResult map[string]interface{}
	if err := json.Unmarshal([]byte(response), &rawResult); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	result := &EnrichmentResult{}

	// Extract summary (string)
	if v, ok := rawResult["summary"].(string); ok {
		result.Summary = v
	}

	// Extract bullets (array of strings)
	if arr, ok := rawResult["bullets"].([]interface{}); ok {
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result.Bullets = append(result.Bullets, s)
			}
		}
	}

	// Extract tags (array of strings)
	if arr, ok := rawResult["tags"].([]interface{}); ok {
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result.Tags = append(result.Tags, s)
			}
		}
	}

	// Extract case_study (can be string or array - convert to string)
	switch v := rawResult["case_study"].(type) {
	case string:
		result.CaseStudy = v
	case []interface{}:
		var bullets []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				bullets = append(bullets, "• "+s)
			}
		}
		result.CaseStudy = strings.Join(bullets, "\n")
	}

	// Extract tech_highlights (array of strings)
	if arr, ok := rawResult["tech_highlights"].([]interface{}); ok {
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result.TechHighlights = append(result.TechHighlights, s)
			}
		}
	}

	return result, nil
}

// DecryptAPIKey decrypts an encrypted API key
func (a *AIService) DecryptAPIKey(encrypted string) (string, error) {
	return a.crypto.Decrypt(encrypted)
}

// EncryptAPIKey encrypts an API key for storage
func (a *AIService) EncryptAPIKey(key string) (string, error) {
	return a.crypto.Encrypt(key)
}
