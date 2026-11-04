package hooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func nativeForeignAccount(t *testing.T, f *securityFixture) *core.Record {
	t.Helper()
	collection := core.NewAuthCollection("other_accounts")
	if err := f.app.Save(collection); err != nil {
		t.Fatal(err)
	}
	record := newAuthRecord(collection, "other@example.invalid", "fixture-only-strong-password")
	if err := f.app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record
}

// Use the native password endpoint rather than a forged auth context: adding
// another auth collection must not turn its valid token into a Facet owner.
func nativeForeignToken(t *testing.T, h http.Handler) string {
	t.Helper()
	response := do(t, h, testRequest{
		method: http.MethodPost, url: "/api/collections/other_accounts/auth-with-password",
		body: `{"identity":"other@example.invalid","password":"fixture-only-strong-password"}`,
	})
	assertStatus(t, response, http.StatusOK)
	var auth struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &auth); err != nil || auth.Token == "" {
		t.Fatal("native authentication did not return a token")
	}
	return auth.Token
}

func TestNativeOwnerRulesRejectForeignAuthCollections(t *testing.T) {
	f := newMigratedSecurityFixture(t)
	RegisterCollectionRules(f.pb)
	RegisterAdminSessionGuard(f.pb)
	nativeForeignAccount(t, f)
	_, ownerToken := f.createAdmin(t, "owner@example.invalid", "owner-strong-password", nil)
	secrets := map[string]*core.Record{}
	for _, name := range []string{"projects", "awards", "demo_projects"} {
		secrets[name] = f.create(t, name, map[string]any{
			"title": "Private fixture " + name, "slug": "private-fixture", "description": "Private native body", "visibility": "private", "is_draft": true,
		})
	}
	h := f.serve(t)
	foreignToken := nativeForeignToken(t, h)
	foreign := map[string]string{"Authorization": foreignToken}
	owner := map[string]string{"Authorization": ownerToken}

	for name, record := range secrets {
		t.Run(name, func(t *testing.T) {
			base := "/api/collections/" + name + "/records"
			listed := do(t, h, testRequest{url: base, headers: foreign})
			assertStatus(t, listed, http.StatusForbidden)
			assertNotContains(t, listed, "Private fixture")
			for operation, request := range map[string]testRequest{
				"view":   {url: base + "/" + record.Id, headers: foreign},
				"create": {method: http.MethodPost, url: base, headers: foreign, body: `{"title":"Foreign create","slug":"foreign-create","visibility":"private"}`},
				"update": {method: http.MethodPatch, url: base + "/" + record.Id, headers: foreign, body: `{"title":"Foreign update"}`},
				"delete": {method: http.MethodDelete, url: base + "/" + record.Id, headers: foreign},
			} {
				response := do(t, h, request)
				if response.Code < 400 {
					t.Errorf("foreign auth %s returned %d", operation, response.Code)
				}
				assertNotContains(t, response, "Private fixture")
			}
			unchanged, err := f.app.FindRecordById(name, record.Id)
			if err != nil || unchanged.GetString("title") != "Private fixture "+name {
				t.Fatal("foreign auth changed or deleted private content")
			}
			if count, err := f.app.CountRecords(name); err != nil || count != 1 {
				t.Fatal("foreign auth created a native record")
			}
			ownerList := do(t, h, testRequest{url: base, headers: owner})
			assertStatus(t, ownerList, http.StatusOK)
			assertContains(t, ownerList, "Private fixture "+name)
			assertStatus(t, do(t, h, testRequest{url: base + "/" + record.Id, headers: owner}), http.StatusOK)
			createOwnedRecord(t, h, owner, name, `{"title":"Owner native record","slug":"owner-native-record","visibility":"private"}`)
		})
	}
	projects, err := f.app.FindCollectionByNameOrId("projects")
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, do(t, h, testRequest{url: "/api/collections/" + projects.Id + "/records", headers: foreign}), http.StatusForbidden)

	// Superusers retain PocketBase's own bypass, including nil server-only rules.
	superusers, err := f.app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		t.Fatal(err)
	}
	root := newAuthRecord(superusers, "root@example.invalid", "root-strong-password")
	if err := f.app.Save(root); err != nil {
		t.Fatal(err)
	}
	rootToken, err := root.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"projects", "course_progress"} {
		assertStatus(t, do(t, h, testRequest{url: "/api/collections/" + name + "/records", headers: map[string]string{"Authorization": rootToken}}), http.StatusOK)
	}
}

