package hooks

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"facet/services"

	"github.com/pocketbase/pocketbase/core"
)

// courseCheckoutPeekLimit caps how much of a checkout body the gate reads.
// A real checkout request is a few hundred bytes.
const courseCheckoutPeekLimit = 64 << 10

// featureForPublicRoute names the feature switch that controls a public
// route, or "" when the route isn't switchable.
//
// Turning a feature off closes the doors new visitors would use: the course
// catalogue, new enrolments and new course purchases, and public discussion
// threads. It never opens anything. Routes that existing learners and buyers
// already depend on (course pages, progress, quizzes, certificates, enrolment
// verification links, purchase access links, the Stripe webhook) stay as they
// were, and admin routes are untouched so the owner can still see and export
// their data or turn the feature back on.
func featureForPublicRoute(method, path string) string {
	switch {
	case strings.HasPrefix(path, "/api/comments/"):
		return services.FeatureDiscussions
	case method == http.MethodGet && (path == "/api/courses" || path == "/api/courses/"):
		return services.FeatureCourses
	case method == http.MethodPost && strings.HasPrefix(path, "/api/courses/") && strings.HasSuffix(path, "/enroll"):
		return services.FeatureCourses
	}
	return ""
}

// isCourseCheckout reports whether a request starts a Stripe checkout for a
// course. It reads the body and puts it back, so the checkout handler still
// sees the whole request.
func isCourseCheckout(r *http.Request) bool {
	if r.Method != http.MethodPost || r.URL.Path != "/api/content/checkout" || r.Body == nil {
		return false
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, courseCheckoutPeekLimit+1))
	if rr, ok := r.Body.(interface{ Reread() }); ok {
		// PocketBase's rereadable body rewinds itself and stays rereadable
		// for whatever reads it next.
		rr.Reread()
	} else {
		rest := r.Body
		r.Body = struct {
			io.Reader
			io.Closer
		}{io.MultiReader(bytes.NewReader(raw), rest), rest}
	}
	if err != nil || len(raw) > courseCheckoutPeekLimit {
		return false
	}

	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mediaType == "application/x-www-form-urlencoded" {
		values, err := url.ParseQuery(string(raw))
		return err == nil && values.Get("content_type") == "courses"
	}
	var body struct {
		ContentType string `json:"content_type"`
	}
	return json.Unmarshal(raw, &body) == nil && body.ContentType == "courses"
}

// bindFeatureGate answers 404 for public routes whose feature the owner has
// switched off, so a hidden component can't be reached directly through the
// API either.
func bindFeatureGate(se *core.ServeEvent, app core.App) {
	se.Router.BindFunc(func(e *core.RequestEvent) error {
		feature := featureForPublicRoute(e.Request.Method, e.Request.URL.Path)
		if feature == "" && e.Request.URL.Path == "/api/content/checkout" &&
			!services.IsFeatureEnabled(app, services.FeatureCourses, true) && isCourseCheckout(e.Request) {
			feature = services.FeatureCourses
		}
		if feature == "" || services.IsFeatureEnabled(app, feature, true) {
			return e.Next()
		}
		return e.JSON(http.StatusNotFound, map[string]string{"error": feature + " is turned off"})
	})
}
