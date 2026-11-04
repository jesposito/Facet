package hooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

var errForcedWrite = errors.New("forced write failure")

func failWrite(*core.RecordEvent) error { return errForcedWrite }

// quizAdmin is a course fixture with an owner session. Its lesson has a quiz
// with two questions, and fresh is a lesson with no quiz yet.
type quizAdmin struct {
	*courseFixture
	auth  map[string]string
	fresh *core.Record
}

func newQuizAdmin(t *testing.T) *quizAdmin {
	t.Helper()
	c := newCourseFixture(t)
	_, token := c.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	fresh := c.create(t, "lessons", map[string]any{"module": c.lesson.GetString("module"), "title": "Fresh", "content_type": "text", "content": "Body", "sort_order": 2})
	return &quizAdmin{courseFixture: c, auth: map[string]string{"Authorization": token}, fresh: fresh}
}

func (a *quizAdmin) url(lesson *core.Record) string {
	return "/api/admin/courses/" + a.course.Id + "/lessons/" + lesson.Id + "/quiz"
}

func (a *quizAdmin) send(t *testing.T, method string, lesson *core.Record, body string) *httptest.ResponseRecorder {
	t.Helper()
	return do(t, a.handler, testRequest{method: method, url: a.url(lesson), body: body, headers: a.auth})
}

func (a *quizAdmin) quiz(t *testing.T, lesson *core.Record) *core.Record {
	t.Helper()
	quiz, err := a.app.FindFirstRecordByFilter("quizzes", "lesson = {:lesson}", dbx.Params{"lesson": lesson.Id})
	if err != nil {
		t.Fatal(err)
	}
	return quiz
}

func (a *quizAdmin) questionsOf(t *testing.T, quiz *core.Record) []*core.Record {
	t.Helper()
	questions, err := a.app.FindRecordsByFilter("quiz_questions", "quiz = {:quiz}", "sort_order", 100, 0, dbx.Params{"quiz": quiz.Id})
	if err != nil {
		t.Fatal(err)
	}
	return questions
}

// linkQuiz connects the fixture's quiz to its lesson, as the create route would.
func (a *quizAdmin) linkQuiz(t *testing.T) *core.Record {
	t.Helper()
	quiz := a.quiz(t, a.lesson)
	lesson, err := a.app.FindRecordById("lessons", a.lesson.Id)
	if err != nil {
		t.Fatal(err)
	}
	lesson.Set("content_type", "quiz")
	lesson.Set("quiz", quiz.Id)
	if err := a.app.Save(lesson); err != nil {
		t.Fatal(err)
	}
	return quiz
}

// state describes everything a failed request must leave untouched: the
// lesson's link, its quiz settings and each question.
func (a *quizAdmin) state(t *testing.T, lesson *core.Record) string {
	t.Helper()
	current, err := a.app.FindRecordById("lessons", lesson.Id)
	if err != nil {
		t.Fatal(err)
	}
	var state strings.Builder
	fmt.Fprintf(&state, "lesson type=%s quiz=%s\n", current.GetString("content_type"), current.GetString("quiz"))

	quizzes, err := a.app.FindRecordsByFilter("quizzes", "lesson = {:lesson}", "", 10, 0, dbx.Params{"lesson": lesson.Id})
	if err != nil {
		t.Fatal(err)
	}
	for _, quiz := range quizzes {
		fmt.Fprintf(&state, "quiz %s title=%s pass=%v attempts=%d\n", quiz.Id, quiz.GetString("title"), quiz.GetFloat("passing_score"), quiz.GetInt("max_attempts"))
		for _, q := range a.questionsOf(t, quiz) {
			fmt.Fprintf(&state, "question %s text=%s points=%d order=%d options=%v\n", q.Id, q.GetString("question_text"), q.GetInt("points"), q.GetInt("sort_order"), q.Get("options"))
		}
	}
	return state.String()
}

