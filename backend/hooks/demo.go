package hooks

import (
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
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
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "Already in demo mode",
				})
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
		"demo_awards", "demo_views", "demo_share_tokens",
	}

	for _, tableName := range tables {
		records, err := app.FindRecordsByFilter(tableName, "", "", 1000, 0)
		if err != nil {
			continue
		}
		for _, record := range records {
			if err := app.Delete(record); err != nil {
				return err
			}
		}
	}

	return nil
}

// loadDemoDataIntoShadowTables loads The Doctor's demo data into demo_* shadow tables
func loadDemoDataIntoShadowTables(app *pocketbase.PocketBase) error {
	app.Logger().Info("Loading demo data into shadow tables...")

	// Create demo profile
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
		{"type": "github", "url": "https://github.com/madman-with-a-box", "label": "GitHub"},
		{"type": "linkedin", "url": "https://linkedin.com/in/the-doctor-900-years", "label": "LinkedIn"},
		{"type": "twitter", "url": "https://twitter.com/TheRealDoctor", "label": "@TheRealDoctor"},
		{"type": "website", "url": "https://police-box-exterior.tardis", "label": "Personal Website"},
		{"type": "email", "url": "mailto:consulting@timelords.gallifrey", "label": "Consulting Email"},
		{"type": "other", "url": "https://calendly.com/the-doctor/save-the-world", "label": "Book a Meeting"},
	})
	profile.Set("visibility", "public")
	if err := app.Save(profile); err != nil {
		return err
	}

	// Create demo experience entries
	expColl, _ := app.FindCollectionByNameOrId("demo_experience")

	// UNIT Experience
	exp1 := core.NewRecord(expColl)
	exp1.Set("company", "UNIT (Unified Intelligence Taskforce)")
	exp1.Set("position", "Scientific Advisor")
	exp1.Set("location", "Earth (Various Timelines)")
	exp1.Set("start_date", "1970-01-01")
	exp1.Set("end_date", "")
	exp1.Set("current", true)
	exp1.Set("description", "Consulting on extraterrestrial threats and impossible problems")
	exp1.Set("highlights", []string{
		"Prevented 23 alien invasions",
		"Established protocols for first contact scenarios",
		"Trained elite task force in dealing with temporal anomalies",
	})
	exp1.Set("visibility", "public")
	app.Save(exp1)

	// Software Company Experience
	exp2 := core.NewRecord(expColl)
	exp2.Set("company", "Totally Normal Software Company")
	exp2.Set("position", "Senior Everything Engineer")
	exp2.Set("location", "Remote (Literally Across All of Time and Space)")
	exp2.Set("start_date", "2015-06-01")
	exp2.Set("end_date", "2023-12-31")
	exp2.Set("current", false)
	exp2.Set("description", "Led development of impossibly complex systems")
	exp2.Set("highlights", []string{
		"Reduced deployment time from 3 weeks to 12 parsecs",
		"Implemented monitoring system that predicts bugs before they're written",
		"Mentored junior developers across 4 dimensional planes",
	})
	exp2.Set("visibility", "public")
	app.Save(exp2)

	// Gallifrey Experience
	exp3 := core.NewRecord(expColl)
	exp3.Set("company", "Gallifrey Time Lord Academy")
	exp3.Set("position", "Student (Mediocre)")
	exp3.Set("location", "Gallifrey")
	exp3.Set("start_date", "1800-01-01")
	exp3.Set("end_date", "1950-01-01")
	exp3.Set("current", false)
	exp3.Set("description", "Studied temporal mechanics and got mostly Cs")
	exp3.Set("highlights", []string{
		"Graduated (eventually)",
		"Borrowed a TARDIS without permission (still haven't returned it)",
	})
	exp3.Set("visibility", "public")
	app.Save(exp3)

	// Torchwood Experience
	exp4 := core.NewRecord(expColl)
	exp4.Set("company", "Torchwood Institute")
	exp4.Set("position", "Occasional Consultant (Against My Will)")
	exp4.Set("location", "Cardiff, Wales")
	exp4.Set("start_date", "2006-01-01")
	exp4.Set("end_date", "2011-12-31")
	exp4.Set("current", false)
	exp4.Set("description", "Helped clean up temporal rifts and alien incidents when I couldn't avoid them")
	exp4.Set("highlights", []string{
		"Prevented 15 alien invasions through Cardiff Rift",
		"Trained team on proper TARDIS maintenance (they didn't listen)",
		"Established protocol for Weevil containment",
	})
	exp4.Set("visibility", "public")
	app.Save(exp4)

	// Freelance Experience
	exp5 := core.NewRecord(expColl)
	exp5.Set("company", "Self-Employed Universe Saver")
	exp5.Set("position", "Freelance Problem Solver")
	exp5.Set("location", "Everywhere and Everywhen")
	exp5.Set("start_date", "1963-11-23")
	exp5.Set("current", true)
	exp5.Set("description", "Taking on impossible challenges across time and space. No problem too small (unless it's wood-related)")
	exp5.Set("highlights", []string{
		"Defeated Daleks 127 times (and counting)",
		"Saved Christmas at least 3 times",
		"Negotiated peace treaties with 42 alien species",
		"Prevented the heat death of the universe (twice)",
	})
	exp5.Set("visibility", "public")
	app.Save(exp5)

	// Create demo projects
	projColl, _ := app.FindCollectionByNameOrId("demo_projects")

	proj1 := core.NewRecord(projColl)
	proj1.Set("name", "TARDIS Operating System")
	proj1.Set("description", "Complete rewrite of a Type-40 TARDIS OS in Rust (because memory safety across dimensions)")
	proj1.Set("start_date", "2020-01-01")
	proj1.Set("current", true)
	proj1.Set("url", "https://github.com/the-doctor/tardis-os")
	proj1.Set("tech_stack", []string{"Rust", "Quantum Computing", "Temporal Mechanics", "Percussive Maintenance"})
	proj1.Set("highlights", []string{
		"Achieved 99.99% uptime (that 0.01% was when I parked it in a black hole)",
		"Supports infinite passengers in finite space",
	})
	proj1.Set("featured", true)
	proj1.Set("visibility", "public")
	app.Save(proj1)

	proj2 := core.NewRecord(projColl)
	proj2.Set("name", "Sonic Screwdriver SDK")
	proj2.Set("description", "Universal API for sonic technology - works on everything except wood")
	proj2.Set("start_date", "2018-06-01")
	proj2.Set("end_date", "2019-12-31")
	proj2.Set("current", false)
	proj2.Set("github_url", "https://github.com/the-doctor/sonic-sdk")
	proj2.Set("tech_stack", []string{"TypeScript", "Hardware Abstraction", "Sonic Frequencies"})
	proj2.Set("featured", true)
	proj2.Set("visibility", "public")
	app.Save(proj2)

	proj3 := core.NewRecord(projColl)
	proj3.Set("name", "react-timeseries")
	proj3.Set("description", "React hooks for managing state across multiple timelines (surprisingly useful for form validation)")
	proj3.Set("start_date", "2021-03-15")
	proj3.Set("end_date", "2021-08-30")
	proj3.Set("current", false)
	proj3.Set("url", "https://npmjs.com/package/@doctor/react-timeseries")
	proj3.Set("github_url", "https://github.com/the-doctor/react-timeseries")
	proj3.Set("tech_stack", []string{"React", "TypeScript", "Temporal Paradoxes"})
	proj3.Set("featured", false)
	proj3.Set("visibility", "public")
	app.Save(proj3)

	proj4 := core.NewRecord(projColl)
	proj4.Set("name", "Companion Management System")
	proj4.Set("description", "CRM for tracking companions, their timelines, and which version of me they last saw")
	proj4.Set("start_date", "2019-01-01")
	proj4.Set("current", true)
	proj4.Set("tech_stack", []string{"Go", "PostgreSQL", "Sentiment Analysis"})
	proj4.Set("featured", false)
	proj4.Set("visibility", "public")
	app.Save(proj4)

	proj5 := core.NewRecord(projColl)
	proj5.Set("name", "Dalek Detector")
	proj5.Set("description", "ML model trained to identify Daleks in security camera footage. 99.7% accuracy (that 0.3% was a trash can)")
	proj5.Set("start_date", "2017-03-01")
	proj5.Set("end_date", "2017-09-30")
	proj5.Set("current", false)
	proj5.Set("url", "https://github.com/the-doctor/dalek-detector")
	proj5.Set("tech_stack", []string{"Python", "TensorFlow", "OpenCV", "Paranoia"})
	proj5.Set("highlights", []string{
		"Deployed to 847 UNIT facilities worldwide",
		"Prevented 3 invasions by early detection",
	})
	proj5.Set("featured", false)
	proj5.Set("visibility", "public")
	app.Save(proj5)

	proj6 := core.NewRecord(projColl)
	proj6.Set("name", "Temporal Merge Conflicts Resolver")
	proj6.Set("description", "Git plugin that handles merge conflicts across parallel timelines. Works surprisingly well for regular merge conflicts too")
	proj6.Set("start_date", "2020-05-15")
	proj6.Set("current", true)
	proj6.Set("github_url", "https://github.com/the-doctor/git-timey-wimey")
	proj6.Set("tech_stack", []string{"Rust", "Git Internals", "Causality Theory"})
	proj6.Set("highlights", []string{
		"10K+ stars on GitHub",
		"Featured in 'Weird Git Tools' article on Hacker News",
	})
	proj6.Set("featured", true)
	proj6.Set("visibility", "public")
	app.Save(proj6)

	// Create demo education
	eduColl, _ := app.FindCollectionByNameOrId("demo_education")

	edu1 := core.NewRecord(eduColl)
	edu1.Set("institution", "Gallifrey Time Lord Academy")
	edu1.Set("degree", "Certificate of Basic Temporal Theory")
	edu1.Set("field", "Time Lord Studies")
	edu1.Set("start_date", "1800-01-01")
	edu1.Set("end_date", "1950-01-01")
	edu1.Set("description", "Scraped by with passing grades. Excelled at running away from responsibility.")
	edu1.Set("visibility", "public")
	app.Save(edu1)

	edu2 := core.NewRecord(eduColl)
	edu2.Set("institution", "Self-Taught (The Hard Way)")
	edu2.Set("degree", "PhD in Universal Problem Solving")
	edu2.Set("field", "Everything")
	edu2.Set("start_date", "1963-11-23")
	edu2.Set("description", "900+ years of on-the-job training. Mostly running.")
	edu2.Set("visibility", "public")
	app.Save(edu2)

	// Create demo certifications
	certColl, _ := app.FindCollectionByNameOrId("demo_certifications")

	cert1 := core.NewRecord(certColl)
	cert1.Set("name", "Certified Kubernetes Administrator")
	cert1.Set("issuer", "CNCF")
	cert1.Set("issue_date", "2022-06-15")
	cert1.Set("credential_id", "TARDIS-K8S-2022")
	cert1.Set("visibility", "public")
	app.Save(cert1)

	cert2 := core.NewRecord(certColl)
	cert2.Set("name", "AWS Solutions Architect")
	cert2.Set("issuer", "Amazon Web Services")
	cert2.Set("issue_date", "2021-03-20")
	cert2.Set("visibility", "public")
	app.Save(cert2)

	cert3 := core.NewRecord(certColl)
	cert3.Set("name", "License to Save the Universe")
	cert3.Set("issuer", "Shadow Proclamation")
	cert3.Set("issue_date", "1985-01-01")
	cert3.Set("credential_id", "ONCOMING-STORM-001")
	cert3.Set("visibility", "public")
	app.Save(cert3)

	// Create demo skills
	skillColl, _ := app.FindCollectionByNameOrId("demo_skills")
	skills := []struct{ name, category string; prof int }{
		{"Go", "Languages", 95},
		{"Rust", "Languages", 90},
		{"TypeScript", "Languages", 92},
		{"Python", "Languages", 88},
		{"Kubernetes", "DevOps", 93},
		{"Docker", "DevOps", 95},
		{"AWS", "Cloud", 90},
		{"PostgreSQL", "Databases", 87},
		{"React", "Frontend", 85},
		{"System Design", "Architecture", 98},
		{"Crisis Management", "Soft Skills", 100},
		{"Running", "Physical", 100},
		{"Improvisation", "Soft Skills", 100},
		{"Time Travel", "Specialized", 95},
		{"Temporal Mechanics", "Specialized", 92},
		{"Alien Languages", "Languages", 89},
		{"Sonic Technology", "Hardware", 100},
		{"Regeneration", "Personal", 85},
		{"TARDIS Piloting", "Specialized", 73},
		{"Making Friends", "Soft Skills", 94},
		{"Defeating Daleks", "Combat", 91},
	}
	for _, s := range skills {
		skill := core.NewRecord(skillColl)
		skill.Set("name", s.name)
		skill.Set("category", s.category)
		skill.Set("proficiency", s.prof)
		skill.Set("visibility", "public")
		app.Save(skill)
	}

	// Create demo views
	viewColl, _ := app.FindCollectionByNameOrId("demo_views")

	// View 1: Senior Engineer (Public)
	view1 := core.NewRecord(viewColl)
	view1.Set("name", "senior-engineer")
	view1.Set("slug", "senior-engineer")
	view1.Set("description", "Senior Engineer View")
	view1.Set("visibility", "public")
	view1.Set("is_active", true)
	view1.Set("is_default", false)
	view1.Set("hero_headline", "Senior Software Engineer & Time Lord")
	view1.Set("sections", []map[string]interface{}{
		{"type": "experience", "visible": true, "order": 1},
		{"type": "projects", "visible": true, "order": 2},
		{"type": "skills", "visible": true, "order": 3},
	})
	app.Save(view1)

	// View 2: Thought Leader (Public, hilariously trying to be a thought leader)
	view2 := core.NewRecord(viewColl)
	view2.Set("name", "thought-leader")
	view2.Set("slug", "thought-leader")
	view2.Set("description", "Thought Leadership Profile")
	view2.Set("visibility", "public")
	view2.Set("is_active", true)
	view2.Set("is_default", false)
	view2.Set("hero_headline", "Visionary | Disruptor | Universe Saver (47x)")
	view2.Set("hero_summary", "Transforming the impossible into the inevitable through synergistic temporal solutions and bleeding-edge sonic technology. Available for keynotes, consulting, and emergency apocalypse prevention.")
	view2.Set("sections", []map[string]interface{}{
		{"type": "posts", "visible": true, "order": 1},
		{"type": "talks", "visible": true, "order": 2},
		{"type": "awards", "visible": true, "order": 3},
		{"type": "skills", "visible": true, "order": 4},
	})
	app.Save(view2)

	// View 3: Just Projects (Unlisted - will have expired token)
	view3 := core.NewRecord(viewColl)
	view3.Set("name", "projects-only")
	view3.Set("slug", "projects-only")
	view3.Set("description", "Projects Portfolio View")
	view3.Set("visibility", "unlisted")
	view3.Set("is_active", true)
	view3.Set("is_default", false)
	view3.Set("hero_headline", "My Greatest Hits (And A Few Explosions)")
	view3.Set("sections", []map[string]interface{}{
		{"type": "projects", "visible": true, "order": 1},
	})
	app.Save(view3)

	// View 4: Writer/Speaker (Unlisted - will have expired token)
	view4 := core.NewRecord(viewColl)
	view4.Set("name", "content-creator")
	view4.Set("slug", "content-creator")
	view4.Set("description", "Content Creator & Speaker Profile")
	view4.Set("visibility", "unlisted")
	view4.Set("is_active", true)
	view4.Set("is_default", false)
	view4.Set("hero_headline", "Professional Rambler & Occasional Wisdom Dispenser")
	view4.Set("sections", []map[string]interface{}{
		{"type": "posts", "visible": true, "order": 1},
		{"type": "talks", "visible": true, "order": 2},
	})
	app.Save(view4)

	// View 5: The Humble One (Public - trying to downplay achievements, failing miserably)
	view5 := core.NewRecord(viewColl)
	view5.Set("name", "definitely-not-showing-off")
	view5.Set("slug", "humble")
	view5.Set("description", "Totally Humble Profile")
	view5.Set("visibility", "public")
	view5.Set("is_active", true)
	view5.Set("is_default", false)
	view5.Set("hero_headline", "Just a Regular Person Who Occasionally Saves Reality")
	view5.Set("hero_summary", "Nothing special really. Anyone could do what I do if they had 900 years of experience, two hearts, and a time machine. I'm basically entry-level.")
	view5.Set("sections", []map[string]interface{}{
		{"type": "awards", "visible": true, "order": 1},
		{"type": "certifications", "visible": true, "order": 2},
		{"type": "experience", "visible": true, "order": 3},
	})
	app.Save(view5)

	// Create demo posts
	postsColl, _ := app.FindCollectionByNameOrId("demo_posts")

	post1 := core.NewRecord(postsColl)
	post1.Set("title", "That Time I Accidentally Invented Kubernetes")
	post1.Set("slug", "accidentally-invented-kubernetes")
	post1.Set("content", "<p>So there I was, trying to manage containers across multiple timelines...</p>")
	post1.Set("excerpt", "A story about container orchestration gone temporally wrong")
	post1.Set("published_at", "2023-06-15")
	post1.Set("tags", []string{"devops", "kubernetes", "time-travel"})
	post1.Set("is_draft", false)
	post1.Set("visibility", "public")
	app.Save(post1)

	post2 := core.NewRecord(postsColl)
	post2.Set("title", "5 Signs Your Codebase Might Be Sentient")
	post2.Set("slug", "codebase-sentience-signs")
	post2.Set("content", "<p>If your CI/CD pipeline starts making its own decisions...</p>")
	post2.Set("excerpt", "Warning signs that your code has gained consciousness")
	post2.Set("published_at", "2023-09-20")
	post2.Set("tags", []string{"ai", "software-engineering", "existential-dread"})
	post2.Set("is_draft", false)
	post2.Set("visibility", "public")
	app.Save(post2)

	post3 := core.NewRecord(postsColl)
	post3.Set("title", "Why I Don't Use Microservices (I Use Micro-TARDIS-es)")
	post3.Set("slug", "micro-tardises-not-microservices")
	post3.Set("content", "<p>Each service runs in its own pocket dimension. Infinitely scalable, literally.</p>")
	post3.Set("excerpt", "Rethinking distributed systems with dimensional engineering")
	post3.Set("published_at", "2023-11-10")
	post3.Set("tags", []string{"architecture", "distributed-systems", "time-lords"})
	post3.Set("is_draft", false)
	post3.Set("visibility", "public")
	app.Save(post3)

	post4 := core.NewRecord(postsColl)
	post4.Set("title", "The Time I Fixed a Bug By Going Back in Time and Preventing It")
	post4.Set("slug", "temporal-debugging-case-study")
	post4.Set("content", "<p>Why waste time debugging when you can just un-write the bug? A practical guide to timeline-based development.</p>")
	post4.Set("excerpt", "A case study in temporal debugging techniques")
	post4.Set("published_at", "2024-01-15")
	post4.Set("tags", []string{"debugging", "time-travel", "best-practices"})
	post4.Set("is_draft", false)
	post4.Set("visibility", "public")
	app.Save(post4)

	post5 := core.NewRecord(postsColl)
	post5.Set("title", "Code Review Comments From My Past Self (Literally)")
	post5.Set("slug", "past-self-code-reviews")
	post5.Set("content", "<p>Benefits of being able to review your own code from next Tuesday: surprisingly therapeutic, occasionally terrifying.</p>")
	post5.Set("excerpt", "What happens when you can review code across timelines")
	post5.Set("published_at", "2024-03-22")
	post5.Set("tags", []string{"code-review", "team-dynamics", "temporal-mechanics"})
	post5.Set("is_draft", false)
	post5.Set("visibility", "public")
	app.Save(post5)

	post6 := core.NewRecord(postsColl)
	post6.Set("title", "My Failed Attempt at Becoming a DevRel (I Accidentally Started a Cult)")
	post6.Set("slug", "devrel-gone-wrong")
	post6.Set("content", "<p>Turns out when you show people technology that's 900 years advanced, they start worshipping you. This is why I stick to consulting.</p>")
	post6.Set("excerpt", "Lessons learned from my brief and disastrous career in developer relations")
	post6.Set("published_at", "2024-05-08")
	post6.Set("tags", []string{"devrel", "community", "mistakes", "humility"})
	post6.Set("is_draft", false)
	post6.Set("visibility", "public")
	app.Save(post6)

	// Create demo talks
	talksColl, _ := app.FindCollectionByNameOrId("demo_talks")

	talk1 := core.NewRecord(talksColl)
	talk1.Set("title", "Debugging Across Dimensions: A Timey-Wimey Approach")
	talk1.Set("event", "GopherCon 2023")
	talk1.Set("date", "2023-09-15")
	talk1.Set("location", "San Diego, CA")
	talk1.Set("description", "How temporal debugging can help you fix bugs before they happen")
	talk1.Set("slides_url", "https://speakerdeck.com/doctor/debugging-dimensions")
	talk1.Set("is_draft", false)
	talk1.Set("visibility", "public")
	app.Save(talk1)

	talk2 := core.NewRecord(talksColl)
	talk2.Set("title", "Why Your Monitoring Is Terrible (And Mine Monitors The Future)")
	talk2.Set("event", "KubeCon 2023")
	talk2.Set("date", "2023-11-08")
	talk2.Set("location", "Chicago, IL")
	talk2.Set("description", "Predictive monitoring using temporal mechanics")
	talk2.Set("video_url", "https://youtube.com/watch?v=fake")
	talk2.Set("is_draft", false)
	talk2.Set("visibility", "public")
	app.Save(talk2)

	talk3 := core.NewRecord(talksColl)
	talk3.Set("title", "I Tried Pair Programming With Myself From Yesterday. It Didn't Go Well.")
	talk3.Set("event", "Strange Loop 2024")
	talk3.Set("date", "2024-09-12")
	talk3.Set("location", "St. Louis, MO")
	talk3.Set("description", "The unexpected challenges of collaborating across timelines")
	talk3.Set("slides_url", "https://speakerdeck.com/doctor/pair-programming-paradox")
	talk3.Set("video_url", "https://youtube.com/watch?v=also-fake")
	talk3.Set("is_draft", false)
	talk3.Set("visibility", "public")
	app.Save(talk3)

	talk4 := core.NewRecord(talksColl)
	talk4.Set("title", "My Stack: Go, Rust, TypeScript, and a Sonic Screwdriver")
	talk4.Set("event", "JSConf EU 2024")
	talk4.Set("date", "2024-06-05")
	talk4.Set("location", "Berlin, Germany")
	talk4.Set("description", "Building full-stack applications when one of your tools can rewrite molecular structures")
	talk4.Set("slides_url", "https://speakerdeck.com/doctor/sonic-stack")
	talk4.Set("is_draft", false)
	talk4.Set("visibility", "public")
	app.Save(talk4)

	talk5 := core.NewRecord(talksColl)
	talk5.Set("title", "No, Seriously, Stop Using Blockchain For That")
	talk5.Set("event", "DeveloperWeek 2024")
	talk5.Set("date", "2024-02-21")
	talk5.Set("location", "San Francisco, CA")
	talk5.Set("description", "A time traveler's perspective on unnecessarily complicated solutions")
	talk5.Set("is_draft", false)
	talk5.Set("visibility", "public")
	app.Save(talk5)

	// Create demo awards
	awardsColl, _ := app.FindCollectionByNameOrId("demo_awards")

	award1 := core.NewRecord(awardsColl)
	award1.Set("title", "Savior of Earth (x47)")
	award1.Set("issuer", "United Nations")
	award1.Set("awarded_at", "2020-12-25")
	award1.Set("description", "For services rendered in preventing multiple apocalypses")
	award1.Set("visibility", "public")
	app.Save(award1)

	award2 := core.NewRecord(awardsColl)
	award2.Set("title", "Most Likely to Accidentally Destroy Timeline")
	award2.Set("issuer", "Gallifrey High Council")
	award2.Set("awarded_at", "1985-03-15")
	award2.Set("description", "Awarded for creative interpretations of the Laws of Time")
	award2.Set("visibility", "public")
	app.Save(award2)

	award3 := core.NewRecord(awardsColl)
	award3.Set("title", "GitHub Star of the Year")
	award3.Set("issuer", "GitHub")
	award3.Set("awarded_at", "2022-10-20")
	award3.Set("description", "For the Temporal Merge Conflicts Resolver project")
	award3.Set("visibility", "public")
	app.Save(award3)

	award4 := core.NewRecord(awardsColl)
	award4.Set("title", "Worst Employee (Keeps Disappearing)")
	award4.Set("issuer", "Totally Normal Software Company")
	award4.Set("awarded_at", "2023-12-31")
	award4.Set("description", "Given upon termination for 'excessive time off'")
	award4.Set("visibility", "public")
	app.Save(award4)

	// Create share tokens for unlisted views (expired for demo purposes)
	tokensColl, _ := app.FindCollectionByNameOrId("demo_share_tokens")

	// Token for projects-only view (expired)
	token1 := core.NewRecord(tokensColl)
	token1.Set("view_id", view3.Id)
	token1.Set("token", "projects-demo-token-expired")
	token1.Set("label", "Portfolio Share Link")
	token1.Set("expires_at", "2024-01-01T00:00:00.000Z")
	token1.Set("max_uses", 100)
	token1.Set("use_count", 42)
	app.Save(token1)

	// Token for content-creator view (expired)
	token2 := core.NewRecord(tokensColl)
	token2.Set("view_id", view4.Id)
	token2.Set("token", "writer-demo-token-expired")
	token2.Set("label", "Conference Organizer Link")
	token2.Set("expires_at", "2024-06-01T00:00:00.000Z")
	token2.Set("max_uses", 50)
	token2.Set("use_count", 50)
	app.Save(token2)

	// Active token for projects view
	token3 := core.NewRecord(tokensColl)
	token3.Set("view_id", view3.Id)
	token3.Set("token", "projects-active-token")
	token3.Set("label", "Recruiter Access")
	token3.Set("expires_at", "2025-12-31T23:59:59.000Z")
	token3.Set("max_uses", 10)
	token3.Set("use_count", 3)
	app.Save(token3)

	app.Logger().Info("Demo data loaded successfully into shadow tables!")
	return nil
}
