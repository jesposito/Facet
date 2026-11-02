package hooks

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"facet/services"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

type courseFixture struct {
	*securityFixture
	handler   http.Handler
	course    *core.Record
	lesson    *core.Record
	questions []*core.Record
}

func newCourseFixture(t *testing.T) *courseFixture {
	t.Helper()
	f := newSecurityFixture(t)
	plan := &services.PlanConfig{Features: services.PlanFeatures{Courses: true}}
	// PocketBase serves a protected file to anyone the collection's view
	// rule admits, so attachment protection relies on the startup sweep.
	RegisterCollectionRules(f.pb)
	RegisterCourseHooks(f.pb, plan, f.crypto, f.rl)
	RegisterQuizHooks(f.pb, plan, f.crypto, f.rl)
	RegisterDownloadHooks(f.pb, f.crypto, f.rl)

	worksheet, err := filesystem.NewFileFromBytes([]byte("LESSON WORKSHEET BYTES"), "worksheet.pdf")
	if err != nil {
		t.Fatal(err)
	}
	course := f.create(t, "courses", map[string]any{"title": "Course", "slug": "course", "access_tier": "paid", "price": 2500, "visibility": "public", "enable_progress": true})
	module := f.create(t, "modules", map[string]any{"course": course.Id, "title": "Module", "sort_order": 1})
	lesson := f.create(t, "lessons", map[string]any{"module": module.Id, "title": "Lesson", "content_type": "text", "content": "PAID LESSON BODY", "sort_order": 1, "attachments": []*filesystem.File{worksheet}})
	quiz := f.create(t, "quizzes", map[string]any{"lesson": lesson.Id, "title": "Check", "passing_score": 100})
	options := []map[string]any{{"text": "Wrong", "is_correct": false}, {"text": "Right", "is_correct": true}}
	q1 := f.create(t, "quiz_questions", map[string]any{"quiz": quiz.Id, "question_text": "One?", "question_type": "multiple_choice", "options": options, "points": 10, "sort_order": 1})
	q2 := f.create(t, "quiz_questions", map[string]any{"quiz": quiz.Id, "question_text": "Two?", "question_type": "multiple_choice", "options": options, "points": 10, "sort_order": 2})

	return &courseFixture{securityFixture: f, handler: f.serve(t), course: course, lesson: lesson, questions: []*core.Record{q1, q2}}
}

func TestEveryBuyerOfACourseCanUseIt(t *testing.T) {
	c := newCourseFixture(t)
	first := c.completedPurchase(t, "first@example.com", "courses", c.course.Id)
	second := c.completedPurchase(t, "second@example.com", "courses", c.course.Id)
	refunded := c.completedPurchase(t, "refunded@example.com", "courses", c.course.Id)
	refunded.Set("status", "refunded")
	if err := c.app.Save(refunded); err != nil {
		t.Fatal(err)
	}

	for _, buyer := range []struct {
		email    string
		purchase *core.Record
	}{{"first@example.com", first}, {"second@example.com", second}} {
		t.Run(buyer.email, func(t *testing.T) {
			cookie := map[string]string{"Cookie": c.buyerCookie(t, buyer.email, buyer.purchase)}

			rec := do(t, c.handler, testRequest{url: "/api/courses/course", headers: cookie})
			assertStatus(t, rec, http.StatusOK)
			assertContains(t, rec, `"is_purchased":true`)
			assertContains(t, rec, "PAID LESSON BODY")

			assertStatus(t, do(t, c.handler, testRequest{url: "/api/courses/course/progress", headers: cookie}), http.StatusOK)
			assertStatus(t, do(t, c.handler, testRequest{method: http.MethodPost, url: "/api/courses/course/lessons/" + c.lesson.Id + "/complete", headers: cookie}), http.StatusOK)
			assertStatus(t, do(t, c.handler, testRequest{url: "/api/courses/course/lessons/" + c.lesson.Id + "/quiz", headers: cookie}), http.StatusOK)
		})
	}

	for name, cookie := range map[string]string{
		"refunded buyer":           c.buyerCookie(t, "refunded@example.com", refunded),
		"another buyer's purchase": c.buyerCookie(t, "intruder@example.com", first),
		"free enrolment token":     enrolmentCookie(t, c.securityFixture, "first@example.com"),
		"no token":                 "",
	} {
		t.Run(name+" is refused", func(t *testing.T) {
			headers := map[string]string{"Cookie": cookie}
			rec := do(t, c.handler, testRequest{url: "/api/courses/course", headers: headers})
			assertStatus(t, rec, http.StatusOK)
			assertContains(t, rec, `"is_purchased":false`)
			assertNotContains(t, rec, "PAID LESSON BODY")

			if rec := do(t, c.handler, testRequest{url: "/api/courses/course/progress", headers: headers}); rec.Code == http.StatusOK {
				t.Fatalf("progress allowed: %s", rec.Body.String())
			}
			if rec := do(t, c.handler, testRequest{method: http.MethodPost, url: "/api/courses/course/lessons/" + c.lesson.Id + "/quiz/submit", headers: headers, body: `{"answers":[]}`}); rec.Code == http.StatusOK {
				t.Fatalf("quiz submit allowed: %s", rec.Body.String())
			}
		})
	}
}

