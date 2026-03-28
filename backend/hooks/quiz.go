package hooks

import (
	"encoding/json"
	"facet/services"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterQuizHooks registers quiz-related API endpoints
func RegisterQuizHooks(app *pocketbase.PocketBase, planConfig *services.PlanConfig, crypto *services.CryptoService, rl *services.RateLimitService) {
	purchaseService := services.NewPurchaseService(crypto)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Feature gate - quizzes require courses feature
		if !planConfig.HasFeature("courses") {
			return se.Next()
		}

		// ──────────────────────────────────────────────────────────────────
		// PUBLIC ENDPOINTS
		// ──────────────────────────────────────────────────────────────────

		// GET /api/courses/{slug}/lessons/{lessonId}/quiz - Get quiz for a lesson
		se.Router.GET("/api/courses/{slug}/lessons/{lessonId}/quiz", RateLimitMiddleware(rl, "normal")(func(e *core.RequestEvent) error {
			slug := e.Request.PathValue("slug")
			lessonID := e.Request.PathValue("lessonId")

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

			// Check access (purchase or enrollment)
			buyerEmail, err := checkQuizAccess(app, purchaseService, e, course)
			if err != nil {
				return respondUnauthorized(e, "authentication required")
			}

			// Find lesson
			lesson, err := app.FindRecordById("lessons", lessonID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found"})
			}

			// Verify lesson belongs to this course
			lessonModule, err := app.FindRecordById("modules", lesson.GetString("module"))
			if err != nil || lessonModule.GetString("course") != course.Id {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found in this course"})
			}

			// Find quiz for this lesson
			quizRecords, err := app.FindRecordsByFilter(
				"quizzes",
				"lesson = {:lesson}",
				"",
				1,
				0,
				map[string]interface{}{"lesson": lessonID},
			)
			if err != nil || len(quizRecords) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "no quiz found for this lesson"})
			}
			quiz := quizRecords[0]

			// Find quiz questions ordered by sort_order
			questionRecords, err := app.FindRecordsByFilter(
				"quiz_questions",
				"quiz = {:quiz}",
				"sort_order",
				500,
				0,
				map[string]interface{}{"quiz": quiz.Id},
			)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch questions"})
			}

			// Build questions list (strip is_correct from options)
			questions := make([]map[string]interface{}, 0, len(questionRecords))
			for _, q := range questionRecords {
				questionData := map[string]interface{}{
					"id":            q.Id,
					"question_text": q.GetString("question_text"),
					"question_type": q.GetString("question_type"),
					"points":        q.GetInt("points"),
					"sort_order":    q.GetInt("sort_order"),
				}

				// Strip is_correct from options
				rawOptions := q.Get("options")
				questionData["options"] = stripCorrectAnswers(rawOptions)

				questions = append(questions, questionData)
			}

			// Shuffle if randomize_questions is true
			if quiz.GetBool("randomize_questions") {
				rng := rand.New(rand.NewSource(time.Now().UnixNano()))
				rng.Shuffle(len(questions), func(i, j int) {
					questions[i], questions[j] = questions[j], questions[i]
				})
			}

			// Get previous attempt info for this buyer
			var previousAttempt map[string]interface{}
			attemptCount := 0

			attemptRecords, err := app.FindRecordsByFilter(
				"quiz_attempts",
				"quiz = {:quiz} && buyer_email = {:email}",
				"-attempted_at",
				100,
				0,
				map[string]interface{}{"quiz": quiz.Id, "email": buyerEmail},
			)
			if err == nil {
				attemptCount = len(attemptRecords)
				if len(attemptRecords) > 0 {
					latest := attemptRecords[0]
					previousAttempt = map[string]interface{}{
						"score":        latest.GetFloat("score"),
						"passed":       latest.GetBool("passed"),
						"attempted_at": latest.GetDateTime("attempted_at"),
					}
				}
			}

			response := map[string]interface{}{
				"id":                  quiz.Id,
				"title":              quiz.GetString("title"),
				"passing_score":      quiz.GetFloat("passing_score"),
				"randomize_questions": quiz.GetBool("randomize_questions"),
				"max_attempts":       quiz.GetInt("max_attempts"),
				"questions":          questions,
				"attempts_used":      attemptCount,
			}
			if previousAttempt != nil {
				response["previous_attempt"] = previousAttempt
			}

			return e.JSON(http.StatusOK, response)
		}))

		// POST /api/courses/{slug}/lessons/{lessonId}/quiz/submit - Submit quiz answers
		se.Router.POST("/api/courses/{slug}/lessons/{lessonId}/quiz/submit", RateLimitMiddleware(rl, "strict")(func(e *core.RequestEvent) error {
			slug := e.Request.PathValue("slug")
			lessonID := e.Request.PathValue("lessonId")

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

			// Check access
			buyerEmail, err := checkQuizAccess(app, purchaseService, e, course)
			if err != nil {
				return respondUnauthorized(e, "authentication required")
			}

			// Find lesson
			lesson, err := app.FindRecordById("lessons", lessonID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found"})
			}

			// Verify lesson belongs to this course
			lessonModule, err := app.FindRecordById("modules", lesson.GetString("module"))
			if err != nil || lessonModule.GetString("course") != course.Id {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found in this course"})
			}

			// Find quiz
			quizRecords, err := app.FindRecordsByFilter(
				"quizzes",
				"lesson = {:lesson}",
				"",
				1,
				0,
				map[string]interface{}{"lesson": lessonID},
			)
			if err != nil || len(quizRecords) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "no quiz found for this lesson"})
			}
			quiz := quizRecords[0]

			// Check max_attempts
			maxAttempts := quiz.GetInt("max_attempts")
			if maxAttempts > 0 {
				existingAttempts, err := app.FindRecordsByFilter(
					"quiz_attempts",
					"quiz = {:quiz} && buyer_email = {:email}",
					"",
					1000,
					0,
					map[string]interface{}{"quiz": quiz.Id, "email": buyerEmail},
				)
				if err == nil && len(existingAttempts) >= maxAttempts {
					return respondForbidden(e, "maximum attempts exceeded")
				}
			}

			// Parse request body
			var req struct {
				Answers []struct {
					QuestionID     string `json:"question_id"`
					SelectedAnswer int    `json:"selected_answer"`
				} `json:"answers"`
			}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			// Fetch all questions for this quiz
			questionRecords, err := app.FindRecordsByFilter(
				"quiz_questions",
				"quiz = {:quiz}",
				"sort_order",
				500,
				0,
				map[string]interface{}{"quiz": quiz.Id},
			)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch questions"})
			}

			// Reject empty submissions
			if len(req.Answers) == 0 {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "no answers submitted"})
			}

			// Build question lookup and calculate total points from ALL questions
			questionMap := make(map[string]*core.Record)
			totalPoints := 0
			for _, q := range questionRecords {
				questionMap[q.Id] = q
				points := q.GetInt("points")
				if points == 0 {
					points = 10 // default
				}
				totalPoints += points
			}

			// Score the quiz
			earnedPoints := 0
			results := make([]map[string]interface{}, 0, len(req.Answers))
			answerData := make([]map[string]interface{}, 0, len(req.Answers))

			for _, answer := range req.Answers {
				question, exists := questionMap[answer.QuestionID]
				if !exists {
					continue
				}

				isCorrect := checkAnswer(question, answer.SelectedAnswer)
				if isCorrect {
					points := question.GetInt("points")
					if points == 0 {
						points = 10
					}
					earnedPoints += points
				}

				results = append(results, map[string]interface{}{
					"question_id": answer.QuestionID,
					"correct":     isCorrect,
					"explanation": question.GetString("explanation"),
				})

				answerData = append(answerData, map[string]interface{}{
					"question_id":     answer.QuestionID,
					"selected_answer": answer.SelectedAnswer,
					"is_correct":      isCorrect,
				})
			}

			// Calculate percentage score
			var score float64
			if totalPoints > 0 {
				score = (float64(earnedPoints) / float64(totalPoints)) * 100
			}

			// Determine pass/fail
			passingScore := quiz.GetFloat("passing_score")
			if passingScore == 0 {
				passingScore = 70 // default
			}
			passed := score >= passingScore

			// Create quiz_attempt record
			attemptCollection, err := app.FindCollectionByNameOrId("quiz_attempts")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "collection not found"})
			}

			attempt := core.NewRecord(attemptCollection)
			attempt.Set("quiz", quiz.Id)
			attempt.Set("buyer_email", buyerEmail)
			attempt.Set("score", score)
			attempt.Set("passed", passed)
			attempt.Set("answers", answerData)

			if err := app.Save(attempt); err != nil {
				app.Logger().Error("Failed to save quiz attempt", "error", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save attempt"})
			}

			// If passed, auto-mark lesson complete (if progress is enabled)
			if passed && course.GetBool("enable_progress") {
				markLessonCompleteForQuiz(app, course, lessonID, buyerEmail)
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"score":         score,
				"passed":        passed,
				"total_points":  totalPoints,
				"earned_points": earnedPoints,
				"results":       results,
			})
		}))

		// ──────────────────────────────────────────────────────────────────
		// ADMIN ENDPOINTS
		// ──────────────────────────────────────────────────────────────────

		// GET /api/admin/courses/{courseId}/lessons/{lessonId}/quiz - Get full quiz with answers
		se.Router.GET("/api/admin/courses/{courseId}/lessons/{lessonId}/quiz", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			courseID := e.Request.PathValue("courseId")
			lessonID := e.Request.PathValue("lessonId")

			// Verify course exists
			_, err := app.FindRecordById("courses", courseID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
			}

			// Find lesson and verify it belongs to course
			lesson, err := app.FindRecordById("lessons", lessonID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found"})
			}
			lessonModule, err := app.FindRecordById("modules", lesson.GetString("module"))
			if err != nil || lessonModule.GetString("course") != courseID {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found in this course"})
			}

			// Find quiz
			quizRecords, err := app.FindRecordsByFilter(
				"quizzes",
				"lesson = {:lesson}",
				"",
				1,
				0,
				map[string]interface{}{"lesson": lessonID},
			)
			if err != nil || len(quizRecords) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "no quiz found for this lesson"})
			}
			quiz := quizRecords[0]

			// Fetch questions with full answers
			questionRecords, err := app.FindRecordsByFilter(
				"quiz_questions",
				"quiz = {:quiz}",
				"sort_order",
				500,
				0,
				map[string]interface{}{"quiz": quiz.Id},
			)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch questions"})
			}

			questions := make([]map[string]interface{}, 0, len(questionRecords))
			for _, q := range questionRecords {
				questions = append(questions, map[string]interface{}{
					"id":            q.Id,
					"question_text": q.GetString("question_text"),
					"question_type": q.GetString("question_type"),
					"options":       q.Get("options"),
					"explanation":   q.GetString("explanation"),
					"points":        q.GetInt("points"),
					"sort_order":    q.GetInt("sort_order"),
				})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"id":                  quiz.Id,
				"lesson":             quiz.GetString("lesson"),
				"title":              quiz.GetString("title"),
				"passing_score":      quiz.GetFloat("passing_score"),
				"randomize_questions": quiz.GetBool("randomize_questions"),
				"max_attempts":       quiz.GetInt("max_attempts"),
				"questions":          questions,
			})
		})

		// POST /api/admin/courses/{courseId}/lessons/{lessonId}/quiz - Create quiz
		se.Router.POST("/api/admin/courses/{courseId}/lessons/{lessonId}/quiz", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			courseID := e.Request.PathValue("courseId")
			lessonID := e.Request.PathValue("lessonId")

			// Verify course exists
			_, err := app.FindRecordById("courses", courseID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
			}

			// Find lesson and verify it belongs to course
			lesson, err := app.FindRecordById("lessons", lessonID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found"})
			}
			lessonModule, err := app.FindRecordById("modules", lesson.GetString("module"))
			if err != nil || lessonModule.GetString("course") != courseID {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found in this course"})
			}

			// Check if quiz already exists
			existingQuiz, _ := app.FindRecordsByFilter(
				"quizzes",
				"lesson = {:lesson}",
				"",
				1,
				0,
				map[string]interface{}{"lesson": lessonID},
			)
			if len(existingQuiz) > 0 {
				return e.JSON(http.StatusConflict, map[string]string{"error": "quiz already exists for this lesson"})
			}

			// Parse request
			var req struct {
				Title              string                   `json:"title"`
				PassingScore       float64                  `json:"passing_score"`
				RandomizeQuestions bool                     `json:"randomize_questions"`
				MaxAttempts        int                      `json:"max_attempts"`
				Questions          []map[string]interface{} `json:"questions"`
			}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			if req.Title == "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "title is required"})
			}

			// Create quiz record
			quizCollection, err := app.FindCollectionByNameOrId("quizzes")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "collection not found"})
			}

			quiz := core.NewRecord(quizCollection)
			quiz.Set("lesson", lessonID)
			quiz.Set("title", req.Title)
			passingScore := req.PassingScore
			if passingScore == 0 {
				passingScore = 70
			}
			quiz.Set("passing_score", passingScore)
			quiz.Set("randomize_questions", req.RandomizeQuestions)
			quiz.Set("max_attempts", req.MaxAttempts)

			if err := app.Save(quiz); err != nil {
				app.Logger().Error("Failed to create quiz", "error", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create quiz"})
			}

			// Create questions
			questionCollection, err := app.FindCollectionByNameOrId("quiz_questions")
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "collection not found"})
			}

			for i, qData := range req.Questions {
				question := core.NewRecord(questionCollection)
				question.Set("quiz", quiz.Id)
				if text, ok := qData["question_text"].(string); ok {
					question.Set("question_text", text)
				}
				if qType, ok := qData["question_type"].(string); ok {
					question.Set("question_type", qType)
				}
				if options, ok := qData["options"]; ok {
					question.Set("options", options)
				}
				if explanation, ok := qData["explanation"].(string); ok {
					question.Set("explanation", explanation)
				}
				if points, ok := qData["points"].(float64); ok && points > 0 {
					question.Set("points", int(points))
				} else {
					question.Set("points", 10)
				}
				if sortOrder, ok := qData["sort_order"].(float64); ok {
					question.Set("sort_order", int(sortOrder))
				} else {
					question.Set("sort_order", i+1)
				}

				if err := app.Save(question); err != nil {
					app.Logger().Error("Failed to create quiz question", "error", err)
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create question"})
				}
			}

			// Update lesson content_type to "quiz" and link quiz relation
			lesson.Set("content_type", "quiz")
			lesson.Set("quiz", quiz.Id)
			if err := app.Save(lesson); err != nil {
				app.Logger().Error("Failed to update lesson", "error", err)
			}

			return e.JSON(http.StatusCreated, map[string]interface{}{
				"id":    quiz.Id,
				"title": quiz.GetString("title"),
			})
		})

		// PATCH /api/admin/courses/{courseId}/lessons/{lessonId}/quiz - Update quiz
		se.Router.PATCH("/api/admin/courses/{courseId}/lessons/{lessonId}/quiz", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			courseID := e.Request.PathValue("courseId")
			lessonID := e.Request.PathValue("lessonId")

			// Verify course exists
			_, err := app.FindRecordById("courses", courseID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
			}

			// Find lesson and verify it belongs to course
			lesson, err := app.FindRecordById("lessons", lessonID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found"})
			}
			lessonModule, err := app.FindRecordById("modules", lesson.GetString("module"))
			if err != nil || lessonModule.GetString("course") != courseID {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found in this course"})
			}

			// Find existing quiz
			quizRecords, err := app.FindRecordsByFilter(
				"quizzes",
				"lesson = {:lesson}",
				"",
				1,
				0,
				map[string]interface{}{"lesson": lessonID},
			)
			if err != nil || len(quizRecords) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "no quiz found for this lesson"})
			}
			quiz := quizRecords[0]

			// Parse request
			var req struct {
				Title              *string                  `json:"title"`
				PassingScore       *float64                 `json:"passing_score"`
				RandomizeQuestions *bool                    `json:"randomize_questions"`
				MaxAttempts        *int                     `json:"max_attempts"`
				Questions          []map[string]interface{} `json:"questions"`
			}
			if err := e.BindBody(&req); err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
			}

			// Update quiz fields
			if req.Title != nil {
				quiz.Set("title", *req.Title)
			}
			if req.PassingScore != nil {
				quiz.Set("passing_score", *req.PassingScore)
			}
			if req.RandomizeQuestions != nil {
				quiz.Set("randomize_questions", *req.RandomizeQuestions)
			}
			if req.MaxAttempts != nil {
				quiz.Set("max_attempts", *req.MaxAttempts)
			}

			if err := app.Save(quiz); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update quiz"})
			}

			// Update questions if provided
			if req.Questions != nil {
				// Fetch existing questions
				existingQuestions, _ := app.FindRecordsByFilter(
					"quiz_questions",
					"quiz = {:quiz}",
					"sort_order",
					500,
					0,
					map[string]interface{}{"quiz": quiz.Id},
				)

				// Build map of existing question IDs
				existingMap := make(map[string]*core.Record)
				for _, q := range existingQuestions {
					existingMap[q.Id] = q
				}

				// Track which IDs are in the update
				updatedIDs := make(map[string]bool)

				questionCollection, err := app.FindCollectionByNameOrId("quiz_questions")
				if err != nil {
					return e.JSON(http.StatusInternalServerError, map[string]string{"error": "collection not found"})
				}

				for i, qData := range req.Questions {
					qID, hasID := qData["id"].(string)
					if hasID && qID != "" {
						// Update existing question
						updatedIDs[qID] = true
						existing, exists := existingMap[qID]
						if !exists {
							continue
						}

						if text, ok := qData["question_text"].(string); ok {
							existing.Set("question_text", text)
						}
						if qType, ok := qData["question_type"].(string); ok {
							existing.Set("question_type", qType)
						}
						if options, ok := qData["options"]; ok {
							existing.Set("options", options)
						}
						if explanation, ok := qData["explanation"].(string); ok {
							existing.Set("explanation", explanation)
						}
						if points, ok := qData["points"].(float64); ok && points > 0 {
							existing.Set("points", int(points))
						}
						if sortOrder, ok := qData["sort_order"].(float64); ok {
							existing.Set("sort_order", int(sortOrder))
						} else {
							existing.Set("sort_order", i+1)
						}

						if err := app.Save(existing); err != nil {
							app.Logger().Error("Failed to update quiz question", "error", err)
						}
					} else {
						// Create new question
						question := core.NewRecord(questionCollection)
						question.Set("quiz", quiz.Id)
						if text, ok := qData["question_text"].(string); ok {
							question.Set("question_text", text)
						}
						if qType, ok := qData["question_type"].(string); ok {
							question.Set("question_type", qType)
						}
						if options, ok := qData["options"]; ok {
							question.Set("options", options)
						}
						if explanation, ok := qData["explanation"].(string); ok {
							question.Set("explanation", explanation)
						}
						if points, ok := qData["points"].(float64); ok && points > 0 {
							question.Set("points", int(points))
						} else {
							question.Set("points", 10)
						}
						if sortOrder, ok := qData["sort_order"].(float64); ok {
							question.Set("sort_order", int(sortOrder))
						} else {
							question.Set("sort_order", i+1)
						}

						if err := app.Save(question); err != nil {
							app.Logger().Error("Failed to create quiz question", "error", err)
						}
					}
				}

				// Delete questions not in update
				for id, record := range existingMap {
					if !updatedIDs[id] {
						if err := app.Delete(record); err != nil {
							app.Logger().Error("Failed to delete quiz question", "error", err)
						}
					}
				}
			}

			// Suppress unused variable warning for lesson
			_ = lesson

			return e.JSON(http.StatusOK, map[string]interface{}{
				"id":    quiz.Id,
				"title": quiz.GetString("title"),
			})
		})

		// DELETE /api/admin/courses/{courseId}/lessons/{lessonId}/quiz - Delete quiz
		se.Router.DELETE("/api/admin/courses/{courseId}/lessons/{lessonId}/quiz", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			courseID := e.Request.PathValue("courseId")
			lessonID := e.Request.PathValue("lessonId")

			// Verify course exists
			_, err := app.FindRecordById("courses", courseID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
			}

			// Find lesson and verify it belongs to course
			lesson, err := app.FindRecordById("lessons", lessonID)
			if err != nil {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found"})
			}
			lessonModule, err := app.FindRecordById("modules", lesson.GetString("module"))
			if err != nil || lessonModule.GetString("course") != courseID {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "lesson not found in this course"})
			}

			// Find quiz
			quizRecords, err := app.FindRecordsByFilter(
				"quizzes",
				"lesson = {:lesson}",
				"",
				1,
				0,
				map[string]interface{}{"lesson": lessonID},
			)
			if err != nil || len(quizRecords) == 0 {
				return e.JSON(http.StatusNotFound, map[string]string{"error": "no quiz found for this lesson"})
			}
			quiz := quizRecords[0]

			// Delete quiz (cascade will handle questions and attempts via DB)
			if err := app.Delete(quiz); err != nil {
				app.Logger().Error("Failed to delete quiz", "error", err)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete quiz"})
			}

			// Reset lesson content_type and clear quiz relation
			lesson.Set("content_type", "text")
			lesson.Set("quiz", "")
			if err := app.Save(lesson); err != nil {
				app.Logger().Error("Failed to update lesson after quiz delete", "error", err)
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
		})

		return se.Next()
	})
}

