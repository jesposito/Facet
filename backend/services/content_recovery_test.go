package services

import "testing"

func TestCollapseWS(t *testing.T) {
	cases := map[string]string{
		"a\n\nb":            "a b",
		"a   b":             "a b",
		"  a\n\n\nb  ":      "a b",
		"a\r\n\r\nb":        "a b",
		"# T\n\nFirst.":     "# T First.",
		"nbsp  x": "nbsp x",
		"":                  "",
		"   ":               "",
	}
	for in, want := range cases {
		if got := collapseWS(in); got != want {
			t.Errorf("collapseWS(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestShouldRestore(t *testing.T) {
	tests := []struct {
		name    string
		current string
		backup  string
		want    bool
	}{
		{
			name:    "flattened paragraphs are restored",
			current: "# My Post First paragraph. Second paragraph.",
			backup:  "# My Post\n\nFirst paragraph.\n\nSecond paragraph.",
			want:    true,
		},
		{
			name:    "flattened list is restored",
			current: "- a - b - c",
			backup:  "- a\n- b\n- c",
			want:    true,
		},
		{
			name:    "genuine re-edit (different words) is left alone",
			current: "Totally different text now",
			backup:  "# My Post\n\nFirst paragraph.",
			want:    false,
		},
		{
			name:    "already-structured content is not touched",
			current: "a\n\nb",
			backup:  "a\n\nb",
			want:    false,
		},
		{
			name:    "identical single line is not touched",
			current: "hello world",
			backup:  "hello world",
			want:    false,
		},
		{
			name:    "backup with same words but no extra structure is skipped",
			current: "one two three",
			backup:  "one two three ",
			want:    false,
		},
		{
			name:    "empty current is skipped",
			current: "",
			backup:  "\n\n",
			want:    false,
		},
		{
			name:    "current already has more structure than backup is skipped",
			current: "a\n\nb",
			backup:  "a b",
			want:    false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRestore(tc.current, tc.backup); got != tc.want {
				t.Errorf("shouldRestore(%q, %q) = %v, want %v", tc.current, tc.backup, got, tc.want)
			}
		})
	}
}

func TestShouldRestoreJSONArray(t *testing.T) {
	tests := []struct {
		name    string
		current string
		backup  string
		want    bool
	}{
		{"flattened bullets restored", `["a b c"]`, `["a","b","c"]`, true},
		{"genuine re-edit left alone", `["totally new bullet"]`, `["a","b","c"]`, false},
		{"identical is skipped", `["a","b"]`, `["a","b"]`, false},
		{"backup not more structured is skipped", `["a","b","c"]`, `["a b c"]`, false},
		{"non-json current is skipped", `not json at all`, `["a","b"]`, false},
		{"empty current is skipped", `[]`, `["a","b"]`, false},
		{"whitespace-only elements skipped", `[" "]`, `["a","b"]`, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRestoreJSONArray(tc.current, tc.backup); got != tc.want {
				t.Errorf("shouldRestoreJSONArray(%q, %q) = %v, want %v", tc.current, tc.backup, got, tc.want)
			}
		})
	}
}

func TestSafeIdent(t *testing.T) {
	good := []string{"posts", "hero_summary", "content", "_x", "a1"}
	bad := []string{"", "1abc", "drop table", "a-b", "a;b", "a.b", "a b"}
	for _, s := range good {
		if !safeIdent(s) {
			t.Errorf("safeIdent(%q) = false, want true", s)
		}
	}
	for _, s := range bad {
		if safeIdent(s) {
			t.Errorf("safeIdent(%q) = true, want false", s)
		}
	}
}
