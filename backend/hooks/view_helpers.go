package hooks

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func getCollectionName(section string) string {
	switch section {
	case "experience":
		return "experience"
	case "projects":
		return "projects"
	case "education":
		return "education"
	case "certifications":
		return "certifications"
	case "awards":
		return "awards"
	case "skills":
		return "skills"
	case "posts":
		return "posts"
	case "talks":
		return "talks"
	case "contacts":
		return "contact_methods"
	case "testimonials":
		return "testimonials"
	default:
		return ""
	}
}

func getDefaultLayout(section string) string {
	switch section {
	case "experience":
		return "default"
	case "projects":
		return "grid-3"
	case "education":
		return "default"
	case "certifications":
		return "grouped"
	case "awards":
		return "grouped"
	case "skills":
		return "grouped"
	case "posts":
		return "grid-3"
	case "talks":
		return "default"
	case "contacts":
		return "vertical"
	case "testimonials":
		return "wall"
	default:
		return "default"
	}
}

func isRecordVisible(record *core.Record) bool {
	visibility := record.GetString("visibility")
	isDraft := record.GetBool("is_draft")
	return visibility != "private" && !isDraft
}

func isRecordVisibleForSection(record *core.Record, section string, viewId string) bool {
	viewVisibility := record.Get("view_visibility")

	if isRecordVisible(record) {
		if section == "contacts" && viewId != "" {
			if viewVisibility != nil {
				var vv map[string]interface{}
				switch v := viewVisibility.(type) {
				case map[string]interface{}:
					vv = v
				case types.JSONRaw:
					if len(v) > 0 && string(v) != "{}" && string(v) != "null" {
						json.Unmarshal(v, &vv)
					}
				case string:
					if v != "" && v != "{}" && v != "null" {
						json.Unmarshal([]byte(v), &vv)
					}
				}
				if vv != nil {
					if enabled, exists := vv[viewId]; exists {
						if b, ok := enabled.(bool); ok {
							return b
						}
					}
				}
			}
		}
		return true
	}

	return isVisibleInView(record, viewId)
}

func isVisibleInView(record *core.Record, viewId string) bool {
	if viewId == "" {
		return false
	}

	viewVisibility := record.Get("view_visibility")
	if viewVisibility == nil {
		return false
	}

	var vv map[string]interface{}

	switch v := viewVisibility.(type) {
	case map[string]interface{}:
		vv = v
	case types.JSONRaw:
		if len(v) == 0 || string(v) == "{}" || string(v) == "null" {
			return false
		}
		if err := json.Unmarshal(v, &vv); err != nil {
			log.Printf("[WARN] Failed to unmarshal types.JSONRaw for record %s: %v", record.Id, err)
			return false
		}
	case string:
		if v == "" || v == "{}" || v == "null" {
			return false
		}
		if err := json.Unmarshal([]byte(v), &vv); err != nil {
			log.Printf("[WARN] Failed to parse view_visibility JSON string for record %s: %v", record.Id, err)
			return false
		}
	default:
		log.Printf("[WARN] view_visibility unexpected type %T for record %s", viewVisibility, record.Id)
		return false
	}

	enabled, exists := vv[viewId]
	if !exists {
		return false
	}

	if b, ok := enabled.(bool); ok {
		return b
	}
	return false
}

func serializeRecords(records []*core.Record) []map[string]interface{} {
	var result []map[string]interface{}
	for _, record := range records {
		item := make(map[string]interface{})
		for key, value := range record.FieldsData() {
			if key == "password_hash" {
				continue
			}
			item[key] = value
		}
		item["id"] = record.Id
		result = append(result, item)
	}
	return result
}

func serializeRecordsWithOverrides(records []*core.Record, itemConfig map[string]map[string]interface{}, sectionName string) []map[string]interface{} {
	var result []map[string]interface{}
	overridableFields := getOverridableFields(sectionName)

	for _, record := range records {
		item := make(map[string]interface{})
		for key, value := range record.FieldsData() {
			if key == "password_hash" {
				continue
			}
			item[key] = value
		}
		item["id"] = record.Id

		if config, exists := itemConfig[record.Id]; exists {
			if overrides, ok := config["overrides"].(map[string]interface{}); ok {
				for field, value := range overrides {
					if containsString(overridableFields, field) {
						item[field] = value
					}
				}
			}
		}

		if sectionName == "projects" || sectionName == "posts" {
			if coverImage := record.GetString("cover_image"); coverImage != "" {
				collectionID := record.Collection().Id
				recordID := record.Id
				item["cover_image_url"] = "/api/files/" + collectionID + "/" + recordID + "/" + coverImage
				item["cover_image_large_url"] = "/api/files/" + collectionID + "/" + recordID + "/" + coverImage + "?thumb=1600x0"
				item["cover_image_thumb_url"] = "/api/files/" + collectionID + "/" + recordID + "/" + coverImage + "?thumb=480x0"
			}
		}

		result = append(result, item)
	}
	return result
}

func getOverridableFields(sectionName string) []string {
	switch sectionName {
	case "experience":
		return []string{"title", "description", "bullets"}
	case "projects":
		return []string{"title", "summary", "description"}
	case "education":
		return []string{"degree", "field", "description"}
	case "talks":
		return []string{"title", "description"}
	default:
		return []string{}
	}
}

func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func extractPasswordToken(e *core.RequestEvent) string {
	authHeader := e.Request.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return e.Request.Header.Get("X-Password-Token")
}

func extractShareToken(e *core.RequestEvent) string {
	if shareToken := e.Request.Header.Get("X-Share-Token"); shareToken != "" {
		return shareToken
	}
	return e.Request.URL.Query().Get("token")
}

func validateShareToken(app *pocketbase.PocketBase, share *services.ShareService, token string, viewID string) (bool, *core.Record) {
	if token == "" {
		return false, nil
	}

	tokenHMAC := share.HMACToken(token)
	tokenRecord, err := app.FindFirstRecordByFilter(
		"share_tokens",
		"token_hash = {:hash} && is_active = true",
		map[string]interface{}{"hash": tokenHMAC},
	)
	if err != nil || tokenRecord == nil {
		return false, nil
	}

	if tokenRecord.GetString("view_id") != viewID {
		return false, nil
	}

	expiresAt := tokenRecord.GetDateTime("expires_at")
	if !expiresAt.IsZero() && time.Now().After(expiresAt.Time()) {
		return false, nil
	}

	useCount := tokenRecord.GetInt("use_count")
	maxUses := tokenRecord.GetInt("max_uses")
	if maxUses > 0 && useCount >= maxUses {
		return false, nil
	}

	return true, tokenRecord
}

func resolveBaseURL(e *core.RequestEvent) string {
	if appURL := strings.TrimSpace(os.Getenv("APP_URL")); appURL != "" {
		return strings.TrimSuffix(appURL, "/")
	}

	req := e.Request
	proto := req.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if req.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}

	host := req.Host
	return fmt.Sprintf("%s://%s", proto, host)
}

func escapeICSText(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		";", "\\;",
		",", "\\,",
		"\n", "\\n",
	)
	return replacer.Replace(value)
}