// checkQuizAccess verifies the request has purchase/enrollment access to a course.
// Returns the buyer email if access is granted, or an error if not.
func checkQuizAccess(app *pocketbase.PocketBase, purchaseService *services.PurchaseService, e *core.RequestEvent, course *core.Record) (string, error) {
	token := extractPurchaseCookie(e)
	if token == "" {
		return "", fmt.Errorf("no token")
	}

	purchaseIDs, buyerEmail, err := purchaseService.ValidatePurchaseJWT(token)
	if err != nil {
		return "", err
	}

	if course.GetString("access_tier") == "paid" {
		// Check purchase
		purchaseRecords, err := app.FindRecordsByFilter(
			"purchases",
			"content_type = 'courses' && content_id = {:id} && status = 'completed'",
			"",
			1,
			0,
			map[string]interface{}{"id": course.Id},
		)
		if err != nil || len(purchaseRecords) == 0 {
			return "", fmt.Errorf("purchase required")
		}
		purchase := purchaseRecords[0]
		if !purchaseService.HasPurchaseAccess(purchaseIDs, purchase.Id) {
			return "", fmt.Errorf("purchase required")
		}
	} else if course.GetBool("require_enrollment") {
		// Free course with enrollment requirement - check enrollment
		enrollRecords, err := app.FindRecordsByFilter(
			"course_progress",
			"course = {:course} && buyer_email = {:email}",
			"", 1, 0,
			map[string]interface{}{"course": course.Id, "email": buyerEmail},
		)
		if err != nil || len(enrollRecords) == 0 {
			return "", fmt.Errorf("enrollment required")
		}
	}
	// Free course without enrollment requirement - always accessible

	return buyerEmail, nil
}

