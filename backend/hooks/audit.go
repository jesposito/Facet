package hooks

import (
	"log"
	"net"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// Collections to audit for CRUD operations.
// Excludes system/internal collections (audit_logs itself, demo_* collections, etc.)
var auditedCollections = []string{
	"profile",
	"experience",
	"education",
	"certifications",
	"awards",
	"skills",
	"projects",
	"posts",
	"talks",
	"testimonials",
	"contact_methods",
	"custom_content",
	"views",
	"share_tokens",
	"admin_tags",
	"ai_providers",
	"site_settings",
	"uploads",
	"external_media",
}

// RegisterAuditLogging registers hooks to log admin actions to the audit_logs collection.
// The audit_logs collection is read via PocketBase's native API (ListRule requires auth).
func RegisterAuditLogging(app *pocketbase.PocketBase) {
	registerAuthAuditHooks(app)
	registerCRUDAuditHooks(app)
}

func registerAuthAuditHooks(app *pocketbase.PocketBase) {
	// Only log actual password logins, not token refreshes.
	// OnRecordAuthWithPasswordRequest fires only on password auth,
	// whereas OnRecordAuthRequest fires on every auth including refreshes.
	app.OnRecordAuthWithPasswordRequest("users").BindFunc(func(e *core.RecordAuthWithPasswordRequestEvent) error {
		if err := e.Next(); err != nil {
			return err
		}

		if e.Record != nil {
			writeAuditLog(app, "login", "auth", e.Record.Id, e.Record.Email(), getIP(e.RequestEvent), getUserAgent(e.RequestEvent), nil)
		}
		return nil
	})

	// Log OAuth2 logins
	app.OnRecordAuthWithOAuth2Request("users").BindFunc(func(e *core.RecordAuthWithOAuth2RequestEvent) error {
		if err := e.Next(); err != nil {
			return err
		}

		if e.Record != nil {
			writeAuditLog(app, "login", "auth", e.Record.Id, e.Record.Email(), getIP(e.RequestEvent), getUserAgent(e.RequestEvent), map[string]any{
				"method": "oauth2",
			})
		}
		return nil
	})
}

func registerCRUDAuditHooks(app *pocketbase.PocketBase) {
	for _, collection := range auditedCollections {
		col := collection // capture for closure

		// Use request-level hooks to capture user email and IP address.
		// These fire after the API request succeeds (after e.Next() completes the operation).
		app.OnRecordCreateRequest(col).BindFunc(func(e *core.RecordRequestEvent) error {
			if err := e.Next(); err != nil {
				return err
			}
			logRequestAction(app, "create", col, e)
			return nil
		})

		app.OnRecordUpdateRequest(col).BindFunc(func(e *core.RecordRequestEvent) error {
			if err := e.Next(); err != nil {
				return err
			}
			logRequestAction(app, "update", col, e)
			return nil
		})

		app.OnRecordDeleteRequest(col).BindFunc(func(e *core.RecordRequestEvent) error {
			if err := e.Next(); err != nil {
				return err
			}
			logRequestAction(app, "delete", col, e)
			return nil
		})
	}
}

func logRequestAction(app *pocketbase.PocketBase, action, collectionName string, e *core.RecordRequestEvent) {
	email := ""
	if e.Auth != nil {
		email = e.Auth.Email()
	}

	resourceID := ""
	var metadata map[string]any
	if e.Record != nil {
		resourceID = e.Record.Id
		if label := extractLabel(collectionName, e.Record); label != "" {
			metadata = map[string]any{"label": label}
		}
	}

	writeAuditLog(app, action, collectionName, resourceID, email, getIP(e.RequestEvent), getUserAgent(e.RequestEvent), metadata)
}

// extractLabel pulls a human-readable label from a record based on its collection.
func extractLabel(collectionName string, record *core.Record) string {
	if record == nil {
		return ""
	}
	switch collectionName {
	case "profile":
		return record.GetString("name")
	case "experience":
		title := record.GetString("title")
		company := record.GetString("company")
		if title != "" && company != "" {
			return title + " at " + company
		}
		return title
	case "education":
		degree := record.GetString("degree")
		institution := record.GetString("institution")
		if degree != "" && institution != "" {
			return degree + " — " + institution
		}
		return institution
	case "certifications":
		return record.GetString("name")
	case "awards":
		return record.GetString("title")
	case "skills":
		name := record.GetString("name")
		category := record.GetString("category")
		if name != "" && category != "" {
			return name + " (" + category + ")"
		}
		return name
	case "projects":
		return record.GetString("title")
	case "posts":
		return record.GetString("title")
	case "talks":
		return record.GetString("title")
	case "testimonials":
		return record.GetString("author_name")
	case "contact_methods":
		label := record.GetString("label")
		ctype := record.GetString("type")
		if label != "" {
			return label
		}
		return ctype
	case "custom_content":
		return record.GetString("title")
	case "views":
		return record.GetString("name")
	case "share_tokens", "admin_tags", "ai_providers":
		return record.GetString("name")
	case "site_settings":
		return "Site Settings"
	case "uploads", "external_media":
		return record.GetString("display_name")
	default:
		return ""
	}
}

// writeAuditLog creates an audit log entry. Failures are logged but never block the operation.
func writeAuditLog(app *pocketbase.PocketBase, action, resourceType, resourceID, userEmail, ipAddress, userAgent string, metadata map[string]any) {
	collection, err := app.FindCollectionByNameOrId("audit_logs")
	if err != nil {
		log.Printf("[AUDIT] collection not found: %v", err)
		return
	}

	record := core.NewRecord(collection)
	record.Set("action", action)
	record.Set("resource_type", resourceType)
	record.Set("resource_id", resourceID)
	record.Set("user_email", userEmail)
	record.Set("ip_address", ipAddress)
	record.Set("user_agent", userAgent)
	record.Set("status", "success")

	if metadata != nil {
		record.Set("metadata", metadata)
	}

	if err := app.Save(record); err != nil {
		log.Printf("[AUDIT] failed to write log: %v (action=%s, resource_type=%s, resource_id=%s)", err, action, resourceType, resourceID)
	}
}

func getIP(e *core.RequestEvent) string {
	if e == nil || e.Request == nil {
		return ""
	}
	// Check X-Forwarded-For first (Caddy sets this)
	if xff := e.Request.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	// Fall back to RemoteAddr (handles IPv4 and IPv6 correctly)
	host, _, err := net.SplitHostPort(e.Request.RemoteAddr)
	if err != nil {
		return e.Request.RemoteAddr
	}
	return host
}

func getUserAgent(e *core.RequestEvent) string {
	if e == nil || e.Request == nil {
		return ""
	}
	ua := e.Request.Header.Get("User-Agent")
	if len(ua) > 500 {
		ua = ua[:500]
	}
	return ua
}

// WriteAuditLog exports audit logging for use by other hook files (e.g. TOTP)
func WriteAuditLog(app *pocketbase.PocketBase, action, resourceType, resourceID, userEmail, ipAddress, userAgent string, metadata map[string]any) {
	writeAuditLog(app, action, resourceType, resourceID, userEmail, ipAddress, userAgent, metadata)
}

// GetIP exports IP extraction for use by other hook files
func GetIP(e *core.RequestEvent) string {
	return getIP(e)
}

// GetUserAgent exports user agent extraction for use by other hook files
func GetUserAgent(e *core.RequestEvent) string {
	return getUserAgent(e)
}
