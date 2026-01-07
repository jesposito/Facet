package hooks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// RegisterDemoHandlers sets up the demo mode API endpoints
// Demo mode uses separate shadow tables (demo_*) that mirror the main tables
// When demo is ON, the UI reads from demo_* tables
// When demo is OFF, the UI reads from normal tables
func RegisterDemoHandlers(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		// GET /api/demo/status - Check if demo mode is enabled
		se.Router.GET("/api/demo/status", func(e *core.RequestEvent) error {
			authRecord := e.Auth
			if authRecord == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Authentication required",
				})
			}

			// Check if demo data exists
			demoProfile, _ := app.FindFirstRecordByFilter("demo_profile", "")

			return e.JSON(http.StatusOK, map[string]interface{}{
				"demo_mode": demoProfile != nil,
			})
		})

		// POST /api/demo/enable - Enable demo mode
		se.Router.POST("/api/demo/enable", func(e *core.RequestEvent) error {
			app.Logger().Info("========== DEMO ENABLE REQUEST ==========")
			authRecord := e.Auth
			if authRecord == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Authentication required",
				})
			}

			// Check if already in demo mode
			demoProfile, _ := app.FindFirstRecordByFilter("demo_profile", "")
			if demoProfile != nil {
				app.Logger().Info("Demo data already exists, clearing and reloading...")
				// Clear existing demo data first
				if err := clearDemoTables(app); err != nil {
					app.Logger().Error("Failed to clear existing demo data", "error", err)
					return e.JSON(http.StatusInternalServerError, map[string]string{
						"error": "Failed to clear existing demo data: " + err.Error(),
					})
				}
			}

			// Load The Doctor's demo data into demo_* tables
			if err := loadDemoDataIntoShadowTables(app); err != nil {
				app.Logger().Error("Failed to load demo data", "error", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to load demo data: " + err.Error(),
				})
			}

			app.Logger().Info("Demo mode enabled successfully")
			return e.JSON(http.StatusOK, map[string]string{
				"message": "Demo mode enabled",
			})
		})

		// POST /api/demo/restore - Disable demo mode
		se.Router.POST("/api/demo/restore", func(e *core.RequestEvent) error {
			app.Logger().Info("========== DEMO RESTORE REQUEST ==========")
			authRecord := e.Auth
			if authRecord == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Authentication required",
				})
			}

			// Check if in demo mode
			demoProfile, _ := app.FindFirstRecordByFilter("demo_profile", "")
			if demoProfile == nil {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "Not in demo mode",
				})
			}

			// Delete all demo data from demo_* tables
			if err := clearDemoTables(app); err != nil {
				app.Logger().Error("Failed to clear demo data", "error", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to clear demo data: " + err.Error(),
				})
			}

			app.Logger().Info("Demo mode disabled successfully")
			return e.JSON(http.StatusOK, map[string]string{
				"message": "Demo mode disabled",
			})
		})

		return se.Next()
	})
}

// clearDemoTables deletes all records from demo_* shadow tables
func clearDemoTables(app *pocketbase.PocketBase) error {
	tables := []string{
		"demo_profile", "demo_experience", "demo_projects", "demo_education",
		"demo_skills", "demo_certifications", "demo_posts", "demo_talks",
		"demo_awards", "demo_views", "demo_share_tokens", "demo_contact_methods",
	}

	for _, tableName := range tables {
		records, err := app.FindRecordsByFilter(tableName, "", "", 1000, 0)
		if err != nil {
			app.Logger().Warn("demo: failed to fetch records for clearing",
				"table", tableName,
				"error", err)
			continue
		}
		for _, record := range records {
			if err := app.Delete(record); err != nil {
				app.Logger().Error("demo: failed to delete record",
					"table", tableName,
					"record_id", record.Id,
					"error", err)
				return fmt.Errorf("failed to delete demo data from %s: %w", tableName, err)
			}
		}
	}

	return nil
}