// stripCorrectAnswers removes is_correct from quiz option objects
func stripCorrectAnswers(rawOptions interface{}) interface{} {
	if rawOptions == nil {
		return nil
	}

	// rawOptions might be a JSON string or already parsed
	var options []map[string]interface{}

	switch v := rawOptions.(type) {
	case []interface{}:
		for _, opt := range v {
			if optMap, ok := opt.(map[string]interface{}); ok {
				stripped := map[string]interface{}{
					"text": optMap["text"],
				}
				options = append(options, stripped)
			}
		}
		return options
	case string:
		if err := json.Unmarshal([]byte(v), &options); err != nil {
			return rawOptions
		}
		stripped := make([]map[string]interface{}, 0, len(options))
		for _, opt := range options {
			stripped = append(stripped, map[string]interface{}{
				"text": opt["text"],
			})
		}
		return stripped
	default:
		return rawOptions
	}
}

// checkAnswer verifies if the selected answer index matches the correct option
func checkAnswer(question *core.Record, selectedAnswer int) bool {
	rawOptions := question.Get("options")
	if rawOptions == nil {
		return false
	}

	var options []map[string]interface{}

	switch v := rawOptions.(type) {
	case []interface{}:
		for _, opt := range v {
			if optMap, ok := opt.(map[string]interface{}); ok {
				options = append(options, optMap)
			}
		}
	case string:
		if err := json.Unmarshal([]byte(v), &options); err != nil {
			return false
		}
	default:
		return false
	}

	if selectedAnswer < 0 || selectedAnswer >= len(options) {
		return false
	}

	isCorrect, ok := options[selectedAnswer]["is_correct"].(bool)
	return ok && isCorrect
}

