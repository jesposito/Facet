package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Find views collection to get its ID for the relation
		viewsCollection, err := app.FindCollectionByNameOrId("views")
		if err != nil {
			return err
		}

		// Find ai_providers collection to get its ID for the relation
		aiProvidersCollection, err := app.FindCollectionByNameOrId("ai_providers")
		if err != nil {
			return err
		}

		// Create view_exports collection for AI-generated resumes
		collection := core.NewBaseCollection("view_exports")

		// Relation to the view this export belongs to
		collection.Fields.Add(&core.RelationField{
			Name:         "view",
			CollectionId: viewsCollection.Id,
			Required:     true,
			MaxSelect:    1,
		})

		// Export format
		collection.Fields.Add(&core.SelectField{
			Name:      "format",
			Values:    []string{"pdf", "docx"},
			Required:  true,
			MaxSelect: 1,
		})

		// The generated file
		collection.Fields.Add(&core.FileField{
			Name:      "file",
			MaxSize:   10485760, // 10MB
			MaxSelect: 1,
			MimeTypes: []string{
				"application/pdf",
				"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			},
		})

		// Optional relation to AI provider used
		collection.Fields.Add(&core.RelationField{
			Name:         "ai_provider",
			CollectionId: aiProvidersCollection.Id,
			MaxSelect:    1,
		})

		// Generation timestamp
		collection.Fields.Add(&core.DateField{
			Name: "generated_at",
		})

		// Generation configuration (target_role, style, length, emphasis)
		collection.Fields.Add(&core.JSONField{
			Name: "generation_config",
		})

		// Status tracking
		collection.Fields.Add(&core.SelectField{
			Name:      "status",
			Values:    []string{"pending", "processing", "completed", "failed"},
			MaxSelect: 1,
		})

		// Error message if failed
		collection.Fields.Add(&core.TextField{
			Name: "error_message",
			Max:  2000,
		})

		// Only authenticated users can access
		authRule := "@request.auth.id != ''"
		collection.ListRule = &authRule
		collection.ViewRule = &authRule
		collection.CreateRule = &authRule
		collection.UpdateRule = &authRule
		collection.DeleteRule = &authRule

		if err := app.Save(collection); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Rollback: delete the collection
		collection, err := app.FindCollectionByNameOrId("view_exports")
		if err != nil {
			return nil // Collection doesn't exist, nothing to do
		}
		return app.Delete(collection)
	})
}
