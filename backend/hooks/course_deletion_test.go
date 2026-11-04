package hooks

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

const courseLearner = "learner@example.com"

// courseCollections holds everything a course delete can reach, plus
// purchases, which it must leave alone.
var courseCollections = []string{"courses", "modules", "lessons", "quizzes", "quiz_questions", "quiz_attempts", "lesson_progress", "course_progress", "course_certificates", "purchases"}

// courseDeletion is a course with a record in every collection it owns, and a
// second course the same learner has records in. owned must go with the course
// and kept must outlive it.
type courseDeletion struct {
	*quizAdmin
	module     *core.Record
	quiz       *core.Record
	owned      []*core.Record
	kept       []*core.Record
	attachment string
}

func newCourseDeletion(t *testing.T) *courseDeletion {
	t.Helper()
	a := newQuizAdmin(t)
	quiz := a.linkQuiz(t)
	lesson, err := a.app.FindRecordById("lessons", a.lesson.Id)
	if err != nil {
		t.Fatal(err)
	}
	module, err := a.app.FindRecordById("modules", lesson.GetString("module"))
	if err != nil {
		t.Fatal(err)
	}
	attachments := lesson.GetStringSlice("attachments")
	if len(attachments) != 1 {
		t.Fatalf("fixture lesson has %d attachments, want 1", len(attachments))
	}
	d := &courseDeletion{quizAdmin: a, module: module, quiz: quiz, attachment: lesson.BaseFilesPath() + "/" + attachments[0]}
	d.owned = append(d.enrol(t, a.course, lesson, quiz), a.course, module, lesson, a.fresh, quiz, a.questions[0], a.questions[1])

	other := a.create(t, "courses", map[string]any{"title": "Other", "slug": "other", "access_tier": "free"})
	otherModule := a.create(t, "modules", map[string]any{"course": other.Id, "title": "Other module", "sort_order": 1})
	otherLesson := a.create(t, "lessons", map[string]any{"module": otherModule.Id, "title": "Other lesson", "content_type": "text", "content": "Body", "sort_order": 1})
	otherQuiz := a.create(t, "quizzes", map[string]any{"lesson": otherLesson.Id, "title": "Other quiz"})
	otherQuestion := a.create(t, "quiz_questions", map[string]any{"quiz": otherQuiz.Id, "question_text": "Other?", "question_type": "multiple_choice", "sort_order": 1})
	purchase := a.completedPurchase(t, courseLearner, "courses", a.course.Id)
	d.kept = append(d.enrol(t, other, otherLesson, otherQuiz), other, otherModule, otherLesson, otherQuiz, otherQuestion, purchase)
	return d
}

// enrol gives the learner a record in each student collection of a course.
func (d *courseDeletion) enrol(t *testing.T, course, lesson, quiz *core.Record) []*core.Record {
	t.Helper()
	return []*core.Record{
		d.create(t, "quiz_attempts", map[string]any{"quiz": quiz.Id, "buyer_email": courseLearner, "score": 50}),
		d.create(t, "lesson_progress", map[string]any{"lesson": lesson.Id, "buyer_email": courseLearner, "completed": true}),
		d.create(t, "course_progress", map[string]any{"course": course.Id, "buyer_email": courseLearner, "progress_percent": 50}),
		d.create(t, "course_certificates", map[string]any{"course": course.Id, "buyer_email": courseLearner, "certificate_id": "cert-" + course.Id, "course_title": "Course"}),
	}
}

func courseRecordURL(id string) string {
	return "/api/collections/courses/records/" + id
}

func (d *courseDeletion) deleteCourse(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	return do(t, d.handler, testRequest{method: http.MethodDelete, url: courseRecordURL(d.course.Id), headers: d.auth})
}

// describe reads records back from the database and renders every field.
func (d *courseDeletion) describe(t *testing.T, records []*core.Record) string {
	t.Helper()
	lines := make([]string, 0, len(records))
	for _, record := range records {
		stored, err := d.app.FindRecordById(record.Collection().Name, record.Id)
		if err != nil {
			t.Fatalf("read %s %s: %v", record.Collection().Name, record.Id, err)
		}
		lines = append(lines, fmt.Sprintf("%s %s %v", record.Collection().Name, record.Id, stored.FieldsData()))
	}
	slices.Sort(lines)
	return strings.Join(lines, "\n")
}

// rows describes every stored row, so a comparison also catches a row that was
// added or a survivor that was edited.
func (d *courseDeletion) rows(t *testing.T) string {
	t.Helper()
	var all []*core.Record
	for _, name := range courseCollections {
		records, err := d.app.FindAllRecords(name)
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, records...)
	}
	return d.describe(t, all)
}

func (d *courseDeletion) assertRows(t *testing.T, want string) {
	t.Helper()
	if got := d.rows(t); got != want {
		t.Fatalf("stored records changed\n got: %s\nwant: %s", got, want)
	}
}

