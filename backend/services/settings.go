package services

import (
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/pocketbase/pocketbase/core"
)

// StripeConfig holds decrypted Stripe credentials and metadata about their source.
type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
	// Source indicates where the keys came from: "database", "environment", or "none"
	Source string
}

// GetStripeConfig returns the active Stripe configuration.
// It checks the database first (decrypted via crypto), then falls back to env vars.
// Pass nil for crypto to skip DB lookup (env-only path).
func GetStripeConfig(app core.App, crypto *CryptoService) *StripeConfig {
	if app != nil && crypto != nil {
		records, err := app.FindRecordsByFilter("site_settings", "", "", 1, 0, nil)
		if err == nil && len(records) > 0 {
			record := records[0]
			encKey := record.GetString("stripe_secret_key")
			encWebhook := record.GetString("stripe_webhook_secret")

			if encKey != "" || encWebhook != "" {
				secretKey := ""
				webhookSecret := ""

				if encKey != "" {
					if decrypted, decErr := crypto.Decrypt(encKey); decErr == nil {
						secretKey = decrypted
					}
				}
				if encWebhook != "" {
					if decrypted, decErr := crypto.Decrypt(encWebhook); decErr == nil {
						webhookSecret = decrypted
					}
				}

				if secretKey != "" || webhookSecret != "" {
					return &StripeConfig{
						SecretKey:     secretKey,
						WebhookSecret: webhookSecret,
						Source:        "database",
					}
				}
			}
		}
	}

	return nil
}

// GetStripeSecretKey returns the Stripe secret key from the database (decrypted).
// Returns empty string if not set in DB.
func GetStripeSecretKey(app core.App, crypto *CryptoService) string {
	cfg := GetStripeConfig(app, crypto)
	if cfg != nil {
		return cfg.SecretKey
	}
	return ""
}

// GetStripeWebhookSecret returns the Stripe webhook secret from the database (decrypted).
// Returns empty string if not set in DB.
func GetStripeWebhookSecret(app core.App, crypto *CryptoService) string {
	cfg := GetStripeConfig(app, crypto)
	if cfg != nil {
		return cfg.WebhookSecret
	}
	return ""
}

// SaveStripeConfig encrypts and persists Stripe credentials to site_settings.
// Pass empty string for a key to leave the existing value unchanged.
func SaveStripeConfig(app core.App, crypto *CryptoService, secretKey, webhookSecret string) error {
	records, err := app.FindRecordsByFilter("site_settings", "", "", 1, 0, nil)
	if err != nil {
		return errors.New("failed to load site settings")
	}
	if len(records) == 0 {
		return errors.New("site settings record not found")
	}

	record := records[0]

	if secretKey != "" {
		encrypted, encErr := crypto.Encrypt(secretKey)
		if encErr != nil {
			return encErr
		}
		record.Set("stripe_secret_key", encrypted)
	}

	if webhookSecret != "" {
		encrypted, encErr := crypto.Encrypt(webhookSecret)
		if encErr != nil {
			return encErr
		}
		record.Set("stripe_webhook_secret", encrypted)
	}

	return app.Save(record)
}

// ClearStripeConfig removes stored Stripe credentials from site_settings,
// reverting to environment variable fallback.
func ClearStripeConfig(app core.App) error {
	records, err := app.FindRecordsByFilter("site_settings", "", "", 1, 0, nil)
	if err != nil {
		return errors.New("failed to load site settings")
	}
	if len(records) == 0 {
		return errors.New("site settings record not found")
	}

	record := records[0]
	record.Set("stripe_secret_key", "")
	record.Set("stripe_webhook_secret", "")

	return app.Save(record)
}

// HomepageCustomContentItem represents a custom content block on the homepage
type HomepageCustomContentItem struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

// HomepageSectionConfig represents per-section settings for the homepage
// Mirrors the structure used in views for consistency
type HomepageSectionConfig struct {
	Enabled       bool           `json:"enabled"`
	Items         []string       `json:"items,omitempty"`         // Selected item IDs (empty = all)
	Layout        string         `json:"layout,omitempty"`        // Section layout
	Width         string         `json:"width,omitempty"`         // Section width
	CategoryOrder []string       `json:"categoryOrder,omitempty"` // For skills: custom category order
	ItemConfig    map[string]any `json:"itemConfig,omitempty"`    // Per-item overrides
}

