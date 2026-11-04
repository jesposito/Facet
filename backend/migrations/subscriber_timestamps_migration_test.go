package migrations

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const subscriberTimestampsMigrationFile = "1793750400_add_subscriber_timestamps.go"

func TestFreshSubscribersGetDatesAndTheSidebarFilterWorks(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	assertSubscriberTimestampFields(t, app)
	assertIndexExists(t, app, "idx_subscribers_created")
	sub := saveRecord(t, app, "subscribers", map[string]any{"email": "new@example.test", "status": "active"})
	if sub.GetDateTime("created").IsZero() || sub.GetDateTime("updated").IsZero() {
		t.Fatalf("new subscriber has created=%q updated=%q", sub.GetString("created"), sub.GetString("updated"))
	}
	if got := recentSubscriberIDs(t, app); !slices.Equal(got, []string{sub.Id}) {
		t.Fatalf("recent subscribers = %v, want only %s", got, sub.Id)
	}
}

func TestSubscriberTimestampMigrationLeavesUnknownDatesBlank(t *testing.T) {
	app := newIsolatedMigrationApp(t)
	if err := app.Save(legacySubscribers()); err != nil {
		t.Fatal(err)
	}
	old := saveRecord(t, app, "subscribers", map[string]any{
		"email": "old@example.test", "name": "Old", "source": "footer", "status": "unsubscribed", "bounce_count": 2,
	})
	assertOldRowKept := func(stage string) {
		t.Helper()
		if created, updated := rawSubscriberTimestamps(t, app, old.Id); created != "" || updated != "" {
			t.Fatalf("%s: legacy subscriber got created=%q updated=%q, want both empty", stage, created, updated)
		}
		got := findRecord(t, app, "subscribers", old.Id)
		if got.GetString("email") != "old@example.test" || got.GetString("name") != "Old" || got.GetString("source") != "footer" ||
			got.GetString("status") != "unsubscribed" || got.GetInt("bounce_count") != 2 {
			t.Fatalf("%s: legacy subscriber changed: %v", stage, got.PublicExport())
		}
	}

	runMigration(t, app, subscriberTimestampsMigrationFile)
	assertSubscriberTimestampFields(t, app)
	assertIndexExists(t, app, "idx_subscribers_created")
	assertOldRowKept("after up")

	schema := subscriberSchema(t, app)
	runMigration(t, app, subscriberTimestampsMigrationFile)
	if got := subscriberSchema(t, app); got != schema {
		t.Fatalf("repeat up changed the schema:\n%s\nwant:\n%s", got, schema)
	}
	assertOldRowKept("after repeat up")

	// Rollback keeps the additive timestamps; see the migration.
	for _, migration := range core.AppMigrations.Items() {
		if migration.File == subscriberTimestampsMigrationFile {
			if err := migration.Down(app); err != nil {
				t.Fatalf("down: %v", err)
			}
		}
	}
	assertSubscriberTimestampFields(t, app)
	assertOldRowKept("after down")
	runMigration(t, app, subscriberTimestampsMigrationFile)
	assertOldRowKept("after up again")

	fresh := saveRecord(t, app, "subscribers", map[string]any{"email": "new@example.test", "status": "active"})
	if fresh.GetDateTime("created").IsZero() || fresh.GetDateTime("updated").IsZero() {
		t.Fatalf("subscriber added after the upgrade has created=%q updated=%q", fresh.GetString("created"), fresh.GetString("updated"))
	}
	// The filter returns the new subscriber and never guesses at the old one.
	if got := recentSubscriberIDs(t, app); !slices.Equal(got, []string{fresh.Id}) {
		t.Fatalf("recent subscribers = %v, want only %s", got, fresh.Id)
	}

	// Editing a legacy row records the edit. Its signup date stays unknown.
	touchSubscriber(t, app, old.Id)
	if created, updated := rawSubscriberTimestamps(t, app, old.Id); created != "" || updated == "" {
		t.Fatalf("edited legacy subscriber has created=%q updated=%q, want only updated", created, updated)
	}
}

