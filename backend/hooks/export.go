package hooks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"gopkg.in/yaml.v3"
)

// ExportMeta contains metadata about the export
type ExportMeta struct {
	Version    string `json:"version" yaml:"version"`
	ExportedAt string `json:"exported_at" yaml:"exported_at"`
	App        string `json:"app" yaml:"app"`
}

// ExportData contains all exportable data. Field ordering preserved from v1.0.0
// for backwards compatibility with existing parsers; new fields appended after.
//
// EXCLUDED for security/privacy reasons (see collectExportData):
//   - users, api_keys, share_tokens, email_verification_tokens (auth/credentials)
//   - audit_logs, access_logs, system_alerts (operator telemetry)
//   - webhook_deliveries (delivery audit; webhook configs are included)
//
// SENSITIVE FIELDS stripped from every record by sanitizeRecord:
//   - password, password_hash, *_secret, *_token_hash, api_key, encryption_key, private_key
type ExportData struct {
	Meta                      ExportMeta               `json:"meta" yaml:"meta"`
	Profile                   map[string]interface{}   `json:"profile,omitempty" yaml:"profile,omitempty"`
	Experience                []map[string]interface{} `json:"experience,omitempty" yaml:"experience,omitempty"`
	Projects                  []map[string]interface{} `json:"projects,omitempty" yaml:"projects,omitempty"`
	Education                 []map[string]interface{} `json:"education,omitempty" yaml:"education,omitempty"`
	Certifications            []map[string]interface{} `json:"certifications,omitempty" yaml:"certifications,omitempty"`
	Awards                    []map[string]interface{} `json:"awards,omitempty" yaml:"awards,omitempty"`
	Skills                    []map[string]interface{} `json:"skills,omitempty" yaml:"skills,omitempty"`
	Posts                     []map[string]interface{} `json:"posts,omitempty" yaml:"posts,omitempty"`
	Talks                     []map[string]interface{} `json:"talks,omitempty" yaml:"talks,omitempty"`
	Views                     []map[string]interface{} `json:"views,omitempty" yaml:"views,omitempty"`
	Testimonials              []map[string]interface{} `json:"testimonials,omitempty" yaml:"testimonials,omitempty"`
	TestimonialRequests       []map[string]interface{} `json:"testimonial_requests,omitempty" yaml:"testimonial_requests,omitempty"`
	ContactMethods            []map[string]interface{} `json:"contact_methods,omitempty" yaml:"contact_methods,omitempty"`
	Sources                   []map[string]interface{} `json:"sources,omitempty" yaml:"sources,omitempty"`
	CustomContent             []map[string]interface{} `json:"custom_content,omitempty" yaml:"custom_content,omitempty"`
	Courses                   []map[string]interface{} `json:"courses,omitempty" yaml:"courses,omitempty"`
	Modules                   []map[string]interface{} `json:"modules,omitempty" yaml:"modules,omitempty"`
	Lessons                   []map[string]interface{} `json:"lessons,omitempty" yaml:"lessons,omitempty"`
	CourseProgress            []map[string]interface{} `json:"course_progress,omitempty" yaml:"course_progress,omitempty"`
	LessonProgress            []map[string]interface{} `json:"lesson_progress,omitempty" yaml:"lesson_progress,omitempty"`
	CourseCertificates        []map[string]interface{} `json:"course_certificates,omitempty" yaml:"course_certificates,omitempty"`
	Quizzes                   []map[string]interface{} `json:"quizzes,omitempty" yaml:"quizzes,omitempty"`
	QuizQuestions             []map[string]interface{} `json:"quiz_questions,omitempty" yaml:"quiz_questions,omitempty"`
	QuizAttempts              []map[string]interface{} `json:"quiz_attempts,omitempty" yaml:"quiz_attempts,omitempty"`
	Comments                  []map[string]interface{} `json:"comments,omitempty" yaml:"comments,omitempty"`
	CommentReports            []map[string]interface{} `json:"comment_reports,omitempty" yaml:"comment_reports,omitempty"`
	Purchases                 []map[string]interface{} `json:"purchases,omitempty" yaml:"purchases,omitempty"`
	Coupons                   []map[string]interface{} `json:"coupons,omitempty" yaml:"coupons,omitempty"`
	NewsletterLists           []map[string]interface{} `json:"newsletter_lists,omitempty" yaml:"newsletter_lists,omitempty"`
	Subscribers               []map[string]interface{} `json:"subscribers,omitempty" yaml:"subscribers,omitempty"`
	SubscriberListMemberships []map[string]interface{} `json:"subscriber_list_memberships,omitempty" yaml:"subscriber_list_memberships,omitempty"`
	NewsletterSends           []map[string]interface{} `json:"newsletter_sends,omitempty" yaml:"newsletter_sends,omitempty"`
	NewsletterEvents          []map[string]interface{} `json:"newsletter_events,omitempty" yaml:"newsletter_events,omitempty"`
	Settings                  []map[string]interface{} `json:"settings,omitempty" yaml:"settings,omitempty"`
	SiteSettings              []map[string]interface{} `json:"site_settings,omitempty" yaml:"site_settings,omitempty"`
	MediaDisplayNames         []map[string]interface{} `json:"media_display_names,omitempty" yaml:"media_display_names,omitempty"`
	ExternalMedia             []map[string]interface{} `json:"external_media,omitempty" yaml:"external_media,omitempty"`
	AdminTags                 []map[string]interface{} `json:"admin_tags,omitempty" yaml:"admin_tags,omitempty"`
	AIProviders               []map[string]interface{} `json:"ai_providers,omitempty" yaml:"ai_providers,omitempty"`
	ImportProposals           []map[string]interface{} `json:"import_proposals,omitempty" yaml:"import_proposals,omitempty"`
	ResumeImports             []map[string]interface{} `json:"resume_imports,omitempty" yaml:"resume_imports,omitempty"`
	ViewExports               []map[string]interface{} `json:"view_exports,omitempty" yaml:"view_exports,omitempty"`
	Webhooks                  []map[string]interface{} `json:"webhooks,omitempty" yaml:"webhooks,omitempty"`
	DownloadLogs              []map[string]interface{} `json:"download_logs,omitempty" yaml:"download_logs,omitempty"`
	Uploads                   []map[string]interface{} `json:"uploads,omitempty" yaml:"uploads,omitempty"`
}

