package hooks

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/crypto/bcrypt"
)

// RegisterSeedHook seeds data on first run
// Environment variable SEED_DATA controls behavior:
//   - "dev": Seeds development data (Jedidiah Esposito) - for development/testing
//   - "minimal" or unset: Seeds ONLY default admin account (production default)
//
// Demo data (Merlin Ambrosius) is available via admin UI toggle, not auto-seeded.
func RegisterSeedHook(app *pocketbase.PocketBase) {
	seedMode := os.Getenv("SEED_DATA")

	// Default to minimal if not set (production default = create admin account)
	if seedMode == "" {
		seedMode = "minimal"
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

// seedDemoData seeds hilarious Doctor Who-themed demo data showcasing all features
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

// createDemoUser creates the demo user account (demo@facet.example)
// Uses direct SQL INSERT because app.Save() fails silently in OnServe hooks (PocketBase v0.23 context issue)
func createDemoUser(app *pocketbase.PocketBase) error {
	const demoEmail = "demo@facet.example"
	const demoPassword = "demo123"

	// Check if demo user already exists
	_, err := app.FindAuthRecordByEmail("users", demoEmail)
	if err == nil {
		// Demo user already exists
		return nil
	}

	// Generate ID (PocketBase pattern: 'r' + 14 random hex chars)
	randomBytes := make([]byte, 7)
	if _, err := rand.Read(randomBytes); err != nil {
		return fmt.Errorf("generate ID: %w", err)
	}
	id := "r" + hex.EncodeToString(randomBytes)

	// Hash password with bcrypt
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// Generate tokenKey (PocketBase uses 50 random bytes)
	tokenBytes := make([]byte, 50)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("generate token: %w", err)
	}
	tokenKey := hex.EncodeToString(tokenBytes)

	// Direct SQL INSERT (workaround for OnServe hook save issue)
	query := `INSERT INTO users (id, email, name, password, tokenKey, verified, emailVisibility, avatar)
	          VALUES ({:id}, {:email}, {:name}, {:password}, {:tokenKey}, 1, 0, '')`

	_, err = app.DB().NewQuery(query).Bind(dbx.Params{
		"id":       id,
		"email":    demoEmail,
		"name":     "The Doctor",
		"password": string(passwordHash),
		"tokenKey": tokenKey,
	}).Execute()

	if err != nil {
		return fmt.Errorf("insert demo user: %w", err)
	}

	log.Println("Created demo account:")
	log.Printf("  Email: %s\n", demoEmail)
	log.Printf("  Password: %s\n", demoPassword)
	log.Println("  This is a read-only demo account")
	log.Println("")

	return nil
}

// createDefaultUser creates the frontend admin user
// Uses direct SQL INSERT because app.Save() silently fails in OnServe hooks (PB v0.23 issue)
func createDefaultUser(app *pocketbase.PocketBase) error {
	userCount, _ := app.CountRecords("users")
	if userCount > 0 {
		return nil
	}

	email := getSeedAdminEmail("admin@example.com")

	// Generate ID (PocketBase pattern: 'r' + 14 random hex chars)
	randomBytes := make([]byte, 7)
	if _, err := rand.Read(randomBytes); err != nil {
		return fmt.Errorf("generate ID: %w", err)
	}
	id := "r" + hex.EncodeToString(randomBytes)

	// Hash password with bcrypt
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("changeme123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// Generate tokenKey (PocketBase uses 50 random bytes)
	tokenBytes := make([]byte, 50)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("generate token: %w", err)
	}
	tokenKey := hex.EncodeToString(tokenBytes)

	// Direct SQL INSERT
	query := `INSERT INTO users (id, email, name, password, tokenKey, verified, emailVisibility, avatar, password_changed_from_default)
	          VALUES ({:id}, {:email}, {:name}, {:password}, {:tokenKey}, 1, 0, '', 0)`

	_, err = app.DB().NewQuery(query).Bind(dbx.Params{
		"id":       id,
		"email":    email,
		"name":     "Admin",
		"password": string(passwordHash),
		"tokenKey": tokenKey,
	}).Execute()

	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	log.Println("Created default frontend admin account:")
	log.Printf("  Email: %s\n", email)
	log.Println("  Password: changeme123")
	log.Println("  ⚠️  You will be prompted to change this password on first login.")
	log.Println("")
	log.Printf("  NOTE: This email (%s) must be in ADMIN_EMAILS to access /admin", email)

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

// loadDemoDataForUser loads The Doctor's demo data WITHOUT creating a separate demo user
// This is used when the admin toggles demo mode ON - it loads sample data into their account
func loadDemoDataForUser(app *pocketbase.PocketBase) error {
	log.Println("Loading demo data for current user...")

	// This is a copy of seedDemoData but WITHOUT the createDemoUser() call
	// Create profile
	profileColl, err := app.FindCollectionByNameOrId("profile")
	if err != nil {
		return err
	}

	profile := core.NewRecord(profileColl)
	profile.Set("name", "The Doctor")
	profile.Set("headline", "Time Lord | Madman with a Box | 900+ Years Experience")
	profile.Set("location", "The TARDIS (Currently Parked Illegally in Central London)")
	profile.Set("summary", "Results-driven problem solver with extensive experience across time, space, and the occasional parallel dimension. Specializing in crisis management, impossible situations, and defeating universe-threatening entities before teatime.\n\nI've saved Earth 47 times (stopped counting after the Cybermen incident), prevented 12 timeline collapses, and successfully debugged a sentient AI that was literally trying to delete reality. My approach combines rapid prototyping, improvisation, and an alarming amount of running.\n\nCurrently seeking interesting challenges. Must involve some form of danger. Bonus points if it's never been done before. References available from UNIT, Torchwood, and various grateful civilizations (please don't contact Gallifrey, we're not on speaking terms).")
	profile.Set("contact_email", "definitely-not-a-timelord@gmail.com")
	profile.Set("contact_links", []map[string]string{
		{"type": "github", "url": "https://github.com/madman-with-a-box"},
		{"type": "linkedin", "url": "https://linkedin.com/in/the-doctor-900-years"},
		{"type": "website", "url": "https://police-box-exterior.tardis"},
	})
	profile.Set("visibility", "public")
	if err := app.Save(profile); err != nil {
		return err
	}

	// Create experience
	expColl, _ := app.FindCollectionByNameOrId("experience")

	exp1 := core.NewRecord(expColl)
	exp1.Set("company", "UNIT (Unified Intelligence Taskforce)")
	exp1.Set("title", "Scientific Advisor")
	exp1.Set("location", "Geneva, Switzerland (Remote - Very Remote)")
	exp1.Set("start_date", "1970-01-01")
	exp1.Set("description", "Consulting role providing expertise on extraterrestrial threats, temporal anomalies, and why the coffee machine is actually a Zygon in disguise. Worked closely with military personnel who insisted on solving everything with explosives (I usually had better ideas).")
	exp1.Set("bullets", []string{
		"Defeated 200+ alien invasions using only a sonic screwdriver and excessive running",
		"Established real-time threat detection system (I just listen for screaming)",
		"Prevented nuclear war 3 times (Tuesdays are always tricky)",
		"Trained military personnel in non-violent conflict resolution (they mostly ignored this)",
		"Maintained 99.99% uptime for Earth's continued existence (that 0.01% was a rough week)",
	})
	exp1.Set("skills", []string{"Crisis Management", "Alien Technology", "Improvisation", "Running Very Fast", "Temporal Mechanics"})
	exp1.Set("visibility", "public")
	exp1.Set("is_draft", false)
	exp1.Set("sort_order", 1)
	app.Save(exp1)

	exp2 := core.NewRecord(expColl)
	exp2.Set("company", "Totally Normal Software Company Inc.")
	exp2.Set("title", "Senior Software Engineer (Definitely Not a Time Lord)")
	exp2.Set("location", "Earth, Sol System, Mutter's Spiral - wait, I mean London")
	exp2.Set("start_date", "2020-01-01")
	exp2.Set("end_date", "2023-12-31")
	exp2.Set("description", "Standard software engineering role. Nothing unusual. Just regular code. Absolutely no time travel involved. Built scalable web applications using modern frameworks (and maybe one or two that haven't been invented yet, but that's fine, right?).")
	exp2.Set("bullets", []string{
		"Architected microservices with 99.99% uptime (would've been 100% but causality is tricky)",
		"Reduced API response time to -3ms (fixed in code review after teammate pointed out time can't go backwards)",
		"Implemented real-time data synchronization across multiple timezones (and timelines, but we don't talk about that)",
		"Mentored junior developers (tried not to mention my 900 years of debugging experience)",
		"Fixed legacy codebase from 1963 - I mean, that's just a typo, obviously meant 2016",
		"On-call rotation for production incidents (I'm very good at predicting outages before they happen)",
	})
	exp2.Set("skills", []string{"JavaScript", "Python", "Go", "React", "Kubernetes", "Time Travel Debugging (wait, delete this)"})
	exp2.Set("visibility", "public")
	exp2.Set("is_draft", false)
	exp2.Set("sort_order", 2)
	app.Save(exp2)

	exp3 := core.NewRecord(expColl)
	exp3.Set("company", "Gallifrey Temporal Engineering")
	exp3.Set("title", "Systems Architect (Classified)")
	exp3.Set("location", "Gallifrey, Constellation of Kasterborous (No Longer Exists - It's Complicated)")
	exp3.Set("start_date", "1000-01-01")
	exp3.Set("end_date", "1963-11-23")
	exp3.Set("description", "Early career role maintaining critical temporal infrastructure. Left under disputed circumstances (they called it 'stealing,' I called it 'borrowing indefinitely'). Gained extensive experience with legacy systems - 900+ year old codebase, zero documentation, and I was the one who wrote most of it.")
	exp3.Set("bullets", []string{
		"Maintained time-series database (literally a database OF time)",
		"Debugged causality loops in production (fixed by future me, broken by past me)",
		"Implemented TARDIS navigation system (still working on making it accurate)",
		"Failed final exam twice before passing with 51% (apparently saving the universe doesn't count for extra credit)",
		"'Borrowed' a TARDIS for testing purposes (still have it, runs great)",
		"Left on good terms (they're still angry, but I'm sure they'll get over it in a few centuries)",
	})
	exp3.Set("skills", []string{"Temporal Mechanics", "TARDIS Engineering", "Paradox Resolution", "Academic Probation"})
	exp3.Set("visibility", "public")
	exp3.Set("is_draft", false)
	exp3.Set("sort_order", 3)
	app.Save(exp3)

	// Create projects
	projColl, _ := app.FindCollectionByNameOrId("projects")

	proj1 := core.NewRecord(projColl)
	proj1.Set("title", "TARDIS Operating System")
	proj1.Set("slug", "tardis-os")
	proj1.Set("summary", "Real-time temporal navigation system with chaotic-good architecture")
	proj1.Set("description", "Advanced navigation system for 5-dimensional travel through time and space. Built on the principle that if it looks dangerous and makes weird noises, it's probably working correctly.\n\n## Key Features\n- Chameleon Circuit (currently stuck as 1960s police box - known issue)\n- Temporal Grace field (security feature - sometimes works)\n- Dimensionally transcendental architecture (bigger on the inside™)\n- Artron energy core with percussive maintenance interface\n- Voice-activated controls (responds to yelling)\n\n## Performance\n- Navigation accuracy: ~30% (getting better!)\n- Dimensions supported: 5 (technically infinite but that's complicated)\n- Time Travel Range: All of it\n- Crashes per century: 247 (mostly my fault)")
	proj1.Set("tech_stack", []string{"Artron Energy", "Temporal Mechanics", "Block Transfer Computation", "Percussive Maintenance", "Wibbly-Wobbly Timey-Wimey Stuff"})
	proj1.Set("links", []map[string]string{
		{"type": "documentation", "url": "https://tardis.wiki/Type_40"},
	})
	proj1.Set("categories", []string{"hardware", "transportation", "time-travel"})
	proj1.Set("visibility", "public")
	proj1.Set("is_draft", false)
	proj1.Set("is_featured", true)
	proj1.Set("sort_order", 1)
	app.Save(proj1)

	proj2 := core.NewRecord(projColl)
	proj2.Set("title", "Sonic Screwdriver API v47")
	proj2.Set("slug", "sonic-screwdriver")
	proj2.Set("summary", "Universal tool API with 10,000+ endpoints (and counting)")
	proj2.Set("description", "Swiss-army-knife REST API that does literally everything except wood. Because it doesn't do wood. I should really fix that.\n\n## Features\n- 10,000+ endpoints (I keep adding more)\n- Lock manipulation, medical scanning, technical analysis, and coffee making\n- Zero documentation (it's sonic, you just sort of... know)\n- Authentication: None (it just knows it's me)\n- Rate limiting: Unlimited (security concern noted)\n- Versioning: Currently on v47 (lost track of v22-v31)\n\n## Known Issues\n- Doesn't work on wood (this is a feature, not a bug)\n- Sometimes opens wrong type of door\n- Makes annoying buzzing sound\n- May explode if used incorrectly (please don't use incorrectly)")
	proj2.Set("tech_stack", []string{"Sonic Technology", "Artron Energy", "Questionable Design Decisions", "Pure Stubbornness"})
	proj2.Set("links", []map[string]string{
		{"type": "github", "url": "https://github.com/madman-with-a-box/sonic-screwdriver"},
	})
	proj2.Set("categories", []string{"api", "tools", "hardware"})
	proj2.Set("visibility", "public")
	proj2.Set("is_draft", false)
	proj2.Set("is_featured", true)
	proj2.Set("sort_order", 2)
	app.Save(proj2)

	proj3 := core.NewRecord(projColl)
	proj3.Set("title", "react-timeseries-visualizer")
	proj3.Set("slug", "react-timeseries")
	proj3.Set("summary", "React component library for visualizing time-series data")
	proj3.Set("description", "Professional, production-ready React components for time-series data visualization. Built with TypeScript, fully tested, completely normal.\n\n## Features\n- Real-time data streaming\n- Multiple chart types (line, bar, scatter, temporal-paradox)\n- Responsive design\n- TypeScript support\n- Zero dependencies on temporal mechanics (that's a joke)\n\n## Installation\n```bash\nnpm install react-timeseries-visualizer\n# Note: May cause timeline issues if used incorrectly\n# That's also a joke. Probably.\n```\n\n## Example Usage\n```tsx\nimport { TimeSeriesChart } from 'react-timeseries-visualizer';\n\n<TimeSeriesChart \n  data={data}\n  // Don't set timeRange to negative values\n  // I learned this the hard way\n/>\n```")
	proj3.Set("tech_stack", []string{"React", "TypeScript", "D3.js", "Vite", "Jest"})
	proj3.Set("links", []map[string]string{
		{"type": "github", "url": "https://github.com/madman-with-a-box/react-timeseries-visualizer"},
		{"type": "npm", "url": "https://npmjs.com/package/react-timeseries-visualizer"},
	})
	proj3.Set("categories", []string{"frontend", "react", "data-visualization"})
	proj3.Set("visibility", "public")
	proj3.Set("is_draft", false)
	proj3.Set("is_featured", true)
	proj3.Set("sort_order", 3)
	app.Save(proj3)

	proj4 := core.NewRecord(projColl)
	proj4.Set("title", "Companion Management System")
	proj4.Set("slug", "companion-mgmt")
	proj4.Set("summary", "CRM for tracking travel companions and their inevitable questions")
	proj4.Set("description", "Specialized CRM system for managing companions across space and time. Tracks important details like allergies to Artron energy, tendency to wander off, and how many times they've asked 'What's that?'\n\n## Features\n- Companion onboarding (very thorough 'don't touch anything' briefing)\n- Real-time location tracking (they ALWAYS wander off)\n- Danger level monitoring per companion\n- Automatic 'I'll explain later' response templates\n- Historical data from previous companions (I miss you all)\n- Emergency extraction protocols\n\n## Statistics\n- Total companions managed: 47\n- Average questions per companion per day: 147\n- Successful returns to correct time period: 98%\n- Times said 'Run!': Too many to count")
	proj4.Set("tech_stack", []string{"TARDIS Integration", "Temporal GPS", "Psychic Paper", "Lots of Patience"})
	proj4.Set("links", []map[string]string{})
	proj4.Set("categories", []string{"crm", "people-management", "time-travel"})
	proj4.Set("visibility", "public")
	proj4.Set("is_draft", false)
	proj4.Set("is_featured", false)
	proj4.Set("sort_order", 4)
	app.Save(proj4)

	// Create education
	eduColl, _ := app.FindCollectionByNameOrId("education")

	edu1 := core.NewRecord(eduColl)
	edu1.Set("institution", "Time Lord Academy, Gallifrey")
	edu1.Set("degree", "Doctorate in... everything? (It's complicated)")
	edu1.Set("field", "Temporal Engineering")
	edu1.Set("start_date", "1523-01-01")
	edu1.Set("end_date", "1963-11-23")
	edu1.Set("description", "Comprehensive training in temporal mechanics, TARDIS operation, and the Laws of Time (which I may have bent a few times). Took the exam three times - first two were just practice, obviously. Final grade: 51% (passing is 51%, so technically perfect). Thesis: 'Why Fixed Points in Time Are Boring and Should Be Ignored' (thesis rejected but I stand by it).")
	edu1.Set("visibility", "public")
	edu1.Set("is_draft", false)
	edu1.Set("sort_order", 1)
	app.Save(edu1)

	edu2 := core.NewRecord(eduColl)
	edu2.Set("institution", "University of Life (YouTube)")
	edu2.Set("degree", "Self-Taught Software Engineering")
	edu2.Set("field", "Web Development")
	edu2.Set("start_date", "2019-01-01")
	edu2.Set("end_date", "2020-12-31")
	edu2.Set("description", "Intensive online coursework trying to fit in with modern developers. Completed freeCodeCamp, Udemy React course, and a very confusing tutorial about hooks (not the TARDIS kind). Constantly had to resist explaining that I'd already built similar systems 400 years ago.")
	edu2.Set("visibility", "public")
	edu2.Set("is_draft", false)
	edu2.Set("sort_order", 2)
	app.Save(edu2)

	// Create certifications
	certColl, _ := app.FindCollectionByNameOrId("certifications")

	cert1 := core.NewRecord(certColl)
	cert1.Set("name", "Licensed TARDIS Pilot")
	cert1.Set("issuer", "Gallifrey Department of Transportation")
	cert1.Set("issue_date", "1523-01-01")
	cert1.Set("expiry_date", "1723-01-01")
	cert1.Set("credential_id", "TL-TYPE40-STOLEN")
	cert1.Set("visibility", "public")
	cert1.Set("is_draft", false)
	cert1.Set("sort_order", 1)
	app.Save(cert1)

	cert2 := core.NewRecord(certColl)
	cert2.Set("name", "AWS Certified Solutions Architect")
	cert2.Set("issuer", "Amazon Web Services")
	cert2.Set("issue_date", "2024-01-15")
	cert2.Set("expiry_date", "2027-01-15")
	cert2.Set("credential_id", "AWS-CSA-DEFINITELY-REAL-12345")
	cert2.Set("credential_url", "https://aws.amazon.com/verification/")
	cert2.Set("visibility", "public")
	cert2.Set("is_draft", false)
	cert2.Set("sort_order", 2)
	app.Save(cert2)

	cert3 := core.NewRecord(certColl)
	cert3.Set("name", "Certified Hero (Self-Issued)")
	cert3.Set("issuer", "Me (but UNIT agrees)")
	cert3.Set("issue_date", "1970-01-01")
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
		// Programming Languages (trying to be normal)
		{"JavaScript", "Programming Languages", "expert", 1},
		{"TypeScript", "Programming Languages", "expert", 2},
		{"Python", "Programming Languages", "expert", 3},
		{"Go", "Programming Languages", "proficient", 4},
		{"Gallifreyan", "Programming Languages", "expert", 5},
		{"Binary (fluent speaker)", "Programming Languages", "expert", 6},

		// Technologies (can't hide the weirdness)
		{"React", "Frontend", "expert", 7},
		{"Node.js", "Backend", "expert", 8},
		{"Docker (bigger on inside!)", "DevOps", "expert", 9},
		{"Kubernetes", "DevOps", "proficient", 10},
		{"TARDIS Operating System", "DevOps", "expert", 11},

		// Time-Related Skills (oops)
		{"Time Travel", "Temporal Mechanics", "expert", 12},
		{"Paradox Resolution", "Temporal Mechanics", "expert", 13},
		{"Timeline Debugging", "Temporal Mechanics", "expert", 14},
		{"Causality Loop Prevention", "Temporal Mechanics", "proficient", 15},

		// Soft Skills
		{"Crisis Management", "Soft Skills", "expert", 16},
		{"Running Away Very Fast", "Soft Skills", "expert", 17},
		{"Improvisation", "Soft Skills", "expert", 18},
		{"Talking Very Quickly", "Soft Skills", "expert", 19},
		{"Companion Management", "Soft Skills", "proficient", 20},
		{"Paperwork", "Soft Skills", "novice", 21},
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
	view.Set("name", "Software Engineer Resume")
	view.Set("slug", "senior-engineer")
	view.Set("description", "Professional resume for software engineering positions. Please hire me. I promise I'm normal.")
	view.Set("visibility", "public")
	view.Set("hero_headline", "Senior Full-Stack Engineer | 900+ YOE | Proficient in Legacy Systems")
	view.Set("hero_summary", "Results-driven engineer with extensive experience in complex distributed systems, crisis management, and emergency hotfixes. Seeking senior individual contributor role. Strong preference for remote work (VERY remote). Available immediately. References available upon request (but please don't contact them).")
	view.Set("cta_text", "Download Resume")
	view.Set("cta_url", "mailto:definitely-not-a-timelord@gmail.com")
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

	// Add 2 blog posts
	postsColl, _ := app.FindCollectionByNameOrId("posts")

	post1 := core.NewRecord(postsColl)
	post1.Set("title", "That Time I Accidentally Invented Kubernetes")
	post1.Set("slug", "accidentally-invented-kubernetes")
	post1.Set("content", "So there I was, trying to orchestrate a fleet of TARDIS control systems across multiple timelines. Wrote a little bash script. Fast forward 2000 years, and apparently I'd invented container orchestration. The humans call it 'Kubernetes' now. I call it 'that thing that almost destroyed the fabric of spacetime because I forgot to set resource limits.'\n\nLesson learned: Always set your pod resource limits, people.")
	post1.Set("excerpt", "A cautionary tale about temporal container orchestration and why resource limits matter.")
	post1.Set("published_at", "2024-03-15 10:00:00.000Z")
	post1.Set("visibility", "public")
	post1.Set("is_draft", false)
	post1.Set("tags", []string{"DevOps", "Time Travel", "Kubernetes", "Mistakes Were Made"})
	app.Save(post1)

	post2 := core.NewRecord(postsColl)
	post2.Set("title", "5 Signs Your Codebase Might Be Sentient")
	post2.Set("slug", "sentient-codebase-signs")
	post2.Set("content", "1. It starts refusing pull requests\n2. It generates its own feature requests\n3. It threatens to delete itself if you don't buy it coffee\n4. It's actively plotting against you in Slack\n5. It's reading this blog post right now\n\nIf you're experiencing 3 or more of these symptoms, congratulations! You've created artificial intelligence. Please report to UNIT immediately for debriefing (and possibly containment).")
	post2.Set("excerpt", "A totally not based on true events guide to identifying rogue AI in your repositories.")
	post2.Set("published_at", "2024-02-28 14:30:00.000Z")
	post2.Set("visibility", "public")
	post2.Set("is_draft", false)
	post2.Set("tags", []string{"AI", "Humor", "Code Quality", "Help Me"})
	app.Save(post2)

	// Add 2 conference talks
	talksColl, _ := app.FindCollectionByNameOrId("talks")

	talk1 := core.NewRecord(talksColl)
	talk1.Set("title", "Debugging Across Dimensions: A Timey-Wimey Approach")
	talk1.Set("slug", "debugging-across-dimensions")
	talk1.Set("event", "Time Lords Conference 2024")
	talk1.Set("location", "Gallifrey (Virtual)")
	talk1.Set("date", "2024-04-20")
	talk1.Set("description", "Ever tried to debug code that hasn't been written yet? Or fix a bug that's propagating backwards through time? In this talk, I'll share my 900+ years of experience debugging temporal paradoxes, recursive timeline loops, and that one time I had to git revert the entire 16th century.\n\nKey Takeaways:\n- How to use console.log() across multiple timelines\n- The dangers of merge conflicts in temporal repositories  \n- Why 'Works On My TARDIS' is never an acceptable excuse")
	talk1.Set("recording_url", "https://youtube.com/watch?v=dQw4w9WgXcQ")
	talk1.Set("slides_url", "https://slides.com/the-doctor/debugging-dimensions")
	talk1.Set("visibility", "public")
	talk1.Set("is_draft", false)
	app.Save(talk1)

	talk2 := core.NewRecord(talksColl)
	talk2.Set("title", "Why Your Monitoring Is Terrible (And Mine Monitors The Future)")
	talk2.Set("slug", "future-monitoring")
	talk2.Set("event", "ObservabilityCon 2024")
	talk2.Set("location", "San Francisco, CA")
	talk2.Set("date", "2024-06-15")
	talk2.Set("description", "Your monitoring tells you what happened. My monitoring tells me what's GOING to happen. In this keynote, I'll demonstrate temporal observability patterns that let you fix incidents before they occur.\n\nWarning: Side effects may include paradoxes, existential dread, and knowing about that production outage three days before it happens but being unable to prevent it because it's a fixed point in time.")
	talk2.Set("visibility", "public")
	talk2.Set("is_draft", false)
	app.Save(talk2)

	// Add 1 award
	awardsColl, _ := app.FindCollectionByNameOrId("awards")

	award1 := core.NewRecord(awardsColl)
	award1.Set("title", "Most Creative Excuse For Missing A Deadline")
	award1.Set("issuer", "Galactic Developers Association")
	award1.Set("date", "2023-12-01")
	award1.Set("description", "Awarded for the excuse: 'Sorry, I was busy preventing the heat death of the universe. Also, time is relative.'")
	award1.Set("visibility", "public")
	app.Save(award1)

	log.Println("Demo data loaded successfully for current user!")
	return nil
}
