package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds coupons collection for discount codes
func init() {
	m.Register(func(app core.App) error {
		if _, err := app.FindCollectionByNameOrId("coupons"); err == nil {
			return nil // Already exists
		}

		coupons := core.NewBaseCollection("coupons")
		// No public access - admin only via API hooks
		coupons.ListRule = nil
		coupons.ViewRule = nil

		coupons.Fields.Add(&core.TextField{
			Name:     "code",
			Required: true,
			Max:      50,
		})
		coupons.Fields.Add(&core.SelectField{
			Name:      "discount_type",
			MaxSelect: 1,
			Values:    []string{"percentage", "fixed"},
			Required:  true,
		})
		coupons.Fields.Add(&core.NumberField{
			Name:     "discount_value",
			Required: true,
			Min:      floatPtr(1),
		})
		coupons.Fields.Add(&core.SelectField{
			Name:      "applicable_type",
			MaxSelect: 1,
			Values:    []string{"all", "courses", "posts", "projects", "custom_content", "talks"},
		})
		coupons.Fields.Add(&core.JSONField{
			Name: "applicable_ids",
		})
		coupons.Fields.Add(&core.NumberField{
			Name: "max_uses",
			Min:  floatPtr(0),
		})
		coupons.Fields.Add(&core.NumberField{
			Name: "current_uses",
			Min:  floatPtr(0),
		})
		coupons.Fields.Add(&core.DateField{
			Name: "expires_at",
		})
		coupons.Fields.Add(&core.BoolField{
			Name: "active",
		})
		coupons.Fields.Add(&core.AutodateField{
			Name:     "created",
			OnCreate: true,
		})

		coupons.Indexes = []string{
			"CREATE UNIQUE INDEX idx_coupons_code ON coupons(code)",
		}

		return app.Save(coupons)
	}, func(app core.App) error {
		if collection, err := app.FindCollectionByNameOrId("coupons"); err == nil {
			return app.Delete(collection)
		}
		return nil
	})
}
