package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// UserError represents an error with both user-friendly and technical details
type UserError struct {
	Message   string `json:"message"`   // User-friendly message
	Action    string `json:"action"`    // Suggested action for user
	Technical string `json:"technical"` // Technical details for support/debugging
}

// NewUserError creates a user-friendly error with technical details
func NewUserError(message, action, technical string) UserError {
	return UserError{
		Message:   message,
		Action:    action,
		Technical: technical,
	}
}

// RegisterResumeUploadHooks registers resume upload and parsing endpoints
func RegisterResumeUploadHooks(app *pocketbase.PocketBase, crypto *services.CryptoService) {
	ai := services.NewAIService(crypto)
	parser := services.NewResumeParser(ai)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		// Upload and parse resume
		// POST /api/resume/upload
		// Uploads PDF/DOCX resume, extracts text, parses with AI, creates records in main collections
		se.Router.POST("/api/resume/upload", func(e *core.RequestEvent) error {
			log.Println("[RESUME-UPLOAD] Starting resume upload")

			// Get uploaded file
			file, header, err := e.Request.FormFile("file")
			if err != nil {
				log.Printf("[RESUME-UPLOAD] Failed to get file: %v", err)
				return e.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": NewUserError(
						"We couldn't find a file to upload.",
						"Please select a PDF or DOCX resume file and try again.",
						fmt.Sprintf("File upload error: %v", err),
					),
				})
			}
			defer file.Close()

			// Check file size (5MB max)
			const maxSize = 5 * 1024 * 1024 // 5MB
			if header.Size > maxSize {
				log.Printf("[RESUME-UPLOAD] File too large: %d bytes", header.Size)
				fileSizeMB := float64(header.Size) / (1024 * 1024)
				return e.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": NewUserError(
						"Your resume file is too large.",
						"Please use a file smaller than 5MB. Try compressing images or saving as a simpler format.",
						fmt.Sprintf("File size: %.2f MB (maximum: 5 MB)", fileSizeMB),
					),
				})
			}

			// Check file type
			mimeType := header.Header.Get("Content-Type")
			if mimeType != "application/pdf" && mimeType != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
				log.Printf("[RESUME-UPLOAD] Invalid file type: %s", mimeType)
				return e.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": NewUserError(
						"This file type isn't supported.",
						"Please upload your resume as a PDF (.pdf) or Word document (.docx).",
						fmt.Sprintf("Unsupported file type: %s (filename: %s)", mimeType, header.Filename),
					),
				})
			}

			log.Printf("[RESUME-UPLOAD] File: %s, Size: %d bytes, Type: %s", header.Filename, header.Size, mimeType)

			// Get AI provider
			providerID := e.Request.FormValue("provider_id")
			provider, err := getActiveProvider(app, crypto, providerID)
			if err != nil {
				log.Printf("[RESUME-UPLOAD] No AI provider: %v", err)
				return e.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": NewUserError(
						"AI provider is not configured.",
						"Please go to Settings and configure an AI provider (OpenAI, Anthropic, or Ollama) to parse resumes.",
						fmt.Sprintf("Provider error: %v", err),
					),
				})
			}
			log.Printf("[RESUME-UPLOAD] Using AI provider: %s (%s)", provider.Name, provider.Type)

			// Read file bytes
			fileBytes, err := services.ReadFileBytes(file)
			if err != nil {
				log.Printf("[RESUME-UPLOAD] Failed to read file: %v", err)
				return e.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": NewUserError(
						"We couldn't read your file.",
						"This is unusual. Try uploading your file again, or try a different file format.",
						fmt.Sprintf("File read error: %v (filename: %s)", err, header.Filename),
					),
				})
			}

			// Extract text
			log.Println("[RESUME-UPLOAD] Extracting text from file...")
			resumeText, err := parser.ExtractText(fileBytes, mimeType)
			if err != nil {
				log.Printf("[RESUME-UPLOAD] Text extraction failed: %v", err)
				// Provide context-specific suggestions based on error
				action := "Try converting your resume to PDF format, or re-save your document and try again."
				if strings.Contains(err.Error(), "corrupted") {
					action = "Your file may be corrupted. Try opening it in Word/PDF viewer and saving a new copy."
				} else if strings.Contains(err.Error(), "no text found") {
					action = "Your file appears to contain only images. Try using a version with selectable text, or use OCR software first."
				}
				return e.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": NewUserError(
						"We couldn't extract text from your resume.",
						action,
						fmt.Sprintf("Text extraction error: %v", err),
					),
				})
			}

			log.Printf("[RESUME-UPLOAD] Extracted %d characters of text", len(resumeText))

			// Parse with AI
			log.Println("[RESUME-UPLOAD] Parsing resume with AI...")
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			parsed, err := parser.ParseResume(ctx, provider, resumeText)
			if err != nil {
				log.Printf("[RESUME-UPLOAD] AI parsing failed: %v", err)
				// Provide helpful suggestions based on error type
				action := "Try uploading your resume again. If the problem persists, try a simpler resume format."
				if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded") {
					action = "The AI service took too long to respond. Try again in a moment, or try a shorter resume."
				} else if strings.Contains(err.Error(), "JSON") {
					action = "The AI had trouble understanding your resume format. Try a simpler layout or contact support."
				}
				return e.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": NewUserError(
						"We couldn't parse your resume with AI.",
						action,
						fmt.Sprintf("AI parsing error: %v", err),
					),
				})
			}

			log.Printf("[RESUME-UPLOAD] Parsing complete. Confidence: %s, Warnings: %d", parsed.Metadata.Confidence, len(parsed.Metadata.Warnings))

			// Create records in collections
			log.Println("[RESUME-UPLOAD] Creating records in collections...")
			imported, err := createResumeRecords(app, parsed)
			if err != nil {
				log.Printf("[RESUME-UPLOAD] Failed to create records: %v", err)
				return e.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": NewUserError(
						"We parsed your resume but couldn't save the data.",
						"This might be a temporary database issue. Please try again in a moment.",
						fmt.Sprintf("Database error: %v", err),
					),
				})
			}

			log.Printf("[RESUME-UPLOAD] Successfully imported: %d experience, %d education, %d skills, %d certifications, %d projects",
				len(imported["experience"]), len(imported["education"]), len(imported["skills"]),
				len(imported["certifications"]), len(imported["projects"]))

			// Return success with import details
			return e.JSON(http.StatusOK, map[string]interface{}{
				"status":   "success",
				"imported": imported,
				"counts": map[string]int{
					"experience":     len(imported["experience"]),
					"education":      len(imported["education"]),
					"skills":         len(imported["skills"]),
					"certifications": len(imported["certifications"]),
					"projects":       len(imported["projects"]),
					"awards":         len(imported["awards"]),
					"talks":          len(imported["talks"]),
				},
				"warnings":   parsed.Metadata.Warnings,
				"confidence": parsed.Metadata.Confidence,
				"filename":   header.Filename,
			})
		}).Bind(apis.RequireAuth()) // Require authentication

		return se.Next()
	})
}

