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

	// Create PocketBase superuser for /_/ admin access (dev mode only)
	superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err == nil && superusers != nil {
		superuserCount, _ := app.CountRecords(core.CollectionNameSuperusers)
		if superuserCount == 0 {
			su := core.NewRecord(superusers)
			su.SetEmail("admin@localhost.dev")
			su.SetPassword("admin123")
			if err := app.Save(su); err != nil {
				log.Printf("Warning: Could not create dev superuser: %v", err)
			} else {
				log.Println("")
				log.Println("========================================")
				log.Println("  DEV MODE: PocketBase Admin Created")
				log.Println("========================================")
				log.Println("  URL:      http://localhost:8090/_/")
				log.Println("  Email:    admin@localhost.dev")
				log.Println("  Password: admin123")
				log.Println("========================================")
				log.Println("")
			}
		}
	}

	// Create default user for frontend admin
	users, err := app.FindCollectionByNameOrId("users")
	if err == nil {
		userCount, _ := app.CountRecords("users")
		if userCount == 0 {
			admin := core.NewRecord(users)
			admin.Set("email", "admin@example.com")
			admin.Set("name", "Admin")
			admin.Set("is_admin", true)
			admin.SetPassword("changeme123")
			if err := app.Save(admin); err != nil {
				log.Printf("Warning: Could not create default admin user: %v", err)
			} else {
				log.Println("Created default frontend admin account:")
				log.Println("  Email: admin@example.com")
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
	profile.Set("name", "Jedidiah Esposito")
	profile.Set("headline", "Front-End Lead | Product Engineering Lead")
	profile.Set("location", "Wellington, New Zealand")
	profile.Set("summary", "Front-end–leaning product engineer and team lead with 16+ years building user-facing systems, tools, and digital platforms. I specialise in content-driven applications, admin-style interfaces, and workflow-heavy front ends where clarity, usability, and adoption matter more than novelty.\n\nMy background includes hands-on product and platform work at Amazon, Okta, and Ryman Healthcare, alongside leadership roles in learning and communications teams. Outside formal roles, I actively design and build a modern full-stack web application. I bring pragmatic engineering judgement, strong UX instincts, and experience working in regulated, high-trust environments.")
	profile.Set("contact_email", "")
	profile.Set("contact_links", []map[string]string{
		{"type": "linkedin", "url": "https://linkedin.com/in/jedidiah-esposito"},
		{"type": "github", "url": "https://github.com/jesposito"},
	})
	profile.Set("visibility", "public")
	if err := app.Save(profile); err != nil {
		return err
	}

	// Create experience
	expColl, _ := app.FindCollectionByNameOrId("experience")

	// Experience 1: NZ Police (Current)
	exp1 := core.NewRecord(expColl)
	exp1.Set("company", "NZ Police")
	exp1.Set("title", "Manager, Communications and Enablement")
	exp1.Set("location", "Wellington, New Zealand")
	exp1.Set("start_date", "2025-05-01")
	exp1.Set("description", "Lead a small team maintaining a national SharePoint site for road policing staff. This role is content and communications focused rather than software development.")
	exp1.Set("bullets", []string{
		"Lead a small team maintaining a national SharePoint site for road policing staff",
		"Develop and publish written guidance, videos, and occasional eLearning modules",
		"Focus on information structure, usability, and clarity for frontline users",
		"Work with subject matter experts to translate policy and operational changes into accessible digital content",
	})
	exp1.Set("skills", []string{"SharePoint", "Content Strategy", "Team Leadership"})
	exp1.Set("visibility", "public")
	exp1.Set("is_draft", false)
	exp1.Set("sort_order", 1)
	app.Save(exp1)

	// Experience 2: Ryman Healthcare
	exp2 := core.NewRecord(expColl)
	exp2.Set("company", "Ryman Healthcare")
	exp2.Set("title", "Senior Learning Designer")
	exp2.Set("location", "Christchurch, New Zealand")
	exp2.Set("start_date", "2023-11-01")
	exp2.Set("end_date", "2025-05-31")
	exp2.Set("description", "Acted as product owner, solution architect, and developer for an internal onboarding automation used across New Zealand and Australia.")
	exp2.Set("bullets", []string{
		"Acted as product owner, solution architect, and developer for an internal onboarding system",
		"Designed user-facing workflows and interfaces focused on clarity and adoption",
		"Built and maintained automation and analytics supporting operational teams",
		"Worked closely with HR, IT, and business stakeholders to iterate on the system",
	})
	exp2.Set("skills", []string{"Power Automate", "Slack", "Product Design", "Analytics"})
	exp2.Set("visibility", "public")
	exp2.Set("is_draft", false)
	exp2.Set("sort_order", 2)
	app.Save(exp2)

	// Experience 3: Okta (Auth0)
	exp3 := core.NewRecord(expColl)
	exp3.Set("company", "Okta (Auth0)")
	exp3.Set("title", "Senior Technical Curriculum Developer")
	exp3.Set("location", "Remote, United States")
	exp3.Set("start_date", "2020-01-01")
	exp3.Set("end_date", "2023-10-31")
	exp3.Set("description", "Built and maintained hands-on technical platforms supporting developer education for identity and API security.")
	exp3.Set("bullets", []string{
		"Built and maintained hands-on technical platforms supporting developer education for identity and API security",
		"Designed containerised lab environments and supporting UI flows",
		"Partnered with engineering teams on infrastructure requirements and implementation",
		"Reduced developer setup friction through improved tooling and platform design",
		"Managed a distributed team using Agile practices and provided technical review",
	})
	exp3.Set("skills", []string{"Docker", "API Security", "Developer Education", "Agile"})
	exp3.Set("visibility", "public")
	exp3.Set("is_draft", false)
	exp3.Set("sort_order", 3)
	app.Save(exp3)

	// Experience 4: Amazon - Alexa Operational Excellence
	exp4 := core.NewRecord(expColl)
	exp4.Set("company", "Amazon")
	exp4.Set("title", "Learning Program Manager, Alexa Operational Excellence")
	exp4.Set("location", "Seattle, United States")
	exp4.Set("start_date", "2019-01-01")
	exp4.Set("end_date", "2020-12-31")
	exp4.Set("description", "Designed and built a lightweight, stateless learning analytics pipeline for Alexa teams.")
	exp4.Set("bullets", []string{
		"Designed and built a lightweight, stateless learning analytics pipeline",
		"Implemented event-based data capture using AWS Lambda, S3, and Athena",
		"Enabled internal reporting without a traditional LMS or persistent application state",
		"Supported operational and leadership decision-making across Alexa teams",
	})
	exp4.Set("skills", []string{"AWS Lambda", "S3", "Athena", "Analytics"})
	exp4.Set("visibility", "public")
	exp4.Set("is_draft", false)
	exp4.Set("sort_order", 4)
	app.Save(exp4)

	// Experience 5: Amazon - Alexa Developer Education
	exp5 := core.NewRecord(expColl)
	exp5.Set("company", "Amazon")
	exp5.Set("title", "Technical Learning Program Manager, Alexa Developer Education")
	exp5.Set("location", "Seattle, United States")
	exp5.Set("start_date", "2017-01-01")
	exp5.Set("end_date", "2019-12-31")
	exp5.Set("description", "Led developer-facing education at the intersection of product, engineering, and UX.")
	exp5.Set("bullets", []string{
		"Led developer-facing education at the intersection of product, engineering, and UX",
		"Published technical tools and content used by over 100,000 external developers",
		"Worked with product teams to test and influence API and console design",
		"Translated complex interaction patterns into clear, usable interfaces",
	})
	exp5.Set("skills", []string{"Developer Education", "API Design", "UX", "Technical Writing"})
	exp5.Set("visibility", "public")
	exp5.Set("is_draft", false)
	exp5.Set("sort_order", 5)
	app.Save(exp5)

	// Experience 6: ChefSteps
	exp6 := core.NewRecord(expColl)
	exp6.Set("company", "ChefSteps")
	exp6.Set("title", "Voice UI Designer")
	exp6.Set("location", "Seattle, United States")
	exp6.Set("start_date", "2019-02-01")
	exp6.Set("end_date", "2019-04-30")
	exp6.Set("description", "Refactored the conversational user experience for the Joule Alexa skill.")
	exp6.Set("bullets", []string{
		"Refactored the conversational user experience for the Joule Alexa skill",
		"Improved flow, state handling, and error recovery in a production voice interface",
		"Collaborated with product and engineering teams to align interaction patterns with real-world cooking workflows",
	})
	exp6.Set("skills", []string{"Voice UI", "Alexa Skills", "Conversational Design"})
	exp6.Set("visibility", "public")
	exp6.Set("is_draft", false)
	exp6.Set("sort_order", 6)
	app.Save(exp6)

	// Create projects
	projColl, _ := app.FindCollectionByNameOrId("projects")

	// Project 1: Me.yaml
	proj1 := core.NewRecord(projColl)
	proj1.Set("title", "Me.yaml")
	proj1.Set("slug", "meyaml")
	proj1.Set("summary", "Personal Publishing Platform | Active Development")
	proj1.Set("description", "Designing and building a full-stack, content-driven web application.\n\n## Key Contributions\n- Own front-end architecture, routing, and UI implementation using SvelteKit\n- Built admin interfaces for creating and managing content\n- Implemented server-side data loading and structured routing patterns\n- Maintain clear architecture, design, and roadmap documentation")
	proj1.Set("tech_stack", []string{"SvelteKit", "JavaScript", "TypeScript", "Docker", "REST APIs"})
	proj1.Set("links", []map[string]string{
		{"type": "github", "url": "https://github.com/jesposito/Me.yaml"},
	})
	proj1.Set("categories", []string{"full-stack", "personal-project"})
	proj1.Set("visibility", "public")
	proj1.Set("is_draft", false)
	proj1.Set("is_featured", true)
	proj1.Set("sort_order", 1)
	app.Save(proj1)

	// Project 2: Automated Onboarding System
	proj2 := core.NewRecord(projColl)
	proj2.Set("title", "Automated Onboarding System")
	proj2.Set("slug", "automated-onboarding")
	proj2.Set("summary", "Internal onboarding automation for Ryman Healthcare across New Zealand and Australia")
	proj2.Set("description", "This was production platform work with real users and measurable outcomes.\n\n## Key Contributions\n- Owned end-to-end design and build of an internal onboarding automation\n- Designed user workflows and UI surfaces using Slack Canvas and automated task provisioning\n- Built orchestration flows integrating HR systems, Slack, and analytics using Power Automate\n- Delivered dashboards providing operational visibility to leaders\n- Eliminated manual handoffs and scaled the solution across multiple facilities")
	proj2.Set("tech_stack", []string{"Power Automate", "Slack Canvas", "Analytics", "HR Integration"})
	proj2.Set("links", []map[string]string{})
	proj2.Set("categories", []string{"enterprise", "automation"})
	proj2.Set("visibility", "public")
	proj2.Set("is_draft", false)
	proj2.Set("is_featured", true)
	proj2.Set("sort_order", 2)
	app.Save(proj2)

	// Create certifications
	certColl, _ := app.FindCollectionByNameOrId("certifications")

	cert1 := core.NewRecord(certColl)
	cert1.Set("name", "Project Management Professional (PMP)")
	cert1.Set("issuer", "Project Management Institute")
	cert1.Set("visibility", "public")
	cert1.Set("is_draft", false)
	cert1.Set("sort_order", 1)
	app.Save(cert1)

	cert2 := core.NewRecord(certColl)
	cert2.Set("name", "Certified Scrum Professional ScrumMaster (CSP-SM)")
	cert2.Set("issuer", "Scrum Alliance")
	cert2.Set("visibility", "public")
	cert2.Set("is_draft", false)
	cert2.Set("sort_order", 2)
	app.Save(cert2)

	cert3 := core.NewRecord(certColl)
	cert3.Set("name", "AWS Certified Solutions Architect Associate")
	cert3.Set("issuer", "Amazon Web Services")
	cert3.Set("visibility", "public")
	cert3.Set("is_draft", false)
	cert3.Set("sort_order", 3)
	app.Save(cert3)

	// Create skills
	skillsColl, _ := app.FindCollectionByNameOrId("skills")

	skills := []struct {
		name        string
		category    string
		proficiency string
		order       int
	}{
		{"Front-end Architecture", "Core Competencies", "expert", 1},
		{"UI Development", "Core Competencies", "expert", 2},
		{"SvelteKit", "Technologies", "expert", 3},
		{"JavaScript", "Technologies", "expert", 4},
		{"TypeScript", "Technologies", "expert", 5},
		{"Content-driven Applications", "Specialisations", "expert", 6},
		{"Admin-style Interfaces", "Specialisations", "expert", 7},
		{"API Integration", "Specialisations", "proficient", 8},
		{"Data-driven UI", "Specialisations", "proficient", 9},
		{"Docker", "Infrastructure", "proficient", 10},
		{"Containerised Services", "Infrastructure", "proficient", 11},
		{"Automation", "Tooling", "proficient", 12},
		{"Workflow Tooling", "Tooling", "proficient", 13},
		{"Agile Delivery", "Leadership", "expert", 14},
		{"Technical Leadership", "Leadership", "expert", 15},
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

	// Create a curated view: Front-End Lead
	viewsColl, _ := app.FindCollectionByNameOrId("views")

	view := core.NewRecord(viewsColl)
	view.Set("name", "Front-End Lead")
	view.Set("slug", "front-end-lead")
	view.Set("description", "Curated view highlighting front-end leadership and product engineering experience")
	view.Set("visibility", "public")
	view.Set("hero_headline", "Front-End Lead | Product Engineering Lead")
	view.Set("hero_summary", "Front-end–leaning product engineer and team lead with 16+ years building user-facing systems, tools, and digital platforms. I specialise in content-driven applications, admin-style interfaces, and workflow-heavy front ends where clarity, usability, and adoption matter more than novelty.")
	view.Set("cta_text", "View LinkedIn")
	view.Set("cta_url", "https://linkedin.com/in/jedidiah-esposito")
	sectionsJSON, _ := json.Marshal([]map[string]interface{}{
		{"section": "experience", "enabled": true, "layout": "default"},
		{"section": "projects", "enabled": true, "layout": "grid-2"},
		{"section": "skills", "enabled": true, "layout": "grouped"},
		{"section": "certifications", "enabled": true, "layout": "grouped"},
	})
	view.Set("sections", string(sectionsJSON))
	view.Set("is_active", true)
	view.Set("is_default", true)
	app.Save(view)

	log.Println("Demo data seeded successfully!")
	log.Println("  Profile: Jedidiah Esposito")
	log.Println("  View: /front-end-lead")

	return nil
}