// RegisterExportHooks registers data export API endpoints
func RegisterExportHooks(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Export all data
		// GET /api/export?format=json|yaml
		se.Router.GET("/api/export", func(e *core.RequestEvent) error {
			format := e.Request.URL.Query().Get("format")
			if format == "" {
				format = "json"
			}

			if format != "json" && format != "yaml" {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "Invalid format. Use 'json' or 'yaml'.",
				})
			}

			// Collect all data
			exportData, err := collectExportData(app)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"error": fmt.Sprintf("Failed to collect data: %v", err),
				})
			}

			// Generate filename with timestamp
			timestamp := time.Now().Format("2006-01-02")
			filename := fmt.Sprintf("facet-export-%s.%s", timestamp, format)

			if format == "yaml" {
				return serveYAML(e, exportData, filename)
			}
			return serveJSON(e, exportData, filename)
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

// fetchAndSanitize loads all records from a collection and returns sanitized maps.
// Returns nil (omitted from JSON) on error or empty result. Errors are deliberately
// swallowed so a single missing/restricted collection cannot break the whole export.
//
// Robustness: if the requested sort references a field that does not exist on this
// collection (schema drift between branches), the query is retried with no sort
// rather than silently dropping the collection from the export.
func fetchAndSanitize(app core.App, collection, sort string) []map[string]interface{} {
	records, err := app.FindRecordsByFilter(collection, "", sort, 0, 0, nil)
	if err != nil && sort != "" {
		records, err = app.FindRecordsByFilter(collection, "", "", 0, 0, nil)
	}
	if err != nil || len(records) == 0 {
		return nil
	}
	return sanitizeRecords(records)
}

