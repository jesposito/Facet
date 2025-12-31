package hooks

import (
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterCollectionRules ensures all collections have proper access rules
// This runs at startup to fix any collections that were created without rules
func RegisterCollectionRules(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Run in background to not block startup
		go func() {
			if err := ensureCollectionRules(app); err != nil {
				log.Printf("Warning: Failed to update collection rules: %v", err)
			}
		}()
		return se.Next()
	})
}

func ensureCollectionRules(app *pocketbase.PocketBase) error {
	// Rule that allows any authenticated user
	authRule := "@request.auth.id != ''"

	// Admin-only collections that authenticated users need to access
	adminCollections := []string{
		"sources",
		"ai_providers",
		"import_proposals",
		"settings",
		"share_tokens",
	}

	for _, name := range adminCollections {
		collection, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			continue // Collection doesn't exist yet
		}

		// Only update if rules are not set (nil means superuser-only)
		needsUpdate := false

		if collection.ListRule == nil {
			collection.ListRule = &authRule
			needsUpdate = true
		}
		if collection.ViewRule == nil {
			collection.ViewRule = &authRule
			needsUpdate = true
		}
		if collection.CreateRule == nil {
			collection.CreateRule = &authRule
			needsUpdate = true
		}
		if collection.UpdateRule == nil {
			collection.UpdateRule = &authRule
			needsUpdate = true
		}
		if collection.DeleteRule == nil {
			collection.DeleteRule = &authRule
			needsUpdate = true
		}

		if needsUpdate {
			if err := app.Save(collection); err != nil {
				log.Printf("Warning: Failed to update rules for %s: %v", name, err)
			} else {
				log.Printf("Updated collection rules: %s", name)
			}
		}
	}

	// Public collections also need authenticated user write access
	publicCollections := []string{
		"profile",
		"experience",
		"projects",
		"education",
		"certifications",
		"skills",
		"posts",
		"talks",
		"views",
	}

	for _, name := range publicCollections {
		collection, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			continue
		}

		needsUpdate := false

		// These already have public read, add authenticated write
		if collection.CreateRule == nil {
			collection.CreateRule = &authRule
			needsUpdate = true
		}
		if collection.UpdateRule == nil {
			collection.UpdateRule = &authRule
			needsUpdate = true
		}
		if collection.DeleteRule == nil {
			collection.DeleteRule = &authRule
			needsUpdate = true
		}

		if needsUpdate {
			if err := app.Save(collection); err != nil {
				log.Printf("Warning: Failed to update rules for %s: %v", name, err)
			} else {
				log.Printf("Updated collection rules: %s", name)
			}
		}
	}

	return nil
}
