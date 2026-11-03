// Crawlers that collect pages for AI training datasets. Facet has blocked
// these since the original static robots.txt; search engines stay allowed.
const AI_TRAINING_AGENTS = [
	'GPTBot',
	'ChatGPT-User',
	'ClaudeBot',
	'Claude-Web',
	'anthropic-ai',
	'Google-Extended',
	'CCBot',
	'Bytespider',
	'Applebot-Extended',
	'PerplexityBot',
	'Diffbot',
	'FacebookBot',
	'Meta-ExternalAgent',
	'cohere-ai',
	'omgili'
] as const;

// Paths that are private, token-bearing or not pages at all. Share links
// (/s/, /v/) and email links (/unsubscribe/, /testimonial/) carry secrets in
// the URL, so even an advisory "please don't" is worth sending.
export const ROBOTS_DISALLOW = [
	'/admin',
	'/api/',
	'/s/',
	'/v/',
	'/unsubscribe',
	'/testimonial/',
	'/checkout'
] as const;

/**
 * Resolve the public origin for absolute URLs such as the sitemap link.
 * The configured app URL wins so links don't follow a spoofed Host header;
 * the request origin is the fallback for unconfigured local installs.
 */
export function publicOrigin(
	env: { PUBLIC_APP_URL?: string; APP_URL?: string },
	requestOrigin: string
): string {
	for (const candidate of [env.PUBLIC_APP_URL, env.APP_URL]) {
		if (!candidate) continue;
		try {
			const url = new URL(candidate.trim());
			if (url.protocol === 'http:' || url.protocol === 'https:') {
				return url.origin + url.pathname.replace(/\/+$/, '');
			}
		} catch {
			/* ignore a malformed setting and try the next one */
		}
	}
	return requestOrigin;
}

export function buildRobotsTxt(origin: string): string {
	const lines = ['# Facet - Personal Profile Platform', ''];

	lines.push('# AI training crawlers');
	for (const agent of AI_TRAINING_AGENTS) {
		lines.push(`User-agent: ${agent}`);
	}
	lines.push('Disallow: /', '');

	// Every other crawler, search engines included, shares one group. A
	// crawler obeys only the most specific group naming it, so giving
	// Googlebot its own "Allow: /" group would drop these disallows for it.
	lines.push('User-agent: *');
	for (const path of ROBOTS_DISALLOW) {
		lines.push(`Disallow: ${path}`);
	}
	lines.push('Allow: /', '');

	lines.push(`Sitemap: ${origin}/sitemap.xml`, '');
	return lines.join('\n');
}
