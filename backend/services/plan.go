package services

import (
	"os"
	"strconv"
)

// PlanTier represents the subscription plan tier
type PlanTier string

const (
	PlanSelfHosted PlanTier = "self-hosted"
)

// PlanFeatures represents which features are available for the current plan
type PlanFeatures struct {
	BasicAnalytics bool `json:"basic_analytics"`
	Analytics      bool `json:"analytics"`
	BadgeForced    bool `json:"badge_forced"`
	Courses        bool `json:"courses"`
	API            bool `json:"api"`
	CustomDomain   bool `json:"custom_domain"`
	Newsletter     bool `json:"newsletter"`
	Discussions    bool `json:"discussions"`
	Pricing        bool `json:"pricing"`
}

// PlanConfig holds the managed mode configuration derived from environment variables.
// Self-hosted instances get all features unlocked by default.
type PlanConfig struct {
	Managed  bool         `json:"managed"`
	Plan     PlanTier     `json:"plan"`
	Features PlanFeatures `json:"features"`
}

// LoadPlanConfig reads plan configuration from environment variables.
// When FACET_MANAGED is not set or false, returns a self-hosted config with all features unlocked.
// Self-hosted is the expected mode for this upstream repo; managed support is minimal for compatibility.
//
// Courses is OFF by default self-hosted because the public /courses/[slug] route is
// not implemented in this repo (cloud has it). Operators who want admin-only course
// authoring (e.g., headless integrations) can opt in with FACET_FEATURE_COURSES=true.
func LoadPlanConfig() *PlanConfig {
	managed := envBool("FACET_MANAGED", false)

	if !managed {
		return &PlanConfig{
			Managed: false,
			Plan:    PlanSelfHosted,
			Features: PlanFeatures{
				BasicAnalytics: true,
				Analytics:      true,
				BadgeForced:    false,
				Courses:        envBool("FACET_FEATURE_COURSES", false),
				API:            true,
				CustomDomain:   true,
				Newsletter:     true,
				Discussions:    true,
				Pricing:        true,
			},
		}
	}

	// Managed mode: minimal support for compatibility.
	// In practice, self-hosted instances never set FACET_MANAGED=true.
	return &PlanConfig{
		Managed: true,
		Plan:    PlanSelfHosted,
		Features: PlanFeatures{
			BasicAnalytics: true,
			Analytics:      true,
			BadgeForced:    false,
			Courses:        true,
			API:            true,
			Newsletter:     true,
			Discussions:    true,
			Pricing:        true,
		},
	}
}

// IsManaged returns whether this instance is running in managed (cloud) mode
func (c *PlanConfig) IsManaged() bool {
	return c.Managed
}

// HasFeature checks if a specific feature is available.
// Self-hosted instances always return true for all features.
func (c *PlanConfig) HasFeature(feature string) bool {
	switch feature {
	case "basic_analytics":
		return c.Features.BasicAnalytics
	case "analytics":
		return c.Features.Analytics
	case "courses":
		return c.Features.Courses
	case "api":
		return c.Features.API
	case "custom_domain":
		return c.Features.CustomDomain
	case "newsletter":
		return c.Features.Newsletter
	case "discussions":
		return c.Features.Discussions
	case "pricing":
		return c.Features.Pricing
	default:
		// Self-hosted gets everything; managed defaults to false for unknown features
		return !c.Managed
	}
}

// SignupAllowed returns whether public signup/registration is allowed.
// Self-hosted instances always allow signup.
func (c *PlanConfig) SignupAllowed() bool {
	return !c.Managed
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
