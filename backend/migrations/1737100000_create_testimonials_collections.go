package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Helper for public read access
		publicRule := ""

		// Helper to check if collection exists
		collectionExists := func(name string) bool {
			_, err := app.FindCollectionByNameOrId(name)
			return err == nil
		}

		// Create Testimonials collection
		if !collectionExists("testimonials") {
			collection := core.NewBaseCollection("testimonials")
			collection.ListRule = &publicRule
			collection.ViewRule = &publicRule

			// Content fields
			collection.Fields.Add(&core.EditorField{Name: "content"})
			collection.Fields.Add(&core.SelectField{
				Name:      "relationship",
				Values:    []string{"client", "colleague", "manager", "report", "mentor", "other"},
				MaxSelect: 1,
			})
			collection.Fields.Add(&core.TextField{Name: "project", Max: 500})

			// Author info
			collection.Fields.Add(&core.TextField{Name: "author_name", Required: true, Max: 200})
			collection.Fields.Add(&core.TextField{Name: "author_title", Max: 200})
			collection.Fields.Add(&core.TextField{Name: "author_company", Max: 200})
			collection.Fields.Add(&core.FileField{
				Name:      "author_photo",
				MaxSize:   5242880, // 5MB
				MaxSelect: 1,
				MimeTypes: []string{"image/jpeg", "image/png", "image/webp"},
			})
			collection.Fields.Add(&core.URLField{Name: "author_website"})

			// Verification fields
			collection.Fields.Add(&core.SelectField{
				Name:      "verification_method",
				Values:    []string{"none", "email", "github", "twitter", "linkedin"},
				MaxSelect: 1,
			})
			collection.Fields.Add(&core.TextField{Name: "verification_identifier", Max: 500})
			collection.Fields.Add(&core.JSONField{Name: "verification_data"})
			collection.Fields.Add(&core.DateField{Name: "verified_at"})

			// Workflow fields
			collection.Fields.Add(&core.SelectField{
				Name:      "status",
				Values:    []string{"pending", "approved", "rejected"},
				MaxSelect: 1,
			})
			collection.Fields.Add(&core.TextField{Name: "request_id", Max: 50})
			collection.Fields.Add(&core.DateField{Name: "submitted_at"})
			collection.Fields.Add(&core.DateField{Name: "approved_at"})
			collection.Fields.Add(&core.DateField{Name: "rejected_at"})
			collection.Fields.Add(&core.TextField{Name: "rejection_reason", Max: 1000})

			// Display fields
			collection.Fields.Add(&core.BoolField{Name: "featured"})
			collection.Fields.Add(&core.NumberField{Name: "sort_order"})

			// Indexes
			collection.Indexes = []string{
				"CREATE INDEX idx_testimonials_status ON testimonials (status)",
				"CREATE INDEX idx_testimonials_request_id ON testimonials (request_id)",
			}

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		// Create Testimonial Requests collection
		if !collectionExists("testimonial_requests") {
			collection := core.NewBaseCollection("testimonial_requests")

			// Security fields (same pattern as share_tokens)
			collection.Fields.Add(&core.TextField{Name: "token_hash", Required: true, Max: 100})
			collection.Fields.Add(&core.TextField{Name: "token_prefix", Max: 20})

			// Customization fields
			collection.Fields.Add(&core.TextField{Name: "label", Max: 200})
			collection.Fields.Add(&core.EditorField{Name: "custom_message"})
			collection.Fields.Add(&core.TextField{Name: "recipient_name", Max: 200})
			collection.Fields.Add(&core.EmailField{Name: "recipient_email"})

			// Constraints
			collection.Fields.Add(&core.DateField{Name: "expires_at"})
			collection.Fields.Add(&core.NumberField{Name: "max_uses"})
			collection.Fields.Add(&core.NumberField{Name: "use_count"})

			// State
			collection.Fields.Add(&core.BoolField{Name: "is_active"})

			// Indexes
			collection.Indexes = []string{
				"CREATE UNIQUE INDEX idx_testimonial_requests_hash ON testimonial_requests (token_hash)",
				"CREATE INDEX idx_testimonial_requests_prefix ON testimonial_requests (token_prefix)",
			}

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		// Create Email Verification Tokens collection (ephemeral)
		if !collectionExists("email_verification_tokens") {
			collection := core.NewBaseCollection("email_verification_tokens")

			collection.Fields.Add(&core.TextField{Name: "testimonial_id", Required: true, Max: 50})
			collection.Fields.Add(&core.EmailField{Name: "email", Required: true})
			collection.Fields.Add(&core.TextField{Name: "token_hash", Required: true, Max: 100})
			collection.Fields.Add(&core.DateField{Name: "expires_at"})
			collection.Fields.Add(&core.DateField{Name: "verified_at"})

			// Indexes
			collection.Indexes = []string{
				"CREATE UNIQUE INDEX idx_email_verification_hash ON email_verification_tokens (token_hash)",
				"CREATE INDEX idx_email_verification_testimonial ON email_verification_tokens (testimonial_id)",
			}

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: delete all testimonial-related collections
		collections := []string{
			"email_verification_tokens",
			"testimonial_requests",
			"testimonials",
		}

		for _, name := range collections {
			collection, err := app.FindCollectionByNameOrId(name)
			if err == nil {
				app.Delete(collection)
			}
		}

		return nil
	})
}
