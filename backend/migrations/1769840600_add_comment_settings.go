package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds comment moderation settings to the site_settings collection.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return err
		}

		if collection.Fields.GetByName("comments_require_approval") == nil {
			collection.Fields.Add(&core.BoolField{
				Name: "comments_require_approval",
			})
		}

		if collection.Fields.GetByName("comments_auto_close_days") == nil {
			collection.Fields.Add(&core.NumberField{
				Name: "comments_auto_close_days",
				Min:  floatPtr(0),
			})
		}

		if collection.Fields.GetByName("comments_blocked_words") == nil {
			collection.Fields.Add(&core.TextField{
				Name: "comments_blocked_words",
				Max:  5000,
			})
		}

		return app.Save(collection)
	}, func(app core.App) error {
		// Rollback: remove comment settings fields
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}

		fields := collection.Fields
		for _, fieldName := range []string{"comments_require_approval", "comments_auto_close_days", "comments_blocked_words"} {
			if f := fields.GetByName(fieldName); f != nil {
				fields.RemoveById(f.GetId())
			}
		}

		_ = app.Save(collection)
		return nil
	})
}
