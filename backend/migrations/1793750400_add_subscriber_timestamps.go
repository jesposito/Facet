package migrations

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Subscribers were created without created and updated fields, so the sidebar's
// new-subscriber count was rejected with a 400 and the admin list had no date
// to show. Subscribers added from now on record both. Rows that already exist
// keep empty values: when they signed up is unknown and is not invented.
func init() {
	m.Register(func(app core.App) error {
		subscribers, err := app.FindCollectionByNameOrId("subscribers")
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		addNewsletterTimestamps(subscribers)
		// The sidebar filters subscribers by created each time it mounts.
		subscribers.AddIndex("idx_subscribers_created", false, "created", "")
		return app.Save(subscribers)
	}, func(app core.App) error {
		// created and updated stay, along with their index. Nothing records
		// whether this migration added them or an install already had them,
		// and dropping them would lose real timestamps.
		return nil
	})
}
