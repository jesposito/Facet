package hooks

import (
	"net/http"
	"testing"
)

// The migrated test database is the state an existing install starts from.
// After the startup sweep, no collection rule may be public.
func TestStartupClosesEveryPublicCollectionRule(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterCollectionRules(f.pb)
	f.serve(t)

	collections, err := f.app.FindAllCollections()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range collections {
		if c.System || c.IsAuth() {
			continue
		}
		for name, rule := range map[string]*string{
			"list": c.ListRule, "view": c.ViewRule, "create": c.CreateRule, "update": c.UpdateRule, "delete": c.DeleteRule,
		} {
			if rule != nil && *rule == "" {
				t.Errorf("%s %s rule is still public", c.Name, name)
			}
		}
	}

	for _, name := range []string{"webhooks", "webhook_deliveries"} {
		c, err := f.app.FindCollectionByNameOrId(name)
		if err != nil {
			t.Fatal(err)
		}
		if c.ListRule != nil || c.CreateRule != nil || c.UpdateRule != nil || c.DeleteRule != nil {
			t.Errorf("%s should be superuser-only", name)
		}
	}

	// OAuth sign-in checks the users create rule, so the sweep leaves it.
	users, err := f.app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	if users.CreateRule == nil || *users.CreateRule != "" {
		t.Errorf("users create rule changed; OAuth sign-up of allowlisted admins depends on it")
	}
}

func TestPreviouslyPublicCollectionsNeedSignIn(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterCollectionRules(f.pb)

	course := f.create(t, "courses", map[string]any{"title": "Paid course", "slug": "paid-course", "access_tier": "paid", "price": 900, "visibility": "private"})
	module := f.create(t, "modules", map[string]any{"course": course.Id, "title": "Module", "sort_order": 1})
	f.create(t, "lessons", map[string]any{"module": module.Id, "title": "Lesson", "content_type": "text", "content": "secret lesson body", "sort_order": 1})
	f.create(t, "course_certificates", map[string]any{"course": course.Id, "buyer_email": "student@example.com", "certificate_id": "CERT-1", "student_name": "Student"})
	f.create(t, "contact_methods", map[string]any{"type": "phone", "value": "+64 21 555 0100", "protection_level": "none", "visibility": "private"})
	_, adminToken := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	h := f.serve(t)

	for _, tc := range []struct{ collection, secret string }{
		{"lessons", "secret lesson body"},
		{"courses", "Paid course"},
		{"course_certificates", "student@example.com"},
		{"contact_methods", "+64 21 555 0100"},
	} {
		t.Run(tc.collection, func(t *testing.T) {
			url := "/api/collections/" + tc.collection + "/records"
			anonymous := do(t, h, testRequest{url: url})
			assertNotContains(t, anonymous, tc.secret)

			admin := do(t, h, testRequest{url: url, headers: map[string]string{"Authorization": adminToken}})
			assertStatus(t, admin, http.StatusOK)
			assertContains(t, admin, tc.secret)
		})
	}

	t.Run("webhooks cannot be created anonymously", func(t *testing.T) {
		rec := do(t, h, testRequest{method: http.MethodPost, url: "/api/collections/webhooks/records", body: `{"url":"https://attacker.example/hook","secret":"x","events":["*"],"enabled":true}`})
		if rec.Code < 400 {
			t.Fatalf("anonymous webhook create returned %d: %s", rec.Code, rec.Body.String())
		}
		if n, _ := f.app.CountRecords("webhooks"); n != 0 {
			t.Fatalf("webhook was created anonymously")
		}
	})
}

func TestAdminAccountsCannotBeCreatedThroughSignUp(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterAdminAuth(f.pb)
	h := f.serve(t)

	signUp := `{"email":"intruder@example.com","password":"intruder-password","passwordConfirm":"intruder-password"}`
	rec := do(t, h, testRequest{method: http.MethodPost, url: "/api/collections/users/records", body: signUp})
	assertStatus(t, rec, http.StatusForbidden)
	if _, err := f.app.FindAuthRecordByEmail("users", "intruder@example.com"); err == nil {
		t.Fatal("anonymous sign-up created a user")
	}

	// A PocketBase superuser can still add an admin from the dashboard.
	superusers, err := f.app.FindCollectionByNameOrId("_superusers")
	if err != nil {
		t.Fatal(err)
	}
	superuser := newAuthRecord(superusers, "root@example.com", "root-password-123")
	if err := f.app.Save(superuser); err != nil {
		t.Fatal(err)
	}
	token, err := superuser.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	rec = do(t, h, testRequest{method: http.MethodPost, url: "/api/collections/users/records", body: `{"email":"second@example.com","password":"second-password","passwordConfirm":"second-password"}`, headers: map[string]string{"Authorization": token}})
	assertStatus(t, rec, http.StatusOK)
}
