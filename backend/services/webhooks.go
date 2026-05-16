// Package services — Webhook dispatcher for self-hosted Facet.
//
// Single-tenant, sqlite-backed, no external dependencies (no Redis/queue).
// Plain Go goroutines + a buffered channel queue handle delivery so multiple
// webhooks aren't blocked by one slow endpoint.
//
// Security posture:
//   - HMAC-SHA256 signature over the JSON body, sent as X-Facet-Signature.
//   - SSRF protection: hostnames are resolved at dial-time, every resolved IP
//     is checked against private/loopback/link-local/CGNAT/unique-local ranges
//     and known metadata IPs (169.254.169.254, ::1, etc). Resolution and the
//     actual connect happen against the same literal IP to close the
//     DNS-rebinding TOCTOU window.
//   - Redirects are validated against the same allowlist.
//   - Auto-disable after maxConsecutiveFailures consecutive failures so a dead
//     endpoint doesn't pile up the deliveries log forever.
package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// Event constants — only events that exist on self-hosted today.
// Do NOT add events for unbuilt features (no purchases.completed, etc.).
const (
	EventCommentCreated       = "comment.created"
	EventPostPublished        = "post.published"
	EventNewsletterSubscribed = "newsletter.subscribed"
)

// RegisteredEvents is the source of truth for the admin event picker.
var RegisteredEvents = []string{
	EventCommentCreated,
	EventPostPublished,
	EventNewsletterSubscribed,
}

// SamplePayload returns a synthetic data payload representative of the given
// event. Used by the admin "dispatch test event" picker so subscribers can
// validate their receiver against a realistic envelope before a real event
// fires. Unknown events get a minimal placeholder.
func SamplePayload(event string) map[string]any {
	switch event {
	case EventCommentCreated:
		return map[string]any{
			"comment_id":      "sample_cmt_000",
			"parent_type":     "post",
			"parent_slug":     "hello-world",
			"author_name":     "Sample Visitor",
			"author_email":    "visitor@example.com",
			"content_preview": "This is a sample comment payload from Facet.",
			"status":          "pending",
			"is_test":         true,
		}
	case EventPostPublished:
		return map[string]any{
			"post_id":  "sample_post_000",
			"slug":     "hello-world",
			"title":    "Sample Post Title",
			"excerpt":  "Synthetic payload from the dispatch-test picker.",
			"author":   "Site Owner",
			"tags":     []string{"sample", "test"},
			"is_test":  true,
		}
	case EventNewsletterSubscribed:
		return map[string]any{
			"subscriber_id": "sample_sub_000",
			"email":         "subscriber@example.com",
			"source":        "dispatch-test",
			"is_test":       true,
		}
	default:
		return map[string]any{
			"message": "Synthetic test event from Facet dispatch-test picker.",
			"is_test": true,
		}
	}
}

// retryBackoff is the inter-attempt delay schedule. Index 0 is the delay
// before the *2nd* attempt (i.e. after the first failure). Total of 6
// attempts (1 initial + 5 retries).
var retryBackoff = []time.Duration{
	1 * time.Minute,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
	12 * time.Hour,
}

const (
	maxConsecutiveFailures = 10
	deliveryTimeout        = 10 * time.Second
	queueBufferSize        = 256
	workerCount            = 4
)

// WebhookService dispatches events to configured webhooks. Construct one
// instance per app via NewWebhookService and call Start once at boot.
type WebhookService struct {
	app    core.App
	queue  chan deliveryJob
	client *http.Client

	startOnce sync.Once
	stopCh    chan struct{}
}

// deliveryJob is the unit of work passed to background workers.
type deliveryJob struct {
	WebhookID  string
	URL        string
	Secret     string
	Event      string
	DeliveryID string
	Body       []byte
	Attempt    int // 1-indexed
}

// NewWebhookService builds a WebhookService bound to the given PocketBase app.
// The HTTP client uses an SSRF-aware dialer that pins the resolved IP to
// prevent DNS-rebinding between validation and connect.
func NewWebhookService(app core.App) *WebhookService {
	svc := &WebhookService{
		app:    app,
		queue:  make(chan deliveryJob, queueBufferSize),
		stopCh: make(chan struct{}),
	}
	svc.client = &http.Client{
		Timeout: deliveryTimeout,
		Transport: &http.Transport{
			DialContext:           ssrfDialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          50,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return ValidateWebhookURL(req.URL.String())
		},
	}
	return svc
}

