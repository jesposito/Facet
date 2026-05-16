package hooks

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// APIKeyPrefix is the prefix on every Facet self-hosted API key.
const APIKeyPrefix = "facet_"

// apiKeyRandBytes is the number of random bytes used in the key body. We hex-encode
// the bytes, so the displayed key body is 2x this number of hex characters.
const apiKeyRandBytes = 32

// keyDisplayPrefixLen is the number of characters from the start of the issued key
// that we store in `key_prefix` for display + fast lookup.
//
// e.g. "facet_a1b2c3d4..." -> stored prefix "facet_a1b2c3" (12 chars total).
const keyDisplayPrefixLen = 12

// lastUsedDebounceWindow keeps us from hammering SQLite with last_used_at writes
// on every single API call. We only update last_used_at if the cached value is
// older than this window.
const lastUsedDebounceWindow = time.Minute

var (
	lastUsedMu    sync.RWMutex
	lastUsedCache = make(map[string]time.Time)
)

// Valid scopes for the self-hosted public API. Keep this list short and read-only
// for now; write scopes can be added in a follow-up.
//
// IMPORTANT: keep these in sync with the i18n keys under `admin.api_keys.scope_*`
// and with the matching frontend list in /admin/api/+page.svelte.
var validScopes = map[string]bool{
	"read:profile":    true,
	"read:posts":      true,
	"read:projects":   true,
	"read:skills":     true,
	"read:experience": true,
}

// generateAPIKey returns a freshly-minted key like "facet_<64 hex chars>".
// The full key is returned ONCE to the caller; only the sha256 hash is stored.
func generateAPIKey() (string, error) {
	buf := make([]byte, apiKeyRandBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return APIKeyPrefix + hex.EncodeToString(buf), nil
}

// hashAPIKey returns the lowercase hex sha256 of the full key. This is what we
// store in the `key_hash` column.
func hashAPIKey(fullKey string) string {
	sum := sha256.Sum256([]byte(fullKey))
	return hex.EncodeToString(sum[:])
}

// extractKeyPrefix returns the first `keyDisplayPrefixLen` characters of a key,
// or the whole key if shorter. We store this for fast lookup and for display in
// the admin UI (so the user can identify a key from its prefix alone).
func extractKeyPrefix(fullKey string) string {
	if len(fullKey) <= keyDisplayPrefixLen {
		return fullKey
	}
	return fullKey[:keyDisplayPrefixLen]
}

// hasScope returns true if `granted` contains `required`.
func hasScope(granted []string, required string) bool {
	for _, s := range granted {
		if s == required {
			return true
		}
	}
	return false
}

// decodeJSONStringSlice safely decodes a PocketBase JSONField value into []string.
// Returns an empty slice if the field is nil or malformed.
func decodeJSONStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	bytes, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var out []string
	if err := json.Unmarshal(bytes, &out); err != nil {
		return nil
	}
	return out
}

// updateLastUsedAsync writes the last_used_at timestamp for a key. Debounced via
// lastUsedCache so we don't write on every single request.
func updateLastUsedAsync(app core.App, keyID string) {
	lastUsedMu.RLock()
	last, ok := lastUsedCache[keyID]
	lastUsedMu.RUnlock()
	if ok && time.Since(last) < lastUsedDebounceWindow {
		return
	}

	lastUsedMu.Lock()
	lastUsedCache[keyID] = time.Now()
	lastUsedMu.Unlock()

	record, err := app.FindRecordById("api_keys", keyID)
	if err != nil {
		return
	}
	record.Set("last_used_at", time.Now().UTC().Format("2006-01-02 15:04:05.000Z"))
	_ = app.Save(record)
}

// APIKeyMiddleware authenticates a request by Authorization: Bearer <key>, looks
// up the matching api_keys row by hash, and enforces is_active + expires_at +
// scope checks. It also sets the request context with the key id + granted scopes.
//
// Self-hosted is single-tenant, so there's no per-tenant rate limit or member
// tier — just the scope check.
func APIKeyMiddleware(app core.App, requiredScope string) func(func(*core.RequestEvent) error) func(*core.RequestEvent) error {
	return func(handler func(*core.RequestEvent) error) func(*core.RequestEvent) error {
		return func(e *core.RequestEvent) error {
			authHeader := e.Request.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				return respondUnauthorized(e, "missing or invalid Authorization header")
			}
			fullKey := strings.TrimPrefix(authHeader, "Bearer ")
			if !strings.HasPrefix(fullKey, APIKeyPrefix) {
				return respondUnauthorized(e, "invalid API key format")
			}

			// Hash and look up. We index on key_prefix so this is O(1) typically,
			// but in case of prefix collisions we filter further on key_hash.
			hash := hashAPIKey(fullKey)
			prefix := extractKeyPrefix(fullKey)

			records, err := app.FindRecordsByFilter(
				"api_keys",
				"key_prefix = {:prefix} && key_hash = {:hash}",
				"",
				1,
				0,
				map[string]any{"prefix": prefix, "hash": hash},
			)
			if err != nil || len(records) == 0 {
				return respondUnauthorized(e, "invalid API key")
			}
			key := records[0]

			if !key.GetBool("is_active") {
				return respondForbidden(e, "API key has been revoked")
			}
			if exp := key.GetDateTime("expires_at"); !exp.IsZero() && exp.Time().Before(time.Now()) {
				return respondForbidden(e, "API key has expired")
			}

			scopes := decodeJSONStringSlice(key.Get("scopes"))
			if !hasScope(scopes, requiredScope) {
				return respondForbidden(e, "insufficient scope")
			}

			e.Set("api_key_id", key.Id)
			e.Set("api_key_scopes", scopes)

			go updateLastUsedAsync(app, key.Id)

			return handler(e)
		}
	}
}

