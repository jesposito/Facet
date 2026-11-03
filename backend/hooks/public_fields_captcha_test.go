package hooks

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestRedactUnavailableProtection(t *testing.T) {
	f := newSecurityFixture(t)
	captcha := f.create(t, "contact_methods", map[string]any{"type": "email", "value": "hidden@example.com", "label": "Email", "protection_level": "captcha", "visibility": "public"})
	reveal := f.create(t, "contact_methods", map[string]any{"type": "email", "value": "shown@example.com", "label": "Email", "protection_level": "click_to_reveal", "visibility": "public"})

	items := serializeRecords([]*core.Record{captcha, reveal}, nil)
	for i, record := range []*core.Record{captcha, reveal} {
		redactUnavailableProtection(record, items[i])
	}
	if _, ok := items[0]["value"]; ok {
		t.Fatalf("captcha contact kept its value: %v", items[0])
	}
	if items[0]["label"] != "Email" || items[0]["protection_level"] != "captcha" {
		t.Fatalf("captcha contact lost its label or level: %v", items[0])
	}
	if items[1]["value"] != "shown@example.com" {
		t.Fatalf("click-to-reveal contact lost its value: %v", items[1])
	}

	// Admins still read and edit the stored value.
	stored, err := f.app.FindRecordById("contact_methods", captcha.Id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.GetString("value") != "hidden@example.com" {
		t.Fatal("redaction changed the stored record")
	}
}
