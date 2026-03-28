package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds access_tier, price, paywall_preview_paragraphs fields to posts, projects,
// custom_content, courses, and talks collections for content gating.
// Also creates the purchases collection to track content purchases.
func init() {
	m.Register(func(app core.App) error {
		// Add access_tier and price to content collections (posts, projects, custom_content already exist)
		contentCollections := []string{"posts", "projects", "custom_content"}

		for _, name := range contentCollections {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				continue // Skip if collection doesn't exist
			}

			if collection.Fields.GetByName("access_tier") == nil {
				collection.Fields.Add(&core.SelectField{
					Name:      "access_tier",
					MaxSelect: 1,
					Values:    []string{"free", "paid"},
				})
			}

			if collection.Fields.GetByName("price") == nil {
				collection.Fields.Add(&core.NumberField{
					Name: "price",
					Min:  floatPtr(0),
				})
			}

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		// Add access_tier, price to talks
		if talks, err := app.FindCollectionByNameOrId("talks"); err == nil {
			if talks.Fields.GetByName("access_tier") == nil {
				talks.Fields.Add(&core.SelectField{
					Name:      "access_tier",
					MaxSelect: 1,
					Values:    []string{"free", "paid"},
				})
			}
			if talks.Fields.GetByName("price") == nil {
				talks.Fields.Add(&core.NumberField{
					Name: "price",
					Min:  floatPtr(0),
				})
			}
			if err := app.Save(talks); err != nil {
				return err
			}
		}

		// Add paywall_preview_paragraphs to posts
		if posts, err := app.FindCollectionByNameOrId("posts"); err == nil {
			if posts.Fields.GetByName("paywall_preview_paragraphs") == nil {
				posts.Fields.Add(&core.NumberField{
					Name: "paywall_preview_paragraphs",
					Min:  floatPtr(0),
					Max:  floatPtr(20),
				})
			}
			if err := app.Save(posts); err != nil {
				return err
			}
		}

		// Create purchases collection
		if _, err := app.FindCollectionByNameOrId("purchases"); err == nil {
			return nil // Already exists
		}

		purchases := core.NewBaseCollection("purchases")

		authRule := "@request.auth.id != ''"
		purchases.ListRule = &authRule
		purchases.ViewRule = &authRule
		purchases.DeleteRule = &authRule
		// No public create/update - backend only

		purchases.Fields.Add(&core.EmailField{
			Name:     "buyer_email",
			Required: true,
		})
		purchases.Fields.Add(&core.TextField{
			Name:     "content_type",
			Required: true,
			Max:      50,
		})
		purchases.Fields.Add(&core.TextField{
			Name:     "content_id",
			Required: true,
			Max:      15,
		})
		purchases.Fields.Add(&core.TextField{
			Name: "stripe_payment_intent_id",
			Max:  255,
		})
		purchases.Fields.Add(&core.TextField{
			Name: "stripe_checkout_session_id",
			Max:  255,
		})
		purchases.Fields.Add(&core.NumberField{
			Name: "amount",
			Min:  floatPtr(0),
		})
		purchases.Fields.Add(&core.TextField{
			Name: "currency",
			Max:  3,
		})
		purchases.Fields.Add(&core.SelectField{
			Name:      "status",
			MaxSelect: 1,
			Values:    []string{"completed", "refunded"},
		})
		purchases.Fields.Add(&core.TextField{
			Name:     "access_token",
			Required: true,
			Max:      512,
		})

		purchases.Indexes = []string{
			"CREATE UNIQUE INDEX idx_purchases_access_token ON purchases (access_token)",
			"CREATE INDEX idx_purchases_content ON purchases (content_type, content_id)",
			"CREATE INDEX idx_purchases_email ON purchases (buyer_email)",
			"CREATE INDEX idx_purchases_stripe ON purchases (stripe_payment_intent_id)",
		}

		return app.Save(purchases)
	}, func(app core.App) error {
		// Rollback: remove purchases collection
		if collection, err := app.FindCollectionByNameOrId("purchases"); err == nil {
			if err := app.Delete(collection); err != nil {
				return err
			}
		}

		// Remove access_tier and price from content collections
		contentCollections := []string{"posts", "projects", "custom_content", "talks"}
		for _, name := range contentCollections {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				continue
			}

			fields := collection.Fields
			if f := fields.GetByName("access_tier"); f != nil {
				fields.RemoveById(f.GetId())
			}
			if f := fields.GetByName("price"); f != nil {
				fields.RemoveById(f.GetId())
			}
			_ = app.Save(collection)
		}

		// Remove paywall_preview_paragraphs from posts
		if posts, err := app.FindCollectionByNameOrId("posts"); err == nil {
			if f := posts.Fields.GetByName("paywall_preview_paragraphs"); f != nil {
				posts.Fields.RemoveById(f.GetId())
			}
			_ = app.Save(posts)
		}

		return nil
	})
}
