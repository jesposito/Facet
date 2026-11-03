package hooks

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSendSemaphoreRespectsCap(t *testing.T) {
	capacity := sendSemaphoreCapacity()
	if capacity <= 0 {
		t.Fatalf("send semaphore must have positive capacity, got %d", capacity)
	}

	const total = 1000
	var inFlight atomic.Int64
	var peak atomic.Int64
	var wg sync.WaitGroup
	wg.Add(total)

	for i := 0; i < total; i++ {
		go func() {
			defer wg.Done()
			sendSemaphore <- struct{}{}
			cur := inFlight.Add(1)
			for {
				prev := peak.Load()
				if cur <= prev || peak.CompareAndSwap(prev, cur) {
					break
				}
			}
			time.Sleep(50 * time.Microsecond)
			inFlight.Add(-1)
			<-sendSemaphore
		}()
	}

	wg.Wait()

	if got := int(peak.Load()); got > capacity {
		t.Fatalf("peak concurrent sends %d exceeded semaphore cap %d", got, capacity)
	}
}

// Many readers' mail clients fetch the pixel and follow a link several times
// at once. Each event must be stored once, the counts must match the stored
// events, and all of it must be finished by the time the responses are.
func TestNewsletterTrackingCountsConcurrentEventsOnce(t *testing.T) {
	n := newNewsletterFixture(t, true)
	const link = "https://example.org/post"
	send := n.create(t, "newsletter_sends", map[string]any{
		"subject":   "Hello",
		"body_html": `<p>Read <a href="` + link + `">this</a></p>`,
		"status":    "sent",
	})
	const token = "reader-token"
	n.create(t, "subscribers", map[string]any{
		"email":                  "reader@example.com",
		"status":                 "active",
		"source":                 "newsletter",
		"unsubscribe_token_hash": n.crypto.HMACToken(token),
	})

	paths := []string{
		"/api/public/newsletter/open/" + send.Id + "/" + token,
		"/api/public/newsletter/click/" + send.Id + "/" + token + "/" + base64.URLEncoding.EncodeToString([]byte(link)),
	}
	const perPath = 16
	codes := make(chan int, perPath*len(paths))
	var wg sync.WaitGroup
	for _, path := range paths {
		for i := 0; i < perPath; i++ {
			wg.Add(1)
			go func(path string, i int) {
				defer wg.Done()
				req := httptest.NewRequest(http.MethodGet, path, nil)
				req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", i+1)
				rec := httptest.NewRecorder()
				n.handler.ServeHTTP(rec, req)
				codes <- rec.Code
			}(path, i)
		}
	}
	wg.Wait()
	close(codes)
	for code := range codes {
		if code != http.StatusOK && code != http.StatusFound {
			t.Fatalf("tracking request returned %d", code)
		}
	}

	// No waiting: the requests above have returned, so recording is done.
	for _, eventType := range []string{"opened", "clicked"} {
		events, err := n.app.FindRecordsByFilter("newsletter_events",
			"send_id = {:send} && event_type = {:type}", "", 0, 0,
			map[string]any{"send": send.Id, "type": eventType})
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 1 {
			t.Fatalf("%s events = %d, want 1", eventType, len(events))
		}
	}
	send, err := n.app.FindRecordById("newsletter_sends", send.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got := send.GetInt("open_count"); got != 1 {
		t.Fatalf("open_count = %d, want 1", got)
	}
	if got := send.GetInt("click_count"); got != 1 {
		t.Fatalf("click_count = %d, want 1", got)
	}
}
