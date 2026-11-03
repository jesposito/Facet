package hooks

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"facet/services"

	"github.com/pocketbase/pocketbase/core"
)

type enrollmentFixture struct {
	*securityFixture
	handler http.Handler
	course  *core.Record
}

func newEnrollmentFixture(t *testing.T, smtp bool) *enrollmentFixture {
	t.Helper()
	f := newSecurityFixture(t)
	settings := f.app.Settings()
	settings.SMTP.Enabled = smtp
	settings.SMTP.Host = "smtp.example"
	settings.Meta.AppURL = "https://facet.example"
	settings.Meta.SenderAddress = "site@facet.example"
	if err := f.app.Save(settings); err != nil {
		t.Fatal(err)
	}

	plan := &services.PlanConfig{Features: services.PlanFeatures{Courses: true}}
	RegisterCourseHooks(f.pb, plan, f.crypto, f.rl)

	course := f.create(t, "courses", map[string]any{"title": "Free Course", "slug": "free-course", "access_tier": "free", "visibility": "public", "enable_progress": true, "enable_certificates": true})
	module := f.create(t, "modules", map[string]any{"course": course.Id, "title": "Module", "sort_order": 1})
	f.create(t, "lessons", map[string]any{"module": module.Id, "title": "Lesson", "content_type": "text", "content": "Body", "sort_order": 1})

	return &enrollmentFixture{securityFixture: f, handler: f.serve(t), course: course}
}

func enroll(t *testing.T, h http.Handler, slug, email string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	return do(t, h, testRequest{method: http.MethodPost, url: "/api/courses/" + slug + "/enroll", body: `{"email":"` + email + `","name":"Learner"}`, headers: headers})
}

func learnerCookie(rec *httptest.ResponseRecorder) string {
	for _, c := range rec.Result().Cookies() {
		if c.Name == services.PurchaseCookieName && c.Value != "" {
			return c.Name + "=" + c.Value
		}
	}
	return ""
}

var linkPattern = regexp.MustCompile(`https://facet\.example(/api/courses/[^\s"<]+)`)

// enrolmentLink returns the path of the link in the last email to addr.
func (f *enrollmentFixture) enrolmentLink(t *testing.T, addr string) string {
	t.Helper()
	msg := f.app.TestMailer.LastMessage()
	if len(msg.To) != 1 || msg.To[0].Address != addr {
		t.Fatalf("last email went to %v, want %s", msg.To, addr)
	}
	m := linkPattern.FindStringSubmatch(msg.Text)
	if m == nil {
		t.Fatalf("no enrolment link in email:\n%s", msg.Text)
	}
	return m[1]
}

func TestEnrolmentNeedsTheEmailOwner(t *testing.T) {
	e := newEnrollmentFixture(t, true)

	// The victim already has a certificate in this course.
	e.create(t, "course_progress", map[string]any{"course": e.course.Id, "buyer_email": "victim@example.com", "progress_percent": 100, "student_name": "Victim"})
	e.create(t, "course_certificates", map[string]any{"course": e.course.Id, "buyer_email": "victim@example.com", "certificate_id": "cert-victim-0001", "course_title": "Free Course"})

	// Typing the victim's address gets the attacker nothing.
	rec := enroll(t, e.handler, "free-course", "victim@example.com", nil)
	assertStatus(t, rec, http.StatusAccepted)
	assertContains(t, rec, `"status":"verification_sent"`)
	if cookie := learnerCookie(rec); cookie != "" {
		t.Fatalf("enrolment set a learner cookie before verification: %s", cookie)
	}
	assertStatus(t, do(t, e.handler, testRequest{url: "/api/my-courses"}), http.StatusUnauthorized)
	assertStatus(t, do(t, e.handler, testRequest{url: "/api/courses/free-course/certificate"}), http.StatusUnauthorized)

	// The link went to the victim, and following it is what signs them in.
	path := e.enrolmentLink(t, "victim@example.com")
	rec = do(t, e.handler, testRequest{url: path})
	assertStatus(t, rec, http.StatusFound)
	if loc := rec.Header().Get("Location"); loc != "/courses/free-course/learn" {
		t.Fatalf("redirect = %q", loc)
	}
	cookie := learnerCookie(rec)
	if cookie == "" {
		t.Fatal("verified link didn't set the learner cookie")
	}
	rec = do(t, e.handler, testRequest{url: "/api/my-courses", headers: map[string]string{"Cookie": cookie}})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"has_certificate":true`)
}

func TestNewLearnerEnrolsThroughTheEmailLink(t *testing.T) {
	e := newEnrollmentFixture(t, true)

	assertStatus(t, enroll(t, e.handler, "free-course", "new@example.com", nil), http.StatusAccepted)
	if n, _ := e.app.CountRecords("course_progress"); n != 0 {
		t.Fatalf("enrolment recorded before verification: %d records", n)
	}

	rec := do(t, e.handler, testRequest{url: e.enrolmentLink(t, "new@example.com")})
	assertStatus(t, rec, http.StatusFound)
	cookie := map[string]string{"Cookie": learnerCookie(rec)}

	progress, err := e.app.FindFirstRecordByFilter("course_progress", "buyer_email = 'new@example.com'")
	if err != nil {
		t.Fatalf("no enrolment after verification: %v", err)
	}
	if progress.GetString("student_name") != "Learner" || progress.GetInt("total_lessons") != 1 {
		t.Fatalf("enrolment lost details: name=%q lessons=%d", progress.GetString("student_name"), progress.GetInt("total_lessons"))
	}
	assertStatus(t, do(t, e.handler, testRequest{url: "/api/courses/free-course/progress", headers: cookie}), http.StatusOK)

	// A learner who already proved this address enrols in another course at once.
	e.create(t, "courses", map[string]any{"title": "Second", "slug": "second-course", "access_tier": "free", "visibility": "public"})
	rec = enroll(t, e.handler, "second-course", "NEW@example.com", cookie)
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"status":"enrolled"`)

	// A cookie for one address doesn't vouch for another.
	rec = enroll(t, e.handler, "second-course", "someone-else@example.com", cookie)
	assertStatus(t, rec, http.StatusAccepted)
	if _, err := e.app.FindFirstRecordByFilter("course_progress", "buyer_email = 'someone-else@example.com'"); err == nil {
		t.Fatal("a learner cookie enrolled a different address")
	}
}

