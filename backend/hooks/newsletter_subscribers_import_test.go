package hooks

import (
	"strings"
	"testing"

	"facet/services"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func newSubscriberImportTestApp(t *testing.T) (*tests.TestApp, string) {
	t.Helper()

	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	subs := core.NewBaseCollection("subscribers")
	subs.Fields.Add(&core.EmailField{Name: "email", Required: true})
	subs.Fields.Add(&core.TextField{Name: "name"})
	subs.Fields.Add(&core.TextField{Name: "source"})
	subs.Fields.Add(&core.SelectField{Name: "status", Required: true, Values: []string{"active", "unsubscribed", "bounced"}})
	subs.Fields.Add(&core.JSONField{Name: "tags", MaxSize: 10000})
	subs.Indexes = []string{"CREATE UNIQUE INDEX idx_subscribers_email_test ON subscribers(email)"}
	if err := app.Save(subs); err != nil {
		t.Fatalf("create subscribers: %v", err)
	}

	tags := core.NewBaseCollection("subscriber_tags")
	tags.Fields.Add(&core.TextField{Name: "name", Required: true})
	tags.Fields.Add(&core.NumberField{Name: "subscriber_count"})
	tags.Indexes = []string{"CREATE UNIQUE INDEX idx_subscriber_tags_name_test ON subscriber_tags(name)"}
	if err := app.Save(tags); err != nil {
		t.Fatalf("create subscriber_tags: %v", err)
	}

	lists := core.NewBaseCollection("newsletter_lists")
	lists.Fields.Add(&core.TextField{Name: "name"})
	lists.Fields.Add(&core.TextField{Name: "slug"})
	lists.Fields.Add(&core.BoolField{Name: "is_default"})
	if err := app.Save(lists); err != nil {
		t.Fatalf("create newsletter_lists: %v", err)
	}
	list := core.NewRecord(lists)
	list.Set("name", "Newsletter")
	list.Set("slug", "default")
	list.Set("is_default", true)
	if err := app.Save(list); err != nil {
		t.Fatalf("create default list: %v", err)
	}

	memberships := core.NewBaseCollection("subscriber_list_memberships")
	memberships.Fields.Add(&core.RelationField{Name: "subscriber_id", Required: true, CollectionId: subs.Id, MaxSelect: 1})
	memberships.Fields.Add(&core.RelationField{Name: "list_id", Required: true, CollectionId: lists.Id, MaxSelect: 1})
	memberships.Indexes = []string{"CREATE UNIQUE INDEX idx_slm_test ON subscriber_list_memberships(subscriber_id, list_id)"}
	if err := app.Save(memberships); err != nil {
		t.Fatalf("create memberships: %v", err)
	}

	return app, list.Id
}

func seedSubscriberTag(t *testing.T, app core.App, name string) string {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("subscriber_tags")
	if err != nil {
		t.Fatal(err)
	}
	rec := core.NewRecord(col)
	rec.Set("name", name)
	if err := app.Save(rec); err != nil {
		t.Fatalf("save tag: %v", err)
	}
	return rec.Id
}

func seedSubscriber(t *testing.T, app core.App, email, status string, tagIDs []string) string {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("subscribers")
	if err != nil {
		t.Fatal(err)
	}
	rec := core.NewRecord(col)
	rec.Set("email", email)
	rec.Set("status", status)
	rec.Set("tags", tagIDs)
	if err := app.Save(rec); err != nil {
		t.Fatalf("save subscriber: %v", err)
	}
	return rec.Id
}

func TestImportSubscribersCSVTagsAreCaseInsensitiveAndPreserveUnsubscribed(t *testing.T) {
	app, listID := newSubscriberImportTestApp(t)
	vipID := seedSubscriberTag(t, app, "VIP")
	existingID := seedSubscriber(t, app, "alice@example.com", "unsubscribed", []string{vipID})

	csv := "Email Address,First Name,Status,Tags\n" +
		"ALICE@example.com,Alice New,active,\"vip, Customers\"\n" +
		"bob@example.com,Bob,active,VIP\n"
	res, err := importSubscribersCSV(app, services.LoadPlanConfig(), strings.NewReader(csv), listID)
	if err != nil {
		t.Fatalf("importSubscribersCSV: %v", err)
	}
	if res.Created != 1 || res.Updated != 1 || res.Failed != 0 {
		t.Fatalf("result created=%d updated=%d failed=%d", res.Created, res.Updated, res.Failed)
	}
	if len(res.CreatedTags) != 1 || res.CreatedTags[0] != "Customers" {
		t.Fatalf("created tags = %v, want [Customers]", res.CreatedTags)
	}

	alice, err := app.FindRecordById("subscribers", existingID)
	if err != nil {
		t.Fatal(err)
	}
	if got := alice.GetString("status"); got != "unsubscribed" {
		t.Fatalf("existing subscriber status = %q, want unsubscribed", got)
	}
	if got := alice.GetString("name"); got != "Alice New" {
		t.Fatalf("existing subscriber name = %q", got)
	}
	if got := len(alice.GetStringSlice("tags")); got != 2 {
		t.Fatalf("existing subscriber tag count = %d, want 2", got)
	}

	tags, err := app.FindRecordsByFilter("subscriber_tags", "id != ''", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 {
		t.Fatalf("tag rows = %d, want 2 (no case duplicate for VIP)", len(tags))
	}
}
