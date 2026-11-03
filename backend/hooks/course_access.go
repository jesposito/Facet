package hooks

import (
	"net/http"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase/core"
)

// Every learner route that reads or changes progress asks the same questions
// in the same order. May this visitor see the course at all? Who are they?
// Have they bought it, or enrolled when enrolment is required? Is this lesson
// open to them yet, or held back by drip scheduling or sequential access?
//
// The quiz routes used to answer only the third question, so a buyer could
// read and pass a locked lesson's quiz, and anyone with any learner cookie
// could use a draft or private course's quizzes. Answering all four here keeps
// the lesson, progress and quiz routes from drifting apart again.

// courseVisibleTo reports whether the request may see a course. Drafts and
// private courses stay hidden from everyone except the site owner, who can
// preview them.
func courseVisibleTo(e *core.RequestEvent, course *core.Record) bool {
	if isSuperuser(e) {
		return true
	}
	if course.GetBool("is_draft") {
		return false
	}
	switch course.GetString("visibility") {
	case "public", "unlisted":
		return true
	}
	return false
}

// learnerEntitled reports whether a verified learner may use a course: a
// completed purchase for paid courses, an existing enrolment for free courses
// that require one. Free courses without the requirement are open to any
// verified learner.
func learnerEntitled(app core.App, purchaseService *services.PurchaseService, purchaseIDs []string, email string, course *core.Record) (bool, string) {
	if email == "" {
		return false, "authentication required"
	}
	if course.GetString("access_tier") == "paid" {
		if findCompletedPurchase(app, purchaseService, purchaseIDs, email, "courses", course.Id) == nil {
			return false, "purchase required"
		}
		return true, ""
	}
	if course.GetBool("require_enrollment") && !isEnrolled(app, course.Id, email) {
		return false, "enrollment required"
	}
	return true, ""
}

func isEnrolled(app core.App, courseID, email string) bool {
	records, err := app.FindRecordsByFilter(
		"course_progress",
		"course = {:course} && buyer_email = {:email}",
		"", 1, 0,
		map[string]interface{}{"course": courseID, "email": email},
	)
	return err == nil && len(records) > 0
}

// requireCourseLearner finds a course by slug and checks that the request
// comes from a learner who may use it. When it refuses, it has already written
// the response and returns errResponseWritten.
func requireCourseLearner(app core.App, purchaseService *services.PurchaseService, e *core.RequestEvent, slug string) (*core.Record, string, error) {
	course, err := app.FindFirstRecordByFilter("courses", "slug = {:slug}", map[string]interface{}{"slug": slug})
	if err != nil || !courseVisibleTo(e, course) {
		return nil, "", respondNotFound(e, "course not found")
	}

	token := extractPurchaseCookie(e)
	if token == "" {
		return nil, "", respondUnauthorized(e, "authentication required")
	}
	purchaseIDs, email, err := purchaseService.ValidatePurchaseJWT(token)
	if err != nil {
		return nil, "", respondUnauthorized(e, "invalid token")
	}

	if ok, reason := learnerEntitled(app, purchaseService, purchaseIDs, email, course); !ok {
		return nil, "", respondForbidden(e, reason)
	}
	return course, email, nil
}

// requireCourseLesson finds a lesson in the course and checks that it's open
// to the learner. Preview lessons are open to everyone who can see the
// course, as on the course page; any other lesson waits for its module's drip
// date and, with sequential access, for the lessons before it. When it
// refuses, it has already written the response and returns errResponseWritten.
func requireCourseLesson(app core.App, e *core.RequestEvent, course *core.Record, lessonID, email string) (*core.Record, error) {
	lesson, err := app.FindRecordById("lessons", lessonID)
	if err != nil {
		return nil, respondNotFound(e, "lesson not found")
	}
	module, err := app.FindRecordById("modules", lesson.GetString("module"))
	if err != nil || module.GetString("course") != course.Id {
		return nil, respondNotFound(e, "lesson not found in this course")
	}

	if lesson.GetBool("is_preview") {
		return lesson, nil
	}

	if locked, unlockDate := checkModuleDripLock(app, course.Id, module, email); locked {
		return nil, respondJSON(e, http.StatusForbidden, map[string]interface{}{
			"error":            "module is not yet available",
			"drip_locked":      true,
			"drip_unlock_date": unlockDate.Format(time.RFC3339),
		})
	}

	if course.GetBool("sequential_access") && !canAccessLesson(app, course.Id, lesson, email) {
		return nil, respondForbidden(e, "complete previous lessons first")
	}
	return lesson, nil
}