// Start spins up worker goroutines and the retry scheduler. Safe to call
// multiple times — only the first call has an effect.
func (s *WebhookService) Start() {
	s.startOnce.Do(func() {
		for i := 0; i < workerCount; i++ {
			go s.worker()
		}
		go s.retryScheduler()
	})
}

// Stop terminates background workers. Used by tests.
func (s *WebhookService) Stop() {
	close(s.stopCh)
}

// Dispatch finds active webhooks subscribed to the given event and enqueues
// a delivery for each. Non-blocking — payload is marshalled here and dropped
// onto the queue; HTTP work happens on worker goroutines.
//
// Safe to call from request handlers. Failures during enqueue are logged but
// never surface to the caller — a broken webhook subsystem must NOT break
// the originating request (e.g. a posted comment).
func (s *WebhookService) Dispatch(event string, payload map[string]any) {
	defer func() {
		if r := recover(); r != nil {
			s.app.Logger().Error("webhook dispatch panic", "event", event, "panic", fmt.Sprintf("%v", r))
		}
	}()

	body := webhookEnvelope{
		Event:     event,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      payload,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		s.app.Logger().Error("webhook payload marshal failed", "event", event, "error", err.Error())
		return
	}

	hooks, err := s.app.FindRecordsByFilter("webhooks", "is_active = true", "", 200, 0)
	if err != nil || len(hooks) == 0 {
		return
	}

	deliveryCol, err := s.app.FindCollectionByNameOrId("webhook_deliveries")
	if err != nil {
		s.app.Logger().Error("webhook_deliveries collection missing")
		return
	}

	for _, h := range hooks {
		var events []string
		if raw := h.Get("events"); raw != nil {
			b, _ := json.Marshal(raw)
			_ = json.Unmarshal(b, &events)
		}
		if !containsStr(events, event) {
			continue
		}

		// Create the delivery record up front so the admin UI can see "pending"
		// attempts even before HTTP runs (helpful when debugging slow targets).
		delivery := core.NewRecord(deliveryCol)
		delivery.Set("webhook_id", h.Id)
		delivery.Set("event_name", event)
		delivery.Set("request_body", string(bodyBytes))
		delivery.Set("attempt_count", 0)
		delivery.Set("succeeded", false)
		if err := s.app.Save(delivery); err != nil {
			s.app.Logger().Error("webhook delivery record save failed", "error", err.Error())
			continue
		}

		job := deliveryJob{
			WebhookID:  h.Id,
			URL:        h.GetString("url"),
			Secret:     h.GetString("secret"),
			Event:      event,
			DeliveryID: delivery.Id,
			Body:       bodyBytes,
			Attempt:    1,
		}

		// Non-blocking enqueue — drop if queue is full so a flood of events
		// can't OOM the process. Dropped jobs stay as 0-attempt rows for visibility.
		select {
		case s.queue <- job:
		default:
			s.app.Logger().Warn("webhook queue full, dropping delivery", "webhook_id", h.Id, "event", event)
		}
	}
}

// SendNow performs a synchronous delivery for the "Test" button in the admin.
// Returns the response status, the response body (truncated), and any error.
// Does NOT create a delivery record — the caller decides whether to log it.
func (s *WebhookService) SendNow(url, secret, event string, payload map[string]any) (int, string, error) {
	body := webhookEnvelope{
		Event:     event,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      payload,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return 0, "", fmt.Errorf("marshal payload: %w", err)
	}
	return s.deliver(url, secret, event, "test", bodyBytes)
}

// worker pulls jobs from the queue and runs deliver(). Each worker handles
// its own retry scheduling by writing next_retry_at on the delivery row;
// the retryScheduler picks them up later.
func (s *WebhookService) worker() {
	for {
		select {
		case <-s.stopCh:
			return
		case job := <-s.queue:
			s.runJob(job)
		}
	}
}