func enrolmentCookie(t *testing.T, f *securityFixture, email string) string {
	t.Helper()
	token, _, err := f.purchase.GenerateEnrollmentJWT("some-free-course", email)
	if err != nil {
		t.Fatal(err)
	}
	return services.PurchaseCookieName + "=" + token
}

func TestLessonAttachmentsUseSignedDownloads(t *testing.T) {
	c := newCourseFixture(t)
	buyer := c.completedPurchase(t, "first@example.com", "courses", c.course.Id)
	filename := c.lesson.GetStringSlice("attachments")[0]

	direct := "/api/files/" + c.lesson.Collection().Id + "/" + c.lesson.Id + "/" + filename
	if rec := do(t, c.handler, testRequest{url: direct}); rec.Code == http.StatusOK {
		t.Fatalf("protected attachment served without a token")
	}

	rec := do(t, c.handler, testRequest{url: "/api/courses/course", headers: map[string]string{"Cookie": c.buyerCookie(t, "first@example.com", buyer)}})
	assertStatus(t, rec, http.StatusOK)
	var course struct {
		Modules []struct {
			Lessons []struct {
				Attachments []struct {
					Filename string `json:"filename"`
					URL      string `json:"url"`
				} `json:"attachments"`
			} `json:"lessons"`
		} `json:"modules"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &course); err != nil {
		t.Fatal(err)
	}
	attachments := course.Modules[0].Lessons[0].Attachments
	if len(attachments) != 1 || !strings.HasPrefix(attachments[0].URL, "/api/downloads/") {
		t.Fatalf("unexpected attachments: %#v", attachments)
	}

	rec = do(t, c.handler, testRequest{url: attachments[0].URL})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, "LESSON WORKSHEET BYTES")
	c.waitForDownloadLogs(t, 1)
}

func TestQuizScoresEachQuestionOnce(t *testing.T) {
	c := newCourseFixture(t)
	buyer := c.completedPurchase(t, "first@example.com", "courses", c.course.Id)
	cookie := map[string]string{"Cookie": c.buyerCookie(t, "first@example.com", buyer)}
	url := "/api/courses/course/lessons/" + c.lesson.Id + "/quiz/submit"
	q1, q2 := c.questions[0].Id, c.questions[1].Id

	repeated := `{"answers":[{"question_id":"` + q1 + `","selected_answer":1},{"question_id":"` + q1 + `","selected_answer":1}]}`
	assertStatus(t, do(t, c.handler, testRequest{method: http.MethodPost, url: url, headers: cookie, body: repeated}), http.StatusBadRequest)
	if n, _ := c.app.CountRecords("quiz_attempts"); n != 0 {
		t.Fatalf("rejected submission was recorded as an attempt")
	}

	half := `{"answers":[{"question_id":"` + q1 + `","selected_answer":1}]}`
	rec := do(t, c.handler, testRequest{method: http.MethodPost, url: url, headers: cookie, body: half})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"score":50`)
	assertContains(t, rec, `"passed":false`)

	full := `{"answers":[{"question_id":"` + q1 + `","selected_answer":1},{"question_id":"` + q2 + `","selected_answer":1}]}`
	rec = do(t, c.handler, testRequest{method: http.MethodPost, url: url, headers: cookie, body: full})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"score":100`)
	assertContains(t, rec, `"passed":true`)
}

func TestLearnerQuizHidesCorrectAnswers(t *testing.T) {
	c := newCourseFixture(t)
	buyer := c.completedPurchase(t, "first@example.com", "courses", c.course.Id)
	rec := do(t, c.handler, testRequest{url: "/api/courses/course/lessons/" + c.lesson.Id + "/quiz", headers: map[string]string{"Cookie": c.buyerCookie(t, "first@example.com", buyer)}})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, "Right")
	assertNotContains(t, rec, "is_correct")
}
