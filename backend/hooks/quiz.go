package hooks

import (
	"encoding/json"
	"errors"
	"facet/services"
	"math/rand"
	"net/http"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

var errQuizAttemptsExhausted = errors.New("maximum attempts exceeded")

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
			lessonID := e.Request.PathValue("lessonId")

			// Same checks as the lesson itself: visibility, purchase or
			// enrolment, drip schedule and sequential access
			course, buyerEmail, err := requireCourseLearner(app, purchaseService, e, e.Request.PathValue("slug"))
			if err != nil {
				return err
			}
			if _, err := requireCourseLesson(app, e, course, lessonID, buyerEmail); err != nil {
				return err
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
				0,
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
				"title":               quiz.GetString("title"),
				"passing_score":       quiz.GetFloat("passing_score"),
				"randomize_questions": quiz.GetBool("randomize_questions"),
				"max_attempts":        quiz.GetInt("max_attempts"),
				"questions":           questions,
				"attempts_used":       attemptCount,
			}
			if previousAttempt != nil {
				response["previous_attempt"] = previousAttempt
			}

			return e.JSON(http.StatusOK, response)
		}))

		// POST /api/courses/{slug}/lessons/{lessonId}/quiz/submit - Submit quiz answers
		se.Router.POST("/api/courses/{slug}/lessons/{lessonId}/quiz/submit", RateLimitMiddleware(rl, "strict")(func(e *core.RequestEvent) error {
			lessonID := e.Request.PathValue("lessonId")

			// Checked before anything is recorded, so a refused submission
			// leaves no attempt and no progress behind
			course, buyerEmail, err := requireCourseLearner(app, purchaseService, e, e.Request.PathValue("slug"))
			if err != nil {
				return err
			}
			if _, err := requireCourseLesson(app, e, course, lessonID, buyerEmail); err != nil {
				return err
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
				0,
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

			// Each question scores once. Repeating a known correct answer
			// would otherwise push the score past 100%.
			answeredQuestions := make(map[string]bool, len(req.Answers))
			for _, answer := range req.Answers {
				if answeredQuestions[answer.QuestionID] {
					return e.JSON(http.StatusBadRequest, map[string]string{"error": "each question can only be answered once"})
				}
				answeredQuestions[answer.QuestionID] = true
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

			// The check above is a quick answer for the common case.
			// Concurrent submissions all pass it, so the limit is enforced
			// again here, with the count and the save in one transaction.
			err = app.RunInTransaction(func(txApp core.App) error {
				if maxAttempts > 0 {
					used, err := txApp.CountRecords("quiz_attempts",
						dbx.NewExp("quiz = {:quiz} AND buyer_email = {:email}", dbx.Params{"quiz": quiz.Id, "email": buyerEmail}))
					if err != nil {
						return err
					}
					if int(used) >= maxAttempts {
						return errQuizAttemptsExhausted
					}
				}
				return txApp.Save(attempt)
			})
			if errors.Is(err, errQuizAttemptsExhausted) {
				return respondForbidden(e, "maximum attempts exceeded")
			}
			if err != nil {
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

			// A failed lookup answers 500 here, because a false "no quiz" would
			// open a blank editor whose next save could replace the real quiz.
			if _, err := findQuizLesson(app, courseID, lessonID); err != nil {
				return respondQuizError(e, err, "failed to load quiz")
			}
			quiz, err := findLessonQuiz(app, lessonID)
			if err != nil {
				return respondQuizError(e, err, "failed to load quiz")
			}

			// Fetch questions with full answers
			questionRecords, err := app.FindRecordsByFilter(
				"quiz_questions",
				"quiz = {:quiz}",
				"sort_order",
				0,
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
				"lesson":              quiz.GetString("lesson"),
				"title":               quiz.GetString("title"),
				"passing_score":       quiz.GetFloat("passing_score"),
				"randomize_questions": quiz.GetBool("randomize_questions"),
				"max_attempts":        quiz.GetInt("max_attempts"),
				"questions":           questions,
			})
		})

		// POST /api/admin/courses/{courseId}/lessons/{lessonId}/quiz - Create quiz
		se.Router.POST("/api/admin/courses/{courseId}/lessons/{lessonId}/quiz", func(e *core.RequestEvent) error {
			if err := requireSuperuser(e); err != nil {
				return err
			}

			var req adminQuizRequest
			bindErr := e.BindBody(&req)

			var quiz *core.Record
			err := app.RunInTransaction(func(txApp core.App) error {
				var err error
				quiz, err = createAdminQuiz(txApp, e.Request.PathValue("courseId"), e.Request.PathValue("lessonId"), req, bindErr)
				return err
			})
			if err != nil {
				return respondQuizError(e, err, "failed to create quiz")
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

			var req adminQuizRequest
			bindErr := e.BindBody(&req)

			var quiz *core.Record
			err := app.RunInTransaction(func(txApp core.App) error {
				var err error
				quiz, err = updateAdminQuiz(txApp, e.Request.PathValue("courseId"), e.Request.PathValue("lessonId"), req, bindErr)
				return err
			})
			if err != nil {
				return respondQuizError(e, err, "failed to update quiz")
			}

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

			err := app.RunInTransaction(func(txApp core.App) error {
				return deleteAdminQuiz(txApp, e.Request.PathValue("courseId"), e.Request.PathValue("lessonId"))
			})
			if err != nil {
				return respondQuizError(e, err, "failed to delete quiz")
			}

			return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
		})

		return se.Next()
	})
}

// parseQuizOptions reads a question's options. A record loaded from the
// database holds them as types.JSONRaw; a record built in memory may hold a
// parsed slice or a JSON string.
func parseQuizOptions(rawOptions interface{}) []map[string]interface{} {
	var options []map[string]interface{}

	switch v := rawOptions.(type) {
	case []interface{}:
		for _, opt := range v {
			if optMap, ok := opt.(map[string]interface{}); ok {
				options = append(options, optMap)
			}
		}
	case []map[string]interface{}:
		options = v
	case types.JSONRaw:
		if err := json.Unmarshal(v, &options); err != nil {
			return nil
		}
	case string:
		if err := json.Unmarshal([]byte(v), &options); err != nil {
			return nil
		}
	}
	return options
}

// stripCorrectAnswers keeps only the text of each option, so learners can't
// read the answer key. Options it can't read come back empty.
func stripCorrectAnswers(rawOptions interface{}) interface{} {
	if rawOptions == nil {
		return nil
	}

	options := parseQuizOptions(rawOptions)
	stripped := make([]map[string]interface{}, 0, len(options))
	for _, opt := range options {
		stripped = append(stripped, map[string]interface{}{
			"text": opt["text"],
		})
	}
	return stripped
}

// checkAnswer verifies if the selected answer index matches the correct option
func checkAnswer(question *core.Record, selectedAnswer int) bool {
	options := parseQuizOptions(question.Get("options"))

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
