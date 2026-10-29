package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		for _, name := range []string{"projects", "demo_projects"} {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil || collection == nil || collection.Fields.GetByName("experience") != nil {
				continue
			}
			collection.Fields.Add(&core.JSONField{Name: "experience", MaxSize: 500000})
			if err := app.Save(collection); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		for _, name := range []string{"projects", "demo_projects"} {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil || collection == nil {
				continue
			}
			collection.Fields.RemoveByName("experience")
			if err := app.Save(collection); err != nil {
				return err
			}
		}
		return nil
	})
}
