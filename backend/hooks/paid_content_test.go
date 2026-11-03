package hooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

const paidPostBody = "Opening paragraph.\n\nSecond paragraph.\n\nThird paragraph.\n\nPAID ENDING"

func newPaidContentFixture(t *testing.T) (*securityFixture, map[string]*core.Record) {
	t.Helper()
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	RegisterDownloadHooks(f.pb, f.crypto, f.rl)

	guide, err := filesystem.NewFileFromBytes([]byte("PAID GUIDE BYTES"), "guide.pdf")
	if err != nil {
		t.Fatal(err)
	}
	records := map[string]*core.Record{
		"post": f.create(t, "posts", map[string]any{
			"title": "Paid post", "slug": "paid-post", "content": paidPostBody,
			"visibility": "public", "access_tier": "paid", "price": 500, "paywall_preview_paragraphs": 2,
			"published_at": "2026-01-01 00:00:00.000Z", "downloadable_files": []*filesystem.File{guide},
		}),
		"freePost": f.create(t, "posts", map[string]any{
			"title": "Free post", "slug": "free-post", "content": "FREE BODY", "visibility": "public",
			"published_at": "2026-01-02 00:00:00.000Z",
		}),
		"project": f.create(t, "projects", map[string]any{
			"title": "Paid project", "slug": "paid-project", "summary": "Project teaser",
			"description": "PAID PROJECT WRITEUP", "visibility": "public", "access_tier": "paid", "price": 700,
		}),
		"talk": f.create(t, "talks", map[string]any{
			"title": "Paid talk", "slug": "paid-talk", "event": "Conf", "date": "2026-02-01 00:00:00.000Z",
			"description": "PAID TALK ABSTRACT", "video_url": "https://video.example/paid", "slides_url": "https://slides.example/paid",
			"visibility": "public", "access_tier": "paid", "price": 900,
		}),
	}
	return f, records
}

func TestPaidContentIsGatedOnDetailEndpoints(t *testing.T) {
	f, r := newPaidContentFixture(t)
	buyer := f.completedPurchase(t, "buyer@example.com", "posts", r["post"].Id)
	projectBuyer := f.completedPurchase(t, "builder@example.com", "projects", r["project"].Id)
	talkBuyer := f.completedPurchase(t, "listener@example.com", "talks", r["talk"].Id)
	refunded := f.completedPurchase(t, "refunded@example.com", "posts", r["post"].Id)
	refunded.Set("status", "refunded")
	if err := f.app.Save(refunded); err != nil {
		t.Fatal(err)
	}
	otherContent := f.completedPurchase(t, "elsewhere@example.com", "projects", r["project"].Id)
	_, adminToken := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	h := f.serve(t)

	t.Run("anonymous visitor gets a preview of a paid post", func(t *testing.T) {
		rec := do(t, h, testRequest{url: "/api/post/paid-post"})
		assertStatus(t, rec, http.StatusOK)
		var post map[string]any
		json.Unmarshal(rec.Body.Bytes(), &post)
		if post["content"] != "Opening paragraph.\n\nSecond paragraph." || post["is_gated"] != true {
			t.Fatalf("unexpected gated post: %#v", post)
		}
		if post["price"] != float64(500) || post["access_tier"] != "paid" {
			t.Fatalf("purchase offer lost its price or tier: %#v", post)
		}
	})

	for name, cookie := range map[string]string{
		"refunded purchase":          f.buyerCookie(t, "refunded@example.com", refunded),
		"purchase of other content":  f.buyerCookie(t, "elsewhere@example.com", otherContent),
		"token naming another buyer": f.buyerCookie(t, "intruder@example.com", buyer),
		"tampered token":             f.buyerCookie(t, "buyer@example.com", buyer) + "x",
	} {
		t.Run(name+" does not unlock the post", func(t *testing.T) {
			rec := do(t, h, testRequest{url: "/api/post/paid-post", headers: map[string]string{"Cookie": cookie}})
			assertStatus(t, rec, http.StatusOK)
			assertNotContains(t, rec, "PAID ENDING")
		})
	}

	t.Run("buyer reads the whole post", func(t *testing.T) {
		rec := do(t, h, testRequest{url: "/api/post/paid-post", headers: map[string]string{"Cookie": f.buyerCookie(t, "buyer@example.com", buyer)}})
		assertStatus(t, rec, http.StatusOK)
		assertContains(t, rec, "PAID ENDING")
		assertContains(t, rec, `"is_purchased":true`)
	})

	t.Run("owner reads the whole post", func(t *testing.T) {
		rec := do(t, h, testRequest{url: "/api/post/paid-post", headers: map[string]string{"Authorization": adminToken}})
		assertContains(t, rec, "PAID ENDING")
	})

	t.Run("project and talk details", func(t *testing.T) {
		rec := do(t, h, testRequest{url: "/api/project/paid-project"})
		assertStatus(t, rec, http.StatusOK)
		assertContains(t, rec, "Project teaser")
		assertContains(t, rec, `"price":700`)
		assertContains(t, rec, `"access_tier":"paid"`)
		assertNotContains(t, rec, "PAID PROJECT WRITEUP")

		rec = do(t, h, testRequest{url: "/api/project/paid-project", headers: map[string]string{"Cookie": f.buyerCookie(t, "builder@example.com", projectBuyer)}})
		assertContains(t, rec, "PAID PROJECT WRITEUP")

		rec = do(t, h, testRequest{url: "/api/talk/paid-talk"})
		assertStatus(t, rec, http.StatusOK)
		for _, secret := range []string{"PAID TALK ABSTRACT", "video.example", "slides.example"} {
			assertNotContains(t, rec, secret)
		}

		rec = do(t, h, testRequest{url: "/api/talk/paid-talk", headers: map[string]string{"Cookie": f.buyerCookie(t, "listener@example.com", talkBuyer)}})
		assertContains(t, rec, "video.example")
	})
}

