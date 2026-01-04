package hooks

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterSeedHook seeds data on first run (development only)
// Environment variable SEED_DATA controls behavior:
//   - "dev": Seeds development data (Jedidiah Esposito) - for development/testing
//   - "minimal": Seeds ONLY a user account (for testing first-run experience)
//   - unset or other: No automatic seeding (production default)
//
// Demo data (Merlin Ambrosius) is available via admin UI toggle, not auto-seeded.
func RegisterSeedHook(app *pocketbase.PocketBase) {
	seedMode := os.Getenv("SEED_DATA")
	if seedMode == "" {
		return
	}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Run seed synchronously - goroutine causes app.Save() to fail silently
		// due to PocketBase v0.23 context handling
		var err error
		switch seedMode {
		case "dev":
			err = seedDevData(app)
		case "minimal":
			err = seedMinimalData(app)
		default:
			log.Printf("Unknown SEED_DATA value: %s (supported: 'dev', 'minimal')", seedMode)
		}
		if err != nil {
			log.Printf("Seed warning: %v", err)
		}
		return se.Next()
	})
}

// SeedDemoData seeds fun Arthurian-themed demo data (exported for admin API)
// Returns error if data already exists
func SeedDemoData(app *pocketbase.PocketBase) error {
	count, _ := app.CountRecords("profile")
	if count > 0 {
		return fmt.Errorf("profile data already exists - clear first")
	}
	return seedDemoData(app)
}

// seedMinimalData creates ONLY a user account - no profile, no content
// Perfect for testing the first-run welcome page experience
func seedMinimalData(app *pocketbase.PocketBase) error {
	// Check if already seeded
	userCount, _ := app.CountRecords("users")
	if userCount > 0 {
		return nil
	}

	log.Println("Seeding minimal data (user only, no profile/content)...")
	log.Println("This mode is for testing the first-run welcome page.")
	log.Println("")

	// Create default user for frontend admin
	if err := createDefaultUser(app); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	log.Println("")
	log.Println("========================================")
	log.Println("  Minimal seed complete!")
	log.Println("  You can now sign in and see the")
	log.Println("  welcome page (no profile/content yet)")
	log.Println("========================================")
	log.Println("")

	return nil
}

// ClearAllData removes all user-created data (for demo reset)
// This clears: profile, experience, projects, education, certifications, skills, posts, talks, views
func ClearAllData(app *pocketbase.PocketBase) error {
	collections := []string{
		"views", "share_tokens", "posts", "talks",
		"experience", "projects", "education", "certifications", "skills",
		"profile",
	}

	for _, collName := range collections {
		records, err := app.FindRecordsByFilter(collName, "1=1", "", 1000, 0, nil)
		if err != nil {
			continue // Collection might not exist
		}
		for _, record := range records {
			if err := app.Delete(record); err != nil {
				log.Printf("Warning: failed to delete %s record: %v", collName, err)
			}
		}
	}

	log.Println("All data cleared")
	return nil
}

