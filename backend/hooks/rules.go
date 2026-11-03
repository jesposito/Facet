package hooks

import (
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterCollectionRules ensures all collections have proper access rules.
// This runs at startup to enforce deny-by-default security posture.
//
// SECURITY MODEL:
// - All direct PocketBase collection access requires authentication
// - Public data is served ONLY through custom API endpoints that enforce visibility rules
// - Server-side app.FindRecordsByFilter() calls bypass these rules (by design)
func RegisterCollectionRules(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := enforceCollectionRules(app); err != nil {
			log.Printf("ERROR: Failed to enforce collection rules: %v", err)
			// Don't fail startup, but log prominently
		}
		return se.Next()
	})
}

func enforceCollectionRules(app *pocketbase.PocketBase) error {
	// Rule that requires any authenticated user
	authRule := "@request.auth.id != ''"

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
		if err != nil {
			continue // Collection doesn't exist yet
		}

		needsUpdate := false

		// Enforce auth-only read access
		// Empty string means "anyone can access" - this is the security hole we're closing
		if collection.ListRule == nil || *collection.ListRule == "" {
			collection.ListRule = &authRule
			needsUpdate = true
		}
		if collection.ViewRule == nil || *collection.ViewRule == "" {
			collection.ViewRule = &authRule
			needsUpdate = true
		}

		// Enforce auth-only write access
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
				log.Printf("Warning: Failed to update rules for %s: %v", name, err)
			} else {
				updated = append(updated, name)
			}
		}
	}

	// Close any remaining public rule on every other collection, including
	// ones added after this list was written. Nothing in Facet is meant to
	// be read or written anonymously through /api/collections.
	collections, err := app.FindAllCollections()
	if err != nil {
		return err
	}
	for _, collection := range collections {
		// Auth collections are left alone: OAuth sign-in checks the users
		// create rule. RegisterAdminAuth blocks direct sign-up instead.
		if collection.System || collection.IsAuth() {
			continue
		}
		if lockPublicRules(collection, authRule) {
			if err := app.Save(collection); err != nil {
				log.Printf("Warning: Failed to close public rules for %s: %v", collection.Name, err)
			} else {
				updated = append(updated, collection.Name)
			}
		}
	}

	if len(updated) > 0 {
		log.Printf("Collection rules: closed anonymous access on %v", updated)
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
