package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Create Profile collection (singleton)
		profileCollection := core.NewBaseCollection("profile")
		profileCollection.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 200})
		profileCollection.Fields.Add(&core.TextField{Name: "headline", Max: 500})
		profileCollection.Fields.Add(&core.TextField{Name: "location", Max: 200})
		profileCollection.Fields.Add(&core.EditorField{Name: "summary"})
		profileCollection.Fields.Add(&core.FileField{Name: "hero_image", MaxSize: 10485760, MimeTypes: []string{"image/jpeg", "image/png", "image/webp", "image/gif"}})
		profileCollection.Fields.Add(&core.FileField{Name: "avatar", MaxSize: 5242880, MimeTypes: []string{"image/jpeg", "image/png", "image/webp"}})
		profileCollection.Fields.Add(&core.EmailField{Name: "contact_email"})
		profileCollection.Fields.Add(&core.JSONField{Name: "contact_links"})
		profileCollection.Fields.Add(&core.SelectField{Name: "visibility", Values: []string{"public", "unlisted", "private"}, MaxSelect: 1})

		if err := app.Save(profileCollection); err != nil {
			return err
		}

		// Create Experience collection
		experienceCollection := core.NewBaseCollection("experience")
		experienceCollection.Fields.Add(&core.TextField{Name: "company", Required: true, Max: 200})
		experienceCollection.Fields.Add(&core.TextField{Name: "title", Required: true, Max: 200})
		experienceCollection.Fields.Add(&core.TextField{Name: "location", Max: 200})
		experienceCollection.Fields.Add(&core.DateField{Name: "start_date"})
		experienceCollection.Fields.Add(&core.DateField{Name: "end_date"})
		experienceCollection.Fields.Add(&core.EditorField{Name: "description"})
		experienceCollection.Fields.Add(&core.JSONField{Name: "bullets"})
		experienceCollection.Fields.Add(&core.JSONField{Name: "skills"})
		experienceCollection.Fields.Add(&core.FileField{Name: "media", MaxSize: 10485760, MaxSelect: 10})
		experienceCollection.Fields.Add(&core.SelectField{Name: "visibility", Values: []string{"public", "unlisted", "private"}, MaxSelect: 1})
		experienceCollection.Fields.Add(&core.BoolField{Name: "is_draft"})
		experienceCollection.Fields.Add(&core.NumberField{Name: "sort_order"})

		if err := app.Save(experienceCollection); err != nil {
			return err
		}

		// Create Projects collection
		projectsCollection := core.NewBaseCollection("projects")
		projectsCollection.Fields.Add(&core.TextField{Name: "title", Required: true, Max: 200})
		projectsCollection.Fields.Add(&core.TextField{Name: "slug", Max: 200})
		projectsCollection.Fields.Add(&core.TextField{Name: "summary", Max: 1000})
		projectsCollection.Fields.Add(&core.EditorField{Name: "description"})
		projectsCollection.Fields.Add(&core.JSONField{Name: "tech_stack"})
		projectsCollection.Fields.Add(&core.JSONField{Name: "links"})
		projectsCollection.Fields.Add(&core.FileField{Name: "media", MaxSize: 10485760, MaxSelect: 20})
		projectsCollection.Fields.Add(&core.FileField{Name: "cover_image", MaxSize: 10485760, MimeTypes: []string{"image/jpeg", "image/png", "image/webp"}})
		projectsCollection.Fields.Add(&core.JSONField{Name: "categories"})
		projectsCollection.Fields.Add(&core.SelectField{Name: "visibility", Values: []string{"public", "unlisted", "private"}, MaxSelect: 1})
		projectsCollection.Fields.Add(&core.BoolField{Name: "is_draft"})
		projectsCollection.Fields.Add(&core.BoolField{Name: "is_featured"})
		projectsCollection.Fields.Add(&core.NumberField{Name: "sort_order"})
		projectsCollection.Fields.Add(&core.TextField{Name: "source_id"})
		projectsCollection.Fields.Add(&core.JSONField{Name: "field_locks"})
		projectsCollection.Fields.Add(&core.DateField{Name: "last_sync"})

		// Add slug unique index
		projectsCollection.Indexes = append(projectsCollection.Indexes, "CREATE UNIQUE INDEX idx_projects_slug ON projects(slug) WHERE slug != ''")

		if err := app.Save(projectsCollection); err != nil {
			return err
		}

		// Create Education collection
		educationCollection := core.NewBaseCollection("education")
		educationCollection.Fields.Add(&core.TextField{Name: "institution", Required: true, Max: 200})
		educationCollection.Fields.Add(&core.TextField{Name: "degree", Max: 200})
		educationCollection.Fields.Add(&core.TextField{Name: "field", Max: 200})
		educationCollection.Fields.Add(&core.DateField{Name: "start_date"})
		educationCollection.Fields.Add(&core.DateField{Name: "end_date"})
		educationCollection.Fields.Add(&core.EditorField{Name: "description"})
		educationCollection.Fields.Add(&core.SelectField{Name: "visibility", Values: []string{"public", "unlisted", "private"}, MaxSelect: 1})
		educationCollection.Fields.Add(&core.BoolField{Name: "is_draft"})
		educationCollection.Fields.Add(&core.NumberField{Name: "sort_order"})

		if err := app.Save(educationCollection); err != nil {
			return err
		}

		// Create Certifications collection
		certsCollection := core.NewBaseCollection("certifications")
		certsCollection.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 200})
		certsCollection.Fields.Add(&core.TextField{Name: "issuer", Max: 200})
		certsCollection.Fields.Add(&core.DateField{Name: "issue_date"})
		certsCollection.Fields.Add(&core.DateField{Name: "expiry_date"})
		certsCollection.Fields.Add(&core.TextField{Name: "credential_id", Max: 200})
		certsCollection.Fields.Add(&core.URLField{Name: "credential_url"})
		certsCollection.Fields.Add(&core.SelectField{Name: "visibility", Values: []string{"public", "unlisted", "private"}, MaxSelect: 1})
		certsCollection.Fields.Add(&core.BoolField{Name: "is_draft"})
		certsCollection.Fields.Add(&core.NumberField{Name: "sort_order"})

		if err := app.Save(certsCollection); err != nil {
			return err
		}

		// Create Skills collection
		skillsCollection := core.NewBaseCollection("skills")
		skillsCollection.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 100})
		skillsCollection.Fields.Add(&core.TextField{Name: "category", Max: 100})
		skillsCollection.Fields.Add(&core.SelectField{Name: "proficiency", Values: []string{"expert", "proficient", "familiar"}, MaxSelect: 1})
		skillsCollection.Fields.Add(&core.SelectField{Name: "visibility", Values: []string{"public", "unlisted", "private"}, MaxSelect: 1})
		skillsCollection.Fields.Add(&core.NumberField{Name: "sort_order"})

		if err := app.Save(skillsCollection); err != nil {
			return err
		}

		// Create Posts collection
		postsCollection := core.NewBaseCollection("posts")
		postsCollection.Fields.Add(&core.TextField{Name: "title", Required: true, Max: 500})
		postsCollection.Fields.Add(&core.TextField{Name: "slug", Max: 200})
		postsCollection.Fields.Add(&core.TextField{Name: "excerpt", Max: 1000})
		postsCollection.Fields.Add(&core.EditorField{Name: "content"})
		postsCollection.Fields.Add(&core.FileField{Name: "cover_image", MaxSize: 10485760, MimeTypes: []string{"image/jpeg", "image/png", "image/webp"}})
		postsCollection.Fields.Add(&core.JSONField{Name: "tags"})
		postsCollection.Fields.Add(&core.SelectField{Name: "visibility", Values: []string{"public", "unlisted", "private"}, MaxSelect: 1})
		postsCollection.Fields.Add(&core.BoolField{Name: "is_draft"})
		postsCollection.Fields.Add(&core.DateField{Name: "published_at"})

		postsCollection.Indexes = append(postsCollection.Indexes, "CREATE UNIQUE INDEX idx_posts_slug ON posts(slug) WHERE slug != ''")

		if err := app.Save(postsCollection); err != nil {
			return err
		}

		// Create Talks collection
		talksCollection := core.NewBaseCollection("talks")
		talksCollection.Fields.Add(&core.TextField{Name: "title", Required: true, Max: 500})
		talksCollection.Fields.Add(&core.TextField{Name: "event", Max: 200})
		talksCollection.Fields.Add(&core.URLField{Name: "event_url"})
		talksCollection.Fields.Add(&core.DateField{Name: "date"})
		talksCollection.Fields.Add(&core.TextField{Name: "location", Max: 200})
		talksCollection.Fields.Add(&core.EditorField{Name: "description"})
		talksCollection.Fields.Add(&core.URLField{Name: "slides_url"})
		talksCollection.Fields.Add(&core.URLField{Name: "video_url"})
		talksCollection.Fields.Add(&core.SelectField{Name: "visibility", Values: []string{"public", "unlisted", "private"}, MaxSelect: 1})
		talksCollection.Fields.Add(&core.BoolField{Name: "is_draft"})
		talksCollection.Fields.Add(&core.NumberField{Name: "sort_order"})

		if err := app.Save(talksCollection); err != nil {
			return err
		}

		// Create Views collection
		viewsCollection := core.NewBaseCollection("views")
		viewsCollection.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 200})
		viewsCollection.Fields.Add(&core.TextField{Name: "slug", Required: true, Max: 100})
		viewsCollection.Fields.Add(&core.TextField{Name: "description", Max: 500})
		viewsCollection.Fields.Add(&core.SelectField{Name: "visibility", Values: []string{"public", "unlisted", "private", "password"}, MaxSelect: 1})
		viewsCollection.Fields.Add(&core.TextField{Name: "password_hash", Hidden: true})
		viewsCollection.Fields.Add(&core.TextField{Name: "hero_headline", Max: 500})
		viewsCollection.Fields.Add(&core.EditorField{Name: "hero_summary"})
		viewsCollection.Fields.Add(&core.TextField{Name: "cta_text", Max: 100})
		viewsCollection.Fields.Add(&core.URLField{Name: "cta_url"})
		viewsCollection.Fields.Add(&core.JSONField{Name: "sections"})
		viewsCollection.Fields.Add(&core.BoolField{Name: "is_active"})

		viewsCollection.Indexes = append(viewsCollection.Indexes, "CREATE UNIQUE INDEX idx_views_slug ON views(slug)")

		if err := app.Save(viewsCollection); err != nil {
			return err
		}

		// Create Share Tokens collection
		tokensCollection := core.NewBaseCollection("share_tokens")
		tokensCollection.Fields.Add(&core.TextField{Name: "view_id", Required: true})
		tokensCollection.Fields.Add(&core.TextField{Name: "token_hash", Required: true})
		tokensCollection.Fields.Add(&core.TextField{Name: "name", Max: 200})
		tokensCollection.Fields.Add(&core.DateField{Name: "expires_at"})
		tokensCollection.Fields.Add(&core.NumberField{Name: "max_uses"})
		tokensCollection.Fields.Add(&core.NumberField{Name: "use_count"})
		tokensCollection.Fields.Add(&core.BoolField{Name: "is_active"})
		tokensCollection.Fields.Add(&core.DateField{Name: "last_used_at"})

		tokensCollection.Indexes = append(tokensCollection.Indexes, "CREATE UNIQUE INDEX idx_tokens_hash ON share_tokens(token_hash)")

		if err := app.Save(tokensCollection); err != nil {
			return err
		}

		// Create Sources collection
		sourcesCollection := core.NewBaseCollection("sources")
		sourcesCollection.Fields.Add(&core.SelectField{Name: "type", Values: []string{"github"}, Required: true, MaxSelect: 1})
		sourcesCollection.Fields.Add(&core.TextField{Name: "identifier", Required: true, Max: 500})
		sourcesCollection.Fields.Add(&core.TextField{Name: "project_id"})
		sourcesCollection.Fields.Add(&core.TextField{Name: "github_token", Hidden: true})
		sourcesCollection.Fields.Add(&core.DateField{Name: "last_sync"})
		sourcesCollection.Fields.Add(&core.SelectField{Name: "sync_status", Values: []string{"pending", "success", "error"}, MaxSelect: 1})
		sourcesCollection.Fields.Add(&core.TextField{Name: "sync_log"})

		if err := app.Save(sourcesCollection); err != nil {
			return err
		}

		// Create AI Providers collection
		aiCollection := core.NewBaseCollection("ai_providers")
		aiCollection.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 200})
		aiCollection.Fields.Add(&core.SelectField{Name: "type", Values: []string{"openai", "anthropic", "ollama", "custom"}, Required: true, MaxSelect: 1})
		aiCollection.Fields.Add(&core.TextField{Name: "api_key", Hidden: true})
		aiCollection.Fields.Add(&core.TextField{Name: "api_key_encrypted", Hidden: true})
		aiCollection.Fields.Add(&core.URLField{Name: "base_url"})
		aiCollection.Fields.Add(&core.TextField{Name: "model", Max: 100})
		aiCollection.Fields.Add(&core.BoolField{Name: "is_default"})
		aiCollection.Fields.Add(&core.BoolField{Name: "is_active"})
		aiCollection.Fields.Add(&core.DateField{Name: "last_test"})
		aiCollection.Fields.Add(&core.TextField{Name: "test_status", Max: 50})

		if err := app.Save(aiCollection); err != nil {
			return err
		}

		// Create Import Proposals collection
		proposalsCollection := core.NewBaseCollection("import_proposals")
		proposalsCollection.Fields.Add(&core.TextField{Name: "source_id", Required: true})
		proposalsCollection.Fields.Add(&core.TextField{Name: "project_id"})
		proposalsCollection.Fields.Add(&core.JSONField{Name: "proposed_data"})
		proposalsCollection.Fields.Add(&core.JSONField{Name: "diff"})
		proposalsCollection.Fields.Add(&core.BoolField{Name: "ai_enriched"})
		proposalsCollection.Fields.Add(&core.SelectField{Name: "status", Values: []string{"pending", "applied", "rejected"}, MaxSelect: 1})
		proposalsCollection.Fields.Add(&core.JSONField{Name: "applied_fields"})

		if err := app.Save(proposalsCollection); err != nil {
			return err
		}

		// Create Settings collection (for app-wide settings)
		settingsCollection := core.NewBaseCollection("settings")
		settingsCollection.Fields.Add(&core.TextField{Name: "key", Required: true})
		settingsCollection.Fields.Add(&core.JSONField{Name: "value"})

		settingsCollection.Indexes = append(settingsCollection.Indexes, "CREATE UNIQUE INDEX idx_settings_key ON settings(key)")

		if err := app.Save(settingsCollection); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Rollback: delete all collections
		collections := []string{
			"settings", "import_proposals", "ai_providers", "sources",
			"share_tokens", "views", "talks", "posts", "skills",
			"certifications", "education", "projects", "experience", "profile",
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
