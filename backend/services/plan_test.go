package services

import "testing"

func TestSelfHostedCoursesDefaultOn(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
		want  bool
	}{
		{"unset", "", true},
		{"explicit true", "true", true},
		{"explicit false", "false", false},
		{"zero", "0", false},
		{"unparseable keeps the default", "maybe", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("FACET_MANAGED", "")
			t.Setenv("FACET_FEATURE_COURSES", tc.value)
			cfg := LoadPlanConfig()
			if cfg.Features.Courses != tc.want || cfg.HasFeature("courses") != tc.want {
				t.Fatalf("courses = %v, want %v", cfg.Features.Courses, tc.want)
			}
		})
	}
}

func TestOtherSelfHostedFeaturesUnchanged(t *testing.T) {
	t.Setenv("FACET_MANAGED", "")
	t.Setenv("FACET_FEATURE_COURSES", "false")
	cfg := LoadPlanConfig()
	if cfg.Managed || !cfg.Features.Analytics || !cfg.Features.API || !cfg.Features.Newsletter || !cfg.Features.Pricing || cfg.Features.BadgeForced {
		t.Fatalf("unexpected self-hosted features: %+v", cfg.Features)
	}
}

func TestManagedCoursesStayOn(t *testing.T) {
	t.Setenv("FACET_MANAGED", "true")
	t.Setenv("FACET_FEATURE_COURSES", "")
	if !LoadPlanConfig().HasFeature("courses") {
		t.Fatal("managed mode lost courses")
	}
}

func TestBuyerTokenCoversOnlyCheckedPurchases(t *testing.T) {
	p := NewPurchaseService(NewCryptoService("kkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkk"))
	if _, _, err := p.GenerateBuyerJWT(""); err == nil {
		t.Fatal("buyer token issued without an email")
	}
	token, _, err := p.GenerateBuyerJWT("buyer@example.com")
	if err != nil {
		t.Fatal(err)
	}
	ids, email, err := p.ValidatePurchaseJWT(token)
	if err != nil || email != "buyer@example.com" {
		t.Fatalf("validate: %v %q", err, email)
	}
	if !p.HasPurchaseAccess(ids, "any-purchase-id") {
		t.Fatal("buyer token should cover the buyer's purchases")
	}
	if p.HasPurchaseAccess(ids, "") {
		t.Fatal("buyer token matched an empty purchase ID")
	}
	if p.HasPurchaseAccess([]string{"one"}, "two") {
		t.Fatal("explicit token matched another purchase")
	}
}