func TestPaidContentIsGatedOnListsFeedsAndViews(t *testing.T) {
	f, r := newPaidContentFixture(t)
	buyer := f.completedPurchase(t, "buyer@example.com", "posts", r["post"].Id)
	projectID := r["project"].Id
	sections := `[{"section":"projects","enabled":true,"items":["` + projectID + `"],"itemConfig":{"` + projectID + `":{"overrides":{"description":"OVERRIDE LEAK"}}}}]`
	f.create(t, "views", map[string]any{"name": "Public", "slug": "portfolio", "visibility": "public", "is_active": true, "sections": sections})
	h := f.serve(t)

	rec := do(t, h, testRequest{url: "/api/posts"})
	assertStatus(t, rec, http.StatusOK)
	assertNotContains(t, rec, "PAID ENDING")
	assertContains(t, rec, "FREE BODY")

	rec = do(t, h, testRequest{url: "/api/posts", headers: map[string]string{"Cookie": f.buyerCookie(t, "buyer@example.com", buyer)}})
	assertContains(t, rec, "PAID ENDING")

	rec = do(t, h, testRequest{url: "/api/homepage"})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, "Project teaser")
	assertContains(t, rec, "Opening paragraph.")
	for _, secret := range []string{"PAID ENDING", "PAID PROJECT WRITEUP", "PAID TALK ABSTRACT"} {
		assertNotContains(t, rec, secret)
	}

	rec = do(t, h, testRequest{url: "/api/talks"})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, "Paid talk")
	assertNotContains(t, rec, "video.example")

	rec = do(t, h, testRequest{url: "/talks.ics"})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, "Paid talk")
	assertNotContains(t, rec, "video.example")

	// A per-view override must not put a paid description back.
	rec = do(t, h, testRequest{url: "/api/view/portfolio/data"})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, "Project teaser")
	assertNotContains(t, rec, "PAID PROJECT WRITEUP")
	assertNotContains(t, rec, "OVERRIDE LEAK")
}