func (a *quizAdmin) assertState(t *testing.T, lesson *core.Record, want string) {
	t.Helper()
	if got := a.state(t, lesson); got != want {
		t.Fatalf("stored quiz changed\n got: %s\nwant: %s", got, want)
	}
}

func quizQuestionJSON(text string) string {
	return fmt.Sprintf(`{"question_text":%q,"question_type":"multiple_choice","options":[{"text":"No","is_correct":false},{"text":"Yes","is_correct":true}]}`, text)
}

func TestQuizCreateRollsBackWhenALaterQuestionFails(t *testing.T) {
	a := newQuizAdmin(t)
	before := a.state(t, a.fresh)
	failing, writes := true, 0
	a.app.OnRecordCreate("quiz_questions").BindFunc(func(e *core.RecordEvent) error {
		writes++
		if failing && writes == 2 {
			return errForcedWrite
		}
		return e.Next()
	})
	body := `{"title":"Real quiz","questions":[` + quizQuestionJSON("One") + `,` + quizQuestionJSON("Two") + `]}`

	assertStatus(t, a.send(t, http.MethodPost, a.fresh, body), http.StatusInternalServerError)
	a.assertState(t, a.fresh, before)
	if n, _ := a.app.CountRecords("quiz_questions"); n != 2 {
		t.Fatalf("%d questions stored, want only the fixture's 2", n)
	}

	// Nothing was left behind, so the same request can simply be retried.
	failing = false
	assertStatus(t, a.send(t, http.MethodPost, a.fresh, body), http.StatusCreated)
}

func TestQuizCreateRollsBackWhenTheLessonLinkFails(t *testing.T) {
	a := newQuizAdmin(t)
	before := a.state(t, a.fresh)
	a.app.OnRecordUpdate("lessons").BindFunc(failWrite)

	body := `{"title":"Real quiz","questions":[` + quizQuestionJSON("One") + `]}`
	assertStatus(t, a.send(t, http.MethodPost, a.fresh, body), http.StatusInternalServerError)
	a.assertState(t, a.fresh, before)
	if n, _ := a.app.CountRecords("quiz_questions"); n != 2 {
		t.Fatalf("%d questions stored, want only the fixture's 2", n)
	}
}

func TestQuizUpdateRollsBackWhenAQuestionWriteFails(t *testing.T) {
	for name, force := range map[string]func(a *quizAdmin){
		"update": func(a *quizAdmin) { a.app.OnRecordUpdate("quiz_questions").BindFunc(failWrite) },
		"create": func(a *quizAdmin) { a.app.OnRecordCreate("quiz_questions").BindFunc(failWrite) },
		"delete": func(a *quizAdmin) { a.app.OnRecordDelete("quiz_questions").BindFunc(failWrite) },
	} {
		t.Run(name, func(t *testing.T) {
			a := newQuizAdmin(t)
			before := a.state(t, a.lesson)
			force(a)

			// Keeps and edits the first question, adds one and drops the second.
			body := fmt.Sprintf(`{"title":"Changed","passing_score":55,"questions":[{"id":%q,"question_text":"Edited"},%s]}`, a.questions[0].Id, quizQuestionJSON("Added"))
			assertStatus(t, a.send(t, http.MethodPatch, a.lesson, body), http.StatusInternalServerError)
			a.assertState(t, a.lesson, before)
		})
	}
}

