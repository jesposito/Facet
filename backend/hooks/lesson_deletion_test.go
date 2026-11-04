package hooks

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// overOneThousand is past the cap the module and lesson routes used to scan.
const overOneThousand = 1001

func moduleDeleteURL(courseID, moduleID string) string {
	return "/api/admin/courses/" + courseID + "/modules/" + moduleID
}

func lessonDeleteURL(courseID, moduleID, lessonID string) string {
	return moduleDeleteURL(courseID, moduleID) + "/lessons/" + lessonID
}

func lessonRoute(d *courseDeletion) string {
	return lessonDeleteURL(d.course.Id, d.module.Id, d.lesson.Id)
}

func moduleRoute(d *courseDeletion) string {
	return moduleDeleteURL(d.course.Id, d.module.Id)
}

func (d *courseDeletion) deleteAt(t *testing.T, url string) *httptest.ResponseRecorder {
	t.Helper()
	return do(t, d.handler, testRequest{method: http.MethodDelete, url: url, headers: d.auth})
}

// assertNothingDeleted checks a failed delete kept every row and the lesson's
// stored file.
func (d *courseDeletion) assertNothingDeleted(t *testing.T, before string, committed *int) {
	t.Helper()
	d.assertRows(t, before)
	if *committed != 0 || !d.attachmentStored(t) {
		t.Fatalf("file cleanup started for a failed delete: committed = %d", *committed)
	}
}

// clearAttempts makes the fixture's module deletable. Its first lesson keeps a
// quiz and has two progress rows, and its second lesson has one.
func (d *courseDeletion) clearAttempts(t *testing.T) {
	t.Helper()
	if err := deleteRecords(d.app, "quiz_attempts", dbx.HashExp{"quiz": d.quiz.Id}); err != nil {
		t.Fatal(err)
	}
	d.create(t, "lesson_progress", map[string]any{"lesson": d.lesson.Id, "buyer_email": "second@example.com", "completed": true})
	d.create(t, "lesson_progress", map[string]any{"lesson": d.fresh.Id, "buyer_email": courseLearner, "completed": true})
}

// failNthDelete lets earlier deletes in a collection through, so the failure
// lands after the transaction has already removed something.
func (d *courseDeletion) failNthDelete(collection string, nth int) {
	seen := 0
	d.app.OnRecordDelete(collection).BindFunc(func(e *core.RecordEvent) error {
		seen++
		if seen == nth {
			return errForcedWrite
		}
		return e.Next()
	})
}

// countDeletes counts deletes that succeeded inside the transaction, whether or
// not it went on to commit.
func (d *courseDeletion) countDeletes(collection string) *int {
	deleted := 0
	d.app.OnRecordDeleteExecute(collection).BindFunc(func(e *core.RecordEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		deleted++
		return nil
	})
	return &deleted
}

