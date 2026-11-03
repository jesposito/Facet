package hooks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newOwnerCollectionAccessFixture(t *testing.T) (*securityFixture, http.Handler) {
	t.Helper()
	f := newSecurityFixture(t)
	RegisterCollectionRules(f.pb)
	RegisterAdminSessionGuard(f.pb)
	return f, f.serve(t)
}

func TestOwnerCanManageCourseCollections(t *testing.T) {
	f, handler := newOwnerCollectionAccessFixture(t)
	_, token := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	auth := map[string]string{"Authorization": token}

	createOwnedRecord(t, handler, auth, "courses", `{"title":"Owner course","slug":"owner-course","access_tier":"free","visibility":"private"}`)
	course := f.create(t, "courses", map[string]any{"title": "Module parent", "slug": "module-parent", "access_tier": "free"})
	createOwnedRecord(t, handler, auth, "modules", fmt.Sprintf(`{"course":%q,"title":"Owner module","sort_order":1}`, course.Id))
	module := f.create(t, "modules", map[string]any{"course": course.Id, "title": "Lesson parent", "sort_order": 1})
	createOwnedRecord(t, handler, auth, "lessons", fmt.Sprintf(`{"module":%q,"title":"Owner lesson","content_type":"text","content":"Owner content","sort_order":1}`, module.Id))
}

func createOwnedRecord(t *testing.T, handler http.Handler, auth map[string]string, collection, body string) string {
	t.Helper()
	base := "/api/collections/" + collection + "/records"

	created := do(t, handler, testRequest{method: http.MethodPost, url: base, body: body, headers: auth})
	assertStatus(t, created, http.StatusOK)
	var record struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &record); err != nil || record.ID == "" {
		t.Fatalf("create %s returned no record ID: %s", collection, created.Body.String())
	}

	listed := do(t, handler, testRequest{url: base, headers: auth})
	assertStatus(t, listed, http.StatusOK)
	assertContains(t, listed, record.ID)
	assertStatus(t, do(t, handler, testRequest{url: base + "/" + record.ID, headers: auth}), http.StatusOK)

	updated := do(t, handler, testRequest{method: http.MethodPatch, url: base + "/" + record.ID, body: `{"title":"Updated by owner"}`, headers: auth})
	assertStatus(t, updated, http.StatusOK)
	assertContains(t, updated, "Updated by owner")
	assertStatus(t, do(t, handler, testRequest{method: http.MethodDelete, url: base + "/" + record.ID, headers: auth}), http.StatusNoContent)
	return record.ID
}

func TestCourseCollectionsStillDenyAnonymousAccess(t *testing.T) {
	f, handler := newOwnerCollectionAccessFixture(t)
	_, token := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	auth := map[string]string{"Authorization": token}
	course := f.create(t, "courses", map[string]any{"title": "Private course", "slug": "private-course", "access_tier": "free", "visibility": "private"})
	module := f.create(t, "modules", map[string]any{"course": course.Id, "title": "Private module", "sort_order": 1})
	lesson := f.create(t, "lessons", map[string]any{"module": module.Id, "title": "Private lesson", "content_type": "text", "content": "Secret lesson", "sort_order": 1})

	for _, tc := range []struct {
		collection string
		id         string
		field      string
		secret     string
		payload    string
	}{
		{"courses", course.Id, "title", "Private course", `{"title":"Anonymous course","slug":"anonymous-course","access_tier":"free","visibility":"private"}`},
		{"modules", module.Id, "title", "Private module", fmt.Sprintf(`{"course":%q,"title":"Anonymous module","sort_order":1}`, course.Id)},
		{"lessons", lesson.Id, "content", "Secret lesson", fmt.Sprintf(`{"module":%q,"title":"Anonymous lesson","content_type":"text","content":"Anonymous content","sort_order":1}`, module.Id)},
	} {
		t.Run(tc.collection, func(t *testing.T) {
			base := "/api/collections/" + tc.collection + "/records"

			listed := do(t, handler, testRequest{url: base})
			assertStatus(t, listed, http.StatusOK)
			var list struct {
				TotalItems int               `json:"totalItems"`
				Items      []json.RawMessage `json:"items"`
			}
			if err := json.Unmarshal(listed.Body.Bytes(), &list); err != nil {
				t.Fatalf("decode anonymous list: %v", err)
			}
			if list.TotalItems != 0 || len(list.Items) != 0 {
				t.Fatalf("anonymous list returned records: %s", listed.Body.String())
			}

			createOwnedRecord(t, handler, auth, tc.collection, tc.payload)
			created := do(t, handler, testRequest{method: http.MethodPost, url: base, body: tc.payload})
			assertFailedCreateRule(t, created)

			for operation, request := range map[string]testRequest{
				"view":   {url: base + "/" + tc.id},
				"update": {method: http.MethodPatch, url: base + "/" + tc.id, body: `{"title":"Changed anonymously"}`},
				"delete": {method: http.MethodDelete, url: base + "/" + tc.id},
			} {
				if rec := do(t, handler, request); rec.Code < 400 {
					t.Fatalf("anonymous %s returned %d: %s", operation, rec.Code, rec.Body.String())
				}
			}

			record, err := f.app.FindRecordById(tc.collection, tc.id)
			if err != nil {
				t.Fatalf("anonymous request deleted %s: %v", tc.collection, err)
			}
			if got := record.GetString(tc.field); got != tc.secret {
				t.Fatalf("anonymous request changed %s %s to %q", tc.collection, tc.field, got)
			}
			if count, err := f.app.CountRecords(tc.collection); err != nil || count != 1 {
				t.Fatalf("%s count = %d, err = %v; want 1", tc.collection, count, err)
			}
		})
	}
}