func TestQuizUpdateRefusesQuestionIDsItCantUse(t *testing.T) {
	a := newQuizAdmin(t)
	foreignQuiz := a.create(t, "quizzes", map[string]any{"lesson": a.fresh.Id, "title": "Other"})
	foreign := a.create(t, "quiz_questions", map[string]any{"quiz": foreignQuiz.Id, "question_text": "Other?", "question_type": "multiple_choice", "sort_order": 1})
	before := a.state(t, a.lesson)
	foreignBefore := a.state(t, a.fresh)
	first := a.questions[0].Id

	for name, questions := range map[string]string{
		"unknown":   `{"id":"doesnotexist123"}`,
		"foreign":   fmt.Sprintf(`{"id":%q,"question_text":"Stolen"}`, foreign.Id),
		"duplicate": fmt.Sprintf(`{"id":%q},{"id":%q}`, first, first),
		"not text":  `{"id":42}`,
	} {
		t.Run(name, func(t *testing.T) {
			body := `{"title":"Changed","questions":[` + questions + `]}`
			assertStatus(t, a.send(t, http.MethodPatch, a.lesson, body), http.StatusBadRequest)
			a.assertState(t, a.lesson, before)
			a.assertState(t, a.fresh, foreignBefore)
		})
	}
}

func TestQuizAdminRoutesKeepTheirContract(t *testing.T) {
	a := newQuizAdmin(t)

	body := `{"title":"Real quiz","questions":[` + quizQuestionJSON("One") + `,` + quizQuestionJSON("Two") + `]}`
	created := a.send(t, http.MethodPost, a.fresh, body)
	assertStatus(t, created, http.StatusCreated)
	assertContains(t, created, `"title":"Real quiz"`)
	quiz := a.quiz(t, a.fresh)
	assertContains(t, created, quiz.Id)
	if quiz.GetFloat("passing_score") != 70 {
		t.Fatalf("passing_score = %v, want the default 70", quiz.GetFloat("passing_score"))
	}
	questions := a.questionsOf(t, quiz)
	if len(questions) != 2 || questions[0].GetInt("points") != 10 || questions[0].GetInt("sort_order") != 1 || questions[1].GetInt("sort_order") != 2 {
		t.Fatalf("questions not stored with their defaults: %v", questions)
	}
	lesson, err := a.app.FindRecordById("lessons", a.fresh.Id)
	if err != nil {
		t.Fatal(err)
	}
	if lesson.GetString("content_type") != "quiz" || lesson.GetString("quiz") != quiz.Id {
		t.Fatalf("lesson not linked: type=%s quiz=%s", lesson.GetString("content_type"), lesson.GetString("quiz"))
	}

	assertStatus(t, a.send(t, http.MethodPost, a.fresh, body), http.StatusConflict)

	read := a.send(t, http.MethodGet, a.fresh, "")
	assertStatus(t, read, http.StatusOK)
	assertContains(t, read, `"is_correct":true`)

	update := fmt.Sprintf(`{"title":"Renamed","passing_score":80,"questions":[{"id":%q,"question_text":"First"},%s]}`, questions[0].Id, quizQuestionJSON("Three"))
	updated := a.send(t, http.MethodPatch, a.fresh, update)
	assertStatus(t, updated, http.StatusOK)
	assertContains(t, updated, `"title":"Renamed"`)
	quiz = a.quiz(t, a.fresh)
	questions = a.questionsOf(t, quiz)
	if quiz.GetFloat("passing_score") != 80 || len(questions) != 2 || questions[0].GetString("question_text") != "First" || questions[1].GetString("question_text") != "Three" {
		t.Fatalf("update not stored: pass=%v questions=%v", quiz.GetFloat("passing_score"), questions)
	}

	assertStatus(t, a.send(t, http.MethodPatch, a.fresh, `{"max_attempts":3}`), http.StatusOK)
	if got := len(a.questionsOf(t, quiz)); got != 2 {
		t.Fatalf("an update without questions left %d questions, want 2", got)
	}

	deleted := a.send(t, http.MethodDelete, a.fresh, "")
	assertStatus(t, deleted, http.StatusOK)
	assertContains(t, deleted, `"status":"deleted"`)
	lesson, err = a.app.FindRecordById("lessons", a.fresh.Id)
	if err != nil {
		t.Fatal(err)
	}
	if lesson.GetString("content_type") != "text" || lesson.GetString("quiz") != "" {
		t.Fatalf("lesson still linked: type=%s quiz=%s", lesson.GetString("content_type"), lesson.GetString("quiz"))
	}
	if got := len(a.questionsOf(t, quiz)); got != 0 {
		t.Fatalf("%d questions outlived their quiz", got)
	}
	assertStatus(t, a.send(t, http.MethodGet, a.fresh, ""), http.StatusNotFound)
}

