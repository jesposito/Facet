/**
 * Legacy view route: /v/<slug>
 *
 * DEPRECATED: This route now redirects to /<slug> (the canonical URL).
 *
 * The redirect preserves cookies so share tokens and password JWTs continue to work.
 * This is a 301 (permanent) redirect to signal to browsers and search engines
 * that /<slug> is the canonical URL.
 */

import type { PageServerLoad } from './$types';
import { redirect } from '@sveltejs/kit';

export const load: PageServerLoad = async ({ params }) => {
	const { slug } = params;

	// 301 Permanent redirect to canonical URL
	// Cookies are preserved automatically by the browser
	throw redirect(301, `/${slug}`);
};
