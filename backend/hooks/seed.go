package hooks

import (
	"encoding/json"
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterSeedHook seeds demo data on first run (dev mode only)
func RegisterSeedHook(app *pocketbase.PocketBase) {
	// Only seed in development
	if os.Getenv("SEED_DATA") != "true" {
		return
	}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		go func() {
			if err := seedDemoData(app); err != nil {
				log.Printf("Seed warning: %v", err)
			}
		}()
		return se.Next()
	})
}

func seedDemoData(app *pocketbase.PocketBase) error {
	// Check if already seeded
	count, _ := app.CountRecords("profile")
	if count > 0 {
		return nil
	}

	log.Println("Seeding demo data...")

	// Create default superuser for first-time setup
	superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err == nil {
		superuserCount, _ := app.CountRecords(core.CollectionNameSuperusers)
		if superuserCount == 0 {
			admin := core.NewRecord(superusers)
			admin.Set("email", "admin@localhost")
			admin.SetPassword("changeme123")
			if err := app.Save(admin); err != nil {
				log.Printf("Warning: Could not create default admin: %v", err)
			} else {
				log.Println("Created default admin account:")
				log.Println("  Email: admin@localhost")
				log.Println("  Password: changeme123")
				log.Println("  ⚠️  CHANGE THIS PASSWORD IMMEDIATELY!")
			}
		}
	}

	// Create profile
	profileColl, err := app.FindCollectionByNameOrId("profile")
	if err != nil {
		return err
	}

	profile := core.NewRecord(profileColl)
	profile.Set("name", "Alex Chen")
	profile.Set("headline", "Full-Stack Developer & Open Source Enthusiast")
	profile.Set("location", "San Francisco, CA")
	profile.Set("summary", "I build things for the web. Passionate about developer tools, distributed systems, and making complex technology accessible.")
	profile.Set("contact_email", "alex@example.com")
	profile.Set("contact_links", []map[string]string{
		{"type": "github", "url": "https://github.com/alexchen"},
		{"type": "linkedin", "url": "https://linkedin.com/in/alexchen"},
	})
	profile.Set("visibility", "public")
	if err := app.Save(profile); err != nil {
		return err
	}

	// Create experience
	expColl, _ := app.FindCollectionByNameOrId("experience")

	exp1 := core.NewRecord(expColl)
	exp1.Set("company", "TechCorp")
	exp1.Set("title", "Senior Software Engineer")
	exp1.Set("location", "San Francisco, CA")
	exp1.Set("start_date", "2021-03-01")
	exp1.Set("description", "Leading development of cloud-native microservices platform.")
	exp1.Set("bullets", []string{
		"Led team of 5 engineers building real-time data pipeline",
		"Reduced infrastructure costs by 40% through optimization",
	})
	exp1.Set("skills", []string{"Go", "Kubernetes", "PostgreSQL"})
	exp1.Set("visibility", "public")
	exp1.Set("is_draft", false)
	exp1.Set("sort_order", 1)
	app.Save(exp1)

	exp2 := core.NewRecord(expColl)
	exp2.Set("company", "StartupXYZ")
	exp2.Set("title", "Software Engineer")
	exp2.Set("location", "Remote")
	exp2.Set("start_date", "2019-01-01")
	exp2.Set("end_date", "2021-02-28")
	exp2.Set("description", "Full-stack development for B2B SaaS platform.")
	exp2.Set("skills", []string{"TypeScript", "React", "Node.js"})
	exp2.Set("visibility", "public")
	exp2.Set("is_draft", false)
	exp2.Set("sort_order", 2)
	app.Save(exp2)

	// Create projects
	projColl, _ := app.FindCollectionByNameOrId("projects")

	proj1 := core.NewRecord(projColl)
	proj1.Set("title", "DataSync")
	proj1.Set("slug", "datasync")
	proj1.Set("summary", "Open-source real-time data synchronization library")
	proj1.Set("description", "DataSync provides conflict-free replicated data types (CRDTs) for building collaborative applications.\n\n## Features\n- Automatic conflict resolution\n- Offline-first architecture\n- Sub-millisecond sync latency")
	proj1.Set("tech_stack", []string{"Go", "Protocol Buffers", "WebSocket"})
	proj1.Set("links", []map[string]string{
		{"type": "github", "url": "https://github.com/alexchen/datasync"},
	})
	proj1.Set("categories", []string{"open-source", "distributed-systems"})
	proj1.Set("visibility", "public")
	proj1.Set("is_draft", false)
	proj1.Set("is_featured", true)
	proj1.Set("sort_order", 1)
	app.Save(proj1)

	proj2 := core.NewRecord(projColl)
	proj2.Set("title", "DevDash")
	proj2.Set("slug", "devdash")
	proj2.Set("summary", "Self-hosted developer dashboard")
	proj2.Set("description", "A single pane of glass for developers. See your PRs, deployments, and alerts in one place.")
	proj2.Set("tech_stack", []string{"SvelteKit", "TypeScript", "SQLite"})
	proj2.Set("links", []map[string]string{
		{"type": "github", "url": "https://github.com/alexchen/devdash"},
	})
	proj2.Set("categories", []string{"self-hosted", "developer-tools"})
	proj2.Set("visibility", "public")
	proj2.Set("is_draft", false)
	proj2.Set("is_featured", true)
	proj2.Set("sort_order", 2)
	app.Save(proj2)

	// Create education
	eduColl, _ := app.FindCollectionByNameOrId("education")

	edu1 := core.NewRecord(eduColl)
	edu1.Set("institution", "UC Berkeley")
	edu1.Set("degree", "B.S.")
	edu1.Set("field", "Computer Science")
	edu1.Set("start_date", "2014-08-01")
	edu1.Set("end_date", "2018-05-15")
	edu1.Set("visibility", "public")
	edu1.Set("is_draft", false)
	edu1.Set("sort_order", 1)
	app.Save(edu1)

	// Create skills
	skillsColl, _ := app.FindCollectionByNameOrId("skills")

	skills := []struct {
		name        string
		category    string
		proficiency string
		order       int
	}{
		{"Go", "Languages", "expert", 1},
		{"TypeScript", "Languages", "expert", 2},
		{"Python", "Languages", "proficient", 3},
		{"Kubernetes", "Infrastructure", "expert", 4},
		{"Docker", "Infrastructure", "expert", 5},
		{"PostgreSQL", "Databases", "expert", 6},
		{"React", "Frontend", "proficient", 7},
		{"Svelte", "Frontend", "proficient", 8},
	}

	for _, s := range skills {
		skill := core.NewRecord(skillsColl)
		skill.Set("name", s.name)
		skill.Set("category", s.category)
		skill.Set("proficiency", s.proficiency)
		skill.Set("visibility", "public")
		skill.Set("sort_order", s.order)
		app.Save(skill)
	}

	// Create a curated view
	viewsColl, _ := app.FindCollectionByNameOrId("views")

	view := core.NewRecord(viewsColl)
	view.Set("name", "For Recruiters")
	view.Set("slug", "recruiters")
	view.Set("description", "Curated view for technical recruiters")
	view.Set("visibility", "public")
	view.Set("hero_headline", "Experienced Full-Stack Engineer")
	view.Set("hero_summary", "8+ years building scalable systems. Open to senior roles.")
	view.Set("cta_text", "Get in Touch")
	view.Set("cta_url", "mailto:alex@example.com")
	sectionsJSON, _ := json.Marshal([]map[string]interface{}{
		{"section": "experience", "enabled": true},
		{"section": "skills", "enabled": true},
		{"section": "projects", "enabled": true},
		{"section": "education", "enabled": true},
	})
	view.Set("sections", string(sectionsJSON))
	view.Set("is_active", true)
	app.Save(view)

	log.Println("Demo data seeded successfully!")
	log.Println("  Profile: Alex Chen")
	log.Println("  View: /v/recruiters")

	return nil
}
