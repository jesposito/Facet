package hooks

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// quizRequestError is a failure the admin should see as written. Persistence
// code returns it from inside a transaction, which rolls the work back, and
// the handler writes the response once the transaction has ended.
type quizRequestError struct {
	status  int
	message string
}

func (e *quizRequestError) Error() string { return e.message }

// respondQuizError answers a failed admin quiz request. Anything other than a
// quizRequestError is a server fault, logged here and answered with fallback.
func respondQuizError(e *core.RequestEvent, err error, fallback string) error {
	var requestErr *quizRequestError
	if errors.As(err, &requestErr) {
		return e.JSON(requestErr.status, map[string]string{"error": requestErr.message})
	}
	e.App.Logger().Error(fallback, "error", err)
	return e.JSON(http.StatusInternalServerError, map[string]string{"error": fallback})
}

// quizLookupError turns a missing row into a 404 and leaves every other
// database error alone, so a failed query can't pass for a missing record.
func quizLookupError(err error, notFound string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return &quizRequestError{http.StatusNotFound, notFound}
	}
	return err
}

// findQuizLesson loads a lesson and checks that it belongs to the course.
func findQuizLesson(app core.App, courseID, lessonID string) (*core.Record, error) {
	if _, err := app.FindRecordById("courses", courseID); err != nil {
		return nil, quizLookupError(err, "course not found")
	}
	lesson, err := app.FindRecordById("lessons", lessonID)
	if err != nil {
		return nil, quizLookupError(err, "lesson not found")
	}
	module, err := app.FindRecordById("modules", lesson.GetString("module"))
	if err != nil {
		return nil, quizLookupError(err, "lesson not found in this course")
	}
	if module.GetString("course") != courseID {
		return nil, &quizRequestError{http.StatusNotFound, "lesson not found in this course"}
	}
	return lesson, nil
}

func findLessonQuiz(app core.App, lessonID string) (*core.Record, error) {
	quiz, err := app.FindFirstRecordByFilter("quizzes", "lesson = {:lesson}", dbx.Params{"lesson": lessonID})
	if err != nil {
		return nil, quizLookupError(err, "no quiz found for this lesson")
	}
	return quiz, nil
}

// adminQuizRequest is the body of the admin create and update routes. Questions
// stay loosely typed so a field of the wrong type is skipped, as it always was.
type adminQuizRequest struct {
	Title              *string                  `json:"title"`
	PassingScore       *float64                 `json:"passing_score"`
	RandomizeQuestions *bool                    `json:"randomize_questions"`
	MaxAttempts        *int                     `json:"max_attempts"`
	Questions          []map[string]interface{} `json:"questions"`
}

// applyTo copies the settings present in the request onto the quiz.
func (r adminQuizRequest) applyTo(quiz *core.Record) {
	if r.Title != nil {
		quiz.Set("title", *r.Title)
	}
	if r.PassingScore != nil {
		quiz.Set("passing_score", *r.PassingScore)
	}
	if r.RandomizeQuestions != nil {
		quiz.Set("randomize_questions", *r.RandomizeQuestions)
	}
	if r.MaxAttempts != nil {
		quiz.Set("max_attempts", *r.MaxAttempts)
	}
}

// createAdminQuiz creates a quiz, its questions and the lesson's link to it.
// Run it in a transaction: a failure at any step must leave none of them. The
// body is read before the transaction so a slow client can't hold the write
// lock, and bindErr is reported here to keep 404 and 409 ahead of 400.
func createAdminQuiz(app core.App, courseID, lessonID string, req adminQuizRequest, bindErr error) (*core.Record, error) {
	lesson, err := findQuizLesson(app, courseID, lessonID)
	if err != nil {
		return nil, err
	}

	// Two parallel creates can't both pass this check: writes run one
	// transaction at a time, so the second sees the first one's quiz.
	existing, err := app.FindRecordsByFilter("quizzes", "lesson = {:lesson}", "", 1, 0, dbx.Params{"lesson": lessonID})
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return nil, &quizRequestError{http.StatusConflict, "quiz already exists for this lesson"}
	}

	if bindErr != nil {
		return nil, &quizRequestError{http.StatusBadRequest, "invalid request"}
	}
	if req.Title == nil || *req.Title == "" {
		return nil, &quizRequestError{http.StatusBadRequest, "title is required"}
	}

	collection, err := app.FindCollectionByNameOrId("quizzes")
	if err != nil {
		return nil, err
	}
	quiz := core.NewRecord(collection)
	quiz.Set("lesson", lessonID)
	req.applyTo(quiz)
	if quiz.GetFloat("passing_score") == 0 {
		quiz.Set("passing_score", 70)
	}
	if err := app.Save(quiz); err != nil {
		return nil, err
	}

	questions, err := app.FindCollectionByNameOrId("quiz_questions")
	if err != nil {
		return nil, err
	}
	orders := quizSortOrders(req.Questions)
	for i, data := range req.Questions {
		if err := saveQuizQuestion(app, newQuizQuestion(questions, quiz), data, orders[i]); err != nil {
			return nil, err
		}
	}

	lesson.Set("content_type", "quiz")
	lesson.Set("quiz", quiz.Id)
	if err := app.Save(lesson); err != nil {
		return nil, err
	}
	return quiz, nil
}

