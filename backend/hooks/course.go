package hooks

import (
	"facet/services"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

// RegisterCourseHooks registers course-related API endpoints
func RegisterCourseHooks(app *pocketbase.PocketBase, planConfig *services.PlanConfig, crypto *services.CryptoService, rl *services.RateLimitService) {
	purchaseService := services.NewPurchaseService(crypto)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Feature gate - return 404 when courses feature is disabled
		if !planConfig.HasFeature("courses") {
			return se.Next()
		}

		// GET /api/courses - List published courses
		se.Router.GET("/api/courses", func(e *core.RequestEvent) error {
			records, err := app.FindRecordsByFilter(
				"courses",
				"visibility = 'public' && is_draft = false",
				"sort_order",
				100,
				0,
				nil,
			)

			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch courses"})
			}

			courses := make([]map[string]interface{}, 0, len(records))
			for _, record := range records {
				course := map[string]interface{}{
					"id":              record.Id,
					"title":           record.GetString("title"),
					"slug":            record.GetString("slug"),
					"description":     record.GetString("description"),
					"instructor_name": record.GetString("instructor_name"),
					"access_tier":     record.GetString("access_tier"),
					"price":           record.GetInt("price"),
					"difficulty":      record.GetString("difficulty"),
					"estimated_hours": record.GetFloat("estimated_hours"),
					"tags":                record.Get("tags"),
					"sort_order":          record.GetInt("sort_order"),
					"enable_progress":     record.GetBool("enable_progress"),
					"show_progress_bar":   record.GetBool("show_progress_bar"),
					"require_enrollment":  record.GetBool("require_enrollment"),
					"enable_certificates": record.GetBool("enable_certificates"),
				}

				if libraryURL := record.GetString("cover_image_library_url"); libraryURL != "" {
					course["cover_image_url"] = libraryURL
				} else if coverImage := record.GetString("cover_image"); coverImage != "" {
					course["cover_image_url"] = "/api/files/" + record.Collection().Id + "/" + record.Id + "/" + coverImage
				}

				// Count total lessons for this course
				totalLessons := 0

				moduleRecords, err := app.FindRecordsByFilter(
					"modules",
					"course = {:course}",
					"",
					500,
					0,
					map[string]interface{}{"course": record.Id},
				)

				if err == nil {
					for _, module := range moduleRecords {
						lessonRecords, err := app.FindRecordsByFilter(
							"lessons",
							"module = {:module}",
							"",
							500,
							0,
							map[string]interface{}{"module": module.Id},
						)
						if err == nil {
							totalLessons += len(lessonRecords)
						}
					}
				}

				course["total_lessons"] = totalLessons

				courses = append(courses, course)
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"courses": courses,
			})
		})

		// GET /api/courses/{slug} - Course detail with curriculum
		se.Router.GET("/api/courses/{slug}", func(e *core.RequestEvent) error {
			slug := e.Request.PathValue("slug")

			courseRecords, err := app.FindRecordsByFilter(
				"courses",
				"slug = {:slug}",
				"",
				1,
				0,
				map[string]interface{}{"slug": slug},
			)

			if err != nil || len(courseRecords) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
			}

			course := courseRecords[0]
			visibility := course.GetString("visibility")
			isDraft := course.GetBool("is_draft")

			// Check visibility - private/draft courses require superuser
			if visibility != "public" && visibility != "unlisted" {
				if !isSuperuser(e) {
					return e.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
				}
			}

			if isDraft {
				if !isSuperuser(e) {
					return e.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
				}
			}

			// Check if user has purchased and extract buyer email
			isPurchased := false
			buyerEmail := ""
			if course.GetString("access_tier") == "paid" {
				token := extractPurchaseCookie(e)
				if token != "" {
					purchaseIDs, email, err := purchaseService.ValidatePurchaseJWT(token)
					if err == nil {
						buyerEmail = email
						purchaseRecords, err := app.FindRecordsByFilter(
							"purchases",
							"content_type = 'courses' && content_id = {:id} && status = 'completed'",
							"",
							1,
							0,
							map[string]interface{}{"id": course.Id},
						)
						if err == nil && len(purchaseRecords) > 0 {
							purchase := purchaseRecords[0]
							if purchaseService.HasPurchaseAccess(purchaseIDs, purchase.Id) {
								isPurchased = true
							}
						}
					}
				}
			} else {
				// Free course - check enrollment requirement
				if token := extractPurchaseCookie(e); token != "" {
					_, email, err := purchaseService.ValidatePurchaseJWT(token)
					if err == nil {
						buyerEmail = email
					}
				}

				if course.GetBool("require_enrollment") {
					if buyerEmail != "" {
						enrollRecords, eErr := app.FindRecordsByFilter(
							"course_progress",
							"course = {:course} && buyer_email = {:email}",
							"", 1, 0,
							map[string]interface{}{"course": course.Id, "email": buyerEmail},
						)
						if eErr == nil && len(enrollRecords) > 0 {
							isPurchased = true
						}
					}
				} else {
					isPurchased = true // Free courses without enrollment requirement are always accessible
				}
			}

			response := map[string]interface{}{
				"id":                  course.Id,
				"title":              course.GetString("title"),
				"slug":               course.GetString("slug"),
				"description":        course.GetString("description"),
				"instructor_name":    course.GetString("instructor_name"),
				"instructor_bio":     course.GetString("instructor_bio"),
				"access_tier":        course.GetString("access_tier"),
				"price":              course.GetInt("price"),
				"sequential_access":  course.GetBool("sequential_access"),
				"difficulty":         course.GetString("difficulty"),
				"estimated_hours":    course.GetFloat("estimated_hours"),
				"tags":               course.Get("tags"),
				"learning_outcomes":  course.Get("learning_outcomes"),
				"visibility":          visibility,
				"is_draft":            isDraft,
				"is_purchased":        isPurchased,
				"enable_progress":     course.GetBool("enable_progress"),
				"show_progress_bar":   course.GetBool("show_progress_bar"),
				"require_enrollment":  course.GetBool("require_enrollment"),
				"enable_certificates": course.GetBool("enable_certificates"),
			}

			if libraryURL := course.GetString("cover_image_library_url"); libraryURL != "" {
				response["cover_image_url"] = libraryURL
			} else if coverImage := course.GetString("cover_image"); coverImage != "" {
				response["cover_image_url"] = "/api/files/" + course.Collection().Id + "/" + course.Id + "/" + coverImage
			}

			// Fetch modules with lessons
			moduleRecords, err := app.FindRecordsByFilter(
				"modules",
				"course = {:course}",
				"sort_order",
				500,
				0,
				map[string]interface{}{"course": course.Id},
			)

			if err == nil {
				modules := make([]map[string]interface{}, 0, len(moduleRecords))
				for _, module := range moduleRecords {
					moduleData := map[string]interface{}{
						"id":          module.Id,
						"title":       module.GetString("title"),
						"description": module.GetString("description"),
						"sort_order":  module.GetInt("sort_order"),
					}

					// Check module-level drip lock
					moduleDripLocked := false
					if isPurchased && buyerEmail != "" {
						if locked, unlockDate := checkModuleDripLock(app, course.Id, module, buyerEmail); locked {
							moduleDripLocked = true
							moduleData["drip_locked"] = true
							moduleData["drip_unlock_date"] = unlockDate.Format(time.RFC3339)
						}
					}

					// Fetch lessons for this module
					lessonRecords, err := app.FindRecordsByFilter(
						"lessons",
						"module = {:module}",
						"sort_order",
						500,
						0,
						map[string]interface{}{"module": module.Id},
					)

					if err == nil {
						lessons := make([]map[string]interface{}, 0, len(lessonRecords))
						for _, lesson := range lessonRecords {
							lessonData := map[string]interface{}{
								"id":                lesson.Id,
								"title":             lesson.GetString("title"),
								"content_type":      lesson.GetString("content_type"),
								"estimated_minutes": lesson.GetFloat("estimated_minutes"),
								"sort_order":        lesson.GetInt("sort_order"),
								"is_preview":        lesson.GetBool("is_preview"),
							}

							// Check sequential access enforcement
							sequentialLocked := false
							if course.GetBool("sequential_access") && isPurchased && !moduleDripLocked && buyerEmail != "" {
								if !lesson.GetBool("is_preview") && !canAccessLesson(app, course.Id, lesson, buyerEmail) {
									sequentialLocked = true
									lessonData["sequential_locked"] = true
								}
							}

							// Include full content only for preview lessons or if user has purchased AND module is not drip-locked AND not sequentially locked
							if lesson.GetBool("is_preview") || (isPurchased && !moduleDripLocked && !sequentialLocked) {
								lessonData["content"] = lesson.GetString("content")
								lessonData["video_url"] = lesson.GetString("video_url")

								// Include attachment URLs
								attachments := lesson.GetStringSlice("attachments")
								if len(attachments) > 0 {
									attachmentURLs := make([]map[string]string, 0, len(attachments))
									for _, filename := range attachments {
										attachmentURLs = append(attachmentURLs, map[string]string{
											"filename": filename,
											"url":      "/api/files/" + lesson.Collection().Id + "/" + lesson.Id + "/" + filename,
										})
									}
									lessonData["attachments"] = attachmentURLs
								}
							}

							lessons = append(lessons, lessonData)
						}
						moduleData["lessons"] = lessons
					}

					modules = append(modules, moduleData)
				}
				response["modules"] = modules
			}

			return e.JSON(http.StatusOK, response)
		})

		// GET /api/courses/{slug}/progress - Get buyer's progress
		se.Router.GET("/api/courses/{slug}/progress", func(e *core.RequestEvent) error {
			slug := e.Request.PathValue("slug")

			// Extract buyer email from purchase cookie
			token := extractPurchaseCookie(e)
			if token == "" {
				return respondUnauthorized(e, "authentication required")
			}

			purchaseIDs, buyerEmail, err := purchaseService.ValidatePurchaseJWT(token)
			if err != nil {
				return respondUnauthorized(e, "invalid token")
			}

			// Find course by slug
			courseRecords, err := app.FindRecordsByFilter(
				"courses",
				"slug = {:slug}",
				"",
				1,
				0,
				map[string]interface{}{"slug": slug},
			)

			if err != nil || len(courseRecords) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
			}

			course := courseRecords[0]

			// Verify purchase for paid courses
			if course.GetString("access_tier") == "paid" {
				purchaseRecords, err := app.FindRecordsByFilter(
					"purchases",
					"content_type = 'courses' && content_id = {:id} && status = 'completed'",
					"",
					1,
					0,
					map[string]interface{}{"id": course.Id},
				)
				if err != nil || len(purchaseRecords) == 0 {
					return respondForbidden(e, "purchase required")
				}
				purchase := purchaseRecords[0]
				if !purchaseService.HasPurchaseAccess(purchaseIDs, purchase.Id) {
					return respondForbidden(e, "purchase required")
				}
			}

			// Find course progress
			progressRecords, err := app.FindRecordsByFilter(
				"course_progress",
				"course = {:course} && buyer_email = {:email}",
				"",
				1,
				0,
				map[string]interface{}{"course": course.Id, "email": buyerEmail},
			)

			if err != nil || len(progressRecords) == 0 {
				// No progress yet - return empty
				return e.JSON(http.StatusOK, map[string]interface{}{
					"course_progress": nil,
					"lesson_progress": []interface{}{},
				})
			}

			progress := progressRecords[0]
			courseProgress := map[string]interface{}{
				"progress_percent":  progress.GetFloat("progress_percent"),
				"lessons_completed": progress.GetInt("lessons_completed"),
				"total_lessons":     progress.GetInt("total_lessons"),
				"started_at":        progress.GetDateTime("started_at"),
				"completed_at":      progress.GetDateTime("completed_at"),
				"last_accessed":     progress.GetDateTime("last_accessed"),
				"student_name":      progress.GetString("student_name"),
			}

			// Find lesson progress scoped to this course's lessons
			lessonProgress := make([]map[string]interface{}, 0)

			courseModules, err := app.FindRecordsByFilter(
				"modules",
				"course = {:course}",
				"",
				500,
				0,
				map[string]interface{}{"course": course.Id},
			)
			if err == nil {
				courseLessonIDs := make(map[string]bool)
				for _, mod := range courseModules {
					modLessons, err := app.FindRecordsByFilter(
						"lessons",
						"module = {:module}",
						"",
						500,
						0,
						map[string]interface{}{"module": mod.Id},
					)
					if err == nil {
						for _, l := range modLessons {
							courseLessonIDs[l.Id] = true
						}
					}
				}

				lessonProgressRecords, err := app.FindRecordsByFilter(
					"lesson_progress",
					"buyer_email = {:email}",
					"",
					500,
					0,
					map[string]interface{}{"email": buyerEmail},
				)
				if err == nil {
					for _, lp := range lessonProgressRecords {
						lessonID := lp.GetString("lesson")
						if courseLessonIDs[lessonID] {
							lessonProgress = append(lessonProgress, map[string]interface{}{
								"lesson_id":    lessonID,
								"completed":    lp.GetBool("completed"),
								"completed_at": lp.GetDateTime("completed_at"),
							})
						}
					}
				}
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"course_progress": courseProgress,
				"lesson_progress": lessonProgress,
			})
		})

		// POST /api/courses/{slug}/lessons/{lessonId}/complete - Mark lesson complete
		se.Router.POST("/api/courses/{slug}/lessons/{lessonId}/complete", func(e *core.RequestEvent) error {
			slug := e.Request.PathValue("slug")
			lessonID := e.Request.PathValue("lessonId")

			token := extractPurchaseCookie(e)
			if token == "" {
				return respondUnauthorized(e, "authentication required")
			}

			purchaseIDs, buyerEmail, err := purchaseService.ValidatePurchaseJWT(token)
			if err != nil {
				return respondUnauthorized(e, "invalid token")
			}

			courseRecords, err := app.FindRecordsByFilter(
				"courses",
				"slug = {:slug}",
				"",
				1,
				0,
				map[string]interface{}{"slug": slug},
			)

			if err != nil || len(courseRecords) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
			}

			course := courseRecords[0]

			// Verify purchase (or course is free)
			if course.GetString("access_tier") == "paid" {
				purchaseRecords, err := app.FindRecordsByFilter(
					"purchases",
					"content_type = 'courses' && content_id = {:id} && status = 'completed'",
					"",
					1,
					0,
					map[string]interface{}{"id": course.Id},
				)
				if err != nil || len(purchaseRecords) == 0 {
					return respondForbidden(e, "purchase required")
				}
				purchase := purchaseRecords[0]
				if !purchaseService.HasPurchaseAccess(purchaseIDs, purchase.Id) {
					return respondForbidden(e, "purchase required")
				}
			}

			// Find lesson
			lesson, err := app.FindRecordById("lessons", lessonID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found"})
			}

			// Verify lesson belongs to a module in this course
			lessonModule, err := app.FindRecordById("modules", lesson.GetString("module"))
			if err != nil || lessonModule.GetString("course") != course.Id {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found in this course"})
			}

			// Check module-level drip lock
			if locked, unlockDate := checkModuleDripLock(app, course.Id, lessonModule, buyerEmail); locked {
				return respondJSON(e, http.StatusForbidden, map[string]interface{}{
					"error":            "module is not yet available",
					"drip_locked":      true,
					"drip_unlock_date": unlockDate.Format(time.RFC3339),
				})
			}

			// If sequential access enabled, verify prerequisites
			if course.GetBool("sequential_access") {
				if !canAccessLesson(app, course.Id, lesson, buyerEmail) {
					return respondForbidden(e, "complete previous lessons first")
				}
			}

			// Check if progress tracking is enabled for this course
			if !course.GetBool("enable_progress") {
				return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
			}

			// Create or update lesson progress
			lessonProgressRecords, err := app.FindRecordsByFilter(
				"lesson_progress",
				"lesson = {:lesson} && buyer_email = {:email}",
				"",
				1,
				0,
				map[string]interface{}{"lesson": lessonID, "email": buyerEmail},
			)

			var lessonProgressRecord *core.Record
			if err == nil && len(lessonProgressRecords) > 0 {
				lessonProgressRecord = lessonProgressRecords[0]
			} else {
				collection, err := app.FindCollectionByNameOrId("lesson_progress")
				if err != nil {
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "collection not found"})
				}
				lessonProgressRecord = core.NewRecord(collection)
				lessonProgressRecord.Set("lesson", lessonID)
				lessonProgressRecord.Set("buyer_email", buyerEmail)
			}

			lessonProgressRecord.Set("completed", true)
			lessonProgressRecord.Set("completed_at", time.Now())

			if err := app.Save(lessonProgressRecord); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save progress"})
			}

			// Recalculate course progress
			if err := recalculateCourseProgress(app, course.Id, buyerEmail); err != nil {
				app.Logger().Error("Failed to recalculate course progress", "error", err)
			}

			// Return updated progress
			progressRecords, err := app.FindRecordsByFilter(
				"course_progress",
				"course = {:course} && buyer_email = {:email}",
				"",
				1,
				0,
				map[string]interface{}{"course": course.Id, "email": buyerEmail},
			)

			if err == nil && len(progressRecords) > 0 {
				progress := progressRecords[0]
				updatedCourseProgress := map[string]interface{}{
					"progress_percent":  progress.GetFloat("progress_percent"),
					"lessons_completed": progress.GetInt("lessons_completed"),
					"total_lessons":     progress.GetInt("total_lessons"),
					"started_at":        progress.GetDateTime("started_at"),
					"completed_at":      progress.GetDateTime("completed_at"),
					"last_accessed":     progress.GetDateTime("last_accessed"),
					"student_name":      progress.GetString("student_name"),
				}

				// Build lesson_progress array scoped to this course
				lessonProgressList := make([]map[string]interface{}, 0)
				courseModules, modErr := app.FindRecordsByFilter(
					"modules",
					"course = {:course}",
					"",
					500,
					0,
					map[string]interface{}{"course": course.Id},
				)
				if modErr == nil {
					courseLessonIDs := make(map[string]bool)
					for _, mod := range courseModules {
						modLessons, lErr := app.FindRecordsByFilter(
							"lessons",
							"module = {:module}",
							"",
							500,
							0,
							map[string]interface{}{"module": mod.Id},
						)
						if lErr == nil {
							for _, l := range modLessons {
								courseLessonIDs[l.Id] = true
							}
						}
					}

					lpRecords, lpErr := app.FindRecordsByFilter(
						"lesson_progress",
						"buyer_email = {:email}",
						"",
						500,
						0,
						map[string]interface{}{"email": buyerEmail},
					)
					if lpErr == nil {
						for _, lp := range lpRecords {
							lpLessonID := lp.GetString("lesson")
							if courseLessonIDs[lpLessonID] {
								lessonProgressList = append(lessonProgressList, map[string]interface{}{
									"lesson_id":    lpLessonID,
									"completed":    lp.GetBool("completed"),
									"completed_at": lp.GetDateTime("completed_at"),
								})
							}
						}
					}
				}

				return e.JSON(http.StatusOK, map[string]interface{}{
					"course_progress": updatedCourseProgress,
					"lesson_progress": lessonProgressList,
				})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"course_progress": nil,
				"lesson_progress": []interface{}{},
			})
		})

		// ========================================
		// Admin Curriculum Management Endpoints
		// ========================================

		// GET /api/admin/courses/{courseId}/curriculum
		se.Router.GET("/api/admin/courses/{courseId}/curriculum", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			courseID := e.Request.PathValue("courseId")

			course, err := app.FindRecordById("courses", courseID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
			}

			moduleRecords, err := app.FindRecordsByFilter(
				"modules",
				"course = {:course}",
				"sort_order",
				500,
				0,
				map[string]interface{}{"course": courseID},
			)

			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch modules"})
			}

			modules := make([]map[string]interface{}, 0, len(moduleRecords))
			for _, module := range moduleRecords {
				moduleData := map[string]interface{}{
					"id":                         module.Id,
					"title":                      module.GetString("title"),
					"description":                module.GetString("description"),
					"sort_order":                 module.GetInt("sort_order"),
					"drip_days_after_enrollment": module.GetInt("drip_days_after_enrollment"),
					"drip_unlock_date":           module.GetDateTime("drip_unlock_date"),
				}

				lessonRecords, err := app.FindRecordsByFilter(
					"lessons",
					"module = {:module}",
					"sort_order",
					500,
					0,
					map[string]interface{}{"module": module.Id},
				)

				if err == nil {
					lessons := make([]map[string]interface{}, 0, len(lessonRecords))
					for _, lesson := range lessonRecords {
						lessonData := map[string]interface{}{
							"id":                lesson.Id,
							"title":             lesson.GetString("title"),
							"content_type":      lesson.GetString("content_type"),
							"video_url":         lesson.GetString("video_url"),
							"content":           lesson.GetString("content"),
							"estimated_minutes": lesson.GetFloat("estimated_minutes"),
							"is_preview":        lesson.GetBool("is_preview"),
							"sort_order":        lesson.GetInt("sort_order"),
						}

						attachments := lesson.GetStringSlice("attachments")
						if len(attachments) > 0 {
							attachmentURLs := make([]map[string]string, 0, len(attachments))
							for _, filename := range attachments {
								attachmentURLs = append(attachmentURLs, map[string]string{
									"filename": filename,
									"url":      "/api/files/" + lesson.Collection().Id + "/" + lesson.Id + "/" + filename,
								})
							}
							lessonData["attachments"] = attachmentURLs
						}

						lessons = append(lessons, lessonData)
					}
					moduleData["lessons"] = lessons
				} else {
					moduleData["lessons"] = []interface{}{}
				}

				modules = append(modules, moduleData)
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"course_id": courseID,
				"title":     course.GetString("title"),
				"modules":   modules,
			})
		})

		// POST /api/admin/courses/{courseId}/modules - Create module
		se.Router.POST("/api/admin/courses/{courseId}/modules", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			courseID := e.Request.PathValue("courseId")

			_, err := app.FindRecordById("courses", courseID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
			}

			var req struct {
				Title       string `json:"title"`
				Description string `json:"description"`
				SortOrder   int    `json:"sort_order"`
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			req.Title = strings.TrimSpace(req.Title)
			if req.Title == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "title is required"})
			}
			if len(req.Title) > 255 {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "title must be 255 characters or less"})
			}

			collection, err := app.FindCollectionByNameOrId("modules")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "collection not found"})
			}

			module := core.NewRecord(collection)
			module.Set("course", courseID)
			module.Set("title", req.Title)
			module.Set("description", req.Description)

			sortOrder := req.SortOrder
			if sortOrder == 0 {
				existingModules, err := app.FindRecordsByFilter(
					"modules",
					"course = {:course}",
					"-sort_order",
					1,
					0,
					map[string]interface{}{"course": courseID},
				)
				if err == nil && len(existingModules) > 0 {
					sortOrder = existingModules[0].GetInt("sort_order") + 1
				} else {
					sortOrder = 1
				}
			}
			module.Set("sort_order", sortOrder)

			if err := app.Save(module); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create module"})
			}

			return e.JSON(http.StatusCreated, map[string]interface{}{
				"id":          module.Id,
				"title":       module.GetString("title"),
				"description": module.GetString("description"),
				"sort_order":  module.GetInt("sort_order"),
			})
		})

		// PATCH /api/admin/courses/{courseId}/modules/{moduleId}
		se.Router.PATCH("/api/admin/courses/{courseId}/modules/{moduleId}", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			courseID := e.Request.PathValue("courseId")
			moduleID := e.Request.PathValue("moduleId")

			module, err := app.FindRecordById("modules", moduleID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "module not found"})
			}
			if module.GetString("course") != courseID {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "module does not belong to course"})
			}

			var req map[string]interface{}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			if title, ok := req["title"].(string); ok {
				module.Set("title", title)
			}
			if description, ok := req["description"].(string); ok {
				module.Set("description", description)
			}
			if sortOrder, ok := req["sort_order"].(float64); ok {
				module.Set("sort_order", int(sortOrder))
			}
			if dripDays, ok := req["drip_days_after_enrollment"].(float64); ok {
				module.Set("drip_days_after_enrollment", int(dripDays))
			}
			if dripDate, ok := req["drip_unlock_date"].(string); ok {
				module.Set("drip_unlock_date", dripDate)
			}

			if err := app.Save(module); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update module"})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"id":          module.Id,
				"title":       module.GetString("title"),
				"description": module.GetString("description"),
				"sort_order":  module.GetInt("sort_order"),
			})
		})

		// DELETE /api/admin/courses/{courseId}/modules/{moduleId}
		se.Router.DELETE("/api/admin/courses/{courseId}/modules/{moduleId}", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			courseID := e.Request.PathValue("courseId")
			moduleID := e.Request.PathValue("moduleId")

			module, err := app.FindRecordById("modules", moduleID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "module not found"})
			}
			if module.GetString("course") != courseID {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "module does not belong to course"})
			}

			// Cascade delete lessons
			lessonRecords, err := app.FindRecordsByFilter(
				"lessons",
				"module = {:module}",
				"",
				1000,
				0,
				map[string]interface{}{"module": moduleID},
			)
			if err == nil {
				for _, lesson := range lessonRecords {
					lpRecords, lpErr := app.FindRecordsByFilter(
						"lesson_progress",
						"lesson = {:lesson}",
						"",
						1000,
						0,
						map[string]interface{}{"lesson": lesson.Id},
					)
					if lpErr == nil {
						for _, lp := range lpRecords {
							if err := app.Delete(lp); err != nil {
								app.Logger().Error("Failed to delete lesson_progress during module cascade", "error", err, "lesson_progress_id", lp.Id)
							}
						}
					}
					if err := app.Delete(lesson); err != nil {
						app.Logger().Error("Failed to delete lesson during module cascade", "error", err, "lesson_id", lesson.Id)
					}
				}
			}

			if err := app.Delete(module); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete module"})
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
		})

		// POST /api/admin/courses/{courseId}/modules/{moduleId}/lessons
		se.Router.POST("/api/admin/courses/{courseId}/modules/{moduleId}/lessons", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			courseID := e.Request.PathValue("courseId")
			moduleID := e.Request.PathValue("moduleId")

			module, err := app.FindRecordById("modules", moduleID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "module not found"})
			}
			if module.GetString("course") != courseID {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "module does not belong to course"})
			}

			var req struct {
				Title            string  `json:"title"`
				ContentType      string  `json:"content_type"`
				VideoURL         string  `json:"video_url"`
				Content          string  `json:"content"`
				EstimatedMinutes float64 `json:"estimated_minutes"`
				IsPreview        bool    `json:"is_preview"`
				SortOrder        int     `json:"sort_order"`
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			req.Title = strings.TrimSpace(req.Title)
			if req.Title == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "title is required"})
			}
			if len(req.Title) > 255 {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "title must be 255 characters or less"})
			}
			validContentTypes := map[string]bool{"text": true, "video": true, "quiz": true}
			if req.ContentType != "" && !validContentTypes[req.ContentType] {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "content_type must be text, video, or quiz"})
			}
			if req.EstimatedMinutes < 0 {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "estimated_minutes must be non-negative"})
			}

			collection, err := app.FindCollectionByNameOrId("lessons")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "collection not found"})
			}

			lesson := core.NewRecord(collection)
			lesson.Set("module", moduleID)
			lesson.Set("title", req.Title)
			lesson.Set("content_type", req.ContentType)
			lesson.Set("video_url", req.VideoURL)
			lesson.Set("content", req.Content)
			lesson.Set("estimated_minutes", req.EstimatedMinutes)
			lesson.Set("is_preview", req.IsPreview)

			lessonSortOrder := req.SortOrder
			if lessonSortOrder == 0 {
				existingLessons, err := app.FindRecordsByFilter(
					"lessons",
					"module = {:module}",
					"-sort_order",
					1,
					0,
					map[string]interface{}{"module": moduleID},
				)
				if err == nil && len(existingLessons) > 0 {
					lessonSortOrder = existingLessons[0].GetInt("sort_order") + 1
				} else {
					lessonSortOrder = 1
				}
			}
			lesson.Set("sort_order", lessonSortOrder)

			if err := app.Save(lesson); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create lesson"})
			}

			return e.JSON(http.StatusCreated, map[string]interface{}{
				"id":                lesson.Id,
				"title":             lesson.GetString("title"),
				"content_type":      lesson.GetString("content_type"),
				"video_url":         lesson.GetString("video_url"),
				"content":           lesson.GetString("content"),
				"estimated_minutes": lesson.GetFloat("estimated_minutes"),
				"is_preview":        lesson.GetBool("is_preview"),
				"sort_order":        lesson.GetInt("sort_order"),
			})
		})

		// PATCH /api/admin/courses/{courseId}/modules/{moduleId}/lessons/{lessonId}
		se.Router.PATCH("/api/admin/courses/{courseId}/modules/{moduleId}/lessons/{lessonId}", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			courseID := e.Request.PathValue("courseId")
			moduleID := e.Request.PathValue("moduleId")
			lessonID := e.Request.PathValue("lessonId")

			lesson, err := app.FindRecordById("lessons", lessonID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found"})
			}
			if lesson.GetString("module") != moduleID {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "lesson does not belong to module"})
			}
			module, err := app.FindRecordById("modules", moduleID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "module not found"})
			}
			if module.GetString("course") != courseID {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "module does not belong to course"})
			}

			var req map[string]interface{}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			if title, ok := req["title"].(string); ok {
				lesson.Set("title", title)
			}
			if contentType, ok := req["content_type"].(string); ok {
				lesson.Set("content_type", contentType)
			}
			if videoURL, ok := req["video_url"].(string); ok {
				lesson.Set("video_url", videoURL)
			}
			if content, ok := req["content"].(string); ok {
				lesson.Set("content", content)
			}
			if estimatedMinutes, ok := req["estimated_minutes"].(float64); ok {
				lesson.Set("estimated_minutes", estimatedMinutes)
			}
			if isPreview, ok := req["is_preview"].(bool); ok {
				lesson.Set("is_preview", isPreview)
			}
			if sortOrder, ok := req["sort_order"].(float64); ok {
				lesson.Set("sort_order", int(sortOrder))
			}

			if err := app.Save(lesson); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update lesson"})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"id":                lesson.Id,
				"title":             lesson.GetString("title"),
				"content_type":      lesson.GetString("content_type"),
				"video_url":         lesson.GetString("video_url"),
				"content":           lesson.GetString("content"),
				"estimated_minutes": lesson.GetFloat("estimated_minutes"),
				"is_preview":        lesson.GetBool("is_preview"),
				"sort_order":        lesson.GetInt("sort_order"),
			})
		})

		// DELETE /api/admin/courses/{courseId}/modules/{moduleId}/lessons/{lessonId}
		se.Router.DELETE("/api/admin/courses/{courseId}/modules/{moduleId}/lessons/{lessonId}", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			courseID := e.Request.PathValue("courseId")
			moduleID := e.Request.PathValue("moduleId")
			lessonID := e.Request.PathValue("lessonId")

			lesson, err := app.FindRecordById("lessons", lessonID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found"})
			}
			if lesson.GetString("module") != moduleID {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "lesson does not belong to module"})
			}
			module, err := app.FindRecordById("modules", moduleID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "module not found"})
			}
			if module.GetString("course") != courseID {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "module does not belong to course"})
			}

			// Delete lesson_progress records
			lpRecords, lpErr := app.FindRecordsByFilter(
				"lesson_progress",
				"lesson = {:lesson}",
				"",
				1000,
				0,
				map[string]interface{}{"lesson": lessonID},
			)
			if lpErr == nil {
				for _, lp := range lpRecords {
					if err := app.Delete(lp); err != nil {
						app.Logger().Error("Failed to delete lesson_progress during lesson delete", "error", err, "lesson_progress_id", lp.Id)
					}
				}
			}

			if err := app.Delete(lesson); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete lesson"})
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
		})

		// PUT /api/admin/courses/{courseId}/modules/reorder
		se.Router.PUT("/api/admin/courses/{courseId}/modules/reorder", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			courseID := e.Request.PathValue("courseId")

			_, err := app.FindRecordById("courses", courseID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
			}

			var req []struct {
				ID        string `json:"id"`
				SortOrder int    `json:"sort_order"`
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			for _, item := range req {
				module, err := app.FindRecordById("modules", item.ID)
				if err != nil {
					continue
				}
				if module.GetString("course") != courseID {
					continue
				}
				module.Set("sort_order", item.SortOrder)
				if err := app.Save(module); err != nil {
					app.Logger().Error("Failed to reorder module", "error", err, "module_id", item.ID)
				}
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "reordered"})
		})

		// PUT /api/admin/courses/{courseId}/modules/{moduleId}/lessons/reorder
		se.Router.PUT("/api/admin/courses/{courseId}/modules/{moduleId}/lessons/reorder", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			courseID := e.Request.PathValue("courseId")
			moduleID := e.Request.PathValue("moduleId")

			module, err := app.FindRecordById("modules", moduleID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "module not found"})
			}
			if module.GetString("course") != courseID {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "module does not belong to course"})
			}

			var req []struct {
				ID        string `json:"id"`
				SortOrder int    `json:"sort_order"`
			}

			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			for _, item := range req {
				lesson, err := app.FindRecordById("lessons", item.ID)
				if err != nil {
					continue
				}
				if lesson.GetString("module") != moduleID {
					continue
				}
				lesson.Set("sort_order", item.SortOrder)
				if err := app.Save(lesson); err != nil {
					app.Logger().Error("Failed to reorder lesson", "error", err, "lesson_id", item.ID)
				}
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "reordered"})
		})

		// GET /api/certificates/verify/{id}
		se.Router.GET("/api/certificates/verify/{id}", func(e *core.RequestEvent) error {
			certificateID := e.Request.PathValue("id")

			certRecords, err := app.FindRecordsByFilter(
				"course_certificates",
				"certificate_id = {:id}",
				"",
				1,
				0,
				map[string]interface{}{"id": certificateID},
			)

			if err != nil || len(certRecords) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "certificate not found"})
			}

			cert := certRecords[0]
			return e.JSON(http.StatusOK, map[string]interface{}{
				"certificate_id":  cert.GetString("certificate_id"),
				"course_title":    cert.GetString("course_title"),
				"instructor_name": cert.GetString("instructor_name"),
				"student_name":    cert.GetString("student_name"),
				"buyer_email":     cert.GetString("buyer_email"),
				"issued_at":       cert.GetDateTime("issued_at"),
			})
		})

		// GET /api/courses/{slug}/certificate
		se.Router.GET("/api/courses/{slug}/certificate", func(e *core.RequestEvent) error {
			slug := e.Request.PathValue("slug")

			token := extractPurchaseCookie(e)
			if token == "" {
				return respondUnauthorized(e, "authentication required")
			}

			_, buyerEmail, err := purchaseService.ValidatePurchaseJWT(token)
			if err != nil {
				return respondUnauthorized(e, "invalid token")
			}

			courseRecords, err := app.FindRecordsByFilter(
				"courses",
				"slug = {:slug}",
				"",
				1,
				0,
				map[string]interface{}{"slug": slug},
			)

			if err != nil || len(courseRecords) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
			}

			course := courseRecords[0]

			certRecords, err := app.FindRecordsByFilter(
				"course_certificates",
				"course = {:course} && buyer_email = {:email}",
				"",
				1,
				0,
				map[string]interface{}{"course": course.Id, "email": buyerEmail},
			)

			if err != nil || len(certRecords) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "certificate not found"})
			}

			cert := certRecords[0]
			return e.JSON(http.StatusOK, map[string]interface{}{
				"certificate_id":  cert.GetString("certificate_id"),
				"course_title":    cert.GetString("course_title"),
				"instructor_name": cert.GetString("instructor_name"),
				"student_name":    cert.GetString("student_name"),
				"buyer_email":     cert.GetString("buyer_email"),
				"issued_at":       cert.GetDateTime("issued_at"),
			})
		})

		// GET /api/admin/courses/{courseId}/students
		se.Router.GET("/api/admin/courses/{courseId}/students", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			courseID := e.Request.PathValue("courseId")

			_, err := app.FindRecordById("courses", courseID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
			}

			// Count total lessons
			moduleRecords, err := app.FindRecordsByFilter(
				"modules",
				"course = {:course}",
				"",
				500,
				0,
				map[string]interface{}{"course": courseID},
			)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch modules"})
			}

			totalLessons := 0
			for _, module := range moduleRecords {
				lessonRecords, err := app.FindRecordsByFilter(
					"lessons",
					"module = {:module}",
					"",
					500,
					0,
					map[string]interface{}{"module": module.Id},
				)
				if err == nil {
					totalLessons += len(lessonRecords)
				}
			}

			progressRecords, err := app.FindRecordsByFilter(
				"course_progress",
				"course = {:course}",
				"-last_accessed",
				500,
				0,
				map[string]interface{}{"course": courseID},
			)

			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch progress"})
			}

			students := make([]map[string]interface{}, 0, len(progressRecords))
			totalProgress := 0.0
			completedCount := 0

			for _, progress := range progressRecords {
				buyerEmail := progress.GetString("buyer_email")
				progressPercent := progress.GetFloat("progress_percent")
				totalProgress += progressPercent

				if progressPercent >= 100 {
					completedCount++
				}

				hasCertificate := false
				certRecords, err := app.FindRecordsByFilter(
					"course_certificates",
					"course = {:course} && buyer_email = {:email}",
					"",
					1,
					0,
					map[string]interface{}{"course": courseID, "email": buyerEmail},
				)
				if err == nil && len(certRecords) > 0 {
					hasCertificate = true
				}

				studentInfo := map[string]interface{}{
					"email":             buyerEmail,
					"student_name":      progress.GetString("student_name"),
					"progress_percent":  progressPercent,
					"lessons_completed": progress.GetInt("lessons_completed"),
					"total_lessons":     totalLessons,
					"started_at":        progress.GetDateTime("started_at"),
					"completed_at":      progress.GetDateTime("completed_at"),
					"last_accessed":     progress.GetDateTime("last_accessed"),
					"has_certificate":   hasCertificate,
				}
				students = append(students, studentInfo)
			}

			totalEnrolled := len(progressRecords)
			averageProgress := 0.0
			if totalEnrolled > 0 {
				averageProgress = totalProgress / float64(totalEnrolled)
			}
			completionRate := 0.0
			if totalEnrolled > 0 {
				completionRate = (float64(completedCount) / float64(totalEnrolled)) * 100
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"students": students,
				"stats": map[string]interface{}{
					"total_enrolled":   totalEnrolled,
					"completed":        completedCount,
					"average_progress": averageProgress,
					"completion_rate":  completionRate,
				},
			})
		})

		// POST /api/courses/{slug}/enroll - Enroll in a free course
		se.Router.POST("/api/courses/{slug}/enroll", RateLimitMiddleware(rl, "strict")(func(e *core.RequestEvent) error {
			slug := e.Request.PathValue("slug")

			var req struct {
				Email string `json:"email"`
				Name  string `json:"name"`
			}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			}

			req.Email = strings.TrimSpace(req.Email)
			if req.Email == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "email is required"})
			}
			emailRegex := regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
			if !emailRegex.MatchString(req.Email) {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid email format"})
			}

			req.Name = strings.TrimSpace(req.Name)

			courseRecords, err := app.FindRecordsByFilter(
				"courses",
				"slug = {:slug}",
				"",
				1,
				0,
				map[string]interface{}{"slug": slug},
			)
			if err != nil || len(courseRecords) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
			}

			course := courseRecords[0]

			if course.GetString("access_tier") == "paid" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "this course requires purchase"})
			}

			existingRecords, err := app.FindRecordsByFilter(
				"course_progress",
				"course = {:course} && buyer_email = {:email}",
				"",
				1,
				0,
				map[string]interface{}{"course": course.Id, "email": req.Email},
			)

			alreadyEnrolled := err == nil && len(existingRecords) > 0

			if !alreadyEnrolled {
				progressCollection, err := app.FindCollectionByNameOrId("course_progress")
				if err != nil {
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to find progress collection"})
				}

				progress := core.NewRecord(progressCollection)
				progress.Set("course", course.Id)
				progress.Set("buyer_email", req.Email)
				progress.Set("started_at", time.Now())
				progress.Set("last_accessed", time.Now())
				progress.Set("progress_percent", 0)
				progress.Set("lessons_completed", 0)

				totalLessons := 0
				moduleRecords, mErr := app.FindRecordsByFilter(
					"modules",
					"course = {:course}",
					"",
					500,
					0,
					map[string]interface{}{"course": course.Id},
				)
				if mErr == nil {
					for _, mod := range moduleRecords {
						lessonRecords, lErr := app.FindRecordsByFilter(
							"lessons",
							"module = {:module}",
							"",
							500,
							0,
							map[string]interface{}{"module": mod.Id},
						)
						if lErr == nil {
							totalLessons += len(lessonRecords)
						}
					}
				}
				progress.Set("total_lessons", totalLessons)

				if req.Name != "" {
					progress.Set("student_name", req.Name)
				}

				if err := app.Save(progress); err != nil {
					if strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "duplicate key") {
						alreadyEnrolled = true
					} else {
						return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create enrollment"})
					}
				}
			}

			jwtToken, expiresAt, err := purchaseService.GenerateEnrollmentJWT(course.Id, req.Email)
			if err != nil {
				app.Logger().Error("Failed to generate enrollment JWT", "error", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
			}

			http.SetCookie(e.Response, &http.Cookie{
				Name:     services.PurchaseCookieName,
				Value:    jwtToken,
				Path:     "/",
				Expires:  expiresAt,
				HttpOnly: true,
				Secure:   isRequestSecure(e),
				SameSite: http.SameSiteLaxMode,
			})

			status := "enrolled"
			if alreadyEnrolled {
				status = "already_enrolled"
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"status":       status,
				"redirect_url": fmt.Sprintf("/courses/%s/learn", slug),
			})
		}))

		// GET /api/my-courses
		se.Router.GET("/api/my-courses", RateLimitMiddleware(rl, "normal")(func(e *core.RequestEvent) error {
			token := extractPurchaseCookie(e)
			if token == "" {
				return respondUnauthorized(e, "authentication required")
			}

			_, buyerEmail, err := purchaseService.ValidatePurchaseJWT(token)
			if err != nil {
				return respondUnauthorized(e, "invalid token")
			}

			progressRecords, err := app.FindRecordsByFilter(
				"course_progress",
				"buyer_email = {:email}",
				"-last_accessed",
				100,
				0,
				map[string]interface{}{"email": buyerEmail},
			)

			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch progress"})
			}

			courses := make([]map[string]interface{}, 0, len(progressRecords))

			for _, progress := range progressRecords {
				courseID := progress.GetString("course")
				course, err := app.FindRecordById("courses", courseID)
				if err != nil {
					continue
				}

				courseData := map[string]interface{}{
					"id":                course.Id,
					"title":             course.GetString("title"),
					"slug":              course.GetString("slug"),
					"instructor_name":   course.GetString("instructor_name"),
					"progress_percent":  progress.GetFloat("progress_percent"),
					"lessons_completed": progress.GetInt("lessons_completed"),
					"total_lessons":     progress.GetInt("total_lessons"),
					"last_accessed":     progress.GetDateTime("last_accessed"),
				}

				if libraryURL := course.GetString("cover_image_library_url"); libraryURL != "" {
					courseData["cover_image"] = libraryURL
				} else if coverImage := course.GetString("cover_image"); coverImage != "" {
					courseData["cover_image"] = "/api/files/" + course.Collection().Id + "/" + course.Id + "/" + coverImage
				}

				hasCertificate := false
				certRecords, certErr := app.FindRecordsByFilter(
					"course_certificates",
					"course = {:course} && buyer_email = {:email}",
					"",
					1,
					0,
					map[string]interface{}{"course": courseID, "email": buyerEmail},
				)
				if certErr == nil && len(certRecords) > 0 {
					hasCertificate = true
				}
				courseData["has_certificate"] = hasCertificate

				courses = append(courses, courseData)
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"courses": courses,
			})
		}))

		// GET /api/admin/courses/{courseId}/analytics
		se.Router.GET("/api/admin/courses/{courseId}/analytics", RateLimitMiddleware(rl, "normal")(func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			courseID := e.Request.PathValue("courseId")

			_, err := app.FindRecordById("courses", courseID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
			}

			progressRecords, err := app.FindRecordsByFilter(
				"course_progress",
				"course = {:course}",
				"",
				10000,
				0,
				map[string]interface{}{"course": courseID},
			)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch progress"})
			}

			totalEnrolled := len(progressRecords)
			activeStudents := 0
			completedCount := 0
			totalProgress := 0.0
			thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

			for _, progress := range progressRecords {
				progressPercent := progress.GetFloat("progress_percent")
				totalProgress += progressPercent

				if progressPercent >= 100 {
					completedCount++
				}

				lastAccessed := progress.GetDateTime("last_accessed")
				if !lastAccessed.IsZero() && lastAccessed.Time().After(thirtyDaysAgo) {
					activeStudents++
				}
			}

			averageProgress := 0.0
			if totalEnrolled > 0 {
				averageProgress = totalProgress / float64(totalEnrolled)
			}
			completionRate := 0.0
			if totalEnrolled > 0 {
				completionRate = (float64(completedCount) / float64(totalEnrolled)) * 100
			}

			purchaseRecords, err := app.FindRecordsByFilter(
				"purchases",
				"content_type = 'courses' && content_id = {:id} && status = 'completed'",
				"",
				10000,
				0,
				map[string]interface{}{"id": courseID},
			)
			totalCourseRevenue := 0
			if err == nil {
				for _, p := range purchaseRecords {
					totalCourseRevenue += p.GetInt("amount")
				}
			}

			moduleRecords, err := app.FindRecordsByFilter(
				"modules",
				"course = {:course}",
				"sort_order",
				500,
				0,
				map[string]interface{}{"course": courseID},
			)

			lessonCompletionRates := make([]map[string]interface{}, 0)

			if err == nil {
				for _, module := range moduleRecords {
					moduleTitle := module.GetString("title")

					lessonRecords, lErr := app.FindRecordsByFilter(
						"lessons",
						"module = {:module}",
						"sort_order",
						500,
						0,
						map[string]interface{}{"module": module.Id},
					)

					if lErr != nil {
						continue
					}

					for _, lesson := range lessonRecords {
						completedLessonCount := 0
						if totalEnrolled > 0 {
							lpRecords, lpErr := app.FindRecordsByFilter(
								"lesson_progress",
								"lesson = {:lesson} && completed = true",
								"",
								10000,
								0,
								map[string]interface{}{"lesson": lesson.Id},
							)
							if lpErr == nil {
								completedLessonCount = len(lpRecords)
							}
						}

						lessonRate := 0.0
						if totalEnrolled > 0 {
							lessonRate = (float64(completedLessonCount) / float64(totalEnrolled)) * 100
						}

						lessonCompletionRates = append(lessonCompletionRates, map[string]interface{}{
							"lesson_id":       lesson.Id,
							"lesson_title":    lesson.GetString("title"),
							"module_title":    moduleTitle,
							"completion_rate": lessonRate,
						})
					}
				}
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"total_enrolled":          totalEnrolled,
				"active_students":         activeStudents,
				"completion_rate":         completionRate,
				"average_progress":        averageProgress,
				"total_revenue":           totalCourseRevenue,
				"lesson_completion_rates": lessonCompletionRates,
			})
		}))

		return se.Next()
	})

	// Cascade delete: when a course is deleted, remove modules, lessons, progress, and related records
	app.OnRecordAfterDeleteSuccess("courses").BindFunc(func(e *core.RecordEvent) error {
		courseID := e.Record.Id

		if cpRecords, err := app.FindRecordsByFilter("course_progress", "course = {:id}", "", 1000, 0, map[string]interface{}{"id": courseID}); err == nil {
			for _, cp := range cpRecords {
				if err := app.Delete(cp); err != nil {
					app.Logger().Error("Failed to cascade delete course_progress", "error", err, "id", cp.Id)
				}
			}
		}

		if certRecords, err := app.FindRecordsByFilter("course_certificates", "course = {:id}", "", 1000, 0, map[string]interface{}{"id": courseID}); err == nil {
			for _, cert := range certRecords {
				if err := app.Delete(cert); err != nil {
					app.Logger().Error("Failed to cascade delete course_certificate", "error", err, "id", cert.Id)
				}
			}
		}

		moduleRecords, err := app.FindRecordsByFilter("modules", "course = {:id}", "", 1000, 0, map[string]interface{}{"id": courseID})
		if err == nil {
			for _, module := range moduleRecords {
				if lessons, err := app.FindRecordsByFilter("lessons", "module = {:id}", "", 1000, 0, map[string]interface{}{"id": module.Id}); err == nil {
					for _, lesson := range lessons {
						if quizzes, err := app.FindRecordsByFilter("quizzes", "lesson = {:id}", "", 1000, 0, map[string]interface{}{"id": lesson.Id}); err == nil {
							for _, quiz := range quizzes {
								if attempts, err := app.FindRecordsByFilter("quiz_attempts", "quiz = {:id}", "", 1000, 0, map[string]interface{}{"id": quiz.Id}); err == nil {
									for _, attempt := range attempts {
										if err := app.Delete(attempt); err != nil {
											app.Logger().Error("Failed to cascade delete quiz_attempt", "error", err, "id", attempt.Id)
										}
									}
								}
								if questions, err := app.FindRecordsByFilter("quiz_questions", "quiz = {:id}", "", 1000, 0, map[string]interface{}{"id": quiz.Id}); err == nil {
									for _, question := range questions {
										if err := app.Delete(question); err != nil {
											app.Logger().Error("Failed to cascade delete quiz_question", "error", err, "id", question.Id)
										}
									}
								}
								if err := app.Delete(quiz); err != nil {
									app.Logger().Error("Failed to cascade delete quiz", "error", err, "id", quiz.Id)
								}
							}
						}

						if lpRecords, err := app.FindRecordsByFilter("lesson_progress", "lesson = {:id}", "", 1000, 0, map[string]interface{}{"id": lesson.Id}); err == nil {
							for _, lp := range lpRecords {
								_ = app.Delete(lp)
							}
						}
						_ = app.Delete(lesson)
					}
				}
				_ = app.Delete(module)
			}
		}

		return e.Next()
	})
}

