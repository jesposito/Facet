package hooks

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"facet/services"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

const (
	visitorFileBytes = "VISITOR FILE BYTES"
	visitorComment   = `{"author_name":"Ada","body":"Hello from the Facet"}`
	badFormat        = `{"format":"exe"}`
)

// visitorContext is a signed-out browser that has opened a single-use share
// link once. From then on it holds two HttpOnly cookies and sends nothing else.
type visitorContext struct {
	f       *securityFixture
	h       http.Handler
	view    *core.Record
	post    *core.Record
	token   *core.Record
	section string
	proof   string
	content string
}

func newVisitorContext(t *testing.T) *visitorContext {
	t.Helper()
	f := newSecurityFixture(t)
	RegisterViewHooks(f.pb, f.crypto, f.share, f.rl)
	RegisterShareHooks(f.pb, f.share, f.crypto, f.rl)
	RegisterDownloadHooks(f.pb, f.crypto, f.rl)
	RegisterCommentHooks(f.pb, &services.PlanConfig{Features: services.PlanFeatures{Discussions: true}}, f.crypto, f.share, f.rl)
	RegisterResumeHooks(f.pb, f.crypto, f.share)

	view := f.create(t, "views", map[string]any{"name": "Pitch", "slug": "pitch", "visibility": "unlisted", "is_active": true})
	file, err := filesystem.NewFileFromBytes([]byte(visitorFileBytes), "guide.pdf")
	if err != nil {
		t.Fatal(err)
	}
	post := f.create(t, "posts", map[string]any{
		"title": "Private post", "slug": "private-post", "content": "PRIVATE POST BODY", "visibility": "private",
		"comments_enabled": true, "view_visibility": map[string]bool{view.Id: true}, "downloadable_files": []*filesystem.File{file},
	})
	section := `{"section":"posts","enabled":true,"items":` + idList(t, post.Id) + `}`
	setViewSection(t, f, view, section)
	token := f.create(t, "share_tokens", map[string]any{"view_id": view.Id, "token_hash": f.share.HMACToken("one-use"), "is_active": true, "max_uses": 1})
	h := f.serve(t)

	// What /s/<token> does: validate once and keep the proof it answers with.
	rec := do(t, h, testRequest{method: http.MethodPost, url: "/api/share/validate", body: `{"token":"one-use"}`})
	assertStatus(t, rec, http.StatusOK)
	proof := rec.Header().Get(shareVisitHeader)
	if proof == "" {
		t.Fatal("entering the share link gave no visit proof")
	}
	return &visitorContext{f: f, h: h, view: view, post: post, token: token, section: section, proof: proof, content: "posts/" + post.Id}
}

func (v *visitorContext) browser() map[string]string {
	return cookieHeader("me_share_token=one-use; me_share_visit=" + v.proof)
}

func cookieHeader(cookie string) map[string]string {
	return map[string]string{"Cookie": cookie}
}

func (v *visitorContext) threadURL() string {
	return "/api/comments/" + v.content + "?from=pitch"
}

func (v *visitorContext) filesURL() string {
	return "/api/content/" + v.content + "/download-urls?from=pitch"
}