func TestSubscriberTimestampMigrationKeepsKnownDates(t *testing.T) {
	const knownCreated = "2024-03-01 09:30:00.000Z"
	const knownUpdated = "2024-04-02 10:45:00.000Z"

	for _, tc := range []struct {
		name       string
		hasUpdated bool
	}{
		{"created and updated already exist", true},
		{"only created already exists", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := newIsolatedMigrationApp(t)
			subscribers := legacySubscribers()
			subscribers.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
			if tc.hasUpdated {
				subscribers.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
			}
			if err := app.Save(subscribers); err != nil {
				t.Fatal(err)
			}
			sub := saveRecord(t, app, "subscribers", map[string]any{"email": "known@example.test", "status": "active"})
			setRawSubscriberTimestamp(t, app, sub.Id, "created", knownCreated)
			wantUpdated := ""
			if tc.hasUpdated {
				setRawSubscriberTimestamp(t, app, sub.Id, "updated", knownUpdated)
				wantUpdated = knownUpdated
			}
			createdFieldID := findCollection(t, app, "subscribers").Fields.GetByName("created").GetId()

			runMigration(t, app, subscriberTimestampsMigrationFile)
			assertSubscriberTimestampFields(t, app)
			if got := findCollection(t, app, "subscribers").Fields.GetByName("created").GetId(); got != createdFieldID {
				t.Fatalf("created field was replaced: id %s, want %s", got, createdFieldID)
			}
			if created, updated := rawSubscriberTimestamps(t, app, sub.Id); created != knownCreated || updated != wantUpdated {
				t.Fatalf("after up created=%q updated=%q, want %q and %q", created, updated, knownCreated, wantUpdated)
			}

			touchSubscriber(t, app, sub.Id)
			if created, updated := rawSubscriberTimestamps(t, app, sub.Id); created != knownCreated || updated <= knownUpdated {
				t.Fatalf("after edit created=%q updated=%q, want created kept and updated past %q", created, updated, knownUpdated)
			}
		})
	}
}

// legacySubscribers is the shape 1769840700_add_subscribers.go creates: no
// created or updated field.
func legacySubscribers() *core.Collection {
	subscribers := core.NewBaseCollection("subscribers")
	subscribers.Fields.Add(&core.EmailField{Name: "email", Required: true})
	subscribers.Fields.Add(&core.TextField{Name: "name", Max: 200})
	subscribers.Fields.Add(&core.TextField{Name: "source", Max: 100})
	subscribers.Fields.Add(&core.SelectField{Name: "status", Required: true, MaxSelect: 1, Values: []string{"active", "unsubscribed", "bounced"}})
	subscribers.Fields.Add(&core.NumberField{Name: "bounce_count"})
	subscribers.Indexes = []string{"CREATE UNIQUE INDEX idx_subscribers_email ON subscribers(email)"}
	return subscribers
}

func assertSubscriberTimestampFields(t *testing.T, app core.App) {
	t.Helper()
	subscribers := findCollection(t, app, "subscribers")
	for _, name := range []string{"created", "updated"} {
		field, ok := subscribers.Fields.GetByName(name).(*core.AutodateField)
		if !ok {
			t.Fatalf("subscribers.%s is missing or not autodate", name)
		}
		if !field.OnCreate || field.OnUpdate != (name == "updated") {
			t.Fatalf("subscribers.%s onCreate=%v onUpdate=%v", name, field.OnCreate, field.OnUpdate)
		}
	}
}

func subscriberSchema(t *testing.T, app core.App) string {
	t.Helper()
	subscribers := findCollection(t, app, "subscribers")
	schema, err := json.Marshal([]any{subscribers.Fields, subscribers.Indexes})
	if err != nil {
		t.Fatal(err)
	}
	return string(schema)
}

func rawSubscriberTimestamps(t *testing.T, app core.App, id string) (created, updated string) {
	t.Helper()
	if err := app.DB().NewQuery("SELECT created, updated FROM subscribers WHERE id = {:id}").
		Bind(map[string]any{"id": id}).Row(&created, &updated); err != nil {
		t.Fatal(err)
	}
	return created, updated
}

func setRawSubscriberTimestamp(t *testing.T, app core.App, id, field, value string) {
	t.Helper()
	if _, err := app.DB().NewQuery("UPDATE subscribers SET "+field+" = {:value} WHERE id = {:id}").
		Bind(map[string]any{"value": value, "id": id}).Execute(); err != nil {
		t.Fatal(err)
	}
}

func touchSubscriber(t *testing.T, app core.App, id string) {
	t.Helper()
	record := findRecord(t, app, "subscribers", id)
	record.Set("name", "Renamed")
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
}

// recentSubscriberIDs sends the request behind AdminSidebar's new-subscriber
// badge through the real API router and returns the ids it matches.
func recentSubscriberIDs(t *testing.T, app core.App) []string {
	t.Helper()
	superuser := core.NewRecord(findCollection(t, app, "_superusers"))
	superuser.SetEmail("owner@example.test")
	superuser.SetPassword("root-password-123")
	if err := app.Save(superuser); err != nil {
		t.Fatal(err)
	}
	token, err := superuser.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}

	since := time.Now().Add(-7 * 24 * time.Hour).UTC().Format("2006-01-02 15:04:05.000Z")
	filter := url.QueryEscape(`created >= "` + since + `"`)
	req := httptest.NewRequest(http.MethodGet, "/api/collections/subscribers/records?filter="+filter, nil)
	req.Header.Set("Authorization", token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("badge filter returned %d: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(body.Items))
	for _, item := range body.Items {
		ids = append(ids, item.ID)
	}
	return ids
}