// canAccessLesson checks if buyer can access a lesson when sequential access is enabled
func canAccessLesson(app core.App, courseID string, lesson *core.Record, buyerEmail string) bool {
	moduleID := lesson.GetString("module")
	lessonSortOrder := lesson.GetInt("sort_order")

	module, err := app.FindRecordById("modules", moduleID)
	if err != nil {
		return false
	}
	moduleSortOrder := module.GetInt("sort_order")

	moduleRecords, err := app.FindRecordsByFilter(
		"modules",
		"course = {:course}",
		"sort_order",
		500,
		0,
		map[string]interface{}{"course": courseID},
	)
	if err != nil {
		return false
	}

	for _, m := range moduleRecords {
		if m.GetInt("sort_order") < moduleSortOrder {
			lessonRecords, err := app.FindRecordsByFilter(
				"lessons",
				"module = {:module}",
				"sort_order",
				500,
				0,
				map[string]interface{}{"module": m.Id},
			)
			if err != nil {
				return false
			}
			for _, l := range lessonRecords {
				if !isLessonCompleted(app, l.Id, buyerEmail) {
					return false
				}
			}
		} else if m.Id == moduleID {
			lessonRecords, err := app.FindRecordsByFilter(
				"lessons",
				"module = {:module} && sort_order < {:sort}",
				"sort_order",
				500,
				0,
				map[string]interface{}{"module": moduleID, "sort": lessonSortOrder},
			)
			if err != nil {
				return false
			}
			for _, l := range lessonRecords {
				if !isLessonCompleted(app, l.Id, buyerEmail) {
					return false
				}
			}
		}
	}

	return true
}