func (v *visitorContext) issueLink(t *testing.T, headers map[string]string) string {
	t.Helper()
	rec := do(t, v.h, testRequest{url: v.filesURL(), headers: headers})
	assertStatus(t, rec, http.StatusOK)
	var body struct {
		Files []struct {
			URL string `json:"url"`
		} `json:"files"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || len(body.Files) != 1 {
		t.Fatalf("unexpected download-urls response: %s", rec.Body.String())
	}
	return body.Files[0].URL
}

func (v *visitorContext) count(t *testing.T, collection string) int64 {
	t.Helper()
	n, err := v.f.app.CountRecords(collection)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func (v *visitorContext) useCount(t *testing.T) int {
	t.Helper()
	token, err := v.f.app.FindRecordById("share_tokens", v.token.Id)
	if err != nil {
		t.Fatal(err)
	}
	return token.GetInt("use_count")
}

// assertClosed checks the thread and file routes answer as if the post didn't
// exist, and that a captured link serves nothing.
func (v *visitorContext) assertClosed(t *testing.T, headers map[string]string, link string) {
	t.Helper()
	comments := v.count(t, "comments")
	for _, r := range []testRequest{
		{url: v.threadURL(), headers: headers},
		{method: http.MethodPost, url: v.threadURL(), body: visitorComment, headers: headers},
		{url: v.filesURL(), headers: headers},
		{url: link, headers: headers},
	} {
		rec := do(t, v.h, r)
		assertStatus(t, rec, http.StatusNotFound)
		assertContains(t, rec, "content not found")
		assertNotContains(t, rec, visitorFileBytes)
	}
	if got := v.count(t, "comments"); got != comments {
		t.Fatalf("a refused visitor added a comment: %d, was %d", got, comments)
	}
}

func TestSignedOutBrowserKeepsASingleUseVisitWithCookiesAlone(t *testing.T) {
	v := newVisitorContext(t)
	browser := v.browser()

	assertStatus(t, do(t, v.h, testRequest{url: "/api/post/private-post?from=pitch", headers: browser}), http.StatusOK)
	assertStatus(t, do(t, v.h, testRequest{url: v.threadURL(), headers: browser}), http.StatusOK)

	rec := do(t, v.h, testRequest{method: http.MethodPost, url: v.threadURL(), body: visitorComment, headers: browser})
	assertStatus(t, rec, http.StatusCreated)
	var comment struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &comment); err != nil {
		t.Fatal(err)
	}
	report := func(t *testing.T, headers map[string]string) {
		t.Helper()
		url := "/api/comments/" + comment.ID + "/report?from=pitch"
		rec := do(t, v.h, testRequest{method: http.MethodPost, url: url, body: `{"reason":"spam"}`, headers: headers})
		assertStatus(t, rec, http.StatusOK)
		assertContains(t, rec, `"status":"received"`)
	}
	report(t, browser)
	if got := v.count(t, "comment_reports"); got != 1 {
		t.Fatalf("the visitor's report wasn't saved: %d", got)
	}

	link := v.issueLink(t, browser)
	rec = do(t, v.h, testRequest{url: link, headers: browser})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, visitorFileBytes)

	// Generation clears access, then stops at the format check before any AI call.
	assertStatus(t, generate(t, v.h, "pitch", browser, badFormat), http.StatusBadRequest)

	// The token has no uses left, so neither half of the visit works alone.
	for name, cookie := range map[string]string{
		"spent token alone": "me_share_token=one-use",
		"proof alone":       "me_share_visit=" + v.proof,
	} {
		t.Run(name, func(t *testing.T) {
			headers := cookieHeader(cookie)
			v.assertClosed(t, headers, link)
			assertStatus(t, generate(t, v.h, "pitch", headers, badFormat), http.StatusNotFound)
			report(t, headers)
		})
	}
	if got := v.count(t, "comment_reports"); got != 1 {
		t.Fatalf("a refused visitor filed a report: %d", got)
	}
	if got := v.useCount(t); got != 1 {
		t.Fatalf("the visit changed use_count to %d, want 1", got)
	}
	v.f.waitForDownloadLogs(t, 1)
}

func TestFacetFilesAndThreadCloseWithTheFacet(t *testing.T) {
	v := newVisitorContext(t)
	browser := v.browser()
	link := v.issueLink(t, browser)
	served := 0
	assertOpen := func(t *testing.T) {
		t.Helper()
		rec := do(t, v.h, testRequest{url: link, headers: browser})
		assertStatus(t, rec, http.StatusOK)
		assertContains(t, rec, visitorFileBytes)
		served++
		assertStatus(t, do(t, v.h, testRequest{url: v.threadURL(), headers: browser}), http.StatusOK)
	}
	section := func(raw string) func(*testing.T) {
		return func(t *testing.T) { setViewSection(t, v.f, v.view, raw) }
	}
	set := func(record func() *core.Record, fields map[string]any) func(*testing.T) {
		return func(t *testing.T) { setFields(t, v.f, record(), fields) }
	}
	view := func() *core.Record { return v.view }
	post := func() *core.Record { return v.post }
	// Read fresh: the stored use_count has moved on from the fixture's copy.
	token := func() *core.Record {
		record, err := v.f.app.FindRecordById("share_tokens", v.token.Id)
		if err != nil {
			t.Fatal(err)
		}
		return record
	}
	items := `"items":` + idList(t, v.post.Id)
	granted := map[string]bool{v.view.Id: true}

	assertOpen(t)
	for _, tc := range []struct {
		name         string
		change, undo func(*testing.T)
	}{
		{"deselected", section(`{"section":"posts","enabled":true,"items":[]}`), section(v.section)},
		{"section disabled", section(`{"section":"posts","enabled":false,` + items + `}`), section(v.section)},
		{"switched to all_public", section(`{"section":"posts","enabled":true,"itemSelection":"all_public",` + items + `}`), section(v.section)},
		{"Facet inactive", set(view, map[string]any{"is_active": false}), set(view, map[string]any{"is_active": true})},
		{"override removed", set(post, map[string]any{"view_visibility": map[string]bool{}}), set(post, map[string]any{"view_visibility": granted})},
		{"draft", set(post, map[string]any{"is_draft": true}), set(post, map[string]any{"is_draft": false})},
		{"token revoked", set(token, map[string]any{"is_active": false}), set(token, map[string]any{"is_active": true})},
		{"token expired", set(token, map[string]any{"expires_at": "2020-01-01 00:00:00.000Z"}), set(token, map[string]any{"expires_at": ""})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.change(t)
			v.assertClosed(t, browser, link)
			tc.undo(t)
			assertOpen(t)
		})
	}
	v.f.waitForDownloadLogs(t, served)
}

func TestFacetClaimIsSignedAndNeverReadFromTheRequest(t *testing.T) {
	v := newVisitorContext(t)
	browser := v.browser()
	downloads := services.NewDownloadService(v.f.crypto)
	filename := v.post.GetStringSlice("downloadable_files")[0]
	redeem := func(link string, headers map[string]string) int {
		t.Helper()
		return do(t, v.h, testRequest{url: link, headers: headers}).Code
	}

	// An old link has no Facet, and `from` on the URL doesn't lend it one.
	plain, _ := downloads.GenerateSignedURL("posts", v.post.Id, filename)
	if got := redeem("/api/downloads/"+plain+"?from=pitch", browser); got != http.StatusNotFound {
		t.Fatalf("plain link on a private post answered %d", got)
	}

	// A Facet that doesn't select the post opens nothing, whatever the claim says.
	other := v.f.create(t, "views", map[string]any{"name": "Other", "slug": "other", "visibility": "public", "is_active": true})
	claimed, _ := downloads.GenerateFacetSignedURL("posts", v.post.Id, filename, other.Id)
	if got := redeem("/api/downloads/"+claimed, browser); got != http.StatusNotFound {
		t.Fatalf("link for an unrelated Facet answered %d", got)
	}

	// Cutting the claim off a real link breaks its signature.
	link := v.issueLink(t, browser)
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(link, "/api/downloads/"))
	if err != nil {
		t.Fatal(err)
	}
	_, rest, _ := strings.Cut(string(raw), ":")
	stripped := base64.RawURLEncoding.EncodeToString([]byte(rest))
	if got := redeem("/api/downloads/"+stripped, browser); got != http.StatusForbidden {
		t.Fatalf("link without its claim answered %d", got)
	}

	// A private paid file still needs its purchase on top of the Facet.
	setFields(t, v.f, v.post, map[string]any{"access_tier": "paid", "price": 500})
	assertStatus(t, do(t, v.h, testRequest{url: v.filesURL(), headers: browser}), http.StatusForbidden)
	if got := redeem(link, browser); got != http.StatusForbidden {
		t.Fatalf("unpaid link answered %d", got)
	}
	buyer := v.f.completedPurchase(t, "buyer@example.com", "posts", v.post.Id)
	paid := cookieHeader(browser["Cookie"] + "; " + v.f.buyerCookie(t, "buyer@example.com", buyer))
	if got := redeem(link, paid); got != http.StatusOK {
		t.Fatalf("paid link answered %d", got)
	}
	v.f.waitForDownloadLogs(t, 1)
}

func TestVisitorCookiesStayOnTheirOwnOrigin(t *testing.T) {
	v := newVisitorContext(t)
	with := func(name, value string) map[string]string {
		headers := v.browser()
		headers[name] = value
		return headers
	}

	for name, headers := range map[string]map[string]string{
		"another site":      with("Sec-Fetch-Site", "cross-site"),
		"sibling subdomain": with("Sec-Fetch-Site", "same-site"),
		"foreign Origin":    with("Origin", "https://evil.example"),
	} {
		t.Run(name, func(t *testing.T) {
			rec := do(t, v.h, testRequest{method: http.MethodPost, url: v.threadURL(), body: visitorComment, headers: headers})
			assertStatus(t, rec, http.StatusNotFound)
			assertStatus(t, generate(t, v.h, "pitch", headers, badFormat), http.StatusNotFound)
		})
	}

	// The same calls from this origin pass access and stop at validation.
	for name, headers := range map[string]map[string]string{
		"same origin":     with("Sec-Fetch-Site", "same-origin"),
		"matching Origin": with("Origin", "http://example.com"),
	} {
		t.Run(name, func(t *testing.T) {
			rec := do(t, v.h, testRequest{method: http.MethodPost, url: v.threadURL(), body: `{}`, headers: headers})
			assertStatus(t, rec, http.StatusBadRequest)
			assertStatus(t, generate(t, v.h, "pitch", headers, badFormat), http.StatusBadRequest)
		})
	}
}

func TestCookiesAloneNeverSpendAShareUse(t *testing.T) {
	v := newVisitorContext(t)
	token := v.f.create(t, "share_tokens", map[string]any{"view_id": v.view.Id, "token_hash": v.f.share.HMACToken("many-uses"), "is_active": true, "max_uses": 5})
	uses := func() int {
		record, err := v.f.app.FindRecordById("share_tokens", token.Id)
		if err != nil {
			t.Fatal(err)
		}
		return record.GetInt("use_count")
	}
	cookie := cookieHeader("me_share_token=many-uses")

	assertStatus(t, do(t, v.h, testRequest{url: v.threadURL(), headers: cookie}), http.StatusOK)
	assertStatus(t, do(t, v.h, testRequest{url: "/api/view/pitch/data", headers: cookie}), http.StatusNotFound)
	if got := uses(); got != 0 {
		t.Fatalf("a cookie spent %d uses", got)
	}

	// A token presented on purpose still counts, as the frontend's server does.
	header := map[string]string{"X-Share-Token": "many-uses"}
	assertStatus(t, do(t, v.h, testRequest{url: "/api/view/pitch/data", headers: header}), http.StatusOK)
	if got := uses(); got != 1 {
		t.Fatalf("use_count = %d, want 1", got)
	}
}

func TestHeadersOutrankVisitorCookies(t *testing.T) {
	v := newVisitorContext(t)
	forged := v.browser()
	forged["X-Share-Token"] = "forged"
	assertStatus(t, do(t, v.h, testRequest{url: v.threadURL(), headers: forged}), http.StatusNotFound)

	oversized := cookieHeader("me_share_token=one-use; me_share_visit=" + v.proof + strings.Repeat("x", visitorCookieMaxLen))
	assertStatus(t, do(t, v.h, testRequest{url: v.threadURL(), headers: oversized}), http.StatusNotFound)
}

func TestPasswordCookieOpensOnlyItsOwnFacet(t *testing.T) {
	f := newSecurityFixture(t)
	RegisterResumeHooks(f.pb, f.crypto, f.share)
	view := func(slug, visibility string) *core.Record {
		return f.create(t, "views", map[string]any{"name": slug, "slug": slug, "visibility": visibility, "is_active": true})
	}
	locked, other := view("locked", "password"), view("other", "password")
	view("hidden", "private")
	_, adminToken := f.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	h := f.serve(t)
	jwtFor := func(viewID string, duration time.Duration) string {
		t.Helper()
		token, _, err := f.crypto.GenerateViewAccessJWT(viewID, duration)
		if err != nil {
			t.Fatal(err)
		}
		return token
	}

	for _, tc := range []struct {
		name   string
		slug   string
		cookie string
		want   int
	}{
		{"no cookie", "locked", "", http.StatusUnauthorized},
		{"its own token", "locked", "me_password_token=" + jwtFor(locked.Id, time.Hour), http.StatusBadRequest},
		{"another Facet's token", "locked", "me_password_token=" + jwtFor(other.Id, time.Hour), http.StatusUnauthorized},
		{"expired token", "locked", "me_password_token=" + jwtFor(locked.Id, -time.Minute), http.StatusUnauthorized},
		{"owner session as a password token", "locked", "me_password_token=" + adminToken, http.StatusUnauthorized},
		{"owner session cookie", "hidden", "pb_auth=" + adminToken, http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertStatus(t, generate(t, h, tc.slug, cookieHeader(tc.cookie), badFormat), tc.want)
		})
	}
}

// newCustomVisit is a visit to a Facet showing one private custom record that
// has no override yet. The thread and file routes now act on that record.
func newCustomVisit(t *testing.T) (*visitorContext, *core.Record) {
	t.Helper()
	v := newVisitorContext(t)
	file, err := filesystem.NewFileFromBytes([]byte(visitorFileBytes), "guide.pdf")
	if err != nil {
		t.Fatal(err)
	}
	custom := v.f.create(t, "custom_content", map[string]any{
		"title": "Private notes", "content": "PRIVATE NOTES", "visibility": "private",
		"comments_enabled": true, "downloadable_files": []*filesystem.File{file},
	})
	v.content = "custom_content/" + custom.Id
	v.section = `{"section":"custom:` + custom.Id + `","enabled":true,"items":[]}`
	setViewSection(t, v.f, v.view, v.section)
	return v, custom
}

func TestManualOverrideOpensACustomSectionWhileItsFacetShowsIt(t *testing.T) {
	v, custom := newCustomVisit(t)
	granted := map[string]bool{v.view.Id: true}
	setFields(t, v.f, custom, map[string]any{"view_visibility": granted})
	browser := v.browser()
	link := v.issueLink(t, browser)
	served := 0
	assertOpen := func(t *testing.T) {
		t.Helper()
		rec := do(t, v.h, testRequest{url: link, headers: browser})
		assertStatus(t, rec, http.StatusOK)
		assertContains(t, rec, visitorFileBytes)
		served++
		assertStatus(t, do(t, v.h, testRequest{url: v.threadURL(), headers: browser}), http.StatusOK)
	}
	section := func(raw string) func(*testing.T) {
		return func(t *testing.T) { setViewSection(t, v.f, v.view, raw) }
	}
	set := func(record func() *core.Record, fields map[string]any) func(*testing.T) {
		return func(t *testing.T) { setFields(t, v.f, record(), fields) }
	}
	view := func() *core.Record { return v.view }
	item := func() *core.Record { return custom }
	// Read fresh: the stored use_count has moved on from the fixture's copy.
	token := func() *core.Record {
		record, err := v.f.app.FindRecordById("share_tokens", v.token.Id)
		if err != nil {
			t.Fatal(err)
		}
		return record
	}
	restore := section(v.section)
	restoreGrant := set(item, map[string]any{"view_visibility": granted})
	items := `"items":` + idList(t, custom.Id)

	assertOpen(t)
	rec := do(t, v.h, testRequest{method: http.MethodPost, url: v.threadURL(), body: visitorComment, headers: browser})
	assertStatus(t, rec, http.StatusCreated)

	// The override only counts through the Facet, for someone on a visit to it.
	bare := "/api/content/" + v.content + "/download-urls"
	assertStatus(t, do(t, v.h, testRequest{url: bare, headers: browser}), http.StatusNotFound)
	v.assertClosed(t, nil, link)
	v.assertClosed(t, cookieHeader("me_share_token=one-use"), link)

	for _, tc := range []struct {
		name         string
		change, undo func(*testing.T)
	}{
		{"section disabled", section(`{"section":"custom:` + custom.Id + `","enabled":false,"items":[]}`), restore},
		{"section removed", section(`{"section":"posts","enabled":true,"items":[]}`), restore},
		{"picked in another custom section", section(`{"section":"custom:missing00000000","enabled":true,` + items + `}`), restore},
		{"picked under the collection name", section(`{"section":"custom_content","enabled":true,` + items + `}`), restore},
		{"override removed", set(item, map[string]any{"view_visibility": map[string]bool{}}), restoreGrant},
		{"override turned off", set(item, map[string]any{"view_visibility": map[string]bool{v.view.Id: false}}), restoreGrant},
		{"override on another Facet", set(item, map[string]any{"view_visibility": map[string]bool{"elsewhere": true}}), restoreGrant},
		{"draft", set(item, map[string]any{"is_draft": true}), set(item, map[string]any{"is_draft": false})},
		{"Facet inactive", set(view, map[string]any{"is_active": false}), set(view, map[string]any{"is_active": true})},
		{"token revoked", set(token, map[string]any{"is_active": false}), set(token, map[string]any{"is_active": true})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.change(t)
			v.assertClosed(t, browser, link)
			tc.undo(t)
			assertOpen(t)
		})
	}
	v.f.waitForDownloadLogs(t, served)
}

func TestGrantEndpointLeavesPrivateCustomSectionsClosed(t *testing.T) {
	v, custom := newCustomVisit(t)
	_, adminToken := v.f.createAdmin(t, "owner@example.com", "a-strong-password", nil)

	assertGranted(t, v.h, v.view.Id, adminToken, 0)
	if hasOverride(t, v.f, "custom_content", custom.Id, v.view.Id) {
		t.Fatal("custom section was granted")
	}

	// A signed claim for this Facet doesn't stand in for the override.
	filename := custom.GetStringSlice("downloadable_files")[0]
	claimed, _ := services.NewDownloadService(v.f.crypto).GenerateFacetSignedURL("custom_content", custom.Id, filename, v.view.Id)
	v.assertClosed(t, v.browser(), "/api/downloads/"+claimed)
}

func TestDetailResponsesSayWhetherFilesExistWithoutNamingThem(t *testing.T) {
	v := newVisitorContext(t)
	browser := v.browser()
	file, err := filesystem.NewFileFromBytes([]byte(visitorFileBytes), "guide.pdf")
	if err != nil {
		t.Fatal(err)
	}
	v.f.create(t, "projects", map[string]any{"title": "Files", "slug": "with-files", "visibility": "public", "downloadable_files": []*filesystem.File{file}})
	v.f.create(t, "projects", map[string]any{"title": "Plain", "slug": "plain-project", "visibility": "public"})
	v.f.create(t, "posts", map[string]any{"title": "Plain", "slug": "plain-post", "content": "x", "visibility": "public"})

	for _, tc := range []struct {
		url     string
		headers map[string]string
		want    string
	}{
		{"/api/post/private-post?from=pitch", browser, `"has_downloads":true`},
		{"/api/project/with-files", nil, `"has_downloads":true`},
		{"/api/post/plain-post", nil, `"has_downloads":false`},
		{"/api/project/plain-project", nil, `"has_downloads":false`},
	} {
		t.Run(tc.url, func(t *testing.T) {
			rec := do(t, v.h, testRequest{url: tc.url, headers: tc.headers})
			assertStatus(t, rec, http.StatusOK)
			assertContains(t, rec, tc.want)
			assertNotContains(t, rec, "guide")
		})
	}

	// A locked page doesn't learn about the files until they're bought.
	setFields(t, v.f, v.post, map[string]any{"access_tier": "paid", "price": 500})
	rec := do(t, v.h, testRequest{url: "/api/post/private-post?from=pitch", headers: browser})
	assertStatus(t, rec, http.StatusOK)
	assertNotContains(t, rec, "has_downloads")

	buyer := v.f.completedPurchase(t, "buyer@example.com", "posts", v.post.Id)
	paid := cookieHeader(browser["Cookie"] + "; " + v.f.buyerCookie(t, "buyer@example.com", buyer))
	rec = do(t, v.h, testRequest{url: "/api/post/private-post?from=pitch", headers: paid})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"has_downloads":true`)
}

