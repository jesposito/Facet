package hooks

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"facet/services"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
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

// Valid scopes for the self-hosted public API.
//
// IMPORTANT: keep these in sync with the i18n keys under `admin.api_keys.scope_*`
// and with the matching frontend list in /admin/api/+page.svelte.
//
// Write scopes are independent of read scopes — granting `write:posts` does NOT
// imply `read:posts`. Callers that need both must request both.
var validScopes = map[string]bool{
	"read:profile":     true,
	"read:posts":       true,
	"read:projects":    true,
	"read:skills":      true,
	"read:experience":  true,
	"write:profile":    true,
	"write:posts":      true,
	"write:projects":   true,
	"write:skills":     true,
	"write:experience": true,
}

// generateAPIKey returns a freshly-minted key like "facet_<64 hex chars>".
// The full key is returned ONCE to the caller; only the HMAC hash is stored.
func generateAPIKey() (string, error) {
	buf := make([]byte, apiKeyRandBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return APIKeyPrefix + hex.EncodeToString(buf), nil
}

// hashAPIKey returns the keyed HMAC of the full key. This is what we store in
// the `key_hash` column.
func hashAPIKey(crypto *services.CryptoService, fullKey string) string {
	return crypto.HMACToken(fullKey)
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
	now := time.Now()

	// Check and claim the slot under one lock so two concurrent requests
	// can't both decide to write.
	lastUsedMu.Lock()
	if last, ok := lastUsedCache[keyID]; ok && now.Sub(last) < lastUsedDebounceWindow {
		lastUsedMu.Unlock()
		return
	}
	lastUsedCache[keyID] = now
	lastUsedMu.Unlock()

	if err := touchAPIKeyLastUsed(app, keyID, now); err != nil {
		app.Logger().Warn("Failed to record API key use", "key_id", keyID, "error", err)
		// Release the slot so the next request tries again, unless a newer
		// request has already claimed it.
		lastUsedMu.Lock()
		if lastUsedCache[keyID].Equal(now) {
			delete(lastUsedCache, keyID)
		}
		lastUsedMu.Unlock()
	}
}

// touchAPIKeyLastUsed sets last_used_at and nothing else. It must not load
// the record and Save it: PocketBase's Save writes every column, so a copy
// read before an admin revoked the key, cut its scopes or changed its expiry
// would quietly put the old values back.
func touchAPIKeyLastUsed(app core.App, keyID string, at time.Time) error {
	_, err := app.NonconcurrentDB().
		Update("api_keys", dbx.Params{"last_used_at": at.UTC().Format(types.DefaultDateLayout)}, dbx.HashExp{"id": keyID}).
		Execute()
	return err
}

// maxAllowedOrigins caps how many origins one key may list.
const maxAllowedOrigins = 20

// canonicalOrigin turns "https://Example.com:443" into "https://example.com"
// so allowlist entries and Origin headers compare exactly. It accepts only
// an http(s) scheme, a host and an optional port: no wildcards, paths,
// queries, fragments or credentials. A single trailing "/" is tolerated
// because that's what a copied address bar URL looks like.
func canonicalOrigin(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("origin is empty")
	}
	if strings.Contains(raw, "*") {
		return "", errors.New("wildcards are not supported")
	}
	u, err := url.Parse(strings.TrimSuffix(raw, "/"))
	if err != nil {
		return "", errors.New("origin is not a valid URL")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", errors.New("origin must start with http:// or https://")
	}
	if u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" ||
		strings.HasSuffix(raw, "?") || strings.HasSuffix(raw, "#") {
		return "", errors.New("origin must be only scheme, host and optional port")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return "", errors.New("origin has no host")
	}
	port := u.Port()
	if port == "" && strings.HasSuffix(u.Host, ":") {
		return "", errors.New("origin has an empty port")
	}
	if port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return "", errors.New("origin port is invalid")
		}
		if (scheme == "http" && n == 80) || (scheme == "https" && n == 443) {
			port = ""
		} else {
			port = strconv.Itoa(n)
		}
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]" // IPv6 literal
	}
	if port != "" {
		return scheme + "://" + host + ":" + port, nil
	}
	return scheme + "://" + host, nil
}

