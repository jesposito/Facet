# Stripe

Checked **2026-11-10**.

Facet directly requires `github.com/stripe/stripe-go/v82` **82.5.1**, which is the latest release on the v82 module path. The official Go proxy reports **87.1.0** on the successive v87 path, so newer API majors exist; those are not drop-in version bumps and were not assessed for compatibility.

The backend uses the SDK for Checkout purchases, webhook signature verification and a settings connection test. Webhook test fixtures use the SDK's `stripe.APIVersion`, which keeps test event versions aligned with the selected library. Upgrade review should cover checkout completion, coupon handling, purchase delivery and webhook signature behaviour.

Sources: [`backend/go.mod`](../backend/go.mod), [`backend/hooks/purchase.go`](../backend/hooks/purchase.go), [`backend/hooks/settings.go`](../backend/hooks/settings.go), [`backend/hooks/public_data_test.go`](../backend/hooks/public_data_test.go), [Go proxy v82 latest](https://proxy.golang.org/github.com/stripe/stripe-go/v82/@latest), [Go proxy v87 latest](https://proxy.golang.org/github.com/stripe/stripe-go/v87/@latest), all checked 2026-11-10.
