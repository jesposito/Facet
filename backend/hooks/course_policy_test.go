package hooks

import (
	"net/http"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// learnerRecordCounts counts what a learner request can leave behind, so a
// refused request can be shown to have changed nothing.
func learnerRecordCounts(t *testing.T, c *courseFixture) map[string]int64 {
	t.Helper()
	counts := map[string]int64{}
	for _, name := range []string{"quiz_attempts", "lesson_progress", "course_progress", "course_certificates"} {
		n, err := c.app.CountRecords(name)
		if err != nil {
			t.Fatal(err)
		}
		counts[name] = n
	}
	return counts
}

func assertNothingRecorded(t *testing.T, c *courseFixture, before map[string]int64) {
	t.Helper()
	// Certificates are issued in the background; give one a moment to show.
	time.Sleep(50 * time.Millisecond)
	after := learnerRecordCounts(t, c)
	for name, n := range before {
		if after[name] != n {
			t.Fatalf("%s went from %d to %d after a refused request", name, n, after[name])
		}
	}
}

func (c *courseFixture) lessonWithQuiz(t *testing.T, module *core.Record, title string, sortOrder int) (*core.Record, *core.Record) {
	t.Helper()
	lesson := c.create(t, "lessons", map[string]any{"module": module.Id, "title": title, "content_type": "text", "content": title + " BODY", "sort_order": sortOrder})
	quiz := c.create(t, "quizzes", map[string]any{"lesson": lesson.Id, "title": title + " quiz", "passing_score": 50})
	question := c.create(t, "quiz_questions", map[string]any{"quiz": quiz.Id, "question_text": "Pick right", "question_type": "multiple_choice", "options": []map[string]any{{"text": "Wrong", "is_correct": false}, {"text": "Right", "is_correct": true}}, "points": 10, "sort_order": 1})
	return lesson, question
}

func passingAnswer(question *core.Record) string {
	return `{"answers":[{"question_id":"` + question.Id + `","selected_answer":1}]}`
}

func TestQuizzesRespectDripAndSequentialLocks(t *testing.T) {
	c := newCourseFixture(t)
	c.course.Set("sequential_access", true)
	c.course.Set("enable_certificates", true)
	if err := c.app.Save(c.course); err != nil {
		t.Fatal(err)
	}
	firstModule, err := c.app.FindRecordById("modules", c.lesson.GetString("module"))
	if err != nil {
		t.Fatal(err)
	}
	second, secondQuestion := c.lessonWithQuiz(t, firstModule, "Second", 2)
	dripModule := c.create(t, "modules", map[string]any{"course": c.course.Id, "title": "Later", "sort_order": 2, "drip_unlock_date": time.Now().Add(72 * time.Hour).UTC().Format("2006-01-02 15:04:05.000Z")})
	dripped, drippedQuestion := c.lessonWithQuiz(t, dripModule, "Dripped", 1)

	buyer := c.completedPurchase(t, "first@example.com", "courses", c.course.Id)
	cookie := map[string]string{"Cookie": c.buyerCookie(t, "first@example.com", buyer)}
	base := "/api/courses/course/lessons/"

	for name, tc := range map[string]struct {
		lesson   *core.Record
		question *core.Record
		want     string
	}{
		"sequentially locked": {second, secondQuestion, "complete previous lessons first"},
		"drip locked":         {dripped, drippedQuestion, `"drip_locked":true`},
	} {
		t.Run(name, func(t *testing.T) {
			before := learnerRecordCounts(t, c)
			rec := do(t, c.handler, testRequest{url: base + tc.lesson.Id + "/quiz", headers: cookie})
			assertStatus(t, rec, http.StatusForbidden)
			assertContains(t, rec, tc.want)
			assertNotContains(t, rec, "Pick right")

			rec = do(t, c.handler, testRequest{method: http.MethodPost, url: base + tc.lesson.Id + "/quiz/submit", headers: cookie, body: passingAnswer(tc.question)})
			assertStatus(t, rec, http.StatusForbidden)
			assertContains(t, rec, tc.want)
			assertNothingRecorded(t, c, before)
		})
	}

	// Once the first lesson is done, the second lesson's quiz opens.
	assertStatus(t, do(t, c.handler, testRequest{method: http.MethodPost, url: base + c.lesson.Id + "/complete", headers: cookie}), http.StatusOK)
	assertStatus(t, do(t, c.handler, testRequest{url: base + second.Id + "/quiz", headers: cookie}), http.StatusOK)
	rec := do(t, c.handler, testRequest{method: http.MethodPost, url: base + second.Id + "/quiz/submit", headers: cookie, body: passingAnswer(secondQuestion)})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"passed":true`)

	// The dripped module stays shut, so the course can't be finished early
	// and no certificate is issued.
	assertStatus(t, do(t, c.handler, testRequest{method: http.MethodPost, url: base + dripped.Id + "/quiz/submit", headers: cookie, body: passingAnswer(drippedQuestion)}), http.StatusForbidden)
	assertStatus(t, do(t, c.handler, testRequest{method: http.MethodPost, url: base + dripped.Id + "/complete", headers: cookie}), http.StatusForbidden)
	time.Sleep(50 * time.Millisecond)
	if n, _ := c.app.CountRecords("course_certificates"); n != 0 {
		t.Fatalf("%d certificates issued for an unfinished course", n)
	}
}

func TestFreeCourseCompletionNeedsEnrolment(t *testing.T) {
	c := newCourseFixture(t)
	course := c.create(t, "courses", map[string]any{"title": "Enrol first", "slug": "enrol-first", "access_tier": "free", "visibility": "public", "require_enrollment": true, "enable_progress": true, "enable_certificates": true})
	module := c.create(t, "modules", map[string]any{"course": course.Id, "title": "Only", "sort_order": 1})
	lesson, question := c.lessonWithQuiz(t, module, "Only lesson", 1)

	// A real learner cookie, but for another course.
	cookie := map[string]string{"Cookie": enrolmentCookie(t, c.securityFixture, "drifter@example.com")}
	base := "/api/courses/enrol-first/lessons/" + lesson.Id

	before := learnerRecordCounts(t, c)
	rec := do(t, c.handler, testRequest{method: http.MethodPost, url: base + "/complete", headers: cookie})
	assertStatus(t, rec, http.StatusForbidden)
	assertContains(t, rec, "enrollment required")
	assertStatus(t, do(t, c.handler, testRequest{url: base + "/quiz", headers: cookie}), http.StatusForbidden)
	assertStatus(t, do(t, c.handler, testRequest{method: http.MethodPost, url: base + "/quiz/submit", headers: cookie, body: passingAnswer(question)}), http.StatusForbidden)
	assertStatus(t, do(t, c.handler, testRequest{url: "/api/courses/enrol-first/progress", headers: cookie}), http.StatusForbidden)
	assertNothingRecorded(t, c, before)

	// Progress recalculation never enrols anyone as a side effect.
	if err := recalculateCourseProgress(c.app, course.Id, "drifter@example.com"); err != nil {
		t.Fatal(err)
	}
	assertNothingRecorded(t, c, before)

	// After verified enrolment the same learner can complete the course.
	if _, err := enrollLearner(c.app, course, "drifter@example.com", ""); err != nil {
		t.Fatal(err)
	}
	assertStatus(t, do(t, c.handler, testRequest{method: http.MethodPost, url: base + "/complete", headers: cookie}), http.StatusOK)
	assertStatus(t, do(t, c.handler, testRequest{url: base + "/quiz", headers: cookie}), http.StatusOK)
	if !isLessonCompleted(c.app, lesson.Id, "drifter@example.com") {
		t.Fatal("enrolled learner's completion was not recorded")
	}
}

func TestHiddenCoursesRefuseLearnerRoutes(t *testing.T) {
	c := newCourseFixture(t)
	cookie := map[string]string{"Cookie": enrolmentCookie(t, c.securityFixture, "outsider@example.com")}
	_, adminToken := c.createAdmin(t, "owner@example.com", "owner-password-123", nil)

	for _, tc := range []struct {
		slug   string
		fields map[string]any
	}{
		{"private-course", map[string]any{"visibility": "private"}},
		{"draft-course", map[string]any{"visibility": "public", "is_draft": true}},
	} {
		t.Run(tc.slug, func(t *testing.T) {
			fields := map[string]any{"title": tc.slug, "slug": tc.slug, "access_tier": "free", "enable_progress": true}
			for k, v := range tc.fields {
				fields[k] = v
			}
			course := c.create(t, "courses", fields)
			module := c.create(t, "modules", map[string]any{"course": course.Id, "title": "M", "sort_order": 1})
			lesson, question := c.lessonWithQuiz(t, module, "Hidden", 1)
			base := "/api/courses/" + tc.slug + "/lessons/" + lesson.Id

			before := learnerRecordCounts(t, c)
			rec := do(t, c.handler, testRequest{url: base + "/quiz", headers: cookie})
			assertStatus(t, rec, http.StatusNotFound)
			assertNotContains(t, rec, "Pick right")
			assertStatus(t, do(t, c.handler, testRequest{method: http.MethodPost, url: base + "/quiz/submit", headers: cookie, body: passingAnswer(question)}), http.StatusNotFound)
			assertStatus(t, do(t, c.handler, testRequest{method: http.MethodPost, url: base + "/complete", headers: cookie}), http.StatusNotFound)
			assertStatus(t, do(t, c.handler, testRequest{url: "/api/courses/" + tc.slug + "/progress", headers: cookie}), http.StatusNotFound)
			assertNothingRecorded(t, c, before)

			// The owner can still preview it.
			rec = do(t, c.handler, testRequest{url: "/api/courses/" + tc.slug, headers: map[string]string{"Authorization": adminToken}})
			assertStatus(t, rec, http.StatusOK)
			assertContains(t, rec, "Hidden BODY")
		})
	}
}

func TestPreviewLessonQuizSkipsLocks(t *testing.T) {
	c := newCourseFixture(t)
	c.course.Set("sequential_access", true)
	if err := c.app.Save(c.course); err != nil {
		t.Fatal(err)
	}
	module, err := c.app.FindRecordById("modules", c.lesson.GetString("module"))
	if err != nil {
		t.Fatal(err)
	}
	preview, _ := c.lessonWithQuiz(t, module, "Preview", 2)
	preview.Set("is_preview", true)
	if err := c.app.Save(preview); err != nil {
		t.Fatal(err)
	}
	buyer := c.completedPurchase(t, "first@example.com", "courses", c.course.Id)
	cookie := map[string]string{"Cookie": c.buyerCookie(t, "first@example.com", buyer)}

	// The course page already shows preview lessons to everyone, so their
	// quizzes follow the same rule for buyers.
	assertStatus(t, do(t, c.handler, testRequest{url: "/api/courses/course/lessons/" + preview.Id + "/quiz", headers: cookie}), http.StatusOK)
}
