package hooks

import (
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// certifiedCourse turns on certificates for the fixture's one-lesson course
// and records that lesson as finished by email, so the next recalculation
// completes the course.
func certifiedCourse(t *testing.T, c *courseFixture, email string) {
	t.Helper()
	c.course.Set("enable_certificates", true)
	if err := c.app.Save(c.course); err != nil {
		t.Fatal(err)
	}
	c.create(t, "lesson_progress", map[string]any{"lesson": c.lesson.Id, "buyer_email": email, "completed": true})
}

func certificateCount(t *testing.T, c *courseFixture, email string) int64 {
	t.Helper()
	n, err := c.app.CountRecords("course_certificates", dbx.HashExp{"course": c.course.Id, "buyer_email": email})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func courseProgress(t *testing.T, c *courseFixture, email string) *core.Record {
	t.Helper()
	p, err := c.app.FindFirstRecordByFilter("course_progress", "course = {:course} && buyer_email = {:email}",
		map[string]any{"course": c.course.Id, "email": email})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCompletingACourseIssuesTheCertificateBeforeResponding(t *testing.T) {
	c := newCourseFixture(t)
	c.course.Set("enable_certificates", true)
	if err := c.app.Save(c.course); err != nil {
		t.Fatal(err)
	}
	purchase := c.completedPurchase(t, "learner@example.com", "courses", c.course.Id)
	cookie := map[string]string{"Cookie": c.buyerCookie(t, "learner@example.com", purchase)}

	assertStatus(t, do(t, c.handler, testRequest{method: http.MethodPost, url: "/api/courses/course/lessons/" + c.lesson.Id + "/complete", headers: cookie}), http.StatusOK)

	// No waiting: the certificate exists as soon as the request returns.
	if n := certificateCount(t, c, "learner@example.com"); n != 1 {
		t.Fatalf("%d certificates after completion, want 1", n)
	}
	if courseProgress(t, c, "learner@example.com").GetDateTime("completed_at").IsZero() {
		t.Fatal("completed_at not recorded")
	}
	rec := do(t, c.handler, testRequest{url: "/api/courses/course/certificate", headers: cookie})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"course_title":"Course"`)
}

func TestFailedProgressSaveIssuesNoCertificate(t *testing.T) {
	c := newCourseFixture(t)
	certifiedCourse(t, c, "learner@example.com")
	c.app.OnRecordCreate("course_progress").BindFunc(func(e *core.RecordEvent) error {
		return errors.New("disk full")
	})

	if err := recalculateCourseProgress(c.app, c.course.Id, "learner@example.com"); err == nil {
		t.Fatal("recalculation reported success although progress was not saved")
	}
	if n := certificateCount(t, c, "learner@example.com"); n != 0 {
		t.Fatalf("%d certificates issued without saved progress", n)
	}
}

func TestCertificateFailureIsRetriedOnTheNextCompletion(t *testing.T) {
	c := newCourseFixture(t)
	certifiedCourse(t, c, "learner@example.com")
	var fail atomic.Bool
	fail.Store(true)
	c.app.OnRecordCreate("course_certificates").BindFunc(func(e *core.RecordEvent) error {
		if fail.Load() {
			return errors.New("temporary failure")
		}
		return e.Next()
	})

	if err := recalculateCourseProgress(c.app, c.course.Id, "learner@example.com"); err == nil {
		t.Fatal("recalculation reported success although the certificate failed")
	}
	if n := certificateCount(t, c, "learner@example.com"); n != 0 {
		t.Fatalf("%d certificates after a failed issue", n)
	}
	// The failed attempt must not leave the course marked complete, or the
	// retry below would have nothing to trigger it.
	if n, _ := c.app.CountRecords("course_progress"); n != 0 {
		t.Fatalf("%d progress rows kept after the certificate failed", n)
	}

	fail.Store(false)
	if err := recalculateCourseProgress(c.app, c.course.Id, "learner@example.com"); err != nil {
		t.Fatal(err)
	}
	if n := certificateCount(t, c, "learner@example.com"); n != 1 {
		t.Fatalf("%d certificates after retry, want 1", n)
	}
}

func TestCompletedCourseWithoutCertificateIsHealed(t *testing.T) {
	// Before issuance became transactional a crash could leave completed_at
	// set with no certificate. A later completion still issues it.
	c := newCourseFixture(t)
	certifiedCourse(t, c, "learner@example.com")
	c.create(t, "course_progress", map[string]any{"course": c.course.Id, "buyer_email": "learner@example.com", "progress_percent": 100, "completed_at": "2026-01-01 00:00:00.000Z"})

	if err := recalculateCourseProgress(c.app, c.course.Id, "learner@example.com"); err != nil {
		t.Fatal(err)
	}
	if n := certificateCount(t, c, "learner@example.com"); n != 1 {
		t.Fatalf("%d certificates, want 1", n)
	}
}

func TestConcurrentCompletionsIssueOneCertificate(t *testing.T) {
	c := newCourseFixture(t)
	certifiedCourse(t, c, "learner@example.com")

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- recalculateCourseProgress(c.app, c.course.Id, "learner@example.com")
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	if n := certificateCount(t, c, "learner@example.com"); n != 1 {
		t.Fatalf("%d certificates from concurrent completions, want 1", n)
	}
	if n, _ := c.app.CountRecords("course_progress"); n != 1 {
		t.Fatalf("%d progress rows from concurrent completions, want 1", n)
	}
}
