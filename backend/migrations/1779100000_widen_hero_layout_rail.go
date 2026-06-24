package migrations

import (
	"slices"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Widen the hero_layout SelectField to include 'rail' as the 6th allowed
// value on both the profile and views collections. Without this, the
// frontend's new rail layout can't be persisted (PocketBase rejects unknown
// SelectField values with 400 before the row reaches the DB).
//
// Migration is additive — it does NOT change defaults. Existing tenants keep
// whatever hero_layout they already have, and self-hosted continues to default
// new profiles to 'standard'.

func init() {
	m.Register(func(app core.App) error {
		for _, collName := range []string{"profile", "views"} {
			coll, err := app.FindCollectionByNameOrId(collName)
			if err != nil {
				return nil
			}
			field := coll.Fields.GetByName("hero_layout")
			if field == nil {
				continue
			}
			sel, ok := field.(*core.SelectField)
			if !ok {
				continue
			}
			if slices.Contains(sel.Values, "rail") {
				continue
			}
			sel.Values = append(sel.Values, "rail")
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
			field := coll.Fields.GetByName("hero_layout")
			if field == nil {
				continue
			}
			sel, ok := field.(*core.SelectField)
			if !ok {
				continue
			}
			next := sel.Values[:0]
			for _, v := range sel.Values {
				if v != "rail" {
					next = append(next, v)
				}
			}
			sel.Values = next
			app.Save(coll)
		}
		return nil
	})
}
