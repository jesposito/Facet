package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return err
		}

		// Add JSON field to store homepage section order
		// Format: ["experience", "projects", "custom:abc123", "education", ...]
		// Standard sections: experience, projects, education, certifications, awards, skills, posts, talks
		// Custom content sections: custom:{id}
		collection.Fields.Add(&core.JSONField{
			Name:    "homepage_section_order",
			MaxSize: 100000,
		})

		if err := app.Save(collection); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("site_settings")
		if err != nil {
			return nil
		}

		field := collection.Fields.GetByName("homepage_section_order")
		if field != nil {
			collection.Fields.RemoveByName("homepage_section_order")
			return app.Save(collection)
		}

		return nil
	})
}
