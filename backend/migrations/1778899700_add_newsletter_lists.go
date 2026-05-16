package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds newsletter_lists + subscriber_list_memberships for segment support.
//
// Ported from facets-sh (cloud) commit set, stripped of tier gating and
// webhook_events_log (which belongs to a separate feature). Self-hosted has
// no plan cap; the cap is exposed as 0 (unlimited) in the API response.
//
// Backfill: every existing subscriber is added to the seeded "default" list
// so existing self-hosted instances see zero behavioural change.
//
// ROLLBACK: drops both collections; no `subscribers.metadata` or
// `newsletter_sends.list_id` are added here (Liquid personalization deferred).
func init() {
	m.Register(func(app core.App) error {
		// ────────────────────────────────────────────────────────────
		// 1. newsletter_lists
		// ────────────────────────────────────────────────────────────
		var lists *core.Collection
		if existing, err := app.FindCollectionByNameOrId("newsletter_lists"); err == nil {
			lists = existing
		} else {
			superuser := "@request.auth.collectionName = \"_superusers\""
			lists = core.NewBaseCollection("newsletter_lists")
			lists.ListRule = &superuser
			lists.ViewRule = &superuser
			// Create/update/delete are server-side only (via hooks).
			lists.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 100})
			lists.Fields.Add(&core.TextField{Name: "slug", Required: true, Max: 50})
			lists.Fields.Add(&core.TextField{Name: "description", Max: 500})
			lists.Fields.Add(&core.TextField{Name: "sender_name", Max: 100})
			lists.Fields.Add(&core.EmailField{Name: "reply_to"})
			lists.Fields.Add(&core.TextField{Name: "welcome_subject", Max: 200})
			lists.Fields.Add(&core.TextField{Name: "welcome_html", Max: 100000})
			lists.Fields.Add(&core.BoolField{Name: "is_default"})
			lists.Fields.Add(&core.BoolField{Name: "is_active"})
			lists.Fields.Add(&core.NumberField{Name: "subscriber_count"})
			lists.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
			lists.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
			lists.Indexes = []string{
				"CREATE UNIQUE INDEX IF NOT EXISTS idx_newsletter_lists_slug ON newsletter_lists (slug)",
				"CREATE UNIQUE INDEX IF NOT EXISTS idx_newsletter_lists_default ON newsletter_lists (is_default) WHERE is_default = 1",
			}
			if err := app.Save(lists); err != nil {
				return err
			}
		}

		// Seed default list if absent.
		defaultList, err := app.FindFirstRecordByFilter("newsletter_lists", "slug = 'default'", nil)
		if err != nil || defaultList == nil {
			defaultList = core.NewRecord(lists)
			defaultList.Set("name", "Newsletter")
			defaultList.Set("slug", "default")
			defaultList.Set("description", "Default newsletter list")
			defaultList.Set("is_default", true)
			defaultList.Set("is_active", true)
			defaultList.Set("subscriber_count", 0)
			if err := app.Save(defaultList); err != nil {
				return err
			}
		}

		// ────────────────────────────────────────────────────────────
		// 2. subscriber_list_memberships (join table, source of truth)
		// ────────────────────────────────────────────────────────────
		if _, err := app.FindCollectionByNameOrId("subscriber_list_memberships"); err != nil {
			superuser := "@request.auth.collectionName = \"_superusers\""
			mem := core.NewBaseCollection("subscriber_list_memberships")
			mem.ListRule = &superuser
			mem.ViewRule = &superuser

			subsCol, sErr := app.FindCollectionByNameOrId("subscribers")
			if sErr != nil {
				return sErr
			}
			mem.Fields.Add(&core.RelationField{
				Name:          "subscriber_id",
				Required:      true,
				CollectionId:  subsCol.Id,
				CascadeDelete: true,
				MaxSelect:     1,
			})
			mem.Fields.Add(&core.RelationField{
				Name:          "list_id",
				Required:      true,
				CollectionId:  lists.Id,
				CascadeDelete: true,
				MaxSelect:     1,
			})
			mem.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
			mem.Indexes = []string{
				"CREATE UNIQUE INDEX IF NOT EXISTS idx_slm_sub_list ON subscriber_list_memberships (subscriber_id, list_id)",
				"CREATE INDEX IF NOT EXISTS idx_slm_list ON subscriber_list_memberships (list_id, created)",
			}
			if err := app.Save(mem); err != nil {
				return err
			}
		}

		// ────────────────────────────────────────────────────────────
		// 3. Backfill: every existing subscriber joins the default list.
		// Uses a single INSERT ... SELECT to avoid N round-trips.
		// ────────────────────────────────────────────────────────────
		_, err = app.DB().NewQuery(`
			INSERT INTO subscriber_list_memberships (id, subscriber_id, list_id, created)
			SELECT lower(hex(randomblob(7))) || '-' || substr(lower(hex(randomblob(4))), 1, 8),
			       s.id, {:list_id}, datetime('now')
			FROM subscribers s
			WHERE NOT EXISTS (
				SELECT 1 FROM subscriber_list_memberships m
				WHERE m.subscriber_id = s.id AND m.list_id = {:list_id}
			)
		`).Bind(dbx.Params{"list_id": defaultList.Id}).Execute()
		if err != nil {
			return err
		}

		// Seed the default list's subscriber_count from the live join table.
		var count int64
		if err := app.DB().NewQuery(
			"SELECT COUNT(*) FROM subscriber_list_memberships WHERE list_id = {:lid}",
		).Bind(dbx.Params{"lid": defaultList.Id}).Row(&count); err == nil {
			defaultList.Set("subscriber_count", count)
			_ = app.Save(defaultList)
		}

		return nil
	}, func(app core.App) error {
		// Rollback — drop in reverse dependency order.
		if col, err := app.FindCollectionByNameOrId("subscriber_list_memberships"); err == nil {
			_ = app.Delete(col)
		}
		if col, err := app.FindCollectionByNameOrId("newsletter_lists"); err == nil {
			_ = app.Delete(col)
		}
		return nil
	})
}