// SiteNavItem represents a navigation button configuration
type SiteNavItem struct {
	ViewID  string `json:"viewId"`  // ID of the view/facet
	Enabled bool   `json:"enabled"` // Whether to show this nav button
	Label   string `json:"label"`   // Custom button label (falls back to view name)
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
	HomepageSections      map[string]HomepageSectionConfig
	SiteNavEnabled        bool
	SiteNavMode           string
	SiteNavPosition       string
	SiteNavItems          []SiteNavItem
	ShowAvatar            bool
	SkillsCategoryOrder   []string
	SiteCtaEnabled        bool
	Favicon               string
	DefaultLocale         string
	DefaultThemeMode      string
	HomepageViewCount     int
	HomepageLastViewedAt  string
	EnabledFeatures       map[string]bool
	Record                *core.Record
}

// loadDefaultThemeMode returns the default_theme_mode field, falling back to
// "system" when the field isn't present (pre-migration records) or is empty.
// Valid values are: "system", "light", "dark".
func loadDefaultThemeMode(record *core.Record) string {
	if record == nil {
		return "system"
	}
	if record.Collection() != nil && record.Collection().Fields.GetByName("default_theme_mode") == nil {
		return "system"
	}
	mode := record.GetString("default_theme_mode")
	if mode == "" {
		return "system"
	}
	return mode
}