// libraryMedia uploads a file to the media library and attaches it the way
// the media picker does, through an external_media entry pointing at it.
func (f *securityFixture) libraryMedia(t *testing.T, name string) (*core.Record, string) {
	t.Helper()
	file, err := filesystem.NewFileFromBytes([]byte("LIBRARY BYTES"), name)
	if err != nil {
		t.Fatal(err)
	}
	upload := f.create(t, "uploads", map[string]any{"title": name, "file": []*filesystem.File{file}})
	url := "/api/files/" + upload.Collection().Id + "/" + upload.Id + "/" + upload.GetString("file")
	ref := f.create(t, "external_media", map[string]any{"title": name, "url": url, "provider": "upload"})
	return ref, url
}

func TestPaidMediaIsGatedWithTheContent(t *testing.T) {
	f, r := newPaidContentFixture(t)
	const cover = "https://cdn.example/public-cover.png"

	type item struct {
		path, collection string
		record           *core.Record
		refURL, fileName string
	}
	items := map[string]*item{
		"posts":    {path: "/api/post/paid-post", record: r["post"]},
		"projects": {path: "/api/project/paid-project", record: r["project"]},
		"talks":    {path: "/api/talk/paid-talk", record: r["talk"]},
	}
	for collection, it := range items {
		ref, url := f.libraryMedia(t, collection+"-gallery.png")
		it.collection, it.refURL = collection, url
		it.record.Set("media_refs", []string{ref.Id})
		it.record.Set("cover_image_library_url", cover)
		if collection == "projects" {
			shot, err := filesystem.NewFileFromBytes(append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...), "project-shot.png")
			if err != nil {
				t.Fatal(err)
			}
			it.record.Set("media", []*filesystem.File{shot})
		}
		if err := f.app.Save(it.record); err != nil {
			t.Fatal(err)
		}
		if collection == "projects" {
			it.fileName = it.record.GetStringSlice("media")[0]
		}
	}
	freeRef, freeURL := f.libraryMedia(t, "free-gallery.png")
	r["freePost"].Set("media_refs", []string{freeRef.Id})
	if err := f.app.Save(r["freePost"]); err != nil {
		t.Fatal(err)
	}

	cookies := map[string]map[string]string{}
	for collection, it := range items {
		buyer := f.completedPurchase(t, "buyer-"+collection+"@example.com", collection, it.record.Id)
		refunded := f.completedPurchase(t, "refunded-"+collection+"@example.com", collection, it.record.Id)
		refunded.Set("status", "refunded")
		if err := f.app.Save(refunded); err != nil {
			t.Fatal(err)
		}
		other := f.completedPurchase(t, "other-"+collection+"@example.com", "posts", r["freePost"].Id)
		cookies[collection] = map[string]string{
			"buyer":    f.buyerCookie(t, "buyer-"+collection+"@example.com", buyer),
			"refunded": f.buyerCookie(t, "refunded-"+collection+"@example.com", refunded),
			"other":    f.buyerCookie(t, "other-"+collection+"@example.com", other),
		}
	}
	_, adminToken := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	h := f.serve(t)

	assertMediaWithheld := func(t *testing.T, rec *httptest.ResponseRecorder, it *item) {
		t.Helper()
		assertStatus(t, rec, http.StatusOK)
		assertContains(t, rec, cover)
		assertContains(t, rec, `"is_gated":true`)
		for _, leak := range []string{it.refURL, `"media_refs"`, `"media_refs_expand"`, `"media_urls"`, `"media"`} {
			assertNotContains(t, rec, leak)
		}
		if it.fileName != "" {
			assertNotContains(t, rec, it.fileName)
		}
	}
	assertMediaShown := func(t *testing.T, rec *httptest.ResponseRecorder, it *item) {
		t.Helper()
		assertStatus(t, rec, http.StatusOK)
		assertContains(t, rec, cover)
		assertContains(t, rec, it.refURL)
		if it.fileName != "" {
			assertContains(t, rec, `"media_urls"`)
			assertContains(t, rec, it.fileName)
		}
	}

	for collection, it := range items {
		t.Run(collection+" detail", func(t *testing.T) {
			assertMediaWithheld(t, do(t, h, testRequest{url: it.path}), it)
			for _, who := range []string{"refunded", "other"} {
				rec := do(t, h, testRequest{url: it.path, headers: map[string]string{"Cookie": cookies[collection][who]}})
				assertMediaWithheld(t, rec, it)
			}
			rec := do(t, h, testRequest{url: it.path, headers: map[string]string{"Cookie": cookies[collection]["buyer"]}})
			assertMediaShown(t, rec, it)
			assertMediaShown(t, do(t, h, testRequest{url: it.path, headers: map[string]string{"Authorization": adminToken}}), it)
		})
	}

	t.Run("free post keeps its media", func(t *testing.T) {
		rec := do(t, h, testRequest{url: "/api/post/free-post"})
		assertStatus(t, rec, http.StatusOK)
		assertContains(t, rec, freeURL)
		assertContains(t, rec, `"media_refs_expand"`)
	})

	t.Run("view data", func(t *testing.T) {
		var sections []string
		for _, collection := range []string{"posts", "projects", "talks"} {
			sections = append(sections, `{"section":"`+collection+`","enabled":true,"items":["`+items[collection].record.Id+`"]}`)
		}
		f.create(t, "views", map[string]any{"name": "Paid", "slug": "paid-media", "visibility": "public", "is_active": true, "sections": "[" + strings.Join(sections, ",") + "]"})
		rec := do(t, h, testRequest{url: "/api/view/paid-media/data"})
		assertStatus(t, rec, http.StatusOK)
		assertContains(t, rec, "Paid talk")
		for _, leak := range []string{`"media_refs"`, `"media_urls"`, `"media"`, items["projects"].fileName} {
			assertNotContains(t, rec, leak)
		}
		for _, it := range items {
			assertNotContains(t, rec, it.record.GetStringSlice("media_refs")[0])
		}
	})
}