// isLessonCompleted checks if a lesson is completed by a buyer
func isLessonCompleted(app core.App, lessonID, buyerEmail string) bool {
	records, err := app.FindRecordsByFilter(
		"lesson_progress",
		"lesson = {:lesson} && buyer_email = {:email} && completed = true",
		"",
		1,
		0,
		map[string]interface{}{"lesson": lessonID, "email": buyerEmail},
	)
	return err == nil && len(records) > 0
}

// recalculateCourseProgress recalculates and updates course progress for a buyer
func recalculateCourseProgress(app core.App, courseID, buyerEmail string) error {
	moduleRecords, err := app.FindRecordsByFilter(
		"modules",
		"course = {:course}",
		"",
		500,
		0,
		map[string]interface{}{"course": courseID},
	)
	if err != nil {
		return err
	}

	moduleIDs := make([]string, 0, len(moduleRecords))
	for _, m := range moduleRecords {
		moduleIDs = append(moduleIDs, m.Id)
	}

	if len(moduleIDs) == 0 {
		return nil
	}

	var filterParts []string
	params := make(map[string]interface{})
	for i, id := range moduleIDs {
		key := fmt.Sprintf("mod%d", i)
		filterParts = append(filterParts, "module = {:"+key+"}")
		params[key] = id
	}
	filter := "(" + filterParts[0]
	for i := 1; i < len(filterParts); i++ {
		filter += " || " + filterParts[i]
	}
	filter += ")"

	lessonRecords, err := app.FindRecordsByFilter("lessons", filter, "", 500, 0, params)
	if err != nil {
		return err
	}
	totalLessons := len(lessonRecords)

	completedCount := 0
	for _, lesson := range lessonRecords {
		if isLessonCompleted(app, lesson.Id, buyerEmail) {
			completedCount++
		}
	}

	progressPercent := float64(0)
	if totalLessons > 0 {
		progressPercent = (float64(completedCount) / float64(totalLessons)) * 100
	}

	progressRecords, err := app.FindRecordsByFilter(
		"course_progress",
		"course = {:course} && buyer_email = {:email}",
		"",
		1,
		0,
		map[string]interface{}{"course": courseID, "email": buyerEmail},
	)

	var progress *core.Record
	now := time.Now()
	if err == nil && len(progressRecords) > 0 {
		progress = progressRecords[0]
	} else {
		collection, err := app.FindCollectionByNameOrId("course_progress")
		if err != nil {
			return err
		}
		progress = core.NewRecord(collection)
		progress.Set("course", courseID)
		progress.Set("buyer_email", buyerEmail)
		progress.Set("started_at", now)
	}

	progress.Set("progress_percent", progressPercent)
	progress.Set("lessons_completed", completedCount)
	progress.Set("total_lessons", totalLessons)
	progress.Set("last_accessed", now)

	if progressPercent >= 100 && progress.GetDateTime("completed_at").IsZero() {
		progress.Set("completed_at", now)

		course, err := app.FindRecordById("courses", courseID)
		if err == nil && course.GetBool("enable_certificates") {
			go func() {
				issueCertificate(app, courseID, buyerEmail)
			}()
		}
	}

	return app.Save(progress)
}

