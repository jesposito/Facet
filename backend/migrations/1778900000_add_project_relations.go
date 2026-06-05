package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds related_experience and related_education relations to projects so a
// project can link to one or more experience/education records for display on
// the public project detail page.
func init() {
	m.Register(func(app core.App) error {
		experience, err := app.FindCollectionByNameOrId("experience")
		if err != nil || experience == nil {
			// If experience doesn't exist (stale data), skip adding the relation gracefully.
			return nil
		}
		education, err := app.FindCollectionByNameOrId("education")
		if err != nil || education == nil {
			return nil
		}

		projects, err := app.FindCollectionByNameOrId("projects")
		if err != nil || projects == nil {
			return nil
		}

		if projects.Fields.GetByName("related_experience") == nil {
			projects.Fields.Add(&core.RelationField{
				Name:          "related_experience",
				CollectionId:  experience.Id,
				MaxSelect:     20,
				Required:      false,
				CascadeDelete: false,
			})
		}
		if projects.Fields.GetByName("related_education") == nil {
			projects.Fields.Add(&core.RelationField{
				Name:          "related_education",
				CollectionId:  education.Id,
				MaxSelect:     20,
				Required:      false,
				CascadeDelete: false,
			})
		}

		return app.Save(projects)
	}, func(app core.App) error {
		projects, err := app.FindCollectionByNameOrId("projects")
		if err != nil || projects == nil {
			return nil
		}
		projects.Fields.RemoveByName("related_experience")
		projects.Fields.RemoveByName("related_education")
		return app.Save(projects)
	})
}
