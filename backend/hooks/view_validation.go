package hooks

import (
	"fmt"

	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func registerViewsValidation(app *pocketbase.PocketBase, crypto *services.CryptoService) {
	app.OnRecordCreate("views").BindFunc(func(e *core.RecordEvent) error {
		slug := e.Record.GetString("slug")

		if !isValidSlug(slug) {
			return fmt.Errorf("invalid or reserved slug: slugs cannot use reserved paths like 'admin', 'api', 's', 'v', etc")
		}

		password := e.Record.GetString("password")
		app.Logger().Info("OnRecordCreate views hook", "slug", slug, "password_len", len(password), "visibility", e.Record.GetString("visibility"))
		if password != "" {
			hash, err := crypto.HashPassword(password)
			if err != nil {
				return fmt.Errorf("failed to hash password: %w", err)
			}
			e.Record.Set("password_hash", hash)
			e.Record.Set("password", "")
			app.Logger().Info("Password hashed successfully", "slug", slug, "hash_len", len(hash))
		} else if e.Record.GetString("visibility") == "password" {
			app.Logger().Warn("Password visibility set but no password provided", "slug", slug)
		}

		if e.Record.GetBool("is_default") {
			if err := clearOtherDefaults(app, ""); err != nil {
				return err
			}
		}

		return e.Next()
	})

	app.OnRecordUpdate("views").BindFunc(func(e *core.RecordEvent) error {
		slug := e.Record.GetString("slug")

		if !isValidSlug(slug) {
			return fmt.Errorf("invalid or reserved slug: slugs cannot use reserved paths like 'admin', 'api', 's', 'v', etc")
		}

		password := e.Record.GetString("password")
		if password != "" {
			hash, err := crypto.HashPassword(password)
			if err != nil {
				return fmt.Errorf("failed to hash password: %w", err)
			}
			e.Record.Set("password_hash", hash)
			e.Record.Set("password", "")
		}

		if e.Record.GetBool("is_default") {
			if err := clearOtherDefaults(app, e.Record.Id); err != nil {
				return err
			}
		}

		return e.Next()
	})
}

func clearOtherDefaults(app *pocketbase.PocketBase, excludeID string) error {
	filter := "is_default = true"
	if excludeID != "" {
		filter += " && id != {:id}"
	}

	records, err := app.FindRecordsByFilter(
		"views",
		filter,
		"",
		100,
		0,
		map[string]interface{}{"id": excludeID},
	)

	if err != nil {
		return err
	}

	for _, record := range records {
		record.Set("is_default", false)
		if err := app.Save(record); err != nil {
			return err
		}
	}

	return nil
}
