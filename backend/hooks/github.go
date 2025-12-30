package hooks

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"ownprofile/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterGitHubHooks registers GitHub-related API endpoints
func RegisterGitHubHooks(app *pocketbase.PocketBase, github *services.GitHubService, ai *services.AIService) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Parse and preview a GitHub repo
		se.Router.POST("/api/github/preview", func(e *core.RequestEvent) error {
			// Require authentication
			if e.Auth == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			}

			var req struct {
				RepoURL string `json:"repo_url"`
				Token   string `json:"token"` // Optional GitHub PAT
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			owner, repo, err := github.ParseRepoURL(req.RepoURL)
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}

			metadata, err := github.FetchRepoMetadata(owner, repo, req.Token)
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}

			return e.JSON(http.StatusOK, metadata)
		}).Bind(RequireAuth(app))

		// Import a GitHub repo as a project
		se.Router.POST("/api/github/import", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			}

			var req struct {
				RepoURL      string `json:"repo_url"`
				Token        string `json:"token"`
				AIEnrich     bool   `json:"ai_enrich"`
				AIProviderID string `json:"ai_provider_id"`
				PrivacyMode  string `json:"privacy_mode"` // full, summary, none
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			owner, repo, err := github.ParseRepoURL(req.RepoURL)
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}

			// Fetch metadata
			metadata, err := github.FetchRepoMetadata(owner, repo, req.Token)
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}

			// Build proposed project data
			proposedData := map[string]interface{}{
				"title":       metadata.Name,
				"summary":     metadata.Description,
				"description": "",
				"tech_stack":  getTopLanguages(metadata.Languages, 10),
				"links": []map[string]string{
					{"type": "github", "url": metadata.HTMLURL},
				},
				"categories":  metadata.Topics,
				"visibility":  "draft",
				"is_draft":    true,
				"is_featured": false,
			}

			if metadata.Homepage != "" {
				links := proposedData["links"].([]map[string]string)
				links = append(links, map[string]string{"type": "website", "url": metadata.Homepage})
				proposedData["links"] = links
			}

			// AI enrichment if requested
			var aiResult *services.EnrichmentResult
			if req.AIEnrich && req.AIProviderID != "" {
				// Fetch AI provider
				providerRecord, err := app.FindRecordById("ai_providers", req.AIProviderID)
				if err == nil {
					// Decrypt API key
					apiKeyEnc := providerRecord.GetString("api_key_encrypted")
					apiKey, _ := ai.DecryptAPIKey(apiKeyEnc)

					provider := &services.AIProvider{
						ID:      providerRecord.Id,
						Type:    providerRecord.GetString("type"),
						APIKey:  apiKey,
						BaseURL: providerRecord.GetString("base_url"),
						Model:   providerRecord.GetString("model"),
					}

					enrichReq := &services.EnrichmentRequest{
						Title:       metadata.Name,
						Description: metadata.Description,
						README:      metadata.README,
						Languages:   metadata.Languages,
						Topics:      metadata.Topics,
						PrivacyMode: req.PrivacyMode,
					}

					ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
					defer cancel()

					aiResult, err = ai.EnrichProject(ctx, provider, enrichReq)
					if err != nil {
						// Log but don't fail - just skip enrichment
						app.Logger().Error("AI enrichment failed", "error", err)
					} else {
						// Apply AI enrichment
						proposedData["summary"] = aiResult.Summary
						if len(aiResult.Bullets) > 0 {
							proposedData["description"] = formatBullets(aiResult.Bullets)
						}
						if len(aiResult.Tags) > 0 {
							proposedData["categories"] = aiResult.Tags
						}
						if len(aiResult.TechHighlights) > 0 {
							techStack := proposedData["tech_stack"].([]string)
							for _, highlight := range aiResult.TechHighlights {
								if !contains(techStack, highlight) {
									techStack = append(techStack, highlight)
								}
							}
							proposedData["tech_stack"] = techStack
						}
					}
				}
			}

			// Create source record
			sourcesCollection, err := app.FindCollectionByNameOrId("sources")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "sources collection not found"})
			}

			sourceRecord := core.NewRecord(sourcesCollection)
			sourceRecord.Set("type", "github")
			sourceRecord.Set("identifier", metadata.FullName)
			sourceRecord.Set("sync_status", "pending")
			sourceRecord.Set("last_sync", time.Now())

			if err := app.Save(sourceRecord); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create source"})
			}

			// Create import proposal
			proposalsCollection, err := app.FindCollectionByNameOrId("import_proposals")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "proposals collection not found"})
			}

			proposedJSON, _ := json.Marshal(proposedData)

			proposalRecord := core.NewRecord(proposalsCollection)
			proposalRecord.Set("source_id", sourceRecord.Id)
			proposalRecord.Set("proposed_data", string(proposedJSON))
			proposalRecord.Set("ai_enriched", aiResult != nil)
			proposalRecord.Set("status", "pending")

			if err := app.Save(proposalRecord); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create proposal"})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"proposal_id": proposalRecord.Id,
				"source_id":   sourceRecord.Id,
				"proposed":    proposedData,
				"ai_enriched": aiResult != nil,
				"metadata":    metadata,
			})
		}).Bind(RequireAuth(app))

		// Refresh an existing source
		se.Router.POST("/api/github/refresh/{id}", func(e *core.RequestEvent) error {
			if e.Auth == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			}

			sourceID := e.Request.PathValue("id")
			sourceRecord, err := app.FindRecordById("sources", sourceID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "source not found"})
			}

			identifier := sourceRecord.GetString("identifier")
			owner, repo, err := github.ParseRepoURL(identifier)
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}

			// Get token if stored (encrypted)
			token := ""
			if tokenEnc := sourceRecord.GetString("github_token"); tokenEnc != "" {
				token, _ = ai.DecryptAPIKey(tokenEnc)
			}

			metadata, err := github.FetchRepoMetadata(owner, repo, token)
			if err != nil {
				sourceRecord.Set("sync_status", "error")
				sourceRecord.Set("sync_log", err.Error())
				app.Save(sourceRecord)
				return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}

			// Get existing project if linked
			var existingData map[string]interface{}
			if projectID := sourceRecord.GetString("project_id"); projectID != "" {
				projectRecord, err := app.FindRecordById("projects", projectID)
				if err == nil {
					existingData = map[string]interface{}{
						"title":       projectRecord.GetString("title"),
						"summary":     projectRecord.GetString("summary"),
						"description": projectRecord.GetString("description"),
						"tech_stack":  projectRecord.Get("tech_stack"),
						"categories":  projectRecord.Get("categories"),
					}
				}
			}

			// Build proposed data
			proposedData := map[string]interface{}{
				"title":      metadata.Name,
				"summary":    metadata.Description,
				"tech_stack": getTopLanguages(metadata.Languages, 10),
				"categories": metadata.Topics,
			}

			// Calculate diff
			diff := calculateDiff(existingData, proposedData)

			// Create proposal
			proposalsCollection, _ := app.FindCollectionByNameOrId("import_proposals")
			proposedJSON, _ := json.Marshal(proposedData)
			diffJSON, _ := json.Marshal(diff)

			proposalRecord := core.NewRecord(proposalsCollection)
			proposalRecord.Set("source_id", sourceRecord.Id)
			if projectID := sourceRecord.GetString("project_id"); projectID != "" {
				proposalRecord.Set("project_id", projectID)
			}
			proposalRecord.Set("proposed_data", string(proposedJSON))
			proposalRecord.Set("diff", string(diffJSON))
			proposalRecord.Set("ai_enriched", false)
			proposalRecord.Set("status", "pending")

			if err := app.Save(proposalRecord); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create proposal"})
			}

			// Update source sync status
			sourceRecord.Set("sync_status", "pending")
			sourceRecord.Set("last_sync", time.Now())
			app.Save(sourceRecord)

			return e.JSON(http.StatusOK, map[string]interface{}{
				"proposal_id": proposalRecord.Id,
				"diff":        diff,
				"proposed":    proposedData,
			})
		}).Bind(RequireAuth(app))

		return se.Next()
	})
}

func getTopLanguages(languages map[string]int, limit int) []string {
	// Simple implementation - just return keys
	var result []string
	for lang := range languages {
		result = append(result, lang)
		if len(result) >= limit {
			break
		}
	}
	return result
}

func formatBullets(bullets []string) string {
	result := ""
	for _, bullet := range bullets {
		result += "- " + bullet + "\n"
	}
	return result
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func calculateDiff(existing, proposed map[string]interface{}) map[string]interface{} {
	diff := make(map[string]interface{})
	for key, newVal := range proposed {
		oldVal, exists := existing[key]
		if !exists {
			diff[key] = map[string]interface{}{
				"type":  "added",
				"new":   newVal,
			}
		} else {
			oldJSON, _ := json.Marshal(oldVal)
			newJSON, _ := json.Marshal(newVal)
			if string(oldJSON) != string(newJSON) {
				diff[key] = map[string]interface{}{
					"type": "changed",
					"old":  oldVal,
					"new":  newVal,
				}
			}
		}
	}
	return diff
}
