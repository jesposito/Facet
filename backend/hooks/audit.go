package hooks

import (
	"net"
	"net/http"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
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
func RegisterAuditLogging(app *pocketbase.PocketBase) {
	// Log successful authentication events
	registerAuthAuditHooks(app)

	// Log CRUD operations on audited collections
	registerCRUDAuditHooks(app)

	// Register admin endpoint to fetch audit logs
	registerAuditEndpoint(app)
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

func registerAuditEndpoint(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// GET /api/admin/audit-logs - Fetch paginated audit logs
		se.Router.GET("/api/admin/audit-logs", func(e *core.RequestEvent) error {
			page := 1
			perPage := 50

			if p := e.Request.URL.Query().Get("page"); p != "" {
				if _, err := parsePositiveInt(p); err == nil {
					page, _ = parsePositiveInt(p)
				}
			}
			if pp := e.Request.URL.Query().Get("perPage"); pp != "" {
				if v, err := parsePositiveInt(pp); err == nil && v <= 200 {
					perPage = v
				}
			}

			// Build filter from query params
			filter := ""
			filterParams := map[string]any{}
			var countExprs []dbx.Expression

			if action := e.Request.URL.Query().Get("action"); action != "" {
				filter = "action = {:action}"
				filterParams["action"] = action
				countExprs = append(countExprs, dbx.NewExp("action = {:action}", dbx.Params{"action": action}))
			}
			if resourceType := e.Request.URL.Query().Get("resource_type"); resourceType != "" {
				if filter != "" {
					filter += " && "
				}
				filter += "resource_type = {:resource_type}"
				filterParams["resource_type"] = resourceType
				countExprs = append(countExprs, dbx.NewExp("resource_type = {:rt}", dbx.Params{"rt": resourceType}))
			}

			records, err := app.FindRecordsByFilter("audit_logs", filter, "-created", perPage, (page-1)*perPage, filterParams)
			if err != nil && !strings.Contains(err.Error(), "no rows") {
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to fetch audit logs",
				})
			}

			// Get total count for pagination (uses efficient COUNT query)
			totalRecords, _ := app.CountRecords("audit_logs", countExprs...)

			items := make([]map[string]any, 0, len(records))
			for _, r := range records {
				items = append(items, map[string]any{
					"id":            r.Id,
					"action":        r.GetString("action"),
					"resource_type": r.GetString("resource_type"),
					"resource_id":   r.GetString("resource_id"),
					"user_id":       r.GetString("user_id"),
					"user_email":    r.GetString("user_email"),
					"ip_address":    r.GetString("ip_address"),
					"user_agent":    r.GetString("user_agent"),
					"metadata":      r.Get("metadata"),
					"status":        r.GetString("status"),
					"created":       r.GetString("created"),
				})
			}

			return e.JSON(http.StatusOK, map[string]any{
				"items":      items,
				"page":       page,
				"perPage":    perPage,
				"totalItems": totalRecords,
				"totalPages": (totalRecords + int64(perPage) - 1) / int64(perPage),
			})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

// writeAuditLog creates an audit log entry. Failures are logged but never block the operation.
func writeAuditLog(app *pocketbase.PocketBase, action, resourceType, resourceID, userEmail, ipAddress, userAgent string, metadata map[string]any) {
	collection, err := app.FindCollectionByNameOrId("audit_logs")
	if err != nil {
		app.Logger().Warn("audit: collection not found", "error", err)
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
		app.Logger().Warn("audit: failed to write log", "error", err, "action", action, "resource_type", resourceType)
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

func parsePositiveInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, apis.NewBadRequestError("invalid integer", nil)
		}
		n = n*10 + int(c-'0')
	}
	if n <= 0 {
		return 1, nil
	}
	return n, nil
}
