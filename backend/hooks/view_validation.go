package hooks

import (
	"fmt"
	"strings"

	"facet/services"

	validation "github.com/pocketbase/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/crypto/bcrypt"
)

func registerViewsValidation(app *pocketbase.PocketBase, crypto *services.CryptoService) {
	// Request hooks run ahead of the save, so a rejected Facet hashes nothing
	// and clears no default.
	app.OnRecordCreateRequest("views").BindFunc(validateViewPasswordRequest)
	app.OnRecordUpdateRequest("views").BindFunc(validateViewPasswordRequest)

	app.OnRecordCreate("views").BindFunc(func(e *core.RecordEvent) error {
		slug := e.Record.GetString("slug")

		if !isValidSlug(slug) {
			return fmt.Errorf("invalid or reserved slug: slugs cannot use reserved paths like 'admin', 'api', 's', 'v', etc")
		}

		password := e.Record.GetString("password")
		app.Logger().Info("OnRecordCreate views hook", "slug", slug, "visibility", e.Record.GetString("visibility"))
		if password != "" {
			hash, err := crypto.HashPassword(password)
			if err != nil {
				return fmt.Errorf("failed to hash password: %w", err)
			}
			e.Record.Set("password_hash", hash)
			e.Record.Set("password", "")
			app.Logger().Info("Password hashed successfully", "slug", slug)
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

func validateViewPasswordRequest(e *core.RecordRequestEvent) error {
	if err := validateViewPassword(e.Record); err != nil {
		return err
	}
	return e.Next()
}

// validateViewPassword requires a password Facet to have one visitors can
// enter: a new nonblank password, or the hash it already stores.
func validateViewPassword(record *core.Record) error {
	if record.GetString("visibility") != "password" {
		return nil
	}
	password := record.GetString("password")
	if strings.TrimSpace(password) != "" {
		if len([]byte(password)) > 72 {
			return apis.NewBadRequestError("Failed to save facet.", validation.Errors{
				"password": validation.NewError("validation_length", "Passwords must be no longer than 72 bytes."),
			})
		}
		return nil
	}
	storedHash := record.Original().GetString("password_hash")
	if password == "" && storedHash != "" && record.GetString("password_hash") == storedHash {
		if _, err := bcrypt.Cost([]byte(storedHash)); err == nil {
			return nil
		}
	}
	return apis.NewBadRequestError("Failed to save facet.", validation.Errors{
		"password": validation.NewError("validation_required", "A password is required for password-protected facets."),
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