func TestOtherAuthCollectionDoesNotGrantOwnerFacetAccess(t *testing.T) {
	v := newVisitorContext(t)
	visitors := core.NewAuthCollection("visitors")
	if err := v.f.app.Save(visitors); err != nil {
		t.Fatal(err)
	}
	visitor := newAuthRecord(visitors, "visitor@example.test", "a-strong-password")
	if err := v.f.app.Save(visitor); err != nil {
		t.Fatal(err)
	}
	token, err := visitor.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	headers := map[string]string{"Authorization": token}
	v.f.create(t, "views", map[string]any{"name": "Owner notes", "slug": "owner-notes", "visibility": "private", "is_active": true})
	for _, path := range []string{
		"/api/view/pitch/access", "/api/view/pitch/data", "/api/post/private-post",
		"/api/post/private-post?from=pitch", "/api/view/owner-notes/access", "/api/view/owner-notes/data",
	} {
		t.Run(path, func(t *testing.T) {
			assertStatus(t, do(t, v.h, testRequest{url: path, headers: headers}), http.StatusNotFound)
		})
	}
	assertStatus(t, do(t, v.h, testRequest{url: "/api/post/private-post?from=pitch", headers: map[string]string{"Authorization": token, "Cookie": v.browser()["Cookie"]}}), http.StatusOK)
}
