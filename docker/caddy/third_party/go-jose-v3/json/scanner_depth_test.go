// Regression test for the GHSA-vvm8-gf78-28g4 backport described in
// ../FACET-PATCH.md.

package json

import (
	"strings"
	"testing"
)

func TestDeepNestingIsRejected(t *testing.T) {
	for _, open := range []string{"[", `{"a":`} {
		var v interface{}
		err := Unmarshal([]byte(strings.Repeat(open, maxNestingDepth+1)), &v)
		if err == nil || !strings.Contains(err.Error(), "exceeded max depth") {
			t.Fatalf("%q nested past the limit: got %v, want max depth error", open, err)
		}
	}
	var v interface{}
	if err := Unmarshal([]byte(strings.Repeat("[", 100)+strings.Repeat("]", 100)), &v); err != nil {
		t.Fatalf("shallow nesting must still parse: %v", err)
	}
}
