package hooks

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/types"
)

var (
	logEncodedASCII = regexp.MustCompile(`(?i)%[0-7]`)
	logTokenPath    = regexp.MustCompile(`(?i)(/(?:s|testimonial|unsubscribe|api/downloads|api/content/access|api/public/newsletter/(?:confirm|open|click)|api/testimonials/request|api/testimonials/verify/email)/)[^?]*`)
	logTokenQuery   = regexp.MustCompile(`(?i)([?&;](?:t|token|code|state|session_id|[^=&;]*%[^=&;]*)=)[^&]*`)
	logTextURL      = regexp.MustCompile(`(?i)(?:https?://|/)[^\s"'<>]+`)
)

// RegisterRequestLogPrivacy replaces PocketBase's activity logger because its
// default message and url include the full request URI, including bearer links.
// Requests themselves are never changed. Keep this registration alongside the
// other request middleware so it applies to native and custom routes alike.
func RegisterRequestLogPrivacy(app *pocketbase.PocketBase) {
	app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Id: "facetRequestLogPrivacy",
		Func: func(se *core.ServeEvent) error {
			se.Router.Unbind(apis.DefaultActivityLoggerMiddlewareId)
			se.Router.Bind(&hook.Handler[*core.RequestEvent]{
				Id:       apis.DefaultActivityLoggerMiddlewareId,
				Priority: apis.DefaultActivityLoggerMiddlewarePriority,
				Func:     logSafeRequest,
			})
			return se.Next()
		},
	})

	// Application messages can also contain URLs or named credentials. Sanitize
	// them before the auxiliary log database is written, including batch flushes.
	app.OnModelCreate(core.LogsTableName).Bind(&hook.Handler[*core.ModelEvent]{
		Id: "facetPersistedLogPrivacy",
		Func: func(e *core.ModelEvent) error {
			if entry, ok := e.Model.(*core.Log); ok {
				entry.Message = redactLogText(entry.Message)
				for key, value := range entry.Data {
					entry.Data[key] = redactLogValue(key, value)
				}
			}
			return e.Next()
		},
	})
}

func logSafeRequest(e *core.RequestEvent) error {
	started := time.Now()
	err := e.Next()
	settings := e.App.Settings().Logs
	if settings.MaxDays == 0 {
		return err
	}
	// SkipSuccessActivityLog is an upstream middleware. Its store key isn't
	// exported in PocketBase v0.40.6; the integration test guards compatibility.
	if err == nil && e.Get("__skipSuccessActivityLogger") != nil {
		return err
	}
	status := e.Status()
	method := boundedLogText(strings.ToUpper(e.Request.Method), 50)
	requestURL := boundedLogText(redactLogURL(e.Request.URL.RequestURI()), 3000)
	attrs := []any{
		"type", "request",
		"execTime", float64(time.Since(started)) / float64(time.Millisecond),
		"url", requestURL,
		"method", method,
		"referer", boundedLogText(redactLogURL(e.Request.Referer()), 2000),
		"userAgent", boundedLogText(redactLogText(e.Request.UserAgent()), 2000),
	}
	if meta := e.Get(apis.RequestEventKeyLogMeta); meta != nil {
		attrs = append(attrs, "meta", redactLogValue("", meta))
	}
	if err != nil {
		var apiErr *router.ApiError
		if errors.As(err, &apiErr) {
			if status == 0 {
				status = apiErr.Status
			}
			attrs = append(attrs, "error", boundedLogText(redactLogText(apiErr.Message), 2000))
		} else {
			if status == 0 {
				status = http.StatusInternalServerError
			}
			// An arbitrary downstream error can include credentials or submitted
			// values. Routes can log a safe error code when more detail is useful.
			attrs = append(attrs, "error", http.StatusText(status))
		}
	}
	attrs = append(attrs, "status", status)
	if e.Auth != nil {
		attrs = append(attrs, "auth", e.Auth.Collection().Name)
		if settings.LogAuthId {
			attrs = append(attrs, "authId", e.Auth.Id)
		}
	} else {
		attrs = append(attrs, "auth", "")
	}
	if settings.LogIP {
		attrs = append(attrs, "userIP", e.RealIP(), "remoteIP", e.RemoteIP())
	}
	level := slog.LevelInfo
	if err != nil {
		level = slog.LevelError
	}
	// Only redacted values reach the logger, including its development printer.
	e.App.Logger().Log(e.Request.Context(), level,
		method+" "+requestURL, attrs...)
	return err
}

func redactLogURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https") {
		return "[REDACTED_URL]"
	}
	path := u.EscapedPath()
	// Preserve our existing markers when a log passes through both the request
	// middleware and the persistence guard. Raw URL spelling detects encoded
	// ASCII, while automatically escaped brackets are not an encoded request.
	path = strings.ReplaceAll(strings.ReplaceAll(path, "%5BREDACTED%5D", "[REDACTED]"), "%5BREDACTED_PATH%5D", "[REDACTED_PATH]")
	rawPath := strings.SplitN(raw, "?", 2)[0]
	if strings.HasPrefix(raw, "//") || logEncodedASCII.MatchString(rawPath) || strings.Contains(path, "//") ||
		strings.Contains(path, "/./") || strings.Contains(path, "/../") {
		path = "[REDACTED_PATH]"
	} else {
		path = logTokenPath.ReplaceAllString(path, "$1[REDACTED]")
	}
	query := ""
	if u.RawQuery != "" {
		query = logTokenQuery.ReplaceAllString("?"+u.RawQuery, "$1[REDACTED]")
	}
	prefix := ""
	if u.Host != "" && !strings.HasPrefix(raw, "//") {
		prefix = "//" + u.Host
		if u.Scheme != "" {
			prefix = u.Scheme + ":" + prefix
		}
	}
	return prefix + path + query
}

func redactLogText(text string) string {
	return logTextURL.ReplaceAllStringFunc(text, redactLogURL)
}

func boundedLogText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	end := limit - 3
	for end > 0 && !utf8.RuneStart(text[end]) {
		end--
	}
	return text[:end] + "..."
}

// Checkout session IDs can confirm a purchase. A one-way reference lets the
// owner correlate errors without storing that bearer in application logs.
func checkoutSessionLogRef(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(sessionID))
	return hex.EncodeToString(digest[:8])
}

func redactLogValue(key string, value any) any {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", "-"), " ", ""))
	switch normalized {
	case "token", "t", "session-id", "stripe-checkout-session-id", "access-token",
		"authorization", "proxy-authorization", "cookie", "set-cookie", "password",
		"x-share-token", "x-share-visit", "x-password-token":
		return "[REDACTED]"
	}
	switch v := value.(type) {
	case string:
		return boundedLogText(redactLogText(v), 3000)
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, item := range v {
			out[k] = redactLogValue(k, item)
		}
		return out
	case types.JSONMap[any]:
		return redactLogValue(key, map[string]any(v))
	case map[string]string:
		out := make(map[string]any, len(v))
		for k, item := range v {
			out[k] = redactLogValue(k, item)
		}
		return out
	case http.Header:
		return redactLogValue(key, map[string][]string(v))
	case map[string][]string:
		out := make(map[string]any, len(v))
		for k, item := range v {
			out[k] = redactLogValue(k, item)
		}
		return out
	case []string:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = redactLogValue("", item)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = redactLogValue("", item)
		}
		return out
	default:
		return value
	}
}
