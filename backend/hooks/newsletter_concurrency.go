package hooks

import (
	"os"
	"strconv"

	"facet/services"

	"github.com/pocketbase/pocketbase"
)

const defaultMaxConcurrentSends = 4

var sendSemaphore = func() chan struct{} {
	max := defaultMaxConcurrentSends
	if raw := os.Getenv("NEWSLETTER_MAX_CONCURRENT_SENDS"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			max = parsed
		}
	}
	return make(chan struct{}, max)
}()

func goSendNewsletter(app *pocketbase.PocketBase, crypto *services.CryptoService, sendID, subject, bodyHTML, templateSlug string, subscriberIDs []string) {
	services.SafeGo(app, "send-newsletter", func() {
		sendSemaphore <- struct{}{}
		defer func() { <-sendSemaphore }()
		sendNewsletter(app, crypto, sendID, subject, bodyHTML, templateSlug, subscriberIDs)
	})
}

func sendSemaphoreCapacity() int {
	return cap(sendSemaphore)
}
