package hooks

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func newListCountFixture(t *testing.T) (*securityFixture, *core.Record) {
	t.Helper()
	f := newSecurityFixture(t)
	RegisterNewsletterListHooks(f.pb)
	list := f.create(t, "newsletter_lists", map[string]any{
		"name": "Greenhouse", "slug": "greenhouse", "description": "Seedlings", "is_active": true,
	})
	return f, list
}

func storedListRow(t *testing.T, f *securityFixture, id string) *core.Record {
	t.Helper()
	r, err := f.app.FindRecordById("newsletter_lists", id)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// addMember saves a new subscriber and their membership through app, which
// may be a transaction.
func addMember(t *testing.T, app core.App, listID, email string) *core.Record {
	t.Helper()
	subs, err := app.FindCollectionByNameOrId("subscribers")
	if err != nil {
		t.Fatal(err)
	}
	sub := core.NewRecord(subs)
	sub.Set("email", email)
	sub.Set("status", "active")
	if err := app.Save(sub); err != nil {
		t.Fatalf("save subscriber: %v", err)
	}
	col, err := app.FindCollectionByNameOrId("subscriber_list_memberships")
	if err != nil {
		t.Fatal(err)
	}
	m := core.NewRecord(col)
	m.Set("subscriber_id", sub.Id)
	m.Set("list_id", listID)
	if err := app.Save(m); err != nil {
		t.Fatalf("save membership: %v", err)
	}
	return m
}

func TestListCountUpdatesAsSoonAsSaveReturns(t *testing.T) {
	f, list := newListCountFixture(t)

	a := addMember(t, f.app, list.Id, "a@example.com")
	addMember(t, f.app, list.Id, "b@example.com")
	if got := storedListRow(t, f, list.Id).GetInt("subscriber_count"); got != 2 {
		t.Fatalf("subscriber_count after two joins = %d, want 2", got)
	}

	if err := f.app.Delete(a); err != nil {
		t.Fatal(err)
	}
	if got := storedListRow(t, f, list.Id).GetInt("subscriber_count"); got != 1 {
		t.Fatalf("subscriber_count after a leave = %d, want 1", got)
	}
}

func TestListCountLeavesOtherFieldsAlone(t *testing.T) {
	f, list := newListCountFixture(t)
	before := storedListRow(t, f, list.Id)

	// The owner renames the list; this copy is now the newest.
	edited := storedListRow(t, f, list.Id)
	edited.Set("name", "Glasshouse")
	edited.Set("description", "Tomatoes")
	if err := f.app.Save(edited); err != nil {
		t.Fatal(err)
	}
	edited = storedListRow(t, f, list.Id)

	addMember(t, f.app, list.Id, "a@example.com")

	after := storedListRow(t, f, list.Id)
	if after.GetString("name") != "Glasshouse" || after.GetString("description") != "Tomatoes" {
		t.Fatalf("list edit overwritten: name=%q description=%q", after.GetString("name"), after.GetString("description"))
	}
	if after.GetString("slug") != before.GetString("slug") || after.GetBool("is_active") != before.GetBool("is_active") {
		t.Fatalf("recount changed other fields: %v", after.PublicExport())
	}
	if after.GetString("updated") != edited.GetString("updated") {
		t.Fatalf("recount rewrote the list row's updated time: %s -> %s", edited.GetString("updated"), after.GetString("updated"))
	}
	if got := after.GetInt("subscriber_count"); got != 1 {
		t.Fatalf("subscriber_count = %d, want 1", got)
	}
}

func TestListCountStaysExactUnderConcurrentChanges(t *testing.T) {
	f, list := newListCountFixture(t)

	const n = 16
	members := make([]*core.Record, n)
	for i := range members {
		members[i] = addMember(t, f.app, list.Id, fmt.Sprintf("seed%d@example.com", i))
	}

	// Half the readers leave while as many new ones join.
	col, err := f.app.FindCollectionByNameOrId("subscriber_list_memberships")
	if err != nil {
		t.Fatal(err)
	}
	newSubs := make([]*core.Record, n/2)
	for i := range newSubs {
		newSubs[i] = f.create(t, "subscribers", map[string]any{"email": fmt.Sprintf("new%d@example.com", i), "status": "active"})
	}
	var wg sync.WaitGroup
	errs := make(chan error, n)
	start := make(chan struct{})
	for i := 0; i < n/2; i++ {
		wg.Add(2)
		go func(m *core.Record) {
			defer wg.Done()
			<-start
			errs <- f.app.Delete(m)
		}(members[i])
		go func(sub *core.Record) {
			defer wg.Done()
			<-start
			m := core.NewRecord(col)
			m.Set("subscriber_id", sub.Id)
			m.Set("list_id", list.Id)
			errs <- f.app.Save(m)
		}(newSubs[i])
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("membership change failed: %v", err)
		}
	}

	actual, err := f.app.CountRecords("subscriber_list_memberships")
	if err != nil {
		t.Fatal(err)
	}
	if got := storedListRow(t, f, list.Id).GetInt("subscriber_count"); int64(got) != actual || got != n {
		t.Fatalf("subscriber_count = %d, memberships = %d, want both %d", got, actual, n)
	}
}

func TestListCountFollowsTransactions(t *testing.T) {
	f, list := newListCountFixture(t)

	err := f.app.RunInTransaction(func(tx core.App) error {
		addMember(t, tx, list.Id, "a@example.com")
		addMember(t, tx, list.Id, "b@example.com")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := storedListRow(t, f, list.Id).GetInt("subscriber_count"); got != 2 {
		t.Fatalf("subscriber_count after committed joins = %d, want 2", got)
	}

	rollback := errors.New("roll back")
	err = f.app.RunInTransaction(func(tx core.App) error {
		addMember(t, tx, list.Id, "c@example.com")
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("transaction err = %v, want the rollback", err)
	}
	if got := storedListRow(t, f, list.Id).GetInt("subscriber_count"); got != 2 {
		t.Fatalf("subscriber_count after a rolled-back join = %d, want 2", got)
	}
}
