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

	"github.com/google/uuid"
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

			// Generate unique session ID for this import
			// This allows us to track which items came from which resume upload
			importSessionID := uuid.New().String()
			log.Printf("[RESUME-UPLOAD] Generated import session ID: %s", importSessionID)

			// Create records in collections with smart deduplication
			log.Println("[RESUME-UPLOAD] Creating records in collections...")
			imported, deduped, err := createResumeRecordsWithDeduplication(app, parsed, header.Filename, importSessionID)
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

			log.Printf("[RESUME-UPLOAD] Successfully imported: %d experience, %d education, %d skills (skipped %d duplicates)",
				len(imported["experience"]), len(imported["education"]), len(imported["skills"]), deduped)

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
				"deduplicated": deduped,
				"warnings":     parsed.Metadata.Warnings,
				"confidence":   parsed.Metadata.Confidence,
				"filename":     header.Filename,
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

// createResumeRecordsWithDeduplication creates records with smart deduplication and import metadata
//
// Deduplication Strategy (for faceted resume views):
// - Skills: Always dedupe across all imports (same skill is same skill, case-insensitive)
// - Experience: Dedupe by filename (same resume = dedupe, different resumes = different facets)
// - Education: Always dedupe across all imports (same degree is same credential)
// - Awards: Always dedupe across all imports (same award is same achievement)
// - Projects: Dedupe by filename (same resume = dedupe, different resumes = different facets)
//
// The filename-based deduplication for experience/projects allows:
// 1. Same resume imported multiple times → prevents duplicates
// 2. Different resumes with same role → creates faceted views
func createResumeRecordsWithDeduplication(app *pocketbase.PocketBase, parsed *services.ParsedResume, filename string, importSessionID string) (map[string][]string, int, error) {
	imported := make(map[string][]string)
	duplicateCount := 0

	log.Printf("[RESUME-UPLOAD] Using smart deduplication strategy for import session: %s", importSessionID)

	// Helper to get table name
	getTableName := func(baseName string) string {
		return baseName
	}

	// Create experience records with within-session deduplication only
	// Strategy: Different resumes = different facets, so we allow same job to appear multiple times
	// but dedupe within the same resume upload to avoid duplicates from parsing errors
	if len(parsed.Experience) > 0 {
		expCollection, err := app.FindCollectionByNameOrId(getTableName("experience"))
		if err != nil {
			return nil, 0, fmt.Errorf("experience collection not found: %w", err)
		}

		for _, exp := range parsed.Experience {
			// Check for duplicate within same resume (by filename)
			// This allows same role from DIFFERENT resumes (faceted views)
			// but prevents duplicates from same resume imported multiple times
			filter := fmt.Sprintf("company = '%s' && title = '%s' && start_date = '%s' && import_filename = '%s'",
				escapeFilter(exp.Company), escapeFilter(exp.Title), exp.StartDate, escapeFilter(filename))
			log.Printf("[RESUME-UPLOAD] [DEBUG] Checking for duplicate experience '%s at %s' with filter: %s", exp.Title, exp.Company, filter)
			existing, err := app.FindRecordsByFilter(expCollection.Name, filter, "", 0, 1)
			if err != nil {
				log.Printf("[RESUME-UPLOAD] [ERROR] Filter query failed for experience '%s at %s': %v", exp.Title, exp.Company, err)
				log.Printf("[RESUME-UPLOAD] [ERROR] Filter was: %s", filter)
			}
			if err == nil && len(existing) > 0 {
				log.Printf("[RESUME-UPLOAD] Skipping duplicate experience from same resume file: %s at %s (found %d existing)", exp.Title, exp.Company, len(existing))
				duplicateCount++
				continue
			}

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
			// Add import metadata for tracking and user visibility
			record.Set("import_session_id", importSessionID)
			record.Set("import_filename", filename)
			record.Set("visibility", "private")
			record.Set("is_draft", false)
			record.Set("sort_order", 0)

			if err := app.Save(record); err != nil {
				log.Printf("[RESUME-UPLOAD] Failed to create experience: %v", err)
				continue
			}
			imported["experience"] = append(imported["experience"], record.Id)
		}
	}

	// Create education records with cross-session deduplication
	// Strategy: Same degree = same credential, regardless of which resume it appears on
	// Match on institution + degree + field + end_date (graduation date)
	if len(parsed.Education) > 0 {
		eduCollection, err := app.FindCollectionByNameOrId(getTableName("education"))
		if err != nil {
			return nil, 0, fmt.Errorf("education collection not found: %w", err)
		}

		for _, edu := range parsed.Education {
			// Check for duplicate across ALL imports (not session-specific)
			// A degree from MIT is the same degree regardless of which resume lists it
			var filter string
			if edu.EndDate != "" && edu.EndDate != "null" {
				filter = fmt.Sprintf("institution = '%s' && degree = '%s' && field = '%s' && end_date = '%s'",
					escapeFilter(edu.Institution), escapeFilter(edu.Degree), escapeFilter(edu.Field), edu.EndDate)
			} else {
				filter = fmt.Sprintf("institution = '%s' && degree = '%s' && field = '%s'",
					escapeFilter(edu.Institution), escapeFilter(edu.Degree), escapeFilter(edu.Field))
			}

			existing, err := app.FindRecordsByFilter(eduCollection.Name, filter, "", 0, 1)
			if err == nil && len(existing) > 0 {
				log.Printf("[RESUME-UPLOAD] Skipping duplicate education: %s from %s", edu.Degree, edu.Institution)
				duplicateCount++
				continue
			}

			record := core.NewRecord(eduCollection)
			record.Set("institution", edu.Institution)
			record.Set("degree", edu.Degree)
			record.Set("field", edu.Field)
			record.Set("start_date", edu.StartDate)
			if edu.EndDate != "" && edu.EndDate != "null" {
				record.Set("end_date", edu.EndDate)
			}
			record.Set("description", edu.Description)
			record.Set("import_session_id", importSessionID)
			record.Set("import_filename", filename)
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

	// Create skills records with cross-session deduplication
	// Strategy: A skill is a skill - "Python" is "Python" regardless of which resume it appears on
	// Always dedupe across all imports (case-insensitive match)
	if len(parsed.Skills) > 0 {
		skillsCollection, err := app.FindCollectionByNameOrId(getTableName("skills"))
		if err != nil {
			return nil, 0, fmt.Errorf("skills collection not found: %w", err)
		}

		for _, skill := range parsed.Skills {
			// Check for duplicate across ALL imports (not session-specific)
			// Use PocketBase :lower modifier for case-insensitive matching
			filter := fmt.Sprintf("name:lower = '%s'", strings.ToLower(escapeFilter(skill.Name)))
			log.Printf("[RESUME-UPLOAD] [DEBUG] Checking for duplicate skill '%s' with filter: %s", skill.Name, filter)
			existing, err := app.FindRecordsByFilter(skillsCollection.Name, filter, "", 0, 1)
			if err != nil {
				log.Printf("[RESUME-UPLOAD] [ERROR] Filter query failed for skill '%s': %v", skill.Name, err)
				log.Printf("[RESUME-UPLOAD] [ERROR] Filter was: %s", filter)
			}
			if err == nil && len(existing) > 0 {
				log.Printf("[RESUME-UPLOAD] Skipping duplicate skill: %s (found %d existing)", skill.Name, len(existing))
				duplicateCount++
				continue
			}

			record := core.NewRecord(skillsCollection)
			record.Set("name", skill.Name)
			record.Set("category", skill.Category)
			record.Set("proficiency", skill.Proficiency)
			record.Set("import_session_id", importSessionID)
			record.Set("import_filename", filename)
			record.Set("visibility", "private")
			record.Set("sort_order", 0)

			if err := app.Save(record); err != nil {
				log.Printf("[RESUME-UPLOAD] Failed to create skill: %v", err)
				continue
			}
			imported["skills"] = append(imported["skills"], record.Id)
		}
	}

	// Create certifications records with deduplication
	if len(parsed.Certifications) > 0 {
		certsCollection, err := app.FindCollectionByNameOrId(getTableName("certifications"))
		if err != nil {
			return nil, 0, fmt.Errorf("certifications collection not found: %w", err)
		}

		for _, cert := range parsed.Certifications {
			// Check for duplicate: same name + issuer
			filter := fmt.Sprintf("name = '%s' && issuer = '%s'",
				escapeFilter(cert.Name), escapeFilter(cert.Issuer))
			existing, err := app.FindRecordsByFilter(certsCollection.Name, filter, "", 0, 1)
			if err == nil && len(existing) > 0 {
				log.Printf("[RESUME-UPLOAD] Skipping duplicate certification: %s", cert.Name)
				duplicateCount++
				continue
			}

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

	// Create projects records with within-session deduplication only
	// Strategy: Like experience, projects are faceted - same project may have different descriptions
	// for different audiences (e.g., emphasizing different tech stacks or outcomes)
	if len(parsed.Projects) > 0 {
		projectsCollection, err := app.FindCollectionByNameOrId(getTableName("projects"))
		if err != nil {
			return nil, 0, fmt.Errorf("projects collection not found: %w", err)
		}

		for _, proj := range parsed.Projects {
			// Check for duplicate within same resume (by filename)
			// This allows same project from DIFFERENT resumes (faceted views)
			// but prevents duplicates from same resume imported multiple times
			filter := fmt.Sprintf("title = '%s' && import_filename = '%s'", escapeFilter(proj.Title), escapeFilter(filename))
			log.Printf("[RESUME-UPLOAD] [DEBUG] Checking for duplicate project '%s' with filter: %s", proj.Title, filter)
			existing, err := app.FindRecordsByFilter(projectsCollection.Name, filter, "", 0, 1)
			if err != nil {
				log.Printf("[RESUME-UPLOAD] [ERROR] Filter query failed for project '%s': %v", proj.Title, err)
				log.Printf("[RESUME-UPLOAD] [ERROR] Filter was: %s", filter)
			}
			if err == nil && len(existing) > 0 {
				log.Printf("[RESUME-UPLOAD] Skipping duplicate project from same resume file: %s (found %d existing)", proj.Title, len(existing))
				duplicateCount++
				continue
			}

			record := core.NewRecord(projectsCollection)
			record.Set("title", proj.Title)
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
			record.Set("import_session_id", importSessionID)
			record.Set("import_filename", filename)
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

	// Create awards records with cross-session deduplication
	// Strategy: Same award = same achievement regardless of which resume it appears on
	// "Best Paper Award at ICML 2024" is the same award everywhere
	if len(parsed.Awards) > 0 {
		awardsCollection, err := app.FindCollectionByNameOrId(getTableName("awards"))
		if err != nil {
			log.Printf("[RESUME-UPLOAD] Awards collection not found (skipping): %v", err)
		} else {
			for _, award := range parsed.Awards {
				// Check for duplicate across ALL imports (not session-specific)
				// Match on title + issuer + date for uniqueness
				var filter string
				if award.AwardedAt != "" && award.AwardedAt != "null" {
					filter = fmt.Sprintf("title = '%s' && issuer = '%s' && awarded_at = '%s'",
						escapeFilter(award.Title), escapeFilter(award.Issuer), award.AwardedAt)
				} else {
					filter = fmt.Sprintf("title = '%s' && issuer = '%s'",
						escapeFilter(award.Title), escapeFilter(award.Issuer))
				}

				existing, err := app.FindRecordsByFilter(awardsCollection.Name, filter, "", 0, 1)
				if err == nil && len(existing) > 0 {
					log.Printf("[RESUME-UPLOAD] Skipping duplicate award: %s", award.Title)
					duplicateCount++
					continue
				}

				record := core.NewRecord(awardsCollection)
				record.Set("title", award.Title)
				record.Set("issuer", award.Issuer)
				record.Set("awarded_at", award.AwardedAt)
				record.Set("description", award.Description)
				record.Set("import_session_id", importSessionID)
				record.Set("import_filename", filename)
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

	// Create talks records with deduplication
	if len(parsed.Talks) > 0 {
		talksCollection, err := app.FindCollectionByNameOrId(getTableName("talks"))
		if err != nil {
			log.Printf("[RESUME-UPLOAD] Talks collection not found (skipping): %v", err)
		} else {
			for _, talk := range parsed.Talks {
				// Check for duplicate: same title + event
				filter := fmt.Sprintf("title = '%s' && event = '%s'",
					escapeFilter(talk.Title), escapeFilter(talk.Event))
				existing, err := app.FindRecordsByFilter(talksCollection.Name, filter, "", 0, 1)
				if err == nil && len(existing) > 0 {
					log.Printf("[RESUME-UPLOAD] Skipping duplicate talk: %s", talk.Title)
					duplicateCount++
					continue
				}

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

	return imported, duplicateCount, nil
}

// escapeFilter escapes single quotes for SQL filter strings
func escapeFilter(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// sanitizeFilename removes special characters from filename
func sanitizeFilename(filename string) string {
	// Remove extension
	name := strings.TrimSuffix(filename, ".pdf")
	name = strings.TrimSuffix(name, ".docx")
	// Replace special chars with underscores
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, name)
	// Limit length
	if len(name) > 30 {
		name = name[:30]
	}
	return name
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