func TestPaidDownloadsNeedAPurchase(t *testing.T) {
	f, r := newPaidContentFixture(t)
	post := r["post"]
	buyer := f.completedPurchase(t, "buyer@example.com", "posts", post.Id)
	filename := post.GetStringSlice("downloadable_files")[0]
	h := f.serve(t)

	direct := "/api/files/" + post.Collection().Id + "/" + post.Id + "/" + filename
	rec := do(t, h, testRequest{url: direct})
	if rec.Code == http.StatusOK || strings.Contains(rec.Body.String(), "PAID GUIDE BYTES") {
		t.Fatalf("protected file served without a token: %d", rec.Code)
	}

	urls := "/api/content/posts/" + post.Id + "/download-urls"
	assertStatus(t, do(t, h, testRequest{url: urls}), http.StatusForbidden)

	cookie := map[string]string{"Cookie": f.buyerCookie(t, "buyer@example.com", buyer)}
	rec = do(t, h, testRequest{url: urls, headers: cookie})
	assertStatus(t, rec, http.StatusOK)
	var body struct {
		Files []struct {
			URL string `json:"url"`
		} `json:"files"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || len(body.Files) != 1 {
		t.Fatalf("unexpected download-urls response: %s", rec.Body.String())
	}

	rec = do(t, h, testRequest{url: body.Files[0].URL, headers: cookie})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, "PAID GUIDE BYTES")

	assertStatus(t, do(t, h, testRequest{url: body.Files[0].URL + "tampered", headers: cookie}), http.StatusForbidden)
	f.waitForDownloadLogs(t, 1)
}

func TestPaywallPreviewAlwaysHoldsSomethingBack(t *testing.T) {
	for _, tc := range []struct {
		content string
		n       int
		want    string
	}{
		{"a\n\nb\n\nc\n\nd", 2, "a\n\nb"},
		{"a\n\nb", 3, "a"},
		{"only one paragraph", 3, ""},
		{"a\r\n\r\nb\r\n\r\nc", 5, "a\n\nb"},
	} {
		if got := paywallPreview(tc.content, tc.n); got != tc.want {
			t.Errorf("paywallPreview(%q, %d) = %q, want %q", tc.content, tc.n, got, tc.want)
		}
	}
}
