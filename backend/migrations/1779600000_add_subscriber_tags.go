package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds subscriber_tags and JSON tag fields used by newsletter CSV import.
func init() {
	m.Register(func(app core.App) error {
		if _, err := app.FindCollectionByNameOrId("subscriber_tags"); err != nil {
			superuser := "@request.auth.collectionName = \"_superusers\""
			tags := core.NewBaseCollection("subscriber_tags")
			tags.ListRule = &superuser
			tags.ViewRule = &superuser
			tags.CreateRule = &superuser
			tags.UpdateRule = &superuser
			tags.DeleteRule = &superuser
			tags.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 100})
			tags.Fields.Add(&core.TextField{Name: "color", Max: 7})
			tags.Fields.Add(&core.NumberField{Name: "subscriber_count"})
			tags.Indexes = []string{
				"CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriber_tags_name ON subscriber_tags(name)",
			}
			if err := app.Save(tags); err != nil {
				return err
			}
		}

		subscribers, err := app.FindCollectionByNameOrId("subscribers")
		if err != nil {
			return err
		}
		if subscribers.Fields.GetByName("tags") == nil {
			subscribers.Fields.Add(&core.JSONField{Name: "tags", MaxSize: 10000})
			if err := app.Save(subscribers); err != nil {
				return err
			}
		}

		sends, err := app.FindCollectionByNameOrId("newsletter_sends")
		if err != nil {
			return err
		}
		if sends.Fields.GetByName("target_tags") == nil {
			sends.Fields.Add(&core.JSONField{Name: "target_tags", MaxSize: 10000})
			if err := app.Save(sends); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		if sends, err := app.FindCollectionByNameOrId("newsletter_sends"); err == nil {
			if f := sends.Fields.GetByName("target_tags"); f != nil {
				sends.Fields.RemoveById(f.GetId())
				_ = app.Save(sends)
			}
		}
		if subscribers, err := app.FindCollectionByNameOrId("subscribers"); err == nil {
			if f := subscribers.Fields.GetByName("tags"); f != nil {
				subscribers.Fields.RemoveById(f.GetId())
				_ = app.Save(subscribers)
			}
		}
		if tags, err := app.FindCollectionByNameOrId("subscriber_tags"); err == nil {
			_ = app.Delete(tags)
		}
		return nil
	})
}