// createResumeRecords creates records in all relevant collections from parsed resume
func createResumeRecords(app *pocketbase.PocketBase, parsed *services.ParsedResume) (map[string][]string, error) {
	imported := make(map[string][]string)

	// Helper to get table name (always use main tables for resume upload)
	getTableName := func(baseName string) string {
		return baseName
	}

	// Create experience records
	if len(parsed.Experience) > 0 {
		expCollection, err := app.FindCollectionByNameOrId(getTableName("experience"))
		if err != nil {
			return nil, fmt.Errorf("experience collection not found: %w", err)
		}

		for _, exp := range parsed.Experience {
			record := core.NewRecord(expCollection)
			record.Set("company", exp.Company)
			record.Set("title", exp.Title)
			record.Set("location", exp.Location)
			record.Set("start_date", exp.StartDate)
			if exp.EndDate != "" && exp.EndDate != "null" {
				record.Set("end_date", exp.EndDate)
			}
			record.Set("description", exp.Description)
			if len(exp.Bullets) > 0 {
				record.Set("bullets", exp.Bullets)
			}
			if len(exp.Skills) > 0 {
				record.Set("skills", exp.Skills)
			}
			record.Set("visibility", "private") // Private by default
			record.Set("is_draft", false)
			record.Set("sort_order", 0)

			if err := app.Save(record); err != nil {
				log.Printf("[RESUME-UPLOAD] Failed to create experience: %v", err)
				continue
			}
			imported["experience"] = append(imported["experience"], record.Id)
		}
	}

	// Create education records
	if len(parsed.Education) > 0 {
		eduCollection, err := app.FindCollectionByNameOrId(getTableName("education"))
		if err != nil {
			return nil, fmt.Errorf("education collection not found: %w", err)
		}

		for _, edu := range parsed.Education {
			record := core.NewRecord(eduCollection)
			record.Set("institution", edu.Institution)
			record.Set("degree", edu.Degree)
			record.Set("field", edu.Field)
			record.Set("start_date", edu.StartDate)
			if edu.EndDate != "" && edu.EndDate != "null" {
				record.Set("end_date", edu.EndDate)
			}
			record.Set("description", edu.Description)
			record.Set("visibility", "private")
			record.Set("is_draft", false)
			record.Set("sort_order", 0)

			if err := app.Save(record); err != nil {
				log.Printf("[RESUME-UPLOAD] Failed to create education: %v", err)
				continue
			}
			imported["education"] = append(imported["education"], record.Id)
		}
	}

	// Create skills records
	if len(parsed.Skills) > 0 {
		skillsCollection, err := app.FindCollectionByNameOrId(getTableName("skills"))
		if err != nil {
			return nil, fmt.Errorf("skills collection not found: %w", err)
		}

		for _, skill := range parsed.Skills {
			record := core.NewRecord(skillsCollection)
			record.Set("name", skill.Name)
			record.Set("category", skill.Category)
			record.Set("proficiency", skill.Proficiency)
			record.Set("visibility", "private")
			record.Set("sort_order", 0)

			if err := app.Save(record); err != nil {
				log.Printf("[RESUME-UPLOAD] Failed to create skill: %v", err)
				continue
			}
			imported["skills"] = append(imported["skills"], record.Id)
		}
	}

	// Create certifications records
	if len(parsed.Certifications) > 0 {
		certsCollection, err := app.FindCollectionByNameOrId(getTableName("certifications"))
		if err != nil {
			return nil, fmt.Errorf("certifications collection not found: %w", err)
		}

		for _, cert := range parsed.Certifications {
			record := core.NewRecord(certsCollection)
			record.Set("name", cert.Name)
			record.Set("issuer", cert.Issuer)
			record.Set("issue_date", cert.IssueDate)
			if cert.ExpiryDate != "" && cert.ExpiryDate != "null" {
				record.Set("expiry_date", cert.ExpiryDate)
			}
			record.Set("credential_id", cert.CredentialID)
			record.Set("credential_url", cert.CredentialURL)
			record.Set("visibility", "private")
			record.Set("is_draft", false)
			record.Set("sort_order", 0)

			if err := app.Save(record); err != nil {
				log.Printf("[RESUME-UPLOAD] Failed to create certification: %v", err)
				continue
			}
			imported["certifications"] = append(imported["certifications"], record.Id)
		}
	}

	// Create projects records
	if len(parsed.Projects) > 0 {
		projectsCollection, err := app.FindCollectionByNameOrId(getTableName("projects"))
		if err != nil {
			return nil, fmt.Errorf("projects collection not found: %w", err)
		}

		for _, proj := range parsed.Projects {
			record := core.NewRecord(projectsCollection)
			record.Set("title", proj.Title)
			// Generate slug from title
			slug := generateSlug(proj.Title)
			record.Set("slug", slug)
			record.Set("summary", proj.Summary)
			record.Set("description", proj.Description)
			if len(proj.TechStack) > 0 {
				record.Set("tech_stack", proj.TechStack)
			}
			if len(proj.Links) > 0 {
				linksJSON, _ := json.Marshal(proj.Links)
				record.Set("links", string(linksJSON))
			}
			record.Set("visibility", "private")
			record.Set("is_draft", false)
			record.Set("is_featured", false)
			record.Set("sort_order", 0)

			if err := app.Save(record); err != nil {
				log.Printf("[RESUME-UPLOAD] Failed to create project: %v", err)
				continue
			}
			imported["projects"] = append(imported["projects"], record.Id)
		}
	}

	// Create awards records
	if len(parsed.Awards) > 0 {
		awardsCollection, err := app.FindCollectionByNameOrId(getTableName("awards"))
		if err != nil {
			log.Printf("[RESUME-UPLOAD] Awards collection not found (skipping): %v", err)
		} else {
			for _, award := range parsed.Awards {
				record := core.NewRecord(awardsCollection)
				record.Set("title", award.Title)
				record.Set("issuer", award.Issuer)
				record.Set("awarded_at", award.AwardedAt)
				record.Set("description", award.Description)
				record.Set("visibility", "private")
				record.Set("is_draft", false)
				record.Set("sort_order", 0)

				if err := app.Save(record); err != nil {
					log.Printf("[RESUME-UPLOAD] Failed to create award: %v", err)
					continue
				}
				imported["awards"] = append(imported["awards"], record.Id)
			}
		}
	}

	// Create talks records
	if len(parsed.Talks) > 0 {
		talksCollection, err := app.FindCollectionByNameOrId(getTableName("talks"))
		if err != nil {
			log.Printf("[RESUME-UPLOAD] Talks collection not found (skipping): %v", err)
		} else {
			for _, talk := range parsed.Talks {
				record := core.NewRecord(talksCollection)
				record.Set("title", talk.Title)
				slug := generateSlug(talk.Title)
				record.Set("slug", slug)
				record.Set("event", talk.Event)
				record.Set("date", talk.Date)
				record.Set("location", talk.Location)
				record.Set("description", talk.Description)
				record.Set("visibility", "private")
				record.Set("is_draft", false)
				record.Set("sort_order", 0)

				if err := app.Save(record); err != nil {
					log.Printf("[RESUME-UPLOAD] Failed to create talk: %v", err)
					continue
				}
				imported["talks"] = append(imported["talks"], record.Id)
			}
		}
	}

	return imported, nil
}

// generateSlug creates a URL-friendly slug from a title
func generateSlug(title string) string {
	// Convert to lowercase
	slug := strings.ToLower(title)
	// Replace spaces and special chars with hyphens
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, slug)
	// Remove consecutive hyphens
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	// Trim hyphens from ends
	slug = strings.Trim(slug, "-")
	// Limit length
	if len(slug) > 50 {
		slug = slug[:50]
	}
	return slug
}
