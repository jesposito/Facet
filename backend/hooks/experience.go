package hooks

import (
	"facet/services"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// NormalizeCompanyName returns a normalized group ID from a company name.
// It lowercases, trims whitespace, strips common suffixes and trailing punctuation.
func NormalizeCompanyName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	if s == "" {
		return ""
	}

	suffixes := []string{
		"s.a.", "inc.", "ltd.", "corp.", "co.", "gmbh", "ag", "sa", "inc", "llc",
		"ltd", "corp", "co", "plc",
	}
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, " "+suffix) {
			s = strings.TrimSuffix(s, " "+suffix)
			break
		}
	}

	s = strings.TrimRight(s, ".,;:!? ")
	s = strings.TrimSpace(s)

	return s
}

// RegisterExperienceHooks sets up hooks for the experience and education collections.
func RegisterExperienceHooks(app *pocketbase.PocketBase) {
	// Before experience is created, auto-populate company_group_id if empty
	app.OnRecordCreate("experience").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("company_group_id") == "" {
			company := e.Record.GetString("company")
			if company != "" {
				e.Record.Set("company_group_id", NormalizeCompanyName(company))
			}
		}
		return e.Next()
	})

	// Before experience is updated, auto-populate company_group_id if empty or company changed
	app.OnRecordUpdate("experience").BindFunc(func(e *core.RecordEvent) error {
		currentGroupID := e.Record.GetString("company_group_id")
		company := e.Record.GetString("company")
		originalCompany := e.Record.Original().GetString("company")

		if currentGroupID == "" || company != originalCompany {
			if company != "" {
				e.Record.Set("company_group_id", NormalizeCompanyName(company))
			}
		}
		return e.Next()
	})

	// After experience is created/updated, set the company logo display name to company name
	app.OnRecordAfterCreateSuccess("experience").BindFunc(func(e *core.RecordEvent) error {
		return setCompanyLogoDisplayName(app, e.Record)
	})

	app.OnRecordAfterUpdateSuccess("experience").BindFunc(func(e *core.RecordEvent) error {
		return setCompanyLogoDisplayName(app, e.Record)
	})

	// After education is created/updated, set the institution logo display name to institution name
	app.OnRecordAfterCreateSuccess("education").BindFunc(func(e *core.RecordEvent) error {
		return setInstitutionLogoDisplayName(app, e.Record)
	})

	app.OnRecordAfterUpdateSuccess("education").BindFunc(func(e *core.RecordEvent) error {
		return setInstitutionLogoDisplayName(app, e.Record)
	})
}

// setCompanyLogoDisplayName sets the display name of a company logo to the company name.
func setCompanyLogoDisplayName(app *pocketbase.PocketBase, record *core.Record) error {
	companyLogo := record.GetString("company_logo")
	if companyLogo == "" {
		return nil
	}

	companyName := record.GetString("company")
	if companyName == "" {
		return nil
	}

	collectionID := record.Collection().Id
	recordID := record.Id

	// Set the display name to company name
	if err := services.SetMediaDisplayName(app, collectionID, recordID, companyLogo, companyName); err != nil {
		app.Logger().Warn("failed to set company logo display name",
			"error", err,
			"record", recordID,
			"company", companyName)
		// Don't fail the request, just log
	}

	return nil
}

// setInstitutionLogoDisplayName sets the display name of an institution logo to the institution name.
func setInstitutionLogoDisplayName(app *pocketbase.PocketBase, record *core.Record) error {
	institutionLogo := record.GetString("institution_logo")
	if institutionLogo == "" {
		return nil
	}

	institutionName := record.GetString("institution")
	if institutionName == "" {
		return nil
	}

	collectionID := record.Collection().Id
	recordID := record.Id

	// Set the display name to institution name
	if err := services.SetMediaDisplayName(app, collectionID, recordID, institutionLogo, institutionName); err != nil {
		app.Logger().Warn("failed to set institution logo display name",
			"error", err,
			"record", recordID,
			"institution", institutionName)
		// Don't fail the request, just log
	}

	return nil
}