// RegisterAPIKeyHooks wires up the admin CRUD endpoints for managing API keys
// and the public read-only v1 endpoints that consume them.
func RegisterAPIKeyHooks(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// ---------------------------------------------------------------
		// Admin endpoints (require superuser)
		// ---------------------------------------------------------------

		// POST /api/admin/api-keys
		// Creates a new key. Returns the FULL key in the response ONCE.
		se.Router.POST("/api/admin/api-keys", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			var req struct {
				Label          string   `json:"label"`
				Scopes         []string `json:"scopes"`
				AllowedOrigins []string `json:"allowed_origins"`
				ExpiresAt      string   `json:"expires_at"`
			}
			if err := e.BindBody(&req); err != nil {
				return respondJSON(e, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			}

			req.Label = strings.TrimSpace(req.Label)
			if req.Label == "" {
				return respondJSON(e, http.StatusBadRequest, map[string]string{"error": "label is required"})
			}
			if len(req.Scopes) == 0 {
				return respondJSON(e, http.StatusBadRequest, map[string]string{"error": "at least one scope is required"})
			}
			for _, s := range req.Scopes {
				if !validScopes[s] {
					return respondJSON(e, http.StatusBadRequest, map[string]string{"error": "invalid scope: " + s})
				}
			}

			fullKey, err := generateAPIKey()
			if err != nil {
				app.Logger().Error("Failed to generate API key", "error", err)
				return respondJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to generate API key"})
			}

			collection, err := app.FindCollectionByNameOrId("api_keys")
			if err != nil {
				return respondJSON(e, http.StatusInternalServerError, map[string]string{"error": "api_keys collection not found"})
			}

			record := core.NewRecord(collection)
			record.Set("label", req.Label)
			record.Set("key_prefix", extractKeyPrefix(fullKey))
			record.Set("key_hash", hashAPIKey(fullKey))
			record.Set("scopes", req.Scopes)
			record.Set("is_active", true)
			if len(req.AllowedOrigins) > 0 {
				record.Set("allowed_origins", req.AllowedOrigins)
			}
			if req.ExpiresAt != "" {
				record.Set("expires_at", req.ExpiresAt)
			}

			if err := app.Save(record); err != nil {
				app.Logger().Error("Failed to save API key", "error", err)
				return respondJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to save API key"})
			}

			// Return the full key ONCE — caller must save it.
			return respondJSON(e, http.StatusCreated, map[string]any{
				"id":         record.Id,
				"label":      req.Label,
				"key":        fullKey, // shown only here
				"key_prefix": record.GetString("key_prefix"),
				"scopes":     req.Scopes,
				"is_active":  true,
				"created":    record.GetString("created"),
				"message":    "Store this key now — it will not be shown again.",
			})
		})

		// GET /api/admin/api-keys
		// Lists all keys (raw key is never returned).
		se.Router.GET("/api/admin/api-keys", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			records, err := app.FindRecordsByFilter("api_keys", "id != ''", "-created", 200, 0)
			if err != nil {
				return respondJSON(e, http.StatusOK, map[string]any{"data": []any{}})
			}

			keys := make([]map[string]any, 0, len(records))
			for _, r := range records {
				keys = append(keys, map[string]any{
					"id":              r.Id,
					"label":           r.GetString("label"),
					"key_prefix":      r.GetString("key_prefix"),
					"scopes":          decodeJSONStringSlice(r.Get("scopes")),
					"allowed_origins": decodeJSONStringSlice(r.Get("allowed_origins")),
					"is_active":       r.GetBool("is_active"),
					"last_used_at":    r.GetString("last_used_at"),
					"expires_at":      r.GetString("expires_at"),
					"created":         r.GetString("created"),
				})
			}
			return respondJSON(e, http.StatusOK, map[string]any{"data": keys})
		})

		// DELETE /api/admin/api-keys/{id}
		// Soft-revokes a key (sets is_active=false) so the audit trail survives.
		se.Router.DELETE("/api/admin/api-keys/{id}", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}
			id := e.Request.PathValue("id")
			record, err := app.FindRecordById("api_keys", id)
			if err != nil {
				return respondNotFound(e, "API key not found")
			}
			record.Set("is_active", false)
			if err := app.Save(record); err != nil {
				return respondJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to revoke API key"})
			}
			return respondJSON(e, http.StatusOK, map[string]string{"status": "revoked"})
		})

		// ---------------------------------------------------------------
		// Public v1 read-only endpoints (require API key with matching scope)
		// ---------------------------------------------------------------

		readAuth := func(scope string, handler func(*core.RequestEvent) error) func(*core.RequestEvent) error {
			return APIKeyMiddleware(app, scope)(handler)
		}

		// GET /api/v1/profile
		se.Router.GET("/api/v1/profile", readAuth("read:profile", func(e *core.RequestEvent) error {
			records, err := app.FindRecordsByFilter("profile", "id != ''", "", 1, 0)
			if err != nil || len(records) == 0 {
				return respondNotFound(e, "profile not found")
			}
			p := records[0]
			return respondJSON(e, http.StatusOK, map[string]any{
				"name":     p.GetString("name"),
				"headline": p.GetString("headline"),
				"summary":  p.GetString("summary"),
				"location": p.GetString("location"),
				"updated":  p.GetString("updated"),
			})
		}))

		// GET /api/v1/posts
		se.Router.GET("/api/v1/posts", readAuth("read:posts", func(e *core.RequestEvent) error {
			return listPublicCollection(app, e, "posts", "-created", mapBasicPost)
		}))

		// GET /api/v1/projects
		se.Router.GET("/api/v1/projects", readAuth("read:projects", func(e *core.RequestEvent) error {
			return listPublicCollection(app, e, "projects", "-created", mapBasicProject)
		}))

		// GET /api/v1/skills
		se.Router.GET("/api/v1/skills", readAuth("read:skills", func(e *core.RequestEvent) error {
			return listPublicCollection(app, e, "skills", "sort_order", mapBasicSkill)
		}))

		// GET /api/v1/experience
		se.Router.GET("/api/v1/experience", readAuth("read:experience", func(e *core.RequestEvent) error {
			return listPublicCollection(app, e, "experience", "-start_date", mapBasicExperience)
		}))

		return se.Next()
	})
}

