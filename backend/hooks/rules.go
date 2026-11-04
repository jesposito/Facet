package hooks

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// RegisterCollectionRules ensures all collections have proper access rules.
// This runs at startup to enforce deny-by-default security posture.
//
// SECURITY MODEL:
// - Facet default collection rules admit only the users owner collection
// - PocketBase superusers retain their native rule bypass
// - Public data is served ONLY through custom API endpoints that enforce visibility rules
// - Server-side app.FindRecordsByFilter() calls bypass these rules (by design)
func RegisterCollectionRules(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := enforceCollectionRules(app); err != nil {
			return fmt.Errorf("enforce native collection access rules: %w", err)
		}
		// Run after the priority-zero admin session guard has removed incomplete
		// owner sign-ins. A denied list must not look like a successfully empty list.
		se.Router.Bind(&hook.Handler[*core.RequestEvent]{
			Id: "facetOwnerRecordLists", Priority: 1, Func: requireFacetOwnerList,
		})
		return se.Next()
	})
}

func enforceCollectionRules(app *pocketbase.PocketBase) error {
	// Default owner access is bound to the users auth collection, not any token.
	authRule := facetOwnerRule

	// Track what we changed for logging
	var updated []string

	// These collections are read or changed directly by the owner admin UI.
	// Public data flows through custom endpoints that apply visibility rules.
	// Nil rules on collections outside this registry remain superuser-only.
	ownerManagedCollections := []string{
		"profile",
		"experience",
		"projects",
		"education",
		"certifications",
		"skills",
		"posts",
		"talks",
		"views",
		"courses",
		"modules",
		"lessons",
		"share_tokens",
		"sources",
		"ai_providers",
		"import_proposals",
		"settings",
	}

	for _, name := range ownerManagedCollections {
		collection, err := app.FindCollectionByNameOrId(name)
		if errors.Is(err, sql.ErrNoRows) {
			continue // Optional collection doesn't exist yet.
		}
		if err != nil {
			return fmt.Errorf("read collection rules for %s: %w", name, err)
		}
		if collection.System || collection.IsAuth() {
			continue
		}

		needsUpdate := false

		// Initialize default owner read access
		// Empty string means "anyone can access" - this is the security hole we're closing
		if collection.ListRule == nil || *collection.ListRule == "" {
			collection.ListRule = &authRule
			needsUpdate = true
		}
		if collection.ViewRule == nil || *collection.ViewRule == "" {
			collection.ViewRule = &authRule
			needsUpdate = true
		}

		// Initialize default owner write access
		if collection.CreateRule == nil || *collection.CreateRule == "" {
			collection.CreateRule = &authRule
			needsUpdate = true
		}
		if collection.UpdateRule == nil || *collection.UpdateRule == "" {
			collection.UpdateRule = &authRule
			needsUpdate = true
		}
		if collection.DeleteRule == nil || *collection.DeleteRule == "" {
			collection.DeleteRule = &authRule
			needsUpdate = true
		}

		if needsUpdate {
			if err := app.Save(collection); err != nil {
				return fmt.Errorf("save collection rules for %s: %w", name, err)
			}
			updated = append(updated, name)
		}
	}

	// Close any remaining public rule on every other collection, including
	// ones added after this list was written. Nothing in Facet is meant to
	// be read or written anonymously through /api/collections.
	collections, err := app.FindAllCollections()
	if err != nil {
		return fmt.Errorf("list collection rules: %w", err)
	}
	for _, collection := range collections {
		// Auth collections are left alone: OAuth sign-in checks the users
		// create rule. RegisterAdminAuth blocks direct sign-up instead.
		if collection.System || collection.IsAuth() {
			continue
		}
		changed := tightenFacetDefaultRules(collection)
		publicRule := authRule
		if !isFacetDefaultRuleCollection(collection.Name) {
			// Preserve the pre-existing public-rule sweep for operator tables;
			// this change tightens legacy defaults only on Facet's own schema.
			publicRule = "@request.auth.id != ''"
		}
		if lockPublicRules(collection, publicRule) {
			changed = true
		}
		if changed {
			if err := app.Save(collection); err != nil {
				return fmt.Errorf("save collection rules for %s: %w", collection.Name, err)
			}
			updated = append(updated, collection.Name)
		}
	}

	if len(updated) > 0 {
		log.Printf("Collection rules: restricted native access on %v", updated)
	}

	return nil
}

