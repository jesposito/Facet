package hooks

import (
	"reflect"
	"testing"
)

func TestFilterBySelectedItemsKeepOrder_NotConfigured(t *testing.T) {
	items := []map[string]interface{}{
		{"id": "a", "category": "X"},
		{"id": "b", "category": "X"},
		{"id": "c", "category": "Y"},
	}
	got := filterBySelectedItemsKeepOrder(items, nil, false)
	if !reflect.DeepEqual(got, items) {
		t.Errorf("unconfigured should return all items unchanged; got %v", got)
	}
}

func TestFilterBySelectedItemsKeepOrder_ConfiguredEmpty(t *testing.T) {
	items := []map[string]interface{}{
		{"id": "a"},
		{"id": "b"},
	}
	got := filterBySelectedItemsKeepOrder(items, []string{}, true)
	if got != nil {
		t.Errorf("configured + empty selection should return nil; got %v", got)
	}
}

// Core regression for issue #259: items are filtered to selectedItems set
// membership, but their order is taken from the input slice (DB sort order),
// NOT from the selectedItems array.
func TestFilterBySelectedItemsKeepOrder_PreservesInputOrder(t *testing.T) {
	items := []map[string]interface{}{
		{"id": "a", "category": "Frontend", "sort_order": 0, "name": "Svelte"},
		{"id": "b", "category": "Frontend", "sort_order": 1, "name": "React"},
		{"id": "c", "category": "Frontend", "sort_order": 2, "name": "Vue"},
		{"id": "d", "category": "Backend", "sort_order": 0, "name": "Go"},
	}
	// selectedItems in DIFFERENT order than input — input order must win.
	selected := []string{"d", "c", "a", "b"}
	got := filterBySelectedItemsKeepOrder(items, selected, true)

	if len(got) != 4 {
		t.Fatalf("expected 4 items, got %d", len(got))
	}
	wantIDs := []string{"a", "b", "c", "d"} // input order, not selected order
	for i, item := range got {
		if item["id"].(string) != wantIDs[i] {
			t.Errorf("position %d: got id=%q, want %q (input order must win over selectedItems order)",
				i, item["id"], wantIDs[i])
		}
	}
}

func TestFilterBySelectedItemsKeepOrder_FiltersUnselected(t *testing.T) {
	items := []map[string]interface{}{
		{"id": "a"}, {"id": "b"}, {"id": "c"}, {"id": "d"},
	}
	selected := []string{"a", "c"}
	got := filterBySelectedItemsKeepOrder(items, selected, true)
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0]["id"] != "a" || got[1]["id"] != "c" {
		t.Errorf("expected [a, c] in input order, got %v", got)
	}
}

// Contrast test: confirm the OLD helper still behaves the old way (selectedItems
// array order wins). This is the behavior the new helper deliberately differs from.
func TestFilterBySelectedItemsWithDefault_UsesSelectedItemsOrder(t *testing.T) {
	items := []map[string]interface{}{
		{"id": "a"}, {"id": "b"}, {"id": "c"},
	}
	selected := []string{"c", "a", "b"}
	got := filterBySelectedItemsWithDefault(items, selected, true)
	if len(got) != 3 {
		t.Fatalf("expected 3 items, got %d", len(got))
	}
	wantIDs := []string{"c", "a", "b"}
	for i, item := range got {
		if item["id"].(string) != wantIDs[i] {
			t.Errorf("position %d: got id=%q, want %q (selectedItems order)",
				i, item["id"], wantIDs[i])
		}
	}
}