// collectExportData gathers all exportable data from the database.
// Excludes auth, credential, and operator-telemetry collections.
func collectExportData(app *pocketbase.PocketBase) (*ExportData, error) {
	export := &ExportData{
		Meta: ExportMeta{
			Version:    "1.1.0",
			ExportedAt: time.Now().UTC().Format(time.RFC3339),
			App:        "Facet",
		},
	}

	// Profile (singleton)
	profileRecords, err := app.FindRecordsByFilter("profile", "", "", 1, 0, nil)
	if err == nil && len(profileRecords) > 0 {
		export.Profile = sanitizeRecord(profileRecords[0])
	}

	// Resume / portfolio content (existing v1.0.0 fields)
	export.Experience = fetchAndSanitize(app, "experience", "-sort_order,-start_date")
	export.Projects = fetchAndSanitize(app, "projects", "-is_featured,-sort_order")
	export.Education = fetchAndSanitize(app, "education", "-sort_order,-end_date")
	export.Certifications = fetchAndSanitize(app, "certifications", "issuer,sort_order,-issue_date")
	export.Awards = fetchAndSanitize(app, "awards", "-sort_order,-awarded_at")
	export.Skills = fetchAndSanitize(app, "skills", "category,sort_order")
	export.Posts = fetchAndSanitize(app, "posts", "-published_at")
	export.Talks = fetchAndSanitize(app, "talks", "-sort_order,-date")

	// Views (excludes password hashes via sanitizeRecord field-strip below)
	viewRecords, err := app.FindRecordsByFilter("views", "", "name", 0, 0, nil)
	if err == nil && len(viewRecords) > 0 {
		export.Views = sanitizeRecords(viewRecords)
	}

	// Public content + social
	export.Testimonials = fetchAndSanitize(app, "testimonials", "-created")
	export.TestimonialRequests = fetchAndSanitize(app, "testimonial_requests", "-created")
	export.ContactMethods = fetchAndSanitize(app, "contact_methods", "sort_order")
	export.Sources = fetchAndSanitize(app, "sources", "")
	export.CustomContent = fetchAndSanitize(app, "custom_content", "sort_order")

	// Courses + learning
	export.Courses = fetchAndSanitize(app, "courses", "-sort_order,-created")
	export.Modules = fetchAndSanitize(app, "modules", "course,sort_order")
	export.Lessons = fetchAndSanitize(app, "lessons", "module,sort_order")
	export.CourseProgress = fetchAndSanitize(app, "course_progress", "-updated")
	export.LessonProgress = fetchAndSanitize(app, "lesson_progress", "-updated")
	export.CourseCertificates = fetchAndSanitize(app, "course_certificates", "-issued_at")
	export.Quizzes = fetchAndSanitize(app, "quizzes", "")
	export.QuizQuestions = fetchAndSanitize(app, "quiz_questions", "quiz,sort_order")
	export.QuizAttempts = fetchAndSanitize(app, "quiz_attempts", "-created")

	// Comments
	export.Comments = fetchAndSanitize(app, "comments", "-created")
	export.CommentReports = fetchAndSanitize(app, "comment_reports", "-created")

	// Commerce
	export.Purchases = fetchAndSanitize(app, "purchases", "-created")
	export.Coupons = fetchAndSanitize(app, "coupons", "code")

	// Newsletter
	export.NewsletterLists = fetchAndSanitize(app, "newsletter_lists", "name")
	export.Subscribers = fetchAndSanitize(app, "subscribers", "-created")
	export.SubscriberListMemberships = fetchAndSanitize(app, "subscriber_list_memberships", "")
	export.NewsletterSends = fetchAndSanitize(app, "newsletter_sends", "-created")
	export.NewsletterEvents = fetchAndSanitize(app, "newsletter_events", "-created")

	// Settings + configuration
	export.Settings = fetchAndSanitize(app, "settings", "")
	export.SiteSettings = fetchAndSanitize(app, "site_settings", "")

	// Media catalog + provenance
	export.MediaDisplayNames = fetchAndSanitize(app, "media_display_names", "")
	export.ExternalMedia = fetchAndSanitize(app, "external_media", "-created")
	export.AdminTags = fetchAndSanitize(app, "admin_tags", "name")
	export.Uploads = fetchAndSanitize(app, "uploads", "-created")

	// Operator-controlled data
	export.AIProviders = fetchAndSanitize(app, "ai_providers", "provider")
	export.ImportProposals = fetchAndSanitize(app, "import_proposals", "-created")
	export.ResumeImports = fetchAndSanitize(app, "resume_imports", "-created")
	export.ViewExports = fetchAndSanitize(app, "view_exports", "-created")
	export.Webhooks = fetchAndSanitize(app, "webhooks", "name")

	// Download analytics (user owns this data; counts which gated assets were accessed)
	export.DownloadLogs = fetchAndSanitize(app, "download_logs", "-created")

	return export, nil
}

// sensitiveFieldSubstrings is a defense-in-depth strip list applied to EVERY
// exported record. A field whose name contains any of these substrings (case-
// insensitive) is removed before export. This guards against future fields
// being added to a collection without an explicit per-collection sanitizer.
//
// Note: substring match — "api_key" catches "api_key", "stripe_api_key",
// "smtp_api_key", etc.
var sensitiveFieldSubstrings = []string{
	"password",
	"password_hash",
	"secret",
	"api_key",
	"apikey",
	"private_key",
	"encryption_key",
	"token_hash",
	"verification_token",
	"reset_token",
	"refresh_token",
	"access_token",
	"session_id",
	"totp_secret",
	"totp_secret_encrypted",
}

// isSensitiveField returns true when the field name matches a known-sensitive
// pattern and should be excluded from exports.
func isSensitiveField(name string) bool {
	lower := strings.ToLower(name)
	for _, needle := range sensitiveFieldSubstrings {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

// sanitizeRecord converts a PocketBase record to a map, removing internal
// fields and any field matching the sensitive-name strip list.
func sanitizeRecord(record *core.Record) map[string]interface{} {
	data := make(map[string]interface{})

	// Get all field values
	for key, value := range record.FieldsData() {
		// Skip internal PocketBase fields
		if key == "collectionId" || key == "collectionName" {
			continue
		}
		// Skip credential/token fields by name pattern (defense in depth)
		if isSensitiveField(key) {
			continue
		}
		data[key] = value
	}

	// Add the record ID (useful for references)
	data["id"] = record.Id

	return data
}

// sanitizeRecords converts multiple records to maps
func sanitizeRecords(records []*core.Record) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		result = append(result, sanitizeRecord(record))
	}
	return result
}

// serveJSON sends the export as a JSON file download
func serveJSON(e *core.RequestEvent, data *ExportData, filename string) error {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to serialize JSON: %v", err),
		})
	}

	e.Response.Header().Set("Content-Type", "application/json")
	e.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	e.Response.WriteHeader(http.StatusOK)
	e.Response.Write(jsonBytes)
	return nil
}

// serveYAML sends the export as a YAML file download
func serveYAML(e *core.RequestEvent, data *ExportData, filename string) error {
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to serialize YAML: %v", err),
		})
	}

	e.Response.Header().Set("Content-Type", "application/x-yaml")
	e.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	e.Response.WriteHeader(http.StatusOK)
	e.Response.Write(yamlBytes)
	return nil
}
