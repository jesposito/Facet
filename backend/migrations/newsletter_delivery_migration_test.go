package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const newsletterDeliveryMigrationFile = "1793664000_harden_newsletter_delivery.go"

func TestNewsletterDeliveryMigrationOrdersRecipientsAndRollsBack(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	recipients := findCollection(t, app, "newsletter_recipients")
	for _, name := range []string{"created", "updated"} {
		if f := recipients.Fields.GetByName(name); f == nil || f.Type() != core.FieldTypeAutodate {
			t.Fatalf("newsletter_recipients.%s is missing or not autodate", name)
		}
	}
	send := saveRecord(t, app, "newsletter_sends", map[string]any{"subject": "S", "status": "interrupted"})
	rec := saveRecord(t, app, "newsletter_recipients", map[string]any{"send_id": send.Id, "subscriber_id": "sub123456789012", "state": "pending"})
	if rec.GetString("created") == "" {
		t.Fatal("recipient has no created timestamp")
	}
	if _, err := app.FindRecordsByFilter("newsletter_recipients", "send_id = {:s}", "created,id", 0, 0, map[string]any{"s": send.Id}); err != nil {
		t.Fatalf("sorting recipients by created failed: %v", err)
	}

	var down func(core.App) error
	for _, migration := range core.AppMigrations.Items() {
		if migration.File == newsletterDeliveryMigrationFile {
			down = migration.Down
		}
	}
	if down == nil {
		t.Fatalf("migration %s is not registered", newsletterDeliveryMigrationFile)
	}
	if err := down(app); err != nil {
		t.Fatalf("down: %v", err)
	}
	if _, err := app.FindCollectionByNameOrId("newsletter_recipients"); err == nil {
		t.Fatal("down kept newsletter_recipients")
	}
	if findCollection(t, app, "subscribers").Fields.GetByName("confirmation_attempted_at") != nil {
		t.Fatal("down kept subscribers.confirmation_attempted_at")
	}
	sends := findCollection(t, app, "newsletter_sends")
	if sends.Fields.GetByName("uncertain_count") != nil {
		t.Fatal("down kept newsletter_sends.uncertain_count")
	}
	rolled, err := app.FindRecordById("newsletter_sends", send.Id)
	if err != nil || rolled.GetString("status") != "failed" {
		t.Fatalf("interrupted send after down = %q, %v; want failed", rolled.GetString("status"), err)
	}
	runMigration(t, app, newsletterDeliveryMigrationFile)
	if findCollection(t, app, "newsletter_recipients").Fields.GetByName("created") == nil {
		t.Fatal("re-running up did not restore recipients")
	}
}

func TestFreshNewsletterSendsAndEventsHaveTimestamps(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	assertNewsletterTimestampFields(t, app)
	send := saveRecord(t, app, "newsletter_sends", map[string]any{"subject": "S", "status": "sent"})
	event := saveRecord(t, app, "newsletter_events", map[string]any{"send_id": send.Id, "subscriber_id": "sub123456789012", "event_type": "opened"})
	for _, r := range []*core.Record{send, event} {
		if r.GetString("created") == "" || r.GetString("updated") == "" {
			t.Fatalf("new %s record has created=%q updated=%q", r.Collection().Name, r.GetString("created"), r.GetString("updated"))
		}
	}
	// The queries send history and exports run.
	if _, err := app.FindRecordsByFilter("newsletter_sends", "id != ''", "-created,-id", 50, 0); err != nil {
		t.Fatalf("sorting sends by created: %v", err)
	}
	if _, err := app.FindRecordsByFilter("newsletter_events", "", "-created", 0, 0); err != nil {
		t.Fatalf("sorting events by created: %v", err)
	}
}

