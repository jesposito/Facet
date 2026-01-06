package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collections := []string{
			"experience",
			"projects",
			"education",
			"certifications",
			"skills",
			"posts",
			"talks",
			"awards",
		}

		for _, collName := range collections {
			collection, err := app.FindCollectionByNameOrId(collName)
			if err != nil {
				continue
			}

			if collection.Fields.GetByName("view_visibility") != nil {
				continue
			}

			collection.Fields.Add(&core.JSONField{
				Name:     "view_visibility",
				Required: false,
			})

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		collections := []string{
			"experience",
			"projects",
			"education",
			"certifications",
			"skills",
			"posts",
			"talks",
			"awards",
		}

		for _, collName := range collections {
			collection, err := app.FindCollectionByNameOrId(collName)
			if err != nil {
				continue
			}

			field := collection.Fields.GetByName("view_visibility")
			if field != nil {
				collection.Fields.RemoveById(field.GetId())
				if err := app.Save(collection); err != nil {
					return err
				}
			}
		}

		return nil
	})
}
