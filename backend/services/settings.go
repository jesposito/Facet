package services

import (
	"errors"
	"log/slog"

	"github.com/pocketbase/pocketbase/core"
)

// SiteSettings holds public site configuration flags.
type SiteSettings struct {
	HomepageEnabled    bool
	LandingPageMessage string
	CustomCSS          string
	Record             *core.Record
}

// LoadSiteSettings returns the current site settings, ensuring a default record exists.
// Falls back to sensible defaults if the collection is missing.
func LoadSiteSettings(app core.App) (*SiteSettings, error) {
	collection, err := app.FindCollectionByNameOrId("site_settings")
	if err != nil {
		return &SiteSettings{
			HomepageEnabled:    true,
			LandingPageMessage: "",
			Record:             nil,
		}, nil
	}

	records, err := app.FindRecordsByFilter(
		collection.Name,
		"",
		"",
		1,
		0,
		nil,
	)
	if err != nil {
		return nil, err
	}

	var record *core.Record
	if len(records) > 0 {
		record = records[0]
	} else {
		// Seed default record if none exists
		record = core.NewRecord(collection)
		record.Set("homepage_enabled", true)
		record.Set("landing_page_message", "This profile is being set up.")
		if err := app.Save(record); err != nil {
			return nil, err
		}
	}

	return &SiteSettings{
		HomepageEnabled:    record.GetBool("homepage_enabled"),
		LandingPageMessage: record.GetString("landing_page_message"),
		CustomCSS:          record.GetString("custom_css"),
		Record:             record,
	}, nil
}

// UpdateSiteSettings updates the settings record with sanitized values.
func UpdateSiteSettings(app core.App, updates map[string]any, logger *slog.Logger) (*SiteSettings, error) {
	settings, err := LoadSiteSettings(app)
	if err != nil {
		return nil, err
	}

	if settings.Record == nil {
		return nil, errors.New("site settings record missing")
	}

	if enabled, ok := updates["homepage_enabled"].(bool); ok {
		settings.Record.Set("homepage_enabled", enabled)
	}
	if msg, ok := updates["landing_page_message"].(string); ok {
		settings.Record.Set("landing_page_message", msg)
	}
	if css, ok := updates["custom_css"].(string); ok {
		if settings.Record.Collection().Fields.GetByName("custom_css") != nil {
			settings.Record.Set("custom_css", css)
		} else if logger != nil {
			logger.Warn("custom_css field missing on site_settings, skipping update")
		}
	}

	if err := app.Save(settings.Record); err != nil {
		return nil, err
	}

	// Reload to ensure stored values are returned
	return LoadSiteSettings(app)
}
