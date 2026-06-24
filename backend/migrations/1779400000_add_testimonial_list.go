package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds a nullable `list` TextField to testimonial_requests and testimonials.
//
// A "list" is a lightweight, free-text grouping label (e.g. "Clients",
// "Conference"). It is set on a testimonial_request when the operator generates
// a request link, and copied onto the resulting testimonial when it is
// submitted, so approved testimonials inherit the list they were collected
// under. A facet's testimonials section can then point at a list and have
// matching approved testimonials surface automatically (see GitHub #283).
//
// Empty `list` ⇒ today's behavior exactly: requests/testimonials carry no list,
// and a section with no list filter shows everything as before. Existing
// instances are unaffected until an operator starts using lists.
func init() {
	m.Register(func(app core.App) error {
		for _, name := range []string{"testimonial_requests", "testimonials"} {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				// Collection doesn't exist (shouldn't happen post-init); skip.
				continue
			}
			if collection.Fields.GetByName("list") == nil {
				collection.Fields.Add(&core.TextField{
					Name: "list",
					Max:  100,
				})
				if err := app.Save(collection); err != nil {
					return err
				}
			}
		}
		return nil
	}, func(app core.App) error {
		for _, name := range []string{"testimonial_requests", "testimonials"} {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				continue
			}
			if f := collection.Fields.GetByName("list"); f != nil {
				collection.Fields.RemoveById(f.GetId())
				if err := app.Save(collection); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