// superuserOnlyCollections are only reached through Facet's own admin
// endpoints. Their migration set "" believing it meant superuser-only, but in
// PocketBase "" is public and nil is superuser-only.
var superuserOnlyCollections = map[string]bool{
	"webhooks":           true,
	"webhook_deliveries": true,
}

// lockPublicRules replaces each public ("") rule on a collection and reports
// whether anything changed. Restricted rules, including nil, are left as set.
func lockPublicRules(collection *core.Collection, authRule string) bool {
	changed := false
	for _, rule := range []**string{
		&collection.ListRule,
		&collection.ViewRule,
		&collection.CreateRule,
		&collection.UpdateRule,
		&collection.DeleteRule,
	} {
		if *rule == nil || **rule != "" {
			continue
		}
		if superuserOnlyCollections[collection.Name] {
			*rule = nil
		} else {
			restricted := authRule
			*rule = &restricted
		}
		changed = true
	}
	return changed
}

// PocketBase exposes collectionName through the request auth record's public
// export. Checking id alone also admits tokens from operator-added auth tables.
const facetOwnerRule = "@request.auth.collectionName = 'users' && @request.auth.id != ''"

var legacyFacetAuthRule = regexp.MustCompile(`^\s*@request\.auth\.id\s*!=\s*(?:''|"")\s*$`)

// Only Facet's own schema and its known demo shadows are eligible for the
// legacy-default migration. Operator-defined collections and custom expressions
// retain their rules. New Facet tables using default owner rules belong here.
func isFacetDefaultRuleCollection(name string) bool {
	switch name {
	case "profile", "experience", "projects", "education", "certifications",
		"skills", "posts", "talks", "views", "courses", "modules", "lessons",
		"share_tokens", "sources", "ai_providers", "import_proposals", "settings",
		"access_logs", "admin_tags", "api_keys", "audit_logs", "awards",
		"comment_reports", "comments", "contact_methods", "coupons",
		"course_certificates", "course_progress", "custom_content", "download_logs",
		"email_verification_tokens", "external_media", "lesson_progress",
		"media_display_names", "newsletter_events", "newsletter_lists",
		"newsletter_recipients", "newsletter_sends", "purchases", "quiz_attempts",
		"quiz_questions", "quizzes", "resume_imports", "site_settings",
		"subscriber_list_memberships", "subscriber_tags", "subscribers", "system_alerts",
		"testimonial_requests", "testimonials", "uploads", "view_exports",
		"webhook_deliveries", "webhooks",
		"demo_profile", "demo_experience", "demo_projects", "demo_education",
		"demo_skills", "demo_certifications", "demo_posts", "demo_talks",
		"demo_awards", "demo_views", "demo_share_tokens", "demo_contact_methods":
		return true
	default:
		return false
	}
}

func tightenFacetDefaultRules(collection *core.Collection) bool {
	if collection.System || collection.IsAuth() || !isFacetDefaultRuleCollection(collection.Name) {
		return false
	}
	changed := false
	for _, rule := range []**string{
		&collection.ListRule, &collection.ViewRule, &collection.CreateRule,
		&collection.UpdateRule, &collection.DeleteRule,
	} {
		if *rule == nil || !legacyFacetAuthRule.MatchString(**rule) {
			continue
		}
		if superuserOnlyCollections[collection.Name] {
			*rule = nil
		} else {
			ownerRule := facetOwnerRule
			*rule = &ownerRule
		}
		changed = true
	}
	return changed
}

func requireFacetOwnerList(e *core.RequestEvent) error {
	if (e.Request.Method != http.MethodGet && e.Request.Method != http.MethodHead) ||
		!strings.HasPrefix(e.Request.URL.Path, "/api/collections/") ||
		!strings.HasSuffix(e.Request.URL.Path, "/records") || e.Request.PathValue("id") != "" {
		return e.Next()
	}
	collection, err := e.App.FindCachedCollectionByNameOrId(e.Request.PathValue("collection"))
	if err != nil || collection == nil || collection.System || collection.IsAuth() ||
		!isFacetDefaultRuleCollection(collection.Name) || collection.ListRule == nil ||
		*collection.ListRule != facetOwnerRule {
		return e.Next()
	}
	if e.Auth == nil {
		return e.UnauthorizedError("Owner sign-in is required.", nil)
	}
	if !isSuperuser(e) {
		return e.ForbiddenError("Owner access is required.", nil)
	}
	return e.Next()
}
