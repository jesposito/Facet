package hooks

import (
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const maxFeaturedPerKind = 3

type featuredConfig struct {
	collection string
	field      string
	draftField string
}

var featuredCollections = []featuredConfig{
	{collection: "posts", field: "featured", draftField: "is_draft"},
	{collection: "projects", field: "is_featured", draftField: "is_draft"},
	{collection: "testimonials", field: "featured"},
}

func RegisterFeaturedValidationHooks(app *pocketbase.PocketBase) {
	for _, cfg := range featuredCollections {
		cfg := cfg

		app.OnRecordCreate(cfg.collection).BindFunc(func(e *core.RecordEvent) error {
			if !e.Record.GetBool(cfg.field) {
				return e.Next()
			}
			if err := enforceFeaturedCap(app, cfg, ""); err != nil {
				return err
			}
			return e.Next()
		})

		app.OnRecordUpdate(cfg.collection).BindFunc(func(e *core.RecordEvent) error {
			if !e.Record.GetBool(cfg.field) {
				return e.Next()
			}
			if e.Record.Original() != nil && e.Record.Original().GetBool(cfg.field) {
				return e.Next()
			}
			if err := enforceFeaturedCap(app, cfg, e.Record.Id); err != nil {
				return err
			}
			return e.Next()
		})
	}
}

func enforceFeaturedCap(app core.App, cfg featuredConfig, excludeID string) error {
	filter := fmt.Sprintf("%s = true", cfg.field)
	params := dbx.Params{}
	if cfg.draftField != "" {
		filter += fmt.Sprintf(" AND %s = false", cfg.draftField)
	}
	if excludeID != "" {
		filter += " AND id != {:excludeId}"
		params["excludeId"] = excludeID
	}

	count, err := app.CountRecords(cfg.collection, dbx.NewExp(filter, params))
	if err != nil {
		app.Logger().Warn("featured cap count failed",
			"collection", cfg.collection,
			"error", err,
		)
		return nil
	}

	if int(count)+1 > maxFeaturedPerKind {
		msg := fmt.Sprintf(
			"Cannot feature more than %d items per content type. Unfeature an existing item first.",
			maxFeaturedPerKind,
		)
		return apis.NewBadRequestError(msg, nil)
	}
	return nil
}