// The admin editor numbers questions from 0, and a required number field
// can't hold 0, so saves from the editor depend on the shift in quizSortOrders.
func TestQuizEditorSavesWithZeroBasedOrder(t *testing.T) {
	a := newQuizAdmin(t)
	one := `{"question_text":"One","question_type":"multiple_choice","sort_order":0}`
	two := `{"question_text":"Two","question_type":"multiple_choice","sort_order":1}`
	assertStatus(t, a.send(t, http.MethodPost, a.fresh, `{"title":"Editor quiz","questions":[`+one+`,`+two+`]}`), http.StatusCreated)

	quiz := a.quiz(t, a.fresh)
	questions := a.questionsOf(t, quiz)
	if len(questions) != 2 || questions[0].GetInt("sort_order") != 1 || questions[1].GetInt("sort_order") != 2 {
		t.Fatalf("questions stored out of order: %v", questions)
	}

	swapped := fmt.Sprintf(`{"questions":[{"id":%q,"sort_order":0},{"id":%q,"sort_order":1}]}`, questions[1].Id, questions[0].Id)
	assertStatus(t, a.send(t, http.MethodPatch, a.fresh, swapped), http.StatusOK)
	if got := a.questionsOf(t, quiz); got[0].GetString("question_text") != "Two" || got[1].GetString("question_text") != "One" {
		t.Fatalf("questions not reordered: %v", got)
	}
}

func TestQuizDeleteChangesNothingWhenAStepFails(t *testing.T) {
	for name, force := range map[string]func(t *testing.T, a *quizAdmin, quiz *core.Record){
		"quiz has attempts": func(t *testing.T, a *quizAdmin, quiz *core.Record) {
			a.create(t, "quiz_attempts", map[string]any{"quiz": quiz.Id, "buyer_email": "learner@example.com", "score": 50})
		},
		"lesson update fails": func(t *testing.T, a *quizAdmin, quiz *core.Record) {
			a.app.OnRecordUpdate("lessons").BindFunc(failWrite)
		},
		"question delete fails": func(t *testing.T, a *quizAdmin, quiz *core.Record) {
			a.app.OnRecordDelete("quiz_questions").BindFunc(failWrite)
		},
	} {
		t.Run(name, func(t *testing.T) {
			a := newQuizAdmin(t)
			quiz := a.linkQuiz(t)
			before := a.state(t, a.lesson)
			force(t, a, quiz)

			assertStatus(t, a.send(t, http.MethodDelete, a.lesson, ""), http.StatusInternalServerError)
			a.assertState(t, a.lesson, before)
		})
	}
}

func TestQuizLookupFailureIsNotAMissingQuiz(t *testing.T) {
	a := newQuizAdmin(t)
	if _, err := a.app.DB().NewQuery("DROP TABLE quizzes").Execute(); err != nil {
		t.Fatal(err)
	}

	read := a.send(t, http.MethodGet, a.lesson, "")
	assertStatus(t, read, http.StatusInternalServerError)
	assertNotContains(t, read, "no quiz found")
	assertStatus(t, a.send(t, http.MethodPost, a.fresh, `{"title":"Overwrite"}`), http.StatusInternalServerError)
}

func TestQuizAdminRoutesAnswer404ForMissingRecords(t *testing.T) {
	a := newQuizAdmin(t)
	other := a.create(t, "courses", map[string]any{"title": "Other", "slug": "other", "access_tier": "free"})

	for _, url := range []string{
		"/api/admin/courses/missing/lessons/" + a.lesson.Id + "/quiz",
		"/api/admin/courses/" + a.course.Id + "/lessons/missing/quiz",
		"/api/admin/courses/" + other.Id + "/lessons/" + a.lesson.Id + "/quiz",
	} {
		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete} {
			assertStatus(t, do(t, a.handler, testRequest{method: method, url: url, body: `{"title":"Quiz"}`, headers: a.auth}), http.StatusNotFound)
		}
	}
	for _, method := range []string{http.MethodGet, http.MethodPatch, http.MethodDelete} {
		assertStatus(t, a.send(t, method, a.fresh, `{"title":"Quiz"}`), http.StatusNotFound)
	}
}