// normalizeAllowedOrigins validates and canonicalizes the admin's list,
// dropping duplicates. Any bad entry rejects the whole list.
func normalizeAllowedOrigins(origins []string) ([]string, error) {
	if len(origins) > maxAllowedOrigins {
		return nil, errors.New("too many allowed origins (max " + strconv.Itoa(maxAllowedOrigins) + ")")
	}
	out := make([]string, 0, len(origins))
	seen := make(map[string]bool, len(origins))
	for _, raw := range origins {
		origin, err := canonicalOrigin(raw)
		if err != nil {
			return nil, errors.New("invalid allowed origin " + strconv.Quote(raw) + ": " + err.Error())
		}
		if !seen[origin] {
			seen[origin] = true
			out = append(out, origin)
		}
	}
	return out, nil
}

// originAllowed applies a key's allowed_origins to a request.
//
// This is a guard against a key being used from someone else's web page. It
// is not authentication: any non-browser client can send whatever Origin
// header it likes. PocketBase handles CORS separately; this check does not
// configure its response headers. So:
//   - an empty allowlist allows every request;
//   - a request with no Origin header (curl, servers, scripts) is allowed;
//   - otherwise the Origin must exactly match an entry. "null" and malformed
//     Origins never match, and neither do stored entries that fail to parse,
//     so a broken allowlist blocks browsers instead of opening up.
func originAllowed(allowed []string, r *http.Request) bool {
	if len(allowed) == 0 {
		return true
	}
	values := r.Header.Values("Origin")
	if len(values) == 0 {
		return true
	}
	if len(values) > 1 {
		return false
	}
	origin, err := canonicalOrigin(values[0])
	if err != nil {
		return false
	}
	for _, entry := range allowed {
		if want, err := canonicalOrigin(entry); err == nil && want == origin {
			return true
		}
	}
	return false
}

// storedAllowedOrigins reads a key's allowlist. A value that isn't a JSON
// list of strings (for example a hand-edited "*") comes back as a list that
// matches nothing, rather than as empty, which would allow everything.
func storedAllowedOrigins(key *core.Record) []string {
	raw, _ := json.Marshal(key.Get("allowed_origins"))
	switch strings.TrimSpace(string(raw)) {
	case "", "null", `""`, "[]":
		return nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return []string{"invalid"}
	}
	return out
}

// parseAPIKeyExpiry reads the optional expires_at from the create form. A
// bare date ("2026-12-31") means the key works through the end of that day
// in UTC, which is what "expires on" reads as in the admin UI.
func parseAPIKeyExpiry(raw string, now time.Time) (types.DateTime, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return types.DateTime{}, nil
	}
	var t time.Time
	if d, err := time.Parse("2006-01-02", raw); err == nil {
		t = d.Add(24*time.Hour - time.Millisecond)
	} else {
		dt, err := types.ParseDateTime(raw)
		if err != nil || dt.IsZero() {
			return types.DateTime{}, errors.New("expires_at is not a valid date")
		}
		t = dt.Time()
	}
	if !t.After(now) {
		return types.DateTime{}, errors.New("expires_at must be in the future")
	}
	return types.ParseDateTime(t.UTC())
}

