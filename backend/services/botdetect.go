package services

import "strings"

// knownBotPatterns contains lowercase substrings that identify non-human user agents.
// Search engine crawlers are included — analytics should count humans only (matches GA4 behavior).
var knownBotPatterns = []string{
	// Generic bot indicators
	"bot", "crawl", "spider", "slurp",

	// HTTP libraries / CLI tools
	"wget", "curl", "python-requests", "python-urllib",
	"go-http-client", "node-fetch", "axios", "httpie",
	"java/", "libwww", "lwp-", "okhttp", "httpclient",

	// Search engines
	"googlebot", "bingbot", "yandex", "baiduspider",
	"duckduckbot", "sogou", "exabot", "ia_archiver",

	// AI training bots
	"gptbot", "chatgpt-user", "claudebot", "claude-web",
	"anthropic-ai", "google-extended", "ccbot", "bytespider",
	"applebot-extended", "perplexitybot", "diffbot",
	"facebookbot", "meta-externalagent", "cohere-ai",

	// Monitoring / infrastructure
	"uptimerobot", "pingdom", "site24x7", "statuscake",
	"facet-health", "facet-provisioning-health",

	// SEO / link checkers
	"semrush", "ahrefs", "mj12bot", "dotbot",
	"screaming frog", "rogerbot", "linkcheck",

	// Misc
	"headlesschrome", "phantomjs", "scrapy",
}

// IsBot returns true if the user agent string matches a known bot pattern.
// An empty user agent is also treated as a bot (no legitimate browser sends empty UA).
func IsBot(userAgent string) bool {
	if userAgent == "" {
		return true
	}
	lower := strings.ToLower(userAgent)
	for _, pattern := range knownBotPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}