func (d *courseDeletion) assertGone(t *testing.T, records ...*core.Record) {
	t.Helper()
	for _, record := range records {
		if _, err := d.app.FindRecordById(record.Collection().Name, record.Id); !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("%s %s outlived the delete: err = %v", record.Collection().Name, record.Id, err)
		}
	}
}

func (d *courseDeletion) attachmentStored(t *testing.T) bool {
	t.Helper()
	fsys, err := d.app.NewFilesystem()
	if err != nil {
		t.Fatal(err)
	}
	defer fsys.Close()
	exists, err := fsys.Exists(d.attachment)
	if err != nil {
		t.Fatal(err)
	}
	return exists
}

// renameTable makes every query on a table fail the way a broken lookup would,
// and renaming it back keeps its rows for comparison.
func (d *courseDeletion) renameTable(t *testing.T, from, to string) {
	t.Helper()
	if _, err := d.app.DB().NewQuery("ALTER TABLE " + from + " RENAME TO " + to).Execute(); err != nil {
		t.Fatal(err)
	}
}

// countCommittedLessonDeletes counts the event that also triggers PocketBase's
// file removal, which runs in the background and can't be awaited directly.
func (d *courseDeletion) countCommittedLessonDeletes() *int {
	committed := 0
	d.app.OnRecordAfterDeleteSuccess("lessons").BindFunc(func(e *core.RecordEvent) error {
		committed++
		return e.Next()
	})
	return &committed
}

