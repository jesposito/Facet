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

	// ALL content collections require authentication for direct access
	// Public data flows through /api/view/{slug}/data which applies visibility rules
	allManagedCollections := []string{
		// Content collections
		"profile",
		"experience",
		"projects",
		"education",
		"certifications",
		"skills",
		"posts",
		"talks",
		"views",
		// Sensitive/admin collections
		"share_tokens",
		"sources",
		"ai_providers",
		"import_proposals",
		"settings",
	}

	for _, name := range allManagedCollections {
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

	if len(updated) > 0 {
	}

	return nil
}
