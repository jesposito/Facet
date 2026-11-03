package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Gives each newsletter send a recorded identity, and lets public signups
// name the list they're joining.
//
// newsletter_sends:
//   - request_key: the compose page's key for one Send click. A retry of the
//     same request reuses it and gets the original send back; a deliberate
//     resend uses a new key. Unique when set.
//   - audience_key: "all" or "list:<id>", so the same newsletter sent to a
//     different list isn't mistaken for a duplicate.
//   - list_id / list_name: which list the send went to, for history.
//   - content_hash: SHA-256 of subject and rendered body.
//   - active_lock: "sending" while a send runs, cleared when it finishes. A
//     partial unique index on it means the database itself refuses a second
//     concurrent send.
//
// Sends recorded before this migration keep blank values. The send handler
// treats a blank audience as unknown and matches it conservatively, so old
// records still block an accidental repeat.
//
// newsletter_lists.is_public marks lists a visitor may join from the public
// signup form. The default list is public; other lists stay private until the
// owner opts them in.
//
// subscribers.pending_list_id holds the list a signup asked for until the
// address owner confirms.
func init() {
	m.Register(func(app core.App) error {
		if sends, err := app.FindCollectionByNameOrId("newsletter_sends"); err == nil {
			if sends.Fields.GetByName("request_key") == nil {
				sends.Fields.Add(&core.TextField{Name: "request_key", Max: 100})
			}
			if sends.Fields.GetByName("audience_key") == nil {
				sends.Fields.Add(&core.TextField{Name: "audience_key", Max: 100})
			}
			if sends.Fields.GetByName("list_id") == nil {
				sends.Fields.Add(&core.TextField{Name: "list_id", Max: 50})
			}
			if sends.Fields.GetByName("list_name") == nil {
				sends.Fields.Add(&core.TextField{Name: "list_name", Max: 200})
			}
			if sends.Fields.GetByName("content_hash") == nil {
				sends.Fields.Add(&core.TextField{Name: "content_hash", Max: 64})
			}
			if sends.Fields.GetByName("active_lock") == nil {
				sends.Fields.Add(&core.TextField{Name: "active_lock", Max: 20})
			}
			sends.AddIndex("idx_newsletter_sends_request_key", true, "request_key", "request_key != ''")
			sends.AddIndex("idx_newsletter_sends_active_lock", true, "active_lock", "active_lock != ''")
			sends.AddIndex("idx_newsletter_sends_content", false, "content_hash, audience_key", "")
			if err := app.Save(sends); err != nil {
				return err
			}
		}

		if lists, err := app.FindCollectionByNameOrId("newsletter_lists"); err == nil {
			if lists.Fields.GetByName("is_public") == nil {
				lists.Fields.Add(&core.BoolField{Name: "is_public"})
				if err := app.Save(lists); err != nil {
					return err
				}
				if _, err := app.DB().NewQuery(
					"UPDATE newsletter_lists SET is_public = 1 WHERE is_default = 1",
				).Execute(); err != nil {
					return err
				}
			}
		}

		if subscribers, err := app.FindCollectionByNameOrId("subscribers"); err == nil {
			if subscribers.Fields.GetByName("pending_list_id") == nil {
				subscribers.Fields.Add(&core.TextField{Name: "pending_list_id", Max: 50, Hidden: true})
				if err := app.Save(subscribers); err != nil {
					return err
				}
			}
		}
		return nil
	}, func(app core.App) error {
		if sends, err := app.FindCollectionByNameOrId("newsletter_sends"); err == nil {
			sends.RemoveIndex("idx_newsletter_sends_request_key")
			sends.RemoveIndex("idx_newsletter_sends_active_lock")
			sends.RemoveIndex("idx_newsletter_sends_content")
			for _, name := range []string{"request_key", "audience_key", "list_id", "list_name", "content_hash", "active_lock"} {
				if f := sends.Fields.GetByName(name); f != nil {
					sends.Fields.RemoveById(f.GetId())
				}
			}
			if err := app.Save(sends); err != nil {
				return err
			}
		}
		if lists, err := app.FindCollectionByNameOrId("newsletter_lists"); err == nil {
			if f := lists.Fields.GetByName("is_public"); f != nil {
				lists.Fields.RemoveById(f.GetId())
				if err := app.Save(lists); err != nil {
					return err
				}
			}
		}
		if subscribers, err := app.FindCollectionByNameOrId("subscribers"); err == nil {
			if f := subscribers.Fields.GetByName("pending_list_id"); f != nil {
				subscribers.Fields.RemoveById(f.GetId())
				return app.Save(subscribers)
			}
		}
		return nil
	})
}
