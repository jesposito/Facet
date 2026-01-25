package services

import (
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

func parseSiteNav(record *core.Record) SiteNavConfig {
	raw := record.GetString("site_nav")
	if raw == "" {
		return DefaultSiteNavConfig()
	}
	var config SiteNavConfig
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return DefaultSiteNavConfig()
	}
	return config
}

func parseHomepageSections(record *core.Record) []HomepageSectionConfig {
	raw := record.GetString("homepage_sections")
	if raw == "" {
		return nil
	}
	var sections []HomepageSectionConfig
	if err := json.Unmarshal([]byte(raw), &sections); err != nil {
		return nil
	}
	return sections
}

const maxNavItems = 20
const maxLabelLength = 50

func sanitizeSiteNavConfig(input map[string]any) map[string]any {
	result := make(map[string]any)

	result["enabled"] = parseBoolValue(input["enabled"])

	if homeLabel, ok := input["home_label"].(string); ok {
		if len(homeLabel) > maxLabelLength {
			homeLabel = homeLabel[:maxLabelLength]
		}
		result["home_label"] = strings.TrimSpace(homeLabel)
	} else {
		result["home_label"] = "Home"
	}

	var sanitizedItems []map[string]any
	if items, ok := input["items"].([]any); ok {
		seen := make(map[string]bool)
		for i, raw := range items {
			if i >= maxNavItems {
				break
			}
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}

			viewID, _ := item["view_id"].(string)
			if viewID == "" || viewID == "home" || seen[viewID] {
				continue
			}
			seen[viewID] = true

			label, _ := item["label"].(string)
			if len(label) > maxLabelLength {
				label = label[:maxLabelLength]
			}

			visible := parseBoolValue(item["visible"])

			sanitizedItems = append(sanitizedItems, map[string]any{
				"view_id": viewID,
				"label":   strings.TrimSpace(label),
				"visible": visible,
			})
		}
	}
	result["items"] = sanitizedItems

	return result
}

func parseBoolValue(v any) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val == "true" || val == "1"
	case float64:
		return val != 0
	case int:
		return val != 0
	default:
		return false
	}
}

// SiteNavItem represents a single item in the site navigation.
type SiteNavItem struct {
	ViewID  string `json:"view_id"` // "home" for homepage, or actual view ID
	Label   string `json:"label"`   // Custom display label
	Visible bool   `json:"visible"` // Show/hide toggle
}

// SiteNavConfig holds the site navigation configuration.
type SiteNavConfig struct {
	Enabled   bool          `json:"enabled"`    // Whether nav is displayed
	HomeLabel string        `json:"home_label"` // Label for home link (default: "Home")
	Items     []SiteNavItem `json:"items"`      // Ordered list of nav items
}

// DefaultSiteNavConfig returns sensible defaults for site navigation.
func DefaultSiteNavConfig() SiteNavConfig {
	return SiteNavConfig{
		Enabled:   false,
		HomeLabel: "Home",
		Items:     []SiteNavItem{},
	}
}

// HomepageSectionConfig represents a single section in the homepage configuration.
type HomepageSectionConfig struct {
	Section string   `json:"section"`
	Enabled bool     `json:"enabled"`
	Items   []string `json:"items,omitempty"`
	Layout  string   `json:"layout,omitempty"`
	Width   string   `json:"width,omitempty"`
}

// SiteSettings holds public site configuration flags.
type SiteSettings struct {
	HomepageEnabled    bool
	LandingPageMessage string
	CustomCSS          string
	GAMeasurementID    string
	HideLoginButton    bool
	HideDemoToggle     bool
	SiteNav            SiteNavConfig
	HomepageSections   []HomepageSectionConfig
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
			SiteNav:            DefaultSiteNavConfig(),
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
		GAMeasurementID:    record.GetString("ga_measurement_id"),
		HideLoginButton:    record.GetBool("hide_login_button"),
		HideDemoToggle:     record.GetBool("hide_demo_toggle"),
		SiteNav:            parseSiteNav(record),
		HomepageSections:   parseHomepageSections(record),
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
	if nav, ok := updates["site_nav"].(map[string]any); ok {
		if settings.Record.Collection().Fields.GetByName("site_nav") != nil {
			sanitized := sanitizeSiteNavConfig(nav)
			navJSON, err := json.Marshal(sanitized)
			if err == nil {
				settings.Record.Set("site_nav", string(navJSON))
			} else if logger != nil {
				logger.Warn("failed to marshal site_nav, skipping update", "error", err)
			}
		} else if logger != nil {
			logger.Warn("site_nav field missing on site_settings, skipping update")
		}
	}
	if sections, ok := updates["homepage_sections"].([]any); ok {
		if settings.Record.Collection().Fields.GetByName("homepage_sections") != nil {
			sectionsJSON, err := json.Marshal(sections)
			if err == nil {
				settings.Record.Set("homepage_sections", string(sectionsJSON))
			} else if logger != nil {
				logger.Warn("failed to marshal homepage_sections, skipping update", "error", err)
			}
		} else if logger != nil {
			logger.Warn("homepage_sections field missing on site_settings, skipping update")
		}
	}

	if err := app.Save(settings.Record); err != nil {
		return nil, err
	}

	// Reload to ensure stored values are returned
	return LoadSiteSettings(app)
}

type BuiltNavItem struct {
	Slug   string `json:"slug"`
	Label  string `json:"label"`
	URL    string `json:"url"`
	IsHome bool   `json:"is_home"`
}

type ViewInfo struct {
	ID   string
	Name string
	Slug string
}

func BuildNavItems(config SiteNavConfig, publicViews []ViewInfo) []BuiltNavItem {
	result := []BuiltNavItem{}

	if !config.Enabled {
		return result
	}

	viewByID := make(map[string]ViewInfo)
	for _, v := range publicViews {
		viewByID[v.ID] = v
	}

	homeLabel := config.HomeLabel
	if homeLabel == "" {
		homeLabel = "Home"
	}
	result = append(result, BuiltNavItem{
		Slug:   "",
		Label:  homeLabel,
		URL:    "/",
		IsHome: true,
	})

	seen := make(map[string]bool)
	for _, item := range config.Items {
		if !item.Visible || item.ViewID == "home" || seen[item.ViewID] {
			continue
		}

		view, exists := viewByID[item.ViewID]
		if !exists {
			continue
		}

		seen[item.ViewID] = true

		label := item.Label
		if label == "" {
			label = view.Name
		}
		if len(label) > 50 {
			label = label[:50]
		}

		result = append(result, BuiltNavItem{
			Slug:   view.Slug,
			Label:  label,
			URL:    "/" + view.Slug,
			IsHome: false,
		})
	}

	return result
}
