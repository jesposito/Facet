package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("views")
		if err != nil {
			return err
		}

		// Check if field already exists
		if collection.Fields.GetByName("hero_location") != nil {
			return nil
		}

		// Add hero_location field for per-view location override
		// Max 500 to allow extended location strings like "Wellington, New Zealand | US Citizen | W-2 or EOR-ready"
		collection.Fields.Add(&core.TextField{
			Name: "hero_location",
			Max:  500,
		})

		return app.Save(collection)
	}, nil)
}