// loadShowAvatar returns the show_avatar field as bool, defaulting to true when
// the field isn't present (pre-migration records) so existing sites keep their avatars.
func loadShowAvatar(record *core.Record) bool {
	if record == nil {
		return true
	}
	if record.Collection() != nil && record.Collection().Fields.GetByName("show_avatar") == nil {
		return true
	}
	// Field exists; honor the saved value. Default DB value is true (set by migration).
	return record.GetBool("show_avatar")
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

	// Parse homepage sections JSON (per-section item selections)
	var homepageSections map[string]HomepageSectionConfig
	if rawJSON := record.GetString("homepage_sections"); rawJSON != "" {
		_ = json.Unmarshal([]byte(rawJSON), &homepageSections)
	}

	// Parse site navigation items JSON
	var siteNavItems []SiteNavItem
	if rawJSON := record.GetString("site_nav_items"); rawJSON != "" {
		_ = json.Unmarshal([]byte(rawJSON), &siteNavItems)
	}

	// Parse skills category order JSON
	var skillsCategoryOrder []string
	if rawJSON := record.GetString("skills_category_order"); rawJSON != "" {
		_ = json.Unmarshal([]byte(rawJSON), &skillsCategoryOrder)
	}

	// Parse enabled features JSON (per-feature overrides for self-hosted instances)
	var enabledFeatures map[string]bool
	if rawJSON := record.GetString("enabled_features"); rawJSON != "" {
		_ = json.Unmarshal([]byte(rawJSON), &enabledFeatures)
	}

	// For site_cta_enabled, default to true if field doesn't exist or is not explicitly set
	// This preserves existing behavior where CTA shows if URL is configured
	siteCtaEnabled := true
	if record.Collection().Fields.GetByName("site_cta_enabled") != nil {
		// Field exists - use its value (but default to true if never set)
		// GetBool returns false for unset fields, so we check if it was explicitly set to false
		if record.Get("site_cta_enabled") != nil {
			siteCtaEnabled = record.GetBool("site_cta_enabled")
		}
	}

	// Build favicon URL if set - use /api/favicon endpoint which serves with no-cache headers
	var faviconURL string
	if faviconFile := record.GetString("favicon"); faviconFile != "" {
		faviconURL = "/api/favicon"
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
		HomepageSections:      homepageSections,
		SiteNavEnabled:        record.GetBool("site_nav_enabled"),
		SiteNavMode:           record.GetString("site_nav_mode"),
		SiteNavPosition:       record.GetString("site_nav_position"),
		SiteNavItems:          siteNavItems,
		ShowAvatar:            loadShowAvatar(record),
		SkillsCategoryOrder:   skillsCategoryOrder,
		SiteCtaEnabled:        siteCtaEnabled,
		Favicon:               faviconURL,
		DefaultLocale:         record.GetString("default_locale"),
		DefaultThemeMode:      loadDefaultThemeMode(record),
		HomepageViewCount:     record.GetInt("homepage_view_count"),
		HomepageLastViewedAt:  record.GetString("homepage_last_viewed_at"),
		EnabledFeatures:       enabledFeatures,
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
	if sections, ok := updates["homepage_sections"]; ok {
		if settings.Record.Collection().Fields.GetByName("homepage_sections") != nil {
			settings.Record.Set("homepage_sections", sections)
		} else if logger != nil {
			logger.Warn("homepage_sections field missing on site_settings, skipping update")
		}
	}
	if enabled, ok := updates["site_nav_enabled"].(bool); ok {
		if settings.Record.Collection().Fields.GetByName("site_nav_enabled") != nil {
			settings.Record.Set("site_nav_enabled", enabled)
		} else if logger != nil {
			logger.Warn("site_nav_enabled field missing on site_settings, skipping update")
		}
	}
	if mode, ok := updates["site_nav_mode"].(string); ok {
		if settings.Record.Collection().Fields.GetByName("site_nav_mode") != nil {
			settings.Record.Set("site_nav_mode", mode)
		} else if logger != nil {
			logger.Warn("site_nav_mode field missing on site_settings, skipping update")
		}
	}
	if pos, ok := updates["site_nav_position"].(string); ok {
		if settings.Record.Collection().Fields.GetByName("site_nav_position") != nil {
			settings.Record.Set("site_nav_position", pos)
		} else if logger != nil {
			logger.Warn("site_nav_position field missing on site_settings, skipping update")
		}
	}
	if show, ok := updates["show_avatar"].(bool); ok {
		if settings.Record.Collection().Fields.GetByName("show_avatar") != nil {
			settings.Record.Set("show_avatar", show)
		} else if logger != nil {
			logger.Warn("show_avatar field missing on site_settings, skipping update")
		}
	}
	if navItems, ok := updates["site_nav_items"]; ok {
		if settings.Record.Collection().Fields.GetByName("site_nav_items") != nil {
			settings.Record.Set("site_nav_items", navItems)
		} else if logger != nil {
			logger.Warn("site_nav_items field missing on site_settings, skipping update")
		}
	}
	if skillsCatOrder, ok := updates["skills_category_order"]; ok {
		if settings.Record.Collection().Fields.GetByName("skills_category_order") != nil {
			settings.Record.Set("skills_category_order", skillsCatOrder)
		} else if logger != nil {
			logger.Warn("skills_category_order field missing on site_settings, skipping update")
		}
	}
	if enabled, ok := updates["site_cta_enabled"].(bool); ok {
		if settings.Record.Collection().Fields.GetByName("site_cta_enabled") != nil {
			settings.Record.Set("site_cta_enabled", enabled)
		} else if logger != nil {
			logger.Warn("site_cta_enabled field missing on site_settings, skipping update")
		}
	}
	if locale, ok := updates["default_locale"].(string); ok {
		if settings.Record.Collection().Fields.GetByName("default_locale") != nil {
			settings.Record.Set("default_locale", locale)
		} else if logger != nil {
			logger.Warn("default_locale field missing on site_settings, skipping update")
		}
	}
	if mode, ok := updates["default_theme_mode"].(string); ok {
		if settings.Record.Collection().Fields.GetByName("default_theme_mode") != nil {
			settings.Record.Set("default_theme_mode", mode)
		} else if logger != nil {
			logger.Warn("default_theme_mode field missing on site_settings, skipping update")
		}
	}
	if features, ok := updates["enabled_features"]; ok {
		if settings.Record.Collection().Fields.GetByName("enabled_features") != nil {
			settings.Record.Set("enabled_features", features)
		}
	}

	if err := app.Save(settings.Record); err != nil {
		return nil, err
	}

	// Reload to ensure stored values are returned
	return LoadSiteSettings(app)
}
