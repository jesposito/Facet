package hooks

import (
	"net/http"
	"testing"

	"facet/services"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stripe/stripe-go/v82"
)

func TestCourseSalesStayOffWhateverTheEnvironment(t *testing.T) {
	if services.CourseSalesAvailable {
		t.Fatal("services.CourseSalesAvailable must be false in the self-hosted build")
	}
	for _, env := range []map[string]string{
		{},
		{"FACET_MANAGED": "false"},
		{"FACET_MANAGED": "true"},
		{"FACET_MANAGED": "true", "FACET_FEATURE_COURSES": "true"},
		{"FACET_MANAGED": "true", "FACET_FEATURE_COURSES": "false"},
		{"FACET_MANAGED": "true", "FACET_FEATURE_COURSE_SALES": "true", "FACET_FEATURE_PRICING": "true"},
		{"FACET_MANAGED": "1", "FACET_FEATURE_COURSE_SALES": "1"},
		{"FACET_FEATURE_COURSE_SALES": "true"},
	} {
		for _, k := range []string{"FACET_MANAGED", "FACET_FEATURE_COURSES", "FACET_FEATURE_COURSE_SALES", "FACET_FEATURE_PRICING"} {
			t.Setenv(k, env[k])
		}
		plan := services.LoadPlanConfig()
		if plan.Features.CourseSales || plan.HasFeature("course_sales") {
			t.Errorf("course_sales on with %v", env)
		}
	}
	// A hand-built plan that claims course sales still can't turn them on:
	// HasFeature answers from the build constant, not the struct.
	forged := &services.PlanConfig{Features: services.PlanFeatures{Courses: true, CourseSales: true}}
	if forged.HasFeature("course_sales") {
		t.Fatal("a plan struct claiming course_sales turned it on")
	}
}

