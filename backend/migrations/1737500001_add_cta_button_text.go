package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		for _, collName := range []string{"profile", "views"} {
			coll, err := app.FindCollectionByNameOrId(collName)
			if err != nil {
				continue
			}

			if coll.Fields.GetByName("cta_button_text") != nil {
				continue
			}

			coll.Fields.Add(&core.TextField{
				Name:     "cta_button_text",
				Required: false,
			})

			if err := app.Save(coll); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		for _, collName := range []string{"profile", "views"} {
			coll, err := app.FindCollectionByNameOrId(collName)
			if err != nil {
				continue
			}

			field := coll.Fields.GetByName("cta_button_text")
			if field != nil {
				coll.Fields.RemoveById(field.GetId())
				app.Save(coll)
			}
		}
		return nil
	})
}
