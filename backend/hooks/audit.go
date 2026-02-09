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
	// Log successful authentication events
	registerAuthAuditHooks(app)

	// Log CRUD operations on audited collections
	registerCRUDAuditHooks(app)
}

func registerAuthAuditHooks(app *pocketbase.PocketBase) {
	// Log password-based login
	app.OnRecordAuthRequest("users").BindFunc(func(e *core.RecordAuthRequestEvent) error {
		if err := e.Next(); err != nil {
			return err
		}

		writeAuditLog(app, "login", "auth", e.Record.Id, e.Record.Email(), getIP(e.RequestEvent), getUserAgent(e.RequestEvent), map[string]any{
			"method": e.Record.Get("oauth2_provider") != nil,
		})
		return nil
	})
}

func registerCRUDAuditHooks(app *pocketbase.PocketBase) {
	for _, collection := range auditedCollections {
		col := collection // capture for closure

		app.OnRecordAfterCreateSuccess(col).BindFunc(func(e *core.RecordEvent) error {
			logRecordAction(app, "create", col, e.Record)
			return e.Next()
		})

		app.OnRecordAfterUpdateSuccess(col).BindFunc(func(e *core.RecordEvent) error {
			logRecordAction(app, "update", col, e.Record)
			return e.Next()
		})

		app.OnRecordAfterDeleteSuccess(col).BindFunc(func(e *core.RecordEvent) error {
			logRecordAction(app, "delete", col, e.Record)
			return e.Next()
		})
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
	} else {
		log.Printf("[AUDIT] logged: %s %s %s", action, resourceType, resourceID)
	}
}

func logRecordAction(app *pocketbase.PocketBase, action, collectionName string, record *core.Record) {
	// For CRUD events we don't have request context, so IP/UA are empty
	writeAuditLog(app, action, collectionName, record.Id, "", "", "", nil)
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