// A course checkout session Stripe already took money for (issued before
// this build, when course sales were possible) is still honoured: the
// buyer gets the purchase and access email they paid for. Nothing about
// that reopens checkout, and the course keeps its tier and price.
func TestAlreadyPaidCourseSessionIsHonouredWithoutReopeningSales(t *testing.T) {
	d := newDeliveryFixture(t, true)
	course := d.create(t, "courses", map[string]any{"title": "Legacy", "slug": "legacy", "access_tier": "paid", "price": 4900, "visibility": "public"})
	sess := d.session("cs_test_legacy_course", "earlier@example.com", "complete", stripe.CheckoutSessionPaymentStatusPaid,
		map[string]string{"content_type": "courses", "content_id": course.Id})

	rec := d.webhook(t, "checkout.session.completed", sess)
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"status":"success"`)
	purchase, err := d.app.FindFirstRecordByFilter("purchases", "stripe_checkout_session_id = {:s}", map[string]any{"s": sess.ID})
	if err != nil {
		t.Fatalf("paid legacy course session was not recorded: %v", err)
	}
	if purchase.GetString("content_type") != "courses" || purchase.GetString("status") != "completed" {
		t.Fatalf("purchase = %s/%s", purchase.GetString("content_type"), purchase.GetString("status"))
	}
	d.accessLink(t, "earlier@example.com")

	// A replay changes nothing.
	assertContains(t, d.webhook(t, "checkout.session.completed", sess), `"status":"already_processed"`)
	if d.purchaseCount(t) != 1 {
		t.Fatalf("purchases = %d, want 1", d.purchaseCount(t))
	}

	// An unpaid session for a course still grants nothing.
	unpaid := d.session("cs_test_legacy_unpaid", "late@example.com", "complete", stripe.CheckoutSessionPaymentStatusUnpaid,
		map[string]string{"content_type": "courses", "content_id": course.Id})
	assertContains(t, d.webhook(t, "checkout.session.completed", unpaid), `"status":"awaiting_payment"`)
	if d.purchaseCount(t) != 1 {
		t.Fatalf("unpaid course session created a purchase")
	}

	// Checkout stays closed for the course, and the course is untouched.
	rec = do(t, d.h, testRequest{method: http.MethodPost, url: "/api/content/checkout", body: `{"content_type":"courses","content_id":"` + course.Id + `","success_url":"https://facet.example/ok","cancel_url":"https://facet.example/no"}`})
	assertStatus(t, rec, http.StatusForbidden)
	assertContains(t, rec, "course_sales_unavailable")
	after, err := d.app.FindRecordById("courses", course.Id)
	if err != nil {
		t.Fatal(err)
	}
	if after.GetString("access_tier") != "paid" || after.GetInt("price") != 4900 {
		t.Fatalf("course became %s/%d", after.GetString("access_tier"), after.GetInt("price"))
	}
}

// Through the real collection API: an admin can edit an imported paid
// course without it being made free, but can't create, switch to or
// reprice a paid course.
func TestAdminCourseSavesKeepImportedPaidDataAndRefuseNewSales(t *testing.T) {
	c := newCourseFixture(t)
	RegisterPurchaseHooks(c.pb, c.crypto, c.rl)
	_, token := c.createAdmin(t, "owner@example.com", "a-strong-password", nil)
	h := c.serve(t)
	auth := map[string]string{"Authorization": token}
	url := "/api/collections/courses/records"

	// Editing the imported paid course without touching tier or price works
	// and leaves both exactly as they were.
	rec := do(t, h, testRequest{method: http.MethodPatch, url: url + "/" + c.course.Id, headers: auth, body: `{"title":"Renamed"}`})
	assertStatus(t, rec, http.StatusOK)
	got, err := c.app.FindRecordById("courses", c.course.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got.GetString("title") != "Renamed" || got.GetString("access_tier") != "paid" || got.GetInt("price") != 2500 {
		t.Fatalf("imported course after edit = %q %s/%d", got.GetString("title"), got.GetString("access_tier"), got.GetInt("price"))
	}

	// Repricing it is refused and stores nothing.
	rec = do(t, h, testRequest{method: http.MethodPatch, url: url + "/" + c.course.Id, headers: auth, body: `{"price":100}`})
	assertStatus(t, rec, http.StatusForbidden)
	assertContains(t, rec, "course_sales_unavailable")
	if got, _ := c.app.FindRecordById("courses", c.course.Id); got.GetInt("price") != 2500 {
		t.Fatalf("refused reprice stored price %d", got.GetInt("price"))
	}

	// New paid courses and free-to-paid switches are refused.
	rec = do(t, h, testRequest{method: http.MethodPost, url: url, headers: auth, body: `{"title":"New","slug":"new-paid","access_tier":"paid","price":1000,"visibility":"public"}`})
	assertStatus(t, rec, http.StatusForbidden)
	if _, err := c.app.FindFirstRecordByFilter("courses", "slug = 'new-paid'"); err == nil {
		t.Fatal("a refused paid course was stored")
	}
	rec = do(t, h, testRequest{method: http.MethodPost, url: url, headers: auth, body: `{"title":"Free","slug":"free-one","access_tier":"free","price":0,"visibility":"public"}`})
	assertStatus(t, rec, http.StatusOK)
	free, err := c.app.FindFirstRecordByFilter("courses", "slug = 'free-one'")
	if err != nil {
		t.Fatal(err)
	}
	rec = do(t, h, testRequest{method: http.MethodPatch, url: url + "/" + free.Id, headers: auth, body: `{"access_tier":"paid","price":1000}`})
	assertStatus(t, rec, http.StatusForbidden)
	if got, _ := c.app.FindRecordById("courses", free.Id); got.GetString("access_tier") != "free" {
		t.Fatal("a free course was switched to paid")
	}
}

// Course selling is Facet Cloud-only. These checks prove the self-hosted
// build refuses every course sale on the server, before a coupon is claimed
// or a purchase is written, no matter what the request or environment says.

func TestCourseCheckoutIsRefusedBeforeAnySideEffect(t *testing.T) {
	t.Setenv("FACET_MANAGED", "true")
	t.Setenv("FACET_FEATURE_COURSES", "true")
	if services.LoadPlanConfig().HasFeature("course_sales") {
		t.Fatal("course_sales must stay false even with FACET_MANAGED=true")
	}

	c := newCourseFixture(t)
	settings := c.app.Settings()
	settings.Meta.AppURL = "https://facet.example"
	if err := c.app.Save(settings); err != nil {
		t.Fatal(err)
	}
	RegisterPurchaseHooks(c.pb, c.crypto, c.rl)
	RegisterCouponHooks(c.pb, c.rl)
	post := c.create(t, "posts", map[string]any{"title": "Paid", "slug": "paid", "visibility": "public", "access_tier": "paid", "price": 500, "content": "x"})
	c.create(t, "coupons", map[string]any{"code": "FREEALL", "discount_type": "percentage", "discount_value": 100, "applicable_type": "all", "max_uses": 5, "current_uses": 0, "active": true})
	c.create(t, "coupons", map[string]any{"code": "FREECOURSE", "discount_type": "percentage", "discount_value": 100, "applicable_type": "courses", "max_uses": 5, "current_uses": 0, "active": true})
	h := c.serve(t)

	for _, body := range []string{
		`{"content_type":"courses","content_id":"` + c.course.Id + `","success_url":"https://facet.example/ok","cancel_url":"https://facet.example/no"}`,
		`{"content_type":"courses","content_id":"` + c.course.Id + `","success_url":"https://facet.example/ok","cancel_url":"https://facet.example/no","coupon_code":"FREEALL"}`,
		`{"content_type":"courses","content_id":"` + c.course.Id + `","success_url":"https://facet.example/ok","cancel_url":"https://facet.example/no","coupon_code":"FREECOURSE"}`,
		// Forged: nonexistent course and off-site redirects still get the same refusal.
		`{"content_type":"courses","content_id":"nope","success_url":"https://evil.example/","cancel_url":"","coupon_code":"FREEALL"}`,
	} {
		rec := do(t, h, testRequest{method: http.MethodPost, url: "/api/content/checkout", body: body})
		assertStatus(t, rec, http.StatusForbidden)
		assertContains(t, rec, "course_sales_unavailable")
	}

	if n, _ := c.app.CountRecords("purchases"); n != 0 {
		t.Fatalf("%d purchases were created by refused course checkouts", n)
	}
	for _, code := range []string{"FREEALL", "FREECOURSE"} {
		coupon, err := c.app.FindFirstRecordByFilter("coupons", "code = {:c}", map[string]any{"c": code})
		if err != nil {
			t.Fatal(err)
		}
		if coupon.GetInt("current_uses") != 0 {
			t.Fatalf("%s current_uses = %d, want 0", code, coupon.GetInt("current_uses"))
		}
	}

	rec := do(t, h, testRequest{method: http.MethodPost, url: "/api/coupons/validate", body: `{"code":"FREEALL","content_type":"courses","content_id":"` + c.course.Id + `"}`})
	assertContains(t, rec, `"valid":false`)

	// Other paid content still checks out, including a free coupon.
	rec = do(t, h, testRequest{method: http.MethodPost, url: "/api/content/checkout", body: `{"content_type":"posts","content_id":"` + post.Id + `","success_url":"https://facet.example/ok","cancel_url":"https://facet.example/no","coupon_code":"FREEALL"}`})
	assertStatus(t, rec, http.StatusOK)
	if n, _ := c.app.CountRecords("purchases"); n != 1 {
		t.Fatalf("post coupon checkout created %d purchases, want 1", n)
	}

	// The public course API states the capability, and an imported paid
	// course stays paid rather than being opened up for free.
	rec = do(t, h, testRequest{url: "/api/courses/course"})
	assertStatus(t, rec, http.StatusOK)
	assertContains(t, rec, `"sales_available":false`)
	assertContains(t, rec, `"access_tier":"paid"`)
	assertNotContains(t, rec, "PAID LESSON BODY")
}

func TestCourseSaveCannotPutACourseUpForSale(t *testing.T) {
	c := newCourseFixture(t)
	courses, err := c.app.FindCollectionByNameOrId("courses")
	if err != nil {
		t.Fatal(err)
	}

	// Free courses save as usual.
	free := c.create(t, "courses", map[string]any{"title": "Free", "slug": "free", "access_tier": "free", "visibility": "public"})
	if courseSaleChangeBlocked(free) {
		t.Fatal("a free course was treated as a sale")
	}

	// A new paid course is refused.
	fresh := core.NewRecord(courses)
	fresh.Set("access_tier", "paid")
	fresh.Set("price", 1000)
	if !courseSaleChangeBlocked(fresh) {
		t.Fatal("a new paid course was allowed")
	}

	// Turning a free course paid is refused.
	free.Set("access_tier", "paid")
	free.Set("price", 1000)
	if !courseSaleChangeBlocked(free) {
		t.Fatal("a free course was turned paid")
	}

	// An imported paid course can be edited without changing its price,
	// but can't be repriced.
	imported, err := c.app.FindRecordById("courses", c.course.Id)
	if err != nil {
		t.Fatal(err)
	}
	imported.Set("title", "Renamed")
	if courseSaleChangeBlocked(imported) {
		t.Fatal("editing an imported paid course was refused")
	}
	imported.Set("price", 9900)
	if !courseSaleChangeBlocked(imported) {
		t.Fatal("an imported paid course was repriced")
	}
}