// saveMany stores overOneThousand records in one transaction, for speed.
func (d *courseDeletion) saveMany(t *testing.T, name string, fields func(i int) map[string]any) {
	t.Helper()
	err := d.app.RunInTransaction(func(txApp core.App) error {
		collection, err := txApp.FindCollectionByNameOrId(name)
		if err != nil {
			return err
		}
		for i := 1; i <= overOneThousand; i++ {
			record := core.NewRecord(collection)
			record.Load(fields(i))
			if err := txApp.Save(record); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func (d *courseDeletion) saveManyProgress(t *testing.T, lesson *core.Record) {
	t.Helper()
	d.saveMany(t, "lesson_progress", func(i int) map[string]any {
		return map[string]any{"lesson": lesson.Id, "buyer_email": fmt.Sprintf("learner%d@example.com", i), "completed": true}
	})
}

func TestLessonDeleteKeepsEverythingWhenItsQuizHasAttempts(t *testing.T) {
	d := newCourseDeletion(t)
	before := d.rows(t)
	committed := d.countCommittedLessonDeletes()

	assertStatus(t, d.deleteAt(t, lessonRoute(d)), http.StatusConflict)
	d.assertNothingDeleted(t, before, committed)
}

func TestModuleDeleteRollsBackWhenALaterLessonHasAttempts(t *testing.T) {
	d := newCourseDeletion(t)
	d.clearAttempts(t)
	freshQuiz := d.create(t, "quizzes", map[string]any{"lesson": d.fresh.Id, "title": "Taken"})
	d.create(t, "quiz_attempts", map[string]any{"quiz": freshQuiz.Id, "buyer_email": courseLearner, "score": 50})
	before := d.rows(t)
	committed := d.countCommittedLessonDeletes()
	deleted := d.countDeletes("lessons")

	assertStatus(t, d.deleteAt(t, moduleRoute(d)), http.StatusConflict)
	if *deleted != 1 {
		t.Fatalf("deleted %d lessons before the refusal, want 1", *deleted)
	}
	d.assertNothingDeleted(t, before, committed)
}

func TestCourseItemDeleteChangesNothingWhenADeleteFails(t *testing.T) {
	cases := []struct {
		name  string
		route func(*courseDeletion) string
		force func(*courseDeletion)
	}{
		{"lesson route second progress row", lessonRoute, func(d *courseDeletion) { d.failNthDelete("lesson_progress", 2) }},
		{"lesson route lesson row", lessonRoute, func(d *courseDeletion) { d.app.OnRecordDeleteExecute("lessons").BindFunc(failWrite) }},
		{"module route later progress row", moduleRoute, func(d *courseDeletion) { d.failNthDelete("lesson_progress", 3) }},
		{"module route second lesson row", moduleRoute, func(d *courseDeletion) { d.failNthDelete("lessons", 2) }},
		{"module route module row", moduleRoute, func(d *courseDeletion) { d.app.OnRecordDeleteExecute("modules").BindFunc(failWrite) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := newCourseDeletion(t)
			d.clearAttempts(t)
			before := d.rows(t)
			committed := d.countCommittedLessonDeletes()
			tc.force(d)

			assertStatus(t, d.deleteAt(t, tc.route(d)), http.StatusInternalServerError)
			d.assertNothingDeleted(t, before, committed)
		})
	}
}

func TestCourseItemDeleteChangesNothingWhenALookupFails(t *testing.T) {
	for name, route := range map[string]func(*courseDeletion) string{"lesson": lessonRoute, "module": moduleRoute} {
		for _, table := range []string{"modules", "lessons", "quizzes", "quiz_attempts", "lesson_progress"} {
			t.Run(name+" route "+table, func(t *testing.T) {
				d := newCourseDeletion(t)
				d.clearAttempts(t)
				before := d.rows(t)
				committed := d.countCommittedLessonDeletes()

				d.renameTable(t, table, table+"_hidden")
				rec := d.deleteAt(t, route(d))
				d.renameTable(t, table+"_hidden", table)

				assertStatus(t, rec, http.StatusInternalServerError)
				d.assertNothingDeleted(t, before, committed)
			})
		}
	}
}

func TestLessonDeleteRemovesOnlyItsLessonPastOneThousandProgressRows(t *testing.T) {
	d := newCourseDeletion(t)
	want := d.rows(t)
	target := d.create(t, "lessons", map[string]any{"module": d.module.Id, "title": "Target", "content_type": "text", "content": "Body", "sort_order": 3})
	d.saveManyProgress(t, target)

	rec := d.deleteAt(t, lessonDeleteURL(d.course.Id, d.module.Id, target.Id))
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"status":"deleted"`)
	d.assertRows(t, want)
}

func TestModuleDeleteRemovesOnlyItsLessonsPastOneThousandRows(t *testing.T) {
	d := newCourseDeletion(t)
	want := d.rows(t)
	target := d.create(t, "modules", map[string]any{"course": d.course.Id, "title": "Target", "sort_order": 2})
	lesson := d.create(t, "lessons", map[string]any{"module": target.Id, "title": "Target lesson", "content_type": "text", "content": "Body", "sort_order": 1})
	quiz := d.create(t, "quizzes", map[string]any{"lesson": lesson.Id, "title": "Untaken"})
	d.create(t, "quiz_questions", map[string]any{"quiz": quiz.Id, "question_text": "Why?", "question_type": "multiple_choice", "sort_order": 1})
	d.saveManyProgress(t, lesson)
	d.saveMany(t, "lessons", func(i int) map[string]any {
		return map[string]any{"module": target.Id, "title": "Bulk", "content_type": "text", "sort_order": i + 1}
	})

	rec := d.deleteAt(t, moduleDeleteURL(d.course.Id, target.Id))
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"status":"deleted"`)
	d.assertRows(t, want)
}

// The targets here have no attempts, so only the check under test can stop them.
func TestCourseItemDeleteKeepsItsOwnershipAndOwnerRefusals(t *testing.T) {
	d := newCourseDeletion(t)
	other := d.create(t, "courses", map[string]any{"title": "Elsewhere", "slug": "elsewhere", "access_tier": "free"})
	spare := d.create(t, "modules", map[string]any{"course": d.course.Id, "title": "Spare", "sort_order": 2})
	before := d.rows(t)

	for _, tc := range []struct {
		url     string
		status  int
		message string
	}{
		{moduleDeleteURL(d.course.Id, "missingmodule00"), http.StatusNotFound, "module not found"},
		{moduleDeleteURL(other.Id, spare.Id), http.StatusBadRequest, "module does not belong to course"},
		{lessonDeleteURL(d.course.Id, d.module.Id, "missinglesson00"), http.StatusNotFound, "lesson not found"},
		{lessonDeleteURL(d.course.Id, spare.Id, d.fresh.Id), http.StatusBadRequest, "lesson does not belong to module"},
		{lessonDeleteURL(other.Id, d.module.Id, d.fresh.Id), http.StatusBadRequest, "module does not belong to course"},
	} {
		rec := d.deleteAt(t, tc.url)
		assertStatus(t, rec, tc.status)
		assertContains(t, rec, tc.message)
	}

	cookie := d.buyerCookie(t, courseLearner, d.kept[len(d.kept)-1])
	for _, url := range []string{moduleDeleteURL(d.course.Id, spare.Id), lessonDeleteURL(d.course.Id, d.module.Id, d.fresh.Id)} {
		for _, headers := range []map[string]string{nil, {"Cookie": cookie}} {
			assertStatus(t, do(t, d.handler, testRequest{method: http.MethodDelete, url: url, headers: headers}), http.StatusUnauthorized)
		}
	}
	d.assertRows(t, before)
}
