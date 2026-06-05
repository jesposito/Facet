package services

import (
	"crypto/sha256"
	"fmt"
	"os"
	"sync"
	"time"
)

// HashIP returns a daily-salted SHA-256 hash of an IP address.
// The salt is the current UTC date, so hashes rotate daily and cannot be
// correlated across days (GDPR-friendly). An empty IP returns an empty hash.
//
// This is the canonical IP-hashing routine — both access_logs (analytics) and
// the in-memory view dedup cache use it so their keys line up exactly.
func HashIP(ip string) string {
	if ip == "" {
		return ""
	}
	dailySalt := time.Now().UTC().Format("2006-01-02")
	return fmt.Sprintf("%x", sha256.Sum256([]byte(ip+dailySalt)))
}

// ViewDedupCache is a cheap, thread-safe, in-memory TTL set used to deduplicate
// raw view-counter increments within a short window. It performs no DB I/O, so
// it is safe to consult on the hot request path.
//
// Keys combine a (daily-salted) visitor hash with a target id (a view id, or a
// "homepage" sentinel). The first hit for a key within the TTL counts; repeats
// within the window are skipped. Because the visitor hash is salted with the
// UTC date, the effective window is naturally capped at the day boundary — at
// most one extra count per visitor at midnight, which is acceptable.
type ViewDedupCache struct {
	ttl     time.Duration
	mu      sync.Mutex
	seen    map[string]time.Time
	lastGC  time.Time
	gcEvery time.Duration
}

// HomepageDedupTarget is the sentinel target id used for homepage view counts,
// which are not tied to a specific view record.
const HomepageDedupTarget = "homepage"

// NewViewDedupCache creates a dedup cache with the given TTL window.
func NewViewDedupCache(ttl time.Duration) *ViewDedupCache {
	return &ViewDedupCache{
		ttl:     ttl,
		seen:    make(map[string]time.Time),
		lastGC:  time.Now(),
		gcEvery: ttl,
	}
}

// ShouldCount reports whether an increment should happen for this visitor/target
// pair. It returns true the first time a key is seen within the TTL window and
// records the key; subsequent calls within the window return false.
//
// An empty visitorHash (e.g. no resolvable IP) is always counted — we cannot
// deduplicate what we cannot key on, and dropping those would undercount.
func (c *ViewDedupCache) ShouldCount(visitorHash, targetID string) bool {
	if visitorHash == "" {
		return true
	}

	key := visitorHash + "|" + targetID
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.gcLocked(now)

	if ts, ok := c.seen[key]; ok && now.Sub(ts) < c.ttl {
		return false
	}
	c.seen[key] = now
	return true
}

// gcLocked lazily removes expired entries. Caller must hold c.mu.
func (c *ViewDedupCache) gcLocked(now time.Time) {
	if now.Sub(c.lastGC) < c.gcEvery {
		return
	}
	for key, ts := range c.seen {
		if now.Sub(ts) >= c.ttl {
			delete(c.seen, key)
		}
	}
	c.lastGC = now
}

// Len returns the number of tracked keys (for tests/monitoring).
func (c *ViewDedupCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.seen)
}

// viewCountWindow is the dedup window for raw view-counter increments.
// Overridable via VIEW_DEDUP_WINDOW (Go duration, e.g. "30m") for testing/tuning.
func viewCountWindow() time.Duration {
	if v := os.Getenv("VIEW_DEDUP_WINDOW"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 30 * time.Minute
}

// defaultViewDedup is the process-wide dedup cache shared by the view hooks.
var defaultViewDedup = NewViewDedupCache(viewCountWindow())

// ShouldCountView reports whether a raw view-counter increment should happen for
// the given visitor IP and target, applying bot filtering and ~30-minute dedup.
//
// It returns false when the request is a bot (so bots stop inflating the visible
// counters, matching access_logs) or when this visitor already counted for this
// target within the window. The visitor is keyed on the same daily-salted hash
// used by access_logs, so no raw IP is retained.
func ShouldCountView(userAgent, ip, targetID string) bool {
	if IsBot(userAgent) {
		return false
	}
	return defaultViewDedup.ShouldCount(HashIP(ip), targetID)
}
