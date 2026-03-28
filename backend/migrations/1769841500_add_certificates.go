package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds course_certificates collection with unique constraints
func init() {
	m.Register(func(app core.App) error {
		// Check if collection already exists
		if _, err := app.FindCollectionByNameOrId("course_certificates"); err == nil {
			return nil // Already exists
		}

		// Find courses collection for relation
		courses, err := app.FindCollectionByNameOrId("courses")
		if err != nil {
			return err
		}

		// Create course_certificates collection
		certificates := core.NewBaseCollection("course_certificates")
		publicRule := ""
		certificates.ListRule = &publicRule
		certificates.ViewRule = &publicRule
		certificates.CreateRule = nil
		certificates.UpdateRule = nil
		certificates.DeleteRule = nil

		certificates.Fields.Add(&core.RelationField{
			Name:         "course",
			MaxSelect:    1,
			Required:     true,
			CollectionId: courses.Id,
		})
		certificates.Fields.Add(&core.EmailField{
			Name:     "buyer_email",
			Required: true,
		})
		certificates.Fields.Add(&core.TextField{
			Name:     "certificate_id",
			Required: true,
			Max:      64,
		})
		certificates.Fields.Add(&core.AutodateField{
			Name:     "issued_at",
			OnCreate: true,
			OnUpdate: false,
		})
		certificates.Fields.Add(&core.TextField{
			Name: "course_title",
			Max:  200,
		})
		certificates.Fields.Add(&core.TextField{
			Name: "instructor_name",
			Max:  200,
		})
		certificates.Fields.Add(&core.TextField{
			Name: "student_name",
			Max:  200,
		})

		certificates.Indexes = []string{
			"CREATE UNIQUE INDEX idx_cert_id ON course_certificates(certificate_id)",
			"CREATE UNIQUE INDEX idx_cert_course_email ON course_certificates(course, buyer_email)",
		}

		return app.Save(certificates)
	}, func(app core.App) error {
		// Rollback: delete the collection
		if collection, err := app.FindCollectionByNameOrId("course_certificates"); err == nil {
			return app.Delete(collection)
		}
		return nil
	})
}
