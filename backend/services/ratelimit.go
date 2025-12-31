package services

import (
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitTier defines rate limit parameters for a category of endpoints
type RateLimitTier struct {
	Rate  rate.Limit // Requests per second
	Burst int        // Maximum burst size
}

// Predefined tiers for different endpoint types
var (
	// TierStrict: Password/token validation - very restrictive
	// 5 requests per minute = 0.083/sec, burst of 3
	TierStrict = RateLimitTier{Rate: rate.Limit(5.0 / 60.0), Burst: 3}

	// TierModerate: Token validation, sensitive operations
	// 10 requests per minute = 0.167/sec, burst of 5
	TierModerate = RateLimitTier{Rate: rate.Limit(10.0 / 60.0), Burst: 5}

	// TierNormal: General public endpoints
	// 60 requests per minute = 1/sec, burst of 10
	TierNormal = RateLimitTier{Rate: rate.Limit(1.0), Burst: 10}
)

// RateLimitService provides per-IP rate limiting using token bucket algorithm
type RateLimitService struct {
	// Per-tier limiters: tier name -> IP -> limiter
	limiters map[string]map[string]*rate.Limiter
	mu       sync.RWMutex

	// Tier configurations
	tiers map[string]RateLimitTier

	// Cleanup interval and TTL
	cleanupInterval time.Duration
	limiterTTL      time.Duration

	// Last access times for cleanup
	lastAccess   map[string]map[string]time.Time
	lastAccessMu sync.RWMutex

	// Proxy trust settings
	trustProxy bool
}

// NewRateLimitService creates a new rate limiting service
func NewRateLimitService() *RateLimitService {
	trustProxy := os.Getenv("TRUST_PROXY") == "true"

	svc := &RateLimitService{
		limiters: make(map[string]map[string]*rate.Limiter),
		tiers: map[string]RateLimitTier{
			"strict":   TierStrict,
			"moderate": TierModerate,
			"normal":   TierNormal,
		},
		cleanupInterval: 5 * time.Minute,
		limiterTTL:      10 * time.Minute,
		lastAccess:      make(map[string]map[string]time.Time),
		trustProxy:      trustProxy,
	}

	// Initialize limiter maps for each tier
	for tier := range svc.tiers {
		svc.limiters[tier] = make(map[string]*rate.Limiter)
		svc.lastAccess[tier] = make(map[string]time.Time)
	}

	// Start background cleanup
	go svc.cleanupLoop()

	return svc
}

// Allow checks if a request should be allowed for the given tier
// Returns (allowed, retryAfter) where retryAfter is seconds until next allowed request
func (s *RateLimitService) Allow(r *http.Request, tier string) (bool, int) {
	ip := s.getClientIP(r)
	limiter := s.getLimiter(ip, tier)

	if limiter.Allow() {
		return true, 0
	}

	// Calculate retry-after (time until next token available)
	reservation := limiter.Reserve()
	delay := reservation.Delay()
	reservation.Cancel() // Don't actually consume the token

	retryAfter := int(delay.Seconds()) + 1 // Round up
	if retryAfter < 1 {
		retryAfter = 1
	}

	return false, retryAfter
}

// getLimiter returns or creates a limiter for the given IP and tier
func (s *RateLimitService) getLimiter(ip, tier string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	tierConfig, ok := s.tiers[tier]
	if !ok {
		tierConfig = TierNormal
	}

	if _, exists := s.limiters[tier]; !exists {
		s.limiters[tier] = make(map[string]*rate.Limiter)
		s.lastAccess[tier] = make(map[string]time.Time)
	}

	limiter, exists := s.limiters[tier][ip]
	if !exists {
		limiter = rate.NewLimiter(tierConfig.Rate, tierConfig.Burst)
		s.limiters[tier][ip] = limiter
	}

	// Update last access time
	s.lastAccessMu.Lock()
	s.lastAccess[tier][ip] = time.Now()
	s.lastAccessMu.Unlock()

	return limiter
}

// getClientIP extracts the client IP address from the request
// Respects TRUST_PROXY setting for X-Forwarded-For and CF-Connecting-IP
func (s *RateLimitService) getClientIP(r *http.Request) string {
	if s.trustProxy {
		// Priority 1: Cloudflare's CF-Connecting-IP (most reliable when using Cloudflare)
		if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
			return cfIP
		}

		// Priority 2: X-Real-IP (set by some proxies like nginx)
		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			return realIP
		}

		// Priority 3: X-Forwarded-For (take the first/leftmost IP, which is the original client)
		// WARNING: This can be spoofed if not behind a trusted proxy
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// X-Forwarded-For format: "client, proxy1, proxy2"
			// The leftmost IP is the original client
			ips := strings.Split(xff, ",")
			if len(ips) > 0 {
				clientIP := strings.TrimSpace(ips[0])
				if clientIP != "" {
					return clientIP
				}
			}
		}
	}

	// Fallback: Use RemoteAddr (direct connection IP)
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr might not have a port
		return r.RemoteAddr
	}
	return ip
}

// cleanupLoop periodically removes stale limiters
func (s *RateLimitService) cleanupLoop() {
	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanup()
	}
}

// cleanup removes limiters that haven't been accessed recently
func (s *RateLimitService) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastAccessMu.Lock()
	defer s.lastAccessMu.Unlock()

	cutoff := time.Now().Add(-s.limiterTTL)

	for tier, accessTimes := range s.lastAccess {
		for ip, lastAccess := range accessTimes {
			if lastAccess.Before(cutoff) {
				delete(s.limiters[tier], ip)
				delete(s.lastAccess[tier], ip)
			}
		}
	}
}

// TrustProxy returns whether proxy headers are trusted
func (s *RateLimitService) TrustProxy() bool {
	return s.trustProxy
}

// Stats returns current limiter statistics (for debugging/monitoring)
func (s *RateLimitService) Stats() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]int)
	for tier, limiters := range s.limiters {
		stats[tier] = len(limiters)
	}
	return stats
}
