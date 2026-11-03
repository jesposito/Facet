package hooks

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"facet/services"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const burst = 40

// concurrently fires the same request burst times at once and returns the
// responses. Each request comes from its own address, so Facet's per-IP
// limiter stays out of the way.
func concurrently(t *testing.T, h http.Handler, req testRequest) []*httptest.ResponseRecorder {
	t.Helper()
	results := make([]*httptest.ResponseRecorder, burst)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < burst; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i] = do(t, h, req)
		}(i)
	}
	close(start)
	wg.Wait()
	codes := map[int]int{}
	for _, r := range results {
		codes[r.Code]++
	}
	t.Logf("%s %s -> status counts %v", req.method, req.url, codes)
	return results
}

func countStatus(results []*httptest.ResponseRecorder, status int) int {
	n := 0
	for _, r := range results {
		if r.Code == status {
			n++
		}
	}
	return n
}

func TestSingleUseShareLinkWorksOnceUnderLoad(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	RegisterShareHooks(f.pb, f.share, f.crypto, f.rl)
	view := f.create(t, "views", map[string]any{"name": "Pitch", "slug": "pitch", "visibility": "unlisted", "is_active": true})
	f.create(t, "share_tokens", map[string]any{"view_id": view.Id, "token_hash": f.share.HMACToken("one-use"), "is_active": true, "max_uses": 1})
	f.create(t, "share_tokens", map[string]any{"view_id": view.Id, "token_hash": f.share.HMACToken("one-use-b"), "is_active": true, "max_uses": 1})
	h := f.serve(t)

	results := concurrently(t, h, testRequest{url: "/api/view/pitch/data", headers: map[string]string{"X-Share-Token": "one-use"}})
	if got := countStatus(results, http.StatusOK); got != 1 {
		t.Fatalf("%d of %d parallel requests opened a single-use link", got, burst)
	}

	results = concurrently(t, h, testRequest{method: http.MethodPost, url: "/api/share/validate", body: `{"token":"one-use-b"}`})
	valid := 0
	for _, r := range results {
		if r.Code == http.StatusOK && strings.Contains(r.Body.String(), `"valid":true`) {
			valid++
		}
	}
	if valid != 1 {
		t.Fatalf("%d of %d parallel validations accepted a single-use link", valid, burst)
	}

	for _, raw := range []string{"one-use", "one-use-b"} {
		token, err := f.app.FindFirstRecordByFilter("share_tokens", "token_hash = {:h}", map[string]any{"h": f.share.HMACToken(raw)})
		if err != nil {
			t.Fatal(err)
		}
		if token.GetInt("use_count") != 1 {
			t.Fatalf("%s use_count = %d, want 1", raw, token.GetInt("use_count"))
		}
	}
}

func TestSingleUseFreeCouponRedeemsOnceUnderLoad(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterPurchaseHooks(f.pb, f.crypto, f.rl)
	post := f.create(t, "posts", map[string]any{"title": "Paid", "slug": "paid", "visibility": "public", "access_tier": "paid", "price": 500, "content": "x"})
	f.create(t, "coupons", map[string]any{"code": "FREEONE", "discount_type": "percentage", "discount_value": 100, "applicable_type": "all", "max_uses": 1, "current_uses": 0, "active": true})
	h := f.serve(t)

	body := `{"content_type":"posts","content_id":"` + post.Id + `","success_url":"http://example.com/ok","cancel_url":"http://example.com/no","coupon_code":"FREEONE"}`
	results := concurrently(t, h, testRequest{method: http.MethodPost, url: "/api/content/checkout", body: body})
	if got := countStatus(results, http.StatusOK); got != 1 {
		t.Fatalf("%d of %d parallel checkouts redeemed a single-use coupon", got, burst)
	}
	if n, _ := f.app.CountRecords("purchases"); n != 1 {
		t.Fatalf("%d free purchases were created", n)
	}
	coupon, err := f.app.FindFirstRecordByFilter("coupons", "code = 'FREEONE'")
	if err != nil {
		t.Fatal(err)
	}
	if coupon.GetInt("current_uses") != 1 {
		t.Fatalf("current_uses = %d, want 1", coupon.GetInt("current_uses"))
	}
}

func TestQuizAttemptLimitHoldsUnderLoad(t *testing.T) {
	c := newCourseFixture(t)
	quiz, err := c.app.FindFirstRecordByFilter("quizzes", "lesson = {:l}", map[string]any{"l": c.lesson.Id})
	if err != nil {
		t.Fatal(err)
	}
	quiz.Set("max_attempts", 1)
	if err := c.app.Save(quiz); err != nil {
		t.Fatal(err)
	}
	buyer := c.completedPurchase(t, "learner@example.com", "courses", c.course.Id)

	body := `{"answers":[{"question_id":"` + c.questions[0].Id + `","selected_answer":0}]}`
	results := concurrently(t, c.handler, testRequest{
		method:  http.MethodPost,
		url:     "/api/courses/course/lessons/" + c.lesson.Id + "/quiz/submit",
		body:    body,
		headers: map[string]string{"Cookie": c.buyerCookie(t, "learner@example.com", buyer)},
	})
	if got := countStatus(results, http.StatusOK); got != 1 {
		t.Fatalf("%d of %d parallel submissions were accepted with max_attempts=1", got, burst)
	}
	if n, _ := c.app.CountRecords("quiz_attempts"); n != 1 {
		t.Fatalf("%d attempts recorded, want 1", n)
	}
}

func TestOnlyOneNewsletterSendStartsUnderLoad(t *testing.T) {
	f := newSecurityFixture(t)
	settings := f.app.Settings()
	settings.SMTP.Enabled = true
	settings.SMTP.Host = "smtp.example"
	settings.Meta.AppURL = "https://facet.example"
	if err := f.app.Save(settings); err != nil {
		t.Fatal(err)
	}
	RegisterNewsletterHooks(f.pb, f.crypto, f.rl, &services.PlanConfig{})
	f.create(t, "subscribers", map[string]any{"email": "reader@example.com", "status": "active"})
	_, adminToken := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	h := f.serve(t)

	// Hold the first send open until the burst is over. Otherwise a slow run
	// (such as -race) can finish it early, and a later request is then
	// rightly allowed to start a new one.
	release := make(chan struct{})
	f.app.OnMailerSend().BindFunc(func(e *core.MailerEvent) error {
		<-release
		return e.Next()
	})

	results := concurrently(t, h, testRequest{
		method:  http.MethodPost,
		url:     "/api/admin/newsletter/send",
		body:    `{"subject":"Hello","body_markdown":"Hi there"}`,
		headers: map[string]string{"Authorization": adminToken},
	})
	close(release)

	// Let the background sender finish before the database goes away.
	finished := false
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		if n, _ := f.app.CountRecords("newsletter_sends", dbx.NewExp("status = 'sending'")); n == 0 {
			finished = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if got := countStatus(results, http.StatusOK); got != 1 {
		t.Fatalf("%d of %d parallel sends started", got, burst)
	}
	if got := countStatus(results, http.StatusConflict); got != burst-1 {
		t.Fatalf("%d sends were refused as in progress, want %d", got, burst-1)
	}
	if !finished {
		t.Fatal("newsletter send never finished")
	}
}
