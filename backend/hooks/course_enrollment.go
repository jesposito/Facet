package hooks

import (
	"fmt"
	"html"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

// Free-course enrolment used to set a learner cookie for whatever email the
// visitor typed. Progress, certificates and the course list are all keyed by
// email, so typing someone else's address handed over their records. Now the
// cookie is only set after the visitor follows a link sent to that address,
// or when they already hold a cookie proving the same address.

// enrollmentEmailCooldown limits how often one address is sent a link, so
// the form can't be used to flood someone's inbox.
const enrollmentEmailCooldown = 2 * time.Minute

type enrollmentMailLimiter struct {
	mu   sync.Mutex
	sent map[string]time.Time
}

func (l *enrollmentMailLimiter) allow(email string) bool {
	key := strings.ToLower(email)
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.sent == nil {
		l.sent = make(map[string]time.Time)
	}
	if last, ok := l.sent[key]; ok && now.Sub(last) < enrollmentEmailCooldown {
		return false
	}
	for k, t := range l.sent {
		if now.Sub(t) >= enrollmentEmailCooldown {
			delete(l.sent, k)
		}
	}
	l.sent[key] = now
	return true
}

// courseOpenForEnrollment reports whether visitors may enrol in a course:
// free, not a draft, and public or reachable by direct link.
func courseOpenForEnrollment(course *core.Record) (bool, string) {
	if course.GetBool("is_draft") {
		return false, ""
	}
	switch course.GetString("visibility") {
	case "public", "unlisted":
	default:
		return false, ""
	}
	if course.GetString("access_tier") == "paid" {
		return false, "this course requires purchase"
	}
	return true, ""
}

// verifiedLearnerEmail returns the email proven by the caller's learner
// cookie, if any.
func verifiedLearnerEmail(e *core.RequestEvent, purchaseService *services.PurchaseService) string {
	token := extractPurchaseCookie(e)
	if token == "" {
		return ""
	}
	_, email, err := purchaseService.ValidatePurchaseJWT(token)
	if err != nil {
		return ""
	}
	return email
}

// enrollLearner records the enrolment if it doesn't exist yet.
func enrollLearner(app core.App, course *core.Record, email, name string) (alreadyEnrolled bool, err error) {
	existing, err := app.FindRecordsByFilter(
		"course_progress",
		"course = {:course} && buyer_email = {:email}",
		"",
		1,
		0,
		map[string]interface{}{"course": course.Id, "email": email},
	)
	if err == nil && len(existing) > 0 {
		return true, nil
	}

	progressCollection, err := app.FindCollectionByNameOrId("course_progress")
	if err != nil {
		return false, err
	}

	progress := core.NewRecord(progressCollection)
	progress.Set("course", course.Id)
	progress.Set("buyer_email", email)
	progress.Set("started_at", time.Now())
	progress.Set("last_accessed", time.Now())
	progress.Set("progress_percent", 0)
	progress.Set("lessons_completed", 0)

	totalLessons := 0
	moduleRecords, mErr := app.FindRecordsByFilter("modules", "course = {:course}", "", 500, 0, map[string]interface{}{"course": course.Id})
	if mErr == nil {
		for _, mod := range moduleRecords {
			lessonRecords, lErr := app.FindRecordsByFilter("lessons", "module = {:module}", "", 500, 0, map[string]interface{}{"module": mod.Id})
			if lErr == nil {
				totalLessons += len(lessonRecords)
			}
		}
	}
	progress.Set("total_lessons", totalLessons)
	if name != "" {
		progress.Set("student_name", name)
	}

	if err := app.Save(progress); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "duplicate key") {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func setLearnerCookie(e *core.RequestEvent, purchaseService *services.PurchaseService, courseID, email string) error {
	jwtToken, expiresAt, err := purchaseService.GenerateEnrollmentJWT(courseID, email)
	if err != nil {
		return err
	}
	http.SetCookie(e.Response, &http.Cookie{
		Name:     services.PurchaseCookieName,
		Value:    jwtToken,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   isRequestSecure(e),
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// publicAppURL is the base for links in emails. It comes from configuration
// only: the request's Host header is chosen by the sender, and a link built
// from it could deliver someone's sign-in token to another site.
func publicAppURL(app core.App) string {
	if base := strings.TrimSpace(app.Settings().Meta.AppURL); base != "" {
		return strings.TrimRight(base, "/")
	}
	return strings.TrimRight(strings.TrimSpace(os.Getenv("APP_URL")), "/")
}

func sendEnrollmentEmail(app core.App, course *core.Record, email, link string) error {
	title := course.GetString("title")
	safeTitle := html.EscapeString(title)
	safeLink := html.EscapeString(link)
	message := &mailer.Message{
		From: mail.Address{
			Name:    app.Settings().Meta.SenderName,
			Address: app.Settings().Meta.SenderAddress,
		},
		To:      []mail.Address{{Address: email}},
		Subject: fmt.Sprintf("Confirm your enrolment in %s", title),
		HTML: fmt.Sprintf(`<p>Someone asked to enrol this address in <strong>%s</strong>.</p>
<p><a href="%s">Start the course</a></p>
<p>The link works for 24 hours. If this wasn't you, ignore this email and nothing will happen.</p>`, safeTitle, safeLink),
		Text: fmt.Sprintf("Someone asked to enrol this address in %s.\n\nStart the course: %s\n\nThe link works for 24 hours. If this wasn't you, ignore this email and nothing will happen.\n", title, link),
	}
	return app.NewMailClient().Send(message)
}

func registerEnrollmentRoutes(se *core.ServeEvent, app core.App, purchaseService *services.PurchaseService, rl *services.RateLimitService) {
	mailLimiter := &enrollmentMailLimiter{}

	// POST /api/courses/{slug}/enroll
	//
	// A visitor whose learner cookie already proves this email is enrolled at
	// once. Anyone else is sent a link, and enrolment happens when they
	// follow it. The reply is the same whether or not an email went out, so
	// the form doesn't reveal who has enrolled.
	se.Router.POST("/api/courses/{slug}/enroll", RateLimitMiddleware(rl, "strict")(func(e *core.RequestEvent) error {
		slug := e.Request.PathValue("slug")

		var req struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := e.BindBody(&req); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		req.Email = strings.TrimSpace(req.Email)
		if req.Email == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "email is required"})
		}
		if !emailRegex.MatchString(req.Email) {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid email format"})
		}
		req.Name = strings.TrimSpace(req.Name)
		if len(req.Name) > 200 {
			req.Name = req.Name[:200]
		}

		course, err := app.FindFirstRecordByFilter("courses", "slug = {:slug}", map[string]interface{}{"slug": slug})
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
		}
		if open, reason := courseOpenForEnrollment(course); !open {
			if reason != "" {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": reason})
			}
			return e.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
		}

		if proven := verifiedLearnerEmail(e, purchaseService); proven != "" && strings.EqualFold(proven, req.Email) {
			alreadyEnrolled, err := enrollLearner(app, course, proven, req.Name)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create enrollment"})
			}
			if err := setLearnerCookie(e, purchaseService, course.Id, proven); err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
			}
			status := "enrolled"
			if alreadyEnrolled {
				status = "already_enrolled"
			}
			return e.JSON(http.StatusOK, map[string]interface{}{
				"status":       status,
				"redirect_url": fmt.Sprintf("/courses/%s/learn", slug),
			})
		}

		base := publicAppURL(app)
		if !app.Settings().SMTP.Enabled || base == "" {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{
				"error": "enrolment needs email confirmation, and this site can't send email yet",
			})
		}

		if mailLimiter.allow(req.Email) {
			token, err := purchaseService.GenerateEnrollmentVerifyJWT(course.Id, req.Email, req.Name)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
			}
			link := fmt.Sprintf("%s/api/courses/%s/enroll/verify?token=%s", base, url.PathEscape(slug), url.QueryEscape(token))
			if err := sendEnrollmentEmail(app, course, req.Email, link); err != nil {
				app.Logger().Warn("Failed to send enrolment email", "error", err, "course", course.Id)
				return e.JSON(http.StatusBadGateway, map[string]string{"error": "failed to send confirmation email"})
			}
		}

		return e.JSON(http.StatusAccepted, map[string]string{"status": "verification_sent"})
	}))

	// GET /api/courses/{slug}/enroll/verify?token=...
	// The link from the enrolment email. It enrols the learner, sets the
	// learner cookie and opens the course.
	se.Router.GET("/api/courses/{slug}/enroll/verify", RateLimitMiddleware(rl, "moderate")(func(e *core.RequestEvent) error {
		slug := e.Request.PathValue("slug")
		claims, err := purchaseService.ValidateEnrollmentVerifyJWT(e.Request.URL.Query().Get("token"))
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "this link is invalid or has expired"})
		}

		course, err := app.FindFirstRecordByFilter("courses", "slug = {:slug}", map[string]interface{}{"slug": slug})
		if err != nil || course.Id != claims.CourseID {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "this link is invalid or has expired"})
		}
		if open, _ := courseOpenForEnrollment(course); !open {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
		}

		if _, err := enrollLearner(app, course, claims.Email, claims.Name); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create enrollment"})
		}
		if err := setLearnerCookie(e, purchaseService, course.Id, claims.Email); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		}
		return e.Redirect(http.StatusFound, fmt.Sprintf("/courses/%s/learn", url.PathEscape(slug)))
	}))
}
