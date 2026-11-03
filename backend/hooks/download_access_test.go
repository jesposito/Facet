package hooks

import (
	"encoding/json"
	"net/http"
	"testing"

	"facet/services"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// A signed download link only names a file. Every redemption asks again
// whether the visitor may have it, so a link captured before a refund or
// before the item was hidden stops working at once, not when it expires.

func signedLink(t *testing.T, f *securityFixture, contentType string, record *core.Record, field string) string {
	t.Helper()
	token, _ := services.NewDownloadService(f.crypto).GenerateSignedURL(contentType, record.Id, record.GetStringSlice(field)[0])
	return "/api/downloads/" + token
}

func setFields(t *testing.T, f *securityFixture, record *core.Record, fields map[string]any) {
	t.Helper()
	for k, v := range fields {
		record.Set(k, v)
	}
	if err := f.app.Save(record); err != nil {
		t.Fatal(err)
	}
}

func TestCapturedPaidDownloadLinkFollowsCurrentAccess(t *testing.T) {
	f, r := newPaidContentFixture(t)
	post := r["post"]
	buyer := f.completedPurchase(t, "buyer@example.com", "posts", post.Id)
	cookie := map[string]string{"Cookie": f.buyerCookie(t, "buyer@example.com", buyer)}
	h := f.serve(t)

	rec := do(t, h, testRequest{url: "/api/content/posts/" + post.Id + "/download-urls", headers: cookie})
	assertStatus(t, rec, http.StatusOK)
	var body struct {
		Files []struct {
			URL string `json:"url"`
		} `json:"files"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || len(body.Files) != 1 {
		t.Fatalf("unexpected download-urls response: %s", rec.Body.String())
	}
	link := body.Files[0].URL

	rec = do(t, h, testRequest{url: link, headers: cookie})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, "PAID GUIDE BYTES")

	for name, c := range map[string]string{
		"no cookie":                "",
		"forged cookie":            services.PurchaseCookieName + "=not-a-real-token",
		"another buyer's purchase": f.buyerCookie(t, "intruder@example.com", buyer),
	} {
		t.Run(name+" is refused", func(t *testing.T) {
			rec := do(t, h, testRequest{url: link, headers: map[string]string{"Cookie": c}})
			assertStatus(t, rec, http.StatusForbidden)
			assertNotContains(t, rec, "PAID GUIDE BYTES")
		})
	}

	t.Run("refund revokes the captured link", func(t *testing.T) {
		setFields(t, f, buyer, map[string]any{"status": "refunded"})
		defer setFields(t, f, buyer, map[string]any{"status": "completed"})
		rec := do(t, h, testRequest{url: link, headers: cookie})
		assertStatus(t, rec, http.StatusForbidden)
		assertNotContains(t, rec, "PAID GUIDE BYTES")
	})

	for name, fields := range map[string]map[string]any{
		"private": {"visibility": "private"},
		"draft":   {"is_draft": true},
	} {
		t.Run(name+" revokes the captured link", func(t *testing.T) {
			setFields(t, f, post, fields)
			defer setFields(t, f, post, map[string]any{"visibility": "public", "is_draft": false})
			rec := do(t, h, testRequest{url: link, headers: cookie})
			assertStatus(t, rec, http.StatusNotFound)
			assertNotContains(t, rec, "PAID GUIDE BYTES")
		})
	}

	t.Run("deleted record stops serving", func(t *testing.T) {
		if err := f.app.Delete(post); err != nil {
			t.Fatal(err)
		}
		assertStatus(t, do(t, h, testRequest{url: link, headers: cookie}), http.StatusNotFound)
	})
	f.waitForDownloadLogs(t, 1)
}

func TestFreePublicDownloadNeedsNoCookie(t *testing.T) {
	f, _ := newPaidContentFixture(t)
	file, err := filesystem.NewFileFromBytes([]byte("FREE FILE BYTES"), "free.pdf")
	if err != nil {
		t.Fatal(err)
	}
	post := f.create(t, "posts", map[string]any{
		"title": "Free files", "slug": "free-files", "content": "FREE", "visibility": "public",
		"published_at": "2026-01-03 00:00:00.000Z", "downloadable_files": []*filesystem.File{file},
	})
	h := f.serve(t)

	assertStatus(t, do(t, h, testRequest{url: "/api/content/posts/" + post.Id + "/download-urls"}), http.StatusOK)
	rec := do(t, h, testRequest{url: signedLink(t, f, "posts", post, "downloadable_files")})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, "FREE FILE BYTES")

	setFields(t, f, post, map[string]any{"visibility": "private"})
	assertStatus(t, do(t, h, testRequest{url: signedLink(t, f, "posts", post, "downloadable_files")}), http.StatusNotFound)
	f.waitForDownloadLogs(t, 1)
}

func TestCapturedLessonAttachmentLinkFollowsCurrentAccess(t *testing.T) {
	c := newCourseFixture(t)
	buyer := c.completedPurchase(t, "first@example.com", "courses", c.course.Id)
	cookie := map[string]string{"Cookie": c.buyerCookie(t, "first@example.com", buyer)}
	link := signedLink(t, c.securityFixture, "lessons", c.lesson, "attachments")

	rec := do(t, c.handler, testRequest{url: link, headers: cookie})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, "LESSON WORKSHEET BYTES")

	for name, h := range map[string]string{
		"no cookie":                "",
		"forged cookie":            services.PurchaseCookieName + "=not-a-real-token",
		"another buyer's purchase": c.buyerCookie(t, "intruder@example.com", buyer),
	} {
		t.Run(name+" is refused", func(t *testing.T) {
			rec := do(t, c.handler, testRequest{url: link, headers: map[string]string{"Cookie": h}})
			assertStatus(t, rec, http.StatusForbidden)
			assertNotContains(t, rec, "LESSON WORKSHEET BYTES")
		})
	}

	t.Run("refund revokes the captured link", func(t *testing.T) {
		setFields(t, c.securityFixture, buyer, map[string]any{"status": "refunded"})
		defer setFields(t, c.securityFixture, buyer, map[string]any{"status": "completed"})
		assertStatus(t, do(t, c.handler, testRequest{url: link, headers: cookie}), http.StatusForbidden)
	})

	for name, fields := range map[string]map[string]any{
		"private course": {"visibility": "private"},
		"draft course":   {"is_draft": true},
	} {
		t.Run(name+" revokes the captured link", func(t *testing.T) {
			setFields(t, c.securityFixture, c.course, fields)
			defer setFields(t, c.securityFixture, c.course, map[string]any{"visibility": "public", "is_draft": false})
			assertStatus(t, do(t, c.handler, testRequest{url: link, headers: cookie}), http.StatusNotFound)
		})
	}

	t.Run("drip lock holds the attachment back", func(t *testing.T) {
		module, err := c.app.FindRecordById("modules", c.lesson.GetString("module"))
		if err != nil {
			t.Fatal(err)
		}
		setFields(t, c.securityFixture, module, map[string]any{"drip_unlock_date": "2099-01-01 00:00:00.000Z"})
		defer setFields(t, c.securityFixture, module, map[string]any{"drip_unlock_date": ""})
		assertStatus(t, do(t, c.handler, testRequest{url: link, headers: cookie}), http.StatusForbidden)
	})

	t.Run("sequential access holds later lessons back", func(t *testing.T) {
		sheet, err := filesystem.NewFileFromBytes([]byte("SECOND LESSON BYTES"), "second.pdf")
		if err != nil {
			t.Fatal(err)
		}
		second := c.create(t, "lessons", map[string]any{"module": c.lesson.GetString("module"), "title": "Second", "content_type": "text", "content": "SECOND", "sort_order": 2, "attachments": []*filesystem.File{sheet}})
		secondLink := signedLink(t, c.securityFixture, "lessons", second, "attachments")
		assertStatus(t, do(t, c.handler, testRequest{url: secondLink, headers: cookie}), http.StatusOK)

		setFields(t, c.securityFixture, c.course, map[string]any{"sequential_access": true})
		defer setFields(t, c.securityFixture, c.course, map[string]any{"sequential_access": false})
		assertStatus(t, do(t, c.handler, testRequest{url: secondLink, headers: cookie}), http.StatusForbidden)

		assertStatus(t, do(t, c.handler, testRequest{method: http.MethodPost, url: "/api/courses/course/lessons/" + c.lesson.Id + "/complete", headers: cookie}), http.StatusOK)
		rec := do(t, c.handler, testRequest{url: secondLink, headers: cookie})
		assertStatus(t, rec, http.StatusOK)
		assertContains(t, rec, "SECOND LESSON BYTES")
	})
	c.waitForDownloadLogs(t, 3)
}

func TestPreviewLessonAttachmentNeedsNoCookie(t *testing.T) {
	c := newCourseFixture(t)
	sheet, err := filesystem.NewFileFromBytes([]byte("PREVIEW BYTES"), "preview.pdf")
	if err != nil {
		t.Fatal(err)
	}
	preview := c.create(t, "lessons", map[string]any{"module": c.lesson.GetString("module"), "title": "Preview", "content_type": "text", "content": "PREVIEW", "sort_order": 3, "is_preview": true, "attachments": []*filesystem.File{sheet}})
	link := signedLink(t, c.securityFixture, "lessons", preview, "attachments")

	rec := do(t, c.handler, testRequest{url: link})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, "PREVIEW BYTES")

	setFields(t, c.securityFixture, c.course, map[string]any{"is_draft": true})
	assertStatus(t, do(t, c.handler, testRequest{url: link}), http.StatusNotFound)
	c.waitForDownloadLogs(t, 1)
}

func TestFreeCourseRequiringEnrolmentGatesAttachments(t *testing.T) {
	c := newCourseFixture(t)
	setFields(t, c.securityFixture, c.course, map[string]any{"access_tier": "free", "price": 0, "require_enrollment": true})
	link := signedLink(t, c.securityFixture, "lessons", c.lesson, "attachments")

	assertStatus(t, do(t, c.handler, testRequest{url: link}), http.StatusForbidden)
	assertStatus(t, do(t, c.handler, testRequest{url: link, headers: map[string]string{"Cookie": enrolmentCookie(t, c.securityFixture, "stranger@example.com")}}), http.StatusForbidden)

	setFields(t, c.securityFixture, c.course, map[string]any{"require_enrollment": false})
	rec := do(t, c.handler, testRequest{url: link})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, "LESSON WORKSHEET BYTES")
	c.waitForDownloadLogs(t, 1)
}
