package migrations

import (
	"slices"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Widen the font_pack SelectField to include 'soft-premium' as an allowed value
// on both the profile and views collections. The original add_font_pack migration
// shipped without 'soft-premium', so PocketBase silently strips the value on write
// (SelectField rejects/drops unknown values before the row reaches the DB) and the
// Soft Premium font pack never persists.
//
// Migration is additive and idempotent — it does NOT change defaults. Existing
// tenants keep whatever font_pack they already have; this only appends the missing
// allowed value so the frontend's soft-premium selection round-trips correctly.

func init() {
	m.Register(func(app core.App) error {
		for _, collName := range []string{"profile", "views"} {
			coll, err := app.FindCollectionByNameOrId(collName)
			if err != nil {
				continue
			}
			field := coll.Fields.GetByName("font_pack")
			if field == nil {
				continue
			}
			sel, ok := field.(*core.SelectField)
			if !ok {
				continue
			}
			if slices.Contains(sel.Values, "soft-premium") {
				continue
			}
			sel.Values = append(sel.Values, "soft-premium")
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
			field := coll.Fields.GetByName("font_pack")
			if field == nil {
				continue
			}
			sel, ok := field.(*core.SelectField)
			if !ok {
				continue
			}
			next := sel.Values[:0]
			for _, v := range sel.Values {
				if v != "soft-premium" {
					next = append(next, v)
				}
			}
			sel.Values = next
			app.Save(coll)
		}
		return nil
	})
}
