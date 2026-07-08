package buildinfo

import "testing"

func TestShortSHA(t *testing.T) {
	original := SHA
	defer func() {
		SHA = original
	}()

	tests := []struct {
		name string
		sha  string
		want string
	}{
		{name: "unset", sha: "unset", want: "unset"},
		{name: "short", sha: "abcdef", want: "abcdef"},
		{name: "long", sha: "abcdef1234567890", want: "abcdef1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SHA = tt.sha
			if got := ShortSHA(); got != tt.want {
				t.Fatalf("ShortSHA() = %q, want %q", got, tt.want)
			}
		})
	}
}
