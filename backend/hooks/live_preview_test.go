package hooks

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
)

// realtimeClient registers a client the way an open SSE connection does.
func realtimeClient(t *testing.T, f *securityFixture, auth *core.Record, topics ...string) subscriptions.Client {
	t.Helper()
	client := subscriptions.NewDefaultClient()
	client.Set(apis.RealtimeClientAuthKey, auth)
	client.Subscribe(topics...)
	f.app.SubscriptionsBroker().Register(client)
	t.Cleanup(func() { f.app.SubscriptionsBroker().Unregister(client.Id()) })
	return client
}

func nextMessage(t *testing.T, client subscriptions.Client) subscriptions.Message {
	t.Helper()
	select {
	case message := <-client.Channel():
		return message
	case <-time.After(5 * time.Second):
		t.Fatal("no realtime message arrived")
		return subscriptions.Message{}
	}
}

func assertNoMessage(t *testing.T, client subscriptions.Client) {
	t.Helper()
	select {
	case message := <-client.Channel():
		t.Fatalf("unexpected realtime message %q", message.Name)
	case <-time.After(150 * time.Millisecond):
	}
}

func createAuthRecord(t *testing.T, f *securityFixture, collection *core.Collection, email string) *core.Record {
	t.Helper()
	record := newAuthRecord(collection, email, "a-strong-password")
	if err := f.app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record
}

func createMember(t *testing.T, f *securityFixture) *core.Record {
	t.Helper()
	members := core.NewAuthCollection("members")
	if err := f.app.Save(members); err != nil {
		t.Fatal(err)
	}
	return createAuthRecord(t, f, members, "member@example.com")
}

func updateSiteConfig(t *testing.T, f *securityFixture) {
	t.Helper()
	if _, err := services.UpdateSiteSettings(f.app, map[string]any{"custom_css": "body {}"}, nil); err != nil {
		t.Fatal(err)
	}
}

// A hook that doesn't call e.Next() also stops PocketBase's realtime broadcast,
// which runs further along the same chain.
func TestLogoHooksLeaveTheChainRunning(t *testing.T) {
	cases := []struct {
		collection string
		fields     map[string]any
		changed    string
	}{
		{"experience", map[string]any{"company": "Acme", "title": "Engineer"}, "title"},
		{"education", map[string]any{"institution": "Example University"}, "degree"},
	}

	for _, tc := range cases {
		t.Run(tc.collection, func(t *testing.T) {
			f := newSecurityFixture(t)
			RegisterExperienceHooks(f.pb)
			var created, updated int
			f.pb.OnRecordAfterCreateSuccess(tc.collection).BindFunc(func(e *core.RecordEvent) error {
				created++
				return e.Next()
			})
			f.pb.OnRecordAfterUpdateSuccess(tc.collection).BindFunc(func(e *core.RecordEvent) error {
				updated++
				return e.Next()
			})
			f.serve(t) // binds PocketBase's realtime broadcasts

			superusers, err := f.app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
			if err != nil {
				t.Fatal(err)
			}
			topic := tc.collection + "/*"
			client := realtimeClient(t, f, createAuthRecord(t, f, superusers, "root@example.com"), topic)

			record := f.create(t, tc.collection, tc.fields)
			assertRecordMessage(t, client, topic, "create")

			record.Set(tc.changed, "Changed")
			if err := f.app.Save(record); err != nil {
				t.Fatal(err)
			}
			assertRecordMessage(t, client, topic, "update")

			if created != 1 || updated != 1 {
				t.Fatalf("later hooks ran %d creates and %d updates, want 1 of each", created, updated)
			}
		})
	}
}

func assertRecordMessage(t *testing.T, client subscriptions.Client, topic, action string) {
	t.Helper()
	message := nextMessage(t, client)
	var data struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal(message.Data, &data); err != nil {
		t.Fatal(err)
	}
	if message.Name != topic || data.Action != action {
		t.Fatalf("message = %s %q, want %s %q", message.Name, data.Action, topic, action)
	}
}

