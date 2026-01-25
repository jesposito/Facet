package services

import (
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/pocketbase/pocketbase/core"
)

// HomepageCustomContentItem represents a custom content block on the homepage
type HomepageCustomContentItem struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

// SiteSettings holds public site configuration flags.
type SiteSettings struct {
	HomepageEnabled       bool
	LandingPageMessage    string
	CustomCSS             string
	GAMeasurementID       string
	HideLoginButton       bool
	HideDemoToggle        bool
	HomepageCustomContent []HomepageCustomContentItem
	HomepageSectionOrder  []string
	Record                *core.Record
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

	// Parse homepage custom content JSON
	var homepageCustomContent []HomepageCustomContentItem
	if rawJSON := record.GetString("homepage_custom_content"); rawJSON != "" {
		_ = json.Unmarshal([]byte(rawJSON), &homepageCustomContent)
	}

	// Parse homepage section order JSON
	var homepageSectionOrder []string
	if rawJSON := record.GetString("homepage_section_order"); rawJSON != "" {
		_ = json.Unmarshal([]byte(rawJSON), &homepageSectionOrder)
	}

	return &SiteSettings{
		HomepageEnabled:       record.GetBool("homepage_enabled"),
		LandingPageMessage:    record.GetString("landing_page_message"),
		CustomCSS:             record.GetString("custom_css"),
		GAMeasurementID:       record.GetString("ga_measurement_id"),
		HideLoginButton:       record.GetBool("hide_login_button"),
		HideDemoToggle:        record.GetBool("hide_demo_toggle"),
		HomepageCustomContent: homepageCustomContent,
		HomepageSectionOrder:  homepageSectionOrder,
		Record:                record,
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
	if ga, ok := updates["ga_measurement_id"].(string); ok {
		if settings.Record.Collection().Fields.GetByName("ga_measurement_id") != nil {
			settings.Record.Set("ga_measurement_id", ga)
		} else if logger != nil {
			logger.Warn("ga_measurement_id field missing on site_settings, skipping update")
		}
	}
	if hide, ok := updates["hide_login_button"].(bool); ok {
		if settings.Record.Collection().Fields.GetByName("hide_login_button") != nil {
			settings.Record.Set("hide_login_button", hide)
		} else if logger != nil {
			logger.Warn("hide_login_button field missing on site_settings, skipping update")
		}
	}
	if hide, ok := updates["hide_demo_toggle"].(bool); ok {
		if settings.Record.Collection().Fields.GetByName("hide_demo_toggle") != nil {
			settings.Record.Set("hide_demo_toggle", hide)
		} else if logger != nil {
			logger.Warn("hide_demo_toggle field missing on site_settings, skipping update")
		}
	}
	if customContent, ok := updates["homepage_custom_content"]; ok {
		if settings.Record.Collection().Fields.GetByName("homepage_custom_content") != nil {
			settings.Record.Set("homepage_custom_content", customContent)
		} else if logger != nil {
			logger.Warn("homepage_custom_content field missing on site_settings, skipping update")
		}
	}
	if sectionOrder, ok := updates["homepage_section_order"]; ok {
		if settings.Record.Collection().Fields.GetByName("homepage_section_order") != nil {
			settings.Record.Set("homepage_section_order", sectionOrder)
		} else if logger != nil {
			logger.Warn("homepage_section_order field missing on site_settings, skipping update")
		}
	}

	if err := app.Save(settings.Record); err != nil {
		return nil, err
	}

	// Reload to ensure stored values are returned
	return LoadSiteSettings(app)
}
