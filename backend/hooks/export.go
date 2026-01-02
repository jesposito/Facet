package hooks

import (
	"encoding/json"
	"fmt"
	"net/http"
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

// ExportData contains all exportable profile data
type ExportData struct {
	Meta           ExportMeta               `json:"meta" yaml:"meta"`
	Profile        map[string]interface{}   `json:"profile,omitempty" yaml:"profile,omitempty"`
	Experience     []map[string]interface{} `json:"experience,omitempty" yaml:"experience,omitempty"`
	Projects       []map[string]interface{} `json:"projects,omitempty" yaml:"projects,omitempty"`
	Education      []map[string]interface{} `json:"education,omitempty" yaml:"education,omitempty"`
	Certifications []map[string]interface{} `json:"certifications,omitempty" yaml:"certifications,omitempty"`
	Awards         []map[string]interface{} `json:"awards,omitempty" yaml:"awards,omitempty"`
	Skills         []map[string]interface{} `json:"skills,omitempty" yaml:"skills,omitempty"`
	Posts          []map[string]interface{} `json:"posts,omitempty" yaml:"posts,omitempty"`
	Talks          []map[string]interface{} `json:"talks,omitempty" yaml:"talks,omitempty"`
	Views          []map[string]interface{} `json:"views,omitempty" yaml:"views,omitempty"`
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

// collectExportData gathers all exportable data from the database
func collectExportData(app *pocketbase.PocketBase) (*ExportData, error) {
	export := &ExportData{
		Meta: ExportMeta{
			Version:    "1.0.0",
			ExportedAt: time.Now().UTC().Format(time.RFC3339),
			App:        "Facet",
		},
	}

	// Profile (singleton)
	profileRecords, err := app.FindRecordsByFilter("profile", "", "", 1, 0, nil)
	if err == nil && len(profileRecords) > 0 {
		export.Profile = sanitizeRecord(profileRecords[0])
	}

	// Experience
	experienceRecords, err := app.FindRecordsByFilter("experience", "", "-sort_order,-start_date", 0, 0, nil)
	if err == nil {
		export.Experience = sanitizeRecords(experienceRecords)
	}

	// Projects
	projectRecords, err := app.FindRecordsByFilter("projects", "", "-is_featured,-sort_order", 0, 0, nil)
	if err == nil {
		export.Projects = sanitizeRecords(projectRecords)
	}

	// Education
	educationRecords, err := app.FindRecordsByFilter("education", "", "-sort_order,-end_date", 0, 0, nil)
	if err == nil {
		export.Education = sanitizeRecords(educationRecords)
	}

	// Certifications
	certRecords, err := app.FindRecordsByFilter("certifications", "", "issuer,sort_order,-issue_date", 0, 0, nil)
	if err == nil {
		export.Certifications = sanitizeRecords(certRecords)
	}

	// Awards
	awardRecords, err := app.FindRecordsByFilter("awards", "", "-sort_order,-awarded_at", 0, 0, nil)
	if err == nil {
		export.Awards = sanitizeRecords(awardRecords)
	}

	// Skills
	skillRecords, err := app.FindRecordsByFilter("skills", "", "category,sort_order", 0, 0, nil)
	if err == nil {
		export.Skills = sanitizeRecords(skillRecords)
	}

	// Posts
	postRecords, err := app.FindRecordsByFilter("posts", "", "-published_at", 0, 0, nil)
	if err == nil {
		export.Posts = sanitizeRecords(postRecords)
	}

	// Talks
	talkRecords, err := app.FindRecordsByFilter("talks", "", "-sort_order,-date", 0, 0, nil)
	if err == nil {
		export.Talks = sanitizeRecords(talkRecords)
	}

	// Views (exclude password hashes)
	viewRecords, err := app.FindRecordsByFilter("views", "", "name", 0, 0, nil)
	if err == nil {
		export.Views = sanitizeViewRecords(viewRecords)
	}

	return export, nil
}

// sanitizeRecord converts a PocketBase record to a map, removing internal fields
func sanitizeRecord(record *core.Record) map[string]interface{} {
	data := make(map[string]interface{})

	// Get all field values
	for key, value := range record.FieldsData() {
		// Skip internal PocketBase fields
		if key == "collectionId" || key == "collectionName" {
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

// sanitizeViewRecords converts view records, removing sensitive fields like password_hash
func sanitizeViewRecords(records []*core.Record) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		data := sanitizeRecord(record)
		// Remove password hash - user must re-set password after import
		delete(data, "password_hash")
		result = append(result, data)
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
