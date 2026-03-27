package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		expCollection, err := app.FindCollectionByNameOrId("experience")
		if err != nil {
			return err
		}

		if expCollection.Fields.GetByName("company_group_id") == nil {
			expCollection.Fields.Add(&core.TextField{
				Name:    "company_group_id",
				Max:     50,
			})

			if err := app.Save(expCollection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		expCollection, err := app.FindCollectionByNameOrId("experience")
		if err == nil {
			field := expCollection.Fields.GetByName("company_group_id")
			if field != nil {
				expCollection.Fields.RemoveById(field.GetId())
				app.Save(expCollection)
			}
		}

		return nil
	})
}
