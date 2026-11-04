package hooks

import (
	"reflect"
	"slices"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/routine"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
)

// publicSiteConfigTopic tells the owner's live preview to reload after a
// site or navigation change. Site settings also hold private values, so they stay
// superuser-only and this message carries no record data.
const publicSiteConfigTopic = "public_site_config"

// RegisterLivePreviewHooks guards the topic and sends its signal.
func RegisterLivePreviewHooks(app *pocketbase.PocketBase) {
	// /api/realtime accepts any topic name, so this one is dropped for
	// everyone but the owner. The client's other subscriptions keep working.
	app.OnRealtimeSubscribeRequest().BindFunc(func(e *core.RealtimeSubscribeRequestEvent) error {
		if !isSuperuser(e.RequestEvent) {
			e.Subscriptions = slices.DeleteFunc(e.Subscriptions, func(sub string) bool {
				return strings.HasPrefix(sub+"?", publicSiteConfigTopic+"?")
			})
		}
		return e.Next()
	})

	signal := func(e *core.RecordEvent) error {
		sendPublicSiteConfigChanged(e.App)
		return e.Next()
	}
	for _, collection := range []string{"site_settings", "views"} {
		app.OnRecordAfterCreateSuccess(collection).BindFunc(signal)
		app.OnRecordAfterDeleteSuccess(collection).BindFunc(signal)
		app.OnRecordAfterUpdateSuccess(collection).BindFunc(func(e *core.RecordEvent) error {
			// Visitor counters don't change the public page's configuration.
			for _, field := range e.Record.Collection().Fields {
				switch field.GetName() {
				case "view_count", "last_viewed_at", "homepage_view_count", "homepage_last_viewed_at", "updated":
					continue
				}
				if !reflect.DeepEqual(e.Record.Get(field.GetName()), e.Record.Original().Get(field.GetName())) {
					return signal(e)
				}
			}
			return e.Next()
		})
	}
}

func sendPublicSiteConfigChanged(app core.App) {
	message := subscriptions.Message{Name: publicSiteConfigTopic, Data: []byte("{}")}
	for _, client := range app.SubscriptionsBroker().Clients() {
		if !client.HasSubscription(publicSiteConfigTopic) {
			continue
		}
		// Check again here: PocketBase clears this value when a token is revoked.
		auth, _ := client.Get(apis.RealtimeClientAuthKey).(*core.Record)
		if isSuperuser(&core.RequestEvent{Auth: auth}) {
			routine.FireAndForget(func() { client.Send(message) })
		}
	}
}