// markLessonCompleteForQuiz auto-marks a lesson as complete when quiz is passed
func markLessonCompleteForQuiz(app *pocketbase.PocketBase, course *core.Record, lessonID, buyerEmail string) {
	// Check if already completed
	existingRecords, err := app.FindRecordsByFilter(
		"lesson_progress",
		"lesson = {:lesson} && buyer_email = {:email}",
		"",
		1,
		0,
		map[string]interface{}{"lesson": lessonID, "email": buyerEmail},
	)

	var lessonProgress *core.Record
	if err == nil && len(existingRecords) > 0 {
		lessonProgress = existingRecords[0]
		if lessonProgress.GetBool("completed") {
			return // Already completed
		}
	} else {
		collection, err := app.FindCollectionByNameOrId("lesson_progress")
		if err != nil {
			return
		}
		lessonProgress = core.NewRecord(collection)
		lessonProgress.Set("lesson", lessonID)
		lessonProgress.Set("buyer_email", buyerEmail)
	}

	lessonProgress.Set("completed", true)
	lessonProgress.Set("completed_at", time.Now())

	if err := app.Save(lessonProgress); err != nil {
		app.Logger().Error("Failed to auto-complete lesson for quiz pass", "error", err)
		return
	}

	// Recalculate course progress
	if err := recalculateCourseProgress(app, course.Id, buyerEmail); err != nil {
		app.Logger().Error("Failed to recalculate course progress after quiz", "error", err)
	}
}
