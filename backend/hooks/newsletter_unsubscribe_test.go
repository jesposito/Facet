package hooks

import "testing"

func TestNewsletterCountFieldForRejectsUnknown(t *testing.T) {
	allowed := map[string]string{
		"open":  "open_count",
		"click": "click_count",
	}
	for eventType, want := range allowed {
		got, ok := newsletterCountFieldFor(eventType)
		if !ok || got != want {
			t.Fatalf("newsletterCountFieldFor(%q) = %q, %v; want %q, true", eventType, got, ok, want)
		}
	}

	for _, eventType := range []string{
		"",
		"OPEN",
		"unsubscribe",
		"open_count",
		"open; DROP TABLE newsletter_sends;--",
		"newsletter_sends.open_count",
	} {
		if got, ok := newsletterCountFieldFor(eventType); ok || got != "" {
			t.Errorf("newsletterCountFieldFor(%q) = %q, %v; want empty, false", eventType, got, ok)
		}
	}
}
