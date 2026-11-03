package migrations

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		lessons, err := app.FindCollectionByNameOrId("lessons")
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("find lessons: %w", err)
		}
		if err == nil && lessons.Fields.GetByName("comments_enabled") == nil {
			lessons.Fields.Add(&core.BoolField{Name: "comments_enabled"})
			if err := app.Save(lessons); err != nil {
				return fmt.Errorf("add lessons.comments_enabled: %w", err)
			}
		}

		repairs := []struct {
			collection string
			indexes    []string
		}{
			{
				collection: "access_logs",
				indexes: []string{
					"CREATE INDEX IF NOT EXISTS idx_access_logs_created ON access_logs(created)",
					"CREATE INDEX IF NOT EXISTS idx_access_logs_view ON access_logs(view_id)",
				},
			},
			{
				collection: "system_alerts",
				indexes: []string{
					"CREATE INDEX IF NOT EXISTS idx_system_alerts_created ON system_alerts(created)",
					"CREATE INDEX IF NOT EXISTS idx_system_alerts_ack ON system_alerts(acknowledged_at)",
				},
			},
		}

		for _, repair := range repairs {
			collection, err := app.FindCollectionByNameOrId(repair.collection)
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			if err != nil {
				return fmt.Errorf("find %s: %w", repair.collection, err)
			}

			// Existing rows keep an empty created value because their original
			// creation time is unknown. New rows receive the current time.
			if collection.Fields.GetByName("created") == nil {
				collection.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
				if err := app.Save(collection); err != nil {
					return fmt.Errorf("add %s.created: %w", repair.collection, err)
				}
			}

			for _, statement := range repair.indexes {
				if _, err := app.DB().NewQuery(statement).Execute(); err != nil {
					return fmt.Errorf("create %s index: %w", repair.collection, err)
				}
			}
		}

		return nil
	}, nil)
}
