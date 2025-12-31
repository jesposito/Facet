/**
 * Param matcher for view slugs
 *
 * This matcher ensures that /<slug> routes don't collide with system routes.
 * Reserved paths are protected at the routing level.
 *
 * Reserved paths include:
 * - admin: Admin dashboard
 * - api: API endpoints (proxied to PocketBase)
 * - s: Share token entry point
 * - v: Legacy view routes (redirects to /<slug>)
 * - _: SvelteKit internal
 * - assets: Static assets
 * - favicon.ico, robots.txt: Standard web files
 * - health: Health check endpoint
 */

import type { ParamMatcher } from '@sveltejs/kit';

// Reserved slugs that cannot be used for views
// These correspond to existing routes or system paths
const RESERVED_SLUGS = new Set([
	// Existing routes
	'admin',
	'api',
	's',
	'v',
	'projects',
	'posts',
	// SvelteKit internal
	'_app',
	'_',
	// Static assets
	'assets',
	'static',
	// Standard web files
	'favicon.ico',
	'robots.txt',
	'sitemap.xml',
	// System endpoints
	'health',
	'healthz',
	'ready',
	// Common reserved paths
	'login',
	'logout',
	'auth',
	'oauth',
	'callback',
	// Prevent confusion
	'home',
	'index',
	'default',
	'profile'
]);

export const match: ParamMatcher = (param) => {
	// Must be a non-empty string
	if (!param || typeof param !== 'string') {
		return false;
	}

	// Must not be a reserved slug
	if (RESERVED_SLUGS.has(param.toLowerCase())) {
		return false;
	}

	// Must be a valid slug format (alphanumeric, hyphens, underscores)
	// Prevents paths like "../" or weird characters
	const slugPattern = /^[a-zA-Z0-9][a-zA-Z0-9_-]*$/;
	if (!slugPattern.test(param)) {
		return false;
	}

	// Must not start with underscore (SvelteKit convention)
	if (param.startsWith('_')) {
		return false;
	}

	// Maximum length to prevent abuse
	if (param.length > 100) {
		return false;
	}

	return true;
};

// Export for backend validation sync
export { RESERVED_SLUGS };