// updateAdminQuiz applies a partial update. Run it in a transaction: the
// metadata and every question change land together or not at all.
func updateAdminQuiz(app core.App, courseID, lessonID string, req adminQuizRequest, bindErr error) (*core.Record, error) {
	lesson, err := findQuizLesson(app, courseID, lessonID)
	if err != nil {
		return nil, err
	}
	quiz, err := findLessonQuiz(app, lessonID)
	if err != nil {
		return nil, err
	}
	if bindErr != nil {
		return nil, &quizRequestError{http.StatusBadRequest, "invalid request"}
	}

	req.applyTo(quiz)
	if err := app.Save(quiz); err != nil {
		return nil, err
	}
	if req.Questions != nil {
		if err := syncQuizQuestions(app, quiz, req.Questions); err != nil {
			return nil, err
		}
	}
	// Repair the missing lesson link left by older partial quiz creates.
	if lesson.GetString("quiz") != quiz.Id || lesson.GetString("content_type") != "quiz" {
		lesson.Set("quiz", quiz.Id)
		lesson.Set("content_type", "quiz")
		if err := app.Save(lesson); err != nil {
			return nil, err
		}
	}
	return quiz, nil
}

// syncQuizQuestions makes the quiz's questions match the submitted list: an
// entry with an id updates that question, one without creates a question, and
// any question left out is deleted. An id that doesn't name a question of this
// quiz stops the update, because carrying on would delete the real question.
func syncQuizQuestions(app core.App, quiz *core.Record, submitted []map[string]interface{}) error {
	existing, err := app.FindRecordsByFilter("quiz_questions", "quiz = {:quiz}", "sort_order", 0, 0, dbx.Params{"quiz": quiz.Id})
	if err != nil {
		return err
	}
	byID := make(map[string]*core.Record, len(existing))
	for _, question := range existing {
		byID[question.Id] = question
	}
	collection, err := app.FindCollectionByNameOrId("quiz_questions")
	if err != nil {
		return err
	}

	kept := make(map[string]bool, len(submitted))
	orders := quizSortOrders(submitted)
	for i, data := range submitted {
		question := newQuizQuestion(collection, quiz)
		if raw := data["id"]; raw != nil && raw != "" {
			id, ok := raw.(string)
			if !ok || byID[id] == nil {
				return &quizRequestError{http.StatusBadRequest, "question id does not belong to this quiz"}
			}
			if kept[id] {
				return &quizRequestError{http.StatusBadRequest, "question id appears more than once"}
			}
			kept[id] = true
			question = byID[id]
		}
		if err := saveQuizQuestion(app, question, data, orders[i]); err != nil {
			return err
		}
	}

	for _, question := range existing {
		if !kept[question.Id] {
			if err := app.Delete(question); err != nil {
				return err
			}
		}
	}
	return nil
}

func newQuizQuestion(collection *core.Collection, quiz *core.Record) *core.Record {
	question := core.NewRecord(collection)
	question.Set("quiz", quiz.Id)
	return question
}

// saveQuizQuestion copies the submitted fields onto a question and saves it.
// Only a new question takes the default points.
func saveQuizQuestion(app core.App, question *core.Record, data map[string]interface{}, sortOrder int) error {
	if text, ok := data["question_text"].(string); ok {
		question.Set("question_text", text)
	}
	if qType, ok := data["question_type"].(string); ok {
		question.Set("question_type", qType)
	}
	if options, ok := data["options"]; ok {
		question.Set("options", options)
	}
	if explanation, ok := data["explanation"].(string); ok {
		question.Set("explanation", explanation)
	}
	if points, ok := data["points"].(float64); ok && points > 0 {
		question.Set("points", int(points))
	} else if question.IsNew() {
		question.Set("points", 10)
	}
	question.Set("sort_order", sortOrder)
	return app.Save(question)
}

// quizSortOrders returns the sort_order to store for each question, defaulting
// to its position. The admin editor numbers questions from 0, but sort_order is
// required and PocketBase counts 0 as blank, so a list using 0 moves up by one.
func quizSortOrders(questions []map[string]interface{}) []int {
	zeroBased := false
	for _, question := range questions {
		if order, ok := question["sort_order"].(float64); ok && int(order) == 0 {
			zeroBased = true
		}
	}

	orders := make([]int, len(questions))
	for i, question := range questions {
		orders[i] = i + 1
		if order, ok := question["sort_order"].(float64); ok {
			orders[i] = int(order)
			if zeroBased {
				orders[i]++
			}
		}
	}
	return orders
}

// deleteAdminQuiz removes a quiz and resets its lesson. Run it in a
// transaction: PocketBase refuses to delete a quiz that has attempts, and the
// lesson must stay linked when it does. Questions go with the quiz by cascade.
func deleteAdminQuiz(app core.App, courseID, lessonID string) error {
	lesson, err := findQuizLesson(app, courseID, lessonID)
	if err != nil {
		return err
	}
	quiz, err := findLessonQuiz(app, lessonID)
	if err != nil {
		return err
	}

	lesson.Set("content_type", "text")
	lesson.Set("quiz", "")
	if err := app.Save(lesson); err != nil {
		return err
	}
	return app.Delete(quiz)
}