func TestNewsletterDeliveryMigrationUpgradesLegacySendHistory(t *testing.T) {
	app := newIsolatedMigrationApp(t)

	// The shape installs had before this migration: no timestamps.
	sends := core.NewBaseCollection("newsletter_sends")
	sends.Fields.Add(&core.TextField{Name: "subject", Required: true})
	sends.Fields.Add(&core.SelectField{Name: "status", Required: true, MaxSelect: 1, Values: []string{"draft", "sending", "sent", "failed"}})
	sends.Fields.Add(&core.NumberField{Name: "sent_count"})
	events := core.NewBaseCollection("newsletter_events")
	events.Fields.Add(&core.TextField{Name: "send_id", Required: true})
	events.Fields.Add(&core.TextField{Name: "subscriber_id"})
	events.Fields.Add(&core.SelectField{Name: "event_type", Required: true, MaxSelect: 1, Values: []string{"delivered", "opened", "clicked", "bounced", "complained"}})
	subscribers := core.NewBaseCollection("subscribers")
	subscribers.Fields.Add(&core.TextField{Name: "email"})
	for _, c := range []*core.Collection{sends, events, subscribers} {
		if err := app.Save(c); err != nil {
			t.Fatalf("create legacy %s: %v", c.Name, err)
		}
	}
	oldSend := saveRecord(t, app, "newsletter_sends", map[string]any{"subject": "Old", "status": "sent", "sent_count": 4})
	oldEvent := saveRecord(t, app, "newsletter_events", map[string]any{"send_id": oldSend.Id, "subscriber_id": "sub123456789012", "event_type": "opened"})

	runMigration(t, app, newsletterDeliveryMigrationFile)
	runMigration(t, app, newsletterDeliveryMigrationFile) // idempotent
	assertNewsletterTimestampFields(t, app)
	assertUnknownTimestamps := func(stage string) {
		t.Helper()
		for _, row := range []struct{ table, id string }{{"newsletter_sends", oldSend.Id}, {"newsletter_events", oldEvent.Id}} {
			var created, updated string
			if err := app.DB().NewQuery("SELECT created, updated FROM "+row.table+" WHERE id = {:id}").
				Bind(map[string]any{"id": row.id}).Row(&created, &updated); err != nil {
				t.Fatalf("%s: read %s: %v", stage, row.table, err)
			}
			if created != "" || updated != "" {
				t.Fatalf("%s: legacy %s row got created=%q updated=%q, want both empty", stage, row.table, created, updated)
			}
		}
	}
	assertUnknownTimestamps("after up")
	if got := findRecord(t, app, "newsletter_sends", oldSend.Id); got.GetInt("sent_count") != 4 || got.GetString("subject") != "Old" {
		t.Fatalf("legacy send changed: %v", got.PublicExport())
	}

	newSend := saveRecord(t, app, "newsletter_sends", map[string]any{"subject": "New", "status": "partial"})
	if newSend.GetString("created") == "" || newSend.GetString("updated") == "" {
		t.Fatal("send created after the upgrade has no timestamps")
	}
	records, err := app.FindRecordsByFilter("newsletter_sends", "id != ''", "-created,-id", 50, 0)
	if err != nil || len(records) != 2 || records[0].Id != newSend.Id {
		t.Fatalf("send history after upgrade = %v, %v; want new send first, then legacy", records, err)
	}

	var down func(core.App) error
	for _, migration := range core.AppMigrations.Items() {
		if migration.File == newsletterDeliveryMigrationFile {
			down = migration.Down
		}
	}
	if err := app.Delete(newSend); err != nil {
		t.Fatal(err)
	}
	if err := down(app); err != nil {
		t.Fatalf("down: %v", err)
	}
	// Rollback keeps the additive timestamps; see the migration.
	assertNewsletterTimestampFields(t, app)
	assertUnknownTimestamps("after down")
	runMigration(t, app, newsletterDeliveryMigrationFile)
	assertNewsletterTimestampFields(t, app)
	assertUnknownTimestamps("after up again")
}

func assertNewsletterTimestampFields(t *testing.T, app core.App) {
	t.Helper()
	for _, collection := range []string{"newsletter_sends", "newsletter_events"} {
		c := findCollection(t, app, collection)
		for _, name := range []string{"created", "updated"} {
			field, ok := c.Fields.GetByName(name).(*core.AutodateField)
			if !ok {
				t.Fatalf("%s.%s is missing or not autodate", collection, name)
			}
			if !field.OnCreate || field.OnUpdate != (name == "updated") {
				t.Fatalf("%s.%s onCreate=%v onUpdate=%v", collection, name, field.OnCreate, field.OnUpdate)
			}
		}
	}
}
