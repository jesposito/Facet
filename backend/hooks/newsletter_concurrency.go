package hooks

import (
	"facet/services"

	"github.com/pocketbase/pocketbase"
)

var sendSemaphore = make(chan struct{}, 1)

func goSendNewsletter(app *pocketbase.PocketBase, crypto *services.CryptoService, sendID, subject, bodyHTML, templateSlug string, subscriberIDs []string, personalization *newsletterPersonalization) {
	services.SafeGo(app, "send-newsletter", func() {
		sendSemaphore <- struct{}{}
		defer func() { <-sendSemaphore }()
		sendNewsletter(app, crypto, sendID, subject, bodyHTML, templateSlug, subscriberIDs, personalization)
	})
}

func sendSemaphoreCapacity() int {
	return cap(sendSemaphore)
}