func TestNativeOwnerListsDoNotTurnLostSessionsIntoEmptyContent(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterCollectionRules(f.pb) // Intentionally before the session guard.
	RegisterAdminSessionGuard(f.pb)
	_, ownerToken := f.createAdmin(t, "owner@example.invalid", "owner-strong-password", nil)
	_, incompleteToken := f.createAdmin(t, "totp@example.invalid", "owner-strong-password", map[string]any{"totp_enabled": true})
	_, defaultToken := f.createAdmin(t, "default@example.invalid", defaultAdminPassword, nil)
	f.create(t, "projects", map[string]any{"title": "Preserve this draft", "slug": "preserve-draft", "visibility": "private"})
	h := f.serve(t)
	for name, token := range map[string]string{"anonymous": "", "invalid token": "invalid", "pending two-factor": incompleteToken, "default password": defaultToken} {
		t.Run(name, func(t *testing.T) {
			response := do(t, h, testRequest{url: "/api/collections/projects/records", headers: map[string]string{"Authorization": token}})
			assertStatus(t, response, http.StatusUnauthorized)
			assertNotContains(t, response, `"items"`)
			assertNotContains(t, response, "Preserve this draft")
		})
	}
	response := do(t, h, testRequest{url: "/api/collections/projects/records", headers: map[string]string{"Authorization": ownerToken}})
	assertStatus(t, response, http.StatusOK)
	assertContains(t, response, "Preserve this draft")
}

func TestNativeOwnerRulesPreserveCustomPoliciesAndServerLocks(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterCollectionRules(f.pb)
	RegisterAdminSessionGuard(f.pb)
	foreignRecord := nativeForeignAccount(t, f)

	// This unrelated operator collection intentionally admits another auth table.
	custom := core.NewBaseCollection("operator_records")
	custom.Fields.Add(&core.TextField{Name: "title"})
	legacy := "@request.auth.id != ''"
	custom.ListRule, custom.ViewRule, custom.CreateRule = &legacy, &legacy, &legacy
	if err := f.app.Save(custom); err != nil {
		t.Fatal(err)
	}
	operatorPublic := core.NewBaseCollection("operator_public_records")
	operatorPublic.Fields.Add(&core.TextField{Name: "title"})
	public := ""
	operatorPublic.ListRule, operatorPublic.ViewRule, operatorPublic.CreateRule = &public, &public, &public
	if err := f.app.Save(operatorPublic); err != nil {
		t.Fatal(err)
	}
	projects, err := f.app.FindCollectionByNameOrId("projects")
	if err != nil {
		t.Fatal(err)
	}
	narrow := "@request.auth.collectionName = 'users' && @request.auth.verified = true"
	projects.ListRule = &narrow
	if err := f.app.Save(projects); err != nil {
		t.Fatal(err)
	}
	users, err := f.app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	usersCreate := *users.CreateRule
	foreignCreate := foreignRecord.Collection().CreateRule
	h := f.serve(t)
	foreign := map[string]string{"Authorization": nativeForeignToken(t, h)}
	response := do(t, h, testRequest{method: http.MethodPost, url: "/api/collections/operator_records/records", headers: foreign, body: `{"title":"Operator intended access"}`})
	assertStatus(t, response, http.StatusOK)
	response = do(t, h, testRequest{url: "/api/collections/operator_records/records", headers: foreign})
	assertStatus(t, response, http.StatusOK)
	assertContains(t, response, "Operator intended access")
	// The old startup sweep closed public operator rules to authenticated access.
	// Keep that policy unchanged rather than silently narrowing unrelated tables.
	response = do(t, h, testRequest{method: http.MethodPost, url: "/api/collections/operator_public_records/records", headers: foreign, body: `{"title":"Existing operator policy"}`})
	assertStatus(t, response, http.StatusOK)
	response = do(t, h, testRequest{url: "/api/collections/operator_public_records/records", headers: foreign})
	assertStatus(t, response, http.StatusOK)
	assertContains(t, response, "Existing operator policy")
	// A custom filtered ListRule keeps PocketBase's normal empty-list semantics.
	assertStatus(t, do(t, h, testRequest{url: "/api/collections/projects/records", headers: foreign}), http.StatusOK)
	projects, err = f.app.FindCollectionByNameOrId("projects")
	if err != nil || projects.ListRule == nil || *projects.ListRule != narrow {
		t.Fatal("custom project policy changed")
	}
	users, err = f.app.FindCollectionByNameOrId("users")
	if err != nil || users.CreateRule == nil || *users.CreateRule != usersCreate {
		t.Fatal("users auth lifecycle rule changed")
	}
	other, err := f.app.FindCollectionByNameOrId("other_accounts")
	if err != nil || (other.CreateRule == nil) != (foreignCreate == nil) {
		t.Fatal("custom auth lifecycle rule changed")
	}
	for _, name := range []string{"course_progress", "lesson_progress", "course_certificates", "newsletter_recipients", "webhooks", "webhook_deliveries"} {
		collection, err := f.app.FindCollectionByNameOrId(name)
		if err != nil {
			t.Fatal(err)
		}
		if collection.CreateRule != nil || collection.UpdateRule != nil || collection.DeleteRule != nil {
			t.Errorf("server-only mutations opened for %s", name)
		}
	}
}

