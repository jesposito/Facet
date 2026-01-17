package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		authRule := "@request.auth.id != ''"

		collections := []string{
			"testimonials",
			"testimonial_requests",
			"email_verification_tokens",
		}

		for _, name := range collections {
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

		return nil
	}, func(app core.App) error {
		return nil
	})
}