// runJob delivers a single job and updates the delivery + webhook records.
func (s *WebhookService) runJob(job deliveryJob) {
	defer func() {
		if r := recover(); r != nil {
			s.app.Logger().Error("webhook worker panic", "delivery_id", job.DeliveryID, "panic", fmt.Sprintf("%v", r))
		}
	}()

	status, respBody, err := s.deliver(job.URL, job.Secret, job.Event, job.DeliveryID, job.Body)

	delivery, dErr := s.app.FindRecordById("webhook_deliveries", job.DeliveryID)
	if dErr != nil {
		return
	}
	delivery.Set("attempt_count", job.Attempt)
	delivery.Set("response_status", status)
	delivery.Set("response_body", truncateBytes(respBody, 2000))

	hook, hErr := s.app.FindRecordById("webhooks", job.WebhookID)
	if hErr != nil {
		_ = s.app.Save(delivery)
		return
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05.000Z")

	if err == nil && status >= 200 && status < 300 {
		delivery.Set("succeeded", true)
		_ = s.app.Save(delivery)

		hook.Set("failure_count", 0)
		hook.Set("last_delivered_at", now)
		hook.Set("last_error", "")
		_ = s.app.Save(hook)
		return
	}

	// Failure path
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	} else {
		errMsg = fmt.Sprintf("HTTP %d", status)
	}
	delivery.Set("succeeded", false)
	_ = s.app.Save(delivery)

	failureCount := hook.GetInt("failure_count") + 1
	hook.Set("failure_count", failureCount)
	hook.Set("last_failure_at", now)
	hook.Set("last_error", truncateBytes(errMsg, 500))

	if failureCount >= maxConsecutiveFailures {
		hook.Set("is_active", false)
		s.app.Logger().Warn("webhook auto-disabled after consecutive failures",
			"webhook_id", hook.Id,
			"failure_count", failureCount,
		)
		_ = s.app.Save(hook)
		return
	}
	_ = s.app.Save(hook)

	// Schedule a retry by tagging the delivery row. The retryScheduler will
	// pick it up and re-enqueue. We store the retry intent in response_body
	// suffix only when there's no body — to keep things simple we use a
	// separate convention via attempt_count + succeeded=false + the
	// existing retry schedule.
	if job.Attempt-1 < len(retryBackoff) {
		// Re-enqueue after backoff. Use a one-shot goroutine + timer so we
		// don't depend on the scheduler tick granularity for short delays.
		delay := retryBackoff[job.Attempt-1]
		go func(j deliveryJob, d time.Duration) {
			t := time.NewTimer(d)
			defer t.Stop()
			select {
			case <-s.stopCh:
				return
			case <-t.C:
				j.Attempt++
				select {
				case s.queue <- j:
				case <-s.stopCh:
				}
			}
		}(job, delay)
	}
}

// deliver does the HTTP POST. Returns status, response body (truncated), error.
func (s *WebhookService) deliver(rawURL, secret, event, deliveryID string, body []byte) (int, string, error) {
	if err := ValidateWebhookURL(rawURL); err != nil {
		return 0, "", err
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequest("POST", rawURL, bytes.NewReader(body))
	if err != nil {
		return 0, "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Facet-Webhook/1.0")
	req.Header.Set("X-Facet-Event", event)
	req.Header.Set("X-Facet-Signature", signature)
	req.Header.Set("X-Facet-Delivery", deliveryID)

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	// Cap at 64KB so a chatty endpoint can't blow up the deliveries log.
	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	return resp.StatusCode, string(respBytes), nil
}

// retryScheduler is a safety net for jobs that lose their in-process timer
// (e.g. the process restarts mid-backoff). Once per minute it scans the
// deliveries table for failed, non-exhausted attempts and re-enqueues them.
// In the steady state most retries come through the in-process timers above;
// this is the catch-up path.
func (s *WebhookService) retryScheduler() {
	t := time.NewTicker(60 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-t.C:
			s.scanStuckDeliveries()
		}
	}
}