func TestQuizAdminRoutesStayOwnerOnly(t *testing.T) {
	a := newQuizAdmin(t)
	before := a.state(t, a.lesson)
	purchase := a.completedPurchase(t, "learner@example.com", "courses", a.course.Id)
	cookie := a.buyerCookie(t, "learner@example.com", purchase)

	for _, headers := range []map[string]string{nil, {"Cookie": cookie}} {
		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete} {
			rec := do(t, a.handler, testRequest{method: method, url: a.url(a.lesson), body: `{"title":"Nope"}`, headers: headers})
			assertStatus(t, rec, http.StatusUnauthorized)
		}
	}
	a.assertState(t, a.lesson, before)
}

func TestOnlyOneParallelQuizCreateSucceeds(t *testing.T) {
	a := newQuizAdmin(t)
	results := concurrently(t, a.handler, testRequest{
		method:  http.MethodPost,
		url:     a.url(a.fresh),
		body:    `{"title":"Real quiz","questions":[` + quizQuestionJSON("One") + `]}`,
		headers: a.auth,
	})

	if got := countStatus(results, http.StatusCreated); got != 1 {
		t.Fatalf("%d of %d parallel creates succeeded", got, burst)
	}
	if got := countStatus(results, http.StatusConflict); got != burst-1 {
		t.Fatalf("%d of %d parallel creates answered 409, want %d", got, burst, burst-1)
	}
	if got := len(a.questionsOf(t, a.quiz(t, a.fresh))); got != 1 {
		t.Fatalf("%d questions stored, want 1", got)
	}
}

func TestQuizUpdateRepairsAnEarlierMissingLessonLink(t *testing.T) {
	a := newQuizAdmin(t)
	quiz := a.linkQuiz(t)
	a.lesson.Set("quiz", "")
	a.lesson.Set("content_type", "text")
	if err := a.app.Save(a.lesson); err != nil {
		t.Fatal(err)
	}
	assertStatus(t, a.send(t, http.MethodPatch, a.lesson, `{"title":"Recovered quiz"}`), http.StatusOK)
	lesson, err := a.app.FindRecordById("lessons", a.lesson.Id)
	if err != nil {
		t.Fatal(err)
	}
	if lesson.GetString("quiz") != quiz.Id || lesson.GetString("content_type") != "quiz" {
		t.Fatal("quiz update did not repair the lesson link")
	}
}

func TestQuizUpdateKeepsQuestionsPastTheOldLimit(t *testing.T) {
	a := newQuizAdmin(t)
	quiz := a.linkQuiz(t)
	for i := 3; i <= 501; i++ {
		a.create(t, "quiz_questions", map[string]any{"quiz": quiz.Id, "question_text": "Question " + strconv.Itoa(i), "question_type": "multiple_choice", "points": 10, "sort_order": i})
	}
	rec := a.send(t, http.MethodGet, a.lesson, "")
	assertStatus(t, rec, http.StatusOK)
	var payload struct {
		Questions []map[string]any `json:"questions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Questions) != 501 {
		t.Fatalf("got %d questions, want 501", len(payload.Questions))
	}
	body, err := json.Marshal(map[string]any{"questions": payload.Questions})
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, a.send(t, http.MethodPatch, a.lesson, string(body)), http.StatusOK)
	if n, err := a.app.CountRecords("quiz_questions"); err != nil || n != 501 {
		t.Fatalf("stored questions %d, error %v", n, err)
	}
}
