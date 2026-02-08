package services

import (
	"testing"
)

func TestCleanupResultTotal(t *testing.T) {
	tests := []struct {
		name   string
		result CleanupResult
		want   int
	}{
		{
			name:   "empty result",
			result: CleanupResult{},
			want:   0,
		},
		{
			name: "all fields populated",
			result: CleanupResult{
				ExpiredShareTokens:        3,
				RevokedShareTokens:        2,
				ExpiredVerificationTokens: 1,
				VerifiedTokens:            4,
				FailedExports:             5,
				StuckExports:              1,
			},
			want: 16,
		},
		{
			name: "only share tokens",
			result: CleanupResult{
				ExpiredShareTokens: 5,
				RevokedShareTokens: 3,
			},
			want: 8,
		},
		{
			name: "only exports",
			result: CleanupResult{
				FailedExports: 2,
				StuckExports:  1,
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.Total()
			if got != tt.want {
				t.Errorf("CleanupResult.Total() = %d, want %d", got, tt.want)
			}
		})
	}
}
