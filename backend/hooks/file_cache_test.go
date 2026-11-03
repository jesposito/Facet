package hooks

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestProtectedFilesAreNeverCached(t *testing.T) {
	f, r := newPaidContentFixture(t)
	RegisterFileCacheHeaders(f.app)
	post := r["post"]

	var cover bytes.Buffer
	if err := png.Encode(&cover, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	coverFile, err := filesystem.NewFileFromBytes(cover.Bytes(), "cover.png")
	if err != nil {
		t.Fatal(err)
	}
	post.Set("cover_image", coverFile)
	if err := f.app.Save(post); err != nil {
		t.Fatal(err)
	}

	superusers, err := f.app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		t.Fatal(err)
	}
	owner := newAuthRecord(superusers, "root@example.com", "a-strong-password")
	if err := f.app.Save(owner); err != nil {
		t.Fatal(err)
	}
	fileToken, err := owner.NewFileToken()
	if err != nil {
		t.Fatal(err)
	}
	h := f.serve(t)
	base := "/api/files/" + post.Collection().Id + "/" + post.Id + "/"

	t.Run("protected file with a valid token", func(t *testing.T) {
		rec := do(t, h, testRequest{url: base + post.GetStringSlice("downloadable_files")[0] + "?token=" + fileToken})
		assertStatus(t, rec, http.StatusOK)
		assertContains(t, rec, "PAID GUIDE BYTES")
		if got := rec.Header().Get("Cache-Control"); got != protectedFileCacheControl {
			t.Fatalf("protected file Cache-Control = %q", got)
		}
	})

	t.Run("public cover image", func(t *testing.T) {
		rec := do(t, h, testRequest{url: base + post.GetString("cover_image")})
		assertStatus(t, rec, http.StatusOK)
		if got := rec.Header().Get("Cache-Control"); got != publicFileCacheControl {
			t.Fatalf("public file Cache-Control = %q", got)
		}
	})

	t.Run("signed buyer download", func(t *testing.T) {
		buyer := f.completedPurchase(t, "buyer@example.com", "posts", post.Id)
		rec := do(t, h, testRequest{url: "/api/content/posts/" + post.Id + "/download-urls", headers: map[string]string{"Cookie": f.buyerCookie(t, "buyer@example.com", buyer)}})
		assertStatus(t, rec, http.StatusOK)
		start := strings.Index(rec.Body.String(), "/api/downloads/")
		if start < 0 {
			t.Fatalf("no signed link in %s", rec.Body.String())
		}
		link := rec.Body.String()[start:]
		link = link[:strings.IndexByte(link, '"')]
		rec = do(t, h, testRequest{url: link})
		assertStatus(t, rec, http.StatusOK)
		if got := rec.Header().Get("Cache-Control"); got != protectedFileCacheControl {
			t.Fatalf("signed download Cache-Control = %q", got)
		}
		f.waitForDownloadLogs(t, 1)
	})
}
