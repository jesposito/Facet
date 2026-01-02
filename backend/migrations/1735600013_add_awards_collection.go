package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		if _, err := app.FindCollectionByNameOrId("awards"); err == nil {
			return nil
		}

		awards := core.NewBaseCollection("awards")
		rule := "visibility = 'public' && is_draft = false"
		awards.ListRule = &rule
		awards.ViewRule = &rule

		awards.Fields.Add(&core.TextField{Name: "title", Required: true, Max: 300})
		awards.Fields.Add(&core.TextField{Name: "issuer", Max: 200})
		awards.Fields.Add(&core.DateField{Name: "awarded_at"})
		awards.Fields.Add(&core.EditorField{Name: "description"})
		awards.Fields.Add(&core.URLField{Name: "url"})
		awards.Fields.Add(&core.SelectField{Name: "visibility", Values: []string{"public", "unlisted", "private"}, MaxSelect: 1})
		awards.Fields.Add(&core.BoolField{Name: "is_draft"})
		awards.Fields.Add(&core.NumberField{Name: "sort_order"})

		return app.Save(awards)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("awards")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