func TestIncompleteAdminSignInCannotManageCourses(t *testing.T) {
	for _, tc := range []struct {
		name     string
		password string
		fields   map[string]any
	}{
		{"pending two-factor", "a-strong-password", map[string]any{"totp_enabled": true}},
		{"default password", defaultAdminPassword, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, handler := newOwnerCollectionAccessFixture(t)
			_, incompleteToken := f.createAdmin(t, "incomplete@example.com", tc.password, tc.fields)
			_, ownerToken := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
			payload := `{"title":"Blocked course","slug":"blocked-course","access_tier":"free","visibility":"private"}`

			createOwnedRecord(t, handler, map[string]string{"Authorization": ownerToken}, "courses", payload)
			rec := do(t, handler, testRequest{
				method:  http.MethodPost,
				url:     "/api/collections/courses/records",
				body:    payload,
				headers: map[string]string{"Authorization": incompleteToken},
			})
			assertFailedCreateRule(t, rec)
			if count, err := f.app.CountRecords("courses"); err != nil || count != 0 {
				t.Fatalf("courses count = %d, err = %v; want 0", count, err)
			}
			if _, err := f.app.FindFirstRecordByFilter("courses", "slug = {:slug}", map[string]any{"slug": "blocked-course"}); err == nil {
				t.Fatal("incomplete admin sign-in created a course")
			}
		})
	}
}

func assertFailedCreateRule(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	assertStatus(t, rec, http.StatusBadRequest)
	const want = `{"data":{},"message":"Failed to create record.","status":400}`
	if rec.Body.String() != want {
		t.Fatalf("create rule response = %s, want %s", rec.Body.String(), want)
	}
}

func TestServerManagedCollectionsRemainSuperuserOnly(t *testing.T) {
	f, handler := newOwnerCollectionAccessFixture(t)
	_, token := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	auth := map[string]string{"Authorization": token}

	for _, name := range []string{"course_progress", "lesson_progress", "course_certificates", "webhooks", "webhook_deliveries"} {
		t.Run(name, func(t *testing.T) {
			collection, err := f.app.FindCollectionByNameOrId(name)
			if err != nil {
				t.Fatal(err)
			}
			for operation, rule := range map[string]*string{
				"create": collection.CreateRule,
				"update": collection.UpdateRule,
				"delete": collection.DeleteRule,
			} {
				if rule != nil {
					t.Errorf("%s %s rule = %q, want superuser-only", name, operation, *rule)
				}
			}
			rec := do(t, handler, testRequest{method: http.MethodPost, url: "/api/collections/" + name + "/records", body: `{}`, headers: auth})
			assertStatus(t, rec, http.StatusForbidden)
		})
	}
}
