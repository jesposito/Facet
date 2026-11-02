package hooks

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "facet/migrations"
	"facet/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// The security tests run against a database built by Facet's real migrations,
// so collection rules and schemas match a deployed instance. Migrating takes a
// few seconds, so it runs once and each test copies the result.
var migratedTemplate struct {
	once sync.Once
	dir  string
	err  error
}

// Importing facet/migrations registers them for every test app in this
// package. Other tests build their own collections on an empty database, so
// the migrations are set aside and only used to build the template.
var facetMigrations core.MigrationsList

func TestMain(m *testing.M) {
	facetMigrations = core.AppMigrations
	core.AppMigrations = core.MigrationsList{}

	code := m.Run()
	if migratedTemplate.dir != "" {
		os.RemoveAll(migratedTemplate.dir)
	}
	os.Exit(code)
}

func newMigratedTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	migratedTemplate.once.Do(func() {
		dir, err := os.MkdirTemp("", "facet-migrated-")
		if err != nil {
			migratedTemplate.err = err
			return
		}
		core.AppMigrations = facetMigrations
		app, err := tests.NewTestApp(dir)
		core.AppMigrations = core.MigrationsList{}
		if err != nil {
			migratedTemplate.err = err
			return
		}
		// NewTestApp copies dir and migrates the copy. Keep the migrated copy
		// as the template; closing the app flushes SQLite to disk.
		migratedTemplate.dir = app.DataDir()
		os.RemoveAll(dir)
		if err := app.ResetBootstrapState(); err != nil {
			migratedTemplate.err = err
		}
	})
	if migratedTemplate.err != nil {
		t.Fatalf("build migrated template: %v", migratedTemplate.err)
	}

	app, err := tests.NewTestApp(migratedTemplate.dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	return app
}

type securityFixture struct {
	app      *tests.TestApp
	pb       *pocketbase.PocketBase
	crypto   *services.CryptoService
	share    *services.ShareService
	purchase *services.PurchaseService
	rl       *services.RateLimitService
}

func newSecurityFixture(t *testing.T) *securityFixture {
	t.Helper()
	app := newMigratedTestApp(t)
	crypto := services.NewCryptoService(strings.Repeat("k", 32))
	return &securityFixture{
		app:      app,
		pb:       &pocketbase.PocketBase{App: app},
		crypto:   crypto,
		share:    services.NewShareService(crypto),
		purchase: services.NewPurchaseService(crypto),
		rl:       services.NewRateLimitService(),
	}
}

// serve runs the OnServe hooks the way PocketBase does at startup and returns
// the resulting router, so one test can make several requests against it.
func (f *securityFixture) serve(t *testing.T) http.Handler {
	t.Helper()
	router, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	var handler http.Handler
	event := &core.ServeEvent{App: f.app, Router: router}
	err = f.app.OnServe().Trigger(event, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		handler = mux
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

type testRequest struct {
	method  string
	url     string
	body    string
	headers map[string]string
	// remoteAddr defaults to a fresh address per request so Facet's per-IP
	// limiter doesn't interfere with tests about something else.
	remoteAddr string
}

var requestCounter atomic.Int64

func do(t *testing.T, h http.Handler, r testRequest) *httptest.ResponseRecorder {
	t.Helper()
	method := r.method
	if method == "" {
		method = http.MethodGet
	}
	var body io.Reader
	if r.body != "" {
		body = strings.NewReader(r.body)
	}
	req := httptest.NewRequest(method, r.url, body)
	req.RemoteAddr = r.remoteAddr
	if req.RemoteAddr == "" {
		n := requestCounter.Add(1)
		req.RemoteAddr = fmt.Sprintf("10.%d.%d.%d:4000", (n>>16)&255, (n>>8)&255, n&255)
	}
	if r.body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func (f *securityFixture) create(t *testing.T, collection string, fields map[string]any) *core.Record {
	t.Helper()
	c, err := f.app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(c)
	for k, v := range fields {
		record.Set(k, v)
	}
	if err := f.app.Save(record); err != nil {
		t.Fatalf("create %s: %v", collection, err)
	}
	return record
}

func newAuthRecord(c *core.Collection, email, password string) *core.Record {
	record := core.NewRecord(c)
	record.SetEmail(email)
	record.SetPassword(password)
	record.SetVerified(true)
	return record
}

func (f *securityFixture) createAdmin(t *testing.T, email, password string, fields map[string]any) (*core.Record, string) {
	t.Helper()
	c, err := f.app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	user := newAuthRecord(c, email, password)
	for k, v := range fields {
		user.Set(k, v)
	}
	if err := f.app.Save(user); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	token, err := user.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	return user, token
}

// buyerCookie returns the purchase cookie /api/content/access/{token} would
// set for these purchases.
func (f *securityFixture) buyerCookie(t *testing.T, email string, purchases ...*core.Record) string {
	t.Helper()
	ids := make([]string, 0, len(purchases))
	for _, p := range purchases {
		ids = append(ids, p.Id)
	}
	token, _, err := f.purchase.GeneratePurchaseJWT(ids, email)
	if err != nil {
		t.Fatal(err)
	}
	return services.PurchaseCookieName + "=" + token
}

func (f *securityFixture) completedPurchase(t *testing.T, email, contentType, contentID string) *core.Record {
	t.Helper()
	return f.create(t, "purchases", map[string]any{
		"buyer_email":                email,
		"content_type":               contentType,
		"content_id":                 contentID,
		"stripe_checkout_session_id": "cs_" + email + "_" + contentID,
		"amount":                     500,
		"currency":                   "usd",
		"status":                     "completed",
		"access_token":               "access-" + email + "-" + contentID,
	})
}

// waitForDownloadLogs waits for the download handler's background logging,
// which would otherwise write to the database after the test has removed it.
func (f *securityFixture) waitForDownloadLogs(t *testing.T, want int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if n, _ := f.app.CountRecords("download_logs"); int(n) >= want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("download was not logged")
}

func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, want, rec.Body.String())
	}
}

func assertContains(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	if !strings.Contains(rec.Body.String(), want) {
		t.Fatalf("response missing %q: %s", want, rec.Body.String())
	}
}

func assertNotContains(t *testing.T, rec *httptest.ResponseRecorder, unwanted string) {
	t.Helper()
	if strings.Contains(rec.Body.String(), unwanted) {
		t.Fatalf("response leaked %q: %s", unwanted, rec.Body.String())
	}
}
