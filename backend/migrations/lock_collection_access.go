package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// SECURITY: Lock down direct PocketBase collection access
		//
		// RATIONALE:
		// - All public data access must flow through our custom API endpoints
		//   (/api/view/{slug}/data, etc.) which enforce visibility and draft rules
		// - Direct collection access via /api/collections/{name}/records bypasses these rules
		// - This migration sets all collections to require authentication
		// - Our custom endpoints use app.FindRecordsByFilter() which bypasses these rules
		//   (server-side calls are not subject to collection rules)
		//
		// COLLECTION CATEGORIES:
		// 1. Content collections (profile, experience, etc.) - auth required for direct access
		// 2. Sensitive collections (share_tokens, ai_providers, etc.) - auth required
		// 3. System collections (users) - managed by PocketBase

		// Rule that requires authentication
		authRule := "@request.auth.id != ''"

		// Content collections: previously had public read, now require auth
		// Public rendering flows through /api/view/{slug}/data which applies visibility rules
		contentCollections := []string{
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

		for _, name := range contentCollections {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				continue // Collection doesn't exist
			}

			// Lock down read access - require authentication
			collection.ListRule = &authRule
			collection.ViewRule = &authRule

			// Write access requires authentication
			collection.CreateRule = &authRule
			collection.UpdateRule = &authRule
			collection.DeleteRule = &authRule

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		// Sensitive/admin collections: require authentication for all operations
		sensitiveCollections := []string{
			"share_tokens",
			"sources",
			"ai_providers",
			"import_proposals",
			"settings",
		}

		for _, name := range sensitiveCollections {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				continue
			}

			collection.ListRule = &authRule
			collection.ViewRule = &authRule
			collection.CreateRule = &authRule
			collection.UpdateRule = &authRule
			collection.DeleteRule = &authRule

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		// Note: 'users' collection is an auth collection managed by PocketBase
		// Its rules are handled separately and shouldn't be modified here

		return nil
	}, func(app core.App) error {
		// Rollback: restore public read access to content collections
		// (This reverts to the insecure state - only use if necessary)
		publicRule := ""

		contentCollections := []string{
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

		for _, name := range contentCollections {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				continue
			}

			collection.ListRule = &publicRule
			collection.ViewRule = &publicRule

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		return nil
	})
}