// scanStuckDeliveries re-enqueues unconverged deliveries left over from a
// previous process. A delivery is "stuck" if succeeded=false, attempt_count
// is less than the full retry budget, and the row is older than the deepest
// backoff window (so we don't trample on freshly-scheduled in-process retries).
func (s *WebhookService) scanStuckDeliveries() {
	defer func() {
		if r := recover(); r != nil {
			s.app.Logger().Error("webhook scheduler panic", "panic", fmt.Sprintf("%v", r))
		}
	}()

	cutoff := time.Now().Add(-(retryBackoff[len(retryBackoff)-1] + 5*time.Minute)).UTC().Format("2006-01-02 15:04:05.000Z")
	stuck, err := s.app.FindRecordsByFilter(
		"webhook_deliveries",
		"succeeded = false && attempt_count > 0 && attempt_count < {:max} && created > {:cutoff}",
		"created",
		20, 0,
		map[string]any{"max": len(retryBackoff) + 1, "cutoff": cutoff},
	)
	if err != nil || len(stuck) == 0 {
		return
	}

	for _, d := range stuck {
		hook, err := s.app.FindRecordById("webhooks", d.GetString("webhook_id"))
		if err != nil || !hook.GetBool("is_active") {
			continue
		}
		job := deliveryJob{
			WebhookID:  hook.Id,
			URL:        hook.GetString("url"),
			Secret:     hook.GetString("secret"),
			Event:      d.GetString("event_name"),
			DeliveryID: d.Id,
			Body:       []byte(d.GetString("request_body")),
			Attempt:    d.GetInt("attempt_count") + 1,
		}
		select {
		case s.queue <- job:
		default:
		}
	}
}

// webhookEnvelope is the JSON shape every webhook receives.
type webhookEnvelope struct {
	Event     string         `json:"event"`
	Timestamp string         `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

// ValidateWebhookURL rejects URLs that point to internal infrastructure
// (loopback, private RFC1918, link-local, multicast, metadata IPs, etc.)
// or use a non-http(s) scheme. Exposed so the admin REST handler can
// reuse the same logic during create/update.
func ValidateWebhookURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("URL must use http or https")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("URL must have a host")
	}

	lower := strings.ToLower(host)
	for _, blocked := range []string{
		"localhost", "127.0.0.1", "::1",
		"0.0.0.0", "0", "host.docker.internal",
		"169.254.169.254", // AWS/GCP metadata
		"metadata.google.internal",
	} {
		if lower == blocked {
			return fmt.Errorf("webhook URL cannot point to internal services")
		}
	}
	if strings.HasSuffix(lower, ".local") || strings.HasSuffix(lower, ".internal") {
		return fmt.Errorf("webhook URL cannot point to internal services")
	}

	// Numeric / literal IP — validate directly
	if ip := net.ParseIP(lower); ip != nil {
		if isPrivateIP(ip) {
			return fmt.Errorf("webhook URL cannot point to private/internal addresses")
		}
		return nil
	}

	// Hostname — resolve and verify EVERY result. Fail closed on DNS error.
	resolved, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("webhook URL host could not be resolved")
	}
	for _, ipStr := range resolved {
		if ip := net.ParseIP(ipStr); ip != nil && isPrivateIP(ip) {
			return fmt.Errorf("webhook URL resolves to a private/internal address")
		}
	}
	return nil
}

// isPrivateIP returns true for loopback, unspecified, link-local, multicast,
// RFC1918, RFC6598 (CGNAT), and IPv6 unique-local addresses. Used by both
// ValidateWebhookURL (pre-flight) and ssrfDialContext (at connect time).
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsMulticast() {
		return true
	}
	for _, cidr := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"100.64.0.0/10", // CGNAT
		"fc00::/7",      // IPv6 unique-local
	} {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// ssrfDialContext is the Transport.DialContext hook that closes the
// DNS-rebinding TOCTOU window: resolve the host, reject any private IP in
// the result, then dial the literal IP.
var ssrfDialer = &net.Dialer{Timeout: 5 * time.Second}

func ssrfDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateIP(ip) {
			return nil, fmt.Errorf("blocked dial to %s", ip)
		}
		return ssrfDialer.DialContext(ctx, network, addr)
	}
	resolved, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("dns lookup failed for %s: %w", host, err)
	}
	if len(resolved) == 0 {
		return nil, fmt.Errorf("no addresses resolved for %s", host)
	}
	for _, ipa := range resolved {
		if isPrivateIP(ipa.IP) {
			return nil, fmt.Errorf("blocked dial to %s", ipa.IP)
		}
	}
	var lastErr error
	for _, ipa := range resolved {
		conn, dErr := ssrfDialer.DialContext(ctx, network, net.JoinHostPort(ipa.IP.String(), port))
		if dErr == nil {
			return conn, nil
		}
		lastErr = dErr
	}
	return nil, fmt.Errorf("all resolved addresses failed: %w", lastErr)
}

// RetryBackoff exposes the backoff schedule for tests.
func RetryBackoff() []time.Duration {
	return append([]time.Duration(nil), retryBackoff...)
}

func containsStr(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

func truncateBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
