package hooks

import (
	"testing"
	"time"
)

func TestFillDailyViewCounts_ZeroFillsMissingDays(t *testing.T) {
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	// Only two of the last 7 days had views.
	counts := map[string]int{
		"2026-06-23": 5, // today
		"2026-06-20": 2,
	}

	series := fillDailyViewCounts(counts, 7, now)

	if len(series) != 7 {
		t.Fatalf("expected 7 days, got %d", len(series))
	}
	// Ascending, ending today.
	wantDates := []string{
		"2026-06-17", "2026-06-18", "2026-06-19", "2026-06-20",
		"2026-06-21", "2026-06-22", "2026-06-23",
	}
	wantCounts := []int{0, 0, 0, 2, 0, 0, 5}
	for i := range series {
		if series[i].Date != wantDates[i] {
			t.Errorf("day %d: date = %q, want %q", i, series[i].Date, wantDates[i])
		}
		if series[i].Count != wantCounts[i] {
			t.Errorf("day %d (%s): count = %d, want %d", i, series[i].Date, series[i].Count, wantCounts[i])
		}
	}
}

func TestFillDailyViewCounts_EmptyCountsAllZero(t *testing.T) {
	now := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)
	series := fillDailyViewCounts(map[string]int{}, 30, now)
	if len(series) != 30 {
		t.Fatalf("expected 30 days, got %d", len(series))
	}
	for _, d := range series {
		if d.Count != 0 {
			t.Errorf("%s: expected 0, got %d", d.Date, d.Count)
		}
	}
	if series[len(series)-1].Date != "2026-06-23" {
		t.Errorf("last day = %q, want today 2026-06-23", series[len(series)-1].Date)
	}
}

func TestFillDailyViewCounts_NormalizesToUTCAndGuardsDays(t *testing.T) {
	// A non-UTC "now" must still anchor on the UTC calendar day.
	loc := time.FixedZone("UTC+5", 5*60*60)
	now := time.Date(2026, 6, 23, 2, 0, 0, 0, loc) // 2026-06-22 21:00 UTC
	series := fillDailyViewCounts(map[string]int{}, 1, now)
	if len(series) != 1 || series[0].Date != "2026-06-22" {
		t.Fatalf("expected single day 2026-06-22 (UTC), got %+v", series)
	}
	if got := fillDailyViewCounts(nil, 0, now); len(got) != 0 {
		t.Errorf("days=0 should yield empty series, got %d", len(got))
	}
}