func TestCourseDeleteRemovesTheCourseAndNothingElse(t *testing.T) {
	d := newCourseDeletion(t)
	kept := d.describe(t, d.kept)
	if !d.attachmentStored(t) {
		t.Fatal("fixture attachment is not in storage")
	}

	// A hook left holding the finished transaction couldn't query anything.
	for _, collection := range []string{"courses", "lessons"} {
		d.app.OnRecordAfterDeleteSuccess(collection).BindFunc(func(e *core.RecordEvent) error {
			if e.App.IsTransactional() {
				t.Errorf("%s after-delete hook ran inside the transaction", collection)
			}
			return e.Next()
		})
	}

	assertStatus(t, d.deleteCourse(t), http.StatusNoContent)
	d.assertGone(t, d.owned...)
	if got := d.describe(t, d.kept); got != kept {
		t.Fatalf("another course's records changed\n got: %s\nwant: %s", got, kept)
	}

	deadline := time.Now().Add(5 * time.Second)
	for d.attachmentStored(t) {
		if time.Now().After(deadline) {
			t.Fatal("lesson attachment was not removed after the commit")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestCourseDeleteChangesNothingWhenADeleteFails(t *testing.T) {
	steps := map[string]func(d *courseDeletion){
		"course row": func(d *courseDeletion) { d.app.OnRecordDeleteExecute("courses").BindFunc(failWrite) },
	}
	for _, collection := range []string{"quiz_attempts", "lesson_progress", "quiz_questions", "quizzes", "lessons", "modules", "course_progress", "course_certificates"} {
		steps[collection] = func(d *courseDeletion) { d.app.OnRecordDelete(collection).BindFunc(failWrite) }
	}

	for name, force := range steps {
		t.Run(name, func(t *testing.T) {
			d := newCourseDeletion(t)
			before := d.rows(t)
			committed := d.countCommittedLessonDeletes()
			force(d)

			assertStatus(t, d.deleteCourse(t), http.StatusBadRequest)
			d.assertRows(t, before)
			if *committed != 0 || !d.attachmentStored(t) {
				t.Fatalf("file cleanup started for a failed delete: committed = %d", *committed)
			}
		})
	}
}

// Each lookup fails at a different depth. course_progress is read after the
// whole curriculum has gone, so that case proves earlier deletes roll back.
func TestCourseDeleteChangesNothingWhenALookupFails(t *testing.T) {
	for _, table := range []string{"modules", "lessons", "quizzes", "quiz_attempts", "lesson_progress", "course_progress", "course_certificates"} {
		t.Run(table, func(t *testing.T) {
			d := newCourseDeletion(t)
			before := d.rows(t)
			committed := d.countCommittedLessonDeletes()

			d.renameTable(t, table, table+"_hidden")
			rec := d.deleteCourse(t)
			d.renameTable(t, table+"_hidden", table)

			assertStatus(t, rec, http.StatusBadRequest)
			d.assertRows(t, before)
			if *committed != 0 || !d.attachmentStored(t) {
				t.Fatalf("file cleanup started for a failed delete: committed = %d", *committed)
			}
		})
	}
}

func TestCourseDeleteCompletesPastOneThousandRelatedRows(t *testing.T) {
	const rows = 1001
	d := newCourseDeletion(t)

	err := d.app.RunInTransaction(func(txApp core.App) error {
		save := func(name string, fields map[string]any) error {
			collection, err := txApp.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			record := core.NewRecord(collection)
			record.Load(fields)
			return txApp.Save(record)
		}
		for i := 1; i <= rows; i++ {
			email := fmt.Sprintf("learner%d@example.com", i)
			if err := errors.Join(
				save("modules", map[string]any{"course": d.course.Id, "title": "Bulk", "sort_order": i + 1}),
				save("lessons", map[string]any{"module": d.module.Id, "title": "Bulk", "content_type": "text", "sort_order": i + 2}),
				save("quiz_questions", map[string]any{"quiz": d.quiz.Id, "question_text": "Bulk?", "question_type": "multiple_choice", "sort_order": i + 2}),
				save("quiz_attempts", map[string]any{"quiz": d.quiz.Id, "buyer_email": email, "score": 50}),
				save("lesson_progress", map[string]any{"lesson": d.lesson.Id, "buyer_email": email, "completed": true}),
				save("course_progress", map[string]any{"course": d.course.Id, "buyer_email": email, "progress_percent": 50}),
				save("course_certificates", map[string]any{"course": d.course.Id, "buyer_email": email, "certificate_id": fmt.Sprintf("bulk-%d", i)}),
			); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	kept := d.describe(t, d.kept)

	assertStatus(t, d.deleteCourse(t), http.StatusNoContent)
	d.assertGone(t, d.owned...)
	if got := d.describe(t, d.kept); got != kept {
		t.Fatalf("another course's records changed\n got: %s\nwant: %s", got, kept)
	}
	// Only the other course's single record should be left in each collection.
	for _, name := range courseCollections {
		if n, err := d.app.CountRecords(name); err != nil || n != 1 {
			t.Errorf("%s has %d records left, err = %v; want 1", name, n, err)
		}
	}
}

func TestCourseDeleteStaysOwnerOnly(t *testing.T) {
	d := newCourseDeletion(t)
	before := d.rows(t)
	purchase := d.kept[len(d.kept)-1]
	cookie := d.buyerCookie(t, courseLearner, purchase)

	for _, headers := range []map[string]string{nil, {"Cookie": cookie}} {
		rec := do(t, d.handler, testRequest{method: http.MethodDelete, url: courseRecordURL(d.course.Id), headers: headers})
		if rec.Code < 400 {
			t.Fatalf("delete without an owner session returned %d: %s", rec.Code, rec.Body.String())
		}
	}
	d.assertRows(t, before)
}

func TestCourseDeleteStillWorksWithNoDependents(t *testing.T) {
	d := newCourseDeletion(t)
	empty := d.create(t, "courses", map[string]any{"title": "Empty", "slug": "empty", "access_tier": "free"})

	assertStatus(t, do(t, d.handler, testRequest{method: http.MethodDelete, url: courseRecordURL(empty.Id), headers: d.auth}), http.StatusNoContent)
	d.assertGone(t, empty)
}

func TestLessonAndModuleRoutesKeepDeletingOnlyTheirOwnRecords(t *testing.T) {
	d := newCourseDeletion(t)
	spare := d.create(t, "modules", map[string]any{"course": d.course.Id, "title": "Spare", "sort_order": 2})
	spareLesson := d.create(t, "lessons", map[string]any{"module": spare.Id, "title": "Spare lesson", "content_type": "text", "content": "Body", "sort_order": 1})
	spareProgress := d.create(t, "lesson_progress", map[string]any{"lesson": spareLesson.Id, "buyer_email": courseLearner, "completed": true})
	stays := d.describe(t, []*core.Record{d.course, d.module, d.lesson})

	base := "/api/admin/courses/" + d.course.Id + "/modules/"
	assertStatus(t, do(t, d.handler, testRequest{method: http.MethodDelete, url: base + d.module.Id + "/lessons/" + d.fresh.Id, headers: d.auth}), http.StatusOK)
	assertStatus(t, do(t, d.handler, testRequest{method: http.MethodDelete, url: base + spare.Id, headers: d.auth}), http.StatusOK)

	d.assertGone(t, d.fresh, spare, spareLesson, spareProgress)
	if got := d.describe(t, []*core.Record{d.course, d.module, d.lesson}); got != stays {
		t.Fatalf("records outside the deleted lesson and module changed\n got: %s\nwant: %s", got, stays)
	}
}

// Only a course delete clears its dependents. Native deletes of the records
// below it keep PocketBase's refusal, so attempts can't vanish that way.
func TestNativeModuleAndLessonDeletesStillRefuseRecordsInUse(t *testing.T) {
	d := newCourseDeletion(t)
	before := d.rows(t)

	for collection, id := range map[string]string{"modules": d.module.Id, "lessons": d.lesson.Id} {
		url := "/api/collections/" + collection + "/records/" + id
		assertStatus(t, do(t, d.handler, testRequest{method: http.MethodDelete, url: url, headers: d.auth}), http.StatusBadRequest)
	}
	d.assertRows(t, before)
}