// seedDemoData seeds fun Arthurian-themed demo data for new users
func seedDemoData(app *pocketbase.PocketBase) error {
	log.Println("Seeding demo data (Merlin Ambrosius)...")

	// Create default user for frontend admin
	if err := createDefaultUser(app); err != nil {
		log.Printf("Warning: %v", err)
	}

	// Create profile
	profileColl, err := app.FindCollectionByNameOrId("profile")
	if err != nil {
		return err
	}

	profile := core.NewRecord(profileColl)
	profile.Set("name", "Merlin Ambrosius")
	profile.Set("headline", "Chief Wizard & Staff Engineer")
	profile.Set("location", "Camelot, Britannia")
	profile.Set("summary", "Seasoned enchanter and technical advisor with centuries of experience guiding kingdoms through digital transformation. I specialise in prophecy-driven development, crystal ball observability, and mentoring future monarchs.\n\nMy background spans advisory roles at the Court of Camelot, architectural work on the Round Table distributed system, and founding the Avalon School of Applied Wizardry. I bring ancient wisdom to modern problems, strong intuition for emerging threats, and experience working in high-stakes, sword-adjacent environments.")
	profile.Set("contact_email", "merlin@camelot.gov.uk")
	profile.Set("contact_links", []map[string]string{
		{"type": "github", "url": "https://github.com/merlin-ambrosius"},
		{"type": "website", "url": "https://avalon.edu"},
	})
	profile.Set("visibility", "public")
	if err := app.Save(profile); err != nil {
		return err
	}

	// Create experience
	expColl, _ := app.FindCollectionByNameOrId("experience")

	exp1 := core.NewRecord(expColl)
	exp1.Set("company", "Court of Camelot")
	exp1.Set("title", "Chief Wizard & Royal Technical Advisor")
	exp1.Set("location", "Camelot, Britannia")
	exp1.Set("start_date", "0500-01-01")
	exp1.Set("description", "Principal advisor to King Arthur on all matters magical and technical. Architected the Round Table—a revolutionary distributed consensus system for knight coordination.")
	exp1.Set("bullets", []string{
		"Designed and implemented the Round Table distributed system, eliminating hierarchy bugs in knight coordination",
		"Built Excalibur authentication system with stone-based 2FA (only rightful heir can extract credentials)",
		"Established prophecy-driven development methodology, reducing surprise dragon attacks by 73%",
		"Mentored Arthur from squire to king, demonstrating strong leadership development skills",
	})
	exp1.Set("skills", []string{"Prophecy", "Mentorship", "Distributed Systems", "Authentication"})
	exp1.Set("visibility", "public")
	exp1.Set("is_draft", false)
	exp1.Set("sort_order", 1)
	app.Save(exp1)

	exp2 := core.NewRecord(expColl)
	exp2.Set("company", "Avalon School of Applied Wizardry")
	exp2.Set("title", "Founder & Headmaster")
	exp2.Set("location", "Isle of Avalon")
	exp2.Set("start_date", "0450-01-01")
	exp2.Set("end_date", "0499-12-31")
	exp2.Set("description", "Founded premier institution for magical education, training the next generation of court wizards and technical advisors.")
	exp2.Set("bullets", []string{
		"Developed comprehensive curriculum covering transmutation, divination, and basic Python",
		"Graduated 200+ wizards now serving courts across Europe",
		"Pioneered crystal ball technology for remote scrying and video conferencing",
		"Established ethical guidelines for magic use that remain industry standard",
	})
	exp2.Set("skills", []string{"Education", "Curriculum Development", "Crystal Ball Tech"})
	exp2.Set("visibility", "public")
	exp2.Set("is_draft", false)
	exp2.Set("sort_order", 2)
	app.Save(exp2)

	exp3 := core.NewRecord(expColl)
	exp3.Set("company", "Vortigern's Kingdom")
	exp3.Set("title", "Junior Seer")
	exp3.Set("location", "Dinas Emrys, Wales")
	exp3.Set("start_date", "0420-01-01")
	exp3.Set("end_date", "0449-12-31")
	exp3.Set("description", "Early career role providing prophetic consulting services. Notable achievement: diagnosed critical infrastructure issue (fighting dragons under castle foundation).")
	exp3.Set("bullets", []string{
		"Identified root cause of castle instability—two dragons in the basement (red vs white, classic merge conflict)",
		"Provided accurate prophecy of Pendragon dynasty, establishing reputation for reliable foresight",
		"Learned valuable lessons about working with difficult stakeholders",
	})
	exp3.Set("skills", []string{"Prophecy", "Debugging", "Stakeholder Management"})
	exp3.Set("visibility", "public")
	exp3.Set("is_draft", false)
	exp3.Set("sort_order", 3)
	app.Save(exp3)

	// Create projects
	projColl, _ := app.FindCollectionByNameOrId("projects")

	proj1 := core.NewRecord(projColl)
	proj1.Set("title", "The Round Table")
	proj1.Set("slug", "round-table")
	proj1.Set("summary", "Distributed consensus system for knight coordination with zero hierarchy")
	proj1.Set("description", "Revolutionary table-based architecture eliminating the 'head of table' single point of failure.\n\n## Key Features\n- Circular topology ensures equal participation from all knights\n- Quest assignment through distributed voting\n- Built-in conflict resolution for Lancelot-related incidents\n- Seats 150 knights with sub-second consensus")
	proj1.Set("tech_stack", []string{"Oak", "Distributed Systems", "Consensus Algorithms", "Carpentry"})
	proj1.Set("links", []map[string]string{})
	proj1.Set("categories", []string{"infrastructure", "distributed-systems"})
	proj1.Set("visibility", "public")
	proj1.Set("is_draft", false)
	proj1.Set("is_featured", true)
	proj1.Set("sort_order", 1)
	app.Save(proj1)

	proj2 := core.NewRecord(projColl)
	proj2.Set("title", "Excalibur Auth")
	proj2.Set("slug", "excalibur-auth")
	proj2.Set("summary", "Stone-based authentication system with divine right verification")
	proj2.Set("description", "Secure authentication framework combining physical challenge with lineage verification.\n\n## Security Features\n- Sword-from-stone 2FA (must physically extract to authenticate)\n- Divine right verification via Lady of the Lake API\n- Automatic succession handling\n- Immune to social engineering (you either pull it or you don't)")
	proj2.Set("tech_stack", []string{"Enchanted Steel", "OAuth 0.1", "Divine APIs", "Stone"})
	proj2.Set("links", []map[string]string{})
	proj2.Set("categories", []string{"security", "authentication"})
	proj2.Set("visibility", "public")
	proj2.Set("is_draft", false)
	proj2.Set("is_featured", true)
	proj2.Set("sort_order", 2)
	app.Save(proj2)

	proj3 := core.NewRecord(projColl)
	proj3.Set("title", "Crystal Ball Observability")
	proj3.Set("slug", "crystal-ball")
	proj3.Set("summary", "Real-time scrying platform for monitoring quests and kingdom health")
	proj3.Set("description", "Enterprise-grade observability solution for medieval IT operations.\n\n## Capabilities\n- Real-time quest tracking across all knights\n- Dragon activity monitoring with early warning\n- Kingdom health dashboards\n- Prophecy-based alerting (issues detected before they occur)")
	proj3.Set("tech_stack", []string{"Quartz", "Divination", "Real-time Scrying", "Prophecy Engine"})
	proj3.Set("links", []map[string]string{})
	proj3.Set("categories", []string{"observability", "monitoring"})
	proj3.Set("visibility", "public")
	proj3.Set("is_draft", false)
	proj3.Set("is_featured", true)
	proj3.Set("sort_order", 3)
	app.Save(proj3)

	proj4 := core.NewRecord(projColl)
	proj4.Set("title", "Holy Grail Search")
	proj4.Set("slug", "grail-search")
	proj4.Set("summary", "Distributed search system for locating sacred artifacts")
	proj4.Set("description", "Large-scale search infrastructure for the Quest for the Holy Grail.\n\n## Architecture\n- Distributed knight agents across Britannia\n- Fuzzy matching for grail-like objects\n- False positive handling (many cups, few grails)\n- Integration with Fisher King legacy systems")
	proj4.Set("tech_stack", []string{"Quest Framework", "Distributed Search", "Faith-based Routing"})
	proj4.Set("links", []map[string]string{})
	proj4.Set("categories", []string{"search", "distributed-systems"})
	proj4.Set("visibility", "public")
	proj4.Set("is_draft", false)
	proj4.Set("is_featured", false)
	proj4.Set("sort_order", 4)
	app.Save(proj4)

	// Create education
	eduColl, _ := app.FindCollectionByNameOrId("education")

	edu1 := core.NewRecord(eduColl)
	edu1.Set("institution", "Druids of Stonehenge")
	edu1.Set("degree", "Master of Mystical Arts")
	edu1.Set("field", "Applied Enchantment")
	edu1.Set("start_date", "0380-01-01")
	edu1.Set("end_date", "0400-12-31")
	edu1.Set("description", "Comprehensive training in prophecy, transmutation, and astronomical computing. Thesis: 'Optimal Stone Placement for Solstice Calculations'.")
	edu1.Set("visibility", "public")
	edu1.Set("is_draft", false)
	edu1.Set("sort_order", 1)
	app.Save(edu1)

	edu2 := core.NewRecord(eduColl)
	edu2.Set("institution", "Bardic College of Wales")
	edu2.Set("degree", "Bachelor of Incantations")
	edu2.Set("field", "Verbal Spell Interfaces")
	edu2.Set("start_date", "0370-01-01")
	edu2.Set("end_date", "0379-12-31")
	edu2.Set("description", "Focus on voice-activated magic, Latin incantations, and the emerging field of spoken-word programming.")
	edu2.Set("visibility", "public")
	edu2.Set("is_draft", false)
	edu2.Set("sort_order", 2)
	app.Save(edu2)

	// Create certifications
	certColl, _ := app.FindCollectionByNameOrId("certifications")

	cert1 := core.NewRecord(certColl)
	cert1.Set("name", "Certified Prophecy Professional (CPP)")
	cert1.Set("issuer", "International Seers Guild")
	cert1.Set("visibility", "public")
	cert1.Set("is_draft", false)
	cert1.Set("sort_order", 1)
	app.Save(cert1)

	cert2 := core.NewRecord(certColl)
	cert2.Set("name", "Licensed Shapeshifter")
	cert2.Set("issuer", "Britannia Magical Registry")
	cert2.Set("visibility", "public")
	cert2.Set("is_draft", false)
	cert2.Set("sort_order", 2)
	app.Save(cert2)

	cert3 := core.NewRecord(certColl)
	cert3.Set("name", "Dragon Handling Safety Certification")
	cert3.Set("issuer", "Camelot Health & Safety")
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
		{"Prophecy", "Core Magic", "expert", 1},
		{"Transmutation", "Core Magic", "expert", 2},
		{"Enchantment", "Core Magic", "proficient", 3},
		{"Shapeshifting", "Core Magic", "proficient", 4},
		{"Latin", "Languages", "expert", 5},
		{"Old Welsh", "Languages", "expert", 6},
		{"Python", "Languages", "familiar", 7},
		{"Crystal Ball Scrying", "Observability", "expert", 8},
		{"Tea Leaf Reading", "Observability", "proficient", 9},
		{"Distributed Systems", "Architecture", "expert", 10},
		{"Consensus Algorithms", "Architecture", "proficient", 11},
		{"Stone-based Auth", "Security", "expert", 12},
		{"Dragon Handling", "Operations", "proficient", 13},
		{"Mentorship", "Leadership", "expert", 14},
		{"Royal Advising", "Leadership", "expert", 15},
		{"Quest Planning", "Project Management", "expert", 16},
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

	// Create view
	viewsColl, _ := app.FindCollectionByNameOrId("views")

	view := core.NewRecord(viewsColl)
	view.Set("name", "For Kingdoms")
	view.Set("slug", "kingdoms")
	view.Set("description", "Curated view for royal courts seeking technical advisors")
	view.Set("visibility", "public")
	view.Set("hero_headline", "Chief Wizard & Staff Engineer")
	view.Set("hero_summary", "Centuries of experience guiding kingdoms through digital transformation. I specialise in prophecy-driven development, crystal ball observability, and mentoring future monarchs.")
	view.Set("cta_text", "Send Raven")
	view.Set("cta_url", "mailto:merlin@camelot.gov.uk")
	sectionsJSON, _ := json.Marshal([]map[string]interface{}{
		{"section": "experience", "enabled": true, "layout": "default"},
		{"section": "projects", "enabled": true, "layout": "grid-2"},
		{"section": "skills", "enabled": true, "layout": "grouped"},
		{"section": "certifications", "enabled": true, "layout": "grouped"},
		{"section": "education", "enabled": true, "layout": "default"},
	})
	view.Set("sections", string(sectionsJSON))
	view.Set("is_active", true)
	view.Set("is_default", true)
	app.Save(view)

	log.Println("Demo data seeded successfully!")
	log.Println("  Profile: Merlin Ambrosius")
	log.Println("  View: /kingdoms")
	log.Println("")
	log.Println("  To use development data instead, set SEED_DATA=dev")

	return nil
}