// issueCertificate creates a certificate for a completed course
func issueCertificate(app core.App, courseID, buyerEmail string) {
	existing, err := app.FindRecordsByFilter(
		"course_certificates",
		"course = {:course} && buyer_email = {:email}",
		"", 1, 0,
		map[string]interface{}{"course": courseID, "email": buyerEmail},
	)
	if err == nil && len(existing) > 0 {
		return
	}

	course, err := app.FindRecordById("courses", courseID)
	if err != nil {
		app.Logger().Error("Failed to find course for certificate", "error", err)
		return
	}

	certCollection, err := app.FindCollectionByNameOrId("course_certificates")
	if err != nil {
		app.Logger().Error("Failed to find certificates collection", "error", err)
		return
	}

	cert := core.NewRecord(certCollection)
	cert.Set("course", courseID)
	cert.Set("buyer_email", buyerEmail)
	cert.Set("certificate_id", security.RandomString(32))
	cert.Set("course_title", course.GetString("title"))
	cert.Set("instructor_name", course.GetString("instructor_name"))

	progressRecords, pErr := app.FindRecordsByFilter(
		"course_progress",
		"course = {:course} && buyer_email = {:email}",
		"", 1, 0,
		map[string]interface{}{"course": courseID, "email": buyerEmail},
	)
	if pErr == nil && len(progressRecords) > 0 {
		if studentName := progressRecords[0].GetString("student_name"); studentName != "" {
			cert.Set("student_name", studentName)
		}
	}

	if err := app.Save(cert); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "duplicate key") {
			app.Logger().Info("Certificate already issued (concurrent request)", "course", courseID, "email", buyerEmail)
			return
		}
		app.Logger().Error("Failed to issue certificate", "error", err)
	}
}

