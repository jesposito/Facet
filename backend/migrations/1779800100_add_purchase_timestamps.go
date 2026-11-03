package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Purchases were created without created/updated fields, so drip schedules
// measured "days after purchase" against a field that didn't exist and every
// module unlocked at once. Purchases made after this migration record when
// they happened. Older rows are left empty: there's no honest date to give
// them, and the drip check treats an unknown date as already unlocked, which
// is how those buyers have always seen the course.
func init() {
	m.Register(func(app core.App) error {
		purchases, err := app.FindCollectionByNameOrId("purchases")
		if err != nil {
			return nil
		}
		changed := false
		if purchases.Fields.GetByName("created") == nil {
			purchases.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
			changed = true
		}
		if purchases.Fields.GetByName("updated") == nil {
			purchases.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
			changed = true
		}
		if !changed {
			return nil
		}
		return app.Save(purchases)
	}, func(app core.App) error {
		purchases, err := app.FindCollectionByNameOrId("purchases")
		if err != nil {
			return nil
		}
		for _, name := range []string{"created", "updated"} {
			if f := purchases.Fields.GetByName(name); f != nil {
				purchases.Fields.RemoveById(f.GetId())
			}
		}
		return app.Save(purchases)
	})
}