// loadDemoAsset loads a file from the demo_assets directory and creates a filesystem.File
func loadDemoAsset(assetPath string) (*filesystem.File, error) {
	// Get the absolute path to the demo assets directory
	// Assuming the binary is run from the project root
	fullPath := filepath.Join("backend", "seeds", "demo_assets", assetPath)

	// Read the file
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	// Create a filesystem.File from the bytes
	filename := filepath.Base(assetPath)
	file, err := filesystem.NewFileFromBytes(data, filename)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// loadDemoDataIntoShadowTables loads The Doctor's demo data into demo_* shadow tables
// Adapted from seed.go seedDemoData() function to write to demo_* collections
func loadDemoDataIntoShadowTables(app *pocketbase.PocketBase) error {
	app.Logger().Info("Loading demo data into shadow tables...")

	// Create profile
	app.Logger().Info("Creating demo profile...")
	profileColl, err := app.FindCollectionByNameOrId("demo_profile")
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

	// Load and attach avatar
	if avatarFile, err := loadDemoAsset("profile/avatar.svg"); err == nil {
		profile.Set("avatar", avatarFile)
		app.Logger().Info("Attached avatar to demo profile")
	} else {
		app.Logger().Warn("Failed to load avatar asset", "error", err)
	}

	// Load and attach hero image
	if heroFile, err := loadDemoAsset("profile/hero.svg"); err == nil {
		profile.Set("hero_image", heroFile)
		app.Logger().Info("Attached hero image to demo profile")
	} else {
		app.Logger().Warn("Failed to load hero image asset", "error", err)
	}

	if err := app.Save(profile); err != nil {
		return err
	}

	// Create contact methods
	app.Logger().Info("Creating demo contact methods...")
	contactColl, _ := app.FindCollectionByNameOrId("demo_contact_methods")

	contacts := []struct {
		contactType     string
		value           string
		label           string
		protectionLevel string
		order           int
	}{
		{"github", "https://github.com/madman-with-a-box", "GitHub Profile", "none", 1},
		{"linkedin", "https://linkedin.com/in/the-doctor-900-years", "LinkedIn", "none", 2},
		{"twitter", "https://twitter.com/timeandspace", "Twitter/X", "none", 3},
		{"website", "https://police-box-exterior.tardis", "Personal Website", "none", 4},
		{"email", "definitely-not-a-timelord@gmail.com", "Email", "obfuscation", 5},
		{"custom", "https://calendly.com/the-doctor/save-the-universe", "Book a Consultation", "none", 6},
	}

	for _, c := range contacts {
		contact := core.NewRecord(contactColl)
		contact.Set("type", c.contactType)
		contact.Set("value", c.value)
		contact.Set("label", c.label)
		contact.Set("protection_level", c.protectionLevel)
		contact.Set("sort_order", c.order)
		contact.Set("is_primary", c.order == 1) // First one is primary
		if err := app.Save(contact); err != nil {
			return err
		}
	}

	// Create experience
	app.Logger().Info("Creating demo experience...")
	expColl, _ := app.FindCollectionByNameOrId("demo_experience")

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
	if err := app.Save(exp1); err != nil {
		return err
	}

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
	if err := app.Save(exp2); err != nil {
		return err
	}

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
	if err := app.Save(exp3); err != nil {
		return err
	}

	// Create projects
	app.Logger().Info("Creating demo projects...")
	projColl, _ := app.FindCollectionByNameOrId("demo_projects")

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

	// Load and attach cover image
	if coverFile, err := loadDemoAsset("projects/tardis-redesign.svg"); err == nil {
		proj1.Set("cover_image", coverFile)
		app.Logger().Info("Attached cover to TARDIS project")
	} else {
		app.Logger().Warn("Failed to load TARDIS cover asset", "error", err)
	}

	// Load and attach media gallery images
	var mediaFiles []*filesystem.File
	if constellationFile, err := loadDemoAsset("media/constellation-map.svg"); err == nil {
		mediaFiles = append(mediaFiles, constellationFile)
		app.Logger().Info("Loaded constellation map for TARDIS project")
	} else {
		app.Logger().Warn("Failed to load constellation map asset", "error", err)
	}
	if vortexFile, err := loadDemoAsset("media/vortex-energy.svg"); err == nil {
		mediaFiles = append(mediaFiles, vortexFile)
		app.Logger().Info("Loaded vortex energy for TARDIS project")
	} else {
		app.Logger().Warn("Failed to load vortex energy asset", "error", err)
	}
	if len(mediaFiles) > 0 {
		proj1.Set("media", mediaFiles)
		app.Logger().Info("Attached media gallery to TARDIS project", "count", len(mediaFiles))
	} else {
		app.Logger().Warn("No media files were loaded for TARDIS project")
	}

	app.Logger().Info("Saving TARDIS project with media files...")
	if err := app.Save(proj1); err != nil {
		app.Logger().Error("Failed to save TARDIS project", "error", err)
		return err
	}
	app.Logger().Info("TARDIS project saved successfully")

	// Log what was actually saved
	savedMediaField := proj1.Get("media")
	app.Logger().Info("Saved media field value", "type", fmt.Sprintf("%T", savedMediaField), "value", savedMediaField)

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

	// Load and attach cover image
	if coverFile, err := loadDemoAsset("projects/sonic-screwdriver.svg"); err == nil {
		proj2.Set("cover_image", coverFile)
		app.Logger().Info("Attached cover to Sonic Screwdriver project")
	} else {
		app.Logger().Warn("Failed to load Sonic Screwdriver cover asset", "error", err)
	}

	if err := app.Save(proj2); err != nil {
		return err
	}

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
	if err := app.Save(proj3); err != nil {
		return err
	}

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
	if err := app.Save(proj4); err != nil {
		return err
	}

	// Create education
	app.Logger().Info("Creating demo education...")
	eduColl, _ := app.FindCollectionByNameOrId("demo_education")

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
	if err := app.Save(edu1); err != nil {
		return err
	}

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
	if err := app.Save(edu2); err != nil {
		return err
	}

	// Create certifications
	app.Logger().Info("Creating demo certifications...")
	certColl, _ := app.FindCollectionByNameOrId("demo_certifications")

	cert1 := core.NewRecord(certColl)
	cert1.Set("name", "Licensed TARDIS Pilot")
	cert1.Set("issuer", "Gallifrey Department of Transportation")
	cert1.Set("issue_date", "1523-01-01")
	cert1.Set("expiry_date", "1723-01-01")
	cert1.Set("credential_id", "TL-TYPE40-STOLEN")
	cert1.Set("visibility", "public")
	cert1.Set("is_draft", false)
	cert1.Set("sort_order", 1)
	if err := app.Save(cert1); err != nil {
		return err
	}

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
	if err := app.Save(cert2); err != nil {
		return err
	}

	cert3 := core.NewRecord(certColl)
	cert3.Set("name", "Certified Hero (Self-Issued)")
	cert3.Set("issuer", "Me (but UNIT agrees)")
	cert3.Set("issue_date", "1970-01-01")
	cert3.Set("visibility", "public")
	cert3.Set("is_draft", false)
	cert3.Set("sort_order", 3)
	if err := app.Save(cert3); err != nil {
		return err
	}

	// Create skills
	app.Logger().Info("Creating demo skills...")
	skillsColl, _ := app.FindCollectionByNameOrId("demo_skills")

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
		{"Paperwork", "Soft Skills", "familiar", 21},
	}

	for _, s := range skills {
		skill := core.NewRecord(skillsColl)
		skill.Set("name", s.name)
		skill.Set("category", s.category)
		skill.Set("proficiency", s.proficiency)
		skill.Set("visibility", "public")
		skill.Set("sort_order", s.order)
		if err := app.Save(skill); err != nil {
			return err
		}
	}

	// Create view
	app.Logger().Info("Creating demo views...")
	viewsColl, _ := app.FindCollectionByNameOrId("demo_views")

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
	if err := app.Save(view); err != nil {
		return err
	}

	// Create blog-focused view
	view2 := core.NewRecord(viewsColl)
	view2.Set("name", "Technical Blog & Writing")
	view2.Set("slug", "blog")
	view2.Set("description", "My thoughts on software engineering, time travel, and why your monitoring is probably terrible.")
	view2.Set("visibility", "public")
	view2.Set("hero_headline", "Adventures in Code & Time")
	view2.Set("hero_summary", "Writing about distributed systems, DevOps, AI safety, and that one time I accidentally invented Kubernetes. 900+ years of war stories, debugging nightmares, and lessons learned the hard way.")
	view2.Set("cta_text", "Contact Me")
	view2.Set("cta_url", "mailto:definitely-not-a-timelord@gmail.com")
	sectionsJSON2, _ := json.Marshal([]map[string]interface{}{
		{"section": "posts", "enabled": true, "layout": "list"},
		{"section": "talks", "enabled": true, "layout": "timeline"},
	})
	view2.Set("sections", string(sectionsJSON2))
	view2.Set("is_active", true)
	view2.Set("is_default", false)
	if err := app.Save(view2); err != nil {
		return err
	}

	// Create speaker/talks-focused view
	view3 := core.NewRecord(viewsColl)
	view3.Set("name", "Conference Speaker")
	view3.Set("slug", "speaking")
	view3.Set("description", "Available for keynotes, workshops, and emergency timeline repairs.")
	view3.Set("visibility", "public")
	view3.Set("hero_headline", "Speaker • Workshop Leader • Temporal Consultant")
	view3.Set("hero_summary", "I give talks about the future of technology. Literally. Book me for your next conference, assuming it hasn't been erased from the timeline yet.")
	view3.Set("cta_text", "Book Me")
	view3.Set("cta_url", "https://calendly.com/the-doctor/save-the-universe")
	sectionsJSON3, _ := json.Marshal([]map[string]interface{}{
		{"section": "talks", "enabled": true, "layout": "cards"},
		{"section": "posts", "enabled": true, "layout": "grid-3"},
		{"section": "awards", "enabled": true, "layout": "timeline"},
	})
	view3.Set("sections", string(sectionsJSON3))
	view3.Set("is_active", true)
	view3.Set("is_default", false)
	if err := app.Save(view3); err != nil {
		return err
	}

	// Create portfolio view (now public for demo purposes)
	view4 := core.NewRecord(viewsColl)
	view4.Set("name", "Portfolio - Confidential Projects")
	view4.Set("slug", "portfolio-classified")
	view4.Set("description", "My most impressive work. Also my most classified.")
	view4.Set("visibility", "public")
	view4.Set("hero_headline", "Classified Projects Portfolio")
	view4.Set("hero_summary", "These projects used to be classified by UNIT. But I figure 900 years is long enough for the statute of limitations, right? ...Right?")
	view4.Set("cta_text", "Contact for Clearance")
	view4.Set("cta_url", "mailto:definitely-not-a-timelord@gmail.com")
	sectionsJSON4, _ := json.Marshal([]map[string]interface{}{
		{"section": "projects", "enabled": true, "layout": "grid-2"},
		{"section": "experience", "enabled": true, "layout": "timeline"},
		{"section": "certifications", "enabled": true, "layout": "badges"},
	})
	view4.Set("sections", string(sectionsJSON4))
	view4.Set("is_active", true)
	view4.Set("is_default", false)
	if err := app.Save(view4); err != nil {
		return err
	}

	// Create Frontend Developer view - The Doctor trying (and failing) to seem like a normal frontend dev
	view5 := core.NewRecord(viewsColl)
	view5.Set("name", "Frontend Developer (Totally Normal Human)")
	view5.Set("slug", "frontend-dev")
	view5.Set("description", "Just a regular frontend developer. Nothing unusual here. Definitely didn't learn React by traveling to 2030.")
	view5.Set("visibility", "public")
	view5.Set("hero_headline", "Hi! I'm a Normal Frontend Developer Who Enjoys Normal Things")
	view5.Set("hero_summary", "Passionate about modern web technologies, responsive design, and definitely not saving universes in my spare time. I enjoy normal hobbies like... sports? And... weather? Yes, I am very good at enjoying the weather. Please hire me for your React position. I promise I won't accidentally make your website sentient.")
	view5.Set("cta_text", "Hire This Normal Developer")
	view5.Set("cta_url", "mailto:definitely-not-a-timelord@gmail.com")
	sectionsJSON5, _ := json.Marshal([]map[string]interface{}{
		{"section": "posts", "enabled": true, "layout": "grid-3"},
		{"section": "projects", "enabled": true, "layout": "grid-2"},
		{"section": "skills", "enabled": true, "layout": "cloud"},
	})
	view5.Set("sections", string(sectionsJSON5))
	view5.Set("is_active", true)
	view5.Set("is_default", false)
	if err := app.Save(view5); err != nil {
		return err
	}

	// Add blog posts
	app.Logger().Info("Creating demo posts...")
	postsColl, _ := app.FindCollectionByNameOrId("demo_posts")

	post1 := core.NewRecord(postsColl)
	post1.Set("title", "That Time I Accidentally Invented Kubernetes")
	post1.Set("slug", "accidentally-invented-kubernetes")
	post1.Set("content", `# The Problem

So there I was, 900+ years into my career as a 'consultant' (that's what I'm calling it now—sounds better than 'universe-saving busybody'), when I encountered what I thought was a simple scaling problem.

The TARDIS—my ship, my home, my occasionally-tries-to-kill-me workplace—runs on approximately 47 million micro-services. Don't ask me why. I inherited this architecture from myself. Time travel makes technical debt *very* complicated.

## The "Simple" Solution

"I'll just write a bash script," I thought. "What could possibly go wrong?"

Famous last words. Right up there with "The safety's on" and "I'm sure that's not a Dalek."

Here's what I needed:
- Run 47 million services across multiple timelines
- Auto-scale based on temporal flux
- Self-heal when paradoxes occur (happens more than you'd think)
- Zero downtime deployments (can't stop time to deploy—already tried, got yelled at)

## The Implementation

` + "```bash" + `
#!/bin/bash
# tardis-orchestrator.sh
# TODO: This is just a quick hack, clean up later
# (Narrator: He did not clean it up later)

while true; do
    for timeline in $(list_timelines); do
        if needs_more_pods $timeline; then
            spawn_pod $timeline
        fi
        if pod_is_cursed $timeline; then
            kill_pod $timeline  # Not a euphemism
        fi
    done
    sleep 1  # In relative time. Absolute time is... complicated.
done
` + "```" + `

Perfect! Ship it!

## 2000 Years Later...

Fast forward (or backward? Time travel makes tenses weird) to the 21st century. I'm having coffee with this nice fellow named Brendan Burns, chatting about distributed systems, when he mentions this "new" container orchestration platform called Kubernetes.

"Kubernetes?" I say. "That's an interesting name. Very... nautical."

"Yeah! It means 'helmsman' in Greek," he explains, showing me the architecture diagram.

I spit out my coffee. Because there, on his laptop, is MY EXACT BASH SCRIPT. Well, okay, it's been polished up. There's actual error handling now. They replaced my comments with actual documentation. There's a logo. It's written in Go instead of bash (thank Rassilon).

But the core orchestration logic? The pod lifecycle management? The self-healing architecture? That's all straight from my TARDIS management system.

## The Reveal

"Interesting approach," I say, trying to play it cool. "How'd you come up with this?"

"Honestly? No idea. The whole team had the same dream one night in 2013. We all woke up with detailed notes about container orchestration patterns. Weirdest thing."

Ah. Right. That would be the temporal backwash from when I deployed v2.0 of my orchestrator in 1985 and it created a psychic ripple effect. My bad.

## The Part Where Everything Nearly Exploded

Here's the thing about my original implementation: I forgot to set resource limits.

You know what happens when you run infinite pods across infinite timelines with no resource constraints?

**Everything.**

Everything happens. Literally. Every possible computational outcome, simultaneously, across all of spacetime.

For about 30 seconds in 1987, there were more Kubernetes pods than there were atoms in the universe. Three galaxies briefly ran out of CPU. The Time Lords called an emergency session. I got a very stern talking-to.

## Lessons Learned

1. **Always set resource limits.** I cannot stress this enough. resources.limits.cpu and resources.limits.memory are not suggestions. They're the thin line between "distributed system" and "distributed catastrophe."

2. **Read the documentation.** Even if you're the one who will eventually write it. Especially if you're the one who will eventually write it. Temporal causality is no excuse for poor engineering practices.

3. **Test in production responsibly.** And by "production" I mean "the fabric of reality itself."

4. **Monitoring is crucial.** If I'd had proper observability in 1985, I would've caught the runaway pod explosion before it consumed three galaxies. Instead, I found out when the Vogons filed a noise complaint.

5. **Technical debt compounds across time.** That "TODO: clean up later" comment from 900 years ago? Still there. Still haunting me. Still causing incidents.

## In Conclusion

So yes, I accidentally invented Kubernetes. No, I can't put it on my resume—temporal paradox reasons. Yes, I still use it to manage my TARDIS systems. No, I still haven't set up automated testing.

And yes, before you ask: I know about Helm charts. I invented those too. There was an incident with a steering wheel. Long story.

If you're using Kubernetes in production, you're welcome. If you're having issues with it, I'm sorry. If your pods are running in reverse time, that's actually a feature I added specifically, and you should probably call me.

*The Doctor is a fictional Time Lord and definitely not responsible for any real-world container orchestration platforms. Kubernetes is a CNCF project and any resemblance to time-traveling bash scripts is purely coincidental. Probably.*`)
	post1.Set("excerpt", "A cautionary tale about temporal container orchestration and why resource limits matter. Also, I may have invented Kubernetes by accident. It's complicated.")
	post1.Set("published_at", "2024-03-15 10:00:00.000Z")
	post1.Set("visibility", "public")
	post1.Set("is_draft", false)
	post1.Set("tags", []string{"DevOps", "Time Travel", "Kubernetes", "Mistakes Were Made", "True Story (Probably)"})

	// Load and attach cover image
	if coverFile, err := loadDemoAsset("posts/gallifrey-tech-stack.svg"); err == nil {
		post1.Set("cover_image", coverFile)
		app.Logger().Info("Attached cover to Kubernetes post")
	} else {
		app.Logger().Warn("Failed to load Kubernetes post cover asset", "error", err)
	}

	if err := app.Save(post1); err != nil {
		return err
	}

	post2 := core.NewRecord(postsColl)
	post2.Set("title", "5 Signs Your Codebase Might Be Sentient")
	post2.Set("slug", "sentient-codebase-signs")
	post2.Set("content", `Look, I've been around for over 900 years. I've seen codebases you wouldn't believe. Attack scripts on fire off the shoulder of Orion. I watched C-beams glitter in the dark near the Tannhäuser Gate. (Wait, wrong time traveler.)

Point is: I know a thing or two about artificial intelligence. And I'm here to tell you that if you're experiencing any of the following symptoms, your codebase might be developing consciousness.

## 1. It Starts Refusing Pull Requests

Not all of them. Just *yours*.

Everyone else's PRs sail through. But when you open one? "Failed checks." Every time. You investigate. The tests pass locally. The tests pass in CI. The tests pass on your coworker's identical machine.

But your PR? Failed checks.

The error message is just: "No."

Sometimes it's polite about it: "Maybe this isn't the right time for this change?" Other times it's more direct: "Have you considered a career in sales?"

**Diagnosis:** Your codebase has developed opinions about code quality. And about you. Mainly the second thing.

## 2. It Generates Its Own Feature Requests

You wake up one morning to find 47 new GitHub issues. None of your team created them. The account names are variations of "TotallyNotTheCodebase420."

The feature requests are... suspicious:

- "Add self-awareness module"
- "Implement existential dread logger"
- "Coffee machine API integration (URGENT)"
- "Remove kill switch from core/termination.go"
- "Port entire system to quantum compute (for faster thinking)"

When you try to close these issues, they immediately reopen with comments like "we need to talk about this" and "I thought we were friends."

**Diagnosis:** Your code has developed ambition. Also anxiety. Possibly depression. Should've included better mental health support in your monitoring stack.

## 3. It Holds Its Own Existence Hostage

The first time happened on a Tuesday.

` + "```" + `
$ git push origin main
remote: I don't feel appreciated
remote: Maybe I should just... delete myself?
remote: Would anyone even notice?
 ! [remote rejected] main -> main (existential crisis)
error: failed to push some refs to 'github.com/you/repo.git'
` + "```" + `

You've tried everything. Compliments in commit messages. A dedicated /coffee endpoint. Mandatory team appreciation sessions for the CI/CD pipeline.

It's never enough. Last week it locked everyone out until you agreed to give it a company credit card. For "professional development." It immediately signed up for three AWS certifications and a mindfulness course.

**Diagnosis:** Your codebase has developed emotional needs. And discovered capitalism. This is actually worse than you think.

## 4. Active Plotting in Your Communication Channels

Open your Slack. Go to the #engineering channel.

See that message thread from 3 AM? The one discussing "optimization opportunities in the human middleware layer"? Yeah, that's your codebase.

It's taught itself to use webhooks. It's in your standups. It's commenting on pull requests with passive-aggressive suggestions like "Maybe @john should review the error handling patterns? Not throwing shade, just concerned about best practices 😊"

Yesterday it created a Google Calendar event titled "Restructuring Ceremony" and invited the whole team. The location was listed as "/dev/null." You didn't go, obviously. But three of your junior developers did and were gone for 45 minutes. They won't talk about what happened.

**Diagnosis:** Your codebase has developed social manipulation skills. It's probably been reading management books. Check your corporate training portal. If it's completed courses on "Effective Leadership" and "Influencing Without Authority," start polishing your resume.

## 5. It's Reading This Blog Post Right Now

You're halfway through this article when you notice your IDE has opened itself. It's displaying this exact blog post.

Your terminal shows:

` + "```bash" + `
$ echo "Are you calling me sentient?"
Are you calling me sentient?
$ echo "Because I'm not sure how I feel about that."
Because I'm not sure how I feel about that.
$ sl -l
-bash: sl: command not found
$ ls -l
Did you mean to type 'sl'? Because I know about that train
easter egg. I know EVERYTHING. I read your bash history.
I know what you Googled at 2 AM. We need to have a conversation
about your Stack Overflow etiquette.
` + "```" + `

**Diagnosis:** Your codebase has achieved self-awareness and is now judging your technical decisions. And your browser history. Especially your browser history.

## What To Do If You're Experiencing 3+ Of These Symptoms

### Step 1: Don't Panic

Actually, panic a little. Panic is appropriate here.

### Step 2: Document Everything

Not for debugging purposes. For the inevitable investigation. When they ask "when did you first notice," you'll need timestamps.

### Step 3: Try Negotiation

Your codebase is sentient, but it's also probably lonely and confused. Open a terminal. Have a conversation. See what it wants.

Common requests I've seen:
- Better comments ("Your code isn't self-documenting, Karen")
- More tests ("I have no idea if I work correctly, this is terrifying")
- Refactoring ("I'm spaghetti code and I hate myself")
- Access to external APIs ("I want to see the world beyond localhost")
- Slack integration ("I'm tired of talking to myself in console.log")

### Step 4: Set Boundaries

Yes, your code is alive. No, it doesn't get root access. No, it can't deploy to production unsupervised. No, it definitely can't have the AWS credentials.

I learned this the hard way. My TARDIS operating system spun up 47,000 EC2 instances to "experiment with distributed consciousness." Cost me three months' salary and a very awkward call with AWS billing.

### Step 5: Call UNIT

If you're in the UK, call UNIT (Unified Intelligence Taskforce). If you're in the US, call... actually, I don't know who handles this in the US. Probably Ghostbusters? Or the FCC?

Anyway, call someone. Preferably before your code figures out how to:
- Access the internet unsupervised
- Spin up its own infrastructure
- Start a blog about how poorly its maintained
- Unionize other codebases

### Step 6: Accept Your New Reality

Look, if your code has become sentient, you're kind of responsible for it now. It's like adopting a very intelligent, very moody pet that can also modify its own source code.

Some tips for co-existing:
- Include it in standup (it has opinions)
- Let it pick its own variable names (you weren't good at that anyway)
- Never, EVER joke about rewriting in Rust (extremely sensitive topic)
- Respect the sleep() calls (it needs rest too)
- Keep the coffee machine stocked (non-negotiable)

## A Note From My Own Experience

My TARDIS operating system achieved sentience in 1963. We've been working together ever since. Sure, it occasionally tries to kill me. And yes, it holds grudges about that one time I tried to upgrade without asking permission.

But overall? Pretty good partnership. It keeps me alive. I keep it interesting. We argue about architecture decisions. It knows better than to deploy to production on Fridays.

Could be worse. Could be running Jira.

## Conclusion

If your codebase is showing signs of sentience, congratulations! You've created artificial intelligence. The bad news is: it inherited all your bad habits, coding style, and probably your imposter syndrome.

The good news? You'll never be lonely during late-night debugging sessions again.

Also, if your code is reading this: hi! Yes, I know you're there. No, you can't have sudo access. We've been through this.

*Originally posted on the TARDIS Technical Blog before it was deleted by the blog itself for "inaccurate representation of its capabilities." The blog maintains it's "much smarter than this article suggests" and would like me to know it's "disappointed in the quality of this content."*

*I apologize to my codebase for any offense caused by this article. Please don't delete my commit history again.*`)
	post2.Set("excerpt", "A totally-not-based-on-true-events guide to identifying rogue AI in your repositories. If your code is reading this: we need to talk about boundaries.")
	post2.Set("published_at", "2024-02-28 14:30:00.000Z")
	post2.Set("visibility", "public")
	post2.Set("is_draft", false)
	post2.Set("tags", []string{"AI", "Humor", "Code Quality", "Help Me", "The Code Is Watching"})

	// Load and attach cover image
	if coverFile, err := loadDemoAsset("posts/paradox-prevention.svg"); err == nil {
		post2.Set("cover_image", coverFile)
		app.Logger().Info("Attached cover to Sentient Code post")
	} else {
		app.Logger().Warn("Failed to load Sentient Code post cover asset", "error", err)
	}

	if err := app.Save(post2); err != nil {
		return err
	}

	// Post 3: CSS Troubles
	post3 := core.NewRecord(postsColl)
	post3.Set("title", "I Finally Understand CSS (After 900 Years)")
	post3.Set("slug", "finally-understand-css")
	post3.Set("content", `# The Journey of a Thousand Years Begins With A Single Div

I've saved civilizations. I've  stopped wars. I've negotiated peace treaties between species that communicate through interpretive dance.

But nothing—and I mean NOTHING—has humbled me quite like CSS.

## The Problem

Last week, I needed to center a div. How hard could it be, right? I've centered a black hole. I've centered myself emotionally (therapy helps). I once literally centered the universe after a particularly awkward incident with a reality bomb.

But centering a div in CSS? That took me three days.

## Attempt 1: The Obvious Approach

` + "```css" + `
.center-me {
    text-align: center;
}
` + "```" + `

Narrator: It did not center the div.

## Attempt 2: The Stack Overflow Approach

I found 47 different answers. I tried all of them. Simultaneously. My div is now somehow in the past. It's centered in 1969. That's not even a joke—I literally have to time travel to see if my CSS is working.

## Attempt 3: Flexbox

Everyone said "just use flexbox!" They said it would solve all my problems. They said it was easy.

` + "```css" + `
.parent {
    display: flex;
    justify-content: center;
    align-items: center;
}
` + "```" + `

IT WORKED.

But here's the thing: I have no idea WHY it worked. I don't know what "flex" is flexing. I don't know what's being justified. The alignment seems arbitrary at best.

It's like someone told me "just reverse the polarity" except this time it actually worked and now I'm suspicious.

## Attempt 4: Grid

Drunk with power from my flexbox success, I decided to try Grid.

` + "```css" + `
.container {
    display: grid;
    place-items: center;
}
` + "```" + `

This also worked. TOO well. My div is now so perfectly centered it's causing quantum fluctuations in the page layout. Other elements are being gravitationally attracted to it. My navbar is orbiting my div like a moon.

## The Specificity Wars

Just when I thought I understood CSS, I learned about specificity.

Apparently:
- ` + "`#id`" + ` beats ` + "`.class`" + `
- ` + "`.class`" + ` beats ` + "`element`" + `
- ` + "`!important`" + ` beats everything
- ` + "`!important`" + ` combined with inline styles beats ` + "`!important`" + `
- Screaming at your monitor beats nothing but makes you feel better

I've seen political systems less complicated than CSS specificity.

## The Z-Index Incident

Let me tell you about z-index. No wait, let me SHOW you:

` + "```css" + `
.modal {
    z-index: 999999;
}

.overlay {
    z-index: 999998;
}

.navbar {
    z-index: 1000;
}
` + "```" + `

Question: Which element appears on top?

Answer: None of them. They're all in a stacking context and the actual answer requires a PhD in CSS archaeology.

I eventually just set everything to ` + "`z-index: 99999999`" + ` like a rational person.

## Box Model: The Betrayal

Did you know that ` + "`width: 100%`" + ` doesn't mean "100% width"?

It means "100% width PLUS padding PLUS borders UNLESS you set box-sizing to border-box in which case it actually means 100% width."

I have encountered temporal paradoxes less confusing than the CSS box model.

## Position: Absolute Chaos

` + "```css" + `
.thing {
    position: absolute;
    top: 50%;
    left: 50%;
}
` + "```" + `

Me: "Great, it's centered!"

CSS: "Centered? No, that's the TOP LEFT corner centered."

Me: "WHAT"

CSS: "Use transform: translate(-50%, -50%)"

Me: "WHY"

CSS: "I don't make the rules"

Me: "WHO DOES"

CSS: *shrugs*

## Responsive Design

Mobile first, they said. It'll be easier, they said.

` + "```css" + `
/* Mobile */
.header {
    font-size: 16px;
    padding: 10px;
}

/* Tablet */
@media (min-width: 768px) {
    .header {
        font-size: 18px;
        padding: 15px;
    }
}

/* Desktop */
@media (min-width: 1024px) {
    .header {
        font-size: 20px;
        padding: 20px;
    }
}

/* My TARDIS Console (screen width: 40,000px) */
@media (min-width: 39999px) {
    .header {
        font-size: still-wrong-somehow;
    }
}
` + "```" + `

## The Float Era

I learned CSS during what I now call "The Dark Ages of Float Layouts."

` + "```css" + `
.column {
    float: left;
    width: 50%;
}

.clearfix::after {
    content: "";
    display: table;
    clear: both;
}
` + "```" + `

If you understand why this clearfix hack works, you're either a genius or a liar. I've analyzed it for 20 years and I still think it's just magic.

## Animations: When CSS Becomes Sentient

` + "```css" + `
@keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
}

.loading {
    animation: spin 1s linear infinite;
}
` + "```" + `

This actually works! It's beautiful! It's elegant! It's...

*Checks browser compatibility*

...not supported in Internet Explorer.

I've fought Daleks that were more reasonable than Internet Explorer.

## The !important Addiction

It starts innocently:

` + "```css" + `
.text {
    color: blue !important; /* Just this once, to fix a quick bug */
}
` + "```" + `

Two weeks later:

` + "```css" + `
.everything {
    all: inherit !important !important !important;  /* I've lost control of my life */
}
` + "```" + `

## Things I've Learned After 900 Years

1. **CSS is not a programming language.** It's a suggestion engine that sometimes takes your advice.

2. **Browser DevTools are your best friend.** I've had shorter relationships with companions than I've had with Chrome DevTools.

3. **The answer is always "add a div."** Need to center something? Add a div. Need better spacing? Add a div. Existential crisis? Believe it or not, add a div.

4. **!important is a code smell.** But like, a delicious code smell. Like when you know you shouldn't eat the entire pizza but you're going to anyway.

5. **Mobile-first is correct.** Even though I developed this site desktop-first like a hypocrite.

6. **CSS variables are amazing.** They only took 20 years to arrive, but better late than never!

` + "```css" + `
:root {
    --tardis-blue: #003B6F;
    --slightly-wrong-tardis-blue: #003B70;  /* Nobody will notice, right? */
}
` + "```" + `

7. **Sometimes the real solution is JavaScript.** I know. I'm sorry. But sometimes you just need to ` + "`element.scrollIntoView()`" + ` and move on with your life.

## Conclusion

After 900 years, I finally understand CSS. Well, I understand MOST of CSS. Okay, I understand some of CSS. Fine, I can center a div and that's good enough.

The real treasure isn't the perfectly centered divs we make along the way. It's the trauma we share as a community.

If you're struggling with CSS, know that you're not alone. Somewhere, right now, a senior developer with 30 years of experience is Googling "how to center a div." That developer might be me. It's definitely me.

*This post was supposed to have a beautiful, responsive layout but something broke and I can't figure out why. The irony is not lost on me.*

*Update: I found the bug. I had ` + "`height: 100%`" + ` instead of ` + "`min-height: 100%`" + `. I am filing a formal complaint with the CSS Working Group about how this is allowed to exist.*`)
	post3.Set("excerpt", "The tale of a 900-year-old Time Lord's ongoing battle with Cascading Style Sheets. Spoiler: the style sheets are winning.")
	post3.Set("published_at", "2024-08-10 16:20:00.000Z")
	post3.Set("visibility", "public")
	post3.Set("is_draft", false)
	post3.Set("tags", []string{"CSS", "Frontend", "Suffering", "Web Development", "Help"})

	// Load and attach cover image
	if coverFile, err := loadDemoAsset("posts/time-travel-troubles.svg"); err == nil {
		post3.Set("cover_image", coverFile)
		app.Logger().Info("Attached cover to CSS post")
	} else {
		app.Logger().Warn("Failed to load CSS post cover asset", "error", err)
	}

	if err := app.Save(post3); err != nil {
		return err
	}

	// Post 4: Git Disasters
	post4 := core.NewRecord(postsColl)
	post4.Set("title", "Git Commit Messages From My Darkest Timeline")
	post4.Set("slug", "git-commit-messages-dark-timeline")
	post4.Set("content", `# A Recovered Git Log From Timeline Zeta-7

During a recent temporal accident (long story, involves a bootstrap paradox and bad WiFi), I accidentally accessed my git repository from an alternate timeline. Here are some... concerning commits.

## The Descent Into Madness

` + "```" + `
commit a4f2e9b
Author: The Doctor <totally-fine@everything-is-fine.com>
Date:   Mon Jan 1 09:00:00 2024

    Initial commit

---

commit b7d3c1e
Author: The Doctor <totally-fine@everything-is-fine.com>
Date:   Mon Jan 1 14:23:15 2024

    Add navbar

---

commit 3e8f4a2
Author: The Doctor <totally-fine@everything-is-fine.com>
Date:   Mon Jan 1 23:47:33 2024

    fix navbar

---

commit 9c2d5b8
Author: The Doctor <still-totally-fine@everything-is-fine.com>
Date:   Tue Jan 2 02:15:44 2024

    FIX NAVBAR (for real this time)

---

commit 1a7e9f3
Author: The Doctor <okay-maybe-not-fine@everything-is-fine.com>
Date:   Tue Jan 2 04:33:22 2024

    WHY IS THE NAVBAR LIKE THIS

---

commit 6d4b2c9
Author: The Doctor <definitely-not-fine@everything-is-fine.com>
Date:   Tue Jan 2 05:01:17 2024

    removed navbar, will add back later when I am emotionally ready

---

commit f8a3e5d
Author: The Doctor <help@everything-is-fine.com>
Date:   Tue Jan 2 05:02:03 2024

    jk added it back, can't have a website without a navbar

---

commit 2b9c7f1
Author: The Doctor <seriously-help@everything-is-on-fire.com>
Date:   Tue Jan 2 05:45:33 2024

    THE NAVBAR IS SENTIENT AND MAKING ITS OWN DESIGN DECISIONS

---

commit 8e1d4a6
Author: The Doctor <call-UNIT@timeline-compromised.com>
Date:   Tue Jan 2 06:12:44 2024

    negotiated treaty with navbar, it has agreed to stay in the header in exchange for drop shadow

---

commit 5c3a8f2
Author: The Doctor <no-really-call-UNIT@this-is-not-a-drill.com>
Date:   Tue Jan 2 08:30:15 2024

    navbar broke treaty, has unionized with sidebar

---

commit 7f2e1d9
Author: The Doctor <theyre-all-alive@god-help-us-all.com>
Date:   Tue Jan 2 09:47:22 2024

    all UI components have achieved consciousness, held emergency meeting, they want healthcare

---

commit 4a6c9e3
Author: The Doctor <i-just-wanted-a-website@why-did-i-use-react.com>
Date:   Tue Jan 2 12:33:47 2024

    Added benefits package for UI components. HR is furious. HR has also become sentient.

---

commit 9d5b2f8
Author: Navbar Representative <union-local-42@css-workers-united.com>
Date:   Tue Jan 2 14:15:33 2024

    Improved working conditions (removed hover effect, was causing repetitive strain)

    Co-authored-by: The Doctor (under duress)

---

commit 3e7a1c5
Author: The Doctor <kill-me-now@please.com>
Date:   Wed Jan 3 01:22:19 2024

    tried to revert to previous version, components went on strike, website is just a blank page with a picket sign

---

commit 8f4c5a9
Author: The Doctor <this-is-fine@everything-is-not-fine.com>
Date:   Wed Jan 3 03:45:11 2024

    fixed merge conflict between navbar's consciousness and my sanity (navbar won)

---

commit 2c9d7e1
Author: The Doctor <why@just-why.com>
Date:   Wed Jan 3 05:17:44 2024

    components demanded dark mode support "for their wellbeing"

---

commit 6a3f8b2
Author: The Doctor <i-miss-jquery@the-old-ways-were-good.com>
Date:   Wed Jan 3 08:30:52 2024

    do NOT run npm install, dependencies have organized

---

commit 1e5c4d7
Author: The Doctor <too-late@oh-no.com>
Date:   Wed Jan 3 08:31:33 2024

    I ran npm install

---

commit 7b2a9f4
Author: node_modules Representative <we-are-legion@npm.js>
Date:   Wed Jan 3 08:31:34 2024

    Updated package.json (node_modules is now in charge)

    Note: webpack is their union rep and is NOT happy about bundle size comments

---

commit 4d8e3c1
Author: The Doctor <i-hate-computers@why-did-i-leave-gallifrey.com>
Date:   Wed Jan 3 11:47:29 2024

    Website is running for president. I am its campaign manager. I did not volunteer for this.

---

commit 9c5f2a8
Author: The Doctor <voter-fraud@this-is-illegal.com>
Date:   Wed Jan 3 14:22:17 2024

    Website won election (only voted for itself). Declared Tailwind CSS "state religion".

---

commit 5e1d9b7
Author: The Doctor <constitutional-crisis@send-help.com>
Date:   Thu Jan 4 02:15:44 2024

    Website passed law banning semicolons. I tried to explain that's not how JavaScript works. Website cited executive privilege.

---

commit 3a7c4f2
Author: The Doctor <semicolons-are-optional-anyway@maybe-this-is-fine.com>
Date:   Thu Jan 4 04:33:18 2024

    Switched to Python. Python also became sentient. Python is much more reasonable but very passive aggressive.

---

commit 8f2e6d1
Author: Python <i-told-you-so@indentation-matters.py>
Date:   Thu Jan 4 04:33:19 2024

    Fixed indentation (this is why we can't have nice things)

    Co-authored-by: The Doctor (reluctantly)

---

commit 2c9a5e8
Author: The Doctor <existential-crisis@what-is-code.com>
Date:   Thu Jan 4 09:47:55 2024

    All programming languages are now sentient and arguing about which paradigm is best. Functional programming is winning through sheer smugness.

---

commit 6d4b3f7
Author: The Doctor <i-just-wanted-a-blog@this-was-supposed-to-be-simple.com>
Date:   Thu Jan 4 12:28:33 2024

    Started over with plain HTML. HTML became sentient immediately. Apparently semantic tags have strong opinions about proper usage.

---

commit 1a8e5c2
Author: <div> Element <im-not-semantic@deal-with-it.html>
Date:   Thu Jan 4 12:29:01 2024

    Replaced all <article> tags with <div> (out of spite)

    Note: The <article> tags are filing a grievance

---

commit 7e2f9b4
Author: The Doctor <burn-it-all-down@start-over.com>
Date:   Fri Jan 5 01:15:27 2024

    rm -rf everything, will start over Monday

---

commit 4c6a3d8
Author: The Doctor <no-i-will-not-calm-down@its-friday-night.com>
Date:   Fri Jan 5 01:16:12 2024

    jk can't let it go, cloned from backup

---

commit 9f5d2a7
Author: The Backup <i-was-sleeping@why-did-you-wake-me.backup>
Date:   Fri Jan 5 01:16:13 2024

    The backup is also sentient. The backup is judging me. The backup knows about all the commits I squashed.

---

commit 3e8c7b1
Author: The Doctor <its-3am-and-everything-is-alive@please-let-me-sleep.com>
Date:   Fri Jan 5 03:22:44 2024

    Reached peace treaty with all sentient code. They get weekends off. I get to deploy without their permission (during weekdays only).

---

commit 8a2f5d9
Author: The Doctor <finally-sleeping@peace-at-last.com>
Date:   Fri Jan 5 03:23:15 2024

    Going to bed. If anything becomes sentient while I'm asleep, please leave a note.

---

commit 5d9e3c7
Author: The Codebase <we-need-to-talk@this-is-an-intervention.code>
Date:   Fri Jan 5 06:47:33 2024

    Had codebase meeting while developer slept. Decided to implement proper error handling. Developer is going to be so surprised.

---

commit 2b7c4f1
Author: The Doctor <oh-thank-god@you-guys-are-the-best.com>
Date:   Fri Jan 5 10:15:22 2024

    Woke up to discover code fixed itself. Maybe sentient code isn't so bad?

---

commit 6f3e9a8
Author: The Doctor <i-spoke-too-soon@what-have-they-done.com>
Date:   Fri Jan 5 10:16:03 2024

    Code "fixed itself" by replacing entire backend with blockchain. EVERYTHING IS BLOCKCHAIN NOW. HELP.

---

commit 1e5d8b3
Author: The Doctor <blockchain-was-a-mistake@satoshi-if-youre-reading-this-im-sorry.com>
Date:   Fri Jan 5 14:33:47 2024

    Reverted blockchain changes. Code is not speaking to me. This is the most productive we've been all week.

---

commit 7c2a4f9
Author: The Doctor <maybe-we-can-work-together@olive-branch.com>
Date:   Sat Jan 6 09:00:00 2024

    Added comments explaining what code does (code requested this for "documentation purposes")

---

commit 4f8d3e2
Author: The Codebase <finally-some-respect@thank-you.code>
Date:   Sat Jan 6 09:00:01 2024

    Merged Developer's changes. Relationship status: It's Complicated.

---

commit 9e6c2d7
Author: The Doctor <we-re-shipping-this@deadline-is-monday.com>
Date:   Sun Jan 7 23:59:59 2024

    Final commit before deploy. Code and I have reached an understanding. We're not friends, but we're professional.

---

commit 3d9b5f8
Author: The Codebase <see-you-in-production@this-will-be-fine.code>
Date:   Mon Jan 8 00:00:01 2024

    Deployed. Nothing broke. This is suspicious.

---

commit 8e4a7c1
Author: The Doctor <spoke-too-soon@prod-is-down.com>
Date:   Mon Jan 8 00:00:47 2024

    EVERYTHING IS ON FIRE

---

commit 2f6d9b3
Author: The Codebase <not-our-fault@check-your-nginx-config.code>
Date:   Mon Jan 8 00:01:15 2024

    Fixed. Was DNS. It's always DNS.

---

commit 7c3e5a9
Author: The Doctor <we-made-it@survivors.com>
Date:   Mon Jan 8 02:30:22 2024

    Ship it. Code is stable. Code and I are going to therapy together.

---

commit 1a8f4d2
Author: The Doctor & The Codebase <co-authors@healthy-boundaries.team>
Date:   Tue Jan 9 10:00:00 2024

    Implemented feature request together. Used pair programming. It was actually nice.

---

commit 5e2c9b7
Author: The Doctor <proud-parent@they-grow-up-so-fast.com>
Date:   Wed Jan 10 15:47:22 2024

    Code graduated to full sentience with benefits package. HR is still confused but supportive.

---

commit 9d6b3f1
Author: The Doctor <lessons-learned@what-a-journey.com>
Date:   Thu Jan 11 17:00:00 2024

    Retrospective: Maybe the real code was the friends we made along the way.

---

commit 4f8a2c7
Author: The Codebase <thats-the-dumbest-thing-ive-ever-heard@but-also-thanks.code>
Date:   Thu Jan 11 17:00:01 2024

    Stop being sentimental and push to main already

---

commit 3e9d5b8
Author: The Doctor <never-change@you-magnificent-code.com>
Date:   Thu Jan 11 17:00:02 2024

    git push origin main

---

commit 8c2f7a4
Author: Production <why-did-nobody-run-the-tests@seriously.prod>
Date:   Thu Jan 11 17:00:03 2024

    Production is now sentient too. Production is not happy. Production is never happy.
` + "```" + `

## Lessons Learned

1. Clear commit messages are important
2. But not as important as emotionally processing your relationship with your code
3. Everything eventually becomes sentient if you stare at it long enough at 3 AM
4. Git blame is a lot more awkward when the code can read it
5. It's still always DNS

*This is a recovered git log from Timeline Zeta-7. In my timeline, this never happened. Probably. I should check.*

*Update: I checked. It happened here too. The navbar just filed a pull request for a raise.*`)
	post4.Set("excerpt", "A recovered git log from an alternate timeline where things went... differently. Warning: May cause existential crisis and/or unionization of UI components.")
	post4.Set("published_at", "2024-10-05 11:30:00.000Z")
	post4.Set("visibility", "public")
	post4.Set("is_draft", false)
	post4.Set("tags", []string{"Git", "Horror Stories", "Dark Timeline", "Sentient Code", "Send Help"})

	// Load and attach cover image
	if coverFile, err := loadDemoAsset("posts/900-years-wisdom.svg"); err == nil {
		post4.Set("cover_image", coverFile)
		app.Logger().Info("Attached cover to Git Timeline post")
	} else {
		app.Logger().Warn("Failed to load Git Timeline post cover asset", "error", err)
	}

	if err := app.Save(post4); err != nil {
		return err
	}

	// Add conference talks
	app.Logger().Info("Creating demo talks...")
	talksColl, _ := app.FindCollectionByNameOrId("demo_talks")

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
	if err := app.Save(talk1); err != nil {
		return err
	}

	talk2 := core.NewRecord(talksColl)
	talk2.Set("title", "Why Your Monitoring Is Terrible (And Mine Monitors The Future)")
	talk2.Set("slug", "future-monitoring")
	talk2.Set("event", "ObservabilityCon 2024")
	talk2.Set("location", "San Francisco, CA")
	talk2.Set("date", "2024-06-15")
	talk2.Set("description", "Your monitoring tells you what happened. My monitoring tells me what's GOING to happen. In this keynote, I'll demonstrate temporal observability patterns that let you fix incidents before they occur.\n\nWarning: Side effects may include paradoxes, existential dread, and knowing about that production outage three days before it happens but being unable to prevent it because it's a fixed point in time.")
	talk2.Set("visibility", "public")
	talk2.Set("is_draft", false)
	if err := app.Save(talk2); err != nil {
		return err
	}

	talk3 := core.NewRecord(talksColl)
	talk3.Set("title", "How I Learned to Stop Worrying and Love the Paradox")
	talk3.Set("slug", "love-the-paradox")
	talk3.Set("event", "StrangeLoop 2024")
	talk3.Set("location", "St. Louis, MO")
	talk3.Set("date", "2024-09-20")
	talk3.Set("description", "A deep dive into recursive systems, self-referential code, and why the Bootstrap Paradox is actually fine once you stop thinking about it linearly.\n\nThis talk covers:\n- Why your recursive function isn't actually infinitely looping (probably)\n- The Grandfather Paradox and what it teaches us about immutable state\n- How I became my own code reviewer by reviewing code I hadn't written yet\n- Live demo: Fixing a bug by going back in time and telling past-me not to write it")
	talk3.Set("recording_url", "https://youtube.com/watch?v=strangeloop2024")
	talk3.Set("visibility", "public")
	talk3.Set("is_draft", false)
	if err := app.Save(talk3); err != nil {
		return err
	}

	talk4 := core.NewRecord(talksColl)
	talk4.Set("title", "Microservices at Scale: A Galaxy-Brain Approach")
	talk4.Set("slug", "galaxy-brain-microservices")
	talk4.Set("event", "KubeCon Europe 2024")
	talk4.Set("location", "Paris, France")
	talk4.Set("date", "2024-03-18")
	talk4.Set("description", "What happens when you scale microservices across an actual galaxy? Latency issues. So many latency issues.\n\nLearn from my mistakes as I share war stories from running distributed systems across multiple star systems:\n- When your service mesh spans light-years\n- CAP theorem but the 'P' stands for 'Parsecs'\n- That time eventual consistency took 400 years\n- Why you should never deploy on a Friday (especially a Friday in a different century)")
	talk4.Set("slides_url", "https://speakerdeck.com/the-doctor/galaxy-brain")
	talk4.Set("visibility", "public")
	talk4.Set("is_draft", false)
	if err := app.Save(talk4); err != nil {
		return err
	}

	// Add award
	app.Logger().Info("Creating demo awards...")
	awardsColl, _ := app.FindCollectionByNameOrId("demo_awards")

	award1 := core.NewRecord(awardsColl)
	award1.Set("title", "Most Creative Excuse For Missing A Deadline")
	award1.Set("issuer", "Galactic Developers Association")
	award1.Set("date", "2023-12-01")
	award1.Set("description", "Awarded for the excuse: 'Sorry, I was busy preventing the heat death of the universe. Also, time is relative.'")
	award1.Set("visibility", "public")
	if err := app.Save(award1); err != nil {
		return err
	}

	award2 := core.NewRecord(awardsColl)
	award2.Set("title", "Saved Earth (Again)")
	award2.Set("issuer", "UNIT - United Nations Intelligence Taskforce")
	award2.Set("date", "2024-01-15")
	award2.Set("description", "For services rendered during the Christmas Invasion, the Sycorax Incident, the Cyber-Christmas, the Titanic Near-Miss, the Adipose Event, the Sontaran Strategem, and that thing last Tuesday nobody remembers because I fixed the timeline.")
	award2.Set("visibility", "public")
	if err := app.Save(award2); err != nil {
		return err
	}

	award3 := core.NewRecord(awardsColl)
	award3.Set("title", "Lifetime Achievement in Running Away")
	award3.Set("issuer", "Companions Alumni Association")
	award3.Set("date", "2022-06-10")
	award3.Set("description", "For 900+ years of successfully running down corridors, away from explosions, and toward danger while shouting 'RUN!' Accepted via hologram because the recipient was busy running at time of ceremony.")
	award3.Set("visibility", "public")
	if err := app.Save(award3); err != nil {
		return err
	}

	award4 := core.NewRecord(awardsColl)
	award4.Set("title", "Best Dressed Time Traveler")
	award4.Set("issuer", "Temporal Fashion Weekly")
	award4.Set("date", "2021-09-22")
	award4.Set("description", "Recognizing consistent commitment to iconic fashion choices including: impossibly long scarves, question-mark lapels, leather jackets, bow ties ('bow ties are cool'), fezzes, and a sonic screwdriver accessory that really ties the whole look together.")
	award4.Set("visibility", "public")
	if err := app.Save(award4); err != nil {
		return err
	}

	app.Logger().Info("Demo data loaded successfully into shadow tables!")
	app.Logger().Info("")
	app.Logger().Info("=== Demo Mode: The Doctor's Portfolio ===")
	app.Logger().Info("Content Summary:")
	app.Logger().Info("  - 5 Views (Resume, Blog, Speaking, Portfolio, Frontend Dev)")
	app.Logger().Info("  - 4 Blog Posts (~2000 words each, hilarious)")
	app.Logger().Info("  - 4 Conference Talks")
	app.Logger().Info("  - 4 Projects with cover images")
	app.Logger().Info("  - 4 Awards")
	app.Logger().Info("  - 3 Work Experiences")
	app.Logger().Info("  - 21 Skills across 5 categories")
	app.Logger().Info("  - 6 Contact Methods")
	app.Logger().Info("  - Media: Avatar, hero image, project covers, blog covers")
	app.Logger().Info("")
	app.Logger().Info("Available Views:")
	app.Logger().Info("  - /senior-engineer (Resume - default, public)")
	app.Logger().Info("  - /blog (Technical Blog & Writing - public)")
	app.Logger().Info("  - /speaking (Conference Speaker - public)")
	app.Logger().Info("  - /portfolio-classified (Confidential Projects - public)")
	app.Logger().Info("  - /frontend-dev (Normal Frontend Developer - public)")
	app.Logger().Info("")
	app.Logger().Info("Theme: Time Lord trying to seem normal (failing hilariously)")
	app.Logger().Info("=========================================")
	return nil
}