func TestNativeOwnerRulesMigrateOnlyKnownLegacyDefaults(t *testing.T) {
	for _, legacy := range []string{"@request.auth.id != ''", `@request.auth.id != ""`, " \n@request.auth.id!=''\t"} {
		for _, name := range []string{"projects", "admin_tags", "api_keys", "demo_projects"} {
			t.Run(name+legacy, func(t *testing.T) {
				collection := core.NewBaseCollection(name)
				collection.ListRule, collection.ViewRule, collection.CreateRule, collection.UpdateRule, collection.DeleteRule = &legacy, &legacy, &legacy, &legacy, &legacy
				if !tightenFacetDefaultRules(collection) {
					t.Fatal("known legacy default not migrated")
				}
				for _, rule := range []*string{collection.ListRule, collection.ViewRule, collection.CreateRule, collection.UpdateRule, collection.DeleteRule} {
					if rule == nil || *rule != facetOwnerRule {
						t.Fatal("legacy default did not become owner-only")
					}
				}
				if tightenFacetDefaultRules(collection) {
					t.Fatal("migration is not idempotent")
				}
			})
		}
	}
	for _, rule := range []string{"@request.auth.verified = true", "@request.auth.id != '' && @request.auth.verified = true", "@request.auth.id = id", "@request.auth.collectionName = '_superusers'"} {
		collection := core.NewBaseCollection("projects")
		collection.ListRule = &rule
		if tightenFacetDefaultRules(collection) || *collection.ListRule != rule {
			t.Fatal("custom policy was changed")
		}
	}
	for _, collection := range []*core.Collection{core.NewBaseCollection("operator_records"), core.NewAuthCollection("projects")} {
		legacy := "@request.auth.id != ''"
		collection.ListRule = &legacy
		if tightenFacetDefaultRules(collection) || *collection.ListRule != legacy {
			t.Fatal("unrelated or auth collection was changed")
		}
	}
	locked := core.NewBaseCollection("course_progress")
	if tightenFacetDefaultRules(locked) || locked.CreateRule != nil || locked.ListRule != nil {
		t.Fatal("nil server-only rules changed")
	}
}

func TestNativeOwnerRulesAbortStartupIfSavingACLsFails(t *testing.T) {
	f := newSecurityFixture(t)
	projects, err := f.app.FindCollectionByNameOrId("projects")
	if err != nil {
		t.Fatal(err)
	}
	public := ""
	projects.ListRule = &public
	if err := f.app.Save(projects); err != nil {
		t.Fatal(err)
	}
	injected := errors.New("injected ACL save failure")
	f.app.OnCollectionUpdate().BindFunc(func(e *core.CollectionEvent) error {
		if e.Collection.Name == "projects" {
			return injected
		}
		return e.Next()
	})
	RegisterCollectionRules(f.pb)
	router, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	ready := false
	err = f.app.OnServe().Trigger(&core.ServeEvent{App: f.app, Router: router}, func(e *core.ServeEvent) error { ready = true; return nil })
	if !errors.Is(err, injected) || !strings.Contains(err.Error(), "projects") || ready {
		t.Fatalf("startup did not fail closed with collection context: %v", err)
	}
}

func TestNativeOwnerRulesAllowMissingOptionalCollections(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.ClearBootstrap() })
	if err := enforceCollectionRules(&pocketbase.PocketBase{App: app}); err != nil {
		t.Fatal(fmt.Errorf("optional collections should be skipped: %w", err))
	}
}
