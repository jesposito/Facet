package services

import (
	"testing"
	"time"
)

func TestViewDedupCache_FirstHitCounts(t *testing.T) {
	c := NewViewDedupCache(30 * time.Minute)

	if !c.ShouldCount("hashA", "view1") {
		t.Error("first hit for a key should count")
	}
	if c.ShouldCount("hashA", "view1") {
		t.Error("repeat hit within window should not count")
	}
}

func TestViewDedupCache_DifferentTargetsIndependent(t *testing.T) {
	c := NewViewDedupCache(30 * time.Minute)

	if !c.ShouldCount("hashA", "view1") {
		t.Error("first hit on view1 should count")
	}
	// Same visitor, different target — must count independently.
	if !c.ShouldCount("hashA", "view2") {
		t.Error("first hit on view2 should count (different target)")
	}
	// Homepage sentinel is a separate target too.
	if !c.ShouldCount("hashA", HomepageDedupTarget) {
		t.Error("first hit on homepage target should count")
	}
}

func TestViewDedupCache_DifferentVisitorsIndependent(t *testing.T) {
	c := NewViewDedupCache(30 * time.Minute)

	if !c.ShouldCount("hashA", "view1") {
		t.Error("visitor A should count")
	}
	if !c.ShouldCount("hashB", "view1") {
		t.Error("visitor B should count (different visitor, same target)")
	}
}

func TestViewDedupCache_EmptyHashAlwaysCounts(t *testing.T) {
	c := NewViewDedupCache(30 * time.Minute)

	// No resolvable IP -> cannot dedup -> always count (avoid undercounting).
	if !c.ShouldCount("", "view1") {
		t.Error("empty visitor hash should always count (1)")
	}
	if !c.ShouldCount("", "view1") {
		t.Error("empty visitor hash should always count (2)")
	}
}

func TestViewDedupCache_WindowExpiry(t *testing.T) {
	// Tiny TTL so the entry expires almost immediately.
	c := NewViewDedupCache(20 * time.Millisecond)

	if !c.ShouldCount("hashA", "view1") {
		t.Error("first hit should count")
	}
	if c.ShouldCount("hashA", "view1") {
		t.Error("immediate repeat should be deduped")
	}

	time.Sleep(30 * time.Millisecond)

	if !c.ShouldCount("hashA", "view1") {
		t.Error("after the window expires the visitor should count again")
	}
}

func TestHashIP_Deterministic(t *testing.T) {
	a := HashIP("203.0.113.7")
	b := HashIP("203.0.113.7")
	if a == "" {
		t.Fatal("HashIP should not return empty for a non-empty IP")
	}
	if a != b {
		t.Error("HashIP should be deterministic within the same day")
	}
	if HashIP("203.0.113.8") == a {
		t.Error("different IPs should hash differently")
	}
	if HashIP("") != "" {
		t.Error("empty IP should hash to empty string")
	}
}

func TestShouldCountView_BotIsSkipped(t *testing.T) {
	// A known bot UA must never count, regardless of IP/target.
	if ShouldCountView("Googlebot/2.1 (+http://www.google.com/bot.html)", "203.0.113.9", "viewX") {
		t.Error("bot user agent should not be counted")
	}
	// Empty UA is treated as a bot by IsBot.
	if ShouldCountView("", "203.0.113.9", "viewX") {
		t.Error("empty user agent should not be counted")
	}
}
