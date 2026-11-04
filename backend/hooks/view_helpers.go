package hooks

import (
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"facet/services"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func getCollectionName(section string) string {
	// Handle custom content sections (format: custom:itemId)
	if strings.HasPrefix(section, "custom:") {
		return "custom_content"
	}

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

// isCustomContentSection checks if a section name refers to a custom content item
func isCustomContentSection(section string) bool {
	return strings.HasPrefix(section, "custom:")
}

// getCustomContentId extracts the custom content ID from a section name
func getCustomContentId(section string) string {
	if !isCustomContentSection(section) {
		return ""
	}
	return strings.TrimPrefix(section, "custom:")
}

func getDefaultLayout(section string) string {
	// Handle custom content sections
	if isCustomContentSection(section) {
		return "default"
	}

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

// Contact view restrictions are evaluated before their metadata is removed from
// public responses. An empty map means all pages; a nonempty map is opt-in.
func contactVisibleInView(record *core.Record, viewID string) bool {
	raw := record.GetString("view_visibility")
	if raw == "" {
		return true
	}
	var visibility map[string]bool
	if err := json.Unmarshal([]byte(raw), &visibility); err != nil {
		return false
	}
	return len(visibility) == 0 || visibility[viewID]
}

// A per-view override can show a private item on one view. It never publishes
// a draft.
func isRecordVisibleForSection(record *core.Record, section string, viewId string) bool {
	if section == "contacts" && !contactVisibleInView(record, viewId) {
		return false
	}
	if section == "testimonials" && record.GetString("status") != "approved" {
		return false
	}
	if record.GetBool("is_draft") {
		return false
	}
	if isRecordVisible(record) {
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

// serializeRecords projects records for public responses. Paid fields are
// removed unless access allows them; pass nil where no buyer can be identified.
func serializeRecords(records []*core.Record, access contentAccess) []map[string]interface{} {
	var result []map[string]interface{}
	for _, record := range records {
		item := make(map[string]interface{})
		for key, value := range record.FieldsData() {
			if !publicFieldAllowed(record.Collection(), key) {
				continue
			}
			item[key] = value
		}
		item["id"] = record.Id

		// Handle testimonials author_photo: convert filename to full URL
		if record.Collection().Name == "testimonials" {
			if photo := record.GetString("author_photo"); publicFieldAllowed(record.Collection(), "author_photo") && photo != "" {
				item["author_photo"] = "/api/files/testimonials/" + record.Id + "/" + photo
			}
		}

		gatePaidContent(record, item, access)
		redactUnavailableProtection(record, item)
		result = append(result, item)
	}
	return result
}

func serializeRecordsWithOverrides(app *pocketbase.PocketBase, records []*core.Record, itemConfig map[string]map[string]interface{}, sectionName string, access contentAccess) []map[string]interface{} {
	var result []map[string]interface{}
	overridableFields := getOverridableFields(sectionName)

	for _, record := range records {
		item := make(map[string]interface{})
		for key, value := range record.FieldsData() {
			if !publicFieldAllowed(record.Collection(), key) {
				continue
			}
			item[key] = value
		}
		item["id"] = record.Id

		if config, exists := itemConfig[record.Id]; exists {
			if overrides, ok := config["overrides"].(map[string]interface{}); ok {
				for field, value := range overrides {
					if containsString(overridableFields, field) && publicFieldAllowed(record.Collection(), field) {
						item[field] = value
					}
				}
			}
		}

		if sectionName == "projects" || sectionName == "posts" || isCustomContentSection(sectionName) {
			// Prefer library URL over file upload
			if libraryURL := record.GetString("cover_image_library_url"); publicFieldAllowed(record.Collection(), "cover_image_library_url") && libraryURL != "" {
				item["cover_image_url"] = libraryURL
				item["cover_image_large_url"] = libraryURL
				item["cover_image_thumb_url"] = libraryURL
			} else if coverImage := record.GetString("cover_image"); publicFieldAllowed(record.Collection(), "cover_image") && coverImage != "" {
				collectionID := record.Collection().Id
				recordID := record.Id
				item["cover_image_url"] = "/api/files/" + collectionID + "/" + recordID + "/" + coverImage
				item["cover_image_large_url"] = "/api/files/" + collectionID + "/" + recordID + "/" + coverImage + "?thumb=1600x0"
				item["cover_image_thumb_url"] = "/api/files/" + collectionID + "/" + recordID + "/" + coverImage + "?thumb=480x0"
			}
		}

		// Handle custom content media gallery
		if isCustomContentSection(sectionName) {
			if mediaField := record.Get("media"); publicFieldAllowed(record.Collection(), "media") && mediaField != nil {
				var mediaFiles []string
				switch v := mediaField.(type) {
				case []string:
					mediaFiles = v
				case []interface{}:
					for _, file := range v {
						if str, ok := file.(string); ok {
							mediaFiles = append(mediaFiles, str)
						}
					}
				}
				if len(mediaFiles) > 0 {
					collectionID := record.Collection().Id
					recordID := record.Id
					var mediaURLs []string
					for _, file := range mediaFiles {
						mediaURLs = append(mediaURLs, "/api/files/"+collectionID+"/"+recordID+"/"+file)
					}
					item["media_urls"] = mediaURLs
				}
			}

			// Handle media_refs from library (external_media references)
			if mediaRefs := record.GetStringSlice("media_refs"); publicFieldAllowed(record.Collection(), "media_refs") && len(mediaRefs) > 0 {
				item["media_refs"] = mediaRefs
				if expanded, err := fetchExternalMedia(app, mediaRefs); err == nil {
					item["media_refs_expand"] = expanded
				}
			}
		}

		if sectionName == "experience" {
			// Prefer library URL over file upload
			if libraryURL := record.GetString("company_logo_library_url"); publicFieldAllowed(record.Collection(), "company_logo_library_url") && libraryURL != "" {
				item["company_logo_url"] = libraryURL
			} else if companyLogo := record.GetString("company_logo"); publicFieldAllowed(record.Collection(), "company_logo") && companyLogo != "" {
				collectionID := record.Collection().Id
				recordID := record.Id
				item["company_logo_url"] = "/api/files/" + collectionID + "/" + recordID + "/" + companyLogo
			}
		}

		if sectionName == "education" {
			// Prefer library URL over file upload
			if libraryURL := record.GetString("institution_logo_library_url"); publicFieldAllowed(record.Collection(), "institution_logo_library_url") && libraryURL != "" {
				item["institution_logo_url"] = libraryURL
			} else if institutionLogo := record.GetString("institution_logo"); publicFieldAllowed(record.Collection(), "institution_logo") && institutionLogo != "" {
				collectionID := record.Collection().Id
				recordID := record.Id
				item["institution_logo_url"] = "/api/files/" + collectionID + "/" + recordID + "/" + institutionLogo
			}
		}

		if sectionName == "testimonials" {
			if photo := record.GetString("author_photo"); publicFieldAllowed(record.Collection(), "author_photo") && photo != "" {
				item["author_photo"] = "/api/files/testimonials/" + record.Id + "/" + photo
			}
		}

		gatePaidContent(record, item, access)
		redactUnavailableProtection(record, item)
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

// The frontend keeps a visitor's view credentials in HttpOnly cookies. Its
// server forwards them as headers, but the browser's own API calls can only
// send the cookies. They identify a visitor to one view and never sign in the
// owner.
const (
	passwordTokenCookie = "me_password_token"
	shareTokenCookie    = "me_share_token"
	shareVisitCookie    = "me_share_visit"
	visitorCookieMaxLen = 2048
)

// crossOriginBrowserRequest reports whether a browser says a request came from
// another origin. SameSite=Lax alone isn't enough, because a sibling subdomain
// counts as the same site. Non-browser clients send neither header.
func crossOriginBrowserRequest(e *core.RequestEvent) bool {
	if site := e.Request.Header.Get("Sec-Fetch-Site"); site != "" {
		return site != "same-origin" && site != "none"
	}
	origin := e.Request.Header.Get("Origin")
	if origin == "" {
		return false
	}
	parsed, err := url.Parse(origin)
	return err != nil || parsed.Host != e.Request.Host
}

// visitorCookie reads a visitor credential, ignoring it on a request from
// another origin so a foreign page can't act with the visitor's access.
func visitorCookie(e *core.RequestEvent, name string) string {
	if crossOriginBrowserRequest(e) {
		return ""
	}
	cookie, err := e.Request.Cookie(name)
	if err != nil || len(cookie.Value) > visitorCookieMaxLen {
		return ""
	}
	return cookie.Value
}

func extractPasswordToken(e *core.RequestEvent) string {
	authHeader := e.Request.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	if token := e.Request.Header.Get("X-Password-Token"); token != "" {
		return token
	}
	return visitorCookie(e, passwordTokenCookie)
}

// explicitShareToken is a share token the caller presented on purpose.
func explicitShareToken(e *core.RequestEvent) string {
	if shareToken := e.Request.Header.Get("X-Share-Token"); shareToken != "" {
		return shareToken
	}
	return e.Request.URL.Query().Get("token")
}

func extractShareToken(e *core.RequestEvent) string {
	if shareToken := explicitShareToken(e); shareToken != "" {
		return shareToken
	}
	return visitorCookie(e, shareTokenCookie)
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

// consumeShareTokenUse counts one use of a share token, unless the token has
// none left. Checking use_count and saving use_count+1 as separate steps let
// a burst of requests all pass the check, so a single-use link opened in
// parallel worked many times. A conditional UPDATE does both at once.
func consumeShareTokenUse(app core.App, tokenID string) bool {
	result, err := app.DB().NewQuery(
		"UPDATE share_tokens SET use_count = use_count + 1, last_used_at = {:now} " +
			"WHERE id = {:id} AND is_active = TRUE AND (max_uses <= 0 OR use_count < max_uses)").
		Bind(dbx.Params{"id": tokenID, "now": types.NowDateTime().String()}).
		Execute()
	if err != nil {
		app.Logger().Warn("Failed to record share token use", "error", err, "token_id", tokenID)
		return false
	}
	n, err := result.RowsAffected()
	return err == nil && n == 1
}

// A share visit proof lets the visitor who spent a use of a share token keep
// browsing that view (its projects, a return to the view) after the token runs
// out of uses. It is issued only after a use is counted, and only for
// shareVisitProofTTL or until the token expires, whichever is sooner.
const (
	shareVisitProofTTL          = 30 * time.Minute
	shareVisitHeader            = "X-Share-Visit"
	shareVisitExpiresHeader     = "X-Share-Visit-Expires"
	shareVisitProofHeaderMaxLen = 2048
)

func extractShareVisitProof(e *core.RequestEvent) string {
	proof := e.Request.Header.Get(shareVisitHeader)
	if proof == "" {
		return visitorCookie(e, shareVisitCookie)
	}
	if len(proof) > shareVisitProofHeaderMaxLen {
		return ""
	}
	return proof
}

// issueShareVisitProof adds a visit proof to the response for a token use the
// caller has just counted. Failure only costs the visitor later navigation.
func issueShareVisitProof(e *core.RequestEvent, crypto *services.CryptoService, tokenRecord *core.Record) {
	expiresAt := time.Now().Add(shareVisitProofTTL)
	if tokenExpiry := tokenRecord.GetDateTime("expires_at"); !tokenExpiry.IsZero() && tokenExpiry.Time().Before(expiresAt) {
		expiresAt = tokenExpiry.Time()
	}
	if !expiresAt.After(time.Now()) {
		return
	}
	proof, err := crypto.GenerateShareVisitProof(tokenRecord.GetString("view_id"), tokenRecord.Id, tokenRecord.GetString("token_hash"), expiresAt)
	if err != nil {
		return
	}
	e.Response.Header().Set(shareVisitHeader, proof)
	e.Response.Header().Set(shareVisitExpiresHeader, expiresAt.UTC().Format(time.RFC3339))
}

// validShareVisit reports whether proof admits a visit to view made with the
// raw share token. The token record is read fresh on every call, so revoking,
// deleting or expiring the token ends the visit; only the use limit is not
// rechecked, since the visit already spent its use.
func validShareVisit(app *pocketbase.PocketBase, crypto *services.CryptoService, share *services.ShareService, proof, token string, view *core.Record) bool {
	if proof == "" || token == "" {
		return false
	}
	claims, err := crypto.ValidateShareVisitProof(proof)
	if err != nil || claims.ViewID != view.Id {
		return false
	}
	tokenRecord, err := app.FindRecordById("share_tokens", claims.TokenID)
	if err != nil || tokenRecord == nil || !tokenRecord.GetBool("is_active") {
		return false
	}
	if tokenRecord.GetString("view_id") != view.Id {
		return false
	}
	tokenHash := tokenRecord.GetString("token_hash")
	if !share.ValidateTokenHMAC(token, tokenHash) {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(crypto.ShareVisitTokenFingerprint(tokenHash)), []byte(claims.TokenHash)) != 1 {
		return false
	}
	expiresAt := tokenRecord.GetDateTime("expires_at")
	return expiresAt.IsZero() || time.Now().Before(expiresAt.Time())
}

// viewAccessDenial explains why a request may not read a view.
type viewAccessDenial struct {
	status  int
	message string
}

// checkViewAccess applies a view's visibility to the current request. It
// returns the share token record that granted access to an unlisted view, so
// the caller can decide whether that request counts as a use of the token.
// A request carrying a valid share visit proof is part of a visit that was
// already counted; it is admitted with no record, so nothing is spent again.
// The site owner can read every view.
func checkViewAccess(app *pocketbase.PocketBase, crypto *services.CryptoService, share *services.ShareService, e *core.RequestEvent, view *core.Record) (*core.Record, *viewAccessDenial) {
	if isSuperuser(e) {
		return nil, nil
	}

	switch view.GetString("visibility") {
	case "private":
		// 404 rather than 401 so private views can't be discovered.
		return nil, &viewAccessDenial{http.StatusNotFound, "view not found"}

	case "password":
		token := extractPasswordToken(e)
		if token == "" {
			return nil, &viewAccessDenial{http.StatusUnauthorized, "password token required"}
		}
		viewID, err := crypto.ValidateViewAccessJWT(token)
		if err != nil {
			return nil, &viewAccessDenial{http.StatusUnauthorized, "invalid or expired token"}
		}
		if viewID != view.Id {
			return nil, &viewAccessDenial{http.StatusUnauthorized, "token not valid for this view"}
		}

	case "unlisted":
		shareToken := extractShareToken(e)
		if shareToken == "" {
			return nil, &viewAccessDenial{http.StatusUnauthorized, "share token required"}
		}
		if validShareVisit(app, crypto, share, extractShareVisitProof(e), shareToken, view) {
			return nil, nil
		}
		valid, tokenRecord := validateShareToken(app, share, shareToken, view.Id)
		if !valid {
			return nil, &viewAccessDenial{http.StatusUnauthorized, "invalid or expired share token"}
		}
		return tokenRecord, nil
	}

	return nil, nil
}

// publicViewDenial is what /access and /data answer a denied request with. An
// unlisted view doesn't exist for anyone without share access, so like a
// private view it reads as missing there.
func publicViewDenial(view *core.Record, denial *viewAccessDenial) *viewAccessDenial {
	if denial != nil && view.GetString("visibility") == "unlisted" {
		return &viewAccessDenial{http.StatusNotFound, "view not found"}
	}
	return denial
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

// filterBySelectedItems filters serialized records based on selected item IDs.
// If selectedItems is empty, returns empty (nothing selected = nothing shown).
// If selectedItems has specific IDs, returns only those items in the specified order.
func filterBySelectedItems(items []map[string]interface{}, selectedItems []string) []map[string]interface{} {
	if len(selectedItems) == 0 {
		return nil // Nothing selected = nothing shown
	}

	// Build a map for quick lookup
	itemMap := make(map[string]map[string]interface{})
	for _, item := range items {
		if id, ok := item["id"].(string); ok {
			itemMap[id] = item
		}
	}

	// Return items in the order specified by selectedItems
	var result []map[string]interface{}
	for _, id := range selectedItems {
		if item, exists := itemMap[id]; exists {
			result = append(result, item)
		}
	}
	return result
}

// filterBySelectedItemsKeepOrder filters items by selectedItems set membership
// but PRESERVES the input items order. Use this for sections whose source has
// its own ordering authority (e.g. skills via per-skill sort_order set in the
// admin/skills drag-and-drop reorder UI). selectedItems acts as a visibility
// filter only.
//
// Issue #259: skills inside categories were ignoring the admin-set sort_order
// on the homepage when the skills section was explicitly configured, because
// filterBySelectedItemsWithDefault returns items in the selectedItems array
// order — overriding sort_order. This helper preserves the DB sort instead.
func filterBySelectedItemsKeepOrder(items []map[string]interface{}, selectedItems []string, isConfigured bool) []map[string]interface{} {
	if !isConfigured {
		return items
	}
	if len(selectedItems) == 0 {
		return nil
	}
	selectedSet := make(map[string]struct{}, len(selectedItems))
	for _, id := range selectedItems {
		selectedSet[id] = struct{}{}
	}
	result := make([]map[string]interface{}, 0, len(selectedItems))
	for _, item := range items {
		id, ok := item["id"].(string)
		if !ok {
			continue
		}
		if _, present := selectedSet[id]; present {
			result = append(result, item)
		}
	}
	return result
}

// filterBySelectedItemsWithDefault filters serialized records based on selected item IDs.
// If selectedItems is nil (not configured), returns all items (default behavior for unconfigured sections).
// If selectedItems is empty array (explicitly configured with no selections), returns empty.
// If selectedItems has specific IDs, returns only those items in the specified order.
func filterBySelectedItemsWithDefault(items []map[string]interface{}, selectedItems []string, isConfigured bool) []map[string]interface{} {
	// If section is not configured at all, show all items (backwards compat)
	if !isConfigured {
		return items
	}

	// If configured but no items selected, show nothing
	if len(selectedItems) == 0 {
		return nil
	}

	// Build a map for quick lookup
	itemMap := make(map[string]map[string]interface{})
	for _, item := range items {
		if id, ok := item["id"].(string); ok {
			itemMap[id] = item
		}
	}

	// Return items in the order specified by selectedItems
	var result []map[string]interface{}
	for _, id := range selectedItems {
		if item, exists := itemMap[id]; exists {
			result = append(result, item)
		}
	}
	return result
}

func incrementHomepageViewCount(app core.App, e *core.RequestEvent) {
	pb, ok := app.(*pocketbase.PocketBase)
	if !ok {
		return
	}
	if !services.IsFeatureEnabled(app, services.FeatureAnalytics, false) {
		return
	}
	// Bot-gate and ~30-min dedup so bots and rapid repeat hits from the same
	// visitor don't inflate the homepage counter. Uses the "homepage" sentinel
	// target and the same daily-salted IP hash as access_logs.
	// Internal SSR requests carry the frontend server's IP/UA, not the visitor's,
	// so count those unconditionally (matching pre-change behavior); direct hits
	// are bot-gated and ~30-min deduped.
	if e.Request.Header.Get("X-Internal") != "true" &&
		!services.ShouldCountView(GetUserAgent(e), GetIP(e), services.HomepageDedupTarget) {
		return
	}
	services.SafeGo(pb, "homepage-view-count", func() {
		settings, err := services.LoadSiteSettings(app)
		if err != nil || settings == nil || settings.Record == nil {
			return
		}
		settings.Record.Set("homepage_view_count", settings.Record.GetInt("homepage_view_count")+1)
		settings.Record.Set("homepage_last_viewed_at", time.Now())
		if err := app.Save(settings.Record); err != nil {
			app.Logger().Warn("Failed to update homepage view metrics", "error", err)
		}
	})
}

func getSectionConfig(settings *services.SiteSettings, sectionKey string) (enabled bool, items []string, isConfigured bool) {
	if settings == nil || settings.HomepageSections == nil {
		return true, nil, false
	}

	config, exists := settings.HomepageSections[sectionKey]
	if !exists {
		return true, nil, false
	}

	return config.Enabled, config.Items, true
}

// getSectionList returns the optional "list" filter configured for a section
// (currently only testimonials). Empty string ⇒ no list filter (show all).
func getSectionList(settings *services.SiteSettings, sectionKey string) string {
	if settings == nil || settings.HomepageSections == nil {
		return ""
	}
	config, exists := settings.HomepageSections[sectionKey]
	if !exists {
		return ""
	}
	return config.List
}

const sectionRecordBatchSize = 500

// sectionUsesAllPublic reports whether a section explicitly opted into
// all_public. A missing or unknown value means selected, so a section saved
// before itemSelection existed never shows more than its picks.
func sectionUsesAllPublic(section map[string]interface{}) bool {
	mode, _ := section["itemSelection"].(string)
	return mode == "all_public"
}

// allPublicQuery returns what all_public publishes for a section: the rules
// and order the homepage uses, with id breaking ties so the editor preview can
// match. ALL_PUBLIC_SORT in frontend/src/lib/facet-content.ts keeps the same sorts.
func allPublicQuery(section string) (filter string, sort string) {
	const published = "visibility = 'public' && is_draft = false"
	switch section {
	case "experience":
		return published, "-sort_order,-start_date,id"
	case "projects":
		return published, "-is_featured,-sort_order,id"
	case "education":
		return published, "-sort_order,-end_date,id"
	case "certifications":
		return published, "issuer,sort_order,-issue_date,id"
	case "awards":
		return published, "-sort_order,-awarded_at,id"
	case "skills":
		return "visibility = 'public'", "category,sort_order,id"
	case "posts":
		return published, "-published_at,id"
	case "talks":
		return published, "-sort_order,-date,id"
	case "testimonials":
		return "status = 'approved' && (visibility = 'public' || visibility = '')", "-featured,-sort_order,id"
	case "contacts":
		return "", "-is_primary,-sort_order,id"
	default:
		return published, "id"
	}
}

// findRecordsInBatches reads every match a batch at a time, so a section isn't
// cut off at a fixed record limit.
func findRecordsInBatches(app core.App, collection, filter, sort string, params dbx.Params) ([]*core.Record, error) {
	var records []*core.Record
	for offset := 0; ; offset += sectionRecordBatchSize {
		batch, err := app.FindRecordsByFilter(collection, filter, sort, sectionRecordBatchSize, offset, params)
		if err != nil {
			return nil, err
		}
		records = append(records, batch...)
		if len(batch) < sectionRecordBatchSize {
			return records, nil
		}
	}
}

// findAllPublicRecords returns a section's all_public records. Contacts have no
// visibility field, so their page scope decides what is available here.
func findAllPublicRecords(app core.App, collection, section, viewID string) ([]*core.Record, error) {
	filter, sort := allPublicQuery(section)
	records, err := findRecordsInBatches(app, collection, filter, sort, nil)
	if err != nil || section != "contacts" {
		return records, err
	}
	var available []*core.Record
	for _, record := range records {
		if contactVisibleInView(record, viewID) {
			available = append(available, record)
		}
	}
	return available, nil
}

// resolveSectionRecords returns what a view's section publishes, in published
// order. A testimonial list decides first, then all_public, then the section's
// visible picks. The data endpoint, the AI resume and view memberships read
// through here so they can't drift apart.
func resolveSectionRecords(app core.App, collection, sectionName string, section map[string]interface{}, viewID string) ([]*core.Record, error) {
	if list, _ := section["list"].(string); sectionName == "testimonials" && list != "" {
		filter, sort := allPublicQuery(sectionName)
		return findRecordsInBatches(app, collection, filter+" && list = {:list}", sort, dbx.Params{"list": list})
	}
	if sectionUsesAllPublic(section) {
		return findAllPublicRecords(app, collection, sectionName, viewID)
	}
	items, _ := section["items"].([]interface{})
	var records []*core.Record
	for _, item := range items {
		id, ok := item.(string)
		if !ok {
			continue
		}
		record, err := app.FindRecordById(collection, id)
		if err == nil && isRecordVisibleForSection(record, sectionName, viewID) {
			records = append(records, record)
		}
	}
	return records, nil
}

// viewSelectsRecord reports whether an enabled section of view picks the record
// right now. A private override alone proves nothing: it outlives deselecting
// the item, and a pick kept behind all_public isn't on the page. A custom
// section has no picks: its name carries the one record it shows, and callers
// ask for that record under its collection, custom_content.
func viewSelectsRecord(view *core.Record, sectionName, recordID string) bool {
	var sections []map[string]interface{}
	if err := json.Unmarshal([]byte(view.GetString("sections")), &sections); err != nil {
		return false
	}
	for _, section := range sections {
		name, _ := section["section"].(string)
		enabled, _ := section["enabled"].(bool)
		if sectionName == "custom_content" {
			if enabled && getCustomContentId(name) == recordID {
				return true
			}
			continue
		}
		if name != sectionName || !enabled || sectionUsesAllPublic(section) {
			continue
		}
		items, _ := section["items"].([]interface{})
		for _, item := range items {
			if id, _ := item.(string); id == recordID {
				return true
			}
		}
	}
	return false
}

// privateRecordOpensFromView reports whether a visitor may open a private
// record from the Facet they came from.
func privateRecordOpensFromView(app *pocketbase.PocketBase, crypto *services.CryptoService, share *services.ShareService, e *core.RequestEvent, record *core.Record, sectionName, fromSlug string) bool {
	return privateRecordFacet(app, crypto, share, e, record, sectionName, fromSlug) != ""
}

// privateRecordFacet returns the id of the Facet named by fromSlug if it opens
// the record for this visitor, otherwise "".
func privateRecordFacet(app *pocketbase.PocketBase, crypto *services.CryptoService, share *services.ShareService, e *core.RequestEvent, record *core.Record, sectionName, fromSlug string) string {
	if fromSlug == "" {
		return ""
	}
	views, err := app.FindRecordsByFilter(getTableName(app, "views"), "slug = {:slug} && is_active = true", "", 1, 0, dbx.Params{"slug": fromSlug})
	if err != nil || len(views) == 0 || !privateRecordOpensInView(app, crypto, share, e, record, sectionName, views[0]) {
		return ""
	}
	return views[0].Id
}

// privateRecordOpensInView reports whether a Facet opens a private record right
// now: it's active, the visitor can read it, it selects the record, and the
// record holds its override. Drafts never open. Opening a record doesn't count
// as another use of a share token; the visit to the Facet already did.
func privateRecordOpensInView(app *pocketbase.PocketBase, crypto *services.CryptoService, share *services.ShareService, e *core.RequestEvent, record *core.Record, sectionName string, view *core.Record) bool {
	if record.GetBool("is_draft") || !view.GetBool("is_active") {
		return false
	}
	if _, denial := checkViewAccess(app, crypto, share, e, view); denial != nil {
		return false
	}
	return isVisibleInView(record, view.Id) && viewSelectsRecord(view, sectionName, record.Id)
}

var errGrantViewNotFound = errors.New("view not found")

// grantSelectedPrivateRecords gives the private items a Facet selects right now
// their override on that Facet, and returns how many it changed. It reads the
// saved sections and records inside one transaction, which holds SQLite's only
// write connection, so two saves can't overwrite each other's grants. Any
// failure rolls every grant back.
func grantSelectedPrivateRecords(app core.App, viewID string) (int, error) {
	granted := 0
	err := app.RunInTransaction(func(txApp core.App) error {
		view, err := txApp.FindRecordById(getTableName(txApp, "views"), viewID)
		if errors.Is(err, sql.ErrNoRows) {
			return errGrantViewNotFound
		}
		if err != nil {
			return err
		}
		raw := view.GetString("sections")
		if raw == "" {
			return nil
		}
		var sections []map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &sections); err != nil {
			return err
		}
		for _, section := range sections {
			name, _ := section["section"].(string)
			enabled, _ := section["enabled"].(bool)
			list, _ := section["list"].(string)
			collection := getCollectionName(name)
			// Contacts keep their own page scope. Custom sections need an explicit
			// selection marker first. A list or all_public ignores the kept picks.
			if !enabled || collection == "" || name == "contacts" || isCustomContentSection(name) ||
				sectionUsesAllPublic(section) || (name == "testimonials" && list != "") {
				continue
			}
			items, _ := section["items"].([]interface{})
			for _, item := range items {
				id, _ := item.(string)
				changed, err := grantPrivateRecord(txApp, getTableName(txApp, collection), name, id, view.Id)
				if err != nil {
					return err
				}
				if changed {
					granted++
				}
			}
		}
		return nil
	})
	return granted, err
}

// grantPrivateRecord adds one pick's override for viewID, merging into the
// stored map so another Facet's override survives. A deleted pick, or one that
// isn't a published private item, is left alone.
func grantPrivateRecord(txApp core.App, collection, sectionName, id, viewID string) (bool, error) {
	record, err := txApp.FindRecordById(collection, id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if record.GetString("visibility") != "private" || record.GetBool("is_draft") || isVisibleInView(record, viewID) ||
		(sectionName == "testimonials" && record.GetString("status") != "approved") {
		return false, nil
	}
	var overrides map[string]interface{}
	if raw := record.GetString("view_visibility"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &overrides); err != nil {
			return false, err
		}
	}
	if overrides == nil {
		overrides = map[string]interface{}{}
	}
	overrides[viewID] = true
	record.Set("view_visibility", overrides)
	return true, txApp.Save(record)
}
