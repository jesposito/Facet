package hooks

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// deleteCourseWithDependents removes a course and everything it owns in one
// transaction, so a failure at any step leaves all of it in place.
//
// The relations pointing at a course are required and don't cascade, so
// PocketBase refuses to delete a course while anything still references it.
// The dependents therefore have to go first, inside the same transaction. A
// required relation missed here stops the delete; it can't leave an orphan.
//
// Stored files are safe until the end: PocketBase removes them in its
// after-delete hooks, which wait for the transaction to commit.
func deleteCourseWithDependents(e *core.RecordEvent) error {
	originalApp := e.App
	txErr := e.App.RunInTransaction(func(txApp core.App) error {
		e.App = txApp
		if err := deleteCourseDependents(txApp, e.Record.Id); err != nil {
			return err
		}
		return e.Next()
	})
	// The after-delete hooks reuse this event once the transaction has ended.
	e.App = originalApp
	return txErr
}

// deleteCourseDependents removes a course's curriculum and learner records.
// Run it in a transaction. Purchases stay: they're the sales record, and they
// point at the course by plain ID.
func deleteCourseDependents(app core.App, courseID string) error {
	modules, err := app.FindAllRecords("modules", dbx.HashExp{"course": courseID})
	if err != nil {
		return err
	}
	for _, module := range modules {
		lessons, err := app.FindAllRecords("lessons", dbx.HashExp{"module": module.Id})
		if err != nil {
			return err
		}
		for _, lesson := range lessons {
			if err := deleteLessonWithDependents(app, lesson); err != nil {
				return err
			}
		}
		if err := app.Delete(module); err != nil {
			return err
		}
	}

	if err := deleteRecords(app, "course_progress", dbx.HashExp{"course": courseID}); err != nil {
		return err
	}
	return deleteRecords(app, "course_certificates", dbx.HashExp{"course": courseID})
}

// deleteLessonWithDependents clears the records that would block a lesson
// delete, then removes it. Its quiz and questions go with it by cascade.
func deleteLessonWithDependents(app core.App, lesson *core.Record) error {
	quizzes, err := app.FindAllRecords("quizzes", dbx.HashExp{"lesson": lesson.Id})
	if err != nil {
		return err
	}
	for _, quiz := range quizzes {
		if err := deleteRecords(app, "quiz_attempts", dbx.HashExp{"quiz": quiz.Id}); err != nil {
			return err
		}
	}
	if err := deleteRecords(app, "lesson_progress", dbx.HashExp{"lesson": lesson.Id}); err != nil {
		return err
	}
	return app.Delete(lesson)
}

func deleteRecords(app core.App, collection string, where dbx.Expression) error {
	records, err := app.FindAllRecords(collection, where)
	if err != nil {
		return err
	}
	for _, record := range records {
		if err := app.Delete(record); err != nil {
			return err
		}
	}
	return nil
}

// courseItemRefusal is a module or lesson delete the route turns down. Returning
// it from the transaction rolls back, and the route answers after that.
type courseItemRefusal struct {
	status  int
	message string
}

func (r courseItemRefusal) Error() string { return r.message }

// deleteCourseModule removes a module and its lessons in one transaction, so a
// failure at any step leaves all of it in place.
func deleteCourseModule(app core.App, courseID, moduleID string) error {
	return app.RunInTransaction(func(txApp core.App) error {
		module, err := findCourseModule(txApp, courseID, moduleID)
		if err != nil {
			return err
		}
		lessons, err := txApp.FindRecordsByFilter("lessons", "module = {:module}", "sort_order", 0, 0, dbx.Params{"module": moduleID})
		if err != nil {
			return err
		}
		for _, lesson := range lessons {
			if err := deleteLessonKeepingAttempts(txApp, lesson); err != nil {
				return err
			}
		}
		return txApp.Delete(module)
	})
}

// deleteCourseLesson removes a lesson and its progress in one transaction.
func deleteCourseLesson(app core.App, courseID, moduleID, lessonID string) error {
	return app.RunInTransaction(func(txApp core.App) error {
		lesson, err := txApp.FindRecordById("lessons", lessonID)
		if errors.Is(err, sql.ErrNoRows) {
			return courseItemRefusal{http.StatusNotFound, "lesson not found"}
		}
		if err != nil {
			return err
		}
		if lesson.GetString("module") != moduleID {
			return courseItemRefusal{http.StatusBadRequest, "lesson does not belong to module"}
		}
		if _, err := findCourseModule(txApp, courseID, moduleID); err != nil {
			return err
		}
		return deleteLessonKeepingAttempts(txApp, lesson)
	})
}

func findCourseModule(app core.App, courseID, moduleID string) (*core.Record, error) {
	module, err := app.FindRecordById("modules", moduleID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, courseItemRefusal{http.StatusNotFound, "module not found"}
	}
	if err != nil {
		return nil, err
	}
	if module.GetString("course") != courseID {
		return nil, courseItemRefusal{http.StatusBadRequest, "module does not belong to course"}
	}
	return module, nil
}

// deleteLessonKeepingAttempts removes a lesson and its progress, and refuses if
// anyone has attempted its quiz. A course delete clears attempts because its
// confirmation says so; module and lesson deletes give no such warning, so they
// must not use deleteLessonWithDependents.
func deleteLessonKeepingAttempts(app core.App, lesson *core.Record) error {
	quizzes, err := app.FindAllRecords("quizzes", dbx.HashExp{"lesson": lesson.Id})
	if err != nil {
		return err
	}
	for _, quiz := range quizzes {
		attempts, err := app.CountRecords("quiz_attempts", dbx.HashExp{"quiz": quiz.Id})
		if err != nil {
			return err
		}
		if attempts > 0 {
			return courseItemRefusal{http.StatusConflict, "lesson has quiz attempts"}
		}
	}
	if err := deleteRecords(app, "lesson_progress", dbx.HashExp{"lesson": lesson.Id}); err != nil {
		return err
	}
	return app.Delete(lesson)
}

// respondCourseItemDelete answers a module or lesson delete once its transaction
// has ended.
func respondCourseItemDelete(e *core.RequestEvent, err error, failure string) error {
	var refusal courseItemRefusal
	switch {
	case err == nil:
		return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
	case errors.As(err, &refusal):
		return e.JSON(refusal.status, map[string]string{"error": refusal.message})
	}
	e.App.Logger().Error(failure, "error", err)
	return e.JSON(http.StatusInternalServerError, map[string]string{"error": failure})
}