// APIKeyMiddleware authenticates a request by Authorization: Bearer <key>, looks
// up the matching api_keys row by hash, and enforces is_active + expires_at +
// allowed_origins + scope checks. It also sets the request context with the
// key id + granted scopes.
//
// Self-hosted is single-tenant, so there's no per-tenant rate limit or member
// tier — just the scope check.
func APIKeyMiddleware(app core.App, crypto *services.CryptoService, requiredScope string) func(func(*core.RequestEvent) error) func(*core.RequestEvent) error {
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
			hash := hashAPIKey(crypto, fullKey)
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
			if !originAllowed(storedAllowedOrigins(key), e.Request) {
				return respondForbidden(e, "origin not allowed for this API key")
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
func RegisterAPIKeyHooks(app *pocketbase.PocketBase, crypto *services.CryptoService, rl *services.RateLimitService) {
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
			origins, err := normalizeAllowedOrigins(req.AllowedOrigins)
			if err != nil {
				return respondJSON(e, http.StatusBadRequest, map[string]string{"error": err.Error()})
			}
			expiresAt, err := parseAPIKeyExpiry(req.ExpiresAt, time.Now())
			if err != nil {
				return respondJSON(e, http.StatusBadRequest, map[string]string{"error": err.Error()})
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
			record.Set("key_hash", hashAPIKey(crypto, fullKey))
			record.Set("scopes", req.Scopes)
			record.Set("is_active", true)
			if len(origins) > 0 {
				record.Set("allowed_origins", origins)
			}
			if !expiresAt.IsZero() {
				record.Set("expires_at", expiresAt)
			}

			if err := app.Save(record); err != nil {
				app.Logger().Error("Failed to save API key", "error", err)
				return respondJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to save API key"})
			}

			// Return the full key ONCE — caller must save it.
			return respondJSON(e, http.StatusCreated, map[string]any{
				"id":              record.Id,
				"label":           req.Label,
				"key":             fullKey, // shown only here
				"key_prefix":      record.GetString("key_prefix"),
				"scopes":          req.Scopes,
				"allowed_origins": origins,
				"expires_at":      record.GetString("expires_at"),
				"is_active":       true,
				"created":         record.GetString("created"),
				"message":         "Store this key now — it will not be shown again.",
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

		// docs/SECURITY.md promises per-IP rate limiting on the public API.
		readAuth := func(scope string, handler func(*core.RequestEvent) error) func(*core.RequestEvent) error {
			return RateLimitMiddleware(rl, "normal")(APIKeyMiddleware(app, crypto, scope)(handler))
		}

		// GET /api/v1/profile (only when the profile isn't private, as on
		// the public site)
		se.Router.GET("/api/v1/profile", readAuth("read:profile", func(e *core.RequestEvent) error {
			records, err := app.FindRecordsByFilter("profile", "visibility != 'private'", "", 1, 0)
			if err != nil || len(records) == 0 {
				return respondNotFound(e, "profile not found")
			}
			return respondJSON(e, http.StatusOK, mapPublicProfile(records[0]))
		}))

		// GET /api/v1/posts
		se.Router.GET("/api/v1/posts", readAuth("read:posts", func(e *core.RequestEvent) error {
			return listPublicCollection(app, e, "posts", "-created", mapPublicPost)
		}))

		// GET /api/v1/projects
		se.Router.GET("/api/v1/projects", readAuth("read:projects", func(e *core.RequestEvent) error {
			return listPublicCollection(app, e, "projects", "-created", mapPublicProject)
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
// Mappers for the v1 endpoints. Each mapBasic* shape covers every field the
// write API accepts for that collection (see apiV1WriteFields), so a write
// response, or a later read, shows what was written. Field names follow the
// collection schema. A few older names stay as read-only aliases:
//   - skills.level        = proficiency
//   - projects.url        = the first entry in links that has a url
//   - experience.is_current = true when end_date is empty, which is how the
//     site decides to show "Present"
//
// Older versions also returned projects.status. Projects have no status
// field, so it was always "" and has been dropped.
// ----------------------------------------------------------------------------

// jsonFieldValue decodes a JSON field so it serializes as data rather than
// raw bytes. Unset fields become fallback (an empty list for list fields).
func jsonFieldValue(r *core.Record, field string, fallback any) any {
	var v any
	if err := r.UnmarshalJSONField(field, &v); err != nil || v == nil {
		return fallback
	}
	return v
}

// withholdPaidFields clears the fields a buyer pays for when the record is
// paid, as the public site does. The API key isn't a purchase.
func withholdPaidFields(r *core.Record, out map[string]any) map[string]any {
	if !requiresPurchase(r) {
		return out
	}
	for _, field := range paidContentFields[r.Collection().Name] {
		if _, ok := out[field].(string); ok {
			out[field] = ""
		} else if _, ok := out[field]; ok {
			out[field] = nil
		}
	}
	return out
}

func mapBasicPost(r *core.Record) map[string]any {
	return map[string]any{
		"id":           r.Id,
		"title":        r.GetString("title"),
		"slug":         r.GetString("slug"),
		"excerpt":      r.GetString("excerpt"),
		"content":      r.GetString("content"),
		"tags":         jsonFieldValue(r, "tags", []any{}),
		"visibility":   r.GetString("visibility"),
		"is_draft":     r.GetBool("is_draft"),
		"access_tier":  r.GetString("access_tier"),
		"published_at": r.GetString("published_at"),
		"created":      r.GetString("created"),
		"updated":      r.GetString("updated"),
	}
}

// mapPublicPost is the read API's post shape. Paid content stays off the
// public API, as it does on the site.
func mapPublicPost(r *core.Record) map[string]any {
	return withholdPaidFields(r, mapBasicPost(r))
}

// mapPublicProject is the read API's project shape. What buyers pay for
// stays off the public API, as it does on the site.
func mapPublicProject(r *core.Record) map[string]any {
	return withholdPaidFields(r, mapBasicProject(r))
}

func mapBasicProject(r *core.Record) map[string]any {
	links := jsonFieldValue(r, "links", []any{})
	return map[string]any{
		"id":          r.Id,
		"title":       r.GetString("title"),
		"slug":        r.GetString("slug"),
		"summary":     r.GetString("summary"),
		"description": r.GetString("description"),
		"experience":  jsonFieldValue(r, "experience", nil),
		"tech_stack":  jsonFieldValue(r, "tech_stack", []any{}),
		"links":       links,
		"url":         firstLinkURL(links),
		"categories":  jsonFieldValue(r, "categories", []any{}),
		"visibility":  r.GetString("visibility"),
		"is_draft":    r.GetBool("is_draft"),
		"is_featured": r.GetBool("is_featured"),
		"sort_order":  r.GetFloat("sort_order"),
		"access_tier": r.GetString("access_tier"),
		"created":     r.GetString("created"),
		"updated":     r.GetString("updated"),
	}
}

// firstLinkURL returns the url of the first project link that has one.
// Links are stored as [{"type": "github", "url": "https://..."}].
func firstLinkURL(links any) string {
	list, _ := links.([]any)
	for _, item := range list {
		if link, ok := item.(map[string]any); ok {
			if u, ok := link["url"].(string); ok && strings.TrimSpace(u) != "" {
				return u
			}
		}
	}
	return ""
}

func mapBasicSkill(r *core.Record) map[string]any {
	return map[string]any{
		"id":          r.Id,
		"name":        r.GetString("name"),
		"category":    r.GetString("category"),
		"proficiency": r.GetString("proficiency"),
		"level":       r.GetString("proficiency"),
		"visibility":  r.GetString("visibility"),
		"sort_order":  r.GetFloat("sort_order"),
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
		"is_current":  r.GetDateTime("end_date").IsZero(),
		"description": r.GetString("description"),
		"bullets":     jsonFieldValue(r, "bullets", []any{}),
		"skills":      jsonFieldValue(r, "skills", []any{}),
		"visibility":  r.GetString("visibility"),
		"is_draft":    r.GetBool("is_draft"),
		"sort_order":  r.GetFloat("sort_order"),
	}
}

// mapPublicProfile is GET /api/v1/profile.
func mapPublicProfile(r *core.Record) map[string]any {
	return map[string]any{
		"name":     r.GetString("name"),
		"headline": r.GetString("headline"),
		"summary":  r.GetString("summary"),
		"location": r.GetString("location"),
		"updated":  r.GetString("updated"),
	}
}

// mapProfileWrite answers PATCH /api/v1/profile: the public shape plus the
// other fields that endpoint accepts, so the caller can see what it set.
func mapProfileWrite(r *core.Record) map[string]any {
	out := mapPublicProfile(r)
	out["contact_email"] = r.GetString("contact_email")
	out["contact_links"] = jsonFieldValue(r, "contact_links", []any{})
	out["visibility"] = r.GetString("visibility")
	return out
}