func TestPublicSiteConfigTopicNeedsAnOwnerToSubscribe(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterAdminSessionGuard(f.pb)
	RegisterLivePreviewHooks(f.pb)
	h := f.serve(t)

	_, ownerToken := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	_, pendingToken := f.createAdmin(t, "pending@example.com", "a-strong-password", map[string]any{"totp_enabled": true})
	memberToken, err := createMember(t, f).NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}

	subscribe := func(token string) subscriptions.Client {
		t.Helper()
		client := realtimeClient(t, f, nil)
		body, err := json.Marshal(map[string]any{
			"clientId": client.Id(),
			"subscriptions": []string{
				"experience/*",
				publicSiteConfigTopic,
				publicSiteConfigTopic + `?options={"query":{}}`,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		assertStatus(t, do(t, h, testRequest{
			method:  http.MethodPost,
			url:     "/api/realtime",
			body:    string(body),
			headers: map[string]string{"Authorization": token},
		}), http.StatusNoContent)
		return client
	}

	denied := map[string]string{"guest": "", "member": memberToken, "admin without 2FA session": pendingToken}
	for name, token := range denied {
		client := subscribe(token)
		if got := client.Subscriptions(publicSiteConfigTopic + "?"); len(got) != 0 {
			t.Errorf("%s kept the topic: %v", name, got)
		}
		if !client.HasSubscription("experience/*") {
			t.Errorf("%s lost its other subscription", name)
		}
	}

	if !subscribe(ownerToken).HasSubscription(publicSiteConfigTopic) {
		t.Error("owner could not subscribe")
	}
}

func TestPublicSiteConfigSignalReachesOnlyOwners(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterLivePreviewHooks(f.pb)
	if _, err := services.LoadSiteSettings(f.app); err != nil {
		t.Fatal(err)
	}
	owner, _ := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	ownerClient := realtimeClient(t, f, owner, publicSiteConfigTopic)

	// These hold the topic directly, so only the delivery check can stop them.
	revoked := realtimeClient(t, f, owner, publicSiteConfigTopic)
	revoked.Unset(apis.RealtimeClientAuthKey)
	others := map[string]subscriptions.Client{
		"guest":   realtimeClient(t, f, nil, publicSiteConfigTopic),
		"member":  realtimeClient(t, f, createMember(t, f), publicSiteConfigTopic),
		"revoked": revoked,
	}

	updateSiteConfig(t, f)

	message := nextMessage(t, ownerClient)
	if message.Name != publicSiteConfigTopic || string(message.Data) != "{}" {
		t.Fatalf("message = %s %s, want an empty signal", message.Name, message.Data)
	}
	for name, client := range others {
		t.Run(name, func(t *testing.T) { assertNoMessage(t, client) })
	}
}

func TestHomepageViewCounterLeavesPreviewAlone(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterLivePreviewHooks(f.pb)
	settings, err := services.LoadSiteSettings(f.app)
	if err != nil {
		t.Fatal(err)
	}
	owner, _ := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	client := realtimeClient(t, f, owner, publicSiteConfigTopic)

	settings.Record.Set("homepage_view_count", settings.Record.GetInt("homepage_view_count")+1)
	settings.Record.Set("homepage_last_viewed_at", time.Now())
	if err := f.app.Save(settings.Record); err != nil {
		t.Fatal(err)
	}
	assertNoMessage(t, client)

	updateSiteConfig(t, f)
	nextMessage(t, client)
}

func TestFacetViewCounterLeavesPreviewAlone(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterLivePreviewHooks(f.pb)
	view := f.create(t, "views", map[string]any{"name": "Preview page", "slug": "preview-counter"})
	// Native requests load the stored record before updating its counters.
	var err error
	view, err = f.app.FindRecordById("views", view.Id)
	if err != nil {
		t.Fatal(err)
	}
	owner, _ := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	client := realtimeClient(t, f, owner, publicSiteConfigTopic)
	view.Set("view_count", 1)
	view.Set("last_viewed_at", time.Now())
	if err := f.app.Save(view); err != nil {
		t.Fatal(err)
	}
	assertNoMessage(t, client)
	// A configuration change still signals, even if a counter changes with it.
	view.Set("name", "Updated page")
	view.Set("view_count", 2)
	if err := f.app.Save(view); err != nil {
		t.Fatal(err)
	}
	nextMessage(t, client)
}