// listPublicCollection serves a simple list endpoint. It applies visibility +
// is_draft filters if those fields exist on the collection (most public
// collections in Facet have both).
func listPublicCollection(
	app core.App,
	e *core.RequestEvent,
	collection string,
	sort string,
	mapper func(*core.Record) map[string]any,
) error {
	coll, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		return respondNotFound(e, "collection not found")
	}

	filter := "id != ''"
	if coll.Fields.GetByName("visibility") != nil {
		filter = "visibility = 'public'"
	}
	if coll.Fields.GetByName("is_draft") != nil {
		filter += " && is_draft = false"
	}

	records, err := app.FindRecordsByFilter(collection, filter, sort, 200, 0)
	if err != nil {
		records = []*core.Record{}
	}
	out := make([]map[string]any, 0, len(records))
	for _, r := range records {
		out = append(out, mapper(r))
	}
	return respondJSON(e, http.StatusOK, map[string]any{"data": out})
}

// ----------------------------------------------------------------------------
// Minimal mappers for v1 read endpoints. Cloud has richer mappers; the
// self-hosted v1 surface intentionally returns a clean, stable subset.
// ----------------------------------------------------------------------------

func mapBasicPost(r *core.Record) map[string]any {
	return map[string]any{
		"id":           r.Id,
		"title":        r.GetString("title"),
		"slug":         r.GetString("slug"),
		"excerpt":      r.GetString("excerpt"),
		"published_at": r.GetString("published_at"),
		"created":      r.GetString("created"),
		"updated":      r.GetString("updated"),
	}
}

func mapBasicProject(r *core.Record) map[string]any {
	return map[string]any{
		"id":          r.Id,
		"title":       r.GetString("title"),
		"slug":        r.GetString("slug"),
		"description": r.GetString("description"),
		"url":         r.GetString("url"),
		"status":      r.GetString("status"),
		"created":     r.GetString("created"),
		"updated":     r.GetString("updated"),
	}
}

func mapBasicSkill(r *core.Record) map[string]any {
	return map[string]any{
		"id":         r.Id,
		"name":       r.GetString("name"),
		"category":   r.GetString("category"),
		"level":      r.GetString("level"),
		"sort_order": r.GetInt("sort_order"),
	}
}

func mapBasicExperience(r *core.Record) map[string]any {
	return map[string]any{
		"id":          r.Id,
		"company":     r.GetString("company"),
		"title":       r.GetString("title"),
		"location":    r.GetString("location"),
		"start_date":  r.GetString("start_date"),
		"end_date":    r.GetString("end_date"),
		"is_current":  r.GetBool("is_current"),
		"description": r.GetString("description"),
	}
}