func TestEnrolmentLinksCannotBeForged(t *testing.T) {
	e := newEnrollmentFixture(t, true)
	other := e.create(t, "courses", map[string]any{"title": "Other", "slug": "other-course", "access_tier": "free", "visibility": "public"})

	otherCourse, err := e.purchase.GenerateEnrollmentVerifyJWT(other.Id, "x@example.com", "")
	if err != nil {
		t.Fatal(err)
	}
	purchaseJWT, _, err := e.purchase.GeneratePurchaseJWT([]string{"p1"}, "x@example.com")
	if err != nil {
		t.Fatal(err)
	}
	for name, token := range map[string]string{
		"garbage":                 "not-a-token",
		"link for another course": otherCourse,
		"purchase cookie value":   purchaseJWT,
	} {
		t.Run(name, func(t *testing.T) {
			rec := do(t, e.handler, testRequest{url: "/api/courses/free-course/enroll/verify?token=" + url.QueryEscape(token)})
			assertStatus(t, rec, http.StatusBadRequest)
			if learnerCookie(rec) != "" {
				t.Fatal("forged link set a learner cookie")
			}
		})
	}
}

func TestEnrolmentEmailUsesTheConfiguredSiteAddress(t *testing.T) {
	e := newEnrollmentFixture(t, true)
	rec := enroll(t, e.handler, "free-course", "learner@example.com", map[string]string{"Host": "evil.example", "X-Forwarded-Host": "evil.example"})
	assertStatus(t, rec, http.StatusAccepted)
	msg := e.app.TestMailer.LastMessage()
	if strings.Contains(msg.Text+msg.HTML, "evil.example") {
		t.Fatalf("link was built from the request's Host header:\n%s", msg.Text)
	}

	// Asking again straight away doesn't send a second email.
	sent := e.app.TestMailer.TotalSend()
	assertStatus(t, enroll(t, e.handler, "free-course", "learner@example.com", nil), http.StatusAccepted)
	if e.app.TestMailer.TotalSend() != sent {
		t.Fatal("repeat request sent another email inside the cooldown")
	}
}

func TestEnrolmentRefusals(t *testing.T) {
	e := newEnrollmentFixture(t, true)
	e.create(t, "courses", map[string]any{"title": "Private", "slug": "private-course", "access_tier": "free", "visibility": "private"})
	e.create(t, "courses", map[string]any{"title": "Draft", "slug": "draft-course", "access_tier": "free", "visibility": "public", "is_draft": true})
	e.create(t, "courses", map[string]any{"title": "Paid", "slug": "paid-course", "access_tier": "paid", "price": 2500, "visibility": "public"})

	assertStatus(t, enroll(t, e.handler, "private-course", "a@example.com", nil), http.StatusNotFound)
	assertStatus(t, enroll(t, e.handler, "draft-course", "a@example.com", nil), http.StatusNotFound)
	assertStatus(t, enroll(t, e.handler, "paid-course", "a@example.com", nil), http.StatusBadRequest)
	if e.app.TestMailer.TotalSend() != 0 {
		t.Fatal("refused enrolments sent email")
	}

	// Without email there's no way to confirm the address, so nothing is
	// granted.
	noMail := newEnrollmentFixture(t, false)
	rec := enroll(t, noMail.handler, "free-course", "a@example.com", nil)
	assertStatus(t, rec, http.StatusServiceUnavailable)
	if learnerCookie(rec) != "" {
		t.Fatal("cookie set without verification")
	}
}