// checkModuleDripLock checks if a module is drip-locked for a buyer
func checkModuleDripLock(app core.App, courseID string, module *core.Record, buyerEmail string) (bool, time.Time) {
	dripDate := module.GetDateTime("drip_unlock_date")
	if !dripDate.IsZero() && dripDate.Time().After(time.Now()) {
		return true, dripDate.Time()
	}

	dripDays := module.GetInt("drip_days_after_enrollment")
	if dripDays > 0 {
		progressRecords, err := app.FindRecordsByFilter(
			"course_progress",
			"course = {:course} && buyer_email = {:email}",
			"",
			1,
			0,
			map[string]interface{}{"course": courseID, "email": buyerEmail},
		)

		if err != nil || len(progressRecords) == 0 {
			purchaseRecords, err := app.FindRecordsByFilter(
				"purchases",
				"content_type = 'courses' && content_id = {:id} && buyer_email = {:email} && status = 'completed'",
				"created",
				1,
				0,
				map[string]interface{}{"id": courseID, "email": buyerEmail},
			)

			if err != nil || len(purchaseRecords) == 0 {
				return false, time.Time{}
			}

			purchaseDate := purchaseRecords[0].GetDateTime("created")
			if purchaseDate.IsZero() {
				return false, time.Time{}
			}

			unlockTime := purchaseDate.Time().AddDate(0, 0, dripDays)
			if unlockTime.After(time.Now()) {
				return true, unlockTime
			}
			return false, time.Time{}
		}

		enrollmentDate := progressRecords[0].GetDateTime("started_at")
		if enrollmentDate.IsZero() {
			enrollmentDate = progressRecords[0].GetDateTime("created")
		}
		if enrollmentDate.IsZero() {
			return false, time.Time{}
		}

		unlockTime := enrollmentDate.Time().AddDate(0, 0, dripDays)
		if unlockTime.After(time.Now()) {
			return true, unlockTime
		}
	}

	return false, time.Time{}
}