// seedDevData seeds development data (Jedidiah Esposito profile)
func seedDevData(app *pocketbase.PocketBase) error {
	// Check if already seeded
	count, _ := app.CountRecords("profile")
	if count > 0 {
		return nil
	}

	log.Println("Seeding development data (Jedidiah Esposito)...")

	// Create PocketBase superuser for /_/ admin access (dev mode only)
	superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err == nil && superusers != nil {
		superuserCount, _ := app.CountRecords(core.CollectionNameSuperusers)
		if superuserCount == 0 {
			su := core.NewRecord(superusers)
			su.SetEmail(getSeedAdminEmail("admin@localhost.dev"))
			su.SetPassword("admin123")
			if err := app.Save(su); err != nil {
				log.Printf("Warning: Could not create dev superuser: %v", err)
			} else {
				log.Println("")
				log.Println("========================================")
				log.Println("  DEV MODE: PocketBase Admin Created")
				log.Println("========================================")
				log.Println("  URL:      http://localhost:8090/_/")
				log.Printf("  Email:    %s\n", su.Email())
				log.Println("  Password: admin123")
				log.Println("========================================")
				log.Println("")
			}
		}
	}

	// Create default user for frontend admin
	if err := createDefaultUser(app); err != nil {
		log.Printf("Warning: %v", err)
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

	// Project 1: Facet
	proj1 := core.NewRecord(projColl)
	proj1.Set("title", "Facet")
	proj1.Set("slug", "facet")
	proj1.Set("summary", "Personal Publishing Platform | Active Development")
	proj1.Set("description", "Designing and building a full-stack, content-driven web application.\n\n## Key Contributions\n- Own front-end architecture, routing, and UI implementation using SvelteKit\n- Built admin interfaces for creating and managing content\n- Implemented server-side data loading and structured routing patterns\n- Maintain clear architecture, design, and roadmap documentation")
	proj1.Set("tech_stack", []string{"SvelteKit", "JavaScript", "TypeScript", "Docker", "REST APIs"})
	proj1.Set("links", []map[string]string{
		{"type": "github", "url": "https://github.com/jesposito/Facet"},
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
	proj2.Set("tech_stack", []string{"Power Automate", "Slack Canvas", "Power BI", "Slack Workflows"})
	proj2.Set("links", []map[string]string{})
	proj2.Set("categories", []string{"enterprise", "automation"})
	proj2.Set("visibility", "public")
	proj2.Set("is_draft", false)
	proj2.Set("is_featured", true)
	proj2.Set("sort_order", 2)
	app.Save(proj2)

	// Project 3: Custom MCP Server Ecosystem
	proj3 := core.NewRecord(projColl)
	proj3.Set("title", "Custom MCP Server Ecosystem")
	proj3.Set("slug", "mcp-servers")
	proj3.Set("summary", "Multiple Model Context Protocol servers for system management and automation")
	proj3.Set("description", "Built multiple Model Context Protocol servers for system management and automation.\n\n## Servers\n- **Unraid Server Manager**: Container orchestration, filesystem operations, backup automation\n- **n8n Workflow Controller**: Workflow CRUD operations, execution monitoring, integration management\n- **System Monitoring Suite**: Cross-platform system info and diagnostic commands (Ubuntu, Pop-OS, Windows)")
	proj3.Set("tech_stack", []string{"Python", "FastMCP", "Docker", "REST APIs", "SSH", "PowerShell", "systemd"})
	proj3.Set("links", []map[string]string{})
	proj3.Set("categories", []string{"automation", "ai-tooling"})
	proj3.Set("visibility", "public")
	proj3.Set("is_draft", false)
	proj3.Set("is_featured", true)
	proj3.Set("sort_order", 3)
	app.Save(proj3)

	// Project 4: Agentic Workflow Automation
	proj4 := core.NewRecord(projColl)
	proj4.Set("title", "Agentic Workflow Automation")
	proj4.Set("slug", "agentic-workflows")
	proj4.Set("summary", "Multi-agent workflows for complex decision-making and data processing using n8n")
	proj4.Set("description", "Multi-agent workflows for complex decision-making and data processing using n8n.\n\n## Implementations\n- Property investment analysis workflows (NZ real estate data → LLM analysis → structured reports)\n- Document processing pipelines with RAG patterns\n- Multi-step reasoning flows with sub-agent delegation\n- API orchestration across multiple services (Anthropic, OpenAI, Google Drive, Gmail)")
	proj4.Set("tech_stack", []string{"n8n", "Prompt Engineering", "RAG", "APIs", "LLMs"})
	proj4.Set("links", []map[string]string{})
	proj4.Set("categories", []string{"automation", "ai-tooling"})
	proj4.Set("visibility", "public")
	proj4.Set("is_draft", false)
	proj4.Set("is_featured", false)
	proj4.Set("sort_order", 4)
	app.Save(proj4)

	// Project 5: Mrs. Doubtfire Multi-Modal Voice Assistant
	proj5 := core.NewRecord(projColl)
	proj5.Set("title", "Mrs. Doubtfire Voice Assistant")
	proj5.Set("slug", "mrs-doubtfire")
	proj5.Set("summary", "Production voice assistant with natural language understanding and multi-modal capabilities")
	proj5.Set("description", "Designed and implemented production voice assistant with natural language understanding.\n\n## Technical Implementation\n- LLM backend with hot-swappable providers (Claude Sonnet/Haiku, GPT-4o) based on task complexity\n- Custom MCP servers (Python/FastMCP) for system control, home automation, and data retrieval\n- ElevenLabs TTS with custom voice personality training (Scottish accent, conversational tone)\n- Wyoming protocol integration for distributed voice satellites (ESP32-S3-BOX hardware)\n- Home Assistant integration via REST API and WebSocket\n- Docker containerization for all services\n- Fallback logic and graceful degradation patterns\n- Memory management across conversation sessions")
	proj5.Set("tech_stack", []string{"LLMs", "MCP", "ElevenLabs", "Wyoming Protocol", "Home Assistant", "Docker"})
	proj5.Set("links", []map[string]string{})
	proj5.Set("categories", []string{"voice-ui", "home-automation", "ai-tooling"})
	proj5.Set("visibility", "public")
	proj5.Set("is_draft", false)
	proj5.Set("is_featured", true)
	proj5.Set("sort_order", 5)
	app.Save(proj5)

	// Project 6: Production Home Infrastructure
	proj6 := core.NewRecord(projColl)
	proj6.Set("title", "Production Home Infrastructure")
	proj6.Set("slug", "home-infrastructure")
	proj6.Set("summary", "Comprehensive self-hosted infrastructure demonstrating enterprise-grade architecture patterns")
	proj6.Set("description", "Built and maintain comprehensive self-hosted infrastructure demonstrating enterprise-grade architecture patterns.\n\n## Infrastructure\n- 20+ Docker services orchestrated on Unraid (Immich, Rocket.Chat, Emby, Home Assistant, n8n, AdGuard)\n- Cloudflare Zero Trust tunnels for secure external access without exposed ports\n- Nginx reverse proxy with SSL/TLS termination and domain routing\n- Tailscale mesh VPN for encrypted peer-to-peer connectivity\n- Federated identity management with OAuth integration across services\n- Custom DNS infrastructure (AdGuard + Unbound) with network-wide ad blocking\n- VLAN segmentation for IoT device isolation\n- ZFS storage management on Ubuntu with automated snapshots\n- Automated backup systems with 3-day retention and restore capabilities")
	proj6.Set("tech_stack", []string{"Docker", "Unraid", "Cloudflare", "Nginx", "Tailscale", "OAuth", "ZFS"})
	proj6.Set("links", []map[string]string{})
	proj6.Set("categories", []string{"infrastructure", "self-hosted"})
	proj6.Set("visibility", "public")
	proj6.Set("is_draft", false)
	proj6.Set("is_featured", false)
	proj6.Set("sort_order", 6)
	app.Save(proj6)

	// Project 7: The Foodie Alexa Skill
	proj7 := core.NewRecord(projColl)
	proj7.Set("title", "The Foodie: A Conversational Alexa Skill")
	proj7.Set("slug", "the-foodie")
	proj7.Set("summary", "Conversational Alexa skill with associated training materials")
	proj7.Set("description", "Worked extensively on both the design of the VUI (voice-user interface) for the skill and the associated eLearning course and in-person workshops on Conversational Design for Voice.\n\nThis project was associated with Amazon and included collaboration on technical enablement materials.")
	proj7.Set("tech_stack", []string{"Alexa Skills", "Voice UI", "Conversational Design"})
	proj7.Set("links", []map[string]string{})
	proj7.Set("categories", []string{"voice-ui", "developer-education"})
	proj7.Set("visibility", "public")
	proj7.Set("is_draft", false)
	proj7.Set("is_featured", false)
	proj7.Set("sort_order", 7)
	app.Save(proj7)

	// Project 8: Alexa Developer Blogs
	proj8 := core.NewRecord(projColl)
	proj8.Set("title", "Alexa Developer Technical Blogs")
	proj8.Set("slug", "alexa-blogs")
	proj8.Set("summary", "Technical blog posts for the Alexa Developer Marketing team")
	proj8.Set("description", "Published technical content for the Alexa Developer Marketing team covering voice UI design, skill development, and developer education.")
	proj8.Set("tech_stack", []string{"Technical Writing", "Voice UI", "Developer Education"})
	proj8.Set("links", []map[string]string{
		{"type": "website", "url": "https://developer.amazon.com/search#q=jedidiah%20esposito&t=Alexa&sort=relevancy"},
	})
	proj8.Set("categories", []string{"technical-writing", "developer-education"})
	proj8.Set("visibility", "public")
	proj8.Set("is_draft", false)
	proj8.Set("is_featured", false)
	proj8.Set("sort_order", 8)
	app.Save(proj8)

	// Create education
	eduColl, _ := app.FindCollectionByNameOrId("education")

	// Education 1: Colorado Technical University
	edu1 := core.NewRecord(eduColl)
	edu1.Set("institution", "Colorado Technical University")
	edu1.Set("degree", "Master of Science")
	edu1.Set("field", "Information Technology Management")
	edu1.Set("start_date", "2005-01-01")
	edu1.Set("end_date", "2007-12-31")
	edu1.Set("description", "Courses in Network Administration, Project Management Processes, Project Planning, Execution, and Closure, Schedule and Cost Control Techniques, Contracting, and Procurement. Focus on Learning Experience Design and Technical Enablement.")
	edu1.Set("visibility", "public")
	edu1.Set("is_draft", false)
	edu1.Set("sort_order", 1)
	app.Save(edu1)

	// Education 2: Northern Arizona University
	edu2 := core.NewRecord(eduColl)
	edu2.Set("institution", "Northern Arizona University")
	edu2.Set("degree", "Bachelor of Science")
	edu2.Set("field", "Education")
	edu2.Set("start_date", "2001-01-01")
	edu2.Set("end_date", "2005-12-31")
	edu2.Set("description", "Courses in Curriculum Development, Contemporary Developments in Education, Evaluation of Learning, Technology in the Classroom, School and Society, and Educational Psychology. Focus on Learning Experience Design.")
	edu2.Set("visibility", "public")
	edu2.Set("is_draft", false)
	edu2.Set("sort_order", 2)
	app.Save(edu2)

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
		{"Python", "Technologies", "proficient", 6},
		{"Content-driven Applications", "Specialisations", "expert", 7},
		{"Admin-style Interfaces", "Specialisations", "expert", 8},
		{"API Integration", "Specialisations", "proficient", 9},
		{"Data-driven UI", "Specialisations", "proficient", 10},
		{"Docker", "Infrastructure", "proficient", 11},
		{"Containerised Services", "Infrastructure", "proficient", 12},
		{"Linux", "Infrastructure", "proficient", 13},
		{"Nginx", "Infrastructure", "proficient", 14},
		{"Automation", "Tooling", "proficient", 15},
		{"Workflow Tooling", "Tooling", "proficient", 16},
		{"n8n", "Tooling", "proficient", 17},
		{"Model Context Protocol (MCP)", "AI & Automation", "proficient", 18},
		{"Large Language Models (LLM)", "AI & Automation", "proficient", 19},
		{"Prompt Engineering", "AI & Automation", "proficient", 20},
		{"Agentic Workflows", "AI & Automation", "proficient", 21},
		{"RAG", "AI & Automation", "familiar", 22},
		{"Voice User Interface Design", "Specialisations", "expert", 23},
		{"Systems Integration", "Specialisations", "proficient", 24},
		{"Agile Delivery", "Leadership", "expert", 25},
		{"Technical Leadership", "Leadership", "expert", 26},
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
		{"section": "education", "enabled": true, "layout": "default"},
	})
	view.Set("sections", string(sectionsJSON))
	view.Set("is_active", true)
	view.Set("is_default", true)
	app.Save(view)

	log.Println("Development data seeded successfully!")
	log.Println("  Profile: Jedidiah Esposito")
	log.Println("  View: /front-end-lead")

	return nil
}

// createDefaultUser creates the frontend admin user
func createDefaultUser(app *pocketbase.PocketBase) error {
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}

	userCount, _ := app.CountRecords("users")
	if userCount > 0 {
		return nil
	}

	admin := core.NewRecord(users)
	admin.Set("email", getSeedAdminEmail("admin@example.com"))
	admin.Set("name", "Admin")
	admin.Set("is_admin", true)
	admin.SetPassword("changeme123")
	if err := app.Save(admin); err != nil {
		return err
	}

	log.Println("Created default frontend admin account:")
	log.Printf("  Email: %s\n", admin.Email())
	log.Println("  Password: changeme123")
	log.Println("  ⚠️  CHANGE THIS PASSWORD IMMEDIATELY!")

	return nil
}

// getSeedAdminEmail chooses the admin email for seeded accounts:
// 1) first non-empty entry in ADMIN_EMAILS (comma-separated)
// 2) DEV_ADMIN_EMAIL env var
// 3) provided fallback
func getSeedAdminEmail(fallback string) string {
	if raw := strings.TrimSpace(os.Getenv("ADMIN_EMAILS")); raw != "" {
		parts := strings.Split(raw, ",")
		for _, part := range parts {
			email := strings.TrimSpace(part)
			if email != "" {
				return email
			}
		}
	}

	if email := strings.TrimSpace(os.Getenv("DEV_ADMIN_EMAIL")); email != "" {
		return email
	}

	return fallback
}
