package hooks

import (
	"fmt"
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// backupRecord converts a record to a map preserving all fields including ID
func backupRecord(record *core.Record) map[string]interface{} {
	data := make(map[string]interface{})
	// Get all field values
	for key, value := range record.FieldsData() {
		// Skip internal PocketBase fields
		if key == "collectionId" || key == "collectionName" {
			continue
		}
		data[key] = value
	}
	// Add the record ID
	data["id"] = record.Id
	return data
}

// backupRecords converts multiple records
func backupRecords(records []*core.Record) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		result = append(result, backupRecord(record))
	}
	return result
}

// RegisterDemoHooks registers demo-related API endpoints
func RegisterDemoHooks(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Check demo mode status
		se.Router.GET("/api/demo/status", func(e *core.RequestEvent) error {
			authRecord := e.Auth
			if authRecord == nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Authentication required",
				})
			}

			// Check if user has demo backup record
			backup, err := app.FindFirstRecordByFilter("demo_backup", "user_id = {:userId}", dbx.Params{"userId": authRecord.Id})
			demoMode := false
			if err == nil && backup != nil {
				demoMode = backup.GetBool("demo_mode")
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"demo_mode": demoMode,
			})
		})

		// Enable demo mode (stash current data, load demo data)
		se.Router.POST("/api/demo/enable", func(e *core.RequestEvent) error {
			app.Logger().Info("========== DEMO ENABLE REQUEST ==========")
			authRecord := e.Auth
			if authRecord == nil {
				app.Logger().Error("No auth")
				return e.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Authentication required",
				})
			}
			app.Logger().Info("User authenticated", "user_id", authRecord.Id)

			// Check if already in demo mode
			backup, _ := app.FindFirstRecordByFilter("demo_backup", "user_id = {:userId}", dbx.Params{"userId": authRecord.Id})
			if backup != nil && backup.GetBool("demo_mode") {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "Already in demo mode",
				})
			}

			// Create or update backup record
			backupColl, _ := app.FindCollectionByNameOrId("demo_backup")
			if backup == nil {
				backup = core.NewRecord(backupColl)
				backup.Set("user_id", authRecord.Id)
			}

			// Stash current profile data
			profile, err := app.FindFirstRecordByFilter("profile", "")
			app.Logger().Info("Enabling demo mode", "user_id", authRecord.Id, "has_existing_profile", err == nil)
			if err == nil {
				// Collect all data to backup
				experience, _ := app.FindRecordsByFilter("experience", "", "start_date DESC", 1000, 0)
				projects, _ := app.FindRecordsByFilter("projects", "", "-featured, -start_date", 1000, 0)
				education, _ := app.FindRecordsByFilter("education", "", "start_date DESC", 1000, 0)
				skills, _ := app.FindRecordsByFilter("skills", "", "name ASC", 1000, 0)
				certifications, _ := app.FindRecordsByFilter("certifications", "", "issue_date DESC", 1000, 0)
				posts, _ := app.FindRecordsByFilter("posts", "", "date DESC", 1000, 0)
				talks, _ := app.FindRecordsByFilter("talks", "", "date DESC", 1000, 0)
				awards, _ := app.FindRecordsByFilter("awards", "", "date DESC", 1000, 0)
				views, _ := app.FindRecordsByFilter("views", "", "name ASC", 1000, 0)

				// Serialize records preserving all fields including ID
				backup.Set("profile_data", backupRecord(profile))
				backup.Set("experience_data", backupRecords(experience))
				backup.Set("projects_data", backupRecords(projects))
				backup.Set("education_data", backupRecords(education))
				backup.Set("skills_data", backupRecords(skills))
				backup.Set("certifications_data", backupRecords(certifications))
				backup.Set("posts_data", backupRecords(posts))
				backup.Set("talks_data", backupRecords(talks))
				backup.Set("awards_data", backupRecords(awards))
				backup.Set("views_data", backupRecords(views))

				// Delete current profile and related data
				for _, exp := range experience {
					app.Delete(exp)
				}
				for _, proj := range projects {
					app.Delete(proj)
				}
				for _, edu := range education {
					app.Delete(edu)
				}
				for _, skill := range skills {
					app.Delete(skill)
				}
				for _, cert := range certifications {
					app.Delete(cert)
				}
				for _, post := range posts {
					app.Delete(post)
				}
				for _, talk := range talks {
					app.Delete(talk)
				}
				for _, award := range awards {
					app.Delete(award)
				}
				for _, view := range views {
					app.Delete(view)
				}
				app.Delete(profile)
			}

			// Mark as demo mode
			backup.Set("demo_mode", true)
			if err := app.Save(backup); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to save backup",
				})
			}

			app.Logger().Info("Backup saved",
				"user_id", authRecord.Id,
				"has_profile_data", backup.Get("profile_data") != nil,
				"has_experience_data", backup.Get("experience_data") != nil,
			)

			// Load demo data (without creating demo user)
			if err := loadDemoDataForUser(app); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to load demo data: " + err.Error(),
				})
			}

			return e.JSON(http.StatusOK, map[string]string{
				"message": "Demo mode enabled",
			})
		})

		// Restore original data and disable demo mode
		se.Router.POST("/api/demo/restore", func(e *core.RequestEvent) error {
			app.Logger().Info("========== DEMO RESTORE REQUEST ==========")
			authRecord := e.Auth
			if authRecord == nil {
				app.Logger().Error("No auth")
				return e.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Authentication required",
				})
			}
			app.Logger().Info("User authenticated", "user_id", authRecord.Id)

			// Get backup record
			backup, err := app.FindFirstRecordByFilter("demo_backup", "user_id = {:userId}", dbx.Params{"userId": authRecord.Id})
			if err != nil || backup == nil || !backup.GetBool("demo_mode") {
				app.Logger().Error("Demo restore failed", "error", err, "backup_nil", backup == nil)
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "Not in demo mode",
				})
			}

			app.Logger().Info("Starting demo restore",
				"user_id", authRecord.Id,
				"has_profile_data", backup.Get("profile_data") != nil,
				"has_experience_data", backup.Get("experience_data") != nil,
				"profile_data", backup.Get("profile_data"),
			)

			// Check if user had any data before enabling demo mode
			hasOriginalData := backup.Get("profile_data") != nil ||
				backup.Get("experience_data") != nil ||
				backup.Get("projects_data") != nil ||
				backup.Get("education_data") != nil ||
				backup.Get("skills_data") != nil ||
				backup.Get("certifications_data") != nil ||
				backup.Get("posts_data") != nil ||
				backup.Get("talks_data") != nil ||
				backup.Get("awards_data") != nil ||
				backup.Get("views_data") != nil

			if !hasOriginalData {
				// User had no data before demo mode - keep demo data as their actual data
				app.Logger().Info("No original data found in backup, keeping demo data as user's data")
				// Just delete the backup record and return success
				if err := app.Delete(backup); err != nil {
					return e.JSON(http.StatusInternalServerError, map[string]string{
						"error": "Failed to delete backup",
					})
				}
				return e.JSON(http.StatusOK, map[string]string{
					"message": "Demo data kept as your profile",
				})
			}

			app.Logger().Info("Original data found, proceeding with restore")

			// Delete current demo data
			profile, err := app.FindFirstRecordByFilter("profile", "")
			if err == nil {
				app.Delete(profile)
			}

			// Delete all related data (even if no profile exists)
			experience, _ := app.FindRecordsByFilter("experience", "", "", 1000, 0)
			for _, exp := range experience {
				app.Delete(exp)
			}
			projects, _ := app.FindRecordsByFilter("projects", "", "", 1000, 0)
			for _, proj := range projects {
				app.Delete(proj)
			}
			education, _ := app.FindRecordsByFilter("education", "", "", 1000, 0)
			for _, edu := range education {
				app.Delete(edu)
			}
			skills, _ := app.FindRecordsByFilter("skills", "", "", 1000, 0)
			for _, skill := range skills {
				app.Delete(skill)
			}
			certifications, _ := app.FindRecordsByFilter("certifications", "", "", 1000, 0)
			for _, cert := range certifications {
				app.Delete(cert)
			}
			posts, _ := app.FindRecordsByFilter("posts", "", "", 1000, 0)
			for _, post := range posts {
				app.Delete(post)
			}
			talks, _ := app.FindRecordsByFilter("talks", "", "", 1000, 0)
			for _, talk := range talks {
				app.Delete(talk)
			}
			awards, _ := app.FindRecordsByFilter("awards", "", "", 1000, 0)
			for _, award := range awards {
				app.Delete(award)
			}
			views, _ := app.FindRecordsByFilter("views", "", "", 1000, 0)
			for _, view := range views {
				app.Delete(view)
			}

			// Restore profile from backup
			if profileData := backup.Get("profile_data"); profileData != nil {
				app.Logger().Info("Attempting to restore profile", "profileData_type", fmt.Sprintf("%T", profileData))
				if profileMap, ok := profileData.(map[string]interface{}); ok {
					app.Logger().Info("Profile data is map", "keys", len(profileMap))
					profileColl, err := app.FindCollectionByNameOrId("profile")
					if err != nil {
						return e.JSON(http.StatusInternalServerError, map[string]string{
							"error": "Failed to find profile collection",
						})
					}
					profile := core.NewRecord(profileColl)
					profile.Load(profileMap)
					app.Logger().Info("Loaded profile into record", "name", profile.Get("name"))
					if err := app.Save(profile); err != nil {
						app.Logger().Error("Failed to save profile", "error", err)
						return e.JSON(http.StatusInternalServerError, map[string]string{
							"error": "Failed to restore profile",
						})
					}
					app.Logger().Info("Profile restored successfully")
				} else {
					app.Logger().Warn("Profile data is not a map")
				}
			} else {
				app.Logger().Warn("No profile data in backup")
			}

			// Helper function to restore collections from backup
			restoreCollection := func(collName string, dataKey string) error {
				if items := backup.Get(dataKey); items != nil {
					if itemsSlice, ok := items.([]interface{}); ok {
						coll, err := app.FindCollectionByNameOrId(collName)
						if err != nil {
							return err
						}
						for _, itemData := range itemsSlice {
							if itemMap, ok := itemData.(map[string]interface{}); ok {
								record := core.NewRecord(coll)
								record.Load(itemMap)
								if err := app.Save(record); err != nil {
									return err
								}
							}
						}
					}
				}
				return nil
			}

			// Restore all collections
			restoreCollection("experience", "experience_data")
			restoreCollection("projects", "projects_data")
			restoreCollection("education", "education_data")
			restoreCollection("skills", "skills_data")
			restoreCollection("certifications", "certifications_data")
			restoreCollection("posts", "posts_data")
			restoreCollection("talks", "talks_data")
			restoreCollection("awards", "awards_data")
			restoreCollection("views", "views_data")

			// Delete backup record
			if err := app.Delete(backup); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to delete backup",
				})
			}

			return e.JSON(http.StatusOK, map[string]string{
				"message": "Original data restored",
			})
		})

		return se.Next()
	})
}
